package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/wesleybatista/gateforge/internal/circuit"
	"github.com/wesleybatista/gateforge/internal/ratelimit"
)

func TestRateLimit_AllowedWithinLimit(t *testing.T) {
	t.Parallel()

	store := ratelimit.NewInMemoryStore(100, 10)
	defer store.Stop()

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	handler := RateLimit(store, "/api/test", logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if rec.Header().Get("X-RateLimit-Limit") != "10" {
		t.Errorf("expected X-RateLimit-Limit=10, got %s", rec.Header().Get("X-RateLimit-Limit"))
	}
	if rec.Header().Get("X-RateLimit-Remaining") == "" {
		t.Error("expected X-RateLimit-Remaining header")
	}
	if rec.Header().Get("X-RateLimit-Reset") == "" {
		t.Error("expected X-RateLimit-Reset header")
	}
}

func TestRateLimit_Rejected(t *testing.T) {
	t.Parallel()

	store := ratelimit.NewInMemoryStore(1, 1)
	defer store.Stop()

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	handler := RateLimit(store, "/api/test", logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// First request — allowed
	req1 := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req1.RemoteAddr = "10.0.0.1:9999"
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)

	if rec1.Code != http.StatusOK {
		t.Fatalf("first request should be allowed, got %d", rec1.Code)
	}

	// Second request — should be rejected
	req2 := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req2.RemoteAddr = "10.0.0.1:9999"
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429, got %d", rec2.Code)
	}
	if rec2.Header().Get("Retry-After") == "" {
		t.Error("expected Retry-After header")
	}
	if rec2.Header().Get("Content-Type") != "application/json" {
		t.Error("expected JSON content type")
	}

	var body map[string]any
	if err := json.NewDecoder(rec2.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	errObj, ok := body["error"].(map[string]any)
	if !ok {
		t.Fatal("expected error object")
	}
	if errObj["code"] != "RATE_LIMITED" {
		t.Errorf("expected RATE_LIMITED code, got %v", errObj["code"])
	}
}

func TestCircuitBreak_ClosedPassesThrough(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	breaker := circuit.NewBreaker(circuit.DefaultBreakerConfig(), logger)

	handler := CircuitBreak(breaker, "/api/test", logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 when circuit closed, got %d", rec.Code)
	}
}

func TestCircuitBreak_OpenReturns503(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	cfg := circuit.BreakerConfig{
		FailureThreshold: 2,
		ResetTimeout:     10 * time.Second,
		HalfOpenMaxReqs:  1,
		FailureWindow:    10 * time.Second,
	}
	breaker := circuit.NewBreaker(cfg, logger)

	// Force circuit open by causing failures
	for i := 0; i < 2; i++ {
		breaker.Execute(context.Background(), func(ctx context.Context) error {
			return errors.New("fail")
		})
	}

	if breaker.State() != circuit.StateOpen {
		t.Fatalf("expected open state, got %s", breaker.State())
	}

	handler := CircuitBreak(breaker, "/api/test", logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", rec.Code)
	}

	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	errObj, ok := body["error"].(map[string]any)
	if !ok {
		t.Fatal("expected error object")
	}
	if errObj["code"] != "SERVICE_UNAVAILABLE" {
		t.Errorf("expected SERVICE_UNAVAILABLE, got %v", errObj["code"])
	}
}
