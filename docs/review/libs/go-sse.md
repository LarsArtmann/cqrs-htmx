# go-sse Utilization Report

> Audit date: 2026-11-22 (third refresh; prior: 2026-11-14, 2026-11-05) · Method: library-deep-dive (fleet series, [index](index.html))
> Library: `github.com/larsartmann/go-sse` (+ `ssetest` v0.5.0 pinned; `sseparse` v0.2.1 rides indirect) · Checkout: `/home/lars/projects/go-sse`

## Verdict at a glance

| Pin                      | Latest (verified 2026-11-22)                        | Prod import sites | Test import sites | Adoption                          | Score                                  |
| ------------------------ | --------------------------------------------------- | ----------------- | ----------------- | --------------------------------- | -------------------------------------- |
| v0.7.0 (+ssetest v0.5.0) | v0.7.0 / ssetest v0.5.0 / sseparse v0.2.1 (= latest) | 18 files          | 34 files          | **Used — the realtime substrate** | **94/100 count · ~95 impact-weighted** |

## What the library is

Dependency-light SSE serving primitives: `Event`/`EventID` construction and parsing, `Stream` writer (headers, retry, heartbeats), `Broadcaster` fan-out with buffered subscribers, `EventStore`/`FilteredEventStore` replay (incl. `ReplayFiltered`), `LastEventID` reconnect handling, drop hooks (`WithOnDrop`), subscriber health (`BroadcasterHealth`), and a `ssetest` contract kit; `sseparse` (zero-dep parser + WPT-style corpus) rides along as a sibling module. WebSocket is deliberately out of scope — cqrs-htmx is SSE-only (ADR 0046).

## How cqrs-htmx uses it today

Symbol census (2026-11-22): `sse.Event` ×263, `NewEventID` ×77, `NewStream` ×38, `NewBroadcaster` ×33, `EventID` ×29, `Broadcaster` ×23, `EventStore` ×19 (root's `JournalSSEStore` implements it), `Replay` ×12, `WriteEvent` ×10, `LastEventIDFromRequest` ×8, `ParseEventID` ×6, `WriteRetry` ×2; connected/heartbeat event names aliased in `sse_event.go` (`:19`,`:25`) with `MustParseSSEEventID` at `:59`. `transport/serve.go` is the deepest consumer: fail-closed replay via `sse.ReplayFiltered` + `FilteredEventStore` (`transport/serve.go:218` doc, call at `:366`) guarded against duplicate replay by `dedup.Ring` (`:186`,`:193`), and heartbeat comment frames through `stream.Heartbeat` (`transport/serve.go:498`). `ssetest` backs the reconnect + sync integration tests (`sse_reconnect_integration_test.go`, `command_sync_integration_test.go`); SSE paths are additionally exercised live in the chromium e2e battery (root-htmx, datastar, dashboard-live, sync specs at `e2e/tests/`; `sync.spec.ts` scoped off firefox — headless background-tab suspension, 2026-11-08). The DataStar module's fan-out moved upstream to `go-datastar/broadcast` (deprecated alias retained at `datastar/broadcaster.go:16` until v5).

Drop observability (resolved since 2026-11-05): root `sse_drop_stats.go` wraps `sse.WithOnDrop[sse.Event]` in a curated `DropStats` counter (`NewDropStats()`/`Option()`/`Attach`/`Total`, plus `Snapshot()` → `SSEHubSnapshot{Health sse.BroadcasterHealth, Drops int64}`). Wiring map: `setup/sse.go:26-27` constructs it into the shared hub at `NewBroadcaster`; `setup/sse_hub_stats.go` adapts it onto `adminui.SSEHubStatsSource` (live subscribers + lifetime drops on the admin operational cards, gated at `setup/setup.go:299-301`); `Bundle.Close` logs a warning when lifetime drops > 0 (`setup/bundle.go:251`); `dashboardui/dashboard.go:74-75` runs its own hub + counter, surfaced as `core.HubHealth{Subscribers, BufferSize, Closed, Draining, Drops}` on the dashboard overview (`dashboardui/core_bridge.go:180-188`); `HubReadinessCheck` gates `/health` (`readiness.go:170`). Four tests in `sse_drop_stats_test.go` (incl. per-subscriber counting: one broadcast to N stuck subscribers = N drops).

## Capability inventory

| #   | Capability                                                                                      | Status             | Evidence / reason                                                                                                                           |
| --- | ----------------------------------------------------------------------------------------------- | ------------------ | ------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | Event construction & IDs (`Event`, `NewEventID`, `EventID`, parse)                              | 🟢 Fully leveraged | 350+ refs; `MustParseSSEEventID` alias rides `sse.MustParseEventID` (sse_event.go:59)                                                       |
| 2   | Stream serving (`NewStream`, `WriteEvent`, `SetHeaders`, `WriteRetry`)                          | 🟢 Fully leveraged | root SSE handlers + transport                                                                                                               |
| 3   | Heartbeats                                                                                      | 🟢 Fully leveraged | `startHeartbeat` → `stream.Heartbeat` (`transport/serve.go:498`)                                                                            |
| 4   | Fan-out (`NewBroadcaster`, `Broadcaster`)                                                       | 🟢 Fully leveraged | 56+ refs; lifecycle contract (Subscribe/Unsubscribe/Close/SubscriberCount) documented in the skill + guides                                 |
| 5   | Reconnect & replay (`LastEventIDFromRequest`, `Replay`, `ReplayFiltered`, `FilteredEventStore`) | 🟢 Fully leveraged | golden-pinned envelope; fail-closed replay cap + `dedup.Ring` seam (2026-11-04)                                                             |
| 6   | `ssetest` contract kit                                                                          | 🟢 Fully leveraged | `sse_reconnect_integration_test.go`, `command_sync_integration_test.go`; SSE paths also covered live in the chromium e2e battery            |
| 7   | `WithBufferSize` subscriber tuning (incl. `DefaultSubscriberBuffer`)                             | 🟡 Partially used  | default buffers in prod (all 5 `sse.WithBufferSize` refs are tests/docs); **new since 11-14 in the evidence, not in adoption**: v0.7.0 exported `DefaultSubscriberBuffer` = 64 (at the tag fanout.go:19, resolves upstream #6) so hubs can size by construction — still 0 prod refs here (observe-first posture unchanged) |
| 8   | `WithOnDrop` drop observability                                                                 | 🟢 Fully leveraged | `DropStats` (root API) wired in `setup/sse.go:27` + `dashboardui`; surfaced via adminui cards, dashboard overview, and Close warning        |
| 9   | `BroadcasterHealth`                                                                             | 🟢 Fully leveraged | `adminui` subscriber cards, `dashboardui` HubHealth (all 4 fields), `HubReadinessCheck` /health gate (`readiness.go:170`), `SSEHubSnapshot` |
| 10  | Keyed line framing (`KeyedLines`, `WriteKeyedLines`, `JoinLines`)                               | ⚪ N/A             | wire-format assembly helper (event.go:351 at the tag); payloads are composed as single data fields here                                     |

## Score derivation

10 rows − 1 N/A = **9 applicable**; 8 full ×1 + 1 partial ×0.5 = **8.5 → 8.5/9 = 94/100** (band: very high by count). Impact-weighted: the 2026-11-05 misses (rows 8–9, silent SSE drops unobserved) are wired end-to-end — counter → admin cards → dashboard overview → Close warning — leaving only the `WithBufferSize`/`DefaultSubscriberBuffer` tuning knob unused (deliberate: defaults + observe-first posture) → **~95 effective**. No row flipped since 11-14; the score and the inventory are unchanged.

## Version currency

v0.7.0 + ssetest v0.5.0 + sseparse v0.2.1 pinned = latest (verified `git ls-remote --tags origin`, 2026-11-22; sseparse is not imported directly — 0 import sites, indirect pin `go.mod:49` via ssetest). Checkout HEAD 8acf04c (2026-11-21) now sits well past the tag — the 11-14 file's "3 chore commits" note is superseded: the 2026-11-20 red-master repair (`a3808ed`) plus the buildflow 350-line file-size split landed since. The non-test `.go` delta past v0.7.0 is layout-only with zero public-API change: root `event.go`/`fanout.go` split at symbol seams into `fanout_lifecycle.go` + `lines.go` (`DefaultSubscriberBuffer` was fanout.go:19 at the tag, fanout.go:14 at HEAD — same declaration), example `*_templ.go` regens (templ v0.3.1100), and `sseparse/` gained the unreleased json/v2 corpus migration (2026-11-21). One upstream bookkeeping gap worth knowing: the CHANGELOG has no [0.7.0] section — everything since [0.6.2] (including `DefaultSubscriberBuffer`, the ssetest `WithLastEventID` → `sse.EventID` breaking change, SECURITY.md practices, the coverage ratchet) sits under [Unreleased]; v0.7.0 content was verified AT the tag (`git grep … v0.7.0`), not trusted to the section headers.

## Should cqrs-htmx use it (more)?

Nearly maxed. Three residual nits, none urgent: (1) root's `DropStats.Snapshot()`/`SSEHubSnapshot` convenience API is exported but has no prod consumer — adminui/dashboardui compose `Health()` + `Total()` directly (only `sse_drop_stats.go` itself references the type), so the helper is redundant surface unless a future consumer wants the paired struct; (2) drops surface as counts/logs, not metrics — the guide's OTel recipe (`docs/guides/sse-and-datastar.md`, `sse.events.dropped` counter) is the ready upgrade path if a scraper appears; (3) if a hub ever needs non-default buffer sizing, cite `sse.DefaultSubscriberBuffer` instead of duplicating the 64 literal (the constant exists for exactly that, upstream #6).

## How to verify

```bash
rg -oN '\bsse\.[A-Z][A-Za-z0-9]*' --type go -g '!scripts/**' . | sed 's/.*://' | sort | uniq -c | sort -rn | head -16   # Event ×263 · NewEventID ×77 · NewStream ×38 · NewBroadcaster ×33 · EventID ×29 · Broadcaster ×23 · EventStore ×19 …
rg -n 'WithOnDrop|BroadcasterHealth' --type go -g '!scripts/**' . | grep -v _test   # exactly 3 hits: sse_drop_stats.go:43,61 · sse_broadcaster.go:52 — the DropStats wiring
rg -n 'DropStats|SSEDrops' setup/sse.go setup/sse_hub_stats.go setup/setup.go setup/bundle.go dashboardui/dashboard.go | head -8   # :26-27 · :301 (gate :299) · :251 · :74-75 — the wiring map
rg -n 'ReplayFiltered|stream\.Heartbeat' transport/serve.go                          # :366 / :498
rg -n 'func HubReadinessCheck' readiness.go                                          # :170
git -C ~/projects/go-sse ls-remote --tags origin                     | grep -v '\^{}' | awk '{print $2}' | sed 's|refs/tags/||' | grep -v / | sort -V | tail -1   # v0.7.0 (root stream latest)
git -C ~/projects/go-sse ls-remote --tags origin 'refs/tags/ssetest/*'  | grep -v '\^{}' | awk '{print $2}' | sort -V | tail -1   # refs/tags/ssetest/v0.5.0
git -C ~/projects/go-sse ls-remote --tags origin 'refs/tags/sseparse/*' | grep -v '\^{}' | awk '{print $2}' | sort -V | tail -1   # refs/tags/sseparse/v0.2.1
git -C ~/projects/go-sse grep -n 'const DefaultSubscriberBuffer' v0.7.0 -- fanout.go  # v0.7.0:fanout.go:19 — exported at the pinned tag (issue #6)
git -C ~/projects/go-sse diff v0.7.0..HEAD --name-only -- '*.go' ':!*_test.go'       # layout split only: event.go/fanout.go out; lines.go/fanout_lifecycle.go/sseparse splits + example templ regens in — zero public-API change
```

## Prior reports

- 2026-11-14 (prior state of this file): 94/100 count · ~95 impact · no capability flips — census drift only (`NewEventID` ×76→×77; prod sites 19→18, test 33→34); receipts re-pinned (HubReadinessCheck readiness.go:172→:170; SSEHubStats gate → setup/setup.go:299-301; core_bridge HubHealth → :180-188); library truth upgraded: master is no longer "3 chore commits" past v0.7.0 — HEAD 8acf04c carries the 2026-11-20 red-master repair + the 350-line file splits, zero public-API change, and v0.7.0's missing CHANGELOG section was compensated by tag-level verification.
- This report's prior edition: 2026-11-05 (score 72/100 · ~85 impact; rows 8–9 scored missed — then the fleet index's #1 Important issue, "silent SSE drops unobserved"). Resolved by the `DropStats` wiring audited above.
- Realtime context in [docs/guides/sse-and-datastar.md](../../guides/sse-and-datastar.md) and ADR 0046 (WebSocket removal).
