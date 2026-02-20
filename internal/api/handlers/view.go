package handlers

import (
	"net/http"

	"stackit.dev/stackit/internal/actions/merge"
	"stackit.dev/stackit/internal/api/types"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/github"
)

// ViewHandler serves the combined view payload for the frontend.
type ViewHandler struct {
	eng    engine.BranchReader
	gh     github.Client
	remote string
}

// NewViewHandler creates a handler for /api/view.
func NewViewHandler(eng engine.BranchReader, gh github.Client, remote string) *ViewHandler {
	return &ViewHandler{eng: eng, gh: gh, remote: remote}
}

// ServeHTTP handles GET /api/view.
func (h *ViewHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Build repo info
	owner, repo := "", ""
	if h.gh != nil {
		owner, repo = h.gh.GetOwnerRepo()
	}
	repoResp := types.RepoResponse{
		Owner:         owner,
		Repo:          repo,
		Trunk:         h.eng.Trunk().GetName(),
		CurrentBranch: h.eng.CurrentBranch().GetName(),
		Remote:        h.remote,
	}

	// Discover all stacks
	stacks, err := merge.DiscoverStacksWithSort(h.eng, engine.SortStrategySmart)
	if err != nil {
		http.Error(w, "failed to discover stacks: "+err.Error(), http.StatusInternalServerError)
		return
	}

	graph := engine.BuildStackGraph(h.eng, engine.SortStrategySmart, nil)

	// Collect all branch names across all stacks for a single batched CI check
	var allBranches []string
	for _, stack := range stacks {
		allBranches = append(allBranches, stack.AllBranches...)
	}

	var checksMap map[string]*github.CheckStatus
	if h.gh != nil && len(allBranches) > 0 {
		checksMap, _ = h.gh.BatchGetPRChecksStatus(r.Context(), allBranches)
	}

	// Map all stack details using the shared checks map
	details := make([]types.StackDetail, 0, len(stacks))
	for _, stack := range stacks {
		detail := types.MapStackDetail(h.eng, graph, stack.RootBranch, stack.AllBranches, stack.PRCount, stack.Scope, checksMap)
		details = append(details, detail)
	}

	writeJSON(w, types.ViewResponse{
		Repo:   repoResp,
		Stacks: details,
	})
}
