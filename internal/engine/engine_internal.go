package engine

import (
	"context"
	"fmt"
	"slices"

	"stackit.dev/stackit/internal/git"
)

// rebuildInternal is the internal rebuild logic without locking
// refreshCurrentBranch indicates whether to refresh currentBranch from Git
func (e *engineImpl) rebuildInternal(refreshCurrentBranch bool) error {
	// Get all branch names
	branches, err := e.git.GetAllBranchNames()
	if err != nil {
		return fmt.Errorf("failed to get branches: %w", err)
	}

	var currentBranch string
	// Refresh current branch from Git if requested (needed when called from Rebuild/Reset after branch switches)
	if refreshCurrentBranch {
		cb, err := e.git.GetCurrentBranch()
		if err == nil {
			currentBranch = cb
		}
	}

	// Load metadata for each branch in parallel
	allMeta, _ := e.git.BatchReadMetadata(branches)
	allLocalMeta := e.git.BatchReadLocalMetadata(branches)

	e.applyRebuild(branches, currentBranch, allMeta, allLocalMeta)
	return nil
}

// applyRebuild updates the internal state maps from the provided metadata results.
// The caller MUST hold the engine's write lock (e.mu).
func (e *engineImpl) applyRebuild(branches []string, currentBranch string, allMeta map[string]*git.Meta, allLocalMeta map[string]*git.LocalMeta) {
	e.branches = branches
	if currentBranch != "" {
		e.currentBranch = currentBranch
	}

	// Reset maps
	e.parentMap = make(map[string]string)
	e.childrenMap = make(map[string][]string)
	e.scopeMap = make(map[string]string)
	e.lockedMap = make(map[string]string)
	e.frozenMap = make(map[string]bool)

	// Collect results and populate maps sequentially to avoid lock contention/races
	for name, meta := range allMeta {
		if name == e.trunk {
			continue // Trunk branches should never be tracked
		}

		if meta.ParentBranchName != nil {
			parent := *meta.ParentBranchName
			if parent == name {
				continue // Skip self-parenting to avoid cycles
			}
			e.parentMap[name] = parent
			e.childrenMap[parent] = append(e.childrenMap[parent], name)
		}
		if meta.Scope != nil {
			e.scopeMap[name] = *meta.Scope
		}
		if meta.LockReason != LockReasonNone {
			e.lockedMap[name] = string(meta.LockReason)
		}
	}

	for name, meta := range allLocalMeta {
		if meta.Frozen {
			e.frozenMap[name] = true
		}
	}

	// Sort children by name for deterministic traversal
	for _, children := range e.childrenMap {
		slices.Sort(children)
	}
}

// smartSortChildren sorts a list of sibling branches using the "smart" strategy:
// 1. Branches on the path from current branch to trunk are hoisted to the top.
// 2. Other branches are sorted by name descending (newest first).
func (e *engineImpl) smartSortChildren(children []string) {
	if len(children) <= 1 {
		return
	}

	// Calculate active path from current branch to trunk
	activePath := make(map[string]bool)
	if e.currentBranch != "" {
		activePath[e.currentBranch] = true
		curr := e.currentBranch
		visited := make(map[string]bool)
		visited[curr] = true
		for {
			parent, ok := e.parentMap[curr]
			if !ok || parent == "" || parent == e.trunk || visited[parent] {
				break
			}
			activePath[parent] = true
			curr = parent
			visited[curr] = true
		}
	}

	slices.SortFunc(children, func(a, b string) int {
		// First, check if either is on the active path
		aOnPath := activePath[a]
		bOnPath := activePath[b]

		if aOnPath && !bOnPath {
			return -1 // a comes first
		}
		if bOnPath && !aOnPath {
			return 1 // b comes first
		}

		// Otherwise sort by name descending
		if a < b {
			return 1
		}
		if a > b {
			return -1
		}
		return 0
	})
}

// updateBranchInCache updates the cache for a specific branch after restack/metadata changes
func (e *engineImpl) updateBranchInCache(branchName string) {
	if branchName == e.trunk {
		return
	}

	// Read metadata for this branch
	meta, err := e.git.ReadMetadata(branchName)
	if err != nil {
		// If metadata doesn't exist, remove branch from all maps
		if oldParent, exists := e.parentMap[branchName]; exists {
			delete(e.parentMap, branchName)
			// Remove from old parent's children list
			if children, ok := e.childrenMap[oldParent]; ok {
				if i := slices.Index(children, branchName); i >= 0 {
					e.childrenMap[oldParent] = slices.Delete(children, i, i+1)
				}
				// Remove empty children lists
				if len(e.childrenMap[oldParent]) == 0 {
					delete(e.childrenMap, oldParent)
				}
			}
		}
		delete(e.scopeMap, branchName)
		delete(e.lockedMap, branchName)
		delete(e.frozenMap, branchName)
	}

	// Read local metadata too
	localMeta, _ := e.git.ReadLocalMetadata(branchName)

	// Get the old parent before updating
	oldParent := e.parentMap[branchName]

	// Update parent map
	if meta.ParentBranchName != nil {
		e.parentMap[branchName] = *meta.ParentBranchName
	} else {
		delete(e.parentMap, branchName)
	}

	// Update scope map
	if meta.Scope != nil {
		e.scopeMap[branchName] = *meta.Scope
	} else {
		delete(e.scopeMap, branchName)
	}

	// Update locked map
	if meta.LockReason != LockReasonNone {
		e.lockedMap[branchName] = string(meta.LockReason)
	} else {
		delete(e.lockedMap, branchName)
	}

	// Update frozen map
	if localMeta != nil && localMeta.Frozen {
		e.frozenMap[branchName] = true
	} else {
		delete(e.frozenMap, branchName)
	}

	// Update children map - remove from old parent, add to new parent
	newParent := ""
	if meta.ParentBranchName != nil {
		newParent = *meta.ParentBranchName
	}

	// Remove from old parent's children if parent changed
	if oldParent != "" && oldParent != newParent {
		if children, ok := e.childrenMap[oldParent]; ok {
			if i := slices.Index(children, branchName); i >= 0 {
				e.childrenMap[oldParent] = slices.Delete(children, i, i+1)
			}
			// Remove empty children lists
			if len(e.childrenMap[oldParent]) == 0 {
				delete(e.childrenMap, oldParent)
			}
		}
	}

	// Add to new parent's children if it has a parent
	if newParent != "" {
		e.childrenMap[newParent] = append(e.childrenMap[newParent], branchName)
		// Sort for deterministic traversal
		slices.Sort(e.childrenMap[newParent])
	}
}

// rebuild loads all branches and their metadata from Git
func (e *engineImpl) rebuild() error {
	// 1. Get all branch names (slow)
	branches, err := e.git.GetAllBranchNames()
	if err != nil {
		return fmt.Errorf("failed to get branches: %w", err)
	}

	// 2. Get current branch (slow)
	currentBranch, _ := e.git.GetCurrentBranch()

	// 3. Load metadata for each branch in parallel (slow)
	allMeta, _ := e.git.BatchReadMetadata(branches)
	allLocalMeta := e.git.BatchReadLocalMetadata(branches)

	e.mu.Lock()
	defer e.mu.Unlock()

	// 4. Update maps (fast)
	e.applyRebuild(branches, currentBranch, allMeta, allLocalMeta)
	return nil
}

// shouldReparentBranch checks if a parent branch should be reparented
// Returns true if the parent branch:
// - No longer exists locally
// - Has been merged into trunk
// - Has a "MERGED" PR state in metadata
func (e *engineImpl) shouldReparentBranch(ctx context.Context, parentBranchName string, metaMap map[string]*git.Meta) bool {
	// Check if parent is trunk (no need to reparent)
	if parentBranchName == e.trunk {
		return false
	}

	// Check if parent branch still exists locally
	parentExists := false
	for _, name := range e.branches {
		if name == parentBranchName {
			parentExists = true
			break
		}
	}
	if !parentExists {
		return true
	}

	// Check if parent has been merged into trunk
	merged, err := e.git.IsMerged(ctx, parentBranchName, e.trunk)
	if err == nil && merged {
		return true
	}

	// Check if parent has "MERGED" PR state in metadata
	if metaMap != nil {
		if meta, ok := metaMap[parentBranchName]; ok && meta != nil && meta.PrInfo != nil {
			if meta.PrInfo.State != nil && *meta.PrInfo.State == "MERGED" {
				return true
			}
			// If metadata exists but state isn't MERGED, we don't need to check further
			return false
		}
	}

	// Fall back to engine cache/disk if not in metaMap or state unknown
	parentBranch := e.GetBranch(parentBranchName)
	prInfo, err := e.GetPrInfo(parentBranch)
	if err == nil && prInfo != nil && prInfo.State() == "MERGED" {
		return true
	}

	return false
}

// findNearestValidAncestor finds the nearest ancestor that hasn't been merged/deleted
// Returns trunk if all ancestors have been merged
func (e *engineImpl) findNearestValidAncestor(ctx context.Context, branchName string, metaMap map[string]*git.Meta) string {
	current := e.parentMap[branchName]

	for current != "" && current != e.trunk {
		if !e.shouldReparentBranch(ctx, current, metaMap) {
			return current
		}
		// Move to the next parent
		parent, ok := e.parentMap[current]
		if !ok {
			break
		}
		current = parent
	}

	return e.trunk
}

// getRelativeStackUpstackInternal is the internal implementation without lock
// The caller MUST hold at least a read lock (e.mu.RLock())
func (e *engineImpl) getRelativeStackUpstackInternal(branchName string) []Branch {
	result := []Branch{}
	visited := make(map[string]bool)

	var collectDescendants func(string)
	collectDescendants = func(branch string) {
		if visited[branch] {
			return
		}
		visited[branch] = true

		// Don't include the starting branch
		if branch != branchName {
			result = append(result, NewBranch(branch, e))
		}

		// Access childrenMap directly since we already hold the lock
		// This avoids re-acquiring the lock in GetChildren
		if children, ok := e.childrenMap[branch]; ok {
			// Copy child names to avoid modifying the internal map
			childNames := make([]string, len(children))
			copy(childNames, children)
			for _, childName := range childNames {
				collectDescendants(childName)
			}
		}
	}

	collectDescendants(branchName)
	return result
}

// Helper functions
func getStringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func getBoolValue(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}

func stringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}
