package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBranchPattern_ContainsScope(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		pattern  BranchPattern
		expected bool
	}{
		{
			name:     "pattern with scope",
			pattern:  BranchPattern("{scope}/{message}"),
			expected: true,
		},
		{
			name:     "pattern with scope and other placeholders",
			pattern:  BranchPattern("{username}/{scope}/{message}"),
			expected: true,
		},
		{
			name:     "pattern without scope",
			pattern:  BranchPattern("{username}/{message}"),
			expected: false,
		},
		{
			name:     "default pattern",
			pattern:  DefaultBranchPattern,
			expected: false,
		},
		{
			name:     "empty pattern uses default",
			pattern:  BranchPattern(""),
			expected: false,
		},
		{
			name:     "pattern with scope at end",
			pattern:  BranchPattern("{message}-{scope}"),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.pattern.ContainsScope()
			require.Equal(t, tt.expected, result)
		})
	}
}
