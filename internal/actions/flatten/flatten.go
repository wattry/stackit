// Package flatten provides functionality for flattening stacked branches closer to trunk.
package flatten

import (
	"fmt"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/app"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/tui/style"
)

// Options contains options for the flatten command
type Options struct {
	BranchName  string // Branch to start flattening from (defaults to current branch)
	SkipConfirm bool   // Skip confirmation prompt (--yes flag)
}

// Action performs the flatten operation.
// Flatten analyzes the stack and moves branches as close to trunk as possible
// while respecting dependencies (branches that would conflict stay in place).
func Action(ctx *app.Context, opts Options, handler Handler) error {
	eng := ctx.Engine
	out := ctx.Output
	gctx := ctx.Context

	// Use null handler if none provided
	if handler == nil {
		handler = &NullHandler{}
	}
	defer handler.Cleanup()

	// Resolve branch name
	branchName := opts.BranchName
	if branchName == "" {
		currentBranch := eng.CurrentBranch()
		if currentBranch == nil {
			return fmt.Errorf("not on a branch and no branch specified")
		}
		branchName = currentBranch.GetName()
	}

	// Validate branch exists and is tracked
	branch := eng.GetBranch(branchName)
	if !branch.IsTracked() && !branch.IsTrunk() {
		return fmt.Errorf("branch %q is not tracked by Stackit", branchName)
	}

	// Build stack graph to get all related branches
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

	// Sort topologically (parents before children)
	sortedBranches := eng.SortBranchesTopologically(branches)

	// Filter out trunk - we only flatten feature branches
	trunk := eng.Trunk()
	var featureBranches []engine.Branch
	for _, b := range sortedBranches {
		if !b.IsTrunk() && b.GetName() != trunk.GetName() {
			featureBranches = append(featureBranches, b)
		}
	}

	if len(featureBranches) == 0 {
		out.Info("No feature branches to flatten.")
		return nil
	}

	handler.Start(len(featureBranches))
	handler.OnStep(StepAnalyzing, StatusStarted, "Analyzing stack structure...")

	// Take snapshot before any modifications
	snapshotOpts := actions.NewSnapshot("flatten",
		actions.WithArg(branchName),
	)
	if err := eng.TakeSnapshot(snapshotOpts); err != nil {
		out.Debug("Failed to take snapshot: %v", err)
	}

	// Build the flatten plan by testing which branches can move closer to trunk
	plan, err := buildFlattenPlan(ctx, eng, featureBranches, trunk)
	if err != nil {
		handler.OnStep(StepAnalyzing, StatusFailed, err.Error())
		return fmt.Errorf("failed to build flatten plan: %w", err)
	}

	handler.OnStep(StepAnalyzing, StatusCompleted, fmt.Sprintf("Found %d branches to move", len(plan.Moves)))

	// If nothing to move, we're done
	if len(plan.Moves) == 0 {
		out.Info("All branches are already as close to trunk as possible.")
		handler.Complete(Result{
			MovedCount:     0,
			UnchangedCount: len(featureBranches),
		})
		return nil
	}

	// Validate all planned moves upfront
	handler.OnStep(StepValidating, StatusStarted, "Validating moves...")
	validation, err := eng.ValidateRebases(gctx, plan.RebaseSpecs)
	if err != nil {
		handler.OnStep(StepValidating, StatusFailed, err.Error())
		return fmt.Errorf("failed to validate rebases: %w", err)
	}

	// Build preview with conflict info
	preview := Preview{
		Moves:          plan.Moves,
		UnchangedCount: plan.UnchangedCount,
	}

	if !validation.Success {
		preview.HasConflicts = true
		preview.ConflictBranch = validation.FailedBranch
		preview.ConflictError = validation.ErrorMessage
		handler.OnStep(StepValidating, StatusFailed, fmt.Sprintf("Conflicts detected on %s", validation.FailedBranch))
	} else {
		handler.OnStep(StepValidating, StatusCompleted, "All moves validated successfully")
	}

	// Prompt for confirmation (if interactive and not skipping)
	if handler.IsInteractive() && !opts.SkipConfirm {
		confirmed, promptErr := handler.PromptConfirmFlatten(preview)
		if promptErr != nil {
			return fmt.Errorf("failed to prompt for confirmation: %w", promptErr)
		}
		if !confirmed {
			out.Info("Flatten canceled.")
			return nil
		}

		// If there are conflicts, return error after user has seen preview
		if preview.HasConflicts {
			return fmt.Errorf("flatten would cause conflicts: %s on branch %s", preview.ConflictError, preview.ConflictBranch)
		}
	} else if preview.HasConflicts {
		// Non-interactive mode: fail immediately on conflicts
		return fmt.Errorf("flatten would cause conflicts: %s on branch %s", preview.ConflictError, preview.ConflictBranch)
	}

	// Execute the flatten
	handler.OnStep(StepFlattening, StatusStarted, "Moving branches...")

	// Build a map of branch -> oldUpstream from the rebase specs
	// This is needed because SetParent may calculate a different value using merge-base,
	// but we need to preserve the correct divergence point for proper rebasing
	oldUpstreamMap := make(map[string]string)
	for _, spec := range plan.RebaseSpecs {
		oldUpstreamMap[spec.Branch] = spec.OldUpstream
	}

	// Update parent pointers for all planned moves
	for _, move := range plan.Moves {
		moveBranch := eng.GetBranch(move.Branch)
		newParentBranch := eng.GetBranch(move.NewParent)

		if err := eng.SetParent(gctx, moveBranch, newParentBranch); err != nil {
			handler.OnStep(StepFlattening, StatusFailed, err.Error())
			return fmt.Errorf("failed to set parent for %s to %s: %w", move.Branch, move.NewParent, err)
		}

		// Update parent revision with the correct oldUpstream we calculated earlier.
		// SetParent uses merge-base which may be incorrect when flattening branches
		// that have diverged from their original parent.
		if oldUpstream, ok := oldUpstreamMap[move.Branch]; ok {
			if err := eng.UpdateParentRevision(move.Branch, oldUpstream); err != nil {
				out.Debug("Failed to update parent revision for %s: %v", move.Branch, err)
			}
		}

		handler.OnBranchMoved(move.Branch, move.OldParent, move.NewParent)
		out.Info("  %s: %s -> %s",
			style.ColorBranchName(move.Branch, false),
			style.ColorDim(move.OldParent),
			style.ColorBranchName(move.NewParent, false))
	}

	handler.OnStep(StepFlattening, StatusCompleted, "Parent pointers updated")

	// Restack all affected branches
	handler.OnStep(StepRestacking, StatusStarted, "Restacking branches...")

	// Rebuild graph after parent changes
	graph = engine.BuildStackGraph(eng, engine.SortStrategyAlphabetical, nil)

	// Collect all branches that need restacking (moved branches and their descendants)
	branchesToRestack := make([]engine.Branch, 0)
	for _, move := range plan.Moves {
		moveBranch := eng.GetBranch(move.Branch)
		descendants := graph.Range(moveBranch, engine.StackRange{
			RecursiveChildren: true,
			IncludeCurrent:    true,
			RecursiveParents:  false,
		})
		branchesToRestack = append(branchesToRestack, descendants...)
	}

	if err := actions.RestackBranches(ctx, branchesToRestack); err != nil {
		handler.OnStep(StepRestacking, StatusFailed, err.Error())
		return fmt.Errorf("failed to restack branches: %w", err)
	}

	handler.OnStep(StepRestacking, StatusCompleted, "Branches restacked")

	out.Info("\nFlatten complete: %d branches moved, %d unchanged.", len(plan.Moves), plan.UnchangedCount)

	handler.Complete(Result{
		MovedCount:     len(plan.Moves),
		UnchangedCount: plan.UnchangedCount,
	})

	return nil
}

// flattenPlan contains the calculated flatten operations
type flattenPlan struct {
	Moves          []PlannedMove       // Branches that will be moved
	UnchangedCount int                 // Number of branches that won't change
	RebaseSpecs    []engine.RebaseSpec // Specs for validating all moves
}

// buildFlattenPlan calculates which branches can be moved closer to trunk.
// For each branch (in topological order), it tests if the branch can rebase
// onto trunk or any intermediate branch that's closer to trunk.
func buildFlattenPlan(ctx *app.Context, eng engine.Engine, branches []engine.Branch, trunk engine.Branch) (*flattenPlan, error) {
	plan := &flattenPlan{
		Moves:       make([]PlannedMove, 0),
		RebaseSpecs: make([]engine.RebaseSpec, 0),
	}

	// Track the current revision of each potential parent (including trunk)
	// This is needed to build accurate rebase specs
	parentRevisions := make(map[string]string)
	trunkRev, err := trunk.GetRevision()
	if err != nil {
		return nil, fmt.Errorf("failed to get trunk revision: %w", err)
	}
	parentRevisions[trunk.GetName()] = trunkRev

	// potentialParents tracks branches that can serve as parents for subsequent branches
	// Starts with trunk, and grows as we process branches
	potentialParents := []string{trunk.GetName()}

	for _, b := range branches {
		bName := b.GetName()

		// Get current parent info
		origParent := b.GetParent()
		origParentName := trunk.GetName()
		origParentBranch := trunk
		if origParent != nil {
			origParentName = origParent.GetName()
			origParentBranch = *origParent
		}

		// Get the branch's base (divergence point from parent)
		oldUpstream, err := getOldUpstream(eng, bName, origParentBranch)
		if err != nil {
			return nil, fmt.Errorf("failed to get base for %s: %w", bName, err)
		}

		// Try to find the best (closest to trunk) parent this branch can move to
		newParent := findBestParent(ctx, eng, bName, oldUpstream, potentialParents, parentRevisions)

		// Track this branch's revision for subsequent branches
		bRev, err := b.GetRevision()
		if err != nil {
			return nil, fmt.Errorf("failed to get revision for %s: %w", bName, err)
		}
		parentRevisions[bName] = bRev

		// Add this branch as a potential parent for subsequent branches
		potentialParents = append(potentialParents, bName)

		// If we found a better parent, add to the plan
		if newParent != "" && newParent != origParentName {
			plan.Moves = append(plan.Moves, PlannedMove{
				Branch:    bName,
				OldParent: origParentName,
				NewParent: newParent,
			})

			// Add rebase spec for this move
			newParentRev := parentRevisions[newParent]
			plan.RebaseSpecs = append(plan.RebaseSpecs, engine.RebaseSpec{
				Branch:      bName,
				NewParent:   newParentRev,
				OldUpstream: oldUpstream,
			})
		} else {
			plan.UnchangedCount++
		}
	}

	return plan, nil
}

// getOldUpstream returns the divergence point of the branch from its parent.
// This is used as the OldUpstream for rebase operations.
func getOldUpstream(eng engine.Engine, branchName string, parent engine.Branch) (string, error) {
	// First, try to get from metadata
	meta, err := eng.Git().ReadMetadata(branchName)
	if err == nil && meta.ParentBranchRevision != nil && *meta.ParentBranchRevision != "" {
		return *meta.ParentBranchRevision, nil
	}

	// Fall back to parent's current revision (not merge-base, which can include
	// commits from parent branches if the parent was rebased after this branch was created)
	return eng.GetRevision(parent)
}

// findBestParent finds the closest-to-trunk parent that the branch can cleanly rebase onto.
// Returns empty string if the branch should stay with its current parent.
func findBestParent(ctx *app.Context, eng engine.Engine, branchName, oldUpstream string, potentialParents []string, parentRevisions map[string]string) string {
	// Try each potential parent starting from trunk (index 0)
	// and working up through branches that have been processed
	for _, candidateParent := range potentialParents {
		candidateRev, ok := parentRevisions[candidateParent]
		if !ok {
			continue
		}

		// Quick check: if the candidate is already at the same revision as oldUpstream,
		// this branch is already optimally placed relative to this candidate
		if candidateRev == oldUpstream {
			return candidateParent
		}

		// Test if the branch can rebase onto this candidate
		if canRebaseOnto(ctx, eng, branchName, candidateRev, oldUpstream) {
			return candidateParent
		}
	}

	// No better parent found
	return ""
}

// canRebaseOnto tests if a branch can cleanly rebase onto a target.
// Uses ValidateRebases with a single spec to test in a temporary worktree.
func canRebaseOnto(ctx *app.Context, eng engine.Engine, branchName, targetRev, oldUpstream string) bool {
	specs := []engine.RebaseSpec{{
		Branch:      branchName,
		NewParent:   targetRev,
		OldUpstream: oldUpstream,
	}}

	validation, err := eng.ValidateRebases(ctx.Context, specs)
	if err != nil {
		ctx.Logger.Debug("Validation error for %s onto %s: %v", branchName, targetRev, err)
		return false
	}

	return validation.Success
}
