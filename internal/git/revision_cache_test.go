package git

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRevisionCache_GetPut(t *testing.T) {
	t.Parallel()

	var c revisionCache

	// Miss returns empty
	sha, ok := c.Get("main")
	require.False(t, ok)
	require.Empty(t, sha)

	// Put then hit
	c.Put("main", "abc123")
	sha, ok = c.Get("main")
	require.True(t, ok)
	require.Equal(t, "abc123", sha)

	// Overwrite
	c.Put("main", "def456")
	sha, ok = c.Get("main")
	require.True(t, ok)
	require.Equal(t, "def456", sha)
}

func TestRevisionCache_Delete(t *testing.T) {
	t.Parallel()

	var c revisionCache
	c.Put("feature", "aaa")

	c.Delete("feature")
	_, ok := c.Get("feature")
	require.False(t, ok)

	// Delete non-existent key is a no-op
	c.Delete("nonexistent")
}

func TestRevisionCache_InvalidateAll(t *testing.T) {
	t.Parallel()

	var c revisionCache
	c.Put("a", "1")
	c.Put("b", "2")
	c.Put("c", "3")

	c.InvalidateAll()

	_, ok := c.Get("a")
	require.False(t, ok)
	_, ok = c.Get("b")
	require.False(t, ok)
	_, ok = c.Get("c")
	require.False(t, ok)

	// Cache is usable after invalidation
	c.Put("d", "4")
	sha, ok := c.Get("d")
	require.True(t, ok)
	require.Equal(t, "4", sha)
}

func TestRevisionCache_Concurrent(t *testing.T) {
	t.Parallel()

	var c revisionCache
	var wg sync.WaitGroup
	const goroutines = 50

	// Concurrent Put/Get/Delete should not panic
	wg.Add(goroutines)
	for i := range goroutines {
		go func(id int) {
			defer wg.Done()
			key := "branch"
			c.Put(key, "sha")
			c.Get(key)
			c.Delete(key)
			if id%5 == 0 {
				c.InvalidateAll()
			}
		}(i)
	}
	wg.Wait()
}
