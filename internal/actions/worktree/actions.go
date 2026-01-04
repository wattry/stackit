// Package worktree provides actions for managing stackit-managed worktrees.
package worktree

import (
	"fmt"
	"os"

	"stackit.dev/stackit/internal/app"
	"stackit.dev/stackit/internal/tui/style"
)

// ListOptions contains options for the list action
type ListOptions struct {
	// No options for now
}

// ListResult contains the results of listing worktrees
type ListResult struct {
	Worktrees []Entry
}

// Entry represents a single managed worktree
type Entry struct {
	StackRoot string
	Path      string
	Exists    bool // Whether the path still exists on disk
}

// ListAction lists all managed worktrees
func ListAction(ctx *app.Context, _ ListOptions) (*ListResult, error) {
	worktrees, err := ctx.Engine.ListManagedWorktrees()
	if err != nil {
		return nil, fmt.Errorf("failed to list managed worktrees: %w", err)
	}

	result := &ListResult{
		Worktrees: make([]Entry, 0, len(worktrees)),
	}

	for _, wt := range worktrees {
		// Check if path exists
		exists := true
		if _, statErr := os.Stat(wt.Path); os.IsNotExist(statErr) {
			exists = false
		}

		result.Worktrees = append(result.Worktrees, Entry{
			StackRoot: wt.StackRoot,
			Path:      wt.Path,
			Exists:    exists,
		})
	}

	return result, nil
}

// RemoveOptions contains options for the remove action
type RemoveOptions struct {
	StackRoot string // Stack root name to remove worktree for
	Force     bool   // Force removal even if worktree has uncommitted changes
}

// RemoveAction removes a worktree for a stack
func RemoveAction(ctx *app.Context, opts RemoveOptions) error {
	out := ctx.Output

	// Get worktree info
	wtInfo, err := ctx.Engine.GetWorktreeForStack(opts.StackRoot)
	if err != nil {
		return fmt.Errorf("failed to get worktree info: %w", err)
	}

	if wtInfo == nil {
		return fmt.Errorf("no worktree found for stack %s", style.ColorBranchName(opts.StackRoot, false))
	}

	// Check if path exists before trying to remove
	if _, statErr := os.Stat(wtInfo.Path); statErr == nil {
		// Try to remove the git worktree
		if removeErr := ctx.Engine.RemoveWorktree(ctx.Context, wtInfo.Path); removeErr != nil {
			if !opts.Force {
				return fmt.Errorf("failed to remove worktree at %s: %w (use --force to override)", wtInfo.Path, removeErr)
			}
			out.Warn("Failed to remove worktree directory, continuing with unregistration: %v", removeErr)
		}
	} else {
		out.Debug("Worktree path %s does not exist, skipping removal", wtInfo.Path)
	}

	// Unregister from registry
	if unregErr := ctx.Engine.UnregisterWorktree(opts.StackRoot); unregErr != nil {
		return fmt.Errorf("failed to unregister worktree: %w", unregErr)
	}

	out.Success("Removed worktree for stack %s", style.ColorBranchName(opts.StackRoot, false))
	return nil
}

// OpenOptions contains options for the open action
type OpenOptions struct {
	StackRoot string // Stack root name to get path for
}

// OpenAction returns the path to a worktree for a stack
func OpenAction(ctx *app.Context, opts OpenOptions) (string, error) {
	wtInfo, err := ctx.Engine.GetWorktreeForStack(opts.StackRoot)
	if err != nil {
		return "", fmt.Errorf("failed to get worktree info: %w", err)
	}

	if wtInfo == nil {
		return "", fmt.Errorf("no worktree found for stack %s", opts.StackRoot)
	}

	// Check if path exists
	if _, statErr := os.Stat(wtInfo.Path); os.IsNotExist(statErr) {
		return "", fmt.Errorf("worktree path %s does not exist (worktree may have been manually deleted)", wtInfo.Path)
	}

	return wtInfo.Path, nil
}
