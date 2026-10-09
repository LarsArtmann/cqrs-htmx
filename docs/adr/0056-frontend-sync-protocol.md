# ADR 0056: Frontend Sync Protocol — Permission-Aware Event Pull + Batched Command Push

**Status:** Accepted
**Date:** 2026-10-09
**Related:** [ADR 0023](0023-command-sync.md) (sync commands, not events), [ADR 0024](0024-honest-ui.md), [ADR 0027](0027-decide-stays-on-server.md), [ADR 0029](0029-sharedworker-phase2a.md), [ADR 0040](0040-phase2b-indexeddb-persistence.md), [ADR 0042](0042-offline-sync-extraction-to-root.md), [research synthesis](../research/2026-10-09_frontend-sync-protocol-synthesis.md) (LiveStore/Axon mapping)

## Context

Episode 1 of the offline-sync series (ADR-0023 → 0042) shipped the **write** half: an IndexedDB-persisted offline command queue with honest pending→confirmed/rejected provenance. Three gaps remain for a genuine "CQRS in the frontend" capability, inspired by LiveStore's local-first loop and Axon's outcome semantics:

1. **No offline reads.** The queue is write-only; the UI is blank offline.
2. **No payload pull.** The SSE envelope is metadata-only (a deliberate safety default), so clients cannot project events locally even when permitted to see them.
3. **No batched push, cursor catch-up, or backend-reset detection.** One HTTP round-trip per queued command; no durable client cursor; a rebuilt backend confuses stale caches.

LiveStore solves all of this by syncing **events** both ways and rebasing client-side — which ADR-0023 already rejected as the time-machine paradox: facts cannot be re-decided server-side, so invariants and authz would only run on the client.

## Decision

Add a **read-only event pull** and a **batched command push** to the root module, plus client support in the sync assets (v1.5.0). Three laws govern the protocol:

1. **Clients sync intent, never facts** (ADR-0023 unchanged). Push carries commands; the server re-decides.
2. **Clients may cache facts for reads.** Pull is the explicit opt-in surface for event payloads, permission-filtered per request.
3. **Conflicts resolve by server total order.** Pull-before-push; every command answers with a classified outcome; the client rebases by re-pulling.

### `cqrshtmx.SyncPullHandler(journal, opts...)`

`GET ?after=<eventID>&limit=<n>` → `{backendId, events: [...], nextCursor, hasMore}`.

- Cursor = last-read event ID (empty = bootstrap from the beginning; page via `nextCursor`/`hasMore`).
- Events carry full metadata **plus payload** — raw JSON when the event encoding is JSON, base64 + `payloadEncoding` otherwise.
- `WithSyncPullFilter(func(r *http.Request, evt event.Event) bool)` — permission seam (wire Casbin per request); excluded events are invisible. Fail-closed doctrine inherited from `WithSSEFilter`.
- `WithSyncPullBackendID(id)` — LiveStore-style reset detection; the client wipes its cache on mismatch.
- `WithSyncPullLimit(n)` — batch clamp (default 500, max 1000).
- Cache-aware: FNV-1a ETag over `(backendId, nextCursor, hasMore)`; `If-None-Match` → 304; `Cache-Control: private, no-cache`.
- Read path uses `event.SeekableJournal.ReadFrom` when available, `ReadAll` + in-memory seek otherwise (same shape as `transport.JournalSSEStore`).

### `(app *App) SyncPushHandler()`

`POST {commands: [{commandId, type, body, contentType}]}` → `{results: [{commandId, status, error?}]}` (HTTP 200 always; outcomes are per-command).

- The App retains each `Command()`/`CommandTyped()` registration (type → decoder + authz + guards) in an internal registry — batch push composes with **zero new per-command registration**.
- Each envelope replays the endpoint pipeline (authz → decode → requestGuard → context enrichment → dispatch) against a synthesized request carrying the envelope body and the outer request's context and headers. Outcome, not state, is returned (Axon CommandGateway semantics).
- Outcome contract: `confirmed`; `rejected` with `error.family` — `Rejection`/`Conflict` = permanent (client drops the queue entry, surfaces honestly); `Transient` = retryable (re-queue); `Corruption`/`Infrastructure` = permanent-failed.
- The envelope's `commandId` is stamped as `X-Command-Id` on the synthesized request, so `BroadcastOnAck`-style AfterDispatch hooks fire per command and consumer idempotency middleware dedups retries.
- CSRF is validated once on the outer POST by the surrounding middleware; inner envelopes are not independently reachable and do not re-validate.
- Unknown command types in a batch answer per-command `rejected` (batch semantics isolate failures).

### Sync client v1.5.0 (worker stays a coordinator, not a proxy — ADR-0029 unchanged)

- **Pull lives in the tab**: on load/reconnect → paged pull from the cached cursor → hand events to the worker (`cache-events`) → worker persists them (IndexedDB `events` store + cursor/backendId `meta`) and broadcasts `sync:events` → **then** the queue flushes (pull-before-push).
- **Batch push**: queued envelopes gain an optional `commandType` (captured from the issuing element's `data-sync-command-type`); typed commands flush as ONE `POST /sync/push` whose per-command outcomes drive queue deletion/re-queue. Untyped commands keep the per-URL replay (backward compatible).
- **Offline reads**: tabs query the worker (`get-events`) for the cached feed.
- **Reset**: `backendId` mismatch clears the events store + cursor and broadcasts `sync:reset`.

## Consequences

- **Positive:** offline reads (cached event feed), batched reconnect (1 round-trip for N commands), durable cursor catch-up, permission-aware payload exposure as an explicit opt-in, backend-reset detection, and a classified-outcome conflict contract — the LiveStore loop with authority kept server-side.
- **Positive:** zero new registration burden — pull needs only a journal, push needs only existing `app.Command` registrations.
- **Positive:** the SSE metadata envelope stays the safe default; payload exposure is a separate, deliberate endpoint.
- **Negative:** permission filtering happens post-read (a client permitted to see 1% of events pages through invisible ones); acceptable at v1, noted for a store-level filtered read later.
- **Negative:** client-side projections of pulled events are consumer territory (no built-in client fold engine); the shipped client caches and exposes the feed, it does not project it. ServiceWorker-based offline HTML and WASM client deciders are ROADMAP raw ideas.

## Non-goals (explicit)

- No client-appended events, no CRDTs, no client-side deciders (see the [research synthesis](../research/2026-10-09_frontend-sync-protocol-synthesis.md) §2.4 for the reasoning trail).
