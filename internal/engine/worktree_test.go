package engine_test

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/testhelpers"
	"stackit.dev/stackit/testhelpers/scenario"
)

func TestCreateTemporaryWorktree(t *testing.T) {
	s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)

	// Create a branch to use for the worktree
	s.CreateBranch("feature").
		Commit("feature change").
		Checkout("main")

	ctx := context.Background()
	worktreePath, cleanup, err := s.Engine.CreateTemporaryWorktree(ctx, "feature", "stackit-test-*")
	require.NoError(t, err)
	require.NotEmpty(t, worktreePath)
	require.NotNil(t, cleanup)

	// Verify worktree exists
	info, err := os.Stat(worktreePath)
	require.NoError(t, err)
	assert.True(t, info.IsDir())

	// Verify it's a git worktree by checking for .git file (not directory)
	gitFile, err := os.Stat(worktreePath + "/.git")
	require.NoError(t, err)
	assert.False(t, gitFile.IsDir())

	// Run cleanup
	cleanup()

	// Verify worktree and temp dir are gone
	_, err = os.Stat(worktreePath)
	assert.True(t, os.IsNotExist(err))

	// The parent directory of worktreePath (which is the temp dir) should also be gone
	// worktreePath is filepath.Join(tmpDir, "worktree")
	tmpDir := worktreePath[:len(worktreePath)-9] // "/worktree" is 9 chars
	_, err = os.Stat(tmpDir)
	assert.True(t, os.IsNotExist(err))
}
