// Package stack provides CLI commands for operating on entire stacks.
package stack

import (
	"fmt"

	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/actions/move"
	"stackit.dev/stackit/internal/app"
	"stackit.dev/stackit/internal/cli/common"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/tui"
)

// NewMoveCmd creates the move command
func NewMoveCmd() *cobra.Command {
	var (
		all    bool
		dryRun bool
		onto   string
		source string
		yes    bool
	)

	cmd := &cobra.Command{
		Use:   "move",
		Short: "Move this branch to a different parent in the stack",
		Long: `Move a branch to a new parent, rebasing only its own commits onto the target.

This command changes the branch's parent pointer and rebases the branch's commits
onto the new parent. Only the branch's own commits are moved - commits from
ancestor branches are NOT included.

After moving, all descendant branches are automatically restacked.

Examples:
  stackit move --onto main           # Move current branch to be a child of main
  stackit move --source feature-b --onto feature-a  # Move feature-b onto feature-a

If no --onto is specified, opens an interactive selector to choose the target.`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return common.Run(cmd, func(ctx *app.Context) error {
				// Default source to current branch
				sourceBranch := source
				if sourceBranch == "" {
					currentBranch := ctx.Engine.CurrentBranch()
					if currentBranch == nil {
						return fmt.Errorf("not on a branch and no source branch specified")
					}
					sourceBranch = currentBranch.GetName()
				}

				// Handle interactive selection for onto if not provided
				ontoBranch := onto
				if ontoBranch == "" {
					// Dry-run requires --onto to be specified
					if dryRun {
						return fmt.Errorf("--onto flag is required when using --dry-run")
					}
					var err error
					ontoBranch, err = interactiveOntoSelection(ctx, sourceBranch)
					if err != nil {
						return err
					}
				}

				// Create runner and handler
				runner, handler := NewMoveUI(ctx.Output, ctx.Logger, ctx.Interactive)
				if runner != nil {
					defer runner.Cleanup()
				}

				// Run move action
				return move.Action(ctx, move.Options{
					Source:      sourceBranch,
					Onto:        ontoBranch,
					SkipConfirm: yes,
					DryRun:      dryRun,
				}, handler)
			})
		},
	}

	// Add flags
	cmd.Flags().BoolVarP(&all, "all", "a", false, "Show branches across all configured trunks in interactive selection.")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would happen without making changes.")
	cmd.Flags().StringVarP(&onto, "onto", "o", "", "Branch to move the current branch onto.")
	cmd.Flags().StringVar(&source, "source", "", "Branch to move (defaults to current branch).")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt.")

	_ = cmd.RegisterFlagCompletionFunc("onto", common.CompleteBranches)
	_ = cmd.RegisterFlagCompletionFunc("source", common.CompleteBranches)

	return cmd
}

// interactiveOntoSelection shows an interactive branch selector for choosing the "onto" branch
// This uses a compact inline bubbletea model that combines selection and confirmation.
func interactiveOntoSelection(ctx *app.Context, sourceBranch string) (string, error) {
	eng := ctx.Engine

	// Get descendants of source (will be restacked after move)
	graph := engine.BuildStackGraph(eng, engine.SortStrategyAlphabetical, nil)
	source := eng.GetBranch(sourceBranch)
	descendants := graph.Range(source, engine.StackRange{
		RecursiveChildren: true,
		IncludeCurrent:    true,
		RecursiveParents:  false,
	})

	// Get info needed for validation
	oldParent := source.GetParent()
	var oldParentRev string
	if meta, err := eng.Git().ReadMetadata(sourceBranch); err == nil && meta.ParentBranchRevision != nil {
		oldParentRev = *meta.ParentBranchRevision
	}

	// Create validation function that checks for conflicts when moving to a branch
	validator := func(ontoBranch string) (*tui.MoveValidation, error) {
		// Build rebase specs for this potential move
		rebaseSpecs := move.BuildRebaseSpecs(eng, ctx.Output, sourceBranch, ontoBranch, oldParent, oldParentRev, descendants)

		// Validate the rebases
		validation, err := eng.ValidateRebases(ctx.Context, rebaseSpecs)
		if err != nil {
			return nil, err
		}

		// Get commits that will be moved
		commits, _ := eng.GetAllCommits(source, engine.CommitFormatSubject)

		if !validation.Success {
			return &tui.MoveValidation{
				Valid:          false,
				Message:        fmt.Sprintf("Conflicts on %s: %s", validation.FailedBranch, validation.ErrorMessage),
				Commits:        commits,
				HasConflicts:   true,
				ConflictBranch: validation.FailedBranch,
				ConflictError:  validation.ErrorMessage,
			}, nil
		}

		return &tui.MoveValidation{
			Valid:   true,
			Message: "Move will complete without conflicts",
			Commits: commits,
		}, nil
	}

	// Use the compact move model with selection + confirmation
	config := tui.MoveModelConfig{
		SourceBranch: sourceBranch,
		Descendants:  descendants,
		OldParent:    oldParent,
		OldParentRev: oldParentRev,
		Validator:    validator,
	}

	selected, err := tui.PromptMoveSelect(ctx.Engine, config)
	if err != nil {
		return "", err
	}

	return selected, nil
}
