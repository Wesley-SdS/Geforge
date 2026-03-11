package balancer

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// HealthStatus tracks a target's health state.
type HealthStatus struct {
	Healthy            bool
	ConsecutiveFailures int
	LastChecked        time.Time
	LastError          error
}

// healthSetter is implemented by balancers that support health updates.
type healthSetter interface {
	SetHealthy(targetURL string, healthy bool)
}

// HealthCheckerImpl performs periodic health checks on upstream targets.
type HealthCheckerImpl struct {
	balancer  Balancer
	logger    *slog.Logger
	interval  time.Duration
	timeout   time.Duration
	threshold int // consecutive failures before marking unhealthy
	client    *http.Client
	healthPath string

	mu       sync.RWMutex
	statuses map[string]*HealthStatus
}

// HealthCheckerOption configures the health checker.
type HealthCheckerOption func(*HealthCheckerImpl)

// WithInterval sets the health check interval.
func WithInterval(d time.Duration) HealthCheckerOption {
	return func(h *HealthCheckerImpl) { h.interval = d }
}

// WithTimeout sets the health check HTTP timeout.
func WithTimeout(d time.Duration) HealthCheckerOption {
	return func(h *HealthCheckerImpl) { h.timeout = d }
}

// WithThreshold sets the consecutive failure threshold.
func WithThreshold(n int) HealthCheckerOption {
	return func(h *HealthCheckerImpl) { h.threshold = n }
}

// WithHealthPath sets the health check endpoint path.
func WithHealthPath(path string) HealthCheckerOption {
	return func(h *HealthCheckerImpl) { h.healthPath = path }
}

// NewHealthChecker creates a new health checker for the given balancer.
func NewHealthChecker(bal Balancer, logger *slog.Logger, opts ...HealthCheckerOption) *HealthCheckerImpl {
	hc := &HealthCheckerImpl{
		balancer:   bal,
		logger:     logger,
		interval:   10 * time.Second,
		timeout:    5 * time.Second,
		threshold:  3,
		healthPath: "/health",
		statuses:   make(map[string]*HealthStatus),
	}

	for _, opt := range opts {
		opt(hc)
	}

	hc.client = &http.Client{Timeout: hc.timeout}

	// Initialize statuses
	for _, t := range bal.Targets() {
		hc.statuses[t.URL.String()] = &HealthStatus{Healthy: true}
	}

	return hc
}

// Start runs health checks periodically until ctx is cancelled.
func (hc *HealthCheckerImpl) Start(ctx context.Context) error {
	ticker := time.NewTicker(hc.interval)
	defer ticker.Stop()

	// Run initial check
	hc.checkAll(ctx)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			hc.checkAll(ctx)
		}
	}
}

func (hc *HealthCheckerImpl) checkAll(ctx context.Context) {
	targets := hc.balancer.Targets()
	for _, t := range targets {
		hc.checkTarget(ctx, t.URL.String())
	}
}

func (hc *HealthCheckerImpl) checkTarget(ctx context.Context, targetURL string) {
	checkURL := fmt.Sprintf("%s%s", targetURL, hc.healthPath)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, checkURL, nil)
	if err != nil {
		hc.recordFailure(targetURL, err)
		return
	}

	resp, err := hc.client.Do(req)
	if err != nil {
		hc.recordFailure(targetURL, err)
		return
	}
	resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		hc.recordSuccess(targetURL)
	} else {
		hc.recordFailure(targetURL, fmt.Errorf("health check returned status %d", resp.StatusCode))
	}
}

func (hc *HealthCheckerImpl) recordSuccess(targetURL string) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	status, ok := hc.statuses[targetURL]
	if !ok {
		return
	}

	wasUnhealthy := !status.Healthy
	status.Healthy = true
	status.ConsecutiveFailures = 0
	status.LastChecked = time.Now()
	status.LastError = nil

	if setter, ok := hc.balancer.(healthSetter); ok {
		setter.SetHealthy(targetURL, true)
	}

	if wasUnhealthy {
		hc.logger.Info("upstream recovered",
			slog.String("target", targetURL),
		)
	}
}

func (hc *HealthCheckerImpl) recordFailure(targetURL string, err error) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	status, ok := hc.statuses[targetURL]
	if !ok {
		return
	}

	status.ConsecutiveFailures++
	status.LastChecked = time.Now()
	status.LastError = err

	if status.ConsecutiveFailures >= hc.threshold && status.Healthy {
		status.Healthy = false
		if setter, ok := hc.balancer.(healthSetter); ok {
			setter.SetHealthy(targetURL, false)
		}
		hc.logger.Warn("upstream marked unhealthy",
			slog.String("target", targetURL),
			slog.Int("consecutive_failures", status.ConsecutiveFailures),
			slog.String("error", err.Error()),
		)
	}
}

// IsHealthy returns the current health status of a target.
func (hc *HealthCheckerImpl) IsHealthy(targetURL string) bool {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	if status, ok := hc.statuses[targetURL]; ok {
		return status.Healthy
	}
	return false
}
