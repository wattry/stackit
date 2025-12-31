package branch

import (
	"fmt"
	"strings"
	stdsync "sync"

	getAction "stackit.dev/stackit/internal/actions/get"
	"stackit.dev/stackit/internal/handlers"
	"stackit.dev/stackit/internal/tui"
	"stackit.dev/stackit/internal/tui/style"
)

// NewGetHandler creates the appropriate handler based on TTY availability
func NewGetHandler(splog *tui.Splog) getAction.Handler {
	// For now, just use simple handler (can add interactive later)
	return NewSimpleGetHandler(splog)
}

// SimpleGetHandler provides streaming text output for non-TTY environments
type SimpleGetHandler struct {
	splog        *tui.Splog
	currentPhase getAction.Phase
	mu           stdsync.Mutex
	targetBranch string
	prNumber     *int
}

// NewSimpleGetHandler creates a new SimpleGetHandler
func NewSimpleGetHandler(splog *tui.Splog) *SimpleGetHandler {
	return &SimpleGetHandler{
		splog: splog,
	}
}

// Start is called at the beginning of get
func (h *SimpleGetHandler) Start(targetBranch string, prNumber *int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.targetBranch = targetBranch
	h.prNumber = prNumber
}

// EmitEvent handles progress updates
func (h *SimpleGetHandler) EmitEvent(event getAction.Event) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Handle phase transitions
	if event.Type == getAction.EventStarted && event.Phase != h.currentPhase {
		h.currentPhase = event.Phase
		h.printPhaseHeader(event.Phase)
		return
	}

	// Handle progress events
	h.printEventLine(event)
}

// Complete is called when get finishes
func (h *SimpleGetHandler) Complete(summary getAction.Summary) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Print blank line before summary
	h.splog.Newline()

	// Handle up-to-date case
	if summary.UpToDate {
		h.splog.Info("✨ Everything is up to date!")
		return
	}

	// Print summary parts
	parts := []string{}
	if summary.BranchesCreated > 0 {
		parts = append(parts, fmt.Sprintf("synced %d new", summary.BranchesCreated))
	}
	if summary.BranchesUpdated > 0 {
		parts = append(parts, fmt.Sprintf("updated %d", summary.BranchesUpdated))
	}
	if summary.Restacked > 0 {
		parts = append(parts, fmt.Sprintf("restacked %d", summary.Restacked))
	}

	if len(parts) > 0 {
		h.splog.Info("✅ Summary: %s", strings.Join(parts, ", "))
	}

	// Checkout message
	h.splog.Info("Checked out %s", style.ColorBranchName(summary.TargetBranch, true))

	// Locked message if applicable
	if summary.IsLocked {
		h.splog.Info("Branch %s was retrieved in 'locked' mode, making it uneditable",
			style.ColorBranchName(summary.TargetBranch, false))
		h.splog.Info("Use %s to make it editable", style.ColorCyan("st unlock"))
	}
}

func (h *SimpleGetHandler) printPhaseHeader(phase getAction.Phase) {
	// Add spacing between phases (but not before first phase)
	if h.currentPhase != "" && phase != getAction.PhaseFetch {
		h.splog.Newline()
	}

	switch phase {
	case getAction.PhaseFetch:
		h.splog.Info("📥 Fetching from remote...")
	case getAction.PhaseSync:
		h.splog.Info("🔄 Syncing branches...")
	case getAction.PhaseMetadata:
		// Metadata phase is silent
	case getAction.PhaseCheckout:
		// Checkout is handled in Complete
	}
}

func (h *SimpleGetHandler) printEventLine(event getAction.Event) {
	switch event.Phase {
	case getAction.PhaseFetch:
		h.printFetchEvent(event)
	case getAction.PhaseSync:
		h.printSyncEvent(event)
	}
}

func (h *SimpleGetHandler) printFetchEvent(event getAction.Event) {
	if event.Type == getAction.EventCompleted {
		if event.NewRevision != "" {
			h.splog.Info("  %s fast-forwarded to %s",
				style.ColorBranchName(event.Branch, false),
				style.ColorDim(event.NewRevision))
		} else {
			h.splog.Info("  %s is up to date", style.ColorBranchName(event.Branch, false))
		}
	}
}

func (h *SimpleGetHandler) printSyncEvent(event getAction.Event) {
	if event.Type != getAction.EventCompleted {
		return
	}

	prInfo := h.formatPRInfo(event.PRNumber)

	if event.IsNew {
		h.splog.Info("  Synced %s%s from remote",
			style.ColorBranchName(event.Branch, false),
			prInfo)
	} else {
		h.splog.Info("  Updated %s%s from remote",
			style.ColorBranchName(event.Branch, false),
			prInfo)
	}
}

func (h *SimpleGetHandler) formatPRInfo(prNumber *int) string {
	if prNumber == nil {
		return ""
	}
	return fmt.Sprintf(" (PR #%d)", *prNumber)
}

// OnRestackStart implements RestackHandler for restack phase
func (h *SimpleGetHandler) OnRestackStart(_ int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.splog.Newline()
	h.splog.Info("📚 Restacking branches...")
}

// OnRestackBranch implements RestackHandler for restack phase
func (h *SimpleGetHandler) OnRestackBranch(branch string, result handlers.RestackResult, newRev string, prNumber *int) {
	h.mu.Lock()
	defer h.mu.Unlock()

	prInfo := h.formatPRInfo(prNumber)

	switch result {
	case handlers.RestackDone:
		h.splog.Info("  Restacked %s%s -> %s",
			style.ColorBranchName(branch, false),
			prInfo,
			style.ColorDim(newRev))
	case handlers.RestackUnneeded:
		h.splog.Info("  %s%s does not need restacking",
			style.ColorBranchName(branch, false),
			prInfo)
	case handlers.RestackConflict:
		h.splog.Warn("  Skipped %s%s (conflict)",
			style.ColorBranchName(branch, false),
			prInfo)
	}
}

// OnRestackComplete implements RestackHandler for restack phase
func (h *SimpleGetHandler) OnRestackComplete(restacked, skipped int, conflicts []string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if restacked == 0 && skipped == 0 {
		return // No restack summary needed if nothing happened
	}

	parts := []string{}
	if restacked > 0 {
		parts = append(parts, fmt.Sprintf("restacked %d", restacked))
	}
	if skipped > 0 {
		parts = append(parts, fmt.Sprintf("skipped %d (conflict)", skipped))
	}

	if len(parts) > 0 {
		h.splog.Info("  %s", strings.Join(parts, ", "))
	}

	if len(conflicts) > 0 {
		h.splog.Info("  Run %s to resolve and continue",
			style.ColorCyan(fmt.Sprintf("st restack %s", conflicts[0])))
	}
}
