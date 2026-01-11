// Package dashboard provides the interactive shippable work TUI.
package dashboard

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"stackit.dev/stackit/internal/app"
	"stackit.dev/stackit/internal/config"
	"stackit.dev/stackit/internal/utils"
)

// RunShippable starts the shippable work dashboard.
func RunShippable(ctx *app.Context, opts ShippableOptions) error {
	if !utils.IsInteractive() {
		return fmt.Errorf("dashboard requires an interactive terminal")
	}

	// Load config for CI settings
	cfg, err := config.LoadConfig(ctx.RepoRoot)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	m := newShippableModel(ctx, cfg, opts)

	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err = p.Run()
	return err
}
