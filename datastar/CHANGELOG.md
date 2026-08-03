# Changelog

All notable changes to this module will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Added

- Initial Datastar adapter module for cqrs-htmx.
- `ScriptHandler` / `ScriptHandlerWith` / `ScriptTag` / `Version` — embed and serve datastar.js (v1.0.2) with ETag caching.
- `ReadSignals` — decode Datastar client signals from POST body or GET query params.
- `NewResponse` — fluent Datastar SSE response builder with `PatchSignals`, `PatchSignalsIfMissing`, `PatchElements`, `PatchElementsTempl`, `RemoveElement`, `Redirect`, `ExecuteScript`, `ApplyPatches` methods.
- Patch constructors: `ElementsPatch`, `ElementsTemplPatch`, `SignalsPatch`, `SignalsIfMissingPatch`, `RemovePatch`, `ScriptPatch`, `RedirectPatch`.
- `Broadcaster` — fan-out Datastar SSE patches to all connected clients with patch ring-buffer replay on reconnection.
- `EventBridge` — declarative domain-event-to-Datastar-patch mapping via `Map`/`Unmap`/`Handle`.
- `ErrorResponse` / `NotificationResponse` — helpers that send notifications as Datastar signal patches.
- Full SDK option re-exports: all `With*` functions, type aliases, and constants for single-import convenience.
