package git

import (
	"fmt"

	"github.com/go-git/go-git/v6/plumbing"
)

func (r *runner) getMergeBase(repo *Repository, branch1, branch2 string) (string, error) {
	return r.getMergeBaseByRef(repo, branch1, branch2)
}

func (r *runner) getMergeBaseByRef(repo *Repository, ref1Name, ref2Name string) (string, error) {
	hash1, err := r.resolveRefHash(repo, ref1Name)
	if err != nil {
		return "", fmt.Errorf("failed to resolve %s: %w", ref1Name, err)
	}

	hash2, err := r.resolveRefHash(repo, ref2Name)
	if err != nil {
		return "", fmt.Errorf("failed to resolve %s: %w", ref2Name, err)
	}

	return r.getMergeBaseGoGit(repo, hash1, hash2)
}

func (r *runner) getMergeBaseGoGit(repo *Repository, hash1, hash2 plumbing.Hash) (string, error) {
	// Synchronize go-git operations to prevent concurrent packfile access
	r.goGitMu.Lock()
	defer r.goGitMu.Unlock()

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

func (r *runner) isAncestor(repo *Repository, ancestor, descendant string) (bool, error) {
	ancestorHash, err := r.resolveRefHash(repo, ancestor)
	if err != nil {
		return false, fmt.Errorf("failed to resolve ancestor %s: %w", ancestor, err)
	}

	descendantHash, err := r.resolveRefHash(repo, descendant)
	if err != nil {
		return false, fmt.Errorf("failed to resolve descendant %s: %w", descendant, err)
	}

	// If they're the same, ancestor is an ancestor
	if ancestorHash == descendantHash {
		return true, nil
	}

	return r.isAncestorGoGit(repo, ancestorHash, descendantHash)
}

func (r *runner) isAncestorGoGit(repo *Repository, ancestorHash, descendantHash plumbing.Hash) (bool, error) {
	// Synchronize go-git operations to prevent concurrent packfile access
	r.goGitMu.Lock()
	defer r.goGitMu.Unlock()

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
