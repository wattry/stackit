package errors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFailedTo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		action   string
		target   string
		err      error
		expected string
	}{
		{
			name:     "wraps error with action and target",
			action:   "get",
			target:   "branch revision",
			err:      errors.New("ref not found"),
			expected: "failed to get branch revision: ref not found",
		},
		{
			name:     "returns nil for nil error",
			action:   "get",
			target:   "branch",
			err:      nil,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := FailedTo(tt.action, tt.target, tt.err)
			if tt.err == nil {
				require.Nil(t, result)
			} else {
				require.Equal(t, tt.expected, result.Error())
			}
		})
	}
}

func TestWhile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		action   string
		err      error
		expected string
	}{
		{
			name:     "wraps error with action",
			action:   "checking branch status",
			err:      errors.New("git error"),
			expected: "checking branch status: git error",
		},
		{
			name:     "returns nil for nil error",
			action:   "checking",
			err:      nil,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := While(tt.action, tt.err)
			if tt.err == nil {
				require.Nil(t, result)
			} else {
				require.Equal(t, tt.expected, result.Error())
			}
		})
	}
}

func TestOp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		operation string
		err       error
		expected  string
	}{
		{
			name:      "wraps error with operation",
			operation: "rebase",
			err:       errors.New("conflict"),
			expected:  "rebase: conflict",
		},
		{
			name:      "returns nil for nil error",
			operation: "rebase",
			err:       nil,
			expected:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := Op(tt.operation, tt.err)
			if tt.err == nil {
				require.Nil(t, result)
			} else {
				require.Equal(t, tt.expected, result.Error())
			}
		})
	}
}

func TestWrap(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		format   string
		args     []any
		err      error
		expected string
	}{
		{
			name:     "wraps error with formatted message",
			format:   "processing branch %s",
			args:     []any{"feature"},
			err:      errors.New("not found"),
			expected: "processing branch feature: not found",
		},
		{
			name:     "returns nil for nil error",
			format:   "processing",
			args:     nil,
			err:      nil,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := Wrap(tt.err, tt.format, tt.args...)
			if tt.err == nil {
				require.Nil(t, result)
			} else {
				require.Equal(t, tt.expected, result.Error())
			}
		})
	}
}

func TestErrorUnwrapping(t *testing.T) {
	t.Parallel()

	originalErr := errors.New("original error")

	t.Run("FailedTo preserves wrapped error", func(t *testing.T) {
		t.Parallel()
		wrapped := FailedTo("get", "branch", originalErr)
		require.ErrorIs(t, wrapped, originalErr)
	})

	t.Run("While preserves wrapped error", func(t *testing.T) {
		t.Parallel()
		wrapped := While("checking", originalErr)
		require.ErrorIs(t, wrapped, originalErr)
	})

	t.Run("Op preserves wrapped error", func(t *testing.T) {
		t.Parallel()
		wrapped := Op("rebase", originalErr)
		require.ErrorIs(t, wrapped, originalErr)
	})

	t.Run("Wrap preserves wrapped error", func(t *testing.T) {
		t.Parallel()
		wrapped := Wrap(originalErr, "context")
		require.ErrorIs(t, wrapped, originalErr)
	})
}
