// Package foreach provides a TUI component for displaying the progress of foreach command execution.
package foreach

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"stackit.dev/stackit/internal/tui/components/tree"
)

// Model is the bubbletea model for foreach progress
type Model struct {
	Items         []Item
	Renderer      *tree.StackTreeRenderer
	RootBranch    string
	Spinner       spinner.Model
	Done          bool
	Styles        Styles
	GlobalMessage string
	Command       string
}

// ProgressUpdateMsg is sent to update the status of a specific branch execution
type ProgressUpdateMsg struct {
	BranchName string
	Status     string
	Output     string
	Err        error
}

// StartExecutionMsg is sent when the execution phase begins
type StartExecutionMsg struct {
	Items []Item
}

// GlobalMessageMsg is sent to display a global message (e.g., "Running...")
type GlobalMessageMsg string

// ProgressCompleteMsg is sent when all executions are finished
type ProgressCompleteMsg struct{}

// NewModel creates a new foreach model
func NewModel(items []Item) *Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = DefaultStyles().SpinnerStyle

	return &Model{
		Items:   items,
		Spinner: s,
		Styles:  DefaultStyles(),
	}
}

// Init initializes the model.
func (m *Model) Init() tea.Cmd {
	return m.Spinner.Tick
}

// Update handles messages and updates the model.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			return m, tea.Quit
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.Spinner, cmd = m.Spinner.Update(msg)
		return m, cmd

	case StartExecutionMsg:
		// Update status for items that are in msg.Items
		for _, newItem := range msg.Items {
			found := false
			for i, item := range m.Items {
				if item.BranchName == newItem.BranchName {
					m.Items[i].Status = newItem.Status
					found = true
					break
				}
			}
			if !found {
				m.Items = append(m.Items, newItem)
			}
		}
		return m, nil

	case GlobalMessageMsg:
		m.GlobalMessage = string(msg)
		return m, nil

	case ProgressUpdateMsg:
		for i, item := range m.Items {
			if item.BranchName == msg.BranchName {
				m.Items[i].Status = msg.Status
				m.Items[i].Output = msg.Output
				m.Items[i].Error = msg.Err
				break
			}
		}
		return m, m.Spinner.Tick

	case ProgressCompleteMsg:
		m.Done = true
		return m, tea.Quit
	}

	return m, nil
}

// View renders the model as a string.
func (m *Model) View() string {
	var b strings.Builder

	if m.Command != "" {
		b.WriteString(m.Styles.DimStyle.Render("Command: " + m.Command))
		b.WriteString("\n\n")
	}

	if m.GlobalMessage != "" {
		b.WriteString(m.Styles.DimStyle.Render(m.GlobalMessage))
		b.WriteString("\n\n")
	}

	if m.Renderer != nil {
		// Update annotations based on items
		for _, item := range m.Items {
			ann := m.Renderer.Annotations[item.BranchName]

			// Update custom label for status
			switch item.Status {
			case StatusRunning:
				ann.CustomLabel = m.Styles.SpinnerStyle.Render(m.Spinner.View() + " running...")
			case StatusDone:
				ann.CustomLabel = m.Styles.DoneStyle.Render("✓")
				if item.Output != "" {
					// Show truncated output
					output := strings.TrimSpace(item.Output)
					if len(output) > 50 {
						output = output[:47] + "..."
					}
					ann.CustomLabel += " " + m.Styles.OutputStyle.Render(output)
				}
			case StatusError:
				ann.CustomLabel = m.Styles.ErrorStyle.Render("✗")
				if item.Error != nil {
					errMsg := item.Error.Error()
					if len(errMsg) > 50 {
						errMsg = errMsg[:47] + "..."
					}
					ann.CustomLabel += " " + m.Styles.ErrorStyle.Render(errMsg)
				}
			}
			m.Renderer.SetAnnotation(item.BranchName, ann)
		}

		lines := m.Renderer.RenderStack(m.RootBranch, tree.RenderOptions{
			HideStats: true,
		})
		b.WriteString(strings.Join(lines, "\n"))
	} else {
		// Fallback to list view if no renderer
		for i, item := range m.Items {
			var icon string
			var status string

			switch item.Status {
			case StatusPending, "":
				icon = m.Styles.DimStyle.Render("○")
				status = m.Styles.DimStyle.Render("pending")
			case StatusRunning:
				icon = m.Spinner.View()
				status = m.Styles.SpinnerStyle.Render("running...")
			case StatusDone:
				icon = m.Styles.DoneStyle.Render("✓")
				status = m.Styles.DoneStyle.Render("done")
			case StatusError:
				icon = m.Styles.ErrorStyle.Render("✗")
				status = m.Styles.ErrorStyle.Render("failed")
			}

			branchName := m.Styles.BranchStyle.Render(item.BranchName)
			line := fmt.Sprintf("  %s %s %s", icon, branchName, status)

			if item.Status == StatusDone && item.Output != "" {
				output := strings.TrimSpace(item.Output)
				if len(output) > 50 {
					output = output[:47] + "..."
				}
				line += " " + m.Styles.OutputStyle.Render(output)
			}
			if item.Status == StatusError && item.Error != nil {
				errMsg := item.Error.Error()
				if len(errMsg) > 50 {
					errMsg = errMsg[:47] + "..."
				}
				line += " " + m.Styles.ErrorStyle.Render(errMsg)
			}

			b.WriteString(line)
			if i < len(m.Items)-1 {
				b.WriteString("\n")
			}
		}
	}

	if m.Done {
		completed := 0
		failed := 0
		for _, item := range m.Items {
			if item.Status == StatusDone {
				completed++
			} else if item.Status == StatusError {
				failed++
			}
		}
		if failed > 0 {
			b.WriteString("\n\n")
			b.WriteString(m.Styles.ErrorStyle.Render(fmt.Sprintf("Completed: %d, Failed: %d", completed, failed)))
		}
	}

	b.WriteString("\n")
	return b.String()
}
