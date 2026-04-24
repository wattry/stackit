package handlers

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

// EventBroadcaster manages SSE connections and broadcasts events.
type EventBroadcaster struct {
	mu         sync.RWMutex
	clients    map[chan string]struct{}
	shutdownCh chan struct{}
	closed     bool
}

// NewEventBroadcaster creates a new SSE event broadcaster.
func NewEventBroadcaster() *EventBroadcaster {
	return &EventBroadcaster{
		clients:    make(map[chan string]struct{}),
		shutdownCh: make(chan struct{}),
	}
}

// Broadcast sends an SSE event to all connected clients.
func (b *EventBroadcaster) Broadcast(event, data string) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.closed {
		return
	}

	msg := fmt.Sprintf("event: %s\ndata: %s\n\n", event, data)
	for ch := range b.clients {
		select {
		case ch <- msg:
		default:
			// Drop message if client buffer is full
		}
	}
}

// Done returns a channel that closes when the broadcaster is shutting down.
func (b *EventBroadcaster) Done() <-chan struct{} {
	return b.shutdownCh
}

// Close notifies all subscribers to exit and prevents future broadcasts.
// It closes every active client channel so SSE handlers blocked on a receive
// unblock promptly and graceful HTTP shutdown can complete.
func (b *EventBroadcaster) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return
	}
	b.closed = true
	close(b.shutdownCh)
	for ch := range b.clients {
		close(ch)
		delete(b.clients, ch)
	}
}

func (b *EventBroadcaster) subscribe() (chan string, bool) {
	ch := make(chan string, 16)
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		close(ch)
		return nil, false
	}
	b.clients[ch] = struct{}{}
	return ch, true
}

func (b *EventBroadcaster) unsubscribe(ch chan string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, ok := b.clients[ch]; !ok {
		return
	}
	delete(b.clients, ch)
	close(ch)
}

// EventsHandler serves the SSE stream at GET /api/events and /api/v1/events.
type EventsHandler struct {
	broadcaster *EventBroadcaster
}

// NewEventsHandler creates a handler for events endpoints.
func NewEventsHandler(broadcaster *EventBroadcaster) *EventsHandler {
	return &EventsHandler{broadcaster: broadcaster}
}

// ServeHTTP handles the SSE connection.
func (h *EventsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch, ok := h.broadcaster.subscribe()
	if !ok {
		http.Error(w, "server shutting down", http.StatusServiceUnavailable)
		return
	}
	defer h.broadcaster.unsubscribe(ch)

	// Send initial heartbeat
	if _, err := fmt.Fprintf(w, "event: connected\ndata: {\"timestamp\":\"%s\"}\n\n", time.Now().Format(time.RFC3339)); err != nil {
		return
	}
	flusher.Flush()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-h.broadcaster.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			if _, err := fmt.Fprint(w, msg); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}
