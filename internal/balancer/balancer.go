package balancer

import (
	"context"
	"errors"
	"log/slog"
	"net/url"
	"time"

	"github.com/wesleybatista/gateforge/internal/config"
)

// Balancer selects the next healthy upstream target for proxying.
// All implementations must be safe for concurrent use.
type Balancer interface {
	Next() (*url.URL, error)
	Targets() []TargetStatus
}

// HealthChecker performs periodic health checks on upstream targets.
type HealthChecker interface {
	Start(ctx context.Context) error
	IsHealthy(targetURL string) bool
}

// Target represents a single upstream server.
type Target struct {
	URL    *url.URL
	Weight int
}

// TargetStatus extends Target with runtime health information.
type TargetStatus struct {
	Target
	Healthy      bool
	LastChecked  time.Time
	LastError    error
	FailureCount int64
	SuccessCount int64
}

// ErrNoHealthyTargets is returned when no healthy targets are available.
var ErrNoHealthyTargets = errors.New("no healthy upstream targets available")

// NewFromConfig creates a Balancer from route configuration.
func NewFromConfig(routeCfg config.RouteConfig) Balancer {
	targets := make([]Target, 0, len(routeCfg.Targets))
	for _, t := range routeCfg.Targets {
		u, err := url.Parse(t.URL)
		if err != nil {
			slog.Warn("skipping target with invalid URL",
				slog.String("route", routeCfg.Path),
				slog.String("url", t.URL),
				slog.String("error", err.Error()),
			)
			continue
		}
		weight := t.Weight
		if weight <= 0 {
			weight = 1
		}
		targets = append(targets, Target{URL: u, Weight: weight})
	}

	switch routeCfg.BalanceStrategy {
	case "weighted":
		return NewWeightedRoundRobin(targets)
	default:
		return NewRoundRobin(targets)
	}
}
