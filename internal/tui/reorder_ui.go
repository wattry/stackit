package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"stackit.dev/stackit/internal/tui/keys"
	"stackit.dev/stackit/internal/tui/style"
)

// reorderModel is the bubbletea model for reordering branches
type reorderModel struct {
	branches  []string
	cursor    int
	confirmed bool
	canceled  bool
	keys      keys.ReorderKeyMap
}

// newReorderModel creates a new reorder TUI model
func newReorderModel(branches []string) reorderModel {
	return reorderModel{
		branches: branches,
		cursor:   0,
		keys:     keys.DefaultReorder,
	}
}

func (m reorderModel) Init() tea.Cmd {
	return nil
}

func (m reorderModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch {
		case key.Matches(msg, m.keys.Cancel):
			m.canceled = true
			return m, tea.Quit

		case key.Matches(msg, m.keys.Up):
			if len(m.branches) > 0 {
				if m.cursor > 0 {
					m.cursor--
				} else {
					m.cursor = len(m.branches) - 1 // Wrap to last
				}
			}

		case key.Matches(msg, m.keys.Down):
			if len(m.branches) > 0 {
				if m.cursor < len(m.branches)-1 {
					m.cursor++
				} else {
					m.cursor = 0 // Wrap to first
				}
			}

		case key.Matches(msg, m.keys.MoveUp):
			if m.cursor > 0 {
				// Swap with previous
				m.branches[m.cursor], m.branches[m.cursor-1] = m.branches[m.cursor-1], m.branches[m.cursor]
				m.cursor--
			}

		case key.Matches(msg, m.keys.MoveDown):
			if m.cursor < len(m.branches)-1 {
				// Swap with next
				m.branches[m.cursor], m.branches[m.cursor+1] = m.branches[m.cursor+1], m.branches[m.cursor]
				m.cursor++
			}

		case key.Matches(msg, m.keys.Select):
			m.confirmed = true
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m reorderModel) View() string {
	var b strings.Builder

	// Header with title and help (consistent with log view)
	title := "Reorder Branches"
	help := "'↑/k' '↓/j' navigate, 'K' 'J' move, 'enter' confirm, 'esc' cancel"
	header := style.ColorDim(fmt.Sprintf(" %s | %d branches | %s", title, len(m.branches), help))
	b.WriteString(header)
	b.WriteString("\n\n")

	// Branch list with cursor + background selection
	selectionStyle := style.Selection()
	cursorStyle := style.SelectionCursorStyle()
	branchStyle := style.BranchStyle(false, false, false)

	for i, branch := range m.branches {
		if i == m.cursor {
			// Selected: cursor symbol + background highlight
			cursor := cursorStyle.Render(style.SelectionCursor)
			branchText := selectionStyle.Render(branch)
			b.WriteString(fmt.Sprintf("%s%s\n", cursor, branchText))
		} else {
			// Unselected: padding + normal style
			b.WriteString(fmt.Sprintf("%s%s\n", style.SelectionPadding, branchStyle.Render(branch)))
		}
	}

	return b.String()
}

// NewReorderModel creates a tea.Model for a reorder prompt (used by stories/demo)
func NewReorderModel(branches []string) tea.Model {
	return newReorderModel(branches)
}

// RunReorderTUI runs the reorder TUI and returns the new order
func RunReorderTUI(branches []string) ([]string, error) {
	if !IsTTY() {
		return nil, fmt.Errorf("RunReorderTUI called in non-interactive mode")
	}
	m := newReorderModel(branches)
	p := tea.NewProgram(m, tea.WithInput(os.Stdin), tea.WithOutput(os.Stdout))

	finalModel, err := p.Run()
	if err != nil {
		return nil, err
	}

	res := finalModel.(reorderModel)
	if res.canceled {
		return nil, fmt.Errorf("reorder canceled")
	}

	return res.branches, nil
}
