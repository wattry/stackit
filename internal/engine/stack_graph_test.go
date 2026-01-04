package engine_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/testhelpers"
	"stackit.dev/stackit/testhelpers/scenario"
)

func TestStackGraphRangeAncestorsExcludeTrunk(t *testing.T) {
	t.Parallel()

	s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
		WithStack(map[string]string{
			"branch1": "main",
			"branch2": "branch1",
		})

	graph := engine.BuildStackGraph(s.Engine, engine.SortStrategyAlphabetical, nil)
	branches := graph.Range(s.Engine.GetBranch("branch2"), engine.StackRange{RecursiveParents: true})

	names := make([]string, 0, len(branches))
	for _, b := range branches {
		names = append(names, b.GetName())
	}

	require.Equal(t, []string{"branch1"}, names)
	require.NotContains(t, names, "main")
}

func TestStackGraphRangeDescendantsOrderParentsFirst(t *testing.T) {
	t.Parallel()

	s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
		WithStack(map[string]string{
			"stackA":       "main",
			"stackA-child": "stackA",
			"stackB":       "main",
			"stackB-child": "stackB",
		})

	graph := engine.BuildStackGraph(s.Engine, engine.SortStrategyAlphabetical, nil)
	branches := graph.Range(s.Engine.Trunk(), engine.StackRange{
		RecursiveChildren: true,
	})

	names := make([]string, 0, len(branches))
	for _, b := range branches {
		names = append(names, b.GetName())
	}

	require.Len(t, names, 4)
	require.Contains(t, names, "stackA")
	require.Contains(t, names, "stackA-child")
	require.Contains(t, names, "stackB")
	require.Contains(t, names, "stackB-child")

	stackAIdx := indexOfName(names, "stackA")
	stackAChildIdx := indexOfName(names, "stackA-child")
	stackBIdx := indexOfName(names, "stackB")
	stackBChildIdx := indexOfName(names, "stackB-child")

	require.Less(t, stackAIdx, stackAChildIdx, "stackA should come before stackA-child")
	require.Less(t, stackBIdx, stackBChildIdx, "stackB should come before stackB-child")
}

func TestStackGraphFilterPrunesSubtrees(t *testing.T) {
	t.Parallel()

	s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
		WithStack(map[string]string{
			"a":  "main",
			"a1": "a",
			"b":  "main",
			"b1": "b",
		})

	graph := engine.BuildStackGraph(s.Engine, engine.SortStrategyAlphabetical, func(b engine.Branch) bool {
		return b.GetName() != "b"
	})

	branches := graph.Range(s.Engine.Trunk(), engine.StackRange{
		RecursiveChildren: true,
	})

	names := make([]string, 0, len(branches))
	for _, b := range branches {
		names = append(names, b.GetName())
	}

	require.Contains(t, names, "a")
	require.Contains(t, names, "a1")
	require.NotContains(t, names, "b")
	require.NotContains(t, names, "b1")
}

// indexOfName returns the index of item in slice, or -1 if not found
func indexOfName(slice []string, item string) int {
	for i, s := range slice {
		if s == item {
			return i
		}
	}
	return -1
}
