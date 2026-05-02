package common

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReadMessage(t *testing.T) {
	t.Parallel()

	t.Run("returns inline message when no file given", func(t *testing.T) {
		t.Parallel()
		got, err := ReadMessage("feat: inline", "")
		require.NoError(t, err)
		require.Equal(t, "feat: inline", got)
	})

	t.Run("returns empty when neither given", func(t *testing.T) {
		t.Parallel()
		got, err := ReadMessage("", "")
		require.NoError(t, err)
		require.Equal(t, "", got)
	})

	t.Run("errors when both given", func(t *testing.T) {
		t.Parallel()
		_, err := ReadMessage("feat: a", "/tmp/msg")
		require.Error(t, err)
		require.Contains(t, err.Error(), "mutually exclusive")
	})

	t.Run("reads from file path", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		path := filepath.Join(dir, "msg")
		require.NoError(t, writeFile(path, "feat: from file\n"))

		got, err := ReadMessage("", path)
		require.NoError(t, err)
		require.Equal(t, "feat: from file", got)
	})

	t.Run("trims trailing whitespace and newlines", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		path := filepath.Join(dir, "msg")
		require.NoError(t, writeFile(path, "feat: trim me\n\n  \t\n"))

		got, err := ReadMessage("", path)
		require.NoError(t, err)
		require.Equal(t, "feat: trim me", got)
	})

	t.Run("preserves internal newlines for multi-line messages", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		path := filepath.Join(dir, "msg")
		require.NoError(t, writeFile(path, "feat: subject\n\nbody paragraph\n"))

		got, err := ReadMessage("", path)
		require.NoError(t, err)
		require.Equal(t, "feat: subject\n\nbody paragraph", got)
	})

	t.Run("returns error for missing file", func(t *testing.T) {
		t.Parallel()
		_, err := ReadMessage("", "/nonexistent/path/to/msg")
		require.Error(t, err)
		require.Contains(t, err.Error(), "read message file")
	})

	t.Run("reads from stdin when path is dash", func(t *testing.T) {
		t.Parallel()
		got, err := readMessageFrom("-", strings.NewReader("feat: from stdin\n"))
		require.NoError(t, err)
		require.Equal(t, "feat: from stdin", got)
	})
}

func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o600)
}
