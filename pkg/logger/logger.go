package logger

import (
	"log/slog"
	"os"
	"strings"
)

// Logger wraps slog.Logger for structured logging
type Logger struct {
	*slog.Logger
}

// New creates a new logger instance with the specified log level
func New(level string) *Logger {
	var logLevel slog.Level

	switch strings.ToUpper(level) {
	case "DEBUG":
		logLevel = slog.LevelDebug
	case "INFO":
		logLevel = slog.LevelInfo
	case "WARN":
		logLevel = slog.LevelWarn
	case "ERROR":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: logLevel,
	}

	handler := slog.NewJSONHandler(os.Stdout, opts)
	logger := slog.New(handler)

	return &Logger{Logger: logger}
}

// WithTrxID returns a logger with transaction ID context
func (l *Logger) WithTrxID(trxID string) *Logger {
	return &Logger{
		Logger: l.With("trxid", trxID),
	}
}

// WithDestination returns a logger with destination context
func (l *Logger) WithDestination(destination string) *Logger {
	return &Logger{
		Logger: l.With("destination", destination),
	}
}

// WithError returns a logger with error context
func (l *Logger) WithError(err error) *Logger {
	return &Logger{
		Logger: l.With("error", err),
	}
}

