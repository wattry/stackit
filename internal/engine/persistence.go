package engine

import "time"

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
	Locked               bool               `json:"locked,omitempty"`

	// New fields for remote sync
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
	Number  *int    `json:"number,omitempty"`
	Base    *string `json:"base,omitempty"`
	URL     *string `json:"url,omitempty"`
	Title   *string `json:"title,omitempty"`
	Body    *string `json:"body,omitempty"`
	State   *string `json:"state,omitempty"`
	IsDraft *bool   `json:"isDraft,omitempty"`
}
