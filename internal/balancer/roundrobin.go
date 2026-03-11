package balancer

import (
	"net/url"
	"sync"
	"sync/atomic"
)

// RoundRobin implements a simple round-robin load balancer.
// It uses an atomic counter for lock-free concurrent access.
type RoundRobin struct {
	targets  []Target
	healthy  []bool
	counter  atomic.Uint64
	mu       sync.RWMutex
}

// NewRoundRobin creates a new round-robin balancer with the given targets.
func NewRoundRobin(targets []Target) *RoundRobin {
	healthy := make([]bool, len(targets))
	for i := range healthy {
		healthy[i] = true
	}
	return &RoundRobin{
		targets: targets,
		healthy: healthy,
	}
}

// Next returns the next healthy target URL using round-robin selection.
func (rr *RoundRobin) Next() (*url.URL, error) {
	rr.mu.RLock()
	defer rr.mu.RUnlock()

	n := len(rr.targets)
	if n == 0 {
		return nil, ErrNoHealthyTargets
	}

	// Try all targets starting from the current counter position
	idx := rr.counter.Add(1) - 1
	for i := 0; i < n; i++ {
		pos := int((idx + uint64(i)) % uint64(n))
		if rr.healthy[pos] {
			return rr.targets[pos].URL, nil
		}
	}

	return nil, ErrNoHealthyTargets
}

// Targets returns all targets with their health status.
func (rr *RoundRobin) Targets() []TargetStatus {
	rr.mu.RLock()
	defer rr.mu.RUnlock()

	statuses := make([]TargetStatus, len(rr.targets))
	for i, t := range rr.targets {
		statuses[i] = TargetStatus{
			Target:  t,
			Healthy: rr.healthy[i],
		}
	}
	return statuses
}

// SetHealthy sets the health status of a target by URL string.
func (rr *RoundRobin) SetHealthy(targetURL string, healthy bool) {
	rr.mu.Lock()
	defer rr.mu.Unlock()
	for i, t := range rr.targets {
		if t.URL.String() == targetURL {
			rr.healthy[i] = healthy
			return
		}
	}
}
