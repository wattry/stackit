package handlers

import (
	"net/http"
	"strings"

	"stackit.dev/stackit/internal/actions/merge"
	"stackit.dev/stackit/internal/api/types"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/github"
)

// StacksHandler serves stack data.
type StacksHandler struct {
	eng engine.BranchReader
	gh  github.Client
}

// NewStacksHandler creates a handler for /api/stacks.
func NewStacksHandler(eng engine.BranchReader, gh github.Client) *StacksHandler {
	return &StacksHandler{eng: eng, gh: gh}
}

// ServeHTTP handles GET /api/stacks and GET /api/stacks/{root}.
func (h *StacksHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse path: /api/stacks or /api/stacks/{root}
	path := strings.TrimPrefix(r.URL.Path, "/api/stacks")
	path = strings.TrimPrefix(path, "/")

	if path == "" {
		h.listStacks(w)
	} else {
		h.getStack(w, r, path)
	}
}

func (h *StacksHandler) listStacks(w http.ResponseWriter) {
	stacks, err := merge.DiscoverStacks(h.eng)
	if err != nil {
		http.Error(w, "failed to discover stacks: "+err.Error(), http.StatusInternalServerError)
		return
	}

	graph := engine.BuildStackGraph(h.eng, engine.SortStrategyAlphabetical, nil)

	summaries := make([]types.StackSummary, 0, len(stacks))
	for _, stack := range stacks {
		summary := types.MapStackSummary(h.eng, graph, stack.RootBranch, stack.AllBranches, stack.PRCount, stack.Scope)
		summaries = append(summaries, summary)
	}

	writeJSON(w, summaries)
}

func (h *StacksHandler) getStack(w http.ResponseWriter, r *http.Request, rootBranch string) {
	stacks, err := merge.DiscoverStacks(h.eng)
	if err != nil {
		http.Error(w, "failed to discover stacks: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Find the requested stack
	var found *merge.MultiStackInfo
	for i := range stacks {
		if stacks[i].RootBranch == rootBranch {
			found = &stacks[i]
			break
		}
	}
	if found == nil {
		http.Error(w, "stack not found", http.StatusNotFound)
		return
	}

	graph := engine.BuildStackGraph(h.eng, engine.SortStrategyAlphabetical, nil)

	// Fetch CI checks if GitHub client is available
	var checksMap map[string]*github.CheckStatus
	if h.gh != nil {
		checksMap, _ = h.gh.BatchGetPRChecksStatus(r.Context(), found.AllBranches)
	}

	detail := types.MapStackDetail(h.eng, graph, found.RootBranch, found.AllBranches, found.PRCount, found.Scope, checksMap)
	writeJSON(w, detail)
}
