package observability

import "testing"

func TestMetricsRegistered(t *testing.T) {
	t.Parallel()

	if HTTPRequestsTotal == nil {
		t.Error("HTTPRequestsTotal is nil")
	}
	if HTTPRequestDuration == nil {
		t.Error("HTTPRequestDuration is nil")
	}
	if HTTPActiveRequests == nil {
		t.Error("HTTPActiveRequests is nil")
	}
	if CircuitBreakerState == nil {
		t.Error("CircuitBreakerState is nil")
	}
	if UpstreamHealth == nil {
		t.Error("UpstreamHealth is nil")
	}
	if RateLimitRejections == nil {
		t.Error("RateLimitRejections is nil")
	}
}
