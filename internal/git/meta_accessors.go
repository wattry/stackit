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
		ParentBranchName:     f.ParentBranchName,
		ParentBranchRevision: f.ParentBranchRevision,
		PrInfo:               f.PrInfo,
		Scope:                f.Scope,
		LockReason:           f.LockReason,
		BranchType:           f.BranchType,
		LastModifiedBy:       f.LastModifiedBy,
		LastModifiedAt:       f.LastModifiedAt,
		LocalOnlyHash:        f.LocalOnlyHash,
		MergedDownstack:      f.MergedDownstack,
		StackID:              f.StackID,
	}
}

// --- Getters ---
// All getters handle nil receiver for safety.
// Simple value types (*string, LockReason, BranchType) return raw values.
// Complex types (*PrInfoPersistence, *ModifiedBy, *time.Time, []MergedParent) return copies.

func (m *Meta) GetParentBranchName() *string {
	if m == nil {
		return nil
	}
	return m.ParentBranchName
}

func (m *Meta) GetParentBranchRevision() *string {
	if m == nil {
		return nil
	}
	return m.ParentBranchRevision
}

func (m *Meta) GetPrInfo() *PrInfoPersistence {
	if m == nil || m.PrInfo == nil {
		return nil
	}
	cp := *m.PrInfo
	return &cp
}

func (m *Meta) GetScope() *string {
	if m == nil {
		return nil
	}
	return m.Scope
}

func (m *Meta) GetLockReason() LockReason {
	if m == nil {
		return LockReasonNone
	}
	return m.LockReason
}

func (m *Meta) GetBranchType() BranchType {
	if m == nil {
		return ""
	}
	return m.BranchType
}

func (m *Meta) GetLastModifiedBy() *ModifiedBy {
	if m == nil || m.LastModifiedBy == nil {
		return nil
	}
	cp := *m.LastModifiedBy
	return &cp
}

func (m *Meta) GetLastModifiedAt() *time.Time {
	if m == nil || m.LastModifiedAt == nil {
		return nil
	}
	cp := *m.LastModifiedAt
	return &cp
}

func (m *Meta) GetLocalOnlyHash() *string {
	if m == nil {
		return nil
	}
	return m.LocalOnlyHash
}

func (m *Meta) GetMergedDownstack() []MergedParent {
	if m == nil || m.MergedDownstack == nil {
		return nil
	}
	cp := make([]MergedParent, len(m.MergedDownstack))
	copy(cp, m.MergedDownstack)
	return cp
}

func (m *Meta) GetStackID() *string {
	if m == nil {
		return nil
	}
	return m.StackID
}

// --- With* methods ---
// Each returns a new Meta with the specified field changed (shallow struct copy).

func (m *Meta) WithParentBranchName(v *string) *Meta {
	c := *m
	c.ParentBranchName = v
	return &c
}

func (m *Meta) WithParentBranchRevision(v *string) *Meta {
	c := *m
	c.ParentBranchRevision = v
	return &c
}

func (m *Meta) WithPrInfo(v *PrInfoPersistence) *Meta {
	c := *m
	c.PrInfo = v
	return &c
}

func (m *Meta) WithScope(v *string) *Meta {
	c := *m
	c.Scope = v
	return &c
}

func (m *Meta) WithLockReason(v LockReason) *Meta {
	c := *m
	c.LockReason = v
	return &c
}

func (m *Meta) WithBranchType(v BranchType) *Meta {
	c := *m
	c.BranchType = v
	return &c
}

func (m *Meta) WithLastModifiedBy(v *ModifiedBy) *Meta {
	c := *m
	c.LastModifiedBy = v
	return &c
}

func (m *Meta) WithLastModifiedAt(v *time.Time) *Meta {
	c := *m
	c.LastModifiedAt = v
	return &c
}

func (m *Meta) WithLocalOnlyHash(v *string) *Meta {
	c := *m
	c.LocalOnlyHash = v
	return &c
}

func (m *Meta) WithMergedDownstack(v []MergedParent) *Meta {
	c := *m
	c.MergedDownstack = v
	return &c
}

func (m *Meta) WithStackID(v *string) *Meta {
	c := *m
	c.StackID = v
	return &c
}

// --- JSON serialization ---
// These methods use an alias to avoid infinite recursion and work
// correctly regardless of whether fields are exported or unexported.

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
		ParentBranchName:     m.ParentBranchName,
		ParentBranchRevision: m.ParentBranchRevision,
		PrInfo:               m.PrInfo,
		Scope:                m.Scope,
		LockReason:           m.LockReason,
		BranchType:           m.BranchType,
		LastModifiedBy:       m.LastModifiedBy,
		LastModifiedAt:       m.LastModifiedAt,
		LocalOnlyHash:        m.LocalOnlyHash,
		MergedDownstack:      m.MergedDownstack,
		StackID:              m.StackID,
	})
}

// UnmarshalJSON implements json.Unmarshaler for Meta.
func (m *Meta) UnmarshalJSON(data []byte) error {
	var j metaJSON
	if err := json.Unmarshal(data, &j); err != nil {
		return err
	}
	m.ParentBranchName = j.ParentBranchName
	m.ParentBranchRevision = j.ParentBranchRevision
	m.PrInfo = j.PrInfo
	m.Scope = j.Scope
	m.LockReason = j.LockReason
	m.BranchType = j.BranchType
	m.LastModifiedBy = j.LastModifiedBy
	m.LastModifiedAt = j.LastModifiedAt
	m.LocalOnlyHash = j.LocalOnlyHash
	m.MergedDownstack = j.MergedDownstack
	m.StackID = j.StackID
	return nil
}
