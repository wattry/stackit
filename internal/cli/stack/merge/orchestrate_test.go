package merge

import (
	"testing"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/internal/github"
)

func TestIsReadyToMerge(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		mergeStateText string
		expected       bool
	}{
		{name: "CLEAN is ready", mergeStateText: "CLEAN", expected: true},
		{name: "HAS_HOOKS is ready", mergeStateText: "HAS_HOOKS", expected: true},
		{name: "BLOCKED is not ready", mergeStateText: "BLOCKED", expected: false},
		{name: "BEHIND is not ready", mergeStateText: "BEHIND", expected: false},
		{name: "DIRTY is not ready", mergeStateText: "DIRTY", expected: false},
		{name: "UNKNOWN is not ready", mergeStateText: "UNKNOWN", expected: false},
		{name: "empty is not ready", mergeStateText: "", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.expected, isReadyToMerge(tt.mergeStateText))
		})
	}
}

func TestFormatUnmergeableError(t *testing.T) {
	t.Parallel()

	t.Run("includes merge state text when present", func(t *testing.T) {
		t.Parallel()
		err := formatUnmergeableError(42, &github.PRMergeableState{
			MergeStateText: "DIRTY",
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "PR #42")
		require.Contains(t, err.Error(), "DIRTY")
		require.Contains(t, err.Error(), "not mergeable")
	})

	t.Run("generic message when merge state text is empty", func(t *testing.T) {
		t.Parallel()
		err := formatUnmergeableError(99, &github.PRMergeableState{
			MergeStateText: "",
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "PR #99")
		require.Contains(t, err.Error(), "not mergeable")
		require.NotContains(t, err.Error(), "()")
	})
}
