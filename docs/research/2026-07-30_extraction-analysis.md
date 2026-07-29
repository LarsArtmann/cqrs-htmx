# Extraction Analysis: What Could and Should Be Extracted from cqrs-htmx

> **Date:** 2026-07-30
> **Scope:** Full analysis of the cqrs-htmx root module and all sub-modules to identify what could and should be extracted into dedicated projects or consolidated into existing repos.
> **Methodology:** Read all 54 root-module source files, mapped every cross-cluster dependency, analyzed dep trees via `go mod why`, cross-referenced ROADMAP "Not Planned" decisions, compared against existing LarsArtmann repos (httputil, go-sse, go-cqrs-lite), and researched the competitive landscape for each extraction candidate.

---

## Table of Contents

1. [Current Module Structure](#current-module-structure)
2. [Root Module Internal Dependency Map](#root-module-internal-dependency-map)
3. [Extraction Candidates Evaluated](#extraction-candidates-evaluated)
   - [Tier 1: Strongly Should Extract/Consolidate](#tier-1-strongly-should-extractconsolidate)
   - [Tier 2: Should Consolidate into httputil](#tier-2-should-consolidate-into-httputil)
   - [Tier 3: Could Extract (Lower Priority)](#tier-3-could-extract-lower-priority)
   - [Tier 4: Should Not Extract](#tier-4-should-not-extract)
4. [The openapi/ Question: Competitive Landscape Analysis](#the-openapi-question-competitive-landscape-analysis)
5. [Duplication Audit: cqrs-htmx vs httputil](#duplication-audit-cqrs-htmx-vs-httputil)
6. [Summary: Pareto-Optimal Actions](#summary-pareto-optimal-actions)

---

## Current Module Structure

cqrs-htmx is a **multi-module Go workspace** with 15 independent Go modules under one `go.work`:

| Module | Import Path | LOC (non-test) | Provides |
|---|---|---|---|
| **root** | `github.com/larsartmann/cqrs-htmx/v4` | ~25,900 | Core library: HTTP handler builder, HTMX/SSE/WS helpers, authz (Casbin), CSRF, rate limiting, security headers, error mapping, pagination, `openapi/` sub-package |
| **identity-model** | `identity-model/v4` | — | Pure domain types for event-sourced identity management (IDs, events, commands, fold functions, Authz engine, constants) |
| **usermgmt** | `usermgmt/v4` | — | Event-sourced CQRS user management (thin re-export layer over identity-model + infrastructure) |
| **usermgmt/totp** | `usermgmt/totp/v4` | — | TOTP MFA auth strategy |
| **usermgmt/webauthn** | `usermgmt/webauthn/v4` | — | WebAuthn passkey auth strategy |
| **usermgmt/oauth2** | `usermgmt/oauth2/v4` | — | OAuth2/OIDC auth strategy |
| **adminui** | `adminui/v4` | — | Ready-made admin dashboard (templ + HTMX) |
| **loginpage** | `loginpage/v4` | — | Ready-made passwordless login page |
| **dashboardui** | `dashboardui/v4` | — | CQRS/ES observability dashboard (templ + HTMX) |
| **integration_test** | — | — | Cross-module bridge tests |
| **examples/** (6) | — | — | basic, datastar-demo, catalog-demo, admin-demo, dashboard-demo, middleware-demo, observability-demo |

**Dependency direction:** identity-model ← usermgmt (type aliases). Root → usermgmt is zero imports (clean boundary). Auth strategies → root/usermgmt via interfaces only. adminui/loginpage → root + usermgmt. dashboardui → root + usermgmt. Nothing depends on adminui, loginpage, or dashboardui.

**Root module direct dependencies (17 total):**

| Dependency | Used By | Extraction Relevant? |
|---|---|---|
| `github.com/casbin/casbin/v3` | authz.go, errors.go (duck-typed `Enforcer` interface) | No — the root only duck-types the interface, doesn't import casbin directly |
| `github.com/go-playground/form/v4` | decoder.go | Yes — form decoding |
| `github.com/justinas/nosurf` | csrf_*.go (4 files) | **Yes — CSRF is self-contained** |
| `github.com/larsartmann/go-branded-id` | Multiple files (ID types) | No — core ID types |
| `go-cqrs-lite` (6 packages) | Multiple files (command/event/query/id/idempotency) | No — core CQRS types |
| `github.com/larsartmann/go-error-family` | Multiple files (error classification) | No — core error model |
| `github.com/larsartmann/go-sse` | sse_event.go, sse_store.go, ws_broadcaster.go | No — core SSE building blocks |
| `github.com/larsartmann/httputil` | ratelimit_config.go (ClientIP), options_json.go (ParseUintQuery) | **Yes — already partially used** |
| `github.com/oklog/ulid/v2` | context.go, sse_event.go, logging.go | No — core ID generation |
| `github.com/onsi/ginkgo/v2` + `gomega` | Test files only | No — test-only |
| `golang.org/x/time` | ratelimit_config.go, ratelimit_middleware.go | **Yes — only for rate limiting** |

---

## Root Module Internal Dependency Map

The root module has ~54 non-test source files. Here is the complete internal dependency map, grouped by cluster:

### Key Shared Types (The "Spine" Everything Couples To)

| Symbol | Defined In | Referenced By |
|---|---|---|
| `App` struct | `app.go:20` | WS (ws_dispatch.go) |
| `AfterDispatchHook` | `app.go:110` | SSE (sse_broadcaster, ack), WS (ws_broadcaster, ack) |
| `handlerConfig` | `options_types.go:54` | CSRF (csrf_handler), OpenAPI (options_openapi) |
| `HandlerOption` | `options_types.go:30` | CSRF (csrf_handler), OpenAPI (options_openapi) |
| `hashTag` | `options_openapi.go:64` | Event catalog, Projection status |
| `serializeToImmutableHandler` | `event_catalog_handler.go` | OpenAPI (options_openapi) |
| `ContentTypeJSON/Plain` | `constants.go` | Event catalog, Projection, CSRF, OpenAPI |
| `NewStructuredError` | `structured_error.go:81` | SSE (sse_broadcaster), WS (ws_broadcaster) |
| `OOBHTML/SwapStrategy` | `partial.go:45` / `htmx.go:37` | WS (ws.go, ws_broadcaster) |
| `serveJS` | `htmx_serve.go:34` | Offline sync (sync_serve) |
| `marshalJSONOrFallback` | `httputil.go:42` | SSE (ack) |
| `SSEEvent/SSEEventID/SSEStream` | `sse_event.go` | All SSE files, ack |

### Cluster Coupling Matrix

| Cluster | go-cqrs-lite | casbin | App/handlerConfig | Primary Root Coupling |
|---|---|---|---|---|
| **SSE** (5 files, ~900 LOC) | event/v4, id/v4 (event_store_sse only) | No | `AfterDispatchHook` (app.go) | sse_event.go types; ack.go bridges to WS |
| **WebSocket** (4 files, ~700 LOC) | command/v4, event/v4, query/v4 | No | `*App` methods (app.go) — ws_dispatch tightly bound | partial.go/htmx.go (OOBHTML/SwapStrategy); app.go |
| **CSRF** (6 files, ~905 LOC) | No | No | `handlerConfig` via csrf_handler only | constants.go, errors.go — otherwise self-contained |
| **Offline sync** (2 files, ~100 LOC) | No | No | No | htmx_serve.go (`serveJS`) |
| **OpenAPI** (1 root file + 5 sub-pkg files, ~700 LOC) | No | No | `handlerConfig` via WithOpenAPI | options_types.go; openapi/ sub-package (one-way) |
| **Event catalog** (2 files, ~300 LOC) | No | No | No | constants.go, options_openapi.go (`hashTag`) |
| **Projection status** (1 file, ~100 LOC) | No | No | No | constants.go, options_openapi.go (`hashTag`) |
| **Idempotency** (1 file, ~50 LOC) | idempotency/v4 | No | No | none |
| **Rate limiting** (2 files, ~392 LOC) | No | No | No | none — standalone |
| **Server-Timing** (1 file, ~389 LOC) | No | No | No | none — standalone |
| **Recovery** (1 file, ~131 LOC) | event/v4 (errorfamily classification) | No | No | structured_error.go, context.go |
| **Security headers** (1 file, ~157 LOC) | No | No | No | constants.go |
| **Middleware chain** (1 file) | No | No | No | none — standalone |

### Detailed Cluster Dependency Analysis

#### 1. SSE Cluster (sse_broadcaster.go, sse_event.go, sse_store.go, event_store_sse.go, ack.go)

- **sse_event.go:** Leaf file. Pure type aliases (`SSEEvent`, `SSEEventID`, `SSEStream`) + delegating functions wrapping `go-sse`. No root coupling.
- **sse_store.go:** References only sse_event.go types. Internal to cluster.
- **event_store_sse.go:** Imports `go-cqrs-lite/event/v4`, `id/v4`. The bridge between go-cqrs-lite event journal and SSE replay. References sse_event.go/sse_store.go types.
- **sse_broadcaster.go:** References `AfterDispatchHook` (app.go), `NewStructuredError` (structured_error.go), sse_event.go types.
- **ack.go:** The cross-cluster bridge — attaches ACK hooks to both `Broadcaster` (SSE) and `WSBroadcaster` (WS). References httputil.go, app.go, sse_broadcaster.go, ws_broadcaster.go.

#### 2. WebSocket Cluster (ws.go, ws_broadcaster.go, ws_dispatch.go, ws_encoder.go)

- **ws.go:** References `OOBHTML` (partial.go), `SwapStrategy` (htmx.go). Uses event/v4 for error classification.
- **ws_broadcaster.go:** Reuses `go-sse`'s `Broadcaster[string]` internally. References AfterDispatchHook, structured_error.go, ws.go.
- **ws_dispatch.go:** **Most coupled file to App.** Defines methods on `*App` (`DispatchWSCommand`, `DispatchWSQuery`), uses private App internals (`a.commands`, `a.queries`, `a.beforeDispatch`, `a.afterDispatch`, `a.timeoutCtx`). Imports command/v4, event/v4, query/v4.
- **ws_encoder.go:** References WSMessage types from ws.go. Internal to cluster.

#### 3. CSRF Cluster (csrf_config.go, csrf_context.go, csrf_handler.go, csrf_helpers.go, csrf_middleware.go, csrf_testing.go)

- **Self-contained** — no go-cqrs-lite, no casbin, no App.
- Only dep: `github.com/justinas/nosurf`
- Only touch point to the core: `csrf_handler.go` writes `handlerConfig.csrfConfig` via `CSRFProtect` returning a `HandlerOption`.
- All 6 files reference only each other + constants.go (ContentTypePlain) and errors.go (ErrForbidden, ErrCSRFConfig).

#### 4. Offline Sync Cluster (sync_embed.go, sync_serve.go)

- **sync_embed.go:** Pure leaf. Embeds `sync/*.js`, defines `syncVersion`.
- **sync_serve.go:** References `serveJS` (htmx_serve.go). Couples to the HTMX-serving utility layer.

#### 5. OpenAPI Cluster (options_openapi.go root + openapi/ sub-package)

- **options_openapi.go (root):** `WithOpenAPI` returns `HandlerOption`, sets `cfg.openapiMeta` on `handlerConfig`. References `serializeToImmutableHandler` (event_catalog_handler.go) for the immutable-JSON handler pattern. Defines `hashTag` consumed by event_catalog_handler.go and projection_status_handler.go.
- **openapi/ sub-package (builder.go, doc.go, marshal.go, schema.go, types.go):** **Zero dependencies** — pure stdlib (`encoding/json/v2` in marshal.go; no imports at all in builder.go, types.go, schema.go). The only directional dependency is root `options_openapi.go` → `openapi/` (never reverse). `doc.go` explicitly states: "no dependency on the root package."

#### 6. Event Catalog Cluster (event_catalog.go, event_catalog_handler.go)

- **event_catalog.go:** Self-contained data types (`EventCatalog`, `EventMetadata`, `PayloadField`) with a `JSON()` method. Only imports errorfamily.
- **event_catalog_handler.go:** The shared immutable-JSON-serving utility (`serializeToImmutableHandler`, `immutableJSONServer`, `newImmutableJSONHandler`) used by both `EventCatalogHandler` and `OpenAPISpecHandler`. References constants.go, options_openapi.go (`hashTag`).

#### 7. Projection Status (projection_status_handler.go)

- Lightweight. Only 2 root-module touch points: constants.go (`ContentTypeJSON`), options_openapi.go (`hashTag`). Defines `ProjectionStatusProvider` interface (satisfied by `usermgmt.Service`).

#### 8. Idempotency (idempotency.go)

- Pure thin-delegation type aliases to `go-cqrs-lite/idempotency`. No root coupling.

#### 9. Rate Limiting (ratelimit_config.go, ratelimit_middleware.go)

- **Fully standalone** — no root coupling at all. Imports only `golang.org/x/time/rate`, `httputil.ClientIP`, and errorfamily.
- Rich design: min-heap eviction, `MaxKeys` cap, `KeyExtractor` interface, monitoring API (`ActiveKeys()`, `Check()`).

#### 10. Server-Timing (server_timing.go)

- **Fully standalone** — 389 LOC, no root coupling, no external deps. Full W3C Server-Timing spec implementation.

---

## Extraction Candidates Evaluated

### Tier 1: Strongly Should Extract/Consolidate

#### ~~1. `openapi/` → standalone repo~~ **REJECTED — see competitive analysis below**

> See [The openapi/ Question](#the-openapi-question-competitive-landscape-analysis) for why this was rejected after market research.

#### 2. Server-Timing → move to `httputil`

| Criterion | Assessment |
|---|---|
| **Self-contained?** | **100%.** 389 LOC, zero dependencies, zero root-module coupling. |
| **Broadly useful?** | Yes — Server-Timing is a general HTTP performance header, not a CQRS/HTMX concern. |
| **Already duplicated?** | **No counterpart in httputil** — this is unique to cqrs-htmx. |
| **Composability payoff?** | httputil gains a feature no other Go middleware library has. cqrs-htmx loses no functionality (just changes an import). |
| **Effort** | Low — move 1 file + test files, update imports. |
| **Risk** | Near zero. |

**Extraction plan:**
1. Copy `server_timing.go` + `server_timing_test.go` → httputil repo
2. cqrs-htmx `go.mod` adds `require github.com/larsartmann/httputil` (already present)
3. cqrs-htmx replaces `ServerTiming`/`ServerTimingMiddleware`/`MeasureServerTiming` with httputil versions (re-export type aliases for backward compat)

---

### Tier 2: Should Consolidate into httputil

#### 3. Rate limiting → consolidate with `httputil`

| Criterion | Assessment |
|---|---|
| **Self-contained?** | **100%.** No root-module coupling. Only dep: `golang.org/x/time/rate`. |
| **Duplicated?** | **Yes** — httputil has `ratelimit.go` (token bucket, interface-based, simpler). cqrs-htmx version is richer (min-heap eviction, `MaxKeys` cap, monitoring API). |
| **Composability payoff?** | Eliminates `golang.org/x/time` from cqrs-htmx dep tree. Merges two implementations into one canonical version. |
| **Effort** | Medium — merge the best of both into httputil, then cqrs-htmx delegates. |
| **Risk** | Low — API surface changes are backward-compatible via re-exports. |

**cqrs-htmx version advantages over httputil:**
- Min-heap eviction (O(log n) vs httputil's O(n) full map scan)
- `MaxKeys` cap to prevent unbounded memory growth
- `ActiveKeys()` and `Check()` monitoring methods
- `KeyExtractor` + helpers (`KeyExtractorFromRemoteAddr`, `KeyExtractorFromClientIP`)
- Configurable `OnAllowed`/`OnRejected` hooks

#### 4. CSRF middleware → move core to `httputil`

| Criterion | Assessment |
|---|---|
| **Self-contained?** | **Nearly.** 6 files, 905 LOC. Only dep: `justinas/nosurf`. Only touch point to cqrs-htmx core: `csrf_handler.go` writing `handlerConfig.csrfConfig`. |
| **Broadly useful?** | Yes — CSRF protection is a general HTTP concern. httputil currently has NO CSRF support. |
| **Composability payoff?** | Removes `justinas/nosurf` from cqrs-htmx's dep tree. httputil gains a missing feature. |
| **Effort** | Medium — move the 4 core files (config, context, middleware, helpers, testing), keep `csrf_handler.go` in cqrs-htmx as the `HandlerOption` adapter. |
| **Risk** | Low — the `CSRFProtect` handler option stays in cqrs-htmx as a thin adapter. |

**What moves to httputil:**
- `CSRFConfig`, `CSRFToken`, cookie/header/field accessors
- `CSRFMiddleware`, `CSRFResponseHeaderMiddleware`
- `CSRFTokenFromContext`, `WithCSRFToken`
- `CSRFTestToken` (test helper)

**What stays in cqrs-htmx:**
- `CSRFProtect` (returns `HandlerOption`, writes `handlerConfig`)

#### 5. Recovery + Security headers → enrich httputil, then delegate

| Criterion | Assessment |
|---|---|
| **Self-contained?** | Nearly — recovery.go has minor coupling to structured_error.go and context.go. Security headers reference only constants.go. |
| **Duplicated?** | **Yes** — both are reimplementations of httputil's `recovery.go` and `security.go`. cqrs-htmx versions are richer supersets. |
| **Composability payoff?** | Eliminates duplication. httputil becomes the canonical home. cqrs-htmx delegates. |
| **Effort** | Medium — port cqrs-htmx extensions upstream, then cqrs-htmx re-exports. |
| **Risk** | Low-medium — need to ensure the error-family classification in recovery is compatible with httputil's design. |

**cqrs-htmx recovery.go advantages over httputil:**
- Pluggable `ErrorHandler` (vs httputil's hardcoded 500 text/plain)
- `http.ErrAbortHandler` re-raising (httputil swallows it)
- Error classification as `event.Infrastructure` via errorfamily
- Request context recovery (RequestID + CorrelationID)

**cqrs-htmx security.go advantages over httputil:**
- `PermissionsPolicy` header support
- `Custom map[string]string` for arbitrary headers
- `SecurityHeaderSkip` (`"-"`) sentinel to suppress any default header
- `RecommendedHSTS` / `RecommendedCSP` convenience constants

---

### Tier 3: Could Extract (Lower Priority)

#### 6. Event Catalog → consolidate with `go-cqrs-lite/catalog/v4`

- ~300 LOC across `event_catalog.go` + `event_catalog_handler.go`
- Already noted in ROADMAP: *"evaluate consolidating the hand-rolled `EventCatalog`/`openapi/` with `catalog/v4`"*
- This is a **CQRS concern** (Published Language pattern), not an HTTP/HTMX concern
- go-cqrs-lite already has `catalog/v4` — this belongs there
- Minor coupling: `hashTag` and `serializeToImmutableHandler` shared with OpenAPI (extract the immutable-handler utility to httputil as an `ETagHandler`)

---

### Tier 4: Should Not Extract

| Candidate | Why Not |
|---|---|
| **SSE/WS building blocks** | Thin wrappers over `go-sse` (already extracted). The value-add (`AfterDispatchHook` integration) is cqrs-htmx-specific. |
| **HTMX serving/embedding** | Too thin (~200 LOC). Just serves embedded JS with ETag/cache headers. Not enough standalone value. |
| **Offline sync** | Two files (~100 LOC). Tightly coupled to `serveJS` pattern. Too thin. |
| **usermgmt god-package split** | Already deferred to v5 per ROADMAP. Zero consumer benefit while in same repo — same `go.mod` = same dep tree. |
| **Root SSE/WS/ratelimit into sub-modules** | Explicitly "Not Planned" in ROADMAP: same `go.mod` = same dep tree = zero consumer benefit within a repo. |
| **Identity-model further extraction** | Already extracted. Works perfectly. No action needed. |
| **Idempotency** | 50 LOC of type aliases to go-cqrs-lite. Nothing to extract. |
| **Projection status** | 100 LOC. Too thin. The `ProjectionStatusProvider` interface is designed for cqrs-htmx consumers. |

---

## The openapi/ Question: Competitive Landscape Analysis

### Initial Assessment (Pre-Research)

The `openapi/` sub-package appeared to be the #1 extraction candidate:
- 577 LOC across 5 files
- **Zero dependencies** — pure stdlib (`encoding/json/v2` only in marshal.go)
- Self-contained sub-package with explicit "can be used standalone" documentation
- Currently buried inside a CQRS+HTMX+Casbin+SSE+WS+CSRF library
- Today, using the OpenAPI builder drags in 17 direct deps (casbin, nosurf, x/time, go-cqrs-lite×6, go-sse, etc.)

### Market Research: The Go OpenAPI Library Landscape

The space is **saturated**:

| Library | Stars | Approach | OpenAPI 3.1? | Type |
|---|---|---|---|---|
| **swaggo/swag** | ~12.9k | Annotation-driven code→spec | No (2.0 only) | Spec generator |
| **go-swagger** | ~10k | Full toolkit (spec↔codegen) | No (2.0 only) | Toolkit |
| **oapi-codegen** | ~8.5k | Spec→code codegen | Yes (3.0+3.1) | Codegen |
| **goa** | ~6.1k | DSL→everything framework | Yes (3.0) | Framework |
| **huma** | ~4.3k | Framework, reflection-based spec gen from types | Yes (3.1) | Framework |
| **kin-openapi** | ~3.3k | Parse/validate/route against specs | Yes (3.0+3.1) | Library |
| **fuego** | ~1.7k | Framework, generics-based spec gen | No (3.0) | Framework |
| **pb33f/libopenapi** | ~860 | High-performance parser/validator | Yes (3.0+3.1+3.2) | Library |
| **swaggest/openapi-go** | ~115 | **Programmatic builder** (closest competitor) | No (3.0) | Builder |

### What cqrs-htmx's openapi/ Does Differently

1. **Explicit, imperative fluent builder** — no reflection, no annotations, no codegen. You construct the spec by hand:
   ```go
   spec := openapi.New("My API", "1.0.0").
       Path("/items",
           openapi.Post("CreateItem").
               Summary("Create a new item").
               JSONBody(openapi.Object(
                   openapi.Prop("name", openapi.String().MinLength(1)),
               )).
               Response(201, "Created"),
       )
   ```
2. **Zero dependencies** — pure stdlib
3. **577 LOC** — tiny
4. **OpenAPI 3.1** (most programmatic builders target 3.0)

### The Problem: Wrong Side of Market Demand

**Nobody in the market wants manual spec building.** The trends are:
- **Code-first with reflection** (huma, fuego) — spec is a free byproduct of writing typed handlers
- **Spec-first codegen** (oapi-codegen) — spec is the source of truth, code is generated
- **Annotation-driven** (swaggo/swag) — bolt docs onto existing code

The cqrs-htmx builder occupies the narrow "I want to hand-write an OpenAPI spec in Go without a framework" niche. The closest competitor (`swaggest/openapi-go`) has only **115 importers**. That's the market size signal.

**What it's missing vs. the big players:** no validation, no parsing, no routing, no schema inference from Go types, no codegen. It's just a typed JSON serializer with a fluent API.

### Verdict: REJECTED for Standalone Extraction

**Don't extract it as a standalone project.** Two better options:

1. **Keep it in cqrs-htmx** — it's small, self-contained, zero-dep, and serves cqrs-htmx's own `WithOpenAPI` option. Nobody outside cqrs-htmx is asking for it. The 577 LOC isn't weighing down the module.

2. **Replace it with `kin-openapi`** — if you want to stop maintaining a hand-rolled builder, depend on the established library (3.3k stars, full parse/validate/generate). But this adds a heavy dependency for a feature that's currently zero-dep. Not recommended — the current builder is fine for its purpose.

---

## Duplication Audit: cqrs-htmx vs httputil

cqrs-htmx root already imports `httputil` for only 2 things: `ClientIP` (in ratelimit_config.go) and `ParseUintQuery` (in options_json.go). It does NOT use httputil for recovery, security headers, rate-limit middleware, or server timing — those are all reimplemented.

### 1. Recovery Middleware — DUPLICATE (reimplemented + significantly extended)

| Aspect | httputil `recovery.go` | cqrs-htmx `recovery.go` |
|---|---|---|
| API | `Recovery(logger *slog.Logger) Middleware` | `RecoveryMiddleware(next)` standalone + `App.RecoverHandler()` method |
| Response | Hardcoded 500 text/plain | Delegates to pluggable `ErrorHandler` |
| `http.ErrAbortHandler` | Not handled (swallowed) | Re-raised per net/http convention |
| Logging | Injected `*slog.Logger` | `slog.ErrorContext` (default logger), structured fields |
| Error classification | None | Classifies as `event.Infrastructure` via `errorfamily`, code `"panic"` |
| Request context | None | Recovers RequestID + CorrelationID into context |

**cqrs-htmx version is a superset. Not a wrapper.**

### 2. Security Headers — DUPLICATE (reimplemented + extended)

| Aspect | httputil `security.go` | cqrs-htmx `security.go` |
|---|---|---|
| Config shape | `ContentTypeNosniff bool`, `FrameOptions`, `ReferrerPolicy`, `CSP`, `HSTS` | `ContentTypeOptions string` + same + **`PermissionsPolicy`** + **`Custom map[string]string`** |
| Skip/suppress | No (bool toggle) | `SecurityHeaderSkip = "-"` sentinel to omit any header |
| Defaults helper | `DefaultSecurityHeadersConfig()` | `DefaultRateLimiterConfig`-style + **`RecommendedHSTS`**, **`RecommendedCSP`** constants |
| Headers set | 5 | 7 (adds Permissions-Policy + custom) |

**Same concept reimplemented. cqrs-htmx is richer. Not a wrapper.**

### 3. Rate Limiting — DUPLICATE (reimplemented with different design)

| Aspect | httputil `ratelimit.go` | cqrs-htmx `ratelimit_config.go` + `ratelimit_middleware.go` |
|---|---|---|
| Abstraction | `RateLimiter` interface + `TokenBucketLimiter` impl | Concrete `perKeyLimiter` (no interface) |
| Eviction | Lazy `sweep()`: O(n) full map scan on every TTL interval | **Min-heap**: O(log n) eviction + `MaxKeys` cap eviction |
| Config | `RateLimitConfig{Limiter, KeyFunc, Status, OnDenied}` | `RateLimiterConfig{Limit, Window, Burst, KeyExtractor, TTL, MaxKeys, OnAllowed, OnRejected, RejectionHandler}` |
| Monitoring | None | `RateLimiter` struct exposes `ActiveKeys()`, `Check()` |
| Key extractor | `KeyFunc` (defaults to `RemoteAddr`) | `KeyExtractor` + helpers `KeyExtractorFromRemoteAddr`, `KeyExtractorFromClientIP` |

**Reimplemented from scratch. The only httputil touchpoint is `KeyExtractorFromClientIP → httputil.ClientIP`.**

### 4. Server-Timing — UNIQUE to cqrs-htmx (no duplication)

`server_timing.go` does not exist in httputil. The full W3C Server-Timing implementation lives only in cqrs-htmx. No counterpart, no duplication.

---

## Summary: Pareto-Optimal Actions

### If you do ONE thing

**Move Server-Timing to `httputil`.** It's 389 LOC, zero deps, zero coupling, unique to cqrs-htmx but general-purpose by nature. httputil gains a feature no other Go middleware library has. Effort: ~1 hour. Risk: near zero.

### If you do THREE things

Add **Server-Timing** and **CSRF** to httputil (they're general HTTP concerns already missing there), and **consolidate rate-limiting**. This removes `nosurf` and `golang.org/x/time` from cqrs-htmx's dep tree and makes httputil the canonical home for HTTP middleware in the LarsArtmann ecosystem.

### If you do FIVE things

Also port **recovery** and **security headers** extensions upstream into httputil (cqrs-htmx versions are richer supersets), then have cqrs-htmx delegate to httputil. This turns cqrs-htmx's root module into a lean CQRS+HTMX integration layer that delegates all general HTTP concerns to httputil — the Unix-philosophy ideal: each module does one thing well.

### What NOT to extract

- **openapi/** — market is saturated, the manual-builder niche has ~115 importers. Keep it in cqrs-htmx.
- **SSE/WS** — thin wrappers over already-extracted `go-sse`. cqrs-htmx-specific value-add.
- **HTMX serving/sync** — too thin, too coupled.
- **usermgmt god-package** — deferred to v5, zero consumer benefit while same repo.
- **identity-model** — already extracted, works perfectly.

### Dep Tree Impact of Recommended Actions

| Action | Deps removed from cqrs-htmx root |
|---|---|
| Server-Timing → httputil | none (already zero-dep) |
| CSRF → httputil | `justinas/nosurf` |
| Rate limiting → httputil | `golang.org/x/time` |
| Recovery → httputil (enrich + delegate) | none (already uses errorfamily, shared dep) |
| Security headers → httputil (enrich + delegate) | none |

**Combined: removes 2 heavy external deps from cqrs-htmx root**, establishes httputil as the canonical HTTP middleware repo, and eliminates 3 duplicated reimplementations.

---

## Appendix A: Huma Deep Dive — Is Reflection-From-Types the Better Idea?

> Added 2026-07-30 after a deeper investigation into `danielgtaylor/huma` (v2), prompted by the question "Code-first with reflection (huma, fuego) — feel like the better idea or not?"

### Corrections to Earlier Claims

Several statements in the main analysis (above) about huma were **wrong**. This section corrects them with verified facts from the source code and documentation.

| Earlier Claim | Reality (Verified) |
|---|---|
| "huma/fuego are frameworks that own the router" | **Wrong.** Huma is explicitly router-agnostic via an `Adapter` interface. 9 official adapters ship: net/http (`humago`), Chi (`humachi`), Gin (`humagin`), Echo (`humaecho`), Fiber (`humafiber`), gorilla/mux (`humamux`), httprouter (`humahttprouter`), bunrouter (`humabunrouter`), Flow (`humaflow`). You bring your own router. |
| "every endpoint is JSON-in/JSON-out" | **Wrong.** Huma supports multipart forms (`form:` struct tags, typed file uploads via `FormFile`/`[]FormFile`), SSE (first-class `sse` subpackage with typed event maps), streaming responses (arbitrary byte streams via `huma.StreamResponse`), and HTML/HTMX (documented how-to guide with templ + gomponents integration). |
| "reflection can't work without owning the router" | **Wrong.** Router ownership is not the blocker. Huma's `Register[I, O]()` reflects on the generic type parameters at registration time, independent of which adapter handles the routing. |
| "reflection means the library can't see the path/method" | **Wrong.** The `huma.Operation` struct carries `Method` and `Path` explicitly — the developer provides them at registration. Reflection is only used to infer schemas from the `I` (input) and `O` (output) type parameters. |

### How Huma Actually Works (Verified from Source)

#### Handler Registration

The universal handler signature:

```go
func(ctx context.Context, input *I) (*O, error)
```

Where `I` and `O` are generic type parameters (must be structs). Two registration styles:

```go
// Low-level: explicit Operation
huma.Register(api, huma.Operation{
    OperationID: "get-greeting",
    Method:      http.MethodGet,
    Path:        "/greeting/{name}",
    Summary:     "Get a greeting",
}, func(ctx context.Context, input *GreetingInput) (*GreetingOutput, error) { ... })

// Convenience wrapper (auto-generates OperationID + Summary)
huma.Get(api, "/greeting/{name}", func(ctx context.Context, input *struct{
    Name string `path:"name" maxLength:"30" example:"world" doc:"Name to greet"`
}) (*GreetingOutput, error) { ... })
```

#### Spec Generation Mechanism

- **When:** At **registration time** (when you call `huma.Register`/`huma.Get`/etc.), not build-time, not first-request.
- **How:** `Register[I, O]()` calls `reflect.TypeFor[I]()` and `reflect.TypeFor[O]()` to introspect the input and output structs. `processInputType()` and `processOutputType()` in `huma.go` walk the struct fields, reading struct tags to build OpenAPI parameters, request bodies, and response schemas.
- **Serialization:** The `/openapi.json` spec is serialized lazily on first request (cached in a closure variable).
- **Tags → schema:** Struct tags like `path:"id"`, `query:"q"`, `header:"Authorization"`, `maxLength:"30"`, `minimum:"0"`, `pattern:"^[a-z]+$"`, `enum:"a,b,c"`, `doc:"description"`, `example:"value"` each map to both OpenAPI schema fields AND runtime validation rules. One tag line generates both docs and validation.

#### Router Integration

Huma defines an `Adapter` interface:

```go
type Adapter interface {
    Handle(op *Operation, handler func(ctx Context))
    ServeHTTP(http.ResponseWriter, *http.Request)
}
```

Any router implementing `Handle` can be wrapped. The stdlib adapter (`humago`) wraps Go 1.22+ `http.ServeMux`:

```go
func New(m Mux, config huma.Config) huma.API {
    return huma.NewAPI(config, &goAdapter{m, ""})
}
```

#### SSE Support

First-class via `github.com/danielgtaylor/huma/v2/sse`:

```go
sse.Register(api, huma.Operation{
    OperationID: "stream-events",
    Method:      http.MethodGet,
    Path:        "/events",
}, map[string]any{
    "message": MyEvent{},
}, func(ctx context.Context, input *struct{}, send sse.Sender) {
    send.Data(MyEvent{Msg: "hello"})
    send.Comment("heartbeat")
})
```

- Event types are mapped via `eventTypeMap` → JSON Schema `oneOf` in the spec
- `Sender` supports `.Data()`, `.Comment()` methods
- Flushing handled automatically (requires `http.Flusher`)
- Response Content-Type set to `text/event-stream`

#### HTML / HTMX Support

Documented at https://huma.rocks/how-to/html-response/:

```go
type MyHTMLOutput struct {
    ContentType string `header:"Content-Type"`
    Body        []byte
}

huma.Register(api, huma.Operation{
    OperationID: "get-html",
    Method:      http.MethodGet,
    Path:        "/html",
}, func(ctx context.Context, input *struct{}) (*MyHTMLOutput, error) {
    return &MyHTMLOutput{
        ContentType: "text/html",
        Body:        []byte("<html><body><h1>Hello World</h1></body></html>"),
    }, nil
})
```

Docs show integration with **Templ** and **Gomponents** for type-safe HTML templating. Also supports `huma.StreamResponse` for larger templates.

#### OpenAPI Version

Primary: **OpenAPI 3.1.0**. Also auto-generates downgraded **3.0.3** via `Downgrade()`/`DowngradeYAML()`. Served at:
- `/openapi.json` / `/openapi.yaml` (3.1.0)
- `/openapi-3.0.json` / `/openapi-3.0.yaml` (3.0.3)

#### Middleware

Three layers:
1. **Router-native** — use your router's middleware before creating the Huma API
2. **Huma's own** — `api.UseMiddleware(func(ctx huma.Context, next func(huma.Context)))`
3. **Per-operation** — `huma.Operation{ Middlewares: huma.Middlewares{...} }`

Standard `net/http` middleware can be used via `humago.Unwrap(ctx)` to extract `(r, w)`.

#### Dependencies

Core files (`huma.go`, `schema.go`, `openapi.go`, `validate.go`) depend only on stdlib. However, `go.mod` bundles all 9 router adapters in the same module, so the transitive dependency tree includes chi, gin, fiber, echo, gorilla/mux, etc. — even if you only use `humago`. The author has written about this: ["Reducing Go Dependencies"](https://dgt.hashnode.dev/reducing-go-dependencies).

Direct deps: `shorthand/v2` (patch syntax), `json-patch/v5`, `cbor/v2`, `google/uuid`, `spf13/cobra`+`pflag` (CLI), `stretchr/testify`, plus all router libraries.

#### Standalone Schema Generation (Without Handler Registration)

The reflection-based schema generation IS usable standalone:

```go
registry := huma.NewMapRegistry("#/components/schemas/", huma.DefaultSchemaNamer)
schema := huma.SchemaFromType(registry, reflect.TypeOf(MyType{}))
```

Manual OpenAPI spec construction also works:

```go
oapi := &huma.OpenAPI{
    OpenAPI: "3.1.0",
    Info:    &huma.Info{Title: "My API", Version: "1.0.0"},
}
oapi.AddOperation(&huma.Operation{...})
```

However: the automatic input/output struct → OpenAPI mapping (the killer feature) is coupled to `huma.Register()`, which requires an `API` instance (which requires an `Adapter`). The workaround for spec-only generation is to create a throwaway API, register operations, then extract the spec.

### What Huma CANNOT Do (vs a Manual Builder)

1. **Nullable objects** — Huma panics if you try `nullable:"true"` on a struct/object field
2. **Same-named types from different packages** — default registry panics on name collision (e.g., `foo.Thing` + `bar.Thing`)
3. **Complex polymorphic schemas** — `oneOf`/`anyOf`/`allOf` exist but have no declarative struct-field mapping; require `SchemaProvider` interface or manual construction
4. **Arbitrary schema composition per-field** — each field maps to exactly one schema; can't express `patternProperties` or complex inline `oneOf`
5. **Lossy OpenAPI 3.0 downgrade** — type arrays with `"null"` are simplified; `exclusiveMinimum`/`Maximum` values become booleans
6. **No WebSocket support or documentation**
7. **No compile-time/codegen** — generation is runtime reflection; can't produce the spec as a build artifact without running the server
8. **Response status code documentation is limited** — `Status int` allows dynamic codes but the spec only documents the default; multiple response schemas per status require manual `op.Responses` config

### Verdict: Is Reflection-From-Types the Better Idea?

**For a greenfield Go API framework: yes, absolutely.** Huma's approach — define a struct, get validation + docs + typed handler for free — eliminates docs drift entirely. That's why huma (4.3k stars) and fuego (1.7k stars) are the fastest-growing Go API frameworks.

**But adopting it for cqrs-htmx's `openapi/` package specifically is NOT the right move:**

1. **Fundamentally different handler model.** Huma's `Register[I, O](api, op, func(ctx, *I) (*O, error))` is a complete paradigm shift from cqrs-htmx's `app.Command("Type", DecodeJSON(mapper), ...)`. The mapper closures deliberately decouple HTTP request structs from CQRS command structs — that's the architectural boundary cqrs-htmx is built around. Reflection-from-types assumes `I` (HTTP input) IS the thing you document. In cqrs-htmx, the documented type and the dispatched command are intentionally different types connected by a closure the reflection system cannot see through.

2. **Adding reflection means adding dependencies.** Either huma's schema generator or `invopop/jsonschema`. The current builder is zero-dep stdlib only. That is a feature, not a bug — it means cqrs-htmx's OpenAPI support has the smallest possible footprint.

3. **The right integration path is a Huma adapter for cqrs-htmx**, not the reverse. A `cqrs-htmx` adapter implementing Huma's `Adapter` interface would let consumers get automatic OpenAPI generation by registering CQRS commands through Huma's typed handler system. But that is a major architectural integration project, not a small extraction — and it would only work for consumers whose commands map cleanly to HTTP types (i.e., no mapper closures).

**Bottom line:** Reflection-from-types is the better idea for building Go APIs in general. Huma is the proof. But this research strengthens the earlier verdict: cqrs-htmx's `openapi/` builder is even less worth extracting as a standalone project than initially thought — the market is moving toward the reflection-from-types model (Huma, Fuego), away from manual builders entirely. The manual builder's niche (115 importers for the closest competitor) is shrinking, not growing.

### Sources

- https://huma.rocks/ — Main documentation
- https://huma.rocks/features/ — Features overview
- https://huma.rocks/how-to/html-response/ — HTML/HTMX rendering guide
- https://github.com/danielgtaylor/huma — Repository (README, go.mod)
- `huma.go` source — `Register[I, O]`, `processInputType`, `processOutputType`, convenience functions
- `schema.go` source — `SchemaFromType`, `SchemaFromField`, reflection logic
- `api.go` source — `Adapter` interface, `NewAPI`, `Config`
- `defaults.go` source — `DefaultConfig` with OpenAPI 3.1.0
- `adapters/humago/humago.go` source — stdlib ServeMux adapter
- `sse/sse.go` source — SSE implementation with `Sender`, `Message`, `eventTypeMap`
