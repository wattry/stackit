package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/errors"
	"stackit.dev/stackit/internal/tui/style"
)

// Direction represents where to place a new branch
type Direction string

// Direction constants for split operations.
const (
	DirectionBelow Direction = "below"
	DirectionAbove Direction = "above"
)

// directionSelectKeyMap defines the keybindings for direction selection
type directionSelectKeyMap struct {
	Up     key.Binding
	Down   key.Binding
	Submit key.Binding
	Cancel key.Binding
}

func (k directionSelectKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Submit, k.Cancel}
}

func (k directionSelectKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.Up, k.Down, k.Submit, k.Cancel}}
}

var defaultDirectionKeys = directionSelectKeyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "above"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "below"),
	),
	Submit: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	),
	Cancel: key.NewBinding(
		key.WithKeys("ctrl+c", "esc", "q"),
		key.WithHelp("esc", "cancel"),
	),
}

// DirectionSelectModel is a bubbletea model for selecting split direction
type DirectionSelectModel struct {
	currentBranch string
	parentBranch  string
	children      []string
	direction     Direction
	done          bool
	ready         bool
	err           error
	help          help.Model
	keys          directionSelectKeyMap

	// Stack path from trunk to current branch (in order)
	stackPath []string
}

// NewDirectionSelectModel creates a model for selecting split direction
func NewDirectionSelectModel(eng engine.BranchReader, currentBranch, parentBranch string, children []string) *DirectionSelectModel {
	// Build the path from trunk to current branch
	graph := engine.BuildStackGraph(eng, engine.SortStrategySmart, nil)

	var stackPath []string
	current := currentBranch
	for current != "" && current != graph.Trunk {
		stackPath = append([]string{current}, stackPath...)
		node := graph.Nodes[current]
		if node == nil {
			break
		}
		current = graph.Parent(node.Branch)
	}
	// Add trunk at the beginning
	stackPath = append([]string{graph.Trunk}, stackPath...)

	return &DirectionSelectModel{
		currentBranch: currentBranch,
		parentBranch:  parentBranch,
		children:      children,
		direction:     DirectionBelow, // Default to below
		help:          help.New(),
		keys:          defaultDirectionKeys,
		stackPath:     stackPath,
	}
}

// Init implements tea.Model.
func (m *DirectionSelectModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m *DirectionSelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.ready = true
		return m, nil
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Up):
			m.direction = DirectionAbove
		case key.Matches(msg, m.keys.Down):
			m.direction = DirectionBelow
		case key.Matches(msg, m.keys.Submit):
			m.done = true
			return m, tea.Quit
		case key.Matches(msg, m.keys.Cancel):
			m.err = errors.ErrCanceled
			m.done = true
			return m, tea.Quit
		}
	}
	return m, nil
}

// View implements tea.Model.
func (m *DirectionSelectModel) View() string {
	if m.done {
		return ""
	}

	if !m.ready {
		return "Loading..."
	}

	var sb strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	sb.WriteString(titleStyle.Render("Where should the new branch be placed?"))
	sb.WriteString("\n\n")

	// Direction options at the top
	sb.WriteString(m.renderOptions())
	sb.WriteString("\n")

	// Render the stack tree with insertion point
	sb.WriteString(m.renderStackTree())
	sb.WriteString("\n\n")

	// Help
	sb.WriteString(m.help.View(m.keys))

	return lipgloss.NewStyle().Margin(1, 2).Render(sb.String())
}

// renderOptions renders the direction selection options
func (m *DirectionSelectModel) renderOptions() string {
	var sb strings.Builder

	selectedStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	normalStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))

	// Above option (shown first - visually at top of tree)
	if m.direction == DirectionAbove {
		sb.WriteString(selectedStyle.Render("▸ Above"))
		sb.WriteString(normalStyle.Render(" - Insert as child of current"))
	} else {
		sb.WriteString(normalStyle.Render("  Above - Insert as child of current"))
	}
	sb.WriteString("\n")

	// Below option (default)
	if m.direction == DirectionBelow {
		sb.WriteString(selectedStyle.Render("▸ Below"))
		sb.WriteString(normalStyle.Render(" - Insert between parent and current"))
	} else {
		sb.WriteString(normalStyle.Render("  Below - Insert between parent and current"))
	}
	sb.WriteString("\n")

	return sb.String()
}

// renderStackTree renders only the current stack with insertion point indicator.
// Tree is rendered with main at the bottom (like st log).
func (m *DirectionSelectModel) renderStackTree() string {
	// Pre-allocate: stack path + potential insertion point + potential children
	lines := make([]string, 0, len(m.stackPath)+1+len(m.children))

	insertStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10")) // green
	currentStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("243"))

	// Calculate max depth for prefix calculation (we render bottom-up)
	maxDepth := len(m.stackPath) - 1

	// Render in reverse order (current branch first, trunk last)
	for i := len(m.stackPath) - 1; i >= 0; i-- {
		branch := m.stackPath[i]
		isCurrent := branch == m.currentBranch
		isParentOfCurrent := i < len(m.stackPath)-1 && m.stackPath[i+1] == m.currentBranch

		// Build prefix based on reversed depth
		depth := maxDepth - i
		prefix := strings.Repeat("│ ", depth)

		// Determine symbol and style
		symbol := "◯"
		branchStyle := dimStyle
		if isCurrent {
			symbol = "◉"
			branchStyle = currentStyle
		}

		// If direction is "above" and this is the current branch, show children and insertion point first
		if m.direction == DirectionAbove && isCurrent {
			// Show children that will be re-parented (at top)
			if len(m.children) > 0 {
				childPrefix := strings.Repeat("│ ", depth+2)
				for _, child := range m.children {
					lines = append(lines, fmt.Sprintf("%s%s %s", childPrefix, "◯", dimStyle.Render(child+" (re-parented)")))
				}
			}

			// Show insertion point
			insertPrefix := strings.Repeat("│ ", depth+1)
			lines = append(lines, fmt.Sprintf("%s%s %s", insertPrefix, "◆", insertStyle.Render("[new branch]")))
		}

		// Render the branch line
		line := fmt.Sprintf("%s%s %s", prefix, symbol, branchStyle.Render(branch))
		if isCurrent {
			line += style.ColorDim(" ← current")
		}
		lines = append(lines, line)

		// If direction is "below" and this is the parent of current, show insertion point after parent
		if m.direction == DirectionBelow && isParentOfCurrent {
			insertPrefix := strings.Repeat("│ ", depth+1)
			lines = append(lines, fmt.Sprintf("%s%s %s", insertPrefix, "◆", insertStyle.Render("[new branch]")))
		}
	}

	return strings.Join(lines, "\n")
}

// Direction returns the selected direction
func (m *DirectionSelectModel) Direction() Direction {
	return m.direction
}

// Err returns any error that occurred
func (m *DirectionSelectModel) Err() error {
	return m.err
}

// PromptDirectionSelect shows an interactive direction selector and returns the chosen direction
func PromptDirectionSelect(eng engine.BranchReader, currentBranch, parentBranch string, children []string) (Direction, error) {
	if err := CheckInteractiveAllowed(); err != nil {
		return "", err
	}

	m := NewDirectionSelectModel(eng, currentBranch, parentBranch, children)

	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithInput(os.Stdin), tea.WithOutput(os.Stdout))
	model, err := p.Run()
	if err != nil {
		return "", err
	}

	if finalModel, ok := model.(*DirectionSelectModel); ok {
		if finalModel.Err() != nil {
			return "", finalModel.Err()
		}
		return finalModel.Direction(), nil
	}

	return "", fmt.Errorf("unexpected model type")
}
