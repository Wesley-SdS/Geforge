package ratelimit

import (
	"context"
	"net"
	"net/http"
	"strings"
	"time"
)

// Limiter evaluates whether a request should be allowed based on rate limits.
// Implementations must be safe for concurrent use.
type Limiter interface {
	Allow(ctx context.Context, key string) Decision
}

// Decision contains the result of a rate limit evaluation.
type Decision struct {
	Allowed    bool
	Limit      int
	Remaining  int
	ResetAt    time.Time
	RetryAfter time.Duration
}

// KeyExtractor determines the rate limit key from an HTTP request.
type KeyExtractor interface {
	Extract(r *http.Request) string
}

// IPKeyExtractor extracts client IP for rate limiting.
type IPKeyExtractor struct{}

// Extract returns the client IP from the request, checking proxy headers first.
func (e *IPKeyExtractor) Extract(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if idx := strings.IndexByte(xff, ','); idx != -1 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
