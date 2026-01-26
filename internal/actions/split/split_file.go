package split

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/output"
	"stackit.dev/stackit/internal/tui"
	"stackit.dev/stackit/internal/tui/style"
	"stackit.dev/stackit/internal/utils"
)

// splitByFileEngine is a minimal interface needed for splitting by file
type splitByFileEngine interface {
	engine.BranchReader
	engine.BranchWriter
	engine.StackRewriter
}

// splitByFileOptions contains options for the splitByFile operation
type splitByFileOptions struct {
	// AsSibling creates the split branch as a sibling instead of a parent.
	// When true, the extracted files go to a new branch on the same parent,
	// and the original branch is unchanged (files are NOT removed).
	AsSibling bool
	// Name specifies a custom name for the split branch.
	// If empty, defaults to "{original}_split".
	Name string
	// Message specifies a custom commit message for the extraction.
	// If empty, auto-generates: "Extract {files} from {branch}"
	Message string
}

// recoverToOriginalBranch attempts to restore the user to the original branch after an error.
// It wraps the original error with recovery guidance if the checkout fails.
func recoverToOriginalBranch(ctx context.Context, eng splitByFileEngine, branch engine.Branch, originalErr error) error {
	if err := eng.ForceCheckoutBranch(ctx, branch); err != nil {
		// Include recovery instructions when checkout fails
		recoveryMsg := fmt.Sprintf("run 'git checkout %s' to recover", branch.GetName())
		return fmt.Errorf("%w (WARNING: failed to restore to %s: %s; %s)",
			originalErr, branch.GetName(), err.Error(), recoveryMsg)
	}
	return originalErr
}

// splitByFile splits a branch by extracting CHANGES to specified files to a new branch.
// Unlike the legacy behavior, this extracts only the diff hunks for the specified files,
// not the complete file contents. This is the correct semantic for "split by file".
//
// Default behavior (AsSibling=false):
// Creates a new PARENT branch containing the changes to the extracted files.
// The original branch becomes a child of the split branch.
// Algorithm:
//  1. Get the diff between parent and branch, parse into hunks
//  2. Filter hunks to only those for specified files
//  3. Detach HEAD and soft reset to parent (all changes become unstaged)
//  4. Stage the filtered hunks (changes go to new parent branch)
//  5. Stash staged changes
//  6. Commit remaining changes (stay on original branch)
//  7. Pop stash, commit on new parent branch
//  8. Update parent relationship
//
// Sibling mode (AsSibling=true):
// Creates a new SIBLING branch containing the changes to the extracted files.
// The original branch is unchanged (changes are NOT removed).
func splitByFile(ctx context.Context, branchToSplit engine.Branch, pathspecs []string, eng splitByFileEngine, opts splitByFileOptions) (*Result, error) {
	// Get parent branch
	parentBranchName := branchToSplit.GetParentPrecondition()
	parentBranch := eng.GetBranch(parentBranchName)

	// Generate new branch name
	newBranchName := opts.Name
	if newBranchName == "" {
		newBranchName = branchToSplit.GetName() + "_split"
	}
	allBranches := eng.AllBranches()
	branchNames := make([]string, len(allBranches))
	for i, b := range allBranches {
		branchNames[i] = b.GetName()
	}
	// Ensure unique name (only if we're auto-generating)
	if opts.Name == "" {
		for slices.Contains(branchNames, newBranchName) {
			newBranchName += "_split"
		}
	} else if slices.Contains(branchNames, newBranchName) {
		return nil, fmt.Errorf("branch %s already exists", newBranchName)
	}

	// Get the diff between parent and branchToSplit (raw output for parsing)
	diffOutput, err := eng.GetDiffBetween(ctx, parentBranchName, branchToSplit.GetName())
	if err != nil {
		return nil, fmt.Errorf("failed to get diff: %w", err)
	}

	// Parse diff into hunks
	allHunks, err := git.ParseDiffOutput(diffOutput)
	if err != nil {
		return nil, fmt.Errorf("failed to parse diff: %w", err)
	}

	// Filter hunks to only those for specified files
	filteredHunks := filterHunksByFiles(allHunks, pathspecs)
	if len(filteredHunks) == 0 {
		return nil, fmt.Errorf("no changes found for files: %s", strings.Join(pathspecs, ", "))
	}

	// Check for binary files and reject them
	var binaryFiles []string
	for _, h := range filteredHunks {
		if h.Binary {
			binaryFiles = append(binaryFiles, h.File)
		}
	}
	if len(binaryFiles) > 0 {
		return nil, fmt.Errorf("cannot split binary files: %s. Binary files must be split as whole files using a different approach",
			strings.Join(binaryFiles, ", "))
	}

	// Get commit message
	commitMessages, err := branchToSplit.GetAllCommits(engine.CommitFormatMessage)
	if err != nil {
		return nil, fmt.Errorf("failed to get commit messages: %w", err)
	}
	defaultCommitMessage := strings.Join(commitMessages, "\n\n")
	if defaultCommitMessage == "" {
		defaultCommitMessage = fmt.Sprintf("Split from %s", branchToSplit.GetName())
	}

	commitMessage := opts.Message
	if commitMessage == "" {
		commitMessage = defaultCommitMessage
	}

	// For sibling mode, use simpler approach - create branch at parent and apply hunks directly
	if opts.AsSibling {
		return splitByFileSibling(ctx, branchToSplit, parentBranch, newBranchName, filteredHunks, commitMessage, eng)
	}

	// Default mode: extract to parent branch

	// Detach and reset branch changes (all changes become unstaged)
	if err := eng.DetachAndResetBranchChanges(ctx, branchToSplit.GetName()); err != nil {
		return nil, fmt.Errorf("failed to detach and reset: %w", err)
	}

	// Stage the filtered hunks (these will go to the new parent branch)
	if err := eng.StageHunks(ctx, filteredHunks); err != nil {
		return nil, recoverToOriginalBranch(ctx, eng, branchToSplit,
			fmt.Errorf("failed to stage hunks: %w", err))
	}

	// Check if anything was staged
	hasStaged, err := eng.HasStagedChanges(ctx)
	if err != nil {
		return nil, recoverToOriginalBranch(ctx, eng, branchToSplit,
			fmt.Errorf("failed to check staged changes: %w", err))
	}
	if !hasStaged {
		return nil, recoverToOriginalBranch(ctx, eng, branchToSplit,
			fmt.Errorf("no changes staged for files: %s", strings.Join(pathspecs, ", ")))
	}

	// Check if there are remaining changes (to keep on branchToSplit)
	hasUnstaged, err := eng.HasUnstagedChanges(ctx)
	if err != nil {
		return nil, recoverToOriginalBranch(ctx, eng, branchToSplit,
			fmt.Errorf("failed to check unstaged changes: %w", err))
	}
	hasUntracked, err := eng.HasUntrackedFiles(ctx)
	if err != nil {
		return nil, recoverToOriginalBranch(ctx, eng, branchToSplit,
			fmt.Errorf("failed to check untracked files: %w", err))
	}
	if !hasUnstaged && !hasUntracked {
		return nil, recoverToOriginalBranch(ctx, eng, branchToSplit,
			fmt.Errorf("all changes were selected - nothing would remain on %s", branchToSplit.GetName()))
	}

	// Stash the staged changes (these will become the new parent branch content)
	stashName := fmt.Sprintf("stackit-split-file-parent-%d", time.Now().UnixNano())
	_, err = eng.StashPushStaged(ctx, stashName)
	if err != nil {
		_ = eng.ForceCheckoutBranch(ctx, branchToSplit)
		return nil, fmt.Errorf("failed to stash staged changes: %w", err)
	}

	// Track stash state for cleanup
	stashPopped := false
	cleanupStash := func() {
		if !stashPopped {
			_ = eng.StashPop(ctx)
			stashPopped = true
		}
	}

	// Stage and commit remaining changes - these stay on branchToSplit
	if err := eng.StageAll(ctx); err != nil {
		cleanupStash()
		return nil, recoverToOriginalBranch(ctx, eng, branchToSplit,
			fmt.Errorf("failed to stage remaining changes: %w", err))
	}

	if err := eng.CommitWithOptions(ctx, git.CommitOptions{
		Message:  defaultCommitMessage,
		NoVerify: true,
	}); err != nil {
		cleanupStash()
		return nil, recoverToOriginalBranch(ctx, eng, branchToSplit,
			fmt.Errorf("failed to commit remaining changes: %w", err))
	}

	// Update branchToSplit to point to this commit (contains remaining changes)
	if err := eng.UpdateBranchRef(ctx, branchToSplit.GetName(), "HEAD"); err != nil {
		cleanupStash()
		return nil, recoverToOriginalBranch(ctx, eng, branchToSplit,
			fmt.Errorf("failed to update branch reference: %w", err))
	}

	// Reset to original parent to create the new parent branch
	if err := eng.ResetHard(ctx, parentBranchName); err != nil {
		cleanupStash()
		return nil, recoverToOriginalBranch(ctx, eng, branchToSplit,
			fmt.Errorf("failed to reset to original parent: %w", err))
	}

	// Pop the stash to get the parent branch changes
	if err := eng.StashPop(ctx); err != nil {
		return nil, fmt.Errorf("failed to pop stash: %w. Recovery: run 'git stash pop' to restore changes", err)
	}
	stashPopped = true

	// Stage and commit - this becomes the NEW PARENT branch content
	if err := eng.StageAll(ctx); err != nil {
		// Changes are in the working tree after stash pop, not staged
		return nil, recoverToOriginalBranch(ctx, eng, branchToSplit,
			fmt.Errorf("failed to stage parent branch changes: %w; changes are in working tree", err))
	}

	if err := eng.CommitWithOptions(ctx, git.CommitOptions{
		Message:  commitMessage,
		NoVerify: true,
	}); err != nil {
		return nil, recoverToOriginalBranch(ctx, eng, branchToSplit,
			fmt.Errorf("failed to commit parent branch changes: %w", err))
	}

	// Create the new parent branch at HEAD
	if err := eng.CreateBranch(ctx, newBranchName, "HEAD"); err != nil {
		return nil, recoverToOriginalBranch(ctx, eng, branchToSplit,
			fmt.Errorf("failed to create parent branch: %w", err))
	}

	// Track the new parent branch with originalParent as its parent
	newBranch := eng.GetBranch(newBranchName)
	if err := eng.TrackBranch(ctx, newBranchName, parentBranchName); err != nil {
		return nil, recoverToOriginalBranch(ctx, eng, branchToSplit,
			fmt.Errorf("failed to track parent branch: %w", err))
	}

	// Update branchToSplit to have newBranch as its parent
	if err := eng.SetParent(ctx, branchToSplit, newBranch); err != nil {
		return nil, recoverToOriginalBranch(ctx, eng, branchToSplit,
			fmt.Errorf("failed to update parent of %s: %w", branchToSplit.GetName(), err))
	}

	// Checkout branchToSplit (we end up on the original branch)
	if err := eng.CheckoutBranch(ctx, branchToSplit); err != nil {
		return nil, fmt.Errorf("failed to checkout original branch: %w", err)
	}

	return &Result{
		BranchNames:  []string{newBranchName},
		BranchPoints: []int{0},
	}, nil
}

// splitByFileSibling creates a sibling branch with the specified file changes.
// The original branch is unchanged.
func splitByFileSibling(ctx context.Context, branchToSplit engine.Branch, parentBranch engine.Branch, newBranchName string, hunks []git.Hunk, commitMessage string, eng splitByFileEngine) (*Result, error) {
	// First checkout the parent branch so the new branch starts from there
	if err := eng.CheckoutBranch(ctx, parentBranch); err != nil {
		return nil, fmt.Errorf("failed to checkout parent branch: %w", err)
	}

	// Create new branch from parent
	newBranch := eng.GetBranch(newBranchName)
	if err := eng.CreateAndCheckoutBranch(ctx, newBranch); err != nil {
		_ = eng.CheckoutBranch(ctx, branchToSplit)
		return nil, fmt.Errorf("failed to create branch: %w", err)
	}

	// Stage the hunks directly
	if err := eng.StageHunks(ctx, hunks); err != nil {
		_ = eng.DeleteBranch(ctx, newBranch)
		_ = eng.CheckoutBranch(ctx, branchToSplit)
		return nil, fmt.Errorf("failed to stage hunks: %w", err)
	}

	// Check if anything was staged
	hasStaged, err := eng.HasStagedChanges(ctx)
	if err != nil {
		_ = eng.DeleteBranch(ctx, newBranch)
		_ = eng.CheckoutBranch(ctx, branchToSplit)
		return nil, fmt.Errorf("failed to check staged changes: %w", err)
	}
	if !hasStaged {
		_ = eng.DeleteBranch(ctx, newBranch)
		_ = eng.CheckoutBranch(ctx, branchToSplit)
		return nil, fmt.Errorf("no changes staged")
	}

	// Commit
	if err := eng.CommitWithOptions(ctx, git.CommitOptions{
		Message:  commitMessage,
		NoVerify: true,
	}); err != nil {
		_ = eng.DeleteBranch(ctx, newBranch)
		_ = eng.CheckoutBranch(ctx, branchToSplit)
		return nil, fmt.Errorf("failed to commit: %w", err)
	}

	// Track the new branch
	if err := eng.TrackBranch(ctx, newBranchName, parentBranch.GetName()); err != nil {
		_ = eng.DeleteBranch(ctx, newBranch)
		_ = eng.CheckoutBranch(ctx, branchToSplit)
		return nil, fmt.Errorf("failed to track branch: %w", err)
	}

	// Return to original branch
	if err := eng.CheckoutBranch(ctx, branchToSplit); err != nil {
		return nil, fmt.Errorf("failed to checkout original branch: %w", err)
	}

	return &Result{
		BranchNames:  []string{newBranchName},
		BranchPoints: []int{0},
	}, nil
}

// filterHunksByFiles filters hunks to only those affecting the specified files.
func filterHunksByFiles(hunks []git.Hunk, files []string) []git.Hunk {
	// Create a map for O(1) lookup
	fileSet := make(map[string]bool)
	for _, f := range files {
		fileSet[f] = true
	}

	var filtered []git.Hunk
	for _, h := range hunks {
		if fileSet[h.File] {
			filtered = append(filtered, h)
		}
	}
	return filtered
}

// promptForFiles shows an interactive file selector for split --by-file
func promptForFiles(ctx context.Context, branchToSplit engine.Branch, eng splitByFileEngine, splog output.Output, asSibling bool) ([]string, error) {
	if !utils.IsInteractive() {
		return nil, fmt.Errorf("file selection must be specified via pathspecs in non-interactive mode")
	}
	// Get the parent branch to compare against
	parentBranchName := branchToSplit.GetParentPrecondition()

	// Get merge base between branch and parent
	mergeBase, err := eng.GetMergeBase(branchToSplit.GetName(), parentBranchName)
	if err != nil {
		return nil, fmt.Errorf("failed to get merge base: %w", err)
	}

	// Get list of changed files
	changedFiles, err := eng.GetChangedFiles(ctx, mergeBase, branchToSplit.GetName())
	if err != nil {
		return nil, fmt.Errorf("failed to get changed files: %w", err)
	}

	if len(changedFiles) == 0 {
		return nil, fmt.Errorf("no files changed in branch %s", branchToSplit.GetName())
	}

	if len(changedFiles) == 1 {
		return nil, fmt.Errorf("only one file changed in branch - nothing to split")
	}

	// Show instructions
	splog.Info("Splitting %s by file.", style.ColorBranchName(branchToSplit.GetName(), true))
	if asSibling {
		splog.Info("Select the files to extract to a new sibling branch.")
		splog.Info("The original branch will remain unchanged.")
	} else {
		splog.Info("Select the files to extract to a new parent branch.")
		splog.Info("The remaining files will stay on %s.", style.ColorBranchName(branchToSplit.GetName(), true))
	}
	splog.Info("")

	// Prompt for file selection
	selectedFiles, err := tui.PromptMultiSelect("Select files to extract:", changedFiles)
	if err != nil {
		return nil, err
	}

	// Validate that not all files were selected (only in default mode where files are removed)
	if !asSibling && len(selectedFiles) == len(changedFiles) {
		return nil, fmt.Errorf("cannot extract all files - at least one must remain on the original branch")
	}

	return selectedFiles, nil
}
