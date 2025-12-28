// Package operations provides async, cancellable wrappers for stackit actions.
//
// Operations emit progress via channels, allowing UIs to show live progress
// and support cancellation. They wrap existing action implementations without
// modifying them.
package operations

import (
	"context"
	"sync/atomic"
)

// Status represents the current state of an operation or step.
type Status string

const (
	// StatusPending indicates the operation/step has not started.
	StatusPending Status = "pending"
	// StatusRunning indicates the operation/step is in progress.
	StatusRunning Status = "running"
	// StatusCompleted indicates the operation/step finished successfully.
	StatusCompleted Status = "completed"
	// StatusFailed indicates the operation/step failed with an error.
	StatusFailed Status = "failed"
	// StatusCanceled indicates the operation was canceled.
	StatusCanceled Status = "canceled"
	// StatusSkipped indicates the step was skipped.
	StatusSkipped Status = "skipped"
)

// Progress represents an update from a running operation.
type Progress struct {
	// OperationID uniquely identifies this operation instance.
	OperationID string

	// Status is the current state of the operation.
	Status Status

	// Step is a human-readable description of the current step.
	// Example: "Submitting feature-1"
	Step string

	// Branch is the branch currently being processed, if applicable.
	Branch string

	// Current is the 0-based index of the current step.
	Current int

	// Total is the total number of steps in the operation.
	Total int

	// Result contains step-specific output (e.g., PR URL).
	Result any

	// Error contains the error if Status is StatusFailed.
	Error error
}

// Operation represents an async, cancellable operation.
type Operation interface {
	// ID returns the unique identifier for this operation instance.
	ID() string

	// Start begins the operation and returns a channel that will receive
	// progress updates. The channel is closed when the operation completes.
	// The provided context can be used to cancel the operation.
	Start(ctx context.Context) <-chan Progress

	// Cancel requests cancellation of the operation. This is best-effort;
	// the operation may complete before cancellation takes effect.
	Cancel()
}

// operationID is a counter for generating unique operation IDs.
var operationID atomic.Uint64

// nextID generates a unique operation ID.
func nextID(prefix string) string {
	id := operationID.Add(1)
	return prefix + "-" + string(rune('0'+id%10))
}

// BaseOperation provides common functionality for operations.
type BaseOperation struct {
	id         string
	cancelFunc context.CancelFunc
}

// ID returns the operation ID.
func (o *BaseOperation) ID() string {
	return o.id
}

// Cancel cancels the operation.
func (o *BaseOperation) Cancel() {
	if o.cancelFunc != nil {
		o.cancelFunc()
	}
}

// SetCancel sets the cancel function for the operation.
func (o *BaseOperation) SetCancel(cancel context.CancelFunc) {
	o.cancelFunc = cancel
}
