package git

import (
	"fmt"
	"regexp"
	"strings"
)

// Hunk represents a single hunk of changes in a diff
type Hunk struct {
	File      string // File path
	OldStart  int    // Line number in old file (1-indexed)
	OldCount  int    // Number of lines in old file
	NewStart  int    // Line number in new file (1-indexed)
	NewCount  int    // Number of lines in new file
	Content   string // The actual diff content (including header)
	IndexLine string // The index line from the diff (e.g., "index abc123..def456 100644") for --3way merging
	Binary    bool   // True if this represents a binary file change
}

// ParseDiffOutput parses a diff output into structured hunks
func ParseDiffOutput(diffOutput string) ([]Hunk, error) {
	if diffOutput == "" {
		return []Hunk{}, nil
	}

	var hunks []Hunk
	lines := strings.Split(diffOutput, "\n")

	// Regex to match hunk headers: @@ -old_start,old_count +new_start,new_count @@
	// Example: @@ -10,5 +10,6 @@
	hunkHeaderRegex := regexp.MustCompile(`^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@`)

	var currentHunk *Hunk
	var currentFile string
	var currentIndexLine string
	var hunkLines []string

	for _, line := range lines {
		line = strings.TrimRight(line, "\r")
		// Check for file header (starts with "diff --git" or "--- a/" or "+++ b/")
		if strings.HasPrefix(line, "diff --git") {
			// Save previous hunk if exists
			if currentHunk != nil {
				currentHunk.Content = strings.Join(hunkLines, "\n")
				hunks = append(hunks, *currentHunk)
				currentHunk = nil
				hunkLines = nil
			}
			// Extract file path from "diff --git a/path b/path"
			// Format: "diff --git a/path/to/file b/path/to/file"
			parts := strings.Split(line, " ")
			if len(parts) >= 4 {
				// parts[2] = "a/path/to/file", parts[3] = "b/path/to/file"
				bPath := parts[len(parts)-1]
				if strings.HasPrefix(bPath, "b/") {
					currentFile = strings.TrimPrefix(bPath, "b/")
				}
			}
			// Reset index line for new file
			currentIndexLine = ""
			continue
		}

		// Capture the index line (e.g., "index abc123..def456 100644")
		// This is needed for --3way merge to work
		if strings.HasPrefix(line, "index ") {
			currentIndexLine = line
			continue
		}

		// Check for binary file marker
		// Format: "Binary files a/path and b/path differ"
		if strings.HasPrefix(line, "Binary files ") && strings.HasSuffix(line, " differ") {
			// Save previous hunk if exists
			if currentHunk != nil {
				currentHunk.Content = strings.Join(hunkLines, "\n")
				hunks = append(hunks, *currentHunk)
				currentHunk = nil
				hunkLines = nil
			}
			// Create a binary file hunk
			hunks = append(hunks, Hunk{
				File:      currentFile,
				Content:   line,
				IndexLine: currentIndexLine,
				Binary:    true,
			})
			continue
		}

		// Check for hunk header
		if match := hunkHeaderRegex.FindStringSubmatch(line); match != nil {
			// Save previous hunk if exists
			if currentHunk != nil {
				currentHunk.Content = strings.Join(hunkLines, "\n")
				hunks = append(hunks, *currentHunk)
			}

			// Parse hunk header
			oldStart := parseInt(match[1])
			oldCount := parseInt(match[2])
			if oldCount == 0 {
				oldCount = 1 // Default to 1 if not specified
			}
			newStart := parseInt(match[3])
			newCount := parseInt(match[4])
			if newCount == 0 {
				newCount = 1 // Default to 1 if not specified
			}

			currentHunk = &Hunk{
				File:      currentFile,
				OldStart:  oldStart,
				OldCount:  oldCount,
				NewStart:  newStart,
				NewCount:  newCount,
				IndexLine: currentIndexLine,
			}
			hunkLines = []string{line}
			continue
		}

		// Accumulate hunk content
		if currentHunk != nil {
			hunkLines = append(hunkLines, line)
		}
	}

	// Save last hunk
	if currentHunk != nil {
		currentHunk.Content = strings.Join(hunkLines, "\n")
		hunks = append(hunks, *currentHunk)
	}

	return hunks, nil
}

// parseInt parses a string to int, returns 0 if empty or invalid
func parseInt(s string) int {
	if s == "" {
		return 0
	}
	var result int
	_, _ = fmt.Sscanf(s, "%d", &result)
	return result
}

// BuildPatchFromHunks constructs a unified diff patch from selected hunks.
// The patch can be applied using git apply --cached to stage specific hunks.
func BuildPatchFromHunks(hunks []Hunk) string {
	if len(hunks) == 0 {
		return ""
	}

	// Group hunks by file
	fileHunks := make(map[string][]Hunk)
	fileOrder := make([]string, 0)
	for _, h := range hunks {
		if _, exists := fileHunks[h.File]; !exists {
			fileOrder = append(fileOrder, h.File)
		}
		fileHunks[h.File] = append(fileHunks[h.File], h)
	}

	var sb strings.Builder

	// Build patch for each file
	for _, file := range fileOrder {
		hunksForFile := fileHunks[file]

		// Write diff header
		sb.WriteString(fmt.Sprintf("diff --git a/%s b/%s\n", file, file))

		// Add index line if available (needed for --3way)
		if hunksForFile[0].IndexLine != "" {
			sb.WriteString(hunksForFile[0].IndexLine + "\n")
		}

		sb.WriteString(fmt.Sprintf("--- a/%s\n", file))
		sb.WriteString(fmt.Sprintf("+++ b/%s\n", file))

		// Write each hunk
		for _, h := range hunksForFile {
			sb.WriteString(h.Content)
			// Ensure content ends with newline
			if !strings.HasSuffix(h.Content, "\n") {
				sb.WriteString("\n")
			}
		}
	}

	return sb.String()
}

// CountHunkLines returns the number of added and removed lines in a hunk
func CountHunkLines(hunk Hunk) (added, removed int) {
	lines := strings.Split(hunk.Content, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			added++
		} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
			removed++
		}
	}
	return added, removed
}
