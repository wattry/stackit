package sync

import (
	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/app"
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

	result, err := actions.CleanBranches(ctx, actions.CleanBranchesOptions{
		Force: opts.Force,
	})

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

	return result, nil
}
