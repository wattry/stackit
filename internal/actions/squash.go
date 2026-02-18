package actions

import (
	"fmt"

	"stackit.dev/stackit/internal/app"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/errors"
	"stackit.dev/stackit/internal/tui/style"
)

// SquashOptions contains options for the squash command
type SquashOptions struct {
	Message string
	NoEdit  bool
}

// SquashAction performs the squash operation
func SquashAction(ctx *app.Context, opts SquashOptions) error {
	eng := ctx.History()
	out := ctx.Output
	context := ctx.Context

	// Get current branch
	currentBranch := ctx.Navigator().CurrentBranch()
	if currentBranch == nil {
		return errors.ErrNotOnBranch
	}

	if err := currentBranch.EnsureCanModify(); err != nil {
		return err
	}

	// Log entry point for diagnostics
	ctx.Logger.Info("squash started branch=%v", currentBranch.GetName())

	// Take snapshot before modifying the repository
	snapshotOpts := NewSnapshot("squash",
		WithFlagValue("-m", opts.Message),
		WithFlag(opts.NoEdit, "--no-edit"),
	)
	if err := ctx.Undo().TakeSnapshot(snapshotOpts); err != nil {
		// Log but don't fail - snapshot is best effort
		out.Debug("Failed to take snapshot: %v", err)
	}

	// Squash current branch
	if err := eng.SquashCurrentBranch(context, engine.SquashOptions{
		Message:  opts.Message,
		NoEdit:   opts.NoEdit,
		NoVerify: !ctx.Verify,
	}); err != nil {
		return fmt.Errorf("failed to squash branch: %w", err)
	}

	out.Info("Squashed commits in %s.", style.ColorBranchName(currentBranch.GetName(), true))
	ctx.Logger.Info("squash completed branch=%v", currentBranch.GetName())

	// Get upstack branches (recursive children only, excluding current branch)
	rng := engine.StackRange{
		RecursiveParents:  false,
		IncludeCurrent:    false,
		RecursiveChildren: true,
	}
	graph := engine.BuildStackGraph(ctx.Engine, engine.SortStrategyAlphabetical, nil)
	upstackBranches := graph.Range(*currentBranch, rng)

	// Log upstack branches for diagnostics
	if len(upstackBranches) > 0 {
		upstackNames := make([]string, len(upstackBranches))
		for i, b := range upstackBranches {
			upstackNames[i] = b.GetName()
		}
		ctx.Logger.Info("squash restacking upstack branches=%v count=%v", upstackNames, len(upstackBranches))
	} else {
		ctx.Logger.Info("squash no upstack branches to restack")
	}

	// Restack upstack branches
	if len(upstackBranches) > 0 {
		if err := RestackBranches(ctx, upstackBranches); err != nil {
			return fmt.Errorf("failed to restack upstack branches: %w", err)
		}
	}

	return nil
}
