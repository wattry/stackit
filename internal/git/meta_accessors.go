package git

import (
	"encoding/json"
	"time"
)

// MetaFields provides exported fields for constructing an immutable Meta.
// Use with NewMetaFrom to build Meta values in a single expression.
type MetaFields struct {
	ParentBranchName     *string
	ParentBranchRevision *string
	PrInfo               *PrInfoPersistence
	Scope                *string
	LockReason           LockReason
	BranchType           BranchType
	LastModifiedBy       *ModifiedBy
	LastModifiedAt       *time.Time
	LocalOnlyHash        *string
	MergedDownstack      []MergedParent
	StackID              *string
}

// NewMeta returns a new zero-value Meta.
func NewMeta() *Meta {
	return &Meta{}
}

// NewMetaFrom constructs a Meta from the given fields.
func NewMetaFrom(f MetaFields) *Meta {
	return &Meta{
		parentBranchName:     f.ParentBranchName,
		parentBranchRevision: f.ParentBranchRevision,
		prInfo:               f.PrInfo,
		scope:                f.Scope,
		lockReason:           f.LockReason,
		branchType:           f.BranchType,
		lastModifiedBy:       f.LastModifiedBy,
		lastModifiedAt:       f.LastModifiedAt,
		localOnlyHash:        f.LocalOnlyHash,
		mergedDownstack:      f.MergedDownstack,
		stackID:              f.StackID,
	}
}

// --- Getters ---
// All getters handle nil receiver for safety.
// Simple value types (*string, LockReason, BranchType) return raw values.
// Complex types (*PrInfoPersistence, *ModifiedBy, *time.Time, []MergedParent) return copies.

// GetParentBranchName returns the parent branch name.
func (m *Meta) GetParentBranchName() *string {
	if m == nil {
		return nil
	}
	return m.parentBranchName
}

// GetParentBranchRevision returns the parent branch revision.
func (m *Meta) GetParentBranchRevision() *string {
	if m == nil {
		return nil
	}
	return m.parentBranchRevision
}

// GetPrInfo returns a shallow copy of the PR info.
func (m *Meta) GetPrInfo() *PrInfoPersistence {
	if m == nil || m.prInfo == nil {
		return nil
	}
	cp := *m.prInfo
	return &cp
}

// GetScope returns the scope.
func (m *Meta) GetScope() *string {
	if m == nil {
		return nil
	}
	return m.scope
}

// GetLockReason returns the lock reason.
func (m *Meta) GetLockReason() LockReason {
	if m == nil {
		return LockReasonNone
	}
	return m.lockReason
}

// GetBranchType returns the branch type.
func (m *Meta) GetBranchType() BranchType {
	if m == nil {
		return ""
	}
	return m.branchType
}

// GetLastModifiedBy returns a shallow copy of the last-modified-by info.
func (m *Meta) GetLastModifiedBy() *ModifiedBy {
	if m == nil || m.lastModifiedBy == nil {
		return nil
	}
	cp := *m.lastModifiedBy
	return &cp
}

// GetLastModifiedAt returns a copy of the last-modified-at timestamp.
func (m *Meta) GetLastModifiedAt() *time.Time {
	if m == nil || m.lastModifiedAt == nil {
		return nil
	}
	cp := *m.lastModifiedAt
	return &cp
}

// GetLocalOnlyHash returns the local-only hash.
func (m *Meta) GetLocalOnlyHash() *string {
	if m == nil {
		return nil
	}
	return m.localOnlyHash
}

// GetMergedDownstack returns a copy of the merged downstack history.
func (m *Meta) GetMergedDownstack() []MergedParent {
	if m == nil || m.mergedDownstack == nil {
		return nil
	}
	cp := make([]MergedParent, len(m.mergedDownstack))
	copy(cp, m.mergedDownstack)
	return cp
}

// GetStackID returns the stack ID.
func (m *Meta) GetStackID() *string {
	if m == nil {
		return nil
	}
	return m.stackID
}

// --- With* methods ---
// Each returns a new Meta with the specified field changed (shallow struct copy).

// WithParentBranchName returns a new Meta with the parent branch name set.
func (m *Meta) WithParentBranchName(v *string) *Meta {
	c := *m
	c.parentBranchName = v
	return &c
}

// WithParentBranchRevision returns a new Meta with the parent branch revision set.
func (m *Meta) WithParentBranchRevision(v *string) *Meta {
	c := *m
	c.parentBranchRevision = v
	return &c
}

// WithPrInfo returns a new Meta with the PR info set.
func (m *Meta) WithPrInfo(v *PrInfoPersistence) *Meta {
	c := *m
	c.prInfo = v
	return &c
}

// WithScope returns a new Meta with the scope set.
func (m *Meta) WithScope(v *string) *Meta {
	c := *m
	c.scope = v
	return &c
}

// WithLockReason returns a new Meta with the lock reason set.
func (m *Meta) WithLockReason(v LockReason) *Meta {
	c := *m
	c.lockReason = v
	return &c
}

// WithBranchType returns a new Meta with the branch type set.
func (m *Meta) WithBranchType(v BranchType) *Meta {
	c := *m
	c.branchType = v
	return &c
}

// WithLastModifiedBy returns a new Meta with the last-modified-by info set.
func (m *Meta) WithLastModifiedBy(v *ModifiedBy) *Meta {
	c := *m
	c.lastModifiedBy = v
	return &c
}

// WithLastModifiedAt returns a new Meta with the last-modified-at timestamp set.
func (m *Meta) WithLastModifiedAt(v *time.Time) *Meta {
	c := *m
	c.lastModifiedAt = v
	return &c
}

// WithLocalOnlyHash returns a new Meta with the local-only hash set.
func (m *Meta) WithLocalOnlyHash(v *string) *Meta {
	c := *m
	c.localOnlyHash = v
	return &c
}

// WithMergedDownstack returns a new Meta with the merged downstack history set.
func (m *Meta) WithMergedDownstack(v []MergedParent) *Meta {
	c := *m
	c.mergedDownstack = v
	return &c
}

// WithStackID returns a new Meta with the stack ID set.
func (m *Meta) WithStackID(v *string) *Meta {
	c := *m
	c.stackID = v
	return &c
}

// --- JSON serialization ---
// These methods use an alias to avoid infinite recursion and work
// correctly with unexported fields.

// metaJSON is the JSON wire format for Meta.
type metaJSON struct {
	ParentBranchName     *string            `json:"parentBranchName,omitempty"`
	ParentBranchRevision *string            `json:"parentBranchRevision,omitempty"`
	PrInfo               *PrInfoPersistence `json:"prInfo,omitempty"`
	Scope                *string            `json:"scope,omitempty"`
	LockReason           LockReason         `json:"lockReason,omitempty"`
	BranchType           BranchType         `json:"branchType,omitempty"`
	LastModifiedBy       *ModifiedBy        `json:"lastModifiedBy,omitempty"`
	LastModifiedAt       *time.Time         `json:"lastModifiedAt,omitempty"`
	LocalOnlyHash        *string            `json:"localOnlyHash,omitempty"`
	MergedDownstack      []MergedParent     `json:"mergedDownstack,omitempty"`
	StackID              *string            `json:"stackId,omitempty"`
}

// MarshalJSON implements json.Marshaler for Meta.
func (m Meta) MarshalJSON() ([]byte, error) {
	return json.Marshal(metaJSON{
		ParentBranchName:     m.parentBranchName,
		ParentBranchRevision: m.parentBranchRevision,
		PrInfo:               m.prInfo,
		Scope:                m.scope,
		LockReason:           m.lockReason,
		BranchType:           m.branchType,
		LastModifiedBy:       m.lastModifiedBy,
		LastModifiedAt:       m.lastModifiedAt,
		LocalOnlyHash:        m.localOnlyHash,
		MergedDownstack:      m.mergedDownstack,
		StackID:              m.stackID,
	})
}

// UnmarshalJSON implements json.Unmarshaler for Meta.
func (m *Meta) UnmarshalJSON(data []byte) error {
	var j metaJSON
	if err := json.Unmarshal(data, &j); err != nil {
		return err
	}
	m.parentBranchName = j.ParentBranchName
	m.parentBranchRevision = j.ParentBranchRevision
	m.prInfo = j.PrInfo
	m.scope = j.Scope
	m.lockReason = j.LockReason
	m.branchType = j.BranchType
	m.lastModifiedBy = j.LastModifiedBy
	m.lastModifiedAt = j.LastModifiedAt
	m.localOnlyHash = j.LocalOnlyHash
	m.mergedDownstack = j.MergedDownstack
	m.stackID = j.StackID
	return nil
}
