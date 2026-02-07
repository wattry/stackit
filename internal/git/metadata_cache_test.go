package git

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMetadataCache_GetPut(t *testing.T) {
	t.Parallel()

	var c metadataCache

	// Miss returns nil
	require.Nil(t, c.Get("main"))

	// Put then hit
	meta := NewMeta().WithLockReason(LockReasonUser)
	c.Put("main", meta)
	got := c.Get("main")
	require.NotNil(t, got)
	require.Equal(t, LockReasonUser, got.GetLockReason())

	// Overwrite
	meta2 := NewMeta().WithLockReason(LockReasonConsolidating)
	c.Put("main", meta2)
	got = c.Get("main")
	require.Equal(t, LockReasonConsolidating, got.GetLockReason())
}

func TestMetadataCache_Delete(t *testing.T) {
	t.Parallel()

	var c metadataCache
	c.Put("feature", NewMeta())

	c.Delete("feature")
	require.Nil(t, c.Get("feature"))

	// Delete non-existent key is a no-op
	c.Delete("nonexistent")
}

func TestMetadataCache_InvalidateForRefs(t *testing.T) {
	t.Parallel()

	var c metadataCache
	c.Put("feature-a", NewMeta())
	c.Put("feature-b", NewMeta())
	c.Put("feature-c", NewMeta())

	updates := []RefUpdate{
		{RefName: MetadataRefPrefix + "feature-a", NewSHA: "abc"},
		{RefName: "refs/heads/feature-b", NewSHA: "def"}, // Not a metadata ref
		{RefName: MetadataRefPrefix + "feature-c", NewSHA: "ghi"},
	}

	c.InvalidateForRefs(updates)

	// Metadata refs should be invalidated
	require.Nil(t, c.Get("feature-a"))
	require.Nil(t, c.Get("feature-c"))

	// Non-metadata ref should not be invalidated
	require.NotNil(t, c.Get("feature-b"))
}

func TestMetadataCache_InvalidateForRefNames(t *testing.T) {
	t.Parallel()

	var c metadataCache
	c.Put("feature-x", NewMeta())
	c.Put("feature-y", NewMeta())

	refNames := []string{
		MetadataRefPrefix + "feature-x",
		"refs/heads/feature-y", // Not a metadata ref
	}

	c.InvalidateForRefNames(refNames)

	require.Nil(t, c.Get("feature-x"))
	require.NotNil(t, c.Get("feature-y"))
}

func TestMetadataCache_Concurrent(t *testing.T) {
	t.Parallel()

	var c metadataCache
	var wg sync.WaitGroup
	const goroutines = 50

	// Concurrent Put/Get/Delete should not panic
	wg.Add(goroutines)
	for range goroutines {
		go func() {
			defer wg.Done()
			key := "branch"
			c.Put(key, NewMeta())
			c.Get(key)
			c.Delete(key)
			c.InvalidateForRefs([]RefUpdate{
				{RefName: MetadataRefPrefix + key, NewSHA: "abc"},
			})
		}()
	}
	wg.Wait()
}
