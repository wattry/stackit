package engine

import (
	"context"
	"iter"
	"time"

	"stackit.dev/stackit/internal/git"
)

// StackNavigator handles stack relationship queries
type StackNavigator interface {
	AllBranches() []Branch
	CurrentBranch() *Branch
	Trunk() Branch
	GetBranch(branchName string) Branch
	GetChildren(branch Branch) []Branch
	GetRelativeStack(branch Branch, rng StackRange) []Branch
	BranchesDepthFirst(startBranch Branch) iter.Seq2[Branch, int]
	SortBranchesTopologically(branches []Branch) []Branch
	FindBranchForCommit(commitSHA string) (string, error)
}

// BranchStatus provides branch state information
type BranchStatus interface {
	IsTrunk(branch Branch) bool
	IsTracked(branch Branch) bool
	IsUpToDate(branch Branch) bool
	IsMergedIntoTrunk(ctx context.Context, branchName string) (bool, error)
	IsBranchEmpty(ctx context.Context, branchName string) (bool, error)
	GetDeletionStatus(ctx context.Context, branchName string) (DeletionStatus, error)
	GetScope(branch Branch) Scope
	IsLocked(branch Branch) bool
	IsFrozen(branch Branch) bool
	FindMostRecentTrackedAncestors(ctx context.Context, branchName string) ([]string, error)
	GetRemote() string
	GetBranchRemoteDifference(branchName string) (string, error)
}

// BranchInfo provides commit and diff metadata
type BranchInfo interface {
	GetCommitDate(branch Branch) (time.Time, error)
	GetCommitAuthor(branch Branch) (string, error)
	GetRevision(branch Branch) (string, error)
	GetCommitCount(branch Branch) (int, error)
	GetDiffStats(branch Branch) (added int, deleted int, err error)
	GetAllCommits(branch Branch, format CommitFormat) ([]string, error)
	GetParentCommitSHA(commitSHA string) (string, error)
	GetCommitSHA(branchName string, offset int) (string, error)
}

// MetadataStore handles ref-based metadata operations
type MetadataStore interface {
	ListMetadataRefs() (map[string]string, error)
	ReadMetadataRef(branchName string) (*Meta, error)
	BatchReadMetadataRefs(branchNames []string) (map[string]*Meta, map[string]error)
	WriteMetadataRef(branch Branch, meta *Meta) error
	DeleteMetadataRef(branch Branch) error
	RenameMetadataRef(oldBranch, newBranch Branch) error
}

// GitDiffer handles diff and merge operations
type GitDiffer interface {
	GetMergeBase(rev1, rev2 string) (string, error)
	GetChangedFiles(ctx context.Context, base, head string) ([]string, error)
	ShowDiff(ctx context.Context, left, right string, stat bool) (string, error)
	ShowCommits(ctx context.Context, base, head string, patch, stat bool) (string, error)
	IsAncestor(ancestor, descendant string) (bool, error)
}

// WorkingTree handles worktree and staging area operations
type WorkingTree interface {
	HasStagedChanges(ctx context.Context) (bool, error)
	HasUnstagedChanges(ctx context.Context) (bool, error)
	GetPendingChanges(ctx context.Context) ([]PendingChange, error)
	GetUnmergedFiles(ctx context.Context) ([]string, error)
	ParseStagedHunks(ctx context.Context) ([]git.Hunk, error)
	ListWorktrees(ctx context.Context) ([]string, error)
}

// GitRunner provides low-level git command execution
type GitRunner interface {
	RunGitCommandWithContext(ctx context.Context, args ...string) (string, error)
	RunGitCommandRawWithContext(ctx context.Context, args ...string) (string, error)
}

// BranchReader is a composite interface for backward compatibility
// Prefer using the smaller, focused interfaces above for new code
type BranchReader interface {
	StackNavigator
	BranchStatus
	BranchInfo
	MetadataStore
	GitDiffer
	WorkingTree
	GitRunner
}

// BranchTracking handles branch tracking operations
type BranchTracking interface {
	TrackBranch(ctx context.Context, branchName string, parentBranchName string) error
	UntrackBranch(branchName string) error
	SetParent(ctx context.Context, branch Branch, parentBranch Branch) error
	UpdateParentRevision(branchName string, parentRev string) error
	SetScope(branch Branch, scope Scope) error
	SetLocked(branch Branch, locked bool) error
	SetFrozen(branch Branch, frozen bool) error
}

// BranchMutations handles branch lifecycle operations
type BranchMutations interface {
	RenameBranch(ctx context.Context, oldBranch, newBranch Branch) error
	DeleteBranch(ctx context.Context, branch Branch) error
	DeleteBranches(ctx context.Context, branches []Branch) ([]string, error)
	CheckoutBranch(ctx context.Context, branch Branch) error
	CreateAndCheckoutBranch(ctx context.Context, branch Branch) error
}

// CommitOperations handles staging and committing
type CommitOperations interface {
	Commit(ctx context.Context, message string, verbose int, noVerify bool) error
	StageAll(ctx context.Context) error
	StashPush(ctx context.Context, message string) (string, error)
	StashPop(ctx context.Context) error
}

// WorktreeOperations handles worktree management
type WorktreeOperations interface {
	AddWorktree(ctx context.Context, path string, branch string, detach bool) error
	RemoveWorktree(ctx context.Context, path string) error
}

// GitCommandRunner provides low-level git command execution
type GitCommandRunner interface {
	RunGitCommand(args ...string) (string, error)
	RunGitCommandWithEnv(ctx context.Context, env []string, args ...string) (string, error)
}

// Initializer handles repository initialization operations
type Initializer interface {
	Reset(newTrunkName string) error
	Rebuild(newTrunkName string) error
}

// BranchWriter is a composite interface for backward compatibility
// Prefer using the smaller, focused interfaces above for new code
type BranchWriter interface {
	BranchTracking
	BranchMutations
	MetadataStore
	CommitOperations
	WorktreeOperations
	GitCommandRunner
	Initializer
}

// Absorber applies staged hunks to appropriate commits
type Absorber interface {
	ApplyHunksToBranch(ctx context.Context, branch Branch, hunksByCommit map[string][]git.Hunk) error
	FindTargetCommitForHunk(hunk git.Hunk, commitSHAs []string) (string, int, error)
}

// AbsorbManager is deprecated. Use Absorber instead.
type AbsorbManager = Absorber
