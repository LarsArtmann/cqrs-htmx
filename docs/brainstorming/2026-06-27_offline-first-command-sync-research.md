Offline-First Command Sync — Research & Brainstorming Brainstorming · Research 2026-06-27 cqrs-htmx + go-localsync Contents Executive Summary Context LiveStore Deep Dive Service Worker vs SharedWorker fanOut vs Watermill IndexedDB is Dead Honest Optimistic UI The Command Problem Command-Sync Architecture Decisions Made Open Questions Glossary References BRAINSTORMING · 2026-06-27 
# Offline-First Command Sync 

Research session exploring how cqrs-htmx and go-localsync can leverage the browser's
 native event loop, Service Workers, OPFS, and LiveStore-inspired architecture to build a
 truly offline-first, honest-UI system where commands — not events
 — are the sync unit. 

Projects: cqrs-htmx, go-localsync Inspiration: LiveStore Status: research / brainstorming 
## ★ Executive Summary 

This research session explored how cqrs-htmx and go-localsync can leverage
 browser-native event primitives, Service Workers, OPFS-backed SQLite, and
 LiveStore-inspired architecture to build a truly offline-first system. The central
 finding: sync commands, not events — because commands are re-decidable and
 events are not. cqrs-htmx already has every server-side building block needed; the gap
 is the client half. 

6 Building blocks cqrs-htmx already has 4 Broken diagrams fixed 3 Decisions made 10 Open questions The bottom line LiveStore is the best reference architecture for browser-side SQLite + offline sync, but
 it has a fatal flaw: no commands — clients emit events directly,
 bypassing all domain validation. LiveStore itself acknowledges this (Issue #717, RFC PR
 #945, milestone 0.5.0). cqrs-htmx already has `command.Command `, `decider.Repository `, `foldUser() `, `DispatchWSCommand `, and `Broadcaster `. The missing piece is the
 client: a Service Worker with a durable command queue, local SQLite (OPFS), and honest
 UI that never lies about pending state. 
### What you'll find in this document 
Section What it covers LiveStore Deep Dive Three-tier thread model, OPFS+sqlite-wasm, dual-database pattern, sync layer SW vs SharedWorker Why LiveStore rejected Service Workers, and the one case where you might still
 want one fanOut vs Watermill Why two event-bus abstractions exist with opposite delivery semantics Honest Optimistic UI Why standard optimistic UI lies, and the patterns that don't — with visual
 mockups The Command Problem LiveStore's biggest flaw: no domain validation, no re-validation during rebase Command-Sync Architecture The proposed solution: sync commands, server re-decides, client rebases Decisions Made What's already settled (OPFS, command-sync, honest UI) Open Questions 10 unresolved questions, ordered by leverage — with priority labels 
## 00 Context & Starting Point 

The exploration began with a question: how can we use JavaScript and the web's native event loop to our benefit? There
 are events we want on the backend, events we need on the backend, different transport
 mechanisms, and the desire to work superbly offline. Two existing projects form the
 foundation: 

### cqrs-htmx 

A Go library for CQRS + HTMX + event sourcing. Has a clean transport-agnostic `fanOut[T] `(shared by `Broadcaster `/ `WSBroadcaster `), SSE with Last-Event-ID reconnection, an `SSEEventStore `interface ( no production impl yet ), and
 full CQRS command/decider/event-store infrastructure in the `usermgmt `submodule. 

### go-localsync 

An SDK for syncing data from external providers to local event-sourced storage. Has `ConflictResolver `/ `LWWResolver `wired in, but `VectorClock `/ `Operation[T] `/ `SyncMessage `are implemented + tested yet not wired . Strictly provider→local
 PULL. Vector clocks currently empty — LWW always falls through to timestamp. 

### The core opportunity 

The web gives you event primitives most people ignore: Service Worker (a sync daemon that can survive page close, intercept fetch, run Background Sync), SharedWorker (one instance across all tabs), OPFS (synchronous file I/O for sqlite-wasm), and the microtask/macrotask split (optimistic UI now, reconcile later). The
 cqrs-htmx `fanOut `pattern maps cleanly onto a client-side `EventTarget `+ `BroadcastChannel `bus. 

The fundamental question Is the browser a write peer (a CRDT node with local mutation authority) or a command-queuing thin client (the server is authoritative, the client replays)?
 These are radically different architectures, and the answer determines everything
 downstream. 
## 01 LiveStore Architecture Deep Dive 

LiveStore [https://livestore.dev/]is a local-first, event-sourced, reactive
 state management layer built on SQLite-WASM in the browser. The research revealed
 several critical design decisions. 

### Three-tier thread architecture 

LiveStore does NOT use a Service Worker. It uses a SharedWorker + DedicatedWorker topology with leader election. 

Tab A — Main Thread (Leader) ClientSession + in-memory SQLite (synchronous reactive queries).
 Holds the `WebLock `→ spawns the Dedicated Worker below. Dedicated Worker (LEADER) LeaderThread : State DB (OPFS, materialized view), Eventlog DB
 (OPFS, source of truth), LeaderSyncProcessor, SyncBackend connection. Single writer to OPFS + single sync connection. ↑ MessagePort — elected via `navigator.locks `SharedWorker (1 per store, shared across all tabs) Proxy / message router. Holds termination WebLock. Validates store invariants across
 leader transitions. ↑ MessagePort Tab B — Main Thread (Follower) ClientSession + in-memory SQLite (receives changesets from leader).
 Routes all requests through SharedWorker → Leader Worker. Key property Only the leader worker is the single writer to persistent storage and the single
 connection to the sync backend. All other tabs are followers that receive changesets.
 When the leader tab closes, the lock releases and another tab takes over automatically. 
### Dual-database pattern 
Database Role Properties `eventlog.db `Append-only event log — the source of truth Immutable, git-inspired, sync unit `state.db `Materialized view derived from events Optimized for queries, recreated on schema change 
### Leader election via Web Locks 

A lock named `livestore-tab-lock-${storeId} `is acquired via the Web Locks API ( `navigator.locks `). The tab holding the lock
 spawns a Dedicated Web Worker (the "leader worker") and connects it to the SharedWorker.
 When the leader tab closes, the lock releases and another tab takes over automatically. 

Key property Only the leader worker is the single writer to persistent storage and the single
 connection to the sync backend. All other tabs are followers that receive changesets. 
### OPFS + sqlite-wasm (no IndexedDB, no SharedArrayBuffer) 

LiveStore ships its own WebAssembly SQLite build ( `@livestore/wa-sqlite `, a
 fork of rhashimoto/wa-sqlite with synchronous APIs). Persistence uses OPFS (Origin Private File System) exclusively via `FileSystemSyncAccessHandle `. 

 - AccessHandlePoolVFS — a custom SQLite VFS that wraps OPFS's `FileSystemSyncAccessHandle `for synchronous file I/O. SQLite (a
 synchronous C library) requires this. 
 - Pre-allocates a pool of 20 OPFS files to avoid async operations
 during queries (covering main DBs, journal files, WAL files, temp files). 
 - Each OPFS file has a 4096-byte header : 512 bytes for the SQLite path,
 4 bytes for open flags, 8 bytes for integrity digest. 
 - Private browsing fallback : When OPFS is unavailable (Safari/Firefox
 private mode), falls back to in-memory storage. 
 - No SharedArrayBuffer / Atomics needed — unlike upstream
 sqlite-wasm's async-proxy approach, LiveStore's VFS uses sync OPFS handles directly. 
### Sync layer (event-sourcing-based, "git-inspired") 

Sync is two-tier with rebasing: 

Processor Location Role `ClientSessionSyncProcessor `Each client session (main thread) Manages local rebase logic, materializes events into in-memory DB, pushes batches `LeaderSyncProcessor `Leader worker Persists to eventlog, materializes to state DB, communicates with sync backend 
Each event has a composite sequence number with three components: 

 - `global `— global ordering across all clients 
 - `client `— per-client ordering 
 - `rebaseGeneration `— for conflict resolution during rebasing 
When incoming remote events conflict with pending local events, the system performs a rebase : rolls back local changesets (using SQLite session changeset
 tracking), re-materializes the remote events, then re-chains the pending local events. 

### Browser APIs used 
API Where Why OPFS ( `FileSystemSyncAccessHandle `) Dedicated leader worker Synchronous file I/O for SQLite-WASM persistence Web Locks ( `navigator.locks `) Main thread of each tab Leader election SharedWorker Cross-tab coordination Single shared instance; message routing hub Dedicated Web Worker Leader tab Runs SQLite + OPFS + sync off the main thread MessagePort Between all layers Binary message channels (transferable, zero-copy) Notably NOT used `SharedArrayBuffer `/ `Atomics `, `BroadcastChannel `, `IndexedDB `, and Service Workers . 
## 02 Service Worker vs SharedWorker 

Question: Should all sync from frontend to backend happen in the
 Service Worker? 

### LiveStore's explicit rejection of the Service Worker 

LiveStore gives three concrete reasons for not using a SW: 

SW limitation Why it kills the LiveStore pattern Idle eviction Browsers tear down SWs after ~30s of inactivity. A sync engine holding a WebSocket
 + SQLite handles must be always alive . Non-negotiable for push-based
 sync. No OPFS `FileSystemSyncAccessHandle `(synchronous OPFS) is unavailable in SW
 context. sqlite-wasm needs sync I/O. This alone is fatal. Wrong abstraction SW is a network interceptor (fetch handler, cache). A sync leader is a long-lived compute+state node . Conflating them creates fragile lifecycle
 bugs. 
### The nuance worth stealing 

LiveStore uses `navigator.locks `(leader election) and the SharedWorker as a
 message router, then pushes heavy work into a DedicatedWorker that the SharedWorker
 proxies to. Clean separation. 

### The contrarian case for Service Workers 

There is one thing a SW is uniquely good at that LiveStore doesn't exploit: Background Sync API + fetch interception . If you want queued writes
 to flush after the tab closes (true background sync), only a SW can do that.
 LiveStore's SharedWorker dies with the last tab. 

The real question: Do you need writes to flush when zero tabs are open? If yes → you need a SW (for Background Sync), and
 you must solve the OPFS problem (run sqlite-wasm in a DedicatedWorker spawned by the SW, or accept async storage). If no → copy LiveStore's
 SharedWorker+DedicatedWorker topology exactly and ignore the SW entirely. 

## 03 fanOut[T] vs Watermill — Why Two Exist 

Question: Does `fanOut[T] `use `go-cqrs-lite/watermill/ `, and if not, why not? 

Answer: No. They solve opposite problems with opposite delivery
 semantics. A deliberate, correct split. 

Attribute `fanOut[T] `(root module) Watermill `EventBus `(usermgmt) Role Browser-push hub (SSE/WS to N tabs) CQRS domain event bus (decider→projection) Contract 4 methods: Subscribe/Unsubscribe/Broadcast/Count Full `event.Bus `: Publish/Subscribe/SubscribeAll/Use/UsePublish +
 Ack/Nack Delivery Drop to slow consumers Block publisher until ACK Trigger `AfterDispatchHook `(HTTP request lifecycle) `decider.Repository.Execute() `(command dispatch) Types Generic `T `( `SSEEvent `, `string `) `event.Event `with metadata + codec Dep weight 0 (pure `sync.RWMutex `+ `map `) `ThreeDotsLabs/watermill `+ gochannel 
### The `fanOut[T] `implementation (111 lines, zero deps) 

```
`// fanout.go — transport-agnostic subscriber hub type fanOut[T any ] struct {
 mu sync.RWMutex
 subscribers map [ uintptr ] chan T
} // Non-blocking broadcast — drops to slow consumers func (f *fanOut[T]) Broadcast (msg T) {
 f.mu. RLock () defer f.mu. RUnlock () for _, ch := range f.subscribers { select { case ch <- msg: default : // drop to slow consumer }
 }
} `
```

Using watermill for fanout would (a) force a transitive dep on every root-module
 consumer, (b) stall command dispatch behind a slow web client (opposite of what you want for real-time push), and (c) carry the entire
 Ack/Nack/middleware contract that nobody calls. 

The interesting question for offline-first work The `fanOut `drops messages on purpose, but the client-side reconnection
 story needs them. Should the durable replay path read from the same watermill-backed `event.Store `that the decider writes to — giving `SSEEventStore `its first production impl for free — while `fanOut `stays the best-effort live-push layer? 
## 04 IndexedDB is Dead — OPFS or Nothing 

Decision: Never use IndexedDB. OPFS + sqlite-wasm is strictly better. 

Property IndexedDB OPFS + sqlite-wasm Query model Key-value with async cursors, no joins Full SQL, joins, transactions, indexes Sync I/O Async-only, callback/promise hell Synchronous via `FileSystemSyncAccessHandle `Ecosystem Fragile wrapper libs (Dexie, idb) SQLite — the most battle-tested DB in history Performance Unpredictable, serialization overhead Native-speed WASM, predictable Developer experience Everyone hates it Just SQL. The universal language. 
LiveStore, the reference architecture for this work, uses OPFS exclusively. `IndexedDB `is notably absent from their entire codebase. 

## 05 Honest Optimistic UI 

Requirement: The frontend code must not lie to the user. There
 should be a human-centric communication that a state was applied locally but not yet on
 the server, and when the server confirms, that should be clear in the UI. 

### The problem with standard "optimistic UI" 

The research surfaced a key finding: the most-cited failure mode of standard optimistic
 UI is the "silent rollback" — the mutation vanishes on error with
 no explanation, and screen readers announce pending state as confirmed truth (an
 accessibility violation). 

Cardinal sin · Silent rollback 
### The Silent Rollback 

When the server rejects an optimistic mutation, the default behavior in most
 frameworks (React's `useOptimistic `, Remix, etc.) is to silently discard
 the optimistic state and revert to the "true" state. The user sees their change
 disappear with no explanation. 

From the research: "The action throws, the optimistic state vanishes, and the user sees their change
 disappear with no explanation." And: "Optimistic UI is by definition not the truth yet. Screen readers will announce
 optimistic content as if it were confirmed, which is misleading." 

### The insight that makes this nearly free 
You're already event-sourcing. Sync provenance is already tracked — standard
 optimistic UI just throws it away. LiveStore tracks `clientHead `(events committed locally) vs `backendHead `(events confirmed by server). Every event knows whether it's
 ACKed. cqrs-htmx's `event.Store `has the same information. The gap is purely
 in the materialization layer : standard materializers project events → rows
 and discard the provenance. An honest materializer projects `event → (row, sync_state) `. 
### The mutation lifecycle state machine 

Each mutation should be modeled as a finite state machine with explicitly named states: 

State Meaning UI treatment `idle `No in-flight mutation Normal `pending `Local state reflects intended outcome; request in flight Muted/dashed, glanceable `confirmed `Server confirmed; local state matches canonical representation Solid — transitions from pending `rejected `Server rejected or transport failed Inline error with reason + retry/discard `superseded `A newer mutation replaced this operation's view Show diff, let user reconcile 
### What this looks like in practice 

Three concrete UI states, showing how the same todo list communicates sync provenance
 without lying to the user: 

Pending (applied locally) Buy groceries ✓ Walk the dog ✓ Write report Confirmed (server ACKed) ✓ Buy groceries ✓ Walk the dog ✓ Write report Rejected (server said no) ✓ Walk the dog × Delete todo #42 — not found Retry ✓ Write report 
The global sync indicator at the top of the page tracks the aggregate state: 

All confirmed ✓ All changes saved 3 pending ↻ 3 pending — syncing… 1 failed ⚠ 1 failed — tap to retry The cardinal rule Pending items appear immediately (no spinner, no delay) but look different (dashed border, muted opacity) until confirmed. The sync dot
 gives a glanceable per-item status. The global bar gives the aggregate count. Nothing ever silently disappears. 
### Production UI patterns surveyed 

#### Ambient indicator 

"Saving…", "Saved", "All changes saved" 

Used by: Notion, Google Docs, Figma 

Verdict: Necessary but insufficient — hides per-item state. 

#### WhatsApp checkmarks 

1✓ sent, ✓✓ delivered, blue ✓✓ read 

The purest provenance model. 

Verdict: Strong but message-specific; wrong metaphor for most CRUD. 

#### Muted/dashed pending items 

Pending items appear immediately but look different (dashed border, muted colors)
 until server confirms. 

Verdict: Recommended default. Glanceable, honest, non-modal. 

#### Per-item status pill 

Explicit `pending `/ `failed `tags on items. 

Verdict: Honest but noisy. Good for destructive/important ops. 

#### Power Apps globe icon 

6 states + badge count + outbox screen. The most complete sync indicator. 

Verdict: Right for offline-heavy field apps. Heavy for general use. 

#### Typed SyncState 

Evolu: `SyncStateInitial | IsSyncing | IsSynced | IsNotSynced `as
 discriminated union. 

Verdict: Good as a complement, never alone. 

Recommended default Muted/dashed-until-confirmed pattern as the baseline (subtle, glanceable, honest), plus
 a global sync indicator (✓ saved / ↻ 3 pending / ⚠ 1 failed), plus never-silent rollback on rejection (inline error with retry, input
 preserved). Reserve explicit status pills for destructive operations. 

Principle: Pending state should be felt, not read. A glanceable visual
 shift communicates state without forcing the user to parse labels — but the global
 indicator gives them a precise count when they want to check. 
## 06 The Command Problem — LiveStore's Biggest Flaw 

Observation: LiveStore doesn't have the concept of Commands, which is a
 flaw. How can we improve this with Commands in mind? 

Validated by LiveStore itself This is now an acknowledged gap with active work: Issue #717 [https://github.com/livestorejs/livestore/issues/717]("Commands
 Primitive", milestone 0.5.0), PR #945 [https://github.com/livestorejs/livestore/pull/945]("RFC 0002:
 Command Replay", merged), and PR #995 [https://github.com/livestorejs/livestore/pull/995](experimental
 implementation). They're trying to bolt commands onto an event-sync architecture. You don't have to — cqrs-htmx already has commands. 
### LiveStore's paper-thin write path 

In LiveStore, the entire write-path validation is schema check then blind append . There is no command/decision layer: 

User Action Event Constructor 
Effect Schema validation — structural only (throws if shape doesn't
 match) store.commit() NO validation, NO decision, NO business rules Append to local eventlog (pending) Materialize via SQL (INSERT/UPDATE/DELETE) 
transactional; can throw and rollback, but can't "reject" Sync to leader → backend → all clients During rebase: replay materializer against potentially-changed state NO re-validation of business rules — THIS IS THE CORE PROBLEM Potential invariant violation 
### The three fatal consequences 
Critical · Invariant violations 
### No domain validation 

Committing `todoCompleted `for a non-existent todo, or twice, just appends
 the event. The materializer runs `UPDATE ... WHERE id = ? `which silently
 affects 0 rows — but the event is permanently in the history, synced to
 everyone, with no observable effect on state. 

Critical · Rebase corruption 
### No re-validation during sync rebase 

When a client's pending events are rebased on top of upstream events during sync, the
 business rules that were valid at commit time may no longer hold. Without command
 replay, you get invariant violations : referential integrity
 violations, business rule violations, uniqueness constraint violations. 

Important · Security 
### Client can emit arbitrary events 

All domain validation is advisory. The client can emit any event that matches the
 schema shape. The server has no opportunity to reject based on business rules before
 the event is committed and synced. 

### Greg Young's warning (2010) 
"It is not natural for the domain to tell a client that something in the past tense no
 longer happened." 
Events are immutable facts — but if the client emits them, the server can't reject
 them without a "time machine." LiveStore's rebase replays the materializer (SQL), never the decider (domain logic). 

## 07 The Command-Sync Architecture 

The core insight: sync commands, not events. Commands are re-decidable. Events are
 not. 

This single property unlocks local-first correctness: 

Client queues command Local `decide(state, command) `→ optimistic events (pending) Sync COMMAND to server Server re-decides against authoritative state 
→ authoritative events (confirmed) OR rejection Client rebases: replace optimistic events with server events Honest UI: `pending `→ `confirmed `/ `rejected `/ `superseded `
The server re-validates intent against authoritative state . If two
 clients sent concurrent commands, the server processes them in order — each
 command's decider runs against fresh state. No invariant violations. No time machine
 paradox. The client's optimistic events are disposable — they were a
 prediction, and the server has the real answer. 

### How this maps to cqrs-htmx (already closer than LiveStore) 
Concept LiveStore (missing/bolt-on) cqrs-htmx (already exists) Command type Not modeled `command.Command `Decider (decide + fold) Events only, evolve via materializer `decider.Repository `, `foldUser() `, `decide*() `Client→server transport Events via push/pull `DispatchWSCommand `Server→client notification Event stream `Broadcaster `/ SSE Read model projection Materializers → SQLite `UserReadModel `+ SQL read models Authoritative event store Backend eventlog `event.Store `cqrs-htmx is already closer to the correct architecture than LiveStore. What's missing is the client half: local command queue + local decide + honest UI +
 rebase. 
### The three-layer model 
Service Worker / SharedWorker (the client's "leader") Command Queue (OPFS-backed, durable): seq command status result 1 AddTodo pending (local) 2 CompleteTodo confirmed events[…] 3 DeleteTodo rejected error Local SQLite (OPFS): `eventlog.db `← optimistic
 events from local `decide() `, `state.db `← materialized
 from optimistic events. 
Decider : runs `decide(local_state, command) `→
 events (pure function — WASM or ported logic). ↕ WebSocket: push commands ↑, pull confirmed events ↓ Go Backend (authoritative) `command.Dispatch(AddTodo) `→ `decider.Execute `: load
 state → `fold(events) `→ `decide(state, command) `→ events OR rejection → save to `event.Store `→ publish via watermill → projections → Broadcast confirmed events via SSE/WS to all tabs . 
### Why command-sync is strictly better than event-sync 
Concern Event-sync (LiveStore) Command-sync (proposed) Domain validation Schema only Full `decide() `guards Server re-validation on rebase Impossible (no intent) Natural (re-decide against fresh state) Concurrent mutations Silent invariant violation Server decides in order, rejects if invalid Security Client can emit arbitrary events Client can only request (command), server decides Conflict resolution Materializer replay (SQL) Decider replay (domain logic) Intent preservation Lost (only outcome recorded) Preserved (command IS intent) Honest UI Hard (events are facts, can't be "pending") Natural (commands are requests, naturally pending) 
### How honest UI falls out for free 

Each command in the queue has a visible lifecycle. No silent rollback. The command never disappears — it transitions states visibly. The user always
 knows the provenance of what they see. 

## ✓ Decisions Made 

Before the open questions, here's what the research already settled . These are
 not up for debate — the evidence was conclusive. 

✓ OPFS over IndexedDB — no contest OPFS + sqlite-wasm gives synchronous I/O, full SQL, joins, transactions, and
 battle-tested SQLite. IndexedDB gives async key-value hell. LiveStore uses OPFS
 exclusively. Decision final. ✓ Sync commands, not events Commands are re-decidable against fresh state; events are immutable facts. Syncing
 commands lets the server re-validate intent during rebase. Syncing events makes
 rejection a "time machine paradox" (Greg Young, 2010). LiveStore's acknowledged gap
 (Issue #717) confirms this. ✓ Honest UI — never lie about pending state Standard optimistic UI's "silent rollback" is the cardinal sin. The UI must
 communicate local-vs-confirmed provenance visibly (dashed/muted pending items +
 global sync indicator + never-silent rejection). Sync provenance is already in the
 event log — honest UI just surfaces it instead of throwing it away. ✓ fanOut stays hand-rolled; watermill stays server-side They solve opposite problems (browser-push hub vs CQRS event bus) with opposite
 delivery semantics (drop vs block). Merging them would stall command dispatch behind
 slow web clients. The SSEEventStore should get a production impl backed by the
 watermill event.Store — bridging the two without merging them. ✓ cqrs-htmx is the right foundation, not LiveStore cqrs-htmx already has command.Command, decider.Repository, foldUser(), decide*(),
 DispatchWSCommand, Broadcaster, event.Store, and SQL read models. LiveStore has none
 of these. The work is additive (client half), not a rewrite. 
## 08 Open Questions 

These are the unresolved questions from the research session, ordered by leverage.
 Answering the first few unlocks the design for all the rest. Priority labels: Keystone gates the entire architecture, High must be resolved before implementation, Medium can be deferred. 

### Q1: Where does `decide() `run on the client? Keystone 

The decider is pure logic (no I/O) — it can run anywhere. Options: 

 - (a) Compile Go decider to WASM and run in the Service Worker 
 - (b) Port decider to TypeScript — second implementation with
 contract tests 
 - (c) Don't run it locally at all — just queue commands, show
 intent in UI, get events back from server 
Option (c) is the most honest for HTMX (thin client) but means offline = "queued, not
 executed." Options (a)/(b) give full offline execution but require the domain logic to
 exist in the browser. 

Which level of offline execution do you actually need — "I can queue work" or
 "I can fully work"? 

### Q2: Closed-tab writes — hard requirement? Keystone 

Is "flush queued mutations when no tab is open" a hard requirement, or can you accept
 "sync runs while ≥1 tab is alive" (the LiveStore contract)? This single answer
 determines whether you need a Service Worker at all. 

### Q3: Provenance granularity — per-row or per-aggregate? High 

Per-row is simplest for UI rendering. Per-aggregate is more precise (a "create todo" +
 "assign tag" share a fate — one aggregate, two rows). Do you stamp every row, or
 derive row-state from aggregate-level provenance at query time? 

### Q4: Event log divergence strategy Medium 

The client has optimistic events; the server has authoritative events. On
 confirmation, do you (a) replace the client's entire local eventlog with server events
 (clean slate, simple), or (b) merge by sequence number (complex, but preserves local
 history for debugging)? Clean-slate is simpler and more honest — the client's
 eventlog was always a prediction. 

### Q5: Cross-tab asymmetry High 

If the client pushes events up, the server's `SSEEventStore `would
 broadcast them to other tabs. Those other tabs should see incoming events as
 "confirmed" (they came from the server), while the originating tab sees its own writes
 as "pending" until the round-trip completes. Is that asymmetry acceptable, or should
 all tabs see uniform state? 

### Q6: Conflict visibility — invisible-merge or honest-merge? Medium 

CRDT-based apps (Yjs, Automerge) make conflicts invisible by design —
 the data structure converges. But you want honesty. Does "honesty" mean showing the
 user that a concurrent edit was merged into theirs, or is invisible-merge acceptable
 when the CRDT guarantees convergence? (Hypothesis: invisible-merge is fine for text,
 visible for structural changes — but that's a heuristic.) 

### Q7: Command ordering across clients Medium 

If client A and client B send concurrent commands for the same aggregate, the server
 serializes them. Client A's optimistic events might be wrong because B's command
 changed the state first. The server's re-decide handles this — but does the
 client need to re-fetch the full aggregate state after every sync, or can it receive
 just the delta (server's authoritative events) and re-fold? 

### Q8: What is a "command" over the wire? High 

cqrs-htmx's `command.Command `carries a type string + JSON payload. The `DispatchWSCommand `path already exists. But the server currently
 dispatches and responds in a single HTTP/WS round-trip. For command-sync, you need:
 push command → receive async ACK (confirmed/rejected/superseded) → receive
 confirmed events. Is the existing `DispatchWSCommand `+ `Broadcaster `enough, or do you need a dedicated sync protocol with cursors? 

### Q9: How honest is too honest? Medium 

If a user creates 50 todos offline, do you show 50 dashed items, or one "50 pending"
 indicator? There's a tension between per-item honesty and cognitive load. Where's the
 threshold — is it a UI setting, or baked into the design? 

### Q10: Rejection UX — rollback or preserve-and-flag? Medium 

The "silent rollback" is the cardinal sin. But the alternative has two flavors: (a)
 snap back to pre-edit state + toast with retry, or (b) keep the rejected edit visible
 inline, flagged as "server rejected: [reason]," let the user decide to
 edit/discard/retry. Option (b) is more honest but more complex. Which feels right? 

Recommended next step Question 1 is the keystone. The answer determines whether you're building "LiveStore
 with commands" (full WASM decider in SW) or "honest optimistic HTMX" (queue commands in
 SW, server decides, honest pending UI). Both are valuable. The second is dramatically
 simpler and still a massive improvement over standard optimistic UI. 
## β Glossary 

Key terms used throughout this document, defined in context. 

OPFS (Origin Private File System) A browser API providing a private file system scoped to the web origin. Supports `FileSystemSyncAccessHandle `for synchronous file I/O in worker
 contexts — required by SQLite-WASM. Available in Chrome, Edge, Safari 17+. Not
 available in Service Workers (only Dedicated/Shared Workers). VFS (Virtual File System) SQLite's pluggable storage abstraction. LiveStore ships `AccessHandlePoolVFS `— a VFS that pools 20 OPFS sync access handles to provide the synchronous file
 operations SQLite requires, without needing SharedArrayBuffer or Atomics. Decider A pure function `decide(state, command) → events `that validates a
 command against current aggregate state and either emits events or rejects. The core
 of CQRS domain logic. In cqrs-htmx: `decideRegisterUser() `, `decideAddCredential() `, etc. In go-cqrs-lite: `decider.Repository `. Fold A pure function `fold(state, event) → new_state `that reconstructs
 aggregate state from events. In cqrs-htmx: `foldUser() `. Combined with `decide `, forms the complete event-sourcing domain model. Materializer LiveStore's term for a function that maps an event to SQL mutations ( `event → INSERT/UPDATE/DELETE `). Equivalent to a CQRS projection, but runs synchronously during event commit, not
 asynchronously. In cqrs-htmx: the equivalent is `UserReadModel.Handle() `+ `syncToSQL() `. Rebase LiveStore's conflict resolution: when remote events arrive that conflict with pending
 local events, the client rolls back its local changesets, applies the remote events,
 then re-chains its local events on top with an incremented `rebaseGeneration `. Critically, LiveStore replays only the materializer (SQL), never the decider (domain logic) — which
 is why invariants can break. CRDT (Conflict-free Replicated Data Type) A data structure that converges across replicas without coordination. Used by Yjs,
 Automerge. go-localsync has `ConflictResolver[T] `+ `LWWResolver[T] `(Last-Writer-Wins), with `VectorClock `/ `Operation[T] `implemented but not wired. CRDTs guarantee convergence but not invariant preservation . fanOut[T] cqrs-htmx's transport-agnostic subscriber hub (111 lines, zero deps). Shared by `Broadcaster `(SSE) and `WSBroadcaster `(WebSocket).
 Non-blocking broadcast — drops to slow consumers. Intentionally NOT watermill's `EventBus `, which blocks publishers until ACK. SSEEventStore cqrs-htmx interface ( `EventsAfter(lastID) []SSEEvent `) for durable event
 replay on SSE reconnection. No production implementation exists — only a test helper. The
 proposed production impl would read from the same watermill-backed `event.Store `the decider writes to. Provenance The metadata tracking where data came from — specifically, whether a
 given row/state was derived from local-only (optimistic) events or server-confirmed
 (authoritative) events. The foundation of honest UI. LiveStore tracks this as `clientHead `vs `backendHead `. 
## ♦ References & Sources 

### LiveStore 

 - LiveStore — landing page [https://livestore.dev/]
 - LiveStore — GitHub repo [https://github.com/livestorejs/livestore]
 - Web adapter docs (SW rationale, SharedWorker/DedicatedWorker roles, OPFS) [https://dev.docs.livestore.dev/platform-adapters/web-adapter/]
 - Materializers documentation [https://dev.docs.livestore.dev/building-with-livestore/state/materializers]
 - Design decisions [https://dev.docs.livestore.dev/understanding-livestore/design-decisions]
 - Issue #717: Commands Primitive (tracking, milestone 0.5.0) [https://github.com/livestorejs/livestore/issues/717]
 - PR #945: RFC 0002 — Command Replay (merged) [https://github.com/livestorejs/livestore/pull/945]
 - PR #995: Commands experimental API [https://github.com/livestorejs/livestore/pull/995]
### CQRS / Event Sourcing Theory 

 - Greg Young — CQRS Documents (2010) [https://github.com/keyvanakbary/cqrs-documents]
 - CQRS Documents Reader's Guide [https://treeru.com/en/blog/cqrs-documents-greg-young-korean]
 - CodeOpinion — Greg Young answers Event Sourcing questions [https://codeopinion.com/greg-young-answers-your-event-sourcing-questions/]
 - Sourcing Framework — Commands [https://fco.github.io/Sourcing-2/concepts/commands.html]
### Optimistic / Honest UI 

 - Remix — Pending UI taxonomy [https://remix.run/docs/en/main/discussion/pending-ui]
 - Palma — Optimistic UI with server reconciliation (mutation lifecycle) [https://matheuspalma.com/blog/optimistic-ui-server-reconciliation-patterns]
 - 72Technologies — React 19 useOptimistic and network failures [https://www.72technologies.com/blog/react-19-useoptimistic-network-failures]
 - AppMaster — Offline-first background sync UX [https://appmaster.io/blog/offline-first-background-sync-conflict-retries-ux]
 - Power Apps — Offline sync icon states [https://github.com/MicrosoftDocs/powerapps-docs/blob/main/powerapps-docs/mobile/offline-sync-icon.md]
 - Evolu — Typed SyncState [https://www.evolu.dev/docs/api-reference/common/local-first/interfaces/SyncStateIsSynced]
 - RxDB — Optimistic UI [https://rxdb.info/articles/optimistic-ui.html]
### Local-first / Production Examples 

 - How Linear achieves sub-100ms updates [https://performance.dev/how-is-linear-so-fast-a-technical-breakdown]
 - Figma — How multiplayer technology works [https://www.figma.com/blog/how-figmas-multiplayer-technology-works/]
 - WhatsApp — Checkmark meaning [https://faq.whatsapp.com/665923838265756]
### Browser APIs 

 - OPFS / FileSystemSyncAccessHandle (MDN) [https://developer.mozilla.org/en-US/docs/Web/API/File_System_Access_API]
 - Web Locks API (MDN) [https://developer.mozilla.org/en-US/docs/Web/API/Web_Locks_API]
 - SharedWorker (MDN) [https://developer.mozilla.org/en-US/docs/Web/API/SharedWorker]
 - Service Worker API (MDN) [https://developer.mozilla.org/en-US/docs/Web/API/Service_Worker_API]
### Project files referenced 
File Relevance `cqrs-htmx/fanout.go `Transport-agnostic fan-out hub (111 lines, zero deps) `cqrs-htmx/sse_broadcaster.go `SSE Broadcaster embedding `fanOut[SSEEvent] ``cqrs-htmx/ws_broadcaster.go `WS Broadcaster embedding `fanOut[string] ``cqrs-htmx/ws_dispatch.go ``DispatchWSCommand `— WS→CQRS bridge `cqrs-htmx/sse_store.go ``SSEEventStore `interface (no production impl) `cqrs-htmx/usermgmt/es_setup.go `watermill EventBus wiring (read-your-writes) `cqrs-htmx/usermgmt/es_decide.go `Pure decide functions (guards + event creation) `cqrs-htmx/usermgmt/es_state.go ``foldUser() `— pure event→state fold `go-localsync/pkg/crdt/conflict.go ``ConflictResolver[T] `, `LWWResolver[T] ``go-localsync/pkg/crdt/vectorclock.go `VectorClock (implemented, not wired) `go-localsync/pkg/crdt/operation.go `Operation[T] (implemented, not wired) `go-localsync/pkg/sync/sync.go `Syncer (provider→local pull) 
Generated 2026-06-27 · Research session exploring offline-first command sync for
 cqrs-htmx + go-localsync · Inspired by LiveStore architecture 

