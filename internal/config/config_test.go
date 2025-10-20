package config

import (
	"os"
	"strings"
	"testing"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid configuration",
			envVars: map[string]string{
				"DB_USER":         "testuser",
				"DB_PASSWORD":     "testpass",
				"BEACON_NODE_URL": "http://localhost:5052",
			},
			wantErr: false,
		},
		{
			name: "missing DB_USER",
			envVars: map[string]string{
				"DB_PASSWORD":     "testpass",
				"BEACON_NODE_URL": "http://localhost:5052",
			},
			wantErr: true,
			errMsg:  "DB_USER is required",
		},
		{
			name: "missing DB_PASSWORD",
			envVars: map[string]string{
				"DB_USER":         "testuser",
				"BEACON_NODE_URL": "http://localhost:5052",
			},
			wantErr: true,
			errMsg:  "DB_PASSWORD is required",
		},
		{
			name: "invalid REDIS_DB",
			envVars: map[string]string{
				"DB_USER":         "testuser",
				"DB_PASSWORD":     "testpass",
				"BEACON_NODE_URL": "http://localhost:5052",
				"REDIS_DB":        "99",
			},
			wantErr: true,
			errMsg:  "REDIS_DB must be between 0 and 15",
		},
		{
			name: "invalid HTTP_PORT",
			envVars: map[string]string{
				"DB_USER":         "testuser",
				"DB_PASSWORD":     "testpass",
				"BEACON_NODE_URL": "http://localhost:5052",
				"HTTP_PORT":       "invalid",
			},
			wantErr: true,
			errMsg:  "HTTP_PORT must be a valid port number",
		},
		{
			name: "invalid GIN_MODE",
			envVars: map[string]string{
				"DB_USER":         "testuser",
				"DB_PASSWORD":     "testpass",
				"BEACON_NODE_URL": "http://localhost:5052",
				"GIN_MODE":        "invalid",
			},
			wantErr: true,
			errMsg:  "GIN_MODE must be 'debug', 'release', or 'test'",
		},
		{
			name: "invalid BEACON_NODE_URL scheme",
			envVars: map[string]string{
				"DB_USER":         "testuser",
				"DB_PASSWORD":     "testpass",
				"BEACON_NODE_URL": "ftp://localhost:5052",
			},
			wantErr: true,
			errMsg:  "BEACON_NODE_URL must use http or https scheme",
		},
		{
			name: "invalid DB_SSL_MODE",
			envVars: map[string]string{
				"DB_USER":         "testuser",
				"DB_PASSWORD":     "testpass",
				"BEACON_NODE_URL": "http://localhost:5052",
				"DB_SSL_MODE":     "invalid",
			},
			wantErr: true,
			errMsg:  "DB_SSL_MODE must be one of",
		},
		{
			name: "invalid REDIS_ADDR format",
			envVars: map[string]string{
				"DB_USER":         "testuser",
				"DB_PASSWORD":     "testpass",
				"BEACON_NODE_URL": "http://localhost:5052",
				"REDIS_ADDR":      "invalid_format",
			},
			wantErr: true,
			errMsg:  "REDIS_ADDR must be in format host:port",
		},
		{
			name: "all custom values",
			envVars: map[string]string{
				"HTTP_PORT":         "9000",
				"GIN_MODE":          "debug",
				"DB_HOST":           "db.example.com",
				"DB_PORT":           "5433",
				"DB_USER":           "customuser",
				"DB_PASSWORD":       "custompass",
				"DB_NAME":           "custom_db",
				"DB_SSL_MODE":       "require",
				"REDIS_ADDR":        "redis.example.com:6380",
				"REDIS_PASSWORD":    "redispass",
				"REDIS_DB":          "5",
				"BEACON_NODE_URL":   "https://beacon.example.com",
				"PROMETHEUS_PORT":   "9091",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment
			clearTestEnv()

			// Set test env vars
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}
			defer clearTestEnv()

			cfg, err := Load()

			if tt.wantErr {
				if err == nil {
					t.Errorf("Load() expected error, got nil")
				}
				if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("Load() error = %v, want error containing %v", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Load() unexpected error = %v", err)
				}
				if cfg == nil {
					t.Error("Load() returned nil config")
				} else {
					// Verify custom values are applied
					if tt.envVars["HTTP_PORT"] != "" && cfg.Server.HTTPPort != tt.envVars["HTTP_PORT"] {
						t.Errorf("HTTP_PORT = %v, want %v", cfg.Server.HTTPPort, tt.envVars["HTTP_PORT"])
					}
					if tt.envVars["DB_USER"] != "" && cfg.Database.User != tt.envVars["DB_USER"] {
						t.Errorf("DB_USER = %v, want %v", cfg.Database.User, tt.envVars["DB_USER"])
					}
				}
			}
		})
	}
}

func TestDatabaseConnectionString(t *testing.T) {
	cfg := &Config{
		Database: DatabaseConfig{
			Host:     "localhost",
			Port:     "5432",
			User:     "testuser",
			Password: "testpass",
			Name:     "testdb",
			SSLMode:  "disable",
		},
	}

	connStr := cfg.DatabaseConnectionString()
	expected := "host=localhost port=5432 user=testuser password=testpass dbname=testdb sslmode=disable"

	if connStr != expected {
		t.Errorf("DatabaseConnectionString() = %v, want %v", connStr, expected)
	}
}

func TestRedisConnectionString(t *testing.T) {
	cfg := &Config{
		Redis: RedisConfig{
			Addr: "localhost:6379",
		},
	}

	connStr := cfg.RedisConnectionString()
	expected := "localhost:6379"

	if connStr != expected {
		t.Errorf("RedisConnectionString() = %v, want %v", connStr, expected)
	}
}

func TestServerPort(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{
			HTTPPort: "8080",
		},
	}

	port := cfg.ServerPort()
	if port != 8080 {
		t.Errorf("ServerPort() = %v, want 8080", port)
	}
}

func TestMetricsPort(t *testing.T) {
	cfg := &Config{
		Monitoring: MonitoringConfig{
			PrometheusPort: "9090",
		},
	}

	port := cfg.MetricsPort()
	if port != 9090 {
		t.Errorf("MetricsPort() = %v, want 9090", port)
	}
}

func TestMustLoad_Panics(t *testing.T) {
	// Clear all env vars to cause validation failure
	clearTestEnv()
	defer clearTestEnv()

	defer func() {
		if r := recover(); r == nil {
			t.Error("MustLoad() should panic on invalid config")
		}
	}()

	MustLoad()
}

func clearTestEnv() {
	vars := []string{
		"HTTP_PORT", "GIN_MODE",
		"DB_HOST", "DB_PORT", "DB_USER", "DB_PASSWORD", "DB_NAME", "DB_SSL_MODE",
		"REDIS_ADDR", "REDIS_PASSWORD", "REDIS_DB",
		"BEACON_NODE_URL",
		"PROMETHEUS_PORT",
	}
	for _, v := range vars {
		os.Unsetenv(v)
	}
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
