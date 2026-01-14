package engine

import (
	"context"
	"fmt"
	"iter"
)

// AllBranches returns all branches
func (e *engineImpl) AllBranches() []Branch {
	e.mu.RLock()
	defer e.mu.RUnlock()
	branches := make([]Branch, len(e.branches))
	for i, name := range e.branches {
		branches[i] = NewBranch(name, e)
	}
	return branches
}

// CurrentBranch returns the current branch (nil if not on a branch)
func (e *engineImpl) CurrentBranch() *Branch {
	current, err := e.git.GetCurrentBranch()
	if err != nil {
		// Not on a branch (e.g., detached HEAD)
		current = ""
	}

	e.mu.Lock()
	e.currentBranch = current
	e.mu.Unlock()

	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.currentBranch == "" {
		return nil
	}
	branch := NewBranch(e.currentBranch, e)
	return &branch
}

// ValidateOnBranch ensures the user is on a branch
func (e *engineImpl) ValidateOnBranch() (string, error) {
	currentBranch := e.CurrentBranch()
	if currentBranch == nil {
		return "", fmt.Errorf("not on a branch")
	}
	return currentBranch.GetName(), nil
}

// Trunk returns the trunk branch
func (e *engineImpl) Trunk() Branch {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return NewBranch(e.trunk, e)
}

// GetBranch returns a Branch wrapper for the given branch name
func (e *engineImpl) GetBranch(branchName string) Branch {
	return NewBranch(branchName, e)
}

// GetParent returns the parent branch (nil if no parent)
func (e *engineImpl) GetParent(branch Branch) *Branch {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if state := e.branchState.Get(branch); state != nil {
		b := NewBranch(state.Parent, e)
		return &b
	}
	return nil
}

// getParent is an internal method for Branch type
func (e *engineImpl) getParent(branch Branch) *Branch {
	return e.GetParent(branch) // Delegate to existing implementation for now
}

// FindMostRecentTrackedAncestors finds the most recent tracked ancestors of a branch
// by checking the branch's commit history against tracked branch tips.
// Returns a slice of branch names that point to the most recent tracked commit in history.
func (e *engineImpl) FindMostRecentTrackedAncestors(ctx context.Context, branchName string) ([]string, error) {
	if e.IsTrunk(e.GetBranch(branchName)) {
		return nil, nil
	}

	e.mu.RLock()
	defer e.mu.RUnlock()

	trunk := e.trunk

	// Map of commit SHA to slice of tracked branch names
	trackedBranchTips := make(map[string][]string)

	// Add trunk tip
	trunkRev, err := e.git.GetRevision(trunk)
	if err == nil {
		trackedBranchTips[trunkRev] = append(trackedBranchTips[trunkRev], trunk)
	}

	// Get all tracked branches and their tips
	for _, candidate := range e.branches {
		// Skip the branch itself and trunk (already handled)
		if candidate == branchName || candidate == trunk {
			continue
		}

		// Only consider tracked branches
		if !e.branchState.HasByName(candidate) {
			continue
		}

		// Skip branches merged into trunk
		if merged, err := e.git.IsMerged(ctx, candidate, trunk); err == nil && merged {
			continue
		}

		// Get candidate revision
		candidateRev, err := e.git.GetRevision(candidate)
		if err != nil {
			continue
		}

		trackedBranchTips[candidateRev] = append(trackedBranchTips[candidateRev], candidate)
	}

	// Get history of the branch we're tracking
	history, err := e.git.GetCommitHistorySHAs(branchName)
	if err != nil {
		return nil, err
	}

	// Iterate through history (newest to oldest) and find the first tracked tip(s)
	for i := 0; i < len(history); i++ {
		sha := history[i]
		if ancestors, ok := trackedBranchTips[sha]; ok {
			// Found the most recent tracked commit(s)
			return ancestors, nil
		}
	}

	return nil, nil
}

// FindBranchForCommit finds which branch a commit belongs to
func (e *engineImpl) FindBranchForCommit(commitSHA string) (string, error) {
	e.mu.RLock()
	branches := make([]string, len(e.branches))
	copy(branches, e.branches)
	e.mu.RUnlock()

	for _, branchName := range branches {
		commits, err := e.GetAllCommits(NewBranch(branchName, e), CommitFormatSHA)
		if err != nil {
			continue
		}

		for _, sha := range commits {
			if sha == commitSHA {
				return branchName, nil
			}
		}
	}

	return "", nil
}

// SortBranchesTopologically sorts branches so parents come before children.
// This ensures correct restack order (bottom of stack first).
func (e *engineImpl) SortBranchesTopologically(branches []Branch) []Branch {
	if len(branches) == 0 {
		return branches
	}

	// Calculate depth for each branch (distance from trunk)
	depths := make(map[string]int)
	visited := make(map[string]bool)
	var getDepth func(branchName string) int
	getDepth = func(branchName string) int {
		if depth, ok := depths[branchName]; ok {
			return depth
		}

		if visited[branchName] {
			return 1000 // Arbitrary high depth for cycles
		}
		visited[branchName] = true
		defer func() { visited[branchName] = false }()

		e.mu.RLock()
		isTrunk := branchName == e.trunk
		state := e.branchState.GetByName(branchName)
		e.mu.RUnlock()

		if isTrunk {
			depths[branchName] = 0
			return 0
		}
		if state == nil || state.Parent == "" || e.IsTrunk(NewBranch(state.Parent, e)) {
			depths[branchName] = 1
			return 1
		}
		depths[branchName] = getDepth(state.Parent) + 1
		return depths[branchName]
	}

	// Calculate depth for all branches
	for _, branch := range branches {
		getDepth(branch.GetName())
	}

	// Sort by depth (parents first, then children)
	result := make([]Branch, len(branches))
	copy(result, branches)
	for i := 0; i < len(result)-1; i++ {
		for j := i + 1; j < len(result); j++ {
			if depths[result[i].GetName()] > depths[result[j].GetName()] {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	return result
}

// BranchesDepthFirst returns an iterator that yields branches starting from startBranch in depth-first order.
// Each iteration yields (branchName, depth) where depth is 0 for the start branch.
// The iterator can be used with range loops and supports early termination with break.
func (e *engineImpl) BranchesDepthFirst(startBranch Branch) iter.Seq2[Branch, int] {
	return func(yield func(Branch, int) bool) {
		visited := make(map[string]bool)
		var visit func(branch string, depth int) bool
		visit = func(branch string, depth int) bool {
			if visited[branch] {
				return true // cycle detection
			}
			visited[branch] = true

			if !yield(NewBranch(branch, e), depth) {
				return false // iterator wants to stop
			}

			// Get children directly from internal map
			e.mu.RLock()
			children := e.childrenMap[branch]
			e.mu.RUnlock()

			for _, childName := range children {
				if !visit(childName, depth+1) {
					return false
				}
			}
			return true
		}

		visit(startBranch.GetName(), 0)
	}
}
