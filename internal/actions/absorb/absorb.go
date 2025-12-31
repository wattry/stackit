// Package absorb provides functionality for absorbing staged changes into commits downstack.
package absorb

import (
	"fmt"
	"strings"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/app"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/tui"
	"stackit.dev/stackit/internal/tui/style"
)

// Options contains options for the absorb command
type Options struct {
	All    bool
	DryRun bool
	Force  bool
	Patch  bool
}

// Action performs the absorb operation
func Action(ctx *app.Context, opts Options) error {
	eng := ctx.Engine
	splog := ctx.Splog

	// Get current branch
	currentBranch := eng.CurrentBranch()
	if currentBranch == nil {
		return fmt.Errorf("not on a branch")
	}

	// Check if current branch is trunk
	if currentBranch.IsTrunk() {
		return fmt.Errorf("cannot absorb into trunk branch %s", currentBranch.GetName())
	}

	if err := currentBranch.EnsureCanModify(); err != nil {
		return err
	}

	// Take snapshot before modifying the repository
	snapshotOpts := actions.NewSnapshot("absorb",
		actions.WithFlag(opts.All, "--all"),
		actions.WithFlag(opts.DryRun, "--dry-run"),
		actions.WithFlag(opts.Force, "--force"),
		actions.WithFlag(opts.Patch, "--patch"),
	)
	if err := eng.TakeSnapshot(snapshotOpts); err != nil {
		// Log but don't fail - snapshot is best effort
		splog.Debug("Failed to take snapshot: %v", err)
	}

	// Check if rebase is in progress
	if err := ctx.Git().CheckRebaseInProgress(ctx.Context); err != nil {
		return err
	}

	// Check if there are staged changes (before handling flags)
	_, err := eng.HasStagedChanges(ctx.Context)
	if err != nil {
		return fmt.Errorf("failed to check staged changes: %w", err)
	}

	// Handle staging flags
	stagingOpts := git.StagingOptions{
		All:   opts.All,
		Patch: opts.Patch,
	}
	if err := ctx.Git().StageChanges(ctx.Context, stagingOpts); err != nil {
		return err
	}

	// Re-check staged changes after flags
	hasStaged, err := eng.HasStagedChanges(ctx.Context)
	if err != nil {
		return fmt.Errorf("failed to check staged changes: %w", err)
	}
	if !hasStaged {
		splog.Info("Nothing to absorb.")
		return nil
	}

	// Parse staged hunks
	hunks, err := eng.ParseStagedHunks(ctx.Context)
	if err != nil {
		return fmt.Errorf("failed to parse staged hunks: %w", err)
	}

	if len(hunks) == 0 {
		splog.Info("Nothing to absorb.")
		return nil
	}

	// Get all commits downstack from current branch
	// We need commits from all branches downstack, not just current branch
	downstackBranches := currentBranch.GetRelativeStackDownstack()
	// Include current branch
	downstackBranches = append([]engine.Branch{*currentBranch}, downstackBranches...)

	// Terminate downstack search if a scope boundary is hit
	currentScope := currentBranch.GetScope()
	if currentScope.IsDefined() {
		limitedDownstack := []engine.Branch{}
		for _, branch := range downstackBranches {
			if branch.IsTrunk() || !branch.GetScope().Equal(currentScope) {
				break
			}
			limitedDownstack = append(limitedDownstack, branch)
		}
		downstackBranches = limitedDownstack
	}

	// Get all commit SHAs from downstack branches (newest to oldest)
	commitSHAs := []string{}
	for _, branch := range downstackBranches {
		commits, err := branch.GetAllCommits(engine.CommitFormatSHA)
		if err != nil {
			return fmt.Errorf("failed to get commits for branch %s: %w", branch.GetName(), err)
		}
		// Commits are returned oldest to newest, but we want newest to oldest for search
		for i := len(commits) - 1; i >= 0; i-- {
			commitSHAs = append(commitSHAs, commits[i])
		}
	}

	// Find target commit for each hunk
	hunkTargets := []git.HunkTarget{}
	unabsorbedHunks := []git.Hunk{}

	for _, hunk := range hunks {
		commitSHA, commitIndex, err := eng.FindTargetCommitForHunk(hunk, commitSHAs)
		if err != nil {
			return fmt.Errorf("failed to find target commit for hunk: %w", err)
		}

		if commitSHA == "" {
			// Hunk commutes with all commits - can't be absorbed
			unabsorbedHunks = append(unabsorbedHunks, hunk)
			continue
		}

		hunkTargets = append(hunkTargets, git.HunkTarget{
			Hunk:        hunk,
			CommitSHA:   commitSHA,
			CommitIndex: commitIndex,
		})
	}

	// Group hunks by branch, then by commit
	hunksByBranch := make(map[string]map[string][]git.Hunk)
	for _, target := range hunkTargets {
		branchName, err := eng.FindBranchForCommit(target.CommitSHA)
		if err != nil {
			continue
		}
		if hunksByBranch[branchName] == nil {
			hunksByBranch[branchName] = make(map[string][]git.Hunk)
		}
		hunksByBranch[branchName][target.CommitSHA] = append(hunksByBranch[branchName][target.CommitSHA], target.Hunk)
	}

	// Check if any target branches are locked or frozen
	for branchName := range hunksByBranch {
		branch := eng.GetBranch(branchName)
		if err := branch.EnsureCanModify(); err != nil {
			return err
		}
	}

	if len(hunksByBranch) == 0 {
		if len(unabsorbedHunks) > 0 {
			splog.Warn("The following hunks could not be absorbed (they commute with all commits):")
			for _, hunk := range unabsorbedHunks {
				splog.Info("  %s (lines %d-%d)", hunk.File, hunk.NewStart, hunk.NewStart+hunk.NewCount-1)
			}
		} else {
			splog.Info("Nothing to absorb.")
		}
		return nil
	}

	// Print dry-run output or confirmation
	if opts.DryRun {
		// Flatten for printing
		flatHunksByCommit := make(map[string][]git.Hunk)
		for _, branchHunks := range hunksByBranch {
			for commitSHA, hunks := range branchHunks {
				flatHunksByCommit[commitSHA] = hunks
			}
		}
		printDryRunOutput(flatHunksByCommit, unabsorbedHunks, eng, splog)
		return nil
	}

	// Print what will be absorbed
	flatHunksByCommit := make(map[string][]git.Hunk)
	for _, branchHunks := range hunksByBranch {
		for commitSHA, hunks := range branchHunks {
			flatHunksByCommit[commitSHA] = hunks
		}
	}
	printAbsorbPlan(flatHunksByCommit, unabsorbedHunks, eng, splog)

	// Prompt for confirmation if not --force
	if !opts.Force && ctx.Interactive {
		confirmed, err := tui.PromptConfirm("Apply these changes to the commits?", false)
		if err != nil {
			return fmt.Errorf("confirmation canceled: %w", err)
		}
		if !confirmed {
			splog.Info("Absorb canceled")
			return nil
		}
	} else if !opts.Force && !ctx.Interactive {
		// Non-interactive without force: default to no
		splog.Info("Non-interactive mode: skipping absorb (use --force to override)")
		return nil
	}

	// Stash all changes (staged and unstaged) before starting to rewrite commits
	// This ensures a clean working directory for checkouts and prevents losing changes
	stashOutput, stashErr := eng.StashPush(ctx.Context, "stackit-absorb-temp")
	if stashErr == nil && !strings.Contains(stashOutput, "No local changes to save") {
		defer func() {
			// Restore stash after we're done
			_ = eng.StashPop(ctx.Context)
		}()
	}

	// Track the oldest modified branch to know where to start restacking from
	var oldestModifiedBranch string

	// Get branches in topological order (bottom-up)
	allBranches := eng.AllBranches()
	sortedBranches := eng.SortBranchesTopologically(allBranches)

	for _, branch := range sortedBranches {
		branchHunks, ok := hunksByBranch[branch.GetName()]
		if !ok {
			continue
		}

		if oldestModifiedBranch == "" {
			oldestModifiedBranch = branch.GetName()
		}

		// Apply all hunks for this branch together
		if err := eng.ApplyHunksToBranch(ctx.Context, branch, branchHunks); err != nil {
			return fmt.Errorf("failed to apply hunks to branch %s: %w", branch.GetName(), err)
		}

		for commitSHA := range branchHunks {
			splog.Info("Absorbed changes into commit %s in %s", commitSHA[:8], style.ColorBranchName(branch.GetName(), false))
		}
	}

	// Warn about unabsorbed hunks
	if len(unabsorbedHunks) > 0 {
		splog.Warn("The following hunks could not be absorbed (they commute with all commits):")
		for _, hunk := range unabsorbedHunks {
			splog.Info("  %s (lines %d-%d)", hunk.File, hunk.NewStart, hunk.NewStart+hunk.NewCount-1)
		}
	}

	// Refresh engine state after modifying branch references directly via git
	if err := eng.Rebuild(""); err != nil {
		return fmt.Errorf("failed to refresh engine after absorb: %w", err)
	}

	// Restack all branches above the oldest modified branch
	if oldestModifiedBranch != "" {
		branch := eng.GetBranch(oldestModifiedBranch)
		upstackBranches := branch.GetRelativeStackUpstack()

		if len(upstackBranches) > 0 {
			if err := actions.RestackBranches(ctx, upstackBranches); err != nil {
				return fmt.Errorf("failed to restack upstack branches: %w", err)
			}
		}
	}

	return nil
}

// printDryRunOutput prints what would be absorbed in dry-run mode
func printDryRunOutput(hunksByCommit map[string][]git.Hunk, unabsorbedHunks []git.Hunk, eng engine.Engine, splog *tui.Splog) {
	splog.Info("Would absorb the following changes:")
	splog.Newline()

	// Get commit info for display
	for commitSHA, hunks := range hunksByCommit {
		branchName, err := eng.FindBranchForCommit(commitSHA)
		if err != nil {
			branchName = unknown
		}

		// Get commit message - show first commit message from the branch
		branch := eng.GetBranch(branchName)
		commits, err := branch.GetAllCommits(engine.CommitFormatReadable)
		if err == nil && len(commits) > 0 {
			splog.Info("  %s in %s:", commitSHA[:8], style.ColorBranchName(branchName, false))
			splog.Info("    %s", commits[0])
		} else {
			splog.Info("  %s in %s:", commitSHA[:8], style.ColorBranchName(branchName, false))
		}

		for _, hunk := range hunks {
			splog.Info("    - %s (lines %d-%d)", hunk.File, hunk.NewStart, hunk.NewStart+hunk.NewCount-1)
		}
	}

	if len(unabsorbedHunks) > 0 {
		splog.Newline()
		splog.Warn("The following hunks would not be absorbed:")
		for _, hunk := range unabsorbedHunks {
			splog.Info("  %s (lines %d-%d)", hunk.File, hunk.NewStart, hunk.NewStart+hunk.NewCount-1)
		}
	}
}

// printAbsorbPlan prints the plan for absorbing changes
func printAbsorbPlan(hunksByCommit map[string][]git.Hunk, unabsorbedHunks []git.Hunk, eng engine.Engine, splog *tui.Splog) {
	splog.Info("Will absorb the following changes:")
	splog.Newline()

	for commitSHA, hunks := range hunksByCommit {
		branchName, err := eng.FindBranchForCommit(commitSHA)
		if err != nil {
			branchName = unknown
		}

		splog.Info("  Commit %s in %s:", commitSHA[:8], style.ColorBranchName(branchName, false))
		for _, hunk := range hunks {
			splog.Info("    - %s (lines %d-%d)", hunk.File, hunk.NewStart, hunk.NewStart+hunk.NewCount-1)
		}
	}

	if len(unabsorbedHunks) > 0 {
		splog.Newline()
		splog.Warn("The following hunks will not be absorbed:")
		for _, hunk := range unabsorbedHunks {
			splog.Info("  %s (lines %d-%d)", hunk.File, hunk.NewStart, hunk.NewStart+hunk.NewCount-1)
		}
	}
}

const (
	unknown           = "unknown"
	absorbStashMarker = "stackit-absorb-temp"
)

// IsAbsorbInProgress checks if there's a failed absorb operation that needs cleanup.
// This is detected by checking for:
// 1. Detached HEAD state, OR
// 2. Presence of absorb stash marker
func IsAbsorbInProgress(ctx *app.Context) bool {
	// Check for absorb stash marker
	stashList, _ := ctx.Engine.StashList(ctx.Context)
	if strings.Contains(stashList, absorbStashMarker) {
		return true
	}

	// Check for detached HEAD (might be mid-absorb failure)
	// engine.CurrentBranch() returns nil if in detached HEAD.
	if ctx.Engine.CurrentBranch() == nil {
		// In detached HEAD - could be absorb or something else
		// Check reflog for signs of absorb
		reflog, _ := ctx.Engine.GetReflog(ctx.Context, 5, "%gs")
		// If recent reflog shows checkout from a branch (not a commit), likely absorb
		if strings.Contains(reflog, "checkout: moving from") {
			return true
		}
	}

	return false
}

// ShowConflict shows information about what absorb would do and helps diagnose conflicts.
// This is useful when absorb fails and you want to understand what was being attempted.
func ShowConflict(ctx *app.Context) error {
	splog := ctx.Splog
	eng := ctx.Engine

	// Check if we're in a detached HEAD state (might be mid-absorb failure)
	if eng.CurrentBranch() == nil {
		splog.Warn("Repository is in detached HEAD state.")
		splog.Info("This may be from a failed absorb. Run 'stackit absorb --abort' to recover.")
		splog.Info("")

		// Show unmerged files if any
		unmerged, _ := eng.GetUnmergedFiles(ctx.Context)
		if len(unmerged) > 0 {
			splog.Info("Unmerged files:")
			for _, file := range unmerged {
				splog.Info("  - %s", file)
			}
			splog.Info("")
		}
		return nil
	}

	// Check for staged changes
	hasStaged, err := eng.HasStagedChanges(ctx.Context)
	if err != nil {
		return fmt.Errorf("failed to check staged changes: %w", err)
	}

	if !hasStaged {
		splog.Info("No staged changes to analyze.")
		splog.Info("Stage some changes first, then run 'stackit absorb --dry-run' to see where they would go.")
		return nil
	}

	// Show what absorb would do (dry run)
	splog.Info("Analyzing staged changes...\n")

	// Parse staged hunks
	hunks, err := eng.ParseStagedHunks(ctx.Context)
	if err != nil {
		return fmt.Errorf("failed to parse staged hunks: %w", err)
	}

	if len(hunks) == 0 {
		splog.Info("No hunks found in staged changes.")
		return nil
	}

	// Show the staged changes
	splog.Info("Staged changes:")
	fileHunks := make(map[string][]git.Hunk)
	for _, hunk := range hunks {
		fileHunks[hunk.File] = append(fileHunks[hunk.File], hunk)
	}
	for file, hunks := range fileHunks {
		splog.Info("  %s:", file)
		for _, hunk := range hunks {
			splog.Info("    lines %d-%d", hunk.NewStart, hunk.NewStart+hunk.NewCount-1)
		}
	}
	splog.Info("")

	// Get current branch info
	currentBranchObj := eng.CurrentBranch()
	if currentBranchObj == nil {
		return fmt.Errorf("not on a tracked branch")
	}

	// Get downstack commits
	downstackBranches := currentBranchObj.GetRelativeStackDownstack()
	downstackBranches = append([]engine.Branch{*currentBranchObj}, downstackBranches...)

	splog.Info("Stack (bottom to top):")
	for i := len(downstackBranches) - 1; i >= 0; i-- {
		branch := downstackBranches[i]
		marker := "  "
		if branch.GetName() == currentBranchObj.GetName() {
			marker = "* "
		}
		splog.Info("%s%s", marker, branch.GetName())
	}
	splog.Info("")

	splog.Info("To see where changes would be absorbed:")
	splog.Info("  stackit absorb --dry-run")
	splog.Info("")
	splog.Info("If absorb fails with conflicts, consider:")
	splog.Info("  1. Manually checkout the target branch and apply the change")
	splog.Info("  2. Split the change into smaller pieces with 'git add -p'")
	splog.Info("  3. Create a new commit instead with 'stackit create'")

	return nil
}

// Abort cleans up from a failed absorb state
func Abort(ctx *app.Context) error {
	splog := ctx.Splog
	eng := ctx.Engine

	// Check if we're in a detached HEAD state
	if eng.CurrentBranch() != nil {
		// Check for stashed changes from absorb
		stashList, _ := eng.StashList(ctx.Context)
		if strings.Contains(stashList, absorbStashMarker) {
			splog.Info("Found stashed changes from previous absorb attempt.")
			if err := eng.StashPop(ctx.Context); err != nil {
				splog.Warn("Failed to restore stashed changes: %v", err)
			} else {
				splog.Info("Restored your staged changes from stash.")
			}
			return nil
		}

		splog.Info("No absorb operation in progress.")
		return nil
	}

	splog.Info("Cleaning up failed absorb operation...")

	// Reset any uncommitted changes
	if err := eng.ResetHard(ctx.Context, "HEAD"); err != nil {
		splog.Warn("Failed to reset: %v", err)
	}

	// Find the original branch from reflog
	reflog, _ := eng.GetReflog(ctx.Context, 20, "%gs")
	var originalBranch string
	for _, line := range strings.Split(reflog, "\n") {
		if strings.Contains(line, "checkout: moving from") {
			parts := strings.Split(line, "moving from ")
			if len(parts) >= 2 {
				branchPart := strings.Split(parts[1], " to ")
				if len(branchPart) >= 1 {
					originalBranch = branchPart[0]
					break
				}
			}
		}
	}

	if originalBranch != "" && originalBranch != "HEAD" {
		branch := eng.GetBranch(originalBranch)
		if err := eng.CheckoutBranch(ctx.Context, branch); err != nil {
			splog.Warn("Failed to return to original branch %s: %v", originalBranch, err)
			splog.Info("You can manually checkout your branch with: git checkout <branch-name>")
		} else {
			splog.Info("Returned to branch %s", originalBranch)
		}
	} else {
		splog.Warn("Could not determine original branch from reflog.")
		splog.Info("Use 'git checkout <branch-name>' to return to your branch.")
	}

	// Pop stash if there is one
	stashList, _ := eng.StashList(ctx.Context)
	if strings.Contains(stashList, absorbStashMarker) {
		if err := eng.StashPop(ctx.Context); err != nil {
			splog.Warn("Failed to restore stashed changes: %v", err)
		} else {
			splog.Info("Restored your staged changes from stash.")
		}
	}

	splog.Info("Abort complete.")
	return nil
}
