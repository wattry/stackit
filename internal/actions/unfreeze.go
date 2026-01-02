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

	branchesToUnfreeze := []engine.Branch{}
	for _, b := range branches {
		if b.IsTrunk() {
			continue
		}
		branchesToUnfreeze = append(branchesToUnfreeze, b)
	}

	if len(branchesToUnfreeze) > 0 {
		res, err := eng.SetFrozen(branchesToUnfreeze, false)
		if err != nil {
			for name, branchErr := range res.Errors {
				splog.Warn("Failed to unfreeze %s: %v", name, branchErr)
			}
			return fmt.Errorf("failed to unfreeze branches: %w", err)
		}

		for _, name := range res.AffectedBranches {
			splog.Info("Unfrozen %s locally.", style.ColorBranchName(name, name == branchName))
		}
	}

	return nil
}
