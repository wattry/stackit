package stack

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/actions/foreach"
	"stackit.dev/stackit/internal/app"
	"stackit.dev/stackit/internal/cli/common"
	"stackit.dev/stackit/internal/engine"
)

// NewForeachCmd creates the foreach command
func NewForeachCmd() *cobra.Command {
	var (
		upstack    bool
		downstack  bool
		stack      bool
		branch     string
		noFailFast bool
		parallel   bool
		jobs       int
		jsonOutput bool
	)

	cmd := &cobra.Command{
		Use:   "foreach <command> [args...]",
		Short: "Run a shell command on each branch in the stack",
		Long: `Executes a shell command on each branch in the current stack, bottom-up.
The command is executed via /bin/sh -c.

By default, it runs on the current branch and all its descendants (up-stack).

Examples:
  st foreach mise run lint
  st foreach --stack 'go test ./... && go build'
  st foreach --downstack go test ./...
  st foreach --parallel mise run test`,
		Args:         cobra.MinimumNArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return common.Run(cmd, func(ctx *app.Context) error {
				opts := foreach.Options{
					Command:    args[0],
					Args:       args[1:],
					BranchName: branch,
					FailFast:   !noFailFast,
					Parallel:   parallel,
					Jobs:       jobs,
				}

				// Define the traversal range
				// If parallel mode is enabled and no explicit scope flags are set, default to --stack
				explicitScopeSet := cmd.Flags().Changed("stack") || cmd.Flags().Changed("downstack") || cmd.Flags().Changed("upstack")
				if parallel && !explicitScopeSet {
					// Parallel mode defaults to entire stack
					opts.Scope = engine.StackRange{
						IncludeCurrent:    true,
						RecursiveParents:  true,
						RecursiveChildren: true,
					}
				} else {
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
				}

				if jsonOutput {
					jsonHandler := foreach.NewJSONHandler()
					err := foreach.Action(ctx, opts, jsonHandler)
					if err != nil {
						jsonHandler.SetError(err)
					}

					data, marshalErr := json.MarshalIndent(jsonHandler.Result, "", "  ")
					if marshalErr != nil {
						return fmt.Errorf("failed to marshal JSON: %w", marshalErr)
					}
					ctx.Output.Info("%s", string(data))
					return err
				}

				// Create runner (manages terminal state) and handler (processes events)
				runner, handler := NewForeachUI(ctx.Output, ctx.Logger, opts.Parallel)
				defer runner.Cleanup()
				return foreach.Action(ctx, opts, handler)
			})
		},
	}

	cmd.Flags().BoolVar(&upstack, "upstack", true, "Run on current branch and descendants (default)")
	cmd.Flags().BoolVar(&downstack, "downstack", false, "Run on current branch and ancestors")
	cmd.Flags().BoolVar(&stack, "stack", false, "Run on the entire stack (ancestors and descendants)")
	cmd.Flags().StringVar(&branch, "branch", "", "Which branch to run this command from. Defaults to the current branch.")
	cmd.Flags().BoolVar(&noFailFast, "no-fail-fast", false, "Don't stop execution on the first failure")
	cmd.Flags().BoolVarP(&parallel, "parallel", "p", false, "Run commands in parallel using git worktrees")
	cmd.Flags().IntVarP(&jobs, "jobs", "j", 0, "Number of parallel jobs (default: number of CPUs)")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output results in JSON format.")

	return cmd
}
