package config

import (
	"fmt"
	"slices"

	"stackit.dev/stackit/internal/git"
)

// GitConfig provides typed access to stackit configuration stored in git config.
// This replaces the JSON-based config storage with native git config.
type GitConfig struct {
	repoRoot string
	store    *git.ConfigStore
}

// LoadGitConfig loads configuration from git config.
// If JSON config exists and needs migration, it will be migrated automatically.
func LoadGitConfig(repoRoot string) (*GitConfig, error) {
	store := git.NewConfigStore(repoRoot)

	cfg := &GitConfig{
		repoRoot: repoRoot,
		store:    store,
	}

	// Check if we need to migrate from JSON
	if needsMigration(repoRoot) {
		if err := migrateFromJSON(repoRoot, store); err != nil {
			return nil, fmt.Errorf("config migration failed: %w", err)
		}
	}

	return cfg, nil
}

// IsInitialized checks if stackit has been initialized (trunk is set).
func (c *GitConfig) IsInitialized() bool {
	return c.store.Exists(KeyTrunk)
}

// Trunk returns the primary trunk branch name.
func (c *GitConfig) Trunk() string {
	trunk, _ := c.store.Get(KeyTrunk)
	if trunk == "" {
		return DefaultTrunk
	}
	return trunk
}

// SetTrunk sets the primary trunk branch name.
func (c *GitConfig) SetTrunk(trunk string) error {
	return c.store.Set(KeyTrunk, trunk)
}

// AllTrunks returns all configured trunk branches (primary + additional).
func (c *GitConfig) AllTrunks() []string {
	trunks := []string{c.Trunk()}

	additional, _ := c.store.GetAll(KeyTrunks)
	for _, t := range additional {
		if !slices.Contains(trunks, t) {
			trunks = append(trunks, t)
		}
	}

	return trunks
}

// IsTrunk checks if a branch is configured as a trunk.
func (c *GitConfig) IsTrunk(branch string) bool {
	return slices.Contains(c.AllTrunks(), branch)
}

// AddTrunk adds an additional trunk branch.
func (c *GitConfig) AddTrunk(trunk string) error {
	if c.IsTrunk(trunk) {
		return fmt.Errorf("'%s' is already configured as a trunk", trunk)
	}
	return c.store.Add(KeyTrunks, trunk)
}

// BranchNamePattern returns the branch name pattern.
func (c *GitConfig) BranchNamePattern() string {
	pattern, _ := c.store.Get(KeyBranchPattern)
	if pattern == "" {
		return DefaultBranchPattern.String()
	}
	// Validate
	if _, err := NewBranchPattern(pattern); err != nil {
		return DefaultBranchPattern.String()
	}
	return pattern
}

// SetBranchNamePattern sets the branch name pattern.
func (c *GitConfig) SetBranchNamePattern(pattern string) error {
	// Validate the pattern
	if _, err := NewBranchPattern(pattern); err != nil {
		return err
	}
	return c.store.Set(KeyBranchPattern, pattern)
}

// GetBranchPattern returns the branch pattern object.
func (c *GitConfig) GetBranchPattern() BranchPattern {
	pattern, err := NewBranchPattern(c.BranchNamePattern())
	if err != nil {
		return DefaultBranchPattern
	}
	return pattern.WithDefault()
}

// SubmitFooter returns whether to include PR footer.
func (c *GitConfig) SubmitFooter() bool {
	return c.store.GetBoolWithDefault(KeySubmitFooter, DefaultSubmitFooter)
}

// SetSubmitFooter sets whether to include PR footer.
func (c *GitConfig) SetSubmitFooter(enabled bool) error {
	return c.store.SetBool(KeySubmitFooter, enabled)
}

// UndoStackDepth returns the max undo depth.
func (c *GitConfig) UndoStackDepth() int {
	depth := c.store.GetIntWithDefault(KeyUndoDepth, DefaultUndoDepth)
	if depth < 1 {
		return DefaultUndoDepth
	}
	return depth
}

// SetUndoStackDepth sets the max undo depth.
func (c *GitConfig) SetUndoStackDepth(depth int) error {
	if depth < 1 {
		return fmt.Errorf("undo depth must be at least 1")
	}
	return c.store.SetInt(KeyUndoDepth, depth)
}

// WorktreeBasePath returns the worktree base path.
func (c *GitConfig) WorktreeBasePath() string {
	path, _ := c.store.Get(KeyWorktreeBasePath)
	return path
}

// SetWorktreeBasePath sets the worktree base path.
func (c *GitConfig) SetWorktreeBasePath(path string) error {
	return c.store.Set(KeyWorktreeBasePath, path)
}

// WorktreeAutoClean returns whether to auto-clean worktrees.
func (c *GitConfig) WorktreeAutoClean() bool {
	return c.store.GetBoolWithDefault(KeyWorktreeAutoClean, DefaultWorktreeAutoClean)
}

// SetWorktreeAutoClean sets whether to auto-clean worktrees.
func (c *GitConfig) SetWorktreeAutoClean(enabled bool) error {
	return c.store.SetBool(KeyWorktreeAutoClean, enabled)
}

// MergeMethod returns the configured merge method (empty if not set).
func (c *GitConfig) MergeMethod() string {
	method, _ := c.store.Get(KeyMergeMethod)
	return method
}

// SetMergeMethod sets the merge method preference.
func (c *GitConfig) SetMergeMethod(method string) error {
	if !slices.Contains(ValidMergeMethods, method) {
		return fmt.Errorf("invalid merge method: %s (must be %v)", method, ValidMergeMethods)
	}
	return c.store.Set(KeyMergeMethod, method)
}

// CICommand returns the CI validation command.
func (c *GitConfig) CICommand() string {
	cmd, _ := c.store.Get(KeyCICommand)
	return cmd
}

// SetCICommand sets the CI validation command.
func (c *GitConfig) SetCICommand(cmd string) error {
	return c.store.Set(KeyCICommand, cmd)
}

// CITimeout returns the CI timeout in seconds.
func (c *GitConfig) CITimeout() int {
	timeout := c.store.GetIntWithDefault(KeyCITimeout, DefaultCITimeout)
	if timeout < 1 {
		return DefaultCITimeout
	}
	return timeout
}

// SetCITimeout sets the CI timeout in seconds.
func (c *GitConfig) SetCITimeout(seconds int) error {
	if seconds < 1 {
		return fmt.Errorf("CI timeout must be at least 1 second")
	}
	return c.store.SetInt(KeyCITimeout, seconds)
}

// SplitHunkSelector returns the hunk selector mode.
func (c *GitConfig) SplitHunkSelector() string {
	selector, _ := c.store.Get(KeySplitHunkSelector)
	if selector == "" {
		return DefaultSplitHunkSelector
	}
	if !slices.Contains(ValidHunkSelectors, selector) {
		return DefaultSplitHunkSelector
	}
	return selector
}

// SetSplitHunkSelector sets the hunk selector mode.
func (c *GitConfig) SetSplitHunkSelector(selector string) error {
	if !slices.Contains(ValidHunkSelectors, selector) {
		return fmt.Errorf("invalid hunk selector: %s (must be %v)", selector, ValidHunkSelectors)
	}
	return c.store.Set(KeySplitHunkSelector, selector)
}

// ApprovedPostWorktreeCreateHooks returns the list of approved hooks.
func (c *GitConfig) ApprovedPostWorktreeCreateHooks() []string {
	hooks, _ := c.store.GetAll(KeyApprovedHooks)
	return hooks
}

// IsPostWorktreeCreateHookApproved checks if a hook is approved.
func (c *GitConfig) IsPostWorktreeCreateHookApproved(hook string) bool {
	return slices.Contains(c.ApprovedPostWorktreeCreateHooks(), hook)
}

// AddApprovedPostWorktreeCreateHook adds a hook to the approved list.
func (c *GitConfig) AddApprovedPostWorktreeCreateHook(hook string) error {
	if c.IsPostWorktreeCreateHookApproved(hook) {
		return nil // Already approved
	}
	return c.store.Add(KeyApprovedHooks, hook)
}

// RemoveApprovedPostWorktreeCreateHook removes a hook from the approved list.
func (c *GitConfig) RemoveApprovedPostWorktreeCreateHook(hook string) error {
	// Get all current hooks
	hooks := c.ApprovedPostWorktreeCreateHooks()
	if !slices.Contains(hooks, hook) {
		return nil // Not in list
	}

	// Remove all and re-add the ones we want to keep
	if err := c.store.Unset(KeyApprovedHooks); err != nil {
		return err
	}

	for _, h := range hooks {
		if h != hook {
			if err := c.store.Add(KeyApprovedHooks, h); err != nil {
				return err
			}
		}
	}
	return nil
}

// ClearApprovedPostWorktreeCreateHooks removes all hook approvals.
func (c *GitConfig) ClearApprovedPostWorktreeCreateHooks() error {
	return c.store.Unset(KeyApprovedHooks)
}

// MaxConcurrency returns the maximum number of concurrent validation operations.
// Returns 0 if not set (engine will use default based on CPU count).
func (c *GitConfig) MaxConcurrency() int {
	concurrency := c.store.GetIntWithDefault(KeyMaxConcurrency, DefaultMaxConcurrency)
	if concurrency < 0 {
		return DefaultMaxConcurrency
	}
	return concurrency
}

// SetMaxConcurrency sets the maximum number of concurrent validation operations.
func (c *GitConfig) SetMaxConcurrency(n int) error {
	if n < 0 {
		return fmt.Errorf("max concurrency must be >= 0")
	}
	return c.store.SetInt(KeyMaxConcurrency, n)
}

// Deprecated methods for backwards compatibility during migration.

// CombineCICommand returns the CI command (deprecated, use CICommand).
func (c *GitConfig) CombineCICommand() string {
	return c.CICommand()
}

// SetCombineCICommand sets the CI command (deprecated, use SetCICommand).
func (c *GitConfig) SetCombineCICommand(cmd string) {
	_ = c.SetCICommand(cmd)
}

// CombineCITimeout returns the CI timeout (deprecated, use CITimeout).
func (c *GitConfig) CombineCITimeout() int {
	return c.CITimeout()
}

// SetCombineCITimeout sets the CI timeout (deprecated, use SetCITimeout).
func (c *GitConfig) SetCombineCITimeout(seconds int) {
	_ = c.SetCITimeout(seconds)
}

// Save is a no-op for GitConfig since git config writes are immediate.
// This method exists for API compatibility with the old Config type.
func (c *GitConfig) Save() error {
	return nil
}
