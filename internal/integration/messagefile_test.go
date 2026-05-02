package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestMessageFile verifies the --message-file flag wires through to create,
// modify, and squash by reading the commit message from a file rather than
// embedding it as a literal -m argument.
func TestMessageFile(t *testing.T) {
	t.Parallel()

	t.Run("create reads commit message from file", func(t *testing.T) {
		t.Parallel()
		sh := NewTestShellInProcess(t)

		msgPath := filepath.Join(sh.Dir(), ".commit-msg")
		require.NoError(t, os.WriteFile(msgPath, []byte("feat: from file\n"), 0o600))

		sh.Write("feature", "content").
			Run("create feature -F " + msgPath).
			OnBranch("feature").
			Git("log -1 --format=%s").
			OutputContains("feat: from file")
	})

	t.Run("create errors when both --message and --message-file are given", func(t *testing.T) {
		t.Parallel()
		sh := NewTestShellInProcess(t)

		msgPath := filepath.Join(sh.Dir(), ".commit-msg")
		require.NoError(t, os.WriteFile(msgPath, []byte("from file"), 0o600))

		sh.Write("feature", "content").
			RunExpectError("create feature -m 'inline' -F " + msgPath).
			OutputContains("mutually exclusive")
	})

	t.Run("modify updates message from file", func(t *testing.T) {
		t.Parallel()
		sh := NewTestShellInProcess(t)

		sh.Write("feature", "content").
			Run("create feature -m 'Original'")

		msgPath := filepath.Join(sh.Dir(), ".commit-msg")
		require.NoError(t, os.WriteFile(msgPath, []byte("feat: updated from file\n"), 0o600))

		sh.Run("modify -F " + msgPath).
			Git("log -1 --format=%s").
			OutputContains("feat: updated from file")
	})
}
