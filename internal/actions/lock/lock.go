// Package lock provides functionality for locking and unlocking branches in a stack.
package lock

import (
	"fmt"
	"strings"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/actions/submit"
	"stackit.dev/stackit/internal/app"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/output"
	"stackit.dev/stackit/internal/tui"
	"stackit.dev/stackit/internal/tui/style"
)

// Action locks the specified branch and all branches downstack of it
func Action(ctx *app.Context, branchName string) error {
	eng := ctx.Engine
	out := ctx.Output

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
			status, err := eng.GetBranchRemoteStatus(b)
			if err == nil && !status.Matches() {
				if status.Ahead() || status.MissingRemote() || status.Diverged() {
					unpushedBranches = append(unpushedBranches, b.GetName())
				}
			}
		}
	}

	if len(unpushedBranches) > 0 && ctx.Interactive {
		out.Warn("The following branches have unpushed commits:")
		for _, b := range unpushedBranches {
			out.Warn("  - %s", b)
		}
		confirm, err := tui.PromptConfirm("Would you like to submit these changes before locking?", true)
		if err == nil && confirm {
			submitOpts := submit.Options{
				Branch:     branchName,
				StackRange: submit.StackRangeDownstack(),
				Confirm:    false,
			}
			handler := &lockSubmitHandler{splog: out}
			if err := submit.Action(ctx, submitOpts, handler); err != nil {
				return fmt.Errorf("failed to submit before locking: %w", err)
			}
		}
	}

	affectedBranches := []string{}
	branchesToLock := []engine.Branch{}
	for _, b := range branches {
		if b.IsTrunk() {
			continue
		}
		if b.IsLocked() {
			out.Info("Branch %s is already locked.", style.ColorBranchName(b.GetName(), b.GetName() == branchName))
			continue
		}
		branchesToLock = append(branchesToLock, b)
	}

	if len(branchesToLock) > 0 {
		res, err := eng.SetLocked(branchesToLock, engine.LockReasonUser)
		if err != nil {
			// Report specific errors if some failed
			for name, branchErr := range res.Errors {
				out.Warn("Failed to lock %s: %v", name, branchErr)
			}
			return fmt.Errorf("failed to lock branches: %w", err)
		}

		for _, name := range res.AffectedBranches {
			out.Info("Locked %s.", style.ColorBranchName(name, name == branchName))
			affectedBranches = append(affectedBranches, name)
		}
	}

	// Push metadata changes to remote and update PRs to trigger CI re-evaluation
	if err := actions.PushMetadataAndSyncPRs(ctx, affectedBranches); err != nil {
		out.Debug("Failed to push metadata changes: %v", err)
	}

	return nil
}

// Unlock unlocks the specified branch and all branches upstack of it
func Unlock(ctx *app.Context, branchName string) error {
	eng := ctx.Engine
	out := ctx.Output

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
	branchesToUnlock := []engine.Branch{}
	for _, b := range branches {
		if b.IsTrunk() {
			continue
		}
		if !b.IsLocked() {
			out.Info("Branch %s is already unlocked.", style.ColorBranchName(b.GetName(), b.GetName() == branchName))
			continue
		}
		branchesToUnlock = append(branchesToUnlock, b)
	}

	if len(branchesToUnlock) > 0 {
		res, err := eng.SetLocked(branchesToUnlock, engine.LockReasonNone)
		if err != nil {
			// Report specific errors if some failed
			for name, branchErr := range res.Errors {
				out.Warn("Failed to unlock %s: %v", name, branchErr)
			}
			return fmt.Errorf("failed to unlock branches: %w", err)
		}

		for _, name := range res.AffectedBranches {
			out.Info("Unlocked %s.", style.ColorBranchName(name, name == branchName))
			affectedBranches = append(affectedBranches, name)
		}
	}

	// Push metadata changes to remote and update PRs to trigger CI re-evaluation
	if err := actions.PushMetadataAndSyncPRs(ctx, affectedBranches); err != nil {
		out.Debug("Failed to push metadata changes: %v", err)
	}

	return nil
}

type lockSubmitHandler struct {
	splog output.Output
}

func (h *lockSubmitHandler) OnEvent(e submit.Event) {
	if ev, ok := e.(submit.BranchProgressEvent); ok {
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
