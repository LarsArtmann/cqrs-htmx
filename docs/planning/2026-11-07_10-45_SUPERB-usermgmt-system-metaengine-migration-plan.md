# SUPERB plan — usermgmt → go-cqrs-lite system/metaengine migration (TODO row 110 deep-dive delivered)

**Created:** 2026-11-07, 10:45 CET · **Repo:** `/home/lars/projects/cqrs-htmx`, branch `master`
**Status:** PLAN (nothing executed). Pinned surface audited: `system/v4 v4.17.0`, `metaengine/v4 v4.22.0`, `metaengine/projectionadapter/v4 v4.6.2` (systemadapter/go.mod:12-24).
**ADR-0051 go-criterion verdict (re-checked 2026-11-07 at source): PARTIALLY MET** — see §6.

---

## 0. Thesis

usermgmt hand-wires ~10 infrastructure seams in `NewService`→`NewEventSourcedSetup` (store, bus, 4 decider repos, dispatcher, projection host, checkpoints, DLQ, drain, 6 projections) that `system.New(systemadapter.DomainConfig(), deployment)` already composes declaratively — equivalence-tested since the systemadapter module landed. The migration is not a rewrite: events stay the source of truth, the journal is untouched, and the legacy path keeps compiling until the v5 bundle. The blocker framing in TODO rows 110/111 is **stale in usermgmt's favor**: the checkpoint half of the old criterion has dissolved at the pinned version (engine-backed `system_checkpoints`, ADR-0142), the index half is met (`PlannedIndex`), and the literal `Hydrator` clause is met *by construction* on persistent engines — leaving one honest residual (no enforced checkpoint/read-model durability pairing) that this plan retires with a test, not a wait.

## 1. Non-goals

- No journal migration, no event re-encoding: mixed CBOR/JSON journals are already valid (ADR-0059) and every declarative fold decodes via encoding stamps (§3 F9).
- No change to the SSE envelope (golden-pinned) or the sync-pull wire (ADR-0056).
- No v4 removal of `NewEventSourcedSetup`/SQL read models — they ride until the v5 bundle (ADR-0051 disposition stands).
- No `ProjectionLayer` resurrection; its removal (TODO row 118) is assumed.
- No new usermgmt public API beyond one experimental constructor (W2, owner-gated D1).

## 2. Invariants

1. `Service`'s public API (21 commands, session/OAuth2/WebAuthn/audit surfaces) survives byte-compatible.
2. The journal is never rewritten by the migration; replay always reconstructs both paths.
3. Equivalence-or-no-cutover: the declarative path ships behind `TestDeclarative_EquivalenceWithProjectionLayer` (systemadapter/declarative_test.go:660) plus the new W1 probes.
4. Rollback = stop `sys`, restart legacy `Service` on the same journal (read models rebuild by replay — §7 R5).
5. Honest counting: V007 findings are only declared "evaporated" when the owning files are deleted, not when shadowed.

## 3. Verified fact base (all read at source this session)

### The bridge (systemadapter)

| # | Fact | Evidence |
| --- | --- | --- |
| F1 | `DomainConfig(opts...)` returns a `system.DomainConfig` pre-wiring: commands closure, event schema, type decoder, declarative projections, checkpoint store, curated host options (DLQ threshold 10 / 3 restarts / batch 256) | systemadapter/domain_config.go:58-94 |
| F2 | 4 deciders (`User/Membership/Tenant/Bot`) + 21 `RegisterCommand` calls registered on the system | domain_config.go:98-146; 21 = count of `RegisterCommand` in file |
| F3 | `NewProjectionLayer` is `// Deprecated:` — "superseded by the declarative projections path… retirement scheduled for the v5 removal bundle"; it defaults to in-memory checkpoint+DLQ stores | systemadapter/projections.go:81-91, 139-148 |
| F4 | Declarative path = 13 `system.RawQuery` declarations (tenant×2, bot×3 incl. `bot_tokens` secondary index, membership×2, user×2, authz×2, audit×1) | systemadapter/declarations.go:23-44; declarations_identity.go:130-133 |
| F5 | `FindByEmail` is declaratively a filter on the `users` scan (`FilterOnField("Email", FilterEq)`), served by `FindUserByEmail` via `system.Where("Email", …)`; 8 deployment presets exist (Memory/SQLite/SplitSQLite/Pebble/BBolt/Postgres/DuckDB/Turso) | declarations_user.go:64-68; systemadapter/queries.go:94-103; presets.go:34-163 |
| F6 | Equivalence/durability tests exist: equivalence-with-layer, SQLite lifecycle, SQLite authz, durable-checkpoint-across-restart | systemadapter/declarative_test.go:660, 768, 855; domain_config_restart_test.go:130 |
| F7 | Type decoder maps **27** event types; schema declares the same 27 — but doc comments say "21 events"/"25 events"/"24 types" in three places (stale doc rot) | type_decoder.go:18-27 (27 `Register`); schema.go:29-68 (27 `Event`), schema.go:18 says 24; domain_config.go:5-6, 36-37 |

### usermgmt wiring (the deep-dive TODO row 110 said was missing)

| # | Fact | Evidence |
| --- | --- | --- |
| F8 | `NewEventSourcedSetup` composes: default store→`wrapEventStore`, default watermill bus→`applyBusMiddleware`, 4 repos via `buildDeciderRepositories`, 4 read models (+SQL wrappers when `ReadModelDB` set), then `StartProjections` | usermgmt/es_setup.go:237-277 |
| F9 | `NewService` adds: session store+TTL defaults, dispatcher + `commandAuditMiddleware` + commandlifecycle pair + consumer middleware, 4 `Register*Commands` (12+3+4+2 = **21**), audit-chained session store (`newAuditingSessionStore`), WebAuthn/verification/OAuth2 wiring | usermgmt/service_factory.go:85-200, 114-119, 126-151, 183; es_dispatch.go:15 (12×`RegisterTyped`), es_membership_dispatch.go (3), es_tenant_dispatch.go (4), es_bot_dispatch.go (2) |
| F10 | Projection host factory: hydration-before-start when checkpoints configured, `hydrationAwareCheckpointStore` (checkpoint only `Hydrator` projections), SQLite-only durable DLQ, blocking `waitForDrain` read-your-writes | usermgmt/es_projection_setup.go:37-70, 103-195, 234-279; sql_hydrate.go:15-67 |
| F11 | Legacy checkpoint keys = projection names: `user-read-model`, `membership-read-model`, `tenant-read-model`, `bot-read-model`, `casbin-projection`, `audit-log`; declarative keys = query names (`user_by_id`, `users`, `bot_tokens`, …) — disjoint ⇒ one-time full replay on swap | es_readmodel.go:58, es_membership_readmodel.go:40, es_tenant_readmodel.go:44, es_bot_readmodel.go:48, es_casbin_projection.go:36, audit_log.go:106; declarations.go:23-44 |
| F12 | Domain truth: 4 deciders + `Auth` stream type unused by deciders; **21 command constants, 27 event constants**; 21 `Decide*` functions in the es_decide* family | identity-model/constants.go:9-13, 48-71, 15-46; count across es_decide*.go/es_{membership,tenant,bot}_decide.go |
| F13 | V007 = 78 upgrade-tool / 68 staticcheck (incl. tests) / 41 lint findings; all 41 lint findings in usermgmt: 5 stack (clusters 2+3, deprecation-only) + 36 view-tier (`sql_readmodel.go`, `sql_readmodel_extra.go`, `sql_readmodel_mysql.go`); stable at 2026-11-04 | docs/guides/v5-removal-inventory.md §5c:158-187 |

### Upstream at pin (criterion audit)

| # | Fact | Evidence |
| --- | --- | --- |
| F14 | `LayoutPlan{Collection,Table,Columns,Indexes []PlannedIndex}` + `BuildLayoutPlan` (filter/sort fields → planned columns, one index per column) + typed builders exist | metaengine/v4@v4.22.0/layout.go:20-41, 49-101; layout_typed.go:60, 140 |
| F15 | **`system.New` no longer defaults to an in-memory checkpoint store**: `resolveCheckpointStore` = consumer store → **engine-backed `system_checkpoints` Map collection (ADR-0142)** → memory only when no engine carries the Map ADT | system/v4@v4.17.0/checkpoint.go:49-61; checkpoint_engine.go:14-42, 135-150 |
| F16 | `System.ResetProjection(ctx, name)` (checkpoint + `Resettable` read-model reset) is in the pinned version; projectionadapter implements `Resettable` via `metaengine.Store.Reset` | system/v4@v4.17.0/lifecycle.go:164-176; projectionadapter/v4@v4.6.2/reset.go:8-36 |
| F17 | **No `Hydrator`-named declarative equivalent exists** — zero `Hydrat*` symbols in metaengine v4.22.0 / system v4.17.0 (grep). The restart contract it served is met by construction on persistent engines (state lives in the engine; F6 restart test) — but volatile-engine placement is only a planning diagnostic (WARN/INFO), never a gate | grep over both modules; metaengine rule_durability.go:9-21 |
| F18 | Declarative folds decode payloads via `event.DecodePayloadAuto` (encoding-stamp driven) — mixed CBOR/JSON journals decode correctly; usermgmt already pins this (`TestMixedJournal_DecodesViaEncodingStamps`) | projectionadapter/v4@v4.6.2/typed_decoder.go:60-72; usermgmt/cbor_payload_test.go:113 |
| F19 | `system` exposes `ProjectionHost()`, `Bus()`, `EventStore()`, `HealthCheck` (engine pings) and `UseCommandMiddleware` — but **no** event-store wrapper / bus publish-handler middleware seam (SecurityHooks/ADR-0011 gap) | system/v4@v4.17.0/system.go:223-250, 237-239; introspection.go:156-176; grep for `event.PublishMiddleware|StoreWrapper` = 0 hits; config_types.go:108-109 (command middleware only) |
| F20 | SSE envelope is metadata-only, byte-golden-pinned, and payload never crosses it — projection-swap cannot drift it as long as event IDs/versions are identical | transport/event_sse_test.go:118-150; ADR-0059 context §"SSE" |

**NV (not verified):** whether `system.New` wires aggregate snapshotting equivalent to `SnapshotConfig` (es_setup.go:115-118) — upstream snapshot module exists, wiring unverified; ADR-0123/ADR-0126/ADR-0142/ADR-0149 texts (they live in the go-cqrs-lite repo, cited here only via second-hand references); the 2026-10-11 status doc `2026-10-11_19-46_state-of-affairs-*.md` cited by row 111 is not at that path in docs/status/archived/ today.

## 4. The seam map (what system.New replaces vs what usermgmt keeps)

**Replaced (10 seams):** store default+wrap → deployment engines · bus default+middleware-apply → system bus · 4 decider repos → `RegisterDecider` · 21 command registrations → `registerAllCommands` · dispatcher wiring → `UseCommandMiddleware` · projection-host factory+DLQ → internal host + curated defaults · checkpoint+hydration-awareness → `resolveCheckpointStore` · 6 projection registrations → 13 fold declarations · `waitForDrain` → `sys.Start` + query-poll (documented divergence, guide §Drain) · projection health → `ProjectionHost().Status()` + `HealthCheck`.

**usermgmt-owned, not covered (keep):** sessions+TTL+auditing session store (F9) · WebAuthn/verification/OAuth2/impersonation/lockout sub-services · the audit chain (`commandAuditMiddleware`, `authAuditor` — runs fine over `sys.EventStore()/Bus()`, F19) · live Casbin enforcer object (declarative `Enforce` is a per-call policy query, queries.go:171-177) · SecurityHooks sign/encrypt (NO seam upstream, F19 — R3) · SQL read models (cluster 1, F13) · `SnapshotConfig` (NV).

## 5. V007 disposition (row 111's 68 findings)

**Recommendation: EVAPORATE at the v5 cut** — views move to metaengine Query (declarations already exist, F4/F5; engines cover SQLite/Postgres/MySQL/DuckDB/Turso for multi-process, F5), and `sql_readmodel.go` + `sql_readmodel_extra.go` + `sql_readmodel_mysql.go` are deleted with the v5 bundle, retiring all 36 view-tier lint findings (F13) and their 68 staticcheck siblings. Keeping the SQL read models would preserve a second view tier the declarative path already equivalence-matches — pure carrying cost. The SQL **session** store is NOT cluster 1 and stays regardless. `kv.ViewStore` (the R9 view cache) survives v5 by design (v5-removal-inventory §5c) and becomes moot here only if the cache moves to engine prefetch — out of scope.

## 6. GO/NO-GO — keyed to the ADR-0051 criterion (docs/adr/0051:68-76)

Criterion: "metaengine layout-planning API covers secondary-index semantics … AND the hydration contract (`Hydrator`) has a declarative equivalent."

- **Secondary-index semantics: MET.** `PlannedIndex`/`LayoutPlan` (F14) + `FilterOnField`→indexed column + `bot_tokens`/`Email` filter declarations in-tree (F4/F5). This half was already met in 2026-10 (row 111 concedes `PlannedIndex` exists).
- **Checkpoint precondition: DISSOLVED.** Row 111's "system.New still has an in-memory checkpoint store" is **false at pin v4.17.0** — engine-backed `system_checkpoints` is the default on Map-ADT engines; memory is the no-engine fallback only (F15). This contradicts both row 111 re-checks and v5-removal-inventory §5c's 2026-10-03 note.
- **Declarative Hydrator equivalent: MET BY CONSTRUCTION, NOT BY NAME.** No interface exists (F17); on persistent engines there is nothing to hydrate (state is engine-resident; F6/F16), and rebuilds ride `ResetProjection`. The residual honesty: volatile-engine placement + durable checkpoints can still starve a collection, and the planner only WARNs (F17).

**Verdict: PARTIALLY MET. Gate:** W0-W2 are **GO** now (docs, probes, façade — all reversible). W3-W4 (read-tier cutover, file deletion) are **GO only when** (a) the W1.2 starvation probe proves the failure mode is absent on preset deployments or loudly diagnosed, and (b) D1/D2 owner decisions are recorded. A literal upstream `Hydrator` interface is **not** required — by-construction satisfaction plus the durability probe is the honest bar; if the probe shows silent starvation, the gate flips to NO-GO pending an upstream durability-pairing rule (W5 ask).

## 7. Waves (coarse 30-100 min; fine ≤12 min; pareto-tiered)

### W0 — truth on disk (1%: ~60 min; docs only)
| # | Task | Done when |
| --- | --- | --- |
| W0.1 | Correct rows 110/111 + v5-removal-inventory §5c checkpoint claim (F15) and record the §6 verdict | TODO_LIST rows annotated; inventory erratum |
| W0.2 | Fix systemadapter doc-rot counts (F7: 20/21/25/24 → 21 commands, 27 events) | 3 comments corrected |
| W0.3 | Owner-decision packet D1-D5 (§8) filed | docs/proposals row |

### W1 — risk retirement probes (4%: ~4 h; the 20%-tier's de-risk)
| # | Task | Done when |
| --- | --- | --- |
| W1.1 | Mixed-journal equivalence: replay a mixed CBOR/JSON journal through both paths, compare every view (rides F18) | systemadapter test green |
| W1.2 | **Starvation probe (the honest gap):** declare a fold on a volatile engine with engine-backed checkpoints; assert either full-replay-consistency or a surfaced diagnostic — never silent emptiness | probe test + verdict recorded in §6 |
| W1.3 | SSE/event-envelope parity: dispatch identical commands via both paths; diff `DomainEventToSSE` bytes (F20) | parity test green |
| W1.4 | Checkpoint-key inventory table (F11) + one-time-replay note in the migration guide | guide § |

### W2 — façade spike (20%: ~2 days)
| # | Task | Done when |
| --- | --- | --- |
| W2.1 | Experimental `NewServiceFromSystem(sys)` behind `//go:build experimental` (D1): Service fields from `sys` accessors; audit middleware via `UseCommandMiddleware`; commandlifecycle pair re-wired (adjacency contract F9) | compiles + Service API unchanged (D1 default: façade) |
| W2.2 | Session/WebAuthn/OAuth2 sub-services verified untouched on the façade | façade scenario test |
| W2.3 | Snapshot parity question (NV) resolved: does the façade need `SnapshotConfig` passthrough | verdict in ADR-0065 draft |

### W3 — read-tier cutover (rest: ~2 days)
Reads served by systemadapter queries; SQL read models kept but shadow-compared in CI; `Enforce` path decision (D4) applied; drain semantics documented divergence (D5).

### W4 — v5 cut execution (rest; rides the v5 window)
Delete `sql_readmodel{,_extra,_mysql}.go` (68 findings evaporate, §5), `es_materialize_adapter.go`, `stack_repositories.go` `buildStackRepositories`, the 3 SQL setup templates + `sql_setup_shared.go`, `Bundle` field, `NewProjectionLayer` (row 118) — exactly ADR-0051's list; `cqrs-htmx-upgrade` rule sweep; v5-removal-inventory reconciliation.

### W5 — upstream + durability (rest)
SecurityHooks seam ask (store wrapper + bus middleware on `system.DomainConfig`, F19); durability-pairing gate ask if W1.2 shows silence; ADR-0065 (this migration as accepted contract); `setup.NewFromSystem` (OQ 14) decided with D1.

## 8. Owner decisions

| # | Question | Default if unanswered |
| --- | --- | --- |
| D1 | Does `usermgmt.NewService` survive as a façade over `system.New` (consumers unchanged), or does composition move to consumers (setup/direct `system.New`)? | façade — public API stability wins; OQ 14 rides this |
| D2 | V007 cluster 1: evaporate (delete SQL view tier at v5, §5) or keep SQL read models? | evaporate (this plan's §5 recommendation) |
| D3 | SecurityHooks/ADR-0011 has NO system.New seam (F19): upstream ask + interim dual-path (`NewEventSourcedSetup` stays for encrypted deployments), or block migration on the seam? | upstream ask + dual-path; encrypted deployments stay legacy until seam |
| D4 | Authz hot path: declarative `Enforce` query (policy scan per check) vs live Casbin enforcer hydrated from a query at boot? | keep live enforcer in the façade; declarative query is the rebuild source |
| D5 | Startup read-your-writes: keep blocking-drain semantics in the façade or adopt `sys.Start` + poll (guide's `waitForView`)? | poll + readiness gate (matches `AsyncStartup` posture) |

## 9. Verification receipts (per wave) + risk register

- W0: `nix run .#check-docs-corpus-budget -- --ratchet` + `.#check-todo-hygiene`. W1-W2: `GOEXPERIMENT=jsonv2 go test ./systemadapter/... ./usermgmt/... -race`; `.#check-modules`. W3-W4: full battery (`.#build .#test .#lint`) + `cqrs-lint scorecard` finding-count delta (41→5→0 path) + `cqrs-htmx-upgrade` dry-run clean. W5: upstream PR links pinned.
- **R1 journal/CBOR:** mixed journals valid (ADR-0059); folds stamp-decode (F18); retired by W1.1. **R2 SSE envelope:** metadata-only golden (F20); retired by W1.3. **R3 SecurityHooks:** no upstream seam (F19); D3. **R4 checkpoint keys:** disjoint key sets ⇒ one-time full replay on first post-swap boot (F11) — same accepted cost class as the 2024 `usermgmt:start_projections`→per-projection key change (es_projection_setup.go:32-34); document, don't migrate. **R5 rollback:** stop `sys`, boot legacy `Service` on the same journal; read models rebuild by replay (invariant 4). **R6 parallel-run:** both hosts CAN share one journal (legacy `NewProjectionLayer` reads `sys.EventStore()`, projections.go:98) — but production dual-write is rejected: one writer (the system), legacy runs read-shadow only, per repo consistency doctrine.

## 10. Errata / honest notes

- The criterion verdict contradicts two standing claims — row 111's 2026-10-03/10-11 "in-memory checkpoint store" re-checks and v5-removal-inventory §5c's same note (both predate pin v4.17.0). W0.1 corrects them; until then this file is the newer evidence.
- "Hydrator met by construction" is a judgment, not upstream fact: no interface, no enforced gate (F17). W1.2 exists precisely to keep §6 honest; if the probe lies, the gate flips to NO-GO.
- Counts 21/27 are verified twice (constants + registrations, F2/F9/F12); systemadapter's own doc comments disagree with themselves (F7) — trusted code over comment.
- `setup.NewFromSystem` does not exist in-tree (grep 0 hits, verified) — OQ 14 remains genuinely open, folded into D1/W5.
