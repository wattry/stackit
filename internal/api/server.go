// Package api provides the HTTP server for the stackit-web application.
package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"stackit.dev/stackit/internal/api/handlers"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/github"
)

// ServerConfig holds configuration for the API server.
type ServerConfig struct {
	Port        int
	CORSOrigins []string
	RepoRoot    string
	Remote      string
}

// Server is the stackit-web HTTP server.
type Server struct {
	config      ServerConfig
	eng         engine.Engine
	gh          github.Client
	broadcaster *handlers.EventBroadcaster
	httpServer  *http.Server
}

// NewServer creates a new API server.
func NewServer(cfg ServerConfig, eng engine.Engine, gh github.Client) *Server {
	return &Server{
		config:      cfg,
		eng:         eng,
		gh:          gh,
		broadcaster: handlers.NewEventBroadcaster(),
	}
}

// Broadcaster returns the event broadcaster for triggering SSE updates.
func (s *Server) Broadcaster() *handlers.EventBroadcaster {
	return s.broadcaster
}

// Start begins serving HTTP requests. It blocks until the server is stopped.
func (s *Server) Start() error {
	mux := http.NewServeMux()

	// Register handlers
	mux.Handle("/api/repo", handlers.NewRepoHandler(s.eng, s.gh, s.config.Remote))
	mux.Handle("/api/stacks", handlers.NewStacksHandler(s.eng, s.gh))
	mux.Handle("/api/stacks/", handlers.NewStacksHandler(s.eng, s.gh))
	mux.Handle("/api/branches", handlers.NewBranchesHandler(s.eng, s.gh))
	mux.Handle("/api/branches/", handlers.NewBranchesHandler(s.eng, s.gh))
	mux.Handle("/api/events", handlers.NewEventsHandler(s.broadcaster))

	// Wrap with middleware
	handler := corsMiddleware(s.config.CORSOrigins, mux)
	handler = loggingMiddleware(handler)

	s.httpServer = &http.Server{
		Addr:              fmt.Sprintf(":%d", s.config.Port),
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	log.Printf("stackit-web server listening on http://localhost:%d", s.config.Port)
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}
