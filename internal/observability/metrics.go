package observability

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTPRequestsTotal counts total HTTP requests processed.
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "gateforge",
			Name:      "http_requests_total",
			Help:      "Total number of HTTP requests processed",
		},
		[]string{"method", "path", "status_code"},
	)

	// HTTPRequestDuration tracks HTTP request duration.
	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "gateforge",
			Name:      "http_request_duration_seconds",
			Help:      "HTTP request duration in seconds",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	// HTTPActiveRequests tracks currently active requests.
	HTTPActiveRequests = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "gateforge",
			Name:      "http_active_requests",
			Help:      "Number of currently active HTTP requests",
		},
		[]string{"method", "path"},
	)

	// CircuitBreakerState tracks circuit breaker state per route.
	CircuitBreakerState = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "gateforge",
			Name:      "circuit_breaker_state",
			Help:      "Circuit breaker state (0=closed, 1=open, 2=half-open)",
		},
		[]string{"route"},
	)

	// UpstreamHealth tracks upstream target health.
	UpstreamHealth = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "gateforge",
			Name:      "upstream_healthy",
			Help:      "Upstream target health (1=healthy, 0=unhealthy)",
		},
		[]string{"route", "target"},
	)

	// RateLimitRejections counts rate limit rejections.
	RateLimitRejections = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "gateforge",
			Name:      "rate_limit_rejections_total",
			Help:      "Total rate limit rejections",
		},
		[]string{"route", "client_ip"},
	)
)
