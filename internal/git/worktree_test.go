package git_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/testhelpers"
)

func TestWorktree(t *testing.T) {
	t.Run("add and remove worktree", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})
		runner := git.NewRunnerWithPath(scene.Repo.Dir)

		// Create a branch to checkout in the worktree
		err := scene.Repo.CreateBranch("test-branch")
		require.NoError(t, err)

		// Create a temporary directory for the worktree
		tmpDir := t.TempDir()

		// Normalize worktree path (on macOS /var is symlinked to /private/var)
		worktreePath, err := filepath.EvalSymlinks(tmpDir)
		require.NoError(t, err)
		worktreePath = filepath.Join(worktreePath, "worktree")

		// Add worktree
		err = runner.AddWorktree(context.Background(), worktreePath, "test-branch", false)
		require.NoError(t, err)

		// Verify worktree exists
		_, err = os.Stat(filepath.Join(worktreePath, ".git"))
		require.NoError(t, err)

		// List worktrees
		worktrees, err := runner.ListWorktrees(context.Background())
		require.NoError(t, err)
		require.Contains(t, worktrees, worktreePath)

		// Remove worktree
		err = runner.RemoveWorktree(context.Background(), worktreePath)
		require.NoError(t, err)

		// Verify worktree is gone from list
		worktrees, err = runner.ListWorktrees(context.Background())
		require.NoError(t, err)
		require.NotContains(t, worktrees, worktreePath)
	})

	t.Run("add detached worktree", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})
		runner := git.NewRunnerWithPath(scene.Repo.Dir)

		// Create a temporary directory for the worktree
		tmpDir := t.TempDir()

		worktreePath := filepath.Join(tmpDir, "worktree-detached")

		// Add detached worktree
		err := runner.AddWorktree(context.Background(), worktreePath, "", true)
		require.NoError(t, err)

		// Verify worktree exists
		_, err = os.Stat(filepath.Join(worktreePath, ".git"))
		require.NoError(t, err)

		// Clean up
		err = runner.RemoveWorktree(context.Background(), worktreePath)
		require.NoError(t, err)
	})
}
