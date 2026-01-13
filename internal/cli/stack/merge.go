// Package stack provides CLI commands for operating on entire stacks.
package stack

import (
	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/actions"
	mergeAction "stackit.dev/stackit/internal/actions/merge"
	"stackit.dev/stackit/internal/actions/sync"
	"stackit.dev/stackit/internal/app"
	"stackit.dev/stackit/internal/cli/common"
	mergeCmd "stackit.dev/stackit/internal/cli/stack/merge"
	"stackit.dev/stackit/internal/tui/style"
)

// NewMergeCmd creates the merge command
func NewMergeCmd() *cobra.Command {
	return mergeCmd.NewMergeCmd(handlePostMergeAction)
}

// handlePostMergeAction handles post-merge follow-up actions
func handlePostMergeAction(ctx *app.Context, action mergeAction.PostMergeAction) error {
	out := ctx.Output

	switch action {
	case mergeAction.PostMergeSyncTrunk:
		result, err := actions.CheckoutAction(ctx, actions.CheckoutOptions{
			CheckoutTrunk: true,
		}, nil)
		if err != nil {
			out.Newline()
			out.Error("%v", err)
			out.Newline()
			out.Info("%s", style.ColorYellow("To fix and continue:"))
			out.Info("  (1) Handle your local changes (e.g., %s or %s)", style.ColorCyan("git stash"), style.ColorCyan("git commit"))
			out.Info("  (2) Switch to trunk: %s", style.ColorCyan("stackit checkout --trunk"))
			out.Info("  (3) Sync your workspace: %s", style.ColorCyan("stackit sync --restack"))
			return nil
		}

		if result.WorktreeSwitchPath != "" {
			if common.HasShellIntegration() {
				ctx.Output.DirectiveCD(result.WorktreeSwitchPath)
				if len(result.RerunArgs) > 0 {
					ctx.Output.DirectiveRerun(result.RerunArgs...)
				}
			} else {
				for _, tip := range result.FallbackTips {
					ctx.Output.Tip("%s", tip)
				}
			}
		}

		runner, handler := NewSyncUI(ctx.Output, ctx.Logger)
		if runner != nil {
			defer runner.Cleanup()
		}

		return sync.Action(ctx, sync.Options{
			Restack: true,
		}, handler)

	case mergeAction.PostMergeDone:
		return nil
	}

	return nil
}
