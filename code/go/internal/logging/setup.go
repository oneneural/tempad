// Package logging configures structured logging with slog.
// See Spec Section 15.1, 15.2.
package logging

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"gopkg.in/natefinch/lumberjack.v2"
)

// Config holds logging configuration.
type Config struct {
	Level string // "debug", "info", "warn", "error"
	File  string // log file path (daemon mode)
	Mode  string // "tui" or "daemon"
}

// Setup creates a configured slog.Logger.
// TUI mode logs to stderr with text format.
// Daemon mode logs to a rotating file with JSON format.
func Setup(cfg Config) *slog.Logger {
	level := ParseLevel(cfg.Level)
	opts := &slog.HandlerOptions{Level: level}

	var handler slog.Handler
	if cfg.Mode == "daemon" && cfg.File != "" {
		writer := newRotatingWriter(cfg.File)
		handler = slog.NewJSONHandler(writer, opts)
	} else {
		handler = slog.NewTextHandler(os.Stderr, opts)
	}

	logger := slog.New(handler).With("mode", cfg.Mode)
	return logger
}

// IssueLogger creates a per-issue logger that writes to a dedicated log file.
// Path: <logDir>/<identifier>/agent.log
func IssueLogger(baseLogDir, identifier string) *slog.Logger {
	dir := filepath.Join(baseLogDir, identifier)
	os.MkdirAll(dir, 0755)

	logPath := filepath.Join(dir, "agent.log")
	writer := &lumberjack.Logger{
		Filename:   logPath,
		MaxSize:    10, // MB
		MaxBackups: 3,
		MaxAge:     14, // days
		Compress:   true,
	}

	handler := slog.NewJSONHandler(writer, &slog.HandlerOptions{Level: slog.LevelDebug})
	return slog.New(handler).With("issue", identifier)
}

// ParseLevel converts a string log level to slog.Level.
func ParseLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// newRotatingWriter creates a lumberjack rotating writer.
func newRotatingWriter(path string) io.Writer {
	// Ensure directory exists.
	os.MkdirAll(filepath.Dir(path), 0755)

	return &lumberjack.Logger{
		Filename:   path,
		MaxSize:    50, // MB
		MaxBackups: 5,
		MaxAge:     30, // days
		Compress:   true,
	}
}
