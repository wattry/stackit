// Package merge provides CLI commands for merging stacked PRs.
package merge

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"stackit.dev/stackit/internal/app"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/github"
)

const (
	// DefaultMergeTimeout is the default timeout for waiting on a merge to complete
	DefaultMergeTimeout = 30 * time.Minute
	// DefaultMergePollInterval is the default interval between merge status checks
	DefaultMergePollInterval = 10 * time.Second
)

// Outcome represents the result of a merge orchestration attempt.
type Outcome int

const (
	// OutcomeMerged means the PR was merged (directly or via wait).
	OutcomeMerged Outcome = iota
	// OutcomeAutomergeEnabled means automerge was enabled (fire-and-forget).
	OutcomeAutomergeEnabled
)

type orchestrateMergeOptions struct {
	branchName  string
	prNumber    int
	prNodeID    string
	mergeMethod github.MergeMethod
	wait        bool
}

// orchestrateMerge implements a 3-tier merge strategy:
//  1. If the PR is already ready (CLEAN/HAS_HOOKS), merge directly via REST API.
//  2. Otherwise, try EnableAutoMerge. On success, optionally wait for merge.
//  3. If EnableAutoMerge fails:
//     - "clean status" error → direct merge (race: became ready between check and automerge).
//     - "not enabled on repo" + wait → poll until mergeable, then merge directly.
//     - "not enabled on repo" + no wait → error with --wait suggestion.
func orchestrateMerge(ctx *app.Context, opts orchestrateMergeOptions) (Outcome, error) {
	out := ctx.Output
	eng := ctx.Engine

	// Step 1: Check current merge state
	mergeableState, err := getMergeableStateWithRetry(ctx.Context, eng.Git(), opts.prNodeID)
	if err != nil {
		return 0, fmt.Errorf("failed to check PR mergeable state: %w", err)
	}
	if mergeableState.State == "MERGED" {
		out.Success("PR #%d is already merged", opts.prNumber)
		return OutcomeMerged, nil
	}
	if mergeableState.State != "OPEN" {
		return 0, fmt.Errorf("PR #%d is %s (not open)", opts.prNumber, mergeableState.State)
	}
	if !mergeableState.Mergeable {
		if strings.EqualFold(mergeableState.MergeStateText, "UNKNOWN") {
			return 0, fmt.Errorf("PR #%d mergeability is still being calculated by GitHub after 5 retries. Try again shortly", opts.prNumber)
		}
		return 0, formatUnmergeableError(opts.prNumber, mergeableState)
	}

	// If already CLEAN or HAS_HOOKS, merge directly
	if isReadyToMerge(mergeableState.MergeStateText) {
		out.Info("PR #%d is ready to merge — merging directly (method: %s)...", opts.prNumber, opts.mergeMethod)
		if err := ctx.GitHubClient.MergePullRequest(ctx.Context, opts.branchName, opts.mergeMethod); err != nil {
			return 0, fmt.Errorf("failed to merge PR #%d: %w", opts.prNumber, err)
		}
		out.Success("PR #%d merged successfully!", opts.prNumber)
		return OutcomeMerged, nil
	}

	// Step 2: Not immediately ready — try automerge
	out.Info("Enabling automerge on PR #%d (method: %s)...", opts.prNumber, opts.mergeMethod)
	if err := github.EnableAutoMerge(ctx.Context, eng.Git(), opts.prNodeID, opts.mergeMethod); err != nil {
		return handleAutoMergeError(ctx, opts, err)
	}
	out.Success("Automerge enabled on PR #%d", opts.prNumber)

	// Step 2a: If --wait, wait for merge to complete
	if opts.wait {
		out.Info("Waiting for PR #%d to be merged...", opts.prNumber)
		if err := github.WaitForPRMerge(ctx.Context, eng.Git(), opts.prNodeID, DefaultMergeTimeout, DefaultMergePollInterval); err != nil {
			return 0, fmt.Errorf("failed waiting for merge: %w", err)
		}
		out.Success("PR #%d merged successfully!", opts.prNumber)
		return OutcomeMerged, nil
	}

	// Fire-and-forget
	return OutcomeAutomergeEnabled, nil
}

// handleAutoMergeError implements step 3 of the merge strategy: handling EnableAutoMerge failures.
func handleAutoMergeError(ctx *app.Context, opts orchestrateMergeOptions, autoMergeErr error) (Outcome, error) {
	out := ctx.Output

	// "clean status" error → PR became ready between our check and the automerge call (race condition)
	if errors.Is(autoMergeErr, github.ErrPRCleanStatus) {
		out.Debug("Automerge returned clean status — PR became ready, merging directly")
		if err := ctx.GitHubClient.MergePullRequest(ctx.Context, opts.branchName, opts.mergeMethod); err != nil {
			return 0, fmt.Errorf("failed to merge PR #%d: %w", opts.prNumber, err)
		}
		out.Success("PR #%d merged successfully!", opts.prNumber)
		return OutcomeMerged, nil
	}

	// "not enabled on repo" → fall back to polling + direct merge if --wait
	if errors.Is(autoMergeErr, github.ErrAutoMergeNotEnabled) {
		if opts.wait {
			out.Info("Auto-merge not enabled on this repo — waiting for PR #%d to become mergeable...", opts.prNumber)
			state, err := github.WaitForMergeable(ctx.Context, ctx.Engine.Git(), opts.prNodeID, DefaultMergeTimeout, DefaultMergePollInterval)
			if errors.Is(err, github.ErrPRAlreadyMerged) {
				out.Success("PR #%d was merged externally!", opts.prNumber)
				return OutcomeMerged, nil
			}
			if err != nil {
				return 0, fmt.Errorf("failed waiting for PR #%d to become mergeable: %w", opts.prNumber, err)
			}

			out.Info("PR #%d is now %s — merging directly (method: %s)...", opts.prNumber, state.MergeStateText, opts.mergeMethod)
			if err := ctx.GitHubClient.MergePullRequest(ctx.Context, opts.branchName, opts.mergeMethod); err != nil {
				return 0, fmt.Errorf("failed to merge PR #%d: %w", opts.prNumber, err)
			}
			out.Success("PR #%d merged successfully!", opts.prNumber)
			return OutcomeMerged, nil
		}

		return 0, fmt.Errorf("auto-merge is not enabled for this repository. Use --wait to poll and merge directly, or enable auto-merge in repository settings")
	}

	// Other automerge error — pass through
	return 0, fmt.Errorf("failed to enable automerge on PR #%d: %w", opts.prNumber, autoMergeErr)
}

// isReadyToMerge returns true if the PR's mergeStateStatus indicates it can be merged immediately.
func isReadyToMerge(mergeStateText string) bool {
	switch mergeStateText {
	case "CLEAN", "HAS_HOOKS":
		return true
	default:
		return false
	}
}

// formatUnmergeableError produces a user-friendly error for a PR that is not mergeable.
func formatUnmergeableError(prNumber int, state *github.PRMergeableState) error {
	if state.MergeStateText != "" {
		return fmt.Errorf("PR #%d is not mergeable (%s). Please resolve conflicts and try again", prNumber, state.MergeStateText)
	}
	return fmt.Errorf("PR #%d is not mergeable. Please resolve conflicts and try again", prNumber)
}

// getMergeableStateWithRetry polls for PR mergeable state, retrying when GitHub reports UNKNOWN.
func getMergeableStateWithRetry(ctx context.Context, runner git.Runner, prNodeID string) (*github.PRMergeableState, error) {
	const (
		maxAttempts = 5
		retryDelay  = 2 * time.Second
	)

	var lastState *github.PRMergeableState
	for attempt := range maxAttempts {
		_ = attempt
		state, err := github.GetPRMergeableState(ctx, runner, prNodeID)
		if err != nil {
			return nil, err
		}
		lastState = state
		if !strings.EqualFold(state.MergeStateText, "UNKNOWN") {
			return state, nil
		}
		time.Sleep(retryDelay)
	}

	return lastState, nil
}
