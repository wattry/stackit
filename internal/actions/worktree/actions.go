// Package worktree provides actions for managing stackit-managed worktrees.
package worktree

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"stackit.dev/stackit/internal/app"
	"stackit.dev/stackit/internal/config"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/output"
	"stackit.dev/stackit/internal/tui/style"
)

// ListOptions contains options for the list action
type ListOptions struct {
	// No options for now
}

// ListResult contains the results of listing worktrees
type ListResult struct {
	Worktrees []Entry
}

// Entry represents a single managed worktree
type Entry struct {
	Name         string // User-provided name (empty for legacy worktrees)
	AnchorBranch string // Anchor branch name
	Path         string
	Exists       bool // Whether the path still exists on disk
}

// ListAction lists all managed worktrees
func ListAction(ctx *app.Context, _ ListOptions) (*ListResult, error) {
	worktrees, err := ctx.Engine.ListManagedWorktrees()
	if err != nil {
		return nil, fmt.Errorf("failed to list managed worktrees: %w", err)
	}

	result := &ListResult{
		Worktrees: make([]Entry, 0, len(worktrees)),
	}

	for _, wt := range worktrees {
		// Check if path exists
		exists := true
		if _, statErr := os.Stat(wt.Path); os.IsNotExist(statErr) {
			exists = false
		}

		result.Worktrees = append(result.Worktrees, Entry{
			Name:         wt.Name,
			AnchorBranch: wt.AnchorBranch,
			Path:         wt.Path,
			Exists:       exists,
		})
	}

	return result, nil
}

// RemoveOptions contains options for the remove action
type RemoveOptions struct {
	AnchorBranch string // Anchor branch name to remove worktree for
	Force        bool   // Force removal even if worktree has uncommitted changes
}

// findWorktreeByNameOrBranch looks up a worktree by name or anchor branch
func findWorktreeByNameOrBranch(ctx *app.Context, nameOrBranch string) (*engine.WorktreeInfo, error) {
	// First try by anchor branch (original behavior)
	wtInfo, err := ctx.Engine.GetWorktreeForStack(nameOrBranch)
	if err != nil {
		return nil, fmt.Errorf("failed to get worktree info: %w", err)
	}
	if wtInfo != nil {
		return wtInfo, nil
	}

	// If not found, try to find by worktree name
	worktrees, err := ctx.Engine.ListManagedWorktrees()
	if err != nil {
		return nil, fmt.Errorf("failed to list worktrees: %w", err)
	}
	for _, wt := range worktrees {
		if wt.Name == nameOrBranch {
			return &wt, nil
		}
	}

	return nil, fmt.Errorf("no worktree found for %s", style.ColorBranchName(nameOrBranch, false))
}

// RemoveAction removes a worktree for a stack
func RemoveAction(ctx *app.Context, opts RemoveOptions) error {
	out := ctx.Output

	// Get worktree info (supports lookup by name or anchor branch)
	wtInfo, err := findWorktreeByNameOrBranch(ctx, opts.AnchorBranch)
	if err != nil {
		return err
	}

	// Check if path exists before trying to remove
	if _, statErr := os.Stat(wtInfo.Path); statErr == nil {
		// Try to remove the git worktree
		if removeErr := ctx.Engine.RemoveWorktree(ctx.Context, wtInfo.Path); removeErr != nil {
			if !opts.Force {
				return fmt.Errorf("failed to remove worktree at %s: %w (use --force to override)", wtInfo.Path, removeErr)
			}
			out.Warn("Failed to remove worktree directory, continuing with unregistration: %v", removeErr)
		}
	} else {
		out.Debug("Worktree path %s does not exist, skipping removal", wtInfo.Path)
	}

	// Unregister from registry (use the anchor branch from worktree info)
	if unregErr := ctx.Engine.UnregisterWorktree(wtInfo.AnchorBranch); unregErr != nil {
		return fmt.Errorf("failed to unregister worktree: %w", unregErr)
	}

	out.Success("Removed worktree for stack %s", style.ColorBranchName(wtInfo.AnchorBranch, false))
	return nil
}

// OpenOptions contains options for the open action
type OpenOptions struct {
	AnchorBranch string // Anchor branch name to get path for
}

// OpenAction returns the path to a worktree for a stack
func OpenAction(ctx *app.Context, opts OpenOptions) (string, error) {
	wtInfo, err := findWorktreeByNameOrBranch(ctx, opts.AnchorBranch)
	if err != nil {
		return "", err
	}

	// Check if path exists
	if _, statErr := os.Stat(wtInfo.Path); os.IsNotExist(statErr) {
		return "", fmt.Errorf("worktree path %s does not exist (worktree may have been manually deleted)", wtInfo.Path)
	}

	return wtInfo.Path, nil
}

// CreateOptions contains options for the create action
type CreateOptions struct {
	Name  string // User-provided name for the worktree
	Scope string // Optional scope to set on the anchor branch
}

// CreateResult contains the results of creating a worktree
type CreateResult struct {
	Name         string // The name of the worktree
	AnchorBranch string // The name of the anchor branch
	Path         string // The path to the worktree
}

// CreateAction creates a new worktree with an anchor branch
func CreateAction(ctx *app.Context, opts CreateOptions) (*CreateResult, error) {
	eng := ctx.Engine
	out := ctx.Output
	repoRoot := ctx.RepoRoot

	// If we're in a managed worktree, we need to create the new worktree from the main repo
	if ctx.InManagedWorktree && ctx.WorktreeInfo != nil {
		out.Info("Creating worktree from main repository (currently in worktree: %s)", ctx.WorktreeInfo.Name)

		// Create a temporary engine for the main repository
		mainRepoRoot := ctx.WorktreeInfo.MainRepoDir
		mainGit := git.NewRunnerWithPath(mainRepoRoot, ctx.Logger)

		// Load config from main repo for trunk and undo settings
		mainCfg, err := config.LoadConfig(mainRepoRoot)
		if err != nil {
			return nil, fmt.Errorf("failed to load config from main repo: %w", err)
		}

		mainEng, err := engine.NewEngine(engine.Options{
			RepoRoot:          mainRepoRoot,
			Trunk:             mainCfg.Trunk(),
			MaxUndoStackDepth: mainCfg.UndoStackDepth(),
			Git:               mainGit,
			Writer:            os.Stderr,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create engine for main repo: %w", err)
		}

		// Use the main repo engine for the rest of the operation
		eng = mainEng
		repoRoot = mainRepoRoot
	}

	trunk := eng.Trunk()

	// Check if trunk is behind remote and warn
	if status, err := eng.GetBranchRemoteStatus(trunk); err == nil && status.Behind() {
		out.Warn("Local %s is behind remote. Consider running 'st sync' first.", trunk.GetName())
	}

	// Inform user if not on trunk (worktree is always created from trunk)
	currentBranch := eng.CurrentBranch()
	if currentBranch == nil {
		out.Info("Creating worktree from %s (currently in detached HEAD state)", trunk.GetName())
	} else if currentBranch.GetName() != trunk.GetName() {
		out.Info("Creating worktree from %s (current branch: %s)", trunk.GetName(), currentBranch.GetName())
	}

	// Validate name
	if opts.Name == "" {
		return nil, fmt.Errorf("worktree name is required")
	}

	// Validate name doesn't contain path separators or other problematic characters
	if strings.ContainsAny(opts.Name, "/\\:*?\"<>|") {
		return nil, fmt.Errorf("worktree name cannot contain path separators or special characters: /\\:*?\"<>|")
	}

	// Check for duplicate worktree names
	existingWorktrees, err := eng.ListManagedWorktrees()
	if err != nil {
		return nil, fmt.Errorf("failed to check existing worktrees: %w", err)
	}
	for _, wt := range existingWorktrees {
		if wt.Name == opts.Name {
			return nil, fmt.Errorf("worktree with name %q already exists", opts.Name)
		}
	}

	// Generate anchor branch name using the branch naming pattern
	cfg, _ := config.LoadConfig(repoRoot)
	patternStr := cfg.BranchNamePattern()
	pattern, err := config.NewBranchPattern(patternStr)
	if err != nil {
		return nil, fmt.Errorf("invalid branch pattern: %w", err)
	}

	// Use the name as the "message" for the branch pattern, with "-wt" suffix
	anchorBranchName, err := pattern.GetBranchName(ctx, opts.Name+"-wt", opts.Scope)
	if err != nil {
		return nil, fmt.Errorf("failed to generate anchor branch name: %w", err)
	}

	// Check if branch already exists
	for _, b := range eng.AllBranches() {
		if b.GetName() == anchorBranchName {
			return nil, fmt.Errorf("branch %s already exists", anchorBranchName)
		}
	}

	// Get trunk SHA for the anchor branch
	trunkSHA, err := trunk.GetRevision()
	if err != nil {
		return nil, fmt.Errorf("failed to get trunk revision: %w", err)
	}

	// Create the anchor branch at trunk HEAD
	if err := eng.CreateBranch(ctx.Context, anchorBranchName, trunkSHA); err != nil {
		return nil, fmt.Errorf("failed to create anchor branch: %w", err)
	}

	// Set up metadata: parent = trunk, branchType = worktree-anchor, scope = provided
	anchorBranch := eng.GetBranch(anchorBranchName)
	if err := eng.SetParent(ctx.Context, anchorBranch, trunk); err != nil {
		// Clean up branch on failure
		cleanupAnchorBranch(ctx.Context, eng, anchorBranchName, out)
		return nil, fmt.Errorf("failed to set parent: %w", err)
	}

	if err := eng.SetBranchType(anchorBranch, git.BranchTypeWorktreeAnchor); err != nil {
		cleanupAnchorBranch(ctx.Context, eng, anchorBranchName, out)
		return nil, fmt.Errorf("failed to set branch type: %w", err)
	}

	if opts.Scope != "" {
		if err := eng.SetScope(ctx.Context, anchorBranch, engine.NewScope(opts.Scope)); err != nil {
			cleanupAnchorBranch(ctx.Context, eng, anchorBranchName, out)
			return nil, fmt.Errorf("failed to set scope: %w", err)
		}
	}

	// Create the worktree
	worktreePath, err := createWorktreeForAnchor(ctx, eng, repoRoot, opts.Name, anchorBranchName)
	if err != nil {
		cleanupAnchorBranch(ctx.Context, eng, anchorBranchName, out)
		return nil, err
	}

	out.Success("Created worktree %s", style.ColorBranchName(opts.Name, false))
	out.Info("  Anchor branch: %s", style.ColorBranchName(anchorBranchName, false))
	out.Info("  Path: %s", style.ColorDim(worktreePath))
	if opts.Scope != "" {
		out.Info("  Scope: %s", style.ColorDim(opts.Scope))
	}
	out.Newline()

	// Run post-create hooks
	if err := RunPostCreateHooks(ctx, worktreePath); err != nil {
		out.Warn("Post-create hooks failed: %v", err)
	}

	return &CreateResult{
		Name:         opts.Name,
		AnchorBranch: anchorBranchName,
		Path:         worktreePath,
	}, nil
}

// createWorktreeForAnchor creates a worktree for the given anchor branch and registers it
func createWorktreeForAnchor(ctx *app.Context, eng engine.Engine, repoRoot string, name string, anchorBranch string) (string, error) {
	// Get worktree base path from config, or use default
	cfg, _ := config.LoadConfig(repoRoot)
	basePath := cfg.WorktreeBasePath()

	// Default: sibling directory named {repo}-stacks
	if basePath == "" {
		repoName := filepath.Base(repoRoot)
		basePath = filepath.Join(filepath.Dir(repoRoot), repoName+"-stacks")
	}

	// Worktree path: basePath/name
	worktreePath := filepath.Join(basePath, name)

	// Check if path already exists
	if _, err := os.Stat(worktreePath); err == nil {
		return "", fmt.Errorf("worktree path %s already exists; remove it first or choose a different name", worktreePath)
	}

	// Create the worktree (non-detached, pointing to the anchor branch)
	if err := eng.AddWorktree(ctx.Context, worktreePath, anchorBranch, false); err != nil {
		return "", fmt.Errorf("failed to create worktree: %w", err)
	}

	// Register the worktree in local refs with the name
	if err := eng.RegisterWorktreeWithName(anchorBranch, worktreePath, name); err != nil {
		// Clean up worktree on registration failure
		_ = eng.RemoveWorktree(ctx.Context, worktreePath)
		return "", fmt.Errorf("failed to register worktree: %w", err)
	}

	return worktreePath, nil
}

// cleanupAnchorBranch cleans up an anchor branch on failure and logs any cleanup errors
func cleanupAnchorBranch(ctx context.Context, eng engine.Engine, branchName string, out output.Output) {
	if err := eng.DeleteBranch(ctx, eng.GetBranch(branchName)); err != nil {
		out.Warn("Failed to clean up anchor branch %s: %v", branchName, err)
	}
}
