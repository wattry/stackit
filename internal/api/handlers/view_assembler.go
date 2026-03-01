package handlers

import (
	"context"
	"fmt"

	"stackit.dev/stackit/internal/actions/merge"
	httpcontract "stackit.dev/stackit/internal/contracts/http"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/github"
)

// ViewAssembler builds the combined /view payload.
type ViewAssembler struct {
	eng    engine.BranchReader
	gh     github.Client
	remote string
}

func NewViewAssembler(eng engine.BranchReader, gh github.Client, remote string) *ViewAssembler {
	return &ViewAssembler{
		eng:    eng,
		gh:     gh,
		remote: remote,
	}
}

func (a *ViewAssembler) Build(ctx context.Context) (httpcontract.ViewResponse, error) {
	stacks, err := merge.DiscoverStacksWithSort(a.eng, engine.SortStrategySmart)
	if err != nil {
		return httpcontract.ViewResponse{}, fmt.Errorf("failed to discover stacks: %w", err)
	}

	graph := engine.BuildStackGraph(a.eng, engine.SortStrategySmart, nil)
	checksMap := a.fetchChecks(ctx, stacks)
	details := a.mapStackDetails(graph, stacks, checksMap)

	recentlyMerged := a.fetchRecentlyMerged()

	return httpcontract.ViewResponse{
		Repo:           a.buildRepo(ctx),
		Stacks:         details,
		RecentlyMerged: recentlyMerged,
	}, nil
}

func (a *ViewAssembler) buildRepo(ctx context.Context) httpcontract.RepoResponse {
	owner, repo := "", ""
	var currentUser string
	if a.gh != nil {
		owner, repo = a.gh.GetOwnerRepo()
		currentUser, _ = a.gh.GetCurrentUser(ctx)
	}

	return httpcontract.RepoResponse{
		Owner:         owner,
		Repo:          repo,
		Trunk:         a.eng.Trunk().GetName(),
		CurrentBranch: a.eng.CurrentBranch().GetName(),
		Remote:        a.remote,
		CurrentUser:   currentUser,
	}
}

func (a *ViewAssembler) fetchChecks(ctx context.Context, stacks []merge.MultiStackInfo) map[string]*github.CheckStatus {
	if a.gh == nil {
		return nil
	}

	var allBranches []string
	for _, stack := range stacks {
		allBranches = append(allBranches, stack.AllBranches...)
	}
	if len(allBranches) == 0 {
		return nil
	}

	checksMap, _ := a.gh.BatchGetPRChecksStatus(ctx, allBranches)
	return checksMap
}

func (a *ViewAssembler) mapStackDetails(
	graph *engine.StackGraph,
	stacks []merge.MultiStackInfo,
	checksMap map[string]*github.CheckStatus,
) []httpcontract.StackDetail {
	details := make([]httpcontract.StackDetail, 0, len(stacks))
	for _, stack := range stacks {
		detail := httpcontract.MapStackDetail(a.eng, graph, stack.RootBranch, stack.AllBranches, stack.PRCount, stack.Scope, checksMap)
		details = append(details, detail)
	}
	return details
}

func (a *ViewAssembler) fetchRecentlyMerged() []httpcontract.TrunkCommitResponse {
	recentCommits, err := a.eng.GetRecentTrunkCommits(10)
	if err != nil || len(recentCommits) == 0 {
		return nil
	}
	return httpcontract.MapTrunkCommits(recentCommits)
}
