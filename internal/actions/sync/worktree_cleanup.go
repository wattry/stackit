package sync

import (
	"os"

	"stackit.dev/stackit/internal/app"
	"stackit.dev/stackit/internal/config"
)

// WorktreeCleanupResult contains the results of worktree cleanup
type WorktreeCleanupResult struct {
	RemovedWorktrees []string // Stack roots whose worktrees were removed
	Errors           []string // Any errors encountered (non-fatal)
}

// cleanOrphanedWorktrees removes worktrees for stacks that no longer exist.
// This is called during sync after branch cleanup to remove worktrees for merged/deleted stacks.
// This function is best-effort and will not fail sync on errors.
func cleanOrphanedWorktrees(ctx *app.Context) *WorktreeCleanupResult {
	result := &WorktreeCleanupResult{
		RemovedWorktrees: []string{},
		Errors:           []string{},
	}

	// Check if auto-clean is enabled
	cfg, err := config.LoadConfig(ctx.RepoRoot)
	if err != nil {
		// If config can't be loaded, skip cleanup but don't fail
		ctx.Output.Debug("Failed to load config for worktree cleanup: %v", err)
		return result
	}

	if !cfg.WorktreeAutoClean() {
		ctx.Output.Debug("Worktree auto-clean is disabled")
		return result
	}

	// Get all managed worktrees
	worktrees, err := ctx.Engine.ListManagedWorktrees()
	if err != nil {
		ctx.Output.Debug("Failed to list managed worktrees: %v", err)
		return result
	}

	if len(worktrees) == 0 {
		return result
	}

	// Check each worktree to see if its stack root still exists
	for _, wt := range worktrees {
		stackRootBranch := ctx.Engine.GetBranch(wt.StackRoot)

		// Check if the stack root branch still exists and is tracked
		// A branch "exists" if it's in the branch list (not just tracked)
		branchExists := false
		for _, b := range ctx.Engine.AllBranches() {
			if b.GetName() == wt.StackRoot {
				branchExists = true
				break
			}
		}

		if !branchExists || (!stackRootBranch.IsTrunk() && !stackRootBranch.IsTracked()) {
			// Stack root no longer exists or is not tracked - clean up worktree
			ctx.Output.Info("Removing worktree for deleted stack %s", wt.StackRoot)

			// Check if worktree path still exists before trying to remove
			if _, statErr := os.Stat(wt.Path); statErr == nil {
				// Path exists, try to remove the worktree
				if removeErr := ctx.Engine.RemoveWorktree(ctx.Context, wt.Path); removeErr != nil {
					result.Errors = append(result.Errors,
						"failed to remove worktree at "+wt.Path+": "+removeErr.Error())
					ctx.Output.Debug("Failed to remove worktree at %s: %v", wt.Path, removeErr)
					// Continue to unregister even if removal fails
				}
			}

			// Unregister the worktree from the registry
			if unregErr := ctx.Engine.UnregisterWorktree(wt.StackRoot); unregErr != nil {
				result.Errors = append(result.Errors,
					"failed to unregister worktree for "+wt.StackRoot+": "+unregErr.Error())
				ctx.Output.Debug("Failed to unregister worktree for %s: %v", wt.StackRoot, unregErr)
			} else {
				result.RemovedWorktrees = append(result.RemovedWorktrees, wt.StackRoot)
			}
		}
	}

	return result
}
