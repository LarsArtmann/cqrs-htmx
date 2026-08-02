# Changelog

All notable changes to this module will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Added

- Initial Datastar adapter module for cqrs-htmx.
- `ScriptHandler` / `ScriptHandlerWith` / `ScriptTag` / `Version` — embed and serve datastar.js with ETag caching.
- `ReadSignals` / `ReadSignalsQuery` — decode Datastar client signals from POST body or GET query params.
- `NewResponse` — fluent Datastar SSE response builder with PatchSignals, PatchElements, PatchElementsTempl, RemoveElement, Redirect, ExecuteScript methods.
- Patch constructors: `ElementsPatch`, `SignalsPatch`, `RemovePatch`, `ScriptPatch`, `RedirectPatch`.
- `Broadcaster` — fan-out Datastar SSE patches to all connected clients with reconnection support.
- `EventBridge` — declarative domain-event-to-Datastar-patch mapping with Start/Stop lifecycle.
- `ErrorResponse` — helper that sends error notifications as Datastar signal patches.
