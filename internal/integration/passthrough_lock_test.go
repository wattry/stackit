package integration

import (
	"testing"
)

func TestPassthroughLock(t *testing.T) {
	t.Parallel()
	binaryPath := getStackitBinary(t)

	t.Run("blocking modifying passthrough on locked branch", func(t *testing.T) {
		t.Parallel()
		sh := NewTestShell(t, binaryPath)

		// Setup
		sh.Run("create feature-a").WriteFile("a", "A").Git("add a").Git("commit -m 'A'")
		sh.Run("lock feature-a")

		// These should be blocked
		sh.WriteFile("a", "modified")
		sh.RunExpectError("add a").OutputContains("locked")
		sh.RunExpectError("reset --hard HEAD").OutputContains("locked")

		// This should be allowed (read-only)
		sh.Run("status").OutputContains("modified:")
	})
}
