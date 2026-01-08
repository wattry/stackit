package merge

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetermineRecommendedStrategy(t *testing.T) {
	tests := []struct {
		name        string
		branchCount int
		want        Strategy
	}{
		{
			name:        "zero branches returns top-down",
			branchCount: 0,
			want:        StrategyTopDown,
		},
		{
			name:        "one branch returns top-down",
			branchCount: 1,
			want:        StrategyTopDown,
		},
		{
			name:        "two branches returns bottom-up",
			branchCount: 2,
			want:        StrategyBottomUp,
		},
		{
			name:        "three branches returns consolidate",
			branchCount: 3,
			want:        StrategyConsolidate,
		},
		{
			name:        "many branches returns consolidate",
			branchCount: 10,
			want:        StrategyConsolidate,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetermineRecommendedStrategy(tt.branchCount)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAnalyzeMidStackScope(t *testing.T) {
	tests := []struct {
		name         string
		plan         *Plan
		currentScope string
		wantBranches []string
	}{
		{
			name:         "empty scope returns nil",
			plan:         &Plan{UpstackBranches: []string{"branch1"}},
			currentScope: "",
			wantBranches: nil,
		},
		{
			name:         "no upstack branches returns nil",
			plan:         &Plan{UpstackBranches: []string{}},
			currentScope: "feature",
			wantBranches: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: Full testing requires a mock engine, these test edge cases
			got := AnalyzeMidStackScope(nil, tt.plan, tt.currentScope)
			assert.Equal(t, tt.wantBranches, got)
		})
	}
}

func TestPostMergeActionRequired_Error(t *testing.T) {
	err := &PostMergeActionRequired{Action: PostMergeSyncTrunk}
	assert.Equal(t, "post-merge action required: sync-trunk", err.Error())

	err2 := &PostMergeActionRequired{Action: PostMergeDone}
	assert.Equal(t, "post-merge action required: done", err2.Error())
}
