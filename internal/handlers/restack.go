// Package handlers provides shared handler interfaces for CLI output.
package handlers

import (
	"stackit.dev/stackit/internal/engine"
)

// RestackResult represents the outcome of a restack operation for a single branch
type RestackResult string

const (
	// RestackDone indicates the branch was successfully restacked
	RestackDone RestackResult = "done"
	// RestackUnneeded indicates the branch didn't need restacking
	RestackUnneeded RestackResult = "unneeded"
	// RestackConflict indicates the branch had a conflict
	RestackConflict RestackResult = "conflict"
)

// RestackHandler abstracts TTY vs non-TTY output for restack operations
// This interface is shared between sync, get, and restack commands
type RestackHandler interface {
	// OnRestackStart is called at the beginning of restack with branch count
	OnRestackStart(branchCount int)

	// OnRestackBranch is called for each branch during restack
	OnRestackBranch(branch string, result RestackResult, newRev string, prNumber *int, lockReason engine.LockReason, frozen bool)

	// OnRestackComplete is called when restack finishes
	OnRestackComplete(restacked, skipped int, conflicts []string)
}

// NullRestackHandler is a no-op handler for testing or when output is not needed
type NullRestackHandler struct{}

// OnRestackStart implements RestackHandler.
func (h *NullRestackHandler) OnRestackStart(_ int) {}

// OnRestackBranch implements RestackHandler.
func (h *NullRestackHandler) OnRestackBranch(_ string, _ RestackResult, _ string, _ *int, _ engine.LockReason, _ bool) {
}

// OnRestackComplete implements RestackHandler.
func (h *NullRestackHandler) OnRestackComplete(_, _ int, _ []string) {}
