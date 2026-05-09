package git

import (
	"context"
	"errors"
	"fmt"

	gogit "github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/transport"
)

// DefaultRemote is the default name for the remote repository
const DefaultRemote = "origin"

func (r *runner) getRemote(repo *Repository) string {
	// Try to get current branch
	head, err := repo.Head()
	if err == nil && head.Name().IsBranch() {
		branchName := head.Name().Short()
		// Try to get remote for the current branch
		cfg, err := repo.Config()
		if err == nil {
			if b, ok := cfg.Branches[branchName]; ok && b.Remote != "" {
				return b.Remote
			}
		}
	}

	// Fallback to origin
	return DefaultRemote
}

func (r *runner) fetchRemoteShas(ctx context.Context, repo *Repository, remote string) (map[string]string, error) {
	// Ensure context has a deadline for the network operation
	if ctx == nil {
		ctx = context.Background()
	}
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, DefaultCommandTimeout)
		defer cancel()
	}

	rem, err := repo.Remote(remote)
	if err != nil {
		return nil, fmt.Errorf("failed to get remote %s: %w", remote, err)
	}

	// List remote references with context for timeout support
	refs, err := rem.ListContext(ctx, &gogit.ListOptions{})
	if err != nil {
		if errors.Is(err, transport.ErrEmptyRemoteRepository) || errors.Is(err, gogit.NoErrAlreadyUpToDate) {
			return make(map[string]string), nil
		}
		return nil, fmt.Errorf("failed to list remote refs for %s: %w", remote, err)
	}

	remoteShas := make(map[string]string)
	for _, ref := range refs {
		// Only process refs/heads/* (branches)
		if ref.Name().IsBranch() {
			branchName := ref.Name().Short()
			remoteShas[branchName] = ref.Hash().String()
		}
	}

	return remoteShas, nil
}

func (r *runner) getRemoteSha(repo *Repository, remote, branchName string) (string, error) {
	refName := plumbing.ReferenceName(fmt.Sprintf("refs/remotes/%s/%s", remote, branchName))
	ref, err := repo.Reference(refName, true)
	if err != nil {
		return "", fmt.Errorf("failed to get remote SHA for %s/%s: %w", remote, branchName, err)
	}

	return ref.Hash().String(), nil
}
