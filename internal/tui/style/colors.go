// Package style provides styling and coloring for the TUI.
package style

import "github.com/charmbracelet/lipgloss"

// StackitColors defines the color palette for branch visualization
// Matching the TypeScript version
var StackitColors = [][]int{
	{76, 203, 241},  // Light blue
	{77, 202, 125},  // Green
	{110, 173, 38},  // Dark green
	{245, 200, 0},   // Yellow
	{248, 144, 72},  // Orange
	{244, 98, 81},   // Red
	{235, 130, 188}, // Pink
	{159, 131, 228}, // Purple
	{80, 132, 243},  // Blue
}

// ColorRed colors text red
func ColorRed(text string) string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("1")).
		Render(text)
}

// ColorYellow colors text yellow
func ColorYellow(text string) string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("3")).
		Render(text)
}

// ColorCyan colors text cyan
func ColorCyan(text string) string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("6")).
		Render(text)
}

// ColorGreen colors text green
func ColorGreen(text string) string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("2")).
		Render(text)
}

// Selection cursor constants
const (
	SelectionCursor  = "▸ " // Cursor symbol for selected items
	SelectionPadding = "  " // Same width as cursor for unselected items
)

// Selection returns a style for selected items
func Selection() lipgloss.Style {
	return lipgloss.NewStyle().
		Background(lipgloss.Color("6")). // Cyan background
		Foreground(lipgloss.Color("0")). // Black foreground
		Bold(true)
}

// SelectionCursorStyle returns the style for the selection cursor
func SelectionCursorStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("6")). // Cyan
		Bold(true)
}
