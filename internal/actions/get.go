package actions

import (
	"fmt"
	"strconv"

	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/internal/tui/style"
	"stackit.dev/stackit/internal/utils"
)

// GetOptions contains options for the get command
type GetOptions struct {
	Downstack bool // Don't sync upstack branches if branch exists locally
	Force     bool // Overwrite all fetched branches with remote source of truth
	Restack   bool // Restack after syncing (default true)
	Unfrozen  bool // Checkout new branches as unfrozen
}

// GetAction performs the get operation
func GetAction(ctx *runtime.Context, branchOrPR string, opts GetOptions) error {
	eng := ctx.Engine
	splog := ctx.Splog
	gctx := ctx.Context

	if utils.HasUncommittedChanges(gctx) {
		return fmt.Errorf("you have uncommitted changes. Please commit or stash them before running 'get'")
	}

	targetBranch := ""
	if branchOrPR == "" {
		current := eng.CurrentBranch()
		if current == nil {
			return fmt.Errorf("not on a branch and no branch/PR specified")
		}
		targetBranch = current.GetName()
	} else {
		// Check if it's a PR number
		if prNum, err := strconv.Atoi(branchOrPR); err == nil {
			if ctx.GitHubClient == nil {
				return fmt.Errorf("GitHub client not configured; cannot resolve PR #%d", prNum)
			}
			owner, repo := ctx.GitHubClient.GetOwnerRepo()
			pr, err := ctx.GitHubClient.GetPullRequest(gctx, owner, repo, prNum)
			if err != nil {
				return fmt.Errorf("failed to get PR #%d: %w", prNum, err)
			}
			targetBranch = pr.Head
		} else {
			targetBranch = branchOrPR
		}
	}

	splog.Info("Syncing stack for %s...", style.ColorBranchName(targetBranch, true))

	// Fetch target branch from origin
	remote := eng.GetRemote()
	if _, err := eng.RunGitCommandWithContext(gctx, "fetch", remote, targetBranch); err != nil {
		return fmt.Errorf("failed to fetch branch %s from %s: %w", targetBranch, remote, err)
	}

	// Identify branches to sync (ancestors + descendants)
	branchesToSync := []string{targetBranch}
	parentMap := make(map[string]string)

	// Crawl ancestors using GitHub PR info if possible
	if ctx.GitHubClient != nil {
		current := targetBranch
		owner, repo := ctx.GitHubClient.GetOwnerRepo()
		for {
			pr, err := ctx.GitHubClient.GetPullRequestByBranch(gctx, owner, repo, current)
			if err != nil || pr == nil {
				break
			}

			base := pr.Base
			if base == "" || base == eng.Trunk().GetName() {
				parentMap[current] = eng.Trunk().GetName()
				break
			}

			parentMap[current] = base
			if !contains(branchesToSync, base) {
				branchesToSync = append([]string{base}, branchesToSync...)
				current = base
			} else {
				break // Avoid cycles
			}
		}
	}

	// If target branch exists locally, identify local descendants
	targetBranchObj := eng.GetBranch(targetBranch)
	if !opts.Downstack && targetBranchObj.IsTracked() {
		upstack := targetBranchObj.GetRelativeStackUpstack()
		for _, b := range upstack {
			if !contains(branchesToSync, b.GetName()) {
				branchesToSync = append(branchesToSync, b.GetName())
			}
		}
	}

	// Sync each branch
	for _, branchName := range branchesToSync {
		if branchName == eng.Trunk().GetName() {
			continue
		}

		// Fetch if not already fetched
		_, _ = eng.RunGitCommandWithContext(gctx, "fetch", remote, branchName)

		branch := eng.GetBranch(branchName)
		isNew := !branch.IsTracked()

		if isNew {
			splog.Info("Creating local branch %s...", style.ColorBranchName(branchName, false))
			if _, err := eng.RunGitCommandWithContext(gctx, "branch", branchName, fmt.Sprintf("%s/%s", remote, branchName)); err != nil {
				return fmt.Errorf("failed to create local branch %s: %w", branchName, err)
			}
			// Set initial metadata
			if parent, ok := parentMap[branchName]; ok {
				if err := eng.TrackBranch(gctx, branchName, parent); err != nil {
					splog.Debug("Failed to track branch %s with parent %s: %v", branchName, parent, err)
				}
			}
			// New branches are locked by default unless --unfrozen
			if !opts.Unfrozen {
				if err := eng.SetLocked(eng.GetBranch(branchName), true); err != nil {
					splog.Debug("Failed to lock new branch %s: %v", branchName, err)
				}
			}
		} else {
			splog.Info("Updating local branch %s...", style.ColorBranchName(branchName, false))
			if opts.Force {
				if _, err := eng.RunGitCommandWithContext(gctx, "reset", "--hard", fmt.Sprintf("%s/%s", remote, branchName)); err != nil {
					return fmt.Errorf("failed to reset branch %s: %w", branchName, err)
				}
			} else {
				// Try to merge. If conflicts, this will error and we'll stop.
				if _, err := eng.RunGitCommandWithContext(gctx, "merge", fmt.Sprintf("%s/%s", remote, branchName)); err != nil {
					return fmt.Errorf("conflict during sync of %s. Resolve conflicts and try again: %w", branchName, err)
				}
			}
			// Update parent if known
			if parent, ok := parentMap[branchName]; ok {
				if err := eng.SetParent(gctx, branch, eng.GetBranch(parent)); err != nil {
					splog.Debug("Failed to update parent for %s: %v", branchName, err)
				}
			}
		}
	}

	// Fetch and apply remote metadata for all branches in the stack
	if err := git.FetchMetadataRefs(); err != nil {
		splog.Debug("No remote metadata to fetch: %v", err)
	} else {
		// Configure refspec so future git fetch commands also fetch metadata
		if err := git.EnsureMetadataRefspecConfigured(); err != nil {
			splog.Debug("Failed to configure metadata refspec: %v", err)
		}
		if err := eng.LoadRemoteMetadataCache(); err != nil {
			splog.Debug("Failed to load remote metadata cache: %v", err)
		} else {
			for _, branchName := range branchesToSync {
				if branchName == eng.Trunk().GetName() {
					continue
				}
				if err := eng.ApplyRemoteMetadataIfExists(branchName); err != nil {
					splog.Debug("Failed to apply metadata for %s: %v", branchName, err)
				}
			}
		}
	}

	// Checkout target branch
	if err := eng.CheckoutBranch(gctx, eng.GetBranch(targetBranch)); err != nil {
		return fmt.Errorf("failed to checkout target branch %s: %w", targetBranch, err)
	}

	// Refresh engine
	if err := eng.Rebuild(""); err != nil {
		return fmt.Errorf("failed to refresh engine: %w", err)
	}

	// Restack if requested
	if opts.Restack {
		uniqueBranches := []engine.Branch{}
		seen := make(map[string]bool)
		for _, name := range branchesToSync {
			if !seen[name] {
				seen[name] = true
				b := eng.GetBranch(name)
				if b.IsTracked() {
					uniqueBranches = append(uniqueBranches, b)
				}
			}
		}
		sorted := eng.SortBranchesTopologically(uniqueBranches)
		if len(sorted) > 0 {
			if err := RestackBranches(ctx, sorted); err != nil {
				return fmt.Errorf("restack failed: %w", err)
			}
		}
	}

	splog.Info("Successfully synced stack for %s.", style.ColorBranchName(targetBranch, true))
	return nil
}

func contains(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}
