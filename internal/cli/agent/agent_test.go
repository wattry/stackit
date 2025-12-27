package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/internal/git"
)

func TestAgentInit_Local(t *testing.T) {
	// Create temp directory
	tempDir := t.TempDir()

	// Initialize git repo
	err := os.Chdir(tempDir)
	require.NoError(t, err, "should change to temp directory")

	_, err = git.RunGitCommand("init")
	require.NoError(t, err, "should initialize git repo")

	// Run agent init --local
	err = runAgentInit(true, false)
	require.NoError(t, err, "agent init should succeed")

	// Verify directory structure
	t.Run("directories created", func(t *testing.T) {
		dirs := []string{
			".claude/skills/stackit",
			".claude/commands",
			".cursor/rules",
		}

		for _, dir := range dirs {
			path := filepath.Join(tempDir, dir)
			info, err := os.Stat(path)
			require.NoError(t, err, "directory should exist: %s", dir)
			require.True(t, info.IsDir(), "should be a directory: %s", dir)
		}
	})

	// Verify skill files
	t.Run("skill files created", func(t *testing.T) {
		files := map[string][]string{
			".claude/skills/stackit/SKILL.md": {
				"name: stackit",
				"description: Manage stacked Git branches",
				"allowed-tools: Bash(stackit:*)",
				"version: 1.0.0",
				"# Stackit - Stacked Branch Management",
			},
			".claude/skills/stackit/reference.md": {
				"# Stackit Command Reference",
				"## Navigation Commands",
				"stackit log",
				"stackit create",
			},
		}

		for file, expectedContent := range files {
			path := filepath.Join(tempDir, file)
			content, err := os.ReadFile(path)
			require.NoError(t, err, "should read file: %s", file)

			for _, expected := range expectedContent {
				require.Contains(t, string(content), expected, "file should contain expected content: %s", file)
			}
		}
	})

	// Verify command files
	t.Run("command files created", func(t *testing.T) {
		commands := []string{
			"stack-status.md",
			"stack-create.md",
			"stack-submit.md",
			"stack-fix.md",
			"stack-sync.md",
			"stack-restack.md",
		}

		for _, cmd := range commands {
			path := filepath.Join(tempDir, ".claude/commands", cmd)
			content, err := os.ReadFile(path)
			require.NoError(t, err, "should read command file: %s", cmd)

			// Verify frontmatter
			require.Contains(t, string(content), "---", "should have frontmatter")
			require.Contains(t, string(content), "description:", "should have description")
			require.Contains(t, string(content), "allowed-tools:", "should have allowed-tools")

			// Verify has instructions
			require.Contains(t, string(content), "## Instructions", "should have instructions section")
		}
	})

	// Verify Cursor rules
	t.Run("cursor rules created", func(t *testing.T) {
		path := filepath.Join(tempDir, ".cursor/rules/stackit.md")
		content, err := os.ReadFile(path)
		require.NoError(t, err, "should read cursor rules")

		require.Contains(t, string(content), "# Stackit Agent Rules")
		require.Contains(t, string(content), "stackit create")
		require.Contains(t, string(content), "stackit submit")
	})
}

func TestAgentInit_VersionCheck(t *testing.T) {
	tempDir := t.TempDir()

	err := os.Chdir(tempDir)
	require.NoError(t, err, "should change to temp directory")

	_, err = git.RunGitCommand("init")
	require.NoError(t, err, "should initialize git repo")

	// First installation
	err = runAgentInit(true, false)
	require.NoError(t, err, "first install should succeed")

	// Modify version to simulate older version
	skillPath := filepath.Join(tempDir, ".claude/skills/stackit/SKILL.md")
	content, err := os.ReadFile(skillPath)
	require.NoError(t, err, "should read SKILL.md")

	oldContent := strings.Replace(string(content), "version: 1.0.0", "version: 0.9.0", 1)
	err = os.WriteFile(skillPath, []byte(oldContent), 0600)
	require.NoError(t, err, "should write modified SKILL.md")

	// Try to install again without force
	err = runAgentInit(true, false)
	require.Error(t, err, "should fail without force flag")
	require.Contains(t, err.Error(), "existing installation found")

	// Install with force flag
	err = runAgentInit(true, true)
	require.NoError(t, err, "should succeed with force flag")

	// Verify version was updated
	content, err = os.ReadFile(skillPath)
	require.NoError(t, err, "should read updated SKILL.md")
	require.Contains(t, string(content), "version: 1.0.0", "should have updated version")
	require.NotContains(t, string(content), "version: 0.9.0", "should not have old version")
}

func TestAgentInit_NotGitRepo(t *testing.T) {
	tempDir := t.TempDir()

	err := os.Chdir(tempDir)
	require.NoError(t, err, "should change to temp directory")

	// Try to run agent init --local without git repo
	err = runAgentInit(true, false)
	require.Error(t, err, "should fail when not in git repo")
	require.Contains(t, err.Error(), "not a git repository")
}

func TestExtractVersion(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name: "valid version in frontmatter",
			content: `---
name: stackit
version: 1.2.3
---
Content here`,
			expected: "1.2.3",
		},
		{
			name: "no version",
			content: `---
name: stackit
---
Content here`,
			expected: "",
		},
		{
			name: "version with spaces",
			content: `---
name: stackit
version:   2.0.0
---
Content here`,
			expected: "2.0.0",
		},
		{
			name:     "no frontmatter",
			content:  "Just content",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractVersion(tt.content)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestGetDisplayPath(t *testing.T) {
	tests := []struct {
		name     string
		baseDir  string
		local    bool
		expected string
	}{
		{
			name:     "global installation",
			baseDir:  "/home/user",
			local:    false,
			expected: "~",
		},
		{
			name:     "local installation",
			baseDir:  "/home/user/project",
			local:    true,
			expected: ".",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getDisplayPath(tt.baseDir, tt.local)
			require.Equal(t, tt.expected, result)
		})
	}
}
