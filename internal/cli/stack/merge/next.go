// Package merge provides CLI commands for merging stacked PRs.
package merge

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/actions"
	mergeAction "stackit.dev/stackit/internal/actions/merge"
	"stackit.dev/stackit/internal/actions/sync"
	"stackit.dev/stackit/internal/app"
	"stackit.dev/stackit/internal/cli/common"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/github"
	"stackit.dev/stackit/internal/tui"
)

const (
	// DefaultMergeTimeout is the default timeout for waiting on a merge to complete
	DefaultMergeTimeout = 30 * time.Minute
	// DefaultMergePollInterval is the default interval between merge status checks
	DefaultMergePollInterval = 10 * time.Second
)

// NewNextCmd creates the merge next subcommand.
// This command merges the bottom-most unmerged PR in the stack, restacks the remaining
// branches, and stops. It uses GitHub automerge by default and waits for the merge to complete.
func NewNextCmd(postMergeHandler PostMergeHandler) *cobra.Command {
	var (
		dryRun bool
		yes    bool
		force  bool
		wait   bool
		method string
	)

	cmd := &cobra.Command{
		Use:   "next",
		Short: "Merge the next (bottom-most) unmerged PR in the stack",
		Long: `Merge the bottom-most unmerged PR in the stack using GitHub automerge.

After enabling automerge, the command returns immediately (fire-and-forget).

Use --wait to block until the PR is merged, then automatically:
1. Pull the latest trunk
2. Restack the remaining branches in the stack
3. Stop (run again to merge the next PR)`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return common.Run(cmd, func(ctx *app.Context) error {
				return runMergeNext(ctx, mergeNextOptions{
					dryRun: dryRun,
					yes:    yes,
					force:  force,
					wait:   wait,
					method: method,
				}, postMergeHandler)
			})
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show merge plan without executing")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")
	cmd.Flags().BoolVar(&force, "force", false, "Skip validation checks (draft PRs, failing CI)")
	cmd.Flags().BoolVar(&wait, "wait", false, "Wait for merge to complete (default: fire-and-forget)")
	cmd.Flags().StringVar(&method, "method", "", "Merge method (squash, merge, rebase). Uses config merge.method if not specified")

	return cmd
}

type mergeNextOptions struct {
	dryRun bool
	yes    bool
	force  bool
	wait   bool
	method string
}

func runMergeNext(ctx *app.Context, opts mergeNextOptions, postMergeHandler PostMergeHandler) error {
	out := ctx.Output
	eng := ctx.Engine

	// Find the bottom-most unmerged PR in the stack
	bottomPR, upstackBranches, err := findBottomUnmergedPR(ctx)
	if err != nil {
		return err
	}

	if bottomPR == nil {
		out.Success("No unmerged PRs found in the stack")
		return nil
	}

	// Show the plan
	out.Info("Merge next PR:")
	out.Info("  Branch: %s", bottomPR.BranchName)
	out.Info("  PR: #%d %s", bottomPR.PRNumber, bottomPR.PRURL)
	if len(upstackBranches) > 0 {
		out.Info("  Upstack: %d branches will be restacked", len(upstackBranches))
	}
	out.Newline()

	// Show validation warnings
	if bottomPR.IsDraft && !opts.force {
		return fmt.Errorf("PR #%d is a draft. Use --force to merge anyway", bottomPR.PRNumber)
	}
	if bottomPR.ChecksStatus == mergeAction.ChecksFailing && !opts.force {
		return fmt.Errorf("PR #%d has failing CI checks. Use --force to merge anyway", bottomPR.PRNumber)
	}
	if !bottomPR.MatchesRemote && !opts.force {
		out.Warn("Branch %s differs from remote", bottomPR.BranchName)
	}

	// Dry run - just show the plan
	if opts.dryRun {
		out.Info("Dry-run mode: No changes were made.")
		return nil
	}

	// Confirm unless --yes
	if !opts.yes && ctx.Interactive {
		confirmed, err := tui.PromptConfirm("Proceed with merge?", false)
		if err != nil {
			return fmt.Errorf("confirmation canceled: %w", err)
		}
		if !confirmed {
			out.Info("Merge canceled")
			return nil
		}
	}

	// Get the PR's NodeID for automerge
	owner, repo := ctx.GitHubClient.GetOwnerRepo()
	prInfo, err := ctx.GitHubClient.GetPullRequest(ctx.Context, owner, repo, bottomPR.PRNumber)
	if err != nil {
		return fmt.Errorf("failed to get PR info: %w", err)
	}
	if prInfo.NodeID == "" {
		return fmt.Errorf("PR #%d does not have a Node ID", bottomPR.PRNumber)
	}

	// Check if PR is mergeable
	mergeableState, err := github.GetPRMergeableState(ctx.Context, eng.Git(), prInfo.NodeID)
	if err != nil {
		return fmt.Errorf("failed to check PR mergeable state: %w", err)
	}
	if !mergeableState.Mergeable {
		return fmt.Errorf("PR #%d has merge conflicts. Please resolve conflicts first", bottomPR.PRNumber)
	}

	// Determine merge method: flag > config > prompt
	var mergeMethod github.MergeMethod
	if opts.method != "" {
		// Flag override
		switch opts.method {
		case "squash":
			mergeMethod = github.MergeMethodSquash
		case "merge":
			mergeMethod = github.MergeMethodMerge
		case "rebase":
			mergeMethod = github.MergeMethodRebase
		default:
			return fmt.Errorf("invalid merge method: %s (must be squash, merge, or rebase)", opts.method)
		}
	} else {
		// Use config or prompt user
		var err error
		mergeMethod, err = mergeAction.GetMergeMethod(ctx, ctx.GitHubClient)
		if err != nil {
			return fmt.Errorf("failed to determine merge method: %w", err)
		}
	}

	// Enable automerge
	out.Info("Enabling automerge on PR #%d (method: %s)...", bottomPR.PRNumber, mergeMethod)
	if err := github.EnableAutoMerge(ctx.Context, eng.Git(), prInfo.NodeID, mergeMethod); err != nil {
		return fmt.Errorf("failed to enable automerge: %w", err)
	}
	out.Success("Automerge enabled on PR #%d", bottomPR.PRNumber)

	// If --wait, wait for merge and perform cleanup
	if opts.wait {
		out.Info("Waiting for PR #%d to be merged...", bottomPR.PRNumber)
		if err := github.WaitForPRMerge(ctx.Context, eng.Git(), prInfo.NodeID, DefaultMergeTimeout, DefaultMergePollInterval); err != nil {
			return fmt.Errorf("failed waiting for merge: %w", err)
		}
		out.Success("PR #%d merged successfully!", bottomPR.PRNumber)

		// Perform post-merge cleanup
		out.Newline()
		out.Info("Performing post-merge cleanup...")

		// Use post-merge handler for cleanup (checkout trunk, sync, restack)
		if postMergeHandler != nil {
			return postMergeHandler(ctx, mergeAction.PostMergeSyncTrunk)
		}

		// Fallback: manual cleanup if no handler
		return performPostMergeCleanup(ctx, bottomPR.BranchName, upstackBranches)
	}

	// Fire-and-forget: return immediately after enabling automerge
	out.Info("PR will be merged automatically when CI passes and requirements are met.")
	out.Tip("Run 'stackit sync --restack' after the PR is merged to update your stack.")
	return nil
}

// findBottomUnmergedPR finds the bottom-most unmerged PR in the current stack.
// It returns the PR info and a list of all branches that will need restacking after the merge.
func findBottomUnmergedPR(ctx *app.Context) (*mergeAction.BranchMergeInfo, []string, error) {
	eng := ctx.Engine

	// Get current branch
	currentBranch := eng.CurrentBranch()
	if currentBranch == nil {
		return nil, nil, fmt.Errorf("not on a branch")
	}
	if currentBranch.IsTrunk() {
		return nil, nil, fmt.Errorf("cannot merge from trunk")
	}
	if !currentBranch.IsTracked() {
		return nil, nil, fmt.Errorf("branch %s is not tracked by stackit", currentBranch.GetName())
	}

	// Build stack graph
	graph := engine.BuildStackGraph(eng, engine.SortStrategyAlphabetical, nil)

	// Get all branches from trunk to current (including current)
	rng := engine.StackRange{RecursiveParents: true}
	parentBranches := graph.Range(*currentBranch, rng)

	// Build list of branches (excluding trunk), ordered from bottom to top
	allBranches := make([]string, 0, len(parentBranches)+1)
	for _, branch := range parentBranches {
		if !branch.IsTrunk() {
			allBranches = append(allBranches, branch.GetName())
		}
	}
	allBranches = append(allBranches, currentBranch.GetName())

	// Batch load metadata
	allMeta, metaErrors := eng.Git().BatchReadMetadata(allBranches)
	// Log any errors but don't fail - missing metadata just means no PR info
	for branch, metaErr := range metaErrors {
		if metaErr != nil {
			ctx.Output.Debug("failed to load metadata for %s: %v", branch, metaErr)
		}
	}

	// Batch load CI status
	var allCheckStatuses map[string]*github.CheckStatus
	if ctx.GitHubClient != nil {
		var checksErr error
		allCheckStatuses, checksErr = ctx.GitHubClient.BatchGetPRChecksStatus(ctx.Context, allBranches)
		if checksErr != nil {
			// Log but don't fail - CI status is optional
			ctx.Output.Debug("failed to load CI status: %v", checksErr)
		}
	}

	// Find the first unmerged PR (from bottom up)
	for i, branchName := range allBranches {
		meta := allMeta[branchName]
		prInfo := engine.NewPrInfoFromMeta(meta)

		// Skip if no PR
		if prInfo == nil || prInfo.Number() == nil {
			continue
		}

		// Skip if already merged
		state := prInfo.State()
		if state == "MERGED" {
			continue
		}

		// Skip if not open
		if state != "OPEN" {
			continue
		}

		// Found the bottom-most unmerged PR
		checksStatus := mergeAction.ChecksNone
		if allCheckStatuses != nil {
			if status, ok := allCheckStatuses[branchName]; ok && status != nil {
				switch {
				case status.Pending:
					checksStatus = mergeAction.ChecksPending
				case !status.Passing:
					checksStatus = mergeAction.ChecksFailing
				default:
					checksStatus = mergeAction.ChecksPassing
				}
			}
		}

		// Check if local matches remote
		status, err := eng.GetBranchRemoteStatus(eng.GetBranch(branchName))
		matchesRemote := true
		if err == nil {
			matchesRemote = status.Matches()
		}

		// Calculate upstack branches: everything after this branch in allBranches,
		// plus any children of the current branch
		upstackBranches := make([]string, 0)

		// Add branches between merged branch and current (these are in allBranches after index i)
		if i+1 < len(allBranches) {
			upstackBranches = append(upstackBranches, allBranches[i+1:]...)
		}

		// Add children of current branch (branches above current in the stack)
		upstack := graph.Range(*currentBranch, engine.StackRange{RecursiveChildren: true})
		for _, ub := range upstack {
			if ub.IsTracked() {
				upstackBranches = append(upstackBranches, ub.GetName())
			}
		}

		return &mergeAction.BranchMergeInfo{
			BranchName:    branchName,
			PRNumber:      *prInfo.Number(),
			PRURL:         prInfo.URL(),
			IsDraft:       prInfo.IsDraft(),
			ChecksStatus:  checksStatus,
			MatchesRemote: matchesRemote,
		}, upstackBranches, nil
	}

	// No unmerged PR found - return empty upstack (nothing to restack)
	return nil, nil, nil
}

// performPostMergeCleanup performs cleanup after a PR is merged
func performPostMergeCleanup(ctx *app.Context, mergedBranch string, upstackBranches []string) error {
	out := ctx.Output
	eng := ctx.Engine

	// Checkout trunk
	out.Info("Checking out trunk...")
	_, err := actions.CheckoutAction(ctx, actions.CheckoutOptions{
		CheckoutTrunk: true,
	}, nil)
	if err != nil {
		out.Warn("Failed to checkout trunk: %v", err)
		out.Tip("Run 'stackit checkout --trunk' to switch to trunk")
		return nil
	}

	// Pull trunk
	out.Info("Pulling latest trunk...")
	pullResult, err := eng.PullTrunk(ctx.Context)
	if err != nil {
		out.Warn("Failed to pull trunk: %v", err)
	} else if pullResult == engine.PullConflict {
		out.Warn("Trunk has conflicts with remote")
	}

	// Delete merged branch
	out.Info("Deleting merged branch %s...", mergedBranch)
	if err := eng.Git().DeleteBranch(ctx.Context, mergedBranch); err != nil {
		out.Warn("Failed to delete branch %s: %v", mergedBranch, err)
	}

	// Restack upstack branches
	if len(upstackBranches) > 0 {
		out.Info("Restacking %d upstack branches...", len(upstackBranches))
		if err := sync.Action(ctx, sync.Options{
			Restack: true,
		}, nil); err != nil {
			out.Warn("Failed to restack: %v", err)
			out.Tip("Run 'stackit sync --restack' to complete the restack")
		}
	}

	out.Newline()
	out.Success("Post-merge cleanup complete!")

	return nil
}
