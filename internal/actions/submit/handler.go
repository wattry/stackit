package submit

// Handler receives events from the submit action and handles user interaction.
// Implementations should handle events appropriately for their UI context
// (interactive terminal, non-interactive, dashboard, etc.)
type Handler interface {
	// OnEvent is called for each event during submission.
	// Handlers should use type switches to handle specific event types.
	OnEvent(event Event)

	// Confirm prompts for user confirmation when the --confirm flag is used.
	// Returns (confirmed, error).
	// Non-interactive handlers should return (defaultYes, nil).
	Confirm(message string, defaultYes bool) (bool, error)
}

// ChannelHandler is a Handler that sends events to a channel.
// Useful for async consumers like the dashboard.
type ChannelHandler struct {
	events chan Event
}

// NewChannelHandler creates a new ChannelHandler with a buffered channel.
func NewChannelHandler(bufferSize int) *ChannelHandler {
	return &ChannelHandler{
		events: make(chan Event, bufferSize),
	}
}

// OnEvent sends the event to the channel.
// Non-blocking: if the channel is full, the event is dropped.
func (h *ChannelHandler) OnEvent(e Event) {
	select {
	case h.events <- e:
	default:
		// Channel full, drop event to avoid blocking
	}
}

// Confirm auto-confirms with the default value.
// Dashboard operations don't support interactive confirmation.
func (h *ChannelHandler) Confirm(_ string, defaultYes bool) (bool, error) {
	return defaultYes, nil
}

// Events returns the event channel for reading.
func (h *ChannelHandler) Events() <-chan Event {
	return h.events
}

// Close closes the event channel.
// Should be called when the action is complete.
func (h *ChannelHandler) Close() {
	close(h.events)
}
