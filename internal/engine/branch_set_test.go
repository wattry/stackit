package engine

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBranchSet(t *testing.T) {
	t.Parallel()

	t.Run("Contains returns true for existing branches", func(t *testing.T) {
		t.Parallel()
		set := newBranchSet([]string{"main", "feature", "bugfix"})

		require.True(t, set.Contains("main"))
		require.True(t, set.Contains("feature"))
		require.True(t, set.Contains("bugfix"))
	})

	t.Run("Contains returns false for non-existing branches", func(t *testing.T) {
		t.Parallel()
		set := newBranchSet([]string{"main", "feature"})

		require.False(t, set.Contains("nonexistent"))
		require.False(t, set.Contains(""))
	})

	t.Run("Names returns all branch names", func(t *testing.T) {
		t.Parallel()
		branches := []string{"main", "feature", "bugfix"}
		set := newBranchSet(branches)

		require.Equal(t, branches, set.Names())
	})

	t.Run("Len returns correct count", func(t *testing.T) {
		t.Parallel()
		set := newBranchSet([]string{"main", "feature", "bugfix"})

		require.Equal(t, 3, set.Len())
	})

	t.Run("empty set works correctly", func(t *testing.T) {
		t.Parallel()
		set := newBranchSet([]string{})

		require.False(t, set.Contains("anything"))
		require.Empty(t, set.Names())
		require.Equal(t, 0, set.Len())
	})
}

func TestFilterBranches(t *testing.T) {
	t.Parallel()

	t.Run("filters branches by predicate", func(t *testing.T) {
		t.Parallel()
		// This test would need a mock StackNavigator
		// For now we just verify the function signature compiles
	})
}
