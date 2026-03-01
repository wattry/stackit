package pr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveUnifiedScope(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		scopes []string
		want   string
	}{
		{
			name:   "all empty",
			scopes: []string{"", ""},
			want:   "",
		},
		{
			name:   "single non-empty scope",
			scopes: []string{"", "PROJ-1", ""},
			want:   "PROJ-1",
		},
		{
			name:   "matching non-empty scopes",
			scopes: []string{"PROJ-1", "PROJ-1"},
			want:   "PROJ-1",
		},
		{
			name:   "mixed scopes",
			scopes: []string{"PROJ-1", "PROJ-2"},
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ResolveUnifiedScope(tt.scopes)
			require.Equal(t, tt.want, got)
		})
	}
}
