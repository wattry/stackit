package branch

import (
	"fmt"

	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/cli/common"
	"stackit.dev/stackit/internal/runtime"
)

// NewLockCmd creates the lock command
func NewLockCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "lock [branch]",
		Short: "Lock specified branch and its downstack",
		Long: `Lock specified branch and branches downstack of it.

Locking a branch prevents local modifications to the branch including any restacks. 
You can still sync remote changes to the branch with st sync or st get. 
You can also build PRs on top of a locked branch. 

Locking can be useful when you want to stack on top of someone else’s PRs 
without making any changes to them. 

This operation can be undone with st unlock.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return common.Run(cmd, func(ctx *runtime.Context) error {
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

				return actions.LockAction(ctx, branchName)
			})
		},
	}
}

// NewUnlockCmd creates the unlock command
func NewUnlockCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "unlock [branch]",
		Short: "Unlock specified branch and its upstack",
		Long: `Unlock specified branch and branches upstack of it.

Locking a branch prevents local modifications to the branch including any restacks. 
Unlocking will enable local modifications to the branch.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return common.Run(cmd, func(ctx *runtime.Context) error {
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

				return actions.UnlockAction(ctx, branchName)
			})
		},
	}
}
