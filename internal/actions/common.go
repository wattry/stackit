// Package actions provides high-level business logic for CLI commands.
//
// Each action corresponds to a stackit command (create, submit, sync, etc.)
// and orchestrates operations across the engine, git, and github packages.
//
// Key patterns:
//   - Actions accept runtime.Context which provides Engine, Splog, and other dependencies
//   - Actions are stateless - all state is managed through the Engine interface
//   - Actions handle user interaction through the tui package
//
// Dependencies:
//   - engine: Core branch state management
//   - git: Low-level git operations
//   - tui: User interface and prompts
package actions

import (
	"context"
	"fmt"

	"github.com/gertd/go-pluralize"

	"stackit.dev/stackit/internal/app"
	"stackit.dev/stackit/internal/config"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/tui/style"
)

var pluralizeClient = pluralize.NewClient()

// Restacker is a minimal interface needed for restacking branches
type Restacker interface {
	engine.BranchReader
	engine.SyncManager
}

// RestackProgressCallback is called for each branch during restack
// branchName: the branch being processed
// result: the restack result (Done, Unneeded, Conflict)
// newRev: the new revision if restacked (empty if not applicable)
// conflict: true if this is a conflict
// lockReason: why the branch is locked (empty if not locked)
// frozen: true if the branch is frozen
// isCurrent: true if this is the current branch
// reparented: true if the branch was reparented
// oldParent: the old parent name if reparented
// newParent: the new parent name if reparented
type RestackProgressCallback func(branchName string, result engine.RestackResult, newRev string, conflict bool, lockReason engine.LockReason, frozen bool, isCurrent bool, reparented bool, oldParent, newParent string)

// RestackBranches restacks a list of branches using the engine's batch restack method
func RestackBranches(ctx *app.Context, branches []engine.Branch) error {
	return RestackBranchesWithHandler(ctx, branches, nil, true)
}

// RestackBranchesWithHandler restacks branches with optional progress callback
// If shouldEnterConflictWorkflow is true, stops at first conflict and enters conflict workflow
// If false, validates all branches first, restacks non-conflicting ones, and reports conflicts via callback
func RestackBranchesWithHandler(ctx *app.Context, branches []engine.Branch, callback RestackProgressCallback, shouldEnterConflictWorkflow bool) error {
	if len(branches) == 0 {
		return nil
	}

	// Log entry point for diagnostics
	branchNames := make([]string, len(branches))
	for i, b := range branches {
		branchNames[i] = b.GetName()
	}
	ctx.Logger.Info("restack started", "branches", branchNames, "count", len(branches))

	// Pre-flight validation: check branch ancestry relationships
	if err := validateBranchAncestry(ctx, branches); err != nil {
		return fmt.Errorf("pre-flight validation failed: %w", err)
	}

	// Build rebase specs for validation
	specs, branchMap := buildRebaseSpecs(ctx, branches)
	if len(specs) == 0 {
		ctx.Logger.Info("restack no specs built, nothing to do")
		return nil
	}

	// Validate all rebases in a temporary worktree (clean, no side effects)
	validation, err := ctx.Engine.ValidateRebases(ctx.Context, specs)
	if err != nil {
		return fmt.Errorf("failed to validate rebases: %w", err)
	}

	// Check if validation failed due to a system error (not a conflict)
	if !validation.Success {
		if validation.ErrorType == engine.ValidationErrorSystem {
			return fmt.Errorf("validation failed for %s: %s", validation.FailedBranch, validation.ErrorMessage)
		}
		// Else: conflict - continue with conflict workflow
		// Log conflicting files for debugging
		if len(validation.ConflictingFiles) > 0 {
			ctx.Logger.Debug("conflict detected during validation",
				"branch", validation.FailedBranch,
				"files", validation.ConflictingFiles)
		}
	}

	// Split branches into successful and conflicting
	var successBranches []engine.Branch
	var conflictBranches []string

	if validation.Success {
		// All branches succeeded - add them all to success list
		for _, branch := range branches {
			if _, exists := branchMap[branch.GetName()]; exists {
				successBranches = append(successBranches, branch)
			}
		}
	} else {
		// Build a position map for fast lookups
		positionMap := make(map[string]int)
		for i, spec := range specs {
			positionMap[spec.Branch] = i
		}

		// Find position of failed branch
		failedPos, failedExists := positionMap[validation.FailedBranch]
		if !failedExists {
			// This shouldn't happen, but handle gracefully
			ctx.Logger.Warn("failed branch not found in specs", "branch", validation.FailedBranch)
			failedPos = len(specs) // Treat as if it failed at the end
		}

		// Classify branches based on their position relative to the conflict
		for _, branch := range branches {
			branchName := branch.GetName()
			if _, exists := branchMap[branchName]; !exists {
				continue // Skip branches without specs
			}

			pos, ok := positionMap[branchName]
			if !ok {
				// Branch not in specs (shouldn't happen)
				continue
			}

			if pos < failedPos {
				// Branch comes before conflict - will succeed
				successBranches = append(successBranches, branch)
			} else if pos == failedPos {
				// This is the conflicted branch
				conflictBranches = append(conflictBranches, branchName)
			}
			// Branches after conflict (pos > failedPos) are not processed
		}
	}

	// For standalone mode, enter conflict workflow on first conflict
	if shouldEnterConflictWorkflow && len(conflictBranches) > 0 {
		firstConflict := conflictBranches[0]

		// Restack successfully up to the conflict
		if len(successBranches) > 0 {
			if _, err := ctx.Engine.RestackBranches(ctx.Context, successBranches); err != nil {
				return fmt.Errorf("failed to restack branches before conflict: %w", err)
			}
		}

		// Enter conflict workflow for the first conflict
		return EnterConflictWorkflow(ctx, firstConflict, branches)
	}

	// For sync mode (or standalone with no conflicts), restack all successful branches
	if len(successBranches) > 0 {
		batchResult, err := ctx.Engine.RestackBranches(ctx.Context, successBranches)
		if err != nil {
			return fmt.Errorf("batch restack failed: %w", err)
		}

		// Log restack results for diagnostics
		for branchName, result := range batchResult.Results {
			resultStr := "unknown"
			switch result.Result {
			case engine.RestackDone:
				resultStr = "done"
			case engine.RestackUnneeded:
				resultStr = "unneeded"
			case engine.RestackConflict:
				resultStr = "conflict"
			}
			ctx.Logger.Info("restack result",
				"branch", branchName,
				"result", resultStr,
				"reparented", result.Reparented,
				"oldParent", result.OldParent,
				"newParent", result.NewParent)
		}

		// Report results via callback or output
		currentBranch := ctx.Engine.CurrentBranch()
		currentBranchName := ""
		if currentBranch != nil {
			currentBranchName = currentBranch.GetName()
		}

		for _, branch := range successBranches {
			branchName := branch.GetName()
			result, exists := batchResult.Results[branchName]
			if !exists {
				continue
			}

			// Get new revision if available
			newRev := ""
			if result.Result == engine.RestackDone {
				if rev, err := branch.GetRevision(); err == nil {
					if len(rev) > 7 {
						newRev = rev[:7]
					} else {
						newRev = rev
					}
				}
			}

			// Report via callback if provided
			if callback != nil {
				callback(branchName, result.Result, newRev, false, result.LockReason, result.Frozen, branchName == currentBranchName, result.Reparented, result.OldParent, result.NewParent)
				continue
			}

			// Log via splog only when no callback is provided
			if result.Reparented {
				isCurrent := branchName == currentBranchName
				ctx.Output.Info("Reparented %s from %s to %s (parent was merged/deleted).",
					style.ColorBranchName(branchName, isCurrent),
					style.ColorBranchName(result.OldParent, false),
					style.ColorBranchName(result.NewParent, false))
			}

			switch result.Result {
			case engine.RestackDone:
				parentName := branch.GetParentPrecondition()
				isCurrent := branchName == currentBranchName
				ctx.Output.Info("Restacked %s on %s.",
					style.ColorBranchName(branchName, isCurrent),
					style.ColorBranchName(parentName, false))
			case engine.RestackUnneeded:
				switch {
				case !branch.CanModify():
					if branch.IsLocked() {
						ctx.Output.Info("%s locked: %s", style.ColorBranchName(branchName, branchName == currentBranchName), branch.GetLockReason())
					} else {
						ctx.Output.Info("%s frozen", style.ColorBranchName(branchName, branchName == currentBranchName))
					}
				case branch.IsTrunk():
					ctx.Output.Info("%s up to date", style.ColorBranchName(branchName, false))
				default:
					isCurrent := branchName == currentBranchName
					ctx.Output.Info("%s up to date", style.ColorBranchName(branchName, isCurrent))
				}
			}
		}
	}

	// Report conflicts via callback for sync mode
	if !shouldEnterConflictWorkflow && len(conflictBranches) > 0 {
		currentBranch := ctx.Engine.CurrentBranch()
		currentBranchName := ""
		if currentBranch != nil {
			currentBranchName = currentBranch.GetName()
		}

		for _, branchName := range conflictBranches {
			if callback != nil {
				callback(branchName, engine.RestackConflict, "", true, engine.LockReasonNone, false, branchName == currentBranchName, false, "", "")
			}
		}
	}

	return nil
}

// EnterConflictWorkflow performs the rebase to enter conflict state and persists continuation state.
// This helper is shared between RestackBranchesWithHandler (standalone mode) and sync.RunSync (sync mode).
func EnterConflictWorkflow(ctx *app.Context, firstConflict string, allBranches []engine.Branch) error {
	// Perform rebase to enter conflict state
	conflictBranch := ctx.Engine.GetBranch(firstConflict)
	batchResult, err := ctx.Engine.RestackBranches(ctx.Context, []engine.Branch{conflictBranch})
	if err != nil {
		return fmt.Errorf("failed to enter conflict state for %s: %w", firstConflict, err)
	}

	// Verify we're actually in conflict state
	if !ctx.Engine.Git().IsRebaseInProgress(ctx.Context) {
		return fmt.Errorf("expected conflict on %s but rebase completed successfully", firstConflict)
	}

	// Note: We don't verify the current branch here because CurrentBranch() doesn't work
	// correctly during an in-progress rebase. The IsRebaseInProgress check above is sufficient
	// to verify we're in the expected state.

	// Build remaining branches list
	var remainingBranches []string
	foundConflict := false
	for _, branch := range allBranches {
		if foundConflict {
			remainingBranches = append(remainingBranches, branch.GetName())
		} else if branch.GetName() == firstConflict {
			foundConflict = true
		}
	}

	// Get RebasedBranchBase from result
	rebasedBranchBase := ""
	if result, ok := batchResult.Results[firstConflict]; ok {
		rebasedBranchBase = result.RebasedBranchBase
	}

	// Persist continuation state
	continuation := &config.ContinuationState{
		BranchesToRestack:     remainingBranches,
		RebasedBranchBase:     rebasedBranchBase,
		CurrentBranchOverride: firstConflict,
	}

	if err := config.PersistContinuationState(ctx.RepoRoot, continuation); err != nil {
		return fmt.Errorf("failed to persist continuation: %w", err)
	}

	if err := PrintConflictStatus(ctx, firstConflict); err != nil {
		return fmt.Errorf("failed to print conflict status: %w", err)
	}

	return fmt.Errorf("restack stopped due to conflict on %s", firstConflict)
}

// validateBranchAncestry performs pre-flight checks on branch ancestry relationships.
// Returns an error if any branch has invalid parent relationships.
// Note: Missing parents are tolerated as buildRebaseSpecs will handle auto-reparenting.
func validateBranchAncestry(ctx *app.Context, branches []engine.Branch) error {
	for _, branch := range branches {
		branchName := branch.GetName()

		// Trunk branches don't need validation
		if branch.IsTrunk() {
			continue
		}

		// Verify branch itself exists
		branchRev, err := branch.GetRevision()
		if err != nil {
			return fmt.Errorf("branch %s cannot be resolved: %w", branchName, err)
		}

		// If branch has a parent, validate it only if it exists
		// (missing parents will be handled by auto-reparenting logic)
		parent := branch.GetParent()
		if parent != nil {
			parentName := parent.GetName()
			parentRev, err := parent.GetRevision()
			if err != nil {
				// Parent doesn't exist - this is OK, auto-reparenting will handle it
				ctx.Logger.Debug("parent branch missing, will auto-reparent",
					"branch", branchName,
					"parent", parentName)
				continue
			}

			// Parent exists - verify they have a common ancestor
			_, err = ctx.Engine.Git().GetMergeBase(parentRev, branchRev)
			if err != nil {
				return fmt.Errorf("branch %s and parent %s have no common ancestor: %w",
					branchName, parentName, err)
			}
		}
	}
	return nil
}

// buildRebaseSpecs builds RebaseSpec list for validation
func buildRebaseSpecs(ctx *app.Context, branches []engine.Branch) ([]engine.RebaseSpec, map[string]bool) {
	specs := make([]engine.RebaseSpec, 0, len(branches))
	branchMap := make(map[string]bool)

	ctx.Logger.Debug("buildRebaseSpecs starting", "branchCount", len(branches))

	for _, branch := range branches {
		branchName := branch.GetName()

		// Check if parent exists or needs reparenting
		parent := branch.GetParent()
		var parentName string
		if parent == nil {
			trunk := ctx.Engine.Trunk()
			parentName = trunk.GetName()
		} else {
			parentName = parent.GetName()
			// Check if parent branch still exists by trying to get its revision
			parentBranch := ctx.Engine.GetBranch(parentName)
			if _, err := parentBranch.GetRevision(); err != nil {
				// Parent doesn't exist, find nearest valid ancestor
				ancestors, err := ctx.Engine.FindMostRecentTrackedAncestors(ctx.Context, branchName)
				if err == nil && len(ancestors) > 0 {
					parentName = ancestors[0]
				} else {
					// Fall back to trunk if no ancestors found
					parentName = ctx.Engine.Trunk().GetName()
				}
			}
		}

		// Get old parent revision from metadata
		meta, err := ctx.Engine.Git().ReadMetadata(branchName)
		if err != nil {
			continue // Skip if can't read metadata
		}

		var oldParentRev string
		if rev := meta.GetParentBranchRevision(); rev != nil {
			oldParentRev = *rev
		}

		// RESILIENCY: If oldParentRev is no longer an ancestor of branchName,
		// or if it's empty, find the actual merge base. This handles cases where
		// the parent was amended, rebased, or deleted.
		if oldParentRev != "" {
			isAncestor, err := ctx.Engine.Git().IsAncestor(oldParentRev, branchName)
			if err != nil {
				ctx.Logger.Warn("failed to check ancestry, will try merge-base",
					"oldParent", oldParentRev, "branch", branchName, "error", err)
				isAncestor = false
			}
			if !isAncestor {
				if mergeBase, err := ctx.Engine.Git().GetMergeBase(branchName, parentName); err == nil {
					oldParentRev = mergeBase
				} else {
					// Can't determine merge base - skip this branch
					ctx.Logger.Warn("failed to determine merge base", "branch", branchName, "parent", parentName, "error", err)
					continue
				}
			}
		} else {
			// No old parent revision in metadata, try to find merge base
			if mergeBase, err := ctx.Engine.Git().GetMergeBase(branchName, parentName); err == nil {
				oldParentRev = mergeBase
			} else {
				// Can't determine merge base - skip this branch
				ctx.Logger.Warn("failed to determine merge base", "branch", branchName, "parent", parentName, "error", err)
				continue
			}
		}

		specs = append(specs, engine.RebaseSpec{
			Branch:      branchName,
			NewParent:   parentName,
			OldUpstream: oldParentRev,
		})
		branchMap[branchName] = true

		ctx.Logger.Debug("buildRebaseSpecs added spec",
			"branch", branchName,
			"newParent", parentName,
			"oldUpstream", oldParentRev)
	}

	ctx.Logger.Debug("buildRebaseSpecs completed", "specCount", len(specs))
	return specs, branchMap
}

// PluralSuffix returns the appropriate plural suffix for the given word if plural is true, otherwise empty string
func PluralSuffix(word string, plural bool) string {
	if !plural {
		return ""
	}
	pluralized := pluralizeClient.Plural(word)
	if len(pluralized) > len(word) {
		return pluralized[len(word):]
	}
	return "s" // fallback
}

// Pluralize returns the plural form of word if count != 1, otherwise returns the singular form
func Pluralize(word string, count int) string {
	if count == 1 {
		return word
	}
	return pluralizeClient.Plural(word)
}

// ShouldDeleteBranch checks if a branch should be deleted
func ShouldDeleteBranch(ctx context.Context, branchName string, eng engine.BranchStatus, force bool) (bool, string) {
	status, err := eng.GetDeletionStatus(ctx, branchName)
	if err != nil {
		return false, ""
	}

	if status.SafeToDelete {
		return true, status.Reason
	}

	if force {
		return false, ""
	}

	// Interactive prompting not yet implemented
	return false, ""
}

type deleteBranchCachedEngine interface {
	engine.StackNavigator
	engine.BranchInfo
	engine.GitDiffer
}

// ShouldDeleteBranchCached checks if a branch should be deleted using pre-fetched metadata and revisions
func ShouldDeleteBranchCached(ctx context.Context, branchName string, eng deleteBranchCachedEngine, force bool, meta *git.Meta, revisions map[string]string, mergedBranches map[string]bool) (bool, string) {
	// 0. Never delete worktree anchor branches - they appear "merged" because they
	// point to the same commit as trunk, but they're needed for worktree management
	branch := eng.GetBranch(branchName)
	if branch.IsWorktreeAnchor() {
		return false, ""
	}

	// 1. Check PR info from cached metadata
	if meta != nil {
		prInfo := meta.GetPrInfo()
		if prInfo != nil {
			const (
				prStateClosed = "CLOSED"
				prStateMerged = "MERGED"
			)
			if prInfo.State != nil {
				if *prInfo.State == prStateClosed {
					return true, "closed on GitHub"
				}
				if *prInfo.State == prStateMerged {
					base := ""
					if prInfo.Base != nil {
						base = *prInfo.Base
					}
					if base == "" {
						base = eng.Trunk().GetName()
					}
					return true, fmt.Sprintf("merged into %s", base)
				}
			}
		}
	}

	// 2. Check if merged into trunk
	trunkName := eng.Trunk().GetName()
	if mergedBranches != nil && mergedBranches[branchName] {
		return true, fmt.Sprintf("merged into %s", trunkName)
	}

	// 3. Check if empty
	// Need parent revision
	var parentRev string
	parent := branch.GetParent()
	parentName := trunkName
	if parent != nil {
		parentName = parent.GetName()
	}

	// Use cached revisions to avoid eng.Git().GetRevision calls
	if rev, ok := revisions[parentName]; ok {
		parentRev = rev
	} else {
		// Fallback
		rev, err := eng.GetRevisionForName(parentName)
		if err == nil {
			parentRev = rev
		}
	}

	if parentRev != "" {
		empty, err := eng.IsDiffEmpty(ctx, branchName, parentRev)
		if err == nil && empty { // IsDiffEmpty returns true if no diff
			// Only delete empty branches if they have a PR
			metaPrInfo := meta.GetPrInfo()
			if meta != nil && metaPrInfo != nil && metaPrInfo.Number != nil && *metaPrInfo.Number != 0 {
				return true, "empty"
			}
		}
	}

	if force {
		return false, ""
	}

	return false, ""
}

// PluralIt returns "them" if plural is true, otherwise "it"
func PluralIt(plural bool) string {
	if plural {
		return "them"
	}
	return "it"
}

// SnapshotOption is a function that modifies SnapshotOptions
type SnapshotOption func(*engine.SnapshotOptions)

// NewSnapshot creates a new SnapshotOptions with the given command and options
func NewSnapshot(command string, options ...SnapshotOption) engine.SnapshotOptions {
	opts := engine.SnapshotOptions{
		Command: command,
		Args:    []string{},
	}
	for _, option := range options {
		option(&opts)
	}
	return opts
}

// WithArg appends a single argument if it's not empty
func WithArg(arg string) SnapshotOption {
	return func(opts *engine.SnapshotOptions) {
		if arg != "" {
			opts.Args = append(opts.Args, arg)
		}
	}
}

// WithArgs appends multiple arguments
func WithArgs(args ...string) SnapshotOption {
	return func(opts *engine.SnapshotOptions) {
		opts.Args = append(opts.Args, args...)
	}
}

// WithFlag appends a flag if condition is true
func WithFlag(condition bool, flag string) SnapshotOption {
	return func(opts *engine.SnapshotOptions) {
		if condition {
			opts.Args = append(opts.Args, flag)
		}
	}
}

// WithFlagValue appends a flag with a value if the value is not empty
func WithFlagValue(flag string, value string) SnapshotOption {
	return func(opts *engine.SnapshotOptions) {
		if value != "" {
			opts.Args = append(opts.Args, flag, value)
		}
	}
}

// PrintConflictStatus displays conflict information and instructions to the user
func PrintConflictStatus(ctx *app.Context, branchName string) error {
	reader := ctx.Reader()
	out := ctx.Output

	msg := style.ColorRed(fmt.Sprintf("Hit conflict restacking %s", branchName))
	out.Info("%s", msg)
	out.Newline()

	// Get unmerged files
	unmergedFiles, err := reader.GetUnmergedFiles(ctx.Context)
	if err == nil && len(unmergedFiles) > 0 {
		out.Info("%s", style.ColorYellow("Unmerged files:"))
		for _, file := range unmergedFiles {
			out.Info("%s", style.ColorRed(file))
		}
		out.Newline()
	}

	// Get rebase head
	rebaseHead, err := reader.GetRebaseHead()
	if err == nil && rebaseHead != "" {
		rebaseHeadShort := rebaseHead
		if len(rebaseHead) > 7 {
			rebaseHeadShort = rebaseHead[:7]
		}
		msg := style.ColorYellow(fmt.Sprintf("You are here (resolving %s):", rebaseHeadShort))
		out.Info("%s", msg)
		// Could show log here if needed
		out.Newline()
	}

	out.Info("%s", style.ColorYellow("To fix and continue your previous Stackit command:"))
	out.Info("(1) resolve the listed merge conflicts")
	out.Info("(2) mark them as resolved with %s", style.ColorCyan("stackit add ."))
	out.Info("(3) run %s to continue executing your previous Stackit command", style.ColorCyan("stackit continue"))
	out.Info("It's safe to cancel the ongoing rebase with %s.", style.ColorCyan("git rebase --abort"))

	return nil
}
