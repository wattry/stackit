package integrations

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"stackit.dev/stackit/internal/actions/integrations"
	"stackit.dev/stackit/internal/git"
)

// InstallGitHub installs GitHub Actions workflow for stackit CI checks.
// This is a convenience wrapper for use during init.
// When called from init, skipIfExists is true to avoid errors if workflows already exist.
func InstallGitHub(runner git.Runner, force bool, out io.Writer) error {
	// When called from init, we use skipIfExists to be friendlier
	// since the user can't pass --force through the init prompts
	return runGithubInstallWithOptions(runner, githubInstallOptions{
		Force:        force,
		SkipIfExists: !force, // If not forcing, skip existing files gracefully
	}, out)
}

// InstallPrecommit installs the pre-commit hook.
// This is a convenience wrapper for use during init.
func InstallPrecommit(runner git.Runner, out io.Writer) error {
	repoRoot, err := runner.DiscoverRepoRoot()
	if err != nil {
		return err
	}
	return integrations.PrecommitInstallActionWithOutput(repoRoot, out)
}

// InstallAgents installs AI agent integration files.
// This is a convenience wrapper for use during init.
func InstallAgents(runner git.Runner, local, force bool, version string, out io.Writer) error {
	return runAgentInstall(runner, local, force, version, out)
}

// stackitWorkflowMarker is a string that identifies a stackit-generated GitHub workflow.
// We check for this to distinguish stackit workflows from manually created ones.
const stackitWorkflowMarker = "refs/stackit/metadata"

// IsGitHubInstalled checks if GitHub Actions workflow is already installed.
// Returns true only if the workflow exists AND contains stackit-specific content,
// to distinguish from manually created workflows with the same name.
func IsGitHubInstalled(runner git.Runner) bool {
	repoRoot, err := runner.DiscoverRepoRoot()
	if err != nil {
		return false
	}

	for _, workflow := range githubWorkflows {
		workflowPath := filepath.Join(repoRoot, ".github", "workflows", workflow)
		content, err := os.ReadFile(workflowPath)
		if err != nil {
			return false
		}
		// Check that it's actually a stackit-generated workflow
		if !strings.Contains(string(content), stackitWorkflowMarker) {
			return false
		}
	}
	return true
}

// IsPrecommitInstalled checks if the pre-commit hook is already installed.
func IsPrecommitInstalled(runner git.Runner) bool {
	repoRoot, err := runner.DiscoverRepoRoot()
	if err != nil {
		return false
	}

	hookPath := filepath.Join(repoRoot, ".git", "hooks", "pre-commit")
	content, err := os.ReadFile(hookPath)
	if err != nil {
		return false
	}

	return strings.Contains(string(content), "stackit precommit verify")
}

// IsAgentsInstalled checks if agent integration files are already installed.
// Checks both global (~/.claude/) and local (.claude/) installations.
func IsAgentsInstalled(runner git.Runner) bool {
	// Check global installation
	homeDir, err := os.UserHomeDir()
	if err == nil {
		skillPath := filepath.Join(homeDir, ".claude", "skills", "stackit", "skill.md")
		if _, err := os.Stat(skillPath); err == nil {
			return true
		}
	}

	// Check local installation
	repoRoot, err := runner.DiscoverRepoRoot()
	if err == nil {
		skillPath := filepath.Join(repoRoot, ".claude", "skills", "stackit", "skill.md")
		if _, err := os.Stat(skillPath); err == nil {
			return true
		}
	}

	return false
}
