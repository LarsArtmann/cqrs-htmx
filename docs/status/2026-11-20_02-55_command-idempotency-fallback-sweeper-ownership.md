# Status: usermgmt default command-idempotency sweeper now owned by Service.Stop

**Date**: 2026-11-20 02:55 CET
**Session scope**: fix the goroutine leak browser-history's construction-failure
pin (`TestNewServer_ConstructionFailureDoesNotLeakBackgroundWorkers`) caught
after its workspace rode this repo's local HEAD via go.work replaces.

**Defect**: `defaultCommandIdempotency`'s no-store fallback created a
`go-idempotency.MemoryStore` sweeper goroutine with no close surface — a
consumer that never injects `ServiceConfig.IdempotencyStore` leaked the
goroutine past construction (browser-history's leak pin failed at HEAD).

**Fix**: the fallback store is built by `newDefaultCommandIdempotencyStore()`;
the factory tracks its `Close` on `Service.stopCommandIdemSweeper` and
`stopEvictions` stops it — the same lifecycle as every other eviction
goroutine. Injecting `ServiceConfig.IdempotencyStore` still takes precedence.

**Verification**: browser-history's leak pin green against this tree
(`ok ... 0.154s`); usermgmt package suite unchanged apart from the
pre-existing `TestSnapshotShape_SQLStoreDropsStampToday` red (fails at
01eb84f2 without this change — not this fix's class).

Residue for the owning wave: the snapshot-shape red + the K2 guard's
release-train story (the fix rides local HEAD; release tags will pick it up
at the next usermgmt train cut).

---

## f) Next steps

_Heading added 2026-11-28 so the harvest can see them: the two rows below transcribe the
"Residue" paragraph above word for word; nothing else in this report changed._

| # | Item |
| --- | --- |
| ~~1~~ | ~~the snapshot-shape red (`TestSnapshotShape_SQLStoreDropsStampToday`)~~ done — fixed at `65078274`, on `master` and in no usermgmt tag at 2026-11-28 — the sentinel test became `TestSnapshotShape_SQLStorePersistsStamp` and asserts the preservation that storage/v4 v4.11.2 now guarantees |
| ~~2~~ | ~~the K2 guard's release-train story (the fix rides local HEAD; release tags will pick it up at the next usermgmt train cut)~~ done — shipped in `usermgmt/v4.21.0`, which contains `40209db0`; its CHANGELOG line was added on 2026-11-28 |
