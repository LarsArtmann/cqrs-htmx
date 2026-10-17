# sync-demo — the frontend sync protocol, runnable

A minimal but complete consumer of cqrs-htmx's frontend sync protocol
(ADR-0056): commands queue while you are offline, flush as one batched push
when you reconnect, and the event feed reads from the local cache either way.
This is the executable version of the snippets in
[`docs/guides/frontend-sync.md`](../../docs/guides/frontend-sync.md).

```bash
GOEXPERIMENT=jsonv2 go run .   # open http://localhost:8098
```

## What it wires

| Endpoint                                     | Role                                                                          |
| -------------------------------------------- | ----------------------------------------------------------------------------- |
| `POST /api/notes`                            | The offline-queueable command endpoint (`data-sync-command-type` on its form) |
| `POST /sync/push`                            | `app.SyncPushHandler()` — batched replay with per-command outcomes            |
| `GET /sync/pull`                             | `cqrshtmx.SyncPullHandler(store, WithSyncPullBackendID, WithSyncPullFilter)`  |
| `GET /events`                                | SSE ACK feed (the sync indicator's confirmed/rejected states)                 |
| `GET /openapi.json`                          | `AddSyncOpenAPI` — the served wire contract for both sync endpoints           |
| `GET /sync-worker.js`, `GET /sync-client.js` | The embedded sync assets                                                      |

The demo domain is deliberately tiny: notes with unique names (duplicates are
rejected — the conflict contract), where "confidential" notes ride the
`SecretNote` stream type so the pull filter can hide them without decoding
payloads. The filter checks a `demo_role` cookie: without `admin`, confidential
events are invisible to pulls (fail-closed, and the cursor still advances past
them — they are never re-read).

## The offline walkthrough (2 minutes)

1. **Open the app.** The indicator flips to _Synced_; the cached feed lists the
   journaled events via `window.cqrsSync.getEvents()`.
2. **Go offline** — DevTools → Network → Offline (or Playwright
   `context.setOffline(true)`). The indicator flips to _offline_.
3. **Add two notes.** The forms queue (the sync client marks them queued, not
   failed); the "server truth" panel stays behind because it needs the network.
4. **Go back online.** The queued commands flush as ONE batched
   `POST /sync/push` round-trip; both notes appear in the server list.
5. **Duplicate a name.** The push outcome is `rejected` with family
   `rejection` — the form shows the rejected state with a retry button, the
   command is dropped from the queue (permanent rejections never retry).
6. **Tick "admin" and reload.** Confidential notes appear in the cached feed —
   the pull now passes them through the filter.

## Where the truth lives

- Events: the in-memory journal (`storage/memory`) is the source of truth; the
  browser only caches what `/sync/pull` exposes.
- Commands: the browser queues _intent_; the server re-decides every command
  (that is why duplicates reject server-side, no matter what the client
  believed).
- Backend rebuilds: `WithSyncPullBackendID("sync-demo-v1")` — change that
  string and every client wipes its cache on the next pull (`sync:reset`),
  while keeping its queued commands.

## Tests

`go test ./...` runs the protocol round-trips in-process: batch push with a
poisoned envelope, duplicate-name rejection, filter visibility with and
without the admin cookie, the ETag/304 re-poll, the embedded assets, and the
served OpenAPI fragment.
