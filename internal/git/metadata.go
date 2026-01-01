package git

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"stackit.dev/stackit/internal/utils"
)

// BranchType indicates the type of branch
type BranchType string

// Branch types
const (
	BranchTypeUser    BranchType = "user"    // Normal stacked branch
	BranchTypeUtility BranchType = "utility" // Created by st merge --consolidate or other internal tasks
)

// Meta represents branch metadata stored in Git refs
type Meta struct {
	ParentBranchName     *string            `json:"parentBranchName,omitempty"`
	ParentBranchRevision *string            `json:"parentBranchRevision,omitempty"`
	PrInfo               *PrInfoPersistence `json:"prInfo,omitempty"`
	Scope                *string            `json:"scope,omitempty"`
	LockReason           string             `json:"lockReason,omitempty"`

	// Fields for remote sync
	BranchType     BranchType  `json:"branchType,omitempty"`     // Type of branch (regular, consolidated)
	LastModifiedBy *ModifiedBy `json:"lastModifiedBy,omitempty"` // Who last changed this metadata
	LastModifiedAt *time.Time  `json:"lastModifiedAt,omitempty"` // When metadata was last changed
	LocalOnlyHash  *string     `json:"localOnlyHash,omitempty"`  // Hash of local-only state for change detection
}

// LocalMeta represents branch metadata that is strictly local and never pushed
type LocalMeta struct {
	Frozen bool `json:"frozen,omitempty"`
}

// ModifiedBy represents information about who last modified the metadata
type ModifiedBy struct {
	GitName        string  `json:"gitName"`
	GitEmail       string  `json:"gitEmail"`
	GitHubUsername *string `json:"githubUsername,omitempty"`
}

// PrInfoPersistence represents PR information for persistence
type PrInfoPersistence struct {
	Number          *int    `json:"number,omitempty"`
	Base            *string `json:"base,omitempty"`
	URL             *string `json:"url,omitempty"`
	Title           *string `json:"title,omitempty"`
	Body            *string `json:"body,omitempty"`
	State           *string `json:"state,omitempty"`
	IsDraft         *bool   `json:"isDraft,omitempty"`
	LockReason      *string `json:"lockReason,omitempty"`
	ConsolidationPR *int    `json:"consolidationPR,omitempty"`
}

const (
	// MetadataRefPrefix is the prefix for Git refs where branch metadata is stored
	MetadataRefPrefix = "refs/stackit/metadata/"
	// LocalMetadataRefPrefix is the prefix for Git refs where local-only branch metadata is stored
	LocalMetadataRefPrefix = "refs/stackit/local-metadata/"
)

// BatchReadMetadata reads metadata for multiple branches in parallel.
// Returns two maps: one with successfully read metadata and one with errors for failed reads.
// Branches that don't have metadata will have an empty Meta struct in the results map.
// Only actual errors (not missing metadata) will be included in the errors map.
func (r *runner) BatchReadMetadata(branchNames []string) (map[string]*Meta, map[string]error) {
	results := make(map[string]*Meta)
	errs := make(map[string]error)
	resultsMu := sync.Mutex{}
	errsMu := sync.Mutex{}

	if len(branchNames) == 0 {
		return results, errs
	}

	utils.Run(branchNames, func(name string) {
		meta, err := r.ReadMetadata(name)
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

// BatchReadLocalMetadata reads local metadata for multiple branches in parallel.
// Returns a map of successfully read metadata. Failures are silently ignored since
// local metadata is not critical and missing metadata is expected for new branches.
func (r *runner) BatchReadLocalMetadata(branchNames []string) map[string]*LocalMeta {
	results := make(map[string]*LocalMeta)
	resultsMu := sync.Mutex{}

	if len(branchNames) == 0 {
		return results
	}

	utils.Run(branchNames, func(name string) {
		meta, err := r.ReadLocalMetadata(name)
		if err != nil {
			// Local metadata failure is not critical, just skip it
			return
		}
		resultsMu.Lock()
		results[name] = meta
		resultsMu.Unlock()
	})

	return results
}

func (r *runner) ReadMetadata(branchName string) (*Meta, error) {
	refName := fmt.Sprintf("%s%s", MetadataRefPrefix, branchName)

	sha, err := r.GetRef(refName)
	if err != nil {
		// If ref doesn't exist, it's not an error, just means no metadata
		return &Meta{}, nil //nolint:nilerr
	}

	content, err := r.ReadBlob(sha)
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

func (r *runner) WriteMetadata(branchName string, meta *Meta) error {
	jsonData, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	sha, err := r.CreateBlob(string(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create metadata blob: %w", err)
	}

	refName := fmt.Sprintf("%s%s", MetadataRefPrefix, branchName)
	if err := r.UpdateRef(refName, sha); err != nil {
		return fmt.Errorf("failed to write metadata ref: %w", err)
	}

	return nil
}

func (r *runner) DeleteMetadata(branchName string) error {
	refName := fmt.Sprintf("%s%s", MetadataRefPrefix, branchName)
	return r.DeleteRef(refName)
}

func (r *runner) RenameMetadata(oldName, newName string) error {
	oldRefName := fmt.Sprintf("%s%s", MetadataRefPrefix, oldName)
	newRefName := fmt.Sprintf("%s%s", MetadataRefPrefix, newName)

	sha, err := r.GetRef(oldRefName)
	if err != nil {
		return nil //nolint:nilerr // Nothing to rename
	}

	// Copy metadata to new ref (keep old ref for cleanup later)
	if err := r.UpdateRef(newRefName, sha); err != nil {
		return fmt.Errorf("failed to create new metadata ref: %w", err)
	}

	return nil
}

func (r *runner) ReadLocalMetadata(branchName string) (*LocalMeta, error) {
	refName := fmt.Sprintf("%s%s", LocalMetadataRefPrefix, branchName)

	sha, err := r.GetRef(refName)
	if err != nil {
		// If ref doesn't exist, it's not an error, just means no local metadata
		return &LocalMeta{}, nil //nolint:nilerr
	}

	content, err := r.ReadBlob(sha)
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

func (r *runner) WriteLocalMetadata(branchName string, meta *LocalMeta) error {
	jsonData, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("failed to marshal local metadata: %w", err)
	}

	sha, err := r.CreateBlob(string(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create local metadata blob: %w", err)
	}

	refName := fmt.Sprintf("%s%s", LocalMetadataRefPrefix, branchName)
	if err := r.UpdateRef(refName, sha); err != nil {
		return fmt.Errorf("failed to write local metadata ref: %w", err)
	}

	return nil
}

func (r *runner) ListMetadata() (map[string]string, error) {
	refs, err := r.ListRefs(MetadataRefPrefix)
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
