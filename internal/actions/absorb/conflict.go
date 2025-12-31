package absorb

import (
	"fmt"
	"strings"

	"stackit.dev/stackit/internal/app"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/git"
)

const (
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
