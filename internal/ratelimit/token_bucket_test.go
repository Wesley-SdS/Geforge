package ratelimit

import (
	"context"
	"net/http"
	"sync"
	"testing"
	"time"
)

func TestTokenBucket_WithinBurst(t *testing.T) {
	t.Parallel()

	tb := NewTokenBucket(10, 5)
	store := NewTokenBucketStore(tb)
	defer store.Stop()

	ctx := context.Background()

	// Should allow up to burst
	for i := 0; i < 5; i++ {
		d := store.Allow(ctx, "client1")
		if !d.Allowed {
			t.Errorf("request %d should be allowed within burst", i)
		}
	}
}

func TestTokenBucket_ExceedsBurst(t *testing.T) {
	t.Parallel()

	tb := NewTokenBucket(10, 3)
	store := NewTokenBucketStore(tb)
	defer store.Stop()

	ctx := context.Background()

	// Exhaust burst
	for i := 0; i < 3; i++ {
		d := store.Allow(ctx, "client1")
		if !d.Allowed {
			t.Errorf("request %d should be allowed", i)
		}
	}

	// Next request should be rejected
	d := store.Allow(ctx, "client1")
	if d.Allowed {
		t.Error("request should be rejected after burst exhausted")
	}
	if d.RetryAfter <= 0 {
		t.Error("expected positive RetryAfter")
	}
	if d.Limit != 3 {
		t.Errorf("expected limit 3, got %d", d.Limit)
	}
}

func TestTokenBucket_RefillOverTime(t *testing.T) {
	t.Parallel()

	tb := NewTokenBucket(100, 2) // 100 rps = 1 token per 10ms
	store := NewTokenBucketStore(tb)
	defer store.Stop()

	ctx := context.Background()

	// Exhaust
	store.Allow(ctx, "client1")
	store.Allow(ctx, "client1")

	d := store.Allow(ctx, "client1")
	if d.Allowed {
		t.Error("should be rejected immediately")
	}

	// Wait for refill
	time.Sleep(25 * time.Millisecond)

	d = store.Allow(ctx, "client1")
	if !d.Allowed {
		t.Error("should be allowed after refill")
	}
}

func TestTokenBucket_PerClientIsolation(t *testing.T) {
	t.Parallel()

	tb := NewTokenBucket(10, 2)
	store := NewTokenBucketStore(tb)
	defer store.Stop()

	ctx := context.Background()

	// Exhaust client1
	store.Allow(ctx, "client1")
	store.Allow(ctx, "client1")
	d := store.Allow(ctx, "client1")
	if d.Allowed {
		t.Error("client1 should be limited")
	}

	// client2 should be unaffected
	d = store.Allow(ctx, "client2")
	if !d.Allowed {
		t.Error("client2 should not be affected by client1's limits")
	}
}

func TestTokenBucket_Concurrent(t *testing.T) {
	t.Parallel()

	tb := NewTokenBucket(1000, 50)
	store := NewTokenBucketStore(tb)
	defer store.Stop()

	ctx := context.Background()

	var wg sync.WaitGroup
	var allowed, rejected int64
	var mu sync.Mutex

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			d := store.Allow(ctx, "client1")
			mu.Lock()
			if d.Allowed {
				allowed++
			} else {
				rejected++
			}
			mu.Unlock()
		}()
	}
	wg.Wait()

	total := allowed + rejected
	if total != 50 {
		t.Errorf("expected 50 total decisions, got %d", total)
	}
}

func TestTokenBucket_DecisionFields(t *testing.T) {
	t.Parallel()

	tb := NewTokenBucket(10, 5)
	store := NewTokenBucketStore(tb)
	defer store.Stop()

	ctx := context.Background()
	d := store.Allow(ctx, "test")

	if !d.Allowed {
		t.Error("first request should be allowed")
	}
	if d.Limit != 5 {
		t.Errorf("expected limit 5, got %d", d.Limit)
	}
	if d.Remaining != 4 {
		t.Errorf("expected remaining 4, got %d", d.Remaining)
	}
	if d.ResetAt.IsZero() {
		t.Error("expected non-zero ResetAt")
	}
}

func TestTokenBucketStore_RunCleanup(t *testing.T) {
	t.Parallel()

	tb := NewTokenBucket(10, 5)
	store := NewTokenBucketStore(tb)
	defer store.Stop()

	ctx := context.Background()

	// Create two entries
	store.Allow(ctx, "stale-client")
	store.Allow(ctx, "fresh-client")

	// Verify both exist
	entries := store.Entries()
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	// Run cleanup with a threshold in the future — both should be removed
	futureThreshold := time.Now().Add(1 * time.Hour)
	store.RunCleanup(futureThreshold)

	entries = store.Entries()
	if len(entries) != 0 {
		t.Errorf("expected 0 entries after cleanup, got %d", len(entries))
	}
}

func TestTokenBucketStore_RunCleanup_KeepsFresh(t *testing.T) {
	t.Parallel()

	tb := NewTokenBucket(10, 5)
	store := NewTokenBucketStore(tb)
	defer store.Stop()

	ctx := context.Background()

	store.Allow(ctx, "fresh-client")

	// Run cleanup with a threshold in the past — nothing should be removed
	pastThreshold := time.Now().Add(-1 * time.Hour)
	store.RunCleanup(pastThreshold)

	entries := store.Entries()
	if len(entries) != 1 {
		t.Errorf("expected 1 entry (fresh), got %d", len(entries))
	}
}

func TestIPKeyExtractor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		xff      string
		xri      string
		remote   string
		expected string
	}{
		{
			name:     "X-Forwarded-For single",
			xff:      "192.168.1.1",
			expected: "192.168.1.1",
		},
		{
			name:     "X-Forwarded-For multiple",
			xff:      "10.0.0.1, 10.0.0.2, 10.0.0.3",
			expected: "10.0.0.1",
		},
		{
			name:     "X-Real-IP",
			xri:      "172.16.0.1",
			expected: "172.16.0.1",
		},
		{
			name:     "RemoteAddr with port",
			remote:   "192.168.1.100:12345",
			expected: "192.168.1.100",
		},
		{
			name:     "RemoteAddr without port",
			remote:   "192.168.1.100",
			expected: "192.168.1.100",
		},
	}

	extractor := &IPKeyExtractor{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := &http.Request{
				Header:     http.Header{},
				RemoteAddr: tt.remote,
			}
			if tt.xff != "" {
				r.Header.Set("X-Forwarded-For", tt.xff)
			}
			if tt.xri != "" {
				r.Header.Set("X-Real-IP", tt.xri)
			}
			got := extractor.Extract(r)
			if got != tt.expected {
				t.Errorf("Extract() = %s, want %s", got, tt.expected)
			}
		})
	}
}
