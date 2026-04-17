// Package logger provides a structured logging module based on log/slog.
// It supports console + file output, log level filtering, and simple file rotation.
package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var (
	once    sync.Once
	logFile *os.File
)

// Init initializes the global logger with the specified level and optional file output.
// If filePath is empty, logs are written to stdout only.
// This function is safe to call multiple times; only the first call takes effect.
func Init(level string, filePath string) error {
	var initErr error

	once.Do(func() {
		lvl := parseLevel(level)

		opts := &slog.HandlerOptions{
			Level:     lvl,
			AddSource: lvl == slog.LevelDebug,
		}

		var writer io.Writer

		if filePath != "" {
			// Ensure log directory exists
			dir := filepath.Dir(filePath)
			if err := os.MkdirAll(dir, 0755); err != nil {
				initErr = fmt.Errorf("create log directory %s: %w", dir, err)
				return
			}

			f, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
			if err != nil {
				initErr = fmt.Errorf("open log file %s: %w", filePath, err)
				return
			}
			logFile = f

			// Write to both stdout and file
			writer = io.MultiWriter(os.Stdout, f)
		} else {
			writer = os.Stdout
		}

		handler := slog.NewTextHandler(writer, opts)
		logger := slog.New(handler)
		slog.SetDefault(logger)
	})

	return initErr
}

// Close closes the log file if one was opened.
func Close() {
	if logFile != nil {
		_ = logFile.Close()
	}
}

// parseLevel converts a string level name to slog.Level.
func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// --- Convenience functions wrapping slog defaults ---

// Debug logs at debug level.
func Debug(msg string, args ...any) {
	slog.Debug(msg, args...)
}

// Info logs at info level.
func Info(msg string, args ...any) {
	slog.Info(msg, args...)
}

// Warn logs at warn level.
func Warn(msg string, args ...any) {
	slog.Warn(msg, args...)
}

// Error logs at error level.
func Error(msg string, args ...any) {
	slog.Error(msg, args...)
}

// With returns a new slog.Logger with the given attributes pre-set.
func With(args ...any) *slog.Logger {
	return slog.Default().With(args...)
}
