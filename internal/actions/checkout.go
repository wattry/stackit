package actions

import (
	"fmt"

	"stackit.dev/stackit/internal/app"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/errors"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/tui"
	"stackit.dev/stackit/internal/tui/style"
	"stackit.dev/stackit/internal/utils"
)

// CheckoutOptions contains options for the checkout command
type CheckoutOptions struct {
	BranchName    string // Optional: branch to checkout directly
	ShowUntracked bool   // Include untracked branches in selection
	All           bool   // Show all branches across trunks
	StackOnly     bool   // Only show current stack (ancestors + descendants)
	CheckoutTrunk bool   // Checkout trunk directly
}

// CheckoutAction performs the checkout operation
func CheckoutAction(ctx *app.Context, opts CheckoutOptions) error {
	eng := ctx.Engine
	out := ctx.Output
	context := ctx.Context

	var branchName string
	var err error

	switch {
	case opts.CheckoutTrunk:
		branchName = eng.Trunk().GetName()
	case opts.BranchName != "":
		branchName = opts.BranchName
	default:
		// Only populate remote SHAs when entering interactive mode
		// (the selector may need remote information for display)
		if err := eng.PopulateRemoteShas(); err != nil {
			return fmt.Errorf("failed to populate remote SHAs: %w", err)
		}

		if !utils.IsInteractive() {
			return fmt.Errorf("interactive branch selection is not available in non-interactive mode; please specify a branch name")
		}
		branchName, err = tui.PromptLogSelect(ctx.Context, ctx.Engine, ctx.GitHubClient, tui.LogOptions{
			Style:         "FULL", // Show stats by default in checkout selector
			ShowUntracked: opts.ShowUntracked,
		})
		if err != nil {
			if errors.Is(err, errors.ErrCanceled) {
				currentBranch := eng.CurrentBranch()
				currentBranchName := "trunk"
				if currentBranch != nil {
					currentBranchName = currentBranch.GetName()
				}
				out.Info("No branch selected; staying on %s.", style.ColorBranchName(currentBranchName, true))
				return nil
			}
			return err
		}
	}

	currentBranch := eng.CurrentBranch()
	if currentBranch != nil && branchName == currentBranch.GetName() {
		out.Info("Already on %s.", style.ColorBranchName(branchName, true))
		return nil
	}

	branch := eng.GetBranch(branchName)

	// Check for cross-worktree checkout and warn if applicable
	if ctx.InManagedWorktree && ctx.WorktreeInfo != nil {
		targetStackRoot := eng.GetStackRootForBranch(branch)
		currentStackRoot := ctx.WorktreeInfo.StackRoot

		// Warn if checking out a branch from a different stack
		if targetStackRoot != "" && targetStackRoot != currentStackRoot {
			// Check if the target stack has its own worktree
			targetWorktree, err := eng.GetWorktreeForStack(targetStackRoot)
			if err == nil && targetWorktree != nil {
				out.Warn("Branch %s belongs to a different stack (%s) that has its own worktree.",
					style.ColorBranchName(branchName, false),
					style.ColorBranchName(targetStackRoot, false))
				out.Tip("cd %s", targetWorktree.Path)
			}
		}
	}

	if err := eng.CheckoutBranch(context, branch); err != nil {
		if git.IsLocalChangesError(err) {
			return fmt.Errorf("cannot checkout branch %s because you have uncommitted changes that would be overwritten; please commit or stash your changes before switching branches", branchName)
		}
		return fmt.Errorf("failed to checkout branch %s: %w", branchName, err)
	}

	out.Info("Checked out %s.", style.ColorBranchName(branchName, false))

	// Skip branch info in quiet mode for faster checkout
	if !ctx.Quiet {
		printBranchInfo(ctx, branch)
	}

	return nil
}

func printBranchInfo(ctx *app.Context, branch engine.Branch) {
	if branch.IsTrunk() {
		return
	}

	if !branch.IsTracked() {
		ctx.Output.Info("This branch is not tracked by Stackit.")
		return
	}

	if !branch.IsBranchUpToDate() {
		parent := branch.GetParentPrecondition()
		ctx.Output.Info("This branch has fallen behind %s - you may want to %s.",
			style.ColorBranchName(parent, false),
			style.ColorCyan("stackit upstack restack"))
		return
	}

	// Check if any downstack branch needs restack
	// Limit to checking up to 10 ancestors to avoid performance issues with deep stacks
	rng := engine.StackRange{
		RecursiveParents:  true,
		IncludeCurrent:    false,
		RecursiveChildren: false,
	}
	graph := engine.BuildStackGraph(ctx.Engine, engine.SortStrategyAlphabetical, nil)
	downstack := graph.Range(branch, rng)

	// Limit the number of branches we check to avoid slow metadata reads
	const maxDownstackChecks = 10
	startIdx := len(downstack) - maxDownstackChecks
	if startIdx < 0 {
		startIdx = 0
	}

	// Check from trunk upward (but limit to last maxDownstackChecks branches)
	for i := len(downstack) - 1; i >= startIdx; i-- {
		ancestor := downstack[i]
		if !ancestor.IsBranchUpToDate() {
			parent := ancestor.GetParentPrecondition()
			ctx.Output.Info("The downstack branch %s has fallen behind %s - you may want to %s.",
				style.ColorBranchName(ancestor.GetName(), false),
				style.ColorBranchName(parent, false),
				style.ColorCyan("stackit stack restack"))
			return
		}
	}
}
