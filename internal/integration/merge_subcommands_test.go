package integration

import (
	"testing"
)

func TestMergeNext(t *testing.T) {
	t.Parallel()
	t.Run("dry-run shows plan for bottom PR", func(t *testing.T) {
		t.Parallel()
		sh := NewTestShellInProcess(t)

		// Create a stack with PRs
		sh.Write("a.txt", "content-a").
			Run("create branch-a -m 'Add branch-a'")

		sh.Write("b.txt", "content-b").
			Run("create branch-b -m 'Add branch-b'")

		// Add PR info to branches
		sh.SetPrMetadata("branch-a", PRMetadata{
			Number: 101,
			State:  "OPEN",
			URL:    "https://github.com/owner/repo/pull/101",
		})
		sh.SetPrMetadata("branch-b", PRMetadata{
			Number: 102,
			State:  "OPEN",
			URL:    "https://github.com/owner/repo/pull/102",
		})

		// Run merge next --dry-run
		sh.Run("merge next --dry-run")

		// Should show the bottom PR
		sh.OutputContains("branch-a").
			OutputContains("#101").
			OutputContains("Dry-run mode")
	})

	t.Run("errors when on trunk", func(t *testing.T) {
		t.Parallel()
		sh := NewTestShellInProcess(t)

		// Stay on main
		sh.OnBranch("main")

		// Should error when trying to merge from trunk
		sh.RunExpectError("merge next --dry-run")
		sh.OutputContains("cannot merge from trunk")
	})

	t.Run("errors when branch is not tracked", func(t *testing.T) {
		t.Parallel()
		sh := NewTestShellInProcess(t)

		// Create an untracked branch
		sh.Git("checkout -b untracked-branch")
		sh.Write("untracked.txt", "content")
		sh.Git("add -A")
		sh.Git("commit -m 'untracked commit'")

		// Should error
		sh.RunExpectError("merge next --dry-run")
		sh.OutputContains("not tracked")
	})

	t.Run("errors when no PRs to merge", func(t *testing.T) {
		t.Parallel()
		sh := NewTestShellInProcess(t)

		// Create a stack without PRs
		sh.Write("a.txt", "content-a").
			Run("create branch-a -m 'Add branch-a'")

		// Run merge next - should error since no PRs exist
		sh.RunExpectError("merge next --dry-run")

		// Should indicate no PRs found
		sh.OutputContains("no open PRs")
	})

	t.Run("skips already merged PRs", func(t *testing.T) {
		t.Parallel()
		sh := NewTestShellInProcess(t)

		// Create a stack
		sh.Write("a.txt", "content-a").
			Run("create branch-a -m 'Add branch-a'")

		sh.Write("b.txt", "content-b").
			Run("create branch-b -m 'Add branch-b'")

		// Mark branch-a as merged, branch-b as open
		sh.SetPrMetadata("branch-a", PRMetadata{
			Number: 101,
			State:  "MERGED",
			URL:    "https://github.com/owner/repo/pull/101",
		})
		sh.SetPrMetadata("branch-b", PRMetadata{
			Number: 102,
			State:  "OPEN",
			URL:    "https://github.com/owner/repo/pull/102",
		})

		// Run merge next --dry-run
		sh.Run("merge next --dry-run")

		// Should find branch-b, not branch-a
		sh.OutputContains("branch-b").
			OutputContains("#102").
			OutputNotContains("#101")
	})

	t.Run("does not block on closed upstack PRs", func(t *testing.T) {
		t.Parallel()
		sh := NewTestShellInProcess(t)

		// Create a stack
		sh.Write("a.txt", "content-a").
			Run("create branch-a -m 'Add branch-a'")

		sh.Write("b.txt", "content-b").
			Run("create branch-b -m 'Add branch-b'")

		// Mark bottom PR as open, upstack PR as closed
		sh.SetPrMetadata("branch-a", PRMetadata{
			Number: 101,
			State:  "OPEN",
			URL:    "https://github.com/owner/repo/pull/101",
		})
		sh.SetPrMetadata("branch-b", PRMetadata{
			Number: 102,
			State:  "CLOSED",
			URL:    "https://github.com/owner/repo/pull/102",
		})

		// Run merge next --dry-run
		sh.Run("merge next --dry-run")

		// Should focus on bottom PR only
		sh.OutputContains("Enable automerge for PR #101").
			OutputNotContains("PR #102")
	})

	t.Run("shows draft PR in plan", func(t *testing.T) {
		t.Parallel()
		sh := NewTestShellInProcess(t)

		// Create a stack with a draft PR
		sh.Write("a.txt", "content-a").
			Run("create branch-a -m 'Add branch-a'")

		// Mark branch-a as draft
		sh.SetPrMetadata("branch-a", PRMetadata{
			Number: 101,
			State:  "OPEN",
			Draft:  true,
			URL:    "https://github.com/owner/repo/pull/101",
		})

		// Dry-run shows the plan (draft check happens during actual merge)
		sh.Run("merge next --dry-run")
		sh.OutputContains("branch-a").
			OutputContains("#101")
	})
}

func TestMergeSquash(t *testing.T) {
	t.Parallel()
	t.Run("dry-run shows consolidation plan", func(t *testing.T) {
		t.Parallel()
		sh := NewTestShellInProcess(t)

		// Create a stack with PRs
		sh.Write("a.txt", "content-a").
			Run("create branch-a -m 'Add branch-a'")

		sh.Write("b.txt", "content-b").
			Run("create branch-b -m 'Add branch-b'")

		// Add PR info to branches
		sh.SetPrMetadata("branch-a", PRMetadata{
			Number: 101,
			State:  "OPEN",
			URL:    "https://github.com/owner/repo/pull/101",
		})
		sh.SetPrMetadata("branch-b", PRMetadata{
			Number: 102,
			State:  "OPEN",
			URL:    "https://github.com/owner/repo/pull/102",
		})

		// Run merge squash --dry-run
		sh.Run("merge squash --dry-run")

		// Should show consolidation plan
		sh.OutputContains("Consolidate").
			OutputContains("branch-a").
			OutputContains("branch-b").
			OutputContains("Dry-run mode")
	})

	t.Run("errors when on trunk", func(t *testing.T) {
		t.Parallel()
		sh := NewTestShellInProcess(t)

		// Stay on main
		sh.OnBranch("main")

		// Should error when trying to squash from trunk
		sh.RunExpectError("merge squash --dry-run")
		sh.OutputContains("cannot merge from trunk")
	})

	t.Run("shows all branches in plan", func(t *testing.T) {
		t.Parallel()
		sh := NewTestShellInProcess(t)

		// Create a longer stack
		sh.Write("a.txt", "content-a").
			Run("create branch-a -m 'Add branch-a'")

		sh.Write("b.txt", "content-b").
			Run("create branch-b -m 'Add branch-b'")

		sh.Write("c.txt", "content-c").
			Run("create branch-c -m 'Add branch-c'")

		// Add PR info to all branches
		sh.SetPrMetadata("branch-a", PRMetadata{Number: 101, State: "OPEN"})
		sh.SetPrMetadata("branch-b", PRMetadata{Number: 102, State: "OPEN"})
		sh.SetPrMetadata("branch-c", PRMetadata{Number: 103, State: "OPEN"})

		// Run merge squash --dry-run
		sh.Run("merge squash --dry-run")

		// Should show all branches
		sh.OutputContains("branch-a").
			OutputContains("branch-b").
			OutputContains("branch-c")
	})

	t.Run("scope flag errors when no branches match", func(t *testing.T) {
		t.Parallel()
		sh := NewTestShellInProcess(t)

		// Create a stack without scope
		sh.Write("a.txt", "content-a").
			Run("create branch-a -m 'Add branch-a'")

		// Add PR info
		sh.SetPrMetadata("branch-a", PRMetadata{Number: 101, State: "OPEN"})

		// Try to squash with non-existent scope
		sh.RunExpectError("merge squash --scope nonexistent --dry-run")
		sh.OutputContains("no branches found")
	})

	t.Run("multi-stack dry-run shows plan without executing", func(t *testing.T) {
		t.Parallel()
		sh := NewTestShellInProcess(t)

		// Create two independent stacks rooted at trunk
		sh.Write("a.txt", "content-a").
			Run("create stack-a -m 'Add stack-a'")

		sh.Checkout("main")

		sh.Write("b.txt", "content-b").
			Run("create stack-b -m 'Add stack-b'")

		// Dry-run multi-stack merge
		sh.Run("merge ship --stacks stack-a,stack-b --dry-run")

		sh.OutputContains("Dry-run mode").
			OutputContains("stack-a").
			OutputContains("stack-b")
	})
}

func TestMergeCommand(t *testing.T) {
	t.Parallel()
	t.Run("shows help with subcommands", func(t *testing.T) {
		t.Parallel()
		sh := NewTestShellInProcess(t)

		// Run merge --help
		sh.Run("merge --help")

		// Should show subcommands
		sh.OutputContains("next").
			OutputContains("ship")
	})

	t.Run("next subcommand is accessible", func(t *testing.T) {
		t.Parallel()
		sh := NewTestShellInProcess(t)

		// Run merge next --help
		sh.Run("merge next --help")

		sh.OutputContains("bottom-most").
			OutputContains("--dry-run").
			OutputContains("--wait")
	})

	t.Run("squash subcommand is accessible", func(t *testing.T) {
		t.Parallel()
		sh := NewTestShellInProcess(t)

		// Run merge squash --help
		sh.Run("merge squash --help")

		sh.OutputContains("Consolidate").
			OutputContains("--scope").
			OutputContains("--stacks")
	})

	t.Run("parent command requires TTY in non-interactive mode", func(t *testing.T) {
		t.Parallel()
		sh := NewTestShellInProcess(t)

		// Create a stack
		sh.Write("a.txt", "content-a").
			Run("create branch-a -m 'Add branch-a'")

		// merge (without subcommand) should error in non-interactive mode
		sh.RunExpectError("merge")
		sh.OutputContains("requires a TTY")
	})
}

func TestMergeNextUpstackCalculation(t *testing.T) {
	t.Parallel()
	t.Run("includes upstack info in output", func(t *testing.T) {
		t.Parallel()
		sh := NewTestShellInProcess(t)

		// Create a deep stack
		sh.Write("a.txt", "content-a").
			Run("create branch-a -m 'Add branch-a'")

		sh.Write("b.txt", "content-b").
			Run("create branch-b -m 'Add branch-b'")

		sh.Write("c.txt", "content-c").
			Run("create branch-c -m 'Add branch-c'")

		sh.Write("d.txt", "content-d").
			Run("create branch-d -m 'Add branch-d'")

		// Add PR info to all branches
		sh.SetPrMetadata("branch-a", PRMetadata{Number: 101, State: "OPEN"})
		sh.SetPrMetadata("branch-b", PRMetadata{Number: 102, State: "OPEN"})
		sh.SetPrMetadata("branch-c", PRMetadata{Number: 103, State: "OPEN"})
		sh.SetPrMetadata("branch-d", PRMetadata{Number: 104, State: "OPEN"})

		// Run from branch-d, should find branch-a and list upstack
		sh.Run("merge next --dry-run")

		// Should mention restacking of upstack branches
		sh.OutputContains("branch-a").
			OutputContains("Restack")
	})

	t.Run("handles mid-stack position correctly", func(t *testing.T) {
		t.Parallel()
		sh := NewTestShellInProcess(t)

		// Create a stack
		sh.Write("a.txt", "content-a").
			Run("create branch-a -m 'Add branch-a'")

		sh.Write("b.txt", "content-b").
			Run("create branch-b -m 'Add branch-b'")

		sh.Write("c.txt", "content-c").
			Run("create branch-c -m 'Add branch-c'")

		// Add PR info
		sh.SetPrMetadata("branch-a", PRMetadata{Number: 101, State: "OPEN"})
		sh.SetPrMetadata("branch-b", PRMetadata{Number: 102, State: "OPEN"})
		sh.SetPrMetadata("branch-c", PRMetadata{Number: 103, State: "OPEN"})

		// Checkout branch-b (mid-stack)
		sh.Checkout("branch-b")

		// Should still find branch-a as bottom
		sh.Run("merge next --dry-run")
		sh.OutputContains("branch-a")
	})
}

func TestMergeSquashValidation(t *testing.T) {
	t.Parallel()
	t.Run("shows all PRs in consolidation plan", func(t *testing.T) {
		t.Parallel()
		sh := NewTestShellInProcess(t)

		// Create a stack with multiple PRs
		sh.Write("a.txt", "content-a").
			Run("create branch-a -m 'Add branch-a'")

		sh.Write("b.txt", "content-b").
			Run("create branch-b -m 'Add branch-b'")

		// Add PR info
		sh.SetPrMetadata("branch-a", PRMetadata{Number: 101, State: "OPEN"})
		sh.SetPrMetadata("branch-b", PRMetadata{Number: 102, State: "OPEN"})

		// Dry-run shows consolidation plan
		sh.Run("merge squash --dry-run")
		sh.OutputContains("Consolidate").
			OutputContains("branch-a").
			OutputContains("branch-b")
	})

	t.Run("errors when no open PRs found", func(t *testing.T) {
		t.Parallel()
		sh := NewTestShellInProcess(t)

		// Create a stack with all merged PRs
		sh.Write("a.txt", "content-a").
			Run("create branch-a -m 'Add branch-a'")

		// Mark as merged
		sh.SetPrMetadata("branch-a", PRMetadata{Number: 101, State: "MERGED"})

		// Should error - no open PRs
		sh.RunExpectError("merge squash --dry-run")
		sh.OutputContains("no open PRs")
	})
}

func TestMergeFlags(t *testing.T) {
	t.Parallel()
	t.Run("yes flag skips confirmation in merge next", func(t *testing.T) {
		t.Parallel()
		sh := NewTestShellInProcess(t)

		// Create a stack with PRs
		sh.Write("a.txt", "content-a").
			Run("create branch-a -m 'Add branch-a'")

		sh.SetPrMetadata("branch-a", PRMetadata{Number: 101, State: "OPEN"})

		// --yes --dry-run should work without prompting
		sh.Run("merge next --yes --dry-run")
		sh.OutputContains("branch-a")
	})

	t.Run("yes flag works with merge squash", func(t *testing.T) {
		t.Parallel()
		sh := NewTestShellInProcess(t)

		// Create a stack with PRs
		sh.Write("a.txt", "content-a").
			Run("create branch-a -m 'Add branch-a'")

		sh.SetPrMetadata("branch-a", PRMetadata{Number: 101, State: "OPEN"})

		// --yes --dry-run should work
		sh.Run("merge squash --yes --dry-run")
		sh.OutputContains("Consolidate")
	})

	t.Run("force flag is available", func(t *testing.T) {
		t.Parallel()
		sh := NewTestShellInProcess(t)

		// Create a PR
		sh.Write("a.txt", "content-a").
			Run("create branch-a -m 'Add branch-a'")

		sh.SetPrMetadata("branch-a", PRMetadata{Number: 101, State: "OPEN"})

		// --force flag works
		sh.Run("merge next --force --dry-run")
		sh.OutputContains("branch-a")
	})
}

func TestMergeSingleBranchStack(t *testing.T) {
	t.Parallel()
	t.Run("merge next works for single branch", func(t *testing.T) {
		t.Parallel()
		sh := NewTestShellInProcess(t)

		// Create single branch stack
		sh.Write("a.txt", "content-a").
			Run("create branch-a -m 'Add branch-a'")

		sh.SetPrMetadata("branch-a", PRMetadata{Number: 101, State: "OPEN"})

		// merge next works for single branch
		sh.Run("merge next --dry-run")
		sh.OutputContains("branch-a")
	})

	t.Run("merge squash works for single branch", func(t *testing.T) {
		t.Parallel()
		sh := NewTestShellInProcess(t)

		// Create single branch stack
		sh.Write("a.txt", "content-a").
			Run("create branch-a -m 'Add branch-a'")

		sh.SetPrMetadata("branch-a", PRMetadata{Number: 101, State: "OPEN"})

		// merge squash also works (trivial consolidation)
		sh.Run("merge squash --dry-run")
		sh.OutputContains("Consolidate")
	})
}
