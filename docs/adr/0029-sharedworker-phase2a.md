# ADR 0029: SharedWorker for Phase 2a Offline Command Sync

**Status:** Accepted
**Date:** 2026-06-28
**Related:** [ADR 0023](0023-command-sync.md), [ADR 0024](0024-honest-ui.md), [ADR 0025](0025-phase2-research.md), [ADR 0027](0027-decide-stays-on-server.md), [Brainstorming](../brainstorming/2026-06-27_offline-first-command-sync-research.html)

## Context

ADR 0025 framed two blocking questions for Phase 2 (offline command sync):

- **Q1** (where does `decide()` run?): answered by [ADR 0027](0027-decide-stays-on-server.md) — Queue-Only. No client-side domain logic.
- **Q2** (must writes survive closed tabs?): this ADR answers it.

Queue-Only eliminates every reason a Service Worker was on the table. The client never runs `decide()`, never materializes local state, never needs sqlite-wasm or OPFS. What remains is a command queue + retry coordinator — and a SharedWorker is the architecturally correct home for that.

## Decision

**Phase 2a uses a SharedWorker with an in-memory command queue.** No Service Worker, no IndexedDB, no OPFS. The worker is a coordinator, not a proxy.

### Architecture

```
Tab A ──┐                        ┌── SharedWorker (1 per origin)
Tab B ──┼── MessagePort ─────────┤   ├── in-memory queue [{commandId, port}]
Tab C ──┘                        │   ├── navigator.onLine / online / offline
                                 └── on reconnect: postMessage 'retry' to tabs
```

The SharedWorker does **three things only**:

1. **Queue** command IDs from tabs when offline
2. **Detect** connectivity changes (`navigator.onLine` + `online`/`offline` events at worker scope)
3. **Retry** — on reconnect, tell each originating tab to re-issue its queued command via `htmx.trigger()`

The SharedWorker does **not**:
- Send HTTP requests (tabs do, via HTMX — preserving response swaps)
- Own the SSE connection (tabs keep per-tab `EventSource` — simpler, already works)
- Persist to disk (in-memory only — Queue-Only accepts lost commands on last-tab-close)
- Run domain logic (ADR 0027: decide() stays on server)

### Offline detection: reactive, not proactive

The tab does NOT proactively intercept requests when offline. Instead:

1. HTMX sends the request normally (`htmx:beforeRequest` stamps `X-Command-Id` + sets pending — already implemented in Phase 1)
2. If offline, the browser fails the request instantly (no wasted bandwidth)
3. The tab catches `htmx:sendError` (network failure) or `htmx:responseError` (5xx)
4. If the failure is network-related, the command ID is enqueued to the SharedWorker
5. The UI shows "pending (queued)" — NOT "rejected" (offline ≠ server rejection)
6. On reconnect, the SharedWorker tells the tab to retry → HTMX re-sends → server processes → SSE ACK confirms

This preserves HTMX's request/response/swap lifecycle. The SharedWorker is only involved when things go wrong.

### Retry mechanism

On reconnect, the SharedWorker posts `{type: "retry", commandId}` to the originating tab's `MessagePort`. The tab:

1. Finds the DOM element: `document.querySelector('[data-command-id="' + commandId + '"]')`
2. If found: `htmx.trigger(element, "click")` — re-issues the full HTMX request (form values, headers, swap)
3. If not found (user navigated away): the command is silently dropped. This is acceptable — Queue-Only + in-memory means DOM-bound retry. The honest-UI shows it as "rejected (element not found)".

### Honest-UI states

Extends [ADR 0024](0024-honest-ui.md) with one new sub-state:

| State | Meaning | UI treatment |
|-------|---------|-------------|
| `pending` | Request in flight (online) | Muted/dashed, yellow dot |
| `pending` + `[data-sync-queued]` | Queued, waiting for network | Amber dashed, "offline" badge |
| `confirmed` | Server confirmed | Solid, green dot |
| `rejected` | Server rejected or element gone | Red border, error + retry |

The `data-sync-queued` attribute distinguishes "pending because in-flight" from "pending because offline." The sync-bar indicator gains an "Offline — N queued" status.

## Why Not Service Worker?

| Factor | SharedWorker | Service Worker |
|--------|-------------|----------------|
| Survives individual tab close | **Yes** | Yes |
| Survives last-tab-close | No (acceptable for Queue-Only) | Yes |
| Background Sync (closed-tab flush) | N/A | **Chrome/Edge only** — Firefox disabled, Safari nonexistent |
| OPFS / sync I/O | **Full access** | **Unavailable** (fatal for sqlite-wasm — irrelevant for Queue-Only) |
| Idle eviction | None (alive while ≥1 tab open) | ~30s inactivity teardown |
| Long-lived SSE connection | **Yes** | Fighting eviction constantly |
| One-per-origin collision | No | **Yes** — can't ship a SW from a library; consumer already owns theirs |
| Matches Queue-Only needs | **Perfect** | Over-engineered |

The SW's one unique capability — Background Sync API for closed-tab writes — is Chrome-only and requires the consumer to surrender their SW scope. For a library, that's unacceptable. The SharedWorker covers 90% of the value (cross-tab queue, survives tab switches, single coordinator) at 10% of the complexity.

## Why Not IndexedDB?

The research doc concluded "IndexedDB is Dead — OPFS or Nothing." IndexedDB is async-only (callback hell), has no query model worth using for a FIFO, and its wrapper libraries (Dexie, idb) are fragile. For Phase 2a's in-memory queue, there is nothing to persist — the queue is a JS array that lives and dies with the SharedWorker. Persistence is a Phase 2b concern (if ever needed).

## Phase 2b (Optional, Future)

If closed-tab writes become a hard requirement:

1. Consumer registers a Service Worker in **their** SW scope
2. The SW reads queued commands from **OPFS** (not IndexedDB) — the same file the SharedWorker writes to
3. On `sync` event (Background Sync API, Chrome only): drain the OPFS queue via `fetch()` + `X-Command-Id`
4. SharedWorker stays as-is — it's the leader while tabs are alive; SW is the revival path when none are

This is purely additive. The Phase 2a SharedWorker, queue format, message protocol, and honest-UI states are all identical. Phase 2b only adds an OPFS write in the SharedWorker + a drain plugin for the consumer's SW.

## Consequences

- **Simple**: SharedWorker is ~60-80 lines of vanilla JS. No dependencies, no framework.
- **Progressive enhancement**: Online behavior is unchanged. Offline capability activates only when the SharedWorker connects.
- **Library-friendly**: Shipped via `go:embed` (like `htmx.min.js` and `admin.js`). Consumer serves it from their Go binary. No CDN, no build step.
- **Honest**: Offline ≠ rejected. Queued commands show "pending (offline)." On reconnect, they retry. If the element is gone, they show "rejected (element not found)" — never silently dropped.
- **Limitation**: Commands are lost when all tabs close. This is the Queue-Only contract (ADR 0027). Phase 2b (SW + OPFS) can extend this if needed.
- **Browser support**: SharedWorker is supported in Chrome, Edge, Firefox, Safari 16+. Feature-detection degrades gracefully (no SharedWorker = no offline queue, online path unaffected).
