package httpcontract

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/internal/git"
)

func TestMapTrunkCommits(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
	commits := []git.RecentCommit{
		{
			SHA:            "1234567890abcdef",
			Subject:        "Consolidate auth stack",
			Author:         "alice",
			Date:           now,
			PRNumber:       123,
			Kind:           git.RecentCommitKindStackMerge,
			StackSize:      2,
			StackPRNumbers: []int{45, 46},
			StackScope:     "PROJ-1",
		},
		{
			SHA:     "abcdef1234567890",
			Subject: "Fix typo",
			Author:  "bob",
			Date:    now,
			Kind:    git.RecentCommitKindRegular,
		},
	}

	got := MapTrunkCommits(commits)
	require.Len(t, got, 2)

	require.Equal(t, "1234567", got[0].SHA)
	require.Equal(t, "Consolidate auth stack", got[0].Message)
	require.Equal(t, "alice", got[0].Author)
	require.Equal(t, now.Format(time.RFC3339), got[0].Date)
	require.Equal(t, 123, got[0].PRNumber)
	require.Equal(t, TrunkCommitKindStackMerge, got[0].Kind)
	require.Equal(t, 2, got[0].StackSize)
	require.Equal(t, []int{45, 46}, got[0].StackPRs)
	require.Equal(t, "PROJ-1", got[0].StackScope)

	require.Equal(t, "abcdef1", got[1].SHA)
	require.Equal(t, TrunkCommitKindRegular, got[1].Kind)
	require.Equal(t, 0, got[1].PRNumber)
	require.Empty(t, got[1].StackPRs)
}

func TestMapTrunkCommits_DefaultKindFallback(t *testing.T) {
	t.Parallel()

	commits := []git.RecentCommit{
		{
			SHA:       "1234567890abcdef",
			Subject:   "Consolidate stack",
			Author:    "alice",
			Date:      time.Now().UTC(),
			StackSize: 3,
		},
	}

	got := MapTrunkCommits(commits)
	require.Len(t, got, 1)
	require.Equal(t, TrunkCommitKindStackMerge, got[0].Kind)
}
