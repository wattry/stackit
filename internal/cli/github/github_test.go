package github

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/internal/git"
)

func TestGithubInstall(t *testing.T) {
	// Create temp directory
	tempDir := t.TempDir()

	// Initialize git repo
	err := os.Chdir(tempDir)
	require.NoError(t, err, "should change to temp directory")

	cmd := exec.Command("git", "init")
	cmd.Dir = tempDir
	err = cmd.Run()
	require.NoError(t, err, "should initialize git repo")

	// Run github install
	runner := git.NewRunner()
	err = runGithubInstall(runner, false)
	require.NoError(t, err, "github install should succeed")

	// Verify file was created
	workflowPath := filepath.Join(tempDir, ".github", "workflows", "stackit-lock-check.yml")
	_, err = os.Stat(workflowPath)
	require.NoError(t, err, "workflow file should exist")

	// Verify content
	content, err := os.ReadFile(workflowPath)
	require.NoError(t, err, "should read workflow file")
	require.Contains(t, string(content), "name: Stackit")
	require.Contains(t, string(content), "check-lock:")

	// Try to install again without force
	err = runGithubInstall(runner, false)
	require.Error(t, err, "should fail when file already exists")
	require.Contains(t, err.Error(), "file already exists")

	// Install with force
	err = runGithubInstall(runner, true)
	require.NoError(t, err, "should succeed with force flag")
}

func TestGithubInstall_NotGitRepo(t *testing.T) {
	tempDir := t.TempDir()

	err := os.Chdir(tempDir)
	require.NoError(t, err, "should change to temp directory")

	// Run github install without git repo
	runner := git.NewRunner()
	err = runGithubInstall(runner, false)
	require.Error(t, err, "should fail when not in git repo")
	require.Contains(t, err.Error(), "not a git repository")
}
