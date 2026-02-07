package git

import (
	"strings"
	"sync"
)

// metadataCache provides thread-safe caching of branch metadata.
// Meta values are immutable, so the cache can safely return shared pointers
// without deep copying.
type metadataCache struct {
	entries sync.Map // map[string]*Meta
}

// Get returns the cached metadata for the given branch, or nil if not cached.
func (c *metadataCache) Get(branchName string) *Meta {
	value, ok := c.entries.Load(branchName)
	if !ok {
		return nil
	}
	meta, _ := value.(*Meta)
	return meta
}

// Put stores the metadata in the cache.
func (c *metadataCache) Put(branchName string, meta *Meta) {
	c.entries.Store(branchName, meta)
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
