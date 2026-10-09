# Frontend Sync Protocol — Offline Reads, Batched Command Push, and Conflict Outcomes

> **Decision record:** [ADR-0056](../adr/0056-frontend-sync-protocol.md) · **Series:** ADR-0023 (sync commands, not events) → ADR-0040 (IndexedDB queue) → ADR-0042 (root extraction) → this guide
> **Research trail:** [2026-10-09 LiveStore/Axon synthesis](../research/2026-10-09_frontend-sync-protocol-synthesis.md)

This guide wires the full offline loop: your UI queues **commands** while offline, flushes them as **one batched push** on reconnect, and caches **events** locally so reads work offline — with permissions enforced server-side on every pull.

## The three laws

1. **Clients sync intent, never facts.** Push carries commands; the server re-decides everything (authz, invariants, conflicts). There is no client-side decider and no client-appended event.
2. **Clients may cache facts for reads.** The pull endpoint is the explicit opt-in surface for event payloads; the SSE envelope stays metadata-only.
3. **Conflicts resolve by server total order.** The client pulls before pushing; every command answers with a classified outcome; rejected commands surface honestly and leave the queue.

## Server wiring

Two endpoints, one worker script, one client script:

```go
mux := http.NewServeMux()

// Read half: cursor-based, permission-filtered event pull WITH payloads.
mux.Handle("GET /sync/pull", cqrshtmx.SyncPullHandler(journal,
    cqrshtmx.WithSyncPullBackendID("prod-2026-10"),
    cqrshtmx.WithSyncPullFilter(myVisibilityFilter),
))

// Write half: batched push of queued commands. Needs NO new registration —
// it replays the pipelines your app.Command()/CommandTyped() calls registered.
mux.Handle("POST /sync/push", app.SyncPushHandler())

// Embedded assets (ETag + immutable caching).
mux.Handle("GET /sync-worker.js", cqrshtmx.SyncWorkerHandler())
mux.Handle("GET /sync-client.js", cqrshtmx.SyncClientHandler())
```

`journal` is any `event.Journal` — `event.SeekableJournal` implementations (SQL stores) get efficient position-based reads; others fall back to `ReadAll` + in-memory seeking.

## HTML wiring

```html
<body data-sync-pull-url="/sync/pull" data-sync-push-url="/sync/push">
  <div data-sync-status="idle">Synced</div>          <!-- indicator: idle|ok|pending -->
  <div data-sync-target>
    <form hx-post="/api/items" hx-target="#item-list" hx-swap="beforeend"
          data-sync-command-type="CreateItem">        <!-- opts into batch push -->
      <input name="name" required>
      <button type="submit">Add</button>
    </form>
  </div>
  <script src="/sync-client.js"></script>
</body>
```

- `data-sync-pull-url` / `data-sync-push-url` on `<body>` override the `/sync/pull` + `/sync/push` defaults.
- `data-sync-command-type` on (or above) the issuing element stamps the envelope's command type, so the offline queue flushes it via **one batched** `POST /sync/push` instead of per-URL HTMX replay. Absent = the classic per-URL replay path (backward compatible).
- `data-sync-status` reflects lifecycle (`idle`/`ok`/`pending`); per-command rows get `data-sync-state` (`pending`/`confirmed`/`rejected`) and `data-command-id` for ACK/SSE correlation.

### Pull wire contract

`GET /sync/pull?after=<eventID>&limit=<n>` →

```json
{
  "backendId": "prod-2026-10",
  "events": [
    {
      "eventId": "01J…", "type": "ItemCreated", "streamType": "item",
      "streamId": "01J…", "version": 1, "occurredAt": "2026-10-09T02:17:55Z",
      "payloadEncoding": "json", "payload": { "name": "grocery-list" }
    }
  ],
  "nextCursor": "01J…",
  "hasMore": false
}
```

- **Cursor:** the last event ID seen. Empty = bootstrap. Page with `nextCursor` while `hasMore`. An empty page returns an empty `nextCursor` — keep the previous cursor (ULIDs order globally, so compacted journals stay navigable).
- **Delivery encoding is honest:** exactly one of `payload` (inline verbatim JSON, `payloadEncoding: "json"`) or `payloadB64` (base64 bytes under their codec name, e.g. `"cbor"`) is set. A json-stamped event whose bytes are not valid JSON reports `"opaque"` (`cqrshtmx.SyncPayloadEncodingOpaque`) — the wire never promises JSON the bytes cannot keep.
- **Caching:** every response carries an FNV-1a ETag over the position fields and `Cache-Control: private, no-cache` — a conditional re-poll at an unchanged cursor is a cheap 304.
- **Backend reset:** persist `backendId`; when it changes, wipe the local event cache and re-bootstrap (the shipped client does this automatically and broadcasts `sync:reset`).

### Push wire contract

`POST /sync/push` with `{commands: [{commandId, type, body, contentType}]}` → HTTP 200 with per-command outcomes:

```json
{
  "results": [
    { "commandId": "offline-1", "status": "confirmed" },
    { "commandId": "offline-2", "status": "rejected",
      "error": { "message": "note already exists: conflict-dup", "code": "note.conflict", "family": "conflict" } }
  ]
}
```

- `body` is the exact body your single-command endpoint's decoder expects (JSON or form-encoded — set `contentType` to match).
- `error.family` is the retry contract: `rejection`/`conflict` = permanent (surface + drop), `transient` = retryable (re-queue), `corruption`/`infrastructure` = permanent-fail. 5xx details are redacted server-side (`SafeDetail`).
- `commandId` is stamped as `X-Command-Id` on the replayed pipeline, so `BroadcastOnAck`-style hooks fire per command and idempotency middleware dedups retries.
- Batches cap at `MaxSyncPushBatch` (100); failures are isolated per command — one rejected command never aborts the batch.

## Permissions (the pull filter)

`WithSyncPullFilter` runs **per request** with the live `*http.Request` — session-derived authorization composes naturally. Filtering is fail-closed, and invisible events still advance the cursor (they are never re-read):

```go
func myVisibilityFilter(r *http.Request, evt event.Event) bool {
    user, ok := usermgmt.UserFromContext(r.Context())
    if !ok {
        return false // fail closed: no session, no payloads
    }
    return evt.StreamType() == "item" || belongsToTenant(evt, user.TenantID())
}
```

A Casbin recipe: assert `enforcer.Enforce(user.ID().Get(), evt.StreamType(), "read")` per event. Remember `nil` filter = **every event visible with payloads** — only acceptable for non-sensitive feeds. Gate the endpoint behind your session middleware like any other authenticated route.

## Client API

The sync client exposes a small public surface on `window`:

```js
window.cqrsSync.version   // "1.5.0" — matches cqrshtmx.SyncVersion()
await window.cqrsSync.getEvents() // cached event feed (works offline; worker-owned IndexedDB)
await window.cqrsSync.pull()      // force a catch-up pull; resolves when caught up
```

Events and resets also arrive as DOM events: `cqrshtmx:sync-events` (new cached events; also triggered through HTMX as `cqrshtmx:sync-events` on `document.body`) and on cache reset. Reuse in your own JS:

```js
document.addEventListener("cqrshtmx:sync-events", (e) => {
  renderFeedFrom(e.detail); // detail carries the newly cached events
});
```

The SharedWorker stays a **coordinator, not a proxy** (ADR-0029): tabs own their fetches and SSE connections; the worker owns persistence (command queue, event cache, cursor/backendId meta) and cross-tab broadcasts. Offline reads ask the worker (`get-events`); the client does not project events — folding cached events into view state is consumer territory (see ROADMAP).

## Security notes

- **Mount behind your middleware**: session (outside CSRF), CSRF (protects the outer POST once — inner envelopes are not independently reachable), security headers as usual.
- **Payload exposure is deliberate**: the pull endpoint is the one surface that emits event payloads. If your events carry PII, the filter is mandatory, not optional.
- **5xx redaction**: push errors from server failures are `SafeDetail`-redacted — no internals leak to the client.

## What's NOT included (by design)

Client-side projections (a fold engine over the cached feed), ServiceWorker-served offline HTML, and WASM client deciders are ROADMAP items — see ADR-0056's non-goals. The shipped client caches and exposes the feed; it never re-decides.
