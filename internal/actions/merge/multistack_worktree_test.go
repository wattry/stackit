package merge

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/testhelpers"
	"stackit.dev/stackit/testhelpers/scenario"
)

func TestMultiStackWorktreeExecutor_ConflictingStackResetsState(t *testing.T) {
	s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)

	// stack1 modifies test.txt to "stack1"
	s.CreateBranch("stack1").
		TrackBranch("stack1", s.Engine.Trunk().GetName())
	require.NoError(t, s.Scene.Repo.CreateChangeAndCommit("stack1", ""))
	s.Rebuild().
		Checkout(s.Engine.Trunk().GetName())

	// stack2 modifies the same file differently to force a conflict when merged after stack1
	s.CreateBranch("stack2").
		TrackBranch("stack2", s.Engine.Trunk().GetName())
	require.NoError(t, s.Scene.Repo.CreateChangeAndCommit("stack2", ""))
	s.Rebuild().
		Checkout(s.Engine.Trunk().GetName())

	executor := NewMultiStackWorktreeExecutor(s.Engine, s.Context.Output)
	result, err := executor.ExecuteInWorktree(context.Background(), []MultiStackInfo{
		{RootBranch: "stack1", AllBranches: []string{"stack1"}},
		{RootBranch: "stack2", AllBranches: []string{"stack2"}},
	})
	require.NoError(t, err)
	defer result.Cleanup()

	require.Len(t, result.MergedStacks, 1)
	assert.Equal(t, "stack1", result.MergedStacks[0].RootBranch)

	require.Len(t, result.ConflictStacks, 1)
	assert.Equal(t, "stack2", result.ConflictStacks[0].Stack.RootBranch)

	// Verify the worktree does not contain partial changes from the conflicting stack
	content, readErr := os.ReadFile(filepath.Join(result.WorktreePath, "test.txt"))
	require.NoError(t, readErr)
	assert.Equal(t, "stack1", string(content))
}

func TestMultiStackWorktreeExecutor_PullsTrunkBeforeMerge(t *testing.T) {
	s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)

	// Set up remote and push initial trunk
	remotePath, err := s.Scene.Repo.CreateBareRemote("origin")
	require.NoError(t, err)
	require.NoError(t, s.Scene.Repo.PushBranch("origin", s.Engine.Trunk().GetName()))

	// Create a new commit on trunk and push it to the remote, then rewind local to make it stale
	require.NoError(t, s.Scene.Repo.CreateChangeAndCommit("remote-change", "remote"))
	require.NoError(t, s.Scene.Repo.PushBranch("origin", s.Engine.Trunk().GetName()))

	latestRemoteSHA, err := s.Scene.Repo.GetRevision("origin/" + s.Engine.Trunk().GetName())
	require.NoError(t, err)

	// Reset local trunk to previous commit to simulate stale trunk
	prevSHA, err := s.Scene.Repo.GetRevision(s.Engine.Trunk().GetName() + "~1")
	require.NoError(t, err)
	require.NoError(t, s.Scene.Repo.RunGitCommand("reset", "--hard", prevSHA))
	s.Rebuild()

	executor := NewMultiStackWorktreeExecutor(s.Engine, s.Context.Output)
	result, err := executor.ExecuteInWorktree(context.Background(), nil)
	require.NoError(t, err)
	defer result.Cleanup()

	worktreeRepo := testhelpers.NewGitRepoFromExisting(t, result.WorktreePath)
	worktreeHead, err := worktreeRepo.GetRevision("HEAD")
	require.NoError(t, err)

	assert.Equal(t, latestRemoteSHA, worktreeHead)
	assert.Contains(t, remotePath, "origin")
}
