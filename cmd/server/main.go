package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/birddigital/eth-validator-monitor/internal/auth"
	"github.com/birddigital/eth-validator-monitor/internal/beacon"
	"github.com/birddigital/eth-validator-monitor/internal/cache"
	"github.com/birddigital/eth-validator-monitor/internal/collector"
	"github.com/birddigital/eth-validator-monitor/internal/config"
	"github.com/birddigital/eth-validator-monitor/internal/database"
	"github.com/birddigital/eth-validator-monitor/internal/database/repository"
	"github.com/birddigital/eth-validator-monitor/internal/logger"
	"github.com/birddigital/eth-validator-monitor/internal/metrics"
	"github.com/birddigital/eth-validator-monitor/internal/server"
	"github.com/birddigital/eth-validator-monitor/internal/services/dashboard"
	"github.com/birddigital/eth-validator-monitor/internal/services/health"
	"github.com/birddigital/eth-validator-monitor/internal/services/validators"
	"github.com/birddigital/eth-validator-monitor/internal/storage"
	"github.com/birddigital/eth-validator-monitor/internal/web"
	"github.com/birddigital/eth-validator-monitor/internal/web/handlers"
	"github.com/birddigital/eth-validator-monitor/internal/web/sse"
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
	dashboardRepo := repository.NewDashboardRepository(pool)
	validatorListRepo := repository.NewValidatorListRepository(pool)
	validatorDetailRepo := repository.NewValidatorDetailRepository(pool)
	alertRepo := repository.NewAlertRepository(pool)

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

	// Initialize SSE broadcaster (needs to be created before health monitor)
	sseBroadcaster := sse.NewBroadcaster(ctx)

	// Initialize health monitor with SSE broadcaster
	healthCfg := health.MonitorConfig{
		CheckInterval: 30 * time.Second,
	}
	// Wrap pgxpool.Pool with adapter to satisfy health.DBPinger interface
	dbPinger := newPgxPoolAdapter(pool)
	healthMonitor := health.NewMonitor(dbPinger, redisClient, sseBroadcaster, healthCfg)

	// Initialize dashboard service and handlers
	dashboardService := dashboard.NewService(dashboardRepo)
	dashboardHandler := handlers.NewDashboardHandler(dashboardService, healthMonitor)

	// Initialize validator list cache and service
	validatorListCache := cache.NewValidatorListCache(redisClient, 30*time.Second)
	validatorListService := validators.NewListService(validatorListRepo, validatorListCache)
	validatorListHandler := handlers.NewValidatorListHandler(validatorListService)

	// Initialize validator detail handler
	validatorDetailHandler := handlers.NewValidatorDetailHandler(validatorDetailRepo, logger.Logger)

	// Initialize alerts handler
	alertsHandler := handlers.NewAlertsHandler(alertRepo, logger.Logger)

	// Initialize SSE handler
	sseHandler := handlers.NewSSEHandler(ctx)

	// Initialize beacon client (mock for development)
	beaconClient := beacon.NewMockClient()
	logger.Logger.Info().Msg("Mock beacon client initialized for development")

	// Initialize Redis cache for collector
	// Parse host and port from cfg.Redis.Addr (format: "host:port")
	parts := strings.Split(cfg.Redis.Addr, ":")
	redisHost := parts[0]
	redisPort := 6379 // default Redis port
	if len(parts) > 1 {
		if p, err := strconv.Atoi(parts[1]); err == nil {
			redisPort = p
		}
	}

	cacheConfig := cache.Config{
		Host:         redisHost,
		Port:         redisPort,
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
		MaxRetries:   3,
		PoolSize:     10,
		MinIdleConns: 2,
		Strategy:     cache.DefaultTTLStrategy(),
		KeyPrefix:    "validator:",
	}

	redisCache, err := cache.NewRedisCache(cacheConfig)
	if err != nil {
		logger.Logger.Fatal().Err(err).Msg("Failed to create Redis cache")
	}

	// Initialize validator collector with SSE broadcaster
	collectorConfig := collector.DefaultCollectorConfig()
	validatorCollector := collector.NewValidatorCollector(
		ctx,
		beaconClient,
		pool,
		redisCache,
		sseBroadcaster,
		collectorConfig,
	)

	// Start collector in background
	go func() {
		logger.Logger.Info().Msg("Starting validator collector")
		if err := validatorCollector.Start(); err != nil {
			logger.Logger.Error().Err(err).Msg("Collector failed to start")
		}
	}()

	// Ensure collector stops on shutdown
	defer func() {
		logger.Logger.Info().Msg("Stopping validator collector")
		if err := validatorCollector.Stop(); err != nil {
			logger.Logger.Error().Err(err).Msg("Error stopping collector")
		}
	}()

	// Register routes
	registerRoutes(router, gqlSrv, cfg, jwtService, sessionStore, authService, authHandlers, dashboardHandler, sseHandler, validatorListHandler, validatorDetailHandler, alertsHandler, &logger.Logger)

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
	dashboardHandler *handlers.DashboardHandler,
	sseHandler *handlers.SSEHandler,
	validatorListHandler *handlers.ValidatorListHandler,
	validatorDetailHandler *handlers.ValidatorDetailHandler,
	alertsHandler *handlers.AlertsHandler,
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
	dashboardPageHandler := handlers.NewDashboardPageHandler()

	// Home page route
	r.Get("/", homeHandler.ServeHTTP)
	logger.Info().Str("url", fmt.Sprintf("http://localhost:%s/", cfg.Server.HTTPPort)).
		Msg("Home page route registered")

	// Dashboard page route
	r.Get("/dashboard", dashboardPageHandler.ServeHTTP)
	logger.Info().Str("url", fmt.Sprintf("http://localhost:%s/dashboard", cfg.Server.HTTPPort)).
		Msg("Dashboard page route registered")

	// Validator list page route
	r.Get("/validators", validatorListHandler.ServeHTTP)
	logger.Info().Str("url", fmt.Sprintf("http://localhost:%s/validators", cfg.Server.HTTPPort)).
		Msg("Validator list page route registered")

	// Validator list API route (JSON)
	r.Get("/api/validators/list", validatorListHandler.ServeJSON)
	logger.Info().Str("route", "/api/validators/list").
		Msg("Validator list JSON API route registered")

	// Validator list HTMX partial route
	r.Get("/validators/list", validatorListHandler.ServeHTTP)
	logger.Info().Str("route", "/validators/list").
		Msg("Validator list HTMX partial route registered")

	// Alerts page route
	r.Get("/alerts", alertsHandler.ServeHTTP)
	logger.Info().Str("url", fmt.Sprintf("http://localhost:%s/alerts", cfg.Server.HTTPPort)).
		Msg("Alerts page route registered")

	// Alerts batch action route
	r.Post("/alerts/batch", alertsHandler.HandleBatchAction)
	logger.Info().Str("route", "POST /alerts/batch").
		Msg("Alerts batch action route registered")

	// Alerts count route for badge
	r.Get("/alerts/count", alertsHandler.HandleAlertCount)
	logger.Info().Str("route", "/alerts/count").
		Msg("Alerts count route registered")

	// Validator detail page routes
	r.Route("/validators/{index}", func(r chi.Router) {
		r.Get("/", validatorDetailHandler.ServeHTTP)
		r.Get("/sse", validatorDetailHandler.HandleSSE)
		r.Get("/export", validatorDetailHandler.HandleExport)
		r.Get("/alerts", validatorDetailHandler.HandleAlertsPartial)
	})
	logger.Info().Str("route", "/validators/{index}/*").
		Msg("Validator detail routes registered (page, SSE, export, alerts)")

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

	// Dashboard API routes for HTMX
	r.Route("/api/dashboard", func(r chi.Router) {
		r.Get("/metrics", dashboardHandler.GetMetrics)
		r.Get("/alerts", dashboardHandler.GetAlerts)
		r.Get("/validators", dashboardHandler.GetTopValidators)
		r.Get("/health", dashboardHandler.GetSystemHealth)
		r.Get("/", dashboardHandler.GetDashboard)

		logger.Info().Str("route_group", "/api/dashboard/*").
			Msg("Dashboard API routes registered")
	})

	// SSE endpoint for real-time updates
	r.Get("/api/sse", sseHandler.ServeHTTP)
	logger.Info().Str("route", "/api/sse").
		Msg("SSE endpoint registered for real-time updates")

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
