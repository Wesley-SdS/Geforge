package proxy

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/wesleybatista/gateforge/internal/balancer"
	"github.com/wesleybatista/gateforge/internal/config"
	"github.com/wesleybatista/gateforge/internal/observability"
)

// Handler is a reverse proxy handler that load-balances across upstream targets.
type Handler struct {
	balancer  balancer.Balancer
	routeCfg  config.RouteConfig
	logger    *slog.Logger
	transport http.RoundTripper
}

// NewHandler creates a new proxy handler.
func NewHandler(bal balancer.Balancer, routeCfg config.RouteConfig, logger *slog.Logger) *Handler {
	return &Handler{
		balancer:  bal,
		routeCfg:  routeCfg,
		logger:    logger,
		transport: NewTransport(DefaultTransportConfig()),
	}
}

// ServeHTTP proxies the request to an upstream target.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	target, err := h.balancer.Next()
	if err != nil {
		h.logger.Error("no healthy upstream targets",
			slog.String("route", h.routeCfg.Path),
			slog.String("error", err.Error()),
		)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"code":    "BAD_GATEWAY",
				"message": "no healthy upstream targets available",
			},
		})
		return
	}

	// Store upstream target in context for logging
	ctx := observability.WithUpstreamTarget(r.Context(), target.String())

	// Apply per-route timeout
	if h.routeCfg.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, h.routeCfg.Timeout)
		defer cancel()
	}

	r = r.WithContext(ctx)

	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = target.Scheme
			req.URL.Host = target.Host
			req.Host = target.Host

			// Strip prefix if configured
			if h.routeCfg.StripPrefix {
				req.URL.Path = stripPrefix(req.URL.Path, h.routeCfg.Path)
			}

			// Set forwarding headers
			if clientIP := req.Header.Get("X-Forwarded-For"); clientIP != "" {
				req.Header.Set("X-Forwarded-For", clientIP)
			} else {
				req.Header.Set("X-Forwarded-For", req.RemoteAddr)
			}
			req.Header.Set("X-Forwarded-Host", req.Host)
			req.Header.Set("X-Forwarded-Proto", schemeFromRequest(req))

			// Propagate request ID
			if reqID := observability.GetRequestID(req.Context()); reqID != "" {
				req.Header.Set("X-Request-ID", reqID)
			}
		},
		Transport: h.transport,
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			h.logger.Error("upstream request failed",
				slog.String("target", target.String()),
				slog.String("route", h.routeCfg.Path),
				slog.String("error", err.Error()),
			)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadGateway)
			json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]any{
					"code":    "BAD_GATEWAY",
					"message": "upstream request failed",
				},
			})
		},
	}

	proxy.ServeHTTP(w, r)
}

// stripPrefix removes the route prefix from the request path.
func stripPrefix(path, prefix string) string {
	p := strings.TrimPrefix(path, prefix)
	if p == "" {
		return "/"
	}
	if !strings.HasPrefix(p, "/") {
		return "/" + p
	}
	return p
}

func schemeFromRequest(r *http.Request) string {
	if r.TLS != nil {
		return "https"
	}
	return "http"
}
