package handlers

import (
	"net/http"
	"strings"

	httpcontract "stackit.dev/stackit/internal/contracts/http"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/github"
)

// BranchesHandler serves branch data.
type BranchesHandler struct {
	eng engine.BranchReader
	gh  github.Client
}

// NewBranchesHandler creates a handler for /api/branches and /api/v1/branches.
func NewBranchesHandler(eng engine.BranchReader, gh github.Client) *BranchesHandler {
	return &BranchesHandler{eng: eng, gh: gh}
}

// ServeHTTP handles GET branches endpoints.
func (h *BranchesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	branchName, hasBranch := parseResourcePath(r.URL.Path, "branches")
	if !hasBranch {
		http.NotFound(w, r)
		return
	}

	switch {
	case branchName == "diff":
		queryBranch := r.URL.Query().Get("branch")
		if queryBranch == "" {
			http.Error(w, "missing branch query parameter", http.StatusBadRequest)
			return
		}
		h.getBranchDiff(w, r, queryBranch)
	case branchName == "":
		h.listBranches(w, r)
	case strings.HasSuffix(branchName, "/diff"):
		h.getBranchDiff(w, r, strings.TrimSuffix(branchName, "/diff"))
	default:
		h.getBranch(w, r, branchName)
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

	responses := make([]httpcontract.BranchResponse, 0, len(branches))
	for _, branch := range branches {
		node := graph.GetNode(branch.GetName())
		if node == nil {
			continue
		}
		var checks *github.CheckStatus
		if checksMap != nil {
			checks = checksMap[branch.GetName()]
		}
		responses = append(responses, httpcontract.MapBranch(h.eng, branch, node, checks))
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

	resp := httpcontract.MapBranch(h.eng, branch, node, checks)
	writeJSON(w, resp)
}

func (h *BranchesHandler) getBranchDiff(w http.ResponseWriter, r *http.Request, branchName string) {
	if branchName == "" {
		http.Error(w, "branch not found or not tracked", http.StatusNotFound)
		return
	}

	branch := h.eng.GetBranch(branchName)
	if !branch.IsTracked() {
		http.Error(w, "branch not found or not tracked", http.StatusNotFound)
		return
	}

	baseRevision, err := h.eng.GetDivergencePoint(branchName)
	if err != nil {
		http.Error(w, "failed to resolve branch base: "+err.Error(), http.StatusInternalServerError)
		return
	}

	headRevision, err := branch.GetRevision()
	if err != nil {
		http.Error(w, "failed to resolve branch revision: "+err.Error(), http.StatusInternalServerError)
		return
	}

	patch, err := h.eng.GetDiffBetween(r.Context(), baseRevision, headRevision)
	if err != nil {
		http.Error(w, "failed to compute branch diff: "+err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, httpcontract.BranchDiffResponse{
		Branch:       branchName,
		BaseRevision: baseRevision,
		HeadRevision: headRevision,
		Patch:        patch,
	})
}
