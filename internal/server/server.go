package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
)

// Server wraps the HTTP server with graceful shutdown
type Server struct {
	httpServer *http.Server
	logger     *zerolog.Logger
}

// ServerOptions configures the HTTP server
type ServerOptions struct {
	Port            int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
}

// NewServer creates a new HTTP server with graceful shutdown
func NewServer(router http.Handler, logger *zerolog.Logger, opts ServerOptions) *Server {
	// Set defaults
	if opts.ReadTimeout == 0 {
		opts.ReadTimeout = 15 * time.Second
	}
	if opts.WriteTimeout == 0 {
		opts.WriteTimeout = 15 * time.Second
	}
	if opts.IdleTimeout == 0 {
		opts.IdleTimeout = 60 * time.Second
	}
	if opts.ShutdownTimeout == 0 {
		opts.ShutdownTimeout = 30 * time.Second
	}

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", opts.Port),
		Handler:      router,
		ReadTimeout:  opts.ReadTimeout,
		WriteTimeout: opts.WriteTimeout,
		IdleTimeout:  opts.IdleTimeout,
	}

	return &Server{
		httpServer: httpServer,
		logger:     logger,
	}
}

// Start begins serving HTTP requests with graceful shutdown
func (s *Server) Start(ctx context.Context) error {
	// Create channel for shutdown signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	// Create channel for server errors
	serverErrors := make(chan error, 1)

	// Start server in goroutine
	go func() {
		s.logger.Info().
			Str("addr", s.httpServer.Addr).
			Msg("starting http server")

		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErrors <- err
		}
	}()

	// Wait for shutdown signal, context cancellation, or server error
	select {
	case err := <-serverErrors:
		return fmt.Errorf("server error: %w", err)
	case <-stop:
		s.logger.Info().Msg("shutdown signal received")
	case <-ctx.Done():
		s.logger.Info().Msg("context cancelled, shutting down")
	}

	// Graceful shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	s.logger.Info().Msg("gracefully shutting down http server")

	if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
		s.logger.Error().Err(err).Msg("http server shutdown error")
		return err
	}

	s.logger.Info().Msg("http server stopped")
	return nil
}

// Shutdown performs graceful shutdown
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}
