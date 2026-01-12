package merge

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/testhelpers"
	"stackit.dev/stackit/testhelpers/scenario"
)

func TestPRContentGenerator_GenerateConsolidationPR(t *testing.T) {
	t.Run("single_branch_no_pr", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)
		s.WithStack(map[string]string{
			"feature-a": "main",
		})

		gen := NewPRContentGenerator(s.Engine)
		content := gen.GenerateConsolidationPR([]BranchMergeInfo{
			{BranchName: "feature-a", PRNumber: 0},
		})

		assert.Equal(t, "Merging 1 PRs", content.Title)
		assert.NotContains(t, content.Body, "#0")
	})

	t.Run("multiple_branches_with_prs", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)
		s.WithStack(map[string]string{
			"feature-a":   "main",
			"feature-a-1": "feature-a",
		})

		gen := NewPRContentGenerator(s.Engine)
		content := gen.GenerateConsolidationPR([]BranchMergeInfo{
			{BranchName: "feature-a", PRNumber: 1},
			{BranchName: "feature-a-1", PRNumber: 2},
		})

		assert.Equal(t, "Merging 2 PRs", content.Title)
		assert.Contains(t, content.Body, "#1")
		assert.Contains(t, content.Body, "#2")
	})

	t.Run("branches_with_scope", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)
		s.WithStack(map[string]string{
			"feature-a": "main",
		})
		_ = s.Engine.SetScope(s.Engine.GetBranch("feature-a"), engine.NewScope("PROJ-123"))

		gen := NewPRContentGenerator(s.Engine)
		content := gen.GenerateConsolidationPR([]BranchMergeInfo{
			{BranchName: "feature-a", PRNumber: 1},
		})

		assert.Equal(t, "Merging PROJ-123", content.Title)
		assert.Contains(t, content.Body, "#1")
	})
}

func TestPRContentGenerator_GenerateMultiStackPR(t *testing.T) {
	t.Run("single_stack", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)
		s.WithStack(map[string]string{
			"feature-a": "main",
		})

		gen := NewPRContentGenerator(s.Engine)
		content := gen.GenerateMultiStackPR(
			[]MultiStackInfo{{RootBranch: "feature-a", AllBranches: []string{"feature-a"}}},
			nil,
		)

		assert.Equal(t, "Merging 1 PRs", content.Title)
		assert.Contains(t, content.Body, "feature-a")
	})

	t.Run("with_excluded_stacks", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)
		s.WithStack(map[string]string{
			"feature-a": "main",
		})

		gen := NewPRContentGenerator(s.Engine)
		content := gen.GenerateMultiStackPR(
			[]MultiStackInfo{{RootBranch: "feature-a", AllBranches: []string{"feature-a"}}},
			[]MultiStackExcluded{{Stack: MultiStackInfo{RootBranch: "feature-b"}, Reason: "conflict"}},
		)

		assert.Contains(t, content.Body, "### Excluded")
		assert.Contains(t, content.Body, "feature-b")
		assert.Contains(t, content.Body, "conflict")
	})
}
