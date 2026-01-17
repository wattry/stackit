package integration

import (
	"testing"
)

func TestMoveMarksBranchesForPRBodyUpdate(t *testing.T) {
	t.Run("marks moved branch for PR body update", func(t *testing.T) {
		sh := NewTestShellInProcess(t)

		// Create a stack: main → A → B
		sh.Write("a", "a").Run("create feature-a -m 'feat: feature A'")
		sh.Write("b", "b").Run("create feature-b -m 'feat: feature B'")

		// Verify initial state - no branches need PR body update
		sh.ExpectNeedsPRBodyUpdate("feature-a", false)
		sh.ExpectNeedsPRBodyUpdate("feature-b", false)

		// Move B onto main (changing its parent from A to main)
		sh.Run("move feature-b --onto main --yes")

		// The moved branch (B) should be marked for PR body update
		sh.ExpectNeedsPRBodyUpdate("feature-b", true)

		// The old parent (A) should also be marked since B was its child
		sh.ExpectNeedsPRBodyUpdate("feature-a", true)
	})

	t.Run("does not mark trunk as old parent", func(t *testing.T) {
		sh := NewTestShellInProcess(t)

		// Create two branches off main: main → A and main → B
		sh.Write("a", "a").Run("create feature-a -m 'feat: feature A'")
		sh.Checkout("main")
		sh.Write("b", "b").Run("create feature-b -m 'feat: feature B'")

		// Move B onto A (parent changes from main to A)
		sh.Run("move feature-b --onto feature-a --yes")

		// B should be marked
		sh.ExpectNeedsPRBodyUpdate("feature-b", true)

		// A should NOT be marked because main (trunk) was the old parent
		// and trunk doesn't have PRs to update
		sh.ExpectNeedsPRBodyUpdate("feature-a", false)
	})

	t.Run("marks old parent when moving off non-trunk branch", func(t *testing.T) {
		sh := NewTestShellInProcess(t)

		// Create a stack: main → A → B → C
		sh.Write("a", "a").Run("create feature-a -m 'feat: feature A'")
		sh.Write("b", "b").Run("create feature-b -m 'feat: feature B'")
		sh.Write("c", "c").Run("create feature-c -m 'feat: feature C'")

		// Move C onto A (changing its parent from B to A)
		sh.Run("move feature-c --onto feature-a --yes")

		// C should be marked (it moved)
		sh.ExpectNeedsPRBodyUpdate("feature-c", true)

		// B should be marked (C was its child)
		sh.ExpectNeedsPRBodyUpdate("feature-b", true)

		// A should NOT be marked (it's just the new parent, not affected)
		sh.ExpectNeedsPRBodyUpdate("feature-a", false)
	})

	t.Run("flag persists across operations", func(t *testing.T) {
		sh := NewTestShellInProcess(t)

		// Create a stack: main → A → B
		sh.Write("a", "a").Run("create feature-a -m 'feat: feature A'")
		sh.Write("b", "b").Run("create feature-b -m 'feat: feature B'")

		// Move B onto main
		sh.Run("move feature-b --onto main --yes")

		// Verify flags are set
		sh.ExpectNeedsPRBodyUpdate("feature-b", true)
		sh.ExpectNeedsPRBodyUpdate("feature-a", true)

		// Run some other commands that shouldn't clear the flag
		sh.Run("log")
		sh.Run("info")

		// Flags should still be set
		sh.ExpectNeedsPRBodyUpdate("feature-b", true)
		sh.ExpectNeedsPRBodyUpdate("feature-a", true)
	})
}
