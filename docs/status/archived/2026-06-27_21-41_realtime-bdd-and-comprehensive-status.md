# Status Report — cqrs-htmx

**Date:** 2026-06-27 21:41 CEST
**Branch:** `master` (clean, up to date with `origin/master`)
**Session focus:** BDD test coverage for the realtime (SSE + WebSocket) layer + comprehensive health snapshot.
**Version:** v3.1.0 (go-cqrs-lite v3.1.0)

---

## TL;DR — Health at a glance

| Check               |   Root    | usermgmt  |  adminui  |  catalog  | integration_test | examples (4) |
| ------------------- | :-------: | :-------: | :-------: | :-------: | :--------------: | :----------: |
| **Build**           |    ✅     |    ✅     |    ✅     |    ✅     |        ✅        |      ✅      |
| **Lint (0 issues)** |    ✅     |    ✅     |     —     |    ✅     |        —         |      —       |
| **Tests (-race)**   |    ✅     |    ✅     |    ✅     |    ✅     |        ✅        |      —       |
| **Coverage**        | **95.6%** | **79.6%** | **56.3%** | **95.3%** |        —         |      —       |
| **ErrorFamily**     |     0     |     0     |     0     |     —     |        —         |      —       |
| **Test count**      |  95 RUN   |    699    |    19     |    ~41    |        15        |      —       |

**Everything is green.** No broken builds, no lint violations, no test failures under the race detector anywhere. The library is in a **release-quality state** for the publishable modules (root, usermgmt, catalog). The new `adminui` module ships and works but has the lowest coverage.

---

## a) FULLY DONE — shipped, tested, documented

### This session: Realtime BDD coverage

- **`bdd_realtime_test.go`** — 15 new behavioral Ginkgo specs covering the **SSE + WebSocket realtime layer** through consumer user stories. This was the highest-value test gap: the realtime layer previously had only mechanical/concurrency tests (event-writing, fan-out, race guards), not behavioral specs framed as what a _user_ (a developer consuming the library) observes.
- Coverage: SSE protocol headers, multi-client fan-out, multi-line payload splitting, `SendHTML`, reconnection replay (`Last-Event-ID`), first-visitor detection, id/retry hints, wire-format corruption rejection, heartbeat comment-frames (concurrency-safe via `lockedRecorder`), `OnDisconnect` cleanup, `BroadcastOnSuccess` CQRS bridge (fires on success, silent on failure), WebSocket body/headers separation, `ParseWSMessageInto[T]` typed parsing, `WSOOBHTML` out-of-band swap.
- Fix collateral: `.golangci.yml` gains `lockedRecorder` to the `exhaustruct` exclusion list (matches existing test-helper convention).

### Library — production-ready surface

- **Root module** (`cqrs-htmx/v3`, 95.6%): App builder, command/query dispatch, 40-file cohesive flat package, handler options (decode/render/validate/HTMX), Casbin `Enforcer` interface, branded types (`UserID`/`CorrelationID`/`RequestID`), CSRF (nosurf, per-handler, TrustedProxies), rate limiting (token-bucket + min-heap LRU eviction), security headers, recovery middleware, request logging (text/JSON/slog), SSE (broadcaster, stream, reconnection, CQRS bridge), WebSocket (parser, encoder, OOB HTML, broadcaster, dispatch bridge), embedded HTMX v2.0.9 JS.
- **usermgmt module** (`usermgmt/v3`, 79.6%, 699 tests): Fully **event-sourced** user aggregate (12 events, 11 commands, Decider pattern). **Passwordless** — WebAuthn/Passkeys (go-webauthn v0.17.4) only. OAuth2/OIDC integration (Google + GitHub patterns). TOTP MFA (pquerna/otp). Email verification. Import/export (JSON/CSV). Identity model: Tenant, Bot, Membership, Impersonation, ActorID. SQL read models (4 aggregates, v3.1.0). One-call SQLite/Postgres stack presets. Casbin RBAC projection from events.
- **catalog module** (`catalog/v3`, 95.3%): API doc generation — OpenAPI/AsyncAPI/D2 handlers, generic Builder, schema reflection. Zero dep on root/usermgmt.
- **adminui module** (`adminui/v3`): Ready-made Admin Dashboard (templ + HTMX). Two scopes (SuperAdmin/TenantAdmin). Auth-agnostic. Embedded CSS/JS. The leaf integration module — nothing depends on it. Has a runnable showcase (`examples/admin-demo`).
- **integration_test module**: Cross-module bridge tests (CSRF + WebAuthn, catalog, typed queries).
- **Examples** (4): `basic`, `datastar-demo`, `catalog-demo`, `admin-demo` — all build clean.
- **Error discipline**: `branching-flow errorfamily` reports **0 stdlib error constructors** across root, usermgmt, AND adminui. Enforced via CI gate.
- **Documentation**: 14 ADRs, DOMAIN_LANGUAGE.md, README, CHANGELOG, FEATURES.md, TODO_LIST.md, ROADMAP.md, CONTRIBUTING, SECURITY.

---

## b) PARTIALLY DONE — ships but with known gaps

| Item                                   | Gap                                                                                                                                                                                                                                                                                                                                                                                | Severity       |
| -------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------- |
| **adminui coverage (56.3%)**           | Lowest module. Brand-new (committed 2026-06-27). Has an end-to-end smoke test (`seed_render_test.go`) but many handler paths untested.                                                                                                                                                                                                                                             | High           |
| **Type safety in context (9 items)**   | `ActorID`/`ImpersonatorID` stored as raw `string` in `context.go`; `foldUser` silently no-ops on unknown events; `BotState.OwnerID` raw string; `TenantID` not used in Authz domain params; `Email` branded type defined but unused in structs; duplicate `ErrUnauthorized` sentinel across modules; impossible `TenantState` (Suspended+Deleted) allowed. From data-model-review. | Med-High       |
| **WebAuthn service layer coupling**    | `webauthn_service.go:52,154` take `*http.Request` — HTTP leaks into the service layer. Should parse in HTTP handler, pass bytes to service.                                                                                                                                                                                                                                        | Medium         |
| **Ephemeral stores have no interface** | 6 in-memory stores (WebAuthn sessions, verification tokens, TOTP, lockout, rate limiter, read models) lack interfaces → multi-instance impossible.                                                                                                                                                                                                                                 | Medium         |
| **v3.1.0 docs not fully closed**       | Consumer migration guide (v2→v3), godoc examples for entry points, `VERSIONING.md` — all Open in ROADMAP.                                                                                                                                                                                                                                                                          | Medium         |
| **Snapshot integration**               | `service_core.go` replays all events on startup. `go-cqrs-lite/snapshot/v3` is an indirect dep but not wired.                                                                                                                                                                                                                                                                      | Low            |
| **FEATURES.md metrics table stale**    | Reports 95.2%/82.8% root/usermgmt (actual 95.6%/79.6%); test counts (59/517) don't match reality (95/699); `adminui` absent from the table entirely. Docs-freshness drift.                                                                                                                                                                                                         | Low (cosmetic) |

---

## c) NOT STARTED

From ROADMAP.md (future versions, all `Planned`):

**v3.2.0 — Observability & Metrics**

- OpenTelemetry wiring via go-cqrs-lite v3 otel module (hook _pattern_ is documented in `example_otel_test.go`, but no real SDK integration).
- Prometheus metrics middleware (dispatch latency, error rates).

**v3.3.0 — Persistence & Scale**

- Redis session store / Redis OAuth2 state store (multi-instance).
- PostgreSQL session store preset, BadgerDB alternative.
- Streaming replay via `SeekableJournal.ReadFrom`.
- Projection replay benchmarking at 10K+ events.

**v4.0.0 — Advanced Event Sourcing**

- `stack.Materialize` for declarative persistent read models (evaluated, rejected for now — our event handling is too complex).
- `CatchUpSubscriber` as alternative to manual journal replay.
- schema/v3 validator for event payloads.

**Smaller open items (TODO_LIST):**

- Service-level impersonation tests through full dispatch.
- Service-level membership tests through full dispatch.
- Projection replay integration test (journal vs live dedup).
- Property-based tests for `foldTenant`/`foldBot`/`foldMembership`.
- Enable `revive:exported` linter + fix violations.
- Remove deprecated `ClientIP()` wrapper.
- Verify/wire `BrandNamer` for root module marker types.

---

## d) TOTALLY FUCKED UP? — honest assessment

Nothing is broken or actively regressing. But three things deserve a blunt call-out:

1. **`adminui` is under-tested for a "ready-made" module (56.3%).** It's marketed as drop-in but has thin handler coverage. If a consumer hits an edge case in member management, tenant admin scoping, or the toast/redirect rendering, there's little test safety net. The module is _new enough_ that this is acceptable _now_, but it should not ship under a "stable" tag at this coverage.

2. **`go.sum` has uncommitted checksum drift.** Two indirect checksum lines (`pborman/getopt`, `check.v1` `/go.mod` hashes) appeared in the working tree — likely from a `go test`/`go build` resolving the transitive graph. It's harmless, but an uncommitted `go.sum` in a library is a smell. (Being committed this session.)

3. **Stale LSP false-positive.** A `gosec G118` warning lingers in the LSP for `bdd_realtime_test.go:29` even though (a) that line no longer contains a `WithCancel`, and (b) the authoritative `golangci-lint run` CLI reports **0 issues**. This is the known "LSP vs CLI discrepancy" (AGENTS.md gotcha #6). Annoying but not real.

**Verdict:** No fucked-up-ness. The codebase is genuinely in strong shape. The risks are _under-tested new code_ (adminui) and _deferred type-safety debt_ (the 9 data-model items), not breakage.

---

## e) WHAT WE SHOULD IMPROVE

### Highest leverage

1. **Raise adminui coverage to ≥75%.** It's the weakest module and the newest — invest now while the code is fresh in mind. The smoke test proves the happy path; add negative paths, authz denial, tenant-scope isolation, and render-fragment assertions.
2. **Pay off the type-safety debt in one focused pass.** The 9 data-model-review items (ActorID in context, foldUser error on unknown events, Email/TenantID/BotID branded types) compound — every day they stay raw `string` is a day someone could introduce a bug the compiler would otherwise catch. "Make impossible states unrepresentable" is an explicit project principle.
3. **Close the FEATURES.md/ROADMAP.md metrics drift.** A stale metrics table erodes trust in the docs. Run the docs-freshness-check skill and reconcile numbers (or generate them).

### Structural

4. **Extract `*http.Request` from the WebAuthn service layer.** This is the cleanest architectural win available — it removes a layering violation and makes the service unit-testable without HTTP fixtures.
5. **Add interfaces for ephemeral stores.** Unblocks multi-instance deployments (Redis adapters) and makes the in-memory defaults swappable. The `OAuth2StateStore` interface is the model to copy.
6. **Decide adminui's release status explicitly.** It's on the `/v3` path and in `go.work` but absent from the ROADMAP version table. Either add it to a version milestone with a coverage gate, or mark it experimental.

### Quality of life

7. **Wire the coverage gate into CI properly** — `nix run .#coverage-gate` exists (root 90%, usermgmt 75%, catalog 90%) but adminui has no threshold. Add one (e.g., 70%, rising).
8. **Snapshot integration** — event replay on startup will get slow as event counts grow. `go-cqrs-lite/snapshot/v3` is already an indirect dep; wiring it is low-risk and high-scalability-value.
9. **More behavioral BDD elsewhere.** The realtime layer now has good BDD coverage; the _command/query dispatch + error mapping_ surface has `bdd_test.go` but the _auth/CSRF/rate-limit_ surfaces are tested mechanically. User-story framing would catch integration regressions better.

---

## f) Top 25 things to get done next

Sorted by impact × urgency (Pareto). Status: 🔴 blocker / 🟠 high / 🟡 medium / 🟢 nice-to-have.

| #  | Priority | Task                                                                                                    |
| -- | -------- | ------------------------------------------------------------------------------------------------------- |
| 1  | 🟠       | Raise **adminui coverage** to ≥75% — handler negative paths, authz denial, tenant-scope isolation       |
| 2  | 🟠       | Type `ActorID`/`ImpersonatorID` in `context.go` (stop storing raw `string`) — CRITICAL type-safety item |
| 3  | 🟠       | Make `foldUser` return error on unknown events (parity with foldMembership/foldTenant/foldBot)          |
| 4  | 🟠       | Extract `*http.Request` from `webauthn_service.go` (HTTP out of service layer)                          |
| 5  | 🟠       | Add interfaces for the 6 ephemeral in-memory stores (multi-instance enabler)                            |
| 6  | 🟠       | Use branded `Email`/`TenantID`/`BotID` types in domain structs (stop raw `string`)                      |
| 7  | 🟡       | Reconcile **FEATURES.md/ROADMAP.md metrics** (numbers are stale; adminui missing)                       |
| 8  | 🟡       | Add adminui to a ROADMAP version milestone (decide: stable vs experimental)                             |
| 9  | 🟡       | Set adminui **coverage gate** threshold in CI (`.#coverage-gate`)                                       |
| 10 | 🟡       | Service-level **impersonation tests** through full dispatch                                             |
| 11 | 🟡       | Service-level **membership tests** through full dispatch                                                |
| 12 | 🟡       | Projection replay integration test (journal vs live dedup)                                              |
| 13 | 🟡       | Resolve duplicate `ErrUnauthorized` sentinel across module boundary                                     |
| 14 | 🟡       | Prevent impossible `TenantState` (Suspended + Deleted simultaneously)                                   |
| 15 | 🟡       | Validate `actorKindFromString` (silently defaults unknown → ActorUser)                                  |
| 16 | 🟡       | Consumer **migration guide** (v2→v3: import paths, bus, projections)                                    |
| 17 | 🟡       | Enable `revive:exported` linter + fix violations                                                        |
| 18 | 🟢       | Godoc examples for App, Handler, Service entry points                                                   |
| 19 | 🟢       | Property-based tests for `foldTenant`/`foldBot`/`foldMembership`                                        |
| 20 | 🟢       | Wire **snapshot integration** (`snapshot/v3`) to speed up startup replay                                |
| 21 | 🟢       | Remove deprecated `ClientIP()` wrapper                                                                  |
| 22 | 🟢       | `LastEventIDFromRequest` should delegate to `SSEStream.LastEventID()` (dedupe)                          |
| 23 | 🟢       | OpenTelemetry real-SDK integration (v3.2.0) — pattern documented, SDK not wired                         |
| 24 | 🟢       | Redis session/OAuth2 state stores for multi-instance (v3.3.0)                                           |
| 25 | 🟢       | `VERSIONING.md` documenting semver policy                                                               |

---

## g) Top #1 question I cannot figure out myself

> **What is `adminui`'s intended release status?**
>
> It's brand-new (committed this week), on the `/v3` module path, in `go.work`, has a runnable demo, and is documented in AGENTS.md as a first-class module — yet it's **absent from the ROADMAP version table** and has **56.3% coverage** (vs 79–96% everywhere else).
>
> **Is adminui a first-class module that should ship under the next stable tag (making its coverage gap a release blocker that I should prioritize), or is it experimental and exempt from the coverage gate?**
>
> This determines whether item #1 above is urgent or deferrable, and whether I add it to the v3.1.x/v3.2.0 milestone. I can't infer the answer from the repo state alone — the signals conflict (production-quality polish vs. low coverage vs. no version assignment).

---

## Session artifacts

- **New file:** `bdd_realtime_test.go` (15 BDD specs, realtime layer)
- **Modified:** `.golangci.yml` (`lockedRecorder` exhaustruct exclusion), `go.sum` (2 indirect checksum lines)
- **Verified:** full suite green (`go test ./... -count=1 -race` all modules), `golangci-lint run` 0 issues, `errorfamily` 0 violations, all 4 examples build.
