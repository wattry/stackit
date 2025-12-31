package git

import (
	"fmt"
	"io"
	"strings"

	"github.com/go-git/go-git/v5/plumbing"
)

func (r *runner) getRef(repo *Repository, name string) (string, error) {
	return r.getRevision(repo, name)
}

func (r *runner) readBlob(repo *Repository, sha string) (string, error) {
	// Synchronize go-git operations to prevent concurrent packfile access
	goGitMu.Lock()
	defer goGitMu.Unlock()

	hash := plumbing.NewHash(sha)
	blob, err := repo.BlobObject(hash)
	if err != nil {
		return "", fmt.Errorf("failed to get blob %s: %w", sha, err)
	}

	reader, err := blob.Reader()
	if err != nil {
		return "", fmt.Errorf("failed to get blob reader: %w", err)
	}
	defer func() { _ = reader.Close() }()

	content, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("failed to read blob: %w", err)
	}

	return string(content), nil
}

func (r *runner) listRefs(repo *Repository, prefix string) (map[string]string, error) {
	// Synchronize go-git operations to prevent concurrent packfile access
	goGitMu.Lock()
	defer goGitMu.Unlock()

	result := make(map[string]string)
	refs, err := repo.References()
	if err != nil {
		return nil, fmt.Errorf("failed to list references: %w", err)
	}
	defer refs.Close()

	err = refs.ForEach(func(ref *plumbing.Reference) error {
		refName := string(ref.Name())
		if strings.HasPrefix(refName, prefix) {
			result[refName] = ref.Hash().String()
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to iterate references: %w", err)
	}

	return result, nil
}
