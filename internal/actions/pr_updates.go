package actions

import (
	"fmt"
	"regexp"
	"strings"

	"stackit.dev/stackit/internal/app"
	"stackit.dev/stackit/internal/github"
	"stackit.dev/stackit/internal/utils"
)

var scopeRegex = regexp.MustCompile(`^\[[^\]]+\]\s*`)

// UpdateStackPRMetadata updates PR titles and body footers for a list of branches
func UpdateStackPRMetadata(ctx *app.Context, branches []string, repoOwner, repoName string) {
	// Update PRs in parallel using a worker pool
	if len(branches) == 0 {
		return
	}

	utils.Run(branches, func(name string) {
		updatePRMetadata(ctx, name, repoOwner, repoName)
	})
}

func updatePRMetadata(ctx *app.Context, name string, repoOwner, repoName string) {
	branch := ctx.Engine.GetBranch(name)
	prInfo, err := branch.GetPrInfo()
	if err != nil || prInfo == nil || prInfo.Number() == nil {
		return
	}

	prNumber := *prInfo.Number()

	// 1. Fetch latest PR state from GitHub (Option 1)
	latestPR, err := ctx.GitHubClient.GetPullRequest(ctx.Context, repoOwner, repoName, prNumber)
	if err != nil {
		return
	}

	// 2. Calculate updates
	scope := ctx.Engine.GetScope(branch)
	currentTitle := latestPR.Title
	currentBody := latestPR.Body

	updatedTitle := currentTitle
	if !scope.IsEmpty() {
		if scopeRegex.MatchString(updatedTitle) {
			if !strings.HasPrefix(strings.ToUpper(updatedTitle), "["+strings.ToUpper(scope.String())+"]") {
				updatedTitle = scopeRegex.ReplaceAllString(updatedTitle, "["+scope.String()+"] ")
			}
		} else {
			updatedTitle = fmt.Sprintf("[%s] %s", scope.String(), updatedTitle)
		}
	}

	footer := CreatePRBodyFooter(name, ctx.Engine)
	updatedBody := UpdatePRBodyFooter(currentBody, footer)

	// 3. Apply updates if needed (Option 2)
	// Don't update body if:
	// - It would become empty (preserve existing body instead)
	// Allow footer to be added even when body is empty (user expects stack information)
	shouldUpdateBody := updatedBody != currentBody && updatedBody != ""
	if updatedTitle != currentTitle || shouldUpdateBody {
		updateOpts := github.UpdatePROptions{}
		if updatedTitle != currentTitle {
			updateOpts.Title = &updatedTitle
		}
		if shouldUpdateBody {
			updateOpts.Body = &updatedBody
		}

		err = ctx.GitHubClient.UpdatePullRequest(ctx.Context, repoOwner, repoName, prNumber, updateOpts)
		if err != nil {
			return
		}
	}

	// Successfully updated (or already up to date), update local engine state
	_ = ctx.Engine.UpsertPrInfo(branch, prInfo.WithTitleAndBody(updatedTitle, updatedBody).WithLocked(branch.IsEffectivelyLocked()))
}
