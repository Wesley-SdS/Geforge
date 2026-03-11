package balancer

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestHealthChecker_HealthyTarget(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy"}`))
	}))
	defer srv.Close()

	targets := []Target{{URL: parseURL(srv.URL), Weight: 1}}
	bal := NewRoundRobin(targets)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	hc := NewHealthChecker(bal, logger,
		WithInterval(50*time.Millisecond),
		WithTimeout(1*time.Second),
		WithThreshold(2),
		WithHealthPath("/health"),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	go hc.Start(ctx)
	time.Sleep(150 * time.Millisecond)

	if !hc.IsHealthy(srv.URL) {
		t.Error("expected target to be healthy")
	}
}

func TestHealthChecker_UnhealthyTarget(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	targets := []Target{{URL: parseURL(srv.URL), Weight: 1}}
	bal := NewRoundRobin(targets)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	hc := NewHealthChecker(bal, logger,
		WithInterval(50*time.Millisecond),
		WithTimeout(1*time.Second),
		WithThreshold(2),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	go hc.Start(ctx)
	time.Sleep(250 * time.Millisecond)

	if hc.IsHealthy(srv.URL) {
		t.Error("expected target to be unhealthy after consecutive failures")
	}
}

func TestHealthChecker_Recovery(t *testing.T) {
	t.Parallel()

	var healthy bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if healthy {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer srv.Close()

	targets := []Target{{URL: parseURL(srv.URL), Weight: 1}}
	bal := NewRoundRobin(targets)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	hc := NewHealthChecker(bal, logger,
		WithInterval(50*time.Millisecond),
		WithThreshold(2),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	go hc.Start(ctx)

	// Let it fail
	time.Sleep(200 * time.Millisecond)
	if hc.IsHealthy(srv.URL) {
		t.Error("expected unhealthy")
	}

	// Recover
	healthy = true
	time.Sleep(200 * time.Millisecond)
	if !hc.IsHealthy(srv.URL) {
		t.Error("expected healthy after recovery")
	}
}
