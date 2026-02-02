package integration

import (
	"testing"
)

func TestDescribeCommand(t *testing.T) {
	t.Parallel()

	t.Run("set description with -m flag", func(t *testing.T) {
		t.Parallel()
		sh := NewTestShellInProcess(t)

		sh.Write("feature.txt", "content")
		sh.Run("create -m 'root feature'")

		sh.Run("describe -m 'Auth Feature'").
			OutputContains("Set stack description")
	})

	t.Run("set description with -m and -d flags", func(t *testing.T) {
		t.Parallel()
		sh := NewTestShellInProcess(t)

		sh.Write("feature.txt", "content")
		sh.Run("create -m 'root feature'")

		sh.Run("describe -m 'Auth Feature' -d 'OAuth2 implementation'").
			OutputContains("Set stack description").
			OutputContains("Auth Feature")
	})

	t.Run("show description", func(t *testing.T) {
		t.Parallel()
		sh := NewTestShellInProcess(t)

		sh.Write("feature.txt", "content")
		sh.Run("create -m 'root feature'")

		sh.Run("describe -m 'Auth Feature' -d 'OAuth2 implementation'")
		sh.Run("describe --show").
			OutputContains("Auth Feature")
	})

	t.Run("show description when none set", func(t *testing.T) {
		t.Parallel()
		sh := NewTestShellInProcess(t)

		sh.Write("feature.txt", "content")
		sh.Run("create -m 'root feature'")

		sh.Run("describe --show").
			OutputContains("has no description set")
	})

	t.Run("clear description", func(t *testing.T) {
		t.Parallel()
		sh := NewTestShellInProcess(t)

		sh.Write("feature.txt", "content")
		sh.Run("create -m 'root feature'")

		sh.Run("describe -m 'Auth Feature'")
		sh.Run("describe --clear").
			OutputContains("Cleared stack description")
	})

	t.Run("describe from child branch updates root", func(t *testing.T) {
		t.Parallel()
		sh := NewTestShellInProcess(t)

		// Create root branch
		sh.Write("root.txt", "content")
		sh.Run("create -m 'root feature'")

		// Create child branch
		sh.Write("child.txt", "content")
		sh.Run("create -m 'child feature'")

		// Set description from child
		sh.Run("describe -m 'Stack Title'").
			OutputContains("Set stack description")

		// Verify description is visible from child
		sh.Run("describe --show").
			OutputContains("Stack Title")

		// Switch to root and verify description is also there
		sh.Checkout("root-feature")
		sh.Run("describe --show").
			OutputContains("Stack Title")
	})

	t.Run("description appears in stack info", func(t *testing.T) {
		t.Parallel()
		sh := NewTestShellInProcess(t)

		sh.Write("feature.txt", "content")
		sh.Run("create -m 'root feature'")

		sh.Run("describe -m 'Auth Feature' -d 'OAuth2 implementation'")
		sh.Run("info --stack").
			OutputContains("Stack: Auth Feature").
			OutputContains("OAuth2 implementation")
	})

	t.Run("error on trunk", func(t *testing.T) {
		t.Parallel()
		sh := NewTestShellInProcess(t)

		sh.RunExpectError("describe -m 'Title'").
			OutputContains("cannot set stack description on trunk")
	})

	t.Run("error on untracked branch", func(t *testing.T) {
		t.Parallel()
		sh := NewTestShellInProcess(t)

		sh.Git("checkout -b untracked")
		sh.Write("file.txt", "content")
		sh.Git("commit -m 'commit'")

		sh.RunExpectError("describe -m 'Title'").
			OutputContains("not tracked")
	})
}
