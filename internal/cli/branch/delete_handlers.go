package branch

import (
	"stackit.dev/stackit/internal/actions/delete"
	"stackit.dev/stackit/internal/cli/common"
	"stackit.dev/stackit/internal/output"
	"stackit.dev/stackit/internal/tui"
	"stackit.dev/stackit/internal/tui/style"
)

// NewDeleteUI creates a runner and handler pair for delete operations.
// The runner manages terminal state; the handler processes events.
// Caller must defer runner.Cleanup() to restore terminal on exit.
// Currently returns nil runner as there's no TUI component yet.
func NewDeleteUI(out output.Output, _ output.Logger) (*tui.Runner, delete.Handler) {
	// TODO: Add interactive TUI handler when needed
	// For now, use simple handler for both TTY and non-TTY
	return nil, NewSimpleDeleteHandler(out)
}

// SimpleDeleteHandler provides streaming text output for delete operations
type SimpleDeleteHandler struct {
	common.BaseHandler
	deleted int
	skipped int
}

// NewSimpleDeleteHandler creates a new SimpleDeleteHandler
func NewSimpleDeleteHandler(out output.Output) *SimpleDeleteHandler {
	return &SimpleDeleteHandler{
		BaseHandler: common.NewBaseHandler(out),
	}
}

// Start is called at the beginning of delete
func (h *SimpleDeleteHandler) Start(_ int) {
	h.Lock()
	defer h.Unlock()
	h.deleted = 0
	h.skipped = 0
}

// OnBranch is called for each branch being deleted
func (h *SimpleDeleteHandler) OnBranch(name string, status delete.Status, _ *int) {
	h.Lock()
	defer h.Unlock()

	switch status {
	case delete.StatusDeleted:
		h.deleted++
		h.Output.Info("Deleted branch %s", style.ColorBranchName(name, false))
	case delete.StatusSkipped:
		h.skipped++
		h.Output.Info("Skipped branch %s", style.ColorBranchName(name, false))
	case delete.StatusRestacked:
		h.Output.Info("Restacked branch %s", style.ColorBranchName(name, false))
	}
}

// OnRestack is called when restacking children
func (h *SimpleDeleteHandler) OnRestack(childCount int) {
	h.Lock()
	defer h.Unlock()

	if childCount > 0 {
		h.Output.Info("Restacking %d child branch(es)...", childCount)
	}
}

// Complete is called when delete finishes
func (h *SimpleDeleteHandler) Complete(_, _ int) {
	// Summary is implicit from OnBranch calls
}

// PromptConfirm returns false for simple handler (non-interactive)
func (h *SimpleDeleteHandler) PromptConfirm(_ string, _ string) (bool, error) {
	return false, nil
}
