package engine

import (
	"context"

	"stackit.dev/stackit/internal/git"
)

// PlanRestack builds the rebase specs and branch decisions for a restack.
// Branches that are locked or provably up to date are reported in
// PlannedResults so callers can skip validation worktrees for them.
func (e *engineImpl) PlanRestack(ctx context.Context, branches []Branch) (*RestackPlan, error) {
	plan := &RestackPlan{
		Specs:          make([]RebaseSpec, 0, len(branches)),
		BranchMap:      make(map[string]bool),
		PlannedResults: make(map[string]RestackBranchResult),
		Items:          make(map[string]RestackPlanItem),
	}

	for _, branch := range branches {
		item, ok := e.planRestackBranch(ctx, branch, plan.BranchMap)
		if !ok {
			continue
		}
		plan.Items[item.Branch] = item
		if item.Skip {
			plan.PlannedResults[item.Branch] = item.SkipResult
			continue
		}
		plan.Specs = append(plan.Specs, RebaseSpec{
			Branch:      item.Branch,
			NewParent:   item.NewParent,
			OldUpstream: item.OldUpstream,
		})
		plan.BranchMap[item.Branch] = true
	}

	return plan, nil
}

func (e *engineImpl) planRestackBranch(ctx context.Context, branch Branch, plannedBranches map[string]bool) (RestackPlanItem, bool) {
	branchName := branch.GetName()
	item := RestackPlanItem{Branch: branchName}

	lockReason := e.GetLockReason(branch)
	if lockReason.IsLocked() && lockReason != LockReasonDraining {
		item.Skip = true
		item.SkipResult = RestackBranchResult{
			Result:     RestackUnneeded,
			LockReason: lockReason,
		}
		return item, true
	}

	parent := branch.GetParent()
	parentName := e.trunk
	if parent != nil {
		parentName = parent.GetName()
		if _, err := e.GetBranch(parentName).GetRevision(); err != nil {
			if ancestors, ancestorErr := e.FindMostRecentTrackedAncestors(ctx, branchName); ancestorErr == nil && len(ancestors) > 0 {
				parentName = ancestors[0]
			} else {
				parentName = e.trunk
			}
		}
	}
	item.NewParent = parentName

	meta, err := e.git.ReadMetadata(branchName)
	if err != nil {
		return item, false
	}

	oldParentRev := ""
	if rev := meta.GetParentBranchRevision(); rev != nil {
		oldParentRev = *rev
	}

	if oldParentRev != "" {
		isAncestor, err := e.git.IsAncestor(oldParentRev, branchName)
		if err != nil {
			isAncestor = false
		}
		if !isAncestor {
			mergeBase, err := e.git.GetMergeBase(branchName, parentName)
			if err != nil {
				return item, false
			}
			oldParentRev = mergeBase
		}
	} else {
		mergeBase, err := e.git.GetMergeBase(branchName, parentName)
		if err != nil {
			return item, false
		}
		oldParentRev = mergeBase
	}
	item.OldUpstream = oldParentRev

	parentRev, err := e.GetBranch(parentName).GetRevision()
	if err != nil {
		return item, false
	}
	item.ParentRev = parentRev

	item.UseLegacy = branch.IsFrozen() || branch.IsWorktreeAnchor() || e.branchNeedsReparent(ctx, branch, parentName, nil)
	if parentRev == oldParentRev && !plannedBranches[parentName] && !item.UseLegacy {
		item.Skip = true
		item.SkipResult = RestackBranchResult{
			Result:            RestackUnneeded,
			RebasedBranchBase: parentRev,
		}
	}

	return item, true
}

func (e *engineImpl) branchNeedsReparent(ctx context.Context, branch Branch, parentName string, metaMap map[string]*git.Meta) bool {
	if branch.IsTrunk() {
		return false
	}

	e.mu.RLock()
	state := e.state.branchState.GetByName(branch.GetName())
	e.mu.RUnlock()
	if state != nil && e.shouldReparentBranch(ctx, state.Parent, metaMap) {
		return true
	}
	return e.shouldReparentBranch(ctx, parentName, metaMap)
}
