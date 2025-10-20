package database

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name      string
		config    *Config
		wantError bool
		errorMsg  string
	}{
		{
			name:      "valid config",
			config:    DefaultConfig(),
			wantError: false,
		},
		{
			name: "empty host",
			config: &Config{
				Host:     "",
				Port:     5432,
				User:     "user",
				Database: "db",
				SSLMode:  SSLModeRequire,
			},
			wantError: true,
			errorMsg:  "host cannot be empty",
		},
		{
			name: "invalid port - too low",
			config: &Config{
				Host:     "localhost",
				Port:     0,
				User:     "user",
				Database: "db",
				SSLMode:  SSLModeRequire,
			},
			wantError: true,
			errorMsg:  "port must be between 1 and 65535",
		},
		{
			name: "invalid port - too high",
			config: &Config{
				Host:     "localhost",
				Port:     70000,
				User:     "user",
				Database: "db",
				SSLMode:  SSLModeRequire,
			},
			wantError: true,
			errorMsg:  "port must be between 1 and 65535",
		},
		{
			name: "empty user",
			config: &Config{
				Host:     "localhost",
				Port:     5432,
				User:     "",
				Database: "db",
				SSLMode:  SSLModeRequire,
			},
			wantError: true,
			errorMsg:  "user cannot be empty",
		},
		{
			name: "empty database",
			config: &Config{
				Host:     "localhost",
				Port:     5432,
				User:     "user",
				Database: "",
				SSLMode:  SSLModeRequire,
			},
			wantError: true,
			errorMsg:  "name cannot be empty",
		},
		{
			name: "max connections less than min",
			config: &Config{
				Host:           "localhost",
				Port:           5432,
				User:           "user",
				Database:       "db",
				SSLMode:        SSLModeRequire,
				MaxConnections: 5,
				MinConnections: 10,
				ConnectTimeout: time.Second * 5,
			},
			wantError: true,
			errorMsg:  "max connections (5) must be >= min connections (10)",
		},
		{
			name: "invalid SSL mode",
			config: &Config{
				Host:           "localhost",
				Port:           5432,
				User:           "user",
				Database:       "db",
				SSLMode:        SSLMode("invalid"),
				MaxConnections: 10,
				MinConnections: 5,
				ConnectTimeout: time.Second * 5,
			},
			wantError: true,
			errorMsg:  "invalid SSL mode: invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.wantError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfig_Validate_ProductionSSLMode(t *testing.T) {
	// Save and restore ENV variable
	originalEnv := os.Getenv("ENV")
	defer func() {
		if originalEnv != "" {
			os.Setenv("ENV", originalEnv)
		} else {
			os.Unsetenv("ENV")
		}
	}()

	tests := []struct {
		name      string
		env       string
		sslMode   SSLMode
		wantError bool
		errorMsg  string
	}{
		{
			name:      "production with require - valid",
			env:       "production",
			sslMode:   SSLModeRequire,
			wantError: false,
		},
		{
			name:      "production with verify-ca - valid",
			env:       "production",
			sslMode:   SSLModeVerifyCA,
			wantError: false,
		},
		{
			name:      "production with verify-full - valid",
			env:       "production",
			sslMode:   SSLModeVerifyFull,
			wantError: false,
		},
		{
			name:      "production with disable - invalid",
			env:       "production",
			sslMode:   SSLModeDisable,
			wantError: true,
			errorMsg:  "production environment requires SSL mode",
		},
		{
			name:      "production with allow - invalid",
			env:       "production",
			sslMode:   SSLModeAllow,
			wantError: true,
			errorMsg:  "production environment requires SSL mode",
		},
		{
			name:      "production with prefer - invalid",
			env:       "production",
			sslMode:   SSLModePrefer,
			wantError: true,
			errorMsg:  "production environment requires SSL mode",
		},
		{
			name:      "development with disable - valid",
			env:       "development",
			sslMode:   SSLModeDisable,
			wantError: false,
		},
		{
			name:      "no env with disable - valid",
			env:       "",
			sslMode:   SSLModeDisable,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment
			if tt.env != "" {
				os.Setenv("ENV", tt.env)
			} else {
				os.Unsetenv("ENV")
			}

			config := &Config{
				Host:           "localhost",
				Port:           5432,
				User:           "user",
				Database:       "db",
				SSLMode:        tt.sslMode,
				MaxConnections: 10,
				MinConnections: 5,
				ConnectTimeout: time.Second * 5,
			}

			err := config.Validate()

			if tt.wantError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfig_BuildDSN(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		contains []string
	}{
		{
			name: "basic DSN",
			config: &Config{
				Host:           "localhost",
				Port:           5432,
				User:           "testuser",
				Password:       "testpass",
				Database:       "testdb",
				SSLMode:        SSLModeRequire,
				ConnectTimeout: 5 * time.Second,
			},
			contains: []string{
				"postgres://",
				"testuser",
				"testpass",
				"localhost:5432",
				"testdb",
				"sslmode=require",
				"connect_timeout=5",
			},
		},
		{
			name: "DSN with SSL certificates",
			config: &Config{
				Host:           "localhost",
				Port:           5432,
				User:           "testuser",
				Password:       "testpass",
				Database:       "testdb",
				SSLMode:        SSLModeVerifyFull,
				SSLCert:        "/path/to/client.crt",
				SSLKey:         "/path/to/client.key",
				SSLRootCert:    "/path/to/ca.crt",
				ConnectTimeout: 5 * time.Second,
			},
			contains: []string{
				"sslmode=verify-full",
				"sslcert=",
				"sslkey=",
				"sslrootcert=",
			},
		},
		{
			name: "DSN with special characters in password",
			config: &Config{
				Host:           "localhost",
				Port:           5432,
				User:           "user@domain",
				Password:       "p@ss&word#123",
				Database:       "testdb",
				SSLMode:        SSLModeRequire,
				ConnectTimeout: 5 * time.Second,
			},
			contains: []string{
				"postgres://",
				"localhost:5432",
				"testdb",
			},
		},
		{
			name: "DSN with disable SSL",
			config: &Config{
				Host:           "localhost",
				Port:           5432,
				User:           "testuser",
				Password:       "testpass",
				Database:       "testdb",
				SSLMode:        SSLModeDisable,
				ConnectTimeout: 5 * time.Second,
			},
			contains: []string{
				"sslmode=disable",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dsn := tt.config.BuildDSN()

			for _, substr := range tt.contains {
				assert.Contains(t, dsn, substr, "DSN should contain: %s", substr)
			}

			// Verify DSN starts with postgres://
			assert.True(t, len(dsn) > 0, "DSN should not be empty")
			assert.Contains(t, dsn, "postgres://", "DSN should start with postgres://")
		})
	}
}

func TestConfig_BuildPoolConfig(t *testing.T) {
	config := &Config{
		Host:                  "localhost",
		Port:                  5432,
		User:                  "testuser",
		Password:              "testpass",
		Database:              "testdb",
		SSLMode:               SSLModeRequire,
		MaxConnections:        25,
		MinConnections:        5,
		MaxConnectionLifetime: time.Hour,
		MaxConnectionIdleTime: 30 * time.Minute,
		HealthCheckPeriod:     time.Minute,
		ConnectTimeout:        5 * time.Second,
	}

	poolConfig, err := config.BuildPoolConfig()
	require.NoError(t, err)
	require.NotNil(t, poolConfig)

	// Verify pool settings
	assert.Equal(t, config.MaxConnections, poolConfig.MaxConns)
	assert.Equal(t, config.MinConnections, poolConfig.MinConns)
	assert.Equal(t, config.MaxConnectionLifetime, poolConfig.MaxConnLifetime)
	assert.Equal(t, config.MaxConnectionIdleTime, poolConfig.MaxConnIdleTime)
	assert.Equal(t, config.HealthCheckPeriod, poolConfig.HealthCheckPeriod)

	// Verify hooks are set
	assert.NotNil(t, poolConfig.BeforeConnect)
	assert.NotNil(t, poolConfig.AfterConnect)
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	assert.Equal(t, "localhost", config.Host)
	assert.Equal(t, 5432, config.Port)
	assert.Equal(t, SSLModeRequire, config.SSLMode)
	assert.Equal(t, int32(25), config.MaxConnections)
	assert.Equal(t, int32(5), config.MinConnections)
	assert.Equal(t, time.Hour, config.MaxConnectionLifetime)
	assert.Equal(t, 30*time.Minute, config.MaxConnectionIdleTime)
	assert.Equal(t, time.Minute, config.HealthCheckPeriod)
	assert.Equal(t, 5*time.Second, config.ConnectTimeout)

	// Validate default config is valid
	err := config.Validate()
	assert.NoError(t, err)
}

func TestSSLModeConstants(t *testing.T) {
	// Verify all SSL mode constants are defined
	assert.Equal(t, SSLMode("disable"), SSLModeDisable)
	assert.Equal(t, SSLMode("allow"), SSLModeAllow)
	assert.Equal(t, SSLMode("prefer"), SSLModePrefer)
	assert.Equal(t, SSLMode("require"), SSLModeRequire)
	assert.Equal(t, SSLMode("verify-ca"), SSLModeVerifyCA)
	assert.Equal(t, SSLMode("verify-full"), SSLModeVerifyFull)
}

// Integration test - requires actual database connection
func TestNewPool_SSLVerification(t *testing.T) {
	// Skip if not in integration test mode
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test - set INTEGRATION_TEST=true to run")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tests := []struct {
		name       string
		sslMode    SSLMode
		shouldFail bool
	}{
		{
			name:       "require SSL - should succeed if DB supports SSL",
			sslMode:    SSLModeRequire,
			shouldFail: false,
		},
		{
			name:       "disable SSL - may fail if DB requires SSL",
			sslMode:    SSLModeDisable,
			shouldFail: true, // Assuming test DB requires SSL
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				Host:                  os.Getenv("TEST_DB_HOST"),
				Port:                  5432,
				User:                  os.Getenv("TEST_DB_USER"),
				Password:              os.Getenv("TEST_DB_PASSWORD"),
				Database:              os.Getenv("TEST_DB_NAME"),
				SSLMode:               tt.sslMode,
				MaxConnections:        10,
				MinConnections:        2,
				MaxConnectionLifetime: time.Hour,
				MaxConnectionIdleTime: 30 * time.Minute,
				HealthCheckPeriod:     time.Minute,
				ConnectTimeout:        5 * time.Second,
			}

			pool, err := NewPool(ctx, config)

			if tt.shouldFail {
				if err == nil && pool != nil {
					pool.Close()
					t.Errorf("Expected connection to fail with sslmode=%s, but it succeeded", tt.sslMode)
				}
			} else {
				require.NoError(t, err, "Connection should succeed with sslmode=%s", tt.sslMode)
				require.NotNil(t, pool)
				defer pool.Close()

				// Verify SSL is actually in use
				var sslInUse bool
				err = pool.QueryRow(ctx, "SELECT pg_catalog.ssl_is_used()").Scan(&sslInUse)
				require.NoError(t, err)
				assert.True(t, sslInUse, "SSL should be in use")
			}
		})
	}
}
