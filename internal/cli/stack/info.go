package stack

import (
	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/app"
	"stackit.dev/stackit/internal/cli/common"
)

// newStackInfoCmd creates the stack info command
func newStackInfoCmd() *cobra.Command {
	var json bool

	cmd := &cobra.Command{
		Use:   "info",
		Short: "Display information about the entire stack",
		Long:  `Display information about all branches in the current stack.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return common.Run(cmd, func(ctx *app.Context) error {
				return actions.StackInfoAction(ctx, actions.StackInfoOptions{
					JSON: json,
				})
			})
		},
	}

	cmd.Flags().BoolVar(&json, "json", false, "Output in JSON format")

	return cmd
}
