package actions

import (
	"fmt"
	"regexp"
	"strings"
	"sync"

	"stackit.dev/stackit/internal/github"
	"stackit.dev/stackit/internal/runtime"
)

var scopeRegex = regexp.MustCompile(`^\[[^\]]+\]\s*`)

// UpdateStackPRMetadata updates PR titles and body footers for a list of branches
func UpdateStackPRMetadata(ctx *runtime.Context, branches []string, repoOwner, repoName string) {
	var wg sync.WaitGroup
	for _, branchName := range branches {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
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
		}(branchName)
	}
	wg.Wait()
}
