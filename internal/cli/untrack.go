package cli

import (
	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/actions/untrack"
	"stackit.dev/stackit/internal/app"
	"stackit.dev/stackit/internal/cli/common"
	"stackit.dev/stackit/internal/cli/stack"
	"stackit.dev/stackit/internal/errors"
	"stackit.dev/stackit/internal/utils"
)

// newUntrackCmd creates the untrack command
func newUntrackCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "untrack [branch]",
		Short: "Stop tracking a branch with stackit",
		Long: `Stop tracking the current (or provided) branch with stackit.
If the branch has children, they will also be untracked.`,
		SilenceUsage:      true,
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: common.CompleteBranches,
		RunE: func(cmd *cobra.Command, args []string) error {
			return common.Run(cmd, func(ctx *app.Context) error {
				// Get branch name from args or use current branch
				branchName := ""
				if len(args) > 0 {
					branchName = args[0]
				} else {
					currentBranch := ctx.Engine.CurrentBranch()
					if currentBranch == nil {
						return errors.ErrNotOnBranch
					}
					branchName = currentBranch.GetName()
				}

				// Execute untrack action
				handler := stack.NewUntrackUI(ctx.Output, utils.IsInteractive())
				return untrack.Action(ctx, untrack.Options{
					BranchName: branchName,
					Force:      force,
				}, handler)
			})
		},
	}

	// Add flags
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Will not prompt for confirmation before untracking a branch with children")

	return cmd
}
