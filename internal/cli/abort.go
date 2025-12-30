package cli

import (
	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/actions/absorb"
	"stackit.dev/stackit/internal/cli/common"
	"stackit.dev/stackit/internal/runtime"
)

// newAbortCmd creates the abort command
func newAbortCmd() *cobra.Command {
	var (
		force bool
	)

	cmd := &cobra.Command{
		Use:   "abort",
		Short: "Abort the current stackit command halted by a conflict",
		Long: `Aborts the current stackit command halted by a conflict.

This command cancels any in-progress operation (such as restack, sync, merge,
or absorb) that has been paused due to a conflict. Any changes made during
the operation will be rolled back.`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return common.Run(cmd, func(ctx *runtime.Context) error {
				// Check for absorb in progress first - it has its own abort logic
				if absorb.IsAbsorbInProgress(ctx) {
					return absorb.Abort(ctx)
				}

				// Otherwise use the standard abort action
				return actions.AbortAction(ctx, actions.AbortOptions{
					Force: force,
				})
			})
		},
	}

	// Add flags
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Do not prompt for confirmation; abort immediately.")

	return cmd
}
