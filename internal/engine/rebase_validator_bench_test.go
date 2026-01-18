package engine

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"stackit.dev/stackit/internal/git"
)

// BenchmarkWorktreeCreation specifically benchmarks worktree creation
func BenchmarkWorktreeCreation(b *testing.B) {
	// Setup: Create a test repo
	tmpDir, err := os.MkdirTemp("", "stackit-bench-worktree-*")
	if err != nil {
		b.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	repoPath := filepath.Join(tmpDir, "repo")
	if err := os.Mkdir(repoPath, 0o755); err != nil {
		b.Fatalf("failed to create repo dir: %v", err)
	}

	// Initialize git repo
	g := git.NewRunnerWithPath(repoPath, nil)
	ctx := context.Background()

	if _, err := g.RunGitCommandWithContext(ctx, "init"); err != nil {
		b.Fatalf("failed to init repo: %v", err)
	}

	// Configure git
	if _, err := g.RunGitCommandWithContext(ctx, "config", "user.name", "Test User"); err != nil {
		b.Fatalf("failed to configure git: %v", err)
	}
	if _, err := g.RunGitCommandWithContext(ctx, "config", "user.email", "test@example.com"); err != nil {
		b.Fatalf("failed to configure git: %v", err)
	}

	// Create initial commit
	testFile := filepath.Join(repoPath, "test.txt")
	if err := os.WriteFile(testFile, []byte("initial content\n"), 0o644); err != nil {
		b.Fatalf("failed to write test file: %v", err)
	}
	if _, err := g.RunGitCommandWithContext(ctx, "add", "test.txt"); err != nil {
		b.Fatalf("failed to add file: %v", err)
	}
	if err := g.CommitWithOptions(git.CommitOptions{Message: "Initial commit"}); err != nil {
		b.Fatalf("failed to create initial commit: %v", err)
	}

	// Create engine
	eng, err := NewEngine(Options{
		RepoRoot: repoPath,
		Trunk:    "main",
	})
	if err != nil {
		b.Fatalf("failed to create engine: %v", err)
	}

	b.ResetTimer()

	// Benchmark worktree creation
	for i := 0; i < b.N; i++ {
		_, cleanup, err := eng.CreateTemporaryWorktree(ctx, "HEAD", "stackit-bench-*")
		if err != nil {
			b.Fatalf("CreateTemporaryWorktree failed: %v", err)
		}
		cleanup()
	}
}
