package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStatusWriterImplementsFlusherWhenWrappedWriterDoes(t *testing.T) {
	t.Parallel()

	sw := &statusWriter{
		ResponseWriter: httptest.NewRecorder(),
		status:         http.StatusOK,
	}

	if _, ok := any(sw).(http.Flusher); !ok {
		t.Fatal("statusWriter must implement http.Flusher for SSE endpoints")
	}
}

func TestCORSMiddlewareAllowsConfiguredOrigin(t *testing.T) {
	t.Parallel()

	handler := corsMiddleware([]string{"http://example.com"}, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/view", nil)
	req.Header.Set("Origin", "http://example.com")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "http://example.com" {
		t.Fatalf("expected configured origin to be allowed, got %q", got)
	}
}

func TestCORSMiddlewareAllowsLocalhostAnyPort(t *testing.T) {
	t.Parallel()

	handler := corsMiddleware(nil, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/view", nil)
	req.Header.Set("Origin", "http://localhost:5100")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5100" {
		t.Fatalf("expected localhost origin to be allowed, got %q", got)
	}
}

func TestCORSMiddlewareRejectsUnconfiguredNonLocalOrigin(t *testing.T) {
	t.Parallel()

	handler := corsMiddleware(nil, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/view", nil)
	req.Header.Set("Origin", "http://example.com")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("expected non-local, unconfigured origin to be rejected, got %q", got)
	}
}

func TestIsLocalDevOrigin(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		origin string
		want   bool
	}{
		{name: "localhost with port", origin: "http://localhost:5100", want: true},
		{name: "ipv4 loopback", origin: "http://127.0.0.1:3000", want: true},
		{name: "ipv6 loopback", origin: "http://[::1]:5173", want: true},
		{name: "non-loopback host", origin: "http://example.com:3000", want: false},
		{name: "invalid", origin: "not a url", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := isLocalDevOrigin(tc.origin); got != tc.want {
				t.Fatalf("isLocalDevOrigin(%q): want %v, got %v", tc.origin, tc.want, got)
			}
		})
	}
}
