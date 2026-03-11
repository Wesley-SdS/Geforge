package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// Server wraps the HTTP server with lifecycle management.
type Server struct {
	httpServer *http.Server
	logger     *slog.Logger
}

// New creates a new Server.
func New(handler http.Handler, port int, readTimeout, writeTimeout, idleTimeout time.Duration, logger *slog.Logger) *Server {
	return &Server{
		httpServer: &http.Server{
			Addr:         fmt.Sprintf(":%d", port),
			Handler:      handler,
			ReadTimeout:  readTimeout,
			WriteTimeout: writeTimeout,
			IdleTimeout:  idleTimeout,
			ErrorLog:     slog.NewLogLogger(logger.Handler(), slog.LevelError),
		},
		logger: logger,
	}
}

// Start begins listening for connections.
func (s *Server) Start() error {
	s.logger.Info("server listening", slog.String("addr", s.httpServer.Addr))
	err := s.httpServer.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server failed: %w", err)
	}
	return nil
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("shutting down server")
	return s.httpServer.Shutdown(ctx)
}

// SetHandler replaces the server's handler (used for hot reload).
func (s *Server) SetHandler(handler http.Handler) {
	s.httpServer.Handler = handler
}
