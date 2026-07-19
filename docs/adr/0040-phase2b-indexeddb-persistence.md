# ADR-0040: Phase 2b — Reverse ADR-0030, Add IndexedDB Persistence to the Offline Command Queue

## Status

ACCEPTED — 2026-07-19

**This decision reverses ADR-0030.** ADR-0030 rejected client-side durability for
the offline command queue with "Phase 2b will never ship." The maintainer has
re-evaluated that decision and now accepts a focused, opt-in persistence layer.
ADR-0030 is now SUPERSEDED.

## Context

ADR-0029 shipped Phase 2a: a SharedWorker that coordinates offline command
retries while at least one tab is open. The SharedWorker's queue lives in
memory, so queued commands are lost when **all** tabs close (the SharedWorker
lifecycle ends). ADR-0030 was asked "must writes survive closed tabs?" and
answered **no**, arguing that client-side persistence "doesn't belong in a
server-side Go library."

Two things changed since ADR-0030:

1. **Real-world usage.** Users close laptops, browsers crash, and tabs get
   accidentally dismissed. The Queue-Only contract (the server still owns
   `decide()`) is correct, but losing a queued mutation the user believed was
   saved undermines trust in the optimistic UI. The cost of persistence is now
   judged worth the durability benefit.

2. **A clean abstraction boundary exists.** The persistence layer is entirely
   contained in `adminui/assets/sync-worker.js` + `admin.js` — vanilla JS,
   zero new Go code, zero new server dependencies. Consumers who never mount
   the admin panel are unaffected. Consumers who do not enable the sync worker
   are unaffected. The library principle (never enforce defaults consumers
   might disagree with) is preserved because persistence is confined to the
   optional admin UI's own assets.

## Decision

**Add IndexedDB persistence to the SharedWorker command queue.**

- The SharedWorker opens an IndexedDB database (`cqrshtmx-sync`, store
  `commands`) and persists each queued command envelope
  `{commandId, verb, url, values, headers, queuedAt}` on enqueue.
- On ACK (command confirmed **or** permanently rejected by the server), the
  persisted entry is deleted.
- On SharedWorker spawn (new tab opens after all tabs were closed), the worker
  drains all persisted commands and broadcasts `retry` with the full envelope to
  every connected tab, enabling cross-tab and cross-session retry.
- The tab reconstructs the HTMX request via `htmx.ajax(verb, url, {target,
values, headers})` when the originating DOM element is gone, so a persisted
  command is not silently dropped.
- The worker broadcasts a `pending` count so the UI indicator can show
  "N commands syncing…".
- If IndexedDB is unavailable (private browsing, quota exceeded), the worker
  degrades gracefully to the previous in-memory-only behavior.

## Why IndexedDB (not OPFS, not Service Worker)

The original TODO referenced "OPFS persistence." OPFS (Origin Private File
System) is synchronous-capable only in dedicated Workers; in a SharedWorker the
async OPFS API offers no advantage over IndexedDB, and IndexedDB is the
universally available, well-understood browser persistence primitive (works in
page, SharedWorker, and Service Worker contexts across all modern browsers).
Service Worker + Background Sync API was rejected in ADR-0030 as Chrome-only;
that reasoning still holds.

## Consequences

- **Positive:** Queued commands survive closed tabs and browser restarts. The
  optimistic UI's promise ("queued, not lost") now holds across sessions.
- **Positive:** Zero server-side changes; the feature is confined to admin UI
  assets. Non-admin consumers see no change.
- **Positive:** Graceful degradation when IndexedDB is blocked — the queue
  falls back to in-memory, matching the pre-ADR-0040 behavior exactly.
- **Negative:** A small amount of client-side state (queued command envelopes)
  now lives in the browser. This is acceptable because it is (a) confined to the
  optional admin UI, (b) deleted on ACK, and (c) purely an optimization for
  retry — the server remains the source of truth.
- **Negative:** Cross-session retry reconstructs the request into a synthesized
  host element when the original DOM is gone; the result rendering may differ
  from the original page context. This is an acceptable tradeoff for not
  silently losing the command.

## Relationship to ADR-0030

ADR-0030 is now **SUPERSEDED by ADR-0040**. ADR-0030's architectural objection
("persistence doesn't belong in a server-side Go library") is respected by
confining all persistence logic to optional admin UI JavaScript assets — the Go
library itself remains free of client-side persistence concerns.
