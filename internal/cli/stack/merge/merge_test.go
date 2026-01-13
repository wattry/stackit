package merge

import (
	"testing"

	"github.com/stretchr/testify/require"

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

		squashCmd, _, err := cmd.Find([]string{"squash"})
		require.NoError(t, err)
		require.NotNil(t, squashCmd)
		require.Equal(t, "squash", squashCmd.Use)
	})

	t.Run("has expected flags", func(t *testing.T) {
		cmd := NewMergeCmd(nil)

		// Check root command flags
		dryRunFlag := cmd.Flags().Lookup("dry-run")
		require.NotNil(t, dryRunFlag)

		forceFlag := cmd.Flags().Lookup("force")
		require.NotNil(t, forceFlag)

		noWaitFlag := cmd.Flags().Lookup("no-wait")
		require.NotNil(t, noWaitFlag)
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

		noWaitFlag := cmd.Flags().Lookup("no-wait")
		require.NotNil(t, noWaitFlag)
	})
}

func TestNewSquashCmd(t *testing.T) {
	t.Run("has expected flags", func(t *testing.T) {
		cmd := NewSquashCmd(nil)

		require.Equal(t, "squash", cmd.Use)

		dryRunFlag := cmd.Flags().Lookup("dry-run")
		require.NotNil(t, dryRunFlag)

		yesFlag := cmd.Flags().Lookup("yes")
		require.NotNil(t, yesFlag)

		forceFlag := cmd.Flags().Lookup("force")
		require.NotNil(t, forceFlag)

		noWaitFlag := cmd.Flags().Lookup("no-wait")
		require.NotNil(t, noWaitFlag)

		scopeFlag := cmd.Flags().Lookup("scope")
		require.NotNil(t, scopeFlag)

		stacksFlag := cmd.Flags().Lookup("stacks")
		require.NotNil(t, stacksFlag)

		skipLocalCIFlag := cmd.Flags().Lookup("skip-local-ci")
		require.NotNil(t, skipLocalCIFlag)
	})
}

func TestFindBottomUnmergedPR(t *testing.T) {
	t.Run("returns error when not on a branch", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)

		// Detach HEAD
		s.RunGit("checkout", "HEAD~0")

		// Rebuild engine after git operation
		s.Rebuild()

		_, _, err := findBottomUnmergedPR(s.Context)
		require.Error(t, err)
		require.Contains(t, err.Error(), "not on a branch")
	})

	t.Run("returns error when on trunk", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)

		s.Checkout("main")

		_, _, err := findBottomUnmergedPR(s.Context)
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot merge from trunk")
	})

	t.Run("returns error when branch is not tracked", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			CreateBranch("untracked")

		_, _, err := findBottomUnmergedPR(s.Context)
		require.Error(t, err)
		require.Contains(t, err.Error(), "not tracked")
	})

	t.Run("returns nil when no PRs found", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			WithStack(map[string]string{
				"branch-a": "main",
			})

		s.Checkout("branch-a")

		prInfo, upstack, err := findBottomUnmergedPR(s.Context)
		require.NoError(t, err)
		require.Nil(t, prInfo)
		require.Nil(t, upstack)
	})

	t.Run("finds bottom PR in stack", func(t *testing.T) {
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

		err := s.Engine.UpsertPrInfo(branchA, testhelpers.NewTestPrInfo(101).
			WithURL("https://github.com/owner/repo/pull/101"))
		require.NoError(t, err)

		err = s.Engine.UpsertPrInfo(branchB, testhelpers.NewTestPrInfo(102).
			WithURL("https://github.com/owner/repo/pull/102"))
		require.NoError(t, err)

		err = s.Engine.UpsertPrInfo(branchC, testhelpers.NewTestPrInfo(103).
			WithURL("https://github.com/owner/repo/pull/103"))
		require.NoError(t, err)

		// Switch to branch-c (top of stack)
		s.Checkout("branch-c")

		prInfo, upstack, err := findBottomUnmergedPR(s.Context)
		require.NoError(t, err)
		require.NotNil(t, prInfo)

		// Should find branch-a as the bottom PR
		require.Equal(t, "branch-a", prInfo.BranchName)
		require.Equal(t, 101, prInfo.PRNumber)

		// Upstack should include branch-b and branch-c
		require.Contains(t, upstack, "branch-b")
		require.Contains(t, upstack, "branch-c")
	})

	t.Run("calculates upstack from merged branch not current branch", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			WithStack(map[string]string{
				"branch-a": "main",
				"branch-b": "branch-a",
				"branch-c": "branch-b",
				"branch-d": "branch-c",
			})

		// Add PR info to all branches
		branches := []string{"branch-a", "branch-b", "branch-c", "branch-d"}
		for i, branchName := range branches {
			branch := s.Engine.GetBranch(branchName)
			err := s.Engine.UpsertPrInfo(branch, testhelpers.NewTestPrInfo(100+i+1).
				WithURL("https://github.com/owner/repo/pull/"+branchName))
			require.NoError(t, err)
		}

		// Switch to branch-c (not the top, not the bottom)
		s.Checkout("branch-c")

		prInfo, upstack, err := findBottomUnmergedPR(s.Context)
		require.NoError(t, err)
		require.NotNil(t, prInfo)

		// Should find branch-a as the bottom PR
		require.Equal(t, "branch-a", prInfo.BranchName)

		// Upstack should include everything above branch-a: branch-b, branch-c, branch-d
		require.Contains(t, upstack, "branch-b")
		require.Contains(t, upstack, "branch-c")
		require.Contains(t, upstack, "branch-d")
		require.Len(t, upstack, 3)
	})
}
