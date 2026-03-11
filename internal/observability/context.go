package observability

import (
	"context"
	"log/slog"
)

type contextKey int

const (
	requestIDKey contextKey = iota
	loggerKey
	upstreamTargetKey
)

// WithRequestID stores the request ID in the context.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

// GetRequestID retrieves the request ID from the context.
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		return id
	}
	return ""
}

// WithLogger stores a logger in the context.
func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

// GetLogger retrieves the logger from the context, or returns the default.
func GetLogger(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(loggerKey).(*slog.Logger); ok {
		return l
	}
	return slog.Default()
}

// WithUpstreamTarget stores the upstream target URL in the context.
func WithUpstreamTarget(ctx context.Context, target string) context.Context {
	return context.WithValue(ctx, upstreamTargetKey, target)
}

// GetUpstreamTarget retrieves the upstream target URL from the context.
func GetUpstreamTarget(ctx context.Context) string {
	if t, ok := ctx.Value(upstreamTargetKey).(string); ok {
		return t
	}
	return ""
}
