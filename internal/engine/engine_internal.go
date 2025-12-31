package engine

import (
	"context"
	"fmt"
	"slices"
)

// rebuildData holds the results of gathering engine state from Git/metadata
type rebuildData struct {
	branches      []string
	currentBranch string
	allMeta       map[string]*Meta
	allLocalMeta  map[string]*LocalMeta
}

// gatherRebuildData gathers the data needed to rebuild the engine cache
func (e *engineImpl) gatherRebuildData(refreshCurrentBranch bool) (*rebuildData, error) {
	// Get all branch names
	branches, err := e.git.GetAllBranchNames()
	if err != nil {
		return nil, fmt.Errorf("failed to get branches: %w", err)
	}

	var currentBranch string
	if refreshCurrentBranch {
		cb, err := e.git.GetCurrentBranch()
		if err == nil {
			currentBranch = cb
		}
	}

	// Load metadata for each branch in parallel
	allMeta, _ := e.batchReadMetadataRefs(branches)
	allLocalMeta, _ := e.batchReadLocalMetadataRefs(branches)

	return &rebuildData{
		branches:      branches,
		currentBranch: currentBranch,
		allMeta:       allMeta,
		allLocalMeta:  allLocalMeta,
	}, nil
}

// applyRebuildData updates the engine cache with the gathered data (caller must hold lock if needed)
func (e *engineImpl) applyRebuildData(data *rebuildData, refreshCurrentBranch bool) {
	e.branches = data.branches
	if refreshCurrentBranch {
		e.currentBranch = data.currentBranch
	}

	// Reset maps
	e.parentMap = make(map[string]string)
	e.childrenMap = make(map[string][]string)
	e.scopeMap = make(map[string]string)
	e.lockedMap = make(map[string]bool)
	e.frozenMap = make(map[string]bool)

	// Collect results and populate maps sequentially to avoid lock contention/races
	for name, meta := range data.allMeta {
		if meta.ParentBranchName != nil {
			parent := *meta.ParentBranchName
			e.parentMap[name] = parent
			e.childrenMap[parent] = append(e.childrenMap[parent], name)
		}
		if meta.Scope != nil {
			e.scopeMap[name] = *meta.Scope
		}
		if meta.Locked {
			e.lockedMap[name] = true
		}
	}

	for name, meta := range data.allLocalMeta {
		if meta.Frozen {
			e.frozenMap[name] = true
		}
	}

	// Sort children by name for deterministic traversal
	for _, children := range e.childrenMap {
		slices.Sort(children)
	}
}

// rebuildInternal is the internal rebuild logic without locking
// refreshCurrentBranch indicates whether to refresh currentBranch from Git
func (e *engineImpl) rebuildInternal(refreshCurrentBranch bool) error {
	data, err := e.gatherRebuildData(refreshCurrentBranch)
	if err != nil {
		return err
	}

	e.applyRebuildData(data, refreshCurrentBranch)
	return nil
}

// updateBranchInCache updates the cache for a specific branch after restack/metadata changes
func (e *engineImpl) updateBranchInCache(branchName string) {
	// Read metadata for this branch
	meta, err := e.readMetadataRef(branchName)
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
	localMeta, _ := e.readLocalMetadataRef(branchName)

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
	if meta.Locked {
		e.lockedMap[branchName] = true
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
	// Gather data outside the lock
	data, err := e.gatherRebuildData(false)
	if err != nil {
		return err
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	// Apply data inside the lock
	e.applyRebuildData(data, false)
	return nil
}

// shouldReparentBranch checks if a parent branch should be reparented
// Returns true if the parent branch:
// - No longer exists locally
// - Has been merged into trunk
// - Has a "MERGED" PR state in metadata
func (e *engineImpl) shouldReparentBranch(ctx context.Context, parentBranchName string, metaMap map[string]*Meta) bool {
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
func (e *engineImpl) findNearestValidAncestor(ctx context.Context, branchName string, metaMap map[string]*Meta) string {
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

		children := e.GetChildren(NewBranch(branch, e))
		for _, child := range children {
			collectDescendants(child.GetName())
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

// FindCommonlyNamedTrunk checks for common trunk branch names
// Returns the branch name if exactly one is found, empty string otherwise
func FindCommonlyNamedTrunk(branchNames []string) string {
	commonNames := []string{"main", "master", "development", "develop"}
	var found []string

	for _, name := range branchNames {
		for _, common := range commonNames {
			if name == common {
				found = append(found, name)
				break
			}
		}
	}

	if len(found) == 1 {
		return found[0]
	}
	return ""
}
