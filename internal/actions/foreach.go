package actions

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	stdruntime "runtime"
	"strings"
	"sync"

	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/internal/tui/style"
)

// ForeachOptions contains options for the foreach command
type ForeachOptions struct {
	Command  string
	Args     []string
	Scope    engine.StackRange
	FailFast bool
	Parallel bool
	Jobs     int
}

// ForeachAction executes a command on each branch in the stack
func ForeachAction(ctx *runtime.Context, opts ForeachOptions) error {
	eng := ctx.Engine

	currentBranch := eng.CurrentBranch()
	if currentBranch == nil {
		return fmt.Errorf("not on a branch")
	}

	// Get branches based on scope
	branches := currentBranch.GetRelativeStack(opts.Scope)
	if len(branches) == 0 {
		return nil
	}

	if opts.Parallel {
		return foreachParallel(ctx, opts, branches)
	}
	return foreachSequential(ctx, opts, branches)
}

func foreachSequential(ctx *runtime.Context, opts ForeachOptions, branches []engine.Branch) error {
	eng := ctx.Engine
	splog := ctx.Splog

	currentBranch := eng.CurrentBranch()
	originalBranchName := ""
	if currentBranch != nil {
		originalBranchName = currentBranch.GetName()
	}

	defer func() {
		// Always try to return to the original branch
		if originalBranchName != "" {
			if err := eng.CheckoutBranch(ctx.Context, eng.GetBranch(originalBranchName)); err != nil {
				splog.Error("Failed to return to original branch %s: %v", originalBranchName, err)
			}
		}
	}()

	// Join command and args into a single string for shell execution
	fullCommand := strings.Join(append([]string{opts.Command}, opts.Args...), " ")

	for _, branch := range branches {
		// Skip trunk
		if branch.IsTrunk() {
			continue
		}

		isCurrent := branch.GetName() == originalBranchName
		splog.Info("\nRunning on branch %s...", style.ColorBranchName(branch.GetName(), isCurrent))

		if err := eng.CheckoutBranch(ctx.Context, branch); err != nil {
			splog.Error("Failed to checkout %s: %v", branch.GetName(), err)
			if opts.FailFast {
				return err
			}
			continue
		}

		// Execute the command via shell
		cmd := exec.CommandContext(ctx.Context, "/bin/sh", "-c", fullCommand)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		cmd.Dir = ctx.RepoRoot

		if err := cmd.Run(); err != nil {
			splog.Error("✗ Command failed on branch %s: %v", branch.GetName(), err)
			if opts.FailFast {
				return fmt.Errorf("command failed on branch %s", branch.GetName())
			}
		} else {
			splog.Info("✓ Command succeeded on branch %s", branch.GetName())
		}
	}

	return nil
}

func foreachParallel(ctx *runtime.Context, opts ForeachOptions, branches []engine.Branch) error {
	eng := ctx.Engine
	splog := ctx.Splog

	numJobs := opts.Jobs
	if numJobs <= 0 {
		numJobs = stdruntime.NumCPU()
	}

	splog.Info("Running on %d branches in parallel (jobs=%d)...", len(branches), numJobs)

	fullCommand := strings.Join(append([]string{opts.Command}, opts.Args...), " ")

	type result struct {
		branchName string
		output     string
		err        error
	}

	results := make(chan result, len(branches))
	sem := make(chan struct{}, numJobs)
	var wg sync.WaitGroup

	// Context for canceling remaining jobs if fail-fast is on
	runCtx, cancel := context.WithCancel(ctx.Context)
	defer cancel()

	var mu sync.Mutex
	var firstErr error

	for _, b := range branches {
		if b.IsTrunk() {
			continue
		}

		wg.Add(1)
		go func(branch engine.Branch) {
			defer wg.Done()

			select {
			case <-runCtx.Done():
				return
			case sem <- struct{}{}:
				defer func() { <-sem }()
			}

			res := result{branchName: branch.GetName()}

			// Create a temporary directory for the worktree
			tmpDir, err := os.MkdirTemp("", "stackit-foreach-*")
			if err != nil {
				res.err = fmt.Errorf("failed to create temporary directory: %w", err)
				results <- res
				return
			}
			defer func() {
				_ = os.RemoveAll(tmpDir)
			}()

			worktreePath := filepath.Join(tmpDir, "worktree")

			// Add detached worktree
			if err := eng.AddWorktree(runCtx, worktreePath, branch.GetName(), true); err != nil {
				res.err = fmt.Errorf("failed to add worktree: %w", err)
				results <- res
				return
			}
			defer func() {
				// Use context.Background for cleanup to ensure it runs even if runCtx is canceled
				if cleanupErr := eng.RemoveWorktree(context.Background(), worktreePath); cleanupErr != nil {
					splog.Debug("Failed to remove worktree at %s: %v", worktreePath, cleanupErr)
				}
			}()

			var output strings.Builder
			cmd := exec.CommandContext(runCtx, "/bin/sh", "-c", fullCommand)
			cmd.Stdout = &output
			cmd.Stderr = &output
			cmd.Dir = worktreePath
			cmd.Env = os.Environ()

			if err := cmd.Run(); err != nil {
				res.err = err
				if opts.FailFast {
					mu.Lock()
					if firstErr == nil {
						firstErr = err
					}
					mu.Unlock()
					cancel()
				}
			}
			res.output = output.String()
			results <- res
		}(b)
	}

	// Close results channel when all goroutines are done
	go func() {
		wg.Wait()
		close(results)
	}()

	for res := range results {
		splog.Info("\nBranch: %s", style.ColorBranchName(res.branchName, false))
		if res.output != "" {
			// Using strings.TrimSuffix to avoid double newlines from splog.Info
			splog.Info("%s", strings.TrimSuffix(res.output, "\n"))
		}

		if res.err != nil {
			splog.Error("✗ Command failed: %v", res.err)
		} else {
			splog.Info("✓ Command succeeded")
		}
	}

	if firstErr != nil {
		return fmt.Errorf("command failed on one or more branches")
	}

	return nil
}
