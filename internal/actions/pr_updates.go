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

	scope := ctx.Engine.GetScope(branch)
	updatedTitle := prInfo.Title()
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
	updatedBody := UpdatePRBodyFooter(prInfo.Body(), footer)

	if updatedTitle != prInfo.Title() || updatedBody != prInfo.Body() {
		updateOpts := github.UpdatePROptions{}
		if updatedTitle != prInfo.Title() {
			updateOpts.Title = &updatedTitle
		}
		if updatedBody != prInfo.Body() {
			updateOpts.Body = &updatedBody
		}

		if err := ctx.GitHubClient.UpdatePullRequest(ctx.Context, repoOwner, repoName, *prInfo.Number(), updateOpts); err != nil {
			return
		}

		_ = ctx.Engine.UpsertPrInfo(branch, prInfo.WithTitleAndBody(updatedTitle, updatedBody))
	}
}
