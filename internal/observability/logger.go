package observability

import (
	"io"
	"log/slog"
	"os"

	"github.com/wesleybatista/gateforge/internal/config"
)

// NewLogger creates a structured logger based on configuration.
func NewLogger(cfg config.LoggingConfig) *slog.Logger {
	return NewLoggerWithWriter(cfg, os.Stdout)
}

// NewLoggerWithWriter creates a structured logger writing to the given writer.
func NewLoggerWithWriter(cfg config.LoggingConfig, w io.Writer) *slog.Logger {
	level := parseLevel(cfg.Level)
	opts := &slog.HandlerOptions{
		Level: level,
	}

	var handler slog.Handler
	switch cfg.Format {
	case "text":
		handler = slog.NewTextHandler(w, opts)
	default:
		handler = slog.NewJSONHandler(w, opts)
	}

	return slog.New(handler)
}

func parseLevel(level string) slog.Level {
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
