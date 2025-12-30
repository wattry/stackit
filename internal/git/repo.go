package git

import (
	"fmt"
	"os"
	"strings"
	"time"

	gogit "github.com/go-git/go-git/v5"
)

// GetRepoRoot returns the root directory of the Git repository
func GetRepoRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}

	repo, err := gogit.PlainOpenWithOptions(wd, &gogit.PlainOpenOptions{
		DetectDotGit: true,
	})
	if err != nil {
		return "", fmt.Errorf("not a git repository: %w", err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return "", fmt.Errorf("failed to get worktree: %w", err)
	}

	return worktree.Filesystem.Root(), nil
}

// GetRef returns the SHA of a ref
func GetRef(name string) (string, error) {
	return RunGitCommand("rev-parse", "--verify", name)
}

// UpdateRef updates a ref to point to a SHA
func UpdateRef(name, sha string) error {
	_, err := RunGitCommand("update-ref", name, sha)
	return err
}

// DeleteRef deletes a ref
func DeleteRef(name string) error {
	_, err := RunGitCommand("update-ref", "-d", name)
	return err
}

// CreateBlob creates a blob and returns its SHA
func CreateBlob(content string) (string, error) {
	return RunGitCommandWithInput(content, "hash-object", "-w", "--stdin")
}

// ReadBlob returns the content of a blob
func ReadBlob(sha string) (string, error) {
	return RunGitCommand("cat-file", "-p", sha)
}

// ListRefs returns all refs matching a prefix
func ListRefs(prefix string) (map[string]string, error) {
	result := make(map[string]string)
	output, err := RunGitCommand("for-each-ref", "--format=%(refname) %(objectname)", prefix)
	if err != nil || output == "" {
		return result, err
	}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		if len(parts) == 2 {
			result[parts[0]] = parts[1]
		}
	}
	return result, nil
}

// PushMetadataRefs pushes metadata refs to remote
func PushMetadataRefs(branches []string) error {
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
	_, err := RunGitCommand(args...)
	return err
}

// FetchMetadataRefs fetches metadata refs from remote
func FetchMetadataRefs() error {
	// git fetch origin 'refs/stackit/metadata/*:refs/stackit/remote-metadata/*'
	_, err := RunGitCommand("fetch", "origin", "+refs/stackit/metadata/*:refs/stackit/remote-metadata/*")
	return err
}

// DeleteRemoteMetadataRef deletes a remote metadata ref
func DeleteRemoteMetadataRef(branch string) error {
	// git push origin --delete refs/stackit/metadata/<branch>
	_, err := RunGitCommand("push", "origin", "--delete", fmt.Sprintf("refs/stackit/metadata/%s", branch))
	return err
}

// TestRemoteRefCompatibility verifies GitHub accepts custom ref pushes
func TestRemoteRefCompatibility() error {
	testRef := "refs/stackit/metadata/stackit-compat-test"
	testContent := fmt.Sprintf(`{"test":true,"timestamp":%d}`, time.Now().Unix())

	// Create test blob
	sha, err := CreateBlob(testContent)
	if err != nil {
		return fmt.Errorf("failed to create test blob: %w", err)
	}

	// Try to push test ref
	if err := UpdateRef(testRef, sha); err != nil {
		return fmt.Errorf("failed to update local test ref: %w", err)
	}

	if _, err := RunGitCommand("push", "origin", "+"+testRef); err != nil {
		_ = DeleteRef(testRef) // Cleanup local
		return fmt.Errorf("remote rejected metadata ref push: %w", err)
	}

	// Cleanup: delete remote and local test ref
	_, _ = RunGitCommand("push", "origin", "--delete", testRef)
	_ = DeleteRef(testRef)

	return nil
}

// GetConfig returns a git config value
func GetConfig(key string) (string, error) {
	return RunGitCommand("config", "--get", key)
}

// SetConfig sets a git config value
func SetConfig(key, value string) error {
	_, err := RunGitCommand("config", key, value)
	return err
}

// GetConfigAll returns all values for a git config key
func GetConfigAll(key string) ([]string, error) {
	output, err := RunGitCommand("config", "--get-all", key)
	if err == nil {
		if output == "" {
			return []string{}, nil
		}
		return strings.Split(output, "\n"), nil
	}

	// If key not found, git exits with 1. We treat this as an empty result.
	return []string{}, nil
}

// AddConfigValue adds a value to a multi-valued git config key
func AddConfigValue(key, value string) error {
	_, err := RunGitCommand("config", "--add", key, value)
	return err
}

// EnsureMetadataRefspecConfigured ensures the metadata fetch refspec is configured
// This allows fresh clones to automatically fetch metadata refs on git fetch
func EnsureMetadataRefspecConfigured() error {
	const metadataRefspec = "+refs/stackit/metadata/*:refs/stackit/remote-metadata/*"

	refspecs, err := GetConfigAll("remote.origin.fetch")
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
	return AddConfigValue("remote.origin.fetch", metadataRefspec)
}
