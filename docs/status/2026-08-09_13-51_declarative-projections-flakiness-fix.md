# Status Report: Declarative Metaengine Projections — Flakiness Fix, Coverage, Equivalence

**Date:** 2026-08-09 13:51
**Session Focus:** Fix test flakiness under `-race`, raise coverage to 70%+, write equivalence tests, cleanup code quality issues.

---

## a) FULLY DONE

### 1. Test Flakiness Completely Eliminated

**Root cause identified:** The projection host worker drains the journal, calls `SubscribeAll` (which **returns immediately** — it spawns a background event-loop goroutine), then exits cleanly (`WorkerStopped`). Once stopped, the bus subscription is permanently active. Combined with `BlockPublishUntilSubscriberAck=true` on the GoChannel bus, `Dispatch()` blocks until the subscriber processes and acks the event — effectively synchronous.

**Previous approach (broken):** The old `eventually` helper used a 100ms initial sleep + 50ms polling. Under `-race`, the goroutine scheduler introduces enough jitter that 100ms was sometimes insufficient for the event to propagate. This gave ~40% pass rate.

**Previous broken attempts:**
1. `waitForHostLive` + fixed 50ms delay → ~40% pass rate (WorkerLive is too transient — the worker passes through it and exits)
2. Processed-counter stabilization (200ms window) → ~20% pass rate (counter looks stable before events arrive)
3. `eventually` retry pattern with 20ms polling → ~30% pass rate (too aggressive, events not yet processed)
4. 100ms initial delay + 50ms polling → ~40% pass rate under race, 100% with GOMAXPROCS=1

**Fix applied:** `waitForProjectionsReady` waits for `WorkerStopped` status on all workers. This guarantees the subscription has been registered. Once stopped, `Dispatch` is synchronous via `BlockPublishUntilSubscriberAck`.

**Verification:** 5/5 consecutive runs pass with `-race` flag, 3/3 full-suite runs pass. Each test runs in ~0.01s.

### 2. Coverage Raised from 67.2% to 89.2%

Added 11 new test functions covering previously-untested code paths:

| Test | Covers |
|------|--------|
| `TestDeclarative_TenantDelete` | `removeTenant` fold, `AllTenants` after delete |
| `TestDeclarative_BotDelete` | `removeBot` fold, `FindBotsByOwner` after delete |
| `TestDeclarative_MembershipRolesUpdate` | `updateMembershipRoles` fold, `updateMemberPolicy` fold |
| `TestDeclarative_MembershipRemove` | `removeMembership` fold, `removeMemberPolicy` fold |
| `TestDeclarative_UserDisplayNameChange` | `updateDisplayNameChanged` fold |
| `TestDeclarative_UserCredentials` | `updateCredentialAdded`, `updateCredentialRemoved` folds |
| `TestDeclarative_UserTOTP` | `updateTOTPEnabled`, `updateTOTPDisabled` folds |
| `TestDeclarative_UserExternalAccounts` | `updateExternalAccountLinked`, `updateExternalAccountUnlinked` folds, `FindUserByExternalAccount` query, `externalAccountLinkScan` projection |
| `TestDeclarative_UserDelete` | `removeUser` fold, `AllUsers` after delete |
| `TestDeclarative_AllUsers` | `AllUsers` query with multiple users |
| `TestDeclarative_EquivalenceWithProjectionLayer` | Cross-validates declarative queries vs ProjectionLayer read models |

### 3. Code Quality Fixes

- **`bytesEqual` replaced with `bytes.Equal`** — removed custom byte-slice comparison helper in `declarations.go`, replaced with stdlib.
- **`waitForHostLive` removed** — unused function deleted from test file.
- **`intrange` lint fix** — `for i := 0; i < 3; i++` → `for range 3` (Go 1.22+ idiom).
- **`go.mod` fixed** — `metaengine/v4` promoted from `// indirect` to direct dependency (gopls warning resolved).
- **`fmt.Errorf` → `errors.New`** — all error-format lint warnings resolved by switching to `errors.New` for static error strings.
- **`eventually` simplified** — removed the 100ms initial sleep (no longer needed after `waitForProjectionsReady`).

### 4. Equivalence Test Written

`TestDeclarative_EquivalenceWithProjectionLayer` dispatches the same event stream through a system running BOTH the declarative projections (internal host) AND the ProjectionLayer (external host). Asserts tenant Name/DisplayName/Suspended match, and user Email/DisplayName/EmailVerified/TOTPEnabled match between the two projection systems.

### 5. Build + Lint + Test Verification

- `GOEXPERIMENT=jsonv2 go build ./systemadapter/` — zero errors
- `GOEXPERIMENT=jsonv2 golangci-lint run` — 0 issues
- All 25 tests pass (18 declarative + 7 old ProjectionLayer) with `-race`

---

## b) PARTIALLY DONE

### Coverage Gaps Remaining (89.2%, some functions below 100%)

| Function | Coverage | Missing scenario |
|----------|----------|------------------|
| `updateUserPolicy` | 33.3% | `RolesUpdated` event is legacy — no command emits it. Can only be tested by directly applying the fold. |
| `actionLevel` | 50.0% | Untested action levels: "use"/"write"/"edit" (level 2), "super"/"everything" (level 4), unknown action (0, false) |
| `roleGrantsAction` | 75.0% | `super_admin` and `owner` roles not directly tested |
| `FindUserByEmail` | 66.7% | Not-found path (empty result) untested |
| `FindTenantByName` | 66.7% | Not-found path untested |
| `FindBotByTokenHash` | 66.7% | Not-found path untested |
| `FindUserByExternalAccount` | 66.7% | Not-found path untested |
| `updateCredentialRemoved` | 87.5% | Removing a non-existent credential ID (no-op filter) |
| `updateExternalAccountUnlinked` | 87.5% | Unlinking a non-existent external account (no-op filter) |
| `Enforce` | 87.5% | Unknown action / empty policies path |

### Declarative Projections Architecture (from prior session)

The declarative projections are wired into `DomainConfig` and the old `ProjectionLayer` is still fully functional. Both hosts run simultaneously when `NewProjectionLayer(sys)` is called. The `ProjectionLayer` is NOT yet marked deprecated.

---

## c) NOT STARTED

1. **Deprecation of ProjectionLayer** — not marked `// Deprecated` yet. All existing tests and examples depend on it.
2. **examples/system-demo update** — still uses `ProjectionLayer` pattern, not declarative.
3. **CI/flake.nix coverage gate** — systemadapter coverage gate is at 70% from Phase 1 config; now at 89.2% but the gate config hasn't been verified against `nix run .#coverage-gate`.
4. **SQLite engine tests for declarative projections** — only memory engine tested. SQLite serializes to JSON, exercising different code paths (`reifyTo`/`reifyReflect`).
5. **go-cqrs-lite upstream commit** — `EventWithID.OccurredAt` extension in `typed_decoder.go` is still uncommitted.
6. **Migration guide** — no documentation for consumers migrating from ProjectionLayer to declarative.
7. **Negative/error path tests** — `ErrNotFound` when querying non-existent IDs, empty result sets, filter mismatches.
8. **ExternalAccountLink proper remove** — unlink event uses a hacky update-to-empty-struct instead of a proper remove fold. The `deriveKeys` mechanism can't derive the composite key from the unlink event because the insert key is `e.ID + ":" + provider + ":" + subject` but the unlink event doesn't have the same composite format.

---

## d) TOTALLY FUCKED UP

### Previous Session's Flakiness Debugging (now fixed, but worth documenting)

The previous session burned ~4 rounds of trial-and-error on timing-based approaches (fixed delays, processed-counter stabilization, lag checks, polling intervals) without understanding the fundamental architecture:

1. **Misdiagnosed the problem** — thought it was a "timing gap between Dispatch returning and projection processing." The real issue was that the worker hadn't entered live subscription yet when the first event was dispatched. `Persistent: false` on the GoChannel means events published before `SubscribeAll` registers are dropped.

2. **Tried GOMAXPROCS=1 as workaround** — this "worked" but masked the real issue. Under GOMAXPROCS=1, the goroutine scheduler serializes execution, so the worker always finishes draining + subscribing before the test goroutine resumes. This is a workaround, not a fix.

3. **Removed `waitForHostLive` calls thinking they caused lock contention** — the real problem was checking for `WorkerLive` (transient) instead of `WorkerStopped` (terminal). The calls were removed, making things worse.

4. **Didn't research the bus configuration** — `BlockPublishUntilSubscriberAck: true` and `Persistent: false` were the two key config values that explain the entire behavior. This should have been the first thing investigated.

### Session's Own Mistake

- **Syntax error in equivalence test** — missing closing `)` on `NewSuspendTenantCmd` call. Would have been caught by `go build` before running the test, but I ran the test first. Minor but avoidable.

---

## e) WHAT WE SHOULD IMPROVE

### Architecture

1. **ExternalAccountLink remove fold is broken** — the unlink event (`ExternalAccountUnlinkedPayload`) has `Provider` and `Subject` fields but NOT the composite key (`streamID:provider:subject`) used by the insert fold. The current implementation uses a hacky "update to empty struct" which doesn't actually remove the entry from the map — it just sets it to zero-value. The `FindUserByExternalAccount` query then finds the zero-value entry and returns the wrong user. This should use a proper remove fold with a custom key extractor, or a separate projection keyed by `provider:subject` alone.

2. **Two-host split brain persists** — declarative projections run in the system's internal host, but `NewProjectionLayer(sys)` creates a SECOND host. Both subscribe to the same bus. Events are processed twice. This is intentional for backward compatibility but creates confusion and doubles memory usage.

3. **PolicyEntry model is simplistic** — the fixed 5-role hierarchy (`super_admin > admin > user > viewer`) with hardcoded action levels doesn't support custom roles or Casbin-style pattern matching. This is a placeholder that works for the built-in roles but won't generalize.

4. **AuditLog as insert-fold, not append-fold** — the plan recommended `LogBackend` append semantics for ordered log guarantees. The implementation uses insert folds with composite keys. This works but loses append-only ordering guarantees. With the memory engine it's equivalent (both use maps), but a LogBackend-backed engine would give proper ordered log semantics.

5. **`actionLevel` is a hardcoded placeholder** — "view"/"read" → 1, "write"/"edit" → 2, etc. This doesn't match any real authorization model. It should be consumer-configurable.

### Testing

6. **No SQLite engine tests** — the memory engine stores Go structs directly; SQLite serializes to JSON. The `reifyTo`/`reifyReflect` fallback paths for JSON round-tripping are completely untested. A bug in JSON serialization (e.g., `time.Time` formatting, `[]byte` base64 encoding) would not be caught.

7. **No negative tests** — no test for `ErrNotFound` when querying a non-existent ID, no test for empty result sets, no test for filter mismatches. These paths exist in the code (returning `system.ErrNotFound`) but are untested.

8. **Equivalence test is minimal** — only tests Tenant and User, not Bot, Membership, Authz, or AuditLog. A comprehensive equivalence test would cover all 6 projections.

9. **No concurrent dispatch tests** — all tests dispatch commands sequentially. No test verifies correctness under concurrent command dispatch from multiple goroutines.

### Code Quality

10. **`updateUserPolicy` fold handler is dead code** — the `RolesUpdated` event is marked "legacy: no longer emitted" in identity-model constants. The fold handler will never fire in practice. It exists for backward compatibility with old event streams, but this should be documented.

11. **`declarations.go` is 575 lines** — all fold declarations are in one file. Could be split into `tenant_declarations.go`, `bot_declarations.go`, etc. for navigability.

12. **No doc comments on individual fold functions** — the function names are descriptive (`insertTenant`, `updateTenantSuspended`, `removeUser`) but one-line doc comments would help.

13. **`setupDeclarativeSystem` duplicates `setupTestSystem`** — both create a system, but one starts the system and waits for projections, while the other creates a ProjectionLayer. These could be unified or extracted into shared helpers.

---

## f) Next Steps (up to 50)

### Fix the ExternalAccountLink bug (CRITICAL)

1. Fix the ExternalAccountLink remove fold — either use a custom key extractor or restructure the projection to be keyed by `provider:subject` instead of `streamID:provider:subject`
2. Add a test that verifies unlinking actually removes the entry (query `FindUserByExternalAccount` after unlink, expect `ErrNotFound`)

### Coverage to 95%+

3. Add negative tests: query non-existent tenant/user/bot/membership, expect `ErrNotFound`
4. Test `actionLevel` with all action strings: "use", "write", "edit", "super", "everything", and unknown
5. Test `roleGrantsAction` with `super_admin` and `owner` roles directly
6. Test `FindUserByEmail` / `FindTenantByName` / `FindBotByTokenHash` not-found paths
7. Test `updateCredentialRemoved` with a non-existent credential ID
8. Test `updateExternalAccountUnlinked` with a non-existent external account

### SQLite engine tests

9. Write `TestDeclarative_SQLiteEngine` — use `sqliteDeployment()` instead of memory, run same test scenarios
10. Verify `time.Time` round-trips correctly through JSON serialization
11. Verify `[]byte` fields (TokenHash, credential ID, PublicKey) round-trip correctly
12. Verify filter/sort operations work on JSON-serialized values

### Equivalence test expansion

13. Expand equivalence test to cover Bot (name, ownerID, tokenHash, scopes)
14. Expand equivalence test to cover Membership (actorID, tenantID, roles)
15. Expand equivalence test to cover Authz enforcement (compare `pl.Authz.Enforce()` vs `systemadapter.Enforce()`)
16. Expand equivalence test to cover AuditLog (compare `pl.AuditLog.Entries()` vs `systemadapter.AuditEntries()`)
17. Expand equivalence test to cover delete operations (verify both systems agree on deletion)

### Deprecation & migration

18. Mark `ProjectionLayer` with `// Deprecated:` comment pointing to declarative approach
19. Write migration guide in `docs/guides/declarative-projections-migration.md`
20. Update `examples/system-demo/` to use declarative projections
21. Add `nix run .#coverage-gate` verification for systemadapter module

### Upstream go-cqrs-lite

22. Commit `EventWithID.OccurredAt` extension in go-cqrs-lite
23. Consider upstreaming a `System.WaitForProjectionsReady()` helper (waits for `WorkerStopped`)
24. Consider upstreaming documentation about `BlockPublishUntilSubscriberAck` + `Persistent: false` interaction

### Architecture improvements

25. Fix the two-host split brain — either disable declarative projections when ProjectionLayer is used, or auto-redirect ProjectionLayer queries to the declarative store
26. Replace `actionLevel` hardcoded mapping with a consumer-configurable action-to-role map
27. Consider using `LogBackend` for AuditLog to get proper append-only ordering
28. Split `declarations.go` into per-aggregate files
29. Add doc comments to all fold functions

### Testing robustness

30. Add concurrent dispatch test — dispatch from N goroutines, verify all projections are consistent
31. Add test for projection recovery after worker restart
32. Add test for projection with checkpoint (verify restart from checkpoint doesn't reprocess)
33. Add test for DLQ behavior when a fold handler returns an error
34. Add test for projection rebuild (`RebuildProjection` equivalent for declarative)

### Code quality

35. Consolidate `setupDeclarativeSystem` and `setupTestSystem` into shared helpers
36. Extract `waitForProjectionsReady` into a reusable test helper (maybe export it)
37. Add `// Deprecated` markers on ProjectionLayer methods
38. Run `nix run .#lint` on systemadapter module (verify against the full lint config, not just golangci-lint)
39. Run `nix run .#test` for the full workspace (verify no cross-module breakage)
40. Run `nix run .#coverage-gate` (verify the gate passes)

### Documentation

41. Update AGENTS.md with the `waitForProjectionsReady` pattern and the `WorkerStopped` insight
42. Document the `BlockPublishUntilSubscriberAck` + `Persistent: false` bus behavior in the leveraging-system-metaengine guide
43. Update the status report from the prior session with the flakiness fix
44. Add CHANGELOG entry for the declarative projections + flakiness fix
45. Document the ExternalAccountLink bug as a known issue

### Future features

46. Add support for custom roles in the declarative Authz model
47. Add `systemadapter.WireDeclarativeOnly(sys)` helper that skips ProjectionLayer
48. Consider a `systemadapter.NewDeclarativeSystem()` convenience constructor
49. Add OpenAPI schema generation for declarative query endpoints
50. Consider streaming/SSE support for declarative projection updates

---

## g) Questions (cannot figure out myself)

### 1. Should the ExternalAccountLink fix change the key format?

The current insert key is `e.ID + ":" + provider + ":" + subject`. The unlink event has `provider` and `subject` but not `e.ID` (well, it has `e.ID` from `EventWithID`, but that's the user stream ID, not the link's composite key). Options:
- **Option A:** Change the insert key to just `provider + ":" + subject` (drops the stream ID, making removes work via `deriveKeys`)
- **Option B:** Add a custom key extractor to the remove fold (requires upstream metaengine support for explicit key functions on remove folds)
- **Option C:** Keep the current hacky update-to-empty but add a post-filter in `FindUserByExternalAccount` to skip zero-value entries

Which approach do you prefer?

### 2. Should we commit the go-cqrs-lite `OccurredAt` change now, or wait?

The `EventWithID.OccurredAt` field is uncommitted in go-cqrs-lite. The local replace in `go.work` makes it work for development, but publishing a tag requires the upstream change. Should I commit it now, or is there a reason to wait (e.g., other pending changes to the same file)?

### 3. Should the declarative projections replace or coexist with ProjectionLayer?

The current design has both hosts running simultaneously. This doubles memory usage and processes every event twice. Options:
- **Replace:** Mark ProjectionLayer deprecated, remove it in v5, consumers use declarative only
- **Coexist:** Keep both, let consumers choose (current state)
- **Auto-detect:** If declarative projections are wired in DomainConfig, skip creating ProjectionLayer read models

What's the target end-state?
