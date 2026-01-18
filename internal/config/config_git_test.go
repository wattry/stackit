package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/testhelpers"
)

// removeDefaultConfig removes the default JSON config file created by test setup
// so we can test fresh/uninitialized state.
func removeDefaultConfig(t *testing.T, dir string) {
	t.Helper()
	configPath := filepath.Join(dir, ".git", ".stackit_config")
	_ = os.Remove(configPath)
}

func TestGitConfigBasicOperations(t *testing.T) {
	t.Parallel()

	t.Run("returns defaults when not initialized", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, nil)
		removeDefaultConfig(t, scene.Dir)

		cfg, err := LoadGitConfig(scene.Dir)
		require.NoError(t, err)
		require.False(t, cfg.IsInitialized())
		require.Equal(t, DefaultTrunk, cfg.Trunk())
		require.True(t, cfg.SubmitFooter())
		require.Equal(t, DefaultUndoDepth, cfg.UndoStackDepth())
		require.Equal(t, DefaultCITimeout, cfg.CITimeout())
		require.Equal(t, DefaultSplitHunkSelector, cfg.SplitHunkSelector())
	})

	t.Run("sets and gets trunk", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, nil)

		cfg, err := LoadGitConfig(scene.Dir)
		require.NoError(t, err)

		err = cfg.SetTrunk("develop")
		require.NoError(t, err)

		require.True(t, cfg.IsInitialized())
		require.Equal(t, "develop", cfg.Trunk())

		// Reload to verify persistence
		cfg2, err := LoadGitConfig(scene.Dir)
		require.NoError(t, err)
		require.Equal(t, "develop", cfg2.Trunk())
	})

	t.Run("sets and gets submit footer", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, nil)

		cfg, err := LoadGitConfig(scene.Dir)
		require.NoError(t, err)

		err = cfg.SetSubmitFooter(false)
		require.NoError(t, err)
		require.False(t, cfg.SubmitFooter())

		// Reload to verify
		cfg2, err := LoadGitConfig(scene.Dir)
		require.NoError(t, err)
		require.False(t, cfg2.SubmitFooter())
	})

	t.Run("sets and gets undo depth", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, nil)

		cfg, err := LoadGitConfig(scene.Dir)
		require.NoError(t, err)

		err = cfg.SetUndoStackDepth(20)
		require.NoError(t, err)
		require.Equal(t, 20, cfg.UndoStackDepth())

		// Reload to verify
		cfg2, err := LoadGitConfig(scene.Dir)
		require.NoError(t, err)
		require.Equal(t, 20, cfg2.UndoStackDepth())
	})

	t.Run("rejects invalid undo depth", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, nil)

		cfg, err := LoadGitConfig(scene.Dir)
		require.NoError(t, err)

		err = cfg.SetUndoStackDepth(0)
		require.Error(t, err)

		err = cfg.SetUndoStackDepth(-1)
		require.Error(t, err)
	})
}

func TestGitConfigTrunks(t *testing.T) {
	t.Parallel()

	t.Run("manages multiple trunks", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, nil)

		cfg, err := LoadGitConfig(scene.Dir)
		require.NoError(t, err)

		err = cfg.SetTrunk("main")
		require.NoError(t, err)

		err = cfg.AddTrunk("develop")
		require.NoError(t, err)

		err = cfg.AddTrunk("staging")
		require.NoError(t, err)

		require.Equal(t, []string{"main", "develop", "staging"}, cfg.AllTrunks())
		require.True(t, cfg.IsTrunk("main"))
		require.True(t, cfg.IsTrunk("develop"))
		require.True(t, cfg.IsTrunk("staging"))
		require.False(t, cfg.IsTrunk("feature"))
	})

	t.Run("prevents duplicate trunks", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, nil)

		cfg, err := LoadGitConfig(scene.Dir)
		require.NoError(t, err)

		err = cfg.SetTrunk("main")
		require.NoError(t, err)

		err = cfg.AddTrunk("main")
		require.Error(t, err)
		require.Contains(t, err.Error(), "already configured")
	})
}

func TestGitConfigMergeMethod(t *testing.T) {
	t.Parallel()

	t.Run("returns empty when not set", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, nil)

		cfg, err := LoadGitConfig(scene.Dir)
		require.NoError(t, err)

		require.Empty(t, cfg.MergeMethod())
	})

	t.Run("sets valid merge methods", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, nil)

		cfg, err := LoadGitConfig(scene.Dir)
		require.NoError(t, err)

		for _, method := range []string{"squash", "merge", "rebase"} {
			err = cfg.SetMergeMethod(method)
			require.NoError(t, err)
			require.Equal(t, method, cfg.MergeMethod())
		}
	})

	t.Run("rejects invalid merge method", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, nil)

		cfg, err := LoadGitConfig(scene.Dir)
		require.NoError(t, err)

		err = cfg.SetMergeMethod("invalid")
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid merge method")
	})
}

func TestGitConfigCISettings(t *testing.T) {
	t.Parallel()

	t.Run("sets and gets CI command", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, nil)

		cfg, err := LoadGitConfig(scene.Dir)
		require.NoError(t, err)

		err = cfg.SetCICommand("mise run check")
		require.NoError(t, err)
		require.Equal(t, "mise run check", cfg.CICommand())
	})

	t.Run("sets and gets CI timeout", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, nil)

		cfg, err := LoadGitConfig(scene.Dir)
		require.NoError(t, err)

		err = cfg.SetCITimeout(300)
		require.NoError(t, err)
		require.Equal(t, 300, cfg.CITimeout())
	})

	t.Run("rejects invalid CI timeout", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, nil)

		cfg, err := LoadGitConfig(scene.Dir)
		require.NoError(t, err)

		err = cfg.SetCITimeout(0)
		require.Error(t, err)
	})
}

func TestGitConfigSplitHunkSelector(t *testing.T) {
	t.Parallel()

	t.Run("defaults to tui", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, nil)

		cfg, err := LoadGitConfig(scene.Dir)
		require.NoError(t, err)

		require.Equal(t, "tui", cfg.SplitHunkSelector())
	})

	t.Run("sets valid selectors", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, nil)

		cfg, err := LoadGitConfig(scene.Dir)
		require.NoError(t, err)

		err = cfg.SetSplitHunkSelector("git")
		require.NoError(t, err)
		require.Equal(t, "git", cfg.SplitHunkSelector())

		err = cfg.SetSplitHunkSelector("tui")
		require.NoError(t, err)
		require.Equal(t, "tui", cfg.SplitHunkSelector())
	})

	t.Run("rejects invalid selector", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, nil)

		cfg, err := LoadGitConfig(scene.Dir)
		require.NoError(t, err)

		err = cfg.SetSplitHunkSelector("invalid")
		require.Error(t, err)
	})
}

func TestGitConfigBranchPattern(t *testing.T) {
	t.Parallel()

	t.Run("returns default pattern", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, nil)

		cfg, err := LoadGitConfig(scene.Dir)
		require.NoError(t, err)

		require.Equal(t, DefaultBranchPattern.String(), cfg.BranchNamePattern())
	})

	t.Run("sets and gets valid pattern", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, nil)

		cfg, err := LoadGitConfig(scene.Dir)
		require.NoError(t, err)

		err = cfg.SetBranchNamePattern("feature/{message}")
		require.NoError(t, err)
		require.Equal(t, "feature/{message}", cfg.BranchNamePattern())
	})

	t.Run("rejects invalid pattern", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, nil)

		cfg, err := LoadGitConfig(scene.Dir)
		require.NoError(t, err)

		// Pattern without {slug} is invalid
		err = cfg.SetBranchNamePattern("feature/")
		require.Error(t, err)
	})
}

func TestGitConfigApprovedHooks(t *testing.T) {
	t.Parallel()

	t.Run("manages approved hooks", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, nil)

		cfg, err := LoadGitConfig(scene.Dir)
		require.NoError(t, err)

		require.Empty(t, cfg.ApprovedPostWorktreeCreateHooks())

		err = cfg.AddApprovedPostWorktreeCreateHook("mise trust")
		require.NoError(t, err)

		err = cfg.AddApprovedPostWorktreeCreateHook("npm install")
		require.NoError(t, err)

		require.True(t, cfg.IsPostWorktreeCreateHookApproved("mise trust"))
		require.True(t, cfg.IsPostWorktreeCreateHookApproved("npm install"))
		require.False(t, cfg.IsPostWorktreeCreateHookApproved("other"))

		err = cfg.RemoveApprovedPostWorktreeCreateHook("mise trust")
		require.NoError(t, err)

		require.False(t, cfg.IsPostWorktreeCreateHookApproved("mise trust"))
		require.True(t, cfg.IsPostWorktreeCreateHookApproved("npm install"))
	})

	t.Run("clears all hooks", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, nil)

		cfg, err := LoadGitConfig(scene.Dir)
		require.NoError(t, err)

		err = cfg.AddApprovedPostWorktreeCreateHook("mise trust")
		require.NoError(t, err)

		err = cfg.AddApprovedPostWorktreeCreateHook("npm install")
		require.NoError(t, err)

		err = cfg.ClearApprovedPostWorktreeCreateHooks()
		require.NoError(t, err)

		require.Empty(t, cfg.ApprovedPostWorktreeCreateHooks())
	})
}

func TestMigrationFromJSON(t *testing.T) {
	t.Parallel()

	t.Run("migrates full config", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, nil)

		// Create JSON config (overwrite the default one created by test setup)
		configPath := filepath.Join(scene.Dir, ".git", ".stackit_config")
		footer := false
		depth := 15
		timeout := 300
		autoClean := false
		config := &RepoConfig{
			Trunk:                           stringPtr("develop"),
			Trunks:                          []string{"staging", "production"},
			BranchNamePattern:               stringPtr("feature/{message}"),
			SubmitFooter:                    &footer,
			UndoStackDepth:                  &depth,
			MergeMethod:                     stringPtr("squash"),
			CICommand:                       stringPtr("npm test"),
			CITimeout:                       &timeout,
			WorktreeBasePath:                stringPtr("/tmp/worktrees"),
			WorktreeAutoClean:               &autoClean,
			SplitHunkSelector:               stringPtr("git"),
			ApprovedPostWorktreeCreateHooks: []string{"mise trust"},
		}
		configJSON, err := json.MarshalIndent(config, "", "  ")
		require.NoError(t, err)
		err = os.WriteFile(configPath, configJSON, 0600)
		require.NoError(t, err)

		// Load should trigger migration
		cfg, err := LoadGitConfig(scene.Dir)
		require.NoError(t, err)

		// Verify all values migrated
		require.Equal(t, "develop", cfg.Trunk())
		require.Equal(t, []string{"develop", "staging", "production"}, cfg.AllTrunks())
		require.Equal(t, "feature/{message}", cfg.BranchNamePattern())
		require.False(t, cfg.SubmitFooter())
		require.Equal(t, 15, cfg.UndoStackDepth())
		require.Equal(t, "squash", cfg.MergeMethod())
		require.Equal(t, "npm test", cfg.CICommand())
		require.Equal(t, 300, cfg.CITimeout())
		require.Equal(t, "/tmp/worktrees", cfg.WorktreeBasePath())
		require.False(t, cfg.WorktreeAutoClean())
		require.Equal(t, "git", cfg.SplitHunkSelector())
		require.True(t, cfg.IsPostWorktreeCreateHookApproved("mise trust"))

		// Verify backup file exists
		backupPath := filepath.Join(scene.Dir, ".git", ".stackit_config.migrated")
		_, err = os.Stat(backupPath)
		require.NoError(t, err)

		// Verify original file is gone
		_, err = os.Stat(configPath)
		require.True(t, os.IsNotExist(err))
	})

	t.Run("migrates legacy CI config", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, nil)

		// Create JSON config with legacy combine.* fields
		configPath := filepath.Join(scene.Dir, ".git", ".stackit_config")
		timeout := 120
		config := &RepoConfig{
			Trunk:            stringPtr("main"),
			CombineCICommand: stringPtr("make test"),
			CombineCITimeout: &timeout,
		}
		configJSON, err := json.MarshalIndent(config, "", "  ")
		require.NoError(t, err)
		err = os.WriteFile(configPath, configJSON, 0600)
		require.NoError(t, err)

		cfg, err := LoadGitConfig(scene.Dir)
		require.NoError(t, err)

		require.Equal(t, "make test", cfg.CICommand())
		require.Equal(t, 120, cfg.CITimeout())
	})

	t.Run("prefers new CI config over legacy", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, nil)

		// Create JSON config with both new and legacy CI fields
		configPath := filepath.Join(scene.Dir, ".git", ".stackit_config")
		newTimeout := 300
		legacyTimeout := 120
		config := &RepoConfig{
			Trunk:            stringPtr("main"),
			CICommand:        stringPtr("mise run check"),
			CITimeout:        &newTimeout,
			CombineCICommand: stringPtr("make test"),
			CombineCITimeout: &legacyTimeout,
		}
		configJSON, err := json.MarshalIndent(config, "", "  ")
		require.NoError(t, err)
		err = os.WriteFile(configPath, configJSON, 0600)
		require.NoError(t, err)

		cfg, err := LoadGitConfig(scene.Dir)
		require.NoError(t, err)

		require.Equal(t, "mise run check", cfg.CICommand())
		require.Equal(t, 300, cfg.CITimeout())
	})

	t.Run("does not migrate if already migrated", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, nil)

		// First, set up git config directly
		cfg, err := LoadGitConfig(scene.Dir)
		require.NoError(t, err)
		err = cfg.SetTrunk("main")
		require.NoError(t, err)
		err = cfg.SetSubmitFooter(false)
		require.NoError(t, err)

		// Now create a JSON config that would migrate differently
		configPath := filepath.Join(scene.Dir, ".git", ".stackit_config")
		footer := true
		config := &RepoConfig{
			Trunk:        stringPtr("develop"),
			SubmitFooter: &footer,
		}
		configJSON, err := json.MarshalIndent(config, "", "  ")
		require.NoError(t, err)
		err = os.WriteFile(configPath, configJSON, 0600)
		require.NoError(t, err)

		// Load again - should NOT migrate because trunk already exists
		cfg2, err := LoadGitConfig(scene.Dir)
		require.NoError(t, err)

		// Values should still be from git config, not JSON
		require.Equal(t, "main", cfg2.Trunk())
		require.False(t, cfg2.SubmitFooter())

		// JSON file should still exist (not moved to backup)
		_, err = os.Stat(configPath)
		require.NoError(t, err)
	})

	t.Run("handles missing JSON gracefully", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, nil)
		removeDefaultConfig(t, scene.Dir)

		// No JSON config exists
		cfg, err := LoadGitConfig(scene.Dir)
		require.NoError(t, err)
		require.False(t, cfg.IsInitialized())
	})

	t.Run("handles empty JSON config", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, nil)

		// Create empty JSON config
		configPath := filepath.Join(scene.Dir, ".git", ".stackit_config")
		err := os.WriteFile(configPath, []byte("{}"), 0600)
		require.NoError(t, err)

		cfg, err := LoadGitConfig(scene.Dir)
		require.NoError(t, err)
		require.False(t, cfg.IsInitialized())
	})
}

func TestGitConfigSaveNoop(t *testing.T) {
	t.Parallel()

	t.Run("Save is a no-op", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, nil)

		cfg, err := LoadGitConfig(scene.Dir)
		require.NoError(t, err)

		err = cfg.SetTrunk("main")
		require.NoError(t, err)

		// Save should succeed but do nothing (writes are already persisted)
		err = cfg.Save()
		require.NoError(t, err)

		// Value should still be there
		cfg2, err := LoadGitConfig(scene.Dir)
		require.NoError(t, err)
		require.Equal(t, "main", cfg2.Trunk())
	})
}
