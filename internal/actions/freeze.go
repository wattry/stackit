package actions

import (
	"fmt"

	"stackit.dev/stackit/internal/app"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/tui/style"
)

// FreezeAction freezes the specified branch and all branches downstack of it (recursive parents)
func FreezeAction(ctx *app.Context, branchName string) error {
	eng := ctx.Engine
	splog := ctx.Splog

	branch := eng.GetBranch(branchName)
	if branch.IsTrunk() {
		return fmt.Errorf("cannot freeze trunk branch %s", branchName)
	}

	if !branch.IsTracked() {
		return fmt.Errorf("branch %s is not tracked by stackit", branchName)
	}

	// Get downstack (ancestors including current)
	branches := branch.GetRelativeStack(engine.StackRange{
		RecursiveParents: true,
		IncludeCurrent:   true,
	})

	for _, b := range branches {
		if b.IsTrunk() {
			continue
		}
		if err := eng.SetFrozen(b, true); err != nil {
			return fmt.Errorf("failed to freeze branch %s: %w", b.GetName(), err)
		}
		splog.Info("Frozen %s locally.", style.ColorBranchName(b.GetName(), b.GetName() == branchName))
	}

	return nil
}
