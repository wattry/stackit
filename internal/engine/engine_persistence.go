package engine

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"stackit.dev/stackit/internal/utils/concurrency"
)

const (
	// MetadataRefPrefix is the prefix for Git refs where branch metadata is stored
	MetadataRefPrefix = "refs/stackit/metadata/"
	// LocalMetadataRefPrefix is the prefix for Git refs where local-only branch metadata is stored
	LocalMetadataRefPrefix = "refs/stackit/local-metadata/"
)

// ReadMetadataRef reads metadata for a branch from Git refs
func (e *engineImpl) ReadMetadataRef(branchName string) (*Meta, error) {
	return e.readMetadataRef(branchName)
}

// readMetadataRef reads metadata for a branch from Git refs
func (e *engineImpl) readMetadataRef(branchName string) (*Meta, error) {
	refName := fmt.Sprintf("%s%s", MetadataRefPrefix, branchName)

	sha, err := e.git.GetRef(refName)
	if err != nil {
		// If ref doesn't exist, it's not an error, just means no metadata
		return &Meta{}, nil //nolint:nilerr
	}

	content, err := e.git.ReadBlob(sha)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata blob %s: %w", sha, err)
	}

	if content == "" {
		return &Meta{}, nil
	}

	var meta Meta
	if err := json.Unmarshal([]byte(content), &meta); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata for %s: %w", branchName, err)
	}

	return &meta, nil
}

// readLocalMetadataRef reads local-only metadata for a branch from Git refs
func (e *engineImpl) readLocalMetadataRef(branchName string) (*LocalMeta, error) {
	refName := fmt.Sprintf("%s%s", LocalMetadataRefPrefix, branchName)

	sha, err := e.git.GetRef(refName)
	if err != nil {
		// If ref doesn't exist, it's not an error, just means no local metadata
		return &LocalMeta{}, nil //nolint:nilerr
	}

	content, err := e.git.ReadBlob(sha)
	if err != nil {
		return nil, fmt.Errorf("failed to read local metadata blob %s: %w", sha, err)
	}

	if content == "" {
		return &LocalMeta{}, nil
	}

	var meta LocalMeta
	if err := json.Unmarshal([]byte(content), &meta); err != nil {
		return nil, fmt.Errorf("failed to unmarshal local metadata for %s: %w", branchName, err)
	}

	return &meta, nil
}

// writeLocalMetadataRef writes local-only metadata for a branch to Git refs
func (e *engineImpl) writeLocalMetadataRef(branchName string, meta *LocalMeta) error {
	jsonData, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("failed to marshal local metadata: %w", err)
	}

	sha, err := e.git.CreateBlob(string(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create local metadata blob: %w", err)
	}

	refName := fmt.Sprintf("%s%s", LocalMetadataRefPrefix, branchName)
	if err := e.git.UpdateRef(refName, sha); err != nil {
		return fmt.Errorf("failed to write local metadata ref: %w", err)
	}

	return nil
}

// WriteMetadataRef writes metadata for a branch to Git refs
func (e *engineImpl) WriteMetadataRef(branch Branch, meta *Meta) error {
	return e.writeMetadataRef(branch.GetName(), meta)
}

// writeMetadataRef writes metadata for a branch to Git refs
func (e *engineImpl) writeMetadataRef(branchName string, meta *Meta) error {
	jsonData, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	sha, err := e.git.CreateBlob(string(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create metadata blob: %w", err)
	}

	refName := fmt.Sprintf("%s%s", MetadataRefPrefix, branchName)
	if err := e.git.UpdateRef(refName, sha); err != nil {
		return fmt.Errorf("failed to write metadata ref: %w", err)
	}

	return nil
}

// DeleteMetadataRef deletes a metadata ref for a branch
func (e *engineImpl) DeleteMetadataRef(branch Branch) error {
	refName := fmt.Sprintf("%s%s", MetadataRefPrefix, branch.GetName())
	return e.git.DeleteRef(refName)
}

// RenameMetadataRef copies metadata from old branch name to new branch name.
// The old metadata ref is kept for potential cleanup by garbage collection
// or for collaborative scenarios where others still reference the old name.
func (e *engineImpl) RenameMetadataRef(oldBranch, newBranch Branch) error {
	oldRefName := fmt.Sprintf("%s%s", MetadataRefPrefix, oldBranch.GetName())
	newRefName := fmt.Sprintf("%s%s", MetadataRefPrefix, newBranch.GetName())

	sha, err := e.git.GetRef(oldRefName)
	if err != nil {
		return nil //nolint:nilerr // Nothing to rename
	}

	// Copy metadata to new ref (keep old ref for cleanup later)
	if err := e.git.UpdateRef(newRefName, sha); err != nil {
		return fmt.Errorf("failed to create new metadata ref: %w", err)
	}

	// Note: We intentionally keep the old ref around for collaborative scenarios.
	// The old metadata will be cleaned up during garbage collection or when
	// the orphaned metadata detection runs during sync.

	return nil
}

// BatchReadMetadataRefs reads metadata for multiple branches in parallel
func (e *engineImpl) BatchReadMetadataRefs(branchNames []string) (map[string]*Meta, map[string]error) {
	return e.batchReadMetadataRefs(branchNames)
}

// batchReadMetadataRefs reads metadata for multiple branches in parallel using a worker pool
func (e *engineImpl) batchReadMetadataRefs(branchNames []string) (map[string]*Meta, map[string]error) {
	results := make(map[string]*Meta)
	errs := make(map[string]error)
	resultsMu := sync.Mutex{}
	errsMu := sync.Mutex{}

	if len(branchNames) == 0 {
		return results, errs
	}

	concurrency.Run(branchNames, func(name string) {
		meta, err := e.readMetadataRef(name)
		if err != nil {
			errsMu.Lock()
			errs[name] = err
			errsMu.Unlock()
			return
		}
		resultsMu.Lock()
		results[name] = meta
		resultsMu.Unlock()
	})

	return results, errs
}

// batchReadLocalMetadataRefs reads local-only metadata for multiple branches in parallel using a worker pool
func (e *engineImpl) batchReadLocalMetadataRefs(branchNames []string) (map[string]*LocalMeta, map[string]error) {
	results := make(map[string]*LocalMeta)
	errs := make(map[string]error)
	resultsMu := sync.Mutex{}
	errsMu := sync.Mutex{}

	if len(branchNames) == 0 {
		return results, errs
	}

	concurrency.Run(branchNames, func(name string) {
		meta, err := e.readLocalMetadataRef(name)
		if err != nil {
			errsMu.Lock()
			errs[name] = err
			errsMu.Unlock()
			return
		}
		resultsMu.Lock()
		results[name] = meta
		resultsMu.Unlock()
	})

	return results, errs
}

// ListMetadataRefs returns all metadata refs
func (e *engineImpl) ListMetadataRefs() (map[string]string, error) {
	refs, err := e.git.ListRefs(MetadataRefPrefix)
	if err != nil {
		return nil, err
	}

	// Remove prefix from branch names
	result := make(map[string]string)
	for refName, sha := range refs {
		branchName := strings.TrimPrefix(refName, MetadataRefPrefix)
		result[branchName] = sha
	}
	return result, nil
}
