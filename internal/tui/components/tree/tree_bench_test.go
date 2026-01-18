package tree

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func init() {
	// Force color output for benchmarks to simulate realistic rendering
	lipgloss.SetColorProfile(termenv.TrueColor)
}

const benchTrunk = "main"

// generateLargeTree creates a tree with the specified number of branches
// arranged in a binary tree structure for testing performance.
func generateLargeTree(numBranches int) *MockTreeData {
	childrenMap := make(map[string][]string)
	parentsMap := make(map[string]string)
	fixedMap := make(map[string]bool)

	// Create branch names
	branches := make([]string, numBranches)
	branches[0] = benchTrunk
	for i := 1; i < numBranches; i++ {
		branches[i] = branchName(i)
	}

	// Build binary tree structure
	fixedMap[benchTrunk] = true
	for i := 1; i < numBranches; i++ {
		branch := branches[i]
		parentIdx := (i - 1) / 2
		parent := branches[parentIdx]

		parentsMap[branch] = parent
		childrenMap[parent] = append(childrenMap[parent], branch)
		fixedMap[branch] = true
	}

	// Initialize empty children lists for leaf nodes
	for _, branch := range branches {
		if _, exists := childrenMap[branch]; !exists {
			childrenMap[branch] = []string{}
		}
	}

	return &MockTreeData{
		CurrentVal:  branches[numBranches/2], // Middle branch
		TrunkVal:    benchTrunk,
		ChildrenMap: childrenMap,
		ParentsMap:  parentsMap,
		FixedMap:    fixedMap,
	}
}

func branchName(i int) string {
	// Generate branch names like "branch-1", "branch-2", etc.
	return "branch-" + string(rune('0'+i%10)) + string(rune('0'+i/10%10)) + string(rune('0'+i/100%10))
}

// generateDeepLinearTree creates a deep linear stack (no branching).
func generateDeepLinearTree(depth int) *MockTreeData {
	childrenMap := make(map[string][]string)
	parentsMap := make(map[string]string)
	fixedMap := make(map[string]bool)

	branches := make([]string, depth)
	branches[0] = benchTrunk
	for i := 1; i < depth; i++ {
		branches[i] = branchName(i)
	}

	fixedMap[benchTrunk] = true
	for i := 1; i < depth; i++ {
		branch := branches[i]
		parent := branches[i-1]
		parentsMap[branch] = parent
		childrenMap[parent] = []string{branch}
		fixedMap[branch] = true
	}
	childrenMap[branches[depth-1]] = []string{}

	return &MockTreeData{
		CurrentVal:  branches[depth-1],
		TrunkVal:    benchTrunk,
		ChildrenMap: childrenMap,
		ParentsMap:  parentsMap,
		FixedMap:    fixedMap,
	}
}

func BenchmarkRenderStack_SmallTree(b *testing.B) {
	mock := NewMockTreeData()
	renderer := NewRenderer(mock)

	b.ResetTimer()
	for b.Loop() {
		renderer.RenderStack(mock.TrunkVal, RenderOptions{Short: true})
	}
}

func BenchmarkRenderStack_SmallTree_Full(b *testing.B) {
	mock := NewMockTreeData()
	renderer := NewRenderer(mock)

	b.ResetTimer()
	for b.Loop() {
		renderer.RenderStack(mock.TrunkVal, RenderOptions{Short: false})
	}
}

func BenchmarkRenderStack_LargeTree_100Branches(b *testing.B) {
	mock := generateLargeTree(100)
	renderer := NewRenderer(mock)

	b.ResetTimer()
	for b.Loop() {
		renderer.RenderStack(mock.TrunkVal, RenderOptions{Short: true})
	}
}

func BenchmarkRenderStack_LargeTree_100Branches_Full(b *testing.B) {
	mock := generateLargeTree(100)
	renderer := NewRenderer(mock)

	b.ResetTimer()
	for b.Loop() {
		renderer.RenderStack(mock.TrunkVal, RenderOptions{Short: false})
	}
}

func BenchmarkRenderStack_DeepLinear_50Levels(b *testing.B) {
	mock := generateDeepLinearTree(50)
	renderer := NewRenderer(mock)

	b.ResetTimer()
	for b.Loop() {
		renderer.RenderStack(mock.TrunkVal, RenderOptions{Short: true})
	}
}

func BenchmarkRenderStack_DeepLinear_50Levels_Full(b *testing.B) {
	mock := generateDeepLinearTree(50)
	renderer := NewRenderer(mock)

	b.ResetTimer()
	for b.Loop() {
		renderer.RenderStack(mock.TrunkVal, RenderOptions{Short: false})
	}
}

func BenchmarkRenderStack_WithAnnotations(b *testing.B) {
	mock := generateLargeTree(50)
	renderer := NewRenderer(mock)

	// Add annotations to all branches
	prNum := 1
	for branch := range mock.ChildrenMap {
		renderer.SetAnnotation(branch, BranchAnnotation{
			PRNumber:     &prNum,
			CheckStatus:  CheckStatusPassing,
			ReviewStatus: "Approved",
			CommitCount:  3,
			LinesAdded:   100,
			LinesDeleted: 20,
			Scope:        "test",
		})
		prNum++
	}

	b.ResetTimer()
	for b.Loop() {
		renderer.RenderStack(mock.TrunkVal, RenderOptions{Short: false})
	}
}

func BenchmarkRenderStack_Reversed(b *testing.B) {
	mock := generateLargeTree(50)
	renderer := NewRenderer(mock)

	b.ResetTimer()
	for b.Loop() {
		renderer.RenderStack(mock.TrunkVal, RenderOptions{Short: true, Reverse: true})
	}
}

func BenchmarkRenderStackDetailed(b *testing.B) {
	mock := generateLargeTree(50)
	renderer := NewRenderer(mock)

	b.ResetTimer()
	for b.Loop() {
		renderer.RenderStackDetailed(mock.TrunkVal, RenderOptions{Short: false})
	}
}
