package circuit

import (
	"testing"
)

func TestState_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		state    State
		expected string
	}{
		{StateClosed, "closed"},
		{StateOpen, "open"},
		{StateHalfOpen, "half-open"},
		{State(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			t.Parallel()
			if got := tt.state.String(); got != tt.expected {
				t.Errorf("String() = %s, want %s", got, tt.expected)
			}
		})
	}
}

func TestDefaultBreakerConfig(t *testing.T) {
	t.Parallel()

	cfg := DefaultBreakerConfig()
	if cfg.FailureThreshold != 5 {
		t.Errorf("expected threshold 5, got %d", cfg.FailureThreshold)
	}
	if cfg.HalfOpenMaxReqs != 3 {
		t.Errorf("expected half-open max 3, got %d", cfg.HalfOpenMaxReqs)
	}
}
