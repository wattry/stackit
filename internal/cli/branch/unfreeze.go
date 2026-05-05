package branch

import (
	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/app"
	"stackit.dev/stackit/internal/cli/common"
	"stackit.dev/stackit/internal/errors"
)

// NewUnfreezeCmd creates the unfreeze command
func NewUnfreezeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "unfreeze [branch]",
		Short: "Unfreeze specified branch and its upstack locally",
		Long: `Unfreeze specified branch and branches upstack of it locally.

Unfreezing a branch re-enables local modifications and restacking. This only 
affects the local frozen status and does not affect shared locks.`,
		SilenceUsage: true,
		Args:         cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return common.Run(cmd, func(ctx *app.Context) error {
				branchName, err := common.ResolveBranchArg(ctx, args, errors.ErrNotOnBranchNoBranchSpecified)
				if err != nil {
					return err
				}

				return actions.UnfreezeAction(ctx, branchName)
			})
		},
	}
}
