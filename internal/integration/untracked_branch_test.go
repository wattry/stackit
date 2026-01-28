package integration

import (
	"testing"
)

func TestSubmitUntrackedBranch(t *testing.T) {
	t.Parallel()

	t.Run("submit on untracked branch shows tracking suggestion in non-interactive mode", func(t *testing.T) {
		t.Parallel()
		sh := NewTestShellInProcess(t, WithRemote())

		// Create an untracked branch using raw git (not stackit)
		sh.Git("checkout -b untracked-feature")
		sh.Write("feature.txt", "some content")
		sh.Git("add .")
		sh.Git("commit -m 'add feature'")

		// Run ss with --dry-run - in non-interactive mode, should inform user to track
		sh.Run("ss --dry-run").
			OutputContains("not tracked").
			OutputContains("stackit track")
	})

	t.Run("submit on untracked branch behind main shows tracking suggestion", func(t *testing.T) {
		t.Parallel()
		sh := NewTestShellInProcess(t, WithRemote())

		// Create an untracked branch
		sh.Git("checkout -b untracked-feature")
		sh.Write("feature.txt", "some content")
		sh.Git("add .")
		sh.Git("commit -m 'add feature'")

		// Go back to main and advance it with new commits
		sh.Checkout("main")
		sh.Write("main-update.txt", "main content")
		sh.Git("add .")
		sh.Git("commit -m 'advance main'")

		// Go back to the untracked branch (now behind main)
		sh.Checkout("untracked-feature")

		// Run ss with --dry-run - should show tracking suggestion
		sh.Run("ss --dry-run").
			OutputContains("not tracked").
			OutputContains("stackit track")
	})

	t.Run("submit with --branch flag on untracked branch shows tracking suggestion", func(t *testing.T) {
		t.Parallel()
		sh := NewTestShellInProcess(t, WithRemote())

		// Create an untracked branch using raw git
		sh.Git("checkout -b untracked-feature")
		sh.Write("feature.txt", "some content")
		sh.Git("add .")
		sh.Git("commit -m 'add feature'")

		// Go back to main
		sh.Checkout("main")

		// Run ss with --branch flag pointing to untracked branch
		// Should still detect the untracked branch and show suggestion
		sh.Run("ss --dry-run --branch untracked-feature").
			OutputContains("not tracked").
			OutputContains("stackit track")
	})

	t.Run("submit on tracked branch works normally", func(t *testing.T) {
		t.Parallel()
		sh := NewTestShellInProcess(t, WithRemote())

		// Create a tracked branch using stackit
		sh.Write("feature.txt", "some content")
		sh.Run("create feature -m 'add feature'")

		// Run ss with --dry-run - should show the branch in submission plan
		sh.Run("ss --dry-run").
			OutputContains("feature").
			OutputContains("create")
	})

	t.Run("submit with --branch flag on tracked branch works normally", func(t *testing.T) {
		t.Parallel()
		sh := NewTestShellInProcess(t, WithRemote())

		// Create a tracked branch using stackit
		sh.Write("feature.txt", "some content")
		sh.Run("create feature -m 'add feature'")

		// Go back to main
		sh.Checkout("main")

		// Run ss with --branch flag - should work normally
		sh.Run("ss --dry-run --branch feature").
			OutputContains("feature").
			OutputContains("create")
	})
}
