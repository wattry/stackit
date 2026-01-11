// Package tui provides terminal UI utilities.
package tui

import (
	"stackit.dev/stackit/internal/tui/core"
)

// Status is an alias to core.Status for backwards compatibility.
// New code should import from tui/core directly to avoid import cycles.
type Status = core.Status

// Status constants re-exported from core for backwards compatibility.
const (
	StatusPending = core.StatusPending
	StatusActive  = core.StatusActive
	StatusDone    = core.StatusDone
	StatusError   = core.StatusError
	StatusSkipped = core.StatusSkipped
	StatusWaiting = core.StatusWaiting
)
