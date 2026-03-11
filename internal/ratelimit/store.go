package ratelimit

import (
	"context"
	"time"
)

// Store provides a generic key-value store for rate limiter state.
// This abstraction allows swapping backends (in-memory, Redis, etc).
type Store interface {
	// Allow checks and decrements the rate limit for the given key.
	Allow(ctx context.Context, key string) Decision
	// Stop cleans up resources.
	Stop()
}

// InMemoryStore wraps TokenBucketStore to implement Store.
type InMemoryStore struct {
	inner *TokenBucketStore
}

// NewInMemoryStore creates an in-memory rate limit store.
func NewInMemoryStore(rps float64, burst int) *InMemoryStore {
	tb := NewTokenBucket(rps, burst)
	return &InMemoryStore{
		inner: NewTokenBucketStore(tb),
	}
}

// Allow checks the rate limit for the given key.
func (s *InMemoryStore) Allow(ctx context.Context, key string) Decision {
	return s.inner.Allow(ctx, key)
}

// Stop stops the background cleanup goroutine.
func (s *InMemoryStore) Stop() {
	s.inner.Stop()
}

// CleanupConfig holds cleanup configuration.
type CleanupConfig struct {
	Interval  time.Duration // How often to run cleanup
	MaxAge    time.Duration // Remove entries older than this
}

// DefaultCleanupConfig returns default cleanup settings.
func DefaultCleanupConfig() CleanupConfig {
	return CleanupConfig{
		Interval: 60 * time.Second,
		MaxAge:   5 * time.Minute,
	}
}

// BucketInfo contains information about a rate limit bucket (for testing/debugging).
type BucketInfo struct {
	Key        string
	Tokens     float64
	LastAccess time.Time
}

// Entries returns all current bucket entries (for testing/debugging).
func (s *TokenBucketStore) Entries() []BucketInfo {
	var entries []BucketInfo
	s.buckets.Range(func(key, value any) bool {
		b := value.(*bucket)
		b.mu.Lock()
		entries = append(entries, BucketInfo{
			Key:        key.(string),
			Tokens:     b.tokens,
			LastAccess: b.lastAccess,
		})
		b.mu.Unlock()
		return true
	})
	return entries
}

// _ ensures InMemoryStore implements Store.
var _ Store = (*InMemoryStore)(nil)
