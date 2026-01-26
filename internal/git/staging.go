package git

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
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
// New files are handled specially since git apply --cached cannot create files
// that don't exist in the working tree.
func (r *runner) StageHunks(ctx context.Context, hunks []Hunk) error {
	if len(hunks) == 0 {
		return nil
	}

	// Separate new files from modifications
	var newFileHunks []Hunk
	var modHunks []Hunk
	for _, h := range hunks {
		if h.IsNewFile {
			newFileHunks = append(newFileHunks, h)
		} else {
			modHunks = append(modHunks, h)
		}
	}

	// Handle new files: extract content and write to disk, then stage with git add
	for _, h := range newFileHunks {
		content := extractContentFromHunk(h)
		filePath := filepath.Join(r.repoRoot, h.File)

		// Create parent directories if they don't exist
		if dir := filepath.Dir(filePath); dir != "." {
			if err := os.MkdirAll(dir, 0o750); err != nil {
				return fmt.Errorf("failed to create directory for new file %s: %w", h.File, err)
			}
		}

		// Determine file mode (default to 0644 for regular files)
		fileMode := os.FileMode(0o644)
		if h.FileMode == "100755" {
			fileMode = 0o755
		}

		if err := os.WriteFile(filePath, []byte(content), fileMode); err != nil {
			return fmt.Errorf("failed to write new file %s: %w", h.File, err)
		}

		// Stage the new file
		if _, err := r.RunGitCommandWithContext(ctx, "add", h.File); err != nil {
			return fmt.Errorf("failed to stage new file %s: %w", h.File, err)
		}
	}

	// Handle modifications with existing git apply --cached
	if len(modHunks) > 0 {
		patch := BuildPatchFromHunks(modHunks)
		if patch != "" {
			// Apply the patch to the index using git apply --cached
			// We use --3way to handle conflicts better
			_, err := r.runGitInternal(ctx, patch, nil, true, "apply", "--cached", "--3way")
			if err != nil {
				// Try without --3way as a fallback (some git versions have issues with --3way)
				_, fallbackErr := r.runGitInternal(ctx, patch, nil, true, "apply", "--cached")
				if fallbackErr != nil {
					return fmt.Errorf("failed to apply patch: %w (note: --3way also failed)", fallbackErr)
				}
			}
		}
	}

	return nil
}

// extractContentFromHunk extracts the file content from a new file hunk.
// It parses the unified diff format and returns only the added lines (without the + prefix).
func extractContentFromHunk(h Hunk) string {
	var lines []string
	for _, line := range strings.Split(h.Content, "\n") {
		// Skip the hunk header line (@@ ... @@)
		if strings.HasPrefix(line, "@@") {
			continue
		}
		// Include lines that start with + (but not the +++ header)
		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			lines = append(lines, strings.TrimPrefix(line, "+"))
		}
	}
	result := strings.Join(lines, "\n")
	// Ensure file ends with newline if there was content
	if len(lines) > 0 && !strings.HasSuffix(result, "\n") {
		result += "\n"
	}
	return result
}

// UnstageAll removes all changes from the staging area.
func (r *runner) UnstageAll(ctx context.Context) error {
	_, err := r.RunGitCommandWithContext(ctx, "reset", "HEAD")
	if err != nil {
		return fmt.Errorf("failed to unstage changes: %w", err)
	}
	return nil
}
