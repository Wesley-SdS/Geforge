package observability

import (
	"context"
	"log/slog"
	"os"
	"testing"
)

func TestRequestID_RoundTrip(t *testing.T) {
	t.Parallel()
	ctx := WithRequestID(context.Background(), "req-123")
	if got := GetRequestID(ctx); got != "req-123" {
		t.Errorf("expected req-123, got %s", got)
	}
}

func TestRequestID_Missing(t *testing.T) {
	t.Parallel()
	if got := GetRequestID(context.Background()); got != "" {
		t.Errorf("expected empty string, got %s", got)
	}
}

func TestLogger_RoundTrip(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx := WithLogger(context.Background(), logger)
	if got := GetLogger(ctx); got != logger {
		t.Error("expected same logger instance")
	}
}

func TestLogger_Missing(t *testing.T) {
	t.Parallel()
	got := GetLogger(context.Background())
	if got == nil {
		t.Error("expected non-nil default logger")
	}
}

func TestUpstreamTarget_RoundTrip(t *testing.T) {
	t.Parallel()
	ctx := WithUpstreamTarget(context.Background(), "http://localhost:3001")
	if got := GetUpstreamTarget(ctx); got != "http://localhost:3001" {
		t.Errorf("expected http://localhost:3001, got %s", got)
	}
}

func TestUpstreamTarget_Missing(t *testing.T) {
	t.Parallel()
	if got := GetUpstreamTarget(context.Background()); got != "" {
		t.Errorf("expected empty string, got %s", got)
	}
}
