package git

import (
	"context"
	"fmt"
	"strings"
)

func (r *runner) GetRepoInfo(ctx context.Context) (string, string, error) {
	// Get remote URL
	url, _ := r.RunGitCommandWithContext(ctx, "config", "--get", "remote.origin.url")
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
		// SSH format: either git@github.com:owner/repo or ssh://git@github.com/owner/repo
		if trimmed, ok := strings.CutPrefix(url, "ssh://"); ok {
			// URL format: ssh://git@hostname/owner/repo
			_, hostAndPath, found := strings.Cut(trimmed, "@")
			if !found {
				return "", "", fmt.Errorf("invalid SSH remote URL: missing @")
			}
			pathParts := strings.Split(hostAndPath, "/")
			if len(pathParts) < 3 {
				return "", "", fmt.Errorf("invalid SSH remote URL: path must be hostname/owner/repo")
			}
			owner = pathParts[1]
		} else {
			// SCP-like format: git@github.com:owner/repo
			sshParts := strings.Split(url, ":")
			if len(sshParts) < 2 {
				return "", "", fmt.Errorf("invalid SSH remote URL")
			}
			pathParts := strings.Split(sshParts[1], "/")
			if len(pathParts) < 2 {
				return "", "", fmt.Errorf("invalid SSH remote URL")
			}
			owner = pathParts[0]
		}
	} else {
		// HTTPS format: https://github.com/owner/repo
		owner = parts[len(parts)-2]
	}

	return owner, repoName, nil
}
