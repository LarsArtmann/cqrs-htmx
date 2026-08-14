# ADR-0048: Liveness/Readiness Decoupling for Projection Startup

## Status

ACCEPTED — 2026-08-14

## Context

cqrs-htmx's event-sourced setups (`usermgmt.NewService`, `usermgmt.NewEventSourcedSetup`, `setup.New`) historically block construction until every projection worker finishes its initial journal drain (`waitForDrain`). This preserves read-your-writes from the first request, but couples HTTP liveness to projection readiness:

- Restarting a deployment with a large journal means the process accepts no traffic for the entire replay window — often minutes.
- Load balancers see connection failures during that window, not a clean 503, because the server has not bound its listener yet.
- Rolling deploys serialize: the new instance cannot serve (or report healthy) until replay completes, extending the effective outage.

The drain exists for a good reason: serving reads from read models that have not caught up returns stale or missing data. The question is how to keep correctness while eliminating the startup outage.

## Decision

Decouple HTTP server liveness from projection readiness with two cooperating mechanisms:

1. **`AsyncStartup` config flag** (`ServiceConfig.AsyncStartup`, `EventSourcedConfig.AsyncStartup`, `setup.Config.AsyncStartup`, all default `false`). When true, construction starts the projection host and returns immediately after `host.Start()` — the HTTP server binds while projection workers replay the journal in the background. The mechanism is a single `block bool` passed to the shared `startProjectionHost` factory: `block=false` skips `waitForDrain`.

2. **Readiness gate: `cqrshtmx.ProjectionReadinessCheck(provider)`** — a `NamedCheck` for `ReadinessHandler` that returns 503 while any projection worker is `idle`/`running`/`backoff`/`draining`, 200 when all workers are `live` (or `stopped`), and 503 with a failure detail when any worker is `failed`. `setup.Bundle` mounts it at `/health` automatically.

The composition is the consumer's deployment contract:

```go
svc, _ := usermgmt.NewService(usermgmt.ServiceConfig{ /* ... */ AsyncStartup: true })

mux := http.NewServeMux()
mux.Handle("/health", cqrshtmx.ReadinessHandler(cqrshtmx.ProjectionReadinessCheck(svc)))
// start server immediately; LB retries /health until 200
```

### Semantics chosen deliberately

- **Backward compatible by default.** `AsyncStartup: false` preserves the historical blocking behavior exactly; no existing consumer changes behavior on upgrade.
- **Failures surface differently in async mode.** In sync mode a drain failure is a construction error; in async mode it is a terminal worker state observable via `ProjectionStatuses()`, the readiness endpoint (503 + detail), and the `OnProjectionFailed` callback. This is the honest trade: fast startup means failures arrive through monitoring surfaces, not constructor returns.
- **Liveness is the server; readiness is the projections.** The health endpoint answers readiness only. A dedicated liveness probe (or the listener itself) answers process health — a draining projection must not cause an orchestrator kill-and-restart loop, which would never converge.
- **Read-your-writes after readiness, not before.** Consumers must route user traffic (or at least reads) behind the readiness gate. The gate turning 200 means every read model has replayed the full journal.

## Alternatives considered

- **Blocking startup with a longer `DrainTimeout`** — keeps the outage; only makes failure louder. Rejected.
- **Serving stale reads with a `Stale-While-Revalidate` header during drain** — hides the consistency window instead of gating it; wrong default for an identity/admin system. Rejected as a default, viable as a consumer-level choice on top of async startup.
- **Per-projection readiness (partial 200)** — allows serving once _some_ projections are live. Rejected: correctness is only as good as the projection backing the request, and consumers cannot express per-route projection affinity through a single health endpoint. A future refinement could expose per-projection checks as separate `NamedCheck`s.

## Consequences

### Positive

- Restart outage window collapses from "full replay duration" to "process start" — the LB simply retries 503 until ready.
- Rolling deploys overlap: the new instance drains while the old one serves.
- The readiness endpoint is generic (`ProjectionStatusProvider`), so custom setups reuse the same gate.

### Negative

- Consumers must wire the readiness gate and configure their proxy/orchestrator to retry on 503; without it, early reads can hit incomplete read models.
- Drain failures no longer fail construction — monitoring (`/health`, `ProjectionStatuses`, `OnProjectionFailed`) becomes mandatory in async mode.
- The first requests after readiness may still be slightly stale for events written _during_ the drain tail; the gate bounds, but does not eliminate, the window (same as sync mode at steady state).

### Verification

`integration_test/async_startup_test.go` exercises the full lifecycle end to end: a real HTTP server with `AsyncStartup: true` answers `/health` with 503 while projections drain (a slowed `ReadFrom` journal makes the window deterministic), flips to 200 once caught up, and seeded users are then readable from the replayed read models.

## See also

- `docs/guides/async-projection-startup.md` — operational guide
- `docs/guides/projection-health-monitoring.md` — projection status monitoring
- `projection_readiness.go` — the readiness check implementation
- ADR-0031 (Superseded) — projection lifecycle via `projectionhost.Host`
