package balancer

import (
	"net/url"
	"sync"
)

// WeightedRoundRobin implements smooth weighted round-robin (Nginx algorithm).
type WeightedRoundRobin struct {
	targets        []Target
	healthy        []bool
	currentWeights []int
	mu             sync.Mutex
}

// NewWeightedRoundRobin creates a new weighted round-robin balancer.
func NewWeightedRoundRobin(targets []Target) *WeightedRoundRobin {
	healthy := make([]bool, len(targets))
	weights := make([]int, len(targets))
	for i := range targets {
		healthy[i] = true
		weights[i] = 0
	}
	return &WeightedRoundRobin{
		targets:        targets,
		healthy:        healthy,
		currentWeights: weights,
	}
}

// Next selects the next target using smooth weighted round-robin.
func (w *WeightedRoundRobin) Next() (*url.URL, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	n := len(w.targets)
	if n == 0 {
		return nil, ErrNoHealthyTargets
	}

	totalWeight := 0
	bestIdx := -1
	bestWeight := 0

	for i := 0; i < n; i++ {
		if !w.healthy[i] {
			continue
		}
		effectiveWeight := w.targets[i].Weight
		w.currentWeights[i] += effectiveWeight
		totalWeight += effectiveWeight

		if bestIdx == -1 || w.currentWeights[i] > bestWeight {
			bestIdx = i
			bestWeight = w.currentWeights[i]
		}
	}

	if bestIdx == -1 {
		return nil, ErrNoHealthyTargets
	}

	w.currentWeights[bestIdx] -= totalWeight
	return w.targets[bestIdx].URL, nil
}

// Targets returns all targets with their health status.
func (w *WeightedRoundRobin) Targets() []TargetStatus {
	w.mu.Lock()
	defer w.mu.Unlock()

	statuses := make([]TargetStatus, len(w.targets))
	for i, t := range w.targets {
		statuses[i] = TargetStatus{
			Target:  t,
			Healthy: w.healthy[i],
		}
	}
	return statuses
}

// SetHealthy sets the health status of a target by URL string.
func (w *WeightedRoundRobin) SetHealthy(targetURL string, healthy bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	for i, t := range w.targets {
		if t.URL.String() == targetURL {
			w.healthy[i] = healthy
			if !healthy {
				w.currentWeights[i] = 0
			}
			return
		}
	}
}
