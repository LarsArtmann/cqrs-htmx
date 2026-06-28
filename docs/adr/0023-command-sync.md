# ADR 0023: Command-Sync Architecture

**Status:** Accepted
**Date:** 2026-06-28
**Related:** [Brainstorming doc](../brainstorming/2026-06-27_offline-first-command-sync-research.html), [Execution plan](../planning/2026-06-28_09-38_offline-first-command-sync.md)

## Context

LiveStore (the reference architecture for browser-side offline-first SQLite) syncs **events** between client and server. This creates a fatal flaw: events are immutable facts, so the server cannot reject them without a "time machine paradox" (Greg Young, 2010). LiveStore's rebase replays only the materializer (SQL), never the decider (domain logic), leading to invariant violations.

LiveStore itself acknowledges this gap (Issue #717, RFC PR #945, milestone 0.5.0) and is trying to bolt commands onto an event-sync architecture.

cqrs-htmx already has commands (`command.Command`), deciders (`decider.Repository`, `foldUser()`, `decide*()`), and the transport (`DispatchWSCommand`). The missing piece is the client half.

## Decision

**Sync commands, not events.** Commands are re-decidable; events are not.

The write path:

1. Client queues a command
2. Local `decide(state, command)` produces optimistic events (pending) — optional, see Q1
3. Sync the **command** to the server
4. Server re-decides against authoritative state → authoritative events (confirmed) OR rejection
5. Client rebases: replaces optimistic events with server events
6. Honest UI transitions: `pending → confirmed / rejected / superseded`

The server re-validates **intent** against **authoritative state**. Concurrent commands are serialized by the server. No invariant violations.

## Implementation (Phase 0 + Phase 1)

### Production SSEEventStore (`event_store_sse.go`)

The `SSEEventStore` interface (`EventsAfter(lastID string) []SSEEvent`) now has a production implementation: `JournalSSEStore`. It wraps `event.Journal` or `event.SeekableJournal` (position-based `ReadFrom` for efficient cursor replay), with an `EventToSSEMapper` function that consumers provide to render event payloads as HTML.

### ACK Protocol (`ack.go`)

`CommandAck` struct carries `{commandId, status, error}` JSON over SSE. The `BroadcastOnAck()` hook factory on `Broadcaster` broadcasts an ACK when the request carries an `X-Command-Id` header (opt-in — no header, no ACK). On success: `{status: "confirmed"}`. On failure: `{status: "rejected", error: msg}`.

## Consequences

- **Server is authoritative**: the client's optimistic events are disposable predictions
- **No invariant violations**: the decider runs on the server against fresh state
- **Security**: clients can only request (command), the server decides
- **Honest UI**: commands are naturally "pending" — the UI can show provenance
- **Intent preserved**: the command log records what the user wanted, not just what happened
