package actions

import (
	"fmt"
	"strings"

	"stackit.dev/stackit/internal/app"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/tui"
	"stackit.dev/stackit/internal/tui/components/tree"
	"stackit.dev/stackit/internal/tui/style"
	"stackit.dev/stackit/internal/utils"
)

// LogOptions contains options for the log command
type LogOptions struct {
	Style         string // "NORMAL" or "FULL"
	Reverse       bool
	Steps         *int
	BranchName    string
	ShowUntracked bool
}

// LogAction displays the branch tree
func LogAction(ctx *app.Context, opts LogOptions) error {
	// Populate remote SHAs if needed (only for FULL mode)
	if opts.Style == "FULL" {
		if err := ctx.Engine.PopulateRemoteShas(); err != nil {
			ctx.Splog.Debug("Failed to populate remote SHAs: %v", err)
		}
	}

	// Create tree renderer
	renderer := tui.NewStackTreeRenderer(ctx.Engine)

	// Render the stack
	// First, collect annotations for all branches in the stack using a worker pool
	annotations := make(map[string]tree.BranchAnnotation)
	allBranches := ctx.Engine.AllBranches()

	type result struct {
		branchName string
		annotation tree.BranchAnnotation
	}
	results := make(chan result, len(allBranches))

	if len(allBranches) > 0 {
		utils.Run(allBranches, func(branchObj engine.Branch) {
			annotation := getBranchAnnotation(ctx, branchObj, opts)
			results <- result{branchObj.GetName(), annotation}
		})
	}
	close(results)

	for res := range results {
		annotations[res.branchName] = res.annotation
	}

	renderer.SetAnnotations(annotations)

	stackLines := renderer.RenderStack(opts.BranchName, tree.RenderOptions{
		Short:   false, // We want the full tree characters with stats
		Reverse: opts.Reverse,
		Steps:   opts.Steps,
	})

	// Add summary footer
	branchCount := 0
	approvedCount := 0
	inReviewCount := 0
	for name, ann := range annotations {
		branch := ctx.Engine.GetBranch(name)
		if branch.IsTrunk() {
			continue
		}
		branchCount++
		if ann.ReviewStatus == "Approved" {
			approvedCount++
		} else if ann.ReviewStatus == "In Review" {
			inReviewCount++
		}
	}

	if branchCount > 0 {
		summaryParts := []string{fmt.Sprintf("%d branches", branchCount)}
		if approvedCount > 0 {
			summaryParts = append(summaryParts, fmt.Sprintf("%d approved", approvedCount))
		}
		if inReviewCount > 0 {
			summaryParts = append(summaryParts, fmt.Sprintf("%d in review", inReviewCount))
		}
		stackLines = append(stackLines, "")
		stackLines = append(stackLines, style.ColorDim(strings.Join(summaryParts, " · ")))
	}

	// Add untracked branches if requested
	if opts.ShowUntracked {
		untracked := getUntrackedBranchNames(ctx)
		if len(untracked) > 0 {
			stackLines = append(stackLines, "")
			stackLines = append(stackLines, "Untracked branches:")
			stackLines = append(stackLines, untracked...)
		}
	}

	// Output the result
	ctx.Splog.Page(strings.Join(stackLines, "\n"))
	ctx.Splog.Newline()

	return nil
}

func getUntrackedBranchNames(ctx *app.Context) []string {
	var untracked []string
	for _, branch := range ctx.Engine.AllBranches() {
		branchName := branch.GetName()
		if !branch.IsTrunk() && !branch.IsTracked() {
			untracked = append(untracked, branchName)
		}
	}
	return untracked
}

func getBranchAnnotation(ctx *app.Context, branchObj engine.Branch, opts LogOptions) tree.BranchAnnotation {
	annotation := tree.BranchAnnotation{
		Scope:         ctx.Engine.GetScope(branchObj).String(),
		ExplicitScope: branchObj.GetExplicitScope().String(),
		IsLocked:      branchObj.IsLocked(),
		IsFrozen:      branchObj.IsFrozen(),
	}

	// Local stats (always fast enough)
	if !branchObj.IsTrunk() {
		if count, err := branchObj.GetCommitCount(); err == nil {
			annotation.CommitCount = count
		}
		if added, deleted, err := branchObj.GetDiffStats(); err == nil {
			annotation.LinesAdded = added
			annotation.LinesDeleted = deleted
		}
	}

	// PR info (local metadata)
	if !branchObj.IsTrunk() {
		prInfo, _ := branchObj.GetPrInfo()
		if prInfo != nil {
			annotation.PRNumber = prInfo.Number()
			annotation.PRState = prInfo.State()
			annotation.IsDraft = prInfo.IsDraft()
		}
	}

	// CI status (only in FULL mode)
	if opts.Style == "FULL" && !branchObj.IsTrunk() && ctx.GitHubClient != nil {
		if status, err := ctx.GitHubClient.GetPRChecksStatus(ctx.Context, branchObj.GetName()); err == nil && status != nil {
			annotation.CheckStatus = tree.CheckStatusPassing
			if status.Pending {
				annotation.CheckStatus = tree.CheckStatusPending
			} else if !status.Passing {
				annotation.CheckStatus = tree.CheckStatusFailing
			}
		}
	}

	return annotation
}
