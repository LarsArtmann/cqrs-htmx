# Status Report: Declarative Metaengine Projections for systemadapter

**Date:** 2026-08-09 13:20
**Session Focus:** Convert all 6 usermgmt projections from imperative `projection.Projection` implementations to declarative `system.Evolve[R]()` + `system.Lookup[R]()` / `system.QuerySet[R]()` metaengine fold declarations.

---

## a) FULLY DONE

### 1. EventWithID Extended with OccurredAt (go-cqrs-lite upstream)

**File:** `go-cqrs-lite/metaengine/projectionadapter/typed_decoder.go`

Added `OccurredAt time.Time` field to `EventWithID[P]` struct. Both `Register[E]()` and `RegisterString[E]()` now populate it from `evt.OccurredAt()`. This is a purely additive change — existing fold handlers that ignore the field are unaffected. Verified: `go build ./metaengine/projectionadapter/` passes clean.

### 2. View Structs Defined (systemadapter/views.go)

**New file** with 10 view structs covering all 6 projections:

| Struct | Projection | Key Fields |
|---|---|---|
| `TenantView` | Tenant | ID, Name, DisplayName, Suspended |
| `BotView` | Bot | ID, Name, OwnerID, TokenHash, Scopes |
| `BotTokenView` | Bot secondary index | TokenHash (hex), BotID |
| `MembershipView` | Membership | ID, ActorID, TenantID, Roles |
| `UserView` | User | ID, Email, DisplayName, Credentials, ExternalAccounts, EmailVerified, TOTPEnabled, CreatedAt, UpdatedAt |
| `CredentialView` | User sub-struct | ID, PublicKey, AttestationType, Transports, etc. |
| `ExternalAccountView` | User sub-struct | Provider, Subject, Email, DisplayName |
| `ExternalAccountLink` | User secondary index | ProviderSubject, UserID |
| `PolicyEntry` | Authz | Key, Subject, Domain, Roles |
| `AuditEntryView` | AuditLog | EventType, AggregateID, OccurredAt, Action |

**Design decision:** `TOTPSecret` intentionally excluded from `UserView` (security — secrets should not be in read model projections). `Deleted` boolean dropped from `TenantView`/`BotView` (deletion becomes a Remove fold, not a soft-delete flag).

### 3. Declarative Projections (systemadapter/declarations.go)

**New file** with `DeclarativeProjections()` returning `[]system.ProjectionDeclaration` — 13 `metaengine.Query` declarations wrapped via `system.RawQuery()`:

| Query Name | Type | Purpose |
|---|---|---|
| `tenant_by_id` | `Lookup[string, TenantView]` | Point lookup by stream ID |
| `tenants` | `Scan[TenantView]` | Filter by Name, Suspended |
| `bot_by_id` | `Lookup[string, BotView]` | Point lookup by stream ID |
| `bots` | `Scan[BotView]` | Filter by OwnerID |
| `bot_tokens` | `Scan[BotTokenView]` | Secondary index: hex token hash → bot ID |
| `membership_by_id` | `Lookup[string, MembershipView]` | Point lookup by stream ID |
| `memberships` | `Scan[MembershipView]` | Filter by ActorID, TenantID |
| `user_by_id` | `Lookup[string, UserView]` | Point lookup by stream ID |
| `users` | `Scan[UserView]` | Filter by Email, EmailVerified; sort by CreatedAt |
| `external_account_links` | `Scan[ExternalAccountLink]` | Secondary index: provider:subject → user ID |
| `authz_policy_by_id` | `Lookup[string, PolicyEntry]` | Policy lookup by aggregate stream ID |
| `authz_policies` | `Scan[PolicyEntry]` | Filter by Subject, Domain |
| `audit_log` | `Scan[AuditEntryView]` | Filter by AggregateID; sort by OccurredAt desc |

**Fold handlers** registered for all 21 identity-model event types via `metaengine.OnTyped()`:
- Insert folds: `TenantCreated`, `BotRegistered`, `MemberAdded`, `UserRegistered`, `ExternalAccountLinked`, 12 audit event types
- Update folds: `TenantSuspended`, `TenantReactivated`, `EmailChanged`, `DisplayNameChanged`, `CredentialAdded`, `CredentialRemoved`, `EmailVerified`, `TOTPEnabled`, `TOTPDisabled`, `ExternalAccountUnlinked`, `RolesUpdated`, `MemberRolesChanged`
- Remove folds: `TenantDeleted`, `BotDeleted`, `MemberRemoved`, `UserDeleted`

**Key design decision:** Authz is modeled as data (PolicyEntry with Subject/Domain/Roles) rather than delegating to the Casbin enforcer. Role inheritance is resolved at query time in the `Enforce` helper (not pre-expanded).

### 4. Query Helpers (systemadapter/queries.go)

**New file** with 19 typed Go query functions:

- **Tenant:** `FindTenantByID`, `FindTenantByName`, `AllTenants`
- **Bot:** `FindBotByID`, `FindBotsByOwner`, `FindBotByTokenHash`
- **Membership:** `FindMembershipByID`, `FindMembershipsByActor`, `FindMembershipsByTenant`
- **User:** `FindUserByID`, `FindUserByEmail`, `AllUsers`, `FindUserByExternalAccount`
- **Authz:** `FindPolicyByStreamID`, `FindPolicies`, `Enforce`
- **Audit:** `AuditEntries`, `AuditEntriesFor`, `RecentAuditEntries`

`Enforce` implements role hierarchy: super_admin (4) > admin/owner (3) > user (2) > viewer (1). Actions map to required levels: view/read (1), use/write/edit (2), manage/admin (3), super/everything (4).

### 5. DomainConfig Wiring (systemadapter/domain_config.go)

Added `Projections: DeclarativeProjections()` to the `system.DomainConfig` return. This makes `system.New()` auto-create the internal projection host with all 13 queries wired — no separate `ProjectionLayer` needed for declarative-mode consumers.

### 6. Lint Clean

`golangci-lint run` reports 0 issues (after `--fix` resolved gci/golines formatting). The module-specific `.golangci.yml` from Phase 1 handles the bridge-module exemptions.

### 7. All 7 Existing Tests Still Pass

The `ProjectionLayer` backward compatibility is fully intact. Existing tests using `pl.User.FindByID()`, `pl.Authz.Enforce()`, etc. are unaffected by the new declarative projections.

---

## b) PARTIALLY DONE

### Declarative Tests (6 new tests, FLAKY under race detector)

**File:** `systemadapter/declarative_test.go`

7 tests written:
1. `TestDeclarative_TenantRoundTrip` — create → suspend → reactivate → find by name
2. `TestDeclarative_BotRoundTrip` — register → find by ID, token hash, owner
3. `TestDeclarative_MembershipRoundTrip` — add member → find by ID, actor, tenant
4. `TestDeclarative_UserRoundTrip` — register → find by email → change email → verify
5. `TestDeclarative_AuthzEnforce` — admin allowed to manage, viewer denied manage but allowed view
6. `TestDeclarative_AuditLog` — register + change email → 2+ entries, filter by aggregate, recent
7. `TestDeclarative_AllProjectionNames` — sanity check on declaration count

**Test results:**
- **GOMAXPROCS=1:** 10/10 runs pass (all 7 tests, every run)
- **Race detector (default GOMAXPROCS):** ~40% pass rate. AuthzEnforce and AuditLog fail intermittently.

**Root cause of flakiness:** The memory engine's projection worker processes events asynchronously from the event bus. The `eventually` retry pattern (100ms initial delay + 50ms polling for 5s timeout) works when goroutine scheduling is deterministic (GOMAXPROCS=1) but loses the race under parallel scheduling. The data IS correctly stored (verified via debug tests that dump the raw engine state) — the issue is purely timing between `Dispatch()` returning and the projection worker applying the event.

### Coverage at 67.2% (BELOW 70% gate)

**Uncovered query helpers** (0% coverage):
- `AllTenants`, `AllUsers`, `FindUserByExternalAccount`
- `FindPolicyByStreamID`
- `AuditEntriesFor`, `RecentAuditEntries`

**Low-coverage fold handlers** (12-25%):
- `updateDisplayNameChanged`, `updateCredentialAdded`, `updateCredentialRemoved`
- `updateTOTPEnabled`, `updateTOTPDisabled`
- `updateExternalAccountLinked`, `updateExternalAccountUnlinked`
- `updateMembershipRoles`, `updateUserPolicy`, `updateMemberPolicy`
- `externalAccountLinkScan` update fold
- `bytesEqual` helper (0%)

**Missing test scenarios:** TOTP enable/disable, credential add/remove, external account link/unlink, membership role updates, user deletion, tenant deletion, bot deletion, RolesUpdated.

---

## c) NOT STARTED

1. **Deprecation of ProjectionLayer** — not marked deprecated yet. The old imperative host is fully functional and all existing tests depend on it.
2. **examples/system-demo update** — still uses `ProjectionLayer` pattern.
3. **CI/flake.nix integration** — coverage gate still at 70% from Phase 1; new code needs additional tests to meet it.
4. **Equivalence tests** — no test verifying that declarative queries return the same results as the old read models for the same event stream.
5. **SQLite engine test** — declarative projections only tested with memory engine.
6. **go-cqrs-lite commit** — `typed_decoder.go` changes are uncommitted upstream.
7. **systemadapter go.mod update** — may need `metaengine/v4` as a direct dep (currently indirect).

---

## d) TOTALLY FUCKED UP

### Test Flakiness Under Race Detector

**The tests are NOT reliably green under `-race`.** They pass 100% with `GOMAXPROCS=1` but ~60% failure rate with default scheduling. This is **unacceptable for CI**.

**What went wrong in the approach:**
1. First attempt: `waitForHostLive` + fixed 50ms delay → ~40% pass rate
2. Second attempt: processed-counter stabilization (200ms window) → ~20% pass rate (counter looks stable before events arrive)
3. Third attempt: `eventually` retry pattern with 20ms polling → ~30% pass rate (too aggressive polling, events not yet processed)
4. Fourth attempt: 100ms initial delay + 50ms polling → ~40% pass rate under race, 100% with GOMAXPROCS=1

**The fundamental problem:** There is a delay between `sys.CommandDispatcher().Dispatch(ctx, cmd)` returning and the event reaching the projection worker goroutine via the bus. The projection host's `WorkerState.Processed` counter only increments AFTER the event is processed. With `-race`, the goroutine scheduler introduces enough jitter that 100ms is sometimes insufficient for the event to propagate from dispatcher → event store → bus → subscriber → projection adapter → metaengine store.

**What I should have done:** Use `sys.Drain(ctx)` or the projection host's built-in drain mechanism instead of polling. Or use a synchronous in-process bus for tests. Or accept GOMAXPROCS=1 for these specific tests. Or ask go-cqrs-lite to expose a `WaitForEvents(timeout)` helper on the System.

---

## e) WHAT WE SHOULD IMPROVE

### Architecture

1. **PolicyEntry design is too simple** — the current `PolicyEntry` stores `Roles []string` per subject+domain, but the old CasbinProjection supported pattern matching and multi-domain role inheritance. The declarative version uses a fixed 5-role hierarchy. If consumers need custom roles or Casbin-style pattern policies, this won't work.

2. **AuditLog as insert-fold, not append-fold** — the plan called for `LogBackend` append semantics, but the implementation uses insert folds with composite keys (`streamID:eventType:timestamp`). This works but loses the append-only log ordering guarantee. With the memory engine it's equivalent (both use maps internally), but a LogBackend-backed engine would give proper ordered log semantics.

3. **ExternalAccountLink uses a hacky update fold** — the `ExternalAccountUnlinked` event tries to "update" the link to an empty struct instead of removing it. The remove fold can't work here because the key derivation via `deriveKeys` matches on the insert fold's key type (string), and the unlink event doesn't have the same composite key format. This needs a proper solution (either matching keys or a separate delete-by-scan).

4. **Two-host architecture persists** — the declarative projections are wired into DomainConfig, but the ProjectionLayer is still fully functional. Consumers get BOTH hosts if they call `NewProjectionLayer(sys)`. This is intentional for backward compat but creates a split brain if both are used.

### Testing

5. **No equivalence tests** — we haven't verified that `FindUserByID` returns the same data as `pl.User.FindByID()` for the same event stream. This is the most critical missing validation.

6. **No SQLite engine tests** — the memory engine stores Go structs directly; SQLite serializes to JSON. The `reifyTo`/`reifyReflect` fallback paths for JSON round-tripping are untested.

7. **Missing negative tests** — no test for `ErrNotFound` when querying a non-existent ID, no test for empty result sets, no test for filter mismatches.

### Code Quality

8. **`actionLevel` in queries.go is a simplistic hardcoded mapping** — "view"/"read" → 1, "write"/"edit" → 2, etc. This doesn't match any real authorization model. It's a placeholder that should be replaced with consumer-defined action-to-role mapping.

9. **`bytesEqual` duplicates `bytes.Equal`** — should use the stdlib.

10. **No doc comments on individual fold functions** — the function names are descriptive but a one-line comment on each would help.

---

## f) Next Steps (up to 50)

### Fix the test flakiness (CRITICAL — do this first)

1. Investigate using `sys.Drain(ctx)` after each dispatch for synchronous-like behavior
2. Or: file an upstream issue/PR for a `System.WaitForProjectionDrain(name, timeout)` helper
3. Or: use `testing.Short()` skip with `-race` flag as a stopgap
4. Or: add `//go:slow` build tag and use `GOMAXPROCS=1` via `runtime.GOMAXPROCS(1)` in TestMain
5. Investigate if the projection host has a `Flush()` or `Sync()` method not yet discovered
6. Check if `sys.ProjectionHost()` has a method to wait for a specific checkpoint/version

### Coverage to 70%+

7. Add test for TOTP enable/disable cycle on UserView
8. Add test for credential add/remove on UserView
9. Add test for external account link/unlink on UserView
10. Add test for `FindUserByExternalAccount` secondary index
11. Add test for membership role update (`UpdateMemberRoles`)
12. Add test for `RolesUpdated` event on PolicyEntry
13. Add test for user deletion (UserView + PolicyEntry removed)
14. Add test for tenant deletion (TenantView removed)
15. Add test for bot deletion (BotView + BotTokenView removed)
16. Add test for `AllTenants`, `AllUsers` queries
17. Add test for `FindPolicyByStreamID`
18. Add test for `AuditEntriesFor` with specific aggregate ID
19. Add test for `RecentAuditEntries` with limit
20. Replace `bytesEqual` with `bytes.Equal`

### Equivalence tests

21. Write a test that dispatches a complex event stream, then queries BOTH `pl.User.FindByID()` and `systemadapter.FindUserByID()` and asserts equality
22. Same for Membership, Tenant, Bot
23. Same for Authz Enforce (old Casbin vs new PolicyEntry)

### SQLite engine tests

24. Add a `setupDeclarativeSystemSQLite(t)` helper using SQLite in-memory
25. Run all 7 declarative tests with SQLite engine
26. Verify JSON round-tripping works for all view types
27. Verify filterable fields work with SQL pushdown

### ProjectionLayer deprecation

28. Add `// Deprecated: Use declarative projections via DomainConfig instead.` to `ProjectionLayer`
29. Update `examples/system-demo/main.go` to use declarative queries
30. Write a migration guide: `docs/guides/declarative-projections.md`
31. Check if adminui/dashboardui reference ProjectionLayer (they shouldn't — they use their own read models)

### Upstream go-cqrs-lite

32. Commit the `EventWithID.OccurredAt` extension
33. Consider adding `EventWithID.Metadata` field for full event context
34. Consider adding a `System.WaitForProjections(timeout)` convenience method
35. Consider adding `projectionhost.Host.WaitForProcessed(timeout)` method

### PolicyEntry improvements

36. Support consumer-defined role hierarchies instead of hardcoded 5-role mapping
37. Add `FindPoliciesByRole(ctx, sys, role)` helper
38. Consider pre-expanding role inheritance at insert time (admin → also insert user, viewer)
39. Add pattern matching support if needed (or document it as not supported)

### AuditLog improvements

40. Switch to `LogBackend` append fold if memory engine supports it (it does — `LogAppend`/`LogTail` confirmed)
41. Add `AuditEntriesFor(ctx, sys, aggID, eventType)` filtered variant
42. Add `AuditCount(ctx, sys)` helper

### CI/Integration

43. Update flake.nix coverage gate if needed
44. Update `.github/workflows/ci.yml` if new files need coverage
45. Run `nix run .#lint` on systemadapter module
46. Run `nix run .#build` workspace-wide to verify no breakage
47. Run `nix run .#test` workspace-wide

### Documentation

48. Update `docs/plans/2026-08-09_metaengine-fold-declarations.md` with completion status
49. Update systemadapter package doc comment with declarative projection usage example
50. Add a section to AGENTS.md about the declarative projection system

---

## g) Questions I CANNOT Answer Myself

### 1. Should the tests use GOMAXPROCS=1 as a permanent workaround, or should I investigate adding a drain/wait mechanism to go-cqrs-lite?

The flakiness is a real concurrency issue in the projection host's async event processing. `GOMAXPROCS=1` makes it 100% deterministic but feels like papering over the problem. The "right" fix is likely upstream (a `Host.WaitForEvents(timeout)` or similar), but that's a go-cqrs-lite change I can't make unilaterally.

### 2. Should PolicyEntry model role inheritance at insert time (pre-expanded) or at query time (traversal)?

Pre-expansion: when inserting "admin", also insert implicit "user" and "viewer" entries. Enforcement is a flat lookup. But it complicates updates (need to re-expand on RolesUpdated) and uses more memory.

Query-time: keep roles as-is, traverse hierarchy during `Enforce`. Simpler inserts, but enforcement is O(roles × hierarchy_depth). For 5 roles this is trivial; for custom consumer-defined hierarchies it could matter.

The old Casbin enforcer did this via Casbin's built-in role manager. The declarative version needs a deliberate choice.

### 3. Should I keep the ProjectionLayer indefinitely or set a deprecation timeline?

The plan says "keep as backward-compat shim." But the declarative projections now duplicate ALL the logic. Every event handler change needs to be reflected in both places. Should I:
- (a) Keep both forever (maintenance burden)
- (b) Mark deprecated now, remove in v5
- (c) Make ProjectionLayer internally delegate to the declarative queries (best of both, but requires bridging `map[string]any` → typed views)

Option (c) is the cleanest but most work. I can't decide this without knowing the v5 roadmap.
