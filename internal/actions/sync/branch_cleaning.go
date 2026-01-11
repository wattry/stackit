package sync

import (
	"sort"
	"strings"
	"time"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/app"
	"stackit.dev/stackit/internal/tui/style"
)

// cleanBranches handles cleaning merged/closed branches
func cleanBranches(ctx *app.Context, opts *Options, dirtyAnchors map[string]bool, handler Handler, summary *Summary) (*actions.CleanBranchesResult, error) {
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

	// Phase 1: Plan which branches to delete
	plan, err := actions.PlanBranchDeletions(ctx, actions.CleanBranchesOptions{
		Force:             opts.Force,
		InManagedWorktree: ctx.InManagedWorktree,
		CurrentBranch:     currentBranchName,
	})
	if err != nil {
		return nil, err
	}

	// Filter out branches in dirty stacks - don't delete them while worktree has uncommitted changes
	for name := range plan.BranchesToDelete {
		if isInDirtyStack(ctx, name, dirtyAnchors) {
			delete(plan.BranchesToDelete, name)
		}
	}

	// Log each branch and its deletion reason
	if len(plan.BranchesToDelete) > 0 {
		// Sort branch names for consistent logging
		names := make([]string, 0, len(plan.BranchesToDelete))
		for name := range plan.BranchesToDelete {
			names = append(names, name)
		}
		sort.Strings(names)

		// Build detailed log message
		var logDetails []string
		for _, name := range names {
			reason := plan.BranchesToDelete[name]
			logDetails = append(logDetails, name+": "+reason)
		}
		ctx.Logger.Info("branches planned for deletion count=%d branches=[%s]",
			len(plan.BranchesToDelete), strings.Join(logDetails, ", "))
	}

	// Phase 2: Get user confirmation for deletions (interactive mode only)
	var branchesToDelete map[string]bool
	if handler.IsInteractive() && len(plan.BranchesToDelete) > 0 {
		confirmed, err := handler.PromptBranchDeletions(plan.BranchesToDelete)
		if err != nil {
			return nil, err
		}
		branchesToDelete = confirmed
	}

	// Phase 3: Execute deletions
	result, err := actions.ExecuteBranchDeletions(ctx, plan, branchesToDelete)
	if err != nil {
		return nil, err
	}

	ctx.Logger.Info("clean branches completed durationMs=%d deletedCount=%d", time.Since(cleanStart).Milliseconds(), len(result.DeletedBranches))

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
