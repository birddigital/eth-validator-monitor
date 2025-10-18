package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/birddigital/eth-validator-monitor/internal/database"
)

// Config holds all configuration for the application
type Config struct {
	Server   ServerConfig
	Database *database.Config
	Metrics  MetricsConfig
	Redis    RedisConfig
	Beacon   BeaconConfig
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Port        int
	Environment string // "development" or "production"
	RateLimit   int    // requests per second
}

// MetricsConfig holds Prometheus metrics server configuration
type MetricsConfig struct {
	Port int
}

// RedisConfig holds Redis cache configuration
type RedisConfig struct {
	Address  string
	Password string
	DB       int
}

// BeaconConfig holds Ethereum Beacon Chain configuration
type BeaconConfig struct {
	URL string
}

// Load loads configuration from environment variables with sensible defaults
func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Port:        getEnvInt("SERVER_PORT", 8080),
			Environment: getEnv("ENVIRONMENT", "development"),
			RateLimit:   getEnvInt("RATE_LIMIT", 100),
		},
		Database: &database.Config{
			Host:                  getEnv("DB_HOST", "localhost"),
			Port:                  getEnvInt("DB_PORT", 5432),
			User:                  getEnv("DB_USER", "validator_monitor"),
			Password:              getEnv("DB_PASSWORD", ""),
			Database:              getEnv("DB_NAME", "validator_monitor"),
			SSLMode:               getEnv("DB_SSL_MODE", "prefer"),
			MaxConnections:        int32(getEnvInt("DB_MAX_CONNECTIONS", 25)),
			MinConnections:        int32(getEnvInt("DB_MIN_CONNECTIONS", 5)),
			MaxConnectionLifetime: getDurationEnv("DB_MAX_CONN_LIFETIME", time.Hour),
			MaxConnectionIdleTime: getDurationEnv("DB_MAX_CONN_IDLE_TIME", 30*time.Minute),
			HealthCheckPeriod:     getDurationEnv("DB_HEALTH_CHECK_PERIOD", time.Minute),
			ConnectTimeout:        getDurationEnv("DB_CONNECT_TIMEOUT", 5*time.Second),
		},
		Metrics: MetricsConfig{
			Port: getEnvInt("METRICS_PORT", 9090),
		},
		Redis: RedisConfig{
			Address:  getEnv("REDIS_ADDRESS", "localhost:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
		},
		Beacon: BeaconConfig{
			URL: getEnv("BEACON_URL", "http://localhost:5052"),
		},
	}

	// Validate database configuration
	if err := cfg.Database.Validate(); err != nil {
		return nil, fmt.Errorf("invalid database configuration: %w", err)
	}

	return cfg, nil
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt gets an integer environment variable or returns a default value
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

// getDurationEnv gets a duration environment variable or returns a default value
func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
