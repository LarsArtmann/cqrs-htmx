# Async Projection Startup

> Eliminate multi-minute restart outages by decoupling server liveness from projection readiness.

---

## The Problem

By default, `usermgmt.NewService()` (and `setup.New()`) blocks until **all**
projection workers finish replaying the full event journal. The HTTP server
does not bind until the drain completes. On a deployment with a large journal
this means a 2-4 minute window on every restart, deploy, and crash-recovery
where the port is closed and the reverse proxy returns 502.

```
t=0s    New process starts → NewService() begins
t=0s    Projection drain starts: 6 projections replay the full journal
t=0-4m  Replaying... HTTP server NOT listening — reverse proxy returns 502
t=~4m   Drain complete, NewService() returns, server binds
```

This conflates two distinct concerns:

- **Liveness** — the process is up and the HTTP server is accepting connections.
- **Readiness** — all projections have caught up and reads reflect all writes.

A well-designed system starts the server immediately (liveness) and gates reads
behind a readiness check. During the catch-up window the reverse proxy retries
on 503 instead of failing with 502.

---

## The Solution: `AsyncStartup`

Set `AsyncStartup: true` on the config. `NewService` / `setup.New` returns
immediately after the projection host starts — the HTTP server binds right away
while projections replay the journal in the background.

### With `setup.New` (recommended for full-stack apps)

```go
bundle, err := setup.New(setup.Config{
    AsyncStartup: true, // <-- server binds immediately; /health gates traffic

    // ...your auth providers, stores, paths...
})
```

The bundle's `/health` endpoint (mounted by `Mount`) already uses the
drain-aware readiness check: it returns **503** while any projection is still
draining, then **200** once every worker reaches `"live"` state. Point your
reverse proxy's health check at `/health`:

```text
# Caddy
reverse_proxy localhost:8087 {
    health_uri      /health
    health_interval 2s
    health_timeout  1s
}

# nginx
location / {
    proxy_pass http://127.0.0.1:8087;
    health_check uri=/health interval=2s;
}
```

### With `usermgmt.NewService` directly

```go
svc, err := usermgmt.NewService(usermgmt.ServiceConfig{
    AsyncStartup: true,
    // ...
})

// Mount the readiness check yourself if you're not using setup.Bundle:
mux.Handle("/health", cqrshtmx.ReadinessHandler(
    cqrshtmx.ProjectionReadinessCheck(svc),
))
```

---

## What Happens During Startup

| Phase             | Without AsyncStartup     | With AsyncStartup                           |
| ----------------- | ------------------------ | ------------------------------------------- |
| `New()` returns   | After drain (minutes)    | Immediately (milliseconds)                  |
| HTTP server binds | After drain              | Immediately                                 |
| `/health`         | 200 (drain already done) | 503 → 200 (drain-aware)                     |
| Writes            | Work after drain         | Work immediately (events append to journal) |
| Reads             | Consistent after drain   | May be stale until `/health` → 200          |

---

## Readiness Check: `ProjectionReadinessCheck`

`cqrshtmx.ProjectionReadinessCheck(provider)` is the reusable building block.
It returns a `NamedCheck` for `ReadinessHandler` that fails while any
projection is still draining or has failed:

| Projection status | Check result                 |
| ----------------- | ---------------------------- |
| `"live"`          | ready                        |
| `"stopped"`       | ready                        |
| `"idle"`          | not ready (draining)         |
| `"running"`       | not ready (draining)         |
| `"backoff"`       | not ready (restarting)       |
| `"draining"`      | not ready (shutting down)    |
| `"failed"`        | not ready (terminal failure) |

Compose it with other checks (event store ping, downstream services):

```go
mux.Handle("/ready", cqrshtmx.ReadinessHandler(
    cqrshtmx.ProjectionReadinessCheck(svc),
    cqrshtmx.NewNamedCheck("event-store", db.Ping),
    cqrshtmx.NewNamedCheck("redis", redisClient.Ping),
))
```

---

## Failure Handling

With synchronous startup, a projection failure during drain is returned as a
`NewService` error — the application refuses to start. With async startup,
failures surface differently:

| Signal                        | How to observe                                    |
| ----------------------------- | ------------------------------------------------- |
| `/health` endpoint            | Returns 503 with the failed projection's error    |
| `ProjectionStatuses()`        | The worker shows `"failed"` + `LastError`         |
| `OnProjectionFailed` callback | Fires when the worker exhausts its restart budget |

Set `OnProjectionFailed` for alerting:

```go
svc, _ := usermgmt.NewService(usermgmt.ServiceConfig{
    AsyncStartup:       true,
    OnProjectionFailed: func(name, lastErr string) {
        slack.Alert("projection %s failed: %s", name, lastErr)
    },
})
```

---

## When to Use Async Startup

| Scenario                            | Recommendation                                                 |
| ----------------------------------- | -------------------------------------------------------------- |
| Production behind a reverse proxy   | **AsyncStartup: true** (recommended)                           |
| Integration tests                   | AsyncStartup: false (default — deterministic read-your-writes) |
| First-run bootstrap (empty journal) | Either — drain is instant on an empty journal                  |
| Large event journal                 | **AsyncStartup: true** (avoids linear drain growth)            |

### Combining with `CheckpointStore`

Async startup eliminates the outage window but does not reduce replay volume —
the journal is still fully replayed on every restart. To bound replay to only
new events, pair `AsyncStartup: true` with a persistent `CheckpointStore` and
SQL-backed read models (`ReadModelDB`):

```go
svc, _ := usermgmt.NewService(usermgmt.ServiceConfig{
    AsyncStartup:    true,
    ReadModelDB:     optimizedSQLDB,   // read models survive restart
    CheckpointStore: sqlCheckpointStore, // resume from last checkpoint
})
```

With both set, restart replays only events since the last checkpoint (seconds,
not minutes), **and** the server binds immediately.

---

## Backward Compatibility

`AsyncStartup` defaults to `false` (zero value), preserving the historical
synchronous startup behavior. Existing consumers see no change until they opt
in. The `/health` readiness check is drain-aware regardless of the setting —
in sync mode it simply never observes a draining state (drain completes before
the server starts).
