# TODO List — cqrs-htmx

**Updated:** 2026-07-12 | **Coverage:** 94.2% root, 75.1% usermgmt, 88.2% totp, 87.5% webauthn, 92.3% oauth2, 66.8% adminui, 80.1% loginpage (~920 tests) | **Lint:** 0 issues (all modules) | **Version:** v4.2.1+unreleased (go-cqrs-lite v4.0.0)

## Status Legend

- [ ] OPEN — actionable, not yet started
- [~] PARTIALLY DONE — started but incomplete
- [x] DONE — completed and verified

---

## Open Items

### P0 — Release & Infrastructure

- [ ] **Create GitHub Releases** for 6 tags (`v4.2.1`, `usermgmt/v4.2.0`, `adminui/v4.2.0`, `totp/v4.0.2`, `webauthn/v4.0.2`, `oauth2/v4.0.2`) — only `v4.0.0` and `v2.0.0` have releases today. Tags are pushed but consumers see no release notes.
- [ ] **Verify `go get github.com/larsartmann/cqrs-htmx/v4@v4.2.1`** resolves from Go proxy — post-push verification never done
- [ ] **Check if pkg.go.dev** picked up v4.2.1 (Go documentation proxy)
- [ ] **Configure `go-auto-upgrade` to skip `encoding/json` → `encoding/json/v2` migration** — broke the build on 2026-07-09 (26 files migrated to experimental stdlib packages). Will recur on every buildflow run unless excluded.
- [ ] **Add depguard lint rule rejecting `encoding/json/v2` + `encoding/json/jsontext` imports** — depguard is NOT yet in `.golangci.yml` enable list despite earlier claim. Need to add `depguard` to linters + configure deny rules for both packages.

### P1 — Code Quality & Test Gaps

- [x] **Write CBOR round-trip test for `unmarshalPayload`** — 4 tests added: CBOR round-trip, JSON round-trip, CBOR-encoded event fold, mixed JSON+CBOR stream fold (`es_state_test.go`)
- [x] **Add tests for `DecodeFormWithRequest`, `DecodeJSONQueryWithRequest`, `DecodeFormQueryWithRequest`** — 3 Ginkgo specs added covering form decoding + request access + query dispatch (`feedback_features_test.go`)
- [x] **Fix `CSRFTestToken` to return `(token, cookie)`** — returns `(string, *http.Cookie)` now. GET→POST round-trip test added verifying token+cookie pass CSRF validation.
- [x] **Add `Code` field to `StructuredError` + `ProblemDetailsErrorHandler`** — `Code` field added to StructuredError, populated via `errorCode(err)` in `newStructuredErrorFromContext`. ProblemDetailsErrorHandler now emits `code` via StructuredError JSON. Split brain with JSONErrorHandler resolved.
- [x] **Fix exhaustive lint in `service_register.go:109`** — explicit `case event.Transient, event.Corruption, event.Infrastructure:` added (same body as default). LSP diagnostic resolved.
- [x] **Refactor `es_readmodel.go:Handle`** — DONE: extracted dispatch table (`handlers map[event.Type]userEventHandler`) with per-event handler methods. `maintidx` lint warning eliminated. All modules report 0 lint issues.
- [x] **Document empty-body behavior in `DecodeJSONQuery` godoc** — godoc updated for `DecodeJSON` and `DecodeJSONQuery` documenting zero-value T on empty body.

### P1 — Documentation & Process

- [ ] **Write pre-release verification script** (`nix run .#release-checklist`) — validates CHANGELOG updated, GitHub release body drafted, README version refs current, migration guide accurate before tagging
- [ ] **Add release process documentation to CONTRIBUTING.md** — tag naming, CHANGELOG update order, GitHub release creation steps
- [ ] **Write `nix run .#check-docs-freshness` app** — scans all `.md` files for version strings that don't match `go.mod`. Prevents `v3.1.0` → `v3.7.4` class of stale-reference bugs
- [ ] **Research go-cqrs-lite v4.0.0 release notes** — features discovered by reading module source (dedup.Ring, codec.ForEncoding, projectionhost, scheduling, testutil); there may be missed capabilities
- [ ] **Research httputil v0.5.0 changes** — upgraded from v0.4.0 but changelog not reviewed. Only use `httputil.ClientIP`
- [ ] **Research templ-components v0.15.0→v0.16.0 changes** — bumped in adminui/admin-demo. Check for new components or breaking changes

### P2 — Architecture

- [ ] **God-package split: domain layer extraction** — 20 pure fold/decide files in usermgmt (zero I/O) → `usermgmt/domain/`. #1 architectural debt. Clean seams identified in Sollbruchstellen analysis. v4.1 target.
- [ ] **Root module: extract SSE/WS/ratelimit into optional sub-packages** — 16 of 46 root files have zero logic coupling to core. Only useful as separate Go modules (same go.mod = same dep tree).
- [ ] **Consider shared types module** (`usermgmt/types/`) — for WebAuthnUserData, OAuth2UserInfo. Eliminates JSON serialization boundary design smell.
- [ ] **Raise strategy module dep budgets** — totp 2→3, webauthn 2→3, oauth2 4→5. All at capacity (0 slots). Any future feature requiring a new dep forces a budget increase.

### P2 — Features (Consumer-Requested)

- [ ] **Add `OnSubscribe`/`OnUnsubscribe` hooks to `fanOut`/`Broadcaster`** — DiscordSync asked for connection metrics. `fanout.go` has Subscribe/Unsubscribe/Broadcast/SubscriberCount/Close but no lifecycle hooks.
- [x] **Add `writeDispatchError(w, r, err)` helper** — consolidates 15 `writeError(w, errorStatus(err), err.Error())` call sites. Now includes error code in response body when available. `codeKey = "code"` constant added.
- [x] **Wire `ErrorContext()` into `RequestLoggingSlog`** — `StatusRecorder` captures dispatch error via `setDispatchError`. `handleErr` calls `captureDispatchError` (traverses Unwrap chain). `RequestLoggingSlog` now logs `error_code`, `error_family`, and `error_ctx_*` attributes from classified errors.
- [ ] **Consider `broadcaster.ServeSSE()` high-level helper** — 2 consumers (Overview, DiscordSync) wrote ~30 lines of identical SSE handler boilerplate. Design decision: does it cross the "building blocks, not a server" line?
- [ ] **Fix 2 remaining high-severity context losses** in `service_oauth2.go` — `providerName`/`state` flow through Service wrapping but aren't attached to the error.

### P2 — Testing

- [ ] **OAuth2 FinishLogin integration test** — only BeginLogin tested cross-module. FinishLogin needs mock token exchange endpoint.
- [ ] **usermgmt HTTP handler coverage** — oauth2_http.go, credential_http.go edge cases, Postgres setup all at 0% coverage. Need httptest.Server fixtures.
- [ ] **adminui coverage improvement** — 66.8%, only `seed_render_test.go` as end-to-end test. Target 70%+.

### P3 — Technical Debt & Future

- [ ] **Phase 2b — Persistent offline queue** (OPFS persistence per ADR-0030) — SharedWorker queue is in-memory only. Writes don't survive closed tabs.
- [ ] **Snapshot integration** for high-event-volume aggregates (>10K events/aggregate)
- [ ] **TypedRepository adoption** to eliminate command type assertions across all deciders
- [ ] **Redis adapters** for SessionStore/OAuth2StateStore/IdempotencyStore (multi-instance deployments)
- [ ] **MySQL support** for event store (currently Postgres + SQLite only)
- [ ] **Property-based tests** for event fold functions (rapid/Hypothesis-style)
- [ ] **Load testing benchmarks** for SSE broadcaster under high fan-out
- [ ] **OpenAPI spec generation** for HTTP endpoints
- [ ] **Consumer-facing v3→v4 codemod** — automated migration tool
- [ ] **Evaluate encoding/json/v2 adoption** (v4.1+ target when Go stabilizes the package)
- [ ] **Document provider implementation guide** — how to write custom TOTPProvider/WebAuthnProvider/OAuth2Provider
- [ ] **Admin UI: TOTP management views** (enable/disable, show QR code)
- [ ] **Admin UI: OAuth2 link/unlink views**
- [ ] **Configurable TTLs** — lockout TTL, OAuth2 state TTL, verification token TTL (same pattern as WebAuthnSessionTTL, TOTPPendingSecretTTL)
- [ ] **Benchmark dedup.Ring vs old map** for typical journal sizes (100, 1K, 10K, 100K events)
- [ ] **Add integration test** that imports the published version (not local replace)
- [ ] **Contract tests** between root module and usermgmt (RateLimiter boundary)
- [ ] **Standardize import grouping** across the codebase (some files separate go-error-family/go-cqrs-lite imports, others group them)
- [ ] **Automate GitHub Release creation via CI on tag push** (`.github/workflows/release.yml`)

---

## Completed

### v4.2.1 (2026-07-08 → 2026-07-09)

- [x] **encoding/json/v2 revert** — 26 source files reverted from experimental `encoding/json/v2` + `jsontext` after buildflow auto-migration broke the build
- [x] **Post-release doc cleanup** — 16 doc fixes: stale version refs, removed API references (MemoryBus, WebAuthnConfig), migration directory consolidation, CHANGELOGs created for all 6 modules, README/CONTRIBUTING/DOMAIN_LANGUAGE audited
- [x] **Dependency upgrade** — all go-cqrs-lite modules bumped to v3.7.4. go-error-family unified to v0.6.1. httputil v0.5.0. templ-components v0.10.0
- [x] **go.work conflict fix** — removed `replace` for eventtest from go.work (can't be in both `use` AND `replace`)
- [x] **Releases tagged + pushed** — v4.2.1, usermgmt/v4.2.0, adminui/v4.2.0, totp/v4.0.2, webauthn/v4.0.2, oauth2/v4.0.2

### v4.2.0 (2026-07-04 → 2026-07-07)

- [x] **go-cqrs-lite v3.7.0 feature adoption** — `dedup.Ring` (O(1) memory vs unbounded map), `codec.ForEncoding` (CBOR compatibility)
- [x] **go-error-family direct dep** in auth strategy modules (32 violations → 0, was indirect via event/v3)
- [x] **Error context enrichment** — `.WithContext()` chaining in OAuth2/TOTP/SQL stores. `classifyDispatchError` gains variadic `kv ...string`.
- [x] **Consumer feedback implementation** (9 new APIs): `DefaultRateLimiterConfig()`, `SSEEventConnected`/`SSEEventHeartbeat`, `SecurityHeaderSkip`, `RenderHTML`, `DecodeJSONWithRequest`, `RequestGuard`, `Broadcaster.Close()`/`fanOut.Close()`, JSON error `"code"` field, `CSRFTestToken`
- [x] **SKILL.md 10/10 rewrite** — cheat sheet, fixed GET query example (was crashing), SSE lifecycle, auth sentinel→HTTP status mapping
- [x] **GET decoder bug fix** — `decodeJSONBody` now returns zero-value T on empty body instead of `json.Unmarshal` error

### v4.0.0 → v4.1.1 (2026-07-02)

- [x] **Auth strategy extraction** — TOTP, WebAuthn, OAuth2 extracted behind primitive-type interfaces as independent Go modules. Core usermgmt has ZERO auth deps. 38 provider tests (W3C spec vectors + real JWT signing). ADR-0035.
- [x] **Module path bump /v3 → /v4** — all 130+ files across 11 modules
- [x] **Migration guide** (`docs/migrations/v3-to-v4.md`) — 4 sections with before/after examples
- [x] **Pseudo-version fix** — `v0.0.0-...` → `v4.0.0-...` in 5 go.mod files (GOWORK=off builds were broken)
- [x] **CI/build coverage** — all 3 sub-modules in isolation/budget/lint/test/coverage/fuzz
- [x] **Configurable WebAuthnSessionTTL** — `ServiceConfig.WebAuthnSessionTTL` (was hardcoded 5min)
- [x] **Fuzz tests on JSON boundary** — `marshalWebAuthnUser`, `parseUser`, `parseSession`
- [x] **WebAuthn cross-module integration test** — Service → JSON → Provider → go-webauthn chain
- [x] **Compile-time interface assertions** — `integration_test/auth_interface_assert_test.go`

### v3.5.0 (2026-07-01 → 2026-07-02)

- [x] **Error model overhaul** — HTTPStatusCarrier (`WithHTTPStatus`), ProblemDetailsErrorHandler (RFC 7807), StructuredError enrichment, 5xx detail redaction (`SafeDetail`)
- [x] **Architecture enforcement** — 4 CI scripts (module-isolation, dep-budgets, version-drift, replace-directives), wired as `nix run .#check-modules`
- [x] **Sollbruchstellen analysis** — cycle debunked (5 string constants, not structural), 7 root module extraction candidates identified, 3 usermgmt seams mapped, 4 D2 diagrams
- [x] **constants.go** — shared constants extracted from response.go
- [x] **Auth strategy interfaces** — TOTPVerifier, WebAuthnProvider, OAuth2Provider in `auth_interfaces.go`
- [x] **Version drift elimination** — all go-cqrs-lite siblings pinned to consistent versions

### Earlier (v1.0.0 → v3.3.0)

- [x] App builder, command/query dispatch, handler options, HTMX middleware, Casbin authorization
- [x] CSRF (nosurf), rate limiting, security headers, recovery, request logging (text/JSON/slog)
- [x] SSE + WebSocket real-time: Broadcaster, reconnection, CQRS dispatch bridges, typed WS parser
- [x] Embedded HTMX v2.0.10 JS + extensions (SSE/WS/idiomorph)
- [x] Event-sourced usermgmt: 12 events, 11 commands, Decider pattern, WebAuthn passwordless
- [x] Identity model: Tenant, Bot, Membership, Impersonation, ActorID (ADR-0015)
- [x] SQL stores: event store, session store, read models (Postgres + SQLite)
- [x] Stack presets: one-call SQLite/Postgres setup
- [x] Event signing + encryption (opt-in seams, ADR-0011)
- [x] OAuth2/OIDC integration (ADR-0014)
- [x] Email verification, TOTP MFA, user import/export, audit log
- [x] Offline-first Phase 2a: SharedWorker in-memory command queue (ADR-0029)
- [x] ACK protocol + idempotency store (ADRs 0023/0024/0026)
- [x] Server-Timing API (W3C, ADR-0033)
- [x] Checkpoint-based projection replay (ADR-0031)
- [x] BasicCommand embedding (ADR-0032 — eliminates zero-cmdID bug)
- [x] Admin UI module (templ + HTMX, ready-made dashboard)
- [x] go-cqrs-lite v3 migration (ADR-0016)

---

_~175 items completed across all sessions. See [CHANGELOG.md](CHANGELOG.md) and [git log](https://github.com/larsartmann/cqrs-htmx/commits/master) for full history._
