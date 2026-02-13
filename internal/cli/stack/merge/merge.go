// Package merge provides CLI commands for merging stacked PRs.
package merge

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	mergeAction "stackit.dev/stackit/internal/actions/merge"
	"stackit.dev/stackit/internal/app"
	"stackit.dev/stackit/internal/cli/common"
)

// PostMergeHandler handles post-merge actions like syncing trunk.
// This is injected by the parent package to avoid circular dependencies.
type PostMergeHandler func(ctx *app.Context, action mergeAction.PostMergeAction) error

// NewMergeCmd creates the merge command with subcommands.
// postMergeHandler is called when a post-merge action is required (e.g., syncing trunk).
func NewMergeCmd(postMergeHandler PostMergeHandler) *cobra.Command {
	var (
		dryRun bool
		force  bool
		wait   bool
	)

	cmd := &cobra.Command{
		Use:   "merge",
		Short: "Merge pull requests for a stack",
		Long: `Merge pull requests associated with your stack.

When run without a subcommand, launches an interactive wizard to guide you
through the merge process.

Subcommands:
  status  Show shippability status of your stacks
  next    Merge the next (bottom-most) unmerged PR in the stack
  ship    Consolidate all branches into a single PR and merge atomically

Examples:
  stackit merge             # Launch interactive merge wizard
  stackit merge status      # Show your mergeable work
  stackit merge status --all # Show entire team's mergeable work
  stackit merge next        # Merge bottom PR, restack, stop
  stackit merge ship        # Consolidate all branches into single PR`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return common.Run(cmd, func(ctx *app.Context) error {
				// Must be interactive for wizard mode
				if !ctx.Interactive {
					return fmt.Errorf("merge wizard requires a TTY. Use 'merge next' or 'merge ship' for non-interactive mode")
				}

				runner, handler := NewMergeUI(ctx.Output, ctx.Logger)
				if runner != nil {
					defer runner.Cleanup()
				}

				// NewMergeUI returns an interactive handler when TTY is available
				// (which we've verified above), so this cast should always succeed
				interactiveHandler, ok := handler.(mergeAction.InteractiveHandler)
				if !ok {
					return fmt.Errorf("failed to initialize interactive handler")
				}

				err := mergeAction.RunWizard(ctx, interactiveHandler, mergeAction.WizardOptions{
					DryRun: dryRun,
					Force:  force,
					Wait:   wait,
				})

				// Handle post-merge follow-up action
				var postMerge *mergeAction.PostMergeActionRequired
				if errors.As(err, &postMerge) {
					if postMergeHandler != nil {
						return postMergeHandler(ctx, postMerge.Action)
					}
					return nil
				}
				return err
			})
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show merge plan without executing")
	cmd.Flags().BoolVar(&force, "force", false, "Skip validation checks (draft PRs, failing CI)")
	cmd.Flags().BoolVar(&wait, "wait", false, "Wait for merge to complete (default: fire-and-forget)")

	// Add subcommands
	cmd.AddCommand(NewStatusCmd())
	cmd.AddCommand(NewNextCmd(postMergeHandler))
	cmd.AddCommand(NewSquashCmd(postMergeHandler))

	return cmd
}
