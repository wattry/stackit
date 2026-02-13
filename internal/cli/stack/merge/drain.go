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
	cmd.MarkFlagsMutuallyExclusive("scope", "branch")

	return cmd
}

type mergeDrainOptions struct {
	dryRun bool
	yes    bool
	force  bool
	method string
	branch string
	scope  string
}

func runMergeDrain(ctx *app.Context, opts mergeDrainOptions, postMergeHandler PostMergeHandler) error {
	out := ctx.Output
	eng := ctx.Engine

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

	// Show the full drain plan
	planText := formatMergeDrainPlan(plan, validation, opts.force)
	out.Print(planText)

	// Validate the initial plan
	if !validation.Valid && !opts.force {
		return mergeAction.FormatValidationError(validation.Errors, validation.Warnings)
	}

	// Dry run - just show the plan
	if opts.dryRun {
		out.Info("Dry-run mode: No changes were made.")
		return nil
	}

	// Confirm unless --yes
	if !opts.yes && ctx.Interactive {
		confirmed, err := tui.PromptConfirm(fmt.Sprintf("Drain all %d PRs in the stack?", totalPRs), false)
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

	// Resolve merge method once upfront (flag > config > prompt)
	mergeMethod, err := resolveMergeMethod(ctx, opts.method)
	if err != nil {
		return err
	}

	// Drain loop: merge one PR at a time, bottom-up
	merged := 0
	for {
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
				break
			}
			return err
		}

		if len(plan.BranchesToMerge) == 0 {
			break
		}

		bottomPR := plan.BranchesToMerge[0]

		out.Newline()
		out.Info("Merging PR #%d (%s) [%d/%d]...", bottomPR.PRNumber, bottomPR.BranchName, merged+1, totalPRs)

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

func formatMergeDrainPlan(plan *mergeAction.Plan, validation *mergeAction.PlanValidation, force bool) string {
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

	if validation != nil {
		if len(validation.Errors) > 0 && !force {
			result.WriteString("Errors:\n")
			for _, err := range validation.Errors {
				fmt.Fprintf(&result, "  ✗ %s\n", err)
			}
			result.WriteString("\n")
		}

		if len(validation.Warnings) > 0 {
			result.WriteString("Warnings:\n")
			for _, warn := range validation.Warnings {
				fmt.Fprintf(&result, "  ⚠ %s\n", warn)
			}
			result.WriteString("\n")
		}

		if len(validation.Infos) > 0 {
			result.WriteString("Information:\n")
			for _, info := range validation.Infos {
				fmt.Fprintf(&result, "  • %s\n", info)
			}
			result.WriteString("\n")
		}
	}

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
