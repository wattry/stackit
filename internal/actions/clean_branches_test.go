package actions_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/testhelpers"
	"stackit.dev/stackit/testhelpers/scenario"
)

func TestCleanBranches(t *testing.T) {
	t.Run("deletes merged branch and updates children", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			WithStack(map[string]string{
				"branch1": "main",
				"branch2": "branch1",
			})

		// Merge branch1 into main
		s.Checkout("main").
			RunGit("merge", "branch1")

		// Rebuild to see changes
		err := s.Engine.Rebuild("main")
		require.NoError(t, err)

		// Mark branch1 as merged via PR info
		prInfo := testhelpers.NewTestPrInfoMerged(1, "main")
		branch := s.Engine.GetBranch("branch1")
		err = s.Engine.UpsertPrInfo(context.Background(), branch, prInfo)
		require.NoError(t, err)

		result, err := actions.CleanBranches(s.Context, actions.CleanBranchesOptions{
			Force: true,
		})
		require.NoError(t, err)

		// branch1 should be deleted
		require.False(t, s.Engine.GetBranch("branch1").IsTracked())

		// branch2 should have new parent (main)
		branchparent2 := s.Engine.GetBranch("branch2")
		parent2 := branchparent2.GetParent()
		require.NotNil(t, parent2)
		require.Equal(t, "main", parent2.GetName())
		require.Contains(t, result.BranchesWithNewParents, "branch2")
	})

	t.Run("handles multiple children when parent is deleted", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			WithStack(map[string]string{
				"branch1": "main",
				"branch2": "branch1",
				"branch3": "branch1",
			})

		// Merge branch1
		s.Checkout("main").
			RunGit("merge", "branch1")

		// Rebuild to see changes
		err := s.Engine.Rebuild("main")
		require.NoError(t, err)

		// Mark branch1 as merged
		prInfo := testhelpers.NewTestPrInfoWithState(1, "MERGED")
		branch := s.Engine.GetBranch("branch1")
		err = s.Engine.UpsertPrInfo(context.Background(), branch, prInfo)
		require.NoError(t, err)

		result, err := actions.CleanBranches(s.Context, actions.CleanBranchesOptions{
			Force: true,
		})
		require.NoError(t, err)

		// Both children should have new parent
		branchparent2 := s.Engine.GetBranch("branch2")
		parent2 := branchparent2.GetParent()
		require.NotNil(t, parent2)
		require.Equal(t, "main", parent2.GetName())
		branchparent3 := s.Engine.GetBranch("branch3")
		parent3 := branchparent3.GetParent()
		require.NotNil(t, parent3)
		require.Equal(t, "main", parent3.GetName())
		require.Contains(t, result.BranchesWithNewParents, "branch2")
		require.Contains(t, result.BranchesWithNewParents, "branch3")
	})

	t.Run("does not delete branch without PR when not merged", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			WithStack(map[string]string{
				"branch1": "main",
			})

		result, err := actions.CleanBranches(s.Context, actions.CleanBranchesOptions{
			Force: false,
		})
		require.NoError(t, err)

		// Branch should still exist
		require.True(t, s.Engine.GetBranch("branch1").IsTracked())
		require.Empty(t, result.BranchesWithNewParents)
	})

	t.Run("deletes locked branch when merged", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			WithStack(map[string]string{
				"branch1": "main",
			})

		// Merge branch1 into main
		s.Checkout("main").
			RunGit("merge", "branch1")

		// Rebuild to see changes
		err := s.Engine.Rebuild("main")
		require.NoError(t, err)

		// Lock the branch (simulating consolidation)
		branch := s.Engine.GetBranch("branch1")
		_, err = s.Engine.SetLocked(context.Background(), []engine.Branch{branch}, engine.LockReasonConsolidating)
		require.NoError(t, err)
		require.True(t, branch.IsLocked(), "branch should be locked")

		// Mark branch1 as merged via PR info
		prInfo := testhelpers.NewTestPrInfoMerged(1, "main")
		err = s.Engine.UpsertPrInfo(context.Background(), branch, prInfo)
		require.NoError(t, err)

		// Clean should delete the locked branch
		_, err = actions.CleanBranches(s.Context, actions.CleanBranchesOptions{
			Force: true,
		})
		require.NoError(t, err)

		// branch1 should be deleted despite being locked
		require.False(t, s.Engine.GetBranch("branch1").IsTracked())
	})

	t.Run("deletes merged child even if parent is NOT merged", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			WithStack(map[string]string{
				"branch1": "main",
				"branch2": "branch1",
			})

		// branch1: NOT merged
		// branch2: IS merged
		prInfo := testhelpers.NewTestPrInfoMerged(2, "branch1")
		branch2 := s.Engine.GetBranch("branch2")
		err := s.Engine.UpsertPrInfo(context.Background(), branch2, prInfo)
		require.NoError(t, err)

		_, err = actions.CleanBranches(s.Context, actions.CleanBranchesOptions{
			Force: true,
		})
		require.NoError(t, err)

		// branch2 should be deleted even though we didn't "visit" it via a deleted branch1
		require.False(t, s.Engine.GetBranch("branch2").IsTracked())
		require.True(t, s.Engine.GetBranch("branch1").IsTracked())
	})
}
