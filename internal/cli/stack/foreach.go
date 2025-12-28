package stack

import (
	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/cli/common"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/runtime"
)

// NewForeachCmd creates the foreach command
func NewForeachCmd() *cobra.Command {
	var (
		upstack    bool
		downstack  bool
		stack      bool
		noFailFast bool
		parallel   bool
		jobs       int
	)

	cmd := &cobra.Command{
		Use:   "foreach <command> [args...]",
		Short: "Run a shell command on each branch in the stack",
		Long: `Executes a shell command on each branch in the current stack, bottom-up.
The command is executed via /bin/sh -c.

By default, it runs on the current branch and all its descendants (up-stack).

Examples:
  st foreach just lint
  st foreach --stack 'go test ./... && go build'
  st foreach --downstack go test ./...
  st foreach --parallel just test`,
		Args:         cobra.MinimumNArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return common.Run(cmd, func(ctx *runtime.Context) error {
				opts := actions.ForeachOptions{
					Command:  args[0],
					Args:     args[1:],
					FailFast: !noFailFast,
					Parallel: parallel,
					Jobs:     jobs,
				}

				// Define the traversal range
				opts.Scope = engine.StackRange{IncludeCurrent: true}
				switch {
				case stack:
					opts.Scope.RecursiveParents = true
					opts.Scope.RecursiveChildren = true
				case downstack:
					opts.Scope.RecursiveParents = true
				case upstack:
					opts.Scope.RecursiveChildren = true
				}

				return actions.ForeachAction(ctx, opts)
			})
		},
	}

	cmd.Flags().BoolVar(&upstack, "upstack", true, "Run on current branch and descendants (default)")
	cmd.Flags().BoolVar(&downstack, "downstack", false, "Run on current branch and ancestors")
	cmd.Flags().BoolVar(&stack, "stack", false, "Run on the entire stack (ancestors and descendants)")
	cmd.Flags().BoolVar(&noFailFast, "no-fail-fast", false, "Don't stop execution on the first failure")
	cmd.Flags().BoolVarP(&parallel, "parallel", "p", false, "Run commands in parallel using git worktrees")
	cmd.Flags().IntVarP(&jobs, "jobs", "j", 0, "Number of parallel jobs (default: number of CPUs)")

	return cmd
}
