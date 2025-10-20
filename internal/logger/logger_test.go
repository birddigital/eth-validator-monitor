package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitialize(t *testing.T) {
	tests := []struct {
		name          string
		config        Config
		expectedLevel zerolog.Level
		expectJSON    bool
		expectError   bool
	}{
		{
			name: "json_production_config",
			config: Config{
				Level:  "info",
				Format: "json",
			},
			expectedLevel: zerolog.InfoLevel,
			expectJSON:    true,
			expectError:   false,
		},
		{
			name: "console_development_config",
			config: Config{
				Level:  "debug",
				Format: "console",
			},
			expectedLevel: zerolog.DebugLevel,
			expectJSON:    false,
			expectError:   false,
		},
		{
			name: "invalid_level_defaults_to_info",
			config: Config{
				Level:  "invalid",
				Format: "json",
			},
			expectedLevel: zerolog.InfoLevel,
			expectJSON:    true,
			expectError:   false,
		},
		{
			name: "warn_level",
			config: Config{
				Level:  "warn",
				Format: "json",
			},
			expectedLevel: zerolog.WarnLevel,
			expectJSON:    true,
			expectError:   false,
		},
		{
			name: "error_level",
			config: Config{
				Level:  "error",
				Format: "json",
			},
			expectedLevel: zerolog.ErrorLevel,
			expectJSON:    true,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Initialize(tt.config)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedLevel, zerolog.GlobalLevel())
		})
	}
}

func TestFromContext(t *testing.T) {
	// Set global log level for tests
	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	tests := []struct {
		name              string
		setupContext      func() context.Context
		expectedRequestID string
	}{
		{
			name: "context_with_request_id",
			setupContext: func() context.Context {
				return WithRequestID(context.Background(), "test-request-123")
			},
			expectedRequestID: "test-request-123",
		},
		{
			name: "context_without_request_id",
			setupContext: func() context.Context {
				return context.Background()
			},
			expectedRequestID: "",
		},
		{
			name: "nil_context",
			setupContext: func() context.Context {
				return nil
			},
			expectedRequestID: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture log output for this test
			var buf bytes.Buffer
			Logger = zerolog.New(&buf).With().Timestamp().Logger()

			ctx := tt.setupContext()

			log := FromContext(ctx)
			log.Info().Msg("test message")

			// Get output and trim whitespace
			output := bytes.TrimSpace(buf.Bytes())
			if len(output) == 0 {
				t.Fatal("No log output captured")
			}

			// Parse logged JSON
			var logEntry map[string]interface{}
			err := json.Unmarshal(output, &logEntry)
			require.NoError(t, err)

			if tt.expectedRequestID != "" {
				assert.Equal(t, tt.expectedRequestID, logEntry["request_id"])
			} else {
				_, hasRequestID := logEntry["request_id"]
				assert.False(t, hasRequestID)
			}
		})
	}
}

func TestGetRequestID(t *testing.T) {
	tests := []struct {
		name     string
		ctx      context.Context
		expected string
	}{
		{
			name:     "with_request_id",
			ctx:      WithRequestID(context.Background(), "req-456"),
			expected: "req-456",
		},
		{
			name:     "without_request_id",
			ctx:      context.Background(),
			expected: "",
		},
		{
			name:     "nil_context",
			ctx:      nil,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetRequestID(tt.ctx)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestWithRequestID(t *testing.T) {
	ctx := context.Background()
	requestID := "test-req-789"

	newCtx := WithRequestID(ctx, requestID)

	// Verify the request ID was added to context
	extractedID := GetRequestID(newCtx)
	assert.Equal(t, requestID, extractedID)

	// Verify original context is unchanged
	originalID := GetRequestID(ctx)
	assert.Empty(t, originalID)
}

func TestLoggerOutput(t *testing.T) {
	// Set global log level for tests
	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	var buf bytes.Buffer
	Logger = zerolog.New(&buf).With().Timestamp().Logger()

	testMsg := "test log message"
	Logger.Info().Msg(testMsg)

	// Get output and trim whitespace
	output := bytes.TrimSpace(buf.Bytes())
	require.NotEmpty(t, output, "No log output captured")

	// Verify output is valid JSON
	var logEntry map[string]interface{}
	err := json.Unmarshal(output, &logEntry)
	require.NoError(t, err)

	// Verify message is present
	assert.Equal(t, testMsg, logEntry["message"])

	// Verify level is correct
	assert.Equal(t, "info", logEntry["level"])

	// Verify timestamp is present
	assert.NotEmpty(t, logEntry["time"])
}

func TestContextAwareLogging(t *testing.T) {
	// Set global log level for tests
	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	var buf bytes.Buffer
	Logger = zerolog.New(&buf).With().Timestamp().Logger()

	ctx := WithRequestID(context.Background(), "ctx-req-123")
	log := FromContext(ctx)

	log.Info().Str("key", "value").Msg("context test")

	// Get output and trim whitespace
	output := bytes.TrimSpace(buf.Bytes())
	require.NotEmpty(t, output, "No log output captured")

	var logEntry map[string]interface{}
	err := json.Unmarshal(output, &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "ctx-req-123", logEntry["request_id"])
	assert.Equal(t, "value", logEntry["key"])
	assert.Equal(t, "context test", logEntry["message"])
}

func TestInitialize_FileOutput(t *testing.T) {
	// Create temp directory for test logs
	tmpDir := t.TempDir()
	logPath := tmpDir + "/test.log"

	cfg := Config{
		Level:      "info",
		Format:     "json",
		OutputPath: logPath,
		MaxSizeMB:  10,
		MaxBackups: 2,
		MaxAgeDays: 7,
		Compress:   false,
	}

	err := Initialize(cfg)
	require.NoError(t, err)

	// Write some log entries
	Logger.Info().Str("test", "value").Msg("test message")
	Logger.Warn().Int("number", 42).Msg("warning message")

	// Give the logger time to flush
	// (lumberjack writes are synchronous, but just to be safe)
	require.Eventually(t, func() bool {
		info, err := os.Stat(logPath)
		if err != nil {
			return false
		}
		return info.Size() > 0
	}, 1*time.Second, 100*time.Millisecond, "log file should be created and contain data")
}

func TestInitialize_ConsoleOutput(t *testing.T) {
	// Test that console format doesn't create files
	cfg := Config{
		Level:      "debug",
		Format:     "console",
		OutputPath: "stdout",
	}

	err := Initialize(cfg)
	require.NoError(t, err)
	assert.Equal(t, zerolog.DebugLevel, zerolog.GlobalLevel())
}

func TestInitialize_LumberjackConfiguration(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name   string
		config Config
	}{
		{
			name: "with_compression",
			config: Config{
				Level:      "info",
				Format:     "json",
				OutputPath: tmpDir + "/compressed.log",
				MaxSizeMB:  50,
				MaxBackups: 5,
				MaxAgeDays: 14,
				Compress:   true,
			},
		},
		{
			name: "without_compression",
			config: Config{
				Level:      "info",
				Format:     "json",
				OutputPath: tmpDir + "/uncompressed.log",
				MaxSizeMB:  100,
				MaxBackups: 3,
				MaxAgeDays: 28,
				Compress:   false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Initialize(tt.config)
			require.NoError(t, err)

			// Write a log entry
			Logger.Info().
				Str("config", tt.name).
				Int("max_size", tt.config.MaxSizeMB).
				Int("max_backups", tt.config.MaxBackups).
				Bool("compress", tt.config.Compress).
				Msg("lumberjack configuration test")

			// Verify log file exists and has content
			info, err := os.Stat(tt.config.OutputPath)
			require.NoError(t, err, "log file should be created")
			assert.Greater(t, info.Size(), int64(0), "log file should contain data")
		})
	}
}
