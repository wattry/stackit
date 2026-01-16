package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"stackit.dev/stackit/internal/errors"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/tui/style"
)

// HunkItem represents a single hunk in the selector
type HunkItem struct {
	Hunk       git.Hunk
	Selected   bool
	Expanded   bool // Manual expand/collapse override
	Splittable bool // Can this hunk be split further?
}

// FileGroup groups hunks by file for rendering
type FileGroup struct {
	File   string
	Hunks  []int // Indices into the items slice
	Binary bool  // Is this a binary file?
}

// hunkSelectorKeyMap defines the keybindings for hunk selection
type hunkSelectorKeyMap struct {
	Up       key.Binding
	Down     key.Binding
	Toggle   key.Binding
	Split    key.Binding
	All      key.Binding
	None     key.Binding
	Expand   key.Binding
	FileAll  key.Binding
	Submit   key.Binding
	Cancel   key.Binding
	PageUp   key.Binding
	PageDown key.Binding
}

func (k hunkSelectorKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Toggle, k.Split, k.All, k.None, k.Submit, k.Cancel}
}

func (k hunkSelectorKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.PageUp, k.PageDown},
		{k.Toggle, k.Split, k.All, k.None},
		{k.Expand, k.FileAll},
		{k.Submit, k.Cancel},
	}
}

var defaultHunkSelectorKeys = hunkSelectorKeyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	Toggle: key.NewBinding(
		key.WithKeys(" "),
		key.WithHelp("space", "toggle"),
	),
	Split: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "split"),
	),
	All: key.NewBinding(
		key.WithKeys("a"),
		key.WithHelp("a", "all"),
	),
	None: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "none"),
	),
	Expand: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "expand"),
	),
	FileAll: key.NewBinding(
		key.WithKeys("f"),
		key.WithHelp("f", "file toggle"),
	),
	Submit: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "done"),
	),
	Cancel: key.NewBinding(
		key.WithKeys("ctrl+c", "esc", "q"),
		key.WithHelp("esc", "cancel"),
	),
	PageUp: key.NewBinding(
		key.WithKeys("pgup", "ctrl+u"),
		key.WithHelp("pgup", "page up"),
	),
	PageDown: key.NewBinding(
		key.WithKeys("pgdown", "ctrl+d"),
		key.WithHelp("pgdn", "page down"),
	),
}

// HunkSelectorModel is a bubbletea model for selecting hunks
type HunkSelectorModel struct {
	items      []HunkItem
	fileGroups []FileGroup
	cursor     int
	viewport   viewport.Model
	help       help.Model
	keys       hunkSelectorKeyMap
	done       bool
	err        error
	width      int
	height     int
	ready      bool

	// Preview settings
	previewLines int // Number of lines to show in collapsed preview
}

// NewHunkSelectorModel creates a new hunk selector model
func NewHunkSelectorModel(hunks []git.Hunk) *HunkSelectorModel {
	items := make([]HunkItem, len(hunks))
	fileMap := make(map[string][]int)
	fileOrder := make([]string, 0)

	for i, h := range hunks {
		items[i] = HunkItem{
			Hunk:       h,
			Selected:   false,
			Expanded:   false,
			Splittable: git.CanSplitHunk(h),
		}
		if _, exists := fileMap[h.File]; !exists {
			fileOrder = append(fileOrder, h.File)
		}
		fileMap[h.File] = append(fileMap[h.File], i)
	}

	fileGroups := make([]FileGroup, len(fileOrder))
	for i, file := range fileOrder {
		fileGroups[i] = FileGroup{
			File:   file,
			Hunks:  fileMap[file],
			Binary: false,
		}
	}

	return &HunkSelectorModel{
		items:        items,
		fileGroups:   fileGroups,
		cursor:       0,
		help:         help.New(),
		keys:         defaultHunkSelectorKeys,
		previewLines: 4,
	}
}

// Init implements tea.Model
func (m *HunkSelectorModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (m *HunkSelectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		headerHeight := 3 // Title + summary line + blank
		footerHeight := 4 // Help + padding
		m.viewport = viewport.New(msg.Width, msg.Height-headerHeight-footerHeight)
		m.viewport.YPosition = headerHeight
		m.ready = true
		m.updateViewportContent()
		return m, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Cancel):
			m.err = errors.ErrCanceled
			m.done = true
			return m, tea.Quit

		case key.Matches(msg, m.keys.Submit):
			m.done = true
			return m, tea.Quit

		case key.Matches(msg, m.keys.Up):
			if m.cursor > 0 {
				m.cursor--
				m.updateViewportContent()
				m.ensureCursorVisible()
			}

		case key.Matches(msg, m.keys.Down):
			if m.cursor < len(m.items)-1 {
				m.cursor++
				m.updateViewportContent()
				m.ensureCursorVisible()
			}

		case key.Matches(msg, m.keys.Toggle):
			if m.cursor >= 0 && m.cursor < len(m.items) {
				m.items[m.cursor].Selected = !m.items[m.cursor].Selected
				m.updateViewportContent()
			}

		case key.Matches(msg, m.keys.Split):
			if m.cursor >= 0 && m.cursor < len(m.items) && m.items[m.cursor].Splittable {
				m.splitCurrentHunk()
				m.updateViewportContent()
			}

		case key.Matches(msg, m.keys.All):
			for i := range m.items {
				m.items[i].Selected = true
			}
			m.updateViewportContent()

		case key.Matches(msg, m.keys.None):
			for i := range m.items {
				m.items[i].Selected = false
			}
			m.updateViewportContent()

		case key.Matches(msg, m.keys.Expand):
			if m.cursor >= 0 && m.cursor < len(m.items) {
				m.items[m.cursor].Expanded = !m.items[m.cursor].Expanded
				m.updateViewportContent()
			}

		case key.Matches(msg, m.keys.FileAll):
			m.toggleFileSelection()
			m.updateViewportContent()

		case key.Matches(msg, m.keys.PageUp):
			m.viewport.SetYOffset(m.viewport.YOffset - m.viewport.Height/2)
			return m, nil

		case key.Matches(msg, m.keys.PageDown):
			m.viewport.SetYOffset(m.viewport.YOffset + m.viewport.Height/2)
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// View implements tea.Model
func (m *HunkSelectorModel) View() string {
	if m.done {
		return ""
	}

	if !m.ready {
		return "Initializing..."
	}

	var sb strings.Builder

	// Header
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	sb.WriteString(titleStyle.Render(m.renderHeader()))
	sb.WriteString("\n\n")

	// Viewport with hunks
	sb.WriteString(m.viewport.View())
	sb.WriteString("\n")

	// Help
	sb.WriteString(m.help.View(m.keys))

	return lipgloss.NewStyle().Margin(1, 2).Render(sb.String())
}

// renderHeader creates the header line with selection info
func (m *HunkSelectorModel) renderHeader() string {
	selected := 0
	for _, item := range m.items {
		if item.Selected {
			selected++
		}
	}

	fileCount := len(m.fileGroups)
	return fmt.Sprintf("Select hunks to stage (%d of %d hunks, %d files)", selected, len(m.items), fileCount)
}

// updateViewportContent renders all hunks into the viewport
func (m *HunkSelectorModel) updateViewportContent() {
	if !m.ready {
		return
	}

	var sb strings.Builder

	// Styles
	fileStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	unselectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	cursorStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	addStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	removeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	contextStyle := style.DimStyle()
	splittableStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("3"))

	itemIndex := 0
	for _, fg := range m.fileGroups {
		// File header
		sb.WriteString(fileStyle.Render(fg.File))
		sb.WriteString("\n")

		for _, hunkIdx := range fg.Hunks {
			item := m.items[hunkIdx]
			isCursor := hunkIdx == m.cursor

			// Checkbox and cursor
			var checkbox string
			if item.Selected {
				checkbox = selectedStyle.Render("[x]")
			} else {
				checkbox = unselectedStyle.Render("[ ]")
			}

			cursor := "  "
			if isCursor {
				cursor = cursorStyle.Render("> ")
			}

			// Hunk header line
			header := git.GetHunkHeader(item.Hunk)
			added, removed := git.CountHunkLines(item.Hunk)

			headerText := fmt.Sprintf("%s +%d -%d", header, added, removed)
			if item.Splittable {
				headerText += " " + splittableStyle.Render("[splittable]")
			}

			if isCursor {
				headerText = cursorStyle.Render(headerText)
			}

			sb.WriteString(fmt.Sprintf("%s%s %s\n", cursor, checkbox, headerText))

			// Show preview for current hunk or expanded hunks
			if isCursor || item.Expanded {
				preview, totalLines, hasMore := git.GetHunkPreview(item.Hunk, m.previewLines)
				if item.Expanded {
					// Show full content
					preview, _, _ = git.GetHunkPreview(item.Hunk, totalLines)
				}

				// Render diff lines with colors
				previewLines := strings.Split(preview, "\n")
				maxPreview := m.previewLines
				if item.Expanded {
					maxPreview = len(previewLines)
				}

				for i, line := range previewLines {
					if i >= maxPreview && !item.Expanded {
						break
					}

					var renderedLine string
					switch {
					case strings.HasPrefix(line, "+"):
						renderedLine = addStyle.Render(line)
					case strings.HasPrefix(line, "-"):
						renderedLine = removeStyle.Render(line)
					default:
						renderedLine = contextStyle.Render(line)
					}
					sb.WriteString("      " + renderedLine + "\n")
				}

				if hasMore && !item.Expanded {
					remaining := totalLines - m.previewLines
					sb.WriteString(fmt.Sprintf("      %s\n", contextStyle.Render(fmt.Sprintf("...%d more lines", remaining))))
				}
			}

			itemIndex++
		}
		sb.WriteString("\n")
	}

	m.viewport.SetContent(sb.String())
}

// ensureCursorVisible scrolls the viewport to keep cursor in view
func (m *HunkSelectorModel) ensureCursorVisible() {
	// Calculate approximate line position of cursor
	linePos := 0
	for _, fg := range m.fileGroups {
		linePos++ // File header
		for _, hunkIdx := range fg.Hunks {
			if hunkIdx == m.cursor {
				// Found cursor position
				if linePos < m.viewport.YOffset {
					m.viewport.SetYOffset(linePos)
				} else if linePos > m.viewport.YOffset+m.viewport.Height-5 {
					m.viewport.SetYOffset(linePos - m.viewport.Height + 5)
				}
				return
			}
			linePos++ // Hunk header line
			if hunkIdx == m.cursor || m.items[hunkIdx].Expanded {
				linePos += m.previewLines + 1 // Preview lines
			}
		}
		linePos++ // Blank line after file
	}
}

// splitCurrentHunk splits the current hunk if possible
func (m *HunkSelectorModel) splitCurrentHunk() {
	if m.cursor < 0 || m.cursor >= len(m.items) {
		return
	}

	item := m.items[m.cursor]
	if !item.Splittable {
		return
	}

	splitHunks, err := git.SplitHunk(item.Hunk)
	if err != nil || len(splitHunks) <= 1 {
		return
	}

	// Replace the current hunk with the split hunks
	newItems := make([]HunkItem, 0, len(m.items)+len(splitHunks)-1)
	newItems = append(newItems, m.items[:m.cursor]...)

	for _, h := range splitHunks {
		newItems = append(newItems, HunkItem{
			Hunk:       h,
			Selected:   item.Selected, // Preserve selection state
			Expanded:   false,
			Splittable: git.CanSplitHunk(h),
		})
	}

	newItems = append(newItems, m.items[m.cursor+1:]...)
	m.items = newItems

	// Rebuild file groups
	m.rebuildFileGroups()
}

// rebuildFileGroups rebuilds the file groups after a split
func (m *HunkSelectorModel) rebuildFileGroups() {
	fileMap := make(map[string][]int)
	fileOrder := make([]string, 0)

	for i, item := range m.items {
		if _, exists := fileMap[item.Hunk.File]; !exists {
			fileOrder = append(fileOrder, item.Hunk.File)
		}
		fileMap[item.Hunk.File] = append(fileMap[item.Hunk.File], i)
	}

	m.fileGroups = make([]FileGroup, len(fileOrder))
	for i, file := range fileOrder {
		m.fileGroups[i] = FileGroup{
			File:  file,
			Hunks: fileMap[file],
		}
	}
}

// toggleFileSelection toggles all hunks in the current file
func (m *HunkSelectorModel) toggleFileSelection() {
	if m.cursor < 0 || m.cursor >= len(m.items) {
		return
	}

	currentFile := m.items[m.cursor].Hunk.File

	// Find all hunks in this file and check if any are unselected
	anyUnselected := false
	for _, item := range m.items {
		if item.Hunk.File == currentFile && !item.Selected {
			anyUnselected = true
			break
		}
	}

	// Toggle: if any unselected, select all; otherwise deselect all
	for i := range m.items {
		if m.items[i].Hunk.File == currentFile {
			m.items[i].Selected = anyUnselected
		}
	}
}

// SelectedHunks returns the hunks that were selected
func (m *HunkSelectorModel) SelectedHunks() []git.Hunk {
	var selected []git.Hunk
	for _, item := range m.items {
		if item.Selected {
			selected = append(selected, item.Hunk)
		}
	}
	return selected
}

// Err returns any error that occurred
func (m *HunkSelectorModel) Err() error {
	return m.err
}

// PromptSelectHunks shows an interactive hunk selector and returns the selected hunks
func PromptSelectHunks(hunks []git.Hunk) ([]git.Hunk, error) {
	if err := CheckInteractiveAllowed(); err != nil {
		return nil, err
	}

	if len(hunks) == 0 {
		return nil, nil
	}

	m := NewHunkSelectorModel(hunks)

	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithInput(os.Stdin), tea.WithOutput(os.Stdout))
	model, err := p.Run()
	if err != nil {
		return nil, err
	}

	if finalModel, ok := model.(*HunkSelectorModel); ok {
		if finalModel.Err() != nil {
			return nil, finalModel.Err()
		}
		return finalModel.SelectedHunks(), nil
	}

	return nil, fmt.Errorf("unexpected model type")
}
