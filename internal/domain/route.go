package domain

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

// Route represents a validated gateway route with parsed target URLs.
type Route struct {
	Path            string
	Methods         []string
	Targets         []Target
	BalanceStrategy string
	StripPrefix     bool
	Timeout         time.Duration
	HasRateLimit    bool
	RPS             float64
	Burst           int
	HasCircuitBreak bool
	FailThreshold   int
	ResetTimeout    time.Duration
	HalfOpenMaxReqs int
}

// Target represents a validated upstream target.
type Target struct {
	URL    *url.URL
	Weight int
}

// Validate checks that the route is well-formed.
func (r *Route) Validate() error {
	var errs []string

	if r.Path == "" {
		errs = append(errs, "path is required")
	}
	if r.Path != "" && !strings.HasPrefix(r.Path, "/") {
		errs = append(errs, "path must start with /")
	}
	if len(r.Targets) == 0 {
		errs = append(errs, "at least one target is required")
	}
	for i, t := range r.Targets {
		if t.URL == nil {
			errs = append(errs, fmt.Sprintf("target[%d]: URL is nil", i))
		}
		if t.Weight < 0 {
			errs = append(errs, fmt.Sprintf("target[%d]: weight must be >= 0", i))
		}
	}
	for _, m := range r.Methods {
		switch m {
		case "GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS":
		default:
			errs = append(errs, fmt.Sprintf("unsupported method: %s", m))
		}
	}
	if r.BalanceStrategy != "" && r.BalanceStrategy != "round-robin" && r.BalanceStrategy != "weighted" {
		errs = append(errs, fmt.Sprintf("unsupported balance strategy: %s", r.BalanceStrategy))
	}
	if r.Timeout < 0 {
		errs = append(errs, "timeout must be >= 0")
	}

	if len(errs) > 0 {
		return fmt.Errorf("route validation failed:\n  %s", strings.Join(errs, "\n  "))
	}
	return nil
}
