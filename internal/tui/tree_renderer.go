package tui

import (
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/github"
	"stackit.dev/stackit/internal/tui/components/tree"
)

// NewStackTreeRenderer creates a tree renderer configured for the current engine state
// using the SMART sorting strategy (active path hoisting + newest first).
func NewStackTreeRenderer(eng engine.BranchReader) *tree.StackTreeRenderer {
	return NewStackTreeRendererWithStrategy(eng, engine.SortStrategySmart, nil)
}

// NewStackTreeRendererWithFilter creates a tree renderer with a filter function
func NewStackTreeRendererWithFilter(eng engine.BranchReader, filter func(string) bool) *tree.StackTreeRenderer {
	return NewStackTreeRendererWithStrategy(eng, engine.SortStrategySmart, filter)
}

// NewStackTreeRendererWithStrategy creates a tree renderer with a specific sorting strategy and optional filter
func NewStackTreeRendererWithStrategy(eng engine.BranchReader, strategy engine.SortStrategy, filter func(string) bool) *tree.StackTreeRenderer {
	return newStackTreeRendererInternal(eng, strategy, filter, nil)
}

// NewStackTreeRendererWithEmptyWorktrees creates a tree renderer that shows empty worktree anchors
func NewStackTreeRendererWithEmptyWorktrees(eng engine.BranchReader, emptyWorktrees map[string]bool) *tree.StackTreeRenderer {
	return newStackTreeRendererInternal(eng, engine.SortStrategySmart, nil, emptyWorktrees)
}

// NewStackTreeRendererWithOptions creates a tree renderer with all options
func NewStackTreeRendererWithOptions(eng engine.BranchReader, strategy engine.SortStrategy, filter func(string) bool, emptyWorktrees map[string]bool) *tree.StackTreeRenderer {
	return newStackTreeRendererInternal(eng, strategy, filter, emptyWorktrees)
}

// graphData implements tree.Data using an engine.StackGraph.
// This decouples the tree renderer from the engine package.
type graphData struct {
	graph *engine.StackGraph
	eng   engine.BranchReader
}

// CurrentBranch returns the currently checked out branch.
func (d *graphData) CurrentBranch() string {
	return d.graph.CurrentBranch()
}

// Trunk returns the trunk/main branch name.
func (d *graphData) Trunk() string {
	return d.graph.TrunkName()
}

// Children returns the child branches of the given branch.
func (d *graphData) Children(branchName string) []string {
	node := d.graph.GetNode(branchName)
	if node == nil {
		return nil
	}
	return d.graph.Children(node.Branch)
}

// Parent returns the parent branch of the given branch.
func (d *graphData) Parent(branchName string) string {
	node := d.graph.GetNode(branchName)
	if node == nil {
		return ""
	}
	return d.graph.Parent(node.Branch)
}

// IsTrunk returns whether the given branch is the trunk branch.
func (d *graphData) IsTrunk(branchName string) bool {
	return branchName == d.graph.TrunkName()
}

// IsFixed returns whether the branch is up-to-date with its parent.
func (d *graphData) IsFixed(branchName string) bool {
	node := d.graph.GetNode(branchName)
	if node == nil {
		return false
	}
	return d.eng.IsUpToDate(node.Branch)
}

// newStackTreeRendererInternal is the internal implementation that handles all renderer options
func newStackTreeRendererInternal(eng engine.BranchReader, strategy engine.SortStrategy, filter func(string) bool, emptyWorktrees map[string]bool) *tree.StackTreeRenderer {
	branchFilter := func(b engine.Branch) bool {
		if b.IsWorktreeAnchor() {
			// Show empty worktree anchors if they're in the set
			if emptyWorktrees != nil {
				return emptyWorktrees[b.GetName()]
			}
			return false
		}
		// Apply user-provided filter if any
		if filter != nil {
			return filter(b.GetName())
		}
		return true
	}

	graph := engine.BuildStackGraph(eng, strategy, branchFilter)

	// Use the Data interface instead of callback functions
	data := &graphData{
		graph: graph,
		eng:   eng,
	}

	return tree.NewRenderer(data)
}

// WorktreeData holds pre-computed worktree information from a single ListManagedWorktrees call.
type WorktreeData struct {
	EmptyWorktrees      map[string]*engine.WorktreeInfo // Anchor branches with no children
	WorktreeByStackRoot map[string]*engine.WorktreeInfo // Stack root -> worktree info
}

// GetWorktreeData builds both empty worktree and stack-root worktree maps from a single
// ListManagedWorktrees call, avoiding redundant lookups.
func GetWorktreeData(eng engine.Engine) *WorktreeData {
	data := &WorktreeData{
		EmptyWorktrees:      make(map[string]*engine.WorktreeInfo),
		WorktreeByStackRoot: make(map[string]*engine.WorktreeInfo),
	}

	worktrees, err := eng.ListManagedWorktrees()
	if err != nil {
		return data
	}

	for i := range worktrees {
		wt := &worktrees[i]
		anchor := eng.GetBranch(wt.AnchorBranch)

		// Build stack-root map for all worktrees (used by addWorktreeInfo)
		data.WorktreeByStackRoot[wt.AnchorBranch] = wt

		if !anchor.IsTracked() || !anchor.IsWorktreeAnchor() {
			continue
		}

		// Check if anchor has any children
		hasChildren := false
		for _, depth := range eng.BranchesDepthFirst(anchor) {
			if depth > 0 {
				hasChildren = true
				break
			}
		}

		if !hasChildren {
			data.EmptyWorktrees[wt.AnchorBranch] = wt
		}
	}

	return data
}

// GetEmptyWorktrees returns a map of worktree anchor branch names to their WorktreeInfo
// for worktrees that have no child branches (empty worktrees).
func GetEmptyWorktrees(eng engine.Engine) map[string]*engine.WorktreeInfo {
	return GetWorktreeData(eng).EmptyWorktrees
}

// GetMinimalAnnotationWithWorktreeAndEmpty returns minimal annotations plus worktree info,
// with support for marking empty worktrees.
// This is used for fast initial rendering before full data is loaded.
// Only includes cached/instant fields - no git or network calls.
func GetMinimalAnnotationWithWorktreeAndEmpty(eng engine.Engine, branch engine.Branch, wtData *WorktreeData) tree.BranchAnnotation {
	ann := tree.BranchAnnotation{
		IsLocked:      branch.IsLocked(),
		IsFrozen:      branch.IsFrozen(),
		Scope:         eng.GetScope(branch).String(),
		ExplicitScope: branch.GetExplicitScope().String(),
	}

	var emptyWorktrees map[string]*engine.WorktreeInfo
	var worktreeByRoot map[string]*engine.WorktreeInfo
	if wtData != nil {
		emptyWorktrees = wtData.EmptyWorktrees
		worktreeByRoot = wtData.WorktreeByStackRoot
	}
	addWorktreeInfo(eng, branch, &ann, emptyWorktrees, worktreeByRoot)

	return ann
}

// addWorktreeInfo populates worktree-related fields on a BranchAnnotation.
// If the branch is an empty worktree anchor, it sets IsEmptyWorktree and WorktreePath.
// Otherwise, if the branch is a stack root with a managed worktree, it sets WorktreePath.
// When worktreeByRoot is provided, uses O(1) map lookup instead of calling GetWorktreeForStack.
func addWorktreeInfo(eng engine.Engine, branch engine.Branch, ann *tree.BranchAnnotation, emptyWorktrees map[string]*engine.WorktreeInfo, worktreeByRoot map[string]*engine.WorktreeInfo) {
	if emptyWorktrees != nil {
		if wtInfo, ok := emptyWorktrees[branch.GetName()]; ok {
			ann.IsEmptyWorktree = true
			ann.WorktreePath = wtInfo.Path
			return
		}
	}

	stackRoot := eng.GetStackRootForBranch(branch)
	if stackRoot == branch.GetName() {
		// Use pre-built map if available, otherwise fall back to engine call
		if worktreeByRoot != nil {
			if wtInfo, ok := worktreeByRoot[stackRoot]; ok {
				ann.WorktreePath = wtInfo.Path
			}
		} else if wtInfo, err := eng.GetWorktreeForStack(stackRoot); err == nil && wtInfo != nil {
			ann.WorktreePath = wtInfo.Path
		}
	}
}

// AnnotationEnrichment holds pre-fetched data for enriching branch annotations.
// This allows CI statuses and worktree info to be computed once and shared
// across all branches, avoiding redundant lookups.
type AnnotationEnrichment struct {
	CIStatuses          map[string]*github.CheckStatus
	EmptyWorktrees      map[string]*engine.WorktreeInfo
	WorktreeByStackRoot map[string]*engine.WorktreeInfo
}

// BuildFullAnnotation returns a fully populated BranchAnnotation including
// git operations (SHA, commits, diff stats), CI status, review status, and worktree info.
// Pass nil for enrichment to get just the base annotation without CI/worktree enrichment.
func BuildFullAnnotation(eng engine.Engine, branch engine.Branch, enrichment *AnnotationEnrichment) tree.BranchAnnotation {
	ann := GetBranchAnnotation(eng, branch)

	if enrichment == nil {
		return ann
	}

	// Apply CI status and review status
	if !branch.IsTrunk() && enrichment.CIStatuses != nil {
		if status := enrichment.CIStatuses[branch.GetName()]; status != nil {
			ann.CheckStatus = tree.CheckStatusPassing
			if status.Pending {
				ann.CheckStatus = tree.CheckStatusPending
			} else if !status.Passing {
				ann.CheckStatus = tree.CheckStatusFailing
			}

			switch status.ReviewDecision {
			case "APPROVED":
				ann.ReviewStatus = "Approved"
			case "CHANGES_REQUESTED":
				ann.ReviewStatus = "Changes Requested"
			case "REVIEW_REQUIRED":
				ann.ReviewStatus = "In Review"
			}
		}
	}

	// Apply worktree info
	addWorktreeInfo(eng, branch, &ann, enrichment.EmptyWorktrees, enrichment.WorktreeByStackRoot)

	return ann
}

// GetBranchAnnotation returns a tree.BranchAnnotation populated with standard branch metadata.
// This includes git operations (SHA, commit count, diff stats) which may be slow.
func GetBranchAnnotation(eng engine.BranchReader, branch engine.Branch) tree.BranchAnnotation {
	ann := tree.BranchAnnotation{
		IsLocked:      branch.IsLocked(),
		IsFrozen:      branch.IsFrozen(),
		Scope:         eng.GetScope(branch).String(),
		ExplicitScope: branch.GetExplicitScope().String(),
	}

	// Get short SHA for the branch
	if sha, err := branch.GetRevision(); err == nil && len(sha) >= 7 {
		ann.LocalSHA = sha[:7]
	}

	if !branch.IsTrunk() {
		// PR info (local metadata)
		if prInfo, _ := branch.GetPrInfo(); prInfo != nil {
			ann.PRNumber = prInfo.Number()
			ann.PRState = prInfo.State()
			ann.IsDraft = prInfo.IsDraft()
			ann.PRURL = prInfo.URL()
		}

		// Commit messages for detailed view
		if commits, err := branch.GetAllCommits(engine.CommitFormatReadable); err == nil {
			ann.CommitMessages = commits
			ann.CommitCount = len(commits)
		}

		// Merged downstack history
		if mergedHistory := branch.GetMergedDownstack(); len(mergedHistory) > 0 {
			ann.MergedDownstack = make([]tree.MergedParentDisplay, len(mergedHistory))
			for i, mp := range mergedHistory {
				ann.MergedDownstack[i] = tree.MergedParentDisplay{
					BranchName: mp.BranchName,
					PRNumber:   mp.PRNumber,
				}
				if mp.PRState != nil {
					ann.MergedDownstack[i].PRState = *mp.PRState
				}
			}
		}

		// Local stats
		if added, deleted, err := branch.GetDiffStats(); err == nil {
			ann.LinesAdded = added
			ann.LinesDeleted = deleted
		}
	}

	return ann
}
