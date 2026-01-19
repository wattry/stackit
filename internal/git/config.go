// Package git provides low-level Git operations, including repository access,
// branch operations, commit information, PR operations, and metadata management.
package git

import (
	"errors"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// GetCurrentDate returns the current date and time in yyyyMMddHHmmss format in UTC
func GetCurrentDate() string {
	now := time.Now().UTC()
	return now.Format("20060102150405")
}

// ConfigStore provides typed access to git config with a simpler interface
// than the full Runner. It operates directly on the repository via git commands.
type ConfigStore struct {
	repoRoot string
}

// NewConfigStore creates a new ConfigStore for the given repository root.
func NewConfigStore(repoRoot string) *ConfigStore {
	return &ConfigStore{repoRoot: repoRoot}
}

// Get retrieves a single config value from local git config.
// Returns empty string if the key doesn't exist.
func (c *ConfigStore) Get(key string) (string, error) {
	cmd := exec.Command("git", "config", "--local", key)
	cmd.Dir = c.repoRoot
	out, err := cmd.Output()
	if err != nil {
		// git config returns exit code 1 if key not found
		// Other exit codes indicate actual errors
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			if exitErr.ExitCode() == 1 {
				return "", nil // key not found
			}
		}
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// GetAll retrieves all values for a multi-value config key.
// Returns empty slice if the key doesn't exist.
func (c *ConfigStore) GetAll(key string) ([]string, error) {
	cmd := exec.Command("git", "config", "--local", "--get-all", key)
	cmd.Dir = c.repoRoot
	out, err := cmd.Output()
	if err != nil {
		// git config returns exit code 1 if key not found
		// Other exit codes indicate actual errors
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			if exitErr.ExitCode() == 1 {
				return nil, nil // key not found
			}
		}
		return nil, err
	}
	trimmed := strings.TrimSpace(string(out))
	if trimmed == "" {
		return nil, nil
	}
	return strings.Split(trimmed, "\n"), nil
}

// Set sets a config value in local git config.
func (c *ConfigStore) Set(key, value string) error {
	cmd := exec.Command("git", "config", "--local", key, value)
	cmd.Dir = c.repoRoot
	return cmd.Run()
}

// SetBool sets a boolean config value.
func (c *ConfigStore) SetBool(key string, value bool) error {
	return c.Set(key, strconv.FormatBool(value))
}

// SetInt sets an integer config value.
func (c *ConfigStore) SetInt(key string, value int) error {
	return c.Set(key, strconv.Itoa(value))
}

// Add adds a value to a multi-value config key.
func (c *ConfigStore) Add(key, value string) error {
	cmd := exec.Command("git", "config", "--local", "--add", key, value)
	cmd.Dir = c.repoRoot
	return cmd.Run()
}

// Unset removes all values for a config key.
// Does not return an error if the key doesn't exist.
func (c *ConfigStore) Unset(key string) error {
	cmd := exec.Command("git", "config", "--local", "--unset-all", key)
	cmd.Dir = c.repoRoot
	err := cmd.Run()
	if err != nil {
		// git config --unset-all returns exit code 5 if key not found
		// Other exit codes indicate actual errors
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			if exitErr.ExitCode() == 5 {
				return nil // key not found, that's fine
			}
		}
		return err
	}
	return nil
}

// GetBool retrieves a boolean config value.
// Returns false and no error if the key doesn't exist.
func (c *ConfigStore) GetBool(key string) (bool, error) {
	val, err := c.Get(key)
	if err != nil || val == "" {
		return false, err
	}
	return strconv.ParseBool(val)
}

// GetBoolWithDefault retrieves a boolean config value with a default.
func (c *ConfigStore) GetBoolWithDefault(key string, defaultValue bool) bool {
	val, err := c.GetBool(key)
	if err != nil {
		return defaultValue
	}
	// If key doesn't exist, Get returns empty string, GetBool returns false
	// We need to check if the key actually exists
	raw, _ := c.Get(key)
	if raw == "" {
		return defaultValue
	}
	return val
}

// GetInt retrieves an integer config value.
// Returns 0 and no error if the key doesn't exist.
func (c *ConfigStore) GetInt(key string) (int, error) {
	val, err := c.Get(key)
	if err != nil || val == "" {
		return 0, err
	}
	return strconv.Atoi(val)
}

// GetIntWithDefault retrieves an integer config value with a default.
func (c *ConfigStore) GetIntWithDefault(key string, defaultValue int) int {
	raw, _ := c.Get(key)
	if raw == "" {
		return defaultValue
	}
	val, err := c.GetInt(key)
	if err != nil {
		return defaultValue
	}
	return val
}

// Exists checks if a config key exists.
func (c *ConfigStore) Exists(key string) bool {
	val, err := c.Get(key)
	return err == nil && val != ""
}
