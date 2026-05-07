package integration

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/internal/actions/sync"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/testhelpers"
	"stackit.dev/stackit/testhelpers/scenario"
)

// disableCommitSigning stops the test repo from inheriting commit.gpgsign=true
// from a global config. The engine's restack invokes `git commit` without
// stripping global config, so any signing requirement from the host env
// breaks rebase. Local config wins over global, so this is safe in CI too.
func disableCommitSigning(t *testing.T, sh *scenario.Scenario) {
	t.Helper()
	require.NoError(t, sh.Scene.Repo.RunGitCommand("config", "commit.gpgsign", "false"))
	require.NoError(t, sh.Scene.Repo.RunGitCommand("config", "tag.gpgsign", "false"))
}

// markPrMerged sets the PR metadata for branch to MERGED with the given base.
// Mirrors the inline pattern used elsewhere in sync_test.go.
func markPrMerged(t *testing.T, sh *scenario.Scenario, branch string, prNumber int, base string) {
	t.Helper()
	meta, err := sh.Engine.Git().ReadMetadata(branch)
	require.NoError(t, err)
	num := prNumber
	state := prStateMerged
	b := base
	meta = meta.WithPrInfo(&git.PrInfoPersistence{
		Number: &num,
		State:  &state,
		Base:   &b,
	})
	require.NoError(t, sh.Engine.Git().WriteMetadata(branch, meta))
}

// TestSquashMergeMiddleOfStack covers the case where a middle PR (B) is
// squash-merged on GitHub while its parent (A) is still open. The squash
// commit lands on A (the PR base), not on trunk. C must reparent to A,
// not skip over to trunk.
func TestSquashMergeMiddleOfStack(t *testing.T) {
	t.Parallel()
	sh := scenario.NewScenario(t, testhelpers.BasicSceneSetup)
	disableCommitSigning(t, sh)

	// main -> A -> B -> C
	sh.CreateBranch("branch-a").
		CommitChange("file-a", "a").
		TrackBranch("branch-a", "main")
	sh.CreateBranch("branch-b").
		CommitChange("file-b", "b").
		TrackBranch("branch-b", "branch-a")
	sh.CreateBranch("branch-c").
		CommitChange("file-c", "c").
		TrackBranch("branch-c", "branch-b")

	// GitHub squash-merges B's PR (base=A). The squash commit lands on A.
	sh.Checkout("branch-a")
	sh.CommitChange("file-b", "b")
	markPrMerged(t, sh, "branch-b", 2, "branch-a")

	sh.Checkout("main")
	require.NoError(t, sync.Action(sh.Context, sync.Options{Restack: true}, nil))

	branches, err := sh.Scene.Repo.GetLocalBranches()
	require.NoError(t, err)
	require.NotContains(t, branches, "branch-b", "merged middle branch should be deleted")
	require.Contains(t, branches, "branch-a", "open parent should remain")
	require.Contains(t, branches, "branch-c", "open child should remain")

	// C must hop to A (B's parent), not skip to main.
	require.Equal(t, "branch-a", sh.Engine.GetBranch("branch-c").GetParent().GetName(),
		"C should reparent to A, not main")
	require.Equal(t, "main", sh.Engine.GetBranch("branch-a").GetParent().GetName())

	// C keeps its own commits — A's squash content shouldn't replay into C.
	cCount, err := sh.Engine.GetCommitCount(sh.Engine.GetBranch("branch-c"))
	require.NoError(t, err)
	require.Equal(t, 1, cCount)
}

// TestSquashMergeMultipleAdjacentMergedInOneSync covers when both A and B
// were squash-merged on GitHub between syncs. C must reparent across both
// deletions to main in a single pass.
func TestSquashMergeMultipleAdjacentMergedInOneSync(t *testing.T) {
	t.Parallel()
	sh := scenario.NewScenario(t, testhelpers.BasicSceneSetup)
	disableCommitSigning(t, sh)

	sh.CreateBranch("branch-a").
		CommitChange("file-a", "a").
		TrackBranch("branch-a", "main")
	sh.CreateBranch("branch-b").
		CommitChange("file-b", "b").
		TrackBranch("branch-b", "branch-a")
	sh.CreateBranch("branch-c").
		CommitChange("file-c", "c").
		TrackBranch("branch-c", "branch-b")

	// Squash A and then B onto trunk (sequential GitHub merges).
	mainName := sh.Engine.Trunk().GetName()
	sh.Checkout("main")
	sh.CommitChange("file-a", "a")
	sh.CommitChange("file-b", "b")

	markPrMerged(t, sh, "branch-a", 1, mainName)
	markPrMerged(t, sh, "branch-b", 2, mainName)

	require.NoError(t, sync.Action(sh.Context, sync.Options{Restack: true}, nil))

	branches, err := sh.Scene.Repo.GetLocalBranches()
	require.NoError(t, err)
	require.NotContains(t, branches, "branch-a")
	require.NotContains(t, branches, "branch-b")
	require.Contains(t, branches, "branch-c")

	// findNonDeletingAncestor must walk past both deletions to main.
	require.Equal(t, mainName, sh.Engine.GetBranch("branch-c").GetParent().GetName())

	// C should not have inherited A's or B's commits during restack.
	cCount, err := sh.Engine.GetCommitCount(sh.Engine.GetBranch("branch-c"))
	require.NoError(t, err)
	require.Equal(t, 1, cCount)

	out, err := sh.Scene.Repo.RunGitCommandAndGetOutput("status", "--porcelain")
	require.NoError(t, err)
	require.Empty(t, strings.TrimSpace(out), "working tree should be clean after sync")
}

// TestSquashMergeDiamondAllChildrenMerged covers a diamond where A is open
// and both children (B and C) get squash-merged on GitHub. Both children
// should be cleaned up; A is unaffected.
func TestSquashMergeDiamondAllChildrenMerged(t *testing.T) {
	t.Parallel()
	sh := scenario.NewScenario(t, testhelpers.BasicSceneSetup)
	disableCommitSigning(t, sh)

	// main -> A -> {B, C}
	sh.CreateBranch("branch-a").
		CommitChange("file-a", "a").
		TrackBranch("branch-a", "main")
	sh.CreateBranch("branch-b").
		CommitChange("file-b", "b").
		TrackBranch("branch-b", "branch-a")
	sh.Checkout("branch-a")
	sh.CreateBranch("branch-c").
		CommitChange("file-c", "c").
		TrackBranch("branch-c", "branch-a")

	// Both B and C squash-merged into A.
	sh.Checkout("branch-a")
	sh.CommitChange("file-b", "b")
	sh.CommitChange("file-c", "c")
	markPrMerged(t, sh, "branch-b", 2, "branch-a")
	markPrMerged(t, sh, "branch-c", 3, "branch-a")

	sh.Checkout("main")
	require.NoError(t, sync.Action(sh.Context, sync.Options{Restack: true}, nil))

	branches, err := sh.Scene.Repo.GetLocalBranches()
	require.NoError(t, err)
	require.NotContains(t, branches, "branch-b")
	require.NotContains(t, branches, "branch-c")
	require.Contains(t, branches, "branch-a", "open parent should survive losing both children")

	require.Equal(t, "main", sh.Engine.GetBranch("branch-a").GetParent().GetName())
}

// TestSquashMergeSyncWhileOnMergedBranch is a safety-invariants test:
// when the user runs sync while checked out on the doomed branch, sync
// must move HEAD off the branch before deletion, not leave detached HEAD.
// Per .claude/rules/safety-invariants.md "No Detached HEAD State".
func TestSquashMergeSyncWhileOnMergedBranch(t *testing.T) {
	t.Parallel()
	sh := scenario.NewScenario(t, testhelpers.BasicSceneSetup)
	disableCommitSigning(t, sh)

	sh.CreateBranch("branch-a").
		CommitChange("file-a", "a").
		TrackBranch("branch-a", "main")
	sh.CreateBranch("branch-b").
		CommitChange("file-b", "b").
		TrackBranch("branch-b", "branch-a")

	mainName := sh.Engine.Trunk().GetName()
	sh.Checkout("main")
	sh.CommitChange("file-a", "a")
	markPrMerged(t, sh, "branch-a", 1, mainName)

	// User is sitting on the about-to-be-deleted branch when they sync.
	sh.Checkout("branch-a")
	require.NoError(t, sync.Action(sh.Context, sync.Options{Restack: true}, nil))

	branches, err := sh.Scene.Repo.GetLocalBranches()
	require.NoError(t, err)
	require.NotContains(t, branches, "branch-a")
	require.Contains(t, branches, "branch-b")

	// HEAD must point at a branch — not detached.
	headRef, err := sh.Scene.Repo.RunGitCommandAndGetOutput("symbolic-ref", "--quiet", "HEAD")
	require.NoError(t, err, "HEAD must be a symbolic ref (not detached) after sync deletes the checked-out branch")
	require.NotEmpty(t, strings.TrimSpace(headRef))

	out, err := sh.Scene.Repo.RunGitCommandAndGetOutput("status", "--porcelain")
	require.NoError(t, err)
	require.Empty(t, strings.TrimSpace(out), "working tree should be clean")
}

// TestSquashMergeSyncWhileOnChildOfMergedBranch verifies that when the
// current branch is reparented mid-sync, HEAD continues to track that
// branch (now at its restacked commit) — the user does not get yanked
// onto trunk.
func TestSquashMergeSyncWhileOnChildOfMergedBranch(t *testing.T) {
	t.Parallel()
	sh := scenario.NewScenario(t, testhelpers.BasicSceneSetup)
	disableCommitSigning(t, sh)

	sh.CreateBranch("branch-a").
		CommitChange("file-a", "a").
		TrackBranch("branch-a", "main")
	sh.CreateBranch("branch-b").
		CommitChange("file-b", "b").
		TrackBranch("branch-b", "branch-a")

	mainName := sh.Engine.Trunk().GetName()
	sh.Checkout("main")
	sh.CommitChange("file-a", "a")
	markPrMerged(t, sh, "branch-a", 1, mainName)

	// User is on the surviving child while syncing.
	sh.Checkout("branch-b")
	require.NoError(t, sync.Action(sh.Context, sync.Options{Restack: true}, nil))

	branches, err := sh.Scene.Repo.GetLocalBranches()
	require.NoError(t, err)
	require.NotContains(t, branches, "branch-a")
	require.Contains(t, branches, "branch-b")

	require.Equal(t, mainName, sh.Engine.GetBranch("branch-b").GetParent().GetName())

	// User stays on branch-b even though its parent changed under it.
	headRef, err := sh.Scene.Repo.RunGitCommandAndGetOutput("symbolic-ref", "--short", "HEAD")
	require.NoError(t, err)
	require.Equal(t, "branch-b", strings.TrimSpace(headRef))

	bCount, err := sh.Engine.GetCommitCount(sh.Engine.GetBranch("branch-b"))
	require.NoError(t, err)
	require.Equal(t, 1, bCount, "B should keep its single commit after restack onto trunk")

	out, err := sh.Scene.Repo.RunGitCommandAndGetOutput("status", "--porcelain")
	require.NoError(t, err)
	require.Empty(t, strings.TrimSpace(out))
}

// TestSquashMergeNoRestackLeavesGitRefsUntouched verifies that
// `--no-restack` still cleans up merged branches and updates metadata
// parents, but does NOT advance the children's git refs. This documents
// the partial-state contract for users who pass --no-restack expecting
// only cleanup.
func TestSquashMergeNoRestackLeavesGitRefsUntouched(t *testing.T) {
	t.Parallel()
	sh := scenario.NewScenario(t, testhelpers.BasicSceneSetup)
	disableCommitSigning(t, sh)

	sh.CreateBranch("branch-a").
		CommitChange("file-a", "a").
		TrackBranch("branch-a", "main")
	sh.CreateBranch("branch-b").
		CommitChange("file-b", "b").
		TrackBranch("branch-b", "branch-a")

	mainName := sh.Engine.Trunk().GetName()
	sh.Checkout("main")
	sh.CommitChange("file-a", "a")
	markPrMerged(t, sh, "branch-a", 1, mainName)

	bSHAbeforeSync, err := sh.Scene.Repo.RunGitCommandAndGetOutput("rev-parse", "branch-b")
	require.NoError(t, err)

	require.NoError(t, sync.Action(sh.Context, sync.Options{NoRestack: true}, nil))

	branches, err := sh.Scene.Repo.GetLocalBranches()
	require.NoError(t, err)
	require.NotContains(t, branches, "branch-a", "cleanup should still run with --no-restack")
	require.Contains(t, branches, "branch-b")

	// Metadata parent updated to main even though restack was skipped.
	require.Equal(t, mainName, sh.Engine.GetBranch("branch-b").GetParent().GetName())

	// Git ref untouched: branch-b still points at its original commit.
	bSHAafterSync, err := sh.Scene.Repo.RunGitCommandAndGetOutput("rev-parse", "branch-b")
	require.NoError(t, err)
	require.Equal(t, strings.TrimSpace(bSHAbeforeSync), strings.TrimSpace(bSHAafterSync),
		"--no-restack must not move branch-b's git ref")

	out, err := sh.Scene.Repo.RunGitCommandAndGetOutput("status", "--porcelain")
	require.NoError(t, err)
	require.Empty(t, strings.TrimSpace(out))
}

// TestSquashMergeChildBecomesEmpty covers the case where a squash merge
// subsumes the child's diff (e.g., the merger combined A and B into one
// squash commit on trunk). After cleanup + restack, B is empty against
// trunk but should not be auto-deleted because no PR is marked merged
// for B.
func TestSquashMergeChildBecomesEmpty(t *testing.T) {
	t.Parallel()
	sh := scenario.NewScenario(t, testhelpers.BasicSceneSetup)
	disableCommitSigning(t, sh)

	// A introduces shared=v1; B advances it to v2.
	sh.CreateBranch("branch-a").
		CommitChange("shared", "v1").
		TrackBranch("branch-a", "main")
	sh.CreateBranch("branch-b").
		CommitChange("shared", "v2").
		TrackBranch("branch-b", "branch-a")

	// Squash-merge of A includes B's content too: trunk jumps straight
	// to v2 in one commit. (The reviewer combined the two PRs.)
	mainName := sh.Engine.Trunk().GetName()
	sh.Checkout("main")
	sh.CommitChange("shared", "v2")
	markPrMerged(t, sh, "branch-a", 1, mainName)

	require.NoError(t, sync.Action(sh.Context, sync.Options{Restack: true}, nil))

	branches, err := sh.Scene.Repo.GetLocalBranches()
	require.NoError(t, err)
	require.NotContains(t, branches, "branch-a")
	require.Contains(t, branches, "branch-b", "B has no PR; should survive even if empty")

	require.Equal(t, mainName, sh.Engine.GetBranch("branch-b").GetParent().GetName())

	empty, err := sh.Engine.IsBranchEmpty(sh.Context.Context, "branch-b")
	require.NoError(t, err)
	require.True(t, empty, "B should be empty relative to main after squash subsumed its diff")

	out, err := sh.Scene.Repo.RunGitCommandAndGetOutput("status", "--porcelain")
	require.NoError(t, err)
	require.Empty(t, strings.TrimSpace(out))
}
