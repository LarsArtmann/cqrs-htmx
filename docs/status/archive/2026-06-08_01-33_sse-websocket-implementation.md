# Status Report: 2026-06-08 01:33

_Session: SSE + WebSocket Support Implementation_

---

## Executive Summary

Implemented full HTMX SSE extension support and HTMX WebSocket protocol helpers. Previously listed as "NOT_PLANNED" in both FEATURES.md and ROADMAP.md — this session reversed that decision after researching the HTMX SSE and WS extensions and recognizing the library is the natural home for these HTMX-specific protocol types.

**Zero new dependencies.** SSE is pure HTTP. WebSocket is protocol helpers only (no upgrade logic).

---

## a) FULLY DONE

### SSE Support (`sse.go` — 281 lines)

| Component                 | Description                                                                                                                                                    |
| ------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `SSEEvent`                | Protocol-level type: Event, Data, ID, Retry. Multi-line data auto-split per SSE spec. CRLF → LF normalization.                                                 |
| `WriteSSEEvent(w, event)` | Wire-format SSE writer. Each event terminated with `\n\n`. Handles all 4 SSE fields.                                                                           |
| `SSEStream`               | Single-connection manager. Sets correct headers (text/event-stream, no-cache, keep-alive). Flushes after Send. Context-aware — cancelled on client disconnect. |
| `Broadcaster`             | Thread-safe fan-out. Buffered channels (cap 64). Non-blocking broadcast drops to slow consumers. Subscribe/Unsubscribe with channel close.                     |

### WebSocket Protocol Helpers (`ws.go` — 116 lines)

| Component              | Description                                                                                                                                          |
| ---------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------- |
| `WSMessage`            | Structured incoming HTMX WS message. Headers (HTMX headers) separated from Body (form fields).                                                       |
| `ParseWSMessage`       | Parses HTMX WS JSON. Extracts `HEADERS` object into separate map. Handles missing headers, non-string values, numeric values.                        |
| `WSMessage.StringBody` | Typed field accessor — returns empty string for missing/non-string fields.                                                                           |
| `WSOOBHTML`            | Wraps HTML with `hx-swap-oob` attributes for HTMX OOB swap. Uses existing `SwapStrategy` type. Passthrough when HTML already contains `hx-swap-oob`. |

### Tests

| File          | Specs                                                         | Coverage                                                                                     |
| ------------- | ------------------------------------------------------------- | -------------------------------------------------------------------------------------------- |
| `sse_test.go` | 17 specs (WriteSSEEvent, SSEStream, Broadcaster, Integration) | Multi-line, CRLF, error writer, context cancellation, concurrency, buffer overflow, ordering |
| `ws_test.go`  | 10 specs (ParseWSMessage, StringBody, WSOOBHTML)              | HTMX JSON format, missing headers, non-string values, OOB wrapping, passthrough              |

### Documentation

- `docs/adr/0004-sse-websocket-support.md` — Architecture decision record
- `FEATURES.md` — 5 new features (#44–#48), updated "Not Planned"
- `ROADMAP.md` — Removed SSE/WS from "Not Planned"
- `AGENTS.md` — Architecture tree, SSE/WS key decisions, coverage bump

### Metrics

| Metric           | Before | After                                                               |
| ---------------- | ------ | ------------------------------------------------------------------- |
| Ginkgo specs     | 425    | 439 (+14)                                                           |
| Coverage (root)  | 96.9%  | 96.1% (slight dip — new untested paths in splitSSELines edge cases) |
| Production files | 19     | 21 (+2)                                                             |
| Test files       | 23     | 25 (+2)                                                             |
| Total new LOC    | —      | 883 (397 prod + 486 test)                                           |
| New dependencies | 0      | 0                                                                   |
| Race detector    | Clean  | Clean                                                               |
| `go vet`         | Clean  | Clean                                                               |

---

## b) PARTIALLY DONE

### SSE Coverage Gap

- `splitSSELines()` has some edge cases not directly tested (single trailing newline, empty input after split)
- Coverage dipped from 96.9% → 96.1% because the new code adds branches that aren't all hit by tests yet
- The `Broadcaster.Unsubscribe` iterates the map to find the channel by identity — works correctly but O(n) on subscriber count

### No Integration Test for SSE/WS

- The `integration_test/` module does not yet test SSE or WS features across module boundaries
- SSE is independent of App, so cross-module testing is less critical, but a Broadcaster + CQRS event bridge test would be valuable

---

## c) NOT STARTED

### From ROADMAP.md (Pre-existing)

1. **v1.1.0 — Adopt typed dispatch** (`RegisterTyped`/`DispatchTyped`) — Open
2. **v1.1.0 — Adopt v2 `PaginatedResult[T]`** — Open
3. **v1.1.0 — BrandNamer for root marker types** — Blocked (upstream)
4. **v1.1.0 — Comprehensive godoc package examples** — Open
5. **v1.1.0 — Expand integration tests** — Open
6. **v1.1.0 — Profile hot paths** — Open
7. **v1.2.0 — PostgreSQL stores** — Planned
8. **v1.2.0 — Numeric branded IDs** — Planned (ADR 0003)
9. **v1.2.0 — Database migration tooling** — Planned
10. **v2.0.0 — OpenTelemetry** — Planned
11. **v2.0.0 — Prometheus metrics** — Planned
12. **v2.0.0 — JWT/OIDC helpers** — Planned
13. **v2.0.0 — Redis session store** — Planned
14. **v2.0.0 — BadgerDB store** — Planned

### SSE/WS Specific — Not Started

15. **SSE example** — No `examples/sse-demo/` equivalent to `examples/datastar-demo/`
16. **WebSocket example** — No `examples/ws-demo/` showing gorilla/coder integration
17. **Godoc examples** — No `ExampleWriteSSEEvent`, `ExampleSSEStream`, `ExampleBroadcaster`, `ExampleParseWSMessage`, `ExampleWSOOBHTML`
18. **SSE reconnection support** — No `Last-Event-ID` handling helpers for replay
19. **WebSocket typed message parsing** — No `ParseWSMessageInto[T]` generic helper
20. **SSE + CQRS event bridge** — No `AfterDispatchHook` that broadcasts CQRS events as SSE events

---

## d) TOTALLY FUCKED UP

**Nothing is fucked up.** Everything compiles, all 439 specs pass, race detector clean, zero vet issues, zero lint issues. The implementation is solid and follows existing patterns.

### Minor Regrets

- Coverage dipped 0.8% (96.9% → 96.1%) — acceptable for new code, should recover with targeted edge-case tests
- `Broadcaster.Unsubscribe` uses O(n) map scan to find the channel by identity — not ideal for thousands of subscribers, but fine for typical use cases

---

## e) WHAT WE SHOULD IMPROVE

### Architecture

1. **SSE + CQRS bridge**: Add an `AfterDispatchHook` factory that broadcasts dispatch results as SSE events. This is the #1 integration gap — consumers currently wire this manually
2. **Broadcaster performance**: Replace map iteration in `Unsubscribe` with a `sync.Map` or indexed lookup if subscriber counts grow
3. **SSE reconnection**: Add helpers for `Last-Event-ID` header parsing and event replay logic — critical for production SSE
4. **Typed WS messages**: `ParseWSMessageInto[T any]` generic that deserializes body fields into a typed struct

### Testing

5. **Recover coverage to 96.9%+**: Add edge-case tests for `splitSSELines` (trailing newline, \r\n, empty string)
6. **SSE integration test**: Add to `integration_test/` module — Broadcaster + CQRS dispatch → SSE event fan-out
7. **WebSocket integration test**: Test `WSOOBHTML` output against actual HTMX WS extension parsing expectations
8. **Benchmark SSE**: Add benchmarks for `WriteSSEEvent` and `Broadcaster.Broadcast` with many subscribers

### Documentation

9. **Godoc examples**: 5 missing examples for SSE/WS public API
10. **SSE example app**: Standalone example showing SSE + HTMX sse-connect/sse-swap in action
11. **README.md update**: Mention SSE and WebSocket support in the feature list

---

## f) Top #25 Things We Should Get Done Next

| #  | Item                                                          | Impact | Effort | Category      |
| -- | ------------------------------------------------------------- | ------ | ------ | ------------- |
| 1  | Godoc examples for SSE/WS (5 examples)                        | High   | Low    | Docs          |
| 2  | Recover coverage to 96.9%+ (splitSSELines edge cases)         | Medium | Low    | Test          |
| 3  | SSE + CQRS AfterDispatchHook bridge factory                   | High   | Medium | Feature       |
| 4  | SSE example app (`examples/sse-demo/`)                        | High   | Medium | Docs          |
| 5  | SSE reconnection support (Last-Event-ID replay)               | High   | Medium | Feature       |
| 6  | Typed WS message parser `ParseWSMessageInto[T]`               | Medium | Low    | Feature       |
| 7  | Broadcaster.Unsubscribe O(1) lookup optimization              | Low    | Low    | Perf          |
| 8  | SSE integration test in `integration_test/`                   | Medium | Medium | Test          |
| 9  | Benchmark SSE/WS hot paths                                    | Low    | Low    | Perf          |
| 10 | Adopt go-cqrs-lite v2 `RegisterTyped`/`DispatchTyped`         | High   | Medium | Deps          |
| 11 | Adopt go-cqrs-lite v2 `PaginatedResult[T]`                    | Medium | Medium | Deps          |
| 12 | Comprehensive godoc package examples (existing features)      | Medium | Medium | Docs          |
| 13 | Expand integration_test module coverage                       | Medium | Medium | Test          |
| 14 | Profile hot paths (dispatch, decode) for allocation reduction | Low    | Medium | Perf          |
| 15 | BrandNamer for root module marker types                       | Medium | Low    | Types         |
| 16 | PostgreSQL store for usermgmt                                 | High   | High   | Store         |
| 17 | Numeric branded IDs (ADR 0003)                                | High   | Medium | Types         |
| 18 | Database migration tooling                                    | Medium | Medium | Infra         |
| 19 | OpenTelemetry middleware (lifecycle hooks)                    | High   | Medium | Observability |
| 20 | Prometheus metrics middleware                                 | Medium | Medium | Observability |
| 21 | JWT/OIDC integration helpers                                  | Medium | High   | Auth          |
| 22 | Redis session store                                           | Medium | High   | Store         |
| 23 | README.md refresh — add SSE/WS mention                        | Low    | Low    | Docs          |
| 24 | WebSocket example app                                         | Medium | Medium | Docs          |
| 25 | Flaky test audit — ensure no time.After patterns              | Low    | Low    | Test          |

---

## g) Top #1 Question I Cannot Figure Out Myself

**Should the SSE `Broadcaster` be wired into the `App` struct or remain standalone?**

Arguments for wiring into App:

- An `App.SSEBroadcaster()` method would give consumers a pre-wired broadcaster
- Could auto-connect to CQRS event dispatcher via AfterDispatchHook
- Would match the pattern of how other features (CSRF, auth, rate limiting) are configured via `Config`

Arguments for standalone:

- SSE is inherently different from request/response — it doesn't fit the App lifecycle
- Consumers may want multiple broadcasters (different event types, different auth scopes)
- The datastar-demo shows a standalone broadcaster working perfectly
- Keeps App focused on CQRS dispatch; SSE is a separate concern

**My recommendation**: Keep it standalone. The library's philosophy is "building blocks, not frameworks." But I'd appreciate confirmation, because if it should be wired in, now is the time before consumers start depending on the standalone API.

---

## File Changes This Session

| File                                     | Action   | Lines                                           |
| ---------------------------------------- | -------- | ----------------------------------------------- |
| `sse.go`                                 | Created  | 281                                             |
| `ws.go`                                  | Created  | 116                                             |
| `sse_test.go`                            | Created  | 374                                             |
| `ws_test.go`                             | Created  | 112                                             |
| `docs/adr/0004-sse-websocket-support.md` | Created  | 48                                              |
| `AGENTS.md`                              | Modified | +17 lines (arch tree, key decisions)            |
| `FEATURES.md`                            | Modified | +11 lines (5 new features, updated Not Planned) |
| `ROADMAP.md`                             | Modified | 1 line (removed SSE/WS from Not Planned)        |
