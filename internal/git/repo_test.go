package git_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/testhelpers"
)

func TestGetRepoInfo(t *testing.T) {
	t.Run("parses SCP-style SSH URL", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Set remote URL to SCP-style SSH format
		err := scene.Repo.RunGitCommand("config", "remote.origin.url", "git@github.com:myowner/myrepo.git")
		require.NoError(t, err)

		runner := git.NewRunnerWithPath(scene.Dir, nil)
		owner, repo, err := runner.GetRepoInfo(context.Background())
		require.NoError(t, err)
		require.Equal(t, "myowner", owner)
		require.Equal(t, "myrepo", repo)
	})

	t.Run("parses ssh:// protocol SSH URL", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Set remote URL to ssh:// protocol format
		err := scene.Repo.RunGitCommand("config", "remote.origin.url", "ssh://git@github.com/myowner/myrepo.git")
		require.NoError(t, err)

		runner := git.NewRunnerWithPath(scene.Dir, nil)
		owner, repo, err := runner.GetRepoInfo(context.Background())
		require.NoError(t, err)
		require.Equal(t, "myowner", owner)
		require.Equal(t, "myrepo", repo)
	})

	t.Run("parses ssh:// protocol SSH URL without .git suffix", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		err := scene.Repo.RunGitCommand("config", "remote.origin.url", "ssh://git@github.com/myowner/myrepo")
		require.NoError(t, err)

		runner := git.NewRunnerWithPath(scene.Dir, nil)
		owner, repo, err := runner.GetRepoInfo(context.Background())
		require.NoError(t, err)
		require.Equal(t, "myowner", owner)
		require.Equal(t, "myrepo", repo)
	})

	t.Run("parses ssh:// protocol with GitHub Enterprise", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		err := scene.Repo.RunGitCommand("config", "remote.origin.url", "ssh://git@github.enterprise.com/org/project.git")
		require.NoError(t, err)

		runner := git.NewRunnerWithPath(scene.Dir, nil)
		owner, repo, err := runner.GetRepoInfo(context.Background())
		require.NoError(t, err)
		require.Equal(t, "org", owner)
		require.Equal(t, "project", repo)
	})

	t.Run("parses HTTPS URL", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		err := scene.Repo.RunGitCommand("config", "remote.origin.url", "https://github.com/myowner/myrepo.git")
		require.NoError(t, err)

		runner := git.NewRunnerWithPath(scene.Dir, nil)
		owner, repo, err := runner.GetRepoInfo(context.Background())
		require.NoError(t, err)
		require.Equal(t, "myowner", owner)
		require.Equal(t, "myrepo", repo)
	})

	t.Run("returns empty for missing remote", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Don't set any remote URL
		runner := git.NewRunnerWithPath(scene.Dir, nil)
		owner, repo, err := runner.GetRepoInfo(context.Background())
		require.NoError(t, err)
		require.Empty(t, owner)
		require.Empty(t, repo)
	})
}
