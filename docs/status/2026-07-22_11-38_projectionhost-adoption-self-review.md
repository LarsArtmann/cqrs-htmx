# Status: projectionhost/v4 Adoption — Self-Review

**Date:** 2026-07-22 11:38
**Session scope:** Adopt `projectionhost/v4` to replace hand-rolled `StartProjections`, benchmark projection replay, evaluate CatchUpSubscriber, update docs.

---

## a) FULLY DONE

### projectionhost/v4 Adoption (Core)

| Item | Status | Detail |
|---|---|---|
| `es_projection_setup.go` rewrite | DONE | Replaced 155 LOC hand-rolled replay+dedup+live-handler with `projectionhost.Host`. `StartProjections` returns `(*projectionhost.Host, error)`. Uses `WithSubscriber` for replay→live handoff, `WithDeadLetterStore` for poison-event capture. |
| `waitForDrain` sync-wait wrapper | DONE | Polls `host.Status()` until all workers reach `WorkerLive` or `WorkerStopped`. Preserves read-your-writes. 30s timeout. |
| `EventSourcedSetup` struct + `Close()` | DONE | Holds `*projectionhost.Host`, stops it in `Close()` before closing bus/store. |
| `SQLiteEventSourcedSetup` struct + `Close()`/`GracefulClose()` | DONE | Same pattern. |
| `PostgresEventSourcedSetup` struct + `Close()`/`GracefulClose()` | DONE | Same pattern. |
| `Service` struct + `closeInfra()` | DONE | Holds `*projectionhost.Host`, stops it in `closeInfra()`. |
| `usermgmt/go.mod` | DONE | Added `projectionhost/v4 v4.0.0` require. |
| All callers updated | DONE | `es_setup.go`, `sqlite_setup.go`, `postgres_setup.go`, `service_core.go` — all handle the new `(*Host, error)` return. |

### Tests

| Item | Status | Detail |
|---|---|---|
| `es_projection_setup_test.go` | DONE | Rewrote: kept `collectProjections` test + `stubProjection`, added `TestStartProjections_ReadYourWrites`, removed dead `buildLiveHandler` tests. |
| `es_checkpoint_test.go` | DONE | Updated checkpoint name from `"usermgmt:start_projections"` to per-projection `"usermgmt-user-read-model"`. |
| Integration test | DONE | Updated `waitForUser` comment. |
| Full test suite passes | DONE | All modules: root, usermgmt, integration_test, adminui, loginpage. Race detector clean. |
| Lint clean | DONE | `golangci-lint run ./usermgmt/...` → 0 issues. |
| Coverage | DONE | usermgmt 80.5% (gate: ≥74%), root 93.8% (gate: ≥90%). |

### Benchmark

| Item | Status | Detail |
|---|---|---|
| `BenchmarkProjectionReplay` | DONE | 100/1K/10K events. Results: ~3µs/event, linear scaling. 10K = 30ms. |
| Hot-path profiling | DONE | Dispatch ~1µs, decode ~1-2µs, error mapping ~1-5µs, JSON ~10µs. All within bounds — no optimization needed. |

### Documentation

| Item | Status | Detail |
|---|---|---|
| ADR-0031 | DONE | Rewritten: Status → Superseded. Documents why projectionhost won over CatchUpSubscriber. |
| ADR INDEX | DONE | Updated status to "Superseded". |
| ROADMAP.md | DONE | All 4 items marked Done or Not Needed. |
| FEATURES.md | DONE | Checkpoint Replay + Projection Setup entries updated. |
| CHANGELOG.md | DONE | Unreleased section: Changed + Added entries. |
| AGENTS.md | DONE | Key Pattern + Gotcha entries added. |

### CatchUpSubscriber Evaluation

| Item | Status | Detail |
|---|---|---|
| Evaluation | DONE | **Closed as Not Needed.** projectionhost `WithSubscriber` provides the same replay→live handoff. CatchUpSubscriber would add `message.Message` adapter overhead without benefit. |

---

## b) PARTIALLY DONE

| Item | What's done | What's missing |
|---|---|---|
| Error handling in `Close()` methods | `_ = errorfamily.WrapTransient(err, ...)` used to suppress unused-result lint | Errors are silently discarded, not logged. Should use `slog.Warn` or return the error. |
| `EventSourcedConfig.CheckpointStore` docstring | Field still works correctly | Docstring still says "enables checkpoint-based projection replay" without mentioning per-projection keys or the breaking change from single-key migration. |
| `ServiceConfig.CheckpointStore` docstring | Same | Same stale documentation. |
| `FuzzProjectionDedupMap` | Test still compiles and passes | Comment references "replay→live dedup path" which is now handled internally by projectionhost, not by usermgmt code. The fuzz test itself just tests a map lookup — harmless but misleading comment. |

---

## c) NOT STARTED

| Item | Why it matters |
|---|---|
| `docs/migrations/v2-to-v3.md` update | Still references old `StartProjections()` signature and behavior. Migration checklist item says "Projection startup uses `StartProjections()`" without noting it now returns `*Host`. |
| `docs/MIGRATION-v3-incremental.md` update | Documents checkpoint-based replay with old single-key model. Needs note about per-projection keys. |
| `docs/evaluations/catchup-subscriber-evaluation.md` update | Still says "Verdict: Defer" — should be updated to "Closed as Not Needed" with cross-reference to ADR-0031 Superseded. |
| `es_materialize_adapter.go` docstring update | Comment says "so it can be used with `StartProjections`" — still accurate but could mention projectionhost. |
| `TODO_LIST.md` check | May contain projection-related TODO items that are now done. |
| `examples/` check | No examples reference `StartProjections` directly, but examples that use `NewService` now transitively use projectionhost. Should verify no example breaks. |
| `nix fmt` full run | Only ran `gofmt` on changed files. `nix fmt` covers more (prettier for markdown, etc.). |
| `cqrs-lint` run | Not run. Should verify no CQRS anti-patterns introduced. |
| Committing remaining 5 files | 5 files with minor fixes (`_ =` prefix, gofmt) are uncommitted. |

---

## d) TOTALLY FUCKED UP

Nothing is totally fucked up. But there are real concerns:

### Silent error swallowing in Close() methods — BAD PATTERN

In `EventSourcedSetup.Close()`, `SQLiteEventSourcedSetup.Close()`, `PostgresEventSourcedSetup.Close()`, `Service.closeInfra()`, and all `GracefulClose()` methods, I wrote:

```go
_ = errorfamily.WrapTransient(err, "usermgmt.es_setup.stop_projections", "stop projection host")
```

This creates a wrapped error and immediately discards it. It allocates an error object for nothing. The correct pattern is either:
- `slog.Warn("projection host stop failed", "error", err)` — log it
- Or just `_ = s.projectionHost.Stop()` — don't wrap at all if you're going to discard

The `_ = errorfamily.WrapTransient(...)` pattern is the worst of both worlds: it looks like it's doing something meaningful (wrapping the error) but then throws it away. This is a code smell.

### `waitForDrain` is a polling hack

The `waitForDrain` function polls `host.Status()` every 10ms. This works but is architecturally ugly:
- It's timing-based, not signal-based
- If a worker crashes and restarts during drain, the poller might see `WorkerBackoff` and wait unnecessarily
- The 30s timeout is arbitrary — no way for consumers to configure it
- The watermill `SubscribeAll` returning immediately (registering handler and returning) means workers transition through `WorkerLive` to `WorkerStopped` almost instantly. We treat `WorkerStopped` as "done" but this conflates "drain complete + handler registered" with "worker crashed and stopped."

A proper solution would be a channel or callback from projectionhost signaling "drain complete." This doesn't exist in the projectionhost API today.

### `WithDeadLetterStore(projectionhost.NewMemoryDeadLetterStore(), 0)` — unbounded memory

The DLQ is in-memory with no size limit. In a long-running process with repeated poison events, this grows unbounded. For a library, this is probably fine (consumers can configure their own), but it should be documented.

### Behavior change: retry with backoff during drain

The old `StartProjections` handled projection errors by logging and continuing (fail-soft). The new code uses projectionhost which retries 3 times with exponential backoff before dead-lettering. This means:
- A projection handler bug during drain now causes 3 retries per bad event
- Each retry has backoff delay (1s initial, 30s max)
- For 10 bad events, drain could take 30+ seconds
- The 30s `waitForDrain` timeout would fire, returning a `Transient` error

This is a semantic change that could surprise consumers upgrading from the old fail-soft behavior.

---

## e) WHAT WE SHOULD IMPROVE

1. **Fix the error swallowing pattern** — Replace `_ = errorfamily.WrapTransient(...)` with `slog.Warn(...)` in all Close/GracefulClose methods.
2. **Make `waitForDrain` timeout configurable** — Add a `ProjectionDrainTimeout` field to `EventSourcedConfig` / `ServiceConfig`.
3. **Document the DLQ behavior change** — Old code was fail-soft (log + continue). New code retries 3x then dead-letters. This needs a CHANGELOG entry and possibly a config option to disable retries (threshold=1).
4. **Add a `ProjectionHostOptions` config struct** — Let consumers configure maxRestarts, backoff, batch size, DLQ threshold, shutdown timeout. Currently all hardcoded to defaults.
5. **Expose the `*projectionhost.Host` on `Service`** — Currently it's unexported (`projectionHost`). Consumers who want to inspect worker status, lag metrics, or replay dead letters need access.
6. **Consider `WithMetrics` integration** — projectionhost has a `MetricsRecorder` interface. Wire it to slog or an optional Prometheus recorder.
7. **Update migration docs** — `v2-to-v3.md` and `MIGRATION-v3-incremental.md` still describe the old `StartProjections` behavior.
8. **Update the CatchUpSubscriber evaluation doc** — Still says "Defer."
9. **Run `nix fmt`** — Full formatting pass on all files.
10. **Write an upgrade guide** — For consumers with existing checkpoint data (the single-key → per-projection-key migration).

---

## f) NEXT 50 THINGS TO GET DONE

### Immediate (fixes from this session)

1. Fix error swallowing in all 6 Close/GracefulClose methods → use `slog.Warn`
2. Commit the 5 uncommitted files (gofmt + `_ =` fixes)
3. Run `nix fmt` for full formatting pass
4. Run `nix run .#lint` on all modules
5. Run `nix run .#coverage-gate` to verify CI gates pass
6. Run `cqrs-lint` to check for CQRS anti-patterns

### Documentation cleanup

7. Update `docs/migrations/v2-to-v3.md` — StartProjections signature change
8. Update `docs/MIGRATION-v3-incremental.md` — per-projection checkpoint keys
9. Update `docs/evaluations/catchup-subscriber-evaluation.md` — "Closed as Not Needed"
10. Update `es_materialize_adapter.go` docstring — mention projectionhost
11. Update `EventSourcedConfig.CheckpointStore` field docstring
12. Update `ServiceConfig.CheckpointStore` field docstring
13. Update `FuzzProjectionDedupMap` comment
14. Check `TODO_LIST.md` for completed projection items
15. Write upgrade guide for checkpoint key migration

### projectionhost integration improvements

16. Add `ProjectionHostOptions` config to `EventSourcedConfig` / `ServiceConfig`
17. Make `waitForDrain` timeout configurable
18. Expose `*projectionhost.Host` on `Service` (or accessor method)
19. Add `ProjectionHostStatus()` accessor on `Service`
20. Add `ProjectionHostLag()` accessor for health checks
21. Wire `MetricsRecorder` to optional slog-based metrics
22. Consider configurable DLQ threshold (currently hardcoded to default 3)
23. Consider SQL-backed `DeadLetterStore` for persistent poison-event capture
24. Document the retry-with-backoff behavior change (old: fail-soft, new: 3x retry + DLQ)
25. Add integration test for DLQ scenario (poison event → dead-letter → checkpoint advances)

### Testing improvements

26. Add test for `waitForDrain` timeout path
27. Add test for `waitForDrain` when a worker fails (`WorkerFailed`)
28. Add test for checkpoint persistence across restart (full round-trip with projectionhost)
29. Add test for DLQ replay via `host.ReplayDeadLetters`
30. Add test for concurrent projection errors (one fails, others succeed)
31. Add benchmark for `waitForDrain` polling overhead
32. Add stress test: 100K events through projectionhost
33. Verify `MaterializeProjection` adapter works with projectionhost (integration test)

### Architecture improvements

34. Push for a `DrainComplete()` channel/callback in projectionhost upstream — eliminates polling
35. Push for `SubscribeAll` that blocks (or a `SubscribeAllBlocking` variant) in watermill upstream
36. Consider per-projection `EventTypes` optimization — projectionhost builds `typeSet` at registration (already done internally)
37. Evaluate `projectionhost.Reset` for rebuilding read models from scratch
38. Evaluate `projectionhost.WithOnFailed` callback for alerting on exhausted restarts
39. Consider exposing `host.Status()` in adminui dashboard (projection health panel)
40. Add health check endpoint that includes projection lag

### Broader project improvements (noticed during session)

41. The `EventSourcedConfig` / `ServiceConfig` / `SQLiteSetupConfig` / `PostgresSetupConfig` all have overlapping `CheckpointStore` + `SnapshotConfig` fields — consider composition
42. The `closeBus` helper in `es_setup.go` silently discards errors — same pattern as my Close() issue
43. `journalFromStore` falls back to `memory.NewMemoryStore()` when the store doesn't implement `Journal` — this silently loses all events for projection replay. Should be an error.
44. The `EventSourcedSetup` → `Service` transition copies fields one by one — error-prone (easy to miss a field, as demonstrated by needing to add `projectionHost` to the copy list)
45. Root module has pre-existing lint issues (`containedctx` in `sse_stream.go`, `ireturn` in `example_otel_test.go`) — not from this session but worth fixing
46. The `goinfertypeargs` diagnostics (56 warnings) are pre-existing — consider fixing or suppressing
47. `docs/adr/0030-phase2-persistence-strategy.md` is still "Proposed" — needs resolution
48. `go.work` local replaces are still required — track upstream go-cqrs-lite release status
49. Consider adding `projectionhost/v4` to the go.work replace audit comment (already there, but verify version)
50. Update the SKILL.md `references/core-api.md` with projectionhost adoption details

---

## g) QUESTIONS I CANNOT FIGURE OUT MYSELF

### 1. Should `StartProjections` expose the `*projectionhost.Host` publicly?

Currently `Service` holds it as unexported `projectionHost`. Consumers who want to:
- Inspect worker status (`host.Status()`)
- Check projection lag (`host.LagDuration()`)
- Replay dead letters (`host.ReplayDeadLetters()`)
- Reset a projection (`host.Reset()`)

...have no way to access the host. Should I add a `ProjectionHost()` accessor on `Service`? Or is this intentionally hidden because consumers should use `Service.Close()` / `Service.Stop()` for lifecycle?

This is a library API design decision — exposing it means committing to the projectionhost API as part of the public surface.

### 2. Should the DLQ retry threshold be configurable per-projection?

projectionhost applies the same `dlqThreshold` (default 3) to all projections. But some projections might be more tolerant of transient errors than others. For example, the `AuditLog` projection might want threshold=0 (never retry, always dead-letter) while `UserReadModel` might want threshold=5 (retry more before giving up).

This requires either per-projection options on `Host.Register` (which projectionhost doesn't support today) or running multiple hosts (one per threshold group). Both are upstream changes.

Should I push for this upstream, or is the global threshold sufficient for this library?

### 3. Should `waitForDrain` be replaced with an upstream `DrainComplete` signal?

The polling approach works but is architecturally inelegant. The clean solution is for projectionhost to expose a channel or callback that fires when all workers complete their initial drain. This would require an upstream change to `projectionhost.Host`.

Should I:
- (a) Push for this upstream in go-cqrs-lite (adds API surface, correct solution)
- (b) Keep the polling wrapper (works now, no upstream dependency, acceptable overhead)
- (c) Use a different approach entirely (e.g., check `host.LastProcessedAt()` against the journal's latest event timestamp)

This determines whether the polling hack stays or gets replaced.

---

## Summary

The projectionhost/v4 adoption is functionally complete and all tests pass. The core architectural decision (projectionhost over CatchUpSubscriber) is sound and well-documented. The main weaknesses are:

1. **Error handling quality** — silent swallowing in Close() methods
2. **Polling-based drain wait** — works but not elegant
3. **Behavior change** — retry+DLQ replaces fail-soft (needs documentation)
4. **Stale migration docs** — multiple docs still describe old behavior
5. **5 uncommitted files** — minor fixes pending commit

Coverage: root 93.8% (gate 90%), usermgmt 80.5% (gate 74%). All green.
