# Status Report — 2026-06-22

> **Snapshot date:** 2026-06-22 13:36
> **Branch:** master @ `1aff99d`
> **Test baseline:** 4/4 modules GREEN with `-race` | 0 lint violations | 0 errorfamily violations

---

## Executive Summary

**go-cqrs-lite v3.0.0 migration is COMPLETE.** All 7 Go modules successfully migrated from v2.6.0 to v3.0.0, including the major projection rewrite (`projection.Runner` → manual journal replay + `bus.SubscribeAll`). A brutal self-review followed, finding and fixing 6 concrete issues. The codebase is in its healthiest state ever: 580+ tests passing, zero lint violations, zero stdlib error constructors, clean module boundaries, no split brains.

**The biggest remaining risk is documentation staleness** — TODO_LIST.md, FEATURES.md, and ROADMAP.md all predate the v3 migration and the identity model redesign, making them actively misleading.

---

## a) FULLY DONE ✅

### go-cqrs-lite v3.0.0 Migration (Completed 2026-06-22)

| Work Item | Status | Details |
|-----------|--------|---------|
| Root module (57 files) | ✅ Done | Pure `/v2`→`/v3` path bumps |
| Catalog module (5 files) | ✅ Done | `catalog/v2`→`catalog/v3` |
| Usermgmt module (72 files) | ✅ Done | 5 structural changes + path bumps |
| Integration test (5 files) | ✅ Done | signing/encryption + memory bumps |
| 3 Example modules | ✅ Done | basic, datastar-demo, catalog-demo |
| Projection rewrite | ✅ Done | Manual replay + SubscribeAll + dedup (ADR-0016) |
| `memory.MemoryBus` → `watermill.EventBus` | ✅ Done | Synchronous delivery preserved |
| `Decider.Fold` → `Apply` | ✅ Done | 4 decider sites renamed |
| `bus.Close()` → `closeBus()` type-assert | ✅ Done | io.Closer removed from interfaces |
| Self-review fixes (6 issues) | ✅ Done | Stale comments, missing assertion, type-safe dedup, tests, ADR, docs |
| AGENTS.md updated | ✅ Done | All v3 references reflected |

### Existing Feature Completeness (Prior to This Session)

| Feature | Status | Coverage |
|---------|--------|----------|
| Event-sourced CQRS (passwordless) | ✅ Done | User, Membership, Tenant, Bot aggregates |
| WebAuthn/Passkey auth | ✅ Done | BeginRegistration/Login ceremonies |
| TOTP MFA | ✅ Done | Enable/Verify/Disable via pquerna/otp |
| OAuth2/OIDC integration | ✅ Done | Google (OIDC), GitHub (OAuth2), PKCE |
| Event signing & encryption | ✅ Done | Opt-in seams (StoreWrapper, PublishMiddleware) |
| Identity model redesign | ✅ Done | Actor, Tenant, Membership, Bot, Impersonation |
| Event schema versioning + upcasters | ✅ Done | v1→v2 migration support |
| CSRF protection | ✅ Done | justinas/nosurf |
| SSE + WebSocket support | ✅ Done | Broadcaster, dispatch bridge, OOB HTML |
| Rate limiting | ✅ Done | Token bucket per-key |
| API token auth (bots) | ✅ Done | HMAC-SHA256 token verification |
| Import/Export | ✅ Done | JSON + CSV |
| Audit log | ✅ Done | Append-only event recorder |
| Email verification | ✅ Done | Token-based verification flow |
| Account lockout | ✅ Done | Configurable max attempts + duration |
| Catalog doc generation | ✅ Done | OpenAPI, AsyncAPI, D2, EventCatalog |
| Casbin RBAC projection | ✅ Done | Event-sourced policy derivation |
| SQL event store | ✅ Done | Postgres + SQLite via upstream facade |
| SQL session store | ✅ Done | Postgres + MySQL + SQLite |
| 16 ADRs documented | ✅ Done | 15 Accepted, 1 Superseded |

**Test metrics:** 162 test files, 580+ test specs, 96.4% root / 88.7% usermgmt / 95.3% catalog coverage.

---

## b) PARTIALLY DONE 🔨

| Item | Current State | Gap |
|------|--------------|-----|
| **TODO_LIST.md** | Exists, 170+ items completed historically | **Stale** — predates v3 migration + identity redesign. OAuth2/OIDC and event schema versioning marked OPEN but are actually DONE |
| **FEATURES.md** | 79 FULLY_FUNCTIONAL features catalogued | **Stale** — missing Tenant, Bot, Impersonation, v3 migration entries |
| **ROADMAP.md** | v1.1.0/v2.2.0/v3.0.0 milestones | **Stale** — still says v2.4.0, doesn't reflect v3 migration completion |
| **Service-level tenant tests** | Decider-level tests exist (`es_tenant_test.go`) | No `svc.CreateTenant()`, `svc.SuspendTenant()` etc. tested through full dispatch path |
| **Service-level bot tests** | Decider-level tests exist (`es_bot_test.go`) | No `svc.RegisterBot()`, `svc.ResolveBotByToken()` tested through full dispatch path |
| **usermgmt coverage** | 88.7% | Below root (96.4%) and catalog (95.3%) — mainly from untested service-level tenant/bot paths |
| **revive:exported linter** | Explicitly disabled in `.golangci.yml` | Unused/uncommented exports won't be caught by CI |

---

## c) NOT STARTED ❌

| Item | Priority | Notes |
|------|----------|-------|
| PostgreSQL session store | Medium | `SQLSessionStore` has Postgres support, but no dedicated preset |
| Redis session store | Low | Multi-instance deployments; `SessionStore` interface ready |
| OpenTelemetry integration | Medium | go-cqrs-lite v3 has otel module; cqrs-htmx doesn't wire it |
| Prometheus metrics | Low | go-cqrs-lite v3 has prometheus module |
| godoc examples | Low | ROADMAP v1.1.0 item, never completed |
| Integration test expansion | Medium | Only 8 test files in integration_test/ |
| `stack.Materialize` adoption | Future | ADR-0016 documents as future option when persistent read models needed |
| CatchUpSubscriber for streaming replay | Future | Current `ReadAll()` loads all events into memory |

---

## d) TOTALLY FUCKED UP! 💥

| Issue | Severity | Impact |
|------|----------|--------|
| **TODO_LIST.md is actively lying** | High | OAuth2/OIDC marked `OPEN` but ADR-0014 is Accepted and code shipped. Event schema versioning marked `OPEN` but ADR-0013 is Accepted and implemented. Anyone reading this file gets wrong information. |
| **FEATURES.md missing 4 major features** | Medium | No entries for Tenant management, Bot authentication, Impersonation, or the v3 migration. Feature inventory is incomplete. |
| **ROADMAP.md version frozen at v2.4.0** | Medium | Doesn't reflect v3 migration, identity redesign, OAuth2, or any work since 2026-06-17 |
| **`es_decide.go` is 539 lines** (god object) | Medium | All 10+ user aggregate decide functions in one file. Should be split by concern (auth, profile, credentials, TOTP, external accounts) |
| **`ExternalAccount.Clone()` is dead code** | Low | Exported, never called. The type is all-value-types so Clone is a no-op |
| **`ClientIP()` is deprecated but still tested** | Low | Marked deprecated, delegates to httputil. Should be removed or callers updated |

---

## e) WHAT WE SHOULD IMPROVE! 🎯

### Architecture

1. **Split `es_decide.go`** (539 lines) into `es_decide_auth.go`, `es_decide_profile.go`, `es_decide_credentials.go`, `es_decide_totp.go`, `es_decide_external.go`
2. **Remove dead code**: `ExternalAccount.Clone()`, deprecated `ClientIP()` wrapper
3. **Enable `revive:exported`** linter to catch unused exports automatically
4. **Consider `stack.Materialize`** for persistent read models when needed (ADR-0016 documents the path)

### Testing

5. **Add service-level tenant tests** — `CreateTenant`, `SuspendTenant`, `Reactivate`, `DeleteTenant` through full dispatch
6. **Add service-level bot tests** — `RegisterBot`, `ResolveBotByToken`, `DeleteBot` through full dispatch
7. **Add projection replay integration test** — verify read-your-writes after journal replay with pre-existing events
8. **Target 95%+ usermgmt coverage** (currently 88.7%)

### Documentation

9. **Update TODO_LIST.md** — mark OAuth2/OIDC and event schema versioning as DONE, add v3 migration as DONE
10. **Update FEATURES.md** — add Tenant, Bot, Impersonation, v3 migration entries
11. **Update ROADMAP.md** — reflect current state, new version target

### Type Safety

12. **Use `stack.Materialize[V,K]`** for typed read models when persistent KV store is needed
13. **Consider branded types for event payloads** — currently `[]byte`, could use `codec.Payload` branded type
14. **Add `go-cqrs-lite/schema/v3` validator** for runtime payload validation

---

## f) Top #25 Things We Should Get Done Next

Sorted by impact/effort ratio (highest first).

| # | Task | Impact | Effort | Ratio |
|---|------|--------|--------|-------|
| 1 | **Fix TODO_LIST.md** — mark OAuth2/OIDC + schema versioning as DONE | High | 10min | ★★★★★ |
| 2 | **Update FEATURES.md** — add Tenant, Bot, Impersonation, v3 migration | High | 30min | ★★★★★ |
| 3 | **Update ROADMAP.md** — reflect v3 migration + current state | High | 20min | ★★★★★ |
| 4 | **Remove `ExternalAccount.Clone()`** dead code | Medium | 5min | ★★★★★ |
| 5 | **Add service-level tenant tests** (CreateTenant/Suspend/Reactivate/Delete) | High | 60min | ★★★★☆ |
| 6 | **Add service-level bot tests** (RegisterBot/ResolveByToken/Delete) | High | 45min | ★★★★☆ |
| 7 | **Split `es_decide.go`** into domain-concern files | Medium | 45min | ★★★★☆ |
| 8 | **Add projection replay integration test** (read-your-writes after replay) | High | 30min | ★★★★☆ |
| 9 | **Remove deprecated `ClientIP()` wrapper** or update callers | Low | 15min | ★★★☆☆ |
| 10 | **Enable `revive:exported` linter** and fix violations | Medium | 30min | ★★★☆☆ |
| 11 | **Add godoc examples** for App, Handler, Service entry points | Medium | 60min | ★★★☆☆ |
| 12 | **Wire OpenTelemetry** — go-cqrs-lite v3 has otel module | Medium | 90min | ★★★☆☆ |
| 13 | **Expand integration_test** — more cross-module scenarios | Medium | 90min | ★★★☆☆ |
| 14 | **Add `schema/v3` validator** for event payload validation | Medium | 60min | ★★★☆☆ |
| 15 | **Add PostgreSQL session store preset** | Medium | 45min | ★★☆☆☆ |
| 16 | **Streaming replay** via `SeekableJournal.ReadFrom` instead of `ReadAll` | Medium | 60min | ★★☆☆☆ |
| 17 | **Add Prometheus metrics** endpoint | Low | 45min | ★★☆☆☆ |
| 18 | **Add Redis session store** for multi-instance | Low | 60min | ★★☆☆☆ |
| 19 | **Consider `stack.Materialize`** for persistent read models | Low | 120min | ★☆☆☆☆ |
| 20 | **Profile and optimize** hot paths (dispatch, decode, render) | Low | 90min | ★☆☆☆☆ |
| 21 | **Add fuzz tests** for projection replay + dedup | Medium | 30min | ★★★☆☆ |
| 22 | **Add benchmark** for projection replay with large event stores | Low | 30min | ★★☆☆☆ |
| 23 | **Document migration guide** for consumers upgrading from v2 to v3 | Medium | 45min | ★★★☆☆ |
| 24 | **Add CI badge** to README.md | Low | 10min | ★★★☆☆ |
| 25 | **Consider versioned module paths** (`/v3` for cqrs-htmx itself) | Low | 60min | ★☆☆☆☆ |

---

## g) Top #1 Question I Cannot Figure Out Myself 🤔

**Should cqrs-htmx itself bump to `/v3` module path?**

The library is now on go-cqrs-lite v3, but its own module path is still `github.com/larsartmann/cqrs-htmx/v2`. This creates a mismatch: consumers importing `cqrs-htmx/v2` get a library that transitively depends on `go-cqrs-lite/*/v3`.

Arguments for bumping to `/v3`:
- Signals breaking changes (v3 migration changed `StartProjections` internals, bus type)
- Consistency with upstream

Arguments against:
- The public API of cqrs-htmx itself barely changed — `StartProjections` signature is identical, consumer code doesn't change
- Module path bumps force all consumers to update import paths
- The internal migration (projection rewrite, bus swap) is an implementation detail

**This is a product/versioning decision, not a technical one. I cannot determine the right answer without knowing how many consumers exist and what the versioning policy is.**

---

## CI Health

| Check | Status |
|-------|--------|
| `nix run .#test` (all 4 modules, -race) | ✅ GREEN |
| `nix run .#build` (all modules) | ✅ GREEN |
| `nix run .#lint` (root + catalog + usermgmt) | ✅ 0 issues |
| `branching-flow errorfamily .` (root) | ✅ 0 violations |
| `branching-flow errorfamily .` (usermgmt) | ✅ 0 violations |
| `nix fmt` | ✅ Clean |
| Split-brain check (v2/v3 imports) | ✅ 0 v2 refs remain |
| go.work integrity | ✅ All 7 modules |

---

## Commit History (This Session)

| Commit | Description |
|--------|-------------|
| `1aff99d` | style: fix gci formatting in projection setup test |
| `deb2d3a` | docs(adr): ADR-0016 — go-cqrs-lite v3.0.0 migration / projection rewrite |
| `4925f34` | test(usermgmt): add unit tests for projection setup machinery |
| `8ea0ac7` | refactor(usermgmt): use id.EventID as dedup map key instead of string |
| `3cd0999` | fix(usermgmt): add missing compile-time event.Projection assertion |
| `d45abf6` | fix(usermgmt): update stale MemoryBus/MemoryStore comments to v3 names |
| `7284a89` | fix(integration_test): fix import ordering after go mod tidy |
| `bf97290` | docs: update AGENTS.md for go-cqrs-lite v3.0.0 migration |
| `92b8ae7` | fix(usermgmt): auto-fix lint formatting after v3 migration |
| `dd4302f` | feat(integration_test): migrate to go-cqrs-lite v3.0.0 |
| `b5c2773` | feat(usermgmt): migrate to go-cqrs-lite v3.0.0 — projection rewrite |
| `c460371` | feat: migrate root, catalog, and example modules to go-cqrs-lite v3.0.0 |
| `6bc036d` | docs(planning): comprehensive go-cqrs-lite v3.0.0 migration plan |

**13 commits, all pushed to origin/master.**

---

_Waiting for further instructions._
