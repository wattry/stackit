package split

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/AlecAivazis/survey/v2"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/app"
	"stackit.dev/stackit/internal/engine"
	sterrors "stackit.dev/stackit/internal/errors"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/output"
	"stackit.dev/stackit/internal/tui"
	"stackit.dev/stackit/internal/tui/style"
	"stackit.dev/stackit/internal/utils"
)

// splitByHunkEngine is a minimal interface needed for splitting by hunk
type splitByHunkEngine interface {
	engine.BranchReader
	engine.BranchWriter
	engine.PRManager
	engine.StackRewriter
}

// splitByHunk splits a branch by interactively staging hunks.
//
// Algorithm:
//  1. Detach HEAD and soft reset to the parent branch's tip, leaving changes unstaged.
//  2. Loop until no unstaged changes remain:
//     a. Show remaining unstaged changes.
//     b. Interactively prompt the user to stage hunks for the next branch.
//     c. Prompt for a commit message and branch name.
//     d. Create a new commit with the staged changes.
//  3. Return the created branch names.
func splitByHunk(ctx context.Context, branchToSplit engine.Branch, eng splitByHunkEngine, splog output.Output) (*Result, error) {
	// Detach and reset branch changes
	if err := eng.DetachAndResetBranchChanges(ctx, branchToSplit.GetName()); err != nil {
		return nil, fmt.Errorf("failed to detach and reset: %w", err)
	}

	branchNames := []string{}

	// Get default commit message
	commitMessages, err := branchToSplit.GetAllCommits(engine.CommitFormatMessage)
	if err != nil {
		return nil, fmt.Errorf("failed to get commit messages: %w", err)
	}
	defaultCommitMessage := strings.Join(commitMessages, "\n\n")

	// Show instructions
	splog.Info("Splitting %s into multiple single-commit branches.", style.ColorBranchName(branchToSplit.GetName(), true))
	branch := eng.GetBranch(branchToSplit.GetName())
	prInfo, _ := branch.GetPrInfo()
	if prInfo != nil && prInfo.Number() != nil {
		splog.Info("If any of the new branches keeps the name %s, it will be linked to PR #%d.",
			style.ColorBranchName(branchToSplit.GetName(), true), *prInfo.Number())
	}
	splog.Info("")
	splog.Info("For each branch you'd like to create:")
	splog.Info("1. Follow the prompts to stage the changes that you'd like to include.")
	splog.Info("2. Enter a commit message.")
	splog.Info("3. Pick a branch name.")
	splog.Info("The command will continue until all changes have been added to a new branch.")
	splog.Info("")

	// Loop while there are unstaged changes
	for {
		hasUnstaged, err := eng.HasUnstagedChanges(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to check unstaged changes: %w", err)
		}
		if !hasUnstaged {
			break
		}

		// Show remaining changes
		unstagedDiff, err := eng.GetUnstagedDiff(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get unstaged diff: %w", err)
		}
		splog.Info("Remaining changes:")
		splog.Info("  %s", strings.ReplaceAll(unstagedDiff, "\n", "\n  "))
		splog.Info("")

		splog.Info("Stage changes for branch %d:", len(branchNames)+1)

		// Stage patch interactively
		if err := eng.StagePatch(ctx); err != nil {
			// If user cancels, restore branch
			_ = eng.ForceCheckoutBranch(ctx, branchToSplit)
			return nil, fmt.Errorf("canceled: no new branches created")
		}

		// Check if anything was staged
		hasStaged, err := eng.HasStagedChanges(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to check staged changes: %w", err)
		}
		if !hasStaged {
			// Nothing was staged - ask user if they want to continue or cancel
			if utils.IsInteractive() {
				var continueChoice string
				prompt := &survey.Select{
					Message: "No changes staged. What would you like to do?",
					Options: []string{"Try again", "Cancel split"},
				}
				if err := survey.AskOne(prompt, &continueChoice); err != nil {
					// Ctrl+C during prompt - restore and exit
					_ = eng.ForceCheckoutBranch(ctx, branchToSplit)
					return nil, fmt.Errorf("canceled")
				}
				if strings.Contains(continueChoice, "Cancel") {
					_ = eng.ForceCheckoutBranch(ctx, branchToSplit)
					if len(branchNames) == 0 {
						return nil, fmt.Errorf("canceled: no new branches created")
					}
					return nil, fmt.Errorf("canceled")
				}
				// User chose to try again
				continue
			}
			// Non-interactive mode - just skip and continue
			splog.Info("No changes staged, skipping this branch.")
			continue
		}

		// Commit with message
		commitMessage := defaultCommitMessage
		var editMessage bool
		if !utils.IsInteractive() {
			// In non-interactive mode, use default message
			editMessage = false
		} else {
			prompt := &survey.Confirm{
				Message: "Edit commit message?",
				Default: true,
			}
			if err := survey.AskOne(prompt, &editMessage); err != nil {
				// If user cancels, restore branch
				_ = eng.ForceCheckoutBranch(ctx, branchToSplit)
				return nil, fmt.Errorf("canceled")
			}
		}

		if editMessage {
			// Get message from user
			msg, err := tui.OpenEditor(defaultCommitMessage, "COMMIT_EDITMSG-*")
			if err != nil {
				// If user cancels, restore branch
				_ = eng.ForceCheckoutBranch(ctx, branchToSplit)
				return nil, err
			}
			commitMessage = utils.CleanCommitMessage(msg)
		}

		// Create commit
		if err := eng.CommitWithOptions(ctx, git.CommitOptions{
			Message:  commitMessage,
			NoVerify: true, // Split hunk commits are internal, hooks usually shouldn't run
		}); err != nil {
			// If user cancels, restore branch
			_ = eng.ForceCheckoutBranch(ctx, branchToSplit)
			return nil, fmt.Errorf("failed to create commit: %w", err)
		}

		// Get branch name
		branchName, err := promptBranchName(branchNames, branchToSplit.GetName(), len(branchNames)+1, eng)
		if err != nil {
			// If user cancels, restore branch
			_ = eng.ForceCheckoutBranch(ctx, branchToSplit)
			return nil, err
		}
		branchNames = append(branchNames, branchName)
	}

	// Update branchToSplit to point to the last commit we created.
	// This is necessary because ApplySplitToCommits will use this branch name
	// to resolve commit SHAs using GetCommitSHA(branchToSplit, offset).
	// Since we've been creating commits in detached HEAD on top of the parent,
	// we need the original branch name to now point to the tip of our new commits.
	if err := eng.UpdateBranchRef(ctx, branchToSplit.GetName(), "HEAD"); err != nil {
		return nil, fmt.Errorf("failed to update branch reference: %w", err)
	}

	return &Result{
		BranchNames:  branchNames,
		BranchPoints: makeRange(len(branchNames)), // Each branch is a single commit
	}, nil
}

// splitByHunkWithHandler splits a branch by interactively staging hunks using an InteractiveHandler.
// This is the new wizard-based implementation that supports direction selection.
//
// For DirectionBelow (downstack) - current behavior:
//  1. Detach HEAD and soft reset to parent's tip
//  2. All changes become unstaged
//  3. User stages hunks for new branch
//  4. New branch created between parent and current
//
// For DirectionAbove (upstack) - new behavior:
//  1. Stay on current branch commit
//  2. User stages hunks to EXTRACT (these will be removed from current)
//  3. Create child branch with staged changes
//  4. Remove staged changes from current branch
//  5. Existing children of current become children of new branch
func splitByHunkWithHandler(ctx *app.Context, branchToSplit engine.Branch, eng splitByHunkEngine, splog output.Output, handler InteractiveHandler, direction Direction) error {
	context := ctx.Context

	// For now, only support DirectionBelow (the current/default behavior)
	// DirectionAbove requires more complex logic that will be implemented later
	if direction == DirectionAbove {
		return fmt.Errorf("split above (upstack) is not yet implemented")
	}

	// Detach and reset branch changes
	if err := eng.DetachAndResetBranchChanges(context, branchToSplit.GetName()); err != nil {
		return fmt.Errorf("failed to detach and reset: %w", err)
	}

	branchNames := []string{}

	// Get default commit message
	commitMessages, err := branchToSplit.GetAllCommits(engine.CommitFormatMessage)
	if err != nil {
		return fmt.Errorf("failed to get commit messages: %w", err)
	}
	defaultCommitMessage := strings.Join(commitMessages, "\n\n")

	// Show instructions
	splog.Info("Splitting %s into multiple single-commit branches.", style.ColorBranchName(branchToSplit.GetName(), true))
	branch := eng.GetBranch(branchToSplit.GetName())
	prInfo, _ := branch.GetPrInfo()
	if prInfo != nil && prInfo.Number() != nil {
		splog.Info("If any of the new branches keeps the name %s, it will be linked to PR #%d.",
			style.ColorBranchName(branchToSplit.GetName(), true), *prInfo.Number())
	}
	splog.Info("")
	splog.Info("For each branch you'd like to create:")
	splog.Info("1. Follow the prompts to stage the changes that you'd like to include.")
	splog.Info("2. Enter a commit message.")
	splog.Info("3. Pick a branch name.")
	splog.Info("The command will continue until all changes have been added to a new branch.")
	splog.Info("")

	// Build list of existing branch names for validation
	existingBranchNames := make(map[string]bool)
	for _, b := range eng.AllBranches() {
		existingBranchNames[b.GetName()] = true
	}

	// Helper to restore branch and return appropriate error on cancel
	cancelWithRestore := func() error {
		_ = eng.ForceCheckoutBranch(context, branchToSplit)
		if len(branchNames) == 0 {
			return sterrors.ErrCanceled
		}
		return sterrors.ErrCanceled
	}

	// Loop while there are unstaged changes
	for {
		hasUnstaged, err := eng.HasUnstagedChanges(context)
		if err != nil {
			return fmt.Errorf("failed to check unstaged changes: %w", err)
		}
		if !hasUnstaged {
			break
		}

		// Show remaining changes via handler
		handler.OnStep(StepStagingHunks, StatusStarted, fmt.Sprintf("Stage changes for branch %d", len(branchNames)+1))

		unstagedDiff, err := eng.GetUnstagedDiff(context)
		if err != nil {
			return fmt.Errorf("failed to get unstaged diff: %w", err)
		}
		handler.ShowHunkSummary(unstagedDiff)

		// Stage patch interactively
		if err := eng.StagePatch(context); err != nil {
			return cancelWithRestore()
		}

		// Check if anything was staged
		hasStaged, err := eng.HasStagedChanges(context)
		if err != nil {
			return fmt.Errorf("failed to check staged changes: %w", err)
		}
		if !hasStaged {
			// Nothing was staged - ask user if they want to continue or cancel via handler
			continueAgain, err := handler.PromptContinueOrCancel()
			if err != nil {
				_ = eng.ForceCheckoutBranch(context, branchToSplit)
				return err
			}
			if !continueAgain {
				return cancelWithRestore()
			}
			continue
		}

		handler.OnStep(StepStagingHunks, StatusCompleted, "Changes staged")

		// Prompt for branch name BEFORE creating commit (so we can validate first)
		handler.OnStep(StepBranchName, StatusStarted, "Enter branch name")

		defaultName := generateDefaultBranchName(branchToSplit.GetName(), branchNames)
		branchName, err := handler.PromptBranchName(defaultName, branchNames, existingBranchNames, branchToSplit.GetName())
		if err != nil {
			_ = eng.ForceCheckoutBranch(context, branchToSplit)
			return err
		}

		handler.OnStep(StepBranchName, StatusCompleted, branchName)

		// Prompt for commit message via handler
		handler.OnStep(StepCommitMessage, StatusStarted, "Enter commit message")

		editMessage, err := handler.PromptEditCommitMessage()
		if err != nil {
			_ = eng.ForceCheckoutBranch(context, branchToSplit)
			return err
		}

		commitMessage := defaultCommitMessage
		if editMessage {
			commitMessage, err = handler.PromptCommitMessage(defaultCommitMessage)
			if err != nil {
				_ = eng.ForceCheckoutBranch(context, branchToSplit)
				return err
			}
		}

		handler.OnStep(StepCommitMessage, StatusCompleted, "Commit message set")

		// Create commit (after all validation passed)
		if err := eng.CommitWithOptions(context, git.CommitOptions{
			Message:  commitMessage,
			NoVerify: true, // Split hunk commits are internal, hooks usually shouldn't run
		}); err != nil {
			_ = eng.ForceCheckoutBranch(context, branchToSplit)
			return fmt.Errorf("failed to create commit: %w", err)
		}

		// Track the new branch name so it's not reused
		existingBranchNames[branchName] = true
		branchNames = append(branchNames, branchName)
		handler.OnBranchCreated(branchName)
	}

	// Update branchToSplit to point to the last commit we created
	if err := eng.UpdateBranchRef(context, branchToSplit.GetName(), "HEAD"); err != nil {
		return fmt.Errorf("failed to update branch reference: %w", err)
	}

	// Apply the split
	result := &Result{
		BranchNames:  branchNames,
		BranchPoints: makeRange(len(branchNames)),
	}

	// Get upstack branches (children)
	upstackRng := engine.StackRange{
		RecursiveParents:  false,
		IncludeCurrent:    false,
		RecursiveChildren: true,
	}
	upstackGraph := engine.BuildStackGraph(eng, engine.SortStrategyAlphabetical, nil)
	upstackBranches := upstackGraph.Range(branchToSplit, upstackRng)

	// Apply the split to commits
	if err := eng.ApplySplitToCommits(context, engine.ApplySplitOptions{
		BranchToSplit: branchToSplit.GetName(),
		BranchNames:   result.BranchNames,
		BranchPoints:  result.BranchPoints,
		AsSibling:     false,
	}); err != nil {
		return fmt.Errorf("failed to apply split: %w", err)
	}

	// Restack upstack branches
	if len(upstackBranches) > 0 {
		if err := actions.RestackBranches(ctx, upstackBranches); err != nil {
			return fmt.Errorf("failed to restack upstack branches: %w", err)
		}
	}

	handler.Complete(ActionResult{
		OriginalBranch: branchToSplit.GetName(),
		NewBranches:    result.BranchNames,
		Style:          StyleHunk,
	})

	return nil
}

// generateDefaultBranchName generates a default branch name for a new split branch.
// It returns "{originalName}_split", or "{originalName}_split_N" if that's already taken.
func generateDefaultBranchName(originalName string, existingNames []string) string {
	// First try the simple suffix
	candidate := originalName + "_split"
	if !slices.Contains(existingNames, candidate) {
		return candidate
	}

	// If that's taken, try numbered suffixes
	for suffix := 2; suffix <= 1000; suffix++ {
		candidate = fmt.Sprintf("%s_split_%d", originalName, suffix)
		if !slices.Contains(existingNames, candidate) {
			return candidate
		}
	}

	// Fallback (should never happen in practice)
	return fmt.Sprintf("%s_split_%d", originalName, len(existingNames)+1)
}
