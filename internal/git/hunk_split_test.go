package git

import (
	"strings"
	"testing"
)

func TestCanSplitHunk(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{
			name: "single addition block - not splittable",
			content: `@@ -1,3 +1,5 @@ func example()
 context
+added1
+added2
 context`,
			expected: false,
		},
		{
			name: "single removal block - not splittable",
			content: `@@ -1,5 +1,3 @@ func example()
 context
-removed1
-removed2
 context`,
			expected: false,
		},
		{
			name: "two change blocks with context between - splittable",
			content: `@@ -1,7 +1,7 @@ func example()
 context
-removed1
+added1
 middle context
-removed2
+added2
 context`,
			expected: true,
		},
		{
			name: "changes at start and end with context - splittable",
			content: `@@ -1,6 +1,6 @@ func example()
+added at start
 context line 1
 context line 2
-removed at end`,
			expected: true,
		},
		{
			name: "only context lines - not splittable",
			content: `@@ -1,3 +1,3 @@ func example()
 context1
 context2
 context3`,
			expected: false,
		},
		{
			name:     "empty hunk - not splittable",
			content:  `@@ -1,0 +1,0 @@ func example()`,
			expected: false,
		},
		{
			name: "changes only at end - not splittable",
			content: `@@ -1,4 +1,5 @@ func example()
 context
 context
+added`,
			expected: false,
		},
		{
			name: "three separate change blocks - splittable",
			content: `@@ -1,9 +1,9 @@ func example()
-first change
 context1
+second change
 context2
-third change`,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hunk := Hunk{
				File:    "test.go",
				Content: tt.content,
			}
			result := CanSplitHunk(hunk)
			if result != tt.expected {
				t.Errorf("CanSplitHunk() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestSplitHunk(t *testing.T) {
	tests := []struct {
		name            string
		content         string
		expectedCount   int
		validateContent func(t *testing.T, hunks []Hunk)
	}{
		{
			name: "non-splittable hunk returns single hunk",
			content: `@@ -1,3 +1,4 @@ func example()
 context
+added
 context`,
			expectedCount: 1,
		},
		{
			name: "two change blocks splits into two hunks",
			content: `@@ -1,7 +1,7 @@ func example()
 context1
-removed1
+added1
 middle
-removed2
+added2
 context2`,
			expectedCount: 2,
			validateContent: func(t *testing.T, hunks []Hunk) {
				// First hunk should contain the first change
				if !strings.Contains(hunks[0].Content, "removed1") {
					t.Error("First hunk should contain 'removed1'")
				}
				// Second hunk should contain the second change
				if !strings.Contains(hunks[1].Content, "removed2") {
					t.Error("Second hunk should contain 'removed2'")
				}
			},
		},
		{
			name: "three change blocks can split further",
			content: `@@ -1,10 +1,10 @@ func example()
-first
 ctx1
+second
 ctx2
-third
 ctx3`,
			expectedCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hunk := Hunk{
				File:     "test.go",
				OldStart: 1,
				NewStart: 1,
				Content:  tt.content,
			}
			result, err := SplitHunk(hunk)
			if err != nil {
				t.Fatalf("SplitHunk() error = %v", err)
			}
			if len(result) != tt.expectedCount {
				t.Errorf("SplitHunk() returned %d hunks, expected %d", len(result), tt.expectedCount)
			}
			if tt.validateContent != nil {
				tt.validateContent(t, result)
			}
		})
	}
}

func TestCountHunkLines(t *testing.T) {
	tests := []struct {
		name            string
		content         string
		expectedAdded   int
		expectedRemoved int
	}{
		{
			name: "simple add and remove",
			content: `@@ -1,3 +1,3 @@
 context
-removed
+added`,
			expectedAdded:   1,
			expectedRemoved: 1,
		},
		{
			name: "multiple adds",
			content: `@@ -1,2 +1,5 @@
 context
+added1
+added2
+added3`,
			expectedAdded:   3,
			expectedRemoved: 0,
		},
		{
			name: "only context",
			content: `@@ -1,2 +1,2 @@
 context1
 context2`,
			expectedAdded:   0,
			expectedRemoved: 0,
		},
		{
			name: "mixed changes",
			content: `@@ -1,5 +1,6 @@
 context
-old1
-old2
+new1
+new2
+new3
 context`,
			expectedAdded:   3,
			expectedRemoved: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hunk := Hunk{Content: tt.content}
			added, removed := CountHunkLines(hunk)
			if added != tt.expectedAdded {
				t.Errorf("CountHunkLines() added = %d, expected %d", added, tt.expectedAdded)
			}
			if removed != tt.expectedRemoved {
				t.Errorf("CountHunkLines() removed = %d, expected %d", removed, tt.expectedRemoved)
			}
		})
	}
}

func TestGetHunkPreview(t *testing.T) {
	content := `@@ -1,6 +1,7 @@ func example()
 line1
+added
 line2
 line3
 line4
 line5`

	hunk := Hunk{Content: content}

	// Test with maxLines = 2
	preview, total, hasMore := GetHunkPreview(hunk, 2)
	if total != 6 {
		t.Errorf("GetHunkPreview() totalLines = %d, expected 6", total)
	}
	if !hasMore {
		t.Error("GetHunkPreview() hasMore should be true")
	}
	lines := strings.Split(preview, "\n")
	if len(lines) != 2 {
		t.Errorf("GetHunkPreview() preview has %d lines, expected 2", len(lines))
	}

	// Test with maxLines >= total
	preview, _, hasMore = GetHunkPreview(hunk, 10)
	if hasMore {
		t.Error("GetHunkPreview() hasMore should be false when maxLines >= total")
	}
	lines = strings.Split(preview, "\n")
	if len(lines) != 6 {
		t.Errorf("GetHunkPreview() preview has %d lines, expected 6", len(lines))
	}
}

func TestGetHunkHeader(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "simple header",
			content:  "@@ -1,3 +1,4 @@\n context\n+added",
			expected: "@@ -1,3 +1,4 @@",
		},
		{
			name:     "header with function context",
			content:  "@@ -10,5 +10,8 @@ func parseConfig()\n context",
			expected: "@@ -10,5 +10,8 @@ func parseConfig()",
		},
		{
			name:     "no header - fallback to metadata",
			content:  " context only",
			expected: "@@ -0,0 +0,0 @@",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hunk := Hunk{
				Content:  tt.content,
				OldStart: 0,
				OldCount: 0,
				NewStart: 0,
				NewCount: 0,
			}
			result := GetHunkHeader(hunk)
			if result != tt.expected {
				t.Errorf("GetHunkHeader() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestBuildPatchFromHunks(t *testing.T) {
	hunks := []Hunk{
		{
			File:      "file1.go",
			IndexLine: "index abc123..def456 100644",
			Content:   "@@ -1,3 +1,4 @@\n context\n+added\n context",
		},
		{
			File:    "file1.go",
			Content: "@@ -10,2 +11,3 @@\n context\n+another add\n context",
		},
		{
			File:    "file2.go",
			Content: "@@ -1,1 +1,2 @@\n+new line",
		},
	}

	patch := BuildPatchFromHunks(hunks)

	// Should contain both file headers
	if !strings.Contains(patch, "diff --git a/file1.go b/file1.go") {
		t.Error("Patch should contain file1.go header")
	}
	if !strings.Contains(patch, "diff --git a/file2.go b/file2.go") {
		t.Error("Patch should contain file2.go header")
	}

	// Should contain index line
	if !strings.Contains(patch, "index abc123..def456 100644") {
		t.Error("Patch should contain index line")
	}

	// Should contain all hunks
	if !strings.Contains(patch, "@@ -1,3 +1,4 @@") {
		t.Error("Patch should contain first hunk header")
	}
	if !strings.Contains(patch, "@@ -10,2 +11,3 @@") {
		t.Error("Patch should contain second hunk header")
	}
	if !strings.Contains(patch, "@@ -1,1 +1,2 @@") {
		t.Error("Patch should contain third hunk header")
	}

	// Empty hunks should return empty string
	if BuildPatchFromHunks(nil) != "" {
		t.Error("BuildPatchFromHunks(nil) should return empty string")
	}
	if BuildPatchFromHunks([]Hunk{}) != "" {
		t.Error("BuildPatchFromHunks([]) should return empty string")
	}
}

func TestParseDiffOutput(t *testing.T) {
	tests := []struct {
		name          string
		diff          string
		expectedCount int
		validateHunks func(t *testing.T, hunks []Hunk)
	}{
		{
			name:          "empty diff",
			diff:          "",
			expectedCount: 0,
		},
		{
			name: "single hunk",
			diff: `diff --git a/file.go b/file.go
index abc123..def456 100644
--- a/file.go
+++ b/file.go
@@ -1,3 +1,4 @@
 context
+added
 context`,
			expectedCount: 1,
			validateHunks: func(t *testing.T, hunks []Hunk) {
				if hunks[0].File != "file.go" {
					t.Errorf("Expected file 'file.go', got '%s'", hunks[0].File)
				}
				if hunks[0].OldStart != 1 || hunks[0].NewStart != 1 {
					t.Errorf("Unexpected line numbers: OldStart=%d, NewStart=%d", hunks[0].OldStart, hunks[0].NewStart)
				}
				if hunks[0].IndexLine != "index abc123..def456 100644" {
					t.Errorf("Expected index line, got '%s'", hunks[0].IndexLine)
				}
			},
		},
		{
			name: "binary file",
			diff: `diff --git a/image.png b/image.png
index abc123..def456 100644
Binary files a/image.png and b/image.png differ`,
			expectedCount: 1,
			validateHunks: func(t *testing.T, hunks []Hunk) {
				if hunks[0].File != "image.png" {
					t.Errorf("Expected file 'image.png', got '%s'", hunks[0].File)
				}
				if !hunks[0].Binary {
					t.Error("Expected Binary to be true")
				}
			},
		},
		{
			name: "multiple files with multiple hunks",
			diff: `diff --git a/file1.go b/file1.go
index abc..def 100644
--- a/file1.go
+++ b/file1.go
@@ -1,2 +1,3 @@
 line1
+added
 line2
@@ -10,2 +11,3 @@
 line10
+added2
 line11
diff --git a/file2.go b/file2.go
--- a/file2.go
+++ b/file2.go
@@ -5,1 +5,2 @@
 context
+new line`,
			expectedCount: 3,
			validateHunks: func(t *testing.T, hunks []Hunk) {
				if hunks[0].File != "file1.go" {
					t.Errorf("First hunk should be file1.go, got '%s'", hunks[0].File)
				}
				if hunks[1].File != "file1.go" {
					t.Errorf("Second hunk should be file1.go, got '%s'", hunks[1].File)
				}
				if hunks[2].File != "file2.go" {
					t.Errorf("Third hunk should be file2.go, got '%s'", hunks[2].File)
				}
			},
		},
		{
			name: "mixed binary and text files",
			diff: `diff --git a/image.png b/image.png
Binary files a/image.png and b/image.png differ
diff --git a/code.go b/code.go
--- a/code.go
+++ b/code.go
@@ -1,1 +1,2 @@
 line1
+added`,
			expectedCount: 2,
			validateHunks: func(t *testing.T, hunks []Hunk) {
				if !hunks[0].Binary {
					t.Error("First hunk should be binary")
				}
				if hunks[1].Binary {
					t.Error("Second hunk should not be binary")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hunks, err := ParseDiffOutput(tt.diff)
			if err != nil {
				t.Fatalf("ParseDiffOutput() error = %v", err)
			}
			if len(hunks) != tt.expectedCount {
				t.Errorf("ParseDiffOutput() returned %d hunks, expected %d", len(hunks), tt.expectedCount)
			}
			if tt.validateHunks != nil {
				tt.validateHunks(t, hunks)
			}
		})
	}
}

func TestBuildPatchFromHunks_Binary(t *testing.T) {
	// Test that binary files produce correct patch format
	hunks := []Hunk{
		{
			File:      "image.png",
			IndexLine: "index abc123..def456 100644",
			Content:   "Binary files a/image.png and b/image.png differ",
			Binary:    true,
		},
	}

	patch := BuildPatchFromHunks(hunks)

	// Binary patches should have diff header but no ---/+++ lines
	if !strings.Contains(patch, "diff --git a/image.png b/image.png") {
		t.Error("Patch should contain diff header")
	}
	if !strings.Contains(patch, "index abc123..def456") {
		t.Error("Patch should contain index line")
	}
	if strings.Contains(patch, "--- a/image.png") {
		t.Error("Binary patch should not contain --- line")
	}
	if strings.Contains(patch, "+++ b/image.png") {
		t.Error("Binary patch should not contain +++ line")
	}
	if !strings.Contains(patch, "Binary files") {
		t.Error("Patch should contain binary marker")
	}
}

func TestBuildPatchFromHunks_MixedBinaryAndText(t *testing.T) {
	hunks := []Hunk{
		{
			File:      "image.png",
			IndexLine: "index abc..def 100644",
			Content:   "Binary files a/image.png and b/image.png differ",
			Binary:    true,
		},
		{
			File:      "code.go",
			IndexLine: "index 123..456 100644",
			Content:   "@@ -1,1 +1,2 @@\n line1\n+added",
			Binary:    false,
		},
	}

	patch := BuildPatchFromHunks(hunks)

	// Binary file should not have ---/+++ lines
	if strings.Contains(patch, "--- a/image.png") {
		t.Error("Binary file should not have --- line")
	}

	// Text file should have ---/+++ lines
	if !strings.Contains(patch, "--- a/code.go") {
		t.Error("Text file should have --- line")
	}
	if !strings.Contains(patch, "+++ b/code.go") {
		t.Error("Text file should have +++ line")
	}
}

func TestSplitHunk_TrailingEmptyLines(t *testing.T) {
	// Test that hunks with trailing empty lines are handled correctly
	content := `@@ -1,5 +1,5 @@ func example()
-removed
 context1
+added
 context2
`
	hunk := Hunk{
		File:     "test.go",
		OldStart: 1,
		NewStart: 1,
		Content:  content,
	}

	result, err := SplitHunk(hunk)
	if err != nil {
		t.Fatalf("SplitHunk() error = %v", err)
	}

	// Should still be able to split despite trailing empty line
	if len(result) != 2 {
		t.Errorf("SplitHunk() returned %d hunks, expected 2", len(result))
	}
}

func TestCanSplitHunk_NoNewlineAtEnd(t *testing.T) {
	// Test hunk that ends with "\ No newline at end of file"
	content := `@@ -1,3 +1,3 @@ func example()
-old
+new
 context
\ No newline at end of file`
	hunk := Hunk{
		File:    "test.go",
		Content: content,
	}

	// This hunk only has one change block, so it shouldn't be splittable
	if CanSplitHunk(hunk) {
		t.Error("Hunk with single change block should not be splittable")
	}
}
