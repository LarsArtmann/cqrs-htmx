# v4 → v5 Migration Guide

> **Status:** Complete — WebSocket transport dropped; library is SSE-only (see [ADR-0046](../adr/0046-drop-websocket-sse-only.md))

v5 removes the entire WebSocket surface. SSE covers every realtime use case
with strictly better operational properties (plain HTTP, native auth/CSRF,
proxy-friendly, auto-reconnect via `Last-Event-ID`). This guide has a
per-symbol recipe for every removed export.

## TL;DR — What changed

| Aspect             | v4                                 | v5                                             |
| ------------------ | ---------------------------------- | ---------------------------------------------- |
| Realtime transport | SSE **and** WebSocket              | SSE only                                       |
| Fan-out            | `Broadcaster` + `WSBroadcaster`    | `Broadcaster` only                             |
| Client→server      | HTTP dispatch + WS dispatch bridge | HTTP dispatch only (`app.Command`/`app.Query`) |
| OOB HTML           | `OOBHTML` + `WSOOBHTML`            | `OOBHTML` only                                 |
| Removed symbols    | —                                  | 14                                             |

If you only used SSE in v4, there are **zero breaking changes** for you. This
guide is only for consumers who used the `WS*` symbols.

---

## Per-Symbol Recipes

### 1. `WSBroadcaster` → `Broadcaster`

The WS broadcaster fanned out `string` messages. The SSE broadcaster fans out
`sse.Event` values. The lifecycle (`Subscribe`/`Unsubscribe`/`Broadcast`/`Close`)
is identical.

**Before (v4):**

```go
wb := cqrshtmx.NewWSBroadcaster()
ch := wb.Subscribe() // chan string
defer wb.Unsubscribe(ch)

// broadcast a pre-encoded message string:
wb.Broadcast(cqrshtmx.WSOOBHTML("tasks", "<ul>...</ul>"))
```

**After (v5):**

```go
b := cqrshtmx.NewBroadcaster()
ch := b.Subscribe() // chan sse.Event
defer b.Unsubscribe(ch)

// broadcast an SSE event carrying the HTML as data.
// Import "github.com/larsartmann/go-sse" as sse:
b.Broadcast(sse.Event{Event: "tasks", Data: cqrshtmx.OOBHTML("tasks", "<ul>...</ul>")})
```

### 2. `WSOOBHTML` → `OOBHTML`

`OOBHTML` has the exact same signature and output — `WSOOBHTML` was already a
delegate to it since v4.3.0.

```diff
- html := cqrshtmx.WSOOBHTML("tasks", "<ul>...</ul>")
+ html := cqrshtmx.OOBHTML("tasks", "<ul>...</ul>")
```

### 3. `ParseWSMessage` / `ParseWSMessageInto` → `json.Unmarshal`

The HTMX WebSocket extension sent a JSON blob with a `HEADERS` section
separated from body fields. That was an HTMX-extension quirk, not a domain
concern. With SSE-only, client→server mutations are plain HTTP POSTs, so you
decode JSON directly.

**Before (v4):**

```go
msg, headers, err := cqrshtmx.ParseWSMessageInto[CreateTaskInput](data)
// or untyped:
wsMsg, err := cqrshtmx.ParseWSMessage(data)
```

**After (v5):** the client POSTs JSON to an HTTP endpoint; decode it with
`DecodeJSON` (or `json.Unmarshal` in a custom decoder):

```go
app.Command("POST /tasks",
    cqrshtmx.DecodeJSON(func(in CreateTaskInput) (command.Command, error) {
        return command.New("CreateTask", in)
    }),
    cqrshtmx.NotifySuccess("Task created"),
)
```

If you still need raw JSON decoding outside a handler, use `json.Unmarshal`:

```go
var input CreateTaskInput
if err := json.Unmarshal(data, &input); err != nil { /* ... */ }
```

### 4. `WriteWSMessage` / `WriteWSMessageInto` → `json.Marshal`

**Before (v4):**

```go
cqrshtmx.WriteWSMessage(w, wsMsg)
cqrshtmx.WriteWSMessageInto[CreateTaskInput](w, body, headers)
```

**After (v5):** marshal the struct yourself and send it as an SSE event's
`Data` field (for server→client), or write it as an HTTP response body:

```go
out, _ := json.Marshal(body)
broadcaster.Broadcast(sse.Event{Event: "taskCreated", Data: string(out)})
```

### 5. `DispatchWSCommand` / `DispatchWSQuery` → `app.Command` / `app.Query`

The WS dispatch bridge ran hooks + timeout but **deliberately omitted auth,
CSRF, and response-writing** (WS was authenticated at upgrade time). The HTTP
dispatch pipeline already does all of that — and it's what HTMX drives over
SSE for server→client updates.

**Before (v4):**

```go
decoder := cqrshtmx.DecodeWSJSON(func(t CreateTaskInput) (command.Command, error) {
    return command.New("CreateTask", t)
})
// in your WS read loop:
err := app.DispatchWSCommand(r, "CreateTask", decoder, data)
```

**After (v5):** register an HTTP command handler and let the client POST to it
(HTMX `hx-post="/tasks"` works out of the box):

```go
app.Command("POST /tasks",
    cqrshtmx.DecodeJSON(func(in CreateTaskInput) (command.Command, error) {
        return command.New("CreateTask", in)
    }),
    cqrshtmx.NotifySuccess("Task created"),
)
```

For queries, `app.Query("GET /tasks", ...)` replaces `DispatchWSQuery`.

### 6. `DecodeWSJSON` / `DecodeWSJSONQuery` → `DecodeJSON` / `DecodeJSONQuery`

The WS decoders operated on raw bytes; the HTTP decoders operate on
`*http.Request`. Since mutations now go through HTTP endpoints, use the HTTP
decoders.

```diff
- decoder := cqrshtmx.DecodeWSJSON(func(in CreateTaskInput) (command.Command, error) { ... })
+ cqrshtmx.DecodeJSON(func(in CreateTaskInput) (command.Command, error) { ... })
```

### 7. `BroadcastOnSuccessWS*` / `BroadcastOnErrorWS*` / `BroadcastOnAckWS*` → SSE equivalents

These hook factories broadcast over the WS broadcaster. The SSE broadcaster has
the exact same hook family (with `sse.Event` payloads instead of `string`):

| Removed (v4)                   | Replacement (v5)                      |
| ------------------------------ | ------------------------------------- |
| `BroadcastOnSuccessWS(msg)`    | `BroadcastOnSuccess(eventName, data)` |
| `BroadcastOnSuccessWSFunc(fn)` | `BroadcastOnSuccessFunc(fn)`          |
| `BroadcastOnErrorWS()`         | `BroadcastOnError(eventName)`         |
| `BroadcastOnErrorWSFunc(fn)`   | `BroadcastOnErrorFunc(fn)`            |
| `BroadcastOnAckWS()`           | `BroadcastOnAck()`                    |
| `BroadcastOnAckWSFunc(fn)`     | `BroadcastOnAckFunc(fn)`              |

Wire them the same way — pass to `cqrshtmx.Config.AfterDispatch`.

### 8. `HTMXExtWS` / `extensions/ws.min.js` → `hx-ext="sse"`

Remove the WS extension script tag and the `hx-ext="ws"` attribute. Switch any
WS-based markup to the SSE extension.

**Before (v4):**

```html
<script src="{{ .WSExtURL }}"></script>
<div hx-ext="ws" ws-connect="/ws">
	<div id="tasks" ws-swap="innerHTML"></div>
</div>
```

**After (v5):**

```html
<div hx-ext="sse" sse-connect="/events">
	<div id="tasks" sse-swap="tasks"></div>
</div>
```

The SSE extension is served via `cqrshtmx.HTMXScriptHandler()` (the standard
HTMX bundle now ships SSE as a core extension).

---

## The SSE Pattern (reference)

If you're new to the SSE broadcaster, here's the canonical setup
(`cqrshtmx.NewBroadcaster`/`Broadcaster` stay; `sse.Event`/`sse.NewStream`
come from `github.com/larsartmann/go-sse`, imported as `sse`):

```go
broadcaster := cqrshtmx.NewBroadcaster()

app := cqrshtmx.MustNew(cqrshtmx.Config{
    Commands:     cmdDisp,
    AfterDispatch: broadcaster.BroadcastOnSuccess("itemCreated", data),
})

mux.HandleFunc("GET /events", func(w http.ResponseWriter, r *http.Request) {
    stream := sse.NewStream(w, r)
    defer stream.Close()
    ch := broadcaster.Subscribe()
    defer broadcaster.Unsubscribe(ch)
    for {
        select {
        case <-stream.Context().Done():
            return
        case evt, ok := <-ch:
            if !ok || stream.Send(evt) != nil {
                return
            }
        }
    }
})
```

See `.agents/skills/cqrs-htmx/references/realtime.md` for the full SSE
reference (broadcaster, ACK protocol, idempotency, reconnection/replay,
heartbeat, event filtering).

---

## The `cqrs-htmx-upgrade` codemod

Most of this migration is mechanical, and the mechanical part is automated.
The codemod rewrites consumer code off the deprecated APIs:

| Rule | Rewrites                                                                         | Notes                                                                                              |
| ---- | -------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------- |
| R1   | root httputil re-exports (CSRF, rate limiting, Server-Timing, security headers)  | → the `github.com/larsartmann/httputil` package                                                    |
| R2   | root go-sse re-export shims (SSE event/store types)                              | → `github.com/larsartmann/go-sse`                                                                  |
| R3   | raw-broadcaster constructors (`NewBroadcasterFromRaw` → `NewBroadcasterFromHub`) | remaining `.Raw()` call sites are REPORTED for manual migration (type-ambiguous to rewrite safely) |
| R9   | usermgmt deprecated type aliases                                                 | → `identity-model`                                                                                 |
| R10  | usermgmt deprecated var/const re-exports                                         | → `identity-model`                                                                                 |

The R1/R2/R9/R10 symbol tables are **not hardcoded**: they are derived at run
time from the alias declarations of the cqrs-htmx version present in _your_
module graph (`go list -m`), so the codemod can never drift from whatever
version you have installed.

```bash
# Dry run (default): a report of every planned edit, nothing written.
go run github.com/larsartmann/cqrs-htmx/v4/cmd/cqrs-htmx-upgrade@latest ./...

# Write the rewrites:
go run github.com/larsartmann/cqrs-htmx/v4/cmd/cqrs-htmx-upgrade@latest -apply ./...

# Print every planned edit verbosely:
go run github.com/larsartmann/cqrs-htmx/v4/cmd/cqrs-htmx-upgrade@latest -v ./...
```

Paths default to `.`. Always run the dry run first and review the report;
non-mechanical rewrites are never guessed — they are reported for manual
migration (per-symbol recipes above).

---

## Why WebSocket Was Removed

The bi-directional capability WebSocket offered was never exercised by any
cqrs-htmx consumer pattern — commands and queries already flow through the
HTTP pipeline with full auth, CSRF, and content negotiation. SSE handles
server→client updates with native HTTP semantics, built-in reconnection, and
zero proxy/CDN friction. Maintaining a second transport that offered no real
capability gain was pure surface-area cost (~1,200 LOC + a bundled JS asset).

See [ADR-0046](../adr/0046-drop-websocket-sse-only.md) for the full rationale.

---

## Rollout plan (Track A / Track B, master plan S54)

Two independent tracks; a consumer may run either or both. Neither requires
touching stored data.

### Track A — code migration (the codemod)

1. Bump module requirements to v5; run the codemod **dry run** (report only).
2. Review the report: every `Raw()`/non-mechanical site is listed for the
   per-symbol recipes above.
3. `-apply`, build, run the consumer's test suite. The codemod refuses to
   write unparseable output and rolls back on red (edit + build + vet per file).
4. Anything it cannot rewrite mechanically lands as `// v5-upgrade: TODO`
   markers — grep them; they are the manual residue list.

### Track B — journal backward compatibility

v5 readers must fold v4 journals unchanged:

- All 25 identity event payload types decode from their v4 wire bytes
  (21-event v4.13.0-era decode goldens + the S32 additions; schema-version
  upcasters via `identity-model/upcaster.go` when the first v2 schema lands).
- Fold-equivalence harness: fold a recorded v4 journal with v4 and v5 and
  assert identical `UserState`/`MembershipState`/`TenantState`/`BotState`.
- Actor/causation/correlation metadata contract unchanged (the enrichment
  chain is version-neutral).

### Compatibility matrix

| Consumer state | Path | Rollback |
| --- | --- | --- |
| v4.14+ code, v4 journals | Track A only | `git revert` the codemod commit — no data changes |
| pre-v4.14 journals | none needed to READ under v5 (Track B); Track A for code | same |
| custom `WebAuthnProvider` | add the `([]byte, error)` `FinishLogin` return (v4.14+ already; see `docs/guides/provider-implementation.md`) | n/a — already shipped as minor |
| WS-heavy consumer | per-symbol recipes §1–8 first, then Track A | revert; recipes are additive |

### Rollback

The migration is code-only: reverting the codemod commit (and the require
bump) restores v4 behavior exactly. There is no journal rewrite, no store
migration, and no config format change to undo.
