package actions

import (
	"fmt"

	"stackit.dev/stackit/internal/app"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/handlers"
	"stackit.dev/stackit/internal/rerere"
	"stackit.dev/stackit/internal/tui"
)

// RestackOptions contains options for the restack command
type RestackOptions struct {
	BranchName string
	Scope      engine.StackRange
}

// RestackAction performs the restack operation
func RestackAction(ctx *app.Context, opts RestackOptions, handler handlers.RestackHandler) error {
	eng := ctx.Engine
	out := ctx.Output

	// Get branches to restack based on scope
	branch := eng.GetBranch(opts.BranchName)
	graph := engine.BuildStackGraph(eng, engine.SortStrategyAlphabetical, nil)
	branches := graph.Range(branch, opts.Scope)

	if len(branches) == 0 {
		out.Info("No branches to restack.")
		return nil
	}

	ctx.Logger.Info("restack started branchCount=%v", len(branches))

	// Take snapshot before modifying the repository
	snapshotOpts := NewSnapshot("restack",
		WithArg(opts.BranchName),
	)
	if err := eng.TakeSnapshot(snapshotOpts); err != nil {
		// Log but don't fail - snapshot is best effort
		out.Debug("Failed to take snapshot: %v", err)
	}

	// If no handler provided, use NullRestackHandler (silent)
	if handler == nil {
		handler = &handlers.NullRestackHandler{}
	}

	interactiveRererePrompt := ctx.Interactive && !ctx.Quiet && tui.IsTTY()
	if _, jsonOutput := handler.(*handlers.JSONRestackHandler); jsonOutput {
		interactiveRererePrompt = false
	}
	pauser, _ := handler.(rerere.Pauser)
	if _, err := rerere.EnsureEnabled(ctx.Context, ctx.Git(), interactiveRererePrompt, pauser); err != nil {
		out.Warn("Failed to enable git rerere: %v", err)
	}

	// For standalone restack, we need to sort branches topologically for correct restack order
	sortedBranches := eng.SortBranchesTopologically(branches)

	// Use RestackHandler for consistent output
	handler.OnRestackStart(len(sortedBranches))

	var restacked, skipped int
	var conflicts []string

	if err := RestackBranchesWithHandler(ctx, sortedBranches, func(p RestackProgress) {
		res := handlers.RestackDone
		switch p.Result {
		case engine.RestackDone:
			restacked++
			res = handlers.RestackDone
		case engine.RestackUnneeded:
			res = handlers.RestackUnneeded
		case engine.RestackConflict:
			skipped++
			conflicts = append(conflicts, p.Branch)
			res = handlers.RestackConflict
		}

		// Determine parent name for consistent output
		parentName := ""
		br := eng.GetBranch(p.Branch)
		if br.GetName() != "" {
			if parent := br.GetParent(); parent != nil {
				parentName = parent.GetName()
			} else {
				parentName = eng.Trunk().GetName()
			}
		}

		// PR number is not always available without extra fetching, but we can try
		var prNumber *int
		if br.GetName() != "" {
			if pr, err := eng.GetPrInfo(br); err == nil && pr != nil {
				num := pr.Number()
				prNumber = num
			}
		}

		handler.OnRestackBranch(p.Branch, res, p.NewRev, prNumber, p.LockReason, p.Frozen, p.IsCurrent, parentName, p.Reparented, p.OldParent, p.NewParent, p.RerereResolvedCount)
	}, true); err != nil {
		handler.OnRestackComplete(restacked, skipped, conflicts)
		return fmt.Errorf("restack failed: %w", err)
	}

	ctx.Logger.Info("restack completed restacked=%v skipped=%v conflicts=%v", restacked, skipped, len(conflicts))

	handler.OnRestackComplete(restacked, skipped, conflicts)
	return nil
}
