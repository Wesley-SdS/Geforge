package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/wesleybatista/gateforge/internal/observability"
)

// Metrics returns middleware that records Prometheus metrics.
// Uses the configured routePath as label to prevent cardinality explosion
// from user-provided paths (e.g. /api/users/123, /api/users/456).
func Metrics(routePath string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			method := r.Method

			observability.HTTPActiveRequests.WithLabelValues(method, routePath).Inc()
			defer observability.HTTPActiveRequests.WithLabelValues(method, routePath).Dec()

			start := time.Now()
			rw := newResponseWriter(w)
			next.ServeHTTP(rw, r)
			duration := time.Since(start).Seconds()

			statusCode := fmt.Sprintf("%d", rw.statusCode)
			observability.HTTPRequestsTotal.WithLabelValues(method, routePath, statusCode).Inc()
			observability.HTTPRequestDuration.WithLabelValues(method, routePath).Observe(duration)
		})
	}
}
