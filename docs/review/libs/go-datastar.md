# go-datastar Utilization Report

> Audit date: 2026-11-14 · Method: library-deep-dive (fleet series, [index](index.html))
> Library: `github.com/larsartmann/go-datastar` (root + `broadcast`/`static`/`datastartest` submodules, lockstep tag stream) · Checkout: `/home/lars/projects/go-datastar`

## Verdict at a glance

| Pin                                             | Latest (verified 2026-11-14) | Prod import sites                        | Adoption                                    | Score               |
| ----------------------------------------------- | ---------------------------- | ---------------------------------------- | ------------------------------------------- | ------------------- |
| go-datastar v0.7.0 + broadcast v0.7.0 (+ static v0.7.0 indirect) | **v0.7.0 = latest** (four-module lockstep cut, 2026-11-07) | adapter (`datastar/`), `setup/`, 2 demos, integration tests | **Used — the whole `datastar/` adapter module is built on it** | **93/100 · at latest** |

## What the library is

Server-side DataStar protocol vocabulary for Go — the opinionated alternative to the official `starfederation/datastar-go` (v1.2.2): patches are **first-class values** (`Patch` → `sse.Event`), not methods on a live connection. Root = patch constructors + inbound signal decoding + fluent `Response` + embedded-JS serving; `broadcast` = fan-out hub (sharing, replay, heartbeats); `static` = the embedded datastar JS client (a **different version line from the Go module**: JS v1.0.x vs Go v0.7.0); `datastartest` = E2E SSE assertions.

## How cqrs-htmx uses it today

- `datastar/` module (v4, ~1219 LOC) — the optional adapter: near-complete re-export surface + thin `(Patch, error)` constructors (`datastar/patch.go:25-199`), `EventBridge` domain-event→patch mapping with `OnError` (`datastar/event_bridge.go:30-128`), deprecated alias shims to `broadcast` (`datastar/broadcaster.go:18-54`).
- `setup/sse.go:60` — `broadcast.NewBroadcasterFromHub(b.Broadcaster.Hub())`: the DataStar endpoint shares the root SSE hub (ADR-0050), session-gated at `setup/sse.go:105-112`; script served at `setup/mount.go:126`.
- `integration_test/datastar_route_split_test.go:27-28` — proves htmx + DataStar ride one hub.
- `examples/datastar-demo` (full stack: ScriptHandler, EventBridge, signals); `e2e/server/go.mod:91-92,115` pins arrive indirect (via `templ-components/datastar` v1.27.0).

## Capability inventory

| #   | Capability                                                               | Status             | Evidence / reason                                                                                                                                              |
| --- | ------------------------------------------------------------------------ | ------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | Patch vocabulary (8 merge modes, namespaces, view transitions, templ)    | 🟢 Fully leveraged | 1:1 delegation, zero reimplementation (`datastar/patch.go:124-164`); used by demo + `setup/setup_datastar_test.go:162`                                          |
| 2   | Inbound decoding (`ReadSignals`, `LastEventID`)                          | 🟢 Fully leveraged | re-exported `datastar/patch.go:169-178`; demo decodes signals from POSTs                                                                                       |
| 3   | Fluent `Response` builder (patch/redirect/remove/prefetch/errors)        | 🟢 Fully leveraged | aliased + wrapped over `sse.NewStream` (`datastar/patch.go:183-185`), covered by `datastar/response_test.go`                                                    |
| 4   | Script serving + v0.7.0 cache-busting (`ScriptTagVersioned`, `ETagFor`)  | 🟢 Fully leveraged | demo splices `godatastar.ScriptTagVersioned("/datastar.js")` into the page head at init (`examples/datastar-demo/main.go`, the versioned-URL cache-busting contract) and `TestSmoke_VersionedScriptServing` pins the versioned tag + ETag/304 revalidation pair (B2 of the follow-through plan closed the unversioned-tag gap 2026-11-15); `ScriptHandler` served at `setup/mount.go:126`, demo `main.go:24` |
| 5   | Broadcast fan-out & hub sharing (`FromHub`, filters, `SubscriberCount`)  | 🟢 Fully leveraged | cross-transport hub sharing is the flagship pattern and it's the prod wiring (`setup/sse.go:60`, integration test above)                                        |
| 6   | Replay ring buffer (`WithReplayCapacity`/`WithStore`/`MemoryStore`)      | ⚪ N/A             | reconnection replay is sourced from the domain EventStore journal through the shared hub (`setup/sse.go:30-33`, `WithMaxReplay`) — ring buffer deliberately bypassed |
| 7   | Broadcast tuning options (buffer size, heartbeat interval/disable)       | ⚪ N/A             | drop/heartbeat policy is owned by the shared cqrs-htmx hub (`SSEDrops.Option()`, `setup/sse.go:27`), not the broadcast wrapper                                  |
| 8   | `datastartest` E2E helpers (SSE parsing, chunk-boundary assertions)      | 🟢 Fully leveraged | adapter tests assert the parsed wire format — event types, dataline-exact selector/mode/elements, exact signals JSON, script content (`datastar/response_test.go`, `coverage_test.go`; module direct in `datastar/go.mod` since the 2026-11-15 B3 pass) |
| 9   | Deprecated alias shims to `broadcast` (v5 glide path, ADR-0062)          | 🟡 Partially used  | 6 `Deprecated` markers, removal bundled with v5 (`datastar/broadcaster.go`); zero non-test in-repo consumers remain (one doc mention, `sse_broadcaster.go:64`) — on track |

## Score derivation

9 rows − 2 N/A = **7 applicable**; 6 full ×1 + 1 partial ×0.5 = **6.5 → 93/100** (band: very high — protocol adoption complete and the two v0.7.0 deltas adopted 2026-11-15; the remaining half-point is the deprecated broadcast shims on their deliberate v5 glide path).

## Version currency

At latest everywhere: v0.7.0 lockstep (root + `broadcast` + `static`) across `datastar/`, `setup/`, `integration_test`, and all examples (indirect elsewhere). v0.6.x→v0.7.0 in one line: four-module lockstep cut repairing v0.6.3's root-only release defect — JS bundle v1.0.3→v1.0.4, `WithHeartbeatDisabled`, `ScriptTagVersioned`+`ETagFor`, `storecontract`; one BREAKING (`ScriptHandlerWith` drops its dead param — no repo call sites). **JS SDK status:** `static` at v0.7.0 embeds datastar JS **v1.0.4** — the embedded-v1.0.3 gap from the 2026-11-07 deep-dive is closed. Stale doc: `docs/guides/datastar-integration.md` still quotes "v1.0.3; measured 2026-11-07".

## Should cqrs-htmx use it (more)?

Both 2026-11-14 cheap wins are DONE (2026-11-15, B3): (1) the demo uses `ScriptTagVersioned` directly (`godatastar` import — a re-export through the adapter was rejected on ADR-0062/D2 policy: a re-export is the permanent-alias class); (2) `datastartest` carries the adapter's SSE assertions. Everything else is as deep as the contracts require — replay via the domain EventStore, hub-owned drop policy, and shims on the v5 glide path are deliberate (ADR-0050/0062).

## How to verify

```bash
rg -n 'go-datastar' --glob 'go.mod' . | grep -v indirect   # all direct pins at v0.7.0
git -C ~/projects/go-datastar for-each-ref --sort=-v:refname --format='%(refname:short)' refs/tags | head -5
rg -n 'const Version' ~/projects/go-datastar/static/static.go  # 1.0.4 — deep-dive gap closed
rg -n 'ScriptTagVersioned|ETagFor|WithHeartbeatDisabled' --glob '*.go' . | wc -l   # 3 — demo splice (main.go:350) + its doc comments (B3, 2026-11-15)
rg -c 'Deprecated' datastar/broadcaster.go                # 6 alias shims, removal bundled with v5
rg -n 'NewBroadcasterFromHub' setup/sse.go integration_test/datastar_route_split_test.go
```

## Prior reports

- [2026-11-07 DataStar client-side deep dive](../../research/2026-11-07_datastar-client-side-deep-dive.html) (client-side; JS SDK v1.0.4 studied against the then-embedded v1.0.3; demo `data-on:load` defect found+fixed same-session; ADR-0050 Tier-4 M11–M14) — see also [research series row 34](../../research/README.md).
- Guides: [datastar-integration.md](../../guides/datastar-integration.md), [sse-and-datastar.md](../../guides/sse-and-datastar.md).
