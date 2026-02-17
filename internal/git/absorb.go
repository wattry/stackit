package git

import (
	"regexp"
	"strings"
)

// HunkTarget represents a hunk and its target commit
type HunkTarget struct {
	Hunk        Hunk
	CommitSHA   string
	CommitIndex int // Index in the commit list (0 = newest)
}

// hunkOverlaps checks if two hunks have overlapping line ranges.
// It includes a safety margin to account for git context lines.
func hunkOverlaps(h1, h2 Hunk) bool {
	if h1.File != h2.File {
		return false
	}

	// Add safety margin of 3 lines (typical git context) to avoid conflicts
	margin := 3

	h1Start := h1.OldStart - margin
	h1End := h1.OldStart + h1.OldCount + margin
	h2Start := h2.NewStart
	h2End := h2.NewStart + h2.NewCount

	overlap := h1Start <= h2End && h2Start <= h1End
	return overlap
}

// parseDiffHunks parses a diff output and extracts hunks for a specific file
func parseDiffHunks(diffOutput, targetFile string) []Hunk {
	if strings.TrimSpace(diffOutput) == "" {
		return []Hunk{}
	}

	var hunks []Hunk
	lines := strings.Split(diffOutput, "\n")

	hunkHeaderRegex := regexp.MustCompile(`^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@`)

	var currentHunk *Hunk
	var currentFile string
	var hunkLines []string

	for _, line := range lines {
		line = strings.TrimRight(line, "\r")
		if strings.HasPrefix(line, "diff --git") {
			if currentHunk != nil {
				currentHunk.Content = strings.Join(hunkLines, "\n")
				if currentHunk.File == targetFile {
					hunks = append(hunks, *currentHunk)
				}
				currentHunk = nil
				hunkLines = nil
			}
			parts := strings.Split(line, " ")
			if len(parts) >= 4 {
				bPath := parts[len(parts)-1]
				if after, ok := strings.CutPrefix(bPath, "b/"); ok {
					currentFile = after
				}
			}
			continue
		}

		if match := hunkHeaderRegex.FindStringSubmatch(line); match != nil {
			if currentHunk != nil {
				currentHunk.Content = strings.Join(hunkLines, "\n")
				if currentHunk.File == targetFile {
					hunks = append(hunks, *currentHunk)
				}
			}

			oldStart := parseInt(match[1])
			oldCount := parseInt(match[2])
			if oldCount == 0 {
				oldCount = 1
			}
			newStart := parseInt(match[3])
			newCount := parseInt(match[4])
			if newCount == 0 {
				newCount = 1
			}

			currentHunk = &Hunk{
				File:     currentFile,
				OldStart: oldStart,
				OldCount: oldCount,
				NewStart: newStart,
				NewCount: newCount,
			}
			hunkLines = []string{line}
			continue
		}

		if currentHunk != nil {
			hunkLines = append(hunkLines, line)
		}
	}

	if currentHunk != nil {
		currentHunk.Content = strings.Join(hunkLines, "\n")
		if currentHunk.File == targetFile {
			hunks = append(hunks, *currentHunk)
		}
	}

	return hunks
}
