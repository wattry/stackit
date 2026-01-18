package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadProjectConfig(t *testing.T) {
	t.Run("loads valid config with hooks", func(t *testing.T) {
		tmpDir := t.TempDir()

		configContent := `hooks:
  post-worktree-create:
    - mise trust
    - npm install
`
		err := os.WriteFile(filepath.Join(tmpDir, ProjectConfigFileName), []byte(configContent), 0600)
		require.NoError(t, err)

		cfg, err := LoadProjectConfig(tmpDir)
		require.NoError(t, err)
		assert.Equal(t, []string{"mise trust", "npm install"}, cfg.Hooks.PostWorktreeCreate)
	})

	t.Run("returns empty config when file does not exist", func(t *testing.T) {
		tmpDir := t.TempDir()

		cfg, err := LoadProjectConfig(tmpDir)
		require.NoError(t, err)
		assert.Empty(t, cfg.Hooks.PostWorktreeCreate)
	})

	t.Run("returns error for malformed YAML", func(t *testing.T) {
		tmpDir := t.TempDir()

		configContent := `hooks:
  post-worktree-create:
    - [invalid
`
		err := os.WriteFile(filepath.Join(tmpDir, ProjectConfigFileName), []byte(configContent), 0600)
		require.NoError(t, err)

		_, err = LoadProjectConfig(tmpDir)
		require.Error(t, err)
	})

	t.Run("handles empty hooks section", func(t *testing.T) {
		tmpDir := t.TempDir()

		configContent := `hooks:
  post-worktree-create: []
`
		err := os.WriteFile(filepath.Join(tmpDir, ProjectConfigFileName), []byte(configContent), 0600)
		require.NoError(t, err)

		cfg, err := LoadProjectConfig(tmpDir)
		require.NoError(t, err)
		assert.Empty(t, cfg.Hooks.PostWorktreeCreate)
	})

	t.Run("handles empty config file", func(t *testing.T) {
		tmpDir := t.TempDir()

		err := os.WriteFile(filepath.Join(tmpDir, ProjectConfigFileName), []byte(""), 0600)
		require.NoError(t, err)

		cfg, err := LoadProjectConfig(tmpDir)
		require.NoError(t, err)
		assert.Empty(t, cfg.Hooks.PostWorktreeCreate)
	})

	t.Run("handles config with only other fields", func(t *testing.T) {
		tmpDir := t.TempDir()

		configContent := `other:
  field: value
`
		err := os.WriteFile(filepath.Join(tmpDir, ProjectConfigFileName), []byte(configContent), 0600)
		require.NoError(t, err)

		cfg, err := LoadProjectConfig(tmpDir)
		require.NoError(t, err)
		assert.Empty(t, cfg.Hooks.PostWorktreeCreate)
	})
}

func TestHasPostWorktreeCreateHooks(t *testing.T) {
	t.Run("returns true when hooks exist", func(t *testing.T) {
		cfg := &ProjectConfig{
			Hooks: HooksConfig{
				PostWorktreeCreate: []string{"mise trust"},
			},
		}
		assert.True(t, cfg.HasPostWorktreeCreateHooks())
	})

	t.Run("returns false when no hooks", func(t *testing.T) {
		cfg := &ProjectConfig{}
		assert.False(t, cfg.HasPostWorktreeCreateHooks())
	})

	t.Run("returns false when hooks list is empty", func(t *testing.T) {
		cfg := &ProjectConfig{
			Hooks: HooksConfig{
				PostWorktreeCreate: []string{},
			},
		}
		assert.False(t, cfg.HasPostWorktreeCreateHooks())
	})
}
