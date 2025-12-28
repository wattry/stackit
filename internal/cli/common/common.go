// Package common provides shared helper functions for CLI commands.
package common

import (
	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/runtime"
)

// GetGlobalOptions returns runtime.GlobalOptions populated from a cobra.Command's flags
func GetGlobalOptions(cmd *cobra.Command) runtime.GlobalOptions {
	interactive, _ := cmd.Flags().GetBool("interactive")
	verify, _ := cmd.Flags().GetBool("verify")
	debug, _ := cmd.Flags().GetBool("debug")
	quiet, _ := cmd.Flags().GetBool("quiet")

	return runtime.GlobalOptions{
		Interactive: interactive,
		Verify:      verify,
		Debug:       debug,
		Quiet:       quiet,
	}
}

// Run is a helper that provides a runtime context to a command's execution function
func Run(cmd *cobra.Command, fn func(ctx *runtime.Context) error) error {
	opts := GetGlobalOptions(cmd)
	ctx, err := runtime.GetContext(cmd.Context(), opts)
	if err != nil {
		return err
	}
	return fn(ctx)
}

// CompleteBranches is a helper for cobra.ValidArgsFunction and RegisterFlagCompletionFunc
// that returns all branch names in the repository.
func CompleteBranches(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	if err := git.InitDefaultRepo(); err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	branches, err := git.GetAllBranchNames()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	return branches, cobra.ShellCompDirectiveNoFileComp
}
