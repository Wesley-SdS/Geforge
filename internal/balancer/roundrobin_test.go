package balancer

import (
	"net/url"
	"sync"
	"testing"
)

func parseURL(raw string) *url.URL {
	u, _ := url.Parse(raw)
	return u
}

func TestRoundRobin_Next(t *testing.T) {
	t.Parallel()

	targets := []Target{
		{URL: parseURL("http://host1:3001"), Weight: 1},
		{URL: parseURL("http://host2:3002"), Weight: 1},
		{URL: parseURL("http://host3:3003"), Weight: 1},
	}
	rr := NewRoundRobin(targets)

	// Sequential selection should cycle through targets
	seen := make(map[string]int)
	for i := 0; i < 9; i++ {
		u, err := rr.Next()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		seen[u.String()]++
	}

	for _, target := range targets {
		if seen[target.URL.String()] != 3 {
			t.Errorf("expected 3 hits for %s, got %d", target.URL, seen[target.URL.String()])
		}
	}
}

func TestRoundRobin_SkipsUnhealthy(t *testing.T) {
	t.Parallel()

	targets := []Target{
		{URL: parseURL("http://host1:3001"), Weight: 1},
		{URL: parseURL("http://host2:3002"), Weight: 1},
		{URL: parseURL("http://host3:3003"), Weight: 1},
	}
	rr := NewRoundRobin(targets)
	rr.SetHealthy("http://host2:3002", false)

	for i := 0; i < 10; i++ {
		u, err := rr.Next()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if u.String() == "http://host2:3002" {
			t.Error("should not select unhealthy target")
		}
	}
}

func TestRoundRobin_AllUnhealthy(t *testing.T) {
	t.Parallel()

	targets := []Target{
		{URL: parseURL("http://host1:3001"), Weight: 1},
	}
	rr := NewRoundRobin(targets)
	rr.SetHealthy("http://host1:3001", false)

	_, err := rr.Next()
	if err != ErrNoHealthyTargets {
		t.Errorf("expected ErrNoHealthyTargets, got %v", err)
	}
}

func TestRoundRobin_Empty(t *testing.T) {
	t.Parallel()
	rr := NewRoundRobin(nil)
	_, err := rr.Next()
	if err != ErrNoHealthyTargets {
		t.Errorf("expected ErrNoHealthyTargets, got %v", err)
	}
}

func TestRoundRobin_Concurrent(t *testing.T) {
	t.Parallel()

	targets := []Target{
		{URL: parseURL("http://host1:3001"), Weight: 1},
		{URL: parseURL("http://host2:3002"), Weight: 1},
		{URL: parseURL("http://host3:3003"), Weight: 1},
	}
	rr := NewRoundRobin(targets)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			u, err := rr.Next()
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

func TestRoundRobin_Targets(t *testing.T) {
	t.Parallel()

	targets := []Target{
		{URL: parseURL("http://host1:3001"), Weight: 1},
		{URL: parseURL("http://host2:3002"), Weight: 2},
	}
	rr := NewRoundRobin(targets)

	statuses := rr.Targets()
	if len(statuses) != 2 {
		t.Fatalf("expected 2 targets, got %d", len(statuses))
	}
	for _, s := range statuses {
		if !s.Healthy {
			t.Errorf("expected all targets healthy initially")
		}
	}
}
