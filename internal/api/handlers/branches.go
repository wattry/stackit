package handlers

import (
	"net/http"
	"strings"

	"stackit.dev/stackit/internal/api/types"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/github"
)

// BranchesHandler serves branch data.
type BranchesHandler struct {
	eng engine.BranchReader
	gh  github.Client
}

// NewBranchesHandler creates a handler for /api/branches.
func NewBranchesHandler(eng engine.BranchReader, gh github.Client) *BranchesHandler {
	return &BranchesHandler{eng: eng, gh: gh}
}

// ServeHTTP handles GET /api/branches and GET /api/branches/{name}.
func (h *BranchesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse path: /api/branches or /api/branches/{name}
	path := strings.TrimPrefix(r.URL.Path, "/api/branches")
	path = strings.TrimPrefix(path, "/")

	if path == "" {
		h.listBranches(w, r)
	} else {
		h.getBranch(w, r, path)
	}
}

func (h *BranchesHandler) listBranches(w http.ResponseWriter, r *http.Request) {
	graph := engine.BuildStackGraph(h.eng, engine.SortStrategyAlphabetical, nil)
	allBranches := h.eng.AllBranches()

	// Filter to only tracked (non-trunk) branches
	branches := make([]engine.Branch, 0, len(allBranches))
	for _, b := range allBranches {
		if b.IsTracked() {
			branches = append(branches, b)
		}
	}

	// Optionally fetch CI checks
	var checksMap map[string]*github.CheckStatus
	if h.gh != nil {
		names := make([]string, len(branches))
		for i, b := range branches {
			names[i] = b.GetName()
		}
		checksMap, _ = h.gh.BatchGetPRChecksStatus(r.Context(), names)
	}

	responses := make([]types.BranchResponse, 0, len(branches))
	for _, branch := range branches {
		node := graph.GetNode(branch.GetName())
		if node == nil {
			continue
		}
		var checks *github.CheckStatus
		if checksMap != nil {
			checks = checksMap[branch.GetName()]
		}
		responses = append(responses, types.MapBranch(h.eng, branch, node, checks))
	}

	writeJSON(w, responses)
}

func (h *BranchesHandler) getBranch(w http.ResponseWriter, r *http.Request, branchName string) {
	branch := h.eng.GetBranch(branchName)
	if !branch.IsTracked() {
		http.Error(w, "branch not found or not tracked", http.StatusNotFound)
		return
	}

	graph := engine.BuildStackGraph(h.eng, engine.SortStrategyAlphabetical, nil)
	node := graph.GetNode(branchName)
	if node == nil {
		http.Error(w, "branch not in stack graph", http.StatusNotFound)
		return
	}

	var checks *github.CheckStatus
	if h.gh != nil {
		checksMap, _ := h.gh.BatchGetPRChecksStatus(r.Context(), []string{branchName})
		if checksMap != nil {
			checks = checksMap[branchName]
		}
	}

	resp := types.MapBranch(h.eng, branch, node, checks)
	writeJSON(w, resp)
}
