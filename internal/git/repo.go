package git

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

func (r *runner) GetRepoInfo(_ context.Context) (string, string, error) {
	repo, err := r.ensureRepo()
	if err != nil {
		return "", "", nil //nolint:nilerr // missing repo/remote info is non-fatal for callers
	}
	cfg, err := repo.Config()
	if err != nil {
		return "", "", nil //nolint:nilerr // preserve previous git config lookup behavior
	}
	origin := cfg.Remotes[DefaultRemote]
	if origin == nil || len(origin.URLs) == 0 {
		return "", "", nil
	}

	remoteURL := strings.TrimSpace(origin.URLs[0])
	if remoteURL == "" {
		return "", "", nil
	}
	remoteURL = strings.TrimSuffix(remoteURL, ".git")

	// Try standard URL parsing first (handles ssh://, https://, git://, etc.)
	parsed, err := url.Parse(remoteURL)
	if err == nil && parsed.Scheme != "" {
		parts := strings.Split(strings.TrimPrefix(parsed.Path, "/"), "/")
		if len(parts) < 2 {
			return "", "", fmt.Errorf("invalid remote URL: path must contain owner/repo")
		}
		return parts[0], parts[len(parts)-1], nil
	}

	// Fall back to SCP-like format: git@github.com:owner/repo
	if strings.Contains(remoteURL, "@") {
		if _, path, found := strings.Cut(remoteURL, ":"); found {
			parts := strings.Split(path, "/")
			if len(parts) < 2 {
				return "", "", fmt.Errorf("invalid SCP-style remote URL: path must contain owner/repo")
			}
			return parts[0], parts[len(parts)-1], nil
		}
	}

	// Local file paths (used in tests) or other formats - return empty
	// This is not an error; the caller may use other means to get repo info
	return "", "", nil
}
