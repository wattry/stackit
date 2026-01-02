package actions

import (
	"context"
	"fmt"
	"sync"

	"stackit.dev/stackit/internal/app"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/tui"
	"stackit.dev/stackit/internal/tui/style"
	"stackit.dev/stackit/internal/utils"
)

// CleanBranchesOptions contains options for cleaning branches
type CleanBranchesOptions struct {
	Force bool
}

// CleanBranchesResult contains the result of cleaning branches
type CleanBranchesResult struct {
	BranchesWithNewParents []string
}

// CleanBranches finds and deletes merged/closed branches
// Returns branches whose parents have changed (need restacking)
func CleanBranches(ctx *app.Context, opts CleanBranchesOptions) (*CleanBranchesResult, error) {
	eng := ctx.Engine
	splog := ctx.Splog
	c := ctx.Context

	// Pre-calculate which branches should be deleted in parallel using a worker pool
	allTrackedBranches := eng.AllBranches()
	type deleteStatus struct {
		shouldDelete bool
		reason       string
	}
	deleteStatuses := make(map[string]deleteStatus)
	var mu sync.Mutex

	// Filter out trunk branches before processing
	branchesToProcessPool := []engine.Branch{}
	branchNames := []string{}
	allRevisionsToFetch := []string{eng.Trunk().GetName()}
	for _, branch := range allTrackedBranches {
		name := branch.GetName()
		allRevisionsToFetch = append(allRevisionsToFetch, name)
		if !branch.IsTrunk() {
			branchesToProcessPool = append(branchesToProcessPool, branch)
			branchNames = append(branchNames, name)

			parent := branch.GetParent()
			if parent != nil {
				allRevisionsToFetch = append(allRevisionsToFetch, parent.GetName())
			}
		}
	}

	// Pre-fetch metadata and revisions in batch to avoid O(N) git calls in worker pool
	metadataMap, _ := eng.Git().BatchReadMetadata(branchNames)
	revisionsMap, _ := eng.Git().BatchGetRevisions(allRevisionsToFetch)

	if len(branchesToProcessPool) > 0 {
		utils.Run(branchesToProcessPool, func(branch engine.Branch) {
			name := branch.GetName()
			shouldDelete, reason := ShouldDeleteBranchCached(c, name, eng, opts.Force, metadataMap[name], revisionsMap)
			mu.Lock()
			deleteStatuses[name] = deleteStatus{shouldDelete: shouldDelete, reason: reason}
			mu.Unlock()
		})
	}

	// Start from trunk children
	trunk := eng.Trunk()
	trunkChildren := trunk.GetChildren()
	branchesToProcess := make([]string, len(trunkChildren))
	for i, c := range trunkChildren {
		branchesToProcess[i] = c.GetName()
	}
	branchesToDelete := make(map[string]map[string]bool) // branch -> set of blocking children
	branchesWithNewParents := []string{}

	// DFS traversal
	for len(branchesToProcess) > 0 {
		// Pop from stack
		branchName := branchesToProcess[len(branchesToProcess)-1]
		branchesToProcess = branchesToProcess[:len(branchesToProcess)-1]

		// Skip if already marked for deletion
		if _, ok := branchesToDelete[branchName]; ok {
			continue
		}

		// Use pre-calculated status
		status := deleteStatuses[branchName]
		if status.shouldDelete {
			branch := eng.GetBranch(branchName)
			children := branch.GetChildren()
			// Add children to process (DFS)
			for _, child := range children {
				branchesToProcess = append(branchesToProcess, child.GetName())
			}

			// Mark for deletion with blockers
			blockers := make(map[string]bool)
			for _, child := range children {
				blockers[child.GetName()] = true
			}
			branchesToDelete[branchName] = blockers

			splog.Debug("Marked %s for deletion. Reason: %s. Blockers: %v", branchName, status.reason, children)
		} else {
			// Branch is not being deleted
			// If its parent IS being deleted, update parent
			branch := eng.GetBranch(branchName)
			parent := branch.GetParent()
			parentName := ""
			if parent == nil {
				parentName = eng.Trunk().GetName()
			} else {
				parentName = parent.GetName()
			}

			// Find nearest ancestor that isn't being deleted
			newParentName := parentName
			for {
				if _, isDeleting := branchesToDelete[newParentName]; !isDeleting {
					break
				}
				newParentBranch := eng.GetBranch(newParentName)
				ancestor := newParentBranch.GetParent()
				if ancestor == nil {
					newParentName = eng.Trunk().GetName()
					break
				}
				newParentName = ancestor.GetName()
			}

			// If parent changed, update it
			if newParentName != parentName {
				if err := eng.SetParent(c, branch, eng.GetBranch(newParentName)); err != nil {
					return nil, fmt.Errorf("failed to set parent for %s: %w", branchName, err)
				}
				splog.Info("Set parent of %s to %s.",
					style.ColorBranchName(branchName, false),
					style.ColorBranchName(newParentName, false))
				branchesWithNewParents = append(branchesWithNewParents, branchName)

				// Remove this branch as a blocker for its old parent
				if blockers, ok := branchesToDelete[parentName]; ok {
					delete(blockers, branchName)
					branchesToDelete[parentName] = blockers
				}
			}
		}

		// Greedily delete unblocked branches
		greedilyDeleteUnblockedBranches(c, branchesToDelete, eng, splog)
	}

	return &CleanBranchesResult{
		BranchesWithNewParents: branchesWithNewParents,
	}, nil
}

// greedilyDeleteUnblockedBranches deletes branches that have no blockers
func greedilyDeleteUnblockedBranches(ctx context.Context, branchesToDelete map[string]map[string]bool, eng engine.Engine, splog *tui.Splog) {
	for {
		// Find all currently unblocked branches
		var batchNames []string
		for branchName, blockers := range branchesToDelete {
			if len(blockers) == 0 {
				batchNames = append(batchNames, branchName)
			}
		}

		if len(batchNames) == 0 {
			break
		}

		// Prepare branches for deletion and track parents for blocker updates
		branches := make([]engine.Branch, len(batchNames))
		parents := make(map[string]string)
		for i, name := range batchNames {
			branch := eng.GetBranch(name)
			branches[i] = branch

			parent := branch.GetParent()
			if parent == nil {
				parents[name] = eng.Trunk().GetName()
			} else {
				parents[name] = parent.GetName()
			}
		}

		// Batch delete branches from engine
		if _, err := eng.DeleteBranches(ctx, branches); err != nil {
			splog.Debug("Failed to batch delete branches: %v", err)
			// Even if some failed, we continue to remove them from our internal tracking
			// to avoid infinite loops and follow the best-effort approach.
		}

		// Post-deletion updates
		if err := eng.Git().BatchDeleteRemoteMetadataRefs(batchNames); err != nil {
			splog.Debug("Failed to batch delete remote metadata: %v", err)
		}

		for _, branchName := range batchNames {
			splog.Info("Deleted branch %s", style.ColorBranchName(branchName, false))

			// Remove from deletion map
			delete(branchesToDelete, branchName)

			// Remove this branch as a blocker for its parent
			parentName := parents[branchName]
			if parentBlockers, ok := branchesToDelete[parentName]; ok {
				delete(parentBlockers, branchName)
			}
		}
	}
}
