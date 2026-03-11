package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/wesleybatista/gateforge/internal/circuit"
	"github.com/wesleybatista/gateforge/internal/observability"
	"github.com/wesleybatista/gateforge/internal/ratelimit"
)

// RateLimit returns middleware that enforces rate limiting.
func RateLimit(store ratelimit.Store, routePath string, logger *slog.Logger) Middleware {
	extractor := &ratelimit.IPKeyExtractor{}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := extractor.Extract(r)
			decision := store.Allow(r.Context(), key)

			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", decision.Limit))
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", decision.Remaining))
			w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", decision.ResetAt.Unix()))

			if !decision.Allowed {
				retryAfterSecs := int(decision.RetryAfter.Seconds()) + 1
				w.Header().Set("Retry-After", fmt.Sprintf("%d", retryAfterSecs))

				observability.RateLimitRejections.WithLabelValues(routePath, key).Inc()

				logger.Warn("rate limit exceeded",
					slog.String("client_ip", key),
					slog.String("route", routePath),
					slog.Duration("retry_after", decision.RetryAfter),
				)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				json.NewEncoder(w).Encode(map[string]any{
					"error": map[string]any{
						"code":               "RATE_LIMITED",
						"message":            "rate limit exceeded",
						"retry_after_seconds": retryAfterSecs,
					},
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// CircuitBreak returns middleware that applies circuit breaking.
func CircuitBreak(breaker circuit.Breaker, routePath string, logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Update metrics
			observability.CircuitBreakerState.WithLabelValues(routePath).Set(float64(breaker.State()))

			err := breaker.Execute(r.Context(), func(ctx context.Context) error {
				rw := newResponseWriter(w)
				next.ServeHTTP(rw, r.WithContext(ctx))
				if rw.statusCode >= 500 {
					return fmt.Errorf("upstream returned %d", rw.statusCode)
				}
				return nil
			})

			if err == circuit.ErrCircuitOpen {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusServiceUnavailable)
				json.NewEncoder(w).Encode(map[string]any{
					"error": map[string]any{
						"code":    "SERVICE_UNAVAILABLE",
						"message": "service temporarily unavailable, please try again later",
					},
				})
			}
		})
	}
}
