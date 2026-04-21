package actions

import (
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
		graph := engine.BuildStackGraph(eng, engine.SortStrategyAlphabetical, nil)
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
		restacked, skipped, conflicts = restackGroupsParallel(ctx, opts, branchGroups, handler)
		ctx.Logger.Info("restack completed (parallel) restacked=%v skipped=%v conflicts=%v", restacked, skipped, len(conflicts))
		handler.OnRestackComplete(restacked, skipped, conflicts)
		return nil
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
		if err := RestackBranchesWithHandler(ctx, sortedBranches, progress, !opts.ContinueOnConflict); err != nil {
			handler.OnRestackComplete(restacked, skipped, conflicts)
			return fmt.Errorf("restack failed: %w", err)
		}
	}

	ctx.Logger.Info("restack completed restacked=%v skipped=%v conflicts=%v", restacked, skipped, len(conflicts))

	handler.OnRestackComplete(restacked, skipped, conflicts)
	return nil
}

// restackGroupsParallel runs each independent stack group in its own temporary
// worktree. The groups have no shared branches, so rebases + ref updates are
// safe to interleave. Progress callbacks and handler calls are serialized via
// a mutex to prevent data races on shared counters.
func restackGroupsParallel(
	ctx *app.Context,
	opts RestackOptions,
	groups []restackBranchGroup,
	handler handlers.RestackHandler,
) (restacked, skipped int, conflicts []string) {
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

	// Shared result counters protected by a mutex.
	var mu sync.Mutex

	utils.RunWithWorkers(groups, numJobs, func(group restackBranchGroup) {
		// Create a temporary worktree for this group. PruneSkip because we
		// pruned once above and don't want N workers racing on prune.
		wtPath, cleanup, err := eng.CreateTemporaryWorktreeSkipPrune(ctx.Context, eng.Trunk().GetName(), "stackit-restack-*")
		if err != nil {
			ctx.Logger.Warn("failed to create worktree for parallel restack: %v", err)
			return
		}
		defer cleanup()

		// Build a per-worktree engine from the snapshot.
		wtEngine, err := engine.NewEngineForWorktree(engine.WorktreeEngineOptions{
			WorktreePath: wtPath,
			Snapshot:     snapshot,
		})
		if err != nil {
			ctx.Logger.Warn("failed to create worktree engine: %v", err)
			return
		}

		// Shallow-copy the app context, swapping in the worktree engine.
		wtCtx := *ctx
		wtCtx.Engine = wtEngine
		wtCtx.RepoRoot = wtPath

		// Thread-safe progress callback that funnels into the shared counters + handler.
		groupRoot := group.rootBranch
		progress := func(p RestackProgress) {
			p.StackRoot = groupRoot
			mu.Lock()
			defer mu.Unlock()
			handleRestackProgress(eng, handler, p, &restacked, &skipped, &conflicts)
		}

		sortedBranches := eng.SortBranchesTopologically(group.branches)
		if err := RestackBranchesWithHandler(&wtCtx, sortedBranches, progress, !opts.ContinueOnConflict); err != nil {
			ctx.Logger.Warn("parallel restack group failed: %v", err)
		}
	})

	// Rebuild the main engine's cache so it sees the ref changes made by the
	// worktree engines (they share the same .git directory).
	if err := eng.Rebuild(eng.Trunk().GetName()); err != nil {
		ctx.Logger.Warn("failed to rebuild engine after parallel restack: %v", err)
	}

	return restacked, skipped, conflicts
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
