// Package git provides a wrapper around git commands and go-git for repository operations.
package git

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

var (
	// goGitMu synchronizes go-git operations that access packfiles to prevent
	// "concurrent map iteration and map write" panics
	goGitMu sync.Mutex
)

// DefaultCommandTimeout is the default timeout for git commands
const DefaultCommandTimeout = 5 * time.Minute

// ErrStaleRemoteInfo indicates that a push failed because the remote has changed
var ErrStaleRemoteInfo = errors.New("stale info")

func (r *runner) UpdateRefWithLog(ctx context.Context, refName, sha, message string) error {
	_, err := r.runGitCommandWithContextInternal(ctx, "update-ref", "-m", message, refName, sha)
	return err
}

func (r *runner) VerifyRef(ctx context.Context, refName string) error {
	_, err := r.runGitCommandWithContextInternal(ctx, "rev-parse", "--verify", refName)
	return err
}

func (r *runner) RunGitCommandWithEnv(ctx context.Context, env []string, args ...string) (string, error) {
	return r.runGitInternal(ctx, "", env, true, args...)
}

func (r *runner) runGitCommandWithContextInternal(ctx context.Context, args ...string) (string, error) {
	return r.runGitInternal(ctx, "", nil, true, args...)
}

func (r *runner) runGitCommandRawWithContextInternal(ctx context.Context, args ...string) (string, error) {
	return r.runGitInternal(ctx, "", nil, false, args...)
}

func (r *runner) runGitCommandWithInputInternal(input string, args ...string) (string, error) {
	return r.runGitInternal(context.Background(), input, nil, true, args...)
}

func (r *runner) runGitInternal(ctx context.Context, input string, env []string, trim bool, args ...string) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, DefaultCommandTimeout)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, "git", args...)
	if r.repoRoot != "" {
		cmd.Dir = r.repoRoot
	}

	if input != "" {
		cmd.Stdin = strings.NewReader(input)
	}

	if len(env) > 0 {
		cmd.Env = append(os.Environ(), env...)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", NewCommandError("git", args, stdout.String(), stderr.String(), err)
	}

	result := stdout.String()
	if trim {
		result = strings.TrimSpace(result)
	}
	return result, nil
}

func (r *runner) RunGHCommandWithContext(ctx context.Context, args ...string) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	// If no timeout/deadline is set in the context, add the default one
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, DefaultCommandTimeout)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, "gh", args...)
	// Use repoRoot for gh commands to ensure they are scoped to the correct repo
	if r.repoRoot != "" {
		cmd.Dir = r.repoRoot
	} else {
		wd, _ := os.Getwd()
		cmd.Dir = wd
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", NewCommandError("gh", args, stdout.String(), stderr.String(), ctx.Err())
		}
		return "", NewCommandError("gh", args, stdout.String(), stderr.String(), err)
	}
	return strings.TrimSpace(stdout.String()), nil
}

func (r *runner) RunGitCommandInteractive(args ...string) error {
	cmd := exec.Command("git", args...)
	if r.repoRoot != "" {
		cmd.Dir = r.repoRoot
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// StagingOptions defines which changes to stage
type StagingOptions struct {
	All    bool
	Update bool
	Patch  bool
}

// MergeOptions contains options for merging branches
type MergeOptions struct {
	FFOnly  bool
	NoEdit  bool
	NoFF    bool
	Message string
}

// Runner defines the interface for git operations used by the engine.
// This allows the engine to be used with both real git and mock implementations.
//
// Runner is a composite interface that embeds smaller, focused interfaces for better
// modularity and testability. Each embedded interface represents a logical grouping
// of related git operations.
type Runner interface {
	// Repository access and configuration
	RepositoryReader
	RepositoryWriter

	// Remote operations
	RemoteOperations

	// Branch operations
	BranchReader
	BranchWriter

	// Commit and revision access
	CommitReader

	// Diff and comparison
	DiffOperations

	// Staging area
	StagingOperations

	// Commit creation
	CommitWriter

	// Advanced git operations
	RebaseOperations
	MergeOperations
	CherryPickOperations
	StashOperations
	ResetOperations
	PathOperations
	PatchOperations

	// Worktree management
	WorktreeOperations

	// Repository status
	StatusOperations

	// Low-level operations
	RefOperations
	ObjectOperations
	MetadataOperations

	// Raw command execution
	RunGitCommandWithEnv(ctx context.Context, env []string, args ...string) (string, error)
	RunGitCommandInteractive(args ...string) error
	RunGHCommandWithContext(ctx context.Context, args ...string) (string, error)
}

// NewRunner returns a standard implementation of Runner that uses the current
// working directory as its repository root.
func NewRunner() Runner {
	return &runner{}
}

// NewRunnerWithPath returns a Runner that operates on a specific repo path.
// This is safe for parallel tests since it doesn't rely on global state.
func NewRunnerWithPath(repoRoot string) Runner {
	abs, err := filepath.Abs(repoRoot)
	if err == nil {
		repoRoot = abs
	}
	return &runner{repoRoot: repoRoot}
}

// runner implements Runner by calling the actual git package functions
type runner struct {
	repo     *Repository
	repoRoot string
	repoMu   sync.Mutex
}

func (r *runner) ensureRepo() (*Repository, error) {
	r.repoMu.Lock()
	defer r.repoMu.Unlock()

	if r.repo != nil {
		return r.repo, nil
	}

	path := r.repoRoot
	if path == "" {
		wd, _ := os.Getwd()
		path = wd
	}

	repo, err := OpenRepository(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository at %s: %w", path, err)
	}

	// Discover and cache the actual root path
	wt, err := repo.Worktree()
	if err == nil {
		r.repoRoot = wt.Filesystem.Root()
	}

	r.repo = repo
	return repo, nil
}

func (r *runner) InitDefaultRepo() error {
	_, err := r.ensureRepo()
	return err
}

func (r *runner) GetRemote() string {
	repo, err := r.ensureRepo()
	if err != nil {
		return DefaultRemote
	}
	return r.getRemote(repo)
}

func (r *runner) FetchRemoteShas(remote string) (map[string]string, error) {
	repo, err := r.ensureRepo()
	if err != nil {
		return nil, err
	}
	return r.fetchRemoteShas(repo, remote)
}

func (r *runner) GetRemoteSha(remote, branchName string) (string, error) {
	repo, err := r.ensureRepo()
	if err != nil {
		return "", err
	}
	return r.getRemoteSha(repo, remote, branchName)
}

func (r *runner) GetConfig(key string) (string, error) {
	return r.runGitCommandInternal("config", "--get", key)
}

func (r *runner) SetConfig(key, value string) error {
	_, err := r.runGitCommandInternal("config", key, value)
	return err
}

func (r *runner) GetConfigAll(key string) ([]string, error) {
	output, err := r.runGitCommandInternal("config", "--get-all", key)
	if err == nil {
		if output == "" {
			return []string{}, nil
		}
		return strings.Split(output, "\n"), nil
	}
	return []string{}, nil
}

func (r *runner) AddConfigValue(key, value string) error {
	_, err := r.runGitCommandInternal("config", "--add", key, value)
	return err
}

func (r *runner) IsInsideRepo() bool {
	_, err := r.runGitCommandInternal("rev-parse", "--is-inside-work-tree")
	return err == nil
}

func (r *runner) DiscoverRepoRoot() (string, error) {
	return r.runGitCommandInternal("rev-parse", "--show-toplevel")
}

func (r *runner) GetRepoRoot() string {
	return r.repoRoot
}

func (r *runner) GetUserName(ctx context.Context) (string, error) {
	return r.runGitCommandWithContextInternal(ctx, "config", "user.name")
}

func (r *runner) EnsureMetadataRefspecConfigured() error {
	const metadataRefspec = "+refs/stackit/metadata/*:refs/stackit/remote-metadata/*"

	refspecs, err := r.GetConfigAll("remote.origin.fetch")
	if err != nil {
		return fmt.Errorf("failed to get fetch refspecs: %w", err)
	}

	// Check if already configured
	for _, rs := range refspecs {
		if rs == metadataRefspec {
			return nil // Already configured
		}
	}

	// Add refspec for metadata refs
	return r.AddConfigValue("remote.origin.fetch", metadataRefspec)
}

func (r *runner) GetCurrentBranch() (string, error) {
	repo, err := r.ensureRepo()
	if err != nil {
		return "", err
	}
	return repo.GetCurrentBranch()
}

func (r *runner) GetAllBranchNames() ([]string, error) {
	repo, err := r.ensureRepo()
	if err != nil {
		return nil, err
	}
	return repo.GetBranchNames()
}

func (r *runner) FindRemoteBranch(ctx context.Context, remote string) (string, error) {
	// Get all branch configs that have this remote
	// Format: "branch.<name>.remote <remote>"
	output, err := r.runGitCommandWithContextInternal(ctx, "config", "--get-regexp", "^branch\\..*\\.remote$")
	if err != nil {
		return "", nil //nolint:nilerr // git config returns 1 if no branches match
	}

	if output == "" {
		return "", nil
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		// Line format: "branch.<name>.remote <remote>"
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			remoteValue := parts[1]
			if remoteValue == remote {
				// Extract branch name from "branch.<name>.remote"
				branchPart := parts[0]
				if strings.HasPrefix(branchPart, "branch.") && strings.HasSuffix(branchPart, ".remote") {
					// Remove "branch." prefix and ".remote" suffix
					branchName := branchPart[7 : len(branchPart)-7]
					return branchName, nil
				}
			}
		}
	}
	return "", nil
}

func (r *runner) CheckoutBranch(ctx context.Context, branchName string) error {
	_, err := r.runGitCommandWithContextInternal(ctx, "checkout", branchName)
	if err != nil {
		return fmt.Errorf("failed to checkout branch %s: %w", branchName, err)
	}
	return nil
}

func (r *runner) CheckoutBranchForce(ctx context.Context, branchName string) error {
	_, err := r.runGitCommandWithContextInternal(ctx, "checkout", "-f", branchName)
	return err
}

func (r *runner) CreateAndCheckoutBranch(ctx context.Context, branchName string) error {
	_, err := r.runGitCommandWithContextInternal(ctx, "checkout", "-b", branchName)
	if err != nil {
		return fmt.Errorf("failed to create and checkout branch %s: %w", branchName, err)
	}
	return nil
}

func (r *runner) DeleteBranch(ctx context.Context, branchName string) error {
	_, err := r.runGitCommandWithContextInternal(ctx, "branch", "-D", branchName)
	if err != nil {
		return fmt.Errorf("failed to delete branch %s: %w", branchName, err)
	}
	return nil
}

func (r *runner) RenameBranch(ctx context.Context, oldName, newName string) error {
	_, err := r.runGitCommandWithContextInternal(ctx, "branch", "-m", oldName, newName)
	if err != nil {
		return fmt.Errorf("failed to rename branch %s to %s: %w", oldName, newName, err)
	}
	return nil
}

func (r *runner) CheckoutDetached(ctx context.Context, revision string) error {
	_, err := r.runGitCommandWithContextInternal(ctx, "checkout", "--detach", revision)
	if err != nil {
		return fmt.Errorf("failed to checkout %s in detached state: %w", revision, err)
	}
	return nil
}

func (r *runner) UpdateBranchRef(branchName, revision string) error {
	_, err := r.runGitCommandWithContextInternal(context.Background(), "update-ref", "refs/heads/"+branchName, revision)
	if err != nil {
		return fmt.Errorf("failed to update branch ref: %w", err)
	}
	return nil
}

func (r *runner) GetRemoteRevision(branchName string) (string, error) {
	repo, err := r.ensureRepo()
	if err != nil {
		return "", err
	}
	return r.getRemoteRevision(repo, branchName)
}

func (r *runner) GetCurrentBranchOrSHA(ctx context.Context) (string, error) {
	branch, err := r.runGitCommandWithContextInternal(ctx, "rev-parse", "--abbrev-ref", "HEAD")
	if err == nil && branch != "HEAD" {
		return branch, nil
	}
	return r.GetCurrentRevision(ctx)
}

func (r *runner) GetCurrentRevision(ctx context.Context) (string, error) {
	return r.runGitCommandWithContextInternal(ctx, "rev-parse", "HEAD")
}

func (r *runner) GetRevision(branchName string) (string, error) {
	repo, err := r.ensureRepo()
	if err != nil {
		return "", err
	}
	return r.getRevision(repo, branchName)
}

func (r *runner) BatchGetRevisions(branchNames []string) (map[string]string, []error) {
	repo, err := r.ensureRepo()
	if err != nil {
		var errors []error
		for i := 0; i < len(branchNames); i++ {
			errors = append(errors, fmt.Errorf("failed to get repository: %w", err))
		}
		return nil, errors
	}
	return r.batchGetRevisions(repo, branchNames)
}

func (r *runner) GetMergeBase(rev1, rev2 string) (string, error) {
	repo, err := r.ensureRepo()
	if err != nil {
		return "", err
	}
	return r.getMergeBase(repo, rev1, rev2)
}

func (r *runner) GetMergeBaseByRef(ref1, ref2 string) (string, error) {
	repo, err := r.ensureRepo()
	if err != nil {
		return "", err
	}
	return r.getMergeBaseByRef(repo, ref1, ref2)
}

func (r *runner) IsAncestor(ancestor, descendant string) (bool, error) {
	repo, err := r.ensureRepo()
	if err != nil {
		return false, err
	}
	return r.isAncestor(repo, ancestor, descendant)
}

func (r *runner) GetCommitDate(branchName string) (time.Time, error) {
	repo, err := r.ensureRepo()
	if err != nil {
		return time.Time{}, err
	}
	return r.getCommitDate(repo, branchName)
}

func (r *runner) GetCommitAuthor(branchName string) (string, error) {
	repo, err := r.ensureRepo()
	if err != nil {
		return "", err
	}
	return r.getCommitAuthor(repo, branchName)
}

func (r *runner) GetCommitRange(base, head, format string) ([]string, error) {
	repo, err := r.ensureRepo()
	if err != nil {
		return nil, err
	}

	// Synchronize go-git operations to prevent concurrent packfile access
	goGitMu.Lock()
	defer goGitMu.Unlock()

	headHash, err := resolveRefHashInternal(repo, head)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve head: %w", err)
	}

	var baseHash plumbing.Hash
	if base != "" {
		baseHash, err = resolveRefHashInternal(repo, base)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve base: %w", err)
		}
	}

	var commits []*object.Commit
	commits, err = iterateCommitsNoLock(repo, headHash, baseHash)
	if err != nil {
		return nil, fmt.Errorf("failed to iterate commits: %w", err)
	}

	result := make([]string, 0, len(commits))
	for _, commit := range commits {
		var formatted string
		switch format {
		case "SHA":
			formatted = commit.Hash.String()
		case "READABLE":
			// Oneline format: short SHA + subject
			shortHash := commit.Hash.String()[:7]
			subject := strings.Split(strings.TrimSpace(commit.Message), "\n")[0]
			formatted = fmt.Sprintf("%s - %s", shortHash, subject)
		case "MESSAGE":
			formatted = strings.TrimSpace(commit.Message)
		case "SUBJECT":
			subject := strings.Split(strings.TrimSpace(commit.Message), "\n")[0]
			formatted = strings.TrimSpace(subject)
		default:
			return nil, fmt.Errorf("unknown commit format: %s", format)
		}

		if formatted != "" {
			result = append(result, formatted)
		}
	}

	return result, nil
}

func (r *runner) GetCommitRangeSHAs(base, head string) ([]string, error) {
	return r.GetCommitRange(base, head, "SHA")
}

func (r *runner) GetCommitHistorySHAs(branchName string) ([]string, error) {
	return r.GetCommitRangeSHAs("", branchName)
}

func (r *runner) GetCommitSHA(branchName string, offset int) (string, error) {
	if offset < 0 {
		return "", fmt.Errorf("offset must be non-negative")
	}

	repo, err := r.ensureRepo()
	if err != nil {
		return "", err
	}

	// Synchronize go-git operations to prevent concurrent packfile access
	goGitMu.Lock()
	defer goGitMu.Unlock()

	// Resolve branch reference
	hash, err := resolveRefHashInternal(repo, branchName)
	if err != nil {
		return "", fmt.Errorf("failed to get branch reference: %w", err)
	}

	commit, err := repo.CommitObject(hash)
	if err != nil {
		return "", fmt.Errorf("failed to get commit: %w", err)
	}

	// Walk back offset number of commits
	for i := 0; i < offset; i++ {
		if commit.NumParents() == 0 {
			return "", fmt.Errorf("commit has no parent at offset %d", i)
		}
		// Get first parent
		commit, err = commit.Parent(0)
		if err != nil {
			return "", fmt.Errorf("failed to get parent commit: %w", err)
		}
	}

	return commit.Hash.String(), nil
}

func (r *runner) PullBranch(ctx context.Context, remote, branchName string) (PullResult, error) {
	// Save current branch/detached HEAD
	currentBranch, err := r.GetCurrentBranch()
	var currentRev string
	if err != nil {
		currentBranch = ""
		currentRev, _ = r.runGitCommandWithContextInternal(ctx, "rev-parse", "HEAD")
	}

	// Get the SHA of the local branch
	oldRev, err := r.runGitCommandWithContextInternal(ctx, "rev-parse", branchName)
	if err != nil {
		return PullConflict, fmt.Errorf("failed to get local revision for %s: %w", branchName, err)
	}

	// Fetch first
	_, _ = r.runGitCommandWithContextInternal(ctx, "fetch", remote, branchName)

	// Get the SHA of the remote branch
	remoteRev, err := r.runGitCommandWithContextInternal(ctx, "rev-parse", fmt.Sprintf("%s/%s", remote, branchName))
	if err != nil {
		// If we can't get remote rev, we can't pull, but it might just be because there's no remote
		return PullUnneeded, nil //nolint:nilerr
	}

	if oldRev == remoteRev {
		return PullUnneeded, nil
	}

	// Check if it's a fast-forward
	isAncestor, err := r.IsAncestor(oldRev, remoteRev)
	if err != nil || !isAncestor {
		return PullConflict, nil //nolint:nilerr
	}

	// Update the local branch reference to the remote commit (fast-forward)
	_, err = r.runGitCommandWithContextInternal(ctx, "update-ref", "refs/heads/"+branchName, remoteRev)
	if err != nil {
		return PullConflict, fmt.Errorf("failed to update local branch %s to %s: %w", branchName, remoteRev, err)
	}

	// If we are currently ON this branch in this worktree, we need to update HEAD
	if currentBranch == branchName {
		_ = r.CheckoutBranch(ctx, branchName)
	} else if currentRev != "" {
		_ = r.CheckoutDetached(ctx, currentRev)
	}

	return PullDone, nil
}

func (r *runner) PushBranch(ctx context.Context, branchName, remote string, opts PushOptions) error {
	args := []string{"push", "-u", remote}

	if opts.Force {
		args = append(args, "--force")
	} else if opts.ForceWithLease {
		args = append(args, "--force-with-lease")
	}

	if opts.NoVerify {
		args = append(args, "--no-verify")
	}

	args = append(args, branchName)

	_, err := r.runGitCommandWithContextInternal(ctx, args...)
	if err != nil {
		if strings.Contains(err.Error(), "stale info") || strings.Contains(err.Error(), "forced update") {
			return fmt.Errorf("%w: force-with-lease push of %s failed due to external changes to the remote branch", ErrStaleRemoteInfo, branchName)
		}
		return fmt.Errorf("failed to push branch %s: %w", branchName, err)
	}

	return nil
}

func (r *runner) Rebase(ctx context.Context, branchName, upstream, oldUpstream string) (RebaseResult, error) {
	// Use detached HEAD to avoid "already used by worktree" errors
	// git rebase --onto <upstream> <oldUpstream> <branchName>
	_, err := r.runGitCommandWithContextInternal(ctx, "rebase", "--onto", upstream, oldUpstream, branchName)
	if err != nil {
		if r.IsRebaseInProgress(ctx) {
			return RebaseConflict, nil
		}
		// Abort rebase if it failed for other reasons
		_, _ = r.runGitCommandWithContextInternal(ctx, "rebase", "--abort")

		return RebaseConflict, nil
	}

	return RebaseDone, nil
}

func (r *runner) RebaseContinueNoEdit(ctx context.Context) (RebaseResult, error) {
	_, err := r.RunGitCommandWithEnv(ctx, []string{"GIT_EDITOR=true"}, "rebase", "--continue")
	if err != nil {
		if strings.Contains(err.Error(), "conflict") || strings.Contains(err.Error(), "patch failed") {
			return RebaseConflict, nil
		}
		return RebaseConflict, err
	}
	return RebaseDone, nil
}

func (r *runner) RebaseContinue(ctx context.Context) (RebaseResult, error) {
	_, err := r.RunGitCommandWithEnv(ctx, []string{"GIT_EDITOR=true"}, "rebase", "--continue")
	if err != nil {
		if r.IsRebaseInProgress(ctx) {
			return RebaseConflict, nil
		}
		return RebaseConflict, fmt.Errorf("rebase continue failed: %w", err)
	}

	return RebaseDone, nil
}

func (r *runner) RebaseAbort(ctx context.Context) error {
	_, err := r.runGitCommandWithContextInternal(ctx, "rebase", "--abort")
	if err != nil {
		return fmt.Errorf("rebase abort failed: %w", err)
	}
	return nil
}

func (r *runner) InteractiveRebase(_ context.Context, onto string) error {
	return r.RunGitCommandInteractive("rebase", "-i", onto)
}

func (r *runner) IsMergeInProgress(_ context.Context) bool {
	if r.repoRoot == "" {
		return false
	}
	mergeHead := filepath.Join(r.repoRoot, ".git", "MERGE_HEAD")
	if _, err := os.Stat(mergeHead); err == nil {
		return true
	}
	return false
}

func (r *runner) MergeAbort(ctx context.Context) error {
	_, err := r.runGitCommandWithContextInternal(ctx, "merge", "--abort")
	if err != nil {
		return fmt.Errorf("merge abort failed: %w", err)
	}
	return nil
}

func (r *runner) CherryPick(ctx context.Context, commitSHA, onto string) (string, error) {
	if _, err := r.runGitCommandWithContextInternal(ctx, "checkout", "--detach", onto); err != nil {
		return "", fmt.Errorf("failed to checkout %s: %w", onto, err)
	}

	if _, err := r.runGitCommandWithContextInternal(ctx, "cherry-pick", commitSHA); err != nil {
		_, _ = r.runGitCommandWithContextInternal(ctx, "cherry-pick", "--abort")
		return "", fmt.Errorf("failed to cherry-pick %s: %w", commitSHA, err)
	}

	newSHA, err := r.runGitCommandWithContextInternal(ctx, "rev-parse", "HEAD")
	if err != nil {
		return "", fmt.Errorf("failed to get new SHA after cherry-pick: %w", err)
	}

	return strings.TrimSpace(newSHA), nil
}

func (r *runner) StashPush(ctx context.Context, message string) (string, error) {
	args := []string{"stash", "push", "-u"}
	if message != "" {
		args = append(args, "-m", message)
	}
	output, err := r.runGitCommandWithContextInternal(ctx, args...)
	if err != nil {
		return "", fmt.Errorf("stash push failed: %w", err)
	}
	return output, nil
}

func (r *runner) StashPop(ctx context.Context) error {
	_, err := r.runGitCommandWithContextInternal(ctx, "stash", "pop")
	if err != nil {
		return fmt.Errorf("stash pop failed: %w", err)
	}
	return nil
}

func (r *runner) Fetch(ctx context.Context, remote, branch string) error {
	_, err := r.runGitCommandWithContextInternal(ctx, "fetch", remote, branch)
	if err != nil {
		return fmt.Errorf("failed to fetch %s from %s: %w", branch, remote, err)
	}
	return nil
}

func (r *runner) CreateBranch(ctx context.Context, branchName, startPoint string) error {
	_, err := r.runGitCommandWithContextInternal(ctx, "branch", branchName, startPoint)
	if err != nil {
		return fmt.Errorf("failed to create branch %s from %s: %w", branchName, startPoint, err)
	}
	return nil
}

func (r *runner) CreateBranchForce(ctx context.Context, branchName, revision string) error {
	_, err := r.runGitCommandWithContextInternal(ctx, "branch", "-f", branchName, revision)
	return err
}

func (r *runner) Merge(ctx context.Context, branchName string, opts MergeOptions) error {
	args := []string{"merge"}
	if opts.FFOnly {
		args = append(args, "--ff-only")
	}
	if opts.NoEdit {
		args = append(args, "--no-edit")
	}
	if opts.NoFF {
		args = append(args, "--no-ff")
	}
	if opts.Message != "" {
		args = append(args, "-m", opts.Message)
	}
	args = append(args, branchName)

	_, err := r.runGitCommandWithContextInternal(ctx, args...)
	if err != nil {
		return fmt.Errorf("failed to merge %s: %w", branchName, err)
	}
	return nil
}

func (r *runner) CheckoutPaths(ctx context.Context, branch string, paths []string) error {
	args := []string{"checkout", branch, "--"}
	args = append(args, paths...)
	_, err := r.runGitCommandWithContextInternal(ctx, args...)
	if err != nil {
		return fmt.Errorf("failed to checkout paths from %s: %w", branch, err)
	}
	return nil
}

func (r *runner) RemovePaths(ctx context.Context, paths []string) error {
	args := []string{"rm"}
	args = append(args, paths...)
	_, err := r.runGitCommandWithContextInternal(ctx, args...)
	if err != nil {
		return fmt.Errorf("failed to remove paths: %w", err)
	}
	return nil
}

func (r *runner) ResetMerge(ctx context.Context, revision string) error {
	_, err := r.runGitCommandWithContextInternal(ctx, "reset", "--merge", revision)
	if err != nil {
		return fmt.Errorf("failed to reset --merge to %s: %w", revision, err)
	}
	return nil
}

func (r *runner) HardReset(ctx context.Context, revision string) error {
	_, err := r.runGitCommandWithContextInternal(ctx, "reset", "--hard", revision)
	if err != nil {
		return fmt.Errorf("failed to hard reset to %s: %w", revision, err)
	}
	return nil
}

func (r *runner) SoftReset(ctx context.Context, revision string) error {
	_, err := r.runGitCommandWithContextInternal(ctx, "reset", "-q", "--soft", revision)
	if err != nil {
		return fmt.Errorf("failed to soft reset to %s: %w", revision, err)
	}
	return nil
}

func (r *runner) MixedReset(ctx context.Context, revision string) error {
	_, err := r.runGitCommandWithContextInternal(ctx, "reset", revision)
	return err
}

func (r *runner) ListStash(ctx context.Context) (string, error) {
	return r.runGitCommandWithContextInternal(ctx, "stash", "list")
}

func (r *runner) GetReflog(ctx context.Context, count int, format string) (string, error) {
	args := []string{"reflog", fmt.Sprintf("-%d", count)}
	if format != "" {
		args = append(args, fmt.Sprintf("--format=%s", format))
	}
	return r.runGitCommandWithContextInternal(ctx, args...)
}

func (r *runner) CommitWithOptions(opts CommitOptions) error {
	args := []string{"commit"}

	if opts.Amend {
		args = append(args, "--amend")
	}

	if opts.NoVerify {
		args = append(args, "--no-verify")
	}

	if opts.ResetAuthor {
		args = append(args, "--reset-author")
	}

	if opts.Verbose > 0 {
		args = append(args, "-v")
	}

	if opts.Message != "" {
		args = append(args, "-m", opts.Message)
	}

	if opts.NoEdit {
		args = append(args, "--no-edit")
	} else if opts.Edit {
		// Only add -e if explicitly requested (git opens editor by default if no message)
		args = append(args, "-e")
	}
	// If neither NoEdit nor Edit is set, and no message is provided,
	// git will open the editor by default (no flag needed)

	return r.RunGitCommandInteractive(args...)
}

func (r *runner) Commit(message string, verbose int, noVerify bool) error {
	return r.CommitWithOptions(CommitOptions{
		Message:  message,
		Verbose:  verbose,
		NoVerify: noVerify,
	})
}

func (r *runner) StageAll(ctx context.Context) error {
	_, err := r.runGitCommandWithContextInternal(ctx, "add", "-A")
	if err != nil {
		return fmt.Errorf("failed to stage all changes: %w", err)
	}
	return nil
}

func (r *runner) StagePatch(_ context.Context) error {
	return r.RunGitCommandInteractive("add", "-p")
}

func (r *runner) StageTracked(ctx context.Context) error {
	_, err := r.runGitCommandWithContextInternal(ctx, "add", "-u")
	if err != nil {
		return fmt.Errorf("failed to stage tracked changes: %w", err)
	}
	return nil
}

func (r *runner) AddAll(ctx context.Context) error {
	return r.StageAll(ctx)
}

func (r *runner) StageChanges(ctx context.Context, opts StagingOptions) error {
	if opts.Patch && !opts.All {
		return r.RunGitCommandInteractive("add", "-p")
	}

	if opts.All {
		return r.StageAll(ctx)
	}

	if opts.Update {
		_, err := r.runGitCommandWithContextInternal(ctx, "add", "-u")
		return err
	}

	return nil
}

func (r *runner) GetRepoInfo(ctx context.Context) (string, string, error) {
	// Get remote URL
	url, _ := r.runGitCommandWithContextInternal(ctx, "config", "--get", "remote.origin.url")
	// url will be empty if there's an error (e.g. remote.origin.url not set)
	// This happens in many tests and is not a fatal error for most operations.
	if url == "" {
		return "", "", nil
	}

	// Parse URL (handles both https and ssh formats)
	url = strings.TrimSpace(url)
	if url == "" {
		return "", "", nil
	}
	url = strings.TrimSuffix(url, ".git")
	parts := strings.Split(url, "/")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("invalid remote URL")
	}

	repoName := parts[len(parts)-1]
	var owner string
	if strings.Contains(url, "@") {
		// SSH format: git@github.com:owner/repo
		sshParts := strings.Split(url, ":")
		if len(sshParts) < 2 {
			return "", "", fmt.Errorf("invalid SSH remote URL")
		}
		pathParts := strings.Split(sshParts[1], "/")
		if len(pathParts) < 2 {
			return "", "", fmt.Errorf("invalid SSH remote URL")
		}
		owner = pathParts[0]
	} else {
		// HTTPS format: https://github.com/owner/repo
		owner = parts[len(parts)-2]
	}

	return owner, repoName, nil
}

func (r *runner) HasStagedChanges(ctx context.Context) (bool, error) {
	output, err := r.runGitCommandWithContextInternal(ctx, "diff", "--cached", "--shortstat")
	if err != nil {
		return false, fmt.Errorf("failed to check staged changes: %w", err)
	}
	return strings.TrimSpace(output) != "", nil
}

func (r *runner) HasUnstagedChanges(ctx context.Context) (bool, error) {
	// Use git diff to check for unstaged changes to tracked files
	// This is more reliable than parsing porcelain output which gets trimmed
	output, err := r.runGitCommandWithContextInternal(ctx, "diff", "--name-only")
	if err != nil {
		return false, fmt.Errorf("failed to check unstaged changes: %w", err)
	}
	return strings.TrimSpace(output) != "", nil
}

func (r *runner) HasUntrackedFiles(ctx context.Context) (bool, error) {
	output, err := r.runGitCommandWithContextInternal(ctx, "ls-files", "--others", "--exclude-standard")
	if err != nil {
		return false, fmt.Errorf("failed to check for untracked files: %w", err)
	}
	return strings.TrimSpace(output) != "", nil
}

func (r *runner) IsMerged(ctx context.Context, branchName, target string) (bool, error) {
	// Get merge base
	mergeBase, err := r.GetMergeBase(branchName, target)
	if err != nil {
		return false, fmt.Errorf("failed to get merge base: %w", err)
	}

	// Get branch revision
	branchRev, err := r.GetRevision(branchName)
	if err != nil {
		return false, fmt.Errorf("failed to get branch revision: %w", err)
	}

	// If merge base equals branch revision, branch is already merged
	if mergeBase == branchRev {
		return true, nil
	}

	// Use git cherry to check if all commits are in trunk
	// git cherry <trunk> <branch> returns commits that are in branch but not in trunk
	// If empty, all commits are merged
	cherryOutput, err := r.runGitCommandWithContextInternal(ctx, "cherry", target, branchName)
	if err != nil {
		// If cherry fails, fall back to simpler check
		// Check if branch tip is reachable from trunk
		return r.IsAncestor(branchRev, target)
	}

	// If cherry output is empty or all lines start with '-', branch is merged
	if cherryOutput == "" {
		return true, nil
	}

	// Check if all commits are marked as merged (lines starting with '-')
	lines := strings.Split(strings.TrimSpace(cherryOutput), "\n")
	for _, line := range lines {
		if line != "" && line[0] != '-' {
			return false, nil
		}
	}

	return true, nil
}

func (r *runner) GetMergedBranches(ctx context.Context, target string) (map[string]bool, error) {
	out, err := r.runGitCommandWithContextInternal(ctx, "branch", "--merged", target)
	if err != nil {
		return nil, fmt.Errorf("failed to get merged branches: %w", err)
	}

	merged := make(map[string]bool)
	lines := strings.Split(out, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Remove current branch indicator '*' if present
		line = strings.TrimPrefix(line, "*")
		line = strings.TrimSpace(line)
		merged[line] = true
	}
	return merged, nil
}

func (r *runner) IsDiffEmpty(ctx context.Context, branchName, base string) (bool, error) {
	branchRev, err := r.GetRevision(branchName)
	if err != nil {
		return false, fmt.Errorf("failed to get branch revision: %w", err)
	}

	if branchRev == base {
		return true, nil
	}

	_, err = r.runGitCommandWithContextInternal(ctx, "diff", "--quiet", base, branchRev)
	return err == nil, nil
}

func (r *runner) GetChangedFiles(ctx context.Context, base, head string) ([]string, error) {
	output, err := r.runGitCommandWithContextInternal(ctx, "diff", "--name-only", base, head)
	if err != nil {
		return nil, fmt.Errorf("failed to get changed files: %w", err)
	}
	if output == "" {
		return []string{}, nil
	}
	return strings.Split(strings.TrimSpace(output), "\n"), nil
}

func (r *runner) ShowDiff(ctx context.Context, left, right string, stat bool) (string, error) {
	args := []string{"-c", "color.ui=always", "--no-pager", "diff", "--no-ext-diff"}
	if stat {
		args = append(args, "--stat")
	}
	args = append(args, left, right, "--")
	return r.runGitCommandWithContextInternal(ctx, args...)
}

func (r *runner) ShowCommits(ctx context.Context, base, head string, patch, stat bool) (string, error) {
	args := []string{"-c", "color.ui=always", "--no-pager", "log"}
	switch {
	case patch && stat:
		args = append(args, "--stat")
	case patch:
		args = append(args, "-p")
	default:
		args = append(args, "--pretty=format:%h - %s")
	}

	// If base is empty, use head~ (parent commit) for trunk
	baseRef := base
	if base == "" {
		baseRef = head + "~"
	}
	args = append(args, fmt.Sprintf("%s..%s", baseRef, head))
	args = append(args, "--")
	return r.runGitCommandWithContextInternal(ctx, args...)
}

func (r *runner) GetUnmergedFiles(ctx context.Context) ([]string, error) {
	output, err := r.runGitCommandWithContextInternal(ctx, "diff", "--name-only", "--diff-filter=U")
	if err != nil {
		return []string{}, nil //nolint:nilerr
	}
	if output == "" {
		return []string{}, nil
	}
	return strings.Split(strings.TrimSpace(output), "\n"), nil
}

func (r *runner) GetStagedDiff(ctx context.Context, files ...string) (string, error) {
	args := []string{"diff", "--cached"}
	if len(files) > 0 {
		args = append(args, "--")
		args = append(args, files...)
	}
	return r.runGitCommandRawWithContextInternal(ctx, args...)
}

func (r *runner) GetUnstagedDiff(ctx context.Context, files ...string) (string, error) {
	args := []string{"diff"}
	if len(files) > 0 {
		args = append(args, "--")
		args = append(args, files...)
	}
	return r.runGitCommandRawWithContextInternal(ctx, args...)
}

func (r *runner) GetDiffNumstat(base, head string) (string, error) {
	return r.runGitCommandInternal("diff", "--numstat", base, head)
}

func (r *runner) GetCommitLog(sha, format string) (string, error) {
	return r.runGitCommandInternal("log", "-1", "--format="+format, sha)
}

func (r *runner) GetStatusPorcelain(ctx context.Context) (string, error) {
	return r.runGitCommandWithContextInternal(ctx, "status", "--porcelain")
}

func (r *runner) GetCommitTemplate(ctx context.Context) (string, error) {
	status, err := r.runGitCommandWithContextInternal(ctx, "status")
	if err != nil {
		return "", fmt.Errorf("failed to get git status: %w", err)
	}

	lines := strings.Split(status, "\n")
	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString("# Please enter the commit message for your changes. Lines starting\n")
	sb.WriteString("# with '#' will be ignored, and an empty message aborts the commit.\n")
	sb.WriteString("#\n")
	for _, line := range lines {
		sb.WriteString("# ")
		sb.WriteString(line)
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

func (r *runner) ParseStagedHunks(ctx context.Context) ([]Hunk, error) {
	diffOutput, err := r.runGitCommandRawWithContextInternal(ctx, "diff", "--cached")
	if err != nil {
		return nil, fmt.Errorf("failed to get staged diff: %w", err)
	}

	return ParseDiffOutput(diffOutput)
}

func (r *runner) AddWorktree(ctx context.Context, path string, branch string, detach bool) error {
	args := []string{"worktree", "add"}
	if detach {
		args = append(args, "--detach")
	}
	args = append(args, path)
	if branch != "" {
		args = append(args, branch)
	}

	_, err := r.runGitCommandWithContextInternal(ctx, args...)
	if err != nil {
		return fmt.Errorf("failed to add worktree at %s: %w", path, err)
	}
	return nil
}

func (r *runner) RemoveWorktree(ctx context.Context, path string) error {
	_, err := r.runGitCommandWithContextInternal(ctx, "worktree", "remove", "--force", path)
	if err != nil {
		return fmt.Errorf("failed to remove worktree at %s: %w", path, err)
	}
	return nil
}

func (r *runner) ListWorktrees(ctx context.Context) ([]string, error) {
	output, err := r.runGitCommandWithContextInternal(ctx, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, fmt.Errorf("failed to list worktrees: %w", err)
	}

	if output == "" {
		return []string{}, nil
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	var worktrees []string
	for _, line := range lines {
		if len(line) > 9 && line[:9] == "worktree " {
			worktrees = append(worktrees, line[9:])
		}
	}
	return worktrees, nil
}

func (r *runner) runGitCommandInternal(args ...string) (string, error) {
	return r.runGitCommandWithContextInternal(context.Background(), args...)
}

func (r *runner) GetRef(name string) (string, error) {
	repo, err := r.ensureRepo()
	if err != nil {
		return "", err
	}
	return r.getRef(repo, name)
}

func (r *runner) UpdateRef(name, sha string) error {
	_, err := r.runGitCommandInternal("update-ref", name, sha)
	return err
}

func (r *runner) DeleteRef(name string) error {
	_, err := r.runGitCommandInternal("update-ref", "-d", name)
	return err
}

func (r *runner) CatFile(sha string) (string, error) {
	return r.runGitCommandInternal("cat-file", "-p", sha)
}

func (r *runner) CreateBlob(content string) (string, error) {
	return r.runGitCommandWithInputInternal(content, "hash-object", "-w", "--stdin")
}

func (r *runner) ReadBlob(sha string) (string, error) {
	repo, err := r.ensureRepo()
	if err != nil {
		return "", err
	}
	return r.readBlob(repo, sha)
}

func (r *runner) ListRefs(prefix string) (map[string]string, error) {
	repo, err := r.ensureRepo()
	if err != nil {
		return nil, err
	}
	return r.listRefs(repo, prefix)
}

func (r *runner) PushMetadataRefs(branches []string) error {
	if len(branches) == 0 {
		return nil
	}
	// git push origin +refs/stackit/metadata/branch1 +refs/stackit/metadata/branch2 ...
	// We use the '+' prefix to force the push because metadata refs point to blobs,
	// and updates to non-commit objects are always considered non-fast-forward by Git.
	args := []string{"push", "origin"}
	for _, branch := range branches {
		args = append(args, fmt.Sprintf("+refs/stackit/metadata/%s", branch))
	}
	_, err := r.runGitCommandInternal(args...)
	return err
}

func (r *runner) FetchMetadataRefs() error {
	// git fetch origin 'refs/stackit/metadata/*:refs/stackit/remote-metadata/*'
	_, err := r.runGitCommandInternal("fetch", "origin", "+refs/stackit/metadata/*:refs/stackit/remote-metadata/*")
	return err
}

func (r *runner) DeleteRemoteMetadataRef(branch string) error {
	// git push origin --delete refs/stackit/metadata/<branch>
	_, err := r.runGitCommandInternal("push", "origin", "--delete", fmt.Sprintf("refs/stackit/metadata/%s", branch))
	return err
}

func (r *runner) BatchDeleteRemoteMetadataRefs(branches []string) error {
	if len(branches) == 0 {
		return nil
	}
	// git push origin --delete refs/stackit/metadata/branch1 refs/stackit/metadata/branch2 ...
	args := []string{"push", "origin", "--delete"}
	for _, branch := range branches {
		args = append(args, fmt.Sprintf("refs/stackit/metadata/%s", branch))
	}
	_, err := r.runGitCommandInternal(args...)
	return err
}

func (r *runner) TestRemoteRefCompatibility() error {
	testRef := "refs/stackit/metadata/stackit-compat-test"
	testContent := fmt.Sprintf(`{"test":true,"timestamp":%d}`, time.Now().Unix())

	// Create test blob
	sha, err := r.CreateBlob(testContent)
	if err != nil {
		return fmt.Errorf("failed to create test blob: %w", err)
	}

	// Try to push test ref
	if err := r.UpdateRef(testRef, sha); err != nil {
		return fmt.Errorf("failed to update local test ref: %w", err)
	}

	if _, err := r.runGitCommandInternal("push", "origin", "+"+testRef); err != nil {
		_ = r.DeleteRef(testRef) // Cleanup local
		return fmt.Errorf("remote rejected metadata ref push: %w", err)
	}

	// Cleanup: delete remote and local test ref
	_, _ = r.runGitCommandInternal("push", "origin", "--delete", testRef)
	_ = r.DeleteRef(testRef)

	return nil
}

func (r *runner) GetParentCommitSHA(commitSHA string) (string, error) {
	repo, err := r.ensureRepo()
	if err != nil {
		return "", err
	}

	hash, err := resolveRefHash(repo, commitSHA)
	if err != nil {
		return "", fmt.Errorf("failed to resolve commit: %w", err)
	}

	goGitMu.Lock()
	defer goGitMu.Unlock()

	commit, err := repo.CommitObject(hash)
	if err != nil {
		return "", fmt.Errorf("failed to get commit: %w", err)
	}

	if commit.NumParents() == 0 {
		return "", fmt.Errorf("commit has no parents")
	}

	return commit.ParentHashes[0].String(), nil
}

func (r *runner) CheckCommutation(hunk Hunk, commitSHA, parentSHA string) (bool, error) {
	commitDiff, err := r.runGitCommandInternal("diff", parentSHA, commitSHA)
	if err != nil {
		return false, fmt.Errorf("failed to get commit diff: %w", err)
	}

	if strings.TrimSpace(commitDiff) == "" {
		return true, nil
	}

	commitHunks := parseDiffHunks(commitDiff, hunk.File)

	fileInDiff := false
	for _, line := range strings.Split(commitDiff, "\n") {
		if strings.Contains(line, hunk.File) {
			fileInDiff = true
			break
		}
	}
	if !fileInDiff {
		return true, nil
	}

	// If file appears in diff but no hunks parsed, might be a rename or parsing failed
	if len(commitHunks) == 0 {
		return false, nil
	}

	for _, commitHunk := range commitHunks {
		if commitHunk.File != hunk.File {
			continue
		}

		if hunkOverlaps(hunk, commitHunk) {
			return false, nil
		}
	}

	return true, nil
}

func (r *runner) GetRebaseHead() (string, error) {
	// Try standard rebase head refs in order:
	// 1. REBASE_HEAD (standard)
	// 2. refs/rebase-merge/head (interactive)
	// 3. refs/rebase-apply/head (non-interactive)
	refs := []string{
		"REBASE_HEAD",
		"refs/rebase-merge/head",
		"refs/rebase-apply/head",
	}

	for _, refName := range refs {
		output, err := r.runGitCommandInternal("rev-parse", "--verify", refName)
		if err == nil && output != "" {
			return strings.TrimSpace(output), nil
		}
	}

	return "", fmt.Errorf("rebase head not found")
}

func (r *runner) IsRebaseInProgress(ctx context.Context) bool {
	output, err := r.runGitCommandWithContextInternal(ctx, "rev-parse", "--git-dir")
	if err != nil {
		return false
	}

	gitDir := strings.TrimSpace(output)
	if _, err := os.Stat(gitDir + "/rebase-merge"); err == nil {
		return true
	}
	if _, err := os.Stat(gitDir + "/rebase-apply"); err == nil {
		return true
	}
	return false
}

func (r *runner) CheckRebaseInProgress(ctx context.Context) error {
	if r.IsRebaseInProgress(ctx) {
		return fmt.Errorf("a rebase is already in progress. Please finish or abort it first")
	}
	return nil
}

func (r *runner) CherryPickSimple(ctx context.Context, commitSHA string) error {
	_, err := r.runGitCommandWithContextInternal(ctx, "cherry-pick", commitSHA)
	return err
}

func (r *runner) CherryPickAbort(ctx context.Context) error {
	_, err := r.runGitCommandWithContextInternal(ctx, "cherry-pick", "--abort")
	return err
}

func (r *runner) ApplyPatch(ctx context.Context, patchFile string, threeWay bool) error {
	args := []string{"apply"}
	if threeWay {
		args = append(args, "--3way")
	}
	args = append(args, patchFile)
	_, err := r.runGitCommandWithContextInternal(ctx, args...)
	return err
}

func (r *runner) CommitAmendNoEdit(ctx context.Context) error {
	_, err := r.runGitCommandWithContextInternal(ctx, "commit", "-a", "--amend", "--no-edit", "--no-verify")
	return err
}

func (r *runner) HasUncommittedChanges(ctx context.Context) bool {
	output, err := r.runGitCommandWithContextInternal(ctx, "status", "--porcelain")
	if err != nil {
		return false
	}
	return strings.TrimSpace(output) != ""
}
