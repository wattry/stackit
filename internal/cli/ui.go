package cli

import (
	"bytes"

	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/app"
	"stackit.dev/stackit/internal/cli/common"
	"stackit.dev/stackit/internal/tui/dashboard"
)

// newUICmd creates the ui command
func newUICmd(version, commit, date string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ui",
		Short: "Open the interactive stack dashboard",
		Long: `Open a live, interactive dashboard for your current stack.
From here you can see the state of your branches, PRs, and CI status,
and run commands directly from the dashboard.`,
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts := common.GetGlobalOptions(cmd)
			ctx, err := app.GetContext(cmd.Context(), opts)
			if err != nil {
				return err
			}

			return dashboard.Run(ctx, dashboard.Options{
				CommandFunc: func(args []string) (string, error) {
					// Create a fresh root command to avoid shared state issues
					rootCmd := NewRootCmd(version, commit, date)

					buf := new(bytes.Buffer)
					rootCmd.SetOut(buf)
					rootCmd.SetErr(buf)
					rootCmd.SetArgs(args)

					err := rootCmd.ExecuteContext(ctx.Context)
					return buf.String(), err
				},
			})
		},
	}

	return cmd
}
