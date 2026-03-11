package middleware

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/wesleybatista/gateforge/internal/observability"
)

const requestIDHeader = "X-Request-ID"

// RequestID returns middleware that generates or propagates request IDs.
func RequestID() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := r.Header.Get(requestIDHeader)
			if id == "" {
				id = uuid.Must(uuid.NewV7()).String()
			}

			ctx := observability.WithRequestID(r.Context(), id)
			w.Header().Set(requestIDHeader, id)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
