# Status Report — cqrs-htmx

**Date:** 2026-06-28 23:03
**Branch:** `master` (clean, pushed)
**Commit:** `9a74f7b`
**Coverage:** 95.4% root, 80.1% usermgmt
**Lint:** 0 issues across all 4 modules
**Tests:** 933 total (136 root + 745 usermgmt + 35 adminui + 17 integration) — all pass with `-race`

---

## Executive Summary

The codebase is in its **cleanest state ever** after a massive day of work: 90
commits, 2 split brains resolved (ActorID branding + rate limiter unification),
2 bugs fixed (TOCTOU race + memory leak), 1 ghost system killed (idempotency
wiring), 1 architectural question answered definitively (decide() stays on
server — ADR-0027), 3 ADRs added (0026, 0027, 0028), and all documentation
updated to match reality.

**Zero lint issues. Zero errorfamily violations. Zero banned dependencies.**
**933 tests pass with -race across all 4 modules.**

---

## A) FULLY DONE

### Core Library (root module) — 136 tests

| Item                               | Notes                                                                                                                                               |
| ---------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------- |
| **ActorID/ImpersonatorID branded** | `brandid.ID[actorBrand, string]` — phantom-typed, `.Get()`, `.IsZero()`, `.Equal()`, BrandNamer. `ImpersonatorID = ActorID` (type alias). ADR-0028. |
| **SSEEventID branded**             | `brandid.ID[sseEventBrand, string]` — same treatment. `.Get()` for raw value, `.String()` for debug.                                                |
| **go-branded-id direct dep**       | Promoted from `// indirect` to direct in root go.mod                                                                                                |
| **Idempotency store**              | `IdempotencyStore` interface + `MemoryIdempotencyStore` with truly atomic `CheckAndRecord`. Lazy expiry in `Seen()`. ADR-0026.                      |
| **Idempotency wired**              | admin-demo rejects duplicate `X-Command-Id` with 409 (ghost system killed)                                                                          |
| **Form decoder**                   | `go-playground/form/v4` replaces JSON round-trip. Case-insensitive matching.                                                                        |
| **Pagination unified**             | Root + usermgmt delegate to `query.NewPagination`. No silent clamping.                                                                              |
| **SSE infrastructure**             | JournalSSEStore, Broadcaster, SSEStream, reconnection, heartbeat                                                                                    |
| **Honest UI protocol**             | CommandAck, BroadcastOnAck hooks, admin-demo lifecycle                                                                                              |
| **WebSocket parity**               | WSBroadcaster, DispatchWSCommand/Query, OOB HTML                                                                                                    |
| **Security**                       | CSRF (nosurf), rate limiting (token bucket), security headers, recovery                                                                             |
| **Embedded HTMX v2.0.9**           | Self-hosted via go:embed                                                                                                                            |
| **Deprecated ClientIP removed**    | Zero callers; tests use httputil.ClientIP directly                                                                                                  |

### usermgmt submodule — 745 tests

| Item                         | Notes                                                                                   |
| ---------------------------- | --------------------------------------------------------------------------------------- |
| **Fully event-sourced CQRS** | 12 events, 11 commands, pure decide/fold, 4 aggregates                                  |
| **Passwordless auth**        | WebAuthn/Passkey, TOTP, OAuth2/OIDC                                                     |
| **Rate limiter unified**     | perIPRateLimiter DELETED → root RateLimiter (token-bucket, proxy-aware, bounded memory) |
| **ActorID bridging fixed**   | `.PrefixedString()` instead of `.String()` — prefix survives crossing                   |
| **SQL read models**          | SQLite + Postgres, 4 aggregates                                                         |
| **Casbin projection**        | RBAC policies derived from events                                                       |

### adminui module — 35 tests

| Item                | Notes                                                    |
| ------------------- | -------------------------------------------------------- |
| **Admin Dashboard** | templ + HTMX, light/dark, SuperAdmin + TenantAdmin modes |

### Documentation

| Item            | Notes                                                                                     |
| --------------- | ----------------------------------------------------------------------------------------- |
| **28 ADRs**     | 0026 (idempotency), 0027 (decide stays on server), 0028 (brand all ID types)              |
| **CHANGELOG**   | [Unreleased] documents all recent work + breaking changes                                 |
| **AGENTS.md**   | Updated: one-way dependency, rate limiter unification, branded types, correct test counts |
| **CI umbrella** | `nix run .#test` + `nix run .#lint` + `nix run .#errorfamily` all green                   |

---

## B) PARTIALLY DONE

### Offline-First Command Sync (Phase 0+1 done, Phase 2 blocked)

- **Phase 0 (server infrastructure):** DONE — JournalSSEStore, Broadcaster, SSEStream, CommandAck
- **Phase 1 (honest UI protocol):** DONE — CSS/JS/templ, ACK protocol, admin-demo
- **Phase 2 (client-side queue + offline persistence):** NOT STARTED — Q1 answered (ADR-0027: Queue-Only), Q2 still open

### Idempotency

- **MemoryIdempotencyStore + admin-demo wiring:** DONE
- **Redis/Postgres-backed store:** NOT DONE — interface exists for future implementation

### Event Signing & Encryption

- **ServiceConfig seams:** DONE — StoreWrapper, PublishMiddleware, HandlerMiddleware
- **Runnable example:** NOT DONE — seams documented in ADR-0011 but unproven end-to-end

### adminui

- **Rendering + routing + honest UI:** DONE
- **Integration test coverage:** NOT DONE — only 35 tests, no integration test mounts the panel

---

## C) NOT STARTED

| Item                                                                       | Impact   | Blocked By             |
| -------------------------------------------------------------------------- | -------- | ---------------------- |
| Phase 2 client-side command queue + offline persistence                    | Critical | Q2 decision            |
| Redis/Postgres IdempotencyStore implementation                             | High     | —                      |
| adminui integration test (mount + route render)                            | High     | —                      |
| Security review: X-Command-Id injection/replay surface                     | High     | —                      |
| Split 7 files >300 lines (sql_session_store, service_core, response, etc.) | Medium   | —                      |
| JS test harness for admin.js sync-state handler                            | Medium   | —                      |
| Consumer wiring recipe doc (SSE + ACK + honest UI)                         | Medium   | —                      |
| Honest UI: inline error + retry button in rejected state                   | Medium   | —                      |
| SSE replay benchmark (10K/100K events)                                     | Low      | —                      |
| v3.3.0 release                                                             | High     | Q2 + release checklist |

---

## D) TOTALLY FUCKED UP

**Nothing is fucked up.** This is the cleanest state the codebase has been in:

- 0 lint issues (all modules)
- 0 errorfamily violations
- 0 banned dependencies (direct)
- 0 race detector failures (933 tests pass with `-race`)
- 0 stdlib error constructors
- 0 ghost systems (all exported types have callers)
- 0 deprecated code (ClientIP removed)

The only "fuckup" was AGENTS.md claiming "zero mutual imports" after the rate
limiter unification made it false — caught and fixed in self-review round 7.

---

## E) WHAT WE SHOULD IMPROVE

### High Priority

1. **Add adminui integration test** — 35 tests vs 747 in usermgmt. A templ regression could ship undetected.

2. **Security review of command ID protocol** — `X-Command-Id` accepts arbitrary client strings. Document max length, character restrictions, replay protection.

3. **Split files >300 lines** — 7 files exceed the limit. `sql_session_store.go` (424), `service_core.go` (418), `response.go` (358) are the worst offenders.

### Medium Priority

4. **usermgmt ActorID still a struct** — Root ActorID is branded, usermgmt ActorID is a kind-discriminated struct. Intentional but creates friction at the boundary. Consider a shared interface or typed conversion helpers.

5. **Consumer wiring recipe** — No end-to-end doc showing how to wire SSE + ACK + idempotency + honest UI in a real app.

6. **No benchmark baselines** — SSE replay performance with 10K/100K events is unknown.

---

## F) Top 25 Things to Do Next

Sorted by impact × effort × customer-value.

| #   | Task                                                                   | Impact   | Effort | Blocked? |
| --- | ---------------------------------------------------------------------- | -------- | ------ | -------- |
| 1   | **Answer Q2: closed-tab persistence?** (SharedWorker / Service Worker) | Critical | 5m     | **User** |
| 2   | Add adminui integration test (mount + route render)                    | High     | 30m    | —        |
| 3   | Security review: X-Command-Id injection/replay surface                 | High     | 30m    | —        |
| 4   | Implement Redis IdempotencyStore (SET NX + TTL)                        | High     | 45m    | —        |
| 5   | Document consumer wiring recipe (SSE + ACK + honest UI)                | Medium   | 30m    | —        |
| 6   | Split sql_session_store.go (424 lines)                                 | Medium   | 30m    | —        |
| 7   | Split service_core.go (418 lines)                                      | Medium   | 30m    | —        |
| 8   | Split response.go (358 lines)                                          | Medium   | 20m    | —        |
| 9   | Split http.go (355 lines)                                              | Medium   | 20m    | —        |
| 10  | Add actor/impersonator context bridge integration test                 | High     | 30m    | —        |
| 11  | Browser-verify admin-demo honest UI lifecycle                          | High     | 30m    | —        |
| 12  | Honest UI: inline error message in rejected state                      | Medium   | 30m    | —        |
| 13  | Honest UI: add retry button to rejected items                          | Medium   | 30m    | —        |
| 14  | Add adminui sync indicator rendering test (DOM flip)                   | Medium   | 30m    | —        |
| 15  | Add coverage gate for adminui                                          | Medium   | 20m    | —        |
| 16  | Signing/encryption runnable example (prove ADR-0011 seams)             | Medium   | 45m    | —        |
| 17  | Add JS unit tests for sync-state handler                               | Medium   | 30m    | —        |
| 18  | SSE replay benchmark (10K/100K events)                                 | Low      | 30m    | —        |
| 19  | Split oauth2.go (332 lines)                                            | Low      | 20m    | —        |
| 20  | Split app.go (331 lines)                                               | Low      | 20m    | —        |
| 21  | Fix responsive .sync-bar mobile layout                                 | Low      | 20m    | —        |
| 22  | Add PWA manifest to admin-demo                                         | Low      | 20m    | —        |
| 23  | Evaluate maypok86/otter/v2 for ephemeral stores                        | Low      | 30m    | —        |
| 24  | Phase 2 client-side command queue implementation                       | Critical | 2h+    | **Q2**   |
| 25  | Cut v3.3.0 release                                                     | High     | 30m    | Q2 + #2  |

---

## G) Top #1 Question

**Q2: Must writes survive closed tabs?**

Q1 (where does `decide()` run?) is **answered** — ADR-0027: Queue-Only.

Q2 remains: when a user closes the browser tab while offline commands are
queued, should those commands survive? This determines the persistence
architecture:

- **Tab-Scoped (SharedWorker)** — commands lost when tab closes. Simpler. Covers 90% of offline UX.
- **Persisted (Service Worker + Background Sync + IndexedDB)** — commands survive browser restart. Complex, Chrome-only Background Sync.

**My recommendation:** Tab-Scoped (SharedWorker) for MVP. The library already
provides the ACK protocol, SSE broadcast, and honest UI states. The consumer
just needs a JS command queue that flushes on reconnect.

---

## Numbers

| Metric                 | Value                     |
| ---------------------- | ------------------------- |
| Go files               | 341                       |
| Lines of Go            | 51,423                    |
| Go modules             | 8                         |
| ADRs                   | 28                        |
| TODOs open             | 2                         |
| TODOs done             | 93                        |
| Commits today          | 90                        |
| Tests                  | 933 (136 + 745 + 35 + 17) |
| Lint issues            | 0                         |
| ErrorFamily violations | 0                         |
| Direct deps (root)     | 12                        |
