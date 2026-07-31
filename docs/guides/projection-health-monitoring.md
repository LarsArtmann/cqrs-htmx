# Projection Health Monitoring

> How to monitor projection lag, detect failures, and set up alerting using the built-in projection status endpoint.

---

## The Problem

CQRS projections process events asynchronously. If a projection falls behind (slow processing, poison events, crash), queries return stale data. Without observability, you discover the problem when users complain.

---

## The Solution: Projection Status Endpoint

cqrs-htmx provides a `ProjectionStatusHandler` that serves live projection health as JSON:

```go
svc, _ := usermgmt.NewService(usermgmt.ServiceConfig{})
defer svc.Close()

mux.Handle("GET /health/projections",
    cqrshtmx.ProjectionStatusHandler(svc))
```

The `*usermgmt.Service` (and `*EventSourcedSetup`) implements `cqrshtmx.ProjectionStatusProvider` automatically.

---

## Response Format

```json
[
	{
		"name": "user-read-model",
		"status": "live",
		"checkpoint": "evt-018a3f...",
		"processed": 5234,
		"errors": 0,
		"restarts": 0,
		"lag_ms": 15,
		"lastError": ""
	},
	{
		"name": "casbin-projection",
		"status": "live",
		"checkpoint": "evt-018a3f...",
		"processed": 5234,
		"errors": 2,
		"restarts": 1,
		"lag_ms": 32,
		"lastError": "connection reset: retrying"
	}
]
```

### Field Reference

| Field        | Type   | Description                                                         |
| ------------ | ------ | ------------------------------------------------------------------- |
| `name`       | string | Projection name (e.g. `"user-read-model"`)                          |
| `status`     | string | Worker lifecycle state (see below)                                  |
| `checkpoint` | string | Last processed event ID                                             |
| `processed`  | int64  | Total events processed since start                                  |
| `errors`     | int64  | Total processing errors                                             |
| `restarts`   | int    | Number of crash-restart cycles                                      |
| `lag_ms`     | int64  | Approximate lag in milliseconds (how far behind the event log head) |
| `lastError`  | string | Last error message (omitted if empty)                               |

### Status Values

| Status     | Meaning                                                          |
| ---------- | ---------------------------------------------------------------- |
| `idle`     | Worker registered but not started                                |
| `running`  | Worker is replaying historical events (initial drain)            |
| `live`     | Worker has finished drain and is processing live events          |
| `backoff`  | Worker crashed and is waiting to restart (exponential backoff)   |
| `draining` | Worker is shutting down (flushing in-flight events)              |
| `stopped`  | Worker has stopped (terminal state for non-blocking subscribers) |
| `failed`   | Worker permanently failed (max retries exceeded)                 |

---

## Interpreting Lag

`lag_ms` tells you how far behind a projection is from the event log head:

| Lag Range | Health   | Action                                              |
| --------- | -------- | --------------------------------------------------- |
| 0-100ms   | Healthy  | No action needed                                    |
| 100ms-1s  | Elevated | Monitor closely                                     |
| 1s-10s    | Degraded | Investigate slow handlers or DB contention          |
| >10s      | Critical | Projection is severely behind; users see stale data |

---

## Alerting

### Prometheus (via JSON export)

Scrape the endpoint and convert to metrics:

```yaml
# prometheus.yml
scrape_configs:
  - job_name: "projection-health"
    static_configs:
      - targets: ["localhost:8080"]
    metrics_path: "/health/projections"
```

### Simple HTTP Polling

```bash
# Alert if any projection has >5s lag
curl -s http://localhost:8080/health/projections | \
  jq '.[] | select(.lag_ms > 5000) | .name'
```

### Error Rate Alerting

If `errors` or `restarts` is increasing, a projection is hitting poison events:

```bash
# Alert if any projection has restarted more than 3 times
curl -s http://localhost:8080/health/projections | \
  jq '.[] | select(.restarts > 3) | .name'
```

---

## Terminal Failure Callback (`OnProjectionFailed`)

When a projection worker exhausts its restart budget (configurable via `projectionhost.WithMaxRestarts`), it transitions to a terminal failure state and stops processing. By default, this is silent (logs only).

To receive a callback when this happens, set `OnProjectionFailed` on your config:

```go
cfg := usermgmt.ServiceConfig{
    OnProjectionFailed: func(projectionName, lastError string) {
        log.Printf("CRITICAL: projection %s entered terminal failure: %s", projectionName, lastError)
        // Emit a metric, send a Slack alert, page on-call, etc.
    },
    // ...
}
```

The callback receives:
- `projectionName` — the projection's `Name()` (e.g., `"user-read-model"`, `"casbin-projection"`)
- `lastError` — the error message from the final crash-restart attempt

> **Note:** The callback is invoked synchronously from the worker goroutine. Keep it fast and non-blocking. For expensive operations (HTTP calls, external APIs), dispatch to a separate goroutine from within the callback.

This is available on both `EventSourcedConfig` and `ServiceConfig`. The same field is forwarded to `projectionhost.WithOnFailed` internally.

---

## Caching Behavior

The handler uses `Cache-Control: no-cache` with a per-request ETag (FNV-1a hash of the current status JSON). This means:

- Clients should not cache the response (it changes every request).
- Conditional GETs (`If-None-Match`) still work — if nothing changed, you get 304.
- No server-side state is stored (status is recomputed from the projection host on each request).

---

## See Also

- [Event Replay and Rebuild](./event-replay-and-rebuild.md) — How to fix a failed projection
- [Consistency Model](./consistency-model.md) — What lag means for consistency
- `projection_status_handler.go` — Handler implementation
