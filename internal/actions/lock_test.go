package actions_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/testhelpers"
	"stackit.dev/stackit/testhelpers/scenario"
)

func TestLockUnlockAction(t *testing.T) {
	t.Run("LockAction locks branch and ancestors", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)
		s.WithInitialCommit().
			CreateBranch("feature-a").
			Commit("a1").
			CreateBranch("feature-b").
			Commit("b1").
			TrackBranch("feature-a", "main").
			TrackBranch("feature-b", "feature-a")

		err := actions.LockAction(s.Context, "feature-b")
		require.NoError(t, err)

		require.True(t, s.Engine.GetBranch("feature-b").IsLocked())
		require.True(t, s.Engine.GetBranch("feature-a").IsLocked())
		require.False(t, s.Engine.GetBranch("main").IsLocked())
	})

	t.Run("UnlockAction unlocks branch and descendants", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)
		s.WithInitialCommit().
			CreateBranch("feature-a").
			Commit("a1").
			CreateBranch("feature-b").
			Commit("b1").
			TrackBranch("feature-a", "main").
			TrackBranch("feature-b", "feature-a")

		// Pre-lock
		require.NoError(t, s.Engine.SetLocked(s.Engine.GetBranch("feature-a"), true))
		require.NoError(t, s.Engine.SetLocked(s.Engine.GetBranch("feature-b"), true))

		err := actions.UnlockAction(s.Context, "feature-a")
		require.NoError(t, err)

		require.False(t, s.Engine.GetBranch("feature-a").IsLocked())
		require.False(t, s.Engine.GetBranch("feature-b").IsLocked())
	})

	t.Run("LockAction fails on untracked branch", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)
		s.WithInitialCommit().
			CreateBranchQuiet("untracked")

		err := actions.LockAction(s.Context, "untracked")
		require.Error(t, err)
		require.Contains(t, err.Error(), "not tracked")
	})

	t.Run("LockAction fails on trunk", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)
		s.WithInitialCommit()

		err := actions.LockAction(s.Context, "main")
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot lock trunk")
	})
}
