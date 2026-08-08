# Datastar Adapter Cleanup — Session 2 Status Report

**Date:** 2026-08-03 09:14\
**Session goal:** Execute the remaining P0/P1 items from the prior session's 50-item follow-up list\
**Result:** 9 of 10 tracked tasks completed. Module verified at **71 tests, 97.3% coverage, 0 lint issues, 0 workspace build errors**. Found 2 real bugs in my own work (documented below).

> **Update 2026-08-03 (commit `9cde5c0`, `dfc18e1`):** Both bugs found in this session (D1 writeHeartbeat SDK bypass, D2 test tautology) were fixed in session 19:34. Coverage corrected from 97.3% to 96.7% after heartbeat refactor. Integration guide updated with new features. Module is in publishable state pending user decision on tagging.

---

## A. Fully Done

### 1. Fixed workspace-wide `flightrecorder/v4.0.0` build failure (P0 — highest impact)

**Root cause:** `flightrecorder/v4` is a transitive dependency of `decider/v4` (go-cqrs-lite). Its tag (`flightrecorder/v4.0.0`) has the same zero-pseudo-version publish bug as 13 other go-cqrs-lite submodules. It was **missing** from the `go.work` replace list — only 35 of 36 required replaces were present.

**Impact:** 19 workspace-wide gopls compiler errors across `adminui`, `examples/admin-demo`, `integration_test`, `loginpage`, `usermgmt` (10+ files). The prior session's status report incorrectly scoped this as "integration_test only."

**Fix:** One line added to `go.work`:\
`replace github.com/larsartmann/go-cqrs-lite/flightrecorder/v4 => /home/lars/projects/go-cqrs-lite/flightrecorder`

**Verified:** `go build ./...` across all 19 modules — 0 errors. Full workspace `go test ./...` — all pass.

### 2. Ran integration contract tests — all 8 PASS (first execution ever)

The 8 datastar contract tests in `integration_test/datastar_contract_test.go` had **never been executed** — they were blocked by the flightrecorder bug. After the fix, all 8 pass:

```
TestDatastarScriptHandlerContract       PASS
TestDatastarReadSignalsContract          PASS
TestDatastarBroadcasterContract          PASS
TestDatastarEventBridgeContract          PASS
TestDatastarPatchConstructorsContract    PASS
TestDatastarOptionsContract              PASS
TestDatastarResponseContract             PASS
TestDatastarVersionContract              PASS
```

### 3. Added `OnError` callback to EventBridge

**Before:** `Handle()` silently swallowed errors from `PatchFunc` handlers — no logging, no metrics, no way to know a patch failed.

**After:** `OnError(func(error))` setter provides observability without changing control flow. Nil callback preserves original behavior.

```go
bridge.OnError(func(err error) {
    slog.Error("datastar event bridge error", "err", err)
})
```

**Files:** `datastar/event_bridge.go` (struct + Handle method + OnError setter), `datastar/event_bridge_test.go` (2 new tests: callback fires, nil callback doesn't panic).

### 4. Added 6 new Response methods

The `Response` builder was missing high-value SDK methods. Added:

| Method                                       | SDK method wrapped                               |
| -------------------------------------------- | ------------------------------------------------ |
| `ConsoleLog(msg, opts...)`                   | `sse.ConsoleLog`                                 |
| `ConsoleError(err, opts...)`                 | `sse.ConsoleError`                               |
| `DispatchCustomEvent(name, detail, opts...)` | `sse.DispatchCustomEvent`                        |
| `ReplaceURL(rawURL string, opts...)`         | `sse.ReplaceURL` (with url.Parse error handling) |
| `RemoveElementByID(id)`                      | `sse.RemoveElementByID`                          |
| `Prefetch(urls...)`                          | `sse.Prefetch`                                   |

**Files:** `datastar/response.go` (6 methods + `net/url` import), `datastar/response_test.go` (8 new tests including invalid-URL edge case).

### 5. Added Broadcaster heartbeat support

**Problem:** SSE connections through reverse proxies (nginx, Cloudflare) get dropped after an idle timeout (typically 60s). Without periodic traffic, clients silently disconnect.

**Solution:** `NewBroadcasterWithHeartbeat(interval)` sends SSE comment lines (`": heartbeat\n\n"`) on a ticker. Comments are ignored by the browser's EventSource but reset proxy idle timers.

**Implementation:** Uses the nil-channel idiom — when `heartbeat` is zero (default), `heartbeatC` is nil and the `case <-heartbeatC` branch in the select never fires (zero overhead). When non-zero, a `time.NewTicker` is created per-connection.

**Files:** `datastar/broadcaster.go` (heartbeat field, constructor, pumpPatches select branch, writeHeartbeat helper), `datastar/broadcaster_test.go` (2 tests: heartbeat appears, no heartbeat by default).

### 6. Migrated demo error paths + removed dead code

- **6 `http.Error()` calls** in `examples/datastar-demo/handlers_helpers.go` migrated to `ds.ErrorResponse(w, r, err)` — consistent Datastar error UX across all handlers
- **Removed `Projector.GetByID`** (9 lines) from `examples/datastar-demo/domain_store.go` — dead code after prior session's `findTodoByID` removal

**Verified:** Demo builds, lints clean, 0 remaining `http.Error()` calls.

### 7. Updated documentation

- `datastar/README.md`: Fixed `mux.HandleFunc` → `mux.Handle` idiom (Broadcaster implements `http.Handler`), expanded API table with 5 new entries (heartbeat, OnError, replay variants, ErrorResponse), added godoc pointer for full Response method surface
- `datastar/CHANGELOG.md`: Updated Response method list, added heartbeat + OnError mentions
- `CHANGELOG.md` (root): Updated datastar entry — 71 tests, 97.3% coverage, new features listed
- `AGENTS.md`: Updated datastar module description + added flightrecorder replace note

### 8. Coverage pushed from 95.3% → 97.3%

Added 14 new tests total (57 → 71):

- 2 EventBridge OnError tests
- 8 Response method tests (ConsoleLog, ConsoleError, DispatchCustomEvent, ReplaceURL, ReplaceURLInvalid, RemoveElementByID, Prefetch, + example tests)
- 2 Broadcaster heartbeat tests
- 2 broadcaster edge-case tests (invalid Last-Event-ID, no-replay live patch has no event ID)
- 1 NewSSE re-export test (was at 0% coverage)

Remaining 2.7% gap: defensive SSE-write-failure paths in `replayPatches` (75%), `pumpPatches` (87.5%), `parseLastEventID` (85.7%), `applyAll` (75%). These require injecting a broken `http.ResponseWriter` to exercise — poor effort-to-value ratio.

### 9. Formatting verified

- `gofumpt -l .` — clean
- `golines -w` applied to `response.go`, `broadcaster.go`, `event_bridge.go`, `example_test.go`
- `golangci-lint run` — 0 issues

### 10. Full workspace verification

| Check                                            | Result   |
| ------------------------------------------------ | -------- |
| `go build ./...` (all 19 modules)                | 0 errors |
| `go test ./...` (all 19 modules)                 | All pass |
| `go test ./... -run Datastar` (integration_test) | 8/8 pass |
| `golangci-lint run` (datastar)                   | 0 issues |
| `golangci-lint run` (datastar-demo)              | 0 issues |
| Demo `go build`                                  | OK       |

---

## B. Partially Done

### 1. Response method coverage — 97.3%, not 98%+

Target was ~98%+. Achieved 97.3%. Remaining gaps are in defensive error paths that require mock infrastructure to exercise. Decided the effort-to-value ratio was poor (would need a custom `http.ResponseWriter` that fails on Nth write). The 2.7% gap maps to ~6 uncovered statements across 4 functions.

### 2. Godoc example tests — added but minimal

Added `example_test.go` with `ExampleNewResponse` and `ExampleNewBroadcaster`. These are bare-bones — they demonstrate the API surface but don't show real-world usage patterns (event bus wiring, error handling, multi-patch responses). Could be richer.

---

## C. Not Started

### 1. Integration guide (`docs/guides/datastar-integration.md`) NOT updated

The guide still describes the pre-cleanup API. It does NOT mention:

- Heartbeat (`NewBroadcasterWithHeartbeat`)
- `OnError` callback on EventBridge
- New Response methods (ConsoleLog, ConsoleError, DispatchCustomEvent, ReplaceURL, RemoveElementByID, Prefetch)
- Replay buffer configuration (`NewBroadcasterWithReplay`)

**Impact:** Consumers reading the guide won't know these features exist. This is a documentation drift I introduced by adding features without updating the primary consumer-facing guide.

### 2. `.golangci.yml` audit NOT performed

P0 item #5 from the prior session's list. Never checked whether `datastar/.golangci.yml` matches project standards (same linters, same thresholds as other modules). The lint passes, but I didn't verify the config is complete/consistent.

### 3. Demo doesn't showcase new features

The demo doesn't use:

- Heartbeat (not critical — demo is local dev, no proxy)
- `OnError` on EventBridge (should be — demonstrates observability)
- New Response methods (ConsoleLog for debug, DispatchCustomEvent for custom events)

### 4. No combined Broadcaster constructor

`NewBroadcasterWithHeartbeat` creates a broadcaster with default replay buffer (256). There's no way to get both custom replay AND heartbeat without manually constructing the struct (fields are unexported). Should add `NewBroadcasterWithReplayAndHeartbeat(maxReplay, heartbeat)` or make heartbeat a functional option on existing constructors.

### 5. No `Broadcaster.ServeHTTP` SSE option pass-through

The Broadcaster calls `sdk.NewSSE(w, r)` without accepting `SSEOption` variadic args. Consumers can't configure SDK-level options (compression, custom context). This is a minor API gap.

---

## D. Totally Fucked Up

### 1. `writeHeartbeat` bypasses the SDK's write path — latent compression bug

**What I did:** `writeHeartbeat` writes raw bytes (`": heartbeat\n\n"`) directly to the `http.ResponseWriter`, then flushes via `w.(http.Flusher)`.

**Why it's wrong:** The Datastar SDK's `ServerSentEventGenerator` has:

- A `sync.Mutex` (`sse.mu`) that protects writes
- An optional compressing writer (brotli/gzip/zstd/zlib via `CAFxX/httpcompression`)
- An `http.ResponseController` for robust flushing

If compression is ever enabled on the SSE generator, my raw heartbeat bytes would be written **uncompressed** to the same stream that the SDK is writing **compressed** data to. This would corrupt the SSE stream — the browser would receive a mix of compressed and uncompressed bytes and fail to parse events.

**Why it doesn't break right now:** The Broadcaster calls `sdk.NewSSE(w, r)` without compression options, so `sse.encoding` is `""` (disabled). The SDK's compression is opt-in only.

**Why it's still fucked up:** This is a **landmine**. If someone wraps the Broadcaster's `ServeHTTP` with compression middleware, or if the SDK adds auto-compression in a future version, the heartbeat will silently corrupt every connection. The fix is to send the heartbeat through `sse.Send()` or to document that compression is incompatible with heartbeat.

**Severity:** Latent — currently safe, but fragile and undocumented.

### 2. `TestResponseReplaceURLInvalidIgnored` test asserts the wrong string

**What I did:** The test for invalid URL handling asserts:\
`require.NotContains(t, w.Body.String(), "replace-url")`

**Why it's wrong:** I verified the actual SDK output for `ReplaceURL` — it produces a `datastar-patch-elements` event containing `window.history.replaceState(...)`. The string `"replace-url"` **never appears** in valid OR invalid output. The assertion passes for both cases — it's a tautology that tests nothing.

**The correct assertion** should check for absence of `replaceState` (the actual indicator that a ReplaceURL was sent) or absence of the invalid URL string itself.

**Impact:** The invalid-URL edge case is not actually tested. The code is probably correct (it returns early on `url.Parse` error), but the test doesn't prove it.

### 3. `writeHeartbeat` uses `w.(http.Flusher)` instead of `http.NewResponseController(w)`

The SDK uses `http.NewResponseController(w)` for flushing (more robust — handles wrapped ResponseWriters). My `writeHeartbeat` uses a direct type assertion `w.(http.Flusher)`, which fails silently if the ResponseWriter is wrapped in a type that doesn't implement `http.Flusher` directly. The SDK's approach is strictly better.

---

## E. What We Should Improve

### Architecture

1. **Heartbeat should go through the SDK, not around it.** The cleanest fix is either:
   - Send an SSE event the browser ignores: `sse.Send("heartbeat", nil)` produces `event: heartbeat\n\n` — EventSource receives it, Datastar ignores unknown event types
   - Or: add a `SendComment` method to the SDK upstream (PR to datastar-go)
   - Either approach respects the SDK's mutex, compression, and flush path

2. **Broadcaster should accept `SSEOption` variadic args** on `ServeHTTP` or via a config struct. Currently there's no way to configure SDK-level behavior (compression, custom context) without subclassing.

3. **Combined constructor or options pattern for Broadcaster.** Currently: `NewBroadcaster()`, `NewBroadcasterWithReplay(n)`, `NewBroadcasterWithHeartbeat(d)` — but no way to combine replay + heartbeat. Should be `NewBroadcaster(opts ...BroadcasterOption)` with `WithReplay(n)`, `WithHeartbeat(d)` options.

4. **ReplaceURL should accept `*url.URL` or return an error**, not silently swallow parse failures. The current "silently ignore" behavior masks bugs in consumer code.

### Testing

5. **ReplaceURLInvalidIgnored test needs fixing** — change assertion from `"replace-url"` to `"replaceState"` or the actual SDK output marker.

6. **Heartbeat test should verify it doesn't corrupt the stream** — connect a real client, send a patch between heartbeats, verify the patch is received correctly. Current test only checks the heartbeat text appears in the body.

7. **Heartbeat compression interaction needs a test or explicit documentation** — verify that heartbeat + compression doesn't corrupt, or document that compression is unsupported.

8. **Test for SSE write failure paths** — the 2.7% coverage gap. A `failingResponseWriter` mock would exercise `replayPatches` and `pumpPatches` error branches.

### Documentation

9. **Integration guide must be updated** — this is the #1 consumer-facing doc and it's now stale.

10. **Heartbeat should have a doc comment explaining the compression caveat** — currently the godoc says nothing about compression incompatibility.

11. **Demo should showcase new features** — OnError on EventBridge, at minimum.

### Process

12. **I should have run lint on the demo module immediately after editing it** — I assumed it would be fine and only checked at the end. It was fine, but the assumption was wrong.

13. **I should have verified the ReplaceURL test assertion against actual SDK output before writing the test** — I guessed the output format instead of checking.

14. **I should have read the SDK's SSE source before adding `writeHeartbeat`** — the mutex, compression, and ResponseController details were all visible in the SDK source. I skipped reading it and wrote raw bytes to the ResponseWriter.

---

## F. Next Steps (50 items, prioritized)

### P0 — Fix bugs I introduced

1. **Fix `writeHeartbeat` to go through SDK** — send via `sse.Send` or refactor to use SDK's internal write path (respects mutex + compression)
2. **Fix `TestResponseReplaceURLInvalidIgnored` assertion** — change `"replace-url"` to `"replaceState"`
3. **Fix `writeHeartbeat` flush** — use `http.NewResponseController(w).Flush()` instead of `w.(http.Flusher)`
4. **Add compression incompatibility test or documentation** for heartbeat

### P0 — Documentation drift

5. **Update `docs/guides/datastar-integration.md`** with heartbeat, OnError, new Response methods, replay config
6. **Audit `datastar/.golangci.yml`** against project standards (P0 from prior session, never done)

### P1 — API improvements

7. **Refactor Broadcaster constructors to options pattern** — `NewBroadcaster(opts ...BroadcasterOption)` with `WithReplay(n)`, `WithHeartbeat(d)`
8. **Add `Broadcaster.ServeHTTP` SSEOption pass-through** — let consumers configure SDK-level options
9. **Make `ReplaceURL` return error or accept `*url.URL`** — stop silently swallowing parse failures
10. **Add `ReplaceURLQuerystring` wrapper** — SDK has it, adapter doesn't
11. **Add `ConsoleLogf` / `Redirectf` / `PatchElementf` wrappers** — SDK has printf variants, adapter doesn't
12. **Add `PatchElementTempl` to godoc examples** — currently only PatchElements is shown

### P1 — Demo improvements

13. **Wire `OnError` on demo's EventBridge** — demonstrates observability best practice
14. **Add heartbeat to demo** — even local, shows the API
15. **Add `ConsoleLog` usage in demo** — debug visibility
16. **Add `DispatchCustomEvent` usage in demo** — custom event pattern

### P1 — Testing

17. **Add `failingResponseWriter` mock** — exercise SSE write-failure paths (covers remaining 2.7%)
18. **Add heartbeat + patch interleaving test** — verify heartbeat doesn't corrupt patch delivery
19. **Add heartbeat + compression interaction test** — or document incompatibility
20. **Add concurrent Broadcast + ServeHTTP race test** — stress-test the subscriber map under load
21. **Add replay buffer overflow test** — verify eviction under rapid broadcast
22. **Add EventBridge concurrent Map/Handle test** — verify RWMutex correctness under load
23. **Add Broadcaster Close during ServeHTTP test** — verify clean shutdown mid-connection
24. **Add Response chaining test with all new methods** — verify fluent API works end-to-end
25. **Fuzz `parseLastEventID`** — verify no panic on arbitrary input
26. **Fuzz `ReadSignals`** — verify no panic on malformed JSON

### P2 — Feature additions

27. **Add `EventBridge.HandleMany(events []event.Event)`** — batch processing
28. **Add `EventBridge.Clear()`** — remove all mappings
29. **Add `Broadcaster.BroadcastFilter(fn func(Patch) bool)`** — per-subscriber filtering
30. **Add `Broadcaster.SubscriberInfo()`** — return metadata about connected clients (user agent, connect time)
31. **Add `Response.NotifySuccess(msg)` / `NotifyError(msg)`** — HTMX-style HX-Trigger equivalents for Datastar
32. **Add typed signal helpers** — `Response.PatchSignal("key", value)` instead of `map[string]any{"key": value}`
33. **Add `Broadcaster.WithMiddleware(mw)`** — wrap patch delivery (logging, metrics, rate limiting)
34. **Add SSE event type constants** — `EventTypeHeartbeat`, `EventTypeConnected` (matching cqrs-htmx root)
35. **Add `ReadSignalsTyped[T]()`** — generic signal decoding (no manual unmarshal)
36. **Add WebSocket support** — Datastar supports WS; adapter only has SSE

### P2 — SDK alignment

37. **Wrap `PatchElementGostar`** — SDK has it, adapter doesn't
38. **Wrap `PatchSignalsIfMissingRaw`** — SDK has it, adapter doesn't
39. **Wrap `PatchElementf`** — SDK has it, adapter doesn't
40. **Wrap `RemoveElementByID` / `RemoveElementf`** — SDK has both, adapter only has RemoveElementByID
41. **Wrap `Redirectf`** — SDK has it, adapter doesn't
42. **Re-export `CompressionStrategy` / `WithCompression`** — SDK SSE options

### P3 — Polish

43. **Add `datastar/doc.go` package-level godoc** — currently has none (only per-function docs)
44. **Add `Broadcaster.Metrics()` struct** — patches sent, patches dropped, subscribers peak, replay hits
45. **Add `EventBridge.Mappings()` map** — return current mappings (not just event types)
46. **Add integration test for heartbeat through real HTTP proxy** — nginx config in test fixtures
47. **Add benchmark: Broadcast to N subscribers** — verify non-blocking send performance
48. **Add benchmark: Response builder method chaining** — verify fluent API overhead
49. **Add `CONTRIBUTING.md` section for datastar module** — testing, linting, coverage commands
50. **Add `datastar/go.sum` audit** — verify no unexpected transitive deps

---

## G. Questions (3 — genuinely cannot answer)

### Q1: Should the heartbeat go through the SDK's `Send` method (safe but produces a visible SSE event), or should we PR a `SendComment` method to the upstream datastar-go SDK?

The SDK has no way to send SSE comments (lines starting with `:`). My current raw-byte approach works but bypasses the SDK's mutex/compression/flush path. Options:

- **A:** Use `sse.Send("heartbeat", nil)` — produces `event: heartbeat\n\n` (visible to EventSource, ignored by Datastar, safe under compression)
- **B:** PR `SendComment(text string)` to `starfederation/datastar-go` upstream (cleanest, but depends on external maintainer)
- **C:** Keep raw bytes, document compression incompatibility (current approach)

I cannot decide because each option has real tradeoffs (visibility vs correctness vs dependency on upstream).

### Q2: Should the Broadcaster's `ServeHTTP` accept `...SSEOption` (breaking change to the `http.Handler` interface), or should we add a separate `ServeHTTPWith(opts)` method?

The Broadcaster implements `http.Handler` (so consumers can `mux.Handle("GET /events", broadcaster)`). But the SDK's `NewSSE(w, r, opts...)` accepts options (compression, custom context). Currently there's no way to pass these through. Options:

- **A:** Add `BroadcasterConfig` struct with `SSEOptions []sdk.SSEOption`, set at construction time (no interface break)
- **B:** Add `ServeHTTPWith(w, r, opts...)` method alongside `ServeHTTP` (consumers who need options use the variant)
- **C:** Don't expose SSE options (consumers who need compression wrap the Broadcaster themselves)

I cannot decide because this affects the public API surface and there's no clear consumer demand yet.

### Q3: Should the `datastar/v4` module tag be published now, or wait until the heartbeat compression bug is fixed and the integration guide is updated?

The module is at 71 tests, 97.3% coverage, 0 lint. But the heartbeat has a latent compression-interaction bug, and the integration guide is stale. Publishing now means early adopters get the heartbeat API; waiting means a cleaner first impression. This is an irreversible decision (once tagged, the version exists forever) that depends on your release strategy and tolerance for post-tag patches.
