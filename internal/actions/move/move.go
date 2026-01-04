// Package move provides functionality for moving branches to different parents in the stack.
package move

import (
	"fmt"
	"strings"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/app"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/tui"
	"stackit.dev/stackit/internal/tui/style"
	"stackit.dev/stackit/internal/utils"
)

// Options contains options for the move command
type Options struct {
	Source string // Branch to move (defaults to current branch)
	Onto   string // Branch to move onto
}

// Action performs the move operation
func Action(ctx *app.Context, opts Options) error {
	eng := ctx.Engine
	out := ctx.Output
	gctx := ctx.Context

	// Default source to current branch
	source := opts.Source
	if source == "" {
		currentBranch := eng.CurrentBranch()
		if currentBranch == nil {
			return fmt.Errorf("not on a branch and no source branch specified")
		}
		source = currentBranch.GetName()
	}

	// Take snapshot before modifying the repository
	snapshotOpts := actions.NewSnapshot("move",
		actions.WithFlagValue("--source", opts.Source),
		actions.WithFlagValue("--onto", opts.Onto),
	)
	if err := eng.TakeSnapshot(snapshotOpts); err != nil {
		// Log but don't fail - snapshot is best effort
		out.Debug("Failed to take snapshot: %v", err)
	}

	// Prevent moving trunk (check before tracking check since trunk might not be tracked)
	sourceBranch := eng.GetBranch(source)
	if sourceBranch.IsTrunk() {
		return fmt.Errorf("cannot move trunk branch")
	}

	// Validate source exists and is tracked
	if !sourceBranch.IsTracked() {
		return fmt.Errorf("branch %s is not tracked by Stackit", source)
	}

	// Validate onto is provided
	onto := opts.Onto
	if onto == "" {
		return fmt.Errorf("onto branch must be specified")
	}

	// Validate onto exists
	ontoBranch := eng.GetBranch(onto)
	if !ontoBranch.IsTrunk() && !ontoBranch.IsTracked() {
		// Check if it's an untracked branch
		allBranches := eng.AllBranches()
		found := false
		for _, branch := range allBranches {
			if branch.GetName() == onto {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("branch %s does not exist", onto)
		}
	}

	// Prevent moving onto itself
	if source == onto {
		return fmt.Errorf("cannot move branch onto itself")
	}

	// Cycle detection: ensure onto is not a descendant of source
	sourceBranch = eng.GetBranch(source)
	descendants := sourceBranch.GetRelativeStack(engine.StackRange{
		RecursiveChildren: true,
		IncludeCurrent:    true,
		RecursiveParents:  false,
	})
	for _, d := range descendants {
		if d.GetName() == onto {
			return fmt.Errorf("cannot move %s onto its own descendant %s", source, onto)
		}
	}

	// Check for scope change and potential rename
	sourceScope := sourceBranch.GetScope()
	ontoScope := ontoBranch.GetScope()
	if sourceScope.IsDefined() && ontoScope.IsDefined() && !sourceScope.Equal(ontoScope) {
		if utils.IsInteractive() && strings.Contains(source, sourceScope.String()) {
			confirmed, err := tui.PromptConfirm(fmt.Sprintf("Branch name contains '%s', but its scope will now be '%s'. Would you like to rename the branch?", sourceScope.String(), ontoScope.String()), true)
			if err == nil && confirmed {
				newName := strings.Replace(source, sourceScope.String(), ontoScope.String(), 1)
				if err := eng.RenameBranch(gctx, eng.GetBranch(source), eng.GetBranch(newName)); err != nil {
					out.Info("Warning: failed to rename branch: %v", err)
				} else {
					out.Info("Renamed branch %s to %s.", style.ColorBranchName(source, false), style.ColorBranchName(newName, true))
					source = newName
					sourceBranch = eng.GetBranch(source)
				}
			}
		}
	}

	// Get current parent for logging
	// sourceBranch already declared above
	oldParent := sourceBranch.GetParent()
	oldParentName := ""
	if oldParent == nil {
		oldParentName = eng.Trunk().GetName()
	} else {
		oldParentName = oldParent.GetName()
	}

	// Capture old divergence point to preserve it after reparenting
	// This ensures we only move the commits that belong to this branch.
	var oldParentRev string
	if meta, err := eng.Git().ReadMetadata(source); err == nil && meta.ParentBranchRevision != nil {
		oldParentRev = *meta.ParentBranchRevision
	}

	// Update parent in engine
	if err := eng.SetParent(gctx, sourceBranch, ontoBranch); err != nil {
		return fmt.Errorf("failed to set parent: %w", err)
	}

	// Restore old divergence point if it's still a valid ancestor
	if oldParentRev != "" {
		if isAncestor, _ := eng.Git().IsAncestor(oldParentRev, source); isAncestor {
			_ = eng.UpdateParentRevision(source, oldParentRev)
		}
	}

	out.Info("Moved %s from %s to %s.",
		style.ColorBranchName(source, true),
		style.ColorBranchName(oldParentName, false),
		style.ColorBranchName(onto, false))

	// Get all branches that need restacking: source and all its descendants
	branchesToRestack := sourceBranch.GetRelativeStack(engine.StackRange{
		RecursiveChildren: true,
		IncludeCurrent:    true,
		RecursiveParents:  false,
	})

	// Restack source and all its descendants
	if err := actions.RestackBranches(ctx, branchesToRestack); err != nil {
		return fmt.Errorf("failed to restack branches: %w", err)
	}

	return nil
}
