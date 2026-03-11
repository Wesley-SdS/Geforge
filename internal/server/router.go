package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/wesleybatista/gateforge/internal/balancer"
	"github.com/wesleybatista/gateforge/internal/circuit"
	"github.com/wesleybatista/gateforge/internal/config"
	"github.com/wesleybatista/gateforge/internal/middleware"
	"github.com/wesleybatista/gateforge/internal/proxy"
	"github.com/wesleybatista/gateforge/internal/ratelimit"
)

// BuildRouter creates the HTTP handler with all routes and middleware wired.
// The provided context controls the lifecycle of background goroutines (health checkers).
func BuildRouter(ctx context.Context, cfg *config.GatewayConfig, logger *slog.Logger) http.Handler {
	mux := http.NewServeMux()

	// System endpoints
	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("GET /ready", readinessHandler(cfg))

	if cfg.Metrics.Enabled {
		mux.Handle("GET "+cfg.Metrics.Path, promhttp.Handler())
	}

	// Route endpoints
	for _, routeCfg := range cfg.Routes {
		rc := routeCfg // capture for closure

		// 1. Create balancer
		bal := balancer.NewFromConfig(rc)

		// 2. Create health checker and start in background
		hc := balancer.NewHealthChecker(bal, logger)
		go func() {
			if err := hc.Start(ctx); err != nil && ctx.Err() == nil {
				logger.Error("health checker stopped", slog.String("route", rc.Path), slog.String("error", err.Error()))
			}
		}()

		// 3. Create proxy handler
		proxyHandler := proxy.NewHandler(bal, rc, logger)

		// 4. Build middleware chain
		chain := []middleware.Middleware{
			middleware.Recovery(logger),
			middleware.RequestID(),
			middleware.Logging(logger),
			middleware.Metrics(rc.Path),
			middleware.CORS(cfg.CORS),
		}

		if rc.RateLimit != nil {
			store := ratelimit.NewInMemoryStore(rc.RateLimit.RequestsPerSecond, rc.RateLimit.Burst)
			chain = append(chain, middleware.RateLimit(store, rc.Path, logger))
		}

		if rc.CircuitBreaker != nil {
			breakerCfg := circuit.BreakerConfig{
				FailureThreshold: rc.CircuitBreaker.FailureThreshold,
				ResetTimeout:     rc.CircuitBreaker.ResetTimeout,
				HalfOpenMaxReqs:  rc.CircuitBreaker.HalfOpenMaxReqs,
				FailureWindow:    60 * time.Second,
			}
			breaker := circuit.NewBreaker(breakerCfg, logger)
			chain = append(chain, middleware.CircuitBreak(breaker, rc.Path, logger))
		}

		// 5. Compose and register
		handler := middleware.Chain(chain...)(proxyHandler)

		// Use path pattern with trailing slash for prefix matching
		pattern := rc.Path
		if !strings.HasSuffix(pattern, "/") {
			pattern += "/"
		}

		if len(rc.Methods) == 0 {
			mux.Handle(pattern, handler)
			// Also register exact path without trailing slash
			mux.Handle(rc.Path, handler)
		} else {
			for _, method := range rc.Methods {
				mux.Handle(method+" "+pattern, handler)
				mux.Handle(method+" "+rc.Path, handler)
			}
		}

		logger.Info("route registered",
			slog.String("path", rc.Path),
			slog.Int("targets", len(rc.Targets)),
			slog.String("strategy", rc.BalanceStrategy),
		)
	}

	return mux
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

func readinessHandler(cfg *config.GatewayConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"status": "ready",
			"routes": len(cfg.Routes),
		})
	}
}
