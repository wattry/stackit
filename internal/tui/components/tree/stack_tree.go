package tree

import "stackit.dev/stackit/internal/engine"

// StackTree represents the structure of a branch stack for rendering.
// It encapsulates parent/child relationships and metadata needed to visualize stacks.
// This type is designed to be embedded in events and other contexts where stack
// visualization is needed.
type StackTree struct {
	Branches      []string            // branches in the stack, in order
	CurrentBranch string              // the currently checked out branch
	TrunkBranch   string              // the trunk/main branch name
	ParentMap     map[string]string   // branch -> parent mapping
	ChildrenMap   map[string][]string // branch -> children mapping
}

// NewStackTree creates a StackTree from a list of branches.
// It builds the parent/child relationship maps from the branch metadata.
func NewStackTree(branches []engine.Branch, currentBranch, trunkBranch string) *StackTree {
	parentMap := make(map[string]string)
	childrenMap := make(map[string][]string)
	branchNames := make([]string, len(branches))

	for i, branch := range branches {
		branchName := branch.GetName()
		branchNames[i] = branchName
		parentName := branch.GetParentPrecondition()
		parentMap[branchName] = parentName

		// Build children map (inverse of parent map)
		if parentName != "" {
			childrenMap[parentName] = append(childrenMap[parentName], branchName)
		}
	}

	return &StackTree{
		Branches:      branchNames,
		CurrentBranch: currentBranch,
		TrunkBranch:   trunkBranch,
		ParentMap:     parentMap,
		ChildrenMap:   childrenMap,
	}
}

// ToRenderer converts the tree data into a StackTreeRenderer with default settings.
// All branches are considered "fixed" (no restack indicators).
func (t *StackTree) ToRenderer() *StackTreeRenderer {
	return t.ToRendererWithFixed(func(_ string) bool {
		return true // All branches are "fixed" by default
	})
}

// ToRendererWithFixed converts the tree data into a StackTreeRenderer with custom
// fixed branch logic. The isBranchFixed function determines which branches should
// be marked as not needing restack.
func (t *StackTree) ToRendererWithFixed(isBranchFixed func(string) bool) *StackTreeRenderer {
	return NewStackTreeRenderer(
		t.CurrentBranch,
		t.TrunkBranch,
		func(branchName string) []string {
			return t.ChildrenMap[branchName]
		},
		func(branchName string) string {
			return t.ParentMap[branchName]
		},
		func(branchName string) bool {
			return branchName == t.TrunkBranch
		},
		isBranchFixed,
	)
}
