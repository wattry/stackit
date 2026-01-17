package stack

import (
	"stackit.dev/stackit/internal/actions/flatten"
	"stackit.dev/stackit/internal/cli/common"
	"stackit.dev/stackit/internal/output"
	"stackit.dev/stackit/internal/tui"
	"stackit.dev/stackit/internal/tui/style"
)

// NewFlattenUI creates a runner and handler pair for flatten operations.
// The runner manages terminal state; the handler processes events.
// Caller must defer runner.Cleanup() to restore terminal on exit.
// Currently returns nil runner as there's no TUI component yet.
func NewFlattenUI(out output.Output, _ output.Logger, interactive bool) (*tui.Runner, flatten.Handler) {
	if interactive {
		return nil, NewInteractiveFlattenHandler(out)
	}
	return nil, NewSimpleFlattenHandler(out)
}

// SimpleFlattenHandler provides streaming text output for flatten operations
type SimpleFlattenHandler struct {
	common.BaseHandler
	branchCount int
}

// NewSimpleFlattenHandler creates a new SimpleFlattenHandler
func NewSimpleFlattenHandler(out output.Output) *SimpleFlattenHandler {
	return &SimpleFlattenHandler{
		BaseHandler: common.NewBaseHandler(out),
	}
}

// Start is called at the beginning of flatten
func (h *SimpleFlattenHandler) Start(branchCount int) {
	h.Lock()
	defer h.Unlock()
	h.branchCount = branchCount
}

// OnStep is called for each step in the flatten process
func (h *SimpleFlattenHandler) OnStep(_ flatten.Step, _ flatten.StepStatus, _ string) {
	// Steps are handled silently in simple handler
}

// OnBranchMoved is called when a branch is moved to a new parent
func (h *SimpleFlattenHandler) OnBranchMoved(_, _, _ string) {
	// Moves are reported by the action itself
}

// Complete is called when flatten finishes
func (h *SimpleFlattenHandler) Complete(_ flatten.Result) {
	// Output already handled by the action
}

// PromptConfirmFlatten returns true (auto-confirm) for simple handler (non-interactive)
func (h *SimpleFlattenHandler) PromptConfirmFlatten(_ flatten.Preview) (bool, error) {
	return true, nil
}

// InteractiveFlattenHandler provides interactive prompts for flatten operations
type InteractiveFlattenHandler struct {
	SimpleFlattenHandler
}

// NewInteractiveFlattenHandler creates a new InteractiveFlattenHandler
func NewInteractiveFlattenHandler(out output.Output) *InteractiveFlattenHandler {
	return &InteractiveFlattenHandler{
		SimpleFlattenHandler: *NewSimpleFlattenHandler(out),
	}
}

// IsInteractive returns true for interactive handler
func (h *InteractiveFlattenHandler) IsInteractive() bool {
	return true
}

// PromptConfirmFlatten displays a preview of the flatten and asks for confirmation
func (h *InteractiveFlattenHandler) PromptConfirmFlatten(preview flatten.Preview) (bool, error) {
	h.Output.Newline()

	// Convert to TUI preview data
	moves := make([]tui.FlattenPlannedMove, len(preview.Moves))
	for i, m := range preview.Moves {
		moves[i] = tui.FlattenPlannedMove{
			Branch:    m.Branch,
			OldParent: m.OldParent,
			NewParent: m.NewParent,
		}
	}

	previewData := tui.FlattenPreviewData{
		Moves:          moves,
		UnchangedCount: preview.UnchangedCount,
		HasConflicts:   preview.HasConflicts,
		ConflictBranch: preview.ConflictBranch,
		ConflictError:  preview.ConflictError,
	}
	h.Output.Print(tui.RenderFlattenPreview(previewData))

	h.Output.Newline()

	// If there are conflicts, warn user more prominently
	if preview.HasConflicts {
		h.Output.Info("%s The flatten cannot proceed due to conflicts.", style.ColorRed("✗"))
		return tui.PromptConfirm("View the conflict details above and cancel?", false)
	}

	return tui.PromptConfirm("Proceed with flatten?", true)
}
