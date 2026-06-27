# Comprehensive Execution Plan — cqrs-htmx

> **Generated:** 2026-06-27 23:01 CEST
> **Source:** Every OPEN item in TODO_LIST.md + ROADMAP.md + status-report findings
> **Granularity:** Every task ≤ 12 min. All TODOs included.
> **Sort:** Pareto tier (1%→51%, 4%→64%, 20%→80%, rest), then by impact/effort ratio (value density).

## Pareto Breakdown

| Tier | What delivers it | Why it's the leverage |
|------|------------------|-----------------------|
| **1% → 51%** | Consumer-build CI gate + type ActorID in context + foldUser error | Prevents every future "shipped broken" failure + closes the #1 type-safety hole |
| **4% → 64%** | TenantState impossible-states + BotState.OwnerID typed + Authz TenantID typed + adminui coverage ≥75% | Makes wrong code unrepresentable + biggest test-risk surface |
| **20% → 80%** | Full type-safety sweep + extract *http.Request + service-level tests + OTel + migration guide | Production hardening + adopter unblock |
| **Rest** | Snapshot, Redis stores, BadgerDB, streaming replay, schema validator, migration tooling | Scale + advanced ES |

## Sorting Key

- **Impact:** Critical > High > Medium > Low
- **Effort:** estimated minutes (all ≤12 per task)
- **Value density:** Impact ÷ Effort (highest first within each tier)
- **Customer value:** "Why it matters" column

---

## TIER 1 — The 1% that delivers 51% (critical quick wins)

| # | Task (≤12 min) | Parent TODO | Impact | Effort | Why it matters |
|---|----------------|-------------|--------|--------|----------------|
| 1 | Add `GOWORK=off go build ./...` per module to CI workflow | Consumer-build gate | Critical | 8 | Prevents shipping unbuildable modules — the recurring 3-day failure |
| 2 | Add `go mod tidy -diff` check per module to CI | Consumer-build gate | Critical | 5 | Catches ghost deps / stale go.mod before merge |
| 3 | Add go.sum cross-module consistency check to CI | Consumer-build gate | Critical | 3 | Prevents go.sum drift between modules |
| 4 | Change `context.go` ActorID/ImpersonatorID from `string` to branded types + update signatures | Type ActorID in context (CRITICAL) | Critical | 10 | Closes #1 type-safety hole — raw string loses all type guarantees |
| 5 | Update all ActorID/ImpersonatorID call sites for branded types | Type ActorID in context | Critical | 8 | Completes the type-safety change across codebase |
| 6 | Update ActorID context tests for branded types | Type ActorID in context | Critical | 6 | Proves the change works |
| 7 | Change `foldUser` to return error on unknown events | foldUser error (HIGH) | High | 6 | Match foldMembership/Tenant/Bot — prevents silent data loss |
| 8 | Update decider wiring to handle foldUser error | foldUser error | High | 6 | Propagate error instead of silent no-op |
| 9 | Add test: foldUser rejects unknown event type | foldUser error | High | 5 | Regression guard |

## TIER 2 — The 4% that delivers 64% (high-impact short tasks)

| # | Task (≤12 min) | Parent TODO | Impact | Effort | Why it matters |
|---|----------------|-------------|--------|--------|----------------|
| 10 | Redesign `TenantState` to make `Suspended+Deleted` unrepresentable | Impossible TenantState | High | 10 | Impossible states can't be bugs |
| 11 | Update `foldTenant` + decide functions for new TenantState | Impossible TenantState | High | 8 | Wire the type change through domain layer |
| 12 | Update tenant state tests | Impossible TenantState | High | 6 | Verify invariant holds |
| 13 | Change `BotState.OwnerID` from `string` to `UserID` | BotState typed OwnerID | High | 8 | Type-safe bot ownership |
| 14 | Update `BotRegisteredPayload` + Bot readmodel for typed OwnerID | BotState typed OwnerID | High | 8 | Consistent across events + projections |
| 15 | Update `service_bot.go` + decider for typed OwnerID | BotState typed OwnerID | High | 6 | Service-layer consistency |
| 16 | Update bot tests for typed OwnerID | BotState typed OwnerID | High | 6 | Regression guard |
| 17 | Change `authz_roles.go` domain params from `string` to `TenantID` | Authz TenantID typed | High | 8 | Type-safe authorization domain scoping |
| 18 | Change `authz_policies.go` domain params to `TenantID` | Authz TenantID typed | High | 8 | Policy methods type-safe |
| 19 | Update CasbinProjection + all call sites for typed TenantID | Authz TenantID typed | High | 10 | Complete the type change |
| 20 | Update authz tests for typed TenantID | Authz TenantID typed | High | 6 | Regression guard |
| 21 | Add validation to `NewActorID` (reject invalid kind+raw pairing) | Validate NewActorID | High | 10 | Prevent constructing invalid actor identities |
| 22 | Update NewActorID call sites for validation | Validate NewActorID | High | 5 | Complete the guard |
| 23 | Make `actorKindFromString` return error on unknown kind | Validate actorKindFromString | Medium | 5 | No silent defaulting to ActorUser |
| 24 | Update foldMembership caller for actorKindFromString error | Validate actorKindFromString | Medium | 5 | Propagate the error |
| 25 | Consolidate `ErrUnauthorized` (root owns; usermgmt wraps/re-exports) | Duplicate sentinels | Medium | 10 | `errors.Is` works across module boundary |
| 26 | Add cross-module `errors.Is(ErrUnauthorized)` test | Duplicate sentinels | Medium | 5 | Proves boundary works |
| 27 | Write v2→v3 import-path migration section | Migration guide | High | 10 | Unblocks adopters upgrading |
| 28 | Write v2→v3 bus/projection migration section | Migration guide | High | 10 | Unblocks event-sourcing migration |
| 29 | Write v2→v3 breaking-changes checklist | Migration guide | Medium | 8 | Adopters know exactly what changed |

## TIER 2b — adminui coverage to ≥75% (biggest test-risk surface)

| # | Task (≤12 min) | Parent TODO | Impact | Effort | Why it matters |
|---|----------------|-------------|--------|--------|----------------|
| 30 | Test `membersIndex` handler (empty + populated + HTMX partial) | adminui coverage | High | 12 | 0% → covered; tenant-admin core path |
| 31 | Test `membersAdd` handler (happy + unknown email) | adminui coverage | High | 12 | 0% → covered; destructive write path |
| 32 | Test `membersRemove` handler | adminui coverage | High | 10 | 0% → covered; destructive write path |
| 33 | Test `membersUpdateRole` handler | adminui coverage | High | 10 | 0% → covered; privilege change |
| 34 | Test `tenantNew` + `tenantCreate` handlers | adminui coverage | Medium | 12 | 0% → covered; super-admin create |
| 35 | Test 403 authz denial paths (unauthorized + forbidden) | adminui coverage | High | 10 | Proves authz guard works on every route |
| 36 | Test tenant-scope isolation (tenant-admin can't see other tenants) | adminui coverage | High | 12 | Security: cross-tenant data leak prevention |
| 37 | Test `defaultAuthorizer` + `RequireAnyRole` (0% → covered) | adminui coverage | Medium | 10 | Authorizer logic currently untested |

## TIER 3 — The 20% that delivers 80% (important medium tasks)

| # | Task (≤12 min) | Parent TODO | Impact | Effort | Why it matters |
|---|----------------|-------------|--------|--------|----------------|
| 38 | Replace `string` with `Email` branded type in User struct + events | Email branded type | Medium | 10 | Type-safe email throughout domain |
| 39 | Update decider/fold/readmodel for Email type | Email branded type | Medium | 8 | Domain-layer consistency |
| 40 | Update service methods for Email type | Email branded type | Medium | 8 | Service-layer consistency |
| 41 | Update tests for Email type | Email branded type | Medium | 6 | Regression guard |
| 42 | Change `BeginRegistration`/`FinishRegistration` to take typed data not `*http.Request` | Extract *http.Request | High | 10 | Keep service layer transport-agnostic |
| 43 | Parse request in `webauthn_http.go`, pass typed data to service | Extract *http.Request | High | 10 | HTTP concern stays in HTTP layer |
| 44 | Update `webauthn_service.go` internals (no *http.Request) | Extract *http.Request | High | 8 | Clean service boundary |
| 45 | Update WebAuthn tests for new signatures | Extract *http.Request | High | 10 | Regression guard |
| 46 | Define `WebAuthnSessionStore` interface (in-memory impl satisfies it) | Ephemeral store interfaces | High | 8 | Enables Redis/SQL alt for multi-instance |
| 47 | Define `VerificationTokenStore` interface | Ephemeral store interfaces | High | 6 | Enables multi-instance verification |
| 48 | Define `LockoutStore` interface | Ephemeral store interfaces | Medium | 6 | Enables distributed lockout |
| 49 | Define `TOTPStore` interface (if separate from session) | Ephemeral store interfaces | Medium | 5 | Interface completeness |
| 50 | Make `LastEventIDFromRequest` delegate to `SSEStream.LastEventID()` | LastEventID dedup | Low | 5 | Remove byte-identical duplication |
| 51 | Remove deprecated `ClientIP()` wrapper (verify zero callers first) | Remove ClientIP | Low | 6 | Dead code removal |
| 52 | Update CHANGELOG for ClientIP removal | Remove ClientIP | Low | 4 | Honest changelog |
| 53 | Write godoc `ExampleApp` for App builder entry point | Godoc examples | Medium | 8 | Discoverable API docs |
| 54 | Write godoc `ExampleHandler` for handler options | Godoc examples | Medium | 8 | Discoverable API docs |
| 55 | Write godoc `ExampleService` for usermgmt Service | Godoc examples | Medium | 8 | Discoverable API docs |
| 56 | Write VERSIONING.md (semver policy) | VERSIONING.md | Low | 10 | Documented versioning contract |
| 57 | Test `BeginImpersonation` happy path through full dispatch | Impersonation tests | High | 12 | Proves end-to-end impersonation works |
| 58 | Test `EndImpersonation` through dispatch | Impersonation tests | High | 8 | Proves session cleanup works |
| 59 | Test impersonation guards (no caller, no target, no reason) | Impersonation tests | High | 10 | Negative-path coverage |
| 60 | Test `AddMember` through full dispatch | Membership tests | High | 12 | Proves membership write path |
| 61 | Test `UpdateMemberRoles` through dispatch | Membership tests | High | 10 | Proves role change path |
| 62 | Test `RemoveMember` through dispatch | Membership tests | High | 10 | Proves membership removal |
| 63 | Set up projection-replay test (journal events + live events) | Projection replay test | Medium | 12 | Proves dedup correctness |
| 64 | Test projection dedup (journal replay vs live subscribe) | Projection replay test | Medium | 10 | No double-processing of events |
| 65 | Write OtelBeforeDispatch/OtelAfterDispatch hook factories | Wire OpenTelemetry | Medium | 12 | Production tracing with zero SDK dep |
| 66 | Add OTel example test with real `otel.Tracer` | Wire OpenTelemetry | Medium | 10 | Proves the wiring works |
| 67 | Document OTel wiring in docs | Wire OpenTelemetry | Low | 8 | Adopter guidance |
| 68 | Write `foldTenant` property-based tests (rapid) | Property tests | Medium | 12 | Verify fold invariants hold for arbitrary inputs |
| 69 | Write `foldBot` property-based tests | Property tests | Medium | 10 | Verify fold invariants |
| 70 | Write `foldMembership` property-based tests | Property tests | Medium | 10 | Verify fold invariants |
| 71 | Fuzz test projection dedup map with arbitrary event IDs | Fuzz tests | Medium | 12 | Crash-safety under malformed input |
| 72 | Fuzz test identity model deciders with arbitrary commands | Fuzz tests | Medium | 12 | Crash-safety under malformed input |
| 73 | Enable `revive:exported` in `.golangci.yml` + run | Exported linter | Low | 5 | Catches missing godoc on exports |
| 74 | Fix revive:exported violations | Exported linter | Low | 12 | Clean exported API docs |
| 75 | Verify marker types exported in go-cqrs-lite v3.1.0 | BrandNamer | Medium | 5 | Confirm upstream support exists |
| 76 | Wire `BrandNamer` for root UserID/CorrelationID/RequestID | BrandNamer | Medium | 10 | Proper branded-type string representation |

## TIER 4 — The rest (planned: v3.2.0 / v3.3.0 / v4.0.0)

| # | Task (≤12 min) | Parent TODO | Impact | Effort | Why it matters |
|---|----------------|-------------|--------|--------|----------------|
| 77 | Wire flake `#coverage-gate` app into GitHub Actions | Coverage CI gate | Medium | 8 | Auto-fail on coverage regression |
| 78 | Design Prometheus metrics middleware (dispatch latency + error rate) | Prometheus metrics | Medium | 10 | Production observability |
| 79 | Implement Prometheus metrics middleware | Prometheus metrics | Medium | 12 | Dispatch latency + error rates |
| 80 | Add Prometheus example + docs | Prometheus metrics | Low | 8 | Adopter guidance |
| 81 | Research go-cqrs-lite snapshot API + design integration | Snapshot integration | Medium | 10 | Cut startup replay time |
| 82 | Wire snapshot store into EventSourcedSetup | Snapshot integration | Medium | 12 | Faster cold starts |
| 83 | Add snapshot integration test | Snapshot integration | Medium | 12 | Verify snapshot+replay correctness |
| 84 | Implement `RedisSessionStore` adapter | Redis session store | Medium | 12 | Distributed deployments |
| 85 | Add Redis connection pooling + TTL to session store | Redis session store | Medium | 10 | Production Redis patterns |
| 86 | Test RedisSessionStore | Redis session store | Medium | 12 | Regression guard |
| 87 | Implement `RedisOAuth2StateStore` | Redis OAuth2 state | Low | 10 | Multi-instance OAuth2 |
| 88 | Test RedisOAuth2StateStore | Redis OAuth2 state | Low | 10 | Regression guard |
| 89 | Create PostgreSQL session store preset (reduce boilerplate) | PG session preset | Low | 8 | Consumer DX |
| 90 | Add PG session preset docs | PG session preset | Low | 5 | Adopter guidance |
| 91 | Research BadgerDB + design event store adapter | BadgerDB store | Low | 8 | Embedded-store alternative |
| 92 | Implement BadgerDB event store adapter | BadgerDB store | Low | 12 | Embedded persistence |
| 93 | Replace `journal.ReadAll` with `ReadFrom` streaming in StartProjections | Streaming replay | Medium | 10 | Lower memory on large event stores |
| 94 | Test streaming replay with large event sets | Streaming replay | Medium | 12 | Verify no OOM on 10K+ events |
| 95 | Profile dispatch + decode hot paths with pprof | Profile hot paths | Low | 10 | Find bottlenecks |
| 96 | Optimize identified dispatch/decode bottlenecks | Profile hot paths | Low | 12 | Performance gains |
| 97 | Write benchmark seeding 10K+ events | Replay benchmark | Low | 12 | Baseline projection replay perf |
| 98 | Run + analyze + document replay benchmark | Replay benchmark | Low | 8 | Documented perf characteristics |
| 99 | Evaluate `stack.Materialize` fit for 12-event read models | stack.Materialize | Low | 10 | Declarative read model option |
| 100 | Prototype one read model with Materialize | stack.Materialize | Low | 12 | Proof of concept |
| 101 | Replace manual replay with `CatchUpSubscriber` | CatchUpSubscriber | Low | 12 | Simpler projection setup |
| 102 | Test CatchUpSubscriber dedup behavior | CatchUpSubscriber | Low | 10 | Verify no double-processing |
| 103 | Design schema/v3 payload validator at registration | Schema validator | Medium | 10 | Validate events at registration time |
| 104 | Implement schema/v3 payload validator | Schema validator | Medium | 12 | Catch invalid payloads early |
| 105 | Add TestMain with Postgres test container | Real PG tests | Medium | 12 | Test against real database |
| 106 | Run event store + read model tests against real Postgres | Real PG tests | Medium | 12 | Catch driver-specific bugs |
| 107 | Evaluate goose / golang-migrate / gnorm | Migration tooling | Medium | 8 | Choose migration framework |
| 108 | Add initial migration set | Migration tooling | Medium | 10 | Versioned schema migrations |

---

## Summary

| Tier | Task count | Total effort (min) | Avg/task |
|------|-----------|--------------------|----------| 
| 1% → 51% (Tier 1) | 9 | 57 | 6.3 |
| 4% → 64% (Tier 2) | 20 | 176 | 8.8 |
| adminui coverage (Tier 2b) | 8 | 88 | 11.0 |
| 20% → 80% (Tier 3) | 39 | 352 | 9.0 |
| Rest (Tier 4) | 32 | 322 | 10.1 |
| **Total** | **108** | **995** | **9.2** |

**108 tasks, all ≤12 min, ~16.5 hours total.** Every open TODO is included and split.

_Arte in Aeternum._
