package sync

import (
	"fmt"
	"time"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/app"
	"stackit.dev/stackit/internal/engine"
)

// restackBranches handles restacking branches after sync operations.
// When restackScope is non-nil, only those branches are restacked (skipping current-branch expansion).
// When expandScope is true, expands to the full current stack (used when --restack is explicitly passed).
func restackBranches(ctx *app.Context, branchesToRestack []string, restackScope []string, expandScope bool, dirtyAnchors map[string]bool, handler Handler, summary *Summary) error {
	nav := ctx.Navigator()

	if restackScope != nil {
		// Scoped restack: use only the explicitly provided branches
		branchesToRestack = append(branchesToRestack, restackScope...)
	} else if expandScope {
		// Explicit --restack: expand from current branch position to full stack
		graph := engine.BuildStackGraph(ctx.Engine, engine.SortStrategyAlphabetical, nil)

		currentBranch := nav.CurrentBranch()
		if currentBranch != nil {
			if currentBranch.IsTracked() {
				// Get full stack (up to trunk)
				stack := graph.Range(*currentBranch, engine.StackRange{
					RecursiveParents:  true,
					IncludeCurrent:    true,
					RecursiveChildren: true,
				})
				// Add branches to restack list
				for _, b := range stack {
					branchesToRestack = append(branchesToRestack, b.GetName())
				}
			} else if currentBranch.IsTrunk() {
				// If on trunk, restack all branches
				stack := graph.Range(*currentBranch, engine.StackRange{
					RecursiveChildren: true,
				})
				for _, b := range stack {
					branchesToRestack = append(branchesToRestack, b.GetName())
				}
			}
		}
	}
	// When expandScope is false and restackScope is nil, only restack
	// branches already in branchesToRestack (reparented branches from sync)

	// Remove duplicates and filter out non-existent/untracked branches and dirty stacks
	seen := make(map[string]bool)
	uniqueBranches := []engine.Branch{}
	for _, branchName := range branchesToRestack {
		if !seen[branchName] && !isInDirtyStack(ctx, branchName, dirtyAnchors) {
			seen[branchName] = true
			branch := nav.GetBranch(branchName)
			// Only include branches that exist, are tracked, and are not trunks
			if branch.IsTracked() && !branch.IsTrunk() {
				uniqueBranches = append(uniqueBranches, branch)
			}
		}
	}

	// Sort branches topologically (parents before children) for correct restack order
	sortedBranches := nav.SortBranchesTopologically(uniqueBranches)

	// Restack branches with handler for progress
	if len(sortedBranches) > 0 {
		restackStart := time.Now()
		if err := actions.RestackBranchesWithHandler(ctx, sortedBranches, func(branchName string, result engine.RestackResult, newRev string, _ bool, lockReason engine.LockReason, frozen bool, isCurrent bool, _ bool, _, _ string) {
			prNumber := getPRNumber(ctx.Status(), branchName)

			parentName := ""
			br := nav.GetBranch(branchName)
			if br.GetName() != "" {
				if p := br.GetParent(); p != nil {
					parentName = p.GetName()
				} else {
					parentName = ctx.Engine.Trunk().GetName()
				}
			}

			switch result {
			case engine.RestackDone:
				summary.BranchesRestacked++
				handler.EmitEvent(Event{
					Phase:       PhaseRestack,
					Type:        EventCompleted,
					Branch:      branchName,
					PRNumber:    prNumber,
					NewRevision: newRev,
					LockReason:  lockReason,
					Frozen:      frozen,
					IsCurrent:   isCurrent,
					Parent:      parentName,
				})
			case engine.RestackUnneeded:
				handler.EmitEvent(Event{
					Phase:      PhaseRestack,
					Type:       EventCompleted,
					Branch:     branchName,
					PRNumber:   prNumber,
					LockReason: lockReason,
					Frozen:     frozen,
					IsCurrent:  isCurrent,
					Parent:     parentName,
				})
			case engine.RestackConflict:
				summary.BranchesSkipped++
				summary.ConflictBranches = append(summary.ConflictBranches, branchName)
				handler.EmitEvent(Event{
					Phase:      PhaseRestack,
					Type:       EventSkipped,
					Branch:     branchName,
					PRNumber:   prNumber,
					Conflict:   true,
					LockReason: lockReason,
					Frozen:     frozen,
					IsCurrent:  isCurrent,
					Parent:     parentName,
				})
			}
		}, false); err != nil {
			return fmt.Errorf("failed to restack branches: %w", err)
		}
		ctx.Logger.Info("restack branches completed durationMs=%d branchCount=%d", time.Since(restackStart).Milliseconds(), len(sortedBranches))
	}

	return nil
}

// getPRNumber returns the PR number for a branch if it has one
func getPRNumber(eng engine.BranchStatus, branchName string) *int {
	branch := eng.GetBranch(branchName)
	prInfo, err := eng.GetPrInfo(branch)
	if err != nil || prInfo == nil {
		return nil
	}
	return prInfo.Number()
}
