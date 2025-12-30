package engine

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"stackit.dev/stackit/internal/git"
)

// IsRemoteSyncEnabled checks if metadata compatibility has been verified and sync is enabled
func (e *engineImpl) IsRemoteSyncEnabled() bool {
	val, err := e.git.GetConfig("stackit.metadata-sync-enabled")
	return err == nil && val == "true"
}

// SetRemoteSyncEnabled marks metadata compatibility as verified and enables/disables sync
func (e *engineImpl) SetRemoteSyncEnabled(enabled bool) {
	val := "false"
	if enabled {
		val = "true"
	}
	_ = e.git.SetConfig("stackit.metadata-sync-enabled", val)
}

// SetLastModifiedBy updates the metadata for a branch with the current user's information
func (e *engineImpl) SetLastModifiedBy(branchName string) error {
	name, err := git.GetUserName(context.Background())
	if err != nil {
		return fmt.Errorf("git user.name is required but not set: %w", err)
	}
	email, _ := e.git.GetConfig("user.email")

	modifiedBy := &ModifiedBy{
		GitName:  name,
		GitEmail: email,
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	meta, err := e.readMetadataRef(branchName)
	if err != nil {
		meta = &Meta{}
	}
	meta.LastModifiedBy = modifiedBy
	now := time.Now()
	meta.LastModifiedAt = &now

	// Update localOnlyHash for change detection
	hash := e.computeMetadataHash(meta)
	meta.LocalOnlyHash = &hash

	return e.writeMetadataRef(branchName, meta)
}

// LoadRemoteMetadataCache loads remote metadata refs into the engine's cache
func (e *engineImpl) LoadRemoteMetadataCache() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	remoteRefs, err := e.git.ListRefs("refs/stackit/remote-metadata/")
	if err != nil {
		return err
	}

	e.remoteMetaCache = make(map[string]*Meta)
	for refName, sha := range remoteRefs {
		branchName := refName[len("refs/stackit/remote-metadata/"):]
		content, err := e.git.ReadBlob(sha)
		if err != nil {
			continue
		}

		var meta Meta
		if err := json.Unmarshal([]byte(content), &meta); err != nil {
			continue
		}
		e.remoteMetaCache[branchName] = &meta
	}

	return nil
}

// ApplyRemoteMetadataIfExists applies remote metadata to a local branch if it exists in the cache
func (e *engineImpl) ApplyRemoteMetadataIfExists(branchName string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	remote, ok := e.remoteMetaCache[branchName]
	if !ok {
		return nil
	}

	// Update local metadata maps
	if remote.Locked {
		e.lockedMap[branchName] = true
	}
	if remote.Scope != nil {
		e.scopeMap[branchName] = *remote.Scope
	}

	// Read existing local metadata to preserve fields not in remote (like PrInfo)
	local, err := e.readMetadataRef(branchName)
	if err != nil {
		local = &Meta{}
	}

	// Update local with remote values
	local.Locked = remote.Locked
	local.Scope = remote.Scope
	local.BranchType = remote.BranchType
	local.LastModifiedBy = remote.LastModifiedBy
	local.LastModifiedAt = remote.LastModifiedAt

	// Store localOnlyHash for change detection
	hash := e.computeMetadataHash(local)
	local.LocalOnlyHash = &hash

	return e.writeMetadataRef(branchName, local)
}

// GetRemoteMetadataCache returns the current remote metadata cache
func (e *engineImpl) GetRemoteMetadataCache() map[string]*Meta {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Return a copy to avoid external modification
	cacheCopy := make(map[string]*Meta)
	for k, v := range e.remoteMetaCache {
		cacheCopy[k] = v
	}
	return cacheCopy
}

// ComputeMetadataDiff compares local and remote metadata for a branch
func (e *engineImpl) ComputeMetadataDiff(branch string) (*MetadataDiff, error) {
	e.mu.RLock()
	local, err := e.readMetadataRef(branch)
	remote := e.remoteMetaCache[branch]
	e.mu.RUnlock()

	if err != nil {
		return nil, err
	}

	if remote == nil {
		return nil, nil // No remote metadata, nothing to diff
	}

	diff := &MetadataDiff{
		Branch:     branch,
		LocalMeta:  local,
		RemoteMeta: remote,
	}

	// Compare syncable fields
	if local.Locked != remote.Locked {
		diff.Differences = append(diff.Differences, FieldDiff{
			Field:       "locked",
			LocalValue:  local.Locked,
			RemoteValue: remote.Locked,
		})
	}

	localScope := ""
	if local.Scope != nil {
		localScope = *local.Scope
	}
	remoteScope := ""
	if remote.Scope != nil {
		remoteScope = *remote.Scope
	}

	if localScope != remoteScope {
		diff.Differences = append(diff.Differences, FieldDiff{
			Field:       "scope",
			LocalValue:  localScope,
			RemoteValue: remoteScope,
		})
	}

	diff.HasConflict = len(diff.Differences) > 0
	return diff, nil
}

// ComputeAllMetadataDiffs computes diffs for all branches in the remote cache
func (e *engineImpl) ComputeAllMetadataDiffs() ([]*MetadataDiff, error) {
	e.mu.RLock()
	branches := make([]string, 0, len(e.remoteMetaCache))
	for b := range e.remoteMetaCache {
		branches = append(branches, b)
	}
	e.mu.RUnlock()

	var diffs []*MetadataDiff
	for _, branch := range branches {
		diff, err := e.ComputeMetadataDiff(branch)
		if err != nil {
			return nil, err
		}
		if diff != nil && diff.HasConflict {
			diffs = append(diffs, diff)
		}
	}
	return diffs, nil
}

// AcceptRemoteMetadata overwrites local metadata with remote values
func (e *engineImpl) AcceptRemoteMetadata(branch string) error {
	return e.ApplyRemoteMetadataIfExists(branch)
}

// RejectRemoteMetadata marks a branch as having local modifications to keep
func (e *engineImpl) RejectRemoteMetadata(branch string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.localModified[branch] = true
}

// HasLocalModifications checks if a branch has local metadata changes that differ from its original fetched state
func (e *engineImpl) HasLocalModifications(branch string) bool {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if e.localModified[branch] {
		return true
	}

	local, err := e.readMetadataRef(branch)
	if err != nil || local.LocalOnlyHash == nil {
		return false // Never synced or error, treat as not modified
	}

	currentHash := e.computeMetadataHash(local)
	return currentHash != *local.LocalOnlyHash
}

// computeMetadataHash calculates a hash of the syncable fields for change detection
func (e *engineImpl) computeMetadataHash(meta *Meta) string {
	// Hash of locked + scope (fields user can modify locally)
	scope := ""
	if meta.Scope != nil {
		scope = *meta.Scope
	}
	data := fmt.Sprintf("locked:%v,scope:%s", meta.Locked, scope)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// MetadataDiff represents the differences between local and remote metadata
type MetadataDiff struct {
	Branch      string
	LocalMeta   *Meta
	RemoteMeta  *Meta
	Differences []FieldDiff
	HasConflict bool
}

// FieldDiff represents a difference in a single metadata field
type FieldDiff struct {
	Field       string
	LocalValue  interface{}
	RemoteValue interface{}
}

// OrphanedMetadataAction represents the action to take for orphaned metadata
type OrphanedMetadataAction string

const (
	// OrphanedActionDelete indicates the local metadata should be deleted
	OrphanedActionDelete OrphanedMetadataAction = "delete"
	// OrphanedActionPrompt indicates the user should be prompted
	OrphanedActionPrompt OrphanedMetadataAction = "prompt"
)

// OrphanedMetadataInfo contains information about orphaned local metadata
type OrphanedMetadataInfo struct {
	BranchName      string
	Action          OrphanedMetadataAction
	HasLocalChanges bool
	LocalMeta       *Meta
}

// FindOrphanedLocalMetadata identifies branches that have local metadata but no remote metadata
// This handles the "dual-checkout scenario" where someone else deleted the branch/metadata
func (e *engineImpl) FindOrphanedLocalMetadata() ([]OrphanedMetadataInfo, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// List all local metadata refs
	localRefs, err := e.git.ListRefs("refs/stackit/metadata/")
	if err != nil {
		return nil, err
	}

	orphaned := make([]OrphanedMetadataInfo, 0, len(localRefs))

	for refName := range localRefs {
		branchName := refName[len("refs/stackit/metadata/"):]

		// Skip if remote metadata exists for this branch
		if _, hasRemote := e.remoteMetaCache[branchName]; hasRemote {
			continue
		}

		// Check if this branch was previously synced (has LocalOnlyHash)
		local, err := e.readMetadataRef(branchName)
		if err != nil {
			continue
		}

		// If LocalOnlyHash is nil, this branch was never synced from remote
		// so it's not orphaned - it's just a local-only branch
		if local.LocalOnlyHash == nil {
			continue
		}

		// This is orphaned metadata - remote was deleted but local still exists
		hasLocalChanges := e.computeMetadataHash(local) != *local.LocalOnlyHash

		action := OrphanedActionDelete
		if hasLocalChanges {
			action = OrphanedActionPrompt
		}

		orphaned = append(orphaned, OrphanedMetadataInfo{
			BranchName:      branchName,
			Action:          action,
			HasLocalChanges: hasLocalChanges,
			LocalMeta:       local,
		})
	}

	return orphaned, nil
}

// DeleteLocalMetadataHash removes the LocalOnlyHash from a branch's metadata
// This effectively "un-syncs" the branch so it won't be considered orphaned
func (e *engineImpl) DeleteLocalMetadataHash(branchName string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	local, err := e.readMetadataRef(branchName)
	if err != nil {
		return err
	}

	local.LocalOnlyHash = nil
	return e.writeMetadataRef(branchName, local)
}
