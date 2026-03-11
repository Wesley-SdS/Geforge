package circuit

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"
)

var testLogger = slog.New(slog.NewTextHandler(os.Stdout, nil))

func testConfig() BreakerConfig {
	return BreakerConfig{
		FailureThreshold: 3,
		ResetTimeout:     100 * time.Millisecond,
		HalfOpenMaxReqs:  2,
		FailureWindow:    1 * time.Second,
	}
}

var errTest = errors.New("test error")

func TestCircuitBreaker_ClosedSuccess(t *testing.T) {
	t.Parallel()

	cb := NewBreaker(testConfig(), testLogger)
	ctx := context.Background()

	err := cb.Execute(ctx, func(ctx context.Context) error {
		return nil
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if cb.State() != StateClosed {
		t.Errorf("expected closed state, got %s", cb.State())
	}
}

func TestCircuitBreaker_OpensAfterThreshold(t *testing.T) {
	t.Parallel()

	cb := NewBreaker(testConfig(), testLogger)
	ctx := context.Background()

	// Cause failures up to threshold
	for i := 0; i < 3; i++ {
		cb.Execute(ctx, func(ctx context.Context) error {
			return errTest
		})
	}

	if cb.State() != StateOpen {
		t.Errorf("expected open state after %d failures, got %s", 3, cb.State())
	}
}

func TestCircuitBreaker_OpenRejectsRequests(t *testing.T) {
	t.Parallel()

	cb := NewBreaker(testConfig(), testLogger)
	ctx := context.Background()

	// Open the breaker
	for i := 0; i < 3; i++ {
		cb.Execute(ctx, func(ctx context.Context) error {
			return errTest
		})
	}

	err := cb.Execute(ctx, func(ctx context.Context) error {
		return nil
	})
	if !errors.Is(err, ErrCircuitOpen) {
		t.Errorf("expected ErrCircuitOpen, got %v", err)
	}
}

func TestCircuitBreaker_TransitionsToHalfOpen(t *testing.T) {
	t.Parallel()

	cb := NewBreaker(testConfig(), testLogger)
	ctx := context.Background()

	// Open the breaker
	for i := 0; i < 3; i++ {
		cb.Execute(ctx, func(ctx context.Context) error {
			return errTest
		})
	}

	// Wait for reset timeout
	time.Sleep(150 * time.Millisecond)

	// This should transition to half-open and succeed
	err := cb.Execute(ctx, func(ctx context.Context) error {
		return nil
	})
	if err != nil {
		t.Errorf("unexpected error in half-open: %v", err)
	}
}

func TestCircuitBreaker_HalfOpenSuccessCloses(t *testing.T) {
	t.Parallel()

	cb := NewBreaker(testConfig(), testLogger)
	ctx := context.Background()

	// Open the breaker
	for i := 0; i < 3; i++ {
		cb.Execute(ctx, func(ctx context.Context) error {
			return errTest
		})
	}

	time.Sleep(150 * time.Millisecond)

	// Successful request in half-open should close the breaker
	cb.Execute(ctx, func(ctx context.Context) error {
		return nil
	})

	if cb.State() != StateClosed {
		t.Errorf("expected closed after half-open success, got %s", cb.State())
	}
}

func TestCircuitBreaker_HalfOpenFailureReopens(t *testing.T) {
	t.Parallel()

	cb := NewBreaker(testConfig(), testLogger)
	ctx := context.Background()

	// Open the breaker
	for i := 0; i < 3; i++ {
		cb.Execute(ctx, func(ctx context.Context) error {
			return errTest
		})
	}

	time.Sleep(150 * time.Millisecond)

	// Failure in half-open should reopen
	cb.Execute(ctx, func(ctx context.Context) error {
		return errTest
	})

	if cb.State() != StateOpen {
		t.Errorf("expected open after half-open failure, got %s", cb.State())
	}
}

func TestCircuitBreaker_Reset(t *testing.T) {
	t.Parallel()

	cb := NewBreaker(testConfig(), testLogger)
	ctx := context.Background()

	// Open the breaker
	for i := 0; i < 3; i++ {
		cb.Execute(ctx, func(ctx context.Context) error {
			return errTest
		})
	}

	if cb.State() != StateOpen {
		t.Fatal("expected open state")
	}

	cb.Reset()

	if cb.State() != StateClosed {
		t.Errorf("expected closed after reset, got %s", cb.State())
	}

	// Should work after reset
	err := cb.Execute(ctx, func(ctx context.Context) error {
		return nil
	})
	if err != nil {
		t.Errorf("unexpected error after reset: %v", err)
	}
}

func TestCircuitBreaker_Stats(t *testing.T) {
	t.Parallel()

	cb := NewBreaker(testConfig(), testLogger)
	ctx := context.Background()

	cb.Execute(ctx, func(ctx context.Context) error { return nil })
	cb.Execute(ctx, func(ctx context.Context) error { return errTest })

	stats := cb.Stats()
	if stats.TotalRequests != 2 {
		t.Errorf("expected 2 total requests, got %d", stats.TotalRequests)
	}
	if stats.TotalFailures != 1 {
		t.Errorf("expected 1 failure, got %d", stats.TotalFailures)
	}
	if stats.State != StateClosed {
		t.Errorf("expected closed, got %s", stats.State)
	}
}

func TestCircuitBreaker_Concurrent(t *testing.T) {
	t.Parallel()

	cb := NewBreaker(testConfig(), testLogger)
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			cb.Execute(ctx, func(ctx context.Context) error {
				if n%3 == 0 {
					return errTest
				}
				return nil
			})
		}(i)
	}
	wg.Wait()

	// Just verify it doesn't panic and state is valid
	state := cb.State()
	if state.String() == "unknown" {
		t.Error("unexpected unknown state")
	}

	stats := cb.Stats()
	if stats.TotalRequests == 0 {
		t.Error("expected some requests recorded")
	}
}
