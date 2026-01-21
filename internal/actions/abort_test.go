package actions_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/internal/actions/abort"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/testhelpers"
	"stackit.dev/stackit/testhelpers/scenario"
)

func TestAbortAction(t *testing.T) {
	t.Run("reports when no operation is in progress", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)
		s.WithInitialCommit()

		err := abort.Action(s.Context, abort.Options{}, nil)
		require.NoError(t, err)
	})

	t.Run("aborts when continuation state exists", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)
		s.WithInitialCommit().
			CreateBranch("feature").
			Commit("feature change").
			Checkout("main").
			TrackBranch("feature", "main")

		// Get initial state
		initialSHA, err := s.Engine.GetBranch("feature").GetRevision()
		require.NoError(t, err)

		// Take snapshot
		err = s.Engine.TakeSnapshot(engine.SnapshotOptions{
			Command: "restack",
			Args:    []string{"feature"},
		})
		require.NoError(t, err)

		// Manually create continuation state
		continuationPath := filepath.Join(s.Scene.Dir, ".git", ".stackit_continue")
		err = os.WriteFile(continuationPath, []byte("{}"), 0644)
		require.NoError(t, err)

		// Run abort
		err = abort.Action(s.Context, abort.Options{Force: true}, nil)
		require.NoError(t, err)

		// Verify continuation state is gone
		_, err = os.Stat(continuationPath)
		require.True(t, os.IsNotExist(err), "continuation state should be gone")

		// Verify state restored
		s.Engine.Rebuild(s.Engine.Trunk().GetName())
		restoredSHA, err := s.Engine.GetBranch("feature").GetRevision()
		require.NoError(t, err)
		require.Equal(t, initialSHA, restoredSHA)
	})
}
