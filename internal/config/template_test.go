package config

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

var update = flag.Bool("update", false, "update golden files")

func TestGenerateConfigTemplate_Golden(t *testing.T) {
	t.Parallel()

	template := GenerateConfigTemplate()
	goldenPath := filepath.Join("testdata", "config_template.golden")

	if *update {
		err := os.WriteFile(goldenPath, []byte(template), 0644)
		require.NoError(t, err, "failed to update golden file")
		t.Logf("Updated golden file: %s", goldenPath)
		return
	}

	expected, err := os.ReadFile(goldenPath)
	require.NoError(t, err, "failed to read golden file (run with -update to create)")

	if template != string(expected) {
		t.Errorf("Template output does not match golden file.\n\n%s\n\nRun with -update flag to update the golden file.",
			diffStrings(string(expected), template))
	}
}

func TestGenerateConfigTemplate_Deterministic(t *testing.T) {
	t.Parallel()

	// Generate template multiple times and verify it's identical
	template1 := GenerateConfigTemplate()
	template2 := GenerateConfigTemplate()

	require.Equal(t, template1, template2, "Template generation should be deterministic")
}

func TestGenerateConfigTemplate_AllOptionsPresent(t *testing.T) {
	t.Parallel()

	template := GenerateConfigTemplate()

	// Verify all config options appear in the template
	for _, opt := range Options {
		// Extract the last part of the YAML path (the actual key name)
		parts := strings.Split(opt.YAMLPath, ".")
		key := parts[len(parts)-1]

		// The key should appear somewhere in the template
		if !strings.Contains(template, key) {
			t.Errorf("Template should contain option key: %s (from %s)", key, opt.YAMLPath)
		}
	}
}

func TestGenerateConfigTemplate_ValidYAMLComments(t *testing.T) {
	t.Parallel()

	template := GenerateConfigTemplate()

	// All non-empty lines should be valid YAML comments
	lines := strings.Split(template, "\n")
	for i, line := range lines {
		if line == "" {
			continue
		}
		// Should start with # (comment)
		if !strings.HasPrefix(line, "#") && !strings.HasPrefix(strings.TrimSpace(line), "#") {
			t.Errorf("Line %d should be a comment: %q", i+1, line)
		}
	}
}

func TestGenerateYAMLExample(t *testing.T) {
	t.Parallel()

	example := GenerateYAMLExample()

	// Should be valid YAML structure (uncommented)
	require.Contains(t, example, "trunk: main")
	require.Contains(t, example, "branch:")
	require.Contains(t, example, "submit:")
	require.Contains(t, example, "navigation:")
	require.Contains(t, example, "hooks:")

	// Should have header comment
	require.Contains(t, example, "# .stackit.yaml")
}

func TestGenerateYAMLExample_Deterministic(t *testing.T) {
	t.Parallel()

	example1 := GenerateYAMLExample()
	example2 := GenerateYAMLExample()

	require.Equal(t, example1, example2, "YAML example generation should be deterministic")
}

func TestGenerateYAMLExample_IncludesAllSections(t *testing.T) {
	t.Parallel()

	example := GenerateYAMLExample()

	// Verify all section groups from Options are represented
	sections := make(map[string]bool)
	for _, opt := range Options {
		sections[opt.Section] = true
	}

	for section := range sections {
		// Each section should have at least one key mentioned
		var found bool
		for _, opt := range Options {
			if opt.Section == section {
				// Extract top-level key from YAML path
				parts := strings.Split(opt.YAMLPath, ".")
				key := parts[0]
				if strings.Contains(example, key+":") || strings.Contains(example, key+" ") {
					found = true
					break
				}
			}
		}
		require.True(t, found, "Section %q should be represented in YAML example", section)
	}
}

// diffStrings returns a line-by-line diff of two strings for debugging.
func diffStrings(expected, actual string) string {
	expectedLines := strings.Split(expected, "\n")
	actualLines := strings.Split(actual, "\n")

	var diff strings.Builder
	diff.WriteString("Diff (- expected, + actual):\n")

	maxLines := max(len(expectedLines), len(actualLines))

	for i := range maxLines {
		var expLine, actLine string
		if i < len(expectedLines) {
			expLine = expectedLines[i]
		}
		if i < len(actualLines) {
			actLine = actualLines[i]
		}

		if expLine != actLine {
			if expLine != "" {
				diff.WriteString("- ")
				diff.WriteString(expLine)
				diff.WriteString("\n")
			}
			if actLine != "" {
				diff.WriteString("+ ")
				diff.WriteString(actLine)
				diff.WriteString("\n")
			}
		}
	}

	return diff.String()
}
