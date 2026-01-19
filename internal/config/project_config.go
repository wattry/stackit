package config

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"gopkg.in/yaml.v3"
)

// ProjectConfigFileName is the name of the project configuration file
const ProjectConfigFileName = ".stackit.yaml"

// BranchConfig contains branch naming configuration
type BranchConfig struct {
	Pattern string `yaml:"pattern,omitempty"`
}

// SubmitConfig contains PR submission configuration
type SubmitConfig struct {
	Footer *bool `yaml:"footer,omitempty"` // Pointer to distinguish unset from false
}

// MergeConfig contains merge method configuration
type MergeConfig struct {
	Method string `yaml:"method,omitempty"`
}

// CIConfig contains CI validation configuration
type CIConfig struct {
	Command string `yaml:"command,omitempty"`
	Timeout int    `yaml:"timeout,omitempty"`
}

// ProjectConfig represents the project-level configuration stored in .stackit.yaml
// This file is committed to the repository and shared across the team.
// Team settings can be overridden by personal git config (git config > project config > defaults).
type ProjectConfig struct {
	Trunk  string   `yaml:"trunk,omitempty"`
	Trunks []string `yaml:"trunks,omitempty"`

	Branch BranchConfig `yaml:"branch,omitempty"`
	Submit SubmitConfig `yaml:"submit,omitempty"`
	Merge  MergeConfig  `yaml:"merge,omitempty"`
	CI     CIConfig     `yaml:"ci,omitempty"`
	Hooks  HooksConfig  `yaml:"hooks,omitempty"`
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

	// Validate the configuration
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &config, nil
}

// HasPostWorktreeCreateHooks returns true if there are any post-worktree-create hooks configured
func (c *ProjectConfig) HasPostWorktreeCreateHooks() bool {
	return len(c.Hooks.PostWorktreeCreate) > 0
}

// HasTrunk returns true if a trunk branch is configured
func (c *ProjectConfig) HasTrunk() bool {
	return c.Trunk != ""
}

// HasTrunks returns true if additional trunk branches are configured
func (c *ProjectConfig) HasTrunks() bool {
	return len(c.Trunks) > 0
}

// HasBranchPattern returns true if a branch naming pattern is configured
func (c *ProjectConfig) HasBranchPattern() bool {
	return c.Branch.Pattern != ""
}

// HasSubmitFooter returns true if the submit footer setting is configured
func (c *ProjectConfig) HasSubmitFooter() bool {
	return c.Submit.Footer != nil
}

// GetSubmitFooter returns the submit footer value (caller should check HasSubmitFooter first)
func (c *ProjectConfig) GetSubmitFooter() bool {
	if c.Submit.Footer == nil {
		return true // Default
	}
	return *c.Submit.Footer
}

// HasMergeMethod returns true if a merge method is configured
func (c *ProjectConfig) HasMergeMethod() bool {
	return c.Merge.Method != ""
}

// HasCICommand returns true if a CI command is configured
func (c *ProjectConfig) HasCICommand() bool {
	return c.CI.Command != ""
}

// HasCITimeout returns true if a CI timeout is configured
func (c *ProjectConfig) HasCITimeout() bool {
	return c.CI.Timeout > 0
}

// Validate checks the configuration for invalid values
func (c *ProjectConfig) Validate() error {
	// Validate branch pattern if set
	if c.HasBranchPattern() {
		if _, err := NewBranchPattern(c.Branch.Pattern); err != nil {
			return fmt.Errorf("invalid branch.pattern in %s: %w", ProjectConfigFileName, err)
		}
	}

	// Validate merge method if set
	if c.HasMergeMethod() {
		if !slices.Contains(ValidMergeMethods, c.Merge.Method) {
			return fmt.Errorf("invalid merge.method in %s: %q (must be one of %v)", ProjectConfigFileName, c.Merge.Method, ValidMergeMethods)
		}
	}

	// Validate CI timeout if set
	if c.CI.Timeout < 0 {
		return fmt.Errorf("invalid ci.timeout in %s: must be >= 0", ProjectConfigFileName)
	}

	return nil
}
