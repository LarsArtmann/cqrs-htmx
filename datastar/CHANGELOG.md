# Changelog

All notable changes to this module will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

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
