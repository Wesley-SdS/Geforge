package circuit

import (
	"errors"
	"time"
)

// State represents the circuit breaker state.
type State int

const (
	StateClosed   State = iota // Normal: requests flow through
	StateOpen                   // Tripped: requests blocked
	StateHalfOpen              // Testing: limited requests allowed
)

// String returns the string representation of the state.
func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// BreakerStats contains circuit breaker statistics for monitoring.
type BreakerStats struct {
	State            State
	TotalRequests    int64
	TotalFailures    int64
	ConsecutiveFails int64
	LastFailure      time.Time
	LastSuccess      time.Time
	LastStateChange  time.Time
}

// BreakerConfig holds circuit breaker configuration.
type BreakerConfig struct {
	FailureThreshold int
	ResetTimeout     time.Duration
	HalfOpenMaxReqs  int
	FailureWindow    time.Duration
}

// DefaultBreakerConfig returns default configuration values.
func DefaultBreakerConfig() BreakerConfig {
	return BreakerConfig{
		FailureThreshold: 5,
		ResetTimeout:     30 * time.Second,
		HalfOpenMaxReqs:  3,
		FailureWindow:    60 * time.Second,
	}
}

// ErrCircuitOpen is returned when the circuit breaker is in open state.
var ErrCircuitOpen = errors.New("circuit breaker is open")
