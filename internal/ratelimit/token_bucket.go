package ratelimit

import (
	"context"
	"math"
	"sync"
	"time"
)

// TokenBucket implements a token bucket rate limiter.
type TokenBucket struct {
	rate  float64 // tokens per second
	burst int     // maximum tokens (bucket capacity)
}

// NewTokenBucket creates a new token bucket rate limiter.
func NewTokenBucket(rps float64, burst int) *TokenBucket {
	return &TokenBucket{
		rate:  rps,
		burst: burst,
	}
}

// bucket tracks token state for a single client.
type bucket struct {
	tokens     float64
	lastRefill time.Time
	lastAccess time.Time
	mu         sync.Mutex
}

// TokenBucketStore manages per-client token buckets.
type TokenBucketStore struct {
	tb      *TokenBucket
	buckets sync.Map
	stopCh  chan struct{}
}

// NewTokenBucketStore creates a store with a background cleanup goroutine.
func NewTokenBucketStore(tb *TokenBucket) *TokenBucketStore {
	s := &TokenBucketStore{
		tb:     tb,
		stopCh: make(chan struct{}),
	}
	go s.cleanup()
	return s
}

// Allow checks whether the given key is within rate limits.
func (s *TokenBucketStore) Allow(_ context.Context, key string) Decision {
	now := time.Now()

	val, _ := s.buckets.LoadOrStore(key, &bucket{
		tokens:     float64(s.tb.burst),
		lastRefill: now,
		lastAccess: now,
	})
	b := val.(*bucket)

	b.mu.Lock()
	defer b.mu.Unlock()

	b.lastAccess = now

	// Refill tokens based on elapsed time
	elapsed := now.Sub(b.lastRefill).Seconds()
	b.tokens += elapsed * s.tb.rate
	if b.tokens > float64(s.tb.burst) {
		b.tokens = float64(s.tb.burst)
	}
	b.lastRefill = now

	remaining := int(math.Floor(b.tokens))
	resetAt := now.Add(time.Duration(float64(time.Second) / s.tb.rate))

	if b.tokens >= 1.0 {
		b.tokens -= 1.0
		return Decision{
			Allowed:   true,
			Limit:     s.tb.burst,
			Remaining: int(math.Floor(b.tokens)),
			ResetAt:   resetAt,
		}
	}

	// Calculate retry after
	deficit := 1.0 - b.tokens
	retryAfter := time.Duration(deficit / s.tb.rate * float64(time.Second))

	return Decision{
		Allowed:    false,
		Limit:      s.tb.burst,
		Remaining:  remaining,
		ResetAt:    resetAt,
		RetryAfter: retryAfter,
	}
}

// cleanup removes stale buckets every 60 seconds.
func (s *TokenBucketStore) cleanup() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.RunCleanup(time.Now().Add(-5 * time.Minute))
		}
	}
}

// RunCleanup performs a single cleanup pass, removing stale entries.
// Exported for testing; the background goroutine calls this periodically.
func (s *TokenBucketStore) RunCleanup(staleThreshold time.Time) {
	s.buckets.Range(func(key, value any) bool {
		b := value.(*bucket)
		b.mu.Lock()
		isStale := b.lastAccess.Before(staleThreshold)
		b.mu.Unlock()
		if isStale {
			s.buckets.Delete(key)
		}
		return true
	})
}

// Stop stops the cleanup goroutine.
func (s *TokenBucketStore) Stop() {
	close(s.stopCh)
}
