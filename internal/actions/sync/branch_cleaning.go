package sync

import (
	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/app"
)

// cleanBranches handles cleaning merged/closed branches
func cleanBranches(ctx *app.Context, opts *Options, handler Handler, _ *Summary) (*actions.CleanBranchesResult, error) {
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

	// The CleanBranches function already logs deletions via splog
	// For now, we just update the summary with reparent info
	// TODO: Enhance CleanBranches to return detailed deletion info for events

	return result, nil
}
