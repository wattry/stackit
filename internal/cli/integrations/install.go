package integrations

import (
	"io"

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
