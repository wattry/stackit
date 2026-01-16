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
