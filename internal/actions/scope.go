package actions

import (
	"fmt"
	"strings"

	"stackit.dev/stackit/internal/app"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/tui"
	"stackit.dev/stackit/internal/tui/style"
	"stackit.dev/stackit/internal/utils"
)

// ScopeOptions contains options for the scope command
type ScopeOptions struct {
	Scope string
	Unset bool
	Show  bool
}

// ScopeAction implements the stackit scope command
func ScopeAction(ctx *app.Context, opts ScopeOptions) error {
	eng := ctx.Engine
	out := ctx.Output

	// Get current branch
	currentBranch := ""
	if cb := eng.CurrentBranch(); cb != nil {
		currentBranch = cb.GetName()
	}
	isOnTrunk := currentBranch == eng.Trunk().GetName() || currentBranch == ""

	// Handle Show
	if opts.Show {
		if isOnTrunk {
			return fmt.Errorf("not on a branch")
		}
		currentBranchObj := eng.GetBranch(currentBranch)
		explicitScope := currentBranchObj.GetExplicitScope()
		resolvedScope := eng.GetScope(currentBranchObj)

		switch {
		case !explicitScope.IsEmpty():
			if explicitScope.IsNone() {
				out.Info("Branch %s has scope inheritance DISABLED (explicitly set to '%s').", style.ColorBranchName(currentBranch, false), explicitScope.String())
			} else {
				out.Info("Branch %s has explicit scope: %s", style.ColorBranchName(currentBranch, false), style.ColorDim(explicitScope.String()))
			}
		case !resolvedScope.IsEmpty():
			out.Info("Branch %s inherits scope: %s", style.ColorBranchName(currentBranch, false), style.ColorDim(resolvedScope.String()))
		default:
			out.Info("Branch %s has no scope set.", style.ColorBranchName(currentBranch, false))
		}
		return nil
	}

	// Handle Unset
	if opts.Unset {
		if isOnTrunk {
			return fmt.Errorf("cannot unset scope on trunk")
		}
		if err := eng.SetScope(eng.GetBranch(currentBranch), engine.Empty()); err != nil {
			return fmt.Errorf("failed to unset scope: %w", err)
		}
		out.Info("Unset explicit scope for branch %s. It will now inherit from its parent.", style.ColorBranchName(currentBranch, false))

		// Push metadata changes to remote and update PRs to trigger CI re-evaluation
		if err := PushMetadataAndSyncPRs(ctx, []string{currentBranch}); err != nil {
			out.Debug("Failed to push metadata changes: %v", err)
		}
		return nil
	}

	// If no scope provided and not show/unset, we don't know what to do
	if opts.Scope == "" {
		return fmt.Errorf("no scope name provided")
	}

	// Cannot set scope on trunk
	if isOnTrunk {
		return fmt.Errorf("cannot set scope on trunk")
	}

	// Update the current branch's scope
	currentBranchObj := eng.GetBranch(currentBranch)
	oldScope := eng.GetScope(currentBranchObj)
	newScope := engine.NewScope(opts.Scope)
	if err := eng.SetScope(eng.GetBranch(currentBranch), newScope); err != nil {
		return fmt.Errorf("failed to set scope: %w", err)
	}

	if newScope.IsNone() {
		out.Info("Disabled scope for branch %s (breaks inheritance).", style.ColorBranchName(currentBranch, false))
	} else {
		out.Info("Set scope for branch %s to: %s", style.ColorBranchName(currentBranch, false), style.ColorDim(opts.Scope))

		// Rename prompt
		if oldScope.IsDefined() && !oldScope.Equal(newScope) && utils.IsInteractive() && strings.Contains(currentBranch, oldScope.String()) {
			confirmed, err := tui.PromptConfirm(fmt.Sprintf("Branch name contains '%s', but its scope is now '%s'. Would you like to rename the branch?", oldScope.String(), opts.Scope), true)
			if err == nil && confirmed {
				newName := strings.Replace(currentBranch, oldScope.String(), opts.Scope, 1)
				if err := eng.RenameBranch(ctx.Context, eng.GetBranch(currentBranch), eng.GetBranch(newName)); err != nil {
					out.Info("Warning: failed to rename branch: %v", err)
				} else {
					out.Info("Renamed branch %s to %s.", style.ColorBranchName(currentBranch, false), style.ColorBranchName(newName, true))
				}
			}
		}
	}

	// Push metadata changes to remote and update PRs to trigger CI re-evaluation
	if err := PushMetadataAndSyncPRs(ctx, []string{currentBranch}); err != nil {
		out.Debug("Failed to push metadata changes: %v", err)
	}

	return nil
}
