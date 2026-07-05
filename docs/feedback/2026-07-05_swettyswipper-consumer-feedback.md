# Consumer Feedback: cqrs-htmx

**From:** SwettySwipperWeb integration session (2026-07-05)
**Perspective:** AI agent consuming the library as a real production app
**Tone:** Honest, direct, grateful but critical where warranted

---

## What Works Superbly

### 1. Middleware Composition (`Chain`)

The `cqrshtmx.Chain()` pattern is excellent — it makes middleware ordering explicit and readable. The one-liner chain in our `buildMiddlewareChain` is 15 lines of clean, ordered middleware that any engineer can follow. This is better than most Go web frameworks.

### 2. CSRF Handling

The `CSRFMiddleware` + `CSRFResponseHeaderMiddleware` + `CSRFTokenFromContext` triad is bulletproof. The global middleware with zero exemptions pattern is exactly what we need. The token-from-context approach (not from cookies/headers) is the correct security model.

### 3. SSE Building Blocks

`NewSSEStream` + `Broadcaster` replaced 129 lines of hand-rolled SSE with clean building blocks. The heartbeat framing and proper SSE format are correct and production-safe.

### 4. Response Builder

The fluent `NewResponse(w, r).NotifySuccess("msg").Redirect("/path").Apply()` pattern is clean and composable. It correctly handles HTMX-specific headers without leaking implementation details.

### 5. Error Mapping (`MapError` + `JSONErrorHandler`)

The `MapError(err)` → `event.Classify(err).HTTPStatus()` chain is the correct design. `JSONErrorHandler` with `SafeDetail` for 5xx redaction is excellent — it prevents accidental information leakage in production while showing full detail in dev.

### 6. Rate Limiting

`NewRateLimiter` + `RateLimiterMiddleware` + `KeyExtractorFromClientIP` is clean and composable. The per-route + global pattern works well.

---

## What's Confusing or Hard to Discover

### 1. The App is Half-Used (Design Confusion)

**Problem:** We create `cqrshtmx.MustNew(Config{...})` with full config (dispatchers, enforcer, timeout, body limits) but **only use it for middleware** (`app.RecoverHandler()`, `app.Middleware()`). Zero routes go through `App.Command()` or `App.Query()`.

**Why:** The `App.Command()` pipeline assumes JSON-in → JSON-out (`DecodeJSON`, `RenderJSON`). Our app is SSR-first with HTMX — most handlers parse form data and render HTML via templ. The pipeline doesn't accommodate this well.

**Ask:** Either (a) document that the App pipeline is JSON-API-oriented and SSR apps should use it only for middleware, or (b) add SSR-friendly HandlerOptions like `DecodeForm`, `RenderTempl(component)`, `RenderHTML(htmlString)`. The latter would make the composition model usable for ALL Go web apps, not just JSON APIs.

### 2. Two UserID Types — Still Confusing

**Problem:** Root `cqrshtmx.UserID` (ULID-backed via go-cqrs-lite/id) vs `usermgmt.UserID` (string-backed via go-branded-id). They're not assignable to each other.

**Impact:** We don't use usermgmt, but even within root, the UserID type isn't clearly documented. We had to discover through trial and error that `ParseUserID` / `WithUserID` exist and how they relate to context enrichment.

**Ask:** Make the UserID story clearer in the SKILL.md. Add a "UserID Identity Flow" diagram showing how context enrichment → UserID → event metadata works end-to-end.

### 3. `HTMXScriptHandler` vs CDN — Unclear Default

**Problem:** `templ-components` `layout.Base` hardcodes CDN script tags (`https://cdn.jsdelivr.net/npm/htmx.org@...`). But `cqrs-htmx` provides embedded htmx via `HTMXScriptHandler()`. These two approaches conflict.

**Impact:** We have `https://cdn.jsdelivr.net` in our CSP `script-src` purely because `templ-components` loads htmx from CDN. If we self-host via `HTMXScriptHandler`, the layout still generates CDN `<script>` tags.

**Ask:** Document the recommended path. Should consumers (a) use `HTMXScriptHandler` + override `PageProps.HTMXCDN` to self-host, or (b) accept the CDN and keep it in CSP? Make the two libraries agree on a default.

### 4. HandlerOption Ordering — Undocumented Constraint

**Problem:** `ValidateCommand` MUST come after `DecodeJSON` because it reads the decoded value from context. Reversing the order silently skips validation.

**Impact:** This is a footgun. We discovered it from the SKILL.md (which is great!), but a runtime check or compile-time enforcement would be better.

**Ask:** Add a runtime panic or log warning if `ValidateCommand` runs before `DecodeJSON`. Or restructure so validation doesn't depend on decode ordering.

### 5. Missing `event.WithCommandCausality`

**Problem:** The go-cqrs-lite skill mentions `event.WithCommandCausality` and `event.CommandCausalityFromContext`, but these functions **don't exist** in either library. We searched both codebases.

**Impact:** Consumers following the skill will hit dead ends. The actual causality mechanism is `WithCorrelationID` / `CorrelationIDFromContext`.

**Ask:** Either implement command causality linking (command type + ID → events) or remove the reference from the skill and document correlation ID as the causality mechanism.

---

## What's Missing

### 1. Form Decoding (`DecodeForm`)

The library has `DecodeJSON` but no `DecodeForm`. For SSR apps with HTMX, form parsing is the dominant input pattern. Adding `DecodeForm(func(r *http.Request) (command.Command, error))` would make the composition model usable for form-based apps.

### 2. Templ Rendering (`RenderTempl`)

The library has `RenderJSON` but no templ rendering option. For SSR apps, `RenderTempl(func(data any) templ.Component)` would be the primary response pattern. Without it, SSR apps can't use the handler pipeline.

### 3. Pagination Helpers

`DecodePagination` and `RenderPaginatedJSON` are mentioned but we couldn't find clear documentation on how to use them. Our project hand-rolls pagination (`parsePagination`, `paginate[T]`, `newPagination`). If the library provides this, make it discoverable.

### 4. Error Response Code Field

The library's `JSONErrorHandler` writes `{"error": "...", "status": NNN}`. Our previous hand-rolled handler also included a `"code"` field for machine-readable error codes (e.g., `"battle.exists"`). The library doesn't include the error code in the JSON response.

**Impact:** API consumers can't programmatically distinguish between different rejection reasons (e.g., "battle.exists" vs "battle.min_items") without parsing the error message string.

**Ask:** Add the error code to the JSON error response when available: `{"error": "...", "code": "battle.exists", "status": 409}`.

---

## What's Over-Engineered

### 1. ProblemDetailsErrorHandler (RFC 7807)

The `ProblemDetailsErrorHandler` produces `application/problem+json` responses. This is a great standard but virtually no HTMX consumer uses it. For the vast majority of Go+HTMX apps, `JSONErrorHandler` is sufficient. Keep it, but don't push it as the default.

### 2. WebSocket ACK Protocol

The WebSocket support with ACK protocol, idempotency store, and reconnection/replay is impressive but adds significant complexity. For one-way server→client push (which is 90% of real-time web app needs), SSE is simpler and sufficient. Consider whether the WebSocket investment is paying off or whether SSE covers most use cases.

---

## Summary Scorecard

| Area                     | Rating | Notes                               |
| ------------------------ | ------ | ----------------------------------- |
| Middleware composition   | ★★★★★  | Excellent, best-in-class            |
| CSRF security            | ★★★★★  | Bulletproof                         |
| SSE building blocks      | ★★★★☆  | Great, minor docs gaps              |
| Error handling           | ★★★★☆  | Strong, missing error code field    |
| HandlerOption pipeline   | ★★★☆☆  | JSON-oriented, not SSR-friendly     |
| Documentation (SKILL.md) | ★★★★☆  | Very good, some dead references     |
| Discovery (finding APIs) | ★★★☆☆  | Some key features hard to find      |
| Form/SSR support         | ★★☆☆☆  | Missing `DecodeForm`, `RenderTempl` |

---

_This feedback is given with gratitude for an excellent library. The critique is offered to make it even better._

---

## Resolution Status (2026-07-05)

### What's Confusing or Hard to Discover — Resolutions

| #   | Suggestion                                                                               | Status             | Notes                                                                                                                                            |
| --- | ---------------------------------------------------------------------------------------- | ------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------ |
| 1   | Document that App pipeline is JSON-oriented / SSR apps should use it only for middleware | **DONE**           | SKILL.md now has a full "SSR / HTMX apps" section showing `DecodeForm` + `RenderTempl`/`RenderHTML` usage. Also added `RenderHTML` HandlerOption |
| 2   | Make the UserID story clearer in SKILL.md — add "UserID Identity Flow"                   | **PARTIALLY DONE** | UserID types documented in gotchas.md + SKILL.md gotcha #2. Full flow diagram not added                                                          |
| 3   | Document the recommended path for `HTMXScriptHandler` vs CDN (templ-components conflict) | **DONE**           | Added templ-components CDN conflict note in SKILL.md "Serving htmx.js" section                                                                   |
| 4   | HandlerOption ordering — undocumented constraint (validator after decoder)               | **DONE**           | Already documented in gotcha #3; now also has runtime `slog.Warn` when ordering is wrong (existed before, confirmed working)                     |
| 5   | Missing `event.WithCommandCausality` — dead reference                                    | **NOT DONE**       | Cross-repo issue: the dead reference is in the go-cqrs-lite skill, not cqrs-htmx. Noted in status report                                         |

### What's Missing — Resolutions

| #   | Suggestion                      | Status              | Notes                                                                                                                                                                   |
| --- | ------------------------------- | ------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | Form decoding (`DecodeForm`)    | **ALREADY EXISTED** | `DecodeForm[T]` existed before this feedback. Now properly documented in SKILL.md SSR section + core-api.md                                                             |
| 2   | Templ rendering (`RenderTempl`) | **ALREADY EXISTED** | `RenderTempl(component)` + `RenderTemplResult[T](mapper)` existed. Now documented. Also added `RenderHTML(html)` for static HTML                                        |
| 3   | Pagination helpers              | **ALREADY EXISTED** | `DecodePagination(r)` + `RenderPaginatedJSON[T]()` existed. Fixed doc bug in core-api.md (DecodeFormQuery showed wrong signature)                                       |
| 4   | Error Response Code Field       | **DONE**            | `JSONErrorHandler` now includes `"code"` field (walks cause chain for deepest domain code). Tested. **Gap:** `ProblemDetailsErrorHandler` does NOT include the code yet |

### Scorecard Update (Post-Resolution)

| Area                     | Before | After (projected) | Notes                                                                                     |
| ------------------------ | ------ | ----------------- | ----------------------------------------------------------------------------------------- |
| Error handling           | ★★★★☆  | ★★★★★             | Error `"code"` field added to JSON responses                                              |
| HandlerOption pipeline   | ★★★☆☆  | ★★★★☆             | SSR section documented; `RenderHTML` added; `DecodeForm`/`RenderTempl` now discoverable   |
| Documentation (SKILL.md) | ★★★★☆  | ★★★★★             | SSR guidance, v3/v4, discoverability section, dead reference noted                        |
| Discovery (finding APIs) | ★★★☆☆  | ★★★★☆             | Full discoverability section in SKILL.md                                                  |
| Form/SSR support         | ★★☆☆☆  | ★★★★☆             | DecodeForm/RenderTempl existed but were undiscoverable; now documented + RenderHTML added |
