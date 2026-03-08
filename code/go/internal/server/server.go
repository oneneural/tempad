// Package server implements the optional HTTP API for daemon observability.
// See Spec Section 15.5.
package server

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/oneneural/tempad/internal/orchestrator"
)

// Server is the HTTP server for daemon observability.
type Server struct {
	httpServer *http.Server
	listener   net.Listener
	logger     *slog.Logger
}

// New creates a new HTTP server bound to loopback only.
// Use port 0 for an ephemeral port (tests).
func New(port int, orch *orchestrator.Orchestrator, logger *slog.Logger) (*Server, error) {
	r := chi.NewRouter()

	// Middleware.
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	// Register routes.
	h := &handlers{orch: orch}
	r.Get("/", h.dashboard)
	r.Get("/healthz", h.healthz)
	r.Get("/api/v1/state", h.getState)
	r.Get("/api/v1/{identifier}", h.getIssue)
	r.Post("/api/v1/refresh", h.triggerRefresh)

	// Bind loopback only.
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("bind %s: %w", addr, err)
	}

	return &Server{
		httpServer: &http.Server{Handler: r},
		listener:   listener,
		logger:     logger,
	}, nil
}

// Addr returns the bound address (useful for port=0 ephemeral ports).
func (s *Server) Addr() string {
	return s.listener.Addr().String()
}

// Serve starts serving HTTP requests. Blocks until the context is canceled
// or an error occurs. Performs graceful shutdown on context cancellation.
func (s *Server) Serve(ctx context.Context) error {
	s.logger.Info("HTTP server starting", "addr", s.Addr())

	// Shutdown when context is canceled.
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.httpServer.Shutdown(shutdownCtx)
	}()

	err := s.httpServer.Serve(s.listener)
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}
