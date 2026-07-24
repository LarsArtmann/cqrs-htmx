# Rebuild Projection Runbook

> Step-by-step operational guide for rebuilding a projection's read model from the event journal.

---

## When to Rebuild

| Scenario | Rebuild? | Why |
| --- | --- | --- |
| Projection returned bad data after a code deploy | Yes | New code produces different read model shape; replay to regenerate |
| Casbin authorization rules drifted from events | Yes | Rebuild `casbin-projection` to re-derive policies |
| Read model table corrupted (disk error, bad SQL) | Yes | Replay to regenerate from source-of-truth events |
| Projection stuck in `failed` state | Yes | Poison event in DLQ; fix handler, rebuild to retry |
| After a schema_version upcaster deploy | Maybe | Only if old projection data is incompatible |
| Routine maintenance | No | Projections self-heal via checkpoints on restart |

---

## Prerequisites

- Application is running and serving traffic (rebuild is live, not offline).
- Choose a low-traffic window (all projections briefly stop during rebuild).
- Know the projection name to rebuild (see table below).

### Projection Names

| Read Model | Name |
| --- | --- |
| User read model | `user-read-model` |
| Membership read model | `membership-read-model` |
| Tenant read model | `tenant-read-model` |
| Bot read model | `bot-read-model` |
| Casbin authorization | `casbin-projection` |
| Audit log | `audit-log` |

---

## Procedure

### Option A: Programmatic (Recommended)

```go
ctx := context.Background()
err := svc.RebuildProjection(ctx, "user-read-model")
if err != nil {
    log.Fatal(err)
}
```

This single call:
1. Stops the projection host (all projections briefly pause).
2. Clears the checkpoint and resets the read model state for the named projection.
3. Creates a fresh host and replays the entire event journal.
4. Blocks until all projections reach live state.
5. Resumes normal operation.

**Duration:** Proportional to event count. ~1s for 1,000 events. ~30s for 100,000 events.

### Option B: Process Restart with Checkpoint Deletion

If the application does not expose `RebuildProjection`:

1. Stop the application process.
2. Delete the checkpoint for the target projection from your checkpoint store:
   ```sql
   -- For SQL-backed checkpoint stores:
   DELETE FROM projection_checkpoints WHERE projection_name = 'user-read-model';
   ```
3. Clear the read model table (if the projection does not implement `Resettable`):
   ```sql
   DELETE FROM users;  -- for user-read-model
   ```
4. Restart the application. The projection replays from scratch.

---

## Verification

After rebuild, verify the read model is correct:

### 1. Check Projection Status

```bash
curl -s http://localhost:8080/health/projections | jq '.[] | select(.name == "user-read-model")'
```

Expected: `"status": "live"`, `"errors": 0`, `"lag_ms": < 100`.

### 2. Compare Event Count to Read Model Count

```sql
-- Total UserRegistered events (source of truth)
SELECT count(*) FROM events WHERE type = 'UserRegistered';

-- Users in read model
SELECT count(*) FROM users;
```

These should match (minus any UserDeleted events).

### 3. Spot-Check a Known User

```bash
curl -s http://localhost:8080/auth/me | jq .
```

Verify the response contains the expected user data.

---

## Troubleshooting

### Rebuild Fails: "journal not seekable"

The event store must implement `event.SeekableJournal`. In-memory stores do; SQL stores do. If using a custom store, ensure it implements `ReadFrom`.

### Rebuild Fails: "projection not registered"

The projection name is case-sensitive and must match `projection.Name()` exactly. Check the table above.

### Rebuild Succeeds But Read Model Is Empty

The projection may not implement `Resettable`. In this case, the read model was not cleared before replay. The replay appends to existing data, which may produce duplicates. Either:
- Implement `Resettable` on your read model, OR
- Manually clear the read model table before rebuilding (Option B above).

### Rebuild Hangs

The drain timeout is 30 seconds. If replay takes longer, it times out with a `Transient` error. For very large event logs (>1M events), increase the timeout or rebuild during a full maintenance window.

---

## Post-Rebuild Actions

1. Monitor the projection status endpoint for 5 minutes.
2. Verify `lag_ms` returns to single-digit milliseconds.
3. Check application logs for any projection errors.
4. If authorization policies changed, test a few role-based access checks.

---

## See Also

- [Event Replay and Rebuild](./event-replay-and-rebuild.md) — Technical details
- [Projection Health Monitoring](./projection-health-monitoring.md) — Status endpoint reference
- [Consistency Model](./consistency-model.md) — What happens during rebuild
