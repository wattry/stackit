// Package merge provides CLI commands for merging stacked PRs.
package merge

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	mergeAction "stackit.dev/stackit/internal/actions/merge"
	"stackit.dev/stackit/internal/app"
	"stackit.dev/stackit/internal/cli/common"
	"stackit.dev/stackit/internal/git"
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
		branch string
		scope  string
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
					branch: branch,
					scope:  scope,
				}, postMergeHandler)
			})
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show merge plan without executing")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")
	cmd.Flags().BoolVar(&force, "force", false, "Skip validation checks (draft PRs, failing CI)")
	cmd.Flags().BoolVar(&wait, "wait", false, "Wait for merge to complete (default: fire-and-forget)")
	cmd.Flags().StringVar(&method, "method", "", "Merge method (squash, merge, rebase). Uses config merge.method if not specified")
	cmd.Flags().StringVar(&branch, "branch", "", "Target branch to merge from (default: current branch)")
	cmd.Flags().StringVar(&scope, "scope", "", "Merge the next PR within the specified scope")
	cmd.MarkFlagsMutuallyExclusive("scope", "branch")

	return cmd
}

type mergeNextOptions struct {
	dryRun bool
	yes    bool
	force  bool
	wait   bool
	method string
	branch string
	scope  string
}

func runMergeNext(ctx *app.Context, opts mergeNextOptions, postMergeHandler PostMergeHandler) error {
	out := ctx.Output
	eng := ctx.Engine

	// Use CreateMergePlan with bottom-up strategy to find what to merge
	plan, validation, err := mergeAction.CreateMergePlan(ctx.Context, eng, out, ctx.GitHubClient, mergeAction.CreatePlanOptions{
		Strategy:     mergeAction.StrategyBottomUp,
		Force:        opts.force,
		TargetBranch: opts.branch,
		Scope:        opts.scope,
	})
	if err != nil {
		return err
	}

	if len(plan.BranchesToMerge) == 0 {
		out.Success("No unmerged PRs found in the stack")
		return nil
	}

	// For "merge next", we only care about the bottom-most PR
	bottomPR := plan.BranchesToMerge[0]

	nextValidation := buildMergeNextValidation(plan, validation, opts.force)

	// Show the plan
	planText := formatMergeNextPlan(plan, nextValidation, opts.wait)
	out.Print(planText)

	// Validate
	if !nextValidation.Valid && !opts.force {
		return mergeAction.FormatValidationError(nextValidation.Errors, nextValidation.Warnings)
	}

	// Dry run - just show the plan
	if opts.dryRun {
		out.Info("Dry-run mode: No changes were made.")
		return nil
	}

	// Confirm unless --yes
	if !opts.yes && ctx.Interactive {
		confirmed, err := tui.PromptConfirm(fmt.Sprintf("Merge PR #%d (%s)?", bottomPR.PRNumber, bottomPR.BranchName), false)
		if err != nil {
			return fmt.Errorf("confirmation canceled: %w", err)
		}
		if !confirmed {
			out.Info("Merge canceled")
			return nil
		}
	}

	if ctx.GitHubClient == nil {
		return fmt.Errorf("GitHub client not available - check your GITHUB_TOKEN or gh auth login")
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
	mergeableState, err := getMergeableStateWithRetry(ctx.Context, eng.Git(), prInfo.NodeID)
	if err != nil {
		return fmt.Errorf("failed to check PR mergeable state: %w", err)
	}
	if mergeableState.State != "OPEN" {
		return fmt.Errorf("PR #%d is %s (not open)", bottomPR.PRNumber, mergeableState.State)
	}
	if !mergeableState.Mergeable {
		if strings.EqualFold(mergeableState.MergeStateText, "UNKNOWN") {
			return fmt.Errorf("PR #%d mergeability is still being calculated by GitHub. Try again shortly", bottomPR.PRNumber)
		}
		if mergeableState.MergeStateText != "" {
			return fmt.Errorf("PR #%d is not mergeable (%s). Please resolve conflicts and try again", bottomPR.PRNumber, mergeableState.MergeStateText)
		}
		return fmt.Errorf("PR #%d is not mergeable. Please resolve conflicts and try again", bottomPR.PRNumber)
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

		return nil
	}

	// Fire-and-forget: return immediately after enabling automerge
	out.Info("PR will be merged automatically when CI passes and requirements are met.")
	out.Tip("Run 'stackit sync --restack' after the PR is merged to update your stack.")
	return nil
}

func buildMergeNextValidation(plan *mergeAction.Plan, base *mergeAction.PlanValidation, force bool) *mergeAction.PlanValidation {
	next := &mergeAction.PlanValidation{
		Valid:    true,
		Errors:   []string{},
		Warnings: []string{},
		Infos:    []string{},
	}
	if plan == nil || len(plan.BranchesToMerge) == 0 {
		return next
	}

	bottom := plan.BranchesToMerge[0]
	if bottom.PRNumber <= 0 {
		next.Errors = append(next.Errors, fmt.Sprintf("Branch %s has no associated PR", bottom.BranchName))
	}

	if !force {
		if bottom.IsDraft {
			next.Errors = append(next.Errors, fmt.Sprintf("Branch %s PR #%d is a draft", bottom.BranchName, bottom.PRNumber))
		}
		if bottom.ChecksStatus == mergeAction.ChecksFailing {
			next.Errors = append(next.Errors, fmt.Sprintf("Branch %s PR #%d has failing CI checks", bottom.BranchName, bottom.PRNumber))
		}
	}

	if base != nil {
		next.Warnings = filterBranchScopedMessages(base.Warnings, bottom.BranchName)
		next.Infos = filterBranchScopedMessages(base.Infos, bottom.BranchName)
	}

	if len(next.Errors) > 0 {
		next.Valid = false
	}

	return next
}

func filterBranchScopedMessages(messages []string, branchName string) []string {
	if len(messages) == 0 {
		return nil
	}

	needle := fmt.Sprintf("Branch %s", branchName)
	filtered := make([]string, 0, len(messages))
	for _, msg := range messages {
		if strings.Contains(msg, "Branch ") {
			if strings.Contains(msg, needle) {
				filtered = append(filtered, msg)
			}
			continue
		}
		filtered = append(filtered, msg)
	}
	return filtered
}

func formatMergeNextPlan(plan *mergeAction.Plan, validation *mergeAction.PlanValidation, wait bool) string {
	var result strings.Builder

	result.WriteString("Merge Strategy: next\n")
	if plan == nil {
		result.WriteString("Current Branch: (unknown)\n\n")
		result.WriteString("Merge Plan:\n")
		result.WriteString("  (no branches to merge)\n")
		return result.String()
	}
	result.WriteString(fmt.Sprintf("Current Branch: %s\n", plan.CurrentBranch))
	result.WriteString("\n")

	if validation != nil {
		if len(validation.Errors) > 0 {
			result.WriteString("Errors:\n")
			for _, err := range validation.Errors {
				result.WriteString(fmt.Sprintf("  ✗ %s\n", err))
			}
			result.WriteString("\n")
		}

		if len(validation.Warnings) > 0 {
			result.WriteString("Warnings:\n")
			for _, warn := range validation.Warnings {
				result.WriteString(fmt.Sprintf("  ⚠ %s\n", warn))
			}
			result.WriteString("\n")
		}

		if len(validation.Infos) > 0 {
			result.WriteString("Information:\n")
			for _, info := range validation.Infos {
				result.WriteString(fmt.Sprintf("  • %s\n", info))
			}
			result.WriteString("\n")
		}
	}

	result.WriteString("Merge Plan:\n")
	if len(plan.BranchesToMerge) == 0 {
		result.WriteString("  (no branches to merge)\n")
		return result.String()
	}

	bottom := plan.BranchesToMerge[0]
	step := 1
	result.WriteString(fmt.Sprintf("  %d. Enable automerge for PR #%d (%s)\n", step, bottom.PRNumber, bottom.BranchName))
	step++

	remaining := len(plan.BranchesToMerge) - 1 + len(plan.UpstackBranches)
	if wait {
		result.WriteString(fmt.Sprintf("  %d. Wait for PR #%d to merge\n", step, bottom.PRNumber))
		step++
		if remaining > 0 {
			result.WriteString(fmt.Sprintf("  %d. Sync trunk and restack %d remaining branches\n", step, remaining))
		} else {
			result.WriteString(fmt.Sprintf("  %d. Sync trunk\n", step))
		}
	} else {
		if remaining > 0 {
			result.WriteString(fmt.Sprintf("  %d. After merge, Restack %d remaining branches (run 'stackit sync --restack')\n", step, remaining))
		} else {
			result.WriteString(fmt.Sprintf("  %d. After merge, sync trunk (run 'stackit sync --restack')\n", step))
		}
	}

	return result.String()
}

func getMergeableStateWithRetry(ctx context.Context, runner git.Runner, prNodeID string) (*github.PRMergeableState, error) {
	const (
		maxAttempts = 5
		retryDelay  = 2 * time.Second
	)

	var lastState *github.PRMergeableState
	for attempt := 0; attempt < maxAttempts; attempt++ {
		state, err := github.GetPRMergeableState(ctx, runner, prNodeID)
		if err != nil {
			return nil, err
		}
		lastState = state
		if !strings.EqualFold(state.MergeStateText, "UNKNOWN") {
			return state, nil
		}
		time.Sleep(retryDelay)
	}

	return lastState, nil
}
