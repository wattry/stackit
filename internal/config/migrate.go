package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"stackit.dev/stackit/internal/git"
)

const (
	jsonConfigFile   = ".stackit_config"
	jsonConfigBackup = ".stackit_config.migrated"
)

// needsMigration checks if JSON config exists and git config doesn't have trunk set.
func needsMigration(repoRoot string) bool {
	gitDir := resolveGitDir(repoRoot)
	jsonPath := filepath.Join(gitDir, jsonConfigFile)

	// Check if JSON exists
	if _, err := os.Stat(jsonPath); os.IsNotExist(err) {
		return false
	}

	// Check if already migrated (git config has trunk)
	store := git.NewConfigStore(repoRoot)
	return !store.Exists(KeyTrunk)
}

// migrateFromJSON migrates config from JSON file to git config.
// On success, the original JSON file is renamed to .stackit_config.migrated as a backup.
func migrateFromJSON(repoRoot string, store *git.ConfigStore) error {
	gitDir := resolveGitDir(repoRoot)
	jsonPath := filepath.Join(gitDir, jsonConfigFile)

	// Read JSON config
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return fmt.Errorf("read JSON config: %w", err)
	}

	var oldConfig RepoConfig
	if err := json.Unmarshal(data, &oldConfig); err != nil {
		return fmt.Errorf("parse JSON config: %w", err)
	}

	// Migrate each field
	if oldConfig.Trunk != nil && *oldConfig.Trunk != "" {
		if err := store.Set(KeyTrunk, *oldConfig.Trunk); err != nil {
			return fmt.Errorf("migrate trunk: %w", err)
		}
	}

	for _, trunk := range oldConfig.Trunks {
		if err := store.Add(KeyTrunks, trunk); err != nil {
			return fmt.Errorf("migrate additional trunk: %w", err)
		}
	}

	if oldConfig.BranchNamePattern != nil && *oldConfig.BranchNamePattern != "" {
		if err := store.Set(KeyBranchPattern, *oldConfig.BranchNamePattern); err != nil {
			return fmt.Errorf("migrate branch pattern: %w", err)
		}
	}

	if oldConfig.SubmitFooter != nil {
		if err := store.SetBool(KeySubmitFooter, *oldConfig.SubmitFooter); err != nil {
			return fmt.Errorf("migrate submit footer: %w", err)
		}
	}

	if oldConfig.UndoStackDepth != nil {
		if err := store.SetInt(KeyUndoDepth, *oldConfig.UndoStackDepth); err != nil {
			return fmt.Errorf("migrate undo depth: %w", err)
		}
	}

	if oldConfig.MaxConcurrency != nil {
		if err := store.SetInt(KeyMaxConcurrency, *oldConfig.MaxConcurrency); err != nil {
			return fmt.Errorf("migrate max concurrency: %w", err)
		}
	}

	if oldConfig.WorktreeBasePath != nil && *oldConfig.WorktreeBasePath != "" {
		if err := store.Set(KeyWorktreeBasePath, *oldConfig.WorktreeBasePath); err != nil {
			return fmt.Errorf("migrate worktree base path: %w", err)
		}
	}

	if oldConfig.WorktreeAutoClean != nil {
		if err := store.SetBool(KeyWorktreeAutoClean, *oldConfig.WorktreeAutoClean); err != nil {
			return fmt.Errorf("migrate worktree auto clean: %w", err)
		}
	}

	if oldConfig.MergeMethod != nil && *oldConfig.MergeMethod != "" {
		if err := store.Set(KeyMergeMethod, *oldConfig.MergeMethod); err != nil {
			return fmt.Errorf("migrate merge method: %w", err)
		}
	}

	// Migrate unified CI config (prefer new over legacy)
	ciCommand := ""
	if oldConfig.CICommand != nil && *oldConfig.CICommand != "" {
		ciCommand = *oldConfig.CICommand
	} else if oldConfig.CombineCICommand != nil && *oldConfig.CombineCICommand != "" {
		ciCommand = *oldConfig.CombineCICommand
	}
	if ciCommand != "" {
		if err := store.Set(KeyCICommand, ciCommand); err != nil {
			return fmt.Errorf("migrate CI command: %w", err)
		}
	}

	ciTimeout := 0
	if oldConfig.CITimeout != nil && *oldConfig.CITimeout > 0 {
		ciTimeout = *oldConfig.CITimeout
	} else if oldConfig.CombineCITimeout != nil && *oldConfig.CombineCITimeout > 0 {
		ciTimeout = *oldConfig.CombineCITimeout
	}
	if ciTimeout > 0 {
		if err := store.SetInt(KeyCITimeout, ciTimeout); err != nil {
			return fmt.Errorf("migrate CI timeout: %w", err)
		}
	}

	if oldConfig.SplitHunkSelector != nil && *oldConfig.SplitHunkSelector != "" {
		if err := store.Set(KeySplitHunkSelector, *oldConfig.SplitHunkSelector); err != nil {
			return fmt.Errorf("migrate split hunk selector: %w", err)
		}
	}

	// Migrate approved hooks
	for _, hook := range oldConfig.ApprovedPostWorktreeCreateHooks {
		if err := store.Add(KeyApprovedHooks, hook); err != nil {
			return fmt.Errorf("migrate approved hook: %w", err)
		}
	}

	// Backup old JSON config
	backupPath := filepath.Join(gitDir, jsonConfigBackup)
	if err := os.Rename(jsonPath, backupPath); err != nil {
		return fmt.Errorf("backup old config: %w", err)
	}

	// Inform user about the migration
	fmt.Fprintln(os.Stderr, "Migrated stackit config from JSON to git config.")
	fmt.Fprintf(os.Stderr, "Backup saved to: %s\n", backupPath)

	return nil
}
