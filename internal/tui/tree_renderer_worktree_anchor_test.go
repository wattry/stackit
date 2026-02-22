package tui_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/tui"
	"stackit.dev/stackit/internal/tui/components/tree"
	"stackit.dev/stackit/testhelpers"
	"stackit.dev/stackit/testhelpers/scenario"
)

func TestNewStackTreeRendererHidesWorktreeAnchorKeepsDescendants(t *testing.T) {
	t.Parallel()

	s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
		WithStack(map[string]string{
			"wt-anchor": "main",
			"feature":   "wt-anchor",
		})

	err := s.Engine.SetBranchType(s.Engine.GetBranch("wt-anchor"), git.BranchTypeWorktreeAnchor)
	require.NoError(t, err)

	renderer := tui.NewStackTreeRenderer(s.Engine)
	lines := renderer.RenderStack("main", tree.RenderOptions{
		Mode:                tree.RenderModeCompact,
		NoStyleBranchName:   true,
		SkipSelectionPrefix: true,
	})

	output := strings.Join(lines, "\n")
	require.NotContains(t, output, "wt-anchor", "worktree anchors should stay hidden in log output")
	require.Contains(t, output, "feature", "children of hidden worktree anchors must remain visible")
}
