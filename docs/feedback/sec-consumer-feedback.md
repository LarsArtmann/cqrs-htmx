# cqrs-htmx — SDK Feedback from SEC

**Consumer:** [SEC](https://github.com/larsartmann/sec) — dice-based game (CQRS + HTMX)
**Date:** 2026-07-05
**Version used:** v3.5.0
**Session:** Full App builder adoption, Response builder, middleware, HealthHandler, ServerTiming

---

## What worked superbly

### 1. `Chain()` middleware composition

`cqrshtmx.Chain(mw1, mw2, mw3...)` is the correct abstraction for middleware ordering. Outer-first is documented and matches expectations. Replaced my custom `chain()` function cleanly.

### 2. `HTMXScriptHandler()` — embedded htmx.js

Serves htmx 2.0.9 from an embedded FS via `GET /htmx.js`. Eliminated the need to download and maintain a vendor `htmx.min.js` for production. One line, done. Templates reference `/htmx.js` and it just works.

### 3. `RecoveryMiddleware` + `RequestLoggingSlog`

Drop-in middleware that works with stdlib `slog`. The request logging adds structured fields (method, path, status, duration) that integrate with my `charmlog` setup. Clean, no config needed.

### 4. `IsHTMXRequest(r)`

The canonical way to detect HTMX requests. Replaced my custom header checks. Correctly checks `HX-Request: true`.

### 5. `WriteJSON(w, status, v)` — buffered encoding

Buffers before writing headers, so on encode failure nothing is committed to the wire. This is subtly important — naive `json.NewEncoder(w).Encode(v)` writes partial JSON before failing. The buffering pattern is correct.

### 6. `NewResponse(w, r).TriggerWithDetail(event, detail).Apply()`

The Response builder produces correct JSON trigger format. `TriggerWithDetail("showError", msg)` generates `{"showError":"msg"}` — exactly what my frontend listens for. Replaced 6 manual `w.Header().Set("Hx-Trigger", fmt.Sprintf(...))` sites. The fluent API is readable and the `Apply()` pattern prevents partial writes.

### 7. `CSRFMiddleware(CSRFConfig)` with nosurf

Justinas/nosurf integration with origin checks + proxy trust. More robust than the old gorilla/csrf. The `CSRFConfig` struct has all the right fields (TrustedProxies, SameSite, Secure, CookieName).

### 8. `KeyExtractorFromClientIP()` for rate limiting

Proxy-aware IP extraction (X-Forwarded-For/X-Real-IP). Critical for reverse-proxy deployments. I combine it with authenticated user ID fallback via a custom key extractor.

### 9. `SecurityHeadersMiddlewareWithConfig()`

X-Content-Type-Options, X-Frame-Options, Referrer-Policy, CSP, HSTS. The "never auto-applied" library principle is correct — I call it explicitly with my CSP config.

---

## Pain points and friction

### 1. `App.Command()` / `App.Query()` — `DecodeJSON[T]` doesn't receive the request

This is the **biggest gap**. The `DecodeJSON[T](func(T) (command.Command, error))` decoder only receives the decoded JSON body, not the `*http.Request`. SEC uses cookie-based anonymous auth (`requireIslandPlayer`) that needs the request to read cookies and check ownership. There's no way to inject request-scoped auth context into the decode callback.

**Impact:** I couldn't migrate islands JSON API endpoints to `App.Command()`. I had to keep manual `http.HandlerFunc` closures with `requireIslandPlayer(w, r, ...)`.

**Suggestion:** Add a `DecodeJSONWithRequest[T](func(r *http.Request, body T) (command.Command, error))` variant, or add a `BeforeDecode` hook that receives the request and can short-circuit with an auth error. Alternatively, document that custom auth (non-casbin, non-usermgmt) should stay as manual handlers, so consumers don't spend time trying to force-fit.

### 2. `UserIDExtractor` + `Enforcer` auth model assumes casbin/usermgmt

The `App` builder's auth model is: `UserIDExtractor` extracts user ID → `Enforcer` (casbin) checks `Authorize(resource, action)`. This covers usermgmt-based auth but not custom cookie-based ownership checks. SEC doesn't have user accounts — it generates anonymous player IDs stored in signed cookies. The `Enforcer` interface (`Allow(resource, action) bool`) doesn't map to "does this player own this game?".

**Impact:** I adopted `App.Middleware()`, `App.HealthHandler()`, and `JSONErrorHandler`, but couldn't adopt `App.Command()` / `App.Query()`.

**Suggestion:** Consider a `RequestGuard func(*http.Request, command.Command) error` option that runs after decode but before dispatch. This would let any auth model participate without forcing casbin.

### 3. `JSONErrorHandler` response format vs my frontend protocol

The SDK's `JSONErrorHandler` writes `{"error": ..., "status": ...}`. My frontend expects `{"error": ..., "code": ..., "family": ...}`. I ended up keeping my custom `writeDispatchError` because the response shape needs to include domain-specific fields.

**Impact:** The `ErrorHandler: JSONErrorHandler` config is set on the App, but my manual handlers (which are most of them) don't go through it.

**Suggestion:** Make the JSON response shape configurable. A `JSONErrorFormatter func(err) map[string]any` option would let consumers customize the shape without writing a full `ErrorHandler`.

### 4. `HealthHandler()` response format

Returns `{"status":"ok"}` or `{"status":"unhealthy","error":"..."}`. My existing handlers returned `{"status":"healthy"}`. Had to update integration tests. Not a real problem, but documenting the exact format in the skill docs would help.

### 5. v3 → v4 migration path unclear

v4 breaking changes are "exclusively in usermgmt." SEC only uses the root module. But the import path changes from `/v3` to `/v4` regardless. I stayed on v3.5.0 because the migration cost (changing all imports) has zero benefit (root module API is stable). Consider whether the major version bump is necessary for root-module-only consumers.

---

## Ideas for improvement

### 1. `App.Query` with request-scoped auth

```go
app.Query(queryType,
    cqrshtmx.DecodeJSONQueryWithRequest(func(r *http.Request) (query.Query, error) {
        // Can read cookies, headers, etc. here
        playerID := extractPlayerID(r, signer)
        return secquery.NewGetRunQuery(runIDFromPath(r), playerID)
    }),
    cqrshtmx.RenderJSON[Result](),
)
```

This would make the builder viable for cookie-auth projects.

### 2. Response builder: `NotifyError` event format flexibility

`NotifyError(msg)` fires `"showMessage"` with `{level, message}` detail. My frontend listens for `"showError"` with a plain string. I adopted `TriggerWithDetail("showError", msg)` instead, which works but bypasses the notification system. Consider making the event name configurable, or document that `TriggerWithDetail` is the escape hatch for non-standard event protocols.

### 3. Rate limiter: per-route config ergonomics

I create separate `RateLimiterMiddleware` instances for different rate limits (global vs islands API). A `RateLimitConfig` map keyed by route prefix would be more ergonomic, but the current explicit-per-middleware approach is fine.

### 4. Server-Timing: nice but hard to test

`ServerTiming func(*http.Request) bool` is a clean predicate-based opt-in. But I can't easily verify it works without a real request with the predicate matching. Consider adding a test helper or integration test example in the skill docs.

---

## Overall verdict

cqrs-htmx v3.5.0 is production-quality. The middleware suite (`Chain`, `RecoveryMiddleware`, `RequestLoggingSlog`, `CSRFMiddleware`, `SecurityHeadersMiddleware`, `RateLimiterMiddleware`) is comprehensive and correctly ordered. The `Response` builder replaces manual header formatting cleanly. `HTMXScriptHandler` eliminates a vendor dependency.

The main gap is `App.Command()` / `App.Query()` being limited to casbin/usermgmt auth. Adding request-scoped hooks (`DecodeWithRequest`, `RequestGuard`) would unlock adoption for projects with custom auth models — which is every project that doesn't use usermgmt.

The root module (middleware, Response, helpers, WriteJSON, HTMXScriptHandler, IsHTMXRequest) is excellent and I'd recommend it to any Go+HTMX project regardless of their auth model.

---

## Resolution Status (2026-07-05)

### Pain Points — Resolutions

| # | Suggestion                                                                                     | Status             | Notes                                                                                                                                                                                                                                        |
| - | ---------------------------------------------------------------------------------------------- | ------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1 | `DecodeJSON[T]` doesn't receive `*http.Request` — can't inject request-scoped auth into decode | **DONE**           | `DecodeJSONWithRequest[T](func(r *http.Request, body T) (command.Command, error))` implemented. Also added `DecodeFormWithRequest`, `DecodeJSONQueryWithRequest`, `DecodeFormQueryWithRequest`. Tested. Documented in SKILL.md + core-api.md |
| 2 | `UserIDExtractor` + `Enforcer` auth model assumes casbin/usermgmt — no `RequestGuard`          | **DONE**           | `RequestGuard(func(r *http.Request, cmdOrQuery any) error)` implemented. Runs after decode, before dispatch. Tested (2 tests: blocks dispatch on error, allows on nil). Documented in SKILL.md + core-api.md                                 |
| 3 | `JSONErrorHandler` response format vs custom frontend protocol — make shape configurable       | **PARTIALLY DONE** | Added `"code"` field to JSON error responses (the most-requested field). Full `JSONErrorFormatter` configurability deferred — noted in status report                                                                                         |
| 4 | `HealthHandler()` response format                                                              | **DOCUMENTED**     | HealthHandler format noted in SKILL.md discoverability: `{"status":"ok"}` (200) or `{"status":"unhealthy","error":"..."}` (503)                                                                                                              |
| 5 | v3 → v4 migration path unclear                                                                 | **DONE**           | "v3 vs v4" section added to SKILL.md. Root module API is unchanged; migration is import path change only for root-only consumers                                                                                                             |

### Ideas for Improvement — Resolutions

| # | Suggestion                                                          | Status         | Notes                                                                                                             |
| - | ------------------------------------------------------------------- | -------------- | ----------------------------------------------------------------------------------------------------------------- |
| 1 | `App.Query` with request-scoped auth (`DecodeJSONQueryWithRequest`) | **DONE**       | Implemented as `DecodeJSONQueryWithRequest[T]`. Same pattern as the command variant                               |
| 2 | `NotifyError` event format flexibility                              | **DOCUMENTED** | `TriggerWithDetail("showError", msg)` documented as the escape hatch for non-standard event protocols in SKILL.md |
| 3 | Rate limiter per-route config ergonomics                            | **NOT DONE**   | Current explicit-per-middleware approach confirmed as acceptable per the original feedback ("fine")               |
| 4 | Server-Timing test helper                                           | **NOT DONE**   | Deferred — noted in status report                                                                                 |

### Overall Verdict Update

**The main gap identified by SEC has been directly addressed.** Both `DecodeJSONWithRequest` and `RequestGuard` are now available, unlocking `App.Command()`/`App.Query()` for cookie-based auth, API keys, and ownership checks — every auth model that isn't Casbin. This was the single highest-impact ask across all 5 feedback files.
