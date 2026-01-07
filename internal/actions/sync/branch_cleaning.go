package sync

import (
	"time"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/app"
	"stackit.dev/stackit/internal/tui/style"
)

// cleanBranches handles cleaning merged/closed branches
func cleanBranches(ctx *app.Context, opts *Options, handler Handler, summary *Summary) (*actions.CleanBranchesResult, error) {
	// Only emit phase start if we have branches that might need cleaning
	allBranches := ctx.Engine.AllBranches()
	hasBranchesToCheck := false
	for _, b := range allBranches {
		if !b.IsTrunk() {
			hasBranchesToCheck = true
			break
		}
	}

	if hasBranchesToCheck {
		handler.EmitEvent(Event{Phase: PhaseClean, Type: EventStarted})
	}

	// Get current branch name for worktree detection
	currentBranchName := ""
	if currentBranch := ctx.Engine.CurrentBranch(); currentBranch != nil {
		currentBranchName = currentBranch.GetName()
	}

	cleanStart := time.Now()
	result, err := actions.CleanBranches(ctx, actions.CleanBranchesOptions{
		Force:             opts.Force,
		InManagedWorktree: ctx.InManagedWorktree,
		CurrentBranch:     currentBranchName,
	})
	ctx.Logger.Info("clean branches completed durationMs=%d deletedCount=%d", time.Since(cleanStart).Milliseconds(), len(result.DeletedBranches))

	if err != nil {
		return result, err
	}

	// Emit events for each deleted branch
	for name, reason := range result.DeletedBranches {
		handler.EmitEvent(Event{
			Phase:   PhaseClean,
			Type:    EventCompleted,
			Branch:  name,
			Message: reason,
		})
		summary.BranchesDeleted++
	}

	// Warn about branches that couldn't be deleted from worktree
	for _, name := range result.SkippedInWorktree {
		ctx.Output.Warn("Cannot delete %s from worktree. Run sync from the main repository to clean up.",
			style.ColorBranchName(name, true))
	}

	return result, nil
}
