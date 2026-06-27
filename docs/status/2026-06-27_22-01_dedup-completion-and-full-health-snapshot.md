# Comprehensive Status Report — cqrs-htmx

> **Generated:** 2026-06-27 22:01 CEST (2026-06-27T22:01:09+0200)
> **HEAD:** `8091422 docs(adminui): round-5 brutal self-review + icon system + TOTP/WebAuthn DRY`
> **Branch:** `master` (up to date with `origin/master`)
> **Version:** v3.1.0 (go-cqrs-lite v3.1.0)
> **Method:** Every green claim below was **re-measured this session** by running the commands, not read from prior reports.

---

## Health Snapshot (measured this session)

| Check               | Root  | usermgmt | adminui | catalog | integration | Examples (4) |
| ------------------- | :---: | :------: | :-----: | :-----: | :---------: | :----------: |
| **Build**           |  ✅   |    ✅    |   ✅    |   ✅    |     ✅      |      ✅      |
| **Tests (`-race`)** |  ✅   |    ✅    |   ✅    |   ✅    |     ✅      |      —       |
| **Lint (0 issues)** |  ✅   |    ✅    |   ✅    |   ✅    |      —      |      —       |
| **ErrorFamily (0)** |  ✅   |    ✅    |   ✅    |    —    |      —      |      —       |
| **Coverage**        | 95.6% |  79.5%   |  56.4%  |  95.3%  |      —      |      —       |
| **Test count**      |  95   |   699    |   19    |   41    |     15      |      —       |
| **Production LOC**  | ~5.5K |  ~10.8K  |  ~3.6K  |  ~0.5K  |      —      |      —       |

**Total: 869 tests passing, race-safe, 0 lint issues, 0 errorfamily violations across all 5 Go modules.**

---

## a) FULLY DONE

### Session work (this commit window)

- ✅ **Deduplication to zero harmful clones** — `art-dupl --semantic -t 3` reduced from **5 → 3 clone groups**. Two genuine DRY violations extracted:
  - `requireUserIDWithWebAuthnRateLimit` helper collapses the duplicated rate-limit + userID guard in both WebAuthn finish ceremonies (`webauthn_http.go`).
  - `handleTOTPDisable` now delegates to the existing `handleTOTPCode` — it was the lone holdout that copy-pasted the auth/decode/verify body instead of using the shared dispatcher (`verification_totp_http.go`).
  - The 3 remaining clones are **irreducible idioms** (mutex+defer scope constraint, independent projection locks, constructor calls in different contexts).
- ✅ **Realtime BDD coverage** (commit `81f46e5`) — behavioral SSE + WebSocket tests.
- ✅ **adminui tenant-info restore** (commit `ff2217c`) — UX regression fix on members page.

### Library core (root module) — FULLY FUNCTIONAL

- App builder, command/query dispatch, handler options (decode/render/auth/validate/notify/HTMX).
- Embedded HTMX v2.0.9 JS (`HTMXScriptHandler`, ETag/caching, pluggable).
- CSRF (justinas/nosurf), rate limiting (token-bucket + heap eviction), security headers, panic recovery, request logging (text/JSON/slog).
- **Realtime parity:** SSE (Stream/Broadcaster/Reconnection/EventStore/CQRS bridges) + WebSocket (Message parser/encoder/OOB HTML/Broadcaster/CQRS bridges), sharing a generic `fanOut[T]` hub.
- Branded identity in context (UserID/CorrelationID/RequestID/ActorID/ImpersonatorID).
- go-error-family error classification → HTTP mapping. RFC 7807 StructuredError.

### usermgmt submodule — FULLY FUNCTIONAL (event-sourced)

- Passwordless WebAuthn (go-webauthn v0.17.4) as sole auth path. No password code.
- 12 events / 11 commands, pure decide + fold. Read-your-writes via synchronous watermill EventBus.
- Identity model (ADR-0015): Tenant (4 events), Bot (2), Membership (3), Impersonation, ActorID kind-discriminated union.
- OAuth2/OIDC (ADR-0014): dual-mode providers, PKCE S256, subject-first matching, global uniqueness.
- TOTP MFA (pquerna/otp), email verification, user import/export (JSON/CSV), audit log.
- SQL stores: Event (Postgres+SQLite, upstream-delegated), Session (Postgres/SQLite/MySQL), 4 SQL read models. One-call stack presets (SQLite/Postgres). `OptimizeSQLiteDB`.
- Event signing + encryption opt-in seams (ADR-0011). Schema versioning + upcasters (ADR-0013).
- Graceful shutdown (`Close`/`GracefulClose`/`Stop`).

### adminui submodule — SHIPPED (buildable for consumers)

- Ready-made Admin Dashboard (templ + HTMX), one-call mount, two scopes (SuperAdmin/TenantAdmin).
- Inline SVG icon system (18 glyphs, **no external icon-package dependency**).
- Auth-agnostic, role-based authorizer, embedded CSS/JS (light/dark), CSRF-ready forms.
- End-to-end smoke test + CSRF proof + tenant lifecycle flows.

### catalog submodule + integration_test

- catalog: OpenAPI/AsyncAPI/D2/EventCatalog generation, 95.3% coverage.
- integration_test: 7 cross-module scenarios (UserID bridge, authz, catalog, CSRF+WebAuthn, crypto E2E, typed dispatch).

---

## b) PARTIALLY DONE

- 🟡 **adminui coverage 56.4%** — the **lowest** module, marketed as "drop-in." Has smoke + CSRF + lifecycle tests, but handler negative paths, authz denial (403), tenant-scope isolation, and the members/tenants write paths are thin (`membersIndex`/`membersAdd`/`membersRemove`/`membersUpdateRole`/`tenantNew` all 0%; `defaultAuthorizer`/`RequireAnyRole` 0%). **Known debt, not a lie — should gate any "stable" tag.**
- 🟡 **Type-safety hardening (9 open items)** — `ActorID`/`ImpersonatorID` stored as raw `string` in context; `foldUser` silently no-ops on unknown events (unlike other folds); `BotState.OwnerID` raw string; `Email` branded type defined but unused in structs; duplicate `ErrUnauthorized` sentinel across modules breaks `errors.Is` at the boundary; `TenantState` allows impossible `Suspended+Deleted`; `actorKindFromString` silently defaults. See TODO_LIST.md "Type Safety Improvements."
- 🟡 **WebAuthn `*http.Request` leaks into service layer** (`webauthn_service.go:52,154`) — HTTP concern in domain service. Should parse bytes in HTTP layer.
- 🟡 **Docs gaps** — v2→v3 consumer migration guide, godoc examples for entry points, VERSIONING.md — all Open in ROADMAP.
- 🟡 **6 in-memory stores have no interface/SQL alt** (WebAuthn sessions, verification tokens, TOTP, lockout, rate limiter, read models) — multi-instance impossible without consumer reimplementation.
- 🟡 **`BrandNamer` for root marker types** — PARTIALLY UNBLOCKED (go-cqrs-lite v3.1.0 exports them) but not yet wired.

---

## c) NOT STARTED

- ⚪ **Observability (v3.2.0):** OpenTelemetry wiring via go-cqrs-lite otel module; Prometheus metrics middleware; (coverage CI gate is partially in via flake app `#coverage-gate`).
- ⚪ **Persistence & Scale (v3.3.0):** Redis session/OAuth2-state stores; BadgerDB; streaming replay via `SeekableJournal.ReadFrom`; projection-replay benchmarks at 10K+ events.
- ⚪ **Advanced ES (v4.0.0):** `stack.Materialize` for declarative read models; `CatchUpSubscriber` alternative to manual replay; schema/v3 payload validators; real-Postgres integration tests; migration tooling.
- ⚪ **Snapshot integration** — `go-cqrs-lite/snapshot/v3` is an indirect dep but not wired; startup replays all events.
- ⚪ **revive:exported linter** — not enabled.

---

## d) TOTALLY FUCKED UP

### 🔴 `adminui/go.mod` + `go.sum` carry a ghost dependency (UNCOMMITTED — see "Top #1 question")

The working tree `adminui/go.mod` and `go.sum` are the **only uncommitted changes** and they **re-introduce the exact ghost dependency** that brutal self-review round 5 (committed in `8091422`) claims to have killed:

```diff
+ github.com/larsartmann/templ-components v0.3.0   # require block
+ github.com/larsartmann/templ-components => ../../templ-components  # replace
```

**Why this is fucked:** The committed `icons.go` (`8091422`) no longer imports `templ-components` (it inlines 18 SVG paths locally). I verified by stashing these go.mod/go.sum changes — **`GOWORK=off go build ./...` succeeds with exit 0 without them.** So `templ-components` is an **unused dependency** in the working tree, plus a `replace` pointing at a sibling repo (`../../templ-components`) that **does not exist for any consumer** and whose published v0.3.0 lacks the symbols the old code used.

This is the **third "committed/near-committed broken" failure** flagged in two days (round 4 = un-generated `_templ.go`; round 5 = ghost import; this = ghost go.mod). If these go.mod/go.sum changes were committed, they'd ship a dependency that builds locally (sibling repo present) but is dead weight / non-resolvable depending on consumer environment. **HEAD is clean and correct; the working-tree drift is the problem.**

### 🔴 Historical (now fixed, documented for honesty)

- adminui **shipped unbuildable for consumers** from its first commit until `8091422` — `icons.go` imported symbols from an **unreleased** sibling repo. A local `replace` masked it. The 2026-06-27 status report claimed the go.sum drift was "2 cosmetic indirect lines being committed" — that was a **lie** (it was a broken cross-repo coupling, never committed). Fixed by inlining icons. The icon-coverage test (`TestIcons_AllReferencedIconsExist`) correctly guarded this change.

---

## e) WHAT WE SHOULD IMPROVE

1. **Add a consumer-build CI gate:** `GOWORK=off go build ./...` (and `go mod tidy -diff`) per module in CI. A sibling-repo `replace` must never ship again. This single check would have caught rounds 3, 4, and 5's failures automatically.
2. **Raise adminui coverage to ≥75%** — it's the biggest remaining risk surface and the newest code. Focus: members/tenants write handlers, 403 denial paths, tenant-scope isolation.
3. **Run `go mod tidy` in adminui** to drop the unused `templ-components` require + replace (the immediate fucked-up item), then verify `go.work.sum` is consistent.
4. **Type-safety sweep** — convert the 9 raw-`string` identity fields to branded types; make `foldUser` return an error on unknown events (consistency with the other 3 folds).
5. **Stop trusting workspace builds as proof of consumer-buildability.** Process smell documented in round 5 — recurring.
6. **Extract `*http.Request` from WebAuthn service** — keep the service layer transport-agnostic.
7. **Wire OpenTelemetry** — the hooks exist (`BeforeDispatchHook`/`AfterDispatchHook`); go-cqrs-lite v3 has an otel module. Low-friction, high-value.
8. **Define ephemeral-store interfaces** (WebAuthn sessions, lockout, TOTP, OAuth2 state) so consumers can provide Redis/SQL alts for multi-instance.
9. **Consolidate duplicate sentinels** — `cqrshtmx.ErrUnauthorized` vs `usermgmt.ErrUnauthorized` break `errors.Is` across the module boundary.

---

## f) Top #25 things to get done next

| #   | Task                                                                                                    | Impact                         | Effort |
| --- | ------------------------------------------------------------------------------------------------------- | ------------------------------ | ------ |
| 1   | **Drop unused `templ-components` from `adminui/go.mod`/`go.sum`** (`go mod tidy`); verify `go.work.sum` | Critical (kills ghost dep)     | XS     |
| 2   | **Add consumer-build CI gate** (`GOWORK=off go build ./...` + `go mod tidy -diff` per module)           | Critical (prevents recurrence) | S      |
| 3   | Raise **adminui coverage ≥75%** (members/tenants write paths, 403 denial, tenant-scope isolation)       | High                           | M      |
| 4   | Type `ActorID`/`ImpersonatorID` in `context.go` instead of raw `string`                                 | High                           | S      |
| 5   | Make `foldUser` return error on unknown events (match foldMembership/Tenant/Bot)                        | High                           | XS     |
| 6   | Extract `*http.Request` from `webauthn_service.go` (parse in HTTP layer)                                | High                           | M      |
| 7   | Use `UserID` branded type for `BotState.OwnerID` (+ events/readmodel/service)                           | High                           | S      |
| 8   | Use `TenantID` in Authz domain parameters (authz_roles.go, authz_policies.go)                           | High                           | S      |
| 9   | Consolidate duplicate `ErrUnauthorized` sentinels across root/usermgmt                                  | Medium                         | S      |
| 10  | Make `TenantState` impossible-states unrepresentable (prevent `Suspended+Deleted`)                      | Medium                         | XS     |
| 11  | Wire OpenTelemetry via go-cqrs-lite v3 otel module                                                      | Medium                         | M      |
| 12  | Define interfaces for the 6 ephemeral in-memory stores (multi-instance)                                 | Medium                         | M      |
| 13  | Use `Email` branded type in domain model structs (not just ExportUser)                                  | Medium                         | S      |
| 14  | Wire snapshot integration (`go-cqrs-lite/snapshot/v3`) to cut startup replay                            | Medium                         | M      |
| 15  | Add v2→v3 consumer migration guide                                                                      | Medium                         | S      |
| 16  | `LastEventIDFromRequest` should delegate to `SSEStream.LastEventID()` (dedup)                           | Low                            | XS     |
| 17  | Add godoc examples for `App`, `Handler`, `Service` entry points                                         | Medium                         | S      |
| 18  | Add `VERSIONING.md` documenting semver policy                                                           | Low                            | XS     |
| 19  | Remove deprecated `ClientIP()` wrapper (delegates to httputil)                                          | Low                            | XS     |
| 20  | Service-level impersonation tests through full dispatch                                                 | High                           | M      |
| 21  | Service-level membership tests through full dispatch                                                    | High                           | M      |
| 22  | Projection-replay integration test (journal vs live dedup)                                              | Medium                         | M      |
| 23  | Property-based tests for foldTenant/foldBot/foldMembership                                              | Medium                         | M      |
| 24  | Redis session store + OAuth2 state store (multi-instance readiness)                                     | Medium                         | L      |
| 25  | Enable `revive:exported` linter + fix violations                                                        | Low                            | S      |

---

## g) Top #1 question I cannot figure out myself

> **Should I discard the uncommitted `adminui/go.mod` / `adminui/go.sum` changes (which re-add the `templ-components` ghost dependency), or are they intentional and I'm missing something?**
>
> Here's why I'm stuck: I did **not** author these two changes — they were present in the working tree when I started this task. They **add** `templ-components v0.3.0` (require) and `replace => ../../templ-components`. But the **committed** `icons.go` (`8091422`) no longer imports that package, and I proved `GOWORK=off go build ./...` **succeeds with these changes stashed**. That makes them a stale artifact that re-introduces a ghost dependency the prior brutal review (committed in the same `8091422`) explicitly claims to have removed.
>
> Per my operating rules I will **not revert changes I didn't author without confirmation.** My default recommendation is `git restore adminui/go.mod adminui/go.sum` (HEAD is clean + builds), but if these were staged for a reason (e.g. you're mid-migration back to `templ-components` once it publishes the missing symbols), tell me and I'll leave them. **Which do you want?**

---

## Verification commands (re-runnable)

```bash
# All modules
nix run .#test          # or per-module GOWORK=off go test ./... -count=1 -race
nix run .#lint          # golangci-lint, 0 issues
nix run .#errorfamily   # branching-flow, 0 violations
# Consumer-build check (the one that should be in CI)
cd adminui && GOWORK=off go build ./...   # exit 0
# Duplication
art-dupl --semantic --sort total-tokens -t 3   # 3 irreducible idiom groups
```

_Arte in Aeternum._
