package actions

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/handlers"
	"stackit.dev/stackit/testhelpers"
	"stackit.dev/stackit/testhelpers/scenario"
)

func TestRestackAction(t *testing.T) {
	t.Run("parallel multi-stack restack returns worker errors", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			WithStack(map[string]string{
				"alpha-root":  "main",
				"alpha-child": "alpha-root",
				"beta-root":   "main",
			})

		originalNewWorktreeEngine := newWorktreeEngine
		newWorktreeEngine = func(engine.WorktreeEngineOptions) (engine.Engine, error) {
			return nil, errors.New("boom")
		}
		t.Cleanup(func() {
			newWorktreeEngine = originalNewWorktreeEngine
		})

		err := RestackAction(s.Context, RestackOptions{
			AllStacks: true,
			Parallel:  true,
			Jobs:      2,
		}, handlers.NewJSONRestackHandler())
		require.Error(t, err)
		require.ErrorContains(t, err, "restack failed")
		require.ErrorContains(t, err, "alpha-root")
		require.ErrorContains(t, err, "beta-root")
		require.ErrorContains(t, err, "create worktree engine")
	})
}
