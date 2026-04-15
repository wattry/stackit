package actions

import (
	"fmt"
	"strings"

	"stackit.dev/stackit/internal/app"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/handlers"
	"stackit.dev/stackit/internal/rerere"
	"stackit.dev/stackit/internal/tui"
)

// RestackOptions contains options for the restack command
type RestackOptions struct {
	BranchName         string
	Scope              engine.StackRange
	AllStacks          bool
	StackRoots         []string
	ContinueOnConflict bool
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

	progress := func(p RestackProgress) {
		handleRestackProgress(eng, handler, p, &restacked, &skipped, &conflicts)
	}

	for _, group := range branchGroups {
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

type restackBranchGroup struct {
	branches []engine.Branch
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
				branches: branches,
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
}
