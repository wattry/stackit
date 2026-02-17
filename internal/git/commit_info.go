package git

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"

	"stackit.dev/stackit/internal/utils"
)

func (r *runner) getCommitDate(repo *Repository, branchName string) (time.Time, error) {
	// Synchronize go-git operations to prevent concurrent packfile access
	r.goGitMu.Lock()
	defer r.goGitMu.Unlock()

	hash, err := r.resolveRefHashInternal(repo, branchName)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to resolve branch reference: %w", err)
	}

	commit, err := repo.CommitObject(hash)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to get commit: %w", err)
	}

	return commit.Author.When, nil
}

func (r *runner) getCommitAuthor(repo *Repository, branchName string) (string, error) {
	// Synchronize go-git operations to prevent concurrent packfile access
	r.goGitMu.Lock()
	defer r.goGitMu.Unlock()

	hash, err := r.resolveRefHashInternal(repo, branchName)
	if err != nil {
		return "", fmt.Errorf("failed to resolve branch reference: %w", err)
	}

	commit, err := repo.CommitObject(hash)
	if err != nil {
		return "", fmt.Errorf("failed to get commit: %w", err)
	}

	return commit.Author.Name, nil
}

func (r *runner) getRevision(repo *Repository, branchName string) (string, error) {
	// Check revision cache first to avoid go-git mutex contention.
	// The cache is populated by LoadAllBranchRevisions (batch preload)
	// but not on individual misses, to avoid stale entries from external mutations.
	if cached, ok := r.revisionCache.Get(branchName); ok {
		return cached, nil
	}

	// Synchronize go-git operations to prevent concurrent packfile access
	r.goGitMu.Lock()
	defer r.goGitMu.Unlock()

	hash, err := r.resolveRefHashInternal(repo, branchName)
	if err != nil {
		return "", fmt.Errorf("failed to resolve branch reference: %w", err)
	}

	return hash.String(), nil
}

func (r *runner) getRemoteRevision(repo *Repository, branchName string) (string, error) {
	// Synchronize go-git operations to prevent concurrent packfile access
	r.goGitMu.Lock()
	defer r.goGitMu.Unlock()

	// Try refs/remotes/origin/branchName
	hash, err := r.resolveRefHashInternal(repo, "origin/"+branchName)
	if err != nil {
		return "", fmt.Errorf("failed to get remote branch reference: %w", err)
	}

	return hash.String(), nil
}

// iterateCommitsNoLock iterates commits from head to base without locking
func iterateCommitsNoLock(repo *Repository, headHash, baseHash plumbing.Hash) ([]*object.Commit, error) {
	var commits []*object.Commit
	currentHash := headHash

	for !currentHash.IsZero() && currentHash != baseHash {
		commit, err := repo.CommitObject(currentHash)
		if err != nil {
			return nil, fmt.Errorf("failed to get commit %s: %w", currentHash, err)
		}

		commits = append(commits, commit)

		if commit.NumParents() == 0 {
			break
		}
		// Follow the first parent for a linear history walk
		currentHash = commit.ParentHashes[0]
	}

	return commits, nil
}

// getCommitRangeGitFallback uses git rev-list as a fallback when go-git fails.
// This is especially important for worktrees where go-git might not find all
// objects because it doesn't handle the 'commondir' redirection properly.
func (r *runner) getCommitRangeGitFallback(base, head string) ([]string, error) {
	var args []string
	if base == "" {
		// Walk to root: git rev-list <head>
		args = []string{"rev-list", head}
	} else {
		// Range: git rev-list <base>..<head>
		args = []string{"rev-list", base + ".." + head}
	}

	output, err := r.RunGitCommandWithContext(context.Background(), args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get commit range via git: %w", err)
	}

	if output == "" {
		return []string{}, nil
	}

	// Split output into lines (one SHA per line)
	lines := strings.Split(strings.TrimSpace(output), "\n")
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		if line != "" {
			result = append(result, line)
		}
	}

	return result, nil
}

// resolveRefHash resolves a ref (branch name, SHA, or ref path) to a hash
func (r *runner) resolveRefHash(repo *Repository, ref string) (plumbing.Hash, error) {
	// Synchronize go-git operations to prevent concurrent packfile access
	r.goGitMu.Lock()
	defer r.goGitMu.Unlock()

	return r.resolveRefHashInternal(repo, ref)
}

// resolveRefHashInternal resolves a ref without locking
func (r *runner) resolveRefHashInternal(repo *Repository, ref string) (plumbing.Hash, error) {
	// 1. Try as a full reference name
	if refInfo, err := repo.Reference(plumbing.ReferenceName(ref), true); err == nil {
		return refInfo.Hash(), nil
	}

	// 2. Try as a local branch
	if refInfo, err := repo.Reference(plumbing.ReferenceName("refs/heads/"+ref), true); err == nil {
		return refInfo.Hash(), nil
	}

	// 3. Try as a remote branch
	if refInfo, err := repo.Reference(plumbing.ReferenceName("refs/remotes/origin/"+ref), true); err == nil {
		return refInfo.Hash(), nil
	}

	// 4. Try as a tag
	if refInfo, err := repo.Reference(plumbing.ReferenceName("refs/tags/"+ref), true); err == nil {
		return refInfo.Hash(), nil
	}

	// 5. Try ResolveRevision (handles SHAs, short SHAs, and complex expressions like HEAD~1)
	hash, err := repo.ResolveRevision(plumbing.Revision(ref))
	if err == nil {
		return *hash, nil
	}

	// FALLBACK: Use git rev-parse if go-git fails.
	// This is especially important for worktrees where go-git might not find all refs
	// because it doesn't always handle the 'commondir' redirection in worktrees.
	if output, err := r.RunGitCommandWithContext(context.Background(), "rev-parse", ref); err == nil {
		trimmed := strings.TrimSpace(output)
		if len(trimmed) == 40 || len(trimmed) == 64 { // valid SHA1 or SHA256
			return plumbing.NewHash(trimmed), nil
		}
	}

	return plumbing.ZeroHash, fmt.Errorf("failed to resolve ref %s: reference not found", ref)
}

// LoadAllBranchRevisions populates the revision cache for all local branches
// using a single `git show-ref --heads` call. This replaces N individual
// go-git ref resolutions (each requiring goGitMu) with one subprocess call.
func (r *runner) LoadAllBranchRevisions() error {
	output, err := r.RunGitCommandWithContext(context.Background(), "show-ref", "--heads")
	if err != nil {
		// show-ref returns exit code 1 if no refs found (empty repo)
		return nil //nolint:nilerr
	}

	lines := strings.SplitSeq(strings.TrimSpace(output), "\n")
	for line := range lines {
		if line == "" {
			continue
		}
		// Format: "SHA refs/heads/branchname"
		parts := strings.SplitN(line, " ", 2)
		if len(parts) != 2 {
			continue
		}
		sha := parts[0]
		ref := parts[1]
		if branchName, ok := strings.CutPrefix(ref, "refs/heads/"); ok {
			r.revisionCache.Put(branchName, sha)
		}
	}
	return nil
}

func (r *runner) batchGetRevisions(repo *Repository, branchNames []string) (map[string]string, []error) {
	results := make(map[string]string)
	var errors []error
	resultsMu := sync.Mutex{}
	errorsMu := sync.Mutex{}

	if len(branchNames) == 0 {
		return results, errors
	}

	utils.Run(branchNames, func(name string) {
		sha, err := r.getRevision(repo, name)
		if err != nil {
			errorsMu.Lock()
			errors = append(errors, fmt.Errorf("failed to get revision for %s: %w", name, err))
			errorsMu.Unlock()
		} else {
			resultsMu.Lock()
			results[name] = sha
			resultsMu.Unlock()
		}
	})

	return results, errors
}
