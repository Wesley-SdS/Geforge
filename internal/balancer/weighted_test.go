package balancer

import (
	"math"
	"sync"
	"testing"
)

func TestWeightedRoundRobin_Distribution(t *testing.T) {
	t.Parallel()

	targets := []Target{
		{URL: parseURL("http://host1:3001"), Weight: 3},
		{URL: parseURL("http://host2:3002"), Weight: 1},
	}
	wrr := NewWeightedRoundRobin(targets)

	counts := make(map[string]int)
	total := 1000
	for i := 0; i < total; i++ {
		u, err := wrr.Next()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		counts[u.String()]++
	}

	// Expect 75% ± 5% for host1 and 25% ± 5% for host2
	host1Pct := float64(counts["http://host1:3001"]) / float64(total) * 100
	host2Pct := float64(counts["http://host2:3002"]) / float64(total) * 100

	if math.Abs(host1Pct-75) > 5 {
		t.Errorf("host1 expected ~75%%, got %.1f%%", host1Pct)
	}
	if math.Abs(host2Pct-25) > 5 {
		t.Errorf("host2 expected ~25%%, got %.1f%%", host2Pct)
	}
}

func TestWeightedRoundRobin_SkipsUnhealthy(t *testing.T) {
	t.Parallel()

	targets := []Target{
		{URL: parseURL("http://host1:3001"), Weight: 3},
		{URL: parseURL("http://host2:3002"), Weight: 1},
	}
	wrr := NewWeightedRoundRobin(targets)
	wrr.SetHealthy("http://host1:3001", false)

	for i := 0; i < 10; i++ {
		u, err := wrr.Next()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if u.String() == "http://host1:3001" {
			t.Error("should not select unhealthy target")
		}
	}
}

func TestWeightedRoundRobin_AllUnhealthy(t *testing.T) {
	t.Parallel()

	targets := []Target{
		{URL: parseURL("http://host1:3001"), Weight: 1},
	}
	wrr := NewWeightedRoundRobin(targets)
	wrr.SetHealthy("http://host1:3001", false)

	_, err := wrr.Next()
	if err != ErrNoHealthyTargets {
		t.Errorf("expected ErrNoHealthyTargets, got %v", err)
	}
}

func TestWeightedRoundRobin_Empty(t *testing.T) {
	t.Parallel()
	wrr := NewWeightedRoundRobin(nil)
	_, err := wrr.Next()
	if err != ErrNoHealthyTargets {
		t.Errorf("expected ErrNoHealthyTargets, got %v", err)
	}
}

func TestWeightedRoundRobin_Concurrent(t *testing.T) {
	t.Parallel()

	targets := []Target{
		{URL: parseURL("http://host1:3001"), Weight: 2},
		{URL: parseURL("http://host2:3002"), Weight: 1},
		{URL: parseURL("http://host3:3003"), Weight: 1},
	}
	wrr := NewWeightedRoundRobin(targets)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			u, err := wrr.Next()
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if u == nil {
				t.Error("expected non-nil URL")
			}
		}()
	}
	wg.Wait()
}

func TestWeightedRoundRobin_Targets(t *testing.T) {
	t.Parallel()

	targets := []Target{
		{URL: parseURL("http://host1:3001"), Weight: 3},
		{URL: parseURL("http://host2:3002"), Weight: 1},
	}
	wrr := NewWeightedRoundRobin(targets)

	statuses := wrr.Targets()
	if len(statuses) != 2 {
		t.Fatalf("expected 2 targets, got %d", len(statuses))
	}
	for _, s := range statuses {
		if !s.Healthy {
			t.Errorf("expected all targets healthy, %s is not", s.URL)
		}
	}

	// Mark one unhealthy and verify
	wrr.SetHealthy("http://host1:3001", false)
	statuses = wrr.Targets()
	for _, s := range statuses {
		if s.URL.String() == "http://host1:3001" && s.Healthy {
			t.Error("host1 should be unhealthy")
		}
		if s.URL.String() == "http://host2:3002" && !s.Healthy {
			t.Error("host2 should still be healthy")
		}
	}
}

func TestWeightedRoundRobin_EqualWeights(t *testing.T) {
	t.Parallel()

	targets := []Target{
		{URL: parseURL("http://host1:3001"), Weight: 1},
		{URL: parseURL("http://host2:3002"), Weight: 1},
	}
	wrr := NewWeightedRoundRobin(targets)

	counts := make(map[string]int)
	for i := 0; i < 100; i++ {
		u, err := wrr.Next()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		counts[u.String()]++
	}

	if counts["http://host1:3001"] != 50 || counts["http://host2:3002"] != 50 {
		t.Errorf("expected 50/50 distribution, got %v", counts)
	}
}
