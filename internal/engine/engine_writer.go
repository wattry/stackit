package engine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"stackit.dev/stackit/internal/git"
)

// timeNow is a variable for time.Now to allow mocking in tests
var timeNow = time.Now

// CreateBranch creates a new branch at the given start point
func (e *engineImpl) CreateBranch(ctx context.Context, branchName string, startPoint string) error {
	return e.git.CreateBranch(ctx, branchName, startPoint)
}

// ResetHard performs a hard reset to the given revision
func (e *engineImpl) ResetHard(ctx context.Context, revision string) error {
	return e.git.HardReset(ctx, revision)
}

// ResetMerge performs a merge reset to the given revision
func (e *engineImpl) ResetMerge(ctx context.Context, revision string) error {
	return e.git.ResetMerge(ctx, revision)
}

// Merge merges a revision into the current branch
func (e *engineImpl) Merge(ctx context.Context, revision string, opts MergeOptions) error {
	return e.git.Merge(ctx, revision, git.MergeOptions{
		FFOnly:  opts.FFOnly,
		NoEdit:  opts.NoEdit,
		NoFF:    opts.NoFF,
		Message: opts.Message,
	})
}

// MergeMultiple performs an octopus merge of multiple branches into the current branch
func (e *engineImpl) MergeMultiple(ctx context.Context, branches []string, opts MergeOptions) error {
	return e.git.MergeMultiple(ctx, branches, git.MergeOptions{
		NoEdit:  opts.NoEdit,
		NoFF:    opts.NoFF,
		Message: opts.Message,
	})
}

// Fetch fetches from a remote
func (e *engineImpl) Fetch(ctx context.Context, remote string, branch string) error {
	return e.git.Fetch(ctx, remote, branch)
}

// InteractiveRebase starts an interactive rebase
func (e *engineImpl) InteractiveRebase(ctx context.Context, onto string) error {
	return e.git.InteractiveRebase(ctx, onto)
}

// PushBranch pushes a branch to the remote
func (e *engineImpl) PushBranch(ctx context.Context, branch Branch, remote string, opts git.PushOptions) error {
	return e.git.PushBranch(ctx, branch.GetName(), remote, opts)
}

// TrackBranch tracks a branch with a parent branch
func (e *engineImpl) TrackBranch(ctx context.Context, branchName string, parentBranchName string) error {
	if branchName == e.trunk {
		return fmt.Errorf("cannot track trunk branch %s", e.trunk)
	}
	if branchName == parentBranchName {
		return fmt.Errorf("branch cannot be its own parent")
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	// Update current branch if it changed
	if current, err := e.git.GetCurrentBranch(); err == nil {
		e.currentBranch = current
	}

	// Validate branch exists
	branchExists := false
	for _, name := range e.branches {
		if name == branchName {
			branchExists = true
			break
		}
	}
	if !branchExists {
		// Refresh branches list
		branches, err := e.git.GetAllBranchNames()
		if err != nil {
			return fmt.Errorf("failed to get branches: %w", err)
		}
		e.branches = branches
		branchExists = false
		for _, name := range e.branches {
			if name == branchName {
				branchExists = true
				break
			}
		}
		if !branchExists {
			return fmt.Errorf("branch %s does not exist", branchName)
		}
	}

	// Validate parent exists (or is trunk)
	if parentBranchName != e.trunk {
		parentExists := false
		for _, name := range e.branches {
			if name == parentBranchName {
				parentExists = true
				break
			}
		}
		if !parentExists {
			// Refresh branches list to check again
			branches, err := e.git.GetAllBranchNames()
			if err != nil {
				return fmt.Errorf("failed to get branches: %w", err)
			}
			e.branches = branches
			parentExists = false
			for _, name := range e.branches {
				if name == parentBranchName {
					parentExists = true
					break
				}
			}
			if !parentExists {
				return fmt.Errorf("parent branch %s does not exist", parentBranchName)
			}
		}
	}

	return e.setParentInternal(ctx, branchName, parentBranchName)
}

// UntrackBranch stops tracking a branch by deleting its metadata
func (e *engineImpl) UntrackBranch(branchName string) error {
	// Delete metadata
	if err := e.git.DeleteMetadata(branchName); err != nil {
		return fmt.Errorf("failed to delete metadata ref: %w", err)
	}

	// Rebuild cache
	return e.rebuild()
}

// DeleteBranch deletes a branch and its metadata
func (e *engineImpl) DeleteBranch(ctx context.Context, branch Branch) error {
	branchName := branch.GetName()
	if e.IsTrunk(branch) {
		return fmt.Errorf("cannot delete trunk branch")
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	// Get children before deletion
	children := make([]string, len(e.childrenMap[branchName]))
	copy(children, e.childrenMap[branchName])

	// Get parent
	parent := e.trunk
	if state := e.branchState.GetByName(branchName); state != nil {
		parent = state.Parent
	}

	// If deleting current branch, switch to trunk first
	if branchName == e.currentBranch {
		// Access trunk directly while holding the lock (avoid deadlock from e.Trunk() trying to acquire RLock)
		trunkBranch := NewBranch(e.trunk, e)
		if err := e.git.CheckoutBranch(ctx, trunkBranch.GetName()); err != nil {
			return fmt.Errorf("failed to switch to trunk before deleting current branch: %w", err)
		}
		e.currentBranch = e.trunk
	}

	// Delete git branch
	if err := e.git.DeleteBranch(ctx, branch.GetName()); err != nil {
		if !git.IsBranchNotFoundError(err) {
			return fmt.Errorf("failed to delete branch: %w", err)
		}
	}

	// Delete metadata
	if err := e.git.DeleteMetadata(branchName); err != nil {
		_, _ = fmt.Fprintf(e.writer, "Warning: failed to delete metadata ref for %s: %v\n", branchName, err)
	}

	// Delete local metadata
	if err := e.git.DeleteRef(fmt.Sprintf("%s%s", git.LocalMetadataRefPrefix, branchName)); err != nil {
		_, _ = fmt.Fprintf(e.writer, "Warning: failed to delete local metadata ref for %s: %v\n", branchName, err)
	}

	// Update children to point to parent
	for _, child := range children {
		if err := e.setParentInternal(ctx, child, parent); err != nil {
			continue
		}
	}

	// Remove from parent's children list
	if parent != "" {
		parentChildren := e.childrenMap[parent]
		if i := slices.Index(parentChildren, branchName); i >= 0 {
			e.childrenMap[parent] = slices.Delete(parentChildren, i, i+1)
		}
	}

	// Remove from maps
	e.branchState.Delete(branchName)
	delete(e.childrenMap, branchName)

	// Remove from branches list
	if i := slices.Index(e.branches, branchName); i >= 0 {
		e.branches = slices.Delete(e.branches, i, i+1)
	}

	return nil
}

// DeleteBranches deletes multiple branches and returns the children that need restacking
func (e *engineImpl) DeleteBranches(ctx context.Context, branches []Branch) ([]string, error) {
	// Identify all children of all branches to be deleted
	allChildren := make(map[string]bool)
	toDeleteSet := make(map[string]bool)
	for _, b := range branches {
		branchName := b.GetName()
		toDeleteSet[branchName] = true
		e.mu.RLock()
		children := e.childrenMap[branchName]
		e.mu.RUnlock()
		for _, child := range children {
			allChildren[child] = true
		}
	}

	// Remove branches that are also being deleted from the children set
	for _, b := range branches {
		delete(allChildren, b.GetName())
	}

	// Delete branches
	for _, b := range branches {
		if err := e.DeleteBranch(ctx, b); err != nil {
			return nil, fmt.Errorf("failed to delete branch %s: %w", b.GetName(), err)
		}
	}

	// Convert children map to slice
	childrenToRestack := make([]string, 0, len(allChildren))
	for child := range allChildren {
		childrenToRestack = append(childrenToRestack, child)
	}

	return childrenToRestack, nil
}

// CheckoutBranch checks out an existing branch
func (e *engineImpl) CheckoutBranch(ctx context.Context, branch Branch) error {
	branchName := branch.GetName()
	if err := e.git.CheckoutBranch(ctx, branchName); err != nil {
		// If it's already used by another worktree, try checking out detached
		if strings.Contains(err.Error(), "already used by worktree") {
			if err := e.git.CheckoutDetached(ctx, branchName); err != nil {
				return err
			}
			e.mu.Lock()
			e.currentBranch = "" // Detached HEAD
			e.mu.Unlock()
			return nil
		}
		return err
	}

	e.mu.Lock()
	e.currentBranch = branchName
	e.mu.Unlock()
	return nil
}

// UpdateBranchRef updates a branch reference to point to a new revision
func (e *engineImpl) UpdateBranchRef(ctx context.Context, branchName, revision string) error {
	return e.git.UpdateBranchRef(ctx, branchName, revision)
}

// CreateAndCheckoutBranch creates and checks out a new branch
func (e *engineImpl) CreateAndCheckoutBranch(ctx context.Context, branch Branch) error {
	branchName := branch.GetName()
	if err := e.git.CreateAndCheckoutBranch(ctx, branchName); err != nil {
		return err
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	e.currentBranch = branchName
	// Add to branches list if not already there
	found := false
	for _, b := range e.branches {
		if b == branchName {
			found = true
			break
		}
	}
	if !found {
		e.branches = append(e.branches, branchName)
	}

	return nil
}

// SetParent updates a branch's parent
func (e *engineImpl) SetParent(ctx context.Context, branch Branch, parentBranch Branch) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.setParentInternal(ctx, branch.GetName(), parentBranch.GetName())
}

// SetParentPreservingDivergence updates a branch's parent while preserving
// the divergence point if it remains a valid ancestor. This is useful when
// moving a branch to a new parent without changing which commits belong to it.
func (e *engineImpl) SetParentPreservingDivergence(ctx context.Context, branch Branch, newParent Branch, oldDivergencePoint string) error {
	if err := e.SetParent(ctx, branch, newParent); err != nil {
		return err
	}

	// If we have an old divergence point and it's still a valid ancestor,
	// restore it to preserve which commits belong to this branch
	if oldDivergencePoint != "" {
		isAncestor, err := e.git.IsAncestor(oldDivergencePoint, branch.GetName())
		if err == nil && isAncestor {
			return e.UpdateParentRevision(branch.GetName(), oldDivergencePoint)
		}
	}

	return nil
}

// UpdateParentRevision updates the parent revision in metadata
func (e *engineImpl) UpdateParentRevision(branchName string, parentRev string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	meta, err := e.git.ReadMetadata(branchName)
	if err != nil {
		return fmt.Errorf("failed to read metadata: %w", err)
	}

	meta.ParentBranchRevision = &parentRev

	if err := e.git.WriteMetadata(branchName, meta); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	return nil
}

// SetScope updates a branch's scope
func (e *engineImpl) SetScope(branch Branch, scope Scope) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	branchName := branch.GetName()

	// Read existing metadata
	meta, err := e.git.ReadMetadata(branchName)
	if err != nil {
		return fmt.Errorf("failed to read metadata: %w", err)
	}

	// Update scope
	if scope.IsEmpty() {
		meta.Scope = nil
	} else {
		scopeStr := scope.String()
		meta.Scope = &scopeStr
	}

	// Write metadata
	if err := e.git.WriteMetadata(branchName, meta); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	// Update in-memory state
	if state := e.branchState.GetByName(branchName); state != nil {
		if scope.IsEmpty() {
			state.Scope = ""
		} else {
			state.Scope = scope.String()
		}
	}

	return nil
}

// SetLocked updates multiple branches' locked status
func (e *engineImpl) SetLocked(branches []Branch, reason LockReason) (BatchLockResult, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	result := BatchLockResult{
		AffectedBranches: make([]string, 0, len(branches)),
		Errors:           make(map[string]error),
	}

	for _, branch := range branches {
		branchName := branch.GetName()

		// Read existing metadata
		meta, err := e.git.ReadMetadata(branchName)
		if err != nil {
			result.Errors[branchName] = fmt.Errorf("failed to read metadata: %w", err)
			continue
		}

		// Update locked status
		meta.LockReason = reason

		// Update in-memory state
		if state := e.branchState.GetByName(branchName); state != nil {
			state.LockReason = reason
		}

		// Write metadata
		if err := e.git.WriteMetadata(branchName, meta); err != nil {
			result.Errors[branchName] = fmt.Errorf("failed to write metadata: %w", err)
			continue
		}

		result.AffectedBranches = append(result.AffectedBranches, branchName)
	}

	if len(result.Errors) > 0 {
		return result, fmt.Errorf("failed to update locked status for some branches")
	}

	return result, nil
}

// SetFrozen updates multiple branches' frozen status
func (e *engineImpl) SetFrozen(branches []Branch, frozen bool) (BatchFreezeResult, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	result := BatchFreezeResult{
		AffectedBranches: make([]string, 0, len(branches)),
		Errors:           make(map[string]error),
	}

	for _, branch := range branches {
		branchName := branch.GetName()

		// Read existing local metadata
		meta, err := e.git.ReadLocalMetadata(branchName)
		if err != nil {
			result.Errors[branchName] = fmt.Errorf("failed to read local metadata: %w", err)
			continue
		}

		// Update frozen status
		meta.Frozen = frozen

		// Write local metadata
		if err := e.git.WriteLocalMetadata(branchName, meta); err != nil {
			result.Errors[branchName] = fmt.Errorf("failed to write local metadata: %w", err)
			continue
		}

		// Update in-memory state (create entry if needed for untracked branches)
		state := e.branchState.GetOrCreate(branchName)
		state.Frozen = frozen

		result.AffectedBranches = append(result.AffectedBranches, branchName)
	}

	if len(result.Errors) > 0 {
		return result, fmt.Errorf("failed to update frozen status for some branches")
	}

	return result, nil
}

// RenameBranch renames a branch and its metadata
func (e *engineImpl) RenameBranch(ctx context.Context, oldBranch, newBranch Branch) error {
	oldName := oldBranch.GetName()
	newName := newBranch.GetName()

	e.mu.RLock()
	// Get children before renaming anything
	children := make([]string, len(e.childrenMap[oldName]))
	copy(children, e.childrenMap[oldName])
	e.mu.RUnlock()

	// Rename git branch
	if err := e.git.RenameBranch(ctx, oldName, newName); err != nil {
		return err
	}

	// Rename metadata ref
	if err := e.git.RenameMetadata(oldName, newName); err != nil {
		// Log but continue if metadata rename fails
		_, _ = fmt.Fprintf(e.writer, "Warning: failed to rename metadata ref: %v\n", err)
	}

	// Rename local metadata ref
	oldLocalRef := fmt.Sprintf("%s%s", git.LocalMetadataRefPrefix, oldName)
	newLocalRef := fmt.Sprintf("%s%s", git.LocalMetadataRefPrefix, newName)
	if sha, err := e.git.GetRef(oldLocalRef); err == nil {
		if err := e.git.UpdateRef(newLocalRef, sha); err == nil {
			if err := e.git.DeleteRef(oldLocalRef); err != nil {
				_, _ = fmt.Fprintf(e.writer, "Warning: failed to delete old local metadata ref: %v\n", err)
			}
		} else {
			_, _ = fmt.Fprintf(e.writer, "Warning: failed to update new local metadata ref: %v\n", err)
		}
	}

	// Update children to point to the new branch name
	for _, child := range children {
		childMeta, err := e.git.ReadMetadata(child)
		if err != nil {
			continue
		}
		childMeta.ParentBranchName = &newName
		if err := e.git.WriteMetadata(child, childMeta); err != nil {
			continue
		}
	}

	// Rebuild in-memory state to be safe
	return e.rebuild()
}

// Commit creates a new commit
func (e *engineImpl) Commit(_ context.Context, message string, verbose int, noVerify bool) error {
	return e.git.CommitWithOptions(git.CommitOptions{
		Message:  message,
		Verbose:  verbose,
		NoVerify: noVerify,
	})
}

// CommitWithOptions creates a new commit with the given options
func (e *engineImpl) CommitWithOptions(_ context.Context, opts git.CommitOptions) error {
	return e.git.CommitWithOptions(opts)
}

// StageAll stages all changes
func (e *engineImpl) StageAll(ctx context.Context) error {
	return e.git.StageAll(ctx)
}

// StagePatch stages changes interactively
func (e *engineImpl) StagePatch(ctx context.Context) error {
	return e.git.StagePatch(ctx)
}

// StageHunks stages specific hunks by applying them as patches
func (e *engineImpl) StageHunks(ctx context.Context, hunks []git.Hunk) error {
	return e.git.StageHunks(ctx, hunks)
}

// StashPush pushes current changes to the stash
func (e *engineImpl) StashPush(ctx context.Context, message string) (string, error) {
	return e.git.StashPush(ctx, message)
}

// StashPushStaged pushes only staged changes to the stash
func (e *engineImpl) StashPushStaged(ctx context.Context, message string) (string, error) {
	return e.git.StashPushStaged(ctx, message)
}

// StashPop pops the most recent stash
func (e *engineImpl) StashPop(ctx context.Context) error {
	return e.git.StashPop(ctx)
}

// AddWorktree adds a new worktree
func (e *engineImpl) AddWorktree(ctx context.Context, path string, branch string, detach bool) error {
	return e.git.AddWorktree(ctx, path, branch, detach)
}

// RemoveWorktree removes a worktree
func (e *engineImpl) RemoveWorktree(ctx context.Context, path string) error {
	return e.git.RemoveWorktree(ctx, path)
}

// CreateTemporaryWorktree creates a temporary directory and adds a detached worktree
func (e *engineImpl) CreateTemporaryWorktree(ctx context.Context, branch string, prefix string) (string, func(), error) {
	tmpDir, err := os.MkdirTemp("", prefix)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create temporary directory: %w", err)
	}

	worktreePath := filepath.Join(tmpDir, "worktree")

	if err := e.AddWorktree(ctx, worktreePath, branch, true); err != nil {
		_ = os.RemoveAll(tmpDir)
		return "", nil, fmt.Errorf("failed to add worktree: %w", err)
	}

	cleanup := func() {
		// Use context.Background for cleanup to ensure it runs even if ctx is canceled
		_ = e.RemoveWorktree(context.Background(), worktreePath)
		_ = os.RemoveAll(tmpDir)
	}

	return worktreePath, cleanup, nil
}

// RegisterWorktree registers a worktree for a stack root in local git refs
func (e *engineImpl) RegisterWorktree(stackRoot string, path string) error {
	return e.RegisterWorktreeWithName(stackRoot, path, "")
}

// RegisterWorktreeWithName registers a worktree with a user-friendly name
func (e *engineImpl) RegisterWorktreeWithName(anchorBranch string, path string, name string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	meta := &git.WorktreeMeta{
		Name:         name,
		Path:         absPath,
		AnchorBranch: anchorBranch,
		CreatedAt:    timeNow(),
		MainRepoDir:  e.repoRoot,
	}

	return e.git.WriteWorktreeMeta(anchorBranch, meta)
}

// UnregisterWorktree removes worktree registration for a stack root
func (e *engineImpl) UnregisterWorktree(stackRoot string) error {
	return e.git.DeleteWorktreeMeta(stackRoot)
}

// GetWorktreeForStack returns worktree info for a stack root, or nil if none
func (e *engineImpl) GetWorktreeForStack(stackRoot string) (*WorktreeInfo, error) {
	meta, err := e.git.ReadWorktreeMeta(stackRoot)
	if err != nil {
		return nil, err
	}
	if meta == nil {
		return nil, nil
	}

	return &WorktreeInfo{
		Name:         meta.Name,
		Path:         meta.Path,
		AnchorBranch: meta.AnchorBranch,
		CreatedAt:    meta.CreatedAt,
		MainRepoDir:  meta.MainRepoDir,
	}, nil
}

// ListManagedWorktrees returns all stackit-managed worktrees, sorted by stack root name
func (e *engineImpl) ListManagedWorktrees() ([]WorktreeInfo, error) {
	metas, err := e.git.ListWorktreeMetas()
	if err != nil {
		return nil, err
	}

	// Sort keys for deterministic output
	keys := make([]string, 0, len(metas))
	for k := range metas {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	result := make([]WorktreeInfo, 0, len(metas))
	for _, k := range keys {
		meta := metas[k]
		result = append(result, WorktreeInfo{
			Name:         meta.Name,
			Path:         meta.Path,
			AnchorBranch: meta.AnchorBranch,
			CreatedAt:    meta.CreatedAt,
			MainRepoDir:  meta.MainRepoDir,
		})
	}

	return result, nil
}

// GetStackRootForBranch returns the stack root for a given branch.
// The stack root is the first ancestor branch whose parent is trunk.
// Returns empty string for trunk or untracked branches.
func (e *engineImpl) GetStackRootForBranch(branch Branch) string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	branchName := branch.GetName()

	// Trunk has no stack root
	if branchName == e.trunk {
		return ""
	}

	// Check if branch is tracked at all
	if !e.branchState.HasByName(branchName) {
		return "" // Untracked branch has no stack root
	}

	current := branchName
	for {
		state := e.branchState.GetByName(current)
		if state == nil {
			// Should not happen since we checked above, but handle gracefully
			return ""
		}

		// If parent is trunk, current is the stack root
		if state.Parent == e.trunk {
			return current
		}

		current = state.Parent
	}
}

// IsInManagedWorktree checks if the current directory is a stackit-managed worktree.
// Returns true and worktree info if in a managed worktree, false otherwise.
func (e *engineImpl) IsInManagedWorktree() (bool, *WorktreeInfo, error) {
	// Check if .git is a file (worktree) vs directory (main repo)
	gitPath := filepath.Join(e.repoRoot, ".git")
	info, err := os.Stat(gitPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil, nil // Not in a git repo
		}
		return false, nil, fmt.Errorf("failed to stat .git: %w", err)
	}

	// If .git is a directory, we're in the main repo, not a worktree
	if info.IsDir() {
		return false, nil, nil
	}

	// .git is a file - we're in a worktree. Now check if it's stackit-managed.
	// Get the current working directory (worktree path)
	currentPath, err := filepath.Abs(e.repoRoot)
	if err != nil {
		return false, nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// List all managed worktrees and check if current path matches
	worktrees, err := e.ListManagedWorktrees()
	if err != nil {
		return false, nil, fmt.Errorf("failed to list managed worktrees: %w", err)
	}

	for _, wt := range worktrees {
		// Compare paths (normalize both)
		wtPath, err := filepath.Abs(wt.Path)
		if err != nil {
			continue
		}
		if wtPath == currentPath {
			return true, &WorktreeInfo{
				Name:         wt.Name,
				Path:         wt.Path,
				AnchorBranch: wt.AnchorBranch,
				CreatedAt:    wt.CreatedAt,
				MainRepoDir:  wt.MainRepoDir,
			}, nil
		}
	}

	// It's a worktree but not managed by stackit
	return false, nil, nil
}

// setParentInternal updates parent without locking (caller must hold lock)
func (e *engineImpl) setParentInternal(ctx context.Context, branchName string, parentBranchName string) error {
	if branchName == parentBranchName {
		return fmt.Errorf("branch %s cannot be its own parent", branchName)
	}

	// Get new parent revision
	parentRev, err := e.git.GetMergeBase(branchName, parentBranchName)
	if err != nil {
		return fmt.Errorf("failed to get merge base: %w", err)
	}

	// Read existing metadata
	meta, err := e.git.ReadMetadata(branchName)
	if err != nil {
		return fmt.Errorf("failed to read metadata: %w", err)
	}

	// Update parent
	oldParent := ""
	if meta.ParentBranchName != nil {
		oldParent = *meta.ParentBranchName
	}

	// Only update ParentBranchRevision if it's currently nil, invalid, or if we're not
	// in a "parent merged into trunk" situation.
	shouldUpdateRevision := true
	if oldParent != "" && oldParent != parentBranchName && meta.ParentBranchRevision != nil && *meta.ParentBranchRevision != "" {
		// Check if existing revision is still a valid ancestor of the branch
		if isAncestor, _ := e.git.IsAncestor(*meta.ParentBranchRevision, branchName); isAncestor {
			// Check if the old parent was merged into the new parent (the "merge" case)
			// OR if the new parent is the same as the old parent (no change)
			// We use the branch name to check for merging.
			if merged, _ := e.git.IsMerged(ctx, oldParent, parentBranchName); merged {
				shouldUpdateRevision = false
			}
		}
	}

	meta.ParentBranchName = &parentBranchName
	if shouldUpdateRevision {
		meta.ParentBranchRevision = &parentRev
	}

	// Write metadata
	if err := e.git.WriteMetadata(branchName, meta); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	// Update in-memory maps
	if oldParent != "" {
		// Remove from old parent's children
		oldChildren := e.childrenMap[oldParent]
		if i := slices.Index(oldChildren, branchName); i >= 0 {
			e.childrenMap[oldParent] = slices.Delete(oldChildren, i, i+1)
		}
	}

	// Update branchState with new parent
	state := e.branchState.GetOrCreate(branchName)
	state.Parent = parentBranchName
	if e.childrenMap[parentBranchName] == nil {
		e.childrenMap[parentBranchName] = []string{}
	}

	// Check if already in children list
	found := false
	for _, c := range e.childrenMap[parentBranchName] {
		if c == branchName {
			found = true
			break
		}
	}
	if !found {
		e.childrenMap[parentBranchName] = append(e.childrenMap[parentBranchName], branchName)
		slices.Sort(e.childrenMap[parentBranchName])
	}

	return nil
}
