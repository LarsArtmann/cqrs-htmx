# Status Report: Offline-First Command Sync — Session 1 Complete

> **Date:** 2026-06-28 10:10
> **Scope:** Offline-first command sync (Phases 0+1 of the execution plan)
> **Branch:** master, pushed
> **Health:** 🟢 ALL GREEN — tests pass, lint clean, errorfamily 0, build OK

---

## a) FULLY DONE

### Research & Planning

- ✅ **Brainstorming doc** (`docs/brainstorming/2026-06-27_offline-first-command-sync-research.html`) — 2,840 lines covering LiveStore architecture, Service Worker vs SharedWorker, fanOut vs watermill, IndexedDB rejection, Honest Optimistic UI, the Command problem, and the command-sync solution
- ✅ **Execution plan** (`docs/planning/2026-06-28_09-38_offline-first-command-sync.md`) — 58 tasks across 5 Pareto tiers with mermaid.js execution graph
- ✅ **ADR 0023** (`docs/adr/0023-command-sync.md`) — Command-sync architecture: sync commands, not events
- ✅ **ADR 0024** (`docs/adr/0024-honest-ui.md`) — Honest UI protocol: never lie about pending state

### Phase 0: Production SSEEventStore (`event_store_sse.go`)

- ✅ `JournalSSEStore` — production `SSEEventStore` backed by `event.Journal`/`event.SeekableJournal`
- ✅ `EventToSSEMapper` — consumer-provided function to render event payloads as HTML
- ✅ `WithMaxReplay(n)` — limits initial sync (default: 1000)
- ✅ Position-based `ReadFrom` when `SeekableJournal` available, `ReadAll`+filter fallback
- ✅ Edge cases: empty store, empty cursor, last event, unknown cursor, invalid cursor
- ✅ Thread-safe (concurrent reads verified with race detector)
- ✅ 12 tests, all pass with `-race`

### Phase 1: ACK Protocol (`ack.go`)

- ✅ `CommandAck` struct with `{commandId, status, error}` JSON wire format
- ✅ `AckStatus` constants (`AckConfirmed` / `AckRejected`)
- ✅ `X-Command-Id` header convention (opt-in — no header, no ACK)
- ✅ `CommandIDFromRequest(r)` helper
- ✅ `Broadcaster.BroadcastOnAck()` — SSE ACK hook factory
- ✅ `Broadcaster.BroadcastOnAckFunc(fn)` — custom SSE ACK mapper
- ✅ `WSBroadcaster.BroadcastOnAckWS()` — WebSocket ACK hook factory (transport parity)
- ✅ `WSBroadcaster.BroadcastOnAckWSFunc(fn)` — custom WS ACK mapper
- ✅ 9 tests (SSE + WS), all pass with `-race`

### Documentation

- ✅ `AGENTS.md` — SSEEventStore + ACK protocol documented in SSE section
- ✅ `doc.go` — package-level documentation with SSE reconnection + ACK examples
- ✅ `FEATURES.md` — SSE + WS sections updated with JournalSSEStore + ACK rows

### Code Quality

- ✅ Zero dead code (removed: `EventType` field, `WrapAckError`, redundant `JSON()` method)
- ✅ Zero stdlib error constructors (errorfamily: 0 violations)
- ✅ golangci-lint: 0 issues
- ✅ 21 new tests (12 SSEEventStore + 9 ACK), all pass with `-race`

---

## b) PARTIALLY DONE

### Honest UI Rendering (adminui)

- 🟡 **CSS classes designed** (in brainstorming doc) but NOT implemented in `tailwind.css`
- 🟡 **JS handlers designed** but NOT implemented in `admin.js`
- 🟡 **Global sync indicator designed** but NOT built in `layout.templ`
- 🟡 **Optimistic pending rendering** concept proven but NOT wired to HTMX events

### Demo Wiring

- 🟡 `examples/admin-demo/` exists but does NOT demonstrate SSEEventStore or ACK
- 🟡 No SSE endpoint in the demo to showcase durable replay

---

## c) NOT STARTED

### Phase 2: Offline Queue (Service Worker + OPFS)

- ⬜ Q1 decision (where does `decide()` run? queue-only vs full-offline) — NOT DECIDED
- ⬜ Q2 decision (closed-tab writes required?) — NOT DECIDED
- ⬜ Service Worker scaffold (`sw.js`, registration)
- ⬜ OPFS availability detection + fallback
- ⬜ Client-side SQLite (sqlite-wasm) integration
- ⬜ Command queue schema + CRUD
- ⬜ Fetch interception → queue when offline
- ⬜ Reconnect → replay pending queue
- ⬜ Leader election via Web Locks API
- ⬜ Cross-tab coordination via SharedWorker

### Phase 3: Local Decider

- ⬜ Go→WASM compile path evaluation (TinyGo?)
- ⬜ TypeScript port effort assessment
- ⬜ Minimal WASM decider prototype
- ⬜ Decision: WASM vs TS port vs skip

---

## d) TOTALLY FUCKED UP

Nothing. All code compiles, all tests pass, all commits are clean. The only thing I'd flag as a **design limitation** (not a fuckup) is that `SSEEventStore.EventsAfter` has no error return — errors are logged and swallowed. This is the pre-existing interface contract; changing it would be a breaking API change. Documented but not fixed.

---

## e) WHAT WE SHOULD IMPROVE

1. **The SSEEventStore interface lacks error returns.** `EventsAfter(lastID string) []SSEEvent` silently drops errors. Consider `EventsAfter(ctx, lastID) ([]SSEEvent, error)` in a future v4 — but that's a breaking change.

2. **No integration test wiring SSEEventStore + Broadcaster + ACK end-to-end.** Unit tests prove each piece works, but there's no test proving they work _together_ in a real HTTP handler. This is the next highest-value test.

3. **The brainstorming HTML report had broken ASCII diagrams.** I fixed them (replaced with CSS flow diagrams), but I should have caught this before committing. Lesson: always view HTML in a browser before committing.

4. **The ack.go initial commit had dead code** (`EventType` field, `WrapAckError`, redundant `JSON()`). I caught and fixed it in the self-review, but it should never have been committed in the first place. Lesson: write less code initially.

5. **No BDD test for the ACK protocol.** The project uses Ginkgo for BDD — the new ACK tests are stdlib `testing`. For consistency, they could be Ginkgo `Describe` blocks, though stdlib is fine for simple unit tests.

6. **`TODO_LIST.md` not updated** with the plan's 58 tasks and their status. The execution plan is in `docs/planning/` but the living `TODO_LIST.md` doesn't reference it.

---

## f) Top 25 Things to Get Done Next

Sorted by **impact / effort ratio** (highest first):

| #   | Task                                                                                           | Impact | Effort | Phase |
| --- | ---------------------------------------------------------------------------------------------- | ------ | ------ | ----- |
| 1   | **Integration test**: SSEEventStore + Broadcaster + ReplayEvents end-to-end in an HTTP handler | 5      | 30m    | P0    |
| 2   | **Answer Q1**: where does `decide()` run? (gates Phase 2)                                      | 5      | 30m    | P2    |
| 3   | **Answer Q2**: closed-tab writes required? (gates SW decision)                                 | 5      | 10m    | P2    |
| 4   | Add SSE endpoint to `examples/admin-demo/` showcasing durable replay                           | 4      | 45m    | Demo  |
| 5   | Add ACK wiring to `examples/admin-demo/` (X-Command-Id → SSE ack)                              | 4      | 45m    | Demo  |
| 6   | Honest UI: `.sync-pending` CSS class in `tailwind.css`                                         | 4      | 15m    | P1    |
| 7   | Honest UI: `.sync-confirmed` + `.sync-rejected` CSS classes                                    | 4      | 15m    | P1    |
| 8   | Honest UI: sync indicator in `layout.templ` header                                             | 3      | 30m    | P1    |
| 9   | Honest UI: JS handler for `sync:ack` SSE event in `admin.js`                                   | 5      | 45m    | P1    |
| 10  | Honest UI: `data-sync-state` attribute transitions (pending→confirmed)                         | 5      | 30m    | P1    |
| 11  | Honest UI: never-silent rejection rendering (inline error + retry)                             | 5      | 30m    | P1    |
| 12  | Honest UI: global sync counter (pending/confirmed/failed)                                      | 3      | 30m    | P1    |
| 13  | Honest UI: `aria-live="polite"` region for confirmed announcements                             | 2      | 15m    | P1    |
| 14  | Honest UI: retry button handler for rejected items                                             | 3      | 15m    | P1    |
| 15  | Update `TODO_LIST.md` with plan tasks + status                                                 | 3      | 20m    | Doc   |
| 16  | Add example: optimistic render on `htmx:beforeRequest`                                         | 4      | 30m    | P1    |
| 17  | Rebuild `admin-tw.css` after tailwind changes                                                  | 2      | 5m     | P1    |
| 18  | Run `templ generate` after layout.templ changes                                                | 2      | 5m     | P1    |
| 19  | BDD test: ACK confirmed/rejected lifecycle (Ginkgo)                                            | 3      | 30m    | QA    |
| 20  | Research: sqlite-wasm integration approaches (wa-sqlite fork vs upstream)                      | 3      | 60m    | P2    |
| 21  | Research: Go→WASM decider path (TinyGo feasibility)                                            | 3      | 60m    | P3    |
| 22  | Decide SW vs SharedWorker topology from Q1+Q2                                                  | 4      | 15m    | P2    |
| 23  | Service Worker scaffold (`sw.js` + registration)                                               | 3      | 30m    | P2    |
| 24  | OPFS availability detection + fallback                                                         | 3      | 30m    | P2    |
| 25  | README.md: document the offline-first sync story                                               | 2      | 30m    | Doc   |

---

## g) Top Question I Cannot Figure Out Myself

**Q: For the offline Phase 2, do you need writes to flush when zero tabs are open?**

This single answer determines whether you need a Service Worker at all:

- **If yes** → Service Worker is mandatory (Background Sync API), and you must solve the OPFS-in-SW problem (run sqlite-wasm in a DedicatedWorker spawned by the SW, or accept async storage)
- **If no** → copy LiveStore's SharedWorker+DedicatedWorker topology exactly, ignore the SW entirely, and the implementation is dramatically simpler

I cannot decide this for you because it's a product requirement, not a technical one. Everything else in the offline architecture flows from this answer.
