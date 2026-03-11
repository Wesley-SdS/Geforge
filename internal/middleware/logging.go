package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/wesleybatista/gateforge/internal/observability"
	"github.com/wesleybatista/gateforge/internal/ratelimit"
)

// responseWriter wraps http.ResponseWriter to capture status code and bytes.
type responseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int64
	written      bool
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.written {
		rw.statusCode = code
		rw.written = true
	}
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.statusCode = http.StatusOK
		rw.written = true
	}
	n, err := rw.ResponseWriter.Write(b)
	rw.bytesWritten += int64(n)
	return n, err
}

// Logging returns middleware that logs completed requests with slog.
func Logging(logger *slog.Logger) Middleware {
	extractor := &ratelimit.IPKeyExtractor{}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := newResponseWriter(w)

			next.ServeHTTP(rw, r)

			duration := time.Since(start)
			requestID := observability.GetRequestID(r.Context())
			upstream := observability.GetUpstreamTarget(r.Context())
			clientIP := extractor.Extract(r)

			attrs := []slog.Attr{
				slog.String("request_id", requestID),
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("query", r.URL.RawQuery),
				slog.Int("status", rw.statusCode),
				slog.Int64("bytes", rw.bytesWritten),
				slog.Duration("latency", duration),
				slog.String("client_ip", clientIP),
				slog.String("user_agent", r.UserAgent()),
			}
			if upstream != "" {
				attrs = append(attrs, slog.String("upstream", upstream))
			}

			level := slog.LevelInfo
			msg := "request completed"
			if rw.statusCode >= 500 {
				level = slog.LevelError
			} else if rw.statusCode >= 400 {
				level = slog.LevelWarn
			}

			logger.LogAttrs(r.Context(), level, msg, attrs...)
		})
	}
}
