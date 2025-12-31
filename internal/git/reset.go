package git

import (
	"fmt"

	"github.com/go-git/go-git/v5/plumbing"
)

func (r *runner) getRemoteSha(repo *Repository, remote, branchName string) (string, error) {
	// Synchronize go-git operations to prevent concurrent packfile access
	goGitMu.Lock()
	defer goGitMu.Unlock()

	refName := plumbing.ReferenceName(fmt.Sprintf("refs/remotes/%s/%s", remote, branchName))
	ref, err := repo.Reference(refName, true)
	if err != nil {
		return "", fmt.Errorf("failed to get remote SHA for %s/%s: %w", remote, branchName, err)
	}

	return ref.Hash().String(), nil
}
