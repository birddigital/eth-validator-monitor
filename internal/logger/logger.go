package logger

import (
	"context"
	"io"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/natefinch/lumberjack.v2"
)

type contextKey string

const (
	requestIDKey contextKey = "request_id"
)

var (
	// Logger is the global logger instance
	Logger zerolog.Logger
)

// Config holds logger configuration
type Config struct {
	Level      string // "debug", "info", "warn", "error"
	Format     string // "json", "console"
	OutputPath string // file path or "stdout"

	// Log rotation settings (for lumberjack)
	MaxSizeMB  int
	MaxBackups int
	MaxAgeDays int
	Compress   bool
}

// Initialize sets up the global logger based on config
func Initialize(cfg Config) error {
	// Parse log level
	level, err := zerolog.ParseLevel(strings.ToLower(cfg.Level))
	if err != nil {
		level = zerolog.InfoLevel // default to info
	}

	// Set global level
	zerolog.SetGlobalLevel(level)

	// Configure output writer
	var output io.Writer = os.Stdout

	if cfg.OutputPath != "" && cfg.OutputPath != "stdout" {
		// Use lumberjack for log rotation
		output = &lumberjack.Logger{
			Filename:   cfg.OutputPath,
			MaxSize:    cfg.MaxSizeMB,
			MaxBackups: cfg.MaxBackups,
			MaxAge:     cfg.MaxAgeDays,
			Compress:   cfg.Compress,
		}
	}

	// Configure format
	if strings.ToLower(cfg.Format) == "console" {
		output = zerolog.ConsoleWriter{
			Out:        output,
			TimeFormat: time.RFC3339,
			NoColor:    false,
		}
	}

	// Create logger with common fields
	Logger = zerolog.New(output).
		With().
		Timestamp().
		Caller().
		Logger()

	// Set as global logger
	log.Logger = Logger

	return nil
}

// WithRequestID adds a request ID to the context
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

// FromContext returns a logger with request ID from context
func FromContext(ctx context.Context) *zerolog.Logger {
	if ctx == nil {
		return &Logger
	}

	// Extract request ID from context
	if requestID, ok := ctx.Value(requestIDKey).(string); ok && requestID != "" {
		logger := Logger.With().Str("request_id", requestID).Logger()
		return &logger
	}

	return &Logger
}

// GetRequestID extracts request ID from context
func GetRequestID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if requestID, ok := ctx.Value(requestIDKey).(string); ok {
		return requestID
	}
	return ""
}
