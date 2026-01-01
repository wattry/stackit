package actions

import (
	"fmt"

	"stackit.dev/stackit/internal/app"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/tui/style"
)

// LockAction locks the specified branch and all branches downstack of it
func LockAction(ctx *app.Context, branchName string) error {
	eng := ctx.Engine
	splog := ctx.Splog

	branch := eng.GetBranch(branchName)
	if branch.IsTrunk() {
		return fmt.Errorf("cannot lock trunk branch %s", branchName)
	}

	if !branch.IsTracked() {
		return fmt.Errorf("branch %s is not tracked by stackit", branchName)
	}

	// Get downstack (ancestors including current)
	branches := branch.GetRelativeStack(engine.StackRange{
		RecursiveParents: true,
		IncludeCurrent:   true,
	})

	affectedBranches := []string{}
	for _, b := range branches {
		if b.IsTrunk() {
			continue
		}
		if b.IsLocked() {
			splog.Info("Branch %s is already locked.", style.ColorBranchName(b.GetName(), b.GetName() == branchName))
			continue
		}
		if err := eng.SetLocked(b, true); err != nil {
			return fmt.Errorf("failed to lock branch %s: %w", b.GetName(), err)
		}
		splog.Info("Locked %s.", style.ColorBranchName(b.GetName(), b.GetName() == branchName))
		affectedBranches = append(affectedBranches, b.GetName())
	}

	// Push metadata changes to remote
	if err := pushMetadataForBranches(ctx, affectedBranches); err != nil {
		splog.Debug("Failed to push metadata changes: %v", err)
	}

	return nil
}

// UnlockAction unlocks the specified branch and all branches upstack of it
func UnlockAction(ctx *app.Context, branchName string) error {
	eng := ctx.Engine
	splog := ctx.Splog

	branch := eng.GetBranch(branchName)
	if !branch.IsTracked() {
		return fmt.Errorf("branch %s is not tracked by stackit", branchName)
	}

	// Get upstack (descendants including current)
	branches := branch.GetRelativeStack(engine.StackRange{
		IncludeCurrent:    true,
		RecursiveChildren: true,
	})

	affectedBranches := []string{}
	for _, b := range branches {
		if b.IsTrunk() {
			continue
		}
		if !b.IsLocked() {
			splog.Info("Branch %s is already unlocked.", style.ColorBranchName(b.GetName(), b.GetName() == branchName))
			continue
		}
		if err := eng.SetLocked(b, false); err != nil {
			return fmt.Errorf("failed to unlock branch %s: %w", b.GetName(), err)
		}
		splog.Info("Unlocked %s.", style.ColorBranchName(b.GetName(), b.GetName() == branchName))
		affectedBranches = append(affectedBranches, b.GetName())
	}

	// Push metadata changes to remote
	if err := pushMetadataForBranches(ctx, affectedBranches); err != nil {
		splog.Debug("Failed to push metadata changes: %v", err)
	}

	return nil
}

// pushMetadataForBranches pushes metadata for the given branches to remote
func pushMetadataForBranches(ctx *app.Context, branchNames []string) error {
	if len(branchNames) == 0 {
		return nil
	}

	eng := ctx.Engine
	splog := ctx.Splog

	// Update LastModifiedBy for each branch
	for _, branchName := range branchNames {
		if err := eng.SetLastModifiedBy(branchName); err != nil {
			splog.Debug("Failed to update metadata for %s: %v", branchName, err)
			continue
		}
	}

	// Check if remote sync is enabled; if not, run compatibility test first
	if !eng.IsRemoteSyncEnabled() {
		if err := eng.Git().TestRemoteRefCompatibility(); err != nil {
			splog.Debug("Remote metadata sync not supported: %v", err)
			return nil // Non-fatal
		}
		eng.SetRemoteSyncEnabled(true)
		// Configure refspec so future git fetch commands also fetch metadata
		if err := eng.Git().EnsureMetadataRefspecConfigured(); err != nil {
			splog.Debug("Failed to configure metadata refspec: %v", err)
		}
	}

	// Push metadata refs
	if err := eng.Git().PushMetadataRefs(branchNames); err != nil {
		splog.Debug("Failed to push metadata refs: %v", err)
		return err
	}

	return nil
}
