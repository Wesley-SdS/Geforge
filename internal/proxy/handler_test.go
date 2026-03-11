package proxy

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/wesleybatista/gateforge/internal/balancer"
	"github.com/wesleybatista/gateforge/internal/config"
)

func TestHandler_ProxiesToUpstream(t *testing.T) {
	t.Parallel()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"server":     "upstream",
			"path":       r.URL.Path,
			"request_id": r.Header.Get("X-Request-ID"),
		})
	}))
	defer upstream.Close()

	targets := []balancer.Target{{URL: parseTestURL(upstream.URL), Weight: 1}}
	bal := balancer.NewRoundRobin(targets)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	routeCfg := config.RouteConfig{
		Path:    "/api/test",
		Timeout: 10 * time.Second,
	}

	handler := NewHandler(bal, routeCfg, logger)

	req := httptest.NewRequest(http.MethodGet, "/api/test/hello", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if body["server"] != "upstream" {
		t.Errorf("expected upstream response, got %v", body)
	}
}

func TestHandler_StripPrefix(t *testing.T) {
	t.Parallel()

	var gotPath string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	targets := []balancer.Target{{URL: parseTestURL(upstream.URL), Weight: 1}}
	bal := balancer.NewRoundRobin(targets)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	routeCfg := config.RouteConfig{
		Path:        "/api/users",
		StripPrefix: true,
		Timeout:     10 * time.Second,
	}

	handler := NewHandler(bal, routeCfg, logger)

	req := httptest.NewRequest(http.MethodGet, "/api/users/123", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if gotPath != "/123" {
		t.Errorf("expected stripped path /123, got %s", gotPath)
	}
}

func TestHandler_NoHealthyTargets(t *testing.T) {
	t.Parallel()

	targets := []balancer.Target{{URL: parseTestURL("http://localhost:1"), Weight: 1}}
	bal := balancer.NewRoundRobin(targets)
	bal.SetHealthy("http://localhost:1", false)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	routeCfg := config.RouteConfig{
		Path:    "/api/test",
		Timeout: 1 * time.Second,
	}

	handler := NewHandler(bal, routeCfg, logger)

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Errorf("expected 502, got %d", rec.Code)
	}
}

func TestHandler_ForwardedHeaders(t *testing.T) {
	t.Parallel()

	var gotHeaders http.Header
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeaders = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	targets := []balancer.Target{{URL: parseTestURL(upstream.URL), Weight: 1}}
	bal := balancer.NewRoundRobin(targets)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	handler := NewHandler(bal, config.RouteConfig{Path: "/api", Timeout: 10 * time.Second}, logger)

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if gotHeaders.Get("X-Forwarded-Proto") == "" {
		t.Error("expected X-Forwarded-Proto header")
	}
}

func TestHandler_Timeout(t *testing.T) {
	t.Parallel()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	targets := []balancer.Target{{URL: parseTestURL(upstream.URL), Weight: 1}}
	bal := balancer.NewRoundRobin(targets)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	routeCfg := config.RouteConfig{
		Path:    "/api/slow",
		Timeout: 50 * time.Millisecond,
	}

	handler := NewHandler(bal, routeCfg, logger)

	req := httptest.NewRequest(http.MethodGet, "/api/slow", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Errorf("expected 502 on timeout, got %d", rec.Code)
	}
}

func TestStripPrefix(t *testing.T) {
	t.Parallel()

	tests := []struct {
		path   string
		prefix string
		want   string
	}{
		{"/api/users/123", "/api/users", "/123"},
		{"/api/users", "/api/users", "/"},
		{"/api/users/", "/api/users", "/"},
	}

	for _, tt := range tests {
		got := stripPrefix(tt.path, tt.prefix)
		if got != tt.want {
			t.Errorf("stripPrefix(%q, %q) = %q, want %q", tt.path, tt.prefix, got, tt.want)
		}
	}
}

func parseTestURL(raw string) *url.URL {
	u, _ := url.Parse(raw)
	return u
}
