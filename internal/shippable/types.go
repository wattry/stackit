// Package shippable provides analysis and management of shippable work.
// It determines which stacks are ready to be merged to trunk based on
// PR approval status, CI checks, and combinability with other stacks.
package shippable

import (
	"stackit.dev/stackit/internal/actions/merge"
)

// Status represents the overall shippability state of a stack.
type Status string

const (
	// StatusShippable indicates the stack is ready to ship (all checks pass, approved).
	StatusShippable Status = "shippable"
	// StatusPending indicates the stack is waiting on CI or review.
	StatusPending Status = "pending"
	// StatusBlocked indicates the stack cannot ship (CI failed or changes requested).
	StatusBlocked Status = "blocked"
	// StatusIncomplete indicates the stack is missing PRs or has drafts.
	StatusIncomplete Status = "incomplete"
)

// BlockingReason describes why a PR is blocking shippability.
type BlockingReason string

const (
	// ReasonChangesRequested indicates a reviewer has requested changes.
	ReasonChangesRequested BlockingReason = "changes_requested"
	// ReasonCIFailing indicates CI checks are failing.
	ReasonCIFailing BlockingReason = "ci_failing"
	// ReasonCIPending indicates CI checks are still running.
	ReasonCIPending BlockingReason = "ci_pending"
	// ReasonDraft indicates the PR is still a draft.
	ReasonDraft BlockingReason = "draft"
	// ReasonNoPR indicates no PR exists for this branch.
	ReasonNoPR BlockingReason = "no_pr"
	// ReasonReviewRequired indicates the PR needs review approval.
	ReasonReviewRequired BlockingReason = "review_required"
)

// BlockingPR describes a PR that is blocking a stack from being shippable.
type BlockingPR struct {
	Branch   string         // Branch name
	PRNumber int            // PR number (0 if no PR)
	Reason   BlockingReason // Why this PR is blocking
}

// Stack represents a stack with its shippability analysis.
type Stack struct {
	Stack  merge.MultiStackInfo // The underlying stack
	Status Status               // Overall shippability status

	// Breakdown of shippability components
	ApprovalOK bool  // All PRs have been approved
	GitHubCIOK bool  // All GitHub CI checks are passing
	LocalCIOK  *bool // Local CI result (nil = not checked)

	// Blocking details
	BlockingPRs []BlockingPR // PRs blocking shippability

	// Compatibility (populated during combination analysis)
	CompatibleWith []string // Root branches this can ship with
	ConflictsWith  []string // Root branches this conflicts with
}

// IsShippable returns true if the stack is ready to ship.
func (s *Stack) IsShippable() bool {
	return s.Status == StatusShippable
}

// IsPending returns true if the stack is waiting on CI or review.
func (s *Stack) IsPending() bool {
	return s.Status == StatusPending
}

// IsBlocked returns true if the stack has failed CI or has changes requested.
func (s *Stack) IsBlocked() bool {
	return s.Status == StatusBlocked
}

// IsIncomplete returns true if the stack is missing PRs or has drafts.
func (s *Stack) IsIncomplete() bool {
	return s.Status == StatusIncomplete
}

// BranchCount returns the total number of branches in this stack.
func (s *Stack) BranchCount() int {
	return len(s.Stack.AllBranches)
}

// RootBranch returns the root branch name of this stack.
func (s *Stack) RootBranch() string {
	return s.Stack.RootBranch
}

// AnalysisResult contains the result of analyzing all stacks.
type AnalysisResult struct {
	Stacks          []Stack // All analyzed stacks
	ShippableCount  int     // Number of shippable stacks
	PendingCount    int     // Number of pending stacks
	BlockedCount    int     // Number of blocked stacks
	IncompleteCount int     // Number of incomplete stacks
}

// GetShippable returns only the stacks that are ready to ship.
func (r *AnalysisResult) GetShippable() []Stack {
	var result []Stack
	for _, s := range r.Stacks {
		if s.IsShippable() {
			result = append(result, s)
		}
	}
	return result
}

// GetPending returns only the stacks that are pending.
func (r *AnalysisResult) GetPending() []Stack {
	var result []Stack
	for _, s := range r.Stacks {
		if s.IsPending() {
			result = append(result, s)
		}
	}
	return result
}

// GetBlocked returns only the stacks that are blocked.
func (r *AnalysisResult) GetBlocked() []Stack {
	var result []Stack
	for _, s := range r.Stacks {
		if s.IsBlocked() {
			result = append(result, s)
		}
	}
	return result
}

// GetIncomplete returns only the stacks that are incomplete.
func (r *AnalysisResult) GetIncomplete() []Stack {
	var result []Stack
	for _, s := range r.Stacks {
		if s.IsIncomplete() {
			result = append(result, s)
		}
	}
	return result
}

// HasShippable returns true if there are any shippable stacks.
func (r *AnalysisResult) HasShippable() bool {
	return r.ShippableCount > 0
}

// TotalStacks returns the total number of stacks analyzed.
func (r *AnalysisResult) TotalStacks() int {
	return len(r.Stacks)
}
