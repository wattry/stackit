package flatten_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"stackit.dev/stackit/internal/actions/flatten"
	"stackit.dev/stackit/testhelpers"
	"stackit.dev/stackit/testhelpers/scenario"
)

func writeFile(t *testing.T, s *scenario.Scenario, name, content string) {
	t.Helper()
	err := os.WriteFile(filepath.Join(s.Scene.Repo.Dir, name), []byte(content), 0644)
	require.NoError(t, err)
}

func TestFlattenAction(t *testing.T) {
	t.Run("flattens linear independent stack to trunk", func(t *testing.T) {
		// main -> A -> B -> C
		// WithStack creates independent changes (separate files)
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			WithStack(map[string]string{
				"A": "main",
				"B": "A",
				"C": "B",
			})

		err := flatten.FlattenAction(s.Context, "C")
		require.NoError(t, err)

		s.Rebuild()
		
		branchA := s.Engine.GetBranch("A")
		branchB := s.Engine.GetBranch("B")
		branchC := s.Engine.GetBranch("C")
		
		require.Equal(t, "main", branchA.GetParent().GetName())
		require.Equal(t, "main", branchB.GetParent().GetName())
		require.Equal(t, "main", branchC.GetParent().GetName())
	})

	t.Run("respects dependencies", func(t *testing.T) {
		// main -> A -> B
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			WithStack(map[string]string{
				"A": "main",
				"B": "A",
			})

		// Make B depend on A
		// A created "A_test.txt". B created "B_test.txt".
		// We modify "A_test.txt" in B.
		s.Checkout("B")
		writeFile(t, s, "A_test.txt", "modified by B")
		s.RunGit("add", ".")
		s.RunGit("commit", "-m", "B depends on A")

		s.Rebuild()

		err := flatten.FlattenAction(s.Context, "B")
		require.NoError(t, err)

		s.Rebuild()
		
		branchA := s.Engine.GetBranch("A")
		branchB := s.Engine.GetBranch("B")

		require.Equal(t, "main", branchA.GetParent().GetName())
		require.Equal(t, "A", branchB.GetParent().GetName())
	})

	t.Run("partial flatten", func(t *testing.T) {
		// main -> A -> B -> C
		// A independent
		// B depends on A
		// C independent of B (and A)
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			WithStack(map[string]string{
				"A": "main",
				"B": "A",
				"C": "B",
			})

		// Make B depend on A
		s.Checkout("B")
		writeFile(t, s, "A_test.txt", "modified by B")
		s.RunGit("add", ".")
		s.RunGit("commit", "-m", "B depends on A")
		
		// C is on top of B, but only touches C_test.txt (from WithStack)
		// So C should be able to skip B and A and go to main?
		// Wait, C starts at B. B depends on A.
		// If C only has "C_test.txt", and main lacks "A_test.txt" and "B_test.txt".
		// C adds "C_test.txt". It should apply cleanly on main.
		// However, C *state* currently includes B's changes.
		// Rebase --onto main B C
		// This takes commits between B..C (which is just C's commit) and plays them on main.
		// C's commit only adds C_test.txt.
		// So it should work.

		s.Rebuild()

		err := flatten.FlattenAction(s.Context, "C")
		require.NoError(t, err)
		if err != nil {
			t.Logf("Output: %s", s.Output.String())
			out, _ := s.Scene.Repo.RunGitCommandAndGetOutput("log", "--graph", "--oneline", "--all")
			t.Logf("Git Log:\n%s", out)
		}
		require.NoError(t, err)

		s.Rebuild()
		
		branchA := s.Engine.GetBranch("A")
		branchB := s.Engine.GetBranch("B")
		branchC := s.Engine.GetBranch("C")

		require.Equal(t, "main", branchA.GetParent().GetName())
		require.Equal(t, "A", branchB.GetParent().GetName())
		require.Equal(t, "main", branchC.GetParent().GetName())
	})
}
