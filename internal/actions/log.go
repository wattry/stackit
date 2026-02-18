package actions

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"stackit.dev/stackit/internal/app"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/github"
	"stackit.dev/stackit/internal/tui"
	"stackit.dev/stackit/internal/tui/components/tree"
	"stackit.dev/stackit/internal/tui/style"
	"stackit.dev/stackit/internal/utils"
)

// LogStyle defines the output style for the log command
const (
	LogStyleNormal = "NORMAL"
	LogStyleFull   = "FULL"
	LogStyleShort  = "SHORT"
)

// LogOptions contains options for the log command
type LogOptions struct {
	Style         string // LogStyleNormal, LogStyleFull, or LogStyleShort
	Steps         *int
	BranchName    string
	ShowUntracked bool
	Interactive   bool
	ShowSHAs      bool // Show commit SHAs next to branch names
}

// LogAction displays the branch tree
func LogAction(ctx *app.Context, opts LogOptions) error {
	// If interactive mode is requested or auto-detected
	if opts.Interactive || (utils.IsInteractive() && opts.Steps == nil) {
		// Run interactive TUI
		m := tui.NewLogModel(ctx.Context, ctx.Engine, ctx.GitHubClient, tui.LogOptions{
			Style:         opts.Style,
			ShowUntracked: opts.ShowUntracked,
			Logger:        ctx.Logger,
		})
		m.SetAltScreen(true)
		p := tea.NewProgram(m)
		_, err := p.Run()
		return err
	}

	// Populate remote SHAs if needed (only for FULL mode)
	if opts.Style == LogStyleFull {
		if err := ctx.Engine.PopulateRemoteShas(); err != nil {
			ctx.Output.Debug("Failed to populate remote SHAs: %v", err)
		}
	}

	// Detect worktrees (builds both empty and stack-root maps in one call)
	wtData := tui.GetWorktreeData(ctx.Engine)

	// Create tree renderer - use empty worktrees-aware version if we have any
	var renderer *tree.StackTreeRenderer
	if len(wtData.EmptyWorktrees) > 0 {
		emptyWorktreeNames := make(map[string]bool)
		for name := range wtData.EmptyWorktrees {
			emptyWorktreeNames[name] = true
		}
		renderer = tui.NewStackTreeRendererWithEmptyWorktrees(ctx.Engine, emptyWorktreeNames)
	} else {
		renderer = tui.NewStackTreeRenderer(ctx.Engine)
	}

	// Pre-load metadata and revisions for all branches to eliminate per-branch
	// cache misses during parallel annotation building.
	ctx.Engine.PreloadBranchData()

	// Render the stack
	// First, collect annotations for all branches in the stack using a worker pool
	annotations := make(map[string]tree.BranchAnnotation)
	allBranches := ctx.Engine.AllBranches()

	// Prefetch CI status in batch if in FULL style
	var ciStatuses map[string]*github.CheckStatus
	if opts.Style == LogStyleFull && ctx.GitHubClient != nil {
		branchNames := make([]string, 0, len(allBranches))
		for _, b := range allBranches {
			if !b.IsTrunk() {
				branchNames = append(branchNames, b.GetName())
			}
		}
		if len(branchNames) > 0 {
			ciStatuses, _ = ctx.GitHubClient.BatchGetPRChecksStatus(ctx.Context, branchNames)
		}
	}

	enrichment := &tui.AnnotationEnrichment{
		CIStatuses:          ciStatuses,
		EmptyWorktrees:      wtData.EmptyWorktrees,
		WorktreeByStackRoot: wtData.WorktreeByStackRoot,
	}

	type result struct {
		branchName string
		annotation tree.BranchAnnotation
	}
	results := make(chan result, len(allBranches))

	if len(allBranches) > 0 {
		utils.Run(allBranches, func(branchObj engine.Branch) {
			annotation := tui.BuildFullAnnotation(ctx.Engine, branchObj, enrichment)
			results <- result{branchObj.GetName(), annotation}
		})
	}
	close(results)

	for res := range results {
		annotations[res.branchName] = res.annotation
	}

	renderer.SetAnnotations(annotations)

	stackLines := renderer.RenderStack(opts.BranchName, tree.RenderOptions{
		Mode:        tree.RenderModeFull, // We want the full tree characters with stats
		Steps:       opts.Steps,
		ShowSHAs:    opts.ShowSHAs,
		HideSummary: opts.Style == LogStyleShort,
	})

	// Add summary footer
	branchCount := 0
	approvedCount := 0
	inReviewCount := 0
	for name, ann := range annotations {
		branch := ctx.Engine.GetBranch(name)
		if branch.IsTrunk() || branch.IsWorktreeAnchor() {
			continue
		}
		branchCount++
		switch ann.ReviewStatus {
		case "Approved":
			approvedCount++
		case "In Review":
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
	ctx.Output.Print(strings.Join(stackLines, "\n"))
	ctx.Output.Newline()

	return nil
}

func getUntrackedBranchNames(ctx *app.Context) []string {
	untracked := engine.FilterBranches(ctx.Engine, func(b engine.Branch) bool {
		return !b.IsTrunk() && !b.IsTracked()
	})
	names := make([]string, len(untracked))
	for i, b := range untracked {
		names[i] = b.GetName()
	}
	return names
}
