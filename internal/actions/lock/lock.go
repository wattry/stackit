package lock

import (
	"fmt"
	"strings"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/actions/submit"
	"stackit.dev/stackit/internal/app"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/tui"
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

	// Check for unpushed commits
	unpushedBranches := []string{}
	if err := eng.PopulateRemoteShas(); err == nil {
		for _, b := range branches {
			if b.IsTrunk() {
				continue
			}
			matches, err := eng.BranchMatchesRemote(b.GetName())
			if err == nil && !matches {
				diff, err := eng.GetBranchRemoteDifference(b.GetName())
				if err == nil && strings.Contains(diff, "ahead") {
					unpushedBranches = append(unpushedBranches, b.GetName())
				}
			}
		}
	}

	if len(unpushedBranches) > 0 && ctx.Interactive {
		splog.Warn("The following branches have unpushed commits:")
		for _, b := range unpushedBranches {
			splog.Warn("  - %s", b)
		}
		confirm, err := tui.PromptConfirm("Would you like to submit these changes before locking?", true)
		if err == nil && confirm {
			submitOpts := submit.Options{
				Branch:  branchName,
				Stack:   false, // We only want to submit the downstack we're locking
				Confirm: false,
			}
			handler := &lockSubmitHandler{splog: splog}
			if err := submit.Action(ctx, submitOpts, handler); err != nil {
				return fmt.Errorf("failed to submit before locking: %w", err)
			}
		}
	}

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

	// Push metadata changes to remote and update PRs to trigger CI re-evaluation
	if err := actions.PushMetadataAndSyncPRs(ctx, affectedBranches); err != nil {
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

	// Check if downstack has locked branches and prompt to unlock them if interactive
	downstack := branch.GetRelativeStack(engine.StackRange{
		RecursiveParents: true,
	})

	lockedDownstack := []engine.Branch{}
	for _, b := range downstack {
		if !b.IsTrunk() && b.IsLocked() {
			lockedDownstack = append(lockedDownstack, b)
		}
	}

	if len(lockedDownstack) > 0 && ctx.Interactive {
		var prompt string
		if len(lockedDownstack) == 1 {
			prompt = fmt.Sprintf("Would you like to also unlock the downstack branch %s?", style.ColorBranchName(lockedDownstack[0].GetName(), false))
		} else {
			names := make([]string, len(lockedDownstack))
			for i, b := range lockedDownstack {
				names[i] = b.GetName()
			}
			prompt = fmt.Sprintf("Would you like to also unlock %d downstack branches (%s)?", len(lockedDownstack), strings.Join(names, ", "))
		}

		confirm, err := tui.PromptConfirm(prompt, true)
		if err == nil && confirm {
			branches = append(branches, lockedDownstack...)
		}
	}

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

	// Push metadata changes to remote and update PRs to trigger CI re-evaluation
	if err := actions.PushMetadataAndSyncPRs(ctx, affectedBranches); err != nil {
		splog.Debug("Failed to push metadata changes: %v", err)
	}

	return nil
}

type lockSubmitHandler struct {
	splog *tui.Splog
}

func (h *lockSubmitHandler) OnEvent(e submit.Event) {
	switch ev := e.(type) {
	case submit.BranchProgressEvent:
		if ev.Status == submit.StatusDone {
			h.splog.Info("  ✓ %s submitted → %s", ev.BranchName, ev.URL)
		} else if ev.Status == submit.StatusError {
			h.splog.Warn("  ✗ %s failed: %v", ev.BranchName, ev.Error)
		}
	}
}

func (h *lockSubmitHandler) Confirm(_ string, defaultYes bool) (bool, error) {
	return defaultYes, nil
}
