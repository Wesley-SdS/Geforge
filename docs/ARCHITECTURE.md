# Architecture Decision Records

## ADR-001: stdlib net/http over web frameworks

### Context
Go has a rich ecosystem of HTTP frameworks (Gin, Echo, Fiber, Chi) that provide
convenience features like routing, middleware, and parameter parsing. However,
Go 1.22 introduced enhanced routing in `net/http` with method and path pattern support.

### Decision
Use only `net/http` from the standard library. No third-party HTTP frameworks.

### Consequences
- **Positive:** Demonstrates deep understanding of Go's standard library. Smaller binary,
  fewer dependencies, smaller attack surface. Full control over request handling.
  The middleware pattern `func(http.Handler) http.Handler` is idiomatic Go.
- **Negative:** Some boilerplate for common patterns (parameter extraction, response helpers).
  No built-in request binding or validation helpers.
- **Mitigated by:** Go 1.22's enhanced `ServeMux` with method-based routing eliminates
  the biggest historical pain point of stdlib routing.

---

## ADR-002: Token bucket rate limiting algorithm

### Context
Rate limiting can be implemented with several algorithms: fixed window, sliding window,
leaky bucket, or token bucket. Each has different characteristics for burst handling,
memory usage, and fairness.

### Decision
Use the token bucket algorithm with per-client bucket isolation.

### Consequences
- **Positive:** Allows controlled bursts while enforcing average rate. Simple to implement
  and reason about. Per-client isolation prevents one bad actor from affecting others.
  Memory-efficient with TTL-based cleanup of inactive clients.
- **Negative:** Slightly more complex than fixed window. Requires per-client state
  (mitigated by `sync.Map` and background cleanup).
- **Trade-off:** Token bucket is more permissive than leaky bucket for burst traffic,
  which is appropriate for API gateway use cases where clients may have legitimate bursts.

---

## ADR-003: Per-route circuit breakers (vs global)

### Context
Circuit breakers prevent cascade failures by stopping requests to failing services.
They can be applied globally (one breaker for all routes) or per-route (independent
breakers for each upstream service).

### Decision
Implement per-route circuit breakers. Each route has its own independent circuit breaker
with configurable thresholds.

### Consequences
- **Positive:** A failing `/api/payments` endpoint won't block `/api/users`. Fine-grained
  control over failure thresholds per service. Better observability with per-route metrics.
- **Negative:** More memory usage (one breaker per route). More configuration to manage.
- **Trade-off:** The isolation benefit far outweighs the marginal resource cost.
  In production, independent failure domains are critical for availability.

---

## ADR-004: Smooth weighted round-robin (Nginx algorithm)

### Context
Load balancing across heterogeneous backends requires weight-aware distribution.
Simple approaches (repeating entries) create uneven distribution patterns.

### Decision
Use the smooth weighted round-robin algorithm, as implemented in Nginx.

### Consequences
- **Positive:** Distributes requests evenly across backends proportional to weights.
  No request clustering — backends receive traffic in a smooth, interleaved pattern.
  Well-tested algorithm with proven production reliability (Nginx).
- **Negative:** Requires mutex for thread safety (vs atomic for simple round-robin).
  Slightly more complex implementation.
- **Algorithm:** Each target has `currentWeight` (runtime) and `effectiveWeight` (configured).
  On each call: increment all `currentWeight` by `effectiveWeight`, select highest,
  subtract `totalWeight` from selected. This produces smooth distribution.

---

## ADR-005: Config hot reload via atomic swap

### Context
Production API gateways need configuration changes without downtime. Options include:
signal-based reload (SIGHUP), API-based reload, or file-watching with automatic reload.

### Decision
Use `fsnotify` to watch the config file. On change: parse, validate, and atomically
swap the config using `sync.atomic.Value` (via handler replacement).

### Consequences
- **Positive:** Zero-downtime configuration changes. Invalid configs are rejected
  (old config remains active). No manual signal sending required.
  Compatible with ConfigMap updates in Kubernetes.
- **Negative:** File system events can be noisy (debouncing may be needed).
  Requires careful atomic operations to avoid race conditions.
- **Safety:** The validate-then-swap pattern ensures the gateway never runs with
  invalid configuration. Failed reloads are logged at WARN level.

---

## ADR-006: Structured logging with slog

### Context
Production systems need structured, machine-parseable logs for observability platforms
(ELK, Datadog, CloudWatch). Go 1.21 introduced `log/slog` as a stdlib structured logger.

### Decision
Use `log/slog` (standard library) for all structured logging. JSON format by default.

### Consequences
- **Positive:** Zero external dependencies for logging. Built-in support for structured
  fields, log levels, and context propagation. JSON output integrates with any log
  aggregation platform. Request IDs propagated through context for correlation.
- **Negative:** Less feature-rich than `zap` or `zerolog` (no sampling, no caller info
  by default). Slightly newer API, fewer community examples.
- **Mitigated by:** slog's Handler interface allows future migration to any backend
  without changing application code. Performance is adequate for gateway workloads.
