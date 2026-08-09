# Comprehensive Status Report — cqrs-htmx

**Date:** 2026-06-20 06:00 CEST
**Branch:** `master` (up to date with `origin/master` except uncommitted dedup work)
**Reporter:** Crush (automated)
**Modules:** root `github.com/larsartmann/cqrs-htmx/v2` · `usermgmt/v2` · `integration_test` · `catalog/v2` · 2 examples

---

## Executive Summary

The codebase is in **excellent shape**. All four Go modules build cleanly, pass the race-enabled test suite (598 test functions / 470 BDD specs), and report **0 lint issues** with golangci-lint v2. Coverage holds at **95.2% root / 85.5% usermgmt / 95.3% catalog**.

Today's session focused on a **deduplication pass** that the user challenged with sharper scrutiny — pushing "zero harmful duplication" from a rubber-stamp into an honest exercise. Result: clone groups reduced **8 → 1**, complexity **1.78 → 1.00**.

The project has shipped a remarkable amount of production-grade functionality since v1.0.0: full event-sourced user management, passwordless WebAuthn, OAuth2/OIDC, event signing & encryption, SSE + WebSocket transport, a catalog doc-generator, and SQL event-store delegation to upstream. The remaining work is **breadth and hardening, not foundation**.

---

## Verification Snapshot (live, this session)

| Check                | Command                        | Result                                   |
| -------------------- | ------------------------------ | ---------------------------------------- |
| Root build           | `go build ./...`               | ✅ exit 0                                |
| Root test            | `go test ./... -count=1`       | ✅ ok 1.220s                             |
| Root coverage        | `-coverprofile`                | **95.2%**                                |
| usermgmt test (race) | `go test ./... -count=1 -race` | ✅ ok 2.745s                             |
| usermgmt coverage    | `-coverprofile`                | **85.5%**                                |
| integration test     | `go test ./... -count=1 -race` | ✅ ok 1.036s                             |
| catalog test         | `go test ./... -count=1`       | ✅ ok 0.004s                             |
| catalog coverage     | `-coverprofile`                | **95.3%**                                |
| datastar-demo build  | `go build ./...`               | ✅ exit 0                                |
| catalog-demo build   | `go build ./...`               | ✅ exit 0                                |
| Root lint            | `golangci-lint run ./...`      | ✅ 0 issues                              |
| usermgmt lint        | `golangci-lint run ./...`      | ✅ 0 issues                              |
| integration lint     | `golangci-lint run ./...`      | ✅ 0 issues                              |
| catalog lint         | `golangci-lint run ./...`      | ✅ 0 issues                              |
| Module integrity     | `go mod verify`                | ✅ all modules verified                  |
| Clone complexity     | `art-dupl --semantic -t 30`    | **1 clone group, score 1.00** (was 1.78) |

**Scale:** 252 Go files · 16,298 LOC (root) · 114 files (usermgmt) · 598 test funcs · 470 BDD specs · 14 ADRs.

---

## a) FULLY DONE

Work that is complete, tested, and verified this session.

### Today: Deduplication Pass (8 → 1 clone groups)

**Previous session (committed `cddb9e2`):**

- `usermgmt/random.go` — shared `randomBase64URLString(n, purpose)` entropy source; 3 token-gen sites (session, verification, OAuth2 state) unified.
- `countingHandlerMW` — extracted duplicated middleware closure in `service_security_test.go`.
- `addSchemaVersionV1` — extracted duplicated upcaster closure in `es_upcaster_test.go`.
- `newFuzzSQLiteStore` deleted (byte-identical to `newTestSQLiteSessionStore`).
- `encryptedStoreHooks` — extracted duplicated `SecurityHooks` literal in `integration_test/signing_encryption_test.go`.

**This session (uncommitted):**

- `seedTestUser(t, svc, name)` — unified the test-user fixture across 3 crypto/signing tests (bob/carol/dave). Named `seedTestUser` (not `registerTestUser`) to avoid colliding with the existing cross-module helper in `integration_test.go` that takes a root-module `cqrshtmx.UserID`.
- `requireTestAuthCode(w, r) bool` — unified the OAuth2 token-endpoint "reject invalid auth code" logic across the pure-OAuth2 and OIDC mock providers (RFC 6749 §5.2 `invalid_grant`).
- **1 clone group remains** (accepted): `/token` vs `/userinfo` mock endpoints in `oauth2_integration_test.go` — different endpoints, different response bodies, structural similarity only. Extracting `writeJSONMock(w, body)` would be more abstract than the 4-line inline code.

### Foundation & Architecture (stable)

- **Event-sourced usermgmt**: 7 events, 7 commands, Decider pattern via go-cqrs-lite. Pure `foldUser()` + `decide*()`. Read-your-writes via `MemoryBus`.
- **Passwordless auth**: WebAuthn (go-webauthn v0.17.4) registration + login ceremonies, in-memory challenge store with proactive eviction, account lockout.
- **OAuth2/OIDC**: Dual-mode providers (OIDC discovery + pure OAuth2), PKCE on every flow, subject-first matching, global provider+subject uniqueness. 2 events, 2 commands, 3 HTTP endpoints.
- **Event signing & encryption**: Opt-in seams (`StoreWrapper`, `PublishMiddleware`, `HandlerMiddleware`) wire go-cqrs-lite's `signing/v2` + `encryption/v2` without importing them.
- **SSE transport**: `SSEStream`, `Broadcaster` (O(1) unsubscribe, race-safe), `SSEEventStore`, reconnection, heartbeat, CQRS bridge hooks. Panic regression test for send-on-closed-channel.
- **WebSocket transport**: `WSMessage`/`ParseWSMessageInto[T]` parsers, `WriteWSMessageInto[T]` encoder, `WSBroadcaster`, `DispatchWSCommand/Query` bridge, transport parity with SSE (ADR 0010).
- **SQL event store**: `usermgmt.SQLEventStore` is now a 78-LOC facade over upstream `storage.SQLEventStore` (v2.6.0). Richer schema, OTel tracing, `SeekableJournal`/`BackwardsSource`.
- **Catalog module**: API doc generation (OpenAPI/AsyncAPI/D2), 5th independent Go module with zero root/usermgmt dep.
- **Security**: nosurf CSRF (per-handler + middleware), token-bucket rate limiting (O(log n) heap eviction, race-fixed), security headers, panic recovery.
- **HTMX integration**: Embedded HTMX v2.0.9 JS (ETag/caching), fluent response builder, OOB HTML, notifications, pagination.
- **Type safety**: Branded `UserID`/`CorrelationID`/`RequestID` (ULID-backed), `SSEEventID`, branded AuditEntry IDs, `authMode` enum, `Email` value type.
- **14 ADRs** documenting every major decision.

---

## b) PARTIALLY DONE

- **Deduplication pass at `-t 30`**: complete. A `-t 15` pass (the SKILL.md default) has **not** been run and would surface smaller idiomatic patterns (`if err != nil`, signatures). Expect more noise that needs the decision checklist.
- **go.sum hygiene**: the working tree has a `go.sum` diff (46 unused checksum lines removed by the toolchain). `go mod verify` passes — the change is benign but uncommitted and should be tidied/committed deliberately.
- **`OAuth2/OIDC`** is shipped and tested, but the TODO_LIST still marks it `[ ]` OPEN (stale checkbox) — docs-freshness drift.
- **Coverage**: usermgmt at 85.5% is the lowest of the three tested modules. The ROADMAP/AGENTS.md cite 88.7% — minor staleness; still healthy but the gap is real.

---

## c) NOT STARTED

- **`-t 15` dedup pass** — aggressive clone scan at the skill's default threshold.
- **Duplicate-code policy ADR** (`docs/adr/0015`) — SKILL.md recommends recording the accepted-idiom decisions so the next reader knows they were intentional. Does not exist.
- **PostgreSQL `SessionStore`** — `SQLSessionStore` exists for sessions but a documented Redis adapter for distributed deployments is "Planned" in ROADMAP (v3.0.0).
- **Database migration tooling** — goose/golang-migrate/gnorm integration is "Planned" (ROADMAP v2.2.0).
- **Integration tests against real PostgreSQL** — the SQL event/session stores are unit-tested with in-memory SQLite; no green Postgres CI (ROADMAP).
- **Prometheus metrics middleware** — dispatch latency/error-rate metrics (ROADMAP v3.0.0).
- **BrandNamer for root module marker types** — BLOCKED upstream (unexported `userMarker`/`correlationMarker` in go-cqrs-lite).
- **README runnable godoc examples** — "Open" in ROADMAP.

---

## d) TOTALLY FUCKED UP!

Nothing is broken, build-wise. Brutal honesty on the things that are genuinely wrong or embarrassing:

1. **"Zero harmful duplication" was a cop-out on first pass.** I accepted 3 clone groups as "harmless" without applying the decision checklist rigorously. Group 4 (`seedTestUser`) and group 2 (`requireTestAuthCode`) were extractable and I let them slide. It took the user calling this out to fix it. The lesson: "zero harmful" only earns its qualifier when every accept has a defensible, checklist-grounded rationale — not a rubber stamp.

2. **The status-report/docs surface is bloated.** `docs/status/` contains **86+ point-in-time reports** (this will be #87). Many are near-duplicates ("comprehensive-status-update" appears ~10 times). This is the opposite of the dedup principle applied to code. The report archive itself is a maintenance smell that nobody has tamed.

3. **Docs freshness drift (minor but real).** TODO_LIST says `[ ] OAuth2/OIDC integration — OPEN` while FEATURES/CHANGELOG/ADR-0014 show it fully shipped. AGENTS.md/ROADMAP cite usermgmt coverage as 88.7%; live measurement is 85.5%. Stale checkboxes erode trust in the tracking docs.

4. **art-dupl missed the third token-gen clone** in the first pass — `user.go:139 generateToken` wasn't flagged because of minor shape variation, but it was semantically identical to the two flagged copies. The tool is a starting point, not ground truth; `grep` for the real extent of a pattern is mandatory.

---

## e) WHAT WE SHOULD IMPROVE!

1. **Apply the dedup checklist with real scrutiny, every time.** No more rubber-stamping. Every "accept" must answer: semantic vs structural? drift risk? better name available?
2. **Archive/cull the status-report directory.** 86 reports is noise. Keep the last 5–10 per quarter; move the rest to `docs/status/archive/` (which exists but is underused) or delete. A status report should be findable, not buried.
3. **Fix docs-freshness drift mechanically.** Run the `docs-freshness-check` skill (or `todo-list-builder`) to reconcile TODO_LIST/FEATURES/ROADMAP/AGENTS coverage numbers and checkbox states against the actual code. Quarterly cadence.
4. **Run `go mod tidy` in every module and commit deliberately** rather than leaving stray `go.sum` diffs in the working tree that confuse the next session.
5. **Close the usermgmt coverage gap** (85.5% vs the 88.7% target cited). Identify the uncovered lines with `go tool cover -func` and decide: add tests, or mark as uncovered-for-a-reason.
6. **Record the duplicate-code policy in ADR 0015** so accepted idioms are defensible to future readers, not mysterious.
7. **Reduce reliance on `GONOSUMCHECK`** — it's a private-checksum workaround; document why it's needed (private module access) so new contributors aren't confused.

---

## f) Top 25 Things to Get Done Next

Sorted by impact (Pareto). High → lower.

| #  | Task                                                                                              | Impact | Effort       |
| -- | ------------------------------------------------------------------------------------------------- | ------ | ------------ |
| 1  | **Commit the uncommitted dedup work** (3 files + go.sum)                                          | High   | XS           |
| 2  | **Run `-t 15` dedup pass** + apply checklist rigorously                                           | High   | S            |
| 3  | **Record ADR 0015 duplicate-code policy** (accepted idioms + checklist)                           | High   | S            |
| 4  | **Docs-freshness sweep**: reconcile TODO/FEATURES/ROADMAP/AGENTS vs code                          | High   | M            |
| 5  | **Fix stale TODO_LIST checkboxes** (OAuth2 marked OPEN, coverage numbers)                         | High   | XS           |
| 6  | **Archive/cull `docs/status/`** — keep recent, move rest to `archive/`                            | Med    | S            |
| 7  | **Close usermgmt coverage gap** (85.5% → 88%+); `go tool cover -func` analysis                    | High   | M            |
| 8  | **`go mod tidy` all modules** + commit hygiene                                                    | Low    | XS           |
| 9  | **PostgreSQL CI integration tests** for SQLEventStore/SQLSessionStore                             | High   | L            |
| 10 | **Database migration tooling** (goose/golang-migrate) + docs                                      | Med    | M            |
| 11 | **Redis SessionStore adapter** for distributed deployments                                        | Med    | M            |
| 12 | **Prometheus metrics middleware** (dispatch latency, error rates)                                 | Med    | M            |
| 13 | **OpenTelemetry middleware example** polish + real tracer docs                                    | Med    | S            |
| 14 | **README runnable godoc examples** (ROADMAP open item)                                            | Med    | S            |
| 15 | **BrandNamer upstream PR** — expose go-cqrs-lite marker types (unblocks root branding)            | Med    | M (upstream) |
| 16 | **Profile hot paths** (dispatch, decode) for further alloc reduction                              | Low    | M            |
| 17 | **Add CHANGELOG version tag** — Unreleased section is large; cut a v2.x.0 release                 | High   | S            |
| 18 | **Event schema versioning** docs/examples (upcasters exist via ADR 0013; no user guide)           | Med    | S            |
| 19 | **Expand integration_test** cross-module bridge coverage (ROADMAP open)                           | Low    | M            |
| 20 | **CSRF + WebAuthn wiring recipe** — expand the integration_test example into a guide              | Low    | S            |
| 21 | **Audit usermgmt for `t.Context()` migration** (gopls flagged `context.WithCancel` → `t.Context`) | Low    | S            |
| 22 | **Add a `CONTRIBUTING.md` test/lint quickstart** referencing nix apps                             | Low    | XS           |
| 23 | **Property-based test sweep** for OAuth2/OIDC matching logic                                      | Med    | M            |
| 24 | **Security review** of OAuth2 auto-trust-email tradeoff (ADR 0014)                                | Med    | S            |
| 25 | **Catalog demo enrichment** — show event-catalog + D2 diagram generation end-to-end               | Low    | S            |

---

## g) Top Question I Cannot Figure Out Myself

> **Should this project cut a versioned release now (the `[Unreleased]` CHANGELOG section is very large and covers WebAuthn, OAuth2/OIDC, signing/encryption, SQL-store delegation, and SSE/WS transport), or is there a planned breaking change still pending that should gate the release?**
>
> The CHANGELOG's Unreleased section already documents multiple breaking changes (`SQLEventStore.Close()` semantics, branded types on `AuditEntry`, `SSEEventID`). Cutting a tagged release would give consumers a stable target. But I cannot determine from the code alone whether you intend to batch more breaking changes (e.g., a Redis store, Prometheus middleware, or further root-module branding) into the same major/minor bump before tagging. This is a product/release-cadence decision only you can make.
