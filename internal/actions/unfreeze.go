package actions

import (
	"fmt"

	"stackit.dev/stackit/internal/app"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/tui/style"
)

// UnfreezeAction unfreezes the specified branch and all branches upstack of it (recursive children)
func UnfreezeAction(ctx *app.Context, branchName string) error {
	eng := ctx.Engine
	splog := ctx.Splog

	branch := eng.GetBranch(branchName)
	if !branch.IsTracked() {
		return fmt.Errorf("branch %s is not tracked by stackit", branchName)
	}

	// Get upstack (descendants including current)
	branches := branch.GetRelativeStack(engine.StackRange{
		IncludeCurrent:    true,
		RecursiveChildren: true,
	})

	for _, b := range branches {
		if b.IsTrunk() {
			continue
		}
		if err := eng.SetFrozen(b, false); err != nil {
			return fmt.Errorf("failed to unfreeze branch %s: %w", b.GetName(), err)
		}
		splog.Info("Unfrozen %s locally.", style.ColorBranchName(b.GetName(), b.GetName() == branchName))
	}

	return nil
}
