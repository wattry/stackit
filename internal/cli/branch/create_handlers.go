package branch

import (
	"stackit.dev/stackit/internal/actions/create"
	"stackit.dev/stackit/internal/cli/common"
	"stackit.dev/stackit/internal/output"
	"stackit.dev/stackit/internal/tui"
	"stackit.dev/stackit/internal/tui/style"
)

// NewCreateUI creates a runner and handler pair for create operations.
// The runner manages terminal state; the handler processes events.
// Caller must defer runner.Cleanup() to restore terminal on exit.
// Currently returns nil runner as there's no TUI component yet.
func NewCreateUI(out output.Output, _ output.Logger) (*tui.Runner, create.Handler) {
	// TODO: Add interactive TUI handler when needed
	// For now, use simple handler for both TTY and non-TTY
	return nil, NewSimpleCreateHandler(out)
}

// SimpleCreateHandler provides streaming text output for create operations
type SimpleCreateHandler struct {
	common.BaseHandler
	parentBranch string
}

// NewSimpleCreateHandler creates a new SimpleCreateHandler
func NewSimpleCreateHandler(out output.Output) *SimpleCreateHandler {
	return &SimpleCreateHandler{
		BaseHandler: common.NewBaseHandler(out),
	}
}

// Start is called at the beginning of create
func (h *SimpleCreateHandler) Start(parentBranch string) {
	h.Lock()
	defer h.Unlock()
	h.parentBranch = parentBranch
}

// OnStep is called for each step in the create process
func (h *SimpleCreateHandler) OnStep(step create.Step, status create.StepStatus, message string) {
	h.Lock()
	defer h.Unlock()

	// Only show meaningful messages for completed/failed steps
	switch status {
	case create.StatusCompleted:
		h.printStepCompleted(step, message)
	case create.StatusFailed:
		h.Output.Error("Failed: %s", message)
	case create.StatusSkipped:
		// Skip silently for most steps
		if step == create.StepCommit {
			h.Output.Info("No staged changes; created a branch with no commit.")
		}
	}
}

// Complete is called when create finishes
func (h *SimpleCreateHandler) Complete(result create.Result) {
	h.Lock()
	defer h.Unlock()

	h.Output.Info("Created branch %s on %s",
		style.ColorBranchName(result.BranchName, true),
		style.ColorBranchName(result.ParentBranch, false))
}

// PromptStageChanges returns false for simple handler (non-interactive)
func (h *SimpleCreateHandler) PromptStageChanges() (bool, error) {
	return false, nil
}

func (h *SimpleCreateHandler) printStepCompleted(_ create.Step, _ string) {
	// Most steps are silent - output is handled in Complete
	// Only show certain steps for verbose feedback
	// Worktree creation is shown by the action itself
}
