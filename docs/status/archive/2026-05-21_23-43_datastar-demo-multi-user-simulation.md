# Status Report — 2026-05-21 23:43

_Session: Datastar vs HTMX comparison + multi-user simulation demo_

---

## a) FULLY DONE

### Datastar vs HTMX Comparison

- Comprehensive table comparison across 15+ dimensions produced
- Analyzed every file in cqrs-htmx for HTMX-specific surface area (htmx.go, response.go, notify.go, handler.go, errors.go, middleware.go, options.go)
- Researched Datastar Go SDK (`starfederation/datastar-go v1.2.1`), SSE API, signals, PatchElements, templ support
- Honest assessment: HTMX fits CQRS command→response better; Datastar excels at real-time streaming

### Datastar Demo (`examples/datastar-demo/`)

- **Complete event-sourced CQRS Todo app** using `go-cqrs-lite/core v1.5.0` + Datastar SSE
- **5 files**: `domain.go`, `handlers.go`, `main.go`, `go.mod`, `go.sum`
- **Domain layer**: Todo aggregate, 3 commands (Create/Toggle/Delete), 3 events, in-memory EventStore, Projector (read model from events)
- **Datastar integration**: `ReadSignals`, `NewSSE`, `PatchElements`, `RemoveElement`, `MarshalAndPatchSignals`
- **Broadcaster fan-out**: Proper subscriber registry (not single channel) — each SSE client gets its own buffered channel
- **User attribution**: Domain events carry `User` field ("you" or bot name), threaded through context into command handlers
- **10 simulated users**: Background goroutines create/toggle/delete todos through the same CQRS pipeline, all events broadcast to every connected client
- **Race-condition fix**: Full todo list re-render on every broadcast event (eliminates `PatchElementsNoTargetsFound` errors from concurrent bot activity)
- **CDN URL fix**: Corrected from `npm/@starfederation/datastar@1.0.0-beta.11` to `gh/starfederation/datastar@v1.0.1`
- **Verified**: Build ✓, vet ✓, all endpoints tested (create, toggle, delete, list, simulate, event stream), no panics or server errors

---

## b) PARTIALLY DONE

Nothing partially done — the demo is functionally complete.

---

## c) NOT STARTED

- Demo tests (no test file in `examples/datastar-demo/`)
- Datastar integration into the main `cqrs-htmx` library (if desired)
- templ integration in the demo (currently uses raw HTML strings)
- Proper stop/cancel for the simulate bots (currently they run until process exits)
- Error handling for SSE write failures (currently silently continues)

---

## d) TOTALLY FUCKED UP (and fixed)

### 1. CDN URL 404 (FIXED)

- **What**: `https://cdn.jsdelivr.net/npm/@starfederation/datastar@1.0.0-beta.11/bundles/datastar.js` → 404
- **Cause**: Wrong CDN path (npm vs gh) and wrong version (beta.11 vs v1.0.1)
- **Fix**: `https://cdn.jsdelivr.net/gh/starfederation/datastar@v1.0.1/bundles/datastar.js`

### 2. Single-channel broadcast bug (FIXED)

- **What**: Used `chan BroadcastEvent` for fan-out — only ONE SSE client received each event
- **Cause**: Go channels are point-to-point, not pub/sub
- **Fix**: `Broadcaster` struct with subscriber registry (`map[chan BroadcastEvent]struct{}`), `Subscribe()`/`Unsubscribe()`/`Send()` methods

### 3. Double-update bug (FIXED)

- **What**: Handlers pushed DOM patches directly AND broadcast pushed the same patches — requesting client saw double updates
- **Cause**: Both handler response and broadcast event stream patched the same DOM elements
- **Fix**: Handlers only clear signals/return 200. ALL DOM updates flow through the broadcast event stream.

### 4. SSE-connection-holding simulate endpoint (FIXED)

- **What**: `handleSimulate` held SSE connection open with `<-r.Context().Done()` — client needed two SSE connections simultaneously
- **Cause**: Overcomplicated design
- **Fix**: Simulate endpoint sends one SSE event then returns immediately. Bots run in detached goroutines.

### 5. `PatchElementsNoTargetsFound` race condition (FIXED)

- **What**: Datastar console errors when trying to morph/remove elements that don't exist in the client's DOM (bot deleted a todo before client's event stream processed the create)
- **Cause**: Individual element patching (`PatchElements(todoHTML)` / `RemoveElement("#todo-{id}")`) breaks with concurrent bot activity
- **Fix**: Re-render the entire todo list (`renderTodoList(cqrs.Read.List())` with `WithModeInner()`) on every broadcast event. Simple, correct, eliminates all race conditions.

### 6. Committed binary (FIXED)

- **What**: `examples/datastar-demo/datastar-demo` (9.3MB compiled binary) was tracked in git
- **Fix**: `git rm --cached` + added to exclusion

---

## e) WHAT WE SHOULD IMPROVE

### Demo Quality

1. **Tests for the demo** — no test coverage at all
2. **HTML injection risk** — `renderTodo` uses `fmt.Sprintf` with user input (title). Should HTML-escape the title string
3. **Bot lifecycle** — no way to stop bots (they run until process exits, no cancel mechanism exposed to UI)
4. **Channel leak on unsubscribe** — `Broadcaster.Unsubscribe` doesn't close the channel. If SSE clients disconnect rapidly, channels accumulate (minor for demo)

### Main Library

5. **Datastar adapter** — could add a `datastar.go` to cqrs-htmx that provides Datastar response builders alongside the existing HTMX ones
6. **Broadcast abstraction** — the `Broadcaster` pattern is generic enough to be part of the library

---

## f) Top 25 Things We Should Get Done Next

| #  | Priority | Item                                                                      |
| -- | -------- | ------------------------------------------------------------------------- |
| 1  | HIGH     | Add HTML escaping to `renderTodo` — prevent XSS from user-provided titles |
| 2  | HIGH     | Add `.gitignore` entry for `examples/datastar-demo/datastar-demo` binary  |
| 3  | HIGH     | Add stop/cancel mechanism for simulate bots (UI toggle)                   |
| 4  | MED      | Add basic tests for the demo (at least domain + handlers)                 |
| 5  | MED      | Consider a `datastar` adapter in the main cqrs-htmx library               |
| 6  | MED      | Verify the demo works in multiple browsers (Chrome, Firefox, Safari)      |
| 7  | MED      | Add error handling for SSE write failures in event stream handler         |
| 8  | MED      | Rate-limit the simulate endpoint to prevent accidental double-start       |
| 9  | LOW      | Add templ integration to the demo (replace raw HTML strings)              |
| 10 | LOW      | Add a "clear all" button to the demo                                      |
| 11 | LOW      | Show event count in the event stream panel                                |
| 12 | LOW      | Add keyboard shortcut (Enter) for the add todo form                       |
| 13 | LOW      | Persist todos across server restarts (file-backed event store)            |
| 14 | LOW      | Add pagination or "load more" to the todo list                            |
| 15 | LOW      | Show todo creation timestamps in the UI                                   |
| 16 | LOW      | Add undo functionality (event sourcing makes this natural)                |
| 17 | LOW      | Color-code event log entries by user                                      |
| 18 | LOW      | Add a "replay events" debug panel showing full event store                |
| 19 | LOW      | WebSocket transport option (Datastar supports SSE only)                   |
| 20 | LOW      | Dockerfile for the demo                                                   |
| 21 | LOW      | Deploy demo somewhere accessible                                          |
| 22 | LOW      | Compare response sizes: Datastar SSE vs HTMX headers for same operations  |
| 23 | LOW      | Benchmark: how many concurrent SSE clients before server struggles?       |
| 24 | LOW      | Add load testing script for the demo                                      |
| 25 | LOW      | Write a blog post / README section comparing the two approaches           |

---

## g) Top #1 Question I Cannot Figure Out Myself

**Should Datastar support be a separate module or integrated into cqrs-htmx?**

The library is called `cqrs-htmx` and its entire API surface (Response builder, HTMXRequest, notification triggers) is HTMX-specific. Adding Datastar would mean either:

- A) A parallel response layer (double maintenance, but keeps HTMX clean)
- B) An abstracted "transport" interface that both HTMX headers and Datastar SSE implement (major refactor of response.go + options.go)
- C) A completely separate library (`cqrs-datastar`)

This is a product/architecture decision only the owner can make.

---

## Test Results

```
go test ./... -count=1 -race  → PASS (289+ tests, 95.7% coverage)
go build ./...                → PASS
golangci-lint run             → 6 issues (errcheck:1, noctx:1, revive:4) — pre-existing, not from this session
```

## Files Changed This Session

```
examples/datastar-demo/domain.go    — Broadcaster fan-out, user attribution, SimulateUser
examples/datastar-demo/handlers.go  — Broadcast-based event stream, race fix, simulate endpoint
examples/datastar-demo/main.go      — CDN fix, simulate button, simulating signal, CSS
examples/datastar-demo/datastar-demo — REMOVED from git tracking (binary)
```
