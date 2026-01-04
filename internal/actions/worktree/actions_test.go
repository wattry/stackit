package worktree_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/internal/actions/worktree"
	"stackit.dev/stackit/testhelpers"
	"stackit.dev/stackit/testhelpers/scenario"
)

func TestListAction(t *testing.T) {
	t.Run("returns empty list when no worktrees registered", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)
		s.WithInitialCommit()

		result, err := worktree.ListAction(s.Context, worktree.ListOptions{})
		require.NoError(t, err)
		assert.Empty(t, result.Worktrees)
	})

	t.Run("lists registered worktrees", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)
		s.WithInitialCommit()

		// Register a fake worktree
		err := s.Engine.RegisterWorktree("feature-stack", "/tmp/fake-worktree")
		require.NoError(t, err)

		result, err := worktree.ListAction(s.Context, worktree.ListOptions{})
		require.NoError(t, err)
		require.Len(t, result.Worktrees, 1)
		assert.Equal(t, "feature-stack", result.Worktrees[0].StackRoot)
		assert.Equal(t, "/tmp/fake-worktree", result.Worktrees[0].Path)
		assert.False(t, result.Worktrees[0].Exists) // Path doesn't actually exist

		// Clean up
		_ = s.Engine.UnregisterWorktree("feature-stack")
	})
}

func TestRemoveAction(t *testing.T) {
	t.Run("fails when worktree not found", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)
		s.WithInitialCommit()

		err := worktree.RemoveAction(s.Context, worktree.RemoveOptions{
			StackRoot: "nonexistent-stack",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no worktree found")
	})

	t.Run("removes worktree registration", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)
		s.WithInitialCommit()

		// Register a fake worktree (path doesn't exist)
		err := s.Engine.RegisterWorktree("feature-stack", "/tmp/nonexistent-worktree")
		require.NoError(t, err)

		// Verify it's registered
		wt, err := s.Engine.GetWorktreeForStack("feature-stack")
		require.NoError(t, err)
		require.NotNil(t, wt)

		// Remove it
		err = worktree.RemoveAction(s.Context, worktree.RemoveOptions{
			StackRoot: "feature-stack",
		})
		require.NoError(t, err)

		// Verify it's gone
		wt, err = s.Engine.GetWorktreeForStack("feature-stack")
		require.NoError(t, err)
		assert.Nil(t, wt)
	})
}

func TestOpenAction(t *testing.T) {
	t.Run("fails when worktree not found", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)
		s.WithInitialCommit()

		_, err := worktree.OpenAction(s.Context, worktree.OpenOptions{
			StackRoot: "nonexistent-stack",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no worktree found")
	})

	t.Run("fails when path does not exist", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)
		s.WithInitialCommit()

		// Register a fake worktree with non-existent path
		err := s.Engine.RegisterWorktree("feature-stack", "/tmp/nonexistent-path-12345")
		require.NoError(t, err)

		_, err = worktree.OpenAction(s.Context, worktree.OpenOptions{
			StackRoot: "feature-stack",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "does not exist")

		// Clean up
		_ = s.Engine.UnregisterWorktree("feature-stack")
	})

	t.Run("returns path when worktree exists", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)
		s.WithInitialCommit()

		// Use a path that exists (the repo root)
		repoRoot := s.Context.RepoRoot
		err := s.Engine.RegisterWorktree("feature-stack", repoRoot)
		require.NoError(t, err)

		path, err := worktree.OpenAction(s.Context, worktree.OpenOptions{
			StackRoot: "feature-stack",
		})
		require.NoError(t, err)
		assert.Equal(t, repoRoot, path)

		// Clean up
		_ = s.Engine.UnregisterWorktree("feature-stack")
	})
}
