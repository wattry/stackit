package integration

import (
	"strings"
	"testing"
)

func TestRestackWithStaleMetadataButMatchingParentRev(t *testing.T) {
	t.Parallel()
	sh := NewTestShellInProcess(t)

	// This test reproduces the bug where restackBranch's early-exit check
	// compares meta.ParentBranchRevision to parentRev BEFORE running resiliency logic.
	// If ParentBranchRevision equals the current parent tip but is NOT an ancestor
	// of the branch (happens after parent was amended/rebased), the early-exit
	// incorrectly returns "unneeded" instead of running the resiliency logic.

	// 1. Create initial stack: main -> A
	sh.WriteFile("file.txt", "original content").
		Git("commit -m 'initial'")
	sh.Run("create A -m 'branch A'").
		WriteFile("file.txt", "A's content").
		Git("commit -m 'A changes file'")

	// 2. Go back to main and make a conflicting change
	sh.Checkout("main").
		WriteFile("file.txt", "main's conflicting content").
		Git("commit -m 'main also changes file'")

	// Get current main revision
	sh.Git("rev-parse main")
	mainRev := strings.TrimSpace(sh.Output())

	// 3. Manually update A's metadata to have ParentBranchRevision = current main tip
	// This creates a situation where:
	// - meta.ParentBranchRevision == parentRev (would trigger early-exit)
	// - But mainRev is NOT an ancestor of A (resiliency should use merge-base)
	metadataJSON := `{"parentBranchName":"main","parentBranchRevision":"` + mainRev + `"}`
	sh.WriteFile(".metadata_tmp", metadataJSON)
	sh.Git("hash-object -w .metadata_tmp")
	blobSha := strings.TrimSpace(sh.Output())
	sh.Git("update-ref refs/stackit/metadata/A " + blobSha)
	// Clean up the temp file and staged changes
	sh.Git("reset HEAD")
	sh.Git("clean -f")

	// 4. Restack should:
	// - NOT early-exit based on meta.ParentBranchRevision == parentRev
	// - Run resiliency logic, detect mainRev is not an ancestor
	// - Use merge-base as oldParentRev
	// - Perform rebase which will hit a real conflict
	sh.Checkout("A").
		RunExpectError("restack")

	// The bug manifests as "expected conflict on A but rebase completed successfully"
	// which means the rebase was skipped entirely due to the early-exit check.
	// The fix should make the rebase actually happen and hit a conflict.
	sh.OutputNotContains("rebase completed successfully").
		OutputContains("restack stopped due to conflict")
}

func TestResiliencyStaleParentSHA(t *testing.T) {
	t.Parallel()
	shell := NewTestShellInProcess(t)

	// 1. Create a stack: main -> branch-a -> branch-b
	shell.Log("Creating stack: main -> branch-a -> branch-b")
	shell.Write("file-a.txt", "content a").Run("create branch-a -m 'Commit A'")
	shell.Write("file-b.txt", "content b").Run("create branch-b -m 'Commit B'")

	// 2. Manually break branch-a's SHA. Keep metadata for branch-b, but it's now stale.
	shell.Log("Amending branch-a outside of stackit")
	shell.Checkout("branch-a")
	shell.Git("commit --amend -m 'Commit A Amended'")

	// 3. Now branch-b's metadata is stale (points to old Commit A SHA).
	// Try to restack branch-b. It should now auto-discover the fork point and succeed.
	shell.Log("Attempting to restack branch-b with stale metadata")
	shell.Checkout("branch-b")
	shell.Run("restack --only")

	// 4. Verify branch-b is now on top of the amended branch-a
	shell.Log("Verifying branch-b is correctly rebased")
	shell.OnBranch("branch-b")
	// Verify that branch-a amended commit is an ancestor of branch-b
	shell.Git("merge-base --is-ancestor branch-a branch-b")
}
