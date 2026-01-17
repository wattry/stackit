package stack

import (
	"fmt"

	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/actions/flatten"
	"stackit.dev/stackit/internal/app"
	"stackit.dev/stackit/internal/cli/common"
)

func NewFlattenCmd() *cobra.Command {
	var branch string

	cmd := &cobra.Command{
		Use:   "flatten [branch]",
		Short: "Flatten the stack by re-parenting branches closer to trunk where possible",
		Long:  "Flatten the stack by re-parenting branches closer to trunk where possible. It starts from the bottom of the stack and tries to rebase each branch onto the trunk or any intermediate branch, keeping the dependencies intact.",
		Args:  cobra.MaximumNArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return common.Run(cmd, func(ctx *app.Context) error {
				targetBranch := branch
				if len(args) > 0 {
					targetBranch = args[0]
				}

				if targetBranch == "" {
					currentBranch := ctx.Engine.CurrentBranch()
					if currentBranch == nil {
						return fmt.Errorf("not on a branch and --branch not specified")
					}
					targetBranch = currentBranch.GetName()
				}

				return flatten.FlattenAction(ctx, targetBranch)
			})
		},
	}

	cmd.Flags().StringVar(&branch, "branch", "", "Which branch to run this command from. Defaults to the current branch.")

	return cmd
}
