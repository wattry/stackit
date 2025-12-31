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

	"stackit.dev/stackit/internal/app"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/tui/style"
	"stackit.dev/stackit/internal/utils/concurrency"
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
func ForeachAction(ctx *app.Context, opts ForeachOptions) error {
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

func foreachSequential(ctx *app.Context, opts ForeachOptions, branches []engine.Branch) error {
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

func foreachParallel(ctx *app.Context, opts ForeachOptions, branches []engine.Branch) error {
	numJobs := opts.Jobs
	if numJobs <= 0 {
		numJobs = stdruntime.NumCPU()
	}

	ctx.Splog.Info("Running on %d branches in parallel (jobs=%d)...", len(branches), numJobs)

	fullCommand := strings.Join(append([]string{opts.Command}, opts.Args...), " ")

	type result struct {
		branchName string
		output     string
		err        error
	}

	results := make(chan result, len(branches))

	// Context for canceling remaining jobs if fail-fast is on
	runCtx, cancel := context.WithCancel(ctx.Context)
	defer cancel()

	var mu sync.Mutex
	var firstErr error

	// Filter out trunk branches
	branchesToProcess := make([]engine.Branch, 0, len(branches))
	for _, b := range branches {
		if !b.IsTrunk() {
			branchesToProcess = append(branchesToProcess, b)
		}
	}

	concurrency.RunWithWorkers(branchesToProcess, numJobs, func(branch engine.Branch) {
		select {
		case <-runCtx.Done():
			return
		default:
		}

		res := executeCommandOnBranch(runCtx, ctx, branch, fullCommand)

		if res.err != nil && opts.FailFast {
			mu.Lock()
			if firstErr == nil {
				firstErr = res.err
			}
			mu.Unlock()
			cancel()
		}
		results <- res
	})
	close(results)

	for res := range results {
		ctx.Splog.Info("\nBranch: %s", style.ColorBranchName(res.branchName, false))
		if res.output != "" {
			// Using strings.TrimSuffix to avoid double newlines from splog.Info
			ctx.Splog.Info("%s", strings.TrimSuffix(res.output, "\n"))
		}

		if res.err != nil {
			ctx.Splog.Error("✗ Command failed: %v", res.err)
		} else {
			ctx.Splog.Info("✓ Command succeeded")
		}
	}

	if firstErr != nil {
		return fmt.Errorf("command failed on one or more branches")
	}

	return nil
}

func executeCommandOnBranch(ctx context.Context, appCtx *app.Context, branch engine.Branch, fullCommand string) struct {
	branchName string
	output     string
	err        error
} {
	res := struct {
		branchName string
		output     string
		err        error
	}{branchName: branch.GetName()}

	// Create a temporary directory for the worktree
	tmpDir, err := os.MkdirTemp("", "stackit-foreach-*")
	if err != nil {
		res.err = fmt.Errorf("failed to create temporary directory: %w", err)
		return res
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	worktreePath := filepath.Join(tmpDir, "worktree")

	// Add detached worktree
	if err := appCtx.Engine.AddWorktree(ctx, worktreePath, branch.GetName(), true); err != nil {
		res.err = fmt.Errorf("failed to add worktree: %w", err)
		return res
	}
	defer func() {
		// Use context.Background for cleanup to ensure it runs even if ctx is canceled
		if cleanupErr := appCtx.Engine.RemoveWorktree(context.Background(), worktreePath); cleanupErr != nil {
			appCtx.Splog.Debug("Failed to remove worktree at %s: %v", worktreePath, cleanupErr)
		}
	}()

	var output strings.Builder
	cmd := exec.CommandContext(ctx, "/bin/sh", "-c", fullCommand)
	cmd.Stdout = &output
	cmd.Stderr = &output
	cmd.Dir = worktreePath
	cmd.Env = os.Environ()

	if err := cmd.Run(); err != nil {
		res.err = err
	}
	res.output = output.String()
	return res
}
