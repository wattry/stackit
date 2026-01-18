package split

import (
	"github.com/charmbracelet/bubbles/key"

	"stackit.dev/stackit/internal/tui/core"
)

// typeSelectKeys defines key bindings for type selection
type typeSelectKeys struct {
	Up     key.Binding
	Down   key.Binding
	Select key.Binding
	Cancel key.Binding
}

func (k typeSelectKeys) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Select, k.Cancel}
}

func (k typeSelectKeys) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.Up, k.Down, k.Select, k.Cancel}}
}

var defaultTypeSelectKeys = typeSelectKeys{
	Up: key.NewBinding(
		key.WithKeys(core.KeyUp, "k"),
		key.WithHelp("up/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys(core.KeyDown, "j"),
		key.WithHelp("down/j", "down"),
	),
	Select: key.NewBinding(
		key.WithKeys(core.KeyEnter),
		key.WithHelp("enter", "select"),
	),
	Cancel: key.NewBinding(
		key.WithKeys(core.KeyCtrlC, core.KeyEsc),
		key.WithHelp("esc", "cancel"),
	),
}

// directionSelectKeys defines key bindings for direction selection
type directionSelectKeys struct {
	Up     key.Binding
	Down   key.Binding
	Select key.Binding
	Cancel key.Binding
}

func (k directionSelectKeys) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Select, k.Cancel}
}

func (k directionSelectKeys) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.Up, k.Down, k.Select, k.Cancel}}
}

var defaultDirectionSelectKeys = directionSelectKeys{
	Up: key.NewBinding(
		key.WithKeys(core.KeyUp, "k"),
		key.WithHelp("up/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys(core.KeyDown, "j"),
		key.WithHelp("down/j", "down"),
	),
	Select: key.NewBinding(
		key.WithKeys(core.KeyEnter),
		key.WithHelp("enter", "select"),
	),
	Cancel: key.NewBinding(
		key.WithKeys(core.KeyCtrlC, core.KeyEsc),
		key.WithHelp("esc", "cancel"),
	),
}

// branchNameKeys defines key bindings for branch name input
type branchNameKeys struct {
	Submit key.Binding
	Cancel key.Binding
}

func (k branchNameKeys) ShortHelp() []key.Binding {
	return []key.Binding{k.Submit, k.Cancel}
}

func (k branchNameKeys) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.Submit, k.Cancel}}
}

var defaultBranchNameKeys = branchNameKeys{
	Submit: key.NewBinding(
		key.WithKeys(core.KeyEnter),
		key.WithHelp("enter", "submit"),
	),
	Cancel: key.NewBinding(
		key.WithKeys(core.KeyCtrlC, core.KeyEsc),
		key.WithHelp("esc", "cancel"),
	),
}

// confirmKeys defines key bindings for yes/no confirmations
type confirmKeys struct {
	Yes    key.Binding
	No     key.Binding
	Cancel key.Binding
}

func (k confirmKeys) ShortHelp() []key.Binding {
	return []key.Binding{k.Yes, k.No, k.Cancel}
}

func (k confirmKeys) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.Yes, k.No, k.Cancel}}
}

var defaultConfirmKeys = confirmKeys{
	Yes: key.NewBinding(
		key.WithKeys("y", core.KeyEnter),
		key.WithHelp("y/enter", "yes"),
	),
	No: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "no"),
	),
	Cancel: key.NewBinding(
		key.WithKeys(core.KeyCtrlC, core.KeyEsc),
		key.WithHelp("esc", "cancel"),
	),
}
