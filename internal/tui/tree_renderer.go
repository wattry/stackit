package tui

import (
	"stackit.dev/stackit/internal/engine"
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
	var branchFilter func(engine.Branch) bool
	if filter != nil {
		branchFilter = func(b engine.Branch) bool {
			return filter(b.GetName())
		}
	}

	graph := engine.BuildStackGraph(eng, strategy, branchFilter)

	return tree.NewStackTreeRenderer(
		graph.Current,
		graph.Trunk,
		func(branchName string) []string {
			return graph.Children(branchName)
		},
		func(branchName string) string {
			return graph.Parent(branchName)
		},
		func(branchName string) bool { return branchName == graph.Trunk },
		func(branchName string) bool {
			node := graph.Node(branchName)
			if node == nil {
				return false
			}
			return eng.IsUpToDate(node.Branch)
		},
	)
}

// GetBranchAnnotation returns a tree.BranchAnnotation populated with standard branch metadata.
func GetBranchAnnotation(eng engine.BranchReader, branch engine.Branch) tree.BranchAnnotation {
	ann := tree.BranchAnnotation{
		IsLocked:      branch.IsLocked(),
		IsFrozen:      branch.IsFrozen(),
		Scope:         eng.GetScope(branch).String(),
		ExplicitScope: branch.GetExplicitScope().String(),
	}

	if !branch.IsTrunk() {
		// PR info (local metadata)
		if prInfo, _ := branch.GetPrInfo(); prInfo != nil {
			ann.PRNumber = prInfo.Number()
			ann.PRState = prInfo.State()
			ann.IsDraft = prInfo.IsDraft()
		}

		// Local stats
		if count, err := branch.GetCommitCount(); err == nil {
			ann.CommitCount = count
		}
		if added, deleted, err := branch.GetDiffStats(); err == nil {
			ann.LinesAdded = added
			ann.LinesDeleted = deleted
		}
	}

	return ann
}
