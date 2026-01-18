package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ProjectConfigFileName is the name of the project configuration file
const ProjectConfigFileName = ".stackit.yaml"

// ProjectConfig represents the project-level configuration stored in .stackit.yaml
// This file is committed to the repository and shared across the team.
type ProjectConfig struct {
	Hooks HooksConfig `yaml:"hooks"`
}

// HooksConfig contains hook configurations
type HooksConfig struct {
	// PostWorktreeCreate contains commands to run after creating a worktree
	PostWorktreeCreate []string `yaml:"post-worktree-create"`
}

// LoadProjectConfig reads the project configuration from .stackit.yaml in the repo root.
// Returns an empty config (not an error) if the file doesn't exist.
func LoadProjectConfig(repoRoot string) (*ProjectConfig, error) {
	configPath := filepath.Join(repoRoot, ProjectConfigFileName)

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist - return empty config
			return &ProjectConfig{}, nil
		}
		return nil, err
	}

	var config ProjectConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w (content: %q)", ProjectConfigFileName, err, string(data))
	}

	return &config, nil
}

// HasPostWorktreeCreateHooks returns true if there are any post-worktree-create hooks configured
func (c *ProjectConfig) HasPostWorktreeCreateHooks() bool {
	return len(c.Hooks.PostWorktreeCreate) > 0
}
