// Package fold provides functionality for folding stacked branches.
package fold

import (
	"fmt"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/app"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/tui/style"
)

// Options contains options for the fold command
type Options struct {
	Keep       bool // If true, keeps the name of the current branch instead of using the name of its parent
	AllowTrunk bool // If true, allows folding into the trunk branch
	DryRun     bool // If true, only shows what would happen
}

func showDryRun(ctx *app.Context, current, parent engine.Branch) error {
	eng := ctx.Engine
	splog := ctx.Splog

	splog.Info("%s", style.ColorYellow("Dry Run: Folding plan"))
	splog.Info("  Fold branch: %s", style.ColorBranchName(current.GetName(), true))
	splog.Info("  Into parent: %s", style.ColorBranchName(parent.GetName(), false))
	splog.Newline()

	// Show combined commit messages
	splog.Info("%s", style.ColorCyan("Proposed Commit History:"))
	parentCommits, err := parent.GetAllCommits(engine.CommitFormatReadable)
	if err != nil {
		splog.Debug("Failed to get parent commits for %s: %v", parent.GetName(), err)
	}
	for _, commit := range parentCommits {
		splog.Info("  %s", style.ColorDim(commit))
	}

	currentCommits, err := current.GetAllCommits(engine.CommitFormatReadable)
	if err != nil {
		splog.Debug("Failed to get current commits for %s: %v", current.GetName(), err)
	}
	for _, commit := range currentCommits {
		splog.Info("  %s", commit)
	}
	splog.Newline()

	// Show combined diff stat
	splog.Info("%s", style.ColorCyan("Combined Diff Stat:"))
	// Base is parent's parent (or trunk)
	grandparentName := parent.GetParentPrecondition()
	baseRev, err := eng.GetRevision(eng.GetBranch(grandparentName))
	if err != nil {
		splog.Debug("Failed to get revision for grandparent %s: %v", grandparentName, err)
		var mbErr error
		baseRev, mbErr = eng.GetMergeBase(eng.Trunk().GetName(), parent.GetName())
		if mbErr != nil {
			splog.Debug("Failed to get merge base for %s: %v", parent.GetName(), mbErr)
		}
	}

	headRev, err := current.GetRevision()
	if err != nil {
		splog.Debug("Failed to get revision for current branch %s: %v", current.GetName(), err)
	}

	diffStat, err := eng.ShowDiff(ctx.Context, baseRev, headRev, true)
	if err == nil && diffStat != "" {
		splog.Info("%s", diffStat)
	} else {
		if err != nil {
			splog.Debug("Failed to get diff stat: %v", err)
		}
		splog.Info("  (No changes or error retrieving diff)")
	}

	splog.Newline()
	splog.Info("%s", style.ColorDim("No changes were applied."))
	return nil
}

// Action performs the fold operation
func Action(ctx *app.Context, opts Options) error {
	eng := ctx.Engine
	splog := ctx.Splog
	gctx := ctx.Context

	// Validate we're on a branch
	currentBranch, err := eng.ValidateOnBranch()
	if err != nil {
		return err
	}

	// Take snapshot before modifying the repository
	snapshotOpts := actions.NewSnapshot("fold",
		actions.WithFlag(opts.Keep, "--keep"),
		actions.WithFlag(opts.AllowTrunk, "--allow-trunk"),
	)
	if err := eng.TakeSnapshot(snapshotOpts); err != nil {
		// Log but don't fail - snapshot is best effort
		splog.Debug("Failed to take snapshot: %v", err)
	}

	// Check if on trunk
	currentBranchObj := eng.GetBranch(currentBranch)
	if currentBranchObj.IsTrunk() {
		return fmt.Errorf("cannot fold trunk branch")
	}

	// Check if branch is tracked
	if !currentBranchObj.IsTracked() {
		return fmt.Errorf("cannot fold untracked branch %s", currentBranch)
	}

	// Check if rebase is in progress
	if err := ctx.Git().CheckRebaseInProgress(gctx); err != nil {
		return err
	}

	// Check for uncommitted changes
	if ctx.Git().HasUncommittedChanges(gctx) {
		return fmt.Errorf("cannot fold with uncommitted changes. Please commit or stash them first")
	}

	// Get parent branch
	// currentBranchObj already declared above
	parent := currentBranchObj.GetParent()
	parentName := ""
	if parent == nil {
		parentName = eng.Trunk().GetName()
	} else {
		parentName = parent.GetName()
	}

	parentBranch := eng.GetBranch(parentName)

	// Prohibit folding if current or parent is locked or frozen
	if err := currentBranchObj.EnsureCanModify(); err != nil {
		return err
	}
	if !parentBranch.IsTrunk() {
		if err := parentBranch.EnsureCanModify(); err != nil {
			return err
		}
	}

	// Prohibit folding branches with different scopes
	if !parentBranch.IsTrunk() {
		currentScope := currentBranchObj.GetScope()
		parentScope := parentBranch.GetScope()
		if !currentScope.Equal(parentScope) {
			return fmt.Errorf("cannot fold branches with different scopes (current: [%s], parent: [%s])", currentScope.String(), parentScope.String())
		}
	}

	if opts.DryRun {
		return showDryRun(ctx, currentBranchObj, parentBranch)
	}

	if opts.Keep {
		// Prevent folding onto trunk with --keep, as that would delete trunk
		if parentBranch.IsTrunk() {
			return fmt.Errorf("cannot fold into trunk with --keep because it would delete the trunk branch")
		}
		return foldWithKeep(gctx, ctx, currentBranchObj, parentBranch, eng, splog, opts)
	}

	// Check if folding into trunk
	if parentBranch.IsTrunk() && !opts.AllowTrunk {
		return fmt.Errorf("cannot fold into trunk branch %s without --allow-trunk. Folding into trunk will modify your local main branch directly", parentName)
	}

	return foldNormal(gctx, ctx, currentBranchObj, parentBranch, eng, splog, opts)
}
