package lock_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/internal/actions/lock"
	"stackit.dev/stackit/internal/tui"
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

		err := lock.LockAction(s.Context, "feature-b")
		require.NoError(t, err)

		require.True(t, s.Engine.GetBranch("feature-b").IsLocked())
		require.True(t, s.Engine.GetBranch("feature-a").IsLocked())
		require.False(t, s.Engine.GetBranch("main").IsLocked())
	})

	t.Run("LockAction indicates already locked branches", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)
		s.WithInitialCommit().
			CreateBranch("feature-a").
			Commit("a1").
			TrackBranch("feature-a", "main")

		// Pre-lock
		require.NoError(t, s.Engine.SetLocked(s.Engine.GetBranch("feature-a"), true))

		var buf bytes.Buffer
		s.Context.Splog = tui.NewSplogToWriter(&buf)

		err := lock.LockAction(s.Context, "feature-a")
		require.NoError(t, err)

		output := buf.String()
		require.Contains(t, output, "feature-a")
		require.Contains(t, output, "already locked")
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

		err := lock.UnlockAction(s.Context, "feature-a")
		require.NoError(t, err)

		require.False(t, s.Engine.GetBranch("feature-a").IsLocked())
		require.False(t, s.Engine.GetBranch("feature-b").IsLocked())
	})

	t.Run("UnlockAction indicates already unlocked branches", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)
		s.WithInitialCommit().
			CreateBranch("feature-a").
			Commit("a1").
			TrackBranch("feature-a", "main")

		// Already unlocked by default

		var buf bytes.Buffer
		s.Context.Splog = tui.NewSplogToWriter(&buf)

		err := lock.UnlockAction(s.Context, "feature-a")
		require.NoError(t, err)

		output := buf.String()
		require.Contains(t, output, "feature-a")
		require.Contains(t, output, "already unlocked")
	})

	t.Run("LockAction fails on untracked branch", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)
		s.WithInitialCommit().
			CreateBranchQuiet("untracked")

		err := lock.LockAction(s.Context, "untracked")
		require.Error(t, err)
		require.Contains(t, err.Error(), "not tracked")
	})

	t.Run("LockAction fails on trunk", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)
		s.WithInitialCommit()

		err := lock.LockAction(s.Context, "main")
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot lock trunk")
	})

	t.Run("LockAction with unpushed commits in non-interactive mode", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)
		s.WithInitialCommit().
			CreateBranch("feature-a").
			Commit("a1").
			TrackBranch("feature-a", "main")

		// Create a remote
		_, err := s.Scene.Repo.CreateBareRemote("origin")
		require.NoError(t, err)

		// Push main but not feature-a
		err = s.Scene.Repo.PushBranch("origin", "main")
		require.NoError(t, err)

		// feature-a is now "ahead" of remote (remote doesn't have it)
		// Or if we pushed an earlier version, it would be ahead.
		// Let's push an earlier version.
		err = s.Scene.Repo.PushBranch("origin", "feature-a")
		require.NoError(t, err)
		s.Commit("a2") // feature-a is now ahead

		s.Context.Interactive = false // Ensure non-interactive

		err = lock.LockAction(s.Context, "feature-a")
		require.NoError(t, err)
		require.True(t, s.Engine.GetBranch("feature-a").IsLocked())
	})

	t.Run("UnlockAction prompts for downstack and unlocks it if confirmed", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)
		s.WithInitialCommit().
			CreateBranch("feature-a").
			Commit("a1").
			CreateBranch("feature-b").
			Commit("b1").
			TrackBranch("feature-a", "main").
			TrackBranch("feature-b", "feature-a")

		// Pre-lock both
		require.NoError(t, s.Engine.SetLocked(s.Engine.GetBranch("feature-a"), true))
		require.NoError(t, s.Engine.SetLocked(s.Engine.GetBranch("feature-b"), true))

		// Mock PromptConfirm to return true
		oldPrompt := tui.PromptConfirm
		defer func() { tui.PromptConfirm = oldPrompt }()
		tui.PromptConfirm = func(prompt string, defaultValue bool) (bool, error) {
			require.Contains(t, prompt, "unlock the downstack branch feature-a")
			return true, nil
		}

		s.Context.Interactive = true
		err := lock.UnlockAction(s.Context, "feature-b")
		require.NoError(t, err)

		require.False(t, s.Engine.GetBranch("feature-b").IsLocked())
		require.False(t, s.Engine.GetBranch("feature-a").IsLocked())
	})

	t.Run("UnlockAction prompts for downstack and does not unlock it if declined", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)
		s.WithInitialCommit().
			CreateBranch("feature-a").
			Commit("a1").
			CreateBranch("feature-b").
			Commit("b1").
			TrackBranch("feature-a", "main").
			TrackBranch("feature-b", "feature-a")

		// Pre-lock both
		require.NoError(t, s.Engine.SetLocked(s.Engine.GetBranch("feature-a"), true))
		require.NoError(t, s.Engine.SetLocked(s.Engine.GetBranch("feature-b"), true))

		// Mock PromptConfirm to return false
		oldPrompt := tui.PromptConfirm
		defer func() { tui.PromptConfirm = oldPrompt }()
		tui.PromptConfirm = func(prompt string, defaultValue bool) (bool, error) {
			require.Contains(t, prompt, "unlock the downstack branch feature-a")
			return false, nil
		}

		s.Context.Interactive = true
		err := lock.UnlockAction(s.Context, "feature-b")
		require.NoError(t, err)

		require.False(t, s.Engine.GetBranch("feature-b").IsLocked())
		require.True(t, s.Engine.GetBranch("feature-a").IsLocked())
	})
}
