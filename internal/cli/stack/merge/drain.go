// Package merge provides CLI commands for merging stacked PRs.
package merge

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	mergeAction "stackit.dev/stackit/internal/actions/merge"
	"stackit.dev/stackit/internal/app"
	"stackit.dev/stackit/internal/cli/common"
	"stackit.dev/stackit/internal/github"
	"stackit.dev/stackit/internal/tui"
)

// NewDrainCmd creates the merge drain subcommand.
// This command merges all PRs in the stack bottom-up, waiting for each merge
// to complete before proceeding to the next one.
func NewDrainCmd(postMergeHandler PostMergeHandler) *cobra.Command {
	var (
		dryRun bool
		yes    bool
		force  bool
		method string
		branch string
		scope  string
		count  int
	)

	cmd := &cobra.Command{
		Use:   "drain",
		Short: "Merge all PRs in the stack bottom-up, waiting for each to complete",
		Long: `Merge all PRs in the stack bottom-up, waiting for each merge to complete
before proceeding to the next one.

For each PR in the stack (from bottom to top):
1. Enable automerge on the bottom-most unmerged PR
2. Wait for the PR to be merged
3. Sync trunk and restack remaining branches
4. Repeat until all PRs are merged

This is equivalent to running "merge next --wait" in a loop.

Examples:
  stackit merge drain              # Drain the entire stack
  stackit merge drain --count 2    # Drain only the first 2 PRs
  stackit merge drain --dry-run    # Show what would be merged
  stackit merge drain --yes        # Skip confirmation prompt
  stackit merge drain --method squash  # Use squash merge for all PRs`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return common.Run(cmd, func(ctx *app.Context) error {
				return runMergeDrain(ctx, mergeDrainOptions{
					dryRun: dryRun,
					yes:    yes,
					force:  force,
					method: method,
					branch: branch,
					scope:  scope,
					count:  count,
				}, postMergeHandler)
			})
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show merge plan without executing")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")
	cmd.Flags().BoolVar(&force, "force", false, "Skip validation checks (draft PRs, failing CI)")
	cmd.Flags().StringVar(&method, "method", "", "Merge method (squash, merge, rebase). Uses config merge.method if not specified")
	cmd.Flags().StringVar(&branch, "branch", "", "Target branch to merge from (default: current branch)")
	cmd.Flags().StringVar(&scope, "scope", "", "Merge PRs within the specified scope")
	cmd.Flags().IntVar(&count, "count", 0, "Maximum number of PRs to drain (0 = all)")
	cmd.MarkFlagsMutuallyExclusive("scope", "branch")
	_ = cmd.RegisterFlagCompletionFunc("count", cobra.NoFileCompletions)

	return cmd
}

type mergeDrainOptions struct {
	dryRun bool
	yes    bool
	force  bool
	method string
	branch string
	scope  string
	count  int // Maximum number of PRs to drain (0 = all)
}

func runMergeDrain(ctx *app.Context, opts mergeDrainOptions, postMergeHandler PostMergeHandler) error {
	out := ctx.Output
	eng := ctx.Engine

	if opts.count < 0 {
		return fmt.Errorf("--count must be non-negative (got %d)", opts.count)
	}

	// Create initial plan to discover what needs to be merged
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

	totalPRs := len(plan.BranchesToMerge)

	// Apply count limit: 0 means unlimited, positive caps at totalPRs
	drainTarget := 0 // 0 = unlimited
	if opts.count > 0 {
		drainTarget = min(opts.count, totalPRs)
	}

	// Show the full drain plan
	planText := formatMergeDrainPlan(plan, validation)
	out.Print(planText)

	if drainTarget > 0 && drainTarget < totalPRs {
		out.Info("Draining first %d of %d PRs (--count=%d)", drainTarget, totalPRs, opts.count)
	}

	// Validate the initial plan
	if !validation.Valid && !opts.force {
		return mergeAction.FormatValidationError(validation.Errors, validation.Warnings)
	}

	// Dry run - just show the plan
	if opts.dryRun {
		out.Info("Dry-run mode: No changes were made.")
		return nil
	}

	// Fail fast if no GitHub client
	if ctx.GitHubClient == nil {
		return fmt.Errorf("GitHub client not available - check your GITHUB_TOKEN or gh auth login")
	}

	// Confirm unless --yes
	if !opts.yes && ctx.Interactive {
		prompt := fmt.Sprintf("Drain all %d PRs in the stack?", totalPRs)
		if drainTarget > 0 && drainTarget < totalPRs {
			prompt = fmt.Sprintf("Drain %d of %d PRs in the stack?", drainTarget, totalPRs)
		}
		confirmed, err := tui.PromptConfirm(prompt, false)
		if err != nil {
			return fmt.Errorf("confirmation canceled: %w", err)
		}
		if !confirmed {
			out.Info("Merge canceled")
			return nil
		}
	}

	// Resolve merge method once upfront (flag > config > prompt)
	mergeMethod, err := resolveMergeMethod(ctx, opts.method)
	if err != nil {
		return err
	}

	// Drain loop: merge one PR at a time, bottom-up
	merged := 0
	for drainTarget == 0 || merged < drainTarget {
		// Re-read state each iteration (branches change after merges + sync)
		plan, _, err = mergeAction.CreateMergePlan(ctx.Context, eng, out, ctx.GitHubClient, mergeAction.CreatePlanOptions{
			Strategy:     mergeAction.StrategyBottomUp,
			Force:        opts.force,
			TargetBranch: opts.branch,
			Scope:        opts.scope,
		})
		if err != nil {
			// After merging some PRs, "not on a branch" or "on trunk" can happen
			// if post-merge sync moved us. This is expected when stack is fully drained.
			if merged > 0 {
				out.Debug("Stopping drain after %d merges: %v", merged, err)
				break
			}
			return err
		}

		if len(plan.BranchesToMerge) == 0 {
			break
		}

		bottomPR := plan.BranchesToMerge[0]

		out.Newline()
		displayTotal := totalPRs
		if drainTarget > 0 {
			displayTotal = drainTarget
		}
		out.Info("Merging PR #%d (%s) [%d/%d]...", bottomPR.PRNumber, bottomPR.BranchName, merged+1, displayTotal)

		// Get the PR's NodeID for merge operations
		owner, repo := ctx.GitHubClient.GetOwnerRepo()
		prInfo, err := ctx.GitHubClient.GetPullRequest(ctx.Context, owner, repo, bottomPR.PRNumber)
		if err != nil {
			return fmt.Errorf("failed to get PR #%d info: %w", bottomPR.PRNumber, err)
		}
		if prInfo.NodeID == "" {
			return fmt.Errorf("PR #%d does not have a Node ID", bottomPR.PRNumber)
		}

		// Orchestrate the merge (direct merge → automerge → poll fallback)
		// Drain always waits for each PR to merge before proceeding.
		_, err = orchestrateMerge(ctx, orchestrateMergeOptions{
			branchName:  bottomPR.BranchName,
			prNumber:    bottomPR.PRNumber,
			prNodeID:    prInfo.NodeID,
			mergeMethod: mergeMethod,
			wait:        true,
		})
		if err != nil {
			return err
		}

		merged++

		// Post-merge cleanup: sync trunk + restack for next iteration
		if postMergeHandler != nil {
			out.Info("Syncing trunk and restacking...")
			if err := postMergeHandler(ctx, mergeAction.PostMergeSyncTrunk); err != nil {
				return fmt.Errorf("post-merge sync failed after PR #%d: %w", bottomPR.PRNumber, err)
			}
		}
	}

	out.Newline()
	out.Success("Drained %d PRs from the stack", merged)
	return nil
}

func resolveMergeMethod(ctx *app.Context, methodFlag string) (github.MergeMethod, error) {
	if methodFlag != "" {
		switch methodFlag {
		case "squash":
			return github.MergeMethodSquash, nil
		case "merge":
			return github.MergeMethodMerge, nil
		case "rebase":
			return github.MergeMethodRebase, nil
		default:
			return "", fmt.Errorf("invalid merge method: %s (must be squash, merge, or rebase)", methodFlag)
		}
	}
	return mergeAction.GetMergeMethod(ctx, ctx.GitHubClient)
}

func formatMergeDrainPlan(plan *mergeAction.Plan, validation *mergeAction.PlanValidation) string {
	var result strings.Builder

	result.WriteString("Merge Strategy: drain (bottom-up, all PRs)\n")
	if plan == nil {
		result.WriteString("Current Branch: (unknown)\n\n")
		result.WriteString("Drain Plan:\n")
		result.WriteString("  (no branches to merge)\n")
		return result.String()
	}
	fmt.Fprintf(&result, "Current Branch: %s\n", plan.CurrentBranch)
	result.WriteString("\n")

	mergeAction.FormatValidationSection(&result, validation)

	result.WriteString("Drain Plan:\n")
	if len(plan.BranchesToMerge) == 0 {
		result.WriteString("  (no branches to merge)\n")
		return result.String()
	}

	for i, branch := range plan.BranchesToMerge {
		fmt.Fprintf(&result, "  %d. Merge PR #%d (%s)\n", i+1, branch.PRNumber, branch.BranchName)
	}
	fmt.Fprintf(&result, "\nTotal: %d PRs to drain\n", len(plan.BranchesToMerge))

	return result.String()
}
