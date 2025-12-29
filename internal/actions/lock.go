package actions

import (
	"fmt"

	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/internal/tui/style"
)

// LockAction locks the specified branch and all branches downstack of it
func LockAction(ctx *runtime.Context, branchName string) error {
	eng := ctx.Engine
	splog := ctx.Splog

	branch := eng.GetBranch(branchName)
	if branch.IsTrunk() {
		return fmt.Errorf("cannot lock trunk branch %s", branchName)
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
		if err := eng.SetLocked(b, true); err != nil {
			return fmt.Errorf("failed to lock branch %s: %w", b.GetName(), err)
		}
		splog.Info("Locked %s.", style.ColorBranchName(b.GetName(), b.GetName() == branchName))
	}

	return nil
}

// UnlockAction unlocks the specified branch and all branches upstack of it
func UnlockAction(ctx *runtime.Context, branchName string) error {
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
		if err := eng.SetLocked(b, false); err != nil {
			return fmt.Errorf("failed to unlock branch %s: %w", b.GetName(), err)
		}
		splog.Info("Unlocked %s.", style.ColorBranchName(b.GetName(), b.GetName() == branchName))
	}

	return nil
}
