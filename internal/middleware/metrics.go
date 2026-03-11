package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/wesleybatista/gateforge/internal/observability"
)

// Metrics returns middleware that records Prometheus metrics.
func Metrics() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path
			method := r.Method

			observability.HTTPActiveRequests.WithLabelValues(method, path).Inc()
			defer observability.HTTPActiveRequests.WithLabelValues(method, path).Dec()

			start := time.Now()
			rw := newResponseWriter(w)
			next.ServeHTTP(rw, r)
			duration := time.Since(start).Seconds()

			statusCode := fmt.Sprintf("%d", rw.statusCode)
			observability.HTTPRequestsTotal.WithLabelValues(method, path, statusCode).Inc()
			observability.HTTPRequestDuration.WithLabelValues(method, path).Observe(duration)
		})
	}
}
