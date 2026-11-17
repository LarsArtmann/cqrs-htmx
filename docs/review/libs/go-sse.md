# go-sse Utilization Report

> Audit date: 2026-11-14 · Method: library-deep-dive (fleet series, [index](index.html))
> Library: `github.com/larsartmann/go-sse` (+ `ssetest` v0.5.0) · Checkout: `/home/lars/projects/go-sse`

## Verdict at a glance

| Pin                      | Latest (verified 2026-11-14)       | Prod import sites | Test import sites | Adoption                          | Score                                  |
| ------------------------ | ---------------------------------- | ----------------- | ----------------- | --------------------------------- | -------------------------------------- |
| v0.7.0 (+ssetest v0.5.0) | v0.7.0 / ssetest v0.5.0 (= latest) | 19 files          | 33 files          | **Used — the realtime substrate** | **94/100 count · ~95 impact-weighted** |

## What the library is

Dependency-light SSE serving primitives: `Event`/`EventID` construction and parsing, `Stream` writer (headers, retry, heartbeats), `Broadcaster` fan-out with buffered subscribers, `EventStore`/`FilteredEventStore` replay (incl. `ReplayFiltered`), `LastEventID` reconnect handling, drop hooks (`WithOnDrop`), subscriber health (`BroadcasterHealth`), and a `ssetest` contract kit. WebSocket is deliberately out of scope — cqrs-htmx is SSE-only (ADR 0046).

## How cqrs-htmx uses it today

Symbol census: `sse.Event` ×263, `NewEventID` ×76, `NewStream` ×38, `NewBroadcaster` ×33, `EventID` ×29, `Broadcaster` ×23, `EventStore` ×19 (root's `JournalSSEStore` implements it), `Replay` ×12, `WriteEvent` ×10, `LastEventIDFromRequest` ×8, `ParseEventID` ×6, `WriteRetry`/`SetHeaders`, connected/heartbeat event names aliased in `sse_event.go`. `transport/serve.go` is the deepest consumer: fail-closed replay via `sse.ReplayFiltered` + `FilteredEventStore` (`transport/serve.go:218,366`) guarded against duplicate replay by `dedup.Ring` (go-cqrs-lite R3, 2026-11-04), and heartbeat comment frames through `stream.Heartbeat` (`transport/serve.go:498`). `ssetest` backs the reconnect + sync integration tests; SSE paths are additionally exercised live in the chromium e2e battery (root-htmx, datastar, dashboard-live, sync specs; `sync.spec.ts` scoped off firefox — headless background-tab suspension, 2026-11-08). The DataStar module's fan-out moved upstream to `go-datastar/broadcast` (deprecated aliases retained until v5).

Drop observability (resolved since 2026-11-05): root `sse_drop_stats.go` wraps `sse.WithOnDrop[sse.Event]` in a curated `DropStats` counter (`NewDropStats()`/`Option()`/`Attach`/`Total`, plus `Snapshot()` → `SSEHubSnapshot{Health sse.BroadcasterHealth, Drops int64}`). Wiring map: `setup/sse.go:26-27` constructs it into the shared hub at `NewBroadcaster`; `setup/sse_hub_stats.go` adapts it onto `adminui.SSEHubStatsSource` (live subscribers + lifetime drops on the admin operational cards, gated at `setup/setup.go:299`); `Bundle.Close` logs a warning when lifetime drops > 0 (`setup/bundle.go:251`); `dashboardui/dashboard.go:74-75` runs its own hub + counter, surfaced as `core.HubHealth{Subscribers, BufferSize, Closed, Draining, Drops}` on the dashboard overview (`dashboardui/core_bridge.go:179-188`). Four tests in `sse_drop_stats_test.go` (incl. per-subscriber counting: one broadcast to N stuck subscribers = N drops).

## Capability inventory

| #   | Capability                                                                                      | Status             | Evidence / reason                                                                                                                           |
| --- | ----------------------------------------------------------------------------------------------- | ------------------ | ------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | Event construction & IDs (`Event`, `NewEventID`, `EventID`, parse)                              | 🟢 Fully leveraged | 350+ refs; `MustParseSSEEventID` alias rides `sse.MustParseEventID`                                                                         |
| 2   | Stream serving (`NewStream`, `WriteEvent`, `SetHeaders`, `WriteRetry`)                          | 🟢 Fully leveraged | root SSE handlers + transport                                                                                                               |
| 3   | Heartbeats                                                                                      | 🟢 Fully leveraged | `startHeartbeat` → `stream.Heartbeat` (`transport/serve.go:498`)                                                                            |
| 4   | Fan-out (`NewBroadcaster`, `Broadcaster`)                                                       | 🟢 Fully leveraged | 56+ refs; lifecycle contract (Subscribe/Unsubscribe/Close/SubscriberCount) documented in the skill + guides                                 |
| 5   | Reconnect & replay (`LastEventIDFromRequest`, `Replay`, `ReplayFiltered`, `FilteredEventStore`) | 🟢 Fully leveraged | golden-pinned envelope; fail-closed replay cap + `dedup.Ring` seam (2026-11-04)                                                             |
| 6   | `ssetest` contract kit                                                                          | 🟢 Fully leveraged | `sse_reconnect_integration_test.go`, `command_sync_integration_test.go`; SSE paths also covered live in the chromium e2e battery            |
| 7   | `WithBufferSize` subscriber tuning                                                              | 🟡 Partially used  | default buffers in prod (all 5 `sse.WithBufferSize` refs are tests/docs); the datastar module keeps only a deprecated alias                 |
| 8   | `WithOnDrop` drop observability                                                                 | 🟢 Fully leveraged | `DropStats` (root API) wired in `setup/sse.go:27` + `dashboardui`; surfaced via adminui cards, dashboard overview, and Close warning        |
| 9   | `BroadcasterHealth`                                                                             | 🟢 Fully leveraged | `adminui` subscriber cards, `dashboardui` HubHealth (all 4 fields), `HubReadinessCheck` /health gate (`readiness.go:172`), `SSEHubSnapshot` |
| 10  | Keyed line framing (`KeyedLines`, `WriteKeyedLines`, `JoinLines`)                               | ⚪ N/A             | wire-format assembly helper; payloads are composed as single data fields here                                                               |

## Score derivation

10 rows − 1 N/A = **9 applicable**; 8 full ×1 + 1 partial ×0.5 = **8.5 → 94/100** (band: very high by count). Impact-weighted: the 2026-11-05 misses (rows 8–9, silent SSE drops unobserved) are now wired end-to-end — counter → admin cards → dashboard overview → Close warning — leaving only the `WithBufferSize` tuning knob unused (deliberate: defaults + observe-first posture) → **~95 effective**.

## Version currency

v0.7.0 + ssetest v0.5.0 pinned = latest (verified `git ls-remote --tags origin`, 2026-11-14; sseparse v0.2.1 also latest). The checkout's master sits 3 chore commits past the tag (CI/nix/verify.sh + sseparse test corpus only — no library `.go` drift; every cited symbol confirmed present at the `v0.7.0` tag itself).

## Should cqrs-htmx use it (more)?

Nearly maxed. Two residual nits, neither urgent: (1) root's `DropStats.Snapshot()`/`SSEHubSnapshot` convenience API is exported but has no prod consumer — adminui/dashboardui compose `Health()` + `Total()` directly, so the helper is redundant surface unless a future consumer wants the paired struct; (2) drops surface as counts/logs, not metrics — the guide's OTel recipe (`docs/guides/sse-and-datastar.md`, `sse.events.dropped` counter) is the ready upgrade path if a scraper appears.

## How to verify

```bash
rg -oN '\bsse\.[A-Z][A-Za-z0-9]*' --type go -g '!scripts/**' . | sed 's/.*://' | sort | uniq -c | sort -rn | head -16
rg -n 'WithOnDrop|BroadcasterHealth' --type go -g '!scripts/**' . | grep -v _test   # now 3 hits: the DropStats wiring itself (sse_drop_stats.go:43,61; sse_broadcaster.go:52)
rg -n 'DropStats|SSEDrops' setup/sse.go setup/sse_hub_stats.go setup/setup.go dashboardui/dashboard.go   # the wiring map
rg -n 'ReplayFiltered|stream.Heartbeat' transport/serve.go                          # :366 / :498
git -C ~/projects/go-sse ls-remote --tags --sort=-v:refname origin | grep -v '\^{}' | head -3
```

## Prior reports

- 2026-11-05 (predecessor of this file): 72/100 · ~85 impact; rows 8–9 (`WithOnDrop`, `BroadcasterHealth`) scored missed — then the fleet index's #1 Important issue ("silent SSE drops unobserved"). Resolved by the `DropStats` wiring audited above.
- Realtime context in [docs/guides/sse-and-datastar.md](../../guides/sse-and-datastar.md) and ADR 0046 (WebSocket removal).
