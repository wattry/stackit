package worktree

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/output"
	"stackit.dev/stackit/testhelpers"
	"stackit.dev/stackit/testhelpers/scenario"
)

func TestExecutor_CreateSession(t *testing.T) {
	t.Run("creates worktree at trunk", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)
		s.WithInitialCommit()

		executor := NewExecutor(s.Engine, output.NewNullOutput())

		session, err := executor.CreateSession(context.Background(), CreateSessionOptions{
			NamePattern: "test-worktree-*",
		})
		require.NoError(t, err)
		defer session.Close()

		// Verify worktree exists
		assert.DirExists(t, session.Path)

		// Verify engine is usable
		trunk := session.Engine.Trunk()
		assert.Equal(t, "main", trunk.GetName())
	})

	t.Run("creates worktree at specific ref", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)
		s.WithInitialCommit()

		// Create a branch
		s.Scene.Repo.CreateAndCheckoutBranch("feature")
		s.Scene.Repo.CreateChangeAndCommit("feature change", "feature")
		s.Scene.Repo.CheckoutBranch("main")

		executor := NewExecutor(s.Engine, output.NewNullOutput())

		session, err := executor.CreateSession(context.Background(), CreateSessionOptions{
			Ref:         "feature",
			NamePattern: "test-worktree-*",
		})
		require.NoError(t, err)
		defer session.Close()

		// Verify worktree is at feature branch (file is named prefix_test.txt)
		assert.DirExists(t, session.Path)
		featureFile := filepath.Join(session.Path, "feature_test.txt")
		assert.FileExists(t, featureFile)
	})

	t.Run("cleanup removes worktree", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)
		s.WithInitialCommit()

		executor := NewExecutor(s.Engine, output.NewNullOutput())

		session, err := executor.CreateSession(context.Background(), CreateSessionOptions{
			NamePattern: "test-worktree-*",
		})
		require.NoError(t, err)

		worktreePath := session.Path
		assert.DirExists(t, worktreePath)

		session.Close()

		// Worktree should be removed
		_, err = os.Stat(worktreePath)
		assert.True(t, os.IsNotExist(err), "worktree should be removed after Close()")
	})
}

func TestSession_ResetToTrunk(t *testing.T) {
	s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)
	s.WithInitialCommit()

	executor := NewExecutor(s.Engine, output.NewNullOutput())

	session, err := executor.CreateSession(context.Background(), CreateSessionOptions{
		NamePattern: "test-worktree-*",
	})
	require.NoError(t, err)
	defer session.Close()

	// Get initial revision
	initialRev, err := session.GetCurrentRevision(context.Background())
	require.NoError(t, err)

	// Make a change in the worktree
	testFile := filepath.Join(session.Path, "test.txt")
	err = os.WriteFile(testFile, []byte("test"), 0644)
	require.NoError(t, err)

	// Stage and commit using the worktree engine's git
	git := session.Engine.Git()
	_, err = git.RunGitCommandWithContext(context.Background(), "add", ".")
	require.NoError(t, err)
	_, err = git.RunGitCommandWithContext(context.Background(), "commit", "-m", "test commit")
	require.NoError(t, err)

	// Verify we're at a different revision
	newRev, err := session.GetCurrentRevision(context.Background())
	require.NoError(t, err)
	assert.NotEqual(t, initialRev, newRev)

	// Reset to trunk
	err = session.ResetToTrunk(context.Background())
	require.NoError(t, err)

	// Verify we're back at initial revision
	afterResetRev, err := session.GetCurrentRevision(context.Background())
	require.NoError(t, err)
	assert.Equal(t, initialRev, afterResetRev)
}

func TestSession_ResetToRef(t *testing.T) {
	s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)
	s.WithInitialCommit()

	// Create a branch with a commit
	s.Scene.Repo.CreateAndCheckoutBranch("feature")
	s.Scene.Repo.CreateChangeAndCommit("feature change", "feature")

	// Get the feature SHA
	featureSHA, err := s.Scene.Repo.GetCurrentSHA()
	require.NoError(t, err)

	s.Scene.Repo.CheckoutBranch("main")

	// Rebuild engine to pick up the new branch
	eng, err := engine.NewEngine(engine.Options{
		RepoRoot: s.Scene.Dir,
		Trunk:    "main",
	})
	require.NoError(t, err)

	executor := NewExecutor(eng, output.NewNullOutput())

	session, err := executor.CreateSession(context.Background(), CreateSessionOptions{
		NamePattern: "test-worktree-*",
	})
	require.NoError(t, err)
	defer session.Close()

	// Reset to the feature branch
	err = session.ResetToRef(context.Background(), "feature")
	require.NoError(t, err)

	// Verify we're at the feature revision
	afterResetRev, err := session.GetCurrentRevision(context.Background())
	require.NoError(t, err)
	assert.Equal(t, featureSHA, afterResetRev)

	// Verify the feature file exists (file is named prefix_test.txt)
	featureFile := filepath.Join(session.Path, "feature_test.txt")
	assert.FileExists(t, featureFile)
}
