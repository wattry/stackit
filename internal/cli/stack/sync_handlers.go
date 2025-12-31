package stack

import (
	"fmt"
	"os"
	"strings"
	stdsync "sync"

	tea "github.com/charmbracelet/bubbletea"

	syncAction "stackit.dev/stackit/internal/actions/sync"
	"stackit.dev/stackit/internal/tui"
	syncComponent "stackit.dev/stackit/internal/tui/components/sync"
	"stackit.dev/stackit/internal/tui/style"
)

// NewSyncHandler creates the appropriate handler based on TTY availability
func NewSyncHandler(splog *tui.Splog) syncAction.Handler {
	if tui.IsTTY() {
		return NewInteractiveSyncHandler(splog)
	}
	return NewSimpleSyncHandler(splog)
}

// SimpleSyncHandler provides streaming text output for non-TTY environments
type SimpleSyncHandler struct {
	splog        *tui.Splog
	currentPhase syncAction.Phase
	mu           stdsync.Mutex
	totalOps     int
	currentOp    int
}

// NewSimpleSyncHandler creates a new SimpleSyncHandler
func NewSimpleSyncHandler(splog *tui.Splog) *SimpleSyncHandler {
	return &SimpleSyncHandler{
		splog: splog,
	}
}

// Start is called at the beginning of sync
func (h *SimpleSyncHandler) Start(totalOps int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.totalOps = totalOps
	h.currentOp = 0
}

// EmitEvent handles progress updates
func (h *SimpleSyncHandler) EmitEvent(event syncAction.Event) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Handle phase transitions
	if event.Type == syncAction.EventStarted && event.Phase != h.currentPhase {
		h.currentPhase = event.Phase
		h.printPhaseHeader(event.Phase)
		return
	}

	// Handle progress events
	h.currentOp++
	h.printEventLine(event)
}

// Complete is called when sync finishes
func (h *SimpleSyncHandler) Complete(summary syncAction.Summary) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Print blank line before summary
	h.splog.Newline()

	// Handle up-to-date case
	if summary.UpToDate {
		h.splog.Info("✨ Everything is up to date!")
		return
	}

	// Print summary
	h.printSummary(summary)
}

func (h *SimpleSyncHandler) printPhaseHeader(phase syncAction.Phase) {
	// Add spacing between phases (but not before first phase)
	if h.currentPhase != "" {
		h.splog.Newline()
	}

	switch phase {
	case syncAction.PhaseTrunk:
		h.splog.Info("📥 Pulling from remote...")
	case syncAction.PhaseGitHub:
		h.splog.Info("🔄 Fetching PR info from GitHub...")
	case syncAction.PhaseClean:
		h.splog.Info("🧹 Cleaning branches...")
	case syncAction.PhaseRestack:
		h.splog.Info("📚 Restacking branches...")
	}
}

func (h *SimpleSyncHandler) printEventLine(event syncAction.Event) {
	switch event.Phase {
	case syncAction.PhaseTrunk:
		h.printTrunkEvent(event)
	case syncAction.PhaseGitHub:
		h.printGitHubEvent(event)
	case syncAction.PhaseClean:
		h.printCleanEvent(event)
	case syncAction.PhaseRestack:
		h.printRestackEvent(event)
	}
}

func (h *SimpleSyncHandler) printTrunkEvent(event syncAction.Event) {
	if event.Type == syncAction.EventCompleted {
		if event.NewRevision != "" {
			h.splog.Info("  %s fast-forwarded to %s",
				style.ColorBranchName(event.Branch, false),
				style.ColorDim(event.NewRevision))
		} else {
			h.splog.Info("  %s is up to date", style.ColorBranchName(event.Branch, false))
		}
	}
}

func (h *SimpleSyncHandler) printGitHubEvent(event syncAction.Event) {
	if event.Type == syncAction.EventCompleted && event.Message != "" {
		h.splog.Info("  %s", event.Message)
	}
}

func (h *SimpleSyncHandler) printCleanEvent(event syncAction.Event) {
	if event.Type == syncAction.EventCompleted && event.Branch != "" {
		prInfo := ""
		if event.PRNumber != nil {
			prInfo = fmt.Sprintf(" (PR #%d)", *event.PRNumber)
		}
		h.splog.Info("  Deleted %s%s %s",
			style.ColorBranchName(event.Branch, false),
			prInfo,
			style.ColorDim(event.Message))
	}
}

func (h *SimpleSyncHandler) printRestackEvent(event syncAction.Event) {
	if event.Branch == "" {
		return
	}

	prInfo := ""
	if event.PRNumber != nil {
		prInfo = fmt.Sprintf(" (PR #%d)", *event.PRNumber)
	}

	switch event.Type {
	case syncAction.EventCompleted:
		if event.NewRevision != "" {
			h.splog.Info("  Restacked %s%s -> %s",
				style.ColorBranchName(event.Branch, false),
				prInfo,
				style.ColorDim(event.NewRevision))
		} else {
			h.splog.Info("  %s%s does not need restacking",
				style.ColorBranchName(event.Branch, false),
				prInfo)
		}
	case syncAction.EventSkipped:
		if event.Conflict {
			h.splog.Warn("Skipped %s%s (conflict)",
				style.ColorBranchName(event.Branch, false),
				prInfo)
		} else {
			h.splog.Info("  Skipped %s%s %s",
				style.ColorBranchName(event.Branch, false),
				prInfo,
				style.ColorDim(event.Message))
		}
	}
}

func (h *SimpleSyncHandler) printSummary(summary syncAction.Summary) {
	parts := []string{}

	if summary.TrunkUpdated {
		parts = append(parts, "pulled trunk")
	}
	if summary.BranchesSynced > 0 {
		parts = append(parts, fmt.Sprintf("synced %d branch%s", summary.BranchesSynced, pluralS(summary.BranchesSynced)))
	}
	if summary.BranchesRestacked > 0 {
		parts = append(parts, fmt.Sprintf("restacked %d", summary.BranchesRestacked))
	}
	if summary.BranchesDeleted > 0 {
		parts = append(parts, fmt.Sprintf("deleted %d", summary.BranchesDeleted))
	}
	if summary.BranchesSkipped > 0 {
		parts = append(parts, fmt.Sprintf("skipped %d (conflict)", summary.BranchesSkipped))
	}

	if len(parts) > 0 {
		h.splog.Info("✅ Summary: %s", strings.Join(parts, ", "))
	}

	// Print actionable advice for conflicts
	if len(summary.ConflictBranches) > 0 {
		h.splog.Info("  Run %s to resolve and continue",
			style.ColorCyan(fmt.Sprintf("st restack %s", summary.ConflictBranches[0])))
	}
}

func pluralS(count int) string {
	if count == 1 {
		return ""
	}
	return "es"
}

// InteractiveSyncHandler provides bubbletea TUI for TTY environments
type InteractiveSyncHandler struct {
	splog        *tui.Splog
	program      *tea.Program
	model        *syncComponent.Model
	mu           stdsync.Mutex
	totalOps     int
	completedOps int
	currentPhase syncAction.Phase
}

// NewInteractiveSyncHandler creates a new InteractiveSyncHandler
func NewInteractiveSyncHandler(splog *tui.Splog) *InteractiveSyncHandler {
	return &InteractiveSyncHandler{
		splog: splog,
	}
}

// Start is called at the beginning of sync
func (h *InteractiveSyncHandler) Start(totalOps int) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.totalOps = totalOps
	h.completedOps = 0

	// Initialize model
	h.model = syncComponent.NewModel(totalOps)

	// Quiet the splog so it doesn't interfere with the TUI
	h.splog.SetQuiet(true)

	// Start bubbletea program
	h.program = tea.NewProgram(h.model, tea.WithInput(os.Stdin), tea.WithOutput(os.Stdout))

	// Run program in background
	go func() {
		if _, err := h.program.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error running sync TUI: %v\n", err)
		}
	}()
}

// EmitEvent handles progress updates
func (h *InteractiveSyncHandler) EmitEvent(event syncAction.Event) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.program == nil {
		return
	}

	// Handle phase transitions
	if event.Type == syncAction.EventStarted && event.Phase != h.currentPhase {
		h.currentPhase = event.Phase
		h.program.Send(syncComponent.PhaseStartMsg{
			Phase: syncComponent.Phase(event.Phase),
		})
		return
	}

	// Build detail message
	detail := h.formatEventDetail(event)
	if detail != "" {
		h.program.Send(syncComponent.PhaseDetailMsg{
			Phase:   syncComponent.Phase(event.Phase),
			Message: detail,
		})
	}

	// Update progress
	h.completedOps++
	h.program.Send(syncComponent.ProgressTickMsg{
		Completed: h.completedOps,
		Total:     h.totalOps,
	})
}

// formatEventDetail formats an event into a detail string
func (h *InteractiveSyncHandler) formatEventDetail(event syncAction.Event) string {
	switch event.Phase {
	case syncAction.PhaseTrunk:
		if event.Type == syncAction.EventCompleted {
			if event.NewRevision != "" {
				return fmt.Sprintf("%s fast-forwarded to %s", event.Branch, event.NewRevision)
			}
			return fmt.Sprintf("%s is up to date", event.Branch)
		}
	case syncAction.PhaseGitHub:
		if event.Type == syncAction.EventCompleted && event.Message != "" {
			return event.Message
		}
	case syncAction.PhaseClean:
		if event.Type == syncAction.EventCompleted && event.Branch != "" {
			prInfo := ""
			if event.PRNumber != nil {
				prInfo = fmt.Sprintf(" (PR #%d)", *event.PRNumber)
			}
			return fmt.Sprintf("Deleted %s%s %s", event.Branch, prInfo, event.Message)
		}
	case syncAction.PhaseRestack:
		if event.Branch == "" {
			return ""
		}
		prInfo := ""
		if event.PRNumber != nil {
			prInfo = fmt.Sprintf(" (PR #%d)", *event.PRNumber)
		}
		switch event.Type {
		case syncAction.EventCompleted:
			if event.NewRevision != "" {
				return fmt.Sprintf("Restacked %s%s -> %s", event.Branch, prInfo, event.NewRevision)
			}
			return fmt.Sprintf("%s%s does not need restacking", event.Branch, prInfo)
		case syncAction.EventSkipped:
			if event.Conflict {
				return fmt.Sprintf("⚠️ Skipped %s%s (conflict)", event.Branch, prInfo)
			}
			return fmt.Sprintf("Skipped %s%s %s", event.Branch, prInfo, event.Message)
		}
	}
	return ""
}

// Complete is called when sync finishes
func (h *InteractiveSyncHandler) Complete(summary syncAction.Summary) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.program == nil {
		return
	}

	// Build summary message
	summaryMsg := h.formatSummary(summary)

	// Send complete message and wait for program to quit
	h.program.Send(syncComponent.CompleteMsg{Summary: summaryMsg})
	h.program.Wait()
	h.program = nil

	// Restore splog
	h.splog.SetQuiet(false)
}

// formatSummary formats the sync summary
func (h *InteractiveSyncHandler) formatSummary(summary syncAction.Summary) string {
	if summary.UpToDate {
		return "✨ Everything is up to date!"
	}

	parts := []string{}

	if summary.TrunkUpdated {
		parts = append(parts, "pulled trunk")
	}
	if summary.BranchesSynced > 0 {
		parts = append(parts, fmt.Sprintf("synced %d branch%s", summary.BranchesSynced, pluralS(summary.BranchesSynced)))
	}
	if summary.BranchesRestacked > 0 {
		parts = append(parts, fmt.Sprintf("restacked %d", summary.BranchesRestacked))
	}
	if summary.BranchesDeleted > 0 {
		parts = append(parts, fmt.Sprintf("deleted %d", summary.BranchesDeleted))
	}
	if summary.BranchesSkipped > 0 {
		parts = append(parts, fmt.Sprintf("skipped %d (conflict)", summary.BranchesSkipped))
	}

	result := ""
	if len(parts) > 0 {
		result = "✅ Summary: " + strings.Join(parts, ", ")
	}

	// Add actionable advice for conflicts
	if len(summary.ConflictBranches) > 0 {
		result += fmt.Sprintf("\n   Run 'st restack %s' to resolve and continue", summary.ConflictBranches[0])
	}

	return result
}
