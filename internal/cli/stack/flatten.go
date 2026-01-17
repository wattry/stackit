package stack

import (
	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/actions/flatten"
	"stackit.dev/stackit/internal/app"
	"stackit.dev/stackit/internal/cli/common"
)

// NewFlattenCmd creates the flatten command
func NewFlattenCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "flatten [branch]",
		Short: "Flatten the stack by moving branches closer to trunk where possible",
		Long: `Flatten the stack by re-parenting branches as close to trunk as possible.

The command analyzes each branch in the stack and tests whether it can be
rebased directly onto trunk (or an intermediate branch closer to trunk).
Branches that depend on changes from their parent will stay in place.

This is useful after landing PRs from the middle of a stack, or when you
have independent changes that were developed as a chain but don't actually
depend on each other.

Examples:
  stackit flatten           # Flatten stack from current branch
  stackit flatten feature   # Flatten stack from the 'feature' branch
  stackit flatten --yes     # Skip confirmation prompt`,
		Args:         cobra.MaximumNArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return common.Run(cmd, func(ctx *app.Context) error {
				// Determine target branch from positional argument
				var targetBranch string
				if len(args) > 0 {
					targetBranch = args[0]
				}

				// Create runner and handler
				runner, handler := NewFlattenUI(ctx.Output, ctx.Logger)
				if runner != nil {
					defer runner.Cleanup()
				}

				// Run flatten action
				return flatten.Action(ctx, flatten.Options{
					BranchName:  targetBranch,
					SkipConfirm: yes,
				}, handler)
			})
		},
	}

	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt.")

	return cmd
}
