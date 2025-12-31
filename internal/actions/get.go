package actions

import (
	"fmt"
	"strconv"

	getAction "stackit.dev/stackit/internal/actions/get"
	"stackit.dev/stackit/internal/app"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/handlers"
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
func GetAction(ctx *app.Context, branchOrPR string, opts GetOptions, handler getAction.Handler) error {
	eng := ctx.Engine
	splog := ctx.Splog
	gctx := ctx.Context

	if utils.HasUncommittedChanges(gctx) {
		return fmt.Errorf("you have uncommitted changes. Please commit or stash them before running 'get'")
	}

	targetBranch := ""
	var targetPRNumber *int
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
			targetPRNumber = &prNum
		} else {
			targetBranch = branchOrPR
		}
	}

	// Start the handler
	handler.Start(targetBranch, targetPRNumber)

	// Emit fetch phase started
	handler.EmitEvent(getAction.Event{
		Phase: getAction.PhaseFetch,
		Type:  getAction.EventStarted,
	})

	// Fetch target branch from origin
	remote := eng.GetRemote()
	if _, err := eng.RunGitCommandWithContext(gctx, "fetch", remote, targetBranch); err != nil {
		return fmt.Errorf("failed to fetch branch %s from %s: %w", targetBranch, remote, err)
	}

	// Emit trunk status (main/master)
	trunkName := eng.Trunk().GetName()
	handler.EmitEvent(getAction.Event{
		Phase:  getAction.PhaseFetch,
		Type:   getAction.EventCompleted,
		Branch: trunkName,
	})

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

	// Emit sync phase started
	handler.EmitEvent(getAction.Event{
		Phase: getAction.PhaseSync,
		Type:  getAction.EventStarted,
	})

	// Track statistics for summary
	var branchesCreated, branchesUpdated int
	branchPRInfo := make(map[string]*int)       // branch -> PR number
	branchFrozenStatus := make(map[string]bool) // branch -> is frozen

	// Fetch PR info for branches if possible
	if ctx.GitHubClient != nil {
		owner, repo := ctx.GitHubClient.GetOwnerRepo()
		for _, branchName := range branchesToSync {
			if branchName == eng.Trunk().GetName() {
				continue
			}
			if pr, err := ctx.GitHubClient.GetPullRequestByBranch(gctx, owner, repo, branchName); err == nil && pr != nil {
				prNum := pr.Number
				branchPRInfo[branchName] = &prNum
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
			if _, err := eng.RunGitCommandWithContext(gctx, "branch", branchName, fmt.Sprintf("%s/%s", remote, branchName)); err != nil {
				return fmt.Errorf("failed to create local branch %s: %w", branchName, err)
			}
			// Set initial metadata
			if parent, ok := parentMap[branchName]; ok {
				if err := eng.TrackBranch(gctx, branchName, parent); err != nil {
					splog.Debug("Failed to track branch %s with parent %s: %v", branchName, parent, err)
				}
			}
			// New branches are frozen by default unless --unfrozen
			isFrozen := !opts.Unfrozen
			branchFrozenStatus[branchName] = isFrozen
			if isFrozen {
				if err := eng.SetFrozen(eng.GetBranch(branchName), true); err != nil {
					splog.Debug("Failed to freeze new branch %s: %v", branchName, err)
				}
			}
			branchesCreated++

			// Emit sync event
			handler.EmitEvent(getAction.Event{
				Phase:    getAction.PhaseSync,
				Type:     getAction.EventCompleted,
				Branch:   branchName,
				PRNumber: branchPRInfo[branchName],
				IsNew:    true,
			})
		} else {
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
			branchesUpdated++

			// Emit sync event
			handler.EmitEvent(getAction.Event{
				Phase:    getAction.PhaseSync,
				Type:     getAction.EventCompleted,
				Branch:   branchName,
				PRNumber: branchPRInfo[branchName],
				IsNew:    false,
			})
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
	var restacked, skipped int
	var conflicts []string
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
			// Use RestackHandler for consistent output
			handler.OnRestackStart(len(sorted))

			if err := RestackBranchesWithHandler(ctx, sorted, func(branchName string, result engine.RestackResult, newRev string, _ bool) {
				prNumber := getPRNumber(eng, branchName)

				switch result {
				case engine.RestackDone:
					restacked++
					handler.OnRestackBranch(branchName, handlers.RestackDone, newRev, prNumber)
				case engine.RestackUnneeded:
					handler.OnRestackBranch(branchName, handlers.RestackUnneeded, "", prNumber)
				case engine.RestackConflict:
					skipped++
					conflicts = append(conflicts, branchName)
					handler.OnRestackBranch(branchName, handlers.RestackConflict, "", prNumber)
				}
			}); err != nil {
				handler.OnRestackComplete(restacked, skipped, conflicts)
				return fmt.Errorf("restack failed: %w", err)
			}

			handler.OnRestackComplete(restacked, skipped, conflicts)
		}
	}

	// Complete with summary
	targetBranchObj = eng.GetBranch(targetBranch)
	isFrozenFinal := targetBranchObj.IsFrozen()
	upToDate := branchesCreated == 0 && branchesUpdated == 0 && restacked == 0
	handler.Complete(getAction.Summary{
		TargetBranch:    targetBranch,
		BranchesCreated: branchesCreated,
		BranchesUpdated: branchesUpdated,
		Restacked:       restacked,
		IsFrozen:        isFrozenFinal,
		UpToDate:        upToDate,
	})

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

// getPRNumber returns the PR number for a branch, or nil if not available
func getPRNumber(eng engine.Engine, branchName string) *int {
	branch := eng.GetBranch(branchName)
	prInfo, err := branch.GetPrInfo()
	if err != nil || prInfo == nil {
		return nil
	}
	return prInfo.Number()
}
