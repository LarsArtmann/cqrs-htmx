# Datastar Cleanup — Session 3 Status Report

> **Date:** 2026-08-03 19:34
> **Scope:** Bug fixes + documentation accuracy for the `datastar/v4` adapter module
> **Predecessor:** `docs/status/2026-08-03_09-14_datastar-cleanup-session-2-status.md`

---

## A. Executive Summary

Session 2 completed all 10 tracked todo items but ended with a self-review that found **2 real bugs** and **1 documentation gap** in the work produced. Session 3 (this session) was a focused cleanup pass to fix those exact issues, verify the fixes comprehensively, and catch additional accuracy problems discovered during re-review.

**Outcome:** Both bugs fixed, documentation updated across 6 files, all 71 tests pass, 0 lint issues, 96.7% coverage. The module is in a publishable state pending user decisions (see Section G).

---

## B. Fully Done (verified this session)

### B1. Bug Fix: `writeHeartbeat` bypassed SDK write path

| Aspect           | Detail                                                                                                                                                                                                                                                       |
| ---------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| **Root cause**   | Session 2 wrote raw bytes (`fmt.Fprint(w, ": heartbeat\n\n")`) directly to `http.ResponseWriter`, bypassing the SDK's `sync.Mutex`, compression writer (`sse.w`), and `http.ResponseController` flush path                                                   |
| **Risk**         | Would corrupt SSE streams if compression were ever enabled on the Broadcaster (currently opt-in and unused, so latent)                                                                                                                                       |
| **Fix**          | Changed `writeHeartbeat` to call `sse.Send(heartbeatEventType, nil)` where `heartbeatEventType = EventType("ping")`. This routes through the SDK's full write pipeline (mutex lock → buffer → compress if configured → write → flush via ResponseController) |
| **Tradeoff**     | Heartbeats now produce visible `event: ping` SSE events instead of invisible SSE comments (`: heartbeat`). The Datastar client ignores unknown event types, so this is safe but slightly more verbose on the wire (~20 bytes/event vs ~14 bytes/comment)     |
| **Files**        | `datastar/broadcaster.go` (const block, `pumpPatches` call site, `writeHeartbeat` function)                                                                                                                                                                  |
| **Tests**        | `TestBroadcasterHeartbeatKeepsConnectionAlive` and `TestBroadcasterNoHeartbeatByDefault` assertions updated from `": heartbeat"` to `"event: ping"` — both pass                                                                                              |
| **Verification** | 71/71 tests pass with `-race`, go vet clean, golangci-lint 0 issues                                                                                                                                                                                          |

### B2. Bug Fix: `TestResponseReplaceURLInvalidIgnored` was a tautology

| Aspect           | Detail                                                                                                                                                                                                                                                                                      |
| ---------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Root cause**   | Session 2 asserted `NotContains(body, "replace-url")` — but the SDK's `ReplaceURL` method emits `window.history.replaceState(...)` inside a `datastar-execute-script` event. The string `"replace-url"` never appears in ANY output (valid or invalid), so the assertion was vacuously true |
| **Fix**          | Changed to `NotContains(body, "replaceState")` (the actual SDK marker for ReplaceURL) + `NotContains(body, "datastar-execute-script")` (proves no event was sent at all)                                                                                                                    |
| **Files**        | `datastar/response_test.go`                                                                                                                                                                                                                                                                 |
| **Verification** | Both `TestResponseReplaceURL` (positive) and `TestResponseReplaceURLInvalidIgnored` (negative) pass                                                                                                                                                                                         |

### B3. Documentation: `writeEventID` compression caveat documented

| Aspect         | Detail                                                                                                                                                                                                                                                                          |
| -------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Root cause** | `writeEventID` writes `fmt.Fprintf(w, "id: %d\n", id)` directly to the raw `http.ResponseWriter` — the same class of bypass as the heartbeat bug. This is pre-existing (not introduced this session) and currently safe (Broadcaster doesn't use compression), but undocumented |
| **Fix**        | Added doc comment explaining the raw-write pattern, why it's safe today, and what must change if compression is added                                                                                                                                                           |
| **Files**      | `datastar/broadcaster.go` (`writeEventID` function doc)                                                                                                                                                                                                                         |

### B4. Documentation: Integration guide updated

| Aspect    | Detail                                                                                                                                                                                                                     |
| --------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Gap**   | `docs/guides/datastar-integration.md` had zero mention of heartbeat, OnError, or the 6 new Response methods added in Session 2                                                                                             |
| **Fix**   | Added 3 sections: Heartbeat (keep-alive) with code example, Response methods table (5 new methods), Error observability (OnError callback) with code example. Also clarified `NewBroadcasterWithReplay(0)` disables replay |
| **Files** | `docs/guides/datastar-integration.md`                                                                                                                                                                                      |

### B5. Config: Lint config Go version fixed

| Aspect   | Detail                                                                                                                                                    |
| -------- | --------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Gap**  | `datastar/.golangci.yml` specified `go: 1.26.4` while the root config and actual toolchain use `1.26.5`                                                   |
| **Fix**  | Updated to `go: 1.26.5`                                                                                                                                   |
| **Note** | Other submodule configs (adminui, identity-model, usermgmt, loginpage, integration_test) also have stale Go versions but are outside this session's scope |

### B6. Accuracy: Terminology consistency sweep

| Aspect    | Detail                                                                                                  |
| --------- | ------------------------------------------------------------------------------------------------------- |
| **Gap**   | After changing heartbeats from SSE comments to SSE events, all references to "comments" needed updating |
| **Fix**   | Updated: Broadcaster struct doc comment, `NewBroadcasterWithHeartbeat` doc comment, README API table    |
| **Files** | `datastar/broadcaster.go` (2 doc comments), `datastar/README.md` (1 table row)                          |

### B7. Accuracy: Coverage numbers corrected

| Aspect   | Detail                                                                                                                                                                                                    |
| -------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Gap**  | CHANGELOG and AGENTS.md claimed 97.3% coverage; actual after heartbeat refactor is 96.7% (the `writeHeartbeat` function changed from a 2-line void to a 1-line return, slightly shifting branch coverage) |
| **Fix**  | Updated to 96.7% in both `CHANGELOG.md` (root) and `AGENTS.md`                                                                                                                                            |
| **Note** | 96.7% is still well above the 90% gate                                                                                                                                                                    |

### B8. Comprehensive verification

All checks pass:

| Check             | Command                                     | Result     |
| ----------------- | ------------------------------------------- | ---------- |
| Build (datastar)  | `go build ./...`                            | Pass       |
| Tests (71, -race) | `go test ./... -count=1 -race -timeout 30s` | 71/71 PASS |
| Go vet            | `go vet ./...`                              | Clean      |
| Lint              | `golangci-lint run --timeout 5m`            | 0 issues   |
| Gofumpt           | `gofumpt -l .`                              | Clean      |
| Golines           | `golines --dry-run --max-len 120 .`         | Clean      |
| Integration tests | `integration_test: go test -run Datastar`   | 8/8 PASS   |
| Workspace build   | `go build ./...` (19 modules)               | Pass       |
| Coverage          | `go test -cover ./...`                      | 96.7%      |

---

## C. Partially Done

### C1. Submodule golangci.yml Go version drift (partially addressed)

Fixed `datastar/.golangci.yml` only. Five other submodules still have stale Go versions:

- `adminui/.golangci.yml` — `1.26.4`
- `identity-model/.golangci.yml` — `1.26.4`
- `usermgmt/.golangci.yml` — `1.26.4`
- `loginpage/.golangci.yml` — `1.26.4`
- `integration_test/.golangci.yml` — `1.26.3`

This is cosmetic — golangci-lint uses the installed Go, not the config value, for actual compilation. But it causes confusion about what version the module targets.

### C2. `writeEventID` raw-write pattern (documented but not fixed)

The same root cause as the heartbeat bug (writing to raw ResponseWriter instead of SDK's internal writer) exists in `writeEventID`. It's now documented as a known limitation with a future-fix note, but the fix itself requires either:

- An SDK method to set the event ID (doesn't exist)
- Writing the `id:` line through `sse.w` (requires access to the unexported field)

Both are blocked without SDK changes. The documentation is the correct interim solution.

---

## D. Not Started (known gaps, intentionally deferred)

### D1. SSE compression support for Broadcaster

The Broadcaster creates the SSE generator via `sdk.NewSSE(w, r)` without any `SSEOption`s — compression is never enabled. The heartbeat and event ID writes are safe in this mode. Adding `WithCompression(...)` support would require redesigning `writeEventID` to go through the SDK's writer path.

### D2. `ReplaceURL` silent error swallowing

`ReplaceURL` silently ignores `url.Parse` failures (returns `*Response`, no error). This matches the write-through pattern of all Response methods but masks consumer bugs. The doc comment says "An unparseable URL is silently ignored" — this is intentional but may surprise consumers who expect an error.

### D3. `datastar/v4` module tag

The module is not yet published/tagged. The root CHANGELOG entry exists under `[Unreleased]`.

### D4. gopls stale diagnostics

gopls/golangci-lint LS reports stale `patch.Apply undefined` and `options.go:65 undefined: http` typecheck errors that don't reflect the actual code. These are LSP cache artifacts — actual `go build`, `go test`, and `golangci-lint run` all pass clean. Restarting gopls would clear them.

---

## E. Totally Fucked Up (things I got wrong or missed)

### E1. First pass missed the "comments" → "events" terminology sweep

After changing `writeHeartbeat` from raw comment bytes to `sse.Send`, I updated the test assertions and the function itself, but **forgot to update all the doc comments and README** that still said "keep-alive comments". I only caught this during the second review pass when I re-read the files. This is the same class of error as Session 2's original bug — making a code change without fully tracing all the documentation that references the old behavior.

**Lesson:** When changing the wire format or semantics of a feature, grep for ALL references to the old behavior across docs, comments, tests, and CHANGELOGs.

### E2. Should have read the SDK source BEFORE writing the heartbeat in Session 2

The entire `writeHeartbeat` bug existed because Session 2 wrote the heartbeat without reading `sse.go`. The SDK's mutex, compression writer, and ResponseController flush were all discoverable in the source. Session 3 fixed it, but the bug should never have been introduced.

### E3. Didn't catch the coverage number drift proactively

The coverage dropped from 97.3% to 96.7% after the heartbeat refactor, but I only noticed it because I ran `go test -cover` as a routine check — not because I predicted the change would affect coverage. The CHANGELOG and AGENTS.md had stale numbers until I caught and fixed them.

### E4. The golangci.yml Go version fix was reactive, not proactive

I only noticed the Go version mismatch because I was comparing the datastar config against the root config for the audit task. I should have verified config accuracy as part of Session 2's original work.

---

## F. What We Should Improve (up to 50 next items)

### F1-F5: P0 — Immediate quality (before tagging datastar/v4)

1. **Publish `datastar/v4` tag** — module is ready (71 tests, 96.7% coverage, 0 lint issues, docs complete)
2. **Fix Go version drift in all submodule `.golangci.yml` files** — adminui, identity-model, usermgmt, loginpage, integration_test all have stale Go versions (1.26.3/1.26.4 → 1.26.5)
3. **Add `writeEventID` integration test** — verify that the `id:` line appears in SSE output when replay is enabled (currently only tested via `TestBroadcasterReplayOnReconnect` which checks `id: 2` and `id: 3` — a dedicated unit test for `writeEventID` itself would be cleaner)
4. **Add heartbeat + patch interleaving test** — verify heartbeats don't interfere with patch delivery or event IDs (current tests check heartbeat presence/absence but not interleaving correctness)
5. **Consider heartbeat event type** — `ping` is generic; verify the Datastar JS client truly ignores it (no console warning, no signal mutation). Alternative: use the SSE comment approach via a upstream SDK `SendComment` PR

### F6-F10: P1 — Module robustness

6. **Add `ReplaceURL` error variant** — consider `ReplaceURLOrError(rawURL string) (*Response, error)` for consumers who want strict validation, keeping `ReplaceURL` as the lenient version
7. **Broadcaster `ServeHTTP` with SSE options** — currently no way to pass `WithContext`, `WithCompression`, etc. to the Broadcaster's SSE generator. Consider `ServeHTTPWith` or a config struct (Q2 from Session 2)
8. **Test heartbeat under connection close** — verify that a heartbeat firing on a closed connection triggers the `IsClosed()` check and returns cleanly (currently the `writeHeartbeat` error return path is tested indirectly but not explicitly)
9. **Add `Broadcaster.Close()` idempotency test** — verify calling Close twice doesn't panic
10. **Add `EventBridge.Handle` concurrent-safety test** — verify Map/Unmap during Handle is race-free (the RWMutex should handle it, but no explicit test exists)

### F11-F15: P1 — Documentation

11. **Add ADR for heartbeat design decision** — document why SSE events (`event: ping`) were chosen over SSE comments (`: heartbeat`) and the tradeoff (SDK path safety vs wire efficiency)
12. **Add SSE compression caveat to integration guide** — the guide doesn't mention that the Broadcaster doesn't support compression and why
13. **Update `datastar/README.md` quick start** — add heartbeat and OnError to the quick start code example
14. **Add godoc examples for new Response methods** — `ExampleResponse_ConsoleLog`, `ExampleResponse_DispatchCustomEvent`, etc. (currently only `ExampleNewResponse` and `ExampleNewBroadcaster` exist)
15. **Document `writeEventID` behavior in the replay guide** — the integration guide explains replay but doesn't mention how event IDs are written

### F16-F20: P2 — Testing improvements

16. **Add `TestBroadcasterHeartbeatRespectsClosedConnection`** — explicitly test the `sse.IsClosed()` + error return path in the heartbeat branch
17. **Add `TestWriteHeartbeatErrorPropagates`** — test that a `Send` error causes `pumpPatches` to return
18. **Add fuzz test for `ReplaceURL`** — fuzz with arbitrary strings, verify no panic and no output for invalid URLs
19. **Add `TestBroadcasterConcurrentBroadcastAndClose`** — verify Broadcast during Close doesn't deadlock or panic
20. **Add `TestEventBridgeOnErrorConcurrent`** — verify OnError callback is safe to call concurrently with Map/Unmap

### F21-F25: P2 — Code quality

21. **Consider extracting `heartbeatEventType` to a exported const** — if consumers need to filter heartbeats client-side, they need to know the event type string
22. **Review `pumpPatches` cyclomatic complexity** — the function has 3 select branches with early returns; verify it stays under the cyclop threshold (12) as features are added
23. **Consider `Broadcaster.Stats()` method** — return subscriber count, total patches sent, heartbeat count for observability
24. **Add `Broadcaster.Shutdown(ctx)` method** — graceful shutdown that waits for in-flight patches to flush before closing subscribers (current `Close()` is immediate)
25. **Review `patchEntry` struct** — consider adding a timestamp for debugging replay timing

### F26-F30: P2 — SDK alignment

26. **Track datastar-go SDK updates** — the SDK is at v1.2.2; monitor for new methods that should be re-exported
27. **Consider re-exporting SDK compression options** — `WithGzip`, `WithBrotli`, etc. for when the Broadcaster gains compression support
28. **Consider `NewSSE` re-export completeness** — verify all SDK SSE options are re-exported (currently only `WithContext` is)
29. **PR upstream: `SendComment` method** — would enable SSE comment heartbeats without the overhead of a full event
30. **PR upstream: `SetEventID` method** — would enable `writeEventID` to go through the SDK writer path

### F31-F35: P3 — Ecosystem

31. **Add datastar module to coverage gate** — currently the root flake.nix coverage gate doesn't include datastar (it's a separate module with its own go.mod)
32. **Add datastar to CI workflow** — verify `.github/workflows/ci.yml` includes datastar build+test+lint
33. **Add `nix run .#test-datastar`** — per-module nix test command (if not already present)
34. **Consider datastar module in `buildflow` config** — verify `.buildflow.yml` includes datastar in its module fan-out
35. **Add datastar to the workspace `go.work` summary** — verify the module count (currently 19) is accurate in AGENTS.md

### F36-F40: P3 — Demo improvements

36. **Add heartbeat to datastar-demo** — the demo doesn't use heartbeat; adding it would demonstrate the feature
37. **Add OnError to datastar-demo** — the demo's EventBridge doesn't set OnError; adding logging would show the pattern
38. **Add ReplaceURL to datastar-demo** — demonstrate URL replacement after todo creation
39. **Add ConsoleLog debugging to datastar-demo** — show how to use ConsoleLog for client-side debugging
40. **Add DispatchCustomEvent to datastar-demo** — demonstrate cross-component communication

### F41-F45: P3 — Observability

41. **Add SSE connection metrics** — track active connections, patches/sec, heartbeat count
42. **Add replay buffer metrics** — track buffer utilization, replay count, replay miss count
43. **Add EventBridge error metrics** — track PatchFunc error rate by event type
44. **Consider OpenTelemetry integration** — trace SSE events through the Broadcaster
45. **Add structured logging for Broadcaster lifecycle** — subscriber connect/disconnect, replay, close

### F46-F50: P3 — Future features

46. **WebSocket transport** — Datastar supports WS alongside SSE; consider a WS Broadcaster variant
47. **Per-client signal filtering** — allow clients to subscribe to specific signal paths only
48. **Backpressure strategy** — currently slow clients silently miss patches (non-blocking send); consider a configurable strategy (block, drop, or disconnect slow clients)
49. **Multi-region broadcast** — consider Redis pub/sub for multi-instance Broadcaster synchronization
50. **Patch coalescing** — batch multiple rapid patches into a single SSE event to reduce wire overhead

---

## G. Questions for User (things I cannot figure out myself)

### Q1: Should the `datastar/v4` tag be published now?

The module has 71 tests, 96.7% coverage, 0 lint issues, complete docs (README + integration guide + CHANGELOG), and passes all integration tests. The two bugs from Session 2 are fixed. Is there anything else you want verified before tagging, or should I proceed with `git tag datastar/v4.x.x`?

**Context:** The module is currently importable via `go.work` replace but has no published tag. Consumers outside the workspace cannot `go get` it.

### Q2: Is the `event: ping` heartbeat format acceptable, or should I PR a `SendComment` method upstream?

The SDK's `Send` method produces a full SSE event (`event: ping\ndata: \n\n` — ~20 bytes). A `SendComment` method would produce `: heartbeat\n\n` (~14 bytes, invisible to EventSource). The current approach is safe and correct, but slightly more verbose on the wire.

**My recommendation:** Keep `event: ping` — it's safe, visible in DevTools for debugging, and the 6-byte difference is negligible at typical heartbeat intervals (15-60s). A `SendComment` PR adds upstream dependency for marginal benefit.

### Q3: Should I fix the Go version drift in the other 5 submodule `.golangci.yml` files as part of this cleanup, or leave it for a separate housekeeping pass?

The files are: `adminui` (1.26.4), `identity-model` (1.26.4), `usermgmt` (1.26.4), `loginpage` (1.26.4), `integration_test` (1.26.3). All should be `1.26.5` to match the root config and actual toolchain.

**My recommendation:** Fix them now — it's a 5-line mechanical change with zero risk, and leaving known-stale config is a documentation lie. But it's outside the datastar scope, so I'm asking.
