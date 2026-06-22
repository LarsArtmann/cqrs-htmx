# Status Report — 2026-06-22 (Post-Review Hardening)

> **Snapshot date:** 2026-06-22 14:05
> **Branch:** master @ `2b24f2c`
> **Test baseline:** 4/4 modules GREEN with `-race` | 585+ test specs | 0 lint violations | 0 errorfamily violations

---

## Executive Summary

**go-cqrs-lite v3.0.0 migration is complete and hardened.** Two rounds of brutal self-review identified and fixed 13 issues across the migration. The codebase is now in its cleanest state ever: god objects split, dead code removed, stale docs fixed, missing tests added, stdlib patterns adopted over reinvented code. The v3 migration touched 7 Go modules (291 Go files) with zero v2 references remaining.

**All work from this session is committed and pushed.** 20 commits, all self-contained, all with detailed messages.

---

## a) FULLY DONE ✅

### Session Work — v3 Migration + Two Self-Review Rounds (20 commits)

| Work Item                          | Status  | Details                                                                                                 |
| ---------------------------------- | ------- | ------------------------------------------------------------------------------------------------------- |
| Root module v3 migration           | ✅ Done | 57 files, pure `/v2`→`/v3` path bumps                                                                   |
| Catalog module v3 migration        | ✅ Done | `catalog/v2`→`catalog/v3`                                                                               |
| Usermgmt v3 migration              | ✅ Done | 72 files, 5 structural changes (projection rewrite, memory→watermill, Fold→Apply, io.Closer, bus.Close) |
| Integration test v3 migration      | ✅ Done | signing/encryption/memory bumps                                                                         |
| 3 Example modules v3 migration     | ✅ Done | basic, datastar-demo, catalog-demo                                                                      |
| Projection rewrite (ADR-0016)      | ✅ Done | Manual replay + SubscribeAll + dedup replaces projection.Runner                                         |
| Stale comment fixes                | ✅ Done | MemoryBus→watermill.EventBus, MemoryStore→storage/memory.MemoryStore                                    |
| Missing compile-time assertion     | ✅ Done | `var _ event.Projection` on MembershipReadModel                                                         |
| Type-safe dedup                    | ✅ Done | `map[id.EventID]struct{}` instead of `map[string]struct{}`                                              |
| Projection setup unit tests        | ✅ Done | 7 tests: shouldDispatch, buildLiveHandler dedup, error continuation, collectProjections                 |
| ADR-0016 created                   | ✅ Done | Documents projection rewrite decision + alternatives                                                    |
| `slices.Contains` over custom code | ✅ Done | Replaced reinvented `shouldDispatch` with stdlib                                                        |
| Dead code removed                  | ✅ Done | `ExternalAccount.Clone()` deleted                                                                       |
| TODO_LIST.md fixed                 | ✅ Done | OAuth2/OIDC, schema versioning, v3 migration marked DONE                                                |
| God object split                   | ✅ Done | `es_decide.go` 539→136 lines, split into 5 concern-based files                                          |
| Service-level tenant tests         | ✅ Done | 5 tests: Create, EmptyID, Suspend+Reactivate, Delete, AllTenants                                        |
| Service-level bot tests            | ✅ Done | 5 tests: Register, NoPepper, EmptyName, ResolveByToken, Delete                                          |
| AGENTS.md updated                  | ✅ Done | All v3 references reflected                                                                             |
| Migration plan document            | ✅ Done | Pareto-ordered, 69 sub-tasks, mermaid graph                                                             |

### Existing Feature Completeness

| Feature Area                        | Status  | Coverage                                       |
| ----------------------------------- | ------- | ---------------------------------------------- |
| Event-sourced CQRS (passwordless)   | ✅ Done | User, Membership, Tenant, Bot aggregates       |
| WebAuthn/Passkey auth               | ✅ Done | BeginRegistration/Login ceremonies             |
| TOTP MFA                            | ✅ Done | Enable/Verify/Disable via pquerna/otp          |
| OAuth2/OIDC integration             | ✅ Done | Google (OIDC), GitHub (OAuth2), PKCE           |
| Event signing & encryption          | ✅ Done | Opt-in seams (StoreWrapper, PublishMiddleware) |
| Identity model redesign             | ✅ Done | Actor, Tenant, Membership, Bot, Impersonation  |
| Event schema versioning + upcasters | ✅ Done | v1→v2 migration support                        |
| CSRF protection                     | ✅ Done | justinas/nosurf                                |
| SSE + WebSocket support             | ✅ Done | Broadcaster, dispatch bridge, OOB HTML         |
| Rate limiting                       | ✅ Done | Token bucket per-key                           |
| API token auth (bots)               | ✅ Done | HMAC-SHA256 token verification                 |
| Import/Export                       | ✅ Done | JSON + CSV                                     |
| Audit log                           | ✅ Done | Append-only event recorder                     |
| Email verification                  | ✅ Done | Token-based verification flow                  |
| Account lockout                     | ✅ Done | Configurable max attempts + duration           |
| Catalog doc generation              | ✅ Done | OpenAPI, AsyncAPI, D2, EventCatalog            |
| Casbin RBAC projection              | ✅ Done | Event-sourced policy derivation                |
| SQL event store                     | ✅ Done | Postgres + SQLite via upstream facade          |
| SQL session store                   | ✅ Done | Postgres + MySQL + SQLite                      |
| 16 ADRs documented                  | ✅ Done | 15 Accepted, 1 Superseded                      |

---

## b) PARTIALLY DONE 🔨

| Item                                  | Current State                                                      | Gap                                                                         |
| ------------------------------------- | ------------------------------------------------------------------ | --------------------------------------------------------------------------- |
| **FEATURES.md**                       | 79 FULLY_FUNCTIONAL features                                       | Missing Tenant, Bot, Impersonation, v3 migration entries                    |
| **ROADMAP.md**                        | v1.1.0/v2.2.0/v3.0.0 milestones                                    | Frozen at v2.4.0 — doesn't reflect v3 migration                             |
| **usermgmt coverage**                 | 88.7% (improved but not re-measured since adding 10 service tests) | Below root (96.4%) and catalog (95.3%)                                      |
| **revive:exported linter**            | Explicitly disabled in `.golangci.yml`                             | Unused/uncommented exports won't be caught by CI                            |
| **Service-level impersonation tests** | Decider-level exists                                               | No `svc.BeginImpersonation`/`EndImpersonation` through full dispatch        |
| **Service-level membership tests**    | Decider-level exists                                               | No `svc.AddMember`/`UpdateMemberRoles`/`RemoveMember` through full dispatch |

---

## c) NOT STARTED ❌

| Item                               | Priority | Notes                                                      |
| ---------------------------------- | -------- | ---------------------------------------------------------- |
| Update FEATURES.md                 | High     | Missing 4 major feature areas                              |
| Update ROADMAP.md                  | Medium   | Version-stale, doesn't reflect v3                          |
| OpenTelemetry integration          | Medium   | go-cqrs-lite v3 has otel module; cqrs-htmx doesn't wire it |
| Prometheus metrics                 | Low      | go-cqrs-lite v3 has prometheus module                      |
| godoc examples                     | Low      | ROADMAP v1.1.0 item, never completed                       |
| Integration test expansion         | Medium   | Only 8 test files in integration_test/                     |
| Streaming replay (SeekableJournal) | Future   | Current `ReadAll()` loads all events into memory           |
| `stack.Materialize` adoption       | Future   | ADR-0016 documents as future option                        |
| PostgreSQL session store preset    | Low      | SessionStore interface ready                               |
| Redis session store                | Low      | Multi-instance deployments                                 |
| Consumer migration guide (v2→v3)   | Medium   | Document what consumers need to change                     |

---

## d) TOTALLY FUCKED UP! 💥

| Issue                                          | Severity | Impact                                                                                                                                                                          |
| ---------------------------------------------- | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **FEATURES.md is stale**                       | High     | Missing Tenant management, Bot authentication, Impersonation, and v3 migration. Feature inventory is incomplete and misleading for anyone assessing the library's capabilities. |
| **ROADMAP.md frozen at v2.4.0**                | Medium   | Doesn't reflect v3 migration, identity redesign, OAuth2, or any work since 2026-06-17. Actively misleading about project direction.                                             |
| **`ClientIP()` deprecated but still exported** | Low      | Marked deprecated in httputil.go, delegates to httputil package. Confusing for consumers.                                                                                       |

**Note:** The TODO_LIST.md was previously in this category (actively lying) but was fixed this session. FEATURES.md and ROADMAP.md remain.

---

## e) WHAT WE SHOULD IMPROVE! 🎯

### Documentation (Highest ROI)

1. **Update FEATURES.md** — add Tenant, Bot, Impersonation, v3 migration entries (30min, huge customer value)
2. **Update ROADMAP.md** — reflect current state, retire completed milestones (20min)
3. **Write consumer migration guide** — what consumers need to change for v3 (45min)

### Testing

4. **Add service-level impersonation tests** — full dispatch path (30min)
5. **Add service-level membership tests** — full dispatch path (30min)
6. **Add projection replay integration test** — verify read-your-writes after pre-existing journal events (30min)
7. **Re-measure usermgmt coverage** — should be ~90%+ now with 10 new service tests
8. **Add fuzz tests** for projection dedup map (edge cases)

### Architecture

9. **Enable `revive:exported`** linter — catch unused exports automatically (30min + fixes)
10. **Remove or update deprecated `ClientIP()`** wrapper (15min)
11. **Consider `schema/v3` validator** for runtime event payload validation (60min)

### Type Safety

12. **Adopt `stack.Materialize[V,K]`** when persistent read models needed
13. **Branded types for event payloads** — currently `[]byte`, could be `codec.Payload`

---

## f) Top #25 Things We Should Get Done Next

Sorted by impact/effort ratio (highest first).

| #   | Task                                                                  | Impact  | Effort | Ratio |
| --- | --------------------------------------------------------------------- | ------- | ------ | ----- |
| 1   | **Update FEATURES.md** — add Tenant, Bot, Impersonation, v3 migration | High    | 30min  | ★★★★★ |
| 2   | **Update ROADMAP.md** — reflect v3 migration + current state          | High    | 20min  | ★★★★★ |
| 3   | **Remove/update deprecated `ClientIP()`**                             | Low     | 15min  | ★★★★☆ |
| 4   | **Add service-level impersonation tests**                             | High    | 30min  | ★★★★☆ |
| 5   | **Add service-level membership tests**                                | High    | 30min  | ★★★★☆ |
| 6   | **Add projection replay integration test**                            | High    | 30min  | ★★★★☆ |
| 7   | **Re-measure coverage** after new tests                               | Medium  | 10min  | ★★★★☆ |
| 8   | **Enable `revive:exported` linter** + fix violations                  | Medium  | 30min  | ★★★☆☆ |
| 9   | **Write consumer migration guide** (v2→v3)                            | Medium  | 45min  | ★★★☆☆ |
| 10  | **Add godoc examples** for App, Handler, Service                      | Medium  | 60min  | ★★★☆☆ |
| 11  | **Wire OpenTelemetry** — go-cqrs-lite v3 otel module                  | Medium  | 90min  | ★★★☆☆ |
| 12  | **Expand integration tests** — more cross-module scenarios            | Medium  | 90min  | ★★★☆☆ |
| 13  | **Add `schema/v3` validator** for event payloads                      | Medium  | 60min  | ★★★☆☆ |
| 14  | **Add fuzz tests** for projection dedup                               | Medium  | 30min  | ★★★☆☆ |
| 15  | **Add benchmark** for projection replay with large stores             | Low     | 30min  | ★★☆☆☆ |
| 16  | **Streaming replay** via `SeekableJournal.ReadFrom`                   | Medium  | 60min  | ★★☆☆☆ |
| 17  | **PostgreSQL session store preset**                                   | Low     | 45min  | ★★☆☆☆ |
| 18  | **Prometheus metrics** endpoint                                       | Low     | 45min  | ★★☆☆☆ |
| 19  | **Redis session store**                                               | Low     | 60min  | ★★☆☆☆ |
| 20  | **Profile and optimize** hot paths                                    | Low     | 90min  | ★☆☆☆☆ |
| 21  | **`stack.Materialize`** for persistent read models                    | Low     | 120min | ★☆☆☆☆ |
| 22  | **CI badge** in README.md                                             | Low     | 10min  | ★★★☆☆ |
| 23  | **Consumer versioning decision** (bump cqrs-htmx to /v3?)             | Product | 0min   | ★☆☆☆☆ |
| 24  | **Add VERSIONING.md** documenting semver policy                       | Low     | 30min  | ★★☆☆☆ |
| 25  | **Consider automated release notes** from commit history              | Low     | 30min  | ★☆☆☆☆ |

---

## g) Top #1 Question I Cannot Figure Out Myself 🤔

**Should cqrs-htmx itself bump to `/v3` module path?**

The library is now on go-cqrs-lite v3, but its own module path is still `github.com/larsartmann/cqrs-htmx/v2`. The internal bus type changed (`MemoryBus` → `watermill.EventBus`) and the projection infrastructure was completely rewritten, but `StartProjections()` signature is identical — consumers don't change their code.

Arguments for `/v3`:

- Signals the breaking infrastructure change (bus type, projection rewrite)
- Consistency with upstream

Arguments against:

- The public API barely changed — consumer code doesn't need updates
- Module path bumps force ALL consumers to change import paths
- This is a product/versioning decision that depends on how many consumers exist

**I cannot answer this without knowing your versioning policy and consumer landscape.**

---

## CI Health

| Check                                            | Status                                               |
| ------------------------------------------------ | ---------------------------------------------------- |
| `nix run .#test` (4 modules, -race)              | ✅ GREEN                                             |
| `nix run .#build` (all modules)                  | ✅ GREEN                                             |
| `nix run .#lint` (root + catalog + usermgmt)     | ✅ 0 issues                                          |
| `branching-flow errorfamily .` (root + usermgmt) | ✅ 0 violations                                      |
| `nix fmt`                                        | ✅ Clean                                             |
| Split-brain check (v2/v3 imports)                | ✅ 0 v2 refs remain                                  |
| go.work integrity                                | ✅ All 7 modules                                     |
| Dead code check                                  | ✅ ExternalAccount.Clone removed                     |
| God object check                                 | ✅ All files < 200 lines (largest: es_decide.go 136) |

---

## Metrics

| Metric                         | Value                                     |
| ------------------------------ | ----------------------------------------- |
| Go source files                | 291                                       |
| Test specs (root)              | 59                                        |
| Test specs (usermgmt)          | 526                                       |
| Total test specs               | 585+                                      |
| Coverage (root)                | 96.4%                                     |
| Coverage (usermgmt)            | ~90%+ (was 88.7%, added 10 service tests) |
| Coverage (catalog)             | 95.3%                                     |
| Lint violations                | 0                                         |
| Errorfamily violations         | 0                                         |
| ADRs                           | 16 (15 Accepted, 1 Superseded)            |
| Commits this session           | 20                                        |
| v2 import references remaining | 0                                         |

---

## Session Commit History (20 commits)

| Commit    | Description                                                                                 |
| --------- | ------------------------------------------------------------------------------------------- |
| `2b24f2c` | style: fix gci formatting after slices import addition                                      |
| `20e0acf` | test(usermgmt): add service-level bot tests through full dispatch                           |
| `60aa858` | test(usermgmt): add service-level tenant tests through full dispatch                        |
| `5b2c6e5` | refactor(usermgmt): split es_decide.go god object (539 lines) into 5 files                  |
| `5fe417d` | docs(todo): fix lying TODO_LIST — mark OAuth2/OIDC, schema versioning, v3 migration as DONE |
| `3b2c95d` | refactor(usermgmt): remove dead ExternalAccount.Clone() method                              |
| `cf10bb6` | refactor(usermgmt): replace custom shouldDispatch with stdlib slices.Contains               |
| `33d4246` | docs(status): comprehensive status report — v3 migration complete + self-review             |
| `1aff99d` | style: fix gci formatting in projection setup test                                          |
| `deb2d3a` | docs(adr): ADR-0016 — go-cqrs-lite v3.0.0 migration / projection rewrite                    |
| `4925f34` | test(usermgmt): add unit tests for projection setup machinery                               |
| `8ea0ac7` | refactor(usermgmt): use id.EventID as dedup map key instead of string                       |
| `3cd0999` | fix(usermgmt): add missing compile-time event.Projection assertion                          |
| `d45abf6` | fix(usermgmt): update stale MemoryBus/MemoryStore comments to v3 names                      |
| `7284a89` | fix(integration_test): fix import ordering after go mod tidy                                |
| `bf97290` | docs: update AGENTS.md for go-cqrs-lite v3.0.0 migration                                    |
| `92b8ae7` | fix(usermgmt): auto-fix lint formatting after v3 migration                                  |
| `dd4302f` | feat(integration_test): migrate to go-cqrs-lite v3.0.0                                      |
| `b5c2773` | feat(usermgmt): migrate to go-cqrs-lite v3.0.0 — projection rewrite                         |
| `c460371` | feat: migrate root, catalog, and example modules to go-cqrs-lite v3.0.0                     |

---

_Waiting for further instructions._
