package actions

import (
	"errors"
	"fmt"
	stdruntime "runtime"
	"strings"
	"sync"

	"stackit.dev/stackit/internal/app"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/handlers"
	"stackit.dev/stackit/internal/rerere"
	"stackit.dev/stackit/internal/tui"
	"stackit.dev/stackit/internal/utils"
)

var newWorktreeEngine = engine.NewEngineForWorktree

// RestackOptions contains options for the restack command
type RestackOptions struct {
	BranchName         string
	Scope              engine.StackRange
	AllStacks          bool
	StackRoots         []string
	ContinueOnConflict bool
	Parallel           bool // Run independent stack groups in parallel worktrees
	Jobs               int  // Number of parallel workers (0 = NumCPU)
}

// RestackAction performs the restack operation
func RestackAction(ctx *app.Context, opts RestackOptions, handler handlers.RestackHandler) error {
	eng := ctx.Engine
	out := ctx.Output

	var branchGroups []restackBranchGroup
	if opts.AllStacks || len(opts.StackRoots) > 0 {
		var err error
		branchGroups, err = branchGroupsForIndependentStacks(eng, opts)
		if err != nil {
			return err
		}
	} else {
		// Get branches to restack based on scope
		branch := eng.GetBranch(opts.BranchName)
		graph := eng.Graph(engine.SortStrategyAlphabetical)
		branchGroups = []restackBranchGroup{{
			branches: graph.Range(branch, opts.Scope),
		}}
	}

	branchCount := countRestackBranches(branchGroups)
	if branchCount == 0 {
		out.Info("No branches to restack.")
		return nil
	}

	ctx.Logger.Info("restack started branchCount=%v", branchCount)

	// Take snapshot before modifying the repository
	snapshotOpts := NewSnapshot("restack",
		WithArg(opts.BranchName),
		WithFlag(opts.AllStacks, "--all-stacks"),
		WithFlagValue("--stacks", joinStrings(opts.StackRoots)),
		WithFlag(opts.ContinueOnConflict, "--continue-on-conflict"),
		WithFlag(opts.Parallel, "--parallel"),
	)
	if err := eng.TakeSnapshot(snapshotOpts); err != nil {
		// Log but don't fail - snapshot is best effort
		out.Debug("Failed to take snapshot: %v", err)
	}

	// If no handler provided, use NullRestackHandler (silent)
	if handler == nil {
		handler = &handlers.NullRestackHandler{}
	}

	interactiveRererePrompt := ctx.Interactive && !ctx.Quiet && tui.IsTTY()
	if _, jsonOutput := handler.(*handlers.JSONRestackHandler); jsonOutput {
		interactiveRererePrompt = false
	}
	pauser, _ := handler.(rerere.Pauser)
	if _, err := rerere.EnsureEnabled(ctx.Context, ctx.Git(), interactiveRererePrompt, pauser); err != nil {
		out.Warn("Failed to enable git rerere: %v", err)
	}

	// Use RestackHandler for consistent output
	handler.OnRestackStart(branchCount)

	var restacked, skipped int
	var conflicts []string

	// Parallel mode: dispatch independent stack groups to separate worktrees.
	if opts.Parallel && len(branchGroups) > 1 {
		var err error
		restacked, skipped, conflicts, err = restackGroupsParallel(ctx, opts, branchGroups, handler)
		ctx.Logger.Info("restack completed (parallel) restacked=%v skipped=%v conflicts=%v", restacked, skipped, len(conflicts))
		handler.OnRestackComplete(restacked, skipped, conflicts)
		if err != nil {
			return fmt.Errorf("restack failed: %w", err)
		}
		return nil
	}

	conflictMode := ConflictModeEnterWorkflow
	if opts.ContinueOnConflict {
		conflictMode = ConflictModeContinue
	}

	for _, group := range branchGroups {
		// Wrap the progress callback to inject the stack root for this group.
		groupRoot := group.rootBranch
		progress := func(p RestackProgress) {
			p.StackRoot = groupRoot
			handleRestackProgress(eng, handler, p, &restacked, &skipped, &conflicts)
		}

		// Sort each independent stack topologically. Keeping groups separate lets
		// --continue-on-conflict skip a conflicted stack while still restacking
		// other independent stacks.
		sortedBranches := eng.SortBranchesTopologically(group.branches)
		if err := RestackBranchesWithHandler(ctx, sortedBranches, progress, conflictMode); err != nil {
			handler.OnRestackComplete(restacked, skipped, conflicts)
			return fmt.Errorf("restack failed: %w", err)
		}
	}

	ctx.Logger.Info("restack completed restacked=%v skipped=%v conflicts=%v", restacked, skipped, len(conflicts))

	handler.OnRestackComplete(restacked, skipped, conflicts)
	return nil
}

// parallelResultCollector serializes progress updates, error accumulation, and
// counter bookkeeping across parallel restack workers. All methods take the
// mutex internally; callers must not hold it.
type parallelResultCollector struct {
	mu        sync.Mutex
	eng       engine.BranchReader
	handler   handlers.RestackHandler
	restacked int
	skipped   int
	conflicts []string
	errs      []error
}

// recordProgress routes a progress event from a worker through handleRestackProgress.
func (c *parallelResultCollector) recordProgress(p RestackProgress) {
	c.mu.Lock()
	defer c.mu.Unlock()
	handleRestackProgress(c.eng, c.handler, p, &c.restacked, &c.skipped, &c.conflicts)
}

// recordGroupFailure attributes every branch in a failed group to the handler
// as a conflict so the final summary reflects what the worker did not process.
// Without this a worktree/engine setup failure silently shows "skipped=0" while
// an entire stack failed to start.
func (c *parallelResultCollector) recordGroupFailure(group restackBranchGroup, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.errs = append(c.errs, err)
	for _, b := range group.branches {
		handleRestackProgress(c.eng, c.handler, RestackProgress{
			Branch:    b.GetName(),
			Result:    engine.RestackConflict,
			Conflict:  true,
			StackRoot: group.rootBranch,
		}, &c.restacked, &c.skipped, &c.conflicts)
	}
}

// recordRebaseError records an unexpected error returned from
// RestackBranchesWithHandler. Per-branch outcomes were already routed through
// recordProgress by the progress callback before the error surfaced, so only
// the error itself needs to be captured.
func (c *parallelResultCollector) recordRebaseError(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.errs = append(c.errs, err)
}

// joinedError returns the accumulated errors as a single joined error, or nil.
func (c *parallelResultCollector) joinedError() error {
	if len(c.errs) == 0 {
		return nil
	}
	return errors.Join(c.errs...)
}

// restackGroupsParallel runs each independent stack group in its own temporary
// worktree. The groups have no shared branches, so rebases + ref updates are
// safe to interleave. Progress callbacks and handler calls are serialized
// through parallelResultCollector to prevent data races on shared counters.
func restackGroupsParallel(
	ctx *app.Context,
	opts RestackOptions,
	groups []restackBranchGroup,
	handler handlers.RestackHandler,
) (restacked, skipped int, conflicts []string, err error) {
	eng := ctx.Engine

	numJobs := opts.Jobs
	if numJobs <= 0 {
		numJobs = stdruntime.NumCPU()
	}
	if numJobs > len(groups) {
		numJobs = len(groups)
	}

	// Prune stale worktrees once before creating new ones.
	_ = eng.PruneWorktrees(ctx.Context)

	// Snapshot the parent engine state once — each worktree engine gets a copy.
	snapshot := eng.SnapshotForWorktree()

	collector := &parallelResultCollector{eng: eng, handler: handler}

	utils.RunWithWorkers(groups, numJobs, func(group restackBranchGroup) {
		// Create a temporary worktree for this group. PruneSkip because we
		// pruned once above and don't want N workers racing on prune.
		wtPath, cleanup, err := eng.CreateTemporaryWorktreeSkipPrune(ctx.Context, eng.Trunk().GetName(), "stackit-restack-*")
		if err != nil {
			wrappedErr := fmt.Errorf("stack %s: create worktree: %w", group.rootBranch, err)
			ctx.Logger.Warn("failed to create worktree for parallel restack: %v", wrappedErr)
			collector.recordGroupFailure(group, wrappedErr)
			return
		}
		defer cleanup()

		// Build a per-worktree engine from the snapshot.
		wtEngine, err := newWorktreeEngine(engine.WorktreeEngineOptions{
			WorktreePath: wtPath,
			Snapshot:     snapshot,
		})
		if err != nil {
			wrappedErr := fmt.Errorf("stack %s: create worktree engine: %w", group.rootBranch, err)
			ctx.Logger.Warn("failed to create worktree engine: %v", wrappedErr)
			collector.recordGroupFailure(group, wrappedErr)
			return
		}

		// Shallow-copy the app context, swapping in the worktree engine.
		wtCtx := *ctx
		wtCtx.Engine = wtEngine
		wtCtx.RepoRoot = wtPath

		groupRoot := group.rootBranch
		progress := func(p RestackProgress) {
			p.StackRoot = groupRoot
			collector.recordProgress(p)
		}

		// Parallel mode always reports conflicts via callback: the interactive conflict
		// workflow writes rebase state into the worktree, which defer cleanup() tears
		// down, so entering it would silently destroy what the user needs to resolve.
		sortedBranches := eng.SortBranchesTopologically(group.branches)
		if err := RestackBranchesWithHandler(&wtCtx, sortedBranches, progress, ConflictModeContinue); err != nil {
			wrappedErr := fmt.Errorf("stack %s: %w", group.rootBranch, err)
			ctx.Logger.Warn("parallel restack group failed: %v", wrappedErr)
			collector.recordRebaseError(wrappedErr)
		}
	})

	// Rebuild the main engine's cache so it sees the ref changes made by the
	// worktree engines (they share the same .git directory).
	if err := eng.Rebuild(eng.Trunk().GetName()); err != nil {
		ctx.Logger.Warn("failed to rebuild engine after parallel restack: %v", err)
	}

	return collector.restacked, collector.skipped, collector.conflicts, collector.joinedError()
}

type restackBranchGroup struct {
	rootBranch string // independent stack root name (direct child of trunk)
	branches   []engine.Branch
}

func branchGroupsForIndependentStacks(eng engine.BranchReader, opts RestackOptions) ([]restackBranchGroup, error) {
	stacks := engine.DiscoverIndependentStacks(eng)
	if len(opts.StackRoots) > 0 {
		stackByRoot := make(map[string]engine.IndependentStack, len(stacks))
		for _, stack := range stacks {
			stackByRoot[stack.RootBranch] = stack
		}

		filtered := make([]engine.IndependentStack, 0, len(opts.StackRoots))
		for _, root := range opts.StackRoots {
			stack, ok := stackByRoot[root]
			if !ok {
				return nil, fmt.Errorf("stack root %s not found", root)
			}
			filtered = append(filtered, stack)
		}
		stacks = filtered
	}

	groups := make([]restackBranchGroup, 0, len(stacks))
	for _, stack := range stacks {
		branches := make([]engine.Branch, 0, len(stack.Branches))
		for _, branchName := range stack.Branches {
			branch := eng.GetBranch(branchName)
			if branch.IsTracked() && !branch.IsTrunk() {
				branches = append(branches, branch)
			}
		}
		if len(branches) > 0 {
			groups = append(groups, restackBranchGroup{
				rootBranch: stack.RootBranch,
				branches:   branches,
			})
		}
	}
	return groups, nil
}

func joinStrings(values []string) string {
	return strings.Join(values, ",")
}

func countRestackBranches(groups []restackBranchGroup) int {
	count := 0
	for _, group := range groups {
		count += len(group.branches)
	}
	return count
}

func handleRestackProgress(
	eng engine.BranchReader,
	handler handlers.RestackHandler,
	p RestackProgress,
	restacked *int,
	skipped *int,
	conflicts *[]string,
) {
	res := handlers.RestackDone
	switch p.Result {
	case engine.RestackDone:
		*restacked++
		res = handlers.RestackDone
	case engine.RestackUnneeded:
		res = handlers.RestackUnneeded
	case engine.RestackConflict:
		*skipped++
		*conflicts = append(*conflicts, p.Branch)
		res = handlers.RestackConflict
	}

	// Determine parent name for consistent output
	parentName := ""
	br := eng.GetBranch(p.Branch)
	if br.GetName() != "" {
		if parent := br.GetParent(); parent != nil {
			parentName = parent.GetName()
		} else {
			parentName = eng.Trunk().GetName()
		}
	}

	// PR number is not always available without extra fetching, but we can try
	var prNumber *int
	if br.GetName() != "" {
		if pr, err := eng.GetPrInfo(br); err == nil && pr != nil {
			num := pr.Number()
			prNumber = num
		}
	}

	handler.OnRestackBranch(p.Branch, res, p.NewRev, prNumber, p.LockReason, p.Frozen, p.IsCurrent, parentName, p.Reparented, p.OldParent, p.NewParent, p.RerereResolvedCount)

	// Enrich JSON handler with stack root info (not part of the interface to avoid
	// touching all implementors for a JSON-only concern).
	if p.StackRoot != "" {
		if jsonHandler, ok := handler.(*handlers.JSONRestackHandler); ok {
			jsonHandler.SetLastBranchStackRoot(p.Branch, p.StackRoot)
		}
	}
}
