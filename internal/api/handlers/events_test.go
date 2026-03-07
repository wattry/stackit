package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type signalingResponseWriter struct {
	*httptest.ResponseRecorder
	wrote chan struct{}
}

func newSignalingResponseWriter() *signalingResponseWriter {
	return &signalingResponseWriter{
		ResponseRecorder: httptest.NewRecorder(),
		wrote:            make(chan struct{}),
	}
}

func (w *signalingResponseWriter) Write(p []byte) (int, error) {
	select {
	case <-w.wrote:
	default:
		close(w.wrote)
	}
	return w.ResponseRecorder.Write(p)
}

func TestEventsHandlerReturnsWhenBroadcasterCloses(t *testing.T) {
	broadcaster := NewEventBroadcaster()
	handler := NewEventsHandler(broadcaster)

	req := httptest.NewRequest(http.MethodGet, "/api/events", nil)
	recorder := newSignalingResponseWriter()

	done := make(chan struct{})
	go func() {
		handler.ServeHTTP(recorder, req)
		close(done)
	}()

	select {
	case <-recorder.wrote:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for initial SSE write")
	}

	broadcaster.Close()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("handler did not exit after broadcaster shutdown")
	}
}
