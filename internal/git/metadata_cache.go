package git

import (
	"strings"
	"sync"
)

// metadataCache provides thread-safe caching of branch metadata.
// Values stored in the cache are treated as immutable — callers receive
// deep copies so mutations never affect cached state.
type metadataCache struct {
	entries sync.Map // map[string]*Meta
}

// Get returns a deep copy of the cached metadata for the given branch, or nil if not cached.
func (c *metadataCache) Get(branchName string) *Meta {
	value, ok := c.entries.Load(branchName)
	if !ok {
		return nil
	}
	original := value.(*Meta)
	return deepCopyMeta(original)
}

// Put stores a deep copy of the metadata in the cache.
// The caller retains ownership of the original — subsequent mutations
// to the passed-in Meta will not affect the cached value.
func (c *metadataCache) Put(branchName string, meta *Meta) {
	c.entries.Store(branchName, deepCopyMeta(meta))
}

// Delete removes the cached metadata for the given branch.
func (c *metadataCache) Delete(branchName string) {
	c.entries.Delete(branchName)
}

// InvalidateForRefs clears cached metadata for any refs in the given batch
// that are metadata refs. This ensures batch operations (transactions)
// don't leave stale cache entries.
func (c *metadataCache) InvalidateForRefs(updates []RefUpdate) {
	for _, update := range updates {
		if branchName, ok := strings.CutPrefix(update.RefName, MetadataRefPrefix); ok {
			c.entries.Delete(branchName)
		}
	}
}

// InvalidateForRefNames clears cached metadata for any ref names that
// are metadata refs. Used by DeleteRefsBatch which operates on raw ref names.
func (c *metadataCache) InvalidateForRefNames(refNames []string) {
	for _, refName := range refNames {
		if branchName, ok := strings.CutPrefix(refName, MetadataRefPrefix); ok {
			c.entries.Delete(branchName)
		}
	}
}

// deepCopyMeta returns a fully independent copy of a Meta struct.
// All pointer fields, slices, and nested structs are copied so that
// mutations to the copy never affect the original (and vice versa).
func deepCopyMeta(m *Meta) *Meta {
	if m == nil {
		return nil
	}

	out := *m // shallow copy of value types (LockReason, BranchType)

	out.ParentBranchName = copyStringPtr(m.ParentBranchName)
	out.ParentBranchRevision = copyStringPtr(m.ParentBranchRevision)
	out.Scope = copyStringPtr(m.Scope)
	out.LocalOnlyHash = copyStringPtr(m.LocalOnlyHash)
	out.StackID = copyStringPtr(m.StackID)

	if m.PrInfo != nil {
		out.PrInfo = deepCopyPrInfo(m.PrInfo)
	}

	if m.LastModifiedBy != nil {
		copied := *m.LastModifiedBy
		copied.GitHubUsername = copyStringPtr(m.LastModifiedBy.GitHubUsername)
		out.LastModifiedBy = &copied
	}

	if m.LastModifiedAt != nil {
		copied := *m.LastModifiedAt
		out.LastModifiedAt = &copied
	}

	if m.MergedDownstack != nil {
		out.MergedDownstack = make([]MergedParent, len(m.MergedDownstack))
		for i, mp := range m.MergedDownstack {
			out.MergedDownstack[i] = MergedParent{
				BranchName: mp.BranchName,
				PRNumber:   copyIntPtr(mp.PRNumber),
				PRState:    copyStringPtr(mp.PRState),
			}
		}
	}

	return &out
}

func deepCopyPrInfo(p *PrInfoPersistence) *PrInfoPersistence {
	out := *p
	out.Number = copyIntPtr(p.Number)
	out.Base = copyStringPtr(p.Base)
	out.URL = copyStringPtr(p.URL)
	out.Title = copyStringPtr(p.Title)
	out.Body = copyStringPtr(p.Body)
	out.State = copyStringPtr(p.State)
	out.MergeBranch = copyStringPtr(p.MergeBranch)

	if p.IsDraft != nil {
		copied := *p.IsDraft
		out.IsDraft = &copied
	}

	if p.LockReason != nil {
		copied := *p.LockReason
		out.LockReason = &copied
	}

	return &out
}

func copyStringPtr(s *string) *string {
	if s == nil {
		return nil
	}
	copied := *s
	return &copied
}

func copyIntPtr(n *int) *int {
	if n == nil {
		return nil
	}
	copied := *n
	return &copied
}
