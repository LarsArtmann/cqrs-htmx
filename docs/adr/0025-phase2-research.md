# ADR 0025: Phase 2 Offline Architecture — Research & Decision Framework

**Status:** Proposed (awaiting Q1/Q2 decisions)
**Date:** 2026-06-28
**Related:** [ADR 0023](0023-command-sync.md), [ADR 0024](0024-honest-ui.md), [Brainstorming](../brainstorming/2026-06-27_offline-first-command-sync-research.html)

## Context

Phase 0 (production SSEEventStore) and Phase 1 (ACK protocol + honest UI) are complete. Phase 2 adds true offline capability — the ability to queue commands when the network is unavailable and sync them when connectivity returns.

Two product decisions block Phase 2 design. This document researches the options, presents tradeoffs, and recommends an approach for each.

---

## Q1: Where does `decide()` run?

### The Question

When offline, should the client run domain validation (`decide()`) locally before queuing a command, or should it queue unvalidated commands and let the server re-validate on sync?

### Option A: Queue-Only (No Client-Side Validation) — **RECOMMENDED**

The client queues commands blindly. No domain logic runs in the browser. On reconnect, the server runs `decide()` against the current authoritative state — rejecting stale or invalid commands.

**Pros:**

- Zero client-side domain code (no WASM, no TS port)
- Server is the single source of truth for domain rules
- No risk of client/server validation divergence
- Honest UI handles rejected commands gracefully (never-silent rollback)
- Simplest implementation — the sync protocol already exists (Phase 0+1)

**Cons:**

- User doesn't know if a command is valid until reconnection
- Potential for "rejection shock" — many queued commands fail at once on reconnect

**Effort:** Low. The queue is a JavaScript array persisted to localStorage/sessionStorage. The sync loop sends commands in order, one at a time, waiting for each ACK before sending the next.

### Option B: Full Offline via WASM (TinyGo or Standard Compiler)

Compile the Go `decide()` functions to WASM and run them in the browser. The client validates commands locally before queuing, giving instant feedback even offline.

**Pros:**

- Instant local validation (user knows immediately if a command is invalid)
- Reuses existing Go domain code — no duplication
- Full offline UX parity with online

**Cons:**

- **TinyGo limitations**: Missing `reflect` support (needed by many Go libraries), limited `net/http`, smaller standard library. Our domain code depends on `event/v3` which uses reflection for event payload marshaling.
- **Standard compiler WASM**: Large binary (10-20MB for non-trivial Go programs), slow startup (~100ms+), `syscall/js` overhead. Requires running `wasm_exec.js` shim.
- **Maintenance burden**: Two runtimes (Go server + WASM client) must stay in sync. Every domain change requires recompiling WASM.
- **Security surface**: Domain logic exposed in client-side WASM (reverse-engineerable).

**Effort:** High. 2-4 weeks for TinyGo feasibility spike + event payload adaptation + WASM module loader + build pipeline.

### Option C: TypeScript Port of Domain Logic

Manually port the `decide()` functions to TypeScript. The client runs the TS port for local validation.

**Pros:**

- Small bundle size (tree-shakeable)
- Fast startup (native JS)
- Full control over client-side behavior

**Cons:**

- **Duplication**: Two implementations of every domain rule. High maintenance cost.
- **Divergence risk**: Go and TS versions drift over time. Bugs in one but not the other.
- **Effort**: Every new domain event requires updating both Go and TS. Non-trivial for 11 commands + 12 events.

**Effort:** Very High. 1-2 weeks per aggregate to port + ongoing sync cost.

### Recommendation for Q1

**Option A (Queue-Only)** is the clear winner for v1:

- 80% of the value (offline write capability) at 10% of the effort
- Honest UI already handles rejection gracefully
- Can upgrade to Option B/C later if instant-validation is needed
- Aligns with the "sync commands not events" principle — the server always re-validates

---

## Q2: Must writes survive closed tabs?

### The Question

If the user writes commands while offline and then closes the browser tab, should those commands sync when the tab is reopened? Or is it acceptable to lose unsynced commands when the tab closes?

### Option A: Tab-Scoped (SharedWorker) — **RECOMMENDED for v1**

Commands persist in a SharedWorker (shared across tabs of the same origin). If all tabs close, the worker is evicted and unsynced commands are lost.

**Pros:**

- Simpler implementation (no Service Worker lifecycle complexity)
- No background sync permission needed
- Faster sync (worker is alive while any tab is open)
- No registration/update/scope complexity

**Cons:**

- Commands lost when ALL tabs close
- Not truly "offline-first" — requires at least one tab open

**Effort:** Medium. SharedWorker + IndexedDB/sessionStorage for queue persistence.

### Option B: Survive Closed Tabs (Service Worker + Background Sync)

Commands persist in a Service Worker with Background Sync API. The SW wakes up when connectivity returns, even if no tab is open.

**Pros:**

- True offline-first: commands sync even with zero tabs open
- Native browser API (no polyfill needed)
- Aligns with Progressive Web App (PWA) model

**Cons:**

- **SW lifecycle complexity**: Idle eviction (Chrome evicts after ~30s idle), update flow, scope management
- **No OPFS in SW context** (SQLite-WASM requires SharedArrayBuffer, not available in SW)
- **Background Sync limitations**: Only Chrome/Edge support it (not Safari, not Firefox stable). Requires `periodicSync` permission.
- **Debugging difficulty**: SW runs in a separate context with limited DevTools
- **Research finding**: LiveStore explicitly rejected SW for these reasons (see brainstorming doc)

**Effort:** High. SW registration + Background Sync + message passing + lifecycle management + cross-browser fallback.

### Recommendation for Q2

**Option A (Tab-Scoped)** for v1:

- SharedWorker covers the common case (user has the tab open but network drops)
- True background sync can be added later for PWAs that need it
- Avoids SW complexity for 90% of the value

---

## Combined Decision Matrix

| Q1 Answer  | Q2 Answer     | Architecture                              | Effort    | Offline Level                          |
| ---------- | ------------- | ----------------------------------------- | --------- | -------------------------------------- |
| Queue-Only | Tab-Scoped    | **SW+Array+ACK** (Phase 0+1 already done) | **Low**   | Write while tab open + offline         |
| Queue-Only | Survive Close | SW + Background Sync + queue              | Medium    | Write while offline, sync on reconnect |
| WASM       | Tab-Scoped    | SharedWorker + WASM decide + queue        | High      | Full offline validation in tab         |
| WASM       | Survive Close | SW + WASM + Background Sync               | Very High | Full PWA offline                       |
| TS Port    | Any           | TS decide + queue + SW/SW                 | Very High | Full offline + small bundle            |

### Recommended Path

**Phase 2a (Queue-Only + Tab-Scoped)**: ~1 week. Commands queue in a SharedWorker, sync on reconnect. Server re-validates everything. Honest UI shows pending → confirmed/rejected for each.

**Phase 2b (Optional — Background Sync)**: +1 week. Add Service Worker for closed-tab sync. Only if product requires it.

**Phase 2c (Optional — WASM Validation)**: +2-4 weeks. Compile decide() to WASM for instant local validation. Only if UX demands zero-latency feedback.

---

## Decision Required

**Q1:** Queue-Only (recommended) vs WASM vs TS Port?
**Q2:** Tab-Scoped/SharedWorker (recommended) vs Service Worker + Background Sync?

Once answered, Phase 2 implementation can begin.
