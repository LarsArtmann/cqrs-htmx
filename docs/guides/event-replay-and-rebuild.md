# Event Replay and Projection Rebuild

> How projection rebuilding works via `projectionhost.Host.Reset()` and the `RebuildProjection` convenience method.

---

## The Rebuild Primitive: `host.Reset()`

`projectionhost.Host` from go-cqrs-lite provides a `Reset(ctx, name, opts...)` method that is the canonical way to rebuild a projection's read model from scratch.

### What Reset Does

1. **Validates the host is stopped** — Reset returns an error if the host is still running. Call `host.Stop()` first.
2. **Calls `Resettable.Reset(ctx)`** on the projection if it implements the `Resettable` interface. This clears the read model's in-memory or SQL state.
3. **Clears the checkpoint** — Saves a zero-value `event.Checkpoint{}` for the projection name, so the next `Start()` replays from the very beginning.
4. **Optionally purges the dead-letter queue** — Pass `projectionhost.WithPurgeDeadLetters` to clear DLQ entries for this projection.

### What Reset Does NOT Do

- Does **not** replay events. Replay happens on the next `Start()`.
- Does **not** restart the host. The host is a one-shot lifecycle: Start once, Stop once. After Reset, create a new host.
- Does **not** delete events from the event store. Events are immutable and permanent.

---

## The Convenience Method: `Service.RebuildProjection()`

cqrs-htmx wraps the stop-reset-recreate-start-drain workflow into a single method:

```go
err := svc.RebuildProjection(ctx, "user-read-model")
```

This method:

1. Stops the current projection host.
2. Calls `host.Reset(ctx, name)` for the named projection.
3. Creates a fresh `projectionhost.Host` with all the same projections.
4. Starts the new host and blocks until all projections have drained the full journal.
5. Replaces the internal host reference.

**The read model for the named projection is rebuilt from scratch. All other projections also briefly stop and resume from their existing checkpoints.**

### Projection Names

| Projection            | Name                    |
| --------------------- | ----------------------- |
| User read model       | `user-read-model`       |
| Membership read model | `membership-read-model` |
| Tenant read model     | `tenant-read-model`     |
| Bot read model        | `bot-read-model`        |
| Casbin authorization  | `casbin-projection`     |
| Audit log (optional)  | `audit-log`             |

### When to Rebuild

- **Projection schema changed** — You deployed a new version of a read model with different columns/fields.
- **Projection bug fixed** — A bug produced incorrect read model data. Reset and replay to regenerate correct state.
- **Read model corrupted** — Manual SQL operation or disk error corrupted the read model table.
- **Casbin policy drift** — Authorization rules drifted from the event log. Rebuild the casbin projection.

### When NOT to Rebuild

- **Routine operation** — Projections self-heal on restart via checkpoints. Don't rebuild unless there's a specific problem.
- **Under load** — Rebuild stops all projections briefly and replays the entire journal. Do it during low-traffic periods.
- **As a "refresh" mechanism** — This is a maintenance operation, not a health check.

---

## See Also

- [Rebuild Runbook](./rebuild-projection-runbook.md) — Step-by-step operational guide
- [Consistency Model](./consistency-model.md) — What consistency guarantees hold during and after rebuild
