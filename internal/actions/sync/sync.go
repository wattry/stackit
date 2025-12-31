package sync

import (
	"fmt"

	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/internal/utils"
)

// Options contains options for the sync command
type Options struct {
	All     bool
	Force   bool
	Restack bool
	DryRun  bool
}

// Action performs the sync operation
func Action(ctx *runtime.Context, opts Options, handler Handler) error {
	eng := ctx.Engine
	splog := ctx.Splog
	gctx := ctx.Context
	summary := &Summary{}

	// Use null handler if none provided
	if handler == nil {
		handler = &NullHandler{}
	}

	// Handle --all flag (stub for now)
	if opts.All {
		// For now, just sync the current trunk
		// In the future, this would sync across all configured trunks
		splog.Info("Syncing branches across all configured trunks...")
	}

	// Check for uncommitted changes
	if utils.HasUncommittedChanges(gctx) {
		return fmt.Errorf("you have uncommitted changes. Please commit or stash them before syncing")
	}

	// Calculate total operations for progress (rough estimate)
	totalOps := 1 // trunk sync
	if opts.Restack {
		// Estimate based on tracked branches
		totalOps += len(eng.AllBranches())
	}
	handler.Start(totalOps)

	// Phase 1: Pull trunk
	handler.EmitEvent(Event{Phase: PhaseTrunk, Type: EventStarted})
	if err := syncTrunk(ctx, &opts, handler, summary); err != nil {
		return err
	}

	// Clean branches (delete merged/closed)
	branchesToRestack := []string{}

	// Phase 2: Sync PR info from GitHub
	handler.EmitEvent(Event{Phase: PhaseGitHub, Type: EventStarted})
	if err := syncGitHubInfo(ctx, &branchesToRestack, handler, summary); err != nil {
		return err
	}

	// Sync remote metadata (internal, not a visible phase unless conflicts)
	if err := syncRemoteMetadata(ctx, &opts); err != nil {
		return err
	}

	// Phase 3: Clean branches (delete merged/closed)
	cleanResult, err := cleanBranches(ctx, &opts, handler, summary)
	if err != nil {
		return fmt.Errorf("failed to clean branches: %w", err)
	}

	// Add branches with new parents to restack list
	for _, branchName := range cleanResult.BranchesWithNewParents {
		branch := eng.GetBranch(branchName)
		upstack := branch.GetRelativeStackUpstack()
		for _, b := range upstack {
			branchesToRestack = append(branchesToRestack, b.GetName())
		}
		branchesToRestack = append(branchesToRestack, branchName)
	}

	// Restack if requested
	if !opts.Restack {
		splog.Tip("Try the --restack flag to automatically restack the current stack.")
		// Check if everything was up to date
		if !summary.HasChanges() {
			summary.UpToDate = true
		}
		handler.Complete(*summary)
		return nil
	}

	// Phase 4: Restack branches
	handler.EmitEvent(Event{Phase: PhaseRestack, Type: EventStarted})
	if err := restackBranches(ctx, branchesToRestack, handler, summary); err != nil {
		// Even on error, complete with summary
		handler.Complete(*summary)
		return err
	}

	// Check if everything was up to date
	if !summary.HasChanges() {
		summary.UpToDate = true
	}

	handler.Complete(*summary)
	return nil
}
