package stack

import (
	"fmt"

	"stackit.dev/stackit/internal/actions/move"
	"stackit.dev/stackit/internal/cli/common"
	"stackit.dev/stackit/internal/output"
	"stackit.dev/stackit/internal/tui"
	"stackit.dev/stackit/internal/tui/style"
)

// NewMoveUI creates a runner and handler pair for move operations.
// The runner manages terminal state; the handler processes events.
// Caller must defer runner.Cleanup() to restore terminal on exit.
// Currently returns nil runner as there's no TUI component yet.
func NewMoveUI(out output.Output, _ output.Logger, interactive bool) (*tui.Runner, move.Handler) {
	if interactive {
		return nil, NewInteractiveMoveHandler(out)
	}
	return nil, NewSimpleMoveHandler(out)
}

// SimpleMoveHandler provides streaming text output for move operations
type SimpleMoveHandler struct {
	common.BaseHandler
	sourceBranch string
	oldParent    string
	newParent    string
}

// NewSimpleMoveHandler creates a new SimpleMoveHandler
func NewSimpleMoveHandler(out output.Output) *SimpleMoveHandler {
	return &SimpleMoveHandler{
		BaseHandler: common.NewBaseHandler(out),
	}
}

// Start is called at the beginning of move
func (h *SimpleMoveHandler) Start(sourceBranch, oldParent, newParent string) {
	h.Lock()
	defer h.Unlock()
	h.sourceBranch = sourceBranch
	h.oldParent = oldParent
	h.newParent = newParent
}

// OnStep is called for each step in the move process
func (h *SimpleMoveHandler) OnStep(_ move.Step, _ move.StepStatus, _ string) {
	// Steps are handled silently in simple handler
}

// OnRename is called when a branch is renamed due to scope change
func (h *SimpleMoveHandler) OnRename(oldName, newName string) {
	h.Lock()
	defer h.Unlock()
	h.Output.Info("Renamed branch %s to %s",
		style.ColorBranchName(oldName, false),
		style.ColorBranchName(newName, true))
}

// Complete is called when move finishes
func (h *SimpleMoveHandler) Complete(_ move.Result) {
	// Output already handled by the action
}

// PromptRename returns false for simple handler (non-interactive)
func (h *SimpleMoveHandler) PromptRename(_, oldScope, newScope string) (bool, error) {
	// In non-interactive mode, print a message but don't rename
	h.Output.Info("Branch name contains '%s', but its scope will now be '%s'. Use interactive mode to rename.",
		oldScope, newScope)
	return false, nil
}

// PromptConfirmMove returns true (auto-confirm) for simple handler (non-interactive)
func (h *SimpleMoveHandler) PromptConfirmMove(_ move.Preview) (bool, error) {
	return true, nil
}

// InteractiveMoveHandler provides interactive prompts for move operations
type InteractiveMoveHandler struct {
	SimpleMoveHandler
}

// NewInteractiveMoveHandler creates a new InteractiveMoveHandler
func NewInteractiveMoveHandler(out output.Output) *InteractiveMoveHandler {
	return &InteractiveMoveHandler{
		SimpleMoveHandler: *NewSimpleMoveHandler(out),
	}
}

// IsInteractive returns true for interactive handler
func (h *InteractiveMoveHandler) IsInteractive() bool {
	return true
}

// PromptRename prompts user to confirm branch rename due to scope change
func (h *InteractiveMoveHandler) PromptRename(_, oldScope, newScope string) (bool, error) {
	return tui.PromptConfirm(fmt.Sprintf("Branch name contains '%s', but its scope will now be '%s'. Would you like to rename the branch?", oldScope, newScope), true)
}

// PromptConfirmMove displays a preview of the move and asks for confirmation
func (h *InteractiveMoveHandler) PromptConfirmMove(preview move.Preview) (bool, error) {
	h.Output.Newline()
	h.Output.Info("Move Preview:")
	h.Output.Info("  Branch: %s", style.ColorBranchName(preview.SourceBranch, true))
	h.Output.Info("  From:   %s", style.ColorBranchName(preview.OldParent, false))
	h.Output.Info("  To:     %s", style.ColorBranchName(preview.NewParent, false))

	if len(preview.Commits) > 0 {
		h.Output.Newline()
		h.Output.Info("Commits to be moved (%d):", len(preview.Commits))
		for _, commit := range preview.Commits {
			h.Output.Info("  • %s", commit)
		}
	}

	if len(preview.Descendants) > 0 {
		h.Output.Newline()
		h.Output.Info("Descendants to be restacked (%d):", len(preview.Descendants))
		for _, desc := range preview.Descendants {
			h.Output.Info("  • %s", style.ColorBranchName(desc, false))
		}
	}

	h.Output.Newline()
	h.Output.Info("This will rebase %s's commits onto %s.",
		style.ColorBranchName(preview.SourceBranch, true),
		style.ColorBranchName(preview.NewParent, false))
	if len(preview.Descendants) > 0 {
		h.Output.Info("All descendant branches will be automatically restacked.")
	}
	h.Output.Newline()

	return tui.PromptConfirm("Proceed with move?", true)
}
