package sync

import (
	"testing"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/testhelpers"
	"stackit.dev/stackit/testhelpers/scenario"
)

func TestSyncAction(t *testing.T) {
	t.Run("syncs when trunk is up to date", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)

		err := Action(s.Context, Options{
			All:     false,
			Force:   false,
			Restack: false,
		}, nil)
		require.NoError(t, err)
	})

	t.Run("fails when there are uncommitted changes", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			WithUncommittedChange("unstaged")

		err := Action(s.Context, Options{
			All:     false,
			Force:   false,
			Restack: false,
		}, nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "uncommitted changes")
	})

	t.Run("syncs with restack flag", func(t *testing.T) {
		s := scenario.NewScenario(t, nil).
			WithStack(map[string]string{
				"branch1": "main",
			})

		err := Action(s.Context, Options{
			All:     false,
			Force:   false,
			Restack: true,
		}, nil)
		// Should succeed (even if no restacking needed)
		require.NoError(t, err)
	})

	t.Run("restacks branches in topological order (parents before children)", func(t *testing.T) {
		s := scenario.NewScenario(t, nil).
			WithStack(map[string]string{
				"branch1": "main",
				"branch2": "branch1",
				"branch3": "branch2",
			})

		err := Action(s.Context, Options{
			All:     false,
			Force:   false,
			Restack: true,
		}, nil)
		// Should succeed - branches should be restacked in correct order
		require.NoError(t, err)
	})

	t.Run("restacks branching stacks in topological order", func(t *testing.T) {
		s := scenario.NewScenario(t, nil).
			WithStack(map[string]string{
				"stackA":        "main",
				"stackA-child1": "stackA",
				"stackA-child2": "stackA",
				"stackB":        "main",
				"stackB-child1": "stackB",
			})

		err := Action(s.Context, Options{
			All:     false,
			Force:   false,
			Restack: true,
		}, nil)
		// Should succeed - branches should be restacked with parents before children
		require.NoError(t, err)

		// Verify all branches still exist and are properly tracked
		s.ExpectStackStructure(map[string]string{
			"stackA":        "main",
			"stackA-child1": "stackA",
			"stackA-child2": "stackA",
			"stackB":        "main",
			"stackB-child1": "stackB",
		})
	})

	t.Run("restacks multiple deep subtrees correctly", func(t *testing.T) {
		s := scenario.NewScenario(t, nil).
			WithStack(map[string]string{
				"P":   "main",
				"C1":  "P",
				"GC1": "C1",
				"C2":  "P",
				"GC2": "C2",
			})

		// Modify P to trigger restacking of all descendants
		s.Checkout("P").
			Commit("P updated")

		// Refresh engine
		err := s.Engine.Rebuild("main")
		require.NoError(t, err)

		err = Action(s.Context, Options{
			All:     true,
			Restack: true,
		}, nil)
		require.NoError(t, err)

		// Verify all branches are fixed
		s.ExpectBranchFixed("C1").
			ExpectBranchFixed("GC1").
			ExpectBranchFixed("C2").
			ExpectBranchFixed("GC2")
	})

	t.Run("partial success in branching restack (one child succeeds, one fails)", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)

		// Create: main -> P -> [0-Success, 1-Failure]
		// We use these names because siblings are sorted ascending by name by default,
		// and we want the successful one to be processed first.
		s.CreateBranch("P").
			Commit("P change").
			TrackBranch("P", "main")

		// 0-Success will restack successfully
		s.Checkout("P").
			CreateBranch("0-Success").
			Commit("Success change").
			TrackBranch("0-Success", "P")

		// 1-Failure will have a conflict
		s.Checkout("P").
			CreateBranch("1-Failure")
		err := s.Scene.Repo.CreateChangeAndCommit("initial content", "conflict")
		require.NoError(t, err)
		s.TrackBranch("1-Failure", "P")

		// Modify P with a change that conflicts with 1-Failure but not 0-Success
		s.Checkout("P")
		err = s.Scene.Repo.CreateChangeAndCommit("conflicting content", "conflict")
		require.NoError(t, err)

		// Refresh engine
		err = s.Engine.Rebuild("main")
		require.NoError(t, err)

		s.Checkout("P")

		err = Action(s.Context, Options{
			All:     true,
			Restack: true,
		}, nil)

		// Should error due to conflict in 1-Failure
		require.Error(t, err)
		require.Contains(t, err.Error(), "conflict")

		// 0-Success should still have been restacked successfully
		s.ExpectBranchFixed("0-Success")
		// 1-Failure should NOT be fixed
		s.ExpectBranchNotFixed("1-Failure")
	})
}
