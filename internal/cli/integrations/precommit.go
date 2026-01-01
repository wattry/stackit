package integrations

import (
	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/actions/integrations"
	"stackit.dev/stackit/internal/app"
	"stackit.dev/stackit/internal/cli/common"
)

func NewPrecommitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "precommit",
		Short: "Manage and run pre-commit hooks",
		Long:  `Manage and run git pre-commit hooks that validate Stackit state.`,
	}

	cmd.AddCommand(newPrecommitInstallCmd())
	cmd.AddCommand(newPrecommitVerifyCmd())

	return cmd
}

func newPrecommitInstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install",
		Short: "Install the pre-commit hook into the current repository",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return common.Run(cmd, func(ctx *app.Context) error {
				return integrations.PrecommitInstallAction(ctx)
			})
		},
	}
}

func newPrecommitVerifyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "verify",
		Short: "Verify that the current branch is not locked or frozen",
		Long: `Verify that the current branch is not locked or frozen.
Exits with a non-zero exit code if the branch is restricted.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return common.Run(cmd, func(ctx *app.Context) error {
				return integrations.PrecommitVerifyAction(ctx)
			})
		},
	}
}
