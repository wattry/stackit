// Package dashboard provides the interactive shippable work TUI.
package dashboard

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"stackit.dev/stackit/internal/app"
	"stackit.dev/stackit/internal/config"
	"stackit.dev/stackit/internal/tui"
)

// RunShippable starts the shippable work dashboard.
func RunShippable(ctx *app.Context, opts ShippableOptions) error {
	if !tui.IsTTY() {
		return fmt.Errorf("dashboard requires an interactive terminal")
	}

	// Load config for CI settings
	cfg, err := config.LoadConfig(ctx.RepoRoot)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	m := newShippableModel(ctx, cfg, opts)

	// Note: The dashboard uses alt-screen mode which is not directly supported
	// by tui.Runner. We use tea.NewProgram directly with appropriate options.
	// The model embeds core.BaseModel for ready signaling and common handling.
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithInput(os.Stdin), tea.WithOutput(os.Stdout))
	_, err = p.Run()
	return err
}
