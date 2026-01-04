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
	out := ctx.Output

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

	branchesToFreeze := []engine.Branch{}
	for _, b := range branches {
		if b.IsTrunk() {
			continue
		}
		branchesToFreeze = append(branchesToFreeze, b)
	}

	if len(branchesToFreeze) > 0 {
		res, err := eng.SetFrozen(branchesToFreeze, true)
		if err != nil {
			for name, branchErr := range res.Errors {
				out.Warn("Failed to freeze %s: %v", name, branchErr)
			}
			return fmt.Errorf("failed to freeze branches: %w", err)
		}

		for _, name := range res.AffectedBranches {
			out.Info("Frozen %s locally.", style.ColorBranchName(name, name == branchName))
		}
	}

	return nil
}
