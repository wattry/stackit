package testhelpers

import (
	"context"

	"github.com/google/go-github/v62/github"

	githubpkg "stackit.dev/stackit/internal/github"
)

// MockGitHubClient implements githubpkg.Client using the mock server
type MockGitHubClient struct {
	client *github.Client
	owner  string
	repo   string
	config *MockGitHubServerConfig
}

// NewMockGitHubClientInterface creates a GitHubClient interface implementation
// using the mock server
func NewMockGitHubClientInterface(client *github.Client, owner, repo string, config *MockGitHubServerConfig) githubpkg.Client {
	return &MockGitHubClient{
		client: client,
		owner:  owner,
		repo:   repo,
		config: config,
	}
}

// GetOwnerRepo returns the repository owner and name
func (c *MockGitHubClient) GetOwnerRepo() (string, string) {
	return c.owner, c.repo
}

// CreatePullRequest creates a new pull request
func (c *MockGitHubClient) CreatePullRequest(ctx context.Context, owner, repo string, opts githubpkg.CreatePROptions) (*githubpkg.PullRequestInfo, error) {
	pr := &github.NewPullRequest{
		Title: github.String(opts.Title),
		Head:  github.String(opts.Head),
		Base:  github.String(opts.Base),
		Draft: github.Bool(opts.Draft),
	}

	if opts.Body != "" {
		pr.Body = github.String(opts.Body)
	}

	createdPR, _, err := c.client.PullRequests.Create(ctx, owner, repo, pr)
	if err != nil {
		return nil, err
	}

	return githubpkg.ToPullRequestInfo(createdPR), nil
}

// UpdatePullRequest updates an existing pull request
func (c *MockGitHubClient) UpdatePullRequest(ctx context.Context, owner, repo string, prNumber int, opts githubpkg.UpdatePROptions) error {
	update := &github.PullRequest{}

	if opts.Title != nil {
		update.Title = opts.Title
	}
	if opts.Body != nil {
		update.Body = opts.Body
	}
	if opts.Base != nil {
		update.Base = &github.PullRequestBranch{
			Ref: opts.Base,
		}
	}

	_, _, err := c.client.PullRequests.Edit(ctx, owner, repo, prNumber, update)
	return err
}

// GetPullRequestByBranch gets a pull request for a branch
func (c *MockGitHubClient) GetPullRequestByBranch(ctx context.Context, owner, repo, branchName string) (*githubpkg.PullRequestInfo, error) {
	prs, _, err := c.client.PullRequests.List(ctx, owner, repo, &github.PullRequestListOptions{
		Head:  owner + ":" + branchName,
		State: "all",
		ListOptions: github.ListOptions{
			PerPage: 1,
		},
	})
	if err != nil {
		return nil, err
	}

	if len(prs) == 0 {
		return nil, nil
	}

	return githubpkg.ToPullRequestInfo(prs[0]), nil
}

// GetPullRequest gets a pull request by number
func (c *MockGitHubClient) GetPullRequest(ctx context.Context, owner, repo string, prNumber int) (*githubpkg.PullRequestInfo, error) {
	pr, _, err := c.client.PullRequests.Get(ctx, owner, repo, prNumber)
	if err != nil {
		return nil, err
	}

	return githubpkg.ToPullRequestInfo(pr), nil
}

// MergePullRequest merges a pull request
func (c *MockGitHubClient) MergePullRequest(_ context.Context, _ string) error {
	// In tests, just return nil
	return nil
}

// GetPRChecksStatus returns the check status for a PR
func (c *MockGitHubClient) GetPRChecksStatus(_ context.Context, _ string) (*githubpkg.CheckStatus, error) {
	// In tests, always return passing
	return &githubpkg.CheckStatus{
		Passing: true,
		Pending: false,
		Checks: []githubpkg.CheckDetail{
			{Name: "Mock Check", Status: "COMPLETED", Conclusion: "SUCCESS"},
		},
	}, nil
}

// BatchGetPRChecksStatus returns the check status for multiple branches
func (c *MockGitHubClient) BatchGetPRChecksStatus(ctx context.Context, branchNames []string) (map[string]*githubpkg.CheckStatus, error) {
	results := make(map[string]*githubpkg.CheckStatus)
	for _, name := range branchNames {
		status, err := c.GetPRChecksStatus(ctx, name)
		if err != nil {
			return nil, err
		}
		results[name] = status
	}
	return results, nil
}
