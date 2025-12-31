package sync

import (
	"fmt"
	"strings"
)

// FormatSummaryParts returns the summary parts as a slice of strings
// This is shared between SimpleSyncHandler and InteractiveSyncHandler
func FormatSummaryParts(summary Summary) []string {
	parts := []string{}

	if summary.TrunkUpdated {
		parts = append(parts, "pulled trunk")
	}
	if summary.BranchesSynced > 0 {
		parts = append(parts, fmt.Sprintf("synced %d branch%s", summary.BranchesSynced, pluralES(summary.BranchesSynced)))
	}
	if summary.BranchesRestacked > 0 {
		parts = append(parts, fmt.Sprintf("restacked %d", summary.BranchesRestacked))
	}
	if summary.BranchesDeleted > 0 {
		parts = append(parts, fmt.Sprintf("deleted %d", summary.BranchesDeleted))
	}
	if summary.BranchesSkipped > 0 {
		parts = append(parts, fmt.Sprintf("skipped %d (conflict)", summary.BranchesSkipped))
	}

	return parts
}

// FormatSummaryString returns the full summary as a string
func FormatSummaryString(summary Summary) string {
	if summary.UpToDate {
		return "Everything is up to date!"
	}

	parts := FormatSummaryParts(summary)
	if len(parts) == 0 {
		return ""
	}

	result := "Summary: " + strings.Join(parts, ", ")

	// Add actionable advice for conflicts
	if len(summary.ConflictBranches) > 0 {
		result += fmt.Sprintf("\n   Run 'st restack %s' to resolve and continue", summary.ConflictBranches[0])
	}

	return result
}

// pluralES returns "es" if count != 1, otherwise empty string (for "branch" -> "branches")
func pluralES(count int) string {
	if count == 1 {
		return ""
	}
	return "es"
}
