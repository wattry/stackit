package merge

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	mergeAction "stackit.dev/stackit/internal/actions/merge"
	"stackit.dev/stackit/testhelpers"
	"stackit.dev/stackit/testhelpers/scenario"
)

func TestNewMergeCmd(t *testing.T) {
	t.Run("creates command with subcommands", func(t *testing.T) {
		cmd := NewMergeCmd(nil)

		require.Equal(t, "merge", cmd.Use)
		require.NotEmpty(t, cmd.Short)
		require.NotEmpty(t, cmd.Long)

		// Check subcommands exist
		nextCmd, _, err := cmd.Find([]string{"next"})
		require.NoError(t, err)
		require.NotNil(t, nextCmd)
		require.Equal(t, "next", nextCmd.Use)

		shipCmd, _, err := cmd.Find([]string{"ship"})
		require.NoError(t, err)
		require.NotNil(t, shipCmd)
		require.Equal(t, "ship", shipCmd.Use)
	})

	t.Run("has expected flags", func(t *testing.T) {
		cmd := NewMergeCmd(nil)

		// Check root command flags
		dryRunFlag := cmd.Flags().Lookup("dry-run")
		require.NotNil(t, dryRunFlag)

		forceFlag := cmd.Flags().Lookup("force")
		require.NotNil(t, forceFlag)

		waitFlag := cmd.Flags().Lookup("wait")
		require.NotNil(t, waitFlag)
	})
}

func TestNewNextCmd(t *testing.T) {
	t.Run("has expected flags", func(t *testing.T) {
		cmd := NewNextCmd(nil)

		require.Equal(t, "next", cmd.Use)

		dryRunFlag := cmd.Flags().Lookup("dry-run")
		require.NotNil(t, dryRunFlag)

		yesFlag := cmd.Flags().Lookup("yes")
		require.NotNil(t, yesFlag)

		forceFlag := cmd.Flags().Lookup("force")
		require.NotNil(t, forceFlag)

		waitFlag := cmd.Flags().Lookup("wait")
		require.NotNil(t, waitFlag)

		branchFlag := cmd.Flags().Lookup("branch")
		require.NotNil(t, branchFlag)

		scopeFlag := cmd.Flags().Lookup("scope")
		require.NotNil(t, scopeFlag)
	})
}

func TestNewShipCmd(t *testing.T) {
	t.Run("has expected flags", func(t *testing.T) {
		cmd := NewShipCmd(nil)

		require.Equal(t, "ship", cmd.Use)

		dryRunFlag := cmd.Flags().Lookup("dry-run")
		require.NotNil(t, dryRunFlag)

		yesFlag := cmd.Flags().Lookup("yes")
		require.NotNil(t, yesFlag)

		forceFlag := cmd.Flags().Lookup("force")
		require.NotNil(t, forceFlag)

		waitFlag := cmd.Flags().Lookup("wait")
		require.NotNil(t, waitFlag)

		scopeFlag := cmd.Flags().Lookup("scope")
		require.NotNil(t, scopeFlag)

		branchFlag := cmd.Flags().Lookup("branch")
		require.NotNil(t, branchFlag)

		stacksFlag := cmd.Flags().Lookup("stacks")
		require.NotNil(t, stacksFlag)

		skipLocalCIFlag := cmd.Flags().Lookup("skip-local-ci")
		require.NotNil(t, skipLocalCIFlag)
	})
}

func TestNewDrainCmd(t *testing.T) {
	t.Run("has expected flags", func(t *testing.T) {
		cmd := NewDrainCmd(nil)

		require.Equal(t, "drain", cmd.Use)

		dryRunFlag := cmd.Flags().Lookup("dry-run")
		require.NotNil(t, dryRunFlag)

		yesFlag := cmd.Flags().Lookup("yes")
		require.NotNil(t, yesFlag)

		forceFlag := cmd.Flags().Lookup("force")
		require.NotNil(t, forceFlag)

		methodFlag := cmd.Flags().Lookup("method")
		require.NotNil(t, methodFlag)

		branchFlag := cmd.Flags().Lookup("branch")
		require.NotNil(t, branchFlag)

		scopeFlag := cmd.Flags().Lookup("scope")
		require.NotNil(t, scopeFlag)

		// drain should NOT have a --wait flag (it always waits)
		waitFlag := cmd.Flags().Lookup("wait")
		require.Nil(t, waitFlag)
	})
}

func TestMergeNextUsesCreateMergePlan(t *testing.T) {
	t.Parallel()

	t.Run("CreateMergePlan returns error when not on a branch", func(t *testing.T) {
		t.Parallel()
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)

		// Detach HEAD
		s.RunGit("checkout", "HEAD~0").Rebuild()

		_, _, err := mergeAction.CreateMergePlan(s.Context.Context, s.Engine, s.Context.Output, nil, mergeAction.CreatePlanOptions{
			Strategy: mergeAction.StrategyBottomUp,
		})
		require.Error(t, err)
	})

	t.Run("CreateMergePlan returns error when on trunk", func(t *testing.T) {
		t.Parallel()
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)

		s.Checkout("main")

		_, _, err := mergeAction.CreateMergePlan(s.Context.Context, s.Engine, s.Context.Output, nil, mergeAction.CreatePlanOptions{
			Strategy: mergeAction.StrategyBottomUp,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot merge from trunk")
	})

	t.Run("CreateMergePlan returns error when branch is not tracked", func(t *testing.T) {
		t.Parallel()
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			CreateBranch("untracked")

		_, _, err := mergeAction.CreateMergePlan(s.Context.Context, s.Engine, s.Context.Output, nil, mergeAction.CreatePlanOptions{
			Strategy: mergeAction.StrategyBottomUp,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "not tracked")
	})

	t.Run("CreateMergePlan returns error when no PRs found", func(t *testing.T) {
		t.Parallel()
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			WithStack(map[string]string{
				"branch-a": "main",
			})

		s.Checkout("branch-a")

		_, _, err := mergeAction.CreateMergePlan(s.Context.Context, s.Engine, s.Context.Output, nil, mergeAction.CreatePlanOptions{
			Strategy: mergeAction.StrategyBottomUp,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "no open PRs found")
	})

	t.Run("finds bottom PR in stack", func(t *testing.T) {
		t.Parallel()
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			WithStack(map[string]string{
				"branch-a": "main",
				"branch-b": "branch-a",
				"branch-c": "branch-b",
			})

		// Add PR info to branches
		branchA := s.Engine.GetBranch("branch-a")
		branchB := s.Engine.GetBranch("branch-b")
		branchC := s.Engine.GetBranch("branch-c")

		err := s.Engine.UpsertPrInfo(context.Background(), branchA, testhelpers.NewTestPrInfo(101).
			WithURL("https://github.com/owner/repo/pull/101"))
		require.NoError(t, err)

		err = s.Engine.UpsertPrInfo(context.Background(), branchB, testhelpers.NewTestPrInfo(102).
			WithURL("https://github.com/owner/repo/pull/102"))
		require.NoError(t, err)

		err = s.Engine.UpsertPrInfo(context.Background(), branchC, testhelpers.NewTestPrInfo(103).
			WithURL("https://github.com/owner/repo/pull/103"))
		require.NoError(t, err)

		// Switch to branch-c (top of stack)
		s.Checkout("branch-c")

		plan, _, err := mergeAction.CreateMergePlan(s.Context.Context, s.Engine, s.Context.Output, nil, mergeAction.CreatePlanOptions{
			Strategy: mergeAction.StrategyBottomUp,
			Force:    true,
		})
		require.NoError(t, err)
		require.NotNil(t, plan)

		// Should find branch-a as the bottom PR
		require.NotEmpty(t, plan.BranchesToMerge)
		require.Equal(t, "branch-a", plan.BranchesToMerge[0].BranchName)
		require.Equal(t, 101, plan.BranchesToMerge[0].PRNumber)
	})
}
