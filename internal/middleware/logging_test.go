package middleware

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLogging_LogsRequest(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	handler := Logging(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("hello"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/test?q=1", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	logOutput := buf.String()
	if !strings.Contains(logOutput, "/test") {
		t.Error("expected path in log output")
	}
	if !strings.Contains(logOutput, "GET") {
		t.Error("expected method in log output")
	}
	if !strings.Contains(logOutput, "request completed") {
		t.Error("expected 'request completed' message")
	}
}

func TestLogging_StatusCodeLevels(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		wantLevel  string
	}{
		{"2xx info", http.StatusOK, "INFO"},
		{"3xx info", http.StatusMovedPermanently, "INFO"},
		{"4xx warn", http.StatusNotFound, "WARN"},
		{"5xx error", http.StatusInternalServerError, "ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

			handler := Logging(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if !strings.Contains(buf.String(), tt.wantLevel) {
				t.Errorf("expected log level %s in output: %s", tt.wantLevel, buf.String())
			}
		})
	}
}

func TestResponseWriter_CapturesStatus(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	rw := newResponseWriter(w)

	rw.WriteHeader(http.StatusNotFound)
	if rw.statusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rw.statusCode)
	}

	// Second call should not override
	rw.WriteHeader(http.StatusOK)
	if rw.statusCode != http.StatusNotFound {
		t.Errorf("expected 404 (unchanged), got %d", rw.statusCode)
	}
}

func TestResponseWriter_CapturesBytes(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	rw := newResponseWriter(w)

	rw.Write([]byte("hello"))
	rw.Write([]byte(" world"))

	if rw.bytesWritten != 11 {
		t.Errorf("expected 11 bytes, got %d", rw.bytesWritten)
	}
}
