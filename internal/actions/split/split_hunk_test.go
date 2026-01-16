package split

import (
	"testing"
)

func TestGenerateDefaultBranchName(t *testing.T) {
	tests := []struct {
		name          string
		originalName  string
		existingNames []string
		want          string
	}{
		{
			name:          "no existing names",
			originalName:  "feature",
			existingNames: []string{},
			want:          "feature_split",
		},
		{
			name:          "simple suffix already taken",
			originalName:  "feature",
			existingNames: []string{"feature_split"},
			want:          "feature_split_2",
		},
		{
			name:          "multiple suffixes taken",
			originalName:  "feature",
			existingNames: []string{"feature_split", "feature_split_2", "feature_split_3"},
			want:          "feature_split_4",
		},
		{
			name:          "non-sequential suffixes taken",
			originalName:  "feature",
			existingNames: []string{"feature_split", "feature_split_3"},
			want:          "feature_split_2",
		},
		{
			name:          "original name in existing doesn't affect result",
			originalName:  "feature",
			existingNames: []string{"feature"},
			want:          "feature_split",
		},
		{
			name:          "empty original name",
			originalName:  "",
			existingNames: []string{},
			want:          "_split",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateDefaultBranchName(tt.originalName, tt.existingNames)
			if got != tt.want {
				t.Errorf("generateDefaultBranchName(%q, %v) = %q, want %q",
					tt.originalName, tt.existingNames, got, tt.want)
			}
		})
	}
}
