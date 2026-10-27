# Frontend Sync Protocol — LiveStore/Axon Synthesis for cqrs-htmx

**Date:** 2026-10-09
**Status:** Design research (episode 2 of the offline-sync series)
**Series:** episode 1 = [`2026-06-27_offline-first-command-sync-research.html`](../brainstorming/2026-06-27_offline-first-command-sync-research.html) → shipped as ADR-0023 (command sync), ADR-0029 (SharedWorker), ADR-0040 (IndexedDB), ADR-0042 (extraction to root). This report extends the series from **writes** to **reads + batching + permissions**. Outcome: ADR-0056 + the `SyncPullHandler`/`SyncPushHandler` protocol + sync client v1.5.0.
**Ask:** "Bring go-cqrs-lite Command and Event Sourcing capabilities to the frontend in a smart and composable way — offline support, distributed conflict resolution, permission- and cache-aware. Get inspired by LiveStore, a little by Axon Framework."

---

## 1. What the inspirations actually do

### 1.1 LiveStore (livestorejs/livestore)

Local-first, client-centric data layer: a reactive embedded SQLite whose state is a **deterministic projection of an append-only client event log**.

| Concept                      | Mechanism                                                                                                                     |
| ---------------------------- | ----------------------------------------------------------------------------------------------------------------------------- |
| Event log                    | Per-store append-only log; events carry `seqNum` + `parentSeqNum` (Git-like causality DAG)                                     |
| Materializers                | Pure (event, state) → SQL functions; same log ⇒ same SQLite state on every client; DB is a rebuildable cache                    |
| Sync                         | Git-like **pull-before-push** with batches + cursor; client **rebases** unpushed events onto new upstream events                |
| Conflict resolution          | LWW on concurrent events (custom merge planned, not built); no CRDTs in core                                                    |
| Backend reset detection      | `backendId` echoed by the backend; client config `onBackendIdMismatch: reset \| shutdown \| ignore`                             |
| Local storage                | wa-sqlite over OPFS; single-writer **leader** (Web Locks) per client; SharedWorker fans out to tabs                             |
| Reactivity                   | `queryDb()` / `computed()` / `signal()` graph; synchronous re-render after commit                                               |
| Auth in sync                 | Mostly TODO — token validation at connection time, per-push hooks in the reference provider                                     |

**The fatal flaw we already rejected (ADR-0023):** LiveStore syncs **events** and rebases only the *materializer*. The server can never reject an appended fact without a "time machine paradox" — domain invariants (and Casbin authz) only run on the client. LiveStore itself is bolting commands onto an event-sync architecture (issue #717, RFC #945). We chose the inverse in June: **sync commands, not events** — the server re-decides intent against authoritative state.

### 1.2 Axon Framework

Server-centric Java CQRS/ES. The transferable client-facing semantics:

| Concept                      | Takeaway for a frontend protocol                                                                                                |
| ---------------------------- | ------------------------------------------------------------------------------------------------------------------------------- |
| CommandGateway               | Commands return **outcomes, not domain state** (`sendAndWait` → result message carrying only exception results); state comes from queries |
| Subscription queries         | Initial result + live updates via `QueryUpdateEmitter` — our SSE metadata envelope is this, minus payloads                        |
| Tracking processors + tokens | Persisted per-processor position in the global stream = our **checkpoint cursor**; replay = re-pull from an old cursor            |
| Conflict detection           | Commands carry **expected aggregate version**; semantic `ConflictResolver` inspects unseen events; concurrent writes fail closed  |
| Distributed command bus      | Aggregate-id routing + consistent hashing: per-aggregate serialization — the event store's stream ordering is our equivalent      |

### 1.3 What cqrs-htmx already has (episode 1)

- Offline **write** queue: SharedWorker coordinator, IndexedDB-persisted command envelopes `{verb, url, values, headers}`, cross-session retry (ADR-0029/0040/0042). Queue-only contract: the server still owns `decide()`.
- ACK protocol: `X-Command-Id` header → `sync:ack` SSE broadcast → pending→confirmed/rejected honest UI (ADR-0023/0024).
- SSE live updates: journal-backed replay (`transport.JournalSSEStore`), `Last-Event-ID` reconnect, permission-scoped endpoints via `WithSSEFilter` — but the envelope is **metadata-only** (deliberately safe default).
- Client identity: `X-Client-Id` (ULID) stamped on mutations for offline attribution.

### 1.4 The gaps (this episode's scope)

1. **No offline reads.** The UI cannot render anything when offline — the queue is write-only. LiveStore's whole value is the local event cache + deterministic projection.
2. **No payload pull.** SSE is metadata-only; a client that wants to project events locally has no permission-aware way to fetch them with payloads.
3. **No batched push.** The queue replays one HTTP request per command; a 50-command offline session is 50 round-trips at reconnect.
4. **No cursor protocol.** SSE `Last-Event-ID` covers live reconnect tails, not durable client-side catch-up after hours offline.
5. **No reset detection.** A backend rebuilt from scratch would feed a client stale-history confusion instead of a clean cache reset (LiveStore's `backendId` lesson).

---

## 2. The design (ADR-0056)

### 2.1 The three laws (carried forward + extended)

1. **Clients sync intent, never facts.** Commands flow client→server; the server re-decides (ADR-0023). Nothing a client sends can bypass validation, authz, or invariants.
2. **Clients may cache facts for reads.** Events flow server→client through an explicit, permission-filtered, cursor-based pull endpoint. Cached events are a disposable projection source, never a write path.
3. **Conflicts resolve by server total order.** Pull-before-push; the server serializes per stream and answers each command with a **classified outcome**; the client rebases (re-pulls) and either retries (transient) or surfaces the rejection (conflict/rejection).

This is the LiveStore loop (pull/push batches + cursor + backendId + rebase) with the authority inverted — exactly the "smart" part: **rebase the prediction, never the fact.**

### 2.2 Server surface (root module)

**`cqrshtmx.SyncPullHandler(journal, opts...)`** — `GET ?after=<eventID>&limit=<n>`

- Reads via `event.SeekableJournal.ReadFrom` (fall back to `ReadAll` like `JournalSSEStore`).
- Response: `{backendId, events, nextCursor, hasMore}`; each event carries full metadata **plus payload** (`payload` raw JSON when the event encoding is JSON; base64 + `payloadEncoding` otherwise).
- `WithSyncPullFilter(func(r, evt) bool)` — the **permission-aware** seam: the consumer asserts Casbin policy per request; excluded events are invisible (fail-closed filtering — the SSE `WithSSEFilter` doctrine applied to payloads).
- `WithSyncPullBackendID` — reset detection; `WithSyncPullLimit` — batch clamp (default 500, hard max 1000).
- **Cache-aware:** FNV-1a ETag over `(backendId, nextCursor, hasMore)`; `If-None-Match` → `304`; `Cache-Control: private, no-cache` (always revalidate, revalidation is one hash).

**`(app *App) SyncPushHandler()`** — `POST {commands: [{commandId, type, body, contentType}]}`

- The App now retains every `Command()`/`CommandTyped()` registration (type → decoder+authz+guard config) — the same registry the HTTP endpoints use, so batch push composes with **zero new per-command registration**.
- Each envelope replays the endpoint pipeline — authz → decode (synthesized request carrying the envelope body + original request context/headers) → requestGuard → context enrichment → dispatch — and answers with an outcome, Axon-style: `{commandId, status: confirmed|rejected, error?: {message, code, family}}`.
- Conflict semantics: the error **family** is the client's contract — `Rejection`/`Conflict` = permanent (surface to user, drop from queue), `Transient` = retryable (re-queue), `Corruption`/`Infrastructure` = permanent-failed (drop + log).
- Idempotency: the envelope's `commandId` rides the synthesized request as `X-Command-Id` (and the outer context), so ACK broadcasting (`BroadcastOnAck` via `AfterDispatch`) fires per command and consumer idempotency middleware dedups retries.
- CSRF: the push endpoint is one POST validated by the surrounding middleware once; inner envelopes do not re-validate CSRF (they are not independently reachable).

### 2.3 Client surface (sync assets v1.5.0 — worker stays a coordinator, not a proxy)

- **Pull loop lives in the tab** (tabs already own SSE): on reconnect/first load → `GET /sync/pull?after=<cursor>` (paged by `hasMore`) → hand events to the worker (`cache-events`) → worker persists to IndexedDB (`events` store + `meta` cursor/backendId), broadcasts `sync:events` to all tabs → **then** flush the queue (pull-before-push law).
- **Offline reads:** any tab can request the cached feed (`get-events`) — enough for event-derived UI, toasts, and "last known state" panels. Full offline HTML rendering is ServiceWorker territory → ROADMAP.
- **Batch push:** queued envelopes gain an optional `commandType` (captured from the issuing element's `data-sync-command-type`); the flush path batches typed commands into ONE `POST /sync/push`; per-command outcomes drive queue deletion (confirmed/rejected) or re-queue (transient). Untyped commands keep the per-URL replay — fully backward compatible.
- **Reset:** `backendId` mismatch → worker clears the events store + cursor, broadcasts `sync:reset` (LiveStore's `onBackendIdMismatch: reset`).

### 2.4 What we deliberately do NOT do

- **No client-side deciders.** Optimistic predictions stay what ADR-0023 made them: disposable UI states (`pending`), never locally-decided events. (LiveStore's materializer-only rebase is the bug class we refuse.)
- **No CRDTs.** Server total order + classified rejections cover the multi-device case; CRDTs would move trust into the client (same reason the [iroh analysis](2026-08-02_iroh-p2p-networking-fit-analysis.md) §5.3 kept P2P off the write path).
- **No client event append, ever.** The pull endpoint is read-only by construction.

### 2.5 Distributed-conflict walkthrough (the "distributed services" ask)

Two devices, one aggregate, offline simultaneously:

1. Device A and B both pull to cursor 100, both queue a command on stream `todo/42`.
2. A reconnects: pull (no change) → push `{A-cmd}` → server decides on v7 → event v8 → outcome `confirmed` → A pulls to 101.
3. B reconnects: pull sees v8 (A's event, permission-filtered if B lacks access) → **rebase**: B's pending command stays pending, UI re-renders on the new state → push `{B-cmd}` → decider runs against v8 → either `confirmed` (compatible intent) or `rejected` with family `Conflict`/`Rejection` (e.g., "already completed") → B's UI shows the honest rejection.
4. Server-side, concurrent pushes on the same stream serialize on the event store's stream ordering (Axon's consistent-hash affinity equivalent); no client ever observes a half-merged state.

That is conflict **resolution** without client-side merge logic: convergence by re-decide + re-pull, honesty by classified outcomes.

---

## 3. Alternatives considered

| Alternative                          | Verdict                                                                                                                                                             |
| ------------------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Adopt LiveStore wholesale (events both ways) | Rejected — the time-machine paradox (ADR-0023); invariants + Casbin must stay server-side.                                                                       |
| CRDT merge layer (iroh-docs / yjs)   | Rejected — moves write trust to the client; the write path stays centralized (iroh analysis §4.3/§5.3). Read-only CRDT replication stays a ROADMAP raw idea.          |
| WASM-compiled Go deciders in the browser | Deferred — `GOOS=js` builds of identity-model folds could give optimistic *predictions* with real domain logic someday; heavy (binary size, codegen) for now. ROADMAP. |
| SSE-with-payloads (extend the envelope) | Rejected for the default (leak risk), but the pull endpoint IS the opt-in payload surface — metadata SSE stays the safe default.                                     |
| ServiceWorker HTML cache for offline reads | Deferred to ROADMAP — biggest UX win for offline, but lifecycle/complexity is a separate episode.                                                                   |

## 4. Sources

- LiveStore: repo + docs.livestore.dev (how-it-works, concepts, syncing, events, reactivity, web adapter, CF sync provider)
- Axon Framework: reference docs 4.11/5.0 (command dispatchers, streaming processors, query dispatchers/subscription queries, conflict resolution, sagas)
- Local: ADR-0023/0024/0027/0029/0040/0042, `transport/` (journal SSE), `ack.go`, `sync/` assets, episode-1 brainstorming doc, iroh fit analysis
