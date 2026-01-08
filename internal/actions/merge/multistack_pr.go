package merge

import (
	"context"
	"fmt"
	"time"

	"stackit.dev/stackit/internal/app"
	"stackit.dev/stackit/internal/config"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/github"
)

// MultiStackPRCreator handles creating the multi-stack PR
type MultiStackPRCreator struct {
	ctx          *app.Context
	worktreeEng  engine.Engine
	worktreePath string
	prGenerator  *PRContentGenerator
}

// NewMultiStackPRCreator creates a new PR creator for multi-stack merge
func NewMultiStackPRCreator(ctx *app.Context, worktreeEng engine.Engine, worktreePath string) *MultiStackPRCreator {
	return &MultiStackPRCreator{
		ctx:          ctx,
		worktreeEng:  worktreeEng,
		worktreePath: worktreePath,
		prGenerator:  NewPRContentGenerator(worktreeEng),
	}
}

// GenerateMultiStackBranchName creates a unique branch name for the multi-stack PR
func GenerateMultiStackBranchName() string {
	return fmt.Sprintf("multi-stack/%s", time.Now().Format("20060102-150405"))
}

// CreateAndPushBranch creates a named branch at the current HEAD and pushes it
func (p *MultiStackPRCreator) CreateAndPushBranch(ctx context.Context, branchName string) error {
	// Create a branch at current HEAD
	if err := p.worktreeEng.CreateBranch(ctx, branchName, "HEAD"); err != nil {
		return fmt.Errorf("failed to create branch %s: %w", branchName, err)
	}

	// Checkout the new branch
	branch := p.worktreeEng.GetBranch(branchName)
	if err := p.worktreeEng.CheckoutBranch(ctx, branch); err != nil {
		return fmt.Errorf("failed to checkout branch %s: %w", branchName, err)
	}

	// Push to remote
	remote := p.worktreeEng.GetRemote()
	if err := p.worktreeEng.PushBranch(ctx, branch, remote, git.PushOptions{
		Force:    false,
		NoVerify: true,
	}); err != nil {
		return fmt.Errorf("failed to push branch %s: %w", branchName, err)
	}

	return nil
}

// CreatePR creates the multi-stack pull request
func (p *MultiStackPRCreator) CreatePR(ctx context.Context, branchName string, included []MultiStackInfo, excluded []MultiStackExcluded) (*github.PullRequestInfo, error) {
	if p.ctx.GitHubClient == nil {
		return nil, fmt.Errorf("GitHub client not available")
	}

	owner, repo := p.ctx.GitHubClient.GetOwnerRepo()
	if owner == "" || repo == "" {
		return nil, fmt.Errorf("could not determine repository owner/name")
	}

	scope := p.getScopeFromStacks(included)
	content := p.prGenerator.GenerateMultiStackPR(included, excluded, scope)

	opts := github.CreatePROptions{
		Title: content.Title,
		Body:  content.Body,
		Head:  branchName,
		Base:  p.worktreeEng.Trunk().GetName(),
		Draft: false,
	}

	pr, err := p.ctx.GitHubClient.CreatePullRequest(ctx, owner, repo, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create PR: %w", err)
	}

	return pr, nil
}

// getScopeFromStacks extracts the common scope from included stacks
func (p *MultiStackPRCreator) getScopeFromStacks(included []MultiStackInfo) string {
	if len(included) == 0 {
		return "multi-stack"
	}

	// Get scope from the first stack
	scope := included[0].Scope
	if scope != "" {
		return scope
	}

	// Fallback to "multi-stack" if no scope found
	return "multi-stack"
}

// WaitAndMerge waits for CI to pass and auto-merges the PR
func (p *MultiStackPRCreator) WaitAndMerge(ctx context.Context, branchName string, pr *github.PullRequestInfo) error {
	if p.ctx.GitHubClient == nil {
		return fmt.Errorf("GitHub client not available for wait")
	}

	// Load config for merge method
	cfg, err := config.LoadConfig(p.ctx.RepoRoot)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	mergeMethod := github.MergeMethodSquash // default
	if method := cfg.MergeMethod(); method != "" {
		switch method {
		case "merge":
			mergeMethod = github.MergeMethodMerge
		case "rebase":
			mergeMethod = github.MergeMethodRebase
		}
	}

	waiter := NewCIWaiter(CIWaiterOptions{
		Client: p.ctx.GitHubClient,
		Output: p.ctx.Output,
	})

	return waiter.WaitAndMerge(ctx, branchName, pr, true, mergeMethod)
}
