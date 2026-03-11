package ratelimit

import (
	"context"
	"testing"
)

func TestInMemoryStore_Allow(t *testing.T) {
	t.Parallel()

	store := NewInMemoryStore(10, 5)
	defer store.Stop()

	ctx := context.Background()

	d := store.Allow(ctx, "test-key")
	if !d.Allowed {
		t.Error("first request should be allowed")
	}
	if d.Limit != 5 {
		t.Errorf("expected limit 5, got %d", d.Limit)
	}
}

func TestInMemoryStore_ImplementsStore(t *testing.T) {
	t.Parallel()

	var s Store = NewInMemoryStore(10, 5)
	defer s.Stop()

	d := s.Allow(context.Background(), "test")
	if !d.Allowed {
		t.Error("should be allowed")
	}
}

func TestTokenBucketStore_Entries(t *testing.T) {
	t.Parallel()

	tb := NewTokenBucket(10, 5)
	store := NewTokenBucketStore(tb)
	defer store.Stop()

	ctx := context.Background()
	store.Allow(ctx, "client-a")
	store.Allow(ctx, "client-b")

	entries := store.Entries()
	if len(entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(entries))
	}

	keys := make(map[string]bool)
	for _, e := range entries {
		keys[e.Key] = true
	}
	if !keys["client-a"] || !keys["client-b"] {
		t.Errorf("expected both clients, got %v", keys)
	}
}
