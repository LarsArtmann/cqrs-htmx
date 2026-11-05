# go-sse Utilization Report

> Audit date: 2026-11-05 · Method: library-deep-dive (fleet series, [index](index.html))
> Library: `github.com/larsartmann/go-sse` (+ `go-sse/ssetest` v0.5.0) · Checkout: `/home/lars/projects/go-sse`

## Verdict at a glance

| Pin | Latest (verified 2026-11-05) | Prod import sites | Test import sites | Adoption | Score |
| --- | --- | --- | --- | --- | --- |
| v0.7.0 (+ssetest v0.5.0) | v0.7.0 / ssetest v0.5.0 (= latest) | 18 files | 28 files | **Used — the realtime substrate** | **72/100 count · ~85 impact-weighted** |

## What the library is

Dependency-light SSE serving primitives: `Event`/`EventID` construction and parsing, `Stream` writer (headers, retry, heartbeats), `Broadcaster` fan-out with buffered subscribers, `EventStore`/`FilteredEventStore` replay (incl. `ReplayFiltered`), `LastEventID` reconnect handling, drop hooks (`WithOnDrop`), subscriber health (`BroadcasterHealth`), and a `ssetest` contract kit. WebSocket is deliberately out of scope — cqrs-htmx is SSE-only (ADR 0046).

## How cqrs-htmx uses it today

Symbol census: `sse.Event` ×242, `NewEventID` ×75, `NewStream` ×38, `NewBroadcaster` ×33, `EventID` ×28, `Broadcaster` ×23, `EventStore` ×19 (root's `JournalSSEStore` implements it), `Replay` ×12, `WriteEvent` ×10, `LastEventIDFromRequest` ×8, `ParseEventID` ×6, `WriteRetry`/`SetHeaders`, connected/heartbeat event names aliased in `sse_event.go`. `transport/serve.go` is the deepest consumer: fail-closed replay via `sse.ReplayFiltered` + `FilteredEventStore` (`transport/serve.go:218,366`) and heartbeat comment frames through `stream.Heartbeat` (`transport/serve.go:490-499`). `ssetest` backs the reconnect + sync integration tests. The DataStar module's fan-out moved upstream to `go-datastar/broadcast` (deprecated aliases retained until v5).

## Capability inventory

| # | Capability | Status | Evidence / reason |
| --- | --- | --- | --- |
| 1 | Event construction & IDs (`Event`, `NewEventID`, `EventID`, parse) | 🟢 Fully leveraged | 350+ refs; `MustParseSSEEventID` alias rides `sse.MustParseEventID` |
| 2 | Stream serving (`NewStream`, `WriteEvent`, `SetHeaders`, `WriteRetry`) | 🟢 Fully leveraged | root SSE handlers + transport |
| 3 | Heartbeats | 🟢 Fully leveraged | `startHeartbeat` → `stream.Heartbeat` (`transport/serve.go:490`) |
| 4 | Fan-out (`NewBroadcaster`, `Broadcaster`) | 🟢 Fully leveraged | 56 refs; lifecycle contract (Subscribe/Unsubscribe/Close/SubscriberCount) documented in the skill + guides |
| 5 | Reconnect & replay (`LastEventIDFromRequest`, `Replay`, `ReplayFiltered`, `FilteredEventStore`) | 🟢 Fully leveraged | golden-pinned envelope; fail-closed replay cap + `dedup.Ring` seam (2026-11-04) |
| 6 | `ssetest` contract kit | 🟢 Fully leveraged | `sse_reconnect_integration_test.go`, `command_sync_integration_test.go` |
| 7 | `WithBufferSize` subscriber tuning | 🟡 Partially used | default buffers everywhere; the datastar module only keeps a deprecated alias (upstream `broadcast` variant) |
| 8 | `WithOnDrop` drop observability | 🔴 Missed | 0 prod uses — slow-consumer drops are **silent by design**; a drop hook (log/metric) is the single highest-value unadopted feature |
| 9 | `BroadcasterHealth` | 🔴 Missed | 0 uses; subscriber counts/health would feed adminui's honest sync indicator + dashboardui |
| 10 | Keyed line framing (`KeyedLines`, `WriteKeyedLines`, `JoinLines`) | ⚪ N/A | wire-format assembly helper; payloads are composed as single data fields here |

## Score derivation

10 rows − 1 N/A = **9 applicable**; 6 full ×1 + 1 partial ×0.5 = **6.5 → 72/100** (band: moderate by count). Impact-weighted: every load-bearing serving capability (1–6) is fully adopted including the hard parts (replay filtering, heartbeats, reconnect); the misses are two observability hooks → **~85 effective, high**.

## Version currency

v0.7.0 + ssetest v0.5.0 pinned = latest (verified `git ls-remote --tags origin`, 2026-11-05).

## Should cqrs-htmx use it (more)?

Yes — one concrete TODO-grade item: wire `WithOnDrop` (log + optional metric) into the root `Broadcaster` constructor seam and surface `BroadcasterHealth` where sync indicators already exist. Everything else is at max.

## How to verify

```bash
rg -oN '\bsse\.[A-Z][A-Za-z0-9]*' --type go -g '!scripts/**' . | sed 's/.*://' | sort | uniq -c | sort -rn | head -16
rg -n 'WithOnDrop|BroadcasterHealth' --type go -g '!scripts/**' . | grep -v _test   # expect none
rg -n 'ReplayFiltered|stream.Heartbeat' transport/serve.go
git -C ~/projects/go-sse ls-remote --tags origin | grep -v '\^{}' | sort -V | tail -3
```

## Prior reports

- None specific; realtime context in [docs/guides/sse-and-datastar.md](../../guides/sse-and-datastar.md) and ADR 0046 (WebSocket removal).
