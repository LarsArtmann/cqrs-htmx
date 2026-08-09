# Transport Parity Status Report

> **Session**: 2026-06-18 · **Focus**: SSE/WebSocket transport parity for cqrs-htmx

---

## Executive Summary

This session delivered the **1% + 4% + 20% Pareto tiers** of transport parity: the StructuredError type, SSE error channel, SSE heartbeat, WebSocket encoder, WSBroadcaster, and all associated AfterDispatch hooks. The library went from "SSE silently swallows errors, WebSocket is receive-only" to **17 of 27 transport cells fully supported** (up from 11). All code ships with comprehensive tests (538 specs), zero lint issues, and race-detector cleanliness.

**What's not done**: README/FEATURES.md updates for new exports, and the remaining "80% for 20%" polish items.

> **UPDATE (2026-06-18)**: `WSDispatchHandler` (WS→CQRS bridge) was subsequently implemented as `DispatchWSCommand`/`DispatchWSQuery` in `ws_dispatch.go`. The "NOT STARTED" section below is preserved for historical accuracy.

---

## a) FULLY DONE ✓

| #  | Item                                          | Commit    | Details                                                                                                                                                                                                                                         |
| -- | --------------------------------------------- | --------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1  | **StructuredError type**                      | `a78ba49` | RFC 7807-shaped payload (`Type`, `Title`, `Status`, `Detail`, `Instance`). `NewStructuredError(err, r)` maps via `MapError` + extracts request ID. `NewStructuredErrorWithContext` variant. `JSON()` method for SSE/WS serialization. 11 tests. |
| 2  | **BroadcastOnError / BroadcastOnErrorFunc**   | `a78ba49` | Symmetric to `BroadcastOnSuccess`. Fires when `err != nil`. Broadcasts `StructuredError` JSON. Closes the SSE silent-error gap. 4 tests.                                                                                                        |
| 3  | **SSEStream.Heartbeat**                       | `a78ba49` | Comment-frame ping (`: keepalive\n\n`) on a ticker. Prevents Nginx/Cloudflare/ALB idle disconnects. Stops on ctx cancellation. 3 tests.                                                                                                         |
| 4  | **SSEStream.OnDisconnect**                    | `a78ba49` | Cleanup callback registration, fired on `Close()`. Multiple callbacks fire in registration order. 2 tests.                                                                                                                                      |
| 5  | **WriteWSMessage / WriteWSMessageInto[T]**    | `193f9a4` | Outbound WS message encoder. Symmetric to `ParseWSMessage`/`ParseWSMessageInto[T]`. Round-trip verified. 4 tests.                                                                                                                               |
| 6  | **WSBroadcaster**                             | `193f9a4` | Thread-safe fan-out for WS messages. Mirrors SSE `Broadcaster` API exactly. O(1) unsubscribe. Buffered channels (64). `BroadcastHTML` convenience. 7 tests.                                                                                     |
| 7  | **BroadcastOnSuccessWS / BroadcastOnErrorWS** | `193f9a4` | AfterDispatchHook factories for `WSBroadcaster`. WS equivalents of SSE hooks. 4 tests.                                                                                                                                                          |
| 8  | **Example tests (5 new)**                     | `f53e331` | `ExampleBroadcaster_BroadcastOnError`, `ExampleStructuredError`, `ExampleWSBroadcaster`, `ExampleWriteWSMessage`. All appear in godoc.                                                                                                          |
| 9  | **doc.go updated**                            | `f53e331` | SSE section now covers BroadcastOnError, Heartbeat, StructuredError. New WebSocket section documents WSBroadcaster, WriteWSMessage.                                                                                                             |
| 10 | **AGENTS.md updated**                         | `8d54d3a` | Architecture tree has 3 new files. Key Decisions SSE/WS sections expanded.                                                                                                                                                                      |
| 11 | **CHANGELOG.md updated**                      | `8d54d3a` | All 7 new exports documented under `[Unreleased] > Added`.                                                                                                                                                                                      |
| 12 | **Matrix HTML updated**                       | `c3f12d8` | 6 cells updated from missing/partial → have. Stat cards: 17 have / 0 partial / 5 missing / 5 impossible. Tech-banner: SSE→Complete, WS→Bidirectional. Danger callout → success callout.                                                         |
| 13 | **Web Communication Matrix**                  | `fc87551` | Comprehensive HTML document surveying 15 web communication technologies across 4 categorized tables, with verdict pills, summary strip, decision framework, and code-block fixes.                                                               |
| 14 | **Pareto execution plan**                     | `a78ba49` | Written to `docs/planning/2026-06-18_13-44_transport-parity-execution-plan.md` with Mermaid execution graph, 15 medium-granularity tasks, and 62 fine-granularity tasks.                                                                        |

---

## b) PARTIALLY DONE ~

| # | Item                            | Status          | What's Missing                                                                                                                                                                                                                  |
| - | ------------------------------- | --------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1 | **README.md**                   | Not started     | SSE/WS API reference tables need new rows for `BroadcastOnError`, `Heartbeat`, `OnDisconnect`, `StructuredError`, `WriteWSMessage`, `WSBroadcaster`, `BroadcastOnSuccessWS`, `BroadcastOnErrorWS`. Code examples need updating. |
| 2 | **FEATURES.md**                 | Not started     | New feature entries needed for all 7 new exports with `FULLY_FUNCTIONAL` status.                                                                                                                                                |
| 3 | **Matrix HTML roadmap section** | Partially stale | The "Target State" and "Critical Gaps" sections still show items as future work, but they're implemented. The "Current State" matrix is accurate (updated).                                                                     |

---

## c) NOT STARTED ✗

| # | Item                      | Impact | Effort  | Notes                                                                                                                                                                                                                          |
| - | ------------------------- | ------ | ------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| 1 | **WSDispatchHandler**     | High   | ~90 min | The WS→CQRS dispatch bridge. Makes WebSocket a first-class transport: parse incoming WS message → route (command/query) → dispatch → frame response (success or StructuredError). Requires design decision on message routing. |
| 2 | **WSEventStore** (replay) | Low    | ~45 min | Mirror of `SSEEventStore` for WS. App-level sequence numbers. Low priority — WS clients typically handle reconnection at the app layer.                                                                                        |
| 3 | **Integration tests**     | Medium | ~45 min | End-to-end tests for: SSE error flow (dispatch fails → BroadcastOnError → client receives), WS dispatch round-trip, SSE heartbeat keeping connection alive under load.                                                         |
| 4 | **Examples update**       | Low    | ~30 min | Update `examples/datastar-demo/` to showcase BroadcastOnError. Possibly add a WS dispatch example.                                                                                                                             |

---

## d) TOTALLY FUCKED UP 💥

**Nothing.** No regressions, no broken builds, no data loss. All tests pass (538 specs root, usermgmt pass), 0 lint issues, 95.4% coverage maintained, race detector clean.

The only "wrong" thing was the **stale matrix HTML** that showed implemented features as "missing" — caught and fixed in this session (`c3f12d8`).

---

## e) WHAT WE SHOULD IMPROVE

### Architecture / Type Model

1. **StructuredError exhaustruct lint** — The `NewStructuredError` constructor explicitly initializes all fields, but the `exhaustruct` linter still flags the zero-value returns. This is a false positive that could be silenced with a linter directive, but the real fix is to consider whether `StructuredError` should be an interface or use a builder pattern for construction. Current struct is fine for now.

2. **WSBroadcaster uses `string` not `WSMessage`** — The WSBroadcaster broadcasts `string` messages (HTML/JSON), not typed `WSMessage` structs. This is intentional (consumers write via their WS library's API), but means the type safety of `WSMessage` is lost at the broadcast boundary. A future `TypedWSBroadcaster[T]` could solve this.

3. **No shared Broadcaster interface** — SSE `Broadcaster` and `WSBroadcaster` have identical APIs but no shared interface. A `FanOut[T]` interface could unify them, but the channel types differ (`SSEEvent` vs `string`), making this a generic interface candidate.

4. **BroadcastOnError signature inconsistency** — `BroadcastOnError(eventName, data string)` takes a `data` parameter that's currently unused (the data comes from `NewStructuredError`). The `data` param exists for API consistency with `BroadcastOnSuccess` but is misleading. Should either use it or remove it.

### Process / Documentation

5. **README is the biggest gap** — The README is the consumer-facing document and hasn't been updated. Anyone reading `pkg.go.dev` or GitHub sees the old SSE/WS story.

6. **No ADR for transport parity** — The architectural decision to add `StructuredError`, `BroadcastOnError`, `WSBroadcaster` should be documented in an ADR (`docs/adr/0010-transport-parity.md`).

7. **LSP false positives persist** — `errchkjson`, `exhaustruct`, and `typecheck` warnings in LSP are stale/false. `golangci-lint run` (the authoritative check) reports 0 issues. Consider restarting LSP or documenting that CLI is authoritative.

### Testing

8. **No benchmark for WSBroadcaster** — SSE Broadcaster has `BenchmarkBroadcasterBroadcastStress` and `BenchmarkBroadcasterConcurrentSubscribe`. WSBroadcaster should have the same for parity.

9. **Heartbeat test uses `time.Sleep`** — The heartbeat test sleeps 50ms to observe ping frames. Could use a more deterministic approach (channel-based confirmation), though the current approach is acceptable.

---

## f) Top 25 Things to Do Next

Sorted by **impact / effort ratio** (highest first).

| #  | Task                                                                      | Impact   | Effort | Category     |
| -- | ------------------------------------------------------------------------- | -------- | ------ | ------------ |
| 1  | **WSDispatchHandler** — WS→CQRS dispatch bridge                           | Critical | 90 min | Feature      |
| 2  | **Update README.md** SSE/WS sections with new exports                     | High     | 45 min | Docs         |
| 3  | **Update FEATURES.md** with new feature rows                              | High     | 30 min | Docs         |
| 4  | **WSDispatchHandler tests** — command/query routing, error framing        | Critical | 45 min | Test         |
| 5  | **ADR 0010: Transport Parity** — document the architectural decisions     | Medium   | 30 min | Docs         |
| 6  | **Fix BroadcastOnError unused `data` param** — remove or use it           | Medium   | 10 min | Code quality |
| 7  | **Integration test: SSE error flow** (dispatch fails → error event)       | Medium   | 30 min | Test         |
| 8  | **Integration test: WS dispatch round-trip**                              | Medium   | 30 min | Test         |
| 9  | **Integration test: SSE heartbeat** keeps connection alive                | Low      | 20 min | Test         |
| 10 | **WSBroadcaster benchmarks** — mirror SSE Broadcaster benchmarks          | Low      | 20 min | Test         |
| 11 | **Update matrix HTML roadmap section** — mark items as done               | Low      | 15 min | Docs         |
| 12 | **Update matrix HTML critical gaps** — mark items as resolved             | Low      | 10 min | Docs         |
| 13 | **TypedWSBroadcaster[T]** — type-safe WS fan-out                          | Low      | 45 min | Feature      |
| 14 | **Shared FanOut[T] interface** — unify SSE/WS broadcasters                | Low      | 30 min | Architecture |
| 15 | **WSEventStore** — replay for WebSocket                                   | Low      | 45 min | Feature      |
| 16 | **Add BroadcastOnError to datastar-demo example**                         | Low      | 20 min | Example      |
| 16 | **Catalog integration** — register new SSE/WS exports in catalog builder  | Low      | 30 min | Feature      |
| 17 | **SSE Go benchmarks** — measure Heartbeat overhead                        | Low      | 15 min | Perf         |
| 18 | **WS dispatch example** in examples/ directory                            | Low      | 30 min | Example      |
| 19 | **Consider `io.WriterTo` interface** for WSMessage — streaming encode     | Low      | 20 min | Architecture |
| 20 | **Consider `fmt.Stringer` for StructuredError** — default to JSON         | Low      | 5 min  | Code quality |
| 21 | **Consider `errors.Unwrap` support** in StructuredError — chain traversal | Low      | 15 min | Code quality |
| 22 | **Document WS backpressure policy** — when buffer is full                 | Low      | 10 min | Docs         |
| 23 | **Fuzz test for WriteWSMessage** — malformed input handling               | Low      | 15 min | Test         |
| 24 | **Fuzz test for StructuredError.JSON()** — encoding edge cases            | Low      | 10 min | Test         |
| 25 | **Consider SSE extension for WS** — `hx-ext="ws"` documentation           | Low      | 15 min | Docs         |

---

## Metrics Summary

| Metric                 | Value                                                                    |
| ---------------------- | ------------------------------------------------------------------------ |
| Test specs (root)      | 538                                                                      |
| Test specs (usermgmt)  | passing                                                                  |
| Lint issues            | 0                                                                        |
| Root coverage          | 95.4%                                                                    |
| Race detector          | Clean                                                                    |
| New `.go` files        | 4 (`structured_error.go`, `ws_encoder.go`, `ws_broadcaster.go`, + tests) |
| New lines of code      | ~1,075                                                                   |
| Commits this session   | 6                                                                        |
| Matrix cells "have"    | 17/27 (63%)                                                              |
| Matrix cells "missing" | 5/27 (19%)                                                               |
| Pre-commit checks      | 37/37 passing                                                            |

---

## g) Top #1 Question

**How should `WSDispatchHandler` route commands vs queries?**

The existing `App.Command(type, opts...)` and `App.Query(type, opts...)` create HTTP handlers that decode the body, run auth, and dispatch. A WS dispatch bridge needs to do the same but from a WS message instead of an HTTP request.

Two options:

1. **Explicit type field** — The WS message JSON includes a `"_type": "command"` or `"_type": "query"` field. The handler reads this and routes accordingly. Simple, but requires clients to include the field.

2. **Separate handler functions** — `WSDispatchHandler(app, cmdType, qryType)` returns a function that tries command dispatch first, falls back to query. Or: two separate functions `WSDispatchCommand(app, type)` and `WSDispatchQuery(app, type)`.

Option 2 is more explicit and matches the HTTP handler pattern (`app.Command()` / `app.Query()`). But it means the consumer needs to know which type each message is before calling the handler, which may not fit the single-connection WS model.

**My recommendation**: Option 2 — `WSDispatchCommand(app, type, opts...)` and `WSDispatchQuery(app, type, opts...)` as separate exports, matching the HTTP handler API exactly. The consumer routes from their WS library's `OnMessage` callback based on message shape.

---

_Generated 2026-06-18 14:26_
