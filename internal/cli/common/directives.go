package common

import (
	"os"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/output"
)

// HasShellIntegration checks if stackit shell integration is installed.
// The shell wrapper sets STACKIT_SHELL_INTEGRATION=1 when running commands.
func HasShellIntegration() bool {
	return os.Getenv("STACKIT_SHELL_INTEGRATION") == "1"
}

// HandleCheckoutResult handles the worktree switch result from CheckoutAction.
// Returns true if shell integration handled the switch (caller should return nil).
// Returns false with tips printed if no shell integration.
func HandleCheckoutResult(out output.Output, result actions.CheckoutResult) bool {
	if result.WorktreeSwitchPath == "" {
		return false
	}
	if HasShellIntegration() {
		out.DirectiveCD(result.WorktreeSwitchPath)
		if len(result.RerunArgs) > 0 {
			out.DirectiveRerun(result.RerunArgs...)
		}
		return true
	}
	for _, tip := range result.FallbackTips {
		out.Tip("%s", tip)
	}
	return false
}
