package merge

import (
	"context"
	"fmt"
	"strings"
	"time"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/app"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/github"
)

// ConsolidationResult contains information about a completed consolidation
type ConsolidationResult struct {
	BranchName string
	PRNumber   int
	PRURL      string
}

// ConsolidateMergeExecutor handles stack consolidation merging
type ConsolidateMergeExecutor struct {
	plan              *Plan
	engine            mergeExecuteEngine
	ctx               *app.Context
	consolidationPR   *github.PullRequestInfo // Set after consolidation PR is merged
	consolidationUser string                  // GitHub username who performed the consolidation
	prGenerator       *PRContentGenerator

	handler   Handler // Optional progress handler for TUI updates
	stepIndex int     // Current step index for the handler
}

// NewConsolidateMergeExecutor creates a new consolidation executor
func NewConsolidateMergeExecutor(plan *Plan, engine mergeExecuteEngine, ctx *app.Context) *ConsolidateMergeExecutor {
	return &ConsolidateMergeExecutor{
		plan:        plan,
		engine:      engine,
		ctx:         ctx,
		prGenerator: NewPRContentGenerator(engine),
	}
}

// SetProgressHandler sets the progress handler and step index for reporting
func (c *ConsolidateMergeExecutor) SetProgressHandler(handler Handler, stepIndex int) {
	c.handler = handler
	c.stepIndex = stepIndex
}

// Execute performs stack consolidation merging
func (c *ConsolidateMergeExecutor) Execute(ctx context.Context, opts ExecuteOptions) (*ConsolidationResult, error) {
	splog := c.ctx.Output
	splog.Info("🔀 Starting stack consolidation merge...")

	if err := c.preValidateStack(ctx, opts.Force); err != nil {
		return nil, fmt.Errorf("pre-validation failed: %w", err)
	}

	splog.Info("Stack to consolidate:")
	for i, branchInfo := range c.plan.BranchesToMerge {
		symbol := "  ○"
		if i == len(c.plan.BranchesToMerge)-1 {
			symbol = "  ◉"
		}
		splog.Info("%s %s PR #%d", symbol, branchInfo.BranchName, branchInfo.PRNumber)
	}
	splog.Info("    ↓")
	splog.Info("  📦 Consolidated PR")
	splog.Newline()

	consolidationBranch, err := c.createMergeBranch(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create consolidation branch: %w", err)
	}

	pr, err := c.createConsolidationPR(ctx, consolidationBranch)
	if err != nil {
		return nil, fmt.Errorf("failed to create consolidation PR: %w", err)
	}

	splog.Info("✅ Created consolidation branch: %s", consolidationBranch)

	// Lock individual PRs and update them with a notice
	if err := c.lockAndNotifyIndividualPRs(ctx, consolidationBranch); err != nil {
		splog.Warn("Failed to lock and notify individual PRs: %v", err)
	}

	if opts.Wait {
		if err := c.waitForConsolidationMerge(ctx, consolidationBranch, pr); err != nil {
			return nil, fmt.Errorf("consolidation merge failed: %w", err)
		}

		// Store consolidation info for footer updates
		c.consolidationPR = pr
		if userName, err := c.ctx.Git().GetUserName(ctx); err == nil && userName != "" {
			c.consolidationUser = userName
		}

		c.postMergeCleanup(ctx)

		splog.Info("🎉 Stack consolidation merge completed successfully!")
	} else {
		splog.Info("🎉 Consolidation PR created: %s", pr.HTMLURL)
		splog.Info("   Individual PRs have been locked. Merge the consolidation PR manually to complete.")
	}

	result := &ConsolidationResult{
		BranchName: consolidationBranch,
		PRNumber:   pr.Number,
		PRURL:      pr.HTMLURL,
	}
	return result, nil
}

// preValidateStack ensures all PRs are ready for consolidation
func (c *ConsolidateMergeExecutor) preValidateStack(ctx context.Context, force bool) error {
	splog := c.ctx.Output

	// We trust the plan created by CreateMergePlan for PR status and remote matching
	// unless --force is used to bypass those checks entirely.
	// We only re-validate if we're not forced and want to be absolutely sure.
	if !force {
		for _, branchInfo := range c.plan.BranchesToMerge {
			// Quick local checks
			if !branchInfo.MatchesRemote {
				return fmt.Errorf("branch %s differs from remote, use --force to proceed", branchInfo.BranchName)
			}
			splog.Debug("✅ Branch %s is ready for consolidation (from plan)", branchInfo.BranchName)
		}
	}

	// This is the only "heavy" operation that really needs to happen here
	pullResult, err := c.engine.PullTrunk(ctx)
	if err != nil {
		return fmt.Errorf("failed to update trunk: %w", err)
	}
	if pullResult == engine.PullConflict {
		return fmt.Errorf("trunk has conflicts with remote")
	}

	return nil
}

// createMergeBranch creates a branch containing all stack commits
func (c *ConsolidateMergeExecutor) createMergeBranch(ctx context.Context) (string, error) {
	splog := c.ctx.Output
	// Generate unique branch name
	timestamp := time.Now().Unix()
	scope := c.getStackScope()
	branchName := fmt.Sprintf("stack-merge-%s-%d", scope, timestamp)

	splog.Info("📋 Creating merge branch: %s", branchName)

	if err := c.engine.CreateAndCheckoutBranch(ctx, c.engine.GetBranch(branchName)); err != nil {
		return "", fmt.Errorf("failed to create and checkout branch: %w", err)
	}

	// Reset to trunk since CreateAndCheckoutBranch creates from current HEAD
	if err := c.engine.ResetHard(ctx, c.engine.Trunk().GetName()); err != nil {
		return "", fmt.Errorf("failed to reset to trunk: %w", err)
	}

	// Collect branch names for octopus merge
	branches := make([]string, len(c.plan.BranchesToMerge))
	for i, branchInfo := range c.plan.BranchesToMerge {
		branches[i] = branchInfo.BranchName
	}

	// Build commit message listing all branches
	commitMsg := fmt.Sprintf("Consolidate stack [%s]", scope)
	if len(branches) <= 3 {
		commitMsg += ": " + strings.Join(branches, ", ")
	} else {
		commitMsg += fmt.Sprintf(" (%d branches)", len(branches))
	}

	splog.Info("  Merging %d branches via octopus merge...", len(branches))
	splog.Debug("Consolidation merge: merging branches %v", branches)

	// Perform octopus merge (single merge commit with multiple parents)
	if err := c.engine.MergeMultiple(ctx, branches, engine.MergeOptions{NoFF: true, Message: commitMsg}); err != nil {
		splog.Debug("Consolidation octopus merge failed: %v", err)
		return "", fmt.Errorf("failed to merge branches: %w", err)
	}

	if err := c.engine.PushBranch(ctx, c.engine.GetBranch(branchName), c.engine.GetRemote(), git.PushOptions{
		Force:    false,
		NoVerify: true,
	}); err != nil {
		return "", fmt.Errorf("failed to push consolidation branch %s: %w", branchName, err)
	}

	splog.Info("✅ Merge branch created and pushed")
	return branchName, nil
}

func (c *ConsolidateMergeExecutor) createConsolidationPR(ctx context.Context, branchName string) (*github.PullRequestInfo, error) {
	content := c.prGenerator.GenerateConsolidationPR(c.plan.BranchesToMerge)

	owner, repo := c.getOwnerRepo()
	opts := github.CreatePROptions{
		Title: content.Title,
		Body:  content.Body,
		Head:  branchName,
		Base:  c.engine.Trunk().GetName(),
		Draft: false,
	}

	pr, err := c.ctx.GitHubClient.CreatePullRequest(ctx, owner, repo, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create PR: %w", err)
	}

	return pr, nil
}

// waitForConsolidationMerge waits for CI to pass and auto-merges the consolidation PR
func (c *ConsolidateMergeExecutor) waitForConsolidationMerge(ctx context.Context, branchName string, pr *github.PullRequestInfo) error {
	expectChecks := AnyPRHasChecks(c.plan.BranchesToMerge)

	// Get merge method (prompts user if not configured)
	mergeMethod, err := GetMergeMethod(c.ctx, c.ctx.GitHubClient)
	if err != nil {
		return fmt.Errorf("failed to get merge method: %w", err)
	}

	waiter := NewCIWaiter(CIWaiterOptions{
		Client: c.ctx.GitHubClient,
		Output: c.ctx.Output,
	})
	waiter.SetProgressHandler(c.handler, c.stepIndex)

	return waiter.WaitAndMerge(ctx, branchName, pr, expectChecks, mergeMethod)
}

func (c *ConsolidateMergeExecutor) postMergeCleanup(_ context.Context) {
	c.ctx.Output.Info("🧹 Updating individual PRs...")

	c.updateIndividualPRs()
}

func (c *ConsolidateMergeExecutor) updateIndividualPRs() {
	// Skip if we don't have consolidation PR info
	if c.consolidationPR == nil || c.consolidationPR.Number == 0 {
		c.ctx.Output.Debug("No consolidation PR info available for footer updates")
		return
	}

	// Collect branch names
	branchNames := make([]string, len(c.plan.BranchesToMerge))
	for i, b := range c.plan.BranchesToMerge {
		branchNames[i] = b.BranchName
	}

	// Use shared PR cleaner
	cleaner := NewPRCleaner(c.ctx, c.engine, PRCleanupConfig{
		Source:                CleanupSourceConsolidate,
		ConsolidationPRNumber: c.consolidationPR.Number,
		UserName:              c.consolidationUser,
	})

	result := cleaner.CleanupBranches(c.ctx.Context, branchNames)
	cleaner.LogResult(result)
}

func (c *ConsolidateMergeExecutor) lockAndNotifyIndividualPRs(_ context.Context, consolidationBranch string) error {
	splog := c.ctx.Output
	splog.Info("🔒 Locking individual PRs and updating status...")

	branchesToLock := []engine.Branch{}
	branchNames := []string{}
	for _, b := range c.plan.BranchesToMerge {
		branch := c.engine.GetBranch(b.BranchName)
		if !branch.IsLocked() {
			branchesToLock = append(branchesToLock, branch)
		}
		branchNames = append(branchNames, b.BranchName)
	}

	if len(branchesToLock) > 0 {
		if _, err := c.engine.SetLocked(c.ctx, branchesToLock, engine.LockReasonConsolidating); err != nil {
			return fmt.Errorf("failed to lock branches: %w", err)
		}
	}

	for _, b := range branchesToLock {
		prInfo, _ := b.GetPrInfo()
		if prInfo != nil {
			if err := c.engine.UpsertPrInfo(b, prInfo.WithMergeBranch(consolidationBranch)); err != nil {
				splog.Debug("Failed to upsert PR info for %s: %v", b.GetName(), err)
			}
		}
	}

	if err := actions.PushMetadataAndSyncPRs(c.ctx, branchNames); err != nil {
		splog.Warn("Failed to sync individual PRs: %v", err)
	}

	return nil
}

// Helper methods

func (c *ConsolidateMergeExecutor) getStackScope() string {
	// Get scope from the first branch in the stack
	if len(c.plan.BranchesToMerge) > 0 {
		branch := c.engine.GetBranch(c.plan.BranchesToMerge[0].BranchName)
		scope := c.engine.GetScope(branch)
		if !scope.IsEmpty() {
			return scope.String()
		}
	}
	return "stack"
}

func (c *ConsolidateMergeExecutor) getOwnerRepo() (owner, repo string) {
	return c.ctx.GitHubClient.GetOwnerRepo()
}
