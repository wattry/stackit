package flatten

import (
	"fmt"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/app"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/tui/style"
)

// FlattenAction flattens the stack by moving branches as high up the stack as possible
func FlattenAction(ctx *app.Context, branchName string) error {
	eng := ctx.Engine
	out := ctx.Output

	// 1. Resolve branch
	branch := eng.GetBranch(branchName)
	
	// 2. Build stack graph to get all related branches
	// We want the whole connected stack component (parents and children)
	graph := engine.BuildStackGraph(eng, engine.SortStrategyAlphabetical, nil)
	branches := graph.Range(branch, engine.StackRange{
		RecursiveParents:  true,
		RecursiveChildren: true,
		IncludeCurrent:    true,
	})

	if len(branches) == 0 {
		out.Info("No branches to flatten.")
		return nil
	}

	// 3. Sort topologically (Base -> Top)
	sortedBranches := eng.SortBranchesTopologically(branches)
	
	// Filter out trunk and ensure we only have feature branches
	var featureBranches []engine.Branch
	trunk := eng.Trunk()
	for _, b := range sortedBranches {
		if !b.IsTrunk() && b.GetName() != trunk.GetName() {
			featureBranches = append(featureBranches, b)
		}
	}

	if len(featureBranches) == 0 {
		out.Info("No feature branches to flatten.")
		return nil
	}

	ctx.Logger.Info("flatten started", "branchCount", len(featureBranches))
	out.Info("Flattening stack with %d branches...", len(featureBranches))

	// Take snapshot for undo safety
	snapshotOpts := actions.NewSnapshot("flatten",
		actions.WithArg(branchName),
	)
	if err := eng.TakeSnapshot(snapshotOpts); err != nil {
		out.Debug("Failed to take snapshot: %v", err)
	}

	// 4. Algorithm: Bottom-Up
	// potentialParents starts with Trunk.
	// We use this list to check where a branch can land.
	potentialParents := []string{trunk.GetName()}
	
	// Track current SHAs of all branches (including Trunk) to handle moves.
	parentSHAs := make(map[string]string)
	
	trunkSHA, err := trunk.GetRevision()
	if err != nil {
		return fmt.Errorf("failed to get trunk revision: %w", err)
	}
	parentSHAs[trunk.GetName()] = trunkSHA

	for _, b := range featureBranches {
		bName := b.GetName()
		
		// Identify original parent
		origParent := b.GetParent()
		var origParentName string
		if origParent == nil {
			origParentName = trunk.GetName()
		} else {
			origParentName = origParent.GetName()
		}

		// Find base SHA of the branch relative to its original parent
		// We use MergeBase to find the divergence point, ensuring we rebase the entire branch content
		baseSHA, err := eng.Git().GetMergeBase(bName, origParentName)
		if err != nil {
			return fmt.Errorf("failed to get merge base for %s and %s: %w", bName, origParentName, err)
		}

		moved := false
		var finalParent string

		for _, pName := range potentialParents {
			targetSHA, ok := parentSHAs[pName]
			if !ok {
				continue
			}

			// Optimization: if we are already on this parent (SHA match)
			if targetSHA == baseSHA {
				moved = true
				finalParent = pName
				
				// Update metadata to reflect new parent
				parentBranch := eng.GetBranch(finalParent)
				if err := eng.SetParent(ctx.Context, b, parentBranch); err != nil {
					return fmt.Errorf("failed to update parent for %s to %s: %w", bName, finalParent, err)
				}
				break
			}

			// Try to rebase
			// rebase --onto targetSHA baseSHA bName
			// If successful, b moves to targetSHA.
			res, err := eng.Rebase(ctx.Context, bName, targetSHA, baseSHA)
			if err != nil {
				// Rebase failed (likely conflict or other git error).
				// We assume Rebase aborts on failure.
				// Log and continue to next potential parent.
				ctx.Logger.Debug("Rebase check failed", "branch", bName, "target", pName, "error", err)
				continue
			}

			if res == engine.RestackDone || res == engine.RestackUnneeded {
				moved = true
				finalParent = pName
				
				// Update metadata to reflect new parent
				parentBranch := eng.GetBranch(finalParent)
				if err := eng.SetParent(ctx.Context, b, parentBranch); err != nil {
					return fmt.Errorf("failed to update parent for %s to %s: %w", bName, finalParent, err)
				}
				break
			} else {
				// Abort the speculative rebase so we can try the next parent
				if err := eng.Git().RebaseAbort(ctx.Context); err != nil {
					ctx.Logger.Debug("Failed to abort rebase", "error", err)
				}
			}
		}

		if !moved {
			finalParent = origParentName
			
			// Check if original parent has moved
			currentOrigParentSHA, ok := parentSHAs[origParentName]
			if !ok {
				// This implies origParentName was not in potentialParents?
				// But potentialParents accumulates all processed branches + Trunk.
				// Since we process topologically, parent *must* have been processed or is Trunk.
				return fmt.Errorf("original parent %s not tracked", origParentName)
			}
			
			if currentOrigParentSHA != baseSHA {
				// Parent moved, we must rebase to follow
				res, err := eng.Rebase(ctx.Context, bName, currentOrigParentSHA, baseSHA)
				if err != nil {
					return fmt.Errorf("failed to update %s onto moved parent %s: %w", bName, origParentName, err)
				}
				if res == engine.RestackConflict {
					// Real conflict in stack structure
					out.Error("Conflict updating %s onto parent %s.", bName, origParentName)
					return fmt.Errorf("conflict flattening %s", bName)
				}
			}
		}
		
		// Update SHA for this branch (it might have changed)
		newSHA, err := eng.Git().GetRevision(bName)
		if err != nil {
			return fmt.Errorf("failed to get revision for %s: %w", bName, err)
		}
		parentSHAs[bName] = newSHA
		
		// This branch is now a valid parent for subsequent branches
		potentialParents = append(potentialParents, bName)
		
		out.Info("  %s -> %s", style.ColorBranchName(bName, false), style.ColorBranchName(finalParent, false))
	}

	out.Info("\nFlatten complete.")
	return nil
}
