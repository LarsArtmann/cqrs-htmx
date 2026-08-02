# Datastar Integration Analysis for cqrs-htmx

> **Date:** 2026-08-02
> **Question:** How could this project benefit from [Datastar](https://data-star.dev/)?
> **Verdict:** High-value, low-risk opportunity. Datastar's philosophy is a near-perfect match for cqrs-htmx's architecture. The existing `datastar-demo` already proves the pattern. The recommended integration is a new optional submodule.

---

## 1. Executive Summary

Datastar is not an HTMX replacement bolted onto a CQRS backend. Its founding philosophy — "The Tao of Datastar" — **is the cqrs-htmx architecture**: backend as single source of truth, CQRS for real-time (one long-lived SSE read connection + short-lived command writes), no optimistic updates, SSE as the default transport. The project already implements every principle Datastar advocates — just with HTMX as the frontend reactivity layer.

The gap Datastar fills is **frontend state management and structured SSE updates**. Today, cqrs-htmx consumers who need anything beyond simple HTML swaps must reach for Alpine.js (separate 15 KiB library), hand-rolled JavaScript, or HTMX extensions (idiomorph, SSE, WS). Datastar replaces all of these with one 11.76 KiB file that includes:

- Reactive **signals** (state management with auto-tracking)
- **Morphing** by default (no idiomorph extension needed)
- **Structured SSE protocol** (`patch-elements`, `patch-signals`, `execute-script`)
- **Two-way data binding** (`data-bind`)
- Built-in **retry/reconnection** primitives

---

## 2. Philosophical Alignment (Why This Isn't Forced)

| Datastar Tao Principle | cqrs-htmx Equivalent | Status |
|---|---|---|
| CQRS for real-time: one SSE read + short writes | `Broadcaster` + SSE stream + `App.Command` dispatch | **Already implemented** |
| Backend as source of truth | Event sourcing — backend IS the only truth | **Already implemented** |
| SSE as default response type | Full SSE infrastructure (`SSEStream`, `Broadcaster`, `JournalSSEStore`) | **Already implemented** |
| No optimistic updates | ADR-0024 "Honest UI" — loading indicators, confirm from backend | **Already the design** |
| Backend templating + compression | templ components embedded in Go | **Already implemented** |
| Fat morphing (send large DOM, let morph diff it) | `RenderPartialOrFull` sends full page sections | **Partial — needs morph** |

The conclusion: **cqrs-htmx is already a Datastar-compatible backend.** What's missing is the Datastar frontend adapter and the structured SSE protocol layer.

---

## 3. Current Pain Points Datastar Solves

### 3.1 Frontend State Management (The #1 Gap)

cqrs-htmx has **zero frontend state management**. Today's options for consumers:

| Need | Current Solution | Problem |
|---|---|---|
| Form input binding | HTMX `hx-post` on every keystroke, or Alpine.js | Server round-trip per keystroke, or add a second library |
| Toggle visibility | `hx-get` round-trip, or Alpine.js | Server load for trivial UI state |
| Loading indicators | `hx-indicator` + custom CSS | Manual wiring per element |
| Multi-step form state | Server-side session or Alpine.js | Either approach adds complexity |
| Filter/search state | URL params + server re-render | Every filter change = full round-trip |

Datastar signals eliminate all of these:

```html
<!-- cqrs-htmx + HTMX today: server round-trip for a search filter -->
<input name="q" hx-get="/users" hx-trigger="keyup changed delay:300ms"
       hx-target="#user-list" hx-push-url="true">

<!-- cqrs-htmx + Datastar: instant client-side filter, no round-trip -->
<input data-bind:searchQuery>
<div data-text="$users.filter(u => u.name.includes($searchQuery)).length + ' results'"></div>
```

### 3.2 The dashboardui Polling Problem

`dashboardui/handler_overview.go` uses **HTMX polling** for projection health:

```html
<div hx-get="/.../-/partials/projection-health" hx-trigger="every 10s" hx-swap="outerHTML">
```

This means:
- 10-second latency on health changes (stale data)
- One HTTP request per connected client every 10 seconds (waste)
- Server re-renders the full partial each time (CPU)

With Datastar, projection health becomes signal-driven:

```html
<!-- Server patches $projectionLag signal via SSE when it changes -->
<div data-signals:projectionLag="0">
  <span data-text="$projectionLag > 5000 ? 'LAGGING' : 'HEALTHY'"
        data-class:text-red-500="$projectionLag > 5000">
  </span>
</div>
```

The server sends one `datastar-patch-signals` event when the lag changes — zero polling, instant updates, minimal bandwidth.

### 3.3 The SSE Protocol Gap

cqrs-htmx's current SSE infrastructure sends **raw events** that consumers must format manually. The `datastar-demo` proves this: `domain_cqrs.go` has a broadcast bridge that manually calls `renderTodo()` (HTML `fmt.Sprintf`) for each event type. This is boilerplate that Datastar's structured protocol eliminates.

**Current (datastar-demo `domain_cqrs.go`):**
```go
events.Subscribe(func(e DomainEvent) {
    evt := BroadcastEvent{User: e.User, Time: e.OccurredAt}
    switch e.Type {
    case "TodoCreated":
        var p TodoCreatedPayload
        _ = json.Unmarshal(e.Payload, &p)
        evt.Kind = "todo_created"
        evt.Data = renderTodo(todo) // manual HTML generation
    case "TodoDeleted":
        evt.Kind = "todo_deleted"
        evt.Data = "#todo-" + e.AggregateID
    // ... more cases
    }
    broadcast.Send(evt)
})
```

**With a Datastar adapter:**
```go
// One line: the adapter maps domain events to Datastar SSE events
dsBridge := datastaradapter.NewEventBridge(broadcaster, renderFunc)
eventBus.SubscribeAll(dsBridge.Handle)
// Handle() auto-maps events to patch-elements/patch-signals
```

### 3.4 The Morphing Gap

HTMX requires the **idiomorph extension** for DOM morphing. cqrs-htmx embeds it (`extensions/idiomorph-ext.min.js`). Datastar morphs by default — no extension, no configuration. For `adminui` and `dashboardui`, this means:

- Event listeners survive DOM updates automatically
- CSS transitions continue uninterrupted
- Form input focus preserved during partial re-renders
- No `hx-preserve` or `hx-disinherit` workarounds needed

### 3.5 The Offline Sync Complexity

cqrs-htmx's offline sync (`sync/sync-worker.js` + `sync/sync-client.js`, v1.3.0) is 500+ lines of hand-rolled vanilla JavaScript implementing:
- SharedWorker lifecycle
- IndexedDB command queue
- SSE `EventSource` for ACK confirmations
- HTMX event interception (`beforeRequest`, `sendError`, `responseError`)
- DOM indicator management (`data-sync-state`, `data-sync-status`)

Datastar's CQRS model (long-lived SSE read + short writes with auto-retry) is the **natural architecture** for offline sync. Datastar provides:
- `retry: 'auto'` with exponential backoff built into `@post()`/`@get()`
- `data-indicator` signals for loading state (no manual DOM manipulation)
- Signal-based state that survives reconnection naturally

The sync-worker.js could potentially be simplified or replaced with Datastar's built-in retry primitives for the common case, keeping IndexedDB persistence as the durable layer.

---

## 4. What Already Exists

The project already has a working proof-of-concept: `examples/datastar-demo/`.

| Aspect | Status |
|---|---|
| Datastar Go SDK (`datastar-go v1.2.2`) | Used in demo go.mod |
| Event-sourced CQRS + Datastar SSE | Fully working (Todo app with Create/Toggle/Delete/Update) |
| Multi-user real-time simulation | 10 bot goroutines broadcasting to all tabs |
| `PatchElements` + `PatchSignals` + `ReadSignals` | All used in handlers |
| SSE event stream with fan-out | Custom `Broadcaster` (mirrors `cqrshtmx.Broadcaster` pattern) |
| templ integration | **Not yet** (demo uses raw HTML strings) |
| Integration with `cqrshtmx.App` | **Not yet** (demo uses vanilla `net/http`) |
| Library-level Datastar adapter | **Not yet** |

The demo deliberately **does not import the `cqrshtmx` root module** — it reimplements the transport layer with the Datastar SDK directly. This was a proof that CQRS + Datastar works, but the integration into the library itself was never done.

---

## 5. The Architectural Decision: How to Integrate

The May 2026 status report (`docs/status/archive/2026-05-21_23-43_datastar-demo-multi-user-simulation.md`) identified three options:

| Option | Description | Verdict |
|---|---|---|
| A) Parallel response layer in root | Add Datastar response builders alongside HTMX ones | **Rejected** — bloats root module, forces Datastar SDK dep on all consumers |
| B) Abstract transport interface | Refactor `response.go` + `options.go` into a transport-agnostic interface | **Rejected** — massive refactor, risks breaking stable HTMX API |
| C) Separate library (`cqrs-datastar`) | Completely new repo | **Rejected** — fragments the ecosystem, duplicates CQRS wiring |

### Recommended: D) New Optional Submodule

**`github.com/larsartmann/cqrs-htmx/datastar/v4`** — a new independent Go module.

Rationale:
- **Follows the existing module pattern** (usermgmt, adminui, dashboardui, loginpage are all separate modules with `/v4` suffixes)
- **Zero coupling** — HTMX users never import Datastar code, the Datastar SDK dep stays optional
- **Shares the CQRS dispatch layer** — go-cqrs-lite is the common foundation, only the HTTP/response layer differs
- **Consumers can use both** — import `cqrs-htmx/v4` for HTMX endpoints AND `cqrs-htmx/datastar/v4` for Datastar endpoints in the same app
- **Mirrors the dependency direction** — root has zero imports of UI modules; datastar module would depend on root (for shared types) + datastar-go SDK

Module structure:
```
cqrs-htmx/
  datastar/                    # NEW MODULE
    go.mod                     # module github.com/larsartmann/cqrs-htmx/datastar/v4
    doc.go
    sse_adapter.go             # Wraps datastar.NewSSE with cqrs-htmx patterns
    signal_handler.go          # Signal-aware command/query decoders
    event_bridge.go            # Domain event -> Datastar SSE event mapper
    script_handler.go          # Embed + serve datastar.js (like HTMXScriptHandler)
    response.go                # Datastar-native response builder
    render.go                  # templ component rendering for Datastar SSE
    datastar/                  # Embedded datastar.js (like htmx.min.js)
      datastar.min.js
  examples/
    datastar-demo/             # EXISTING — upgrade to use the new module
```

---

## 6. Concrete Integration Components

### 6.1 Datastar Script Handler (embed + serve datastar.js)

Mirrors `HTMXScriptHandler()` / `HTMXScriptHandlerWith()`:

```go
// Embed datastar.js for self-hosting (no CDN dependency)
mux.Handle("GET /datastar.js", ds.DatastarScriptHandler())
```

Implementation mirrors `htmx_serve.go`: embed the JS, set ETag (`"datastar-%s"`), 1-year immutable cache, 304 on `If-None-Match`.

### 6.2 Signal-Aware Command/Query Decoders

Datastar automatically sends all signals as `{datastar: {...}}` with every request (query param for GET, JSON body for POST). The adapter provides decoders that extract these:

```go
// Instead of cqrshtmx.DecodeJSON, use signal-aware decoding:
mux.Handle("POST /todos", app.Command("CreateTodo",
    ds.DecodeSignals(func(s CreateTodoSignals) (command.Command, error) {
        return &createTodoCmd{
            aggID: id.NewStreamID(),
            cmdID: id.NewCommandID(),
            Title: s.Title, // extracted from Datastar signals
        }, nil
    }),
))
```

This wraps `datastar.ReadSignals(r, &signals)` and maps to the same `command.Command` interface.

### 6.3 Domain Event -> Datastar SSE Bridge

The highest-value component. Transforms domain events from the event bus into Datastar SSE events (`patch-elements` / `patch-signals`):

```go
// Wire once: domain events -> Datastar SSE for all connected clients
bridge := ds.NewEventBridge(ds.BridgeConfig{
    EventBus:    svc.EventBus,
    Broadcaster: cqrshtmx.NewBroadcaster(),
    Renderer:    ds.TemplateRenderer(todoTemplates), // templ components
    SignalMapper: ds.DefaultSignalMapper,            // auto-map event -> signals
})
bridge.Start()
defer bridge.Stop()
```

The bridge replaces the manual switch statement in `datastar-demo/domain_cqrs.go` with a declarative mapping:

```go
bridge.Map("TodoCreated", func(e event.Event) ds.Patch {
    return ds.PatchElements(renderTodoHTML(e), ds.WithModeAppend(), ds.WithSelectorID("todo-list"))
})
bridge.Map("TodoDeleted", func(e event.Event) ds.Patch {
    return ds.RemoveElement("#todo-" + e.StreamID().String())
})
bridge.Map("TodoToggled", func(e event.Event) ds.Patch {
    return ds.PatchElements(renderTodoHTML(e), ds.WithModeOuter())
})
```

### 6.4 Datastar-Native Response Builder

For non-SSE responses (command dispatch results), a Datastar-aware response builder:

```go
// Instead of cqrshtmx.NewResponse(w, r) (HTMX headers),
// use the Datastar response builder:
ds.NewResponse(w, r).
    PatchSignals(map[string]any{"notification": map[string]string{"level": "success", "message": "Created!"}}).
    PatchElements(renderTodoList(todos), ds.WithSelectorID("todo-list"), ds.WithModeInner()).
    Redirect("/todos").
    Apply()
```

This replaces HTMX's `HX-Trigger`, `HX-Redirect`, `HX-Push-URL` headers with Datastar SSE events.

### 6.5 SSE Stream Handler (Datastar Protocol)

A drop-in replacement for the manual SSE loop in `datastar-demo/handlers_routes.go`:

```go
mux.HandleFunc("GET /events", ds.SSEStreamHandler(broadcaster, sseStore))
```

This wraps the subscribe-replay-pump loop with Datastar's `datastar.NewSSE(w, r)` and automatically converts `cqrshtmx.SSEEvent` to Datastar SSE events.

---

## 7. Impact by Module

### 7.1 dashboardui — Highest Impact

| Current | With Datastar |
|---|---|
| HTMX polling every 10s for projection health | Real-time signal patches via SSE |
| Full partial re-render on each poll | Signal-only delta updates |
| `hx-trigger="every 10s"` on projection panel | `data-text="$projectionLag"` auto-updates |
| Manual event log rendering | `data-on:click="@get('/events')"` with auto-morph |

The dashboard already has an SSE stream (`dashboardui/sse.go`) — but it sends raw JSON that the frontend must process with custom JS. Datastar would turn those into `patch-signals` events, making the dashboard fully reactive with zero custom JavaScript.

### 7.2 adminui — High Impact

| Current | With Datastar |
|---|---|
| HTMX partial swaps for CRUD | Morph-based updates (preserves form focus, scroll position) |
| Server round-trip for filter/pagination | Signal-based client-side filtering |
| `HX-Trigger` toast notifications | `patch-signals` for notification state |
| idiomorph extension for morphing | Built-in morphing (no extension) |
| Complex `hx-target`/`hx-select`/`hx-swap` per interaction | `data-on:click="@post('/url')"` (1 attribute) |

### 7.3 loginpage — Medium Impact

| Current | With Datastar |
|---|---|
| HTMX form submission for WebAuthn | Signal-based form state |
| `hx-post` for each auth step | `@post()` with auto-retry |
| Manual loading indicators | `data-indicator` signals |

### 7.4 Root Module — No Change

The root module (`cqrs-htmx/v4`) stays HTMX-only. No changes to:
- `response.go` (HTMX headers)
- `htmx.go` (HX-* parsing)
- `htmx_serve.go` (embedded htmx.js)
- `handler.go` / `options_*.go` (handler builder pipeline)
- `sse_*.go` / `ws_*.go` (transport-agnostic SSE/WS building blocks)

The datastar module **depends on** root (for shared types, `Broadcaster`, `SSEStream`, error mapping) but root never depends on datastar.

### 7.5 sync/ (Offline Sync) — Potential Simplification

The sync-worker.js/sync-client.js could remain as the durable IndexedDB layer, but the retry/reconnection/DOM-indicator logic (currently hand-rolled) could leverage Datastar's built-in primitives:

| sync-worker.js feature | Datastar equivalent |
|---|---|
| Custom EventSource for ACK | `@post()` with `retry: 'auto'` |
| Manual `data-sync-state` DOM manipulation | `data-indicator:syncing` signal |
| Custom retry with backoff | `retryInterval`, `retryScaler`, `retryMaxWait` options |
| `data-sync-status` indicator management | `data-text` bound to `$syncStatus` signal |

---

## 8. The Datastar Go SDK API (What We'd Build On)

The `starfederation/datastar-go` SDK is mature and well-designed. Key APIs the adapter would wrap:

```go
// Create SSE writer (upgrades http.ResponseWriter)
sse := datastar.NewSSE(w, r)

// Patch DOM elements (morph by default)
sse.PatchElements(htmlString, datastar.WithSelectorID("todo-list"), datastar.WithModeInner())
sse.PatchElementTempl(templComponent) // direct templ support!
sse.RemoveElement("#todo-123")

// Patch reactive signals
sse.MarshalAndPatchSignals(map[string]any{
    "notification": map[string]string{"level": "success", "message": "Created!"},
    "todoCount":    42,
})

// Execute JavaScript from backend
sse.ExecuteScript("console.log('hello')")
sse.Redirect("/dashboard")

// Read client signals from request
var signals struct{ Title string `json:"title"` }
datastar.ReadSignals(r, &signals)
```

The SDK has **native templ support** (`PatchElementTempl`) — critical because cqrs-htmx uses templ throughout adminui, dashboardui, and loginpage.

---

## 9. Prioritized Action Plan

### Phase 1: Foundation (the adapter module) — HIGH IMPACT, MEDIUM EFFORT

1. **Create `datastar/` submodule** with `go.mod` depending on `datastar-go` + `cqrs-htmx/v4`
2. **`DatastarScriptHandler()`** — embed + serve datastar.js (mirror `HTMXScriptHandler`)
3. **`DecodeSignals[Q]()`** — signal-aware command/query decoders (mirror `DecodeJSONTyped`)
4. **`NewResponse(w, r)`** — Datastar-native response builder (mirror `cqrshtmx.NewResponse`)
5. **`SSEStreamHandler(broadcaster, store)`** — Datastar SSE stream with fan-out + replay
6. **Tests** — full coverage gate matching the project standard

### Phase 2: Event Bridge — HIGH IMPACT, MEDIUM EFFORT

7. **`NewEventBridge(config)`** — domain event -> Datastar SSE mapper
8. **Declarative event mapping** — `bridge.Map(eventType, patchFunc)` API
9. **templ renderer integration** — `bridge.Renderer` accepts templ components
10. **Reconnection replay** — bridge integrates with `JournalSSEStore` for backfill

### Phase 3: Upgrade the Demo — MEDIUM IMPACT, LOW EFFORT

11. **Rewrite `examples/datastar-demo/` to use the new module** — replace vanilla handlers with `app.Command` + `ds.DecodeSignals`
12. **Add templ components** — replace raw HTML strings in `renderTodo()`
13. **Demonstrate signal-based filtering** — show client-side search without round-trips

### Phase 4: Module Integration — HIGH IMPACT, HIGH EFFORT

14. **dashboardui Datastar variant** — replace HTMX polling with signal-driven real-time updates
15. **adminui Datastar variant** — optional Datastar rendering mode (templ components stay the same, only the transport changes)
16. **Offline sync evaluation** — prototype Datastar-native retry as alternative to sync-worker.js

### Phase 5: Documentation — LOW EFFORT

17. **Guide: "Datastar Integration"** in `docs/guides/`
18. **ADR: "Datastar as Optional Frontend"** in `docs/adr/`
19. **README section** — "HTMX or Datastar?" decision guide

---

## 10. Risk Assessment

| Risk | Likelihood | Mitigation |
|---|---|---|
| Datastar SDK API changes (pre-2.0) | Medium | Pin SDK version in go.mod; adapter isolates consumers from SDK churn |
| HTMX users accidentally pulled into Datastar deps | None | Separate module — Go module system prevents transitive deps |
| Maintenance burden of two frontend adapters | Low | Shared CQRS dispatch layer; only the HTTP/response layer differs |
| Datastar adoption stalls (project abandoned) | Low | Nonprofit-backed, active community, 14 SDKs; module is isolated and removable |
| Skill gap for contributors unfamiliar with Datastar | Medium | Comprehensive guide + working demo + the existing datastar-demo as reference |

---

## 11. What NOT to Do

- **Do NOT refactor `response.go` or `handler.go`** — the HTMX API is stable, well-tested (93.7% coverage), and used by all consumers. The Datastar adapter is additive.
- **Do NOT rename the project** — "cqrs-htmx" describes the CQRS + hypermedia philosophy. Datastar IS hypermedia. The name still fits.
- **Do NOT make Datastar the default** — HTMX remains the primary path. Datastar is an opt-in for consumers who need richer frontend reactivity.
- **Do NOT add Datastar to the root module's go.mod** — it must stay an optional dependency in a separate module.
- **Do NOT build a transport abstraction layer** — option B from the May 2026 report. The cost/benefit is terrible: massive refactor of stable code to support a hypothetical third transport that may never come.

---

## 12. Comparison: HTMX vs Datastar by Use Case

| Use Case | Better Choice | Why |
|---|---|---|
| Simple CRUD forms (create/edit/delete) | **Either** | Both handle this well. HTMX is simpler for pure request-response. |
| Real-time dashboards (projection health, metrics) | **Datastar** | Signal-based reactivity eliminates polling. Structured SSE protocol. |
| Multi-user collaboration (live updates across tabs) | **Datastar** | CQRS SSE model is built for this. Fan-out + morph = instant sync. |
| Admin panels with filtering/search/pagination | **Datastar** | Client-side signal filtering eliminates round-trips. Morph preserves form state. |
| Simple marketing pages / static content | **HTMX** | No reactivity needed. HTMX is lighter for progressive enhancement. |
| Offline-first apps (offline command queue) | **Datastar** | Built-in retry, signal-based indicators. But sync-worker.js remains for durable IndexedDB. |
| Complex form wizards (multi-step, conditional fields) | **Datastar** | Computed signals, two-way binding, effect-based field visibility. |
| File uploads | **Datastar** | Base64 into signals, no form needed. But HTMX's `hx-encoding` also works. |

---

## 13. The Bottom Line

cqrs-htmx already implements Datastar's philosophy — it just uses HTMX as the frontend layer. Adding a Datastar adapter module would:

1. **Close the frontend state gap** — signals replace the Alpine.js/HTMX-extensions stack
2. **Eliminate dashboard polling** — real-time signal patches via SSE
3. **Structure the SSE protocol** — declarative event-to-patch mapping replaces manual HTML generation
4. **Simplify offline sync** — leverage Datastar's built-in retry/indicator primitives
5. **Give consumers a choice** — HTMX for simple apps, Datastar for real-time/reactive apps, or both

The investment is proportional: a new ~6-file module, not a refactor of existing code. The existing `datastar-demo` proves the pattern works. The Datastar Go SDK is mature with native templ support. The risk is low because the module is fully isolated.

This is the single highest-leverage frontend improvement available to the project.
