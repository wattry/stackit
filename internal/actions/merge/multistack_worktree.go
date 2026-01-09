package merge

import (
	"context"
	"fmt"

	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/output"
)

// MultiStackWorktreeResult contains the result of merging stacks in a worktree
type MultiStackWorktreeResult struct {
	MergedStacks   []MultiStackInfo     // Stacks that were successfully merged
	ConflictStacks []MultiStackExcluded // Stacks that conflicted
	WorktreePath   string               // Path to the worktree
	WorktreeEngine engine.Engine        // Engine for the worktree
	Cleanup        func()               // Function to clean up the worktree
}

// MultiStackWorktreeExecutor handles merging stacks in a worktree
type MultiStackWorktreeExecutor struct {
	eng    engine.Engine
	output output.Output
}

// NewMultiStackWorktreeExecutor creates a new worktree executor for multi-stack merge
func NewMultiStackWorktreeExecutor(eng engine.Engine, out output.Output) *MultiStackWorktreeExecutor {
	return &MultiStackWorktreeExecutor{
		eng:    eng,
		output: out,
	}
}

// ExecuteInWorktree creates a worktree at trunk and attempts to merge all stacks.
// For each stack, it merges all branches in order. If any branch in a stack
// conflicts, the entire stack is skipped.
func (w *MultiStackWorktreeExecutor) ExecuteInWorktree(ctx context.Context, stacks []MultiStackInfo) (*MultiStackWorktreeResult, error) {
	trunk := w.eng.Trunk()

	// Create temporary worktree at trunk
	worktreePath, cleanup, err := w.eng.CreateTemporaryWorktree(ctx, trunk.GetName(), "stackit-multistack-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create worktree: %w", err)
	}

	w.output.Debug("Created worktree at %s", worktreePath)

	// Create engine for worktree
	worktreeEng, err := engine.NewEngine(engine.Options{
		RepoRoot:          worktreePath,
		Trunk:             trunk.GetName(),
		MaxUndoStackDepth: 0, // No undo needed for multi-stack
	})
	if err != nil {
		cleanup()
		return nil, fmt.Errorf("failed to initialize worktree engine: %w", err)
	}

	// Ensure worktree trunk is up to date before merging stacks
	if err := worktreeEng.PopulateRemoteShas(); err != nil {
		w.output.Debug("Failed to populate remote SHAs in multistack worktree: %v", err)
	}
	pullResult, err := worktreeEng.PullTrunk(ctx)
	if err != nil {
		cleanup()
		return nil, fmt.Errorf("failed to update trunk in worktree: %w", err)
	}
	if pullResult == engine.PullConflict {
		cleanup()
		return nil, fmt.Errorf("trunk could not be fast-forwarded (diverged from remote). Please sync trunk before running multi-stack merge")
	}
	// Ensure worktree is checked out at the updated trunk tip
	worktreeTrunk := worktreeEng.GetBranch(trunk.GetName())
	if err := worktreeEng.CheckoutBranch(ctx, worktreeTrunk); err != nil {
		cleanup()
		return nil, fmt.Errorf("failed to checkout trunk in worktree: %w", err)
	}

	result := &MultiStackWorktreeResult{
		MergedStacks:   make([]MultiStackInfo, 0),
		ConflictStacks: make([]MultiStackExcluded, 0),
		WorktreePath:   worktreePath,
		WorktreeEngine: worktreeEng,
		Cleanup:        cleanup,
	}

	// Try to merge each stack
	for _, stack := range stacks {
		baseline, err := worktreeEng.GetCurrentRevision(ctx)
		if err != nil {
			cleanup()
			return nil, fmt.Errorf("failed to capture worktree state: %w", err)
		}

		if err := w.tryMergeStack(ctx, worktreeEng, stack); err != nil {
			w.output.Debug("Stack %s conflicts: %v", stack.RootBranch, err)
			if resetErr := worktreeEng.ResetHard(ctx, baseline); resetErr != nil {
				w.output.Debug("Failed to reset worktree after conflict: %v", resetErr)
			}
			result.ConflictStacks = append(result.ConflictStacks, MultiStackExcluded{
				Stack:  stack,
				Reason: "conflict",
			})
		} else {
			w.output.Debug("Stack %s merged successfully", stack.RootBranch)
			result.MergedStacks = append(result.MergedStacks, stack)
		}
	}

	return result, nil
}

// tryMergeStack attempts to merge all branches of a stack via octopus merge.
// Returns error if any branch conflicts (entire stack is skipped on conflict).
func (w *MultiStackWorktreeExecutor) tryMergeStack(ctx context.Context, eng engine.Engine, stack MultiStackInfo) error {
	// Create a merge commit message
	msg := fmt.Sprintf("Merge stack %s for multi-stack (%d branches)", stack.RootBranch, len(stack.AllBranches))

	// Perform octopus merge (single merge commit with multiple parents)
	err := eng.MergeMultiple(ctx, stack.AllBranches, engine.MergeOptions{
		NoFF:    true,
		NoEdit:  true,
		Message: msg,
	})
	if err != nil {
		// Abort the merge if it's in progress
		git := eng.Git()
		if git.IsMergeInProgress(ctx) {
			if abortErr := git.MergeAbort(ctx); abortErr != nil {
				w.output.Debug("Failed to abort merge: %v", abortErr)
			}
		}
		return fmt.Errorf("conflict in stack %s: %w", stack.RootBranch, err)
	}
	return nil
}

// ResetToTrunk resets the worktree to trunk, discarding all merges.
// This is used by binary search to try different combinations.
func (w *MultiStackWorktreeExecutor) ResetToTrunk(ctx context.Context, eng engine.Engine) error {
	trunk := w.eng.Trunk()
	return eng.ResetHard(ctx, trunk.GetName())
}
