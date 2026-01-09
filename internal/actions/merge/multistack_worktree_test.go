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

func TestMultiStackWorktreeExecutor_OctopusMergeCreatesSingleCommit(t *testing.T) {
	s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)

	// Create a stack with 3 branches: branch1 -> branch2 -> branch3
	s.CreateBranch("branch1").
		TrackBranch("branch1", s.Engine.Trunk().GetName())
	require.NoError(t, s.Scene.Repo.CreateChangeAndCommit("branch1-change", "file1.txt"))
	branch1SHA, err := s.Scene.Repo.GetRevision("branch1")
	require.NoError(t, err)
	s.Rebuild()

	s.CreateBranch("branch2").
		TrackBranch("branch2", "branch1")
	require.NoError(t, s.Scene.Repo.CreateChangeAndCommit("branch2-change", "file2.txt"))
	branch2SHA, err := s.Scene.Repo.GetRevision("branch2")
	require.NoError(t, err)
	s.Rebuild()

	s.CreateBranch("branch3").
		TrackBranch("branch3", "branch2")
	require.NoError(t, s.Scene.Repo.CreateChangeAndCommit("branch3-change", "file3.txt"))
	branch3SHA, err := s.Scene.Repo.GetRevision("branch3")
	require.NoError(t, err)
	s.Rebuild().
		Checkout(s.Engine.Trunk().GetName())

	// Execute octopus merge
	executor := NewMultiStackWorktreeExecutor(s.Engine, s.Context.Output)
	result, err := executor.ExecuteInWorktree(context.Background(), []MultiStackInfo{
		{RootBranch: "branch1", AllBranches: []string{"branch1", "branch2", "branch3"}},
	})
	require.NoError(t, err)
	defer result.Cleanup()

	require.Len(t, result.MergedStacks, 1)
	assert.Equal(t, "branch1", result.MergedStacks[0].RootBranch)

	// Verify we have a single merge commit with multiple parents (octopus merge)
	worktreeRepo := testhelpers.NewGitRepoFromExisting(t, result.WorktreePath)

	// Get the commit object to count parents
	parentOutput, err := worktreeRepo.RunGitCommandAndGetOutput("cat-file", "-p", "HEAD")
	require.NoError(t, err)

	// Count "parent" lines in the commit object
	parentCount := 0
	for _, line := range splitLines(parentOutput) {
		if len(line) > 6 && line[:6] == "parent" {
			parentCount++
		}
	}

	// Octopus merge should have 4 parents: trunk + branch1 + branch2 + branch3
	assert.Equal(t, 4, parentCount, "octopus merge should have 4 parents (trunk + 3 branches)")

	// Verify all original branch commits are ancestors of HEAD
	isAncestor := func(sha string) bool {
		err := worktreeRepo.RunGitCommand("merge-base", "--is-ancestor", sha, "HEAD")
		return err == nil
	}

	assert.True(t, isAncestor(branch1SHA), "branch1 commit should be ancestor of HEAD")
	assert.True(t, isAncestor(branch2SHA), "branch2 commit should be ancestor of HEAD")
	assert.True(t, isAncestor(branch3SHA), "branch3 commit should be ancestor of HEAD")

	// Verify all files are present in the working tree
	_, err = os.Stat(filepath.Join(result.WorktreePath, "file1.txt"))
	assert.NoError(t, err, "file1.txt should exist")
	_, err = os.Stat(filepath.Join(result.WorktreePath, "file2.txt"))
	assert.NoError(t, err, "file2.txt should exist")
	_, err = os.Stat(filepath.Join(result.WorktreePath, "file3.txt"))
	assert.NoError(t, err, "file3.txt should exist")
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
