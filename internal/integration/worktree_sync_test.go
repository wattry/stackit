package integration

import (
	"testing"
)

// TestSyncWithMultipleWorktrees tests the sync command behavior when working
// with multiple worktrees and stacked branches.
func TestSyncWithMultipleWorktrees(t *testing.T) {
	t.Run("restack worktree branch after sibling stack merged", func(t *testing.T) {
		// This test reproduces the bug where running st sync from main
		// causes unexpected conflicts when restacking branches in worktrees.
		//
		// Scenario:
		// 1. Create Stack A in worktree A: main -> stackA -> stackA-child
		// 2. Create Stack B in worktree B: main -> stackB -> stackB-child
		// 3. Simulate Stack A getting merged on GitHub
		// 4. Run sync from main repo
		// 5. Expect Stack B to be restacked without unexpected conflicts

		sh := NewTestShellWithRemoteInProcess(t)

		// === Setup: Create two stacks, each in their own worktree ===

		// Stack A: main -> stackA (in worktree A)
		sh.Log("Creating Stack A with worktree...")
		sh.WriteFile("stackA.txt", "stack A content").
			Run("create stackA -w -m 'Stack A root'").
			OnBranch("main") // -w returns to main

		// Get worktree A path and add a child branch
		worktreeA := sh.GetWorktreePath("stackA")
		shA := sh.InWorktree(worktreeA)
		shA.OnBranch("stackA").
			WriteFile("stackA-child.txt", "stack A child content").
			Run("create stackA-child -m 'Stack A child'").
			OnBranch("stackA-child")

		// Stack B: main -> stackB (in worktree B)
		sh.Log("Creating Stack B with worktree...")
		sh.WriteFile("stackB.txt", "stack B content").
			Run("create stackB -w -m 'Stack B root'").
			OnBranch("main")

		// Get worktree B path and add a child branch
		worktreeB := sh.GetWorktreePath("stackB")
		shB := sh.InWorktree(worktreeB)
		shB.OnBranch("stackB").
			WriteFile("stackB-child.txt", "stack B child content").
			Run("create stackB-child -m 'Stack B child'").
			OnBranch("stackB-child")

		// Verify initial stack structure
		sh.Log("Verifying initial stack structure...")
		sh.ExpectBranchParent("stackA", "main")
		sh.ExpectBranchParent("stackA-child", "stackA")
		sh.ExpectBranchParent("stackB", "main")
		sh.ExpectBranchParent("stackB-child", "stackB")

		// === Simulate: Stack A gets merged on GitHub ===
		sh.Log("Simulating Stack A merge on GitHub...")

		// Fast-forward main to include stackA (simulating GitHub squash/merge)
		sh.Git("checkout main").
			Git("merge stackA --ff-only").
			Git("push origin main")

		// Mark stackA PR as merged in metadata
		sh.SetPrState("stackA", "MERGED")

		// === Action: Run sync from main repo ===
		sh.Log("Running sync from main repo...")

		// This should:
		// 1. Pull main (now includes stackA commits)
		// 2. Clean up merged stackA branch
		// 3. Reparent stackA-child to main
		// 4. Restack stackB onto updated main (THIS IS WHERE THE BUG MAY OCCUR)
		sh.OnBranch("main").
			Run("sync")

		// === Verify: No unexpected conflicts ===
		sh.Log("Verifying sync completed without unexpected conflicts...")
		sh.OutputNotContains("conflict")

		// stackB should be rebased onto updated main
		sh.ExpectBranchParent("stackB", "main")
		sh.ExpectBranchParent("stackB-child", "stackB")

		// stackA-child should be reparented to main (since stackA was merged)
		sh.ExpectBranchParent("stackA-child", "main")

		// Verify worktree B is still functional
		sh.Log("Verifying worktree B is still functional...")
		shB.Checkout("stackB-child").
			OnBranch("stackB-child").
			WriteFile("newfile.txt", "new content").
			Run("create stackB-grandchild -m 'New branch works'").
			OnBranch("stackB-grandchild")
	})

	t.Run("restack deep stack in worktree from main repo", func(t *testing.T) {
		// Test that a deep stack in a worktree can be restacked from the main repo
		// when main advances.

		sh := NewTestShellWithRemoteInProcess(t)

		// Create a deep stack in a worktree: main -> a -> b -> c -> d
		sh.Log("Creating deep stack in worktree...")
		sh.WriteFile("a.txt", "a").
			Run("create a -w -m 'branch a'").
			OnBranch("main")

		worktreeA := sh.GetWorktreePath("a")
		shA := sh.InWorktree(worktreeA)
		shA.OnBranch("a").
			WriteFile("b.txt", "b").
			Run("create b -m 'branch b'").
			OnBranch("b").
			WriteFile("c.txt", "c").
			Run("create c -m 'branch c'").
			OnBranch("c").
			WriteFile("d.txt", "d").
			Run("create d -m 'branch d'").
			OnBranch("d")

		// Advance main in the main repo
		sh.Log("Advancing main...")
		sh.WriteFile("main-update.txt", "main change").
			Git("add main-update.txt").
			Git("commit -m 'Main advanced'").
			Git("push origin main")

		// Sync from main - should restack entire worktree stack
		sh.Log("Running sync from main repo...")
		sh.Run("sync")

		// Verify all branches restacked without conflict
		sh.Log("Verifying no conflicts...")
		sh.OutputNotContains("conflict")

		// Verify stack structure is maintained
		sh.ExpectStackStructure(map[string]string{
			"a": "main",
			"b": "a",
			"c": "b",
			"d": "c",
		})
	})

	t.Run("sync from worktree context", func(t *testing.T) {
		// Test that sync works correctly when run from within a worktree
		// (the opposite direction of the main bug)

		sh := NewTestShellWithRemoteInProcess(t)

		// Create a stack in a worktree
		sh.Log("Creating stack in worktree...")
		sh.WriteFile("feature.txt", "feature").
			Run("create feature -w -m 'feature branch'").
			OnBranch("main")

		worktree := sh.GetWorktreePath("feature")
		shW := sh.InWorktree(worktree)
		shW.OnBranch("feature").
			WriteFile("child.txt", "child").
			Run("create feature-child -m 'feature child'")

		// Advance main in the main repo
		sh.Log("Advancing main...")
		sh.WriteFile("main-update.txt", "main change").
			Git("add main-update.txt").
			Git("commit -m 'Main advanced'").
			Git("push origin main")

		// Sync from the worktree context
		sh.Log("Running sync from worktree...")
		shW.Checkout("feature").
			Run("sync")

		// Verify no conflicts
		shW.OutputNotContains("conflict")

		// Verify stack structure
		sh.ExpectStackStructure(map[string]string{
			"feature":       "main",
			"feature-child": "feature",
		})
	})
}
