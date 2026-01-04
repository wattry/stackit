package doctor

import (
	"fmt"

	"stackit.dev/stackit/internal/app"
	"stackit.dev/stackit/internal/github"
	"stackit.dev/stackit/internal/output"
)

// checkRepository performs repository-related checks
func checkRepository(ctx *app.Context, out output.Output, warnings []string, errors []string, trunk string) ([]string, []string) {
	// Check if we're in a git repository
	if ctx.RepoRoot == "" {
		if !ctx.Engine.Git().IsInsideRepo() {
			errors = append(errors, "not in a git repository")
			out.Error("  not in a git repository")
			return warnings, errors
		}
	}
	out.Info("  ✅ Current directory is a git repository")

	// Check remote configuration
	remoteURL, err := ctx.Engine.GetRemoteURL(ctx.Context)
	if err != nil {
		warnings = append(warnings, "remote 'origin' is not configured")
		out.Warn("  remote 'origin' is not configured")
	} else {
		// Check if it's a GitHub remote
		repoInfo, err := github.ParseGitHubRemoteURL(remoteURL)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("remote 'origin' is not a GitHub repository: %s", remoteURL))
			out.Warn("  remote 'origin' is not a GitHub repository")
		} else {
			out.Info("  ✅ Remote 'origin' is configured to GitHub (%s/%s)", repoInfo.Owner, repoInfo.Repo)
		}
	}

	// Check trunk branch
	if trunk == "" {
		errors = append(errors, "trunk branch not configured")
		out.Error("  trunk branch not configured")
	} else {
		// Check if trunk branch exists
		_, err := ctx.Engine.GetRevision(ctx.Engine.GetBranch(trunk))
		if err != nil {
			errors = append(errors, fmt.Sprintf("trunk branch '%s' does not exist", trunk))
			out.Error("  trunk branch '%s' does not exist", trunk)
		} else {
			out.Info("  ✅ Trunk branch '%s' exists", trunk)
		}
	}

	// Check if stackit is initialized (if trunk is set, it's initialized)
	if trunk == "" {
		errors = append(errors, "stackit is not initialized (run 'stackit init')")
		out.Error("  stackit is not initialized")
	} else {
		out.Info("  ✅ stackit is initialized")
	}

	return warnings, errors
}
