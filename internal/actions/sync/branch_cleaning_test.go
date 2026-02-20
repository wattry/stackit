package sync

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/testhelpers"
	"stackit.dev/stackit/testhelpers/scenario"
)

func TestCleanBranchesNonInteractiveRespectsDirtyStackFilter(t *testing.T) {
	t.Parallel()

	s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
		WithStack(map[string]string{
			"branch1": "main",
		})

	// Mark branch as deletable.
	s.Checkout("main").RunGit("merge", "branch1")
	err := s.Engine.Rebuild("main")
	require.NoError(t, err)
	err = s.Engine.UpsertPrInfo(context.Background(), s.Engine.GetBranch("branch1"), testhelpers.NewTestPrInfoMerged(1, "main"))
	require.NoError(t, err)

	// Non-interactive path uses NullHandler. Dirty stack filter should prevent deletion.
	result, err := cleanBranches(
		s.Context,
		&Options{Force: true},
		map[string]bool{"branch1": true},
		&NullHandler{},
		&Summary{},
	)
	require.NoError(t, err)
	require.Empty(t, result.DeletedBranches)
	require.True(t, s.Engine.GetBranch("branch1").IsTracked())
}
