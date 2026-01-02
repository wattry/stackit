// Package keys provides shared keybindings for TUI views.
package keys

import "github.com/charmbracelet/bubbles/key"

// NavigationKeyMap provides standard navigation keybindings shared across TUI views.
type NavigationKeyMap struct {
	Up     key.Binding
	Down   key.Binding
	Select key.Binding
	Cancel key.Binding
}

// DefaultNavigation returns standard navigation keybindings with vim support.
var DefaultNavigation = NavigationKeyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	Select: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	),
	Cancel: key.NewBinding(
		key.WithKeys("ctrl+c", "esc", "q"),
		key.WithHelp("esc", "cancel"),
	),
}

// ReorderKeyMap extends NavigationKeyMap with move operations for reordering.
type ReorderKeyMap struct {
	NavigationKeyMap
	MoveUp   key.Binding
	MoveDown key.Binding
}

// DefaultReorder returns keybindings for reorder views.
var DefaultReorder = ReorderKeyMap{
	NavigationKeyMap: DefaultNavigation,
	MoveUp: key.NewBinding(
		key.WithKeys("shift+up", "K"),
		key.WithHelp("K", "move up"),
	),
	MoveDown: key.NewBinding(
		key.WithKeys("shift+down", "J"),
		key.WithHelp("J", "move down"),
	),
}

// LogKeyMap extends NavigationKeyMap with log-specific operations.
type LogKeyMap struct {
	NavigationKeyMap
	Search key.Binding
	Expand key.Binding
	Quit   key.Binding
}

// DefaultLog returns keybindings for log views.
var DefaultLog = LogKeyMap{
	NavigationKeyMap: DefaultNavigation,
	Search: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "search"),
	),
	Expand: key.NewBinding(
		key.WithKeys(" ", "enter"),
		key.WithHelp("space", "expand/collapse"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}

// SelectKeyMap provides keybindings for selection mode.
type SelectKeyMap struct {
	NavigationKeyMap
	Search key.Binding
	Expand key.Binding
}

// DefaultSelect returns keybindings for selection views.
var DefaultSelect = SelectKeyMap{
	NavigationKeyMap: DefaultNavigation,
	Search: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "search"),
	),
	Expand: key.NewBinding(
		key.WithKeys(" "),
		key.WithHelp("space", "expand/collapse"),
	),
}
