package navigation

import (
	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/app"
	"stackit.dev/stackit/internal/cli/common"
)

// NewMainCmd creates the main command
func NewMainCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "main",
		Aliases: []string{"m"},
		Short:   "Switch to the main/trunk branch",
		Long: `Switch to the main/trunk branch.

Navigates to the configured trunk branch (typically "main" or "master").`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return common.Run(cmd, func(ctx *app.Context) error {
				_, err := common.Checkout(ctx, actions.CheckoutOptions{CheckoutTrunk: true}, nil)
				return err
			})
		},
	}
	return cmd
}
