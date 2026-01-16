package git

import (
	"context"
	"fmt"
	"strings"
)

func (r *runner) StageAll(ctx context.Context) error {
	_, err := r.RunGitCommandWithContext(ctx, "add", "-A")
	if err != nil {
		return fmt.Errorf("failed to stage all changes: %w", err)
	}
	return nil
}

func (r *runner) StagePatch(_ context.Context) error {
	return r.RunGitCommandInteractive("add", "-p")
}

func (r *runner) StageTracked(ctx context.Context) error {
	_, err := r.RunGitCommandWithContext(ctx, "add", "-u")
	if err != nil {
		return fmt.Errorf("failed to stage tracked changes: %w", err)
	}
	return nil
}

func (r *runner) AddAll(ctx context.Context) error {
	return r.StageAll(ctx)
}

func (r *runner) StageChanges(ctx context.Context, opts StagingOptions) error {
	if opts.Patch && !opts.All {
		return r.RunGitCommandInteractive("add", "-p")
	}

	if opts.All {
		return r.StageAll(ctx)
	}

	if opts.Update {
		_, err := r.RunGitCommandWithContext(ctx, "add", "-u")
		return err
	}

	return nil
}

func (r *runner) HasStagedChanges(ctx context.Context) (bool, error) {
	output, err := r.RunGitCommandWithContext(ctx, "diff", "--cached", "--shortstat")
	if err != nil {
		return false, fmt.Errorf("failed to check staged changes: %w", err)
	}
	return strings.TrimSpace(output) != "", nil
}

func (r *runner) HasUnstagedChanges(ctx context.Context) (bool, error) {
	// Use git diff to check for unstaged changes to tracked files
	// This is more reliable than parsing porcelain output which gets trimmed
	output, err := r.RunGitCommandWithContext(ctx, "diff", "--name-only")
	if err != nil {
		return false, fmt.Errorf("failed to check unstaged changes: %w", err)
	}
	return strings.TrimSpace(output) != "", nil
}

func (r *runner) HasUntrackedFiles(ctx context.Context) (bool, error) {
	output, err := r.RunGitCommandWithContext(ctx, "ls-files", "--others", "--exclude-standard")
	if err != nil {
		return false, fmt.Errorf("failed to check for untracked files: %w", err)
	}
	return strings.TrimSpace(output) != "", nil
}

func (r *runner) GetUntrackedFiles(ctx context.Context) ([]string, error) {
	output, err := r.RunGitCommandWithContext(ctx, "ls-files", "--others", "--exclude-standard")
	if err != nil {
		return nil, fmt.Errorf("failed to get untracked files: %w", err)
	}
	if strings.TrimSpace(output) == "" {
		return nil, nil
	}
	lines := strings.Split(strings.TrimSpace(output), "\n")
	return lines, nil
}

func (r *runner) ParseStagedHunks(ctx context.Context) ([]Hunk, error) {
	diffOutput, err := r.RunGitCommandRawWithContext(ctx, "diff", "--cached")
	if err != nil {
		return nil, fmt.Errorf("failed to get staged diff: %w", err)
	}

	return ParseDiffOutput(diffOutput)
}

// StageHunks stages specific hunks by applying them as patches to the index.
// This allows selective staging without using interactive git add -p.
func (r *runner) StageHunks(ctx context.Context, hunks []Hunk) error {
	if len(hunks) == 0 {
		return nil
	}

	// Build a patch from the selected hunks
	patch := BuildPatchFromHunks(hunks)
	if patch == "" {
		return nil
	}

	// Apply the patch to the index using git apply --cached
	// We use --3way to handle conflicts better and --allow-empty for edge cases
	_, err := r.runGitInternal(ctx, patch, nil, true, "apply", "--cached", "--3way")
	if err != nil {
		// Try without --3way as a fallback (some git versions have issues)
		_, err = r.runGitInternal(ctx, patch, nil, true, "apply", "--cached")
		if err != nil {
			return fmt.Errorf("failed to apply patch: %w", err)
		}
	}

	return nil
}

// UnstageAll removes all changes from the staging area.
func (r *runner) UnstageAll(ctx context.Context) error {
	_, err := r.RunGitCommandWithContext(ctx, "reset", "HEAD")
	if err != nil {
		return fmt.Errorf("failed to unstage changes: %w", err)
	}
	return nil
}
