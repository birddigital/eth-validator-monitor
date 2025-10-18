package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/birddigital/eth-validator-monitor/internal/config"
	"github.com/birddigital/eth-validator-monitor/internal/database"
	"github.com/birddigital/eth-validator-monitor/internal/metrics"
	"github.com/birddigital/eth-validator-monitor/graph"
	"github.com/birddigital/eth-validator-monitor/graph/middleware"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize database connection
	ctx := context.Background()
	pool, err := database.NewPool(ctx, cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	log.Println("✓ Database connected")

	// Initialize metrics server
	apiMetrics := metrics.NewAPIMetrics()
	metricsServer := metrics.NewMetricsServer(cfg.Metrics.Port, apiMetrics)
	go func() {
		log.Printf("🎯 Metrics server starting on :%d", cfg.Metrics.Port)
		if err := metricsServer.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Metrics server failed: %v", err)
		}
	}()

	// Initialize GraphQL server
	resolver := graph.NewResolver(pool)
	srv := handler.NewDefaultServer(graph.NewExecutableSchema(graph.Config{Resolvers: resolver}))

	// Setup HTTP server with middleware
	mux := http.NewServeMux()

	// GraphQL playground (dev only)
	if cfg.Server.Environment == "development" {
		mux.Handle("/", playground.Handler("GraphQL Playground", "/graphql"))
		log.Println("📊 GraphQL Playground available at http://localhost:8080/")
	}

	// GraphQL endpoint with middleware chain
	cors := middleware.NewCORSMiddleware([]string{"*"})
	logging := middleware.NewLoggingMiddleware(log.Printf)
	rateLimiter := middleware.NewRateLimiter(float64(cfg.Server.RateLimit), cfg.Server.RateLimit*2)

	graphqlHandler := cors.Middleware(
		logging.Middleware(
			rateLimiter.Middleware(srv),
		),
	)
	mux.Handle("/graphql", graphqlHandler)

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"healthy","service":"eth-validator-monitor","version":"0.1.0"}`)
	})

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start HTTP server
	go func() {
		log.Printf("🚀 Server starting on http://localhost:%d", cfg.Server.Port)
		log.Printf("📈 GraphQL API: http://localhost:%d/graphql", cfg.Server.Port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("\n🛑 Shutting down server...")

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	if err := metricsServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("Metrics server shutdown error: %v", err)
	}

	log.Println("✓ Server stopped gracefully")
}
