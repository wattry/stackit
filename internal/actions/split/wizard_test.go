package split

import (
	"strings"
	"testing"
)

func TestBuildTypeChoices(t *testing.T) {
	tests := []struct {
		name               string
		hasMultipleCommits bool
		wantAvailableCount int
		wantCommitAvail    bool
	}{
		{
			name:               "single commit branch",
			hasMultipleCommits: false,
			wantAvailableCount: 2, // hunk and file
			wantCommitAvail:    false,
		},
		{
			name:               "multiple commit branch",
			hasMultipleCommits: true,
			wantAvailableCount: 3, // hunk, file, and commit
			wantCommitAvail:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			choices := buildTypeChoices(tt.hasMultipleCommits)

			// Check total count
			if len(choices) != 3 {
				t.Errorf("buildTypeChoices() returned %d choices, want 3", len(choices))
			}

			// Count available choices
			availableCount := 0
			var commitChoice *TypeChoice
			for i := range choices {
				if choices[i].Available {
					availableCount++
				}
				if choices[i].Style == StyleCommit {
					commitChoice = &choices[i]
				}
			}

			if availableCount != tt.wantAvailableCount {
				t.Errorf("buildTypeChoices() has %d available choices, want %d",
					availableCount, tt.wantAvailableCount)
			}

			if commitChoice == nil {
				t.Fatal("buildTypeChoices() missing commit choice")
			}
			if commitChoice.Available != tt.wantCommitAvail {
				t.Errorf("buildTypeChoices() commit.Available = %v, want %v",
					commitChoice.Available, tt.wantCommitAvail)
			}
		})
	}
}

func TestDirection(t *testing.T) {
	tests := []struct {
		direction Direction
		wantStr   string
	}{
		{DirectionBelow, "below"},
		{DirectionAbove, "above"},
		{Direction(""), ""},
	}

	for _, tt := range tests {
		t.Run(string(tt.direction), func(t *testing.T) {
			if got := tt.direction.String(); got != tt.wantStr {
				t.Errorf("Direction.String() = %q, want %q", got, tt.wantStr)
			}
		})
	}
}

func TestBuildDirectionTreeViz(t *testing.T) {
	tests := []struct {
		name           string
		currentBranch  string
		parentBranch   string
		childBranches  []string
		wantContains   []string
		wantNotContain []string
	}{
		{
			name:          "no children",
			currentBranch: "feature",
			parentBranch:  "main",
			childBranches: []string{},
			wantContains: []string{
				"main",
				"feature",
				"you are here",
				"[BELOW]",
				"[ABOVE]",
			},
		},
		{
			name:          "with children",
			currentBranch: "feature",
			parentBranch:  "main",
			childBranches: []string{"child1", "child2"},
			wantContains: []string{
				"main",
				"feature",
				"child1",
				"child2",
				"[BELOW]",
				"[ABOVE]",
			},
		},
		{
			name:          "single child",
			currentBranch: "feature",
			parentBranch:  "develop",
			childBranches: []string{"bugfix"},
			wantContains: []string{
				"develop",
				"feature",
				"bugfix",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildDirectionTreeViz(tt.currentBranch, tt.parentBranch, tt.childBranches)

			for _, want := range tt.wantContains {
				if !strings.Contains(result, want) {
					t.Errorf("buildDirectionTreeViz() missing %q in:\n%s", want, result)
				}
			}

			for _, notWant := range tt.wantNotContain {
				if strings.Contains(result, notWant) {
					t.Errorf("buildDirectionTreeViz() unexpectedly contains %q in:\n%s", notWant, result)
				}
			}
		})
	}
}
