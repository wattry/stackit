package branch

import (
	"fmt"

	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/app"
	"stackit.dev/stackit/internal/cli/common"
)

// NewFreezeCmd creates the freeze command
func NewFreezeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "freeze [branch]",
		Short: "Freeze specified branch and its downstack locally",
		Long: `Freeze specified branch and branches downstack of it locally.

Freezing a branch prevents local modifications (like modify, squash, absorb) and 
restacking. Unlike 'st lock', freezing is strictly local to your machine and is 
not shared with collaborators.

Use 'st freeze' when you want to stack on top of someone else’s PRs without 
accidentally modifying them or affecting their metadata. Frozen branches are 
automatically updated from remote via 'st sync' or 'st get' using hard-resets.

This operation can be undone with 'st unfreeze'.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return common.Run(cmd, func(ctx *app.Context) error {
				branchName := ""
				if len(args) > 0 {
					branchName = args[0]
				} else {
					current := ctx.Engine.CurrentBranch()
					if current == nil {
						return fmt.Errorf("not on a branch and no branch specified")
					}
					branchName = current.GetName()
				}

				return actions.FreezeAction(ctx, branchName)
			})
		},
	}
}
