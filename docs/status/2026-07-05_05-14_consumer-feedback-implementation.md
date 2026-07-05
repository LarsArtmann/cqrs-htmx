# Status Report: Consumer Feedback Implementation

**Date:** 2026-07-05 05:14
**Session scope:** Processed all 5 consumer feedback files in `docs/feedback/`, implemented code + doc improvements
**Baseline:** v4.1.1 (HEAD = `26ac81f`)

---

## a) FULLY DONE (verified: tests pass, lint clean, errorfamily clean)

| #   | Item                                                                                                                                                                                                                         | Files                                        | Tests   |
| --- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------- | ------- |
| 1   | `DefaultRateLimiterConfig()` constructor                                                                                                                                                                                     | `ratelimit_config.go`                        | 2 tests |
| 2   | `SSEEventConnected` / `SSEEventHeartbeat` constants                                                                                                                                                                          | `sse_event.go`                               | 1 test  |
| 3   | `SecurityHeaderSkip` sentinel (suppress default headers)                                                                                                                                                                     | `security.go`                                | 3 tests |
| 4   | `RenderHTML(html)` HandlerOption                                                                                                                                                                                             | `options_render.go`                          | 1 test  |
| 5   | `DecodeJSONWithRequest[T]` (request-aware mapper)                                                                                                                                                                            | `options_decode.go`, `options_types.go`      | 1 test  |
| 6   | `RequestGuard(func(r, cmd) error)` custom auth                                                                                                                                                                               | `authz.go`, `options_types.go`, `handler.go` | 2 tests |
| 7   | `Broadcaster.Close()` + `fanOut.Close()` graceful shutdown                                                                                                                                                                   | `fanout.go`                                  | 3 tests |
| 8   | Error `"code"` field in `JSONErrorHandler` (walks cause chain)                                                                                                                                                               | `errors.go`, `constants.go`                  | 1 test  |
| 9   | `CSRFTestToken(mw)` exported test helper                                                                                                                                                                                     | `csrf_testing.go`                            | 1 test  |
| 10  | SKILL.md: Path 0 (building blocks only)                                                                                                                                                                                      | SKILL.md                                     | —       |
| 11  | SKILL.md: v3 vs v4 version notes                                                                                                                                                                                             | SKILL.md                                     | —       |
| 12  | SKILL.md: SSR/HTMX guidance (DecodeForm/RenderTempl/RenderHTML)                                                                                                                                                              | SKILL.md                                     | —       |
| 13  | SKILL.md: Custom auth guidance (DecodeWithRequest + RequestGuard)                                                                                                                                                            | SKILL.md                                     | —       |
| 14  | SKILL.md: Discoverability section (CSRFResponseHeaderMiddleware, CSRFTestToken, ContextEnrichmentMiddleware(nil), IsHTMXRequest vs RenderPartial, NewSSEEventID, HealthHandler, SecurityHeaderSkip, Chain vs httputil.Chain) | SKILL.md                                     | —       |
| 15  | SKILL.md: SSE lifecycle + channel close semantics                                                                                                                                                                            | SKILL.md                                     | —       |
| 16  | core-api.md: Fixed `DecodeFormQuery` doc bug (showed wrong signature)                                                                                                                                                        | `references/core-api.md`                     | —       |
| 17  | core-api.md: All new APIs documented                                                                                                                                                                                         | `references/core-api.md`                     | —       |
| 18  | realtime.md: Broadcaster.Close(), filter patterns, event constants                                                                                                                                                           | `references/realtime.md`                     | —       |
| 19  | gotchas.md: CSRF test patterns (nosurf masking)                                                                                                                                                                              | `references/gotchas.md`                      | —       |
| 20  | gotchas.md: Broadcaster.Close() shutdown                                                                                                                                                                                     | `references/gotchas.md`                      | —       |
| 21  | AGENTS.md: Updated file descriptions                                                                                                                                                                                         | `AGENTS.md`                                  | —       |

**Verification:**

- `go test ./... -count=1 -race`: PASS (658 specs, 4.07s)
- `go build ./...`: PASS
- `go vet ./...`: PASS
- `golangci-lint run`: 0 issues
- `branching-flow errorfamily .`: 0 violations

**Stats:** 18 production files modified, 3 new files, +493/-26 lines, 13 new tests

---

## b) PARTIALLY DONE

| #   | Item                                | What's done                                                            | What's missing                                                                                                                                                                                                                         |
| --- | ----------------------------------- | ---------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | **`*WithRequest` decoder variants** | `DecodeJSONWithRequest` tested; all 4 variants implemented and compile | `DecodeFormWithRequest`, `DecodeJSONQueryWithRequest`, `DecodeFormQueryWithRequest` have **ZERO tests** — only the JSON command variant is tested                                                                                      |
| 2   | **Error "code" field**              | Added to `JSONErrorHandler`                                            | `ProblemDetailsErrorHandler` (RFC 7807) does NOT include the code field — `StructuredError` has no `Code` field. Inconsistent: JSON errors have code, problem+json errors don't                                                        |
| 3   | **CSRFTestToken helper**            | Extracts token from context/header                                     | **Does NOT return the CSRF cookie** that the POST request also needs. In a real cross-request scenario (GET → extract → POST), the consumer needs both the masked token AND the cookie. The helper is half-baked for actual test usage |
| 4   | **SSE consumer ergonomics**         | Documented filter pattern, lifecycle, Close()                          | Overview explicitly asked for a `broadcaster.ServeSSE(w, r, config)` higher-level helper to eliminate the 30-line boilerplate. Not implemented — only documented                                                                       |
| 5   | **Broadcaster observability**       | `SubscriberCount()` documented, `Close()` added                        | DiscordSync asked for `OnSubscribe`/`OnUnsubscribe` hooks for metrics. Not implemented                                                                                                                                                 |

---

## c) NOT STARTED (noted in feedback, skipped)

| #   | Item                                               | Source                 | Why skipped                                                                     |
| --- | -------------------------------------------------- | ---------------------- | ------------------------------------------------------------------------------- |
| 1   | `OnSubscribe`/`OnUnsubscribe` hooks on Broadcaster | DiscordSync Missing #1 | Prioritized Close() first; hooks are a nice-to-have                             |
| 2   | `broadcaster.ServeSSE()` high-level handler        | Overview #2            | Design decision: building blocks vs higher-level API. Needs consumer feedback   |
| 3   | `JSONErrorFormatter` configurable response shape   | SEC #3                 | Added "code" field instead; full configurability is a bigger API design         |
| 4   | "Using cqrs-htmx with Huma" recipe                 | browser-history #3     | Cross-framework recipe; belongs in examples/                                    |
| 5   | catalog module confusion resolution                | DiscordSync #1         | Cross-repo issue (go-cqrs-lite); noted in discoverability docs                  |
| 6   | `event.WithCommandCausality` dead reference fix    | SwettySwipper #5       | Cross-repo: the reference is in the go-cqrs-lite skill, not this repo           |
| 7   | Pagination helpers discoverability                 | SwettySwipper #3       | Fixed core-api.md doc bug but didn't add recipe-level guidance                  |
| 8   | Rate limiter per-route config map                  | SEC Ideas #3           | Current explicit-per-middleware approach is fine; map config is ergonomic sugar |
| 9   | Server-Timing test helper                          | SEC Ideas #4           | Noted but not implemented                                                       |
| 10  | `NotifyError` event name configurability           | SEC Ideas #2           | Documented `TriggerWithDetail` as escape hatch; not made configurable           |

---

## d) TOTALLY FUCKED UP / MAJOR GAPS

### 1. **CHANGELOG.md not updated** — CRITICAL MISS

I added **9 new exported APIs** to a library and didn't touch the changelog. This is a library — consumers track changes. The last entry is `[v4.1.1] - 2026-07-04`. These changes warrant a `[v4.2.0]` or `[Unreleased]` entry. **This is the biggest miss of the session.**

### 2. **CSRFTestToken is incomplete** — HALF-BAKED HELPER

The helper returns a token string but NOT the CSRF cookie. nosurf requires BOTH the cookie AND a valid masked token in the header. In a real test scenario:

1. GET → sets cookie + returns token
2. POST → needs cookie attached + token in header

My helper only returns the token. The consumer still needs to figure out how to get the cookie. This makes the helper much less useful than it appears. Should return `(token string, cookie *http.Cookie)` or accept a `*httptest.ResponseRecorder` to extract both.

### 3. **3 of 4 `*WithRequest` decoders are untested**

I wrote 4 decoder variants but only tested `DecodeJSONWithRequest`. `DecodeFormWithRequest`, `DecodeJSONQueryWithRequest`, `DecodeFormQueryWithRequest` have zero tests. The `decodeAndSetWithRequest` helper is shared so they probably work, but "probably" isn't tested.

### 4. **ProblemDetailsErrorHandler is now inconsistent**

`JSONErrorHandler` includes the `"code"` field. `ProblemDetailsErrorHandler` (RFC 7807) does not — `StructuredError` has no `Code` field. A consumer switching between the two handlers gets different response shapes. This is a split brain I introduced.

### 5. **Did not verify dependent modules**

I only ran root module tests. The `adminui` and `integration_test` modules depend on root. My changes (handler.go, options_types.go, authz.go) touch the dispatch pipeline. I should have run at least `integration_test` to verify no breakage. The build passed for root, but cross-module interactions are untested.

---

## e) WHAT WE SHOULD IMPROVE (process + quality)

1. **Always update CHANGELOG.md when adding exported APIs** — non-negotiable for a library
2. **Test ALL variants of a generic function, not just the first one** — I tested DecodeJSONWithRequest but assumed the other 3 work because they share a helper. That's an assumption, not a test.
3. **When adding a test helper, actually use it in a realistic end-to-end test** — CSRFTestToken should be validated with a full GET→POST round-trip
4. **Run dependent module tests when touching shared dispatch code** — handler.go and options_types.go are in the hot path for ALL modules
5. **Keep error response shapes consistent across handlers** — JSONErrorHandler and ProblemDetailsErrorHandler should have the same fields
6. **The feedback files have scoring data** (8.5/10, ★★★★☆ etc.) — I didn't aggregate or track this. A feedback summary dashboard would be valuable.
7. **Several feedback items are cross-repo** (go-cqrs-lite skill dead references, catalog module) — these need a tracking mechanism since they can't be resolved in this repo

---

## f) Next 25 Things To Do (sorted by impact/effort)

| #   | Task                                                                                            | Impact   | Effort | Category                    |
| --- | ----------------------------------------------------------------------------------------------- | -------- | ------ | --------------------------- |
| 1   | **Update CHANGELOG.md with all new APIs**                                                       | CRITICAL | 10min  | Fix gap                     |
| 2   | **Fix CSRFTestToken to return cookie + token**                                                  | HIGH     | 15min  | Fix gap                     |
| 3   | **Add tests for DecodeFormWithRequest, DecodeJSONQueryWithRequest, DecodeFormQueryWithRequest** | HIGH     | 15min  | Fix gap                     |
| 4   | **Add `Code` field to StructuredError + ProblemDetailsErrorHandler**                            | HIGH     | 20min  | Fix consistency             |
| 5   | **Run integration_test + adminui module tests**                                                 | HIGH     | 5min   | Verify                      |
| 6   | **Add `OnSubscribe`/`OnUnsubscribe` hooks to fanOut/Broadcaster**                               | MEDIUM   | 30min  | Feature                     |
| 7   | **Implement `broadcaster.ServeSSE()` high-level helper**                                        | MEDIUM   | 45min  | Feature (2 consumers asked) |
| 8   | **Add `JSONErrorFormatter` configurable response shape**                                        | MEDIUM   | 30min  | Feature                     |
| 9   | **Add race test for Broadcaster.Close() (concurrent Subscribe+Close+Broadcast)**                | MEDIUM   | 15min  | Test                        |
| 10  | **Add RequestGuard + DecodeJSONWithRequest integration test (full dispatch flow)**              | MEDIUM   | 20min  | Test                        |
| 11  | **Document HealthHandler format in core-api.md App methods table**                              | LOW      | 5min   | Doc                         |
| 12  | **Add "Using cqrs-htmx with Huma" recipe to examples/ or SKILL.md**                             | LOW      | 30min  | Doc                         |
| 13  | **Fix `event.WithCommandCausality` dead reference in go-cqrs-lite skill**                       | LOW      | 10min  | Cross-repo                  |
| 14  | **Add pagination recipe to core-api.md or SKILL.md**                                            | LOW      | 15min  | Doc                         |
| 15  | **Aggregate consumer feedback scores into a tracking doc**                                      | LOW      | 15min  | Process                     |
| 16  | **Add catalog module deprecation/canonical-source note**                                        | LOW      | 10min  | Cross-repo                  |
| 17  | **Consider `ContextEnrichmentMiddlewareAuto()` zero-arg variant**                               | LOW      | 10min  | Ergonomic                   |
| 18  | **Consider `NewSSEEventIDAuto()` explicit auto-generate constructor**                           | LOW      | 10min  | Ergonomic                   |
| 19  | **Add RequestGuard tests for query path (only command path tested)**                            | MEDIUM   | 10min  | Test                        |
| 20  | **Verify `fanOut.Close()` is safe with WSBroadcaster (same embedding)**                         | MEDIUM   | 10min  | Verify                      |
| 21  | **Add `DefaultSecurityHeadersConfig()` or document the zero-config pattern better**             | LOW      | 10min  | Ergonomic                   |
| 22  | **Consider making `NotifyError` event name configurable**                                       | LOW      | 15min  | Feature                     |
| 23  | **Add Server-Timing test helper or integration test example**                                   | LOW      | 20min  | Test/Doc                    |
| 24  | **Tag v4.2.0 release after CHANGELOG is updated**                                               | HIGH     | 5min   | Release                     |
| 25  | **Review whether ProblemDetailsErrorHandler should be recommended over JSONErrorHandler**       | LOW      | 15min  | Design                      |

---

## g) Top #1 Question I Cannot Figure Out Myself

**Should `broadcaster.ServeSSE()` (the high-level handler that eliminates the 30-line SSE boilerplate) be added to the library, or does it cross the "building blocks, not a server" design line?**

Two consumers (Overview, DiscordSync) independently wrote ~30 lines of identical SSE handler boilerplate (subscribe → defer unsubscribe → for-select with context/heartbeat). Overview explicitly asked for a higher-level helper. But the library's design philosophy is explicitly "building blocks, not a server — you own the HTTP handler." A `ServeSSE()` method on Broadcaster would be a higher-level abstraction that takes ownership of the handler shape.

The tension: adding it makes SSE setup 3 lines instead of 30 (huge DX win). Not adding it keeps the library at the right abstraction level but forces every consumer to write the same boilerplate. This is a design decision, not a technical one — I can't resolve it without the maintainer's intent.
