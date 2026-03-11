<div align="center">

# GateForge

**Production-grade API Gateway built entirely on Go's `net/http` stdlib**

[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?style=for-the-badge&logo=go&logoColor=white)](https://go.dev)
[![Coverage](https://img.shields.io/badge/coverage-93%25+-00C853?style=for-the-badge)](/)
[![License](https://img.shields.io/badge/license-MIT-blue?style=for-the-badge)](LICENSE)

Rate Limiting &nbsp;&bull;&nbsp; Circuit Breaking &nbsp;&bull;&nbsp; Weighted Load Balancing &nbsp;&bull;&nbsp; Real-time Observability

*Zero framework dependencies &nbsp;|&nbsp; Interface-driven &nbsp;|&nbsp; 93%+ test coverage*

---

[Quick Start](#quick-start) &nbsp;&bull;&nbsp; [Features](#features) &nbsp;&bull;&nbsp; [Configuration](#configuration) &nbsp;&bull;&nbsp; [Observability](#observability) &nbsp;&bull;&nbsp; [Architecture](#architecture)

</div>

<br>

## Overview

GateForge routes, balances, protects, and observes traffic to upstream services. Every component is built on Go's standard library with clean architecture and interface-driven design &mdash; no frameworks, no magic.

```
                         ┌──────────────┐
    Clients ────────────►│   GateForge  │────────► upstream-1  (weight: 3)
                         │    :8080     │────────► upstream-2  (weight: 1)
                         └──────┬───────┘────────► upstream-n
                                │
                  ┌─────────────┴─────────────┐
                  │                           │
           ┌──────┴──────┐             ┌──────┴──────┐
           │ Prometheus  │────────────►│   Grafana   │
           │   :9090     │             │   :3000     │
           └─────────────┘             └─────────────┘
```

<br>

## Quick Start

```bash
git clone https://github.com/wesleybatista/gateforge && cd gateforge
```

**Docker** (recommended)

```bash
docker compose -f deployments/docker-compose.yml up --build
```

**Local**

```bash
make build && make run
```

**Verify**

```bash
curl localhost:8080/health        # {"status":"healthy"}
curl localhost:8080/api/users     # proxied to upstream
curl localhost:8080/metrics       # prometheus metrics
```

| Service | URL | Description |
|:--------|:----|:------------|
| Gateway | `localhost:8080` | API Gateway |
| Prometheus | `localhost:9090` | Metrics store |
| Grafana | `localhost:3000` | Auto-provisioned dashboards |

<br>

## Features

<table>
<tr><td width="50%">

### Traffic Management
- **Reverse Proxy** &mdash; `httputil.ReverseProxy` with custom transport
- **Round-Robin** &mdash; atomic counter-based distribution
- **Weighted Round-Robin** &mdash; smooth Nginx algorithm
- **Health Checks** &mdash; background monitoring with failure threshold
- **Per-route Timeouts** &mdash; configurable per upstream

</td><td width="50%">

### Protection
- **Rate Limiting** &mdash; per-client token bucket with burst control
- **Circuit Breaker** &mdash; sliding window, 3-state machine (closed / open / half-open)
- **Panic Recovery** &mdash; catches panics, returns structured 500 JSON
- **CORS** &mdash; configurable origins, methods, headers, preflight

</td></tr>
<tr><td>

### Observability
- **Prometheus** &mdash; 6 metric types (counters, histograms, gauges)
- **Grafana** &mdash; 14-panel dashboard, auto-provisioned
- **Structured Logs** &mdash; `log/slog` JSON with request ID correlation
- **Request Tracing** &mdash; UUIDv7 propagation via `X-Request-ID`

</td><td>

### Operations
- **Hot Reload** &mdash; `fsnotify` watches config, zero downtime
- **Env Overrides** &mdash; `${VAR}` expansion in YAML
- **Docker** &mdash; multi-stage build, ~15 MB final image
- **CI/CD** &mdash; GitHub Actions: lint, test, coverage gate, build

</td></tr>
</table>

<br>

## Request Pipeline

Every request flows through a composable middleware chain:

```
 Request ──► Recovery ──► RequestID ──► Logging ──► Metrics ──► CORS
                                                                  │
 Response ◄── Reverse Proxy ◄── Circuit Breaker ◄── Rate Limiter ◄┘
```

| Middleware | Responsibility |
|:-----------|:---------------|
| **Recovery** | Catches panics, returns `500` JSON without leaking stack traces |
| **Request ID** | Generates UUIDv7 or propagates existing `X-Request-ID` |
| **Logging** | Structured `slog` output: latency, status, client IP, upstream |
| **Metrics** | Prometheus counters, histograms, and active request gauges |
| **CORS** | Origin/method/header validation with preflight support |
| **Rate Limiter** | Per-client token bucket with burst, stale entry cleanup |
| **Circuit Breaker** | Per-route sliding failure window with auto recovery |
| **Reverse Proxy** | Weighted load-balanced forwarding with timeout |

<br>

## Configuration

YAML with environment variable expansion:

```yaml
server:
  port: 8080
  read_timeout: 30s
  write_timeout: 30s
  idle_timeout: 120s

logging:
  level: info                    # debug | info | warn | error
  format: json                   # json | text

metrics:
  enabled: true
  path: /metrics

cors:
  allowed_origins: ["*"]

routes:
  - path: /api/users
    balance_strategy: weighted   # round-robin | weighted
    timeout: 10s
    targets:
      - url: http://users-svc:3001
        weight: 3
      - url: http://users-svc:3002
        weight: 1
    rate_limit:
      requests_per_second: 100
      burst: 20
    circuit_breaker:
      failure_threshold: 5
      reset_timeout: 30s
      half_open_max_requests: 3
```

<details>
<summary><strong>Environment Overrides</strong></summary>

| Variable | Description |
|:---------|:------------|
| `GATEFORGE_PORT` | Server port |
| `GATEFORGE_LOG_LEVEL` | Log level |
| `GATEFORGE_LOG_FORMAT` | Log format |
| `GATEFORGE_METRICS_ENABLED` | Enable/disable metrics endpoint |

</details>

<details>
<summary><strong>Hot Reload</strong></summary>

Edit `gateway.yaml` while the gateway is running. Valid changes apply instantly. Invalid configs are rejected &mdash; the previous config stays active.

</details>

<br>

## Observability

### Prometheus Metrics

| Metric | Type | Labels |
|:-------|:-----|:-------|
| `gateforge_http_requests_total` | Counter | `method`, `path`, `status_code` |
| `gateforge_http_request_duration_seconds` | Histogram | `method`, `path` |
| `gateforge_http_active_requests` | Gauge | `method`, `path` |
| `gateforge_circuit_breaker_state` | Gauge | `route` |
| `gateforge_upstream_healthy` | Gauge | `route`, `target` |
| `gateforge_rate_limit_rejections_total` | Counter | `route`, `client_ip` |

### Grafana Dashboard

Auto-provisioned at `localhost:3000` with 14 panels covering:

> Request rate &bull; Error rate &bull; Latency percentiles (p50/p90/p99) &bull; Active connections
> Per-route breakdown &bull; Status code distribution &bull; Circuit breaker states
> Upstream health &bull; Rate limit rejections

### Structured Logs

```json
{
  "time": "2024-01-15T10:30:00Z",
  "level": "INFO",
  "msg": "request completed",
  "request_id": "01942a3b-...",
  "method": "GET",
  "path": "/api/users",
  "status": 200,
  "bytes": 256,
  "latency": "12.5ms",
  "client_ip": "192.168.1.1",
  "upstream": "http://users-svc:3001"
}
```

<br>

## Architecture

### Project Structure

```
gateforge/
├── cmd/
│   ├── gateforge/main.go              Gateway entry point
│   └── upstream/main.go               Demo upstream server
├── internal/
│   ├── domain/                         Route entity + validation
│   ├── config/                         YAML loader, validation, hot reload
│   ├── proxy/                          Reverse proxy + custom transport
│   ├── balancer/                       Round-robin, weighted, health checker
│   ├── ratelimit/                      Token bucket + per-client store
│   ├── circuit/                        Circuit breaker state machine
│   ├── middleware/                     Chain, logging, metrics, CORS, recovery
│   ├── observability/                  slog factory, Prometheus, context helpers
│   └── server/                         HTTP lifecycle + router wiring
├── configs/gateway.yaml                Production-like config
├── deployments/                        Dockerfile, Compose, Prometheus, Grafana
├── scripts/loadtest.sh                 Load testing (hey/wrk)
├── docs/ARCHITECTURE.md                Architecture Decision Records
└── .github/workflows/ci.yml           Lint + test + coverage gate + build
```

### Design Decisions

> Full ADRs in [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md)

| Decision | Rationale |
|:---------|:----------|
| **stdlib `net/http`** | Minimal attack surface, idiomatic middleware, deep Go knowledge |
| **Token bucket** | Controlled bursts, per-client isolation, O(1) per client |
| **Per-route circuit breakers** | Failure isolation &mdash; one bad upstream doesn't cascade |
| **Smooth weighted round-robin** | Even distribution proportional to weights (Nginx algorithm) |
| **Atomic config swap** | Zero-downtime reload, invalid configs safely rejected |
| **`log/slog`** | Structured logging, zero external deps, context propagation |
| **Interface-driven design** | Every component testable and swappable (SOLID/DIP) |

### Test Coverage

```
Package             Coverage
───────────────────────────────
domain              100.0%
observability       100.0%
middleware           98.3%
ratelimit            96.7%
circuit              93.7%
proxy                90.7%
balancer             88.7%
server               87.8%
config               86.1%
```

<br>

## Tech Stack

| | Technology | Purpose |
|:--|:-----------|:--------|
| **Language** | Go 1.22+ | Performance, concurrency, cloud-native |
| **HTTP** | `net/http` | Zero-dependency request handling |
| **Config** | YAML + fsnotify | Human-readable, hot-reloadable |
| **Metrics** | Prometheus | Industry standard observability |
| **Dashboards** | Grafana | 14 auto-provisioned panels |
| **Logging** | `log/slog` | Structured, stdlib, context-aware |
| **Testing** | `testing` + race detector | Table-driven, concurrent, 93%+ |
| **CI/CD** | GitHub Actions | Lint, test, coverage gate, build |
| **Container** | Docker multi-stage | ~15 MB final image |

<br>

## Development

```bash
make build           # compile gateway + upstream binaries
make test            # run tests with coverage
make test-verbose    # verbose test output
make lint            # golangci-lint
make bench           # benchmarks
make coverage-html   # generate HTML coverage report
make docker-up       # start full stack (Compose)
make docker-down     # teardown
make clean           # remove build artifacts
```

### Load Testing

```bash
go install github.com/rakyll/hey@latest

bash scripts/loadtest.sh                                # 60s, 50 connections
DURATION=30 CONCURRENCY=100 bash scripts/loadtest.sh    # custom
```

<br>

## License

[MIT](LICENSE)

---

<div align="center">
<sub>Built with Go stdlib &nbsp;&bull;&nbsp; Designed for production &nbsp;&bull;&nbsp; Zero framework dependencies</sub>
</div>
