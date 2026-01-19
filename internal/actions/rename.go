package actions

import (
	"fmt"

	"stackit.dev/stackit/internal/app"
	"stackit.dev/stackit/internal/tui"
	"stackit.dev/stackit/internal/tui/style"
	"stackit.dev/stackit/internal/utils"
)

// RenameOptions contains options for the rename command
type RenameOptions struct {
	NewName string
	Force   bool
}

// RenameAction renames the current branch and updates metadata
func RenameAction(ctx *app.Context, opts RenameOptions) error {
	eng := ctx.Engine
	out := ctx.Output

	currentBranch, err := eng.ValidateOnBranch()
	if err != nil {
		return err
	}

	if currentBranch == eng.Trunk().GetName() {
		return fmt.Errorf("cannot rename trunk branch %s", currentBranch)
	}

	branch := eng.GetBranch(currentBranch)
	if err := branch.EnsureCanModify(); err != nil {
		return err
	}

	newName := opts.NewName
	if newName == "" {
		if !utils.IsInteractive() {
			return fmt.Errorf("branch name is required in non-interactive mode")
		}

		newName, err = tui.PromptTextInput("Enter new branch name:", currentBranch)
		if err != nil {
			return err
		}
	}

	newName = utils.SanitizeBranchName(newName)
	if newName == "" {
		return fmt.Errorf("invalid branch name")
	}

	if newName == currentBranch {
		out.Info("Branch is already named %s.", newName)
		return nil
	}

	allBranches := eng.AllBranches()
	for _, b := range allBranches {
		if b.GetName() == newName {
			return fmt.Errorf("branch %s already exists", newName)
		}
	}

	prInfo, _ := branch.GetPrInfo()

	if prInfo != nil && prInfo.Number() != nil {
		if !opts.Force {
			return fmt.Errorf("branch %s is associated with PR #%d. Renaming it will remove this association. Use --force to proceed", currentBranch, *prInfo.Number())
		}
		out.Info("Removing association with PR #%d as GitHub PR branch names are immutable.", *prInfo.Number())
		if err := eng.UpsertPrInfo(ctx.Context, branch, nil); err != nil {
			return fmt.Errorf("failed to update metadata: %w", err)
		}
	}

	snapshotOpts := NewSnapshot("rename",
		WithArg(newName),
		WithFlag(opts.Force, "--force"),
	)
	if err := eng.TakeSnapshot(snapshotOpts); err != nil {
		out.Debug("Failed to take snapshot: %v", err)
	}

	oldBranchObj := eng.GetBranch(currentBranch)
	newBranchObj := eng.GetBranch(newName)
	if err := eng.RenameBranch(ctx.Context, oldBranchObj, newBranchObj); err != nil {
		return fmt.Errorf("failed to rename branch: %w", err)
	}

	out.Info("Renamed %s to %s.", style.ColorBranchName(currentBranch, false), style.ColorBranchName(newName, true))

	return nil
}
