package integrations

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReplaceWorkflowBlock(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		content  string
		newBlock string
		expected string
	}{
		{
			name:     "replace block in middle of file",
			content:  "# Header\n\nSome content\n\n<!-- stackit:start -->\nold block\n<!-- stackit:end -->\n\n## Footer\n\nMore content",
			newBlock: "<!-- stackit:start -->\nnew block\n<!-- stackit:end -->",
			expected: "# Header\n\nSome content\n\n<!-- stackit:start -->\nnew block\n<!-- stackit:end -->\n## Footer\n\nMore content",
		},
		{
			name:     "replace block at start of file",
			content:  "<!-- stackit:start -->\nold block\n<!-- stackit:end -->\n\n## Content",
			newBlock: "<!-- stackit:start -->\nnew block\n<!-- stackit:end -->",
			expected: "<!-- stackit:start -->\nnew block\n<!-- stackit:end -->\n## Content",
		},
		{
			name:     "replace block at end of file",
			content:  "# Header\n\n<!-- stackit:start -->\nold block\n<!-- stackit:end -->",
			newBlock: "<!-- stackit:start -->\nnew block\n<!-- stackit:end -->",
			expected: "# Header\n\n<!-- stackit:start -->\nnew block\n<!-- stackit:end -->",
		},
		{
			name:     "returns original when start marker missing",
			content:  "# Header\n\nold block\n<!-- stackit:end -->\n",
			newBlock: "<!-- stackit:start -->\nnew block\n<!-- stackit:end -->",
			expected: "# Header\n\nold block\n<!-- stackit:end -->\n",
		},
		{
			name:     "returns original when end marker missing",
			content:  "# Header\n\n<!-- stackit:start -->\nold block\n",
			newBlock: "<!-- stackit:start -->\nnew block\n<!-- stackit:end -->",
			expected: "# Header\n\n<!-- stackit:start -->\nold block\n",
		},
		{
			name:     "returns original when end marker before start marker",
			content:  "# Header\n\n<!-- stackit:end -->\n\n<!-- stackit:start -->\n",
			newBlock: "<!-- stackit:start -->\nnew block\n<!-- stackit:end -->",
			expected: "# Header\n\n<!-- stackit:end -->\n\n<!-- stackit:start -->\n",
		},
		{
			name:     "handles empty before section",
			content:  "<!-- stackit:start -->\nold\n<!-- stackit:end -->\n\nAfter",
			newBlock: "<!-- stackit:start -->\nnew\n<!-- stackit:end -->",
			expected: "<!-- stackit:start -->\nnew\n<!-- stackit:end -->\nAfter",
		},
		{
			name:     "handles empty after section",
			content:  "Before\n\n<!-- stackit:start -->\nold\n<!-- stackit:end -->",
			newBlock: "<!-- stackit:start -->\nnew\n<!-- stackit:end -->",
			expected: "Before\n\n<!-- stackit:start -->\nnew\n<!-- stackit:end -->",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := replaceWorkflowBlock(tt.content, tt.newBlock)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestCheckAgentsFile(t *testing.T) {
	t.Parallel()

	t.Run("returns not exists when file missing", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()

		info := checkAgentsFile(tmpDir, "CLAUDE.md")

		require.False(t, info.exists)
		require.False(t, info.hasBlock)
		require.Empty(t, info.content)
		require.NoError(t, info.readErr)
	})

	t.Run("returns exists without block", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "CLAUDE.md")
		err := os.WriteFile(filePath, []byte("# My Project\n\nSome content"), 0600)
		require.NoError(t, err)

		info := checkAgentsFile(tmpDir, "CLAUDE.md")

		require.True(t, info.exists)
		require.False(t, info.hasBlock)
		require.Equal(t, "# My Project\n\nSome content", info.content)
		require.NoError(t, info.readErr)
	})

	t.Run("returns exists with block", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "AGENTS.md")
		content := "# Project\n\n<!-- stackit:start -->\nblock\n<!-- stackit:end -->\n"
		err := os.WriteFile(filePath, []byte(content), 0600)
		require.NoError(t, err)

		info := checkAgentsFile(tmpDir, "AGENTS.md")

		require.True(t, info.exists)
		require.True(t, info.hasBlock)
		require.Equal(t, content, info.content)
		require.NoError(t, info.readErr)
	})
}

func TestDiscoverAgentsFiles(t *testing.T) {
	t.Parallel()

	t.Run("neither file exists", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()

		claude, agents := discoverAgentsFiles(tmpDir)

		require.False(t, claude.exists)
		require.False(t, agents.exists)
	})

	t.Run("only CLAUDE.md exists", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		err := os.WriteFile(filepath.Join(tmpDir, "CLAUDE.md"), []byte("content"), 0600)
		require.NoError(t, err)

		claude, agents := discoverAgentsFiles(tmpDir)

		require.True(t, claude.exists)
		require.False(t, agents.exists)
	})

	t.Run("only AGENTS.md exists", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		err := os.WriteFile(filepath.Join(tmpDir, "AGENTS.md"), []byte("content"), 0600)
		require.NoError(t, err)

		claude, agents := discoverAgentsFiles(tmpDir)

		require.False(t, claude.exists)
		require.True(t, agents.exists)
	})

	t.Run("both files exist", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		err := os.WriteFile(filepath.Join(tmpDir, "CLAUDE.md"), []byte("claude"), 0600)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(tmpDir, "AGENTS.md"), []byte("agents"), 0600)
		require.NoError(t, err)

		claude, agents := discoverAgentsFiles(tmpDir)

		require.True(t, claude.exists)
		require.True(t, agents.exists)
		require.Equal(t, "claude", claude.content)
		require.Equal(t, "agents", agents.content)
	})
}

func TestInstallWorkflowBlock(t *testing.T) {
	t.Parallel()

	t.Run("appends to empty file", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		fileInfo := agentsFileInfo{
			name:    "CLAUDE.md",
			exists:  false,
			content: "",
		}

		installed, path, err := installWorkflowBlock(tmpDir, "CLAUDE.md", fileInfo, false)

		require.NoError(t, err)
		require.True(t, installed)
		require.Equal(t, "CLAUDE.md", path)

		content, err := os.ReadFile(filepath.Join(tmpDir, "CLAUDE.md"))
		require.NoError(t, err)
		require.Contains(t, string(content), "<!-- stackit:start -->")
		require.Contains(t, string(content), "<!-- stackit:end -->")
	})

	t.Run("appends to existing file without block", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		existingContent := "# My Project\n\nSome content"
		fileInfo := agentsFileInfo{
			name:    "CLAUDE.md",
			exists:  true,
			content: existingContent,
		}

		installed, path, err := installWorkflowBlock(tmpDir, "CLAUDE.md", fileInfo, false)

		require.NoError(t, err)
		require.True(t, installed)
		require.Equal(t, "CLAUDE.md", path)

		content, err := os.ReadFile(filepath.Join(tmpDir, "CLAUDE.md"))
		require.NoError(t, err)
		require.Contains(t, string(content), "# My Project")
		require.Contains(t, string(content), "<!-- stackit:start -->")
	})

	t.Run("errors when block exists without force", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		fileInfo := agentsFileInfo{
			name:     "CLAUDE.md",
			exists:   true,
			hasBlock: true,
			content:  "<!-- stackit:start -->\nold\n<!-- stackit:end -->",
		}

		installed, path, err := installWorkflowBlock(tmpDir, "CLAUDE.md", fileInfo, false)

		require.Error(t, err)
		require.Contains(t, err.Error(), "already exists")
		require.Contains(t, err.Error(), "--force")
		require.False(t, installed)
		require.Equal(t, "CLAUDE.md", path)
	})

	t.Run("replaces block with force", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		fileInfo := agentsFileInfo{
			name:     "AGENTS.md",
			exists:   true,
			hasBlock: true,
			content:  "# Header\n\n<!-- stackit:start -->\nold block\n<!-- stackit:end -->\n\n# Footer",
		}

		installed, path, err := installWorkflowBlock(tmpDir, "AGENTS.md", fileInfo, true)

		require.NoError(t, err)
		require.True(t, installed)
		require.Equal(t, "AGENTS.md", path)

		content, err := os.ReadFile(filepath.Join(tmpDir, "AGENTS.md"))
		require.NoError(t, err)
		require.Contains(t, string(content), "# Header")
		require.Contains(t, string(content), "# Footer")
		require.Contains(t, string(content), "<!-- stackit:start -->")
		require.NotContains(t, string(content), "old block")
	})

	t.Run("adds trailing newline when missing", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		fileInfo := agentsFileInfo{
			name:    "CLAUDE.md",
			exists:  true,
			content: "# Header", // No trailing newline
		}

		installed, _, err := installWorkflowBlock(tmpDir, "CLAUDE.md", fileInfo, false)

		require.NoError(t, err)
		require.True(t, installed)

		content, err := os.ReadFile(filepath.Join(tmpDir, "CLAUDE.md"))
		require.NoError(t, err)
		// Should have proper spacing between original content and block
		require.Contains(t, string(content), "# Header\n\n<!-- stackit:start -->")
	})
}
