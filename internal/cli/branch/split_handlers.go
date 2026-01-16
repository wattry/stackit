package branch

import (
	"fmt"
	"slices"
	"strings"
	"sync"

	"stackit.dev/stackit/internal/actions/split"
	"stackit.dev/stackit/internal/errors"
	"stackit.dev/stackit/internal/output"
	"stackit.dev/stackit/internal/tui"
	"stackit.dev/stackit/internal/tui/style"
	"stackit.dev/stackit/internal/utils"
)

// NewSplitUI creates a runner and handler pair for split operations.
// The runner manages terminal state; the handler processes events.
// Caller must defer runner.Cleanup() to restore terminal on exit.
func NewSplitUI(out output.Output, _ output.Logger) (*tui.Runner, split.Handler) {
	// Use TUI handler for interactive mode, simple handler otherwise
	if utils.IsInteractive() {
		return nil, NewTUISplitHandler(out)
	}
	return nil, NewSimpleSplitHandler(out)
}

// SimpleSplitHandler provides streaming text output for split operations
type SimpleSplitHandler struct {
	splog       output.Output
	mu          sync.Mutex
	branchName  string
	newBranches []string
}

// NewSimpleSplitHandler creates a new SimpleSplitHandler
func NewSimpleSplitHandler(splog output.Output) *SimpleSplitHandler {
	return &SimpleSplitHandler{
		splog: splog,
	}
}

// Start is called at the beginning of split
func (h *SimpleSplitHandler) Start(branchName string, _ split.Style) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.branchName = branchName
	h.newBranches = nil
}

// OnStep is called for each step in the split process
func (h *SimpleSplitHandler) OnStep(_ split.Step, _ split.StepStatus, _ string) {
	// Steps are handled silently in simple handler
}

// OnBranchCreated is called when a new branch is created during split
func (h *SimpleSplitHandler) OnBranchCreated(branchName string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.newBranches = append(h.newBranches, branchName)
	h.splog.Info("Created branch %s", style.ColorBranchName(branchName, false))
}

// Complete is called when split finishes
func (h *SimpleSplitHandler) Complete(result split.ActionResult) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if len(result.NewBranches) > 0 {
		h.splog.Info("Split %s into %d branches",
			style.ColorBranchName(result.OriginalBranch, false),
			len(result.NewBranches))
	}
}

// Cleanup is a no-op for the simple handler
func (h *SimpleSplitHandler) Cleanup() {}

// TUISplitHandler provides interactive TUI prompts for split operations
type TUISplitHandler struct {
	*SimpleSplitHandler
}

// NewTUISplitHandler creates a new TUISplitHandler
func NewTUISplitHandler(splog output.Output) *TUISplitHandler {
	return &TUISplitHandler{
		SimpleSplitHandler: NewSimpleSplitHandler(splog),
	}
}

// IsInteractive returns true as this handler supports interactive prompts
func (h *TUISplitHandler) IsInteractive() bool {
	return true
}

// PromptSplitType asks the user to choose between available split types
func (h *TUISplitHandler) PromptSplitType(availableTypes []split.TypeChoice) (split.Style, error) {
	if !utils.IsInteractive() {
		return "", errors.ErrCanceled
	}

	var options []tui.SelectOption
	for _, t := range availableTypes {
		if t.Available {
			options = append(options, tui.SelectOption{
				Label: fmt.Sprintf("%s - %s", t.Label, t.Description),
				Value: string(t.Style),
			})
		}
	}

	if len(options) == 0 {
		return "", fmt.Errorf("no split types available")
	}

	selected, err := tui.PromptSelect("How would you like to split this branch?", options, 0)
	if err != nil {
		return "", err
	}

	return split.Style(selected), nil
}

// PromptDirection asks the user where to place the new branch using an interactive tree view
func (h *TUISplitHandler) PromptDirection(ctx split.DirectionContext) (split.Direction, error) {
	if !utils.IsInteractive() {
		return "", errors.ErrCanceled
	}

	// Use the interactive direction selector with live tree preview
	selected, err := tui.PromptDirectionSelect(ctx.Engine, ctx.CurrentBranch, ctx.ParentBranch, ctx.Children)
	if err != nil {
		return "", err
	}

	return split.Direction(selected), nil
}

// ShowHunkSummary displays a summary of the remaining changes
func (h *TUISplitHandler) ShowHunkSummary(diff string) {
	h.splog.Info("Remaining changes:")
	for _, line := range strings.Split(diff, "\n") {
		h.splog.Info("  %s", line)
	}
	h.splog.Info("")
}

// PromptCommitMessage asks the user to enter or edit a commit message
func (h *TUISplitHandler) PromptCommitMessage(defaultMsg string) (string, error) {
	if !utils.IsInteractive() {
		return defaultMsg, nil
	}

	msg, err := tui.OpenEditor(defaultMsg, "COMMIT_EDITMSG-*")
	if err != nil {
		return "", err
	}
	return utils.CleanCommitMessage(msg), nil
}

// PromptBranchName asks the user to enter a branch name
func (h *TUISplitHandler) PromptBranchName(defaultName string, sessionNames []string, allBranchNames map[string]bool, originalBranchName string) (string, error) {
	if !utils.IsInteractive() {
		return defaultName, nil
	}

	branchName, err := tui.PromptTextInput(
		fmt.Sprintf("Enter branch name (default: %s):", defaultName),
		defaultName,
	)
	if err != nil {
		return "", err
	}

	if branchName == "" {
		branchName = defaultName
	}

	// Validate: empty names not allowed
	if strings.TrimSpace(branchName) == "" {
		return "", fmt.Errorf("branch name cannot be empty")
	}

	// Validate: don't allow names already picked in this split session
	if slices.Contains(sessionNames, branchName) {
		return "", fmt.Errorf("branch name %q is already used by another branch in this split", branchName)
	}

	// Validate: don't allow existing branch names (except the original being split)
	if branchName != originalBranchName && allBranchNames[branchName] {
		return "", fmt.Errorf("branch name %q already exists in the repository", branchName)
	}

	return branchName, nil
}

// PromptContinueOrCancel asks user whether to continue after no changes were staged
func (h *TUISplitHandler) PromptContinueOrCancel() (bool, error) {
	if !utils.IsInteractive() {
		return false, nil
	}

	options := []tui.SelectOption{
		{Label: "Try again", Value: "continue"},
		{Label: "Cancel split", Value: "cancel"},
	}

	selected, err := tui.PromptSelect("No changes staged. What would you like to do?", options, 0)
	if err != nil {
		return false, err
	}

	return selected == "continue", nil
}

// PromptEditCommitMessage asks whether the user wants to edit the commit message
func (h *TUISplitHandler) PromptEditCommitMessage() (bool, error) {
	if !utils.IsInteractive() {
		return false, nil
	}

	return tui.PromptConfirm("Edit commit message?", true)
}
