package actions

import (
	"context"
	"fmt"
	"sync"

	"stackit.dev/stackit/internal/app"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/output"
	"stackit.dev/stackit/internal/tui/style"
	"stackit.dev/stackit/internal/utils"
)

// CleanBranchesOptions contains options for cleaning branches
type CleanBranchesOptions struct {
	Force bool
}

// CleanBranchesResult contains the result of cleaning branches
type CleanBranchesResult struct {
	DeletedBranches        map[string]string // name -> reason
	BranchesWithNewParents []string
}

// branchDeletionInfo stores information about a branch marked for deletion
type branchDeletionInfo struct {
	reason   string
	blockers map[string]bool
}

// deletionPlan manages the state of branches being deleted
type deletionPlan struct {
	branches map[string]*branchDeletionInfo
}

func newDeletionPlan() *deletionPlan {
	return &deletionPlan{
		branches: make(map[string]*branchDeletionInfo),
	}
}

func (p *deletionPlan) add(name, reason string, blockers map[string]bool) {
	p.branches[name] = &branchDeletionInfo{
		reason:   reason,
		blockers: blockers,
	}
}

func (p *deletionPlan) isDeleting(name string) bool {
	_, ok := p.branches[name]
	return ok
}

func (p *deletionPlan) removeBlocker(branchName, blockerName string) {
	if info, ok := p.branches[branchName]; ok {
		delete(info.blockers, blockerName)
	}
}

// CleanBranches finds and deletes merged/closed branches.
// It follows a multi-phase approach:
// 1. Identify which branches SHOULD be deleted (parallel pre-calculation).
// 2. Build a deletion plan by traversing the stack (DFS).
// 3. Reparent branches that are NOT being deleted but whose parents ARE.
// 4. Execute the deletions in batches (greedy iterative approach).
func CleanBranches(ctx *app.Context, opts CleanBranchesOptions) (*CleanBranchesResult, error) {
	// Phase 1: Identify candidates for deletion
	deleteStatuses := identifyBranchesToDelete(ctx, opts)

	// Phase 2: Build deletion plan
	plan, branchesWithNewParents, err := buildDeletionPlanAndReparent(ctx, deleteStatuses)
	if err != nil {
		return nil, err
	}

	// Capture deleted branches before they are removed from the plan
	deletedBranches := make(map[string]string)
	for name, info := range plan.branches {
		deletedBranches[name] = info.reason
	}

	// Phase 3: Execute deletions
	if err := executeDeletions(ctx, plan); err != nil {
		return nil, err
	}

	return &CleanBranchesResult{
		DeletedBranches:        deletedBranches,
		BranchesWithNewParents: branchesWithNewParents,
	}, nil
}

// identifyBranchesToDelete pre-calculates deletion status for all tracked branches in parallel.
func identifyBranchesToDelete(ctx *app.Context, opts CleanBranchesOptions) map[string]string {
	eng := ctx.Engine
	c := ctx.Context
	out := ctx.Output

	allTrackedBranches := eng.AllBranches()
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

	metadataMap, metaErrs := eng.Git().BatchReadMetadata(branchNames)
	if len(metaErrs) > 0 {
		out.Debug("Failed to read metadata for some branches: %v", metaErrs)
	}

	revisionsMap, revErrs := eng.Git().BatchGetRevisions(allRevisionsToFetch)
	if len(revErrs) > 0 {
		out.Debug("Failed to get revisions for some branches: %v", revErrs)
	}

	mergedBranches, err := eng.Git().GetMergedBranches(c, eng.Trunk().GetName())
	if err != nil {
		out.Debug("Failed to get merged branches: %v", err)
	}

	deleteStatuses := make(map[string]string) // name -> reason
	var mu sync.Mutex

	if len(branchesToProcessPool) > 0 {
		utils.Run(branchesToProcessPool, func(branch engine.Branch) {
			name := branch.GetName()
			shouldDelete, reason := ShouldDeleteBranchCached(c, name, eng, opts.Force, metadataMap[name], revisionsMap, mergedBranches)
			if shouldDelete {
				mu.Lock()
				deleteStatuses[name] = reason
				mu.Unlock()
			}
		})
	}

	return deleteStatuses
}

// buildDeletionPlanAndReparent constructs the deletion hierarchy and updates parents of surviving branches.
func buildDeletionPlanAndReparent(ctx *app.Context, deleteReasons map[string]string) (*deletionPlan, []string, error) {
	eng := ctx.Engine
	out := ctx.Output
	c := ctx.Context

	plan := newDeletionPlan()
	branchesWithNewParents := []string{}
	visited := make(map[string]bool)

	// Start DFS from trunk children to handle the tracked hierarchy
	trunk := eng.Trunk()
	trunkChildren := trunk.GetChildren()
	branchesToProcess := make([]string, len(trunkChildren))
	for i, child := range trunkChildren {
		branchesToProcess[i] = child.GetName()
	}

	for len(branchesToProcess) > 0 {
		branchName := branchesToProcess[len(branchesToProcess)-1]
		branchesToProcess = branchesToProcess[:len(branchesToProcess)-1]

		if visited[branchName] {
			continue
		}
		visited[branchName] = true

		reason, shouldDelete := deleteReasons[branchName]
		branch := eng.GetBranch(branchName)
		children := branch.GetChildren()

		// Add children to DFS stack
		for _, child := range children {
			branchesToProcess = append(branchesToProcess, child.GetName())
		}

		if shouldDelete {
			// Add to plan with its children as initial blockers
			blockers := make(map[string]bool)
			for _, child := range children {
				blockers[child.GetName()] = true
			}
			plan.add(branchName, reason, blockers)
			out.Debug("Marked %s for deletion. Reason: %s. Blockers: %v", branchName, reason, blockers)
		} else {
			// Branch is NOT being deleted. Check if it needs a new parent.
			newParentName, err := reparentBranchIfNecessary(c, branch, plan, eng, out)
			if err != nil {
				return nil, nil, err
			}
			if newParentName != "" {
				branchesWithNewParents = append(branchesWithNewParents, branchName)
			}
		}
	}

	// NEW: Handle "orphan" branches (untracked branches identified for deletion)
	for branchName, reason := range deleteReasons {
		if !visited[branchName] {
			// This branch is disconnected from the trunk hierarchy but should still be deleted
			plan.add(branchName, reason, make(map[string]bool))
			visited[branchName] = true
			out.Debug("Marked orphan branch %s for deletion. Reason: %s", branchName, reason)
		}
	}

	return plan, branchesWithNewParents, nil
}

// executeDeletions greedily deletes unblocked branches from the plan.
func executeDeletions(ctx *app.Context, plan *deletionPlan) error {
	eng := ctx.Engine
	out := ctx.Output
	c := ctx.Context

	for {
		var batchNames []string
		for name, info := range plan.branches {
			if len(info.blockers) == 0 {
				batchNames = append(batchNames, name)
			}
		}

		if len(batchNames) == 0 {
			break
		}

		// Prepare engine branches and track parents
		branches := make([]engine.Branch, len(batchNames))
		parents := make(map[string]string)
		for i, name := range batchNames {
			branch := eng.GetBranch(name)
			branches[i] = branch
			parents[name] = getParentName(branch, eng)
		}

		// Batch delete from engine
		if _, err := eng.DeleteBranches(c, branches); err != nil {
			return fmt.Errorf("failed to delete some branches: %w", err)
		}

		// Batch delete remote metadata
		if err := eng.Git().BatchDeleteRemoteMetadataRefs(batchNames); err != nil {
			out.Debug("Failed to batch delete remote metadata: %v", err)
		}

		// Cleanup plan and update parent blockers
		for _, name := range batchNames {
			out.Info("Deleted branch %s", style.ColorBranchName(name, false))
			delete(plan.branches, name)

			parentName := parents[name]
			plan.removeBlocker(parentName, name)
		}
	}

	return nil
}

// getParentName returns the name of the parent branch or trunk if no parent exists
func getParentName(branch engine.Branch, eng engine.Engine) string {
	parent := branch.GetParent()
	if parent == nil {
		return eng.Trunk().GetName()
	}
	return parent.GetName()
}

// findNonDeletingAncestor finds the nearest ancestor that is not marked for deletion
func findNonDeletingAncestor(startParent string, plan *deletionPlan, eng engine.Engine) string {
	current := startParent
	for {
		if !plan.isDeleting(current) {
			return current
		}
		branch := eng.GetBranch(current)
		parent := branch.GetParent()
		if parent == nil {
			return eng.Trunk().GetName()
		}
		current = parent.GetName()
	}
}

// reparentBranchIfNecessary updates a branch's parent if its current parent is being deleted.
// Returns the name of the new parent if changed, or empty string if not changed.
func reparentBranchIfNecessary(ctx context.Context, branch engine.Branch, plan *deletionPlan, eng engine.Engine, out output.Output) (string, error) {
	branchName := branch.GetName()
	parentName := getParentName(branch, eng)

	// Find nearest ancestor that isn't being deleted
	newParentName := findNonDeletingAncestor(parentName, plan, eng)

	// If parent changed, update it
	if newParentName != parentName {
		if err := eng.SetParent(ctx, branch, eng.GetBranch(newParentName)); err != nil {
			return "", fmt.Errorf("failed to set parent for %s: %w", branchName, err)
		}
		out.Info("Set parent of %s to %s.",
			style.ColorBranchName(branchName, false),
			style.ColorBranchName(newParentName, false))

		// Remove this branch as a blocker for its old parent in the plan
		plan.removeBlocker(parentName, branchName)
		return newParentName, nil
	}

	return "", nil
}
