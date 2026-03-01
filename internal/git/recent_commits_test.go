package git_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/testhelpers"
)

func TestGetRecentCommits_DuplicateTrailers(t *testing.T) {
	scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
		return s.Repo.CreateChangeAndCommit("initial", "init")
	})

	err := scene.Repo.CreateChange("change", "dup", false)
	require.NoError(t, err)
	err = scene.Repo.RunGitCommand("add", ".")
	require.NoError(t, err)
	err = scene.Repo.RunGitCommand(
		"commit",
		"-m", "Consolidate stack [PROJ-123]",
		"-m", "Stackit-Stack-Size: 2\nStackit-Stack-Size: 2\nStackit-PRs: 1,2\nStackit-PRs: 1,2\nStackit-Scope: PROJ-123\nStackit-Scope: PROJ-123",
	)
	require.NoError(t, err)

	runner := git.NewRunnerWithPath(scene.Dir, nil)
	commits, err := runner.GetRecentCommits("main", 1)
	require.NoError(t, err)
	require.Len(t, commits, 1)

	require.Equal(t, 2, commits[0].StackSize)
	require.Equal(t, "1,2", commits[0].StackPRs)
	require.Equal(t, "PROJ-123", commits[0].StackScope)
}
