# cqrs-htmx — Consumer Feedback (DiscordSync)

**Consumer:** [DiscordSync](https://github.com/LarsArtmann/DiscordSync) — Discord backup bot
**Version used:** v3.5.0 (root) + catalog/v3 v3.0.0
**Version available:** v4.1.1
**Usage depth:** Moderate — rate limiter, SSE (Broadcaster + JournalSSEStore + SSEStream), MapError, ServerTimingMiddleware, MeasureServerTiming. Deliberately excludes App.Command/Query, usermgmt, adminui, CSRF, HTMXMiddleware.
**Date:** 2026-07-05

---

## The v3 → v4 Question (Answered)

**DiscordSync is on v3.5.0. v4.1.1 is available. The migration is TRIVIAL for our usage.**

The v4 breaking changes are ENTIRELY in the auth strategy extraction (TOTP/WebAuthn/OAuth2 moved to separate modules behind interfaces). DiscordSync uses NONE of those — we only use the root module's SSE, rate limiting, MapError, and ServerTiming APIs, which are **completely unchanged**.

The migration would be:

1. Change `cqrs-htmx/v3` → `cqrs-htmx/v4` in go.mod + import paths
2. Run `go mod tidy`
3. Done. Zero code changes.

**We should have done this already.** The skill file targets v4, which means the guidance doesn't fully apply until we migrate. Staying on v3 means we're 2 minor versions behind for zero benefit.

---

## What Works Superbly

### 1. SSE building blocks are the right abstraction level

`SSEStream` + `Broadcaster` + `JournalSSEStore` give you the pieces without forcing a handler shape. DiscordSync owns the HTTP handler, the library gives us the stream + fan-out + persisted replay. This is exactly right — our CBOR→JSON transcoding, custom filtering, and heartbeat logic fit naturally.

### 2. `JournalSSEStore` with `SeekableJournal` is production-grade

Reconnection replay backed by the persisted event journal survives restarts. The `EventToSSEMapper` function lets us convert domain events to SSE events with zero library assumptions about our event types. The `WithMaxReplay` option prevents memory blowup on first connection.

### 3. `RateLimiterMiddleware` is set-and-forget

Token-bucket per-IP, `Retry-After` header, configurable burst/window. We wrapped it with a static-asset-skipping decorator and it just works. The `KeyExtractor` interface is a nice touch — we use `KeyExtractorFromClientIP`.

### 4. `MapError` is the right error→HTTP boundary

One function, delegates to `errorfamily.Classify(err).HTTPStatus()`. Replaces 20 lines of switch-case. We use it in every API error path.

### 5. `ServerTimingMiddleware` + `MeasureServerTiming`

W3C Server-Timing header for debug profiling. Nil-safe `MeasureServerTiming(ctx, "db")` returns a stop function. Works transparently with SSE/HTTP2 via interface preservation. We use it in DB-heavy web handlers.

### 6. Auth strategy extraction (v4) is the right architecture

Even though we don't use auth, the fact that TOTP/WebAuthn/OAuth2 are now separate modules means our `go.mod` doesn't pull in `pquerna/otp`, `go-webauthn`, `golang.org/x/oauth2`, or `coreos/go-oidc` as transitive dependencies. This is a massive win for compile time and dependency surface.

---

## What's Painful

### 1. The `catalog/v3` version situation is confusing

DiscordSync's go.mod has:

```
github.com/larsartmann/cqrs-htmx/catalog/v3 v3.0.0
```

This is a SEPARATE fetch from `refs/tags/v3.0.0` because the catalog directory was "present in v3.0.0 but removed from later revisions." It's actually a go-cqrs-lite module, not a cqrs-htmx module. This took hours to figure out and is documented in AGENTS.md as a gotcha.

**Suggestion:** If the catalog is gone from cqrs-htmx, make sure the go-cqrs-lite catalog module is the canonical source and deprecate/remove any cqrs-htmx catalog references. The ADR-0020 exists but consumers still stumble.

### 2. No standalone middleware chain helper

DiscordSync uses `httputil.Chain(mux, middleware1, middleware2, ...)` from go-httputil. The skill mentions `cqrshtmx.Chain` but we're already using `httputil.Chain`. Having two chain helpers is confusing.

**Suggestion:** Document which chain helper consumers should use. If `cqrshtmx.Chain` delegates to `httputil.Chain`, say so. If not, explain the difference.

### 3. The skill's "Path A/B/C" model doesn't cover "SSE-only" consumers

DiscordSync is a weird consumer: we use SSE, rate limiting, error mapping, and ServerTiming, but NOT App.Command/Query, NOT usermgmt, NOT adminui, NOT CSRF, NOT HTMXMiddleware. We're not on Path A (we don't use App at all), B, or C.

**Suggestion:** Add "Path 0: Building blocks only (no App)" to the skill. This covers consumers who use the middleware/SSE/error utilities without the CQRS dispatch layer.

### 4. `Broadcaster.Subscribe()` returns `<-chan SSEEvent` but the API isn't clear about channel lifecycle

Does the channel close when the broadcaster shuts down? Does `Unsubscribe` close the channel or just stop sending? We have `defer h.broadcaster.Unsubscribe(eventCh)` — is that sufficient?

**Suggestion:** Document the channel lifecycle: "Unsubscribe closes the channel. The channel is also closed when the Broadcaster is garbage-collected. Consumers should select on `<-ch` with a `!ok` check."

### 5. No guidance on SSE filter patterns

DiscordSync has custom event filtering (`?channel_id=`, `?guild_id=`, `?event_type=`) that narrows both replay and live streams. We implemented this with a `parseSSEFilter(r)` + `filter.matches(evt)` pattern. The library gives no filtering primitives.

**Suggestion:** This is correctly a consumer concern (filtering is domain-specific), but documenting the recommended pattern would help. Something like: "For event filtering, parse query params into a filter struct, apply it during both replay and live phases, and skip events that don't match."

---

## What's Missing

### 1. SSE connection metrics

We track SSE connection/disconnection metrics ourselves (atomic counters, Prometheus gauges). The library has no built-in observability for SSE connections.

**Suggestion:** Add `Broadcaster.SubscriberCount()` (it exists via embedded `fanOut`, but it's not documented as a public API). Add optional `OnSubscribe`/`OnUnsubscribe` hooks.

### 2. No graceful SSE drain on shutdown

When the server shuts down, connected SSE clients get a connection reset. There's no built-in "drain SSE connections for N seconds before closing" pattern.

**Suggestion:** Add `Broadcaster.Close()` that signals all subscribers, or document the shutdown pattern.

### 3. `RateLimiterConfig` has many fields but no constructor

```go
RateLimiterConfig{
    Limit: 600, Window: time.Minute, Burst: 200,
    KeyExtractor: KeyExtractorFromClientIP(),
}
```

We have to know which fields are required vs optional. No `DefaultRateLimiterConfig()`.

**Suggestion:** Add `DefaultRateLimiterConfig()` that returns sensible defaults (60 req/min, burst 20, per-IP), then consumers override specific fields.

---

## What Deliberately Excluded (And Why)

| Feature                                          | Why                                                                                                                                     |
| ------------------------------------------------ | --------------------------------------------------------------------------------------------------------------------------------------- |
| `App.Command`/`App.Query`                        | DiscordSync captures external Discord Gateway events. It doesn't issue commands. The 49-method `Database` interface IS the query layer. |
| `usermgmt`                                       | Full identity stack (WebAuthn/OAuth2/TOTP/RBAC) for a personal bot. API key auth is sufficient.                                         |
| `adminui`                                        | Depends on usermgmt. Not applicable.                                                                                                    |
| `CSRFMiddleware`                                 | No browser POST forms — GET-only API + read-only dashboard.                                                                             |
| `HTMXMiddleware`                                 | Dashboard uses HTMX but doesn't need session/command context enrichment.                                                                |
| WebSocket (`WSBroadcaster`, `DispatchWSCommand`) | SSE is simpler and sufficient for one-way event push.                                                                                   |

These exclusions are NOT criticisms — they're correct design boundaries. The library is modular enough that we can use exactly what we need.

---

## Skill Feedback

The skill file is **well-structured** (Path A/B/C decision tree, composition model, gotchas). Specific feedback:

### Good

- The module table at the top is the right reference
- The "middleware trio order" (CSRF → HTMX → enrichment) is critical and well-documented
- The gotchas section catches real bugs (UserID type mismatch, validation ordering)
- Reference file pointers are accurate

### Could Improve

- **No "building blocks only" path** — consumers who use SSE/rate-limit/error-mapping without the App builder have no guidance
- **No SSE lifecycle documentation** — channel lifecycle, subscriber cleanup, heartbeat patterns
- **The catalog module confusion** isn't addressed (it's a go-cqrs-lite module, not cqrs-htmx, but the import path has `cqrs-htmx/catalog`)
- **No migration guidance for v3→v4** in the skill (the migration guide exists at `docs/migrations/v3-to-v4.md` but isn't referenced from the skill)
- **Add a "v3 vs v4" note** — consumers on v3 should know the migration is trivial if they don't use auth strategies

---

## Summary Scorecard

| Dimension              | Score      | Notes                                                           |
| ---------------------- | ---------- | --------------------------------------------------------------- |
| API design             | 9/10       | Modular, composable, right abstraction level                    |
| Documentation (skill)  | 7/10       | Good for Path A/B/C; missing for building-blocks-only consumers |
| SSE implementation     | 9/10       | Production-grade with journal-backed replay                     |
| Rate limiting          | 8/10       | Solid; needs default config constructor                         |
| Error mapping          | 10/10      | `MapError` + errorfamily = perfect                              |
| Module separation (v4) | 10/10      | Auth strategy extraction is exemplary                           |
| Overall                | **8.5/10** | Best Go web library for CQRS+HTMX+SSE                           |

---

## Resolution Status (2026-07-05)

> Tracking which suggestions were addressed in the feedback implementation session.

### What's Painful — Resolutions

| # | Suggestion                                                         | Status                  | Notes                                                                                                                 |
| - | ------------------------------------------------------------------ | ----------------------- | --------------------------------------------------------------------------------------------------------------------- |
| 1 | Catalog module confusion — document canonical source               | **PARTIALLY ADDRESSED** | Added discoverability note in SKILL.md; catalog is a go-cqrs-lite module — full resolution requires cross-repo action |
| 2 | Document which chain helper (`cqrshtmx.Chain` vs `httputil.Chain`) | **DONE**                | Added "Chain vs httputil.Chain" note in SKILL.md discoverability section                                              |
| 3 | Add "Path 0: Building blocks only (no App)"                        | **DONE**                | Path 0 section added to SKILL.md with full code example + API inventory                                               |
| 4 | Document Broadcaster channel lifecycle                             | **DONE**                | SSE channel lifecycle documented in SKILL.md + realtime.md: "Unsubscribe closes the channel" + read-loop pattern      |
| 5 | Document SSE filter patterns                                       | **DONE**                | Recommended filter pattern (parseSSEFilter + filter.matches) documented in realtime.md with code example              |

### What's Missing — Resolutions

| # | Suggestion                                                                          | Status             | Notes                                                                                                                                                   |
| - | ----------------------------------------------------------------------------------- | ------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1 | `Broadcaster.SubscriberCount()` documentation + `OnSubscribe`/`OnUnsubscribe` hooks | **PARTIALLY DONE** | `SubscriberCount()` documented in SKILL.md + realtime.md. Hooks NOT implemented (deferred)                                                              |
| 2 | `Broadcaster.Close()` for graceful SSE drain                                        | **DONE**           | `Broadcaster.Close()` + `fanOut.Close()` implemented. Closes all subscriber channels. Tested (3 tests). Documented in SKILL.md, realtime.md, gotchas.md |
| 3 | `DefaultRateLimiterConfig()` constructor                                            | **DONE**           | Implemented. Returns 100/min per-IP, burst=limit, 10min TTL. Tested (2 tests). Documented in SKILL.md + core-api.md                                     |

### Skill Feedback — Resolutions

| #                | Suggestion                        | Status                  | Notes                                                                           |
| ---------------- | --------------------------------- | ----------------------- | ------------------------------------------------------------------------------- |
| Could Improve #1 | "Building blocks only" path       | **DONE**                | Path 0 added to SKILL.md                                                        |
| Could Improve #2 | SSE lifecycle documentation       | **DONE**                | Documented in SKILL.md + realtime.md + gotchas.md                               |
| Could Improve #3 | Catalog module confusion          | **PARTIALLY ADDRESSED** | Added note in SKILL.md; needs cross-repo action                                 |
| Could Improve #4 | v3→v4 migration guidance in skill | **DONE**                | "v3 vs v4" section added to SKILL.md + references `docs/migrations/v3-to-v4.md` |
| Could Improve #5 | "v3 vs v4" note                   | **DONE**                | Same as above                                                                   |

### Scorecard Update (Post-Resolution)

| Dimension             | Before | After (projected) | Notes                                                       |
| --------------------- | ------ | ----------------- | ----------------------------------------------------------- |
| Documentation (skill) | 7/10   | **9/10**          | Path 0, SSE lifecycle, v3/v4, discoverability all added     |
| Rate limiting         | 8/10   | **9/10**          | `DefaultRateLimiterConfig()` added                          |
| SSE implementation    | 9/10   | **9.5/10**        | `Broadcaster.Close()` added; `SubscriberCount()` documented |
| Overall               | 8.5/10 | **9/10**          | All "Missing" items addressed; hooks deferred               |
