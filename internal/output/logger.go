package output

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/natefinch/lumberjack.v2"
)

// Logger handles persistent file logging for debugging.
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	Close() error
}

// FileLogger implements Logger with file-based structured logging.
type FileLogger struct {
	logger    *slog.Logger
	logWriter io.WriteCloser
}

// NewFileLogger creates a logger that writes to the specified file with rotation.
func NewFileLogger(logFilePath string) (*FileLogger, error) {
	// Ensure log directory exists
	logDir := filepath.Dir(logFilePath)
	if err := os.MkdirAll(logDir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Create lumberjack logger for file rotation
	lumberjackLogger := createLumberjackLogger(logFilePath)

	// Create file handler with timestamps
	fileHandler := slog.NewTextHandler(lumberjackLogger, &slog.HandlerOptions{
		Level: getLogLevel(),
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{Key: a.Key, Value: slog.StringValue(a.Value.Time().Format("2006-01-02 15:04:05.000"))}
			}
			return a
		},
	})

	return &FileLogger{
		logger:    slog.New(fileHandler),
		logWriter: lumberjackLogger,
	}, nil
}

// getLogLevel returns the log level from STACKIT_LOG_LEVEL environment variable.
// Valid values are: debug, info, warn, error (case-insensitive).
// Defaults to Info if not set or invalid.
func getLogLevel() slog.Level {
	levelStr := strings.ToLower(os.Getenv("STACKIT_LOG_LEVEL"))
	switch levelStr {
	case "debug":
		return slog.LevelDebug
	case "info", "":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// Debug logs a debug message.
func (l *FileLogger) Debug(msg string, args ...any) {
	l.logger.Log(context.Background(), slog.LevelDebug, fmt.Sprintf(msg, args...))
}

// Info logs an info message.
func (l *FileLogger) Info(msg string, args ...any) {
	l.logger.Log(context.Background(), slog.LevelInfo, fmt.Sprintf(msg, args...))
}

// Warn logs a warning message.
func (l *FileLogger) Warn(msg string, args ...any) {
	l.logger.Log(context.Background(), slog.LevelWarn, fmt.Sprintf(msg, args...))
}

// Error logs an error message.
func (l *FileLogger) Error(msg string, args ...any) {
	l.logger.Log(context.Background(), slog.LevelError, fmt.Sprintf(msg, args...))
}

// Close closes the log file.
func (l *FileLogger) Close() error {
	if l.logWriter != nil {
		return l.logWriter.Close()
	}
	return nil
}

// NullLogger discards all log messages.
type NullLogger struct{}

// NewNullLogger creates a logger that discards all output.
func NewNullLogger() *NullLogger {
	return &NullLogger{}
}

// Debug discards the message.
func (l *NullLogger) Debug(_ string, _ ...any) {}

// Info discards the message.
func (l *NullLogger) Info(_ string, _ ...any) {}

// Warn discards the message.
func (l *NullLogger) Warn(_ string, _ ...any) {}

// Error discards the message.
func (l *NullLogger) Error(_ string, _ ...any) {}

// Close does nothing and returns nil.
func (l *NullLogger) Close() error { return nil }

// NewFileLoggerOrNull creates a FileLogger, falling back to NullLogger on error.
// This is useful for contexts where logging is optional (e.g., crash handlers).
func NewFileLoggerOrNull(logFilePath string) Logger {
	logger, err := NewFileLogger(logFilePath)
	if err != nil {
		return NewNullLogger()
	}
	return logger
}

// createLumberjackLogger creates a lumberjack logger with configuration from environment variables.
func createLumberjackLogger(logFilePath string) *lumberjack.Logger {
	config := &lumberjack.Logger{
		Filename:   logFilePath,
		MaxSize:    1,     // 1MB default
		MaxBackups: 2,     // Keep 2 old files
		MaxAge:     30,    // Keep for 30 days
		Compress:   false, // Don't compress
	}

	// Override with environment variables
	if maxSizeStr := os.Getenv("STACKIT_LOG_MAX_SIZE"); maxSizeStr != "" {
		if maxSize, err := strconv.Atoi(maxSizeStr); err == nil && maxSize > 0 {
			config.MaxSize = maxSize
		}
	}

	if maxBackupsStr := os.Getenv("STACKIT_LOG_MAX_BACKUPS"); maxBackupsStr != "" {
		if maxBackups, err := strconv.Atoi(maxBackupsStr); err == nil && maxBackups >= 0 {
			config.MaxBackups = maxBackups
		}
	}

	if maxAgeStr := os.Getenv("STACKIT_LOG_MAX_AGE"); maxAgeStr != "" {
		if maxAge, err := strconv.Atoi(maxAgeStr); err == nil && maxAge > 0 {
			config.MaxAge = maxAge
		}
	}

	return config
}
