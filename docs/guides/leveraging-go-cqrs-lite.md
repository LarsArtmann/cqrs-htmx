# Leveraging go-cqrs-lite from cqrs-htmx

> How to get the most out of the 58-module `go-cqrs-lite` library when building on `cqrs-htmx`.
>
> **Audience:** application developers consuming `cqrs-htmx`, and maintainers deciding which upstream capabilities to surface. This guide is the result of a full capability-vs-usage audit (2026-07-30).

cqrs-htmx already leans heavily on go-cqrs-lite for the event-sourced core (`event`, `command`, `query`, `id`, `decider`, `projectionhost`, `storage`, `stack`, `codec`, `snapshot`, `kv`, `scenario`, `catalog`). But several powerful upstream modules are **available, composable, and entirely undocumented** in cqrs-htmx. The biggest of these — the dispatch `middleware` module — slots in with a single `.Use(...)` call.

---

## TL;DR — the adoption map

| go-cqrs-lite module              | Status in cqrs-htmx                              | How to leverage                                                                                                  |
| -------------------------------- | ------------------------------------------------ | ---------------------------------------------------------------------------------------------------------------- |
| `middleware` (27 factories)      | 🟡 **Undocumented, fully usable**                | `dispatcher.Use(middleware.CommandRetry(...))` — see [§1](#1-dispatch-middleware--the-1-undocumented-capability) |
| `otel` + `prometheus`            | 🟡 **Hooks exist, OTel/Prom wiring is on you**   | [§2](#2-opentelemetry--prometheus)                                                                               |
| `scheduling` (durable timers)    | 🔴 **Not used**                                  | [§3](#3-durable-deadline-scheduling)                                                                             |
| `signing` / `encryption`         | 🟢 **Exposed** via `ServiceConfig` hooks         | [§4](#4-event-signing--encryption)                                                                               |
| `catalog` (docs generation)      | 🟢 **Exposed** (`simple`, `docserver`, `events`) | [§5](#5-api--event-documentation)                                                                                |
| `scenario` (BDD testing)         | 🟢 **Used in usermgmt tests**                    | [§6](#6-scenario-based-testing)                                                                                  |
| `transport/http` (SSE broker)    | ⚪ **Intentionally not adopted**                 | [§7](#7-transporthttp-sse-broker)                                                                                |
| `deriver` (reactive sagas)       | 🔴 **Not used**                                  | [§8](#8-reactive-sagas-deriver)                                                                                  |
| `schema` (store-layer upcasters) | 🟢 **Covered** (decode-time, all paths)          | [§9](#9-schema-evolution-store-layer-upcasting)                                                                  |
| `graph` / `metaengine`           | ⚪ **Niche**                                     | [§10](#10-niche-modules)                                                                                         |

Legend: 🟢 exposed · 🟡 usable but under-documented · 🔴 available, not wired · ⚪ deliberately out of scope.

---

## 1. Dispatch middleware — the #1 undocumented capability

cqrs-htmx's `App` dispatches commands/queries through the **exact** `*command.Dispatcher` / `*query.Dispatcher` types you pass into `Config.Commands` / `Config.Queries`. Those dispatchers implement `Use(middleware ...Middleware)`, so **every go-cqrs-lite middleware factory composes with zero glue** — cqrs-htmx never tells you this.

This unlocks 27 production-grade middleware factories covering 9 concerns:

| Concern         | Command factory                                                              | What it gives you                                        |
| --------------- | ---------------------------------------------------------------------------- | -------------------------------------------------------- |
| Logging         | `middleware.CommandLogging(logger)`                                          | type, stream ID, duration, per attempt                   |
| Recovery        | `middleware.CommandRecovery()`                                               | panic → error (never crash the request)                  |
| Retry           | `middleware.CommandRetry(middleware.DefaultRetryConfig())`                   | exponential backoff on `errorfamily.IsRetryable`         |
| Circuit breaker | `middleware.CommandCircuitBreaker(middleware.DefaultCircuitBreakerConfig())` | failsafe-go breaker to stop cascading failures           |
| Metrics         | `middleware.CommandMetrics(recorder)`                                        | dispatch count / duration / error counters               |
| Tracing         | `middleware.CommandTracing(tracer)`                                          | per-command OTel spans (real `.Type()` in the span name) |
| Validation      | `middleware.CommandValidation(validator)`                                    | pre-handle validation                                    |
| Idempotency     | `middleware.CommandIdempotency(store, ttl, keyFn)`                           | at-least-once dedup by command ID                        |

(Query middleware mirrors these: `middleware.QueryRetry`, `middleware.QueryCircuitBreaker`, …)

### The recipe

```go
import "github.com/larsartmann/go-cqrs-lite/middleware/v4"

cmdDisp := command.NewDispatcher()

// Order is outer-to-inner. Recovery outermost so panics never escape; retry
// inside it so retryable errors re-dispatch; logging innermost to log each attempt.
cmdDisp.Use(middleware.CommandRecovery())
cmdDisp.Use(middleware.CommandRetry(middleware.DefaultRetryConfig(), middleware.WithLogger(logger)))
cmdDisp.Use(middleware.CommandCircuitBreaker(middleware.DefaultCircuitBreakerConfig()))
cmdDisp.Use(middleware.CommandLogging(logger))

// Register handlers, then hand the SAME dispatcher to cqrs-htmx:
app := cqrshtmx.MustNew(cqrshtmx.Config{Commands: cmdDisp, Queries: qryDisp})
```

`RetryConfig` and `CircuitBreakerConfig` default to sane values and classify retry/failure via the shared `errorfamily` taxonomy — so a `Transient` error (HTTP 503) is automatically retried, while a `Rejection` (400) is not.

> **⚠️ Published-version hazard:** go-cqrs-lite submodule tags currently have broken zero pseudo-versions (see `AGENTS.md` → "go-cqrs-lite publish bug"). If you are NOT using the local `go.work` replaces, pin `middleware/v4` to `v4.2.0` explicitly and verify `go build` succeeds before committing. The go-cqrs-lite local replaces in this repo's `go.work` are required until upstream cuts a clean consolidated release.

### Two recovery layers (HTTP vs dispatch)

You need **both** `cqrshtmx.RecoveryMiddleware` (HTTP layer) and `middleware.CommandRecovery()` (dispatch layer). They catch panics at different call sites:

| Layer    | Middleware                     | Catches panics in                      |
| -------- | ------------------------------ | -------------------------------------- |
| HTTP     | `cqrshtmx.RecoveryMiddleware`  | HTTP handler, decode, response writing |
| Dispatch | `middleware.CommandRecovery()` | Command handler body, domain logic     |

Neither is redundant. If a panic occurs inside the command handler (e.g. nil dereference in domain logic), the dispatch recovery catches it and converts it to a `Transient` error. If a panic occurs during JSON decoding or response writing, the HTTP recovery catches it. Using only one leaves a gap.

### Runnable proof

`examples/middleware-demo/` mounts a command whose handler fails transiently twice then recovers. The retry middleware makes the HTTP request still return **204**. Verified:

```
call 1: status=204 body="" took=376ms   ← retried twice, then succeeded
call 2: status=204 body="" took=0s       ← flaky service already recovered
```

> **Why this matters vs `BeforeDispatchHook`/`AfterDispatchHook`:** cqrs-htmx's HTTP-level hooks only see `(ctx, *http.Request)` — they cannot name a span by command type or read the decoded command. The dispatcher middleware runs **per-dispatch with the actual command**, so tracing/metrics/retry are far richer. Use hooks for HTTP-shaped concerns (request IDs, server-timing); use dispatcher middleware for CQRS-shaped concerns (per-type tracing, retry, circuit breaking).

---

## 2. OpenTelemetry & Prometheus

cqrs-htmx intentionally does **not** import `go.opentelemetry.io` (library principle: never enforce an observability dependency — see `example_otel_test.go`). Everything in this section is consumer-side wiring over two upstream modules (`go-cqrs-lite/otel`, `go-cqrs-lite/prometheus`), ordered from zero-dep to full end-to-end tracing:

1. [2.1 Hook-level tracing (dep-free)](#21-hook-level-tracing-dep-free)
2. [2.2 Dispatch tracing via middleware (per-type spans)](#22-dispatch-tracing-via-middleware-per-type-spans)
3. [2.3 Prometheus /metrics](#23-prometheus-metrics)
4. [2.4 Free domain spans — one `Setup` call traces the whole identity domain](#24-free-domain-spans--one-setup-call-traces-the-whole-identity-domain)
5. [2.5 HTTP root spans (otelhttp)](#25-http-root-spans-otelhttp)

### 2.1 Hook-level tracing (dep-free)

Wire `BeforeDispatch`/`AfterDispatch` to start/end a span. Coarse: the span cannot easily be named by command type. Documented in `example_otel_test.go`.

### 2.2 Dispatch tracing via middleware (per-type spans)

Pass an `cqrsotel.Tracer` from go-cqrs-lite's `otel` module into `middleware.CommandTracing(tracer)`. The span is created per-dispatch with the command type baked in. The `otel` module is the canonical re-export layer (never import `go.opentelemetry.io` directly):

```go
import cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v4"

tracer := cqrsotel.NewTracer("my-service")
cmdDisp.Use(middleware.CommandTracing(tracer))
qryDisp.Use(middleware.QueryTracing(tracer))
```

### 2.3 Prometheus /metrics

go-cqrs-lite's `prometheus` module bridges all CQRS OTel instruments to a `/metrics` endpoint:

```go
import cqrsprom "github.com/larsartmann/go-cqrs-lite/prometheus/v4"

provider, err := cqrsprom.Setup()
if err != nil {
    return fmt.Errorf("prometheus setup: %w", err)
}
defer provider.Shutdown(context.Background())
mux.Handle("/metrics", provider.Handler())
```

### 2.4 Free domain spans — one `Setup` call traces the whole identity domain

This is the capability most consumers miss: **go-cqrs-lite's decider and storage layers already emit OTel spans** around every aggregate load, command execution, and store round-trip — and they resolve those spans against the **global** tracer provider (`go-cqrs-lite/decider/otel.go` caches one `cqrsotel.NewTracer("decider")` via `sync.OnceValue`; the storage layer uses `sqlpkg.Tracer()`). `cqrsotel.Setup` registers that global provider. So a single call gives you full command/event/store visibility into the entire identity domain (usermgmt's four aggregates) with **zero middleware and zero cqrs-htmx API**:

```go
otelProvider, err := cqrsotel.Setup(
    cqrsotel.WithService("my-app", "1.0.0", "local"),
    // Dev: pretty-print spans to stdout. Production: WithSpanExporter(otlpExporter).
    cqrsotel.WithStdoutExporter(os.Stdout),
)
if err != nil {
    return fmt.Errorf("otel setup: %w", err)
}
defer otelProvider.Shutdown(context.Background())
```

Dispatch every usermgmt command (`RegisterUser`, `AddMembership`, …) and these spans appear:

| Span name                            | Emitted around                                  | Evidence (go-cqrs-lite)                            |
| ------------------------------------ | ----------------------------------------------- | -------------------------------------------------- |
| `decider.execute`                    | every aggregate command execution               | `decider/decider.go:123`                           |
| `decider.load`                       | every aggregate load (state cache miss)         | `decider/decider.go:324`                           |
| `decider.load_at_version`            | snapshot-aware loads                            | `decider/load.go:119`                              |
| `event.store.save` / `append_batch`  | SQL event-store appends                         | `storage/eventstore/event_store.go:81`             |
| `event.store.load*`                  | SQL event-store reads (stream, from-version, …) | `storage/eventstore/event_store_load.go:57`        |
| `command.store.save` / `load`        | SQL command-backed store round-trips            | `storage/command_store_save.go:29`, `command_store_load.go:38` |

Two scope notes, verified against source:

- The **decider** spans fire for every store backend (including the in-memory default) — they wrap repository load/execute, not storage I/O.
- The **storage** spans are emitted by the SQL store implementations (`storage/sql`, `storage/eventstore`); the in-memory dev store does not instrument I/O. Use a SQL backend to see the store layer of the trace.

`Setup` also registers the W3C `traceparent` propagator globally (see [2.5](#25-http-root-spans-otelhttp) and 2.6 Correlation below), so spans join upstream traces and correlation IDs flow into event metadata.

Runnable proof: `examples/observability-demo/` — one `cqrsotel.Setup` + one `cqrsprom.Setup`, then POST `/ping` and watch `decider.execute` spans hit stdout (§2.2 middleware adds the per-type dispatch span on top).

### 2.5 HTTP root spans (otelhttp)

§2.2/§2.4 spans start at the dispatch — consumer traces have no HTTP root span (no URL, method, status, latency) and extract no incoming `traceparent`. The fix is one wrapper in **your** code, no library change: wrap your mux with `otelhttp.NewHandler` and every dispatch/domain/store span nests under the request span automatically.

Why it works with zero cqrs-htmx changes: `App` dispatch reads `ctx := r.Context()` in `dispatchContext` (`handler.go:22`), and all dispatch middleware + decider/store spans are created from that context. `otelhttp` puts its request span into `r.Context()` — so an otelhttp-wrapped mux makes it the parent of everything downstream.

```go
import otelhttp "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

// cqrsotel.Setup (§2.4) registered the global TracerProvider AND the W3C
// propagator, so the plain constructor is enough — traceparent headers from
// upstream services are extracted and the request span joins their trace.
handler := otelhttp.NewHandler(chainedHandler, "my-app")

srv, err := httputil.NewServer(httputil.ServerConfig{Addr: addr}, handler)
```

What the root span adds to each trace: method + route + HTTP status as span attributes, end-to-end request latency, and `traceparent` extraction (multi-service traces stay connected). Run `cqrsotel.Setup` before wrapping — it must have registered the global provider/propagator first. For isolated needs (tests, multi-tenant providers) pass explicit options instead: `otelhttp.WithTracerProvider(tp)`, `otelhttp.WithPropagators(cqrsotel.NewTextMapPropagator())`.

Runnable proof: `examples/observability-demo/main.go` wraps its handler exactly like this, and `TestOtelHTTPRootSpanAndTraceparentExtraction` in the same directory asserts a server-kind root span exists and an incoming `traceparent` header is extracted (via an isolated `tracetest.SpanRecorder`).

> Use `httputil.Chain` INSIDE the otelhttp wrapper (recovery, security headers, session, CSRF, HTMX, enrichment) so the root span covers the full pipeline; keep `otelhttp` outermost.

> **Recommendation:** run §2.4 (free domain spans) everywhere, add §2.2 (per-type dispatch spans) + §2.3 (Prometheus) as needed, and wrap the mux with §2.5 (HTTP root spans) so traces start at the request. cqrs-htmx stays dep-free; consumers pull two upstream modules.

---

## 3. Durable deadline scheduling

go-cqrs-lite's `scheduling` module provides **durable timers** that survive restarts: _"cancel order after 30 min if unpaid"_, _"expire this session/token at T"_. cqrs-htmx does not use it — usermgmt currently handles time-based domain rules (session TTL, email-verification-token TTL, account-lockout duration) with **in-process sweepers** (`EvictStale()`, `EvictExpired()`). Those are not durable: a restart or multi-instance deploy misses expiries.

```go
import "github.com/larsartmann/go-cqrs-lite/scheduling/v4"

timerStore := scheduling.NewMemoryTimerStore[event]() // or storage.SQLTimerStore[T] for persistence
scheduler := scheduling.New(timerStore, func(ctx context.Context, t scheduling.Timer[event]) error {
    // fire a command when the deadline elapses, e.g. svc.ExpireSession(...)
    return nil
}, scheduling.WithPollInterval(time.Second))
defer scheduler.Close()
```

> **Evaluated and deferred:** durable `scheduling.TimerStore` was investigated via a design doc (`docs/design/durable-scheduling.md`). Conclusion: NOT needed — every expiry mechanism already has a lazy check (correctness preserved on restart), and the SQL store provides multi-instance safety for the longest-TTL items (sessions, 24h). See ROADMAP.md "Not Planned" for full rationale.

---

## 4. Event signing & encryption

Already exposed via `usermgmt.ServiceConfig` seams — no work needed, but worth knowing the full surface:

```go
import (
    "github.com/larsartmann/go-cqrs-lite/encryption/v4"
    "github.com/larsartmann/go-cqrs-lite/signing/v4"
)

cipher, err := encryption.NewAES256GCM(key)       // AES-256-GCM at-rest encryption
if err != nil { /* handle */ }
encryptedStore, err := encryption.NewEncryptedStore(store, cipher)
if err != nil { /* handle */ }

signer, err := signing.NewHMAC(hmacKey)             // HMAC-SHA256 in-transit signing
if err != nil { /* handle */ }

svc, _ := usermgmt.NewService(usermgmt.ServiceConfig{
    StoreWrapper:      encryptedStore,
    PublishMiddleware: []event.PublishMiddleware{signing.SignMiddleware(signer)},
    HandlerMiddleware: []event.Middleware{signing.VerifyMiddleware(signer)},
})
```

Both `NewService` and `NewEventSourcedSetup` expose the same hooks (no split brain). See `integration_test/signing_encryption_test.go` for a verified end-to-end recipe.

---

## 5. API & event documentation

go-cqrs-lite's `catalog` module auto-generates **AsyncAPI 3.0, OpenAPI, D2 diagrams, and EventCatalog MDX** from Go types via reflection, plus a Scalar-backed docserver. cqrs-htmx already exposes this path:

```go
import (
    "github.com/larsartmann/go-cqrs-lite/catalog/v4/simple"
    "github.com/larsartmann/go-cqrs-lite/catalog/v4/docserver"
)

b := simple.New("My Service", "1.0.0")
simple.Command[createItemRequest](b, "CreateItem", simple.WithOperation("POST", "/items"))
// ...register events/queries...

catalogProvider := func() *catalog.Catalog { return b.Build() }
docs := docserver.NewDocsServer(catalogProvider, docserver.Config{})
docs.Mount(mux) // serves OpenAPI/AsyncAPI HTML + D2 + health
```

`examples/catalog-demo/` and `integration_test/catalog_test.go` are verified references. For a lightweight, dependency-free OpenAPI 3.1 builder only, the root module also ships `openapi/` (`WithOpenAPI`, `OpenAPISpecHandler`).

---

## 6. Scenario-based testing

go-cqrs-lite's `scenario` module is a fluent BDD DSL for deciders and projections with **no store or bus required**. usermgmt already uses it (`es_scenario_*_test.go`); any consumer building event-sourced aggregates on top of cqrs-htmx should too:

```go
scenario.Given(t, applyEvent, initialState, priorEvents...).
    When(cmd, decide).
    Then("UserRegistered", "EmailVerified")   // expected emitted event types
// .ThenError(target) / .ThenState(fold, init, expected) also available
```

Faster, clearer, and more deterministic than standing up a full event store for each unit test.

---

## 7. `transport/http` SSE broker

go-cqrs-lite ships a production SSE broker (`transport/http.NewSSEBroker`) with **replay→live catch-up**, `Last-Event-ID`, a dedup ring at the handoff boundary, byte-budgeted replay, and CBOR→JSON payload transcoding. cqrs-htmx instead composes its own `Broadcaster` + `SSEStream` + `JournalSSEStore` (from `go-sse`) because its SSE layer is HTMX-aware (HTML fragment data, `HX-Redirect`, OOB swaps) — adopting `transport/http` would cross the documented _"building blocks, not a server"_ boundary (see FEATURES.md → "Not Planned": `broadcaster.ServeSSE()`).

> **Takeaway:** this is a **deliberate non-adoption**, not a gap. cqrs-htmx's `JournalSSEStore` already provides journal-backed reconnect replay for the dashboard. If you need broker-grade fanout (>500 clients) or CBOR stores with JSON browsers, reach for `transport/http` directly in your app — it interoperates with the same `event.Bus` cqrs-htmx uses.

---

## 8. Reactive sagas (`deriver`)

go-cqrs-lite's `deriver` module is the functional saga primitive: react to an event by emitting zero-or-more commands, deterministically and idempotently. cqrs-htmx does not surface it. Useful when one domain event should trigger a downstream command (e.g. `UserDeleted` → `CancelSubscriptions`):

```go
import "github.com/larsartmann/go-cqrs-lite/deriver/v4"

// Deriver is a function type with chainable combinators.
d := deriver.Deriver(func(ctx context.Context, e event.Event) ([]command.Command, error) {
    return []command.Command{buildCancelSubscriptionsCmd(e)}, nil
})

// Filter to a specific event type, make it idempotent, wire into the bus.
bus.SubscribeAll(d.Filter("UserDeleted").Idempotent().AsHandler(cmdDispatcher))
```

> Not for sagas needing compensation (use a process manager there).

---

## 9. Schema evolution — store-layer upcasting

identity-model ships its own `UpcasterRegistry` that upcasts event payloads **at decode time** via the shared `UnmarshalPayload[T]` helper. This covers every decode path — fold functions (`FoldUser`, `FoldMembership`, etc.), read models, and projections — because all of them route through `UnmarshalPayload`, which calls `applyUpcasters` as its first step. There is **no gap**: the `SetUpcasterRegistry` doc comment confirms it is _"used by all event decode paths (FoldUser, read models, projections)"_.

go-cqrs-lite also offers `schema.VersionedSeekableJournal`, which upcasts **at the store boundary** instead. This is an alternative approach (not complementary for cqrs-htmx's needs):

```go
import "github.com/larsartmann/go-cqrs-lite/schema/v4"

upcaster := schema.NewUpcaster("UserRegistered", 1, func(evt event.Event) (event.Event, error) {
    // v1→v2: mutate the event payload, return the upgraded event
    return evt, nil
})

vs, err := schema.NewVersionedSeekableJournal(journal, upcaster)
if err != nil { /* handle */ }
// pass `vs` to projectionhost.New(...) instead of the raw journal
```

> **Assessment:** decode-time upcasting already covers every projection path in cqrs-htmx (confirmed: `CasbinProjection`, `UserReadModel`, and `MembershipReadModel` all route through `UnmarshalPayload` → `applyUpcasters`). Store-layer upcasting via `schema.VersionedSeekableJournal` is an alternative, not a complement — it would only matter for consumers that decode raw `evt.Payload()` bytes directly, and no such consumer exists in the current projection path.

---

## 9b. State cache & snapshotting — built-in performance

cqrs-htmx's usermgmt module **automatically wires two performance optimizations** from go-cqrs-lite that most consumers don't know about:

### State cache (`decider.WithStateCache`)

Every `Service` and `EventSourcedSetup` repository includes `decider.WithStateCache` by default. This in-memory LRU cache stores the last-folded state per stream. On subsequent `Execute` calls for the same stream, the decider loads only events **since the cached version** instead of replaying the full history:

```
Without cache: Execute() → Load all events → Fold from version 0  → O(total events)
With cache:    Execute() → Load events since cached version       → O(new events)
```

The cache is process-local, best-effort, and auto-invalidated after every successful write. You do not need to configure anything — it is always on.

```go
// Already wired internally in usermgmt/snapshot.go:repositoryOptions():
// opts = append(opts, decider.WithStateCache[State](decider.NewStateCache[State](0)))
```

> **When it matters:** streams with 50+ events see meaningful speedup. For short streams (1-10 events), the fold cost is negligible and the cache adds minimal overhead.

### Snapshotting (`SnapshotConfig`)

For extremely long streams (1000+ events), enable snapshot persistence on `ServiceConfig.SnapshotConfig`:

```go
cfg := usermgmt.ServiceConfig{
    SnapshotConfig: usermgmt.SnapshotConfig{
        Store:   usermgmt.MemorySnapshotStore{}, // dev/test; use SQLSnapshotStore in prod
        Codec:   usermgmt.GobSnapshotCodec{},
        Strategy: usermgmt.SnapshotEveryN(100),
    },
    // ...
}
```

On cache miss (e.g., after a restart), the repository loads the snapshot and replays only the tail via `LoadFromVersion`. See ADR-0041 for details.

---

## 10. Niche modules

| Module                            | When you'd reach for it from a cqrs-htmx app                                                                                                              |
| --------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `graph`                           | Third projection tier (nodes + edges) for traversal-heavy read models — reply chains, social graphs, causation DAGs. Rarely needed for identity/usermgmt. |
| `metaengine`                      | Experimental cost-based storage planner. R&D only.                                                                                                        |
| `transport/grpc`                  | Remote command/query dispatch over gRPC (transparent: client implements the same `Dispatch` interface). Only if your architecture is polyglot-transport.  |
| `watermill` (`CatchUpSubscriber`) | Push-based ordered replay→live for projections across processes. cqrs-htmx uses the pull-based `projectionhost` instead; both are valid.                  |

---

## Decision summary

| Do this now                                                                             | Effort  | Value                 |
| --------------------------------------------------------------------------------------- | ------- | --------------------- |
| **Wire dispatcher middleware** (`Use(...)`) in your app — logging/retry/circuit-breaker | Trivial | High                  |
| **Add OTel tracing via `middleware.CommandTracing` + `prometheus.Setup()`**             | Low     | High (prod readiness) |
| **Use `scenario` for new aggregate unit tests**                                         | Low     | Medium                |
| cqrs-htmx: **document the middleware path** (this guide + `examples/middleware-demo`)   | Done ✅ | High                  |
| cqrs-htmx: **durable scheduling** for usermgmt expiry                                   | Medium  | Medium-High           |

The single highest-leverage change an **app developer** can make today is the one-line `dispatcher.Use(...)` in [§1](#1-dispatch-middleware--the-1-undocumented-capability). The single highest-leverage change **inside cqrs-htmx itself** is making that capability discoverable — which this guide and `examples/middleware-demo` now do.
