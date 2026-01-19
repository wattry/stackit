// Package worktree provides CLI commands for managing stackit-managed worktrees.
package worktree

import (
	"fmt"

	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/actions/worktree"
	"stackit.dev/stackit/internal/app"
	"stackit.dev/stackit/internal/cli/common"
	"stackit.dev/stackit/internal/tui"
	"stackit.dev/stackit/internal/tui/style"
)

// NewWorktreeCmd creates the worktree command group
func NewWorktreeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "worktree",
		Short: "Manage stackit-managed worktrees",
		Long: `Manage stackit-managed worktrees.

Worktrees allow you to work on multiple stacks in parallel, each in its own
directory. Create a worktree with 'stackit worktree create' from trunk.`,
	}

	cmd.AddCommand(newCreateCmd())
	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newRemoveCmd())
	cmd.AddCommand(newOpenCmd())

	return cmd
}

// newCreateCmd creates the worktree create command
func newCreateCmd() *cobra.Command {
	var (
		scope string
		open  bool
	)

	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new worktree",
		Long: `Create a new stackit-managed worktree.

Creates a worktree with an anchor branch that tracks trunk. The anchor branch
serves as the base for stacked branches created within the worktree.

The worktree will be created in a sibling directory to your repository.

Use --open to automatically change to the new worktree directory after creation.
In interactive mode, you will be prompted to open the worktree if --open is not specified.`,
		SilenceUsage: true,
		Args:         cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return common.Run(cmd, func(ctx *app.Context) error {
				result, err := worktree.CreateAction(ctx, worktree.CreateOptions{
					Name:  args[0],
					Scope: scope,
				})
				if err != nil {
					return err
				}

				// Determine if we should open the worktree
				shouldOpen := open
				if !shouldOpen && tui.IsTTY() {
					// Prompt in interactive mode
					confirmed, promptErr := tui.PromptConfirm("Change to worktree directory now?", true)
					if promptErr == nil && confirmed {
						shouldOpen = true
					}
				}

				if shouldOpen && result.Path != "" {
					if common.HasShellIntegration() {
						ctx.Output.DirectiveCD(result.Path)
					} else {
						ctx.Output.Tip("cd %s", result.Path)
					}
				} else if result.Path != "" {
					// User declined or non-interactive without --open
					ctx.Output.Tip("Open with: cd $(stackit worktree open %s)", result.Name)
				}

				return nil
			})
		},
	}

	cmd.Flags().StringVarP(&scope, "scope", "s", "", "Scope to apply to all branches in this worktree")
	cmd.Flags().BoolVarP(&open, "open", "o", false, "Open the worktree after creation (change to its directory)")

	return cmd
}

// newListCmd creates the worktree list command
func newListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all managed worktrees",
		Long: `List all stackit-managed worktrees.

Shows each worktree's anchor branch and path, with an indicator if the
worktree directory no longer exists on disk.`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return common.Run(cmd, func(ctx *app.Context) error {
				result, err := worktree.ListAction(ctx, worktree.ListOptions{})
				if err != nil {
					return err
				}

				if len(result.Worktrees) == 0 {
					ctx.Output.Info("No managed worktrees found.")
					ctx.Output.Tip("Create one with: stackit worktree create <name>")
					return nil
				}

				ctx.Output.Info("Managed worktrees:")
				for _, wt := range result.Worktrees {
					stackName := style.ColorBranchName(wt.AnchorBranch, false)
					if wt.Exists {
						ctx.Output.Print(fmt.Sprintf("  %s %s", stackName, style.ColorDim(wt.Path)))
					} else {
						ctx.Output.Print(fmt.Sprintf("  %s %s %s", stackName, style.ColorDim(wt.Path), style.ColorRed("(missing)")))
					}
				}
				ctx.Output.Newline()

				return nil
			})
		},
	}

	return cmd
}

// newRemoveCmd creates the worktree remove command
func newRemoveCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "remove <name-or-anchor-branch>",
		Short: "Remove a managed worktree",
		Long: `Remove a stackit-managed worktree.

This removes both the worktree directory and unregisters it from stackit.
The stack's branches remain intact. You can specify either the worktree
name or the anchor branch name.`,
		SilenceUsage: true,
		Args:         cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return common.Run(cmd, func(ctx *app.Context) error {
				return worktree.RemoveAction(ctx, worktree.RemoveOptions{
					AnchorBranch: args[0],
					Force:        force,
				})
			})
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Force removal even if there are errors")

	return cmd
}

// newOpenCmd creates the worktree open command
func newOpenCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "open <name-or-anchor-branch>",
		Short: "Open a worktree (with shell integration) or print its path",
		Long: `Open a stackit-managed worktree.

You can specify either the worktree name or the anchor branch name.

With shell integration enabled, this command will automatically change
your working directory to the worktree. Without shell integration, it
prints the path for use with cd:

  cd $(stackit worktree open my-feature)

To enable shell integration, add to your shell config:

  eval "$(stackit shell zsh)"   # or bash/fish`,
		SilenceUsage: true,
		Args:         cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return common.Run(cmd, func(ctx *app.Context) error {
				path, err := worktree.OpenAction(ctx, worktree.OpenOptions{
					AnchorBranch: args[0],
				})
				if err != nil {
					return err
				}

				// Always print the path for scripting compatibility (cd $(stackit worktree open foo))
				ctx.Output.Print(path)
				ctx.Output.Newline()

				// Also emit directive for shell integration auto-cd
				if common.HasShellIntegration() {
					ctx.Output.DirectiveCD(path)
				}
				return nil
			})
		},
	}

	return cmd
}
