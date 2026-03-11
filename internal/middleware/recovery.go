package middleware

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/wesleybatista/gateforge/internal/observability"
)

// Recovery returns middleware that recovers from panics.
func Recovery(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					requestID := observability.GetRequestID(r.Context())
					stack := string(debug.Stack())

					logger.Error("panic recovered",
						slog.Any("panic", rec),
						slog.String("stack", stack),
						slog.String("request_id", requestID),
						slog.String("path", r.URL.Path),
						slog.String("method", r.Method),
					)

					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					json.NewEncoder(w).Encode(map[string]any{
						"error": map[string]any{
							"code":       "INTERNAL_ERROR",
							"message":    "an unexpected error occurred",
							"request_id": requestID,
						},
					})
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
