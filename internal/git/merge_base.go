package git

import (
	"context"
	"fmt"

	"github.com/go-git/go-git/v5/plumbing"
)

func (r *runner) getMergeBase(repo *Repository, branch1, branch2 string) (string, error) {
	result, err := r.getMergeBaseByRef(repo, "refs/heads/"+branch1, "refs/heads/"+branch2)
	if err != nil {
		// Fallback to raw git for worktree compatibility
		return r.getMergeBaseGitFallback(branch1, branch2)
	}
	return result, nil
}

func (r *runner) getMergeBaseByRef(repo *Repository, ref1Name, ref2Name string) (string, error) {
	hash1, err := r.resolveRefHash(repo, ref1Name)
	if err != nil {
		// Fallback to raw git for worktree compatibility
		return r.getMergeBaseGitFallback(ref1Name, ref2Name)
	}

	hash2, err := r.resolveRefHash(repo, ref2Name)
	if err != nil {
		// Fallback to raw git for worktree compatibility
		return r.getMergeBaseGitFallback(ref1Name, ref2Name)
	}

	// Try go-git first, fallback to git command if it fails
	result, err := r.getMergeBaseGoGit(repo, hash1, hash2)
	if err != nil {
		// Fallback to raw git for worktree compatibility
		return r.getMergeBaseGitFallback(ref1Name, ref2Name)
	}
	return result, nil
}

func (r *runner) getMergeBaseGoGit(repo *Repository, hash1, hash2 plumbing.Hash) (string, error) {
	// Synchronize go-git operations to prevent concurrent packfile access
	goGitMu.Lock()
	defer goGitMu.Unlock()

	commit1, err := repo.CommitObject(hash1)
	if err != nil {
		return "", fmt.Errorf("failed to get commit1: %w", err)
	}

	commit2, err := repo.CommitObject(hash2)
	if err != nil {
		return "", fmt.Errorf("failed to get commit2: %w", err)
	}

	// Find merge base
	mergeBases, err := commit1.MergeBase(commit2)
	if err != nil {
		return "", fmt.Errorf("failed to find merge base: %w", err)
	}

	if len(mergeBases) == 0 {
		return "", fmt.Errorf("no merge base found")
	}

	return mergeBases[0].Hash.String(), nil
}

// getMergeBaseGitFallback uses the git merge-base command as a fallback
// when go-git fails. This is especially important for worktrees where go-git
// might not find all refs/objects because it doesn't handle the 'commondir'
// redirection properly.
func (r *runner) getMergeBaseGitFallback(ref1, ref2 string) (string, error) {
	output, err := r.RunGitCommandWithContext(context.Background(), "merge-base", ref1, ref2)
	if err != nil {
		return "", fmt.Errorf("failed to get merge base via git: %w", err)
	}
	return output, nil
}

func (r *runner) isAncestor(repo *Repository, ancestor, descendant string) (bool, error) {
	ancestorHash, err := r.resolveRefHash(repo, ancestor)
	if err != nil {
		return r.isAncestorGitFallback(ancestor, descendant)
	}

	descendantHash, err := r.resolveRefHash(repo, descendant)
	if err != nil {
		return r.isAncestorGitFallback(ancestor, descendant)
	}

	// If they're the same, ancestor is an ancestor
	if ancestorHash == descendantHash {
		return true, nil
	}

	// Try go-git first, fallback to git command if it fails
	result, err := r.isAncestorGoGit(repo, ancestorHash, descendantHash)
	if err != nil {
		return r.isAncestorGitFallback(ancestor, descendant)
	}
	return result, nil
}

func (r *runner) isAncestorGoGit(repo *Repository, ancestorHash, descendantHash plumbing.Hash) (bool, error) {
	// Synchronize go-git operations to prevent concurrent packfile access
	goGitMu.Lock()
	defer goGitMu.Unlock()

	// Get commit objects
	ancestorCommit, err := repo.CommitObject(ancestorHash)
	if err != nil {
		return false, fmt.Errorf("failed to get ancestor commit: %w", err)
	}

	descendantCommit, err := repo.CommitObject(descendantHash)
	if err != nil {
		return false, fmt.Errorf("failed to get descendant commit: %w", err)
	}

	return ancestorCommit.IsAncestor(descendantCommit)
}

// isAncestorGitFallback uses git merge-base --is-ancestor as a fallback
// when go-git fails. This is especially important for worktrees where go-git
// might not find all refs/objects because it doesn't handle the 'commondir'
// redirection properly.
func (r *runner) isAncestorGitFallback(ancestor, descendant string) (bool, error) {
	_, err := r.RunGitCommandWithContext(context.Background(),
		"merge-base", "--is-ancestor", ancestor, descendant)
	if err == nil {
		return true, nil
	}
	// git merge-base --is-ancestor returns exit code 1 if not an ancestor (not an error)
	// and exit code 128+ for actual errors (invalid commits, etc.)
	// We treat both as "not an ancestor" since we can't distinguish easily
	return false, nil
}
