package utils

import "testing"

func TestSupportsTerminalControl(t *testing.T) {
	tests := []struct {
		name string
		term string
		want bool
	}{
		{name: "empty", term: "", want: false},
		{name: "dumb", term: "dumb", want: false},
		{name: "dumb variant", term: "dumb-color", want: false},
		{name: "xterm", term: "xterm-256color", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TERM", tt.term)
			if got := supportsTerminalControl(); got != tt.want {
				t.Fatalf("supportsTerminalControl() = %v, want %v", got, tt.want)
			}
		})
	}
}
