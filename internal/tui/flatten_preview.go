package tui

import (
	"fmt"
	"strings"

	"stackit.dev/stackit/internal/tui/style"
)

// FlattenPlannedMove represents a single branch move in the flatten plan.
// This is a local struct to avoid import cycles with the actions/flatten package.
type FlattenPlannedMove struct {
	Branch    string // Branch being moved
	OldParent string // Current parent branch
	NewParent string // Target parent branch (closer to trunk)
}

// FlattenPreviewData contains the data needed to render a flatten preview.
// This is a local struct to avoid import cycles with the actions/flatten package.
type FlattenPreviewData struct {
	Moves          []FlattenPlannedMove // Branches that will be moved
	UnchangedCount int                  // Number of branches that won't change
	HasConflicts   bool                 // Whether the flatten will cause conflicts
	ConflictBranch string               // Which branch would have conflicts (if any)
	ConflictError  string               // Error message describing the conflict
}

// RenderFlattenPreview renders a flatten preview showing:
// - List of branches that will be moved and their new parents
// - Count of branches that will stay unchanged
// - Conflict status
func RenderFlattenPreview(preview FlattenPreviewData) string {
	var sb strings.Builder

	// Header
	sb.WriteString(style.ColorDim("─────────────────────────────────────\n"))
	sb.WriteString("Flatten Preview:\n")
	sb.WriteString(style.ColorDim("─────────────────────────────────────\n"))
	sb.WriteString("\n")

	// Moves
	if len(preview.Moves) > 0 {
		fmt.Fprintf(&sb, "Branches to move (%d):\n", len(preview.Moves))
		for _, move := range preview.Moves {
			fmt.Fprintf(&sb, "  %s  %s → %s\n",
				style.ColorBranchName(move.Branch, false),
				style.ColorDim(move.OldParent),
				style.ColorGreen(move.NewParent))
		}
		sb.WriteString("\n")
	}

	// Unchanged count
	if preview.UnchangedCount > 0 {
		fmt.Fprintf(&sb, "Branches unchanged: %d %s\n",
			preview.UnchangedCount,
			style.ColorDim("(already optimal or dependencies prevent moving)"))
		sb.WriteString("\n")
	}

	// Conflict status
	sb.WriteString(style.ColorDim("─────────────────────────────────────\n"))
	switch {
	case preview.HasConflicts:
		sb.WriteString(style.ColorRed("✗ ") + style.ColorRed("Conflicts detected") + "\n")
		fmt.Fprintf(&sb, "  Branch: %s\n", style.ColorBranchName(preview.ConflictBranch, false))
		fmt.Fprintf(&sb, "  Error: %s\n", preview.ConflictError)
	case len(preview.Moves) > 0:
		sb.WriteString(style.ColorGreen("✓ ") + style.ColorGreen("All moves validated successfully") + "\n")
	default:
		sb.WriteString(style.ColorGreen("✓ ") + style.ColorGreen("Stack is already flat") + "\n")
	}
	sb.WriteString(style.ColorDim("─────────────────────────────────────\n"))

	return sb.String()
}
