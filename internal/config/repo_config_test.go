package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/testhelpers"
)

func TestConfigSubmitFooter(t *testing.T) {
	t.Parallel()

	t.Run("returns true when config does not exist", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, nil)

		cfg, err := LoadConfig(scene.Dir)
		require.NoError(t, err)
		require.True(t, cfg.SubmitFooter())
	})

	t.Run("returns true when config exists but submit.footer is not set", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, nil)

		// Create config file without submit.footer
		configPath := filepath.Join(scene.Dir, ".git", ".stackit_config")
		config := &RepoConfig{
			Trunk: stringPtr("main"),
		}
		configJSON, err := json.MarshalIndent(config, "", "  ")
		require.NoError(t, err)
		err = os.WriteFile(configPath, configJSON, 0600)
		require.NoError(t, err)

		cfg, err := LoadConfig(scene.Dir)
		require.NoError(t, err)
		require.True(t, cfg.SubmitFooter())
	})

	t.Run("returns true when config has submit.footer set to true", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, nil)

		// Create config file with submit.footer = true
		configPath := filepath.Join(scene.Dir, ".git", ".stackit_config")
		enabled := true
		config := &RepoConfig{
			Trunk:        stringPtr("main"),
			SubmitFooter: &enabled,
		}
		configJSON, err := json.MarshalIndent(config, "", "  ")
		require.NoError(t, err)
		err = os.WriteFile(configPath, configJSON, 0600)
		require.NoError(t, err)

		cfg, err := LoadConfig(scene.Dir)
		require.NoError(t, err)
		require.True(t, cfg.SubmitFooter())
	})

	t.Run("returns false when config has submit.footer set to false", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, nil)

		// Create config file with submit.footer = false
		configPath := filepath.Join(scene.Dir, ".git", ".stackit_config")
		enabled := false
		config := &RepoConfig{
			Trunk:        stringPtr("main"),
			SubmitFooter: &enabled,
		}
		configJSON, err := json.MarshalIndent(config, "", "  ")
		require.NoError(t, err)
		err = os.WriteFile(configPath, configJSON, 0600)
		require.NoError(t, err)

		cfg, err := LoadConfig(scene.Dir)
		require.NoError(t, err)
		require.False(t, cfg.SubmitFooter())
	})
}

func TestConfigSetSubmitFooter(t *testing.T) {
	t.Parallel()

	t.Run("sets submit.footer to true", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, nil)

		cfg, err := LoadConfig(scene.Dir)
		require.NoError(t, err)
		cfg.SetSubmitFooter(true)
		err = cfg.Save()
		require.NoError(t, err)

		// Verify config was written
		config, err := GetRepoConfig(scene.Dir)
		require.NoError(t, err)
		require.NotNil(t, config.SubmitFooter)
		require.True(t, *config.SubmitFooter)

		// Verify Config.SubmitFooter returns true
		cfg2, err := LoadConfig(scene.Dir)
		require.NoError(t, err)
		require.True(t, cfg2.SubmitFooter())
	})

	t.Run("sets submit.footer to false", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, nil)

		cfg, err := LoadConfig(scene.Dir)
		require.NoError(t, err)
		cfg.SetSubmitFooter(false)
		err = cfg.Save()
		require.NoError(t, err)

		// Verify config was written
		config, err := GetRepoConfig(scene.Dir)
		require.NoError(t, err)
		require.NotNil(t, config.SubmitFooter)
		require.False(t, *config.SubmitFooter)

		// Verify Config.SubmitFooter returns false
		cfg2, err := LoadConfig(scene.Dir)
		require.NoError(t, err)
		require.False(t, cfg2.SubmitFooter())
	})

	t.Run("updates existing config without overwriting other fields", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, nil)

		// Create initial config with trunk
		configPath := filepath.Join(scene.Dir, ".git", ".stackit_config")
		initialConfig := &RepoConfig{
			Trunk: stringPtr("main"),
		}
		configJSON, err := json.MarshalIndent(initialConfig, "", "  ")
		require.NoError(t, err)
		err = os.WriteFile(configPath, configJSON, 0600)
		require.NoError(t, err)

		// Set submit.footer
		cfg, err := LoadConfig(scene.Dir)
		require.NoError(t, err)
		cfg.SetSubmitFooter(false)
		err = cfg.Save()
		require.NoError(t, err)

		// Verify both fields are present
		config, err := GetRepoConfig(scene.Dir)
		require.NoError(t, err)
		require.NotNil(t, config.Trunk)
		require.Equal(t, "main", *config.Trunk)
		require.NotNil(t, config.SubmitFooter)
		require.False(t, *config.SubmitFooter)
	})
}

func TestConfigWorktreeSupport(t *testing.T) {
	t.Parallel()

	t.Run("loads config from worktree using main repo git dir", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, testhelpers.BasicSceneSetup)

		// Create and save config in main repo
		cfg, err := LoadConfig(scene.Dir)
		require.NoError(t, err)
		cfg.SetTrunk("main")
		cfg.SetSubmitFooter(false)
		err = cfg.Save()
		require.NoError(t, err)

		// Create a branch for the worktree
		err = scene.Repo.CreateBranch("feature-branch")
		require.NoError(t, err)

		// Create worktree (normalize path for macOS /var -> /private/var symlink)
		tmpDir := t.TempDir()
		worktreePath, err := filepath.EvalSymlinks(tmpDir)
		require.NoError(t, err)
		worktreePath = filepath.Join(worktreePath, "worktree")
		err = scene.Repo.RunGitCommand("worktree", "add", worktreePath, "feature-branch")
		require.NoError(t, err)

		// Load config from worktree - should find main repo's config
		worktreeCfg, err := LoadConfig(worktreePath)
		require.NoError(t, err)
		require.True(t, worktreeCfg.IsInitialized())
		require.Equal(t, "main", worktreeCfg.Trunk())
		require.False(t, worktreeCfg.SubmitFooter())
	})

	t.Run("saves config from worktree to main repo git dir", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, testhelpers.BasicSceneSetup)

		// Initialize config in main repo
		cfg, err := LoadConfig(scene.Dir)
		require.NoError(t, err)
		cfg.SetTrunk("main")
		err = cfg.Save()
		require.NoError(t, err)

		// Create a branch for the worktree
		err = scene.Repo.CreateBranch("feature-branch")
		require.NoError(t, err)

		// Create worktree (normalize path for macOS /var -> /private/var symlink)
		tmpDir := t.TempDir()
		worktreePath, err := filepath.EvalSymlinks(tmpDir)
		require.NoError(t, err)
		worktreePath = filepath.Join(worktreePath, "worktree")
		err = scene.Repo.RunGitCommand("worktree", "add", worktreePath, "feature-branch")
		require.NoError(t, err)

		// Modify config from worktree
		worktreeCfg, err := LoadConfig(worktreePath)
		require.NoError(t, err)
		worktreeCfg.SetSubmitFooter(false)
		err = worktreeCfg.Save()
		require.NoError(t, err)

		// Verify change is visible from main repo
		mainCfg, err := LoadConfig(scene.Dir)
		require.NoError(t, err)
		require.False(t, mainCfg.SubmitFooter())
	})
}

// Helper function to create string pointer
func stringPtr(s string) *string {
	return &s
}
