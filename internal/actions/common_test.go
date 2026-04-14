package actions

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFindConflictMarkerSections(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		content  string
		expected []conflictMarkerSection
	}{
		{
			name: "single section",
			content: strings.Join([]string{
				"before",
				"<<<<<<< HEAD",
				"base",
				"=======",
				"branch",
				">>>>>>> feature",
				"after",
			}, "\n"),
			expected: []conflictMarkerSection{{StartLine: 2, EndLine: 6}},
		},
		{
			name: "multiple sections",
			content: strings.Join([]string{
				"<<<<<<< HEAD",
				"a",
				"=======",
				"b",
				">>>>>>> branch-a",
				"middle",
				"<<<<<<< HEAD",
				"c",
				"=======",
				"d",
				">>>>>>> branch-b",
			}, "\n"),
			expected: []conflictMarkerSection{
				{StartLine: 1, EndLine: 5},
				{StartLine: 7, EndLine: 11},
			},
		},
		{
			name: "no complete section",
			content: strings.Join([]string{
				"<<<<<<< HEAD",
				"base",
				"=======",
				"branch",
			}, "\n"),
			expected: []conflictMarkerSection{},
		},
		{
			name: "diff3 style with ancestor marker",
			content: strings.Join([]string{
				"<<<<<<< HEAD",
				"ours",
				"||||||| common ancestor",
				"base",
				"=======",
				"theirs",
				">>>>>>> branch",
			}, "\n"),
			expected: []conflictMarkerSection{{StartLine: 1, EndLine: 7}},
		},
		{
			name: "unterminated section then complete section",
			content: strings.Join([]string{
				"<<<<<<< HEAD",
				"unterminated",
				"<<<<<<< HEAD",
				"a",
				"=======",
				"b",
				">>>>>>> branch",
			}, "\n"),
			expected: []conflictMarkerSection{{StartLine: 3, EndLine: 7}},
		},
		{
			name: "separator before any start marker is ignored",
			content: strings.Join([]string{
				"heading",
				"=======",
				"more text",
				"<<<<<<< HEAD",
				"a",
				"=======",
				"b",
				">>>>>>> branch",
			}, "\n"),
			expected: []conflictMarkerSection{{StartLine: 4, EndLine: 8}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tt.expected, findConflictMarkerSections(tt.content))
		})
	}
}

func TestConflictMarkerSectionsForFile(t *testing.T) {
	t.Parallel()

	t.Run("reads file via relative path joined to repo root", func(t *testing.T) {
		t.Parallel()
		repoRoot := t.TempDir()
		rel := filepath.Join("sub", "file.txt")
		require.NoError(t, os.MkdirAll(filepath.Join(repoRoot, "sub"), 0o755))
		content := strings.Join([]string{
			"<<<<<<< HEAD",
			"a",
			"=======",
			"b",
			">>>>>>> branch",
		}, "\n")
		require.NoError(t, os.WriteFile(filepath.Join(repoRoot, rel), []byte(content), 0o644))

		sections, err := conflictMarkerSectionsForFile(repoRoot, rel)
		require.NoError(t, err)
		require.Equal(t, []conflictMarkerSection{{StartLine: 1, EndLine: 5}}, sections)
	})

	t.Run("absolute path is used as-is", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		abs := filepath.Join(dir, "file.txt")
		require.NoError(t, os.WriteFile(abs, []byte("no markers here"), 0o644))

		sections, err := conflictMarkerSectionsForFile("/unrelated/root", abs)
		require.NoError(t, err)
		require.Empty(t, sections)
	})

	t.Run("missing file returns error", func(t *testing.T) {
		t.Parallel()
		_, err := conflictMarkerSectionsForFile(t.TempDir(), "does-not-exist.txt")
		require.Error(t, err)
	})
}
