package observability

import (
	"bytes"
	"strings"
	"testing"

	"github.com/wesleybatista/gateforge/internal/config"
)

func TestNewLoggerWithWriter_Levels(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		level     string
		logDebug  bool
		logInfo   bool
		logWarn   bool
		logError  bool
	}{
		{"debug level", "debug", true, true, true, true},
		{"info level", "info", false, true, true, true},
		{"warn level", "warn", false, false, true, true},
		{"error level", "error", false, false, false, true},
		{"default level", "unknown", false, true, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			cfg := config.LoggingConfig{Level: tt.level, Format: "json"}
			logger := NewLoggerWithWriter(cfg, &buf)

			logger.Debug("debug msg")
			logger.Info("info msg")
			logger.Warn("warn msg")
			logger.Error("error msg")

			output := buf.String()

			if tt.logDebug != strings.Contains(output, "debug msg") {
				t.Errorf("debug: expected present=%v in output", tt.logDebug)
			}
			if tt.logInfo != strings.Contains(output, "info msg") {
				t.Errorf("info: expected present=%v in output", tt.logInfo)
			}
			if tt.logWarn != strings.Contains(output, "warn msg") {
				t.Errorf("warn: expected present=%v in output", tt.logWarn)
			}
			if tt.logError != strings.Contains(output, "error msg") {
				t.Errorf("error: expected present=%v in output", tt.logError)
			}
		})
	}
}

func TestNewLoggerWithWriter_Formats(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		format   string
		contains string
	}{
		{"json format", "json", "{"},
		{"text format", "text", "level="},
		{"default format", "", "{"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			cfg := config.LoggingConfig{Level: "info", Format: tt.format}
			logger := NewLoggerWithWriter(cfg, &buf)

			logger.Info("test message")

			if !strings.Contains(buf.String(), tt.contains) {
				t.Errorf("expected output to contain %q, got: %s", tt.contains, buf.String())
			}
		})
	}
}

func TestNewLogger_ReturnsNonNil(t *testing.T) {
	t.Parallel()

	cfg := config.LoggingConfig{Level: "info", Format: "json"}
	logger := NewLogger(cfg)
	if logger == nil {
		t.Fatal("expected non-nil logger")
	}
}
