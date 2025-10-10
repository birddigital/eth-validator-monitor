package metrics

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"runtime"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Server provides an HTTP server for exposing Prometheus metrics
type Server struct {
	port          int
	server        *http.Server
	apiMetrics    *APIMetrics
	updateMetrics bool
}

// NewMetricsServer creates a new metrics server
func NewMetricsServer(port int, apiMetrics *APIMetrics) *Server {
	mux := http.NewServeMux()

	// Prometheus metrics endpoint
	mux.Handle("/metrics", promhttp.Handler())

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	return &Server{
		port: port,
		server: &http.Server{
			Addr:         fmt.Sprintf(":%d", port),
			Handler:      mux,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		apiMetrics:    apiMetrics,
		updateMetrics: true,
	}
}

// Start begins serving metrics on the configured port
func (s *Server) Start() error {
	// Start background goroutine to update system metrics
	go s.updateSystemMetrics()

	log.Printf("Starting metrics server on port %d", s.port)
	log.Printf("Metrics available at http://localhost:%d/metrics", s.port)

	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start metrics server: %w", err)
	}

	return nil
}

// Shutdown gracefully shuts down the metrics server
func (s *Server) Shutdown(ctx context.Context) error {
	s.updateMetrics = false
	return s.server.Shutdown(ctx)
}

// updateSystemMetrics periodically collects and updates system metrics
func (s *Server) updateSystemMetrics() {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for s.updateMetrics {
		select {
		case <-ticker.C:
			s.collectSystemMetrics()
		}
	}
}

// collectSystemMetrics gathers current system resource metrics
func (s *Server) collectSystemMetrics() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	goroutines := runtime.NumGoroutine()

	// Update metrics
	if s.apiMetrics != nil {
		s.apiMetrics.UpdateSystemMetrics(
			goroutines,
			m.Alloc,
			m.Sys,
			0, // CPU usage requires syscall, placeholder for now
		)

		// Record GC pause if there was one
		if m.PauseNs[(m.NumGC+255)%256] > 0 {
			pauseSeconds := float64(m.PauseNs[(m.NumGC+255)%256]) / 1e9
			s.apiMetrics.RecordGCPause("stop-the-world", pauseSeconds)
		}
	}
}
