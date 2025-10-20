package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all application configuration
type Config struct {
	// Server configuration
	Server ServerConfig

	// Database configuration
	Database DatabaseConfig

	// Redis configuration
	Redis RedisConfig

	// Beacon Chain configuration
	BeaconChain BeaconChainConfig

	// Monitoring configuration
	Monitoring MonitoringConfig

	// Logging configuration
	Logging LoggingConfig

	// JWT configuration
	JWT JWTConfig

	// Session configuration
	Session SessionConfig
}

type ServerConfig struct {
	HTTPPort string // e.g., "8080"
	GinMode  string // "debug", "release", or "test"

	// Rate Limiting
	RateLimitEnabled       bool    // Enable/disable rate limiting
	RateLimitRequestsPerSec float64 // Requests per second per IP
	RateLimitBurst         int     // Burst capacity

	// CORS
	CORSEnabled        bool     // Enable/disable CORS
	CORSAllowedOrigins []string // Allowed origins (comma-separated)
	CORSAllowedMethods []string // Allowed HTTP methods
	CORSAllowedHeaders []string // Allowed headers
	CORSMaxAge         int      // Preflight cache duration in seconds
}

type DatabaseConfig struct {
	Host        string
	Port        string
	User        string
	Password    string
	Name        string
	SSLMode     string // "disable", "require", "verify-ca", "verify-full"
	SSLCert     string // Path to client certificate
	SSLKey      string // Path to client key
	SSLRootCert string // Path to root CA certificate
}

type RedisConfig struct {
	Addr     string // host:port, e.g., "localhost:6379"
	Password string // empty string if no password
	DB       int    // Redis database number (0-15)
}

type BeaconChainConfig struct {
	NodeURL string // e.g., "http://localhost:5052"
}

type MonitoringConfig struct {
	PrometheusPort string // e.g., "9090"
}

type LoggingConfig struct {
	Level      string // "debug", "info", "warn", "error"
	Format     string // "json", "console"
	OutputPath string // file path or "stdout"
	MaxSizeMB  int    // Max size in MB before rotation
	MaxBackups int    // Max number of old log files to retain
	MaxAgeDays int    // Max age in days for old log files
	Compress   bool   // Compress rotated logs
}

type JWTConfig struct {
	SecretKey            string        // JWT secret key (min 32 chars)
	Issuer               string        // Token issuer
	AccessTokenDuration  time.Duration // Access token expiration (e.g., 15m)
	RefreshTokenDuration time.Duration // Refresh token expiration (e.g., 168h)
}

type SessionConfig struct {
	SecretKey string        // Session secret key (min 32 chars, used for cookie signing)
	MaxAge    time.Duration // Session expiration (e.g., 168h = 7 days)
	Secure    bool          // Only send cookies over HTTPS
	HttpOnly  bool          // Prevent JavaScript access to cookies
	SameSite  string        // SameSite attribute: "Strict", "Lax", or "None"
}

// Load reads environment variables and returns populated Config
// It will load from .env file if present, but env vars take precedence
func Load() (*Config, error) {
	// Load .env file if it exists (ignore error if file doesn't exist)
	// Environment variables already set will NOT be overwritten
	_ = godotenv.Load()

	cfg := &Config{
		Server: ServerConfig{
			HTTPPort: getEnv("HTTP_PORT", "8080"),
			GinMode:  getEnv("GIN_MODE", "release"),

			// Rate Limiting
			RateLimitEnabled:       getEnvAsBool("RATE_LIMIT_ENABLED", true),
			RateLimitRequestsPerSec: getEnvAsFloat("RATE_LIMIT_RPS", 10.0),
			RateLimitBurst:         getEnvAsInt("RATE_LIMIT_BURST", 20),

			// CORS
			CORSEnabled:        getEnvAsBool("CORS_ENABLED", true),
			CORSAllowedOrigins: getEnvAsSlice("CORS_ALLOWED_ORIGINS", []string{"http://localhost:3000"}),
			CORSAllowedMethods: getEnvAsSlice("CORS_ALLOWED_METHODS", []string{"GET", "POST", "OPTIONS"}),
			CORSAllowedHeaders: getEnvAsSlice("CORS_ALLOWED_HEADERS", []string{"Content-Type", "Authorization", "X-API-Key", "X-Requested-With"}),
			CORSMaxAge:         getEnvAsInt("CORS_MAX_AGE", 300),
		},
		Database: DatabaseConfig{
			Host:        getEnv("DB_HOST", "localhost"),
			Port:        getEnv("DB_PORT", "5432"),
			User:        getEnv("DB_USER", ""),
			Password:    getEnv("DB_PASSWORD", ""),
			Name:        getEnv("DB_NAME", "validator_monitor"),
			SSLMode:     getEnv("DB_SSL_MODE", "require"), // Default to require SSL
			SSLCert:     getEnv("DB_SSL_CERT", ""),
			SSLKey:      getEnv("DB_SSL_KEY", ""),
			SSLRootCert: getEnv("DB_SSL_ROOT_CERT", ""),
		},
		Redis: RedisConfig{
			Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
		},
		BeaconChain: BeaconChainConfig{
			NodeURL: getEnv("BEACON_NODE_URL", "http://localhost:5052"),
		},
		Monitoring: MonitoringConfig{
			PrometheusPort: getEnv("PROMETHEUS_PORT", "9090"),
		},
		Logging: LoggingConfig{
			Level:      getEnv("LOG_LEVEL", "info"),
			Format:     getEnv("LOG_FORMAT", "json"),
			OutputPath: getEnv("LOG_OUTPUT_PATH", "stdout"),
			MaxSizeMB:  getEnvAsInt("LOG_MAX_SIZE_MB", 100),
			MaxBackups: getEnvAsInt("LOG_MAX_BACKUPS", 3),
			MaxAgeDays: getEnvAsInt("LOG_MAX_AGE_DAYS", 28),
			Compress:   getEnvAsBool("LOG_COMPRESS", true),
		},
		JWT: JWTConfig{
			SecretKey:            getEnv("JWT_SECRET_KEY", ""),
			Issuer:               getEnv("JWT_ISSUER", "eth-validator-monitor"),
			AccessTokenDuration:  getEnvAsDuration("JWT_ACCESS_TOKEN_DURATION", 15*time.Minute),
			RefreshTokenDuration: getEnvAsDuration("JWT_REFRESH_TOKEN_DURATION", 168*time.Hour),
		},
		Session: SessionConfig{
			SecretKey: getEnv("SESSION_SECRET_KEY", ""),
			MaxAge:    getEnvAsDuration("SESSION_MAX_AGE", 168*time.Hour), // 7 days default
			Secure:    getEnvAsBool("SESSION_SECURE", true),               // HTTPS only by default
			HttpOnly:  getEnvAsBool("SESSION_HTTP_ONLY", true),            // Prevent XSS by default
			SameSite:  getEnv("SESSION_SAME_SITE", "Lax"),                 // CSRF protection
		},
	}

	// Validate the configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

// MustLoad loads config or panics - useful for main.go
func MustLoad() *Config {
	cfg, err := Load()
	if err != nil {
		panic(fmt.Sprintf("Failed to load configuration: %v", err))
	}
	return cfg
}

// getEnv retrieves environment variable or returns default
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsInt retrieves environment variable as int or returns default
func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}

	return value
}

// getEnvAsBool retrieves environment variable as bool or returns default
func getEnvAsBool(key string, defaultValue bool) bool {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.ParseBool(valueStr)
	if err != nil {
		return defaultValue
	}

	return value
}

// getEnvAsDuration retrieves environment variable as duration or returns default
func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := time.ParseDuration(valueStr)
	if err != nil {
		return defaultValue
	}

	return value
}

// getEnvAsFloat retrieves environment variable as float64 or returns default
func getEnvAsFloat(key string, defaultValue float64) float64 {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		return defaultValue
	}

	return value
}

// getEnvAsSlice retrieves environment variable as string slice (comma-separated) or returns default
func getEnvAsSlice(key string, defaultValue []string) []string {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}

	// Split by comma and trim spaces
	values := []string{}
	for _, v := range splitAndTrim(valueStr, ",") {
		if v != "" {
			values = append(values, v)
		}
	}

	if len(values) == 0 {
		return defaultValue
	}

	return values
}

// splitAndTrim splits a string by delimiter and trims whitespace
func splitAndTrim(s, delimiter string) []string {
	parts := []string{}
	for _, part := range split(s, delimiter) {
		trimmed := trim(part)
		if trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return parts
}

// split is a simple string splitter
func split(s, delimiter string) []string {
	if s == "" {
		return []string{}
	}
	result := []string{}
	current := ""
	for _, char := range s {
		if string(char) == delimiter {
			result = append(result, current)
			current = ""
		} else {
			current += string(char)
		}
	}
	result = append(result, current)
	return result
}

// trim removes leading and trailing whitespace
func trim(s string) string {
	start := 0
	end := len(s)

	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}

	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}

	return s[start:end]
}

// DatabaseConnectionString builds PostgreSQL connection string
func (c *Config) DatabaseConnectionString() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Database.Host,
		c.Database.Port,
		c.Database.User,
		c.Database.Password,
		c.Database.Name,
		c.Database.SSLMode,
	)
}

// RedisConnectionString builds Redis connection address
func (c *Config) RedisConnectionString() string {
	return c.Redis.Addr
}

// ServerPort returns the HTTP port as an integer
func (c *Config) ServerPort() int {
	port, _ := strconv.Atoi(c.Server.HTTPPort)
	return port
}

// MetricsPort returns the Prometheus port as an integer
func (c *Config) MetricsPort() int {
	port, _ := strconv.Atoi(c.Monitoring.PrometheusPort)
	return port
}
