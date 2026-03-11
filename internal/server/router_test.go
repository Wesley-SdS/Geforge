package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/wesleybatista/gateforge/internal/config"
)

func testConfig() *config.GatewayConfig {
	cfg := &config.GatewayConfig{
		Server:  config.ServerConfig{Port: 8080},
		Logging: config.LoggingConfig{Level: "info", Format: "json"},
		Metrics: config.MetricsConfig{Enabled: true, Path: "/metrics"},
		CORS: config.CORSConfig{
			AllowedOrigins: []string{"*"},
			AllowedMethods: []string{"GET", "POST"},
			MaxAge:         3600,
		},
		Routes: []config.RouteConfig{
			{
				Path:            "/api/test",
				BalanceStrategy: "round-robin",
				Targets: []config.TargetConfig{
					{URL: "http://localhost:9999", Weight: 1},
				},
			},
		},
	}
	cfg.ApplyDefaults()
	return cfg
}

func TestBuildRouter_HealthEndpoint(t *testing.T) {
	t.Parallel()

	cfg := testConfig()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := BuildRouter(cfg, logger)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	var body map[string]string
	json.NewDecoder(rec.Body).Decode(&body)
	if body["status"] != "healthy" {
		t.Errorf("expected healthy status, got %v", body)
	}
}

func TestBuildRouter_ReadinessEndpoint(t *testing.T) {
	t.Parallel()

	cfg := testConfig()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := BuildRouter(cfg, logger)

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	var body map[string]any
	json.NewDecoder(rec.Body).Decode(&body)
	if body["status"] != "ready" {
		t.Errorf("expected ready status, got %v", body)
	}
}

func TestBuildRouter_MetricsEndpoint(t *testing.T) {
	t.Parallel()

	cfg := testConfig()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := BuildRouter(cfg, logger)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func testConfigWithRateLimit() *config.GatewayConfig {
	cfg := &config.GatewayConfig{
		Server:  config.ServerConfig{Port: 8080},
		Logging: config.LoggingConfig{Level: "info", Format: "json"},
		Metrics: config.MetricsConfig{Enabled: false},
		CORS: config.CORSConfig{
			AllowedOrigins: []string{"*"},
			AllowedMethods: []string{"GET"},
			MaxAge:         3600,
		},
		Routes: []config.RouteConfig{
			{
				Path:            "/api/limited",
				BalanceStrategy: "round-robin",
				Targets: []config.TargetConfig{
					{URL: "http://localhost:9999", Weight: 1},
				},
				RateLimit: &config.RateLimitConfig{
					RequestsPerSecond: 100,
					Burst:             5,
				},
			},
		},
	}
	cfg.ApplyDefaults()
	return cfg
}

func testConfigWithCircuitBreaker() *config.GatewayConfig {
	cfg := &config.GatewayConfig{
		Server:  config.ServerConfig{Port: 8080},
		Logging: config.LoggingConfig{Level: "info", Format: "json"},
		Metrics: config.MetricsConfig{Enabled: false},
		CORS: config.CORSConfig{
			AllowedOrigins: []string{"*"},
			AllowedMethods: []string{"GET"},
			MaxAge:         3600,
		},
		Routes: []config.RouteConfig{
			{
				Path:            "/api/breaker",
				BalanceStrategy: "round-robin",
				Targets: []config.TargetConfig{
					{URL: "http://localhost:9999", Weight: 1},
				},
				CircuitBreaker: &config.CircuitBreakerConfig{
					FailureThreshold: 5,
					ResetTimeout:     30 * time.Second,
					HalfOpenMaxReqs:  2,
				},
			},
		},
	}
	cfg.ApplyDefaults()
	return cfg
}

func TestBuildRouter_WithRateLimit(t *testing.T) {
	t.Parallel()

	cfg := testConfigWithRateLimit()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := BuildRouter(cfg, logger)

	// Route should be registered — sending a request should not panic
	// The upstream is unreachable, but the middleware chain should work
	req := httptest.NewRequest(http.MethodGet, "/api/limited", nil)
	req.RemoteAddr = "10.0.0.1:5555"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// Should get a response (502 from unreachable upstream, not a panic/nil)
	if rec.Code == 0 {
		t.Error("expected a non-zero status code")
	}
}

func TestBuildRouter_WithCircuitBreaker(t *testing.T) {
	t.Parallel()

	cfg := testConfigWithCircuitBreaker()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := BuildRouter(cfg, logger)

	req := httptest.NewRequest(http.MethodGet, "/api/breaker", nil)
	req.RemoteAddr = "10.0.0.1:5555"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code == 0 {
		t.Error("expected a non-zero status code")
	}
}
