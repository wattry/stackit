// Package tui provides terminal UI utilities.
package tui

import (
	"stackit.dev/stackit/internal/tui/core"
)

// BaseModel is an alias to core.BaseModel for backwards compatibility.
// New code should import from tui/core directly to avoid import cycles.
type BaseModel = core.BaseModel

// ReadySignaler is an alias to core.ReadySignaler for backwards compatibility.
type ReadySignaler = core.ReadySignaler
