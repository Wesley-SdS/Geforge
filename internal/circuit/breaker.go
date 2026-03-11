package circuit

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// Breaker implements the circuit breaker pattern.
// Thread-safe for concurrent use.
type Breaker interface {
	Execute(ctx context.Context, fn func(ctx context.Context) error) error
	State() State
	Stats() BreakerStats
	Reset()
}

// CircuitBreaker is a concrete implementation of the Breaker interface.
type CircuitBreaker struct {
	config BreakerConfig
	logger *slog.Logger

	mu               sync.RWMutex
	state            State
	consecutiveFails int64
	totalRequests    int64
	totalFailures    int64
	halfOpenReqs     int
	lastFailure      time.Time
	lastSuccess      time.Time
	lastStateChange  time.Time
	failures         []time.Time // sliding window
}

// NewBreaker creates a new circuit breaker with the given configuration.
func NewBreaker(cfg BreakerConfig, logger *slog.Logger) *CircuitBreaker {
	return &CircuitBreaker{
		config:          cfg,
		logger:          logger,
		state:           StateClosed,
		lastStateChange: time.Now(),
	}
}

// Execute runs fn if the circuit allows it.
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func(ctx context.Context) error) error {
	if err := cb.beforeRequest(); err != nil {
		return err
	}

	cb.mu.Lock()
	cb.totalRequests++
	cb.mu.Unlock()

	err := fn(ctx)

	if err != nil {
		cb.onFailure()
	} else {
		cb.onSuccess()
	}

	return err
}

func (cb *CircuitBreaker) beforeRequest() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()

	switch cb.state {
	case StateClosed:
		return nil
	case StateOpen:
		if now.Sub(cb.lastStateChange) > cb.config.ResetTimeout {
			cb.transitionTo(StateHalfOpen)
			cb.halfOpenReqs = 0
			return nil
		}
		return ErrCircuitOpen
	case StateHalfOpen:
		if cb.halfOpenReqs >= cb.config.HalfOpenMaxReqs {
			return ErrCircuitOpen
		}
		cb.halfOpenReqs++
		return nil
	}

	return nil
}

func (cb *CircuitBreaker) onSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.lastSuccess = time.Now()
	cb.consecutiveFails = 0

	if cb.state == StateHalfOpen {
		cb.transitionTo(StateClosed)
		cb.failures = nil
	}
}

func (cb *CircuitBreaker) onFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()
	cb.lastFailure = now
	cb.totalFailures++
	cb.consecutiveFails++

	// Add to sliding window
	cb.failures = append(cb.failures, now)

	// Clean old failures outside the window
	cutoff := now.Add(-cb.config.FailureWindow)
	cleaned := cb.failures[:0]
	for _, ft := range cb.failures {
		if ft.After(cutoff) {
			cleaned = append(cleaned, ft)
		}
	}
	cb.failures = cleaned

	switch cb.state {
	case StateClosed:
		if len(cb.failures) >= cb.config.FailureThreshold {
			cb.transitionTo(StateOpen)
		}
	case StateHalfOpen:
		cb.transitionTo(StateOpen)
	}
}

func (cb *CircuitBreaker) transitionTo(newState State) {
	oldState := cb.state
	cb.state = newState
	cb.lastStateChange = time.Now()

	if cb.logger != nil {
		switch newState {
		case StateOpen:
			cb.logger.Warn("circuit breaker opened",
				slog.String("from", oldState.String()),
				slog.Int64("failures", cb.consecutiveFails),
			)
		case StateClosed:
			cb.logger.Info("circuit breaker closed",
				slog.String("from", oldState.String()),
			)
		case StateHalfOpen:
			cb.logger.Info("circuit breaker half-open",
				slog.String("from", oldState.String()),
			)
		}
	}
}

// State returns the current circuit breaker state.
func (cb *CircuitBreaker) State() State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Stats returns current breaker statistics.
func (cb *CircuitBreaker) Stats() BreakerStats {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return BreakerStats{
		State:            cb.state,
		TotalRequests:    cb.totalRequests,
		TotalFailures:    cb.totalFailures,
		ConsecutiveFails: cb.consecutiveFails,
		LastFailure:      cb.lastFailure,
		LastSuccess:      cb.lastSuccess,
		LastStateChange:  cb.lastStateChange,
	}
}

// Reset forces the breaker back to closed state.
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state = StateClosed
	cb.consecutiveFails = 0
	cb.halfOpenReqs = 0
	cb.failures = nil
	cb.lastStateChange = time.Now()

	if cb.logger != nil {
		cb.logger.Info("circuit breaker manually reset")
	}
}
