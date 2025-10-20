package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/birddigital/eth-validator-monitor/internal/auth"
	"github.com/birddigital/eth-validator-monitor/internal/config"
	"github.com/birddigital/eth-validator-monitor/internal/database"
	"github.com/birddigital/eth-validator-monitor/internal/logger"
	"github.com/birddigital/eth-validator-monitor/internal/metrics"
	"github.com/birddigital/eth-validator-monitor/internal/server"
	"github.com/birddigital/eth-validator-monitor/internal/storage"
	"github.com/birddigital/eth-validator-monitor/internal/web"
	"github.com/birddigital/eth-validator-monitor/internal/web/handlers"
	"github.com/birddigital/eth-validator-monitor/graph"
	"github.com/birddigital/eth-validator-monitor/graph/middleware"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		// Use basic logging before logger is initialized
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	loggerCfg := logger.Config{
		Level:      cfg.Logging.Level,
		Format:     cfg.Logging.Format,
		OutputPath: cfg.Logging.OutputPath,
		MaxSizeMB:  cfg.Logging.MaxSizeMB,
		MaxBackups: cfg.Logging.MaxBackups,
		MaxAgeDays: cfg.Logging.MaxAgeDays,
		Compress:   cfg.Logging.Compress,
	}

	if err := logger.Initialize(loggerCfg); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	logger.Logger.Info().
		Str("log_level", cfg.Logging.Level).
		Str("log_format", cfg.Logging.Format).
		Msg("Starting Ethereum Validator Monitor")

	// Initialize database connection
	ctx := context.Background()

	// Convert simple DatabaseConfig to full database.Config with connection pool settings
	dbCfg := database.DefaultConfig()
	dbCfg.Host = cfg.Database.Host
	dbCfg.Port, _ = strconv.Atoi(cfg.Database.Port)
	dbCfg.User = cfg.Database.User
	dbCfg.Password = cfg.Database.Password
	dbCfg.Database = cfg.Database.Name
	dbCfg.SSLMode = database.SSLMode(cfg.Database.SSLMode)

	pool, err := database.NewPool(ctx, dbCfg)
	if err != nil {
		logger.Logger.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer pool.Close()

	logger.Logger.Info().Msg("Database connected")

	// Initialize Redis client for sessions
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	// Verify Redis connection
	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.Logger.Fatal().Err(err).Msg("Failed to connect to Redis")
	}
	defer redisClient.Close()
	logger.Logger.Info().Msg("Redis connected")

	// Initialize repositories
	userRepo := storage.NewUserRepository(pool)

	// Initialize JWT service (optional - only if secret key is configured)
	var jwtService *auth.JWTService
	if cfg.JWT.SecretKey != "" {
		jwtService = auth.NewJWTService(
			cfg.JWT.SecretKey,
			cfg.JWT.Issuer,
			cfg.JWT.AccessTokenDuration,
			cfg.JWT.RefreshTokenDuration,
		)
		logger.Logger.Info().Msg("JWT authentication enabled")
	} else {
		logger.Logger.Warn().Msg("JWT_SECRET_KEY not set - JWT authentication disabled")
	}

	// Initialize session store (optional - only if secret key is configured)
	var sessionStore *auth.SessionStore
	var authService *auth.Service
	var authHandlers *server.AuthHandlers
	if cfg.Session.SecretKey != "" {
		maxAgeSeconds := int(cfg.Session.MaxAge.Seconds())
		sessionStore, err = auth.NewSessionStore(
			redisClient,
			cfg.Session.SecretKey,
			maxAgeSeconds,
			cfg.Session.Secure,
			cfg.Session.HttpOnly,
			cfg.Session.SameSite,
		)
		if err != nil {
			logger.Logger.Fatal().Err(err).Msg("Failed to create session store")
		}

		authService = auth.NewService(userRepo)
		authHandlers = server.NewAuthHandlers(authService, sessionStore)
		logger.Logger.Info().Msg("Session-based authentication enabled")
	} else {
		logger.Logger.Warn().Msg("SESSION_SECRET_KEY not set - session authentication disabled")
	}

	// Initialize metrics server
	apiMetrics := metrics.NewAPIMetrics()
	metricsServer := metrics.NewMetricsServer(cfg.MetricsPort(), apiMetrics)
	go func() {
		logger.Logger.Info().Int("port", cfg.MetricsPort()).Msg("Metrics server starting")
		if err := metricsServer.Start(); err != nil && err != http.ErrServerClosed {
			logger.Logger.Fatal().Err(err).Msg("Metrics server failed")
		}
	}()

	// Initialize GraphQL server with auth dependencies
	resolver := graph.NewResolverWithAuth(pool, userRepo, jwtService, cfg, &logger.Logger)
	gqlSrv := handler.NewDefaultServer(graph.NewExecutableSchema(graph.Config{Resolvers: resolver}))

	// Create Chi router with core middleware
	routerCfg := server.RouterConfig{
		Logger:         &logger.Logger,
		Environment:    cfg.Server.GinMode,
		EnableCORS:     cfg.Server.CORSEnabled,
		AllowedOrigins: cfg.Server.CORSAllowedOrigins,
		CompressLevel:  5, // Medium compression
		RateLimitRPS:   int(cfg.Server.RateLimitRequestsPerSec),
		RateLimitBurst: cfg.Server.RateLimitBurst,
	}

	router := server.NewRouter(routerCfg)

	// Register routes
	registerRoutes(router, gqlSrv, cfg, jwtService, sessionStore, authService, authHandlers, &logger.Logger)

	// Create HTTP server with graceful shutdown
	port, _ := strconv.Atoi(cfg.Server.HTTPPort)
	httpServer := server.NewServer(router, &logger.Logger, server.ServerOptions{
		Port: port,
	})

	logger.Logger.Info().
		Str("http_url", fmt.Sprintf("http://localhost:%s", cfg.Server.HTTPPort)).
		Str("graphql_url", fmt.Sprintf("http://localhost:%s/graphql", cfg.Server.HTTPPort)).
		Msg("Server starting")

	// Start HTTP server with graceful shutdown (blocks until shutdown signal)
	if err := httpServer.Start(ctx); err != nil {
		logger.Logger.Fatal().Err(err).Msg("Server failed")
	}

	// Shutdown metrics server
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := metricsServer.Shutdown(shutdownCtx); err != nil {
		logger.Logger.Error().Err(err).Msg("Metrics server shutdown error")
	}

	logger.Logger.Info().Msg("Server stopped gracefully")
}

// registerRoutes sets up all application routes
func registerRoutes(
	r chi.Router,
	gqlHandler http.Handler,
	cfg *config.Config,
	jwtService *auth.JWTService,
	sessionStore *auth.SessionStore,
	authService *auth.Service,
	authHandlers *server.AuthHandlers,
	logger *zerolog.Logger,
) {
	// Health check endpoint (no additional middleware needed - router already has security headers)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"healthy","service":"eth-validator-monitor","version":"0.1.0"}`)
	})

	// Session-based authentication routes (if session store is configured)
	if sessionStore != nil && authHandlers != nil {
		r.Route("/api/auth", func(r chi.Router) {
			// Add session middleware to all auth routes
			r.Use(auth.SessionMiddleware(sessionStore))

			// Public auth endpoints (no authentication required)
			r.Post("/register", authHandlers.Register)
			r.Post("/login", authHandlers.Login)
			r.Post("/logout", authHandlers.Logout)

			// Protected endpoints (authentication required)
			r.With(auth.RequireSessionAuth).Get("/me", authHandlers.Me)
		})

		logger.Info().Str("route_group", "/api/auth/*").
			Msg("Session authentication routes registered")
	}

	// GraphQL routes group with optional auth
	r.Group(func(r chi.Router) {
		// Add auth middleware if JWT is configured
		if jwtService != nil {
			authMiddleware := middleware.NewAuthMiddleware(jwtService, logger)
			r.Use(authMiddleware.Middleware)
		}

		// GraphQL endpoint
		r.Handle("/graphql", gqlHandler)
	})

	// HTML Page Routes (always available)
	homeHandler := handlers.NewHomeHandler()
	loginHandler := handlers.NewLoginHandler()
	registerHandler := handlers.NewRegisterHandler()

	// Home page route
	r.Get("/", homeHandler.ServeHTTP)
	logger.Info().Str("url", fmt.Sprintf("http://localhost:%s/", cfg.Server.HTTPPort)).
		Msg("Home page route registered")

	// Login page routes
	r.Get("/login", loginHandler.ServeHTTP)
	logger.Info().Str("url", fmt.Sprintf("http://localhost:%s/login", cfg.Server.HTTPPort)).
		Msg("Login page route registered")

	// Registration page routes
	r.Get("/register", registerHandler.ServeHTTP)
	logger.Info().Str("url", fmt.Sprintf("http://localhost:%s/register", cfg.Server.HTTPPort)).
		Msg("Registration page route registered")

	// HTML Form submission routes (require session store and auth service)
	if sessionStore != nil && authService != nil {
		loginPostHandler := handlers.NewLoginPostHandler(authService, sessionStore)
		registerPostHandler := handlers.NewRegisterPostHandler(authService, sessionStore)

		r.Post("/login", loginPostHandler.ServeHTTP)
		logger.Info().Str("route", "POST /login").Msg("Login form submission route registered")

		r.Post("/register", registerPostHandler.ServeHTTP)
		logger.Info().Str("route", "POST /register").Msg("Registration form submission route registered")
	}

	// HTMX routes group for partial page updates
	r.Route("/api/htmx", func(r chi.Router) {
		// All routes in this group automatically have HTMX middleware from the router stack
		// Handlers in this group should check middleware.IsHTMXRequest(r.Context())
		// to determine whether to return full HTML pages or just fragments

		// Example handler demonstrating content negotiation
		htmxExampleHandler := handlers.NewHTMXExampleHandler()
		r.Get("/dashboard", htmxExampleHandler.ServeHTTP)

		// Future HTMX endpoints:
		// r.Get("/validators", validatorsPartialHandler.ServeHTTP)
		// r.Get("/metrics", metricsPartialHandler.ServeHTTP)
		// r.Get("/validator/{id}", validatorDetailPartialHandler.ServeHTTP)

		logger.Info().Str("route_group", "/api/htmx/*").
			Str("example_route", "/api/htmx/dashboard").
			Msg("HTMX route group registered with example handler")
	})

	// GraphQL Playground (dev only) - accessible at /playground in debug mode
	if cfg.Server.GinMode == "debug" {
		playgroundHandler := playground.Handler("GraphQL Playground", "/graphql")
		r.Get("/playground", playgroundHandler)
		logger.Info().Str("url", fmt.Sprintf("http://localhost:%s/playground", cfg.Server.HTTPPort)).
			Msg("GraphQL Playground available")
	}

	// Static file serving with cache headers
	workDir, _ := os.Getwd()
	staticDir := http.Dir(fmt.Sprintf("%s/web/static", workDir))
	fileServer := http.FileServer(staticDir)

	// Serve static files with 1-year cache (31536000 seconds)
	r.With(web.CacheControl(31536000)).Handle("/static/*",
		http.StripPrefix("/static/", fileServer))

	logger.Info().Str("path", "/static/*").Msg("Static file serving configured")
}
