package foreach

// Handler receives events from the foreach action and handles user interaction.
// Implementations should handle events appropriately for their UI context
// (interactive terminal, non-interactive, dashboard, etc.)
type Handler interface {
	// OnEvent is called for each event during execution.
	// Handlers should use type switches to handle specific event types.
	OnEvent(event Event)
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

// Events returns the event channel for reading.
func (h *ChannelHandler) Events() <-chan Event {
	return h.events
}

// Close closes the event channel.
// Should be called when the action is complete.
func (h *ChannelHandler) Close() {
	close(h.events)
}
