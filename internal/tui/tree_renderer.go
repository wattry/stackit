package tui

import (
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/tui/components/tree"
)

// NewStackTreeRenderer creates a tree renderer configured for the current engine state
func NewStackTreeRenderer(eng engine.BranchReader) *tree.StackTreeRenderer {
	currentBranch := eng.CurrentBranch()
	currentBranchName := ""
	if currentBranch != nil {
		currentBranchName = currentBranch.GetName()
	}

	trunk := eng.Trunk()

	return tree.NewStackTreeRenderer(
		currentBranchName,
		trunk.GetName(),
		func(branchName string) []string {
			branch := eng.GetBranch(branchName)
			children := branch.GetChildren()
			childNames := make([]string, len(children))
			for i, c := range children {
				childNames[i] = c.GetName()
			}
			return childNames
		},
		func(branchName string) string {
			branch := eng.GetBranch(branchName)
			parent := branch.GetParent()
			if parent == nil {
				return ""
			}
			return parent.GetName()
		},
		func(branchName string) bool { return eng.IsTrunk(eng.GetBranch(branchName)) },
		func(branchName string) bool {
			return eng.IsUpToDate(eng.GetBranch(branchName))
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
