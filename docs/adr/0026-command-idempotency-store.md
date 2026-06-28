# ADR-0026: Command Idempotency Store

## Status

Accepted — 2026-06-28

## Context

The offline-first command sync protocol (ADR-0023) and honest UI ACK protocol
(ADR-0024) introduce a fundamental problem: **what happens when a client
submits a command, loses the ACK, and retries?**

Without idempotency protection, the same command executes twice. For example:
- A user creates an item, the server processes it, but the network drops
  the SSE ACK. The client shows "pending" and retries.
- The server processes the retry, creating a **duplicate** item.

This is especially dangerous for offline-first clients using Service Workers
or SharedWorkers that queue commands and replay them when connectivity
returns.

## Decision

Add an `IdempotencyStore` interface to the root package:

```go
type IdempotencyStore interface {
    Seen(ctx context.Context, commandID string) (bool, error)
    Record(ctx context.Context, commandID string, ttl time.Duration) error
}
```

With a `MemoryIdempotencyStore` implementation that uses TTL-based
expiration with a background sweep goroutine.

The `CheckAndRecord` interface method provides atomic check-and-record under a
single lock, avoiding the TOCTOU race that separate `Seen` + `Record` calls
would create. Each implementation is responsible for its own atomicity
(MemoryIdempotencyStore: single mutex; future Redis store: `SET NX`; future
SQL store: `INSERT ... ON CONFLICT DO NOTHING`).

`ErrDuplicateCommand` maps to HTTP 409 Conflict via `MapError`.

### Design Principles

1. **Interface, not struct**: Consumers can implement Redis/Postgres-backed
   stores for multi-instance deployments. The in-memory store is for
   single-process use only.

2. **TTL-based**: Command IDs don't live forever. The default TTL should
   match the client's maximum retry window (typically minutes, not hours).

3. **Not auto-wired**: The store is NOT automatically injected into
   `App.Command`. Consumers wire it via `BeforeDispatchHook` or inside their
   command handlers. This follows the library principle of not enforcing
   behavior consumers might disagree with.

4. **Conflict, not Rejection**: Duplicate commands return 409 Conflict, not
   400 Bad Request. The command was valid; it was just already processed.

## Wiring Example

```go
store := cqrshtmx.NewMemoryIdempotencyStore(5 * time.Minute)
defer store.Close()

app := cqrshtmx.New(cqrshtmx.Config{
    Commands: cmdDisp,
    BeforeDispatchHook: func(ctx context.Context, r *http.Request) error {
        cmdID := cqrshtmx.CommandIDFromRequest(r)
        if cmdID == "" {
            return nil // ACK not requested; skip idempotency
        }
        return store.CheckAndRecord(ctx, cmdID, 10*time.Minute)
    },
})
```

## Consequences

- **Positive**: Prevents duplicate command execution on retry.
- **Positive**: Interface enables Redis/Postgres implementations for
  multi-instance deployments.
- **Negative**: In-memory store does not survive restarts. If the server
  crashes between processing a command and the client receiving the ACK,
  the retry will execute again after restart.
- **Negative**: Adds latency (one map lookup per command). Negligible for
  in-memory; measurable for network-backed stores.

## Future Work

- Redis-backed `IdempotencyStore` for multi-instance.
- Optional auto-wiring into `App.Command` when `Config.IdempotencyStore`
  is set (like `Config.Timeout`).
- Integration with the event store's built-in idempotency (if the command
  produces an event with the same aggregate ID + expected version, the
  event store already rejects duplicates).
