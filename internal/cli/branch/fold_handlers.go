package branch

import (
	"stackit.dev/stackit/internal/actions/fold"
	"stackit.dev/stackit/internal/cli/common"
	"stackit.dev/stackit/internal/output"
	"stackit.dev/stackit/internal/tui"
	"stackit.dev/stackit/internal/tui/style"
)

// NewFoldUI creates a runner and handler pair for fold operations.
// The runner manages terminal state; the handler processes events.
// Caller must defer runner.Cleanup() to restore terminal on exit.
// Currently returns nil runner as there's no TUI component yet.
func NewFoldUI(out output.Output, _ output.Logger) (*tui.Runner, fold.Handler) {
	// TODO: Add interactive TUI handler when needed
	// For now, use simple handler for both TTY and non-TTY
	return nil, NewSimpleFoldHandler(out)
}

// SimpleFoldHandler provides streaming text output for fold operations
type SimpleFoldHandler struct {
	common.BaseHandler
	currentBranch string
	parentBranch  string
}

// NewSimpleFoldHandler creates a new SimpleFoldHandler
func NewSimpleFoldHandler(out output.Output) *SimpleFoldHandler {
	return &SimpleFoldHandler{
		BaseHandler: common.NewBaseHandler(out),
	}
}

// Start is called at the beginning of fold
func (h *SimpleFoldHandler) Start(currentBranch, parentBranch string, _ bool) {
	h.Lock()
	defer h.Unlock()
	h.currentBranch = currentBranch
	h.parentBranch = parentBranch
}

// OnStep is called for each step in the fold process
func (h *SimpleFoldHandler) OnStep(_ fold.Step, _ fold.StepStatus, _ string) {
	// Steps are handled silently in simple handler
}

// Complete is called when fold finishes
func (h *SimpleFoldHandler) Complete(result fold.Result) {
	h.Lock()
	defer h.Unlock()

	h.Output.Info("Folded %s into %s",
		style.ColorBranchName(result.FoldedBranch, false),
		style.ColorBranchName(result.IntoBranch, true))
}
