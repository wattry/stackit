package stack

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/actions/merge"
	"stackit.dev/stackit/internal/app"
	"stackit.dev/stackit/internal/cli/common"
)

// NewCombineCmd creates the combine command
func NewCombineCmd() *cobra.Command {
	var (
		stacks []string
		wait   bool
		skipCI bool
		yes    bool
	)

	cmd := &cobra.Command{
		Use:        "combine",
		Short:      "Combine multiple stacks into a single PR",
		Deprecated: "Use 'stackit merge --multi-stack' instead",
		Long: `Combine multiple independent stacks into a single consolidated PR.

DEPRECATED: This command is deprecated. Use 'stackit merge --multi-stack' instead.

This command allows you to select multiple stacks (each rooted at trunk) and merge
them together into a single PR. It handles:

  - Conflict detection: If a stack conflicts with others, it's skipped
  - Local CI validation: Runs a configured command to validate the combined code
  - Binary search: If CI fails, finds the largest subset that passes

Configuration:
  Set the CI command with: stackit config set ci.command "your-command"
  Set the CI timeout with: stackit config set ci.timeout 600

Examples:
  stackit merge --multi-stack                          # Interactive (recommended)
  stackit merge --multi-stack --stacks feat/a,feat/b   # Specify stacks directly
  stackit merge --multi-stack --skip-local-ci          # Skip local CI validation
  stackit merge --multi-stack --wait                   # Wait for remote CI and auto-merge`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return common.Run(cmd, func(ctx *app.Context) error {
				// Parse comma-separated stacks if provided as a single string
				var selectedStacks []string
				for _, s := range stacks {
					for _, part := range strings.Split(s, ",") {
						if trimmed := strings.TrimSpace(part); trimmed != "" {
							selectedStacks = append(selectedStacks, trimmed)
						}
					}
				}

				// Delegate to multi-stack merge
				return runCombine(ctx, selectedStacks, skipCI, wait, yes)
			})
		},
	}

	cmd.Flags().StringSliceVar(&stacks, "stacks", nil, "Stack root branches to combine (comma-separated, skips picker)")
	cmd.Flags().BoolVar(&wait, "wait", false, "Wait for CI and auto-merge the combined PR")
	cmd.Flags().BoolVar(&skipCI, "skip-ci", false, "Skip local CI validation")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompts")

	// Mark deprecated flags
	cmd.Flags().Bool("dry-run", false, "")
	cmd.Flags().Bool("force", false, "")
	_ = cmd.Flags().MarkHidden("dry-run")
	_ = cmd.Flags().MarkHidden("force")

	return cmd
}

func runCombine(ctx *app.Context, selectedStacks []string, skipCI bool, wait bool, yes bool) error {
	// If no stacks specified and interactive mode, show available stacks
	if len(selectedStacks) == 0 && ctx.Interactive && !yes {
		stacks, err := merge.DiscoverStacks(ctx.Engine)
		if err != nil {
			return fmt.Errorf("failed to discover stacks: %w", err)
		}

		if len(stacks) == 0 {
			ctx.Output.Warn("No independent stacks found rooted at trunk")
			return nil
		}

		// Show available stacks
		ctx.Output.Info("Available stacks:")
		for _, stack := range stacks {
			label := fmt.Sprintf("  • %s (%d branches", stack.RootBranch, len(stack.AllBranches))
			if stack.PRCount > 0 {
				label += fmt.Sprintf(", %d PRs", stack.PRCount)
			}
			if stack.Scope != "" {
				label += fmt.Sprintf(", scope: %s", stack.Scope)
			}
			label += ")"
			ctx.Output.Info("%s", label)
		}
		ctx.Output.Newline()
		ctx.Output.Info("Selecting all stacks (use --stacks to specify specific ones)")
	}

	// Execute multi-stack merge
	result, err := merge.ExecuteMultiStack(ctx, merge.MultiStackOptions{
		SelectedStacks: selectedStacks,
		SkipLocalCI:    skipCI,
		Wait:           wait,
	})
	if err != nil {
		return err
	}

	// Show final result
	if result.PRNumber > 0 {
		ctx.Output.Success("Created PR #%d: %s", result.PRNumber, result.PRURL)
	}

	return nil
}
