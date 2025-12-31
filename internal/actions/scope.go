package actions

import (
	"fmt"
	"strings"

	"stackit.dev/stackit/internal/app"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/git"
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
	splog := ctx.Splog

	// Get current branch
	currentBranch, _ := git.GetCurrentBranch()
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
				splog.Info("Branch %s has scope inheritance DISABLED (explicitly set to '%s').", style.ColorBranchName(currentBranch, false), explicitScope.String())
			} else {
				splog.Info("Branch %s has explicit scope: %s", style.ColorBranchName(currentBranch, false), style.ColorDim(explicitScope.String()))
			}
		case !resolvedScope.IsEmpty():
			splog.Info("Branch %s inherits scope: %s", style.ColorBranchName(currentBranch, false), style.ColorDim(resolvedScope.String()))
		default:
			splog.Info("Branch %s has no scope set.", style.ColorBranchName(currentBranch, false))
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
		splog.Info("Unset explicit scope for branch %s. It will now inherit from its parent.", style.ColorBranchName(currentBranch, false))

		// Push metadata changes to remote
		if err := pushMetadataForSingleBranch(ctx, currentBranch); err != nil {
			splog.Debug("Failed to push metadata changes: %v", err)
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
		splog.Info("Disabled scope for branch %s (breaks inheritance).", style.ColorBranchName(currentBranch, false))
	} else {
		splog.Info("Set scope for branch %s to: %s", style.ColorBranchName(currentBranch, false), style.ColorDim(opts.Scope))

		// Rename prompt
		if oldScope.IsDefined() && !oldScope.Equal(newScope) && utils.IsInteractive() && strings.Contains(currentBranch, oldScope.String()) {
			confirmed, err := tui.PromptConfirm(fmt.Sprintf("Branch name contains '%s', but its scope is now '%s'. Would you like to rename the branch?", oldScope.String(), opts.Scope), true)
			if err == nil && confirmed {
				newName := strings.Replace(currentBranch, oldScope.String(), opts.Scope, 1)
				if err := eng.RenameBranch(ctx.Context, eng.GetBranch(currentBranch), eng.GetBranch(newName)); err != nil {
					splog.Info("Warning: failed to rename branch: %v", err)
				} else {
					splog.Info("Renamed branch %s to %s.", style.ColorBranchName(currentBranch, false), style.ColorBranchName(newName, true))
				}
			}
		}
	}

	// Push metadata changes to remote
	if err := pushMetadataForSingleBranch(ctx, currentBranch); err != nil {
		splog.Debug("Failed to push metadata changes: %v", err)
	}

	return nil
}

// pushMetadataForSingleBranch is a helper that pushes metadata for a single branch
func pushMetadataForSingleBranch(ctx *app.Context, branchName string) error {
	eng := ctx.Engine
	splog := ctx.Splog

	// Update LastModifiedBy
	if err := eng.SetLastModifiedBy(branchName); err != nil {
		splog.Debug("Failed to update metadata for %s: %v", branchName, err)
	}

	// Check if remote sync is enabled; if not, run compatibility test first
	if !eng.IsRemoteSyncEnabled() {
		if err := git.TestRemoteRefCompatibility(); err != nil {
			splog.Debug("Remote metadata sync not supported: %v", err)
			return nil // Non-fatal
		}
		eng.SetRemoteSyncEnabled(true)
		// Configure refspec so future git fetch commands also fetch metadata
		if err := git.EnsureMetadataRefspecConfigured(); err != nil {
			splog.Debug("Failed to configure metadata refspec: %v", err)
		}
	}

	// Push metadata ref
	if err := git.PushMetadataRefs([]string{branchName}); err != nil {
		splog.Debug("Failed to push metadata refs: %v", err)
		return err
	}

	return nil
}
