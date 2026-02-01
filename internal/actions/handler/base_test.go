package handler

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNullBase(t *testing.T) {
	t.Parallel()

	var base NullBase

	t.Run("Cleanup is no-op", func(t *testing.T) {
		t.Parallel()
		// Should not panic
		base.Cleanup()
	})

	t.Run("IsInteractive returns false", func(t *testing.T) {
		t.Parallel()
		require.False(t, base.IsInteractive())
	})
}

func TestNullPromptHandler(t *testing.T) {
	t.Parallel()

	var h NullPromptHandler

	t.Run("PromptConfirm returns false", func(t *testing.T) {
		t.Parallel()
		confirmed, err := h.PromptConfirm("test message")
		require.NoError(t, err)
		require.False(t, confirmed)
	})
}

func TestAutoConfirmPromptHandler(t *testing.T) {
	t.Parallel()

	var h AutoConfirmPromptHandler

	t.Run("PromptConfirm returns true", func(t *testing.T) {
		t.Parallel()
		confirmed, err := h.PromptConfirm("test message")
		require.NoError(t, err)
		require.True(t, confirmed)
	})
}

type testStep string

func TestNullProgress(t *testing.T) {
	t.Parallel()

	var h NullProgress[testStep]

	t.Run("OnStep is no-op", func(t *testing.T) {
		t.Parallel()
		// Should not panic
		h.OnStep("test", StatusStarted, "message")
	})
}

func TestStepStatusValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status   StepStatus
		expected string
	}{
		{StatusStarted, "started"},
		{StatusCompleted, "completed"},
		{StatusSkipped, "skipped"},
		{StatusFailed, "failed"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.expected, string(tt.status))
		})
	}
}

// Test that embedding works as expected
type TestHandler struct {
	NullBase
	NullPromptHandler
	NullProgress[testStep]
}

func TestEmbedding(t *testing.T) {
	t.Parallel()

	var h TestHandler

	t.Run("embedded NullBase works", func(t *testing.T) {
		t.Parallel()
		require.False(t, h.IsInteractive())
		h.Cleanup() // Should not panic
	})

	t.Run("embedded NullPromptHandler works", func(t *testing.T) {
		t.Parallel()
		confirmed, err := h.PromptConfirm("test")
		require.NoError(t, err)
		require.False(t, confirmed)
	})

	t.Run("embedded NullProgress works", func(t *testing.T) {
		t.Parallel()
		h.OnStep("test", StatusStarted, "message") // Should not panic
	})
}
