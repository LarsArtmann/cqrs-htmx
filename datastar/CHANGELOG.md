# Changelog

All notable changes to this module will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Fixed

- **`NewEventBridge(nil)` fails at construction and `MappedEventTypes` is sorted (2026-10-25, S39).** A nil broadcaster previously constructed fine and panicked only when the FIRST mapped event was handled — far from the cause; `NewEventBridge` now panics immediately with a message naming the fix (`broadcast.NewBroadcaster`). `MappedEventTypes` sorts its result, making it deterministic — the existing test was named `ReturnsSortedCopy` but only asserted `Contains`, so it could not fail on unsorted output; it now asserts the exact order.

## [v4.12.2] - 2026-10-11

### Changed

- **Dependency/floor alignment (2026-10-11 wave):** go-cqrs-lite storage/memory v4.6.2, snapshot v4.6.2, and the 1.27.2 toolchain floor. No API change.

# [v4.11.0] - 2026-09-19

### Changed

- **`Broadcaster` moved upstream to `github.com/larsartmann/go-datastar/broadcast`** (new optional submodule of go-datastar, 2026-09-17): serve loop, patch fan-out, reconnection replay, hub sharing, and the promoted go-sse methods now live there — domain-agnostic, usable without cqrs-htmx. This module keeps `type Broadcaster = broadcast.Broadcaster` plus all constructors as a deprecated facade (removal bundled with v5); `NewEventBridge` accepts upstream-built broadcasters unchanged via the alias. The behavior test suite moved upstream with the implementation. setup/v4 and integration_test now import `broadcast` directly.
- **`Broadcaster` embeds the `*sse.Broadcaster[sse.Event]` hub** (previously a hidden unexported `inner` field): `Health`, `Shutdown`, `Close`, `OnSubscribe`, `OnUnsubscribe` are now promoted from go-sse with identical signatures (six hand-written pass-through methods deleted), and `Subscribe`/`Unsubscribe`/`SubscribeFilter` are callable directly on the adapter — no unwrapping needed. `Broadcast(patch Patch)` still shadows the hub's `Broadcast(sse.Event)` on purpose (patch ergonomics); raw events use `BroadcastEvent` or the hub. New `Hub()` accessor and `NewBroadcasterFromHub(hub)` constructor are the canonical cross-transport sharing API (see `docs/guides/sse-and-datastar.md`). (Now inherited from the upstream broadcast module.)

### Deprecated

- **`Broadcaster`, `NewBroadcaster`, `NewBroadcasterWithBufferSize`, `NewBroadcasterWithReplay`, `NewBroadcasterFromHub`** — import `github.com/larsartmann/go-datastar/broadcast` and use the same names there. Removal bundled with v5.
- **`Raw()` method removed early with the upstream move** (deprecated since the hub-first vocabulary landed, zero in-repo consumers, and a type alias cannot carry methods). Use `Hub()`. `NewBroadcasterFromRaw` remains functional as a deprecated function until v5.

## [v4.1.0] - 2026-08-07

### Changed — go-datastar Migration

- **Replaced `starfederation/datastar-go` SDK with `go-datastar` + `go-sse`.** Patches are now first-class values implementing `Patch` (`Event() sse.Event`), not method calls on a live SSE generator. This enables composition with go-sse's `Broadcaster`, `SubscribeFilter`, `Shutdown`, and `Health` infrastructure.
- **Broadcaster rewritten** to wrap `sse.Broadcaster[sse.Event]`. Gains Shutdown/Health/SubscribeFilter for free. Drops 286 lines of hand-rolled fan-out, replay, and heartbeat code.
- **Deleted files**: `options.go`, `signals.go`, `response.go`, `errors.go`, `script_handler.go`, `script_embed.go` — all replaced by go-datastar equivalents.
- **Response API changed**: methods return `error` instead of chaining (`*Response`). Use `MarshalAndPatchSignals(v)` instead of `PatchSignals(map)`.
- **Broadcaster API changed**: `NewBroadcasterWithReplay` and `NewBroadcasterWithHeartbeat` removed — replay is handled by go-sse EventStore, heartbeat by `sse.Stream.Heartbeat`.
- Option re-exports renamed to match go-datastar naming (e.g., `WithExecuteScriptAutoRemove` → `WithScriptAutoRemove`, `WithPatchElementsEventID` → `WithElementsEventID`).

### Added

- Initial Datastar adapter module for cqrs-htmx.
- `ScriptHandler` / `ScriptHandlerWith` / `ScriptTag` / `Version` — embed and serve datastar.js (v1.0.2) with ETag caching.
- `ReadSignals` — decode Datastar client signals from POST body or GET query params.
- `NewResponse` — fluent Datastar SSE response builder with `PatchSignals`, `PatchSignalsIfMissing`, `PatchElements`, `PatchElementsTempl`, `RemoveElement`, `RemoveElementByID`, `Redirect`, `ReplaceURL`, `ExecuteScript`, `ConsoleLog`, `ConsoleError`, `DispatchCustomEvent`, `Prefetch`, `ApplyPatches` methods.
- Patch constructors: `ElementsPatch`, `ElementsTemplPatch`, `SignalsPatch`, `SignalsIfMissingPatch`, `RemovePatch`, `ScriptPatch`, `RedirectPatch`.
- `Broadcaster` — fan-out Datastar SSE patches to all connected clients with patch ring-buffer replay on reconnection (`NewBroadcasterWithReplay`). Optional SSE heartbeat keep-alive via `NewBroadcasterWithHeartbeat`.
- `EventBridge` — declarative domain-event-to-Datastar-patch mapping via `Map`/`Unmap`/`Handle`, with optional `OnError` callback for observability.
- `ErrorResponse` / `NotificationResponse` — helpers that send notifications as Datastar signal patches.
- Full SDK option re-exports: all `With*` functions, type aliases, and constants for single-import convenience.
