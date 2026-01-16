package split

import (
	"fmt"
	"strings"

	"stackit.dev/stackit/internal/app"
	"stackit.dev/stackit/internal/engine"
)

// WizardOptions configures the interactive split wizard
type WizardOptions struct {
	// Style is pre-selected split style (empty = prompt user)
	Style Style
	// Direction is pre-selected direction (empty = prompt user)
	Direction Direction
	// BranchName is a pre-selected branch name (empty = prompt or auto-generate)
	BranchName string
}

// RunWizard executes the interactive split wizard.
// It guides the user through selecting split type, direction, and then
// executes the appropriate split operation.
func RunWizard(ctx *app.Context, handler InteractiveHandler, opts WizardOptions) error {
	eng := ctx.Engine
	out := ctx.Output

	out.Debug("split wizard: starting with opts=%+v", opts)

	// Get current branch
	currentBranch := eng.CurrentBranch()
	if currentBranch == nil {
		return fmt.Errorf("not on a branch")
	}

	if err := currentBranch.EnsureCanModify(); err != nil {
		return err
	}

	// Check for uncommitted tracked changes
	hasUnstaged, err := eng.HasUnstagedChanges(ctx.Context)
	if err != nil {
		return fmt.Errorf("failed to check unstaged changes: %w", err)
	}
	if hasUnstaged {
		return fmt.Errorf("cannot split: you have uncommitted tracked changes")
	}

	// Ensure branch is tracked
	currentBranchObj := eng.GetBranch(currentBranch.GetName())
	if !currentBranchObj.IsTracked() {
		parent := currentBranch.GetParent()
		parentName := ""
		if parent == nil {
			parentName = eng.Trunk().GetName()
		} else {
			parentName = parent.GetName()
		}
		if err := eng.TrackBranch(ctx.Context, currentBranch.GetName(), parentName); err != nil {
			return fmt.Errorf("failed to track branch: %w", err)
		}
	}

	// Get commit count to determine available options
	commits, err := currentBranch.GetAllCommits(engine.CommitFormatSHA)
	if err != nil {
		return fmt.Errorf("failed to get commits: %w", err)
	}
	hasMultipleCommits := len(commits) > 1

	// Step 1: Choose split type (if not pre-selected)
	style := opts.Style
	if style == "" {
		handler.OnStep(StepChoosingType, StatusStarted, "Choose split type")

		availableTypes := buildTypeChoices(hasMultipleCommits)
		selectedStyle, err := handler.PromptSplitType(availableTypes)
		if err != nil {
			return err
		}
		style = selectedStyle

		handler.OnStep(StepChoosingType, StatusCompleted, string(style))
		out.Debug("split wizard: user selected style: %s", style)
	}

	// Step 2: Choose direction (if not pre-selected)
	direction := opts.Direction
	if direction == "" {
		handler.OnStep(StepChoosingDirection, StatusStarted, "Choose direction")

		// Build tree visualization for direction prompt
		parentName := ""
		if parent := currentBranch.GetParent(); parent != nil {
			parentName = parent.GetName()
		} else {
			parentName = eng.Trunk().GetName()
		}

		// Get children of current branch
		graph := engine.BuildStackGraph(eng, engine.SortStrategyAlphabetical, nil)
		childNames := graph.Children(*currentBranch)

		treeViz := buildDirectionTreeViz(currentBranch.GetName(), parentName, childNames)

		selectedDirection, err := handler.PromptDirection(treeViz)
		if err != nil {
			return err
		}
		direction = selectedDirection

		handler.OnStep(StepChoosingDirection, StatusCompleted, string(direction))
		out.Debug("split wizard: user selected direction: %s", direction)
	}

	// Start handler with branch info and style
	handler.Start(currentBranch.GetName(), style)

	// Take snapshot before any modifications
	snapshotArgs := []string{string(style), string(direction)}
	if err := eng.TakeSnapshot(engine.SnapshotOptions{
		Command: "split",
		Args:    snapshotArgs,
	}); err != nil {
		return fmt.Errorf("failed to take snapshot: %w", err)
	}

	// Execute the appropriate split based on style
	switch style {
	case StyleHunk:
		return splitByHunkWithHandler(ctx, *currentBranch, eng, out, handler, direction)
	case StyleCommit, StyleFile:
		// For now, these don't support the new wizard flow with direction
		// Fall back to the existing implementation
		return fmt.Errorf("style %s with direction is not yet implemented in wizard mode", style)
	default:
		return fmt.Errorf("unknown split style: %s", style)
	}
}

// buildTypeChoices creates the list of available split type options
func buildTypeChoices(hasMultipleCommits bool) []TypeChoice {
	return []TypeChoice{
		{
			Style:       StyleHunk,
			Label:       "By hunk",
			Description: "Interactively select code changes to extract into new branches",
			Available:   true,
		},
		{
			Style:       StyleFile,
			Label:       "By file",
			Description: "Extract entire files to a new branch",
			Available:   true,
		},
		{
			Style:       StyleCommit,
			Label:       "By commit",
			Description: "Divide commit history into separate branches",
			Available:   hasMultipleCommits,
		},
	}
}

// buildDirectionTreeViz creates a tree visualization showing where new branches
// would be placed for "above" vs "below" directions.
func buildDirectionTreeViz(currentBranch, parentBranch string, childBranches []string) string {
	var sb strings.Builder

	// Show parent (or trunk)
	fmt.Fprintf(&sb, "  ◯ %s\n", parentBranch)
	sb.WriteString("  │\n")

	// Show [BELOW] indicator
	sb.WriteString("  ├─ [BELOW] new branch would be inserted here\n")
	sb.WriteString("  │\n")

	// Show current branch
	fmt.Fprintf(&sb, "  ◉ %s  ← you are here\n", currentBranch)

	// Show children if any
	if len(childBranches) > 0 {
		sb.WriteString("  │\n")
		sb.WriteString("  └─ [ABOVE] new branch would be inserted here\n")
		for i, child := range childBranches {
			if i < len(childBranches)-1 {
				fmt.Fprintf(&sb, "      │\n      ├─ ◯ %s\n", child)
			} else {
				fmt.Fprintf(&sb, "      │\n      └─ ◯ %s\n", child)
			}
		}
	} else {
		sb.WriteString("  │\n")
		sb.WriteString("  └─ [ABOVE] new branch would be inserted here\n")
	}

	return sb.String()
}
