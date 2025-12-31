package navigation

import (
	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/app"
	"stackit.dev/stackit/internal/cli/common"
)

// NewBottomCmd creates the bottom command
func NewBottomCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bottom",
		Short: "Switch to the branch closest to trunk in the current stack",
		Long: `Switch to the branch closest to trunk in the current stack.

This command navigates down the parent chain from the current branch until
it reaches the first branch that has trunk as its parent (or trunk itself).`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return common.Run(cmd, func(ctx *app.Context) error {
				// Execute bottom action
				return actions.SwitchBranchAction(actions.DirectionBottom, ctx)
			})
		},
	}

	return cmd
}
