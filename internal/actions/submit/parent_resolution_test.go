package submit

import (
	"testing"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/testhelpers"
	"stackit.dev/stackit/testhelpers/scenario"
)

func TestResolveSubmitParentNameSkipsWorktreeAnchors(t *testing.T) {
	t.Parallel()

	s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
		WithStack(map[string]string{
			"wt-anchor": "main",
			"feature":   "wt-anchor",
		})

	err := s.Engine.SetBranchType(s.Engine.GetBranch("wt-anchor"), git.BranchTypeWorktreeAnchor)
	require.NoError(t, err)

	parent := resolveSubmitParentName(s.Engine, s.Engine.GetBranch("feature"))
	require.Equal(t, "main", parent)
}

func TestResolveSubmitParentNamePreservesNormalParents(t *testing.T) {
	t.Parallel()

	s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
		WithStack(map[string]string{
			"parent": "main",
			"child":  "parent",
		})

	parent := resolveSubmitParentName(s.Engine, s.Engine.GetBranch("child"))
	require.Equal(t, "parent", parent)
}
