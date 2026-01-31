package branch

import (
	"stackit.dev/stackit/internal/actions/absorb"
	"stackit.dev/stackit/internal/actions/handler"
	"stackit.dev/stackit/internal/cli/common"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/output"
	"stackit.dev/stackit/internal/tui"
)

// NewAbsorbUI creates a runner and handler pair for absorb operations.
// The runner manages terminal state; the handler processes events.
// Caller must defer runner.Cleanup() to restore terminal on exit.
// Currently returns nil runner as there's no TUI component yet.
func NewAbsorbUI(out output.Output, _ output.Logger) (*tui.Runner, absorb.Handler) {
	// TODO: Add interactive TUI handler when needed
	// For now, use simple handler for both TTY and non-TTY
	return nil, NewSimpleAbsorbHandler(out)
}

// SimpleAbsorbHandler provides streaming text output for absorb operations
type SimpleAbsorbHandler struct {
	common.BaseHandler
	absorbed int
}

// NewSimpleAbsorbHandler creates a new SimpleAbsorbHandler
func NewSimpleAbsorbHandler(out output.Output) *SimpleAbsorbHandler {
	return &SimpleAbsorbHandler{
		BaseHandler: common.NewBaseHandler(out),
	}
}

// Start is called at the beginning of absorb
func (h *SimpleAbsorbHandler) Start(_ bool) {
	h.Lock()
	defer h.Unlock()
	h.absorbed = 0
}

// OnStep is called for each step in the absorb process
func (h *SimpleAbsorbHandler) OnStep(_ absorb.Step, _ handler.StepStatus, _ string) {
	// Steps are handled silently in simple handler
}

// OnHunkTarget is called when a target is found for a hunk
func (h *SimpleAbsorbHandler) OnHunkTarget(_ string, _ string, _ string) {
	// Targets are shown via printAbsorbPlan in the action
}

// OnUnabsorbedHunk is called for hunks that could not be absorbed
func (h *SimpleAbsorbHandler) OnUnabsorbedHunk(_ git.Hunk) {
	// Unabsorbed hunks are shown via output in the action
}

// OnApply is called when hunks are applied to a branch
func (h *SimpleAbsorbHandler) OnApply(_ string, _ string) {
	h.Lock()
	defer h.Unlock()
	h.absorbed++
}

// Complete is called when absorb finishes
func (h *SimpleAbsorbHandler) Complete(_ absorb.Result) {
	// Summary is implicit from OnApply calls and action output
}

// PromptConfirm returns false for simple handler (non-interactive)
func (h *SimpleAbsorbHandler) PromptConfirm(_ string) (bool, error) {
	return false, nil
}
