# ADR-0030: Phase 2 Persistence Strategy — SharedWorker with IndexedDB

## Status

REJECTED — 2026-07-01

**Phase 2b will never ship.** IndexedDB persistence for the SharedWorker queue is a fundamentally inconsistent API surface: it introduces a client-side persistence concern that doesn't belong in a server-side Go library. The Queue-Only design (ADR-0027) is correct — the server owns decide(). Extending client-side durability beyond the SharedWorker's in-memory lifetime adds complexity with no clean abstraction boundary for consumers.

## Original Proposal (retained for historical context)

PROPOSED — 2026-06-29

## Context

ADR-0027 resolved Q1: `decide()` stays on the server (Queue-Only strategy).
ADR-0029 shipped Phase 2a: SharedWorker for background command sync while
the tab is open. Q2 remains: **must writes survive closed tabs?**

Three approaches were evaluated:

1. **SharedWorker only (status quo)** — commands queued in SharedWorker
   memory. Lost when ALL tabs close (SharedWorker lifecycle). Sufficient for
   "sync while browsing" but not "sync after closing laptop."

2. **SharedWorker + IndexedDB persistence** — SharedWorker writes queued
   commands to IndexedDB before attempting dispatch. On restart (new tab
   opens → SharedWorker spawns), it drains pending commands from IndexedDB.
   Survives full browser restart.

3. **Service Worker + Background Sync API** — Service Worker with a
   `sync` event fires even when no tab is open. Browser-dependent (Chrome
   only, not Firefox/Safari). Requires `periodicSync` for reliability.

## Decision

**Approach 2: SharedWorker + IndexedDB persistence.**

Rationale:

- Works across all modern browsers (no Background Sync API dependency)
- SharedWorker already shipped (ADR-0029) — this adds a persistence layer
- IndexedDB is available in all contexts (page, SharedWorker)
- Service Worker adds complexity for marginal benefit (Chrome-only Background Sync)
- The persistence boundary is the right level: "survive browser restart" covers
  95% of real-world scenarios (laptop sleep, accidental close, crash)

## Consequences

- SharedWorker gains an IndexedDB-backed command queue (read on spawn, append
  on enqueue, delete on ACK)
- Page load detects pending commands and shows "syncing N commands..." indicator
- No dependency on Background Sync API or Push API
- Phase 2b scope: add IndexedDB persistence to the existing SharedWorker
  (incremental, not a rewrite)

## Next Steps

1. Extend the SharedWorker to persist the command queue to IndexedDB
2. On SharedWorker spawn, drain pending commands from IndexedDB
3. On ACK, delete from IndexedDB
4. Add UI indicator for pending command count
5. Integration test: close tab, reopen, verify commands drain
