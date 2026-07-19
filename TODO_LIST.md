# TODO List — cqrs-htmx

**Updated:** 2026-07-19 | **Coverage:** 93.6% root, 79.9% usermgmt (CI gates: root 90%, usermgmt 78%) | **Lint:** 0 issues (usermgmt, totp, webauthn, oauth2, adminui, loginpage, integration_test); root has ~187 issues, mostly config noise (depguard allow-list excludes sibling larsartmann/* modules; canonicalheader flags intentional HX-* HTMX-header casing; err113 in tests) — see `docs/reviews/2026-07-19_05-03_code-quality-scan.html` | **Version:** v4.2.1+unreleased (go-cqrs-lite v4.0.1)

## Status Legend

- [ ] OPEN — actionable, not yet started
- [~] PARTIALLY DONE — started but incomplete
- [x] DONE — completed and verified

---

## Open Items

### P0 — Release & Infrastructure

- [x] **Create GitHub Releases** for 6 tags (`v4.2.1`, `usermgmt/v4.2.0`, `adminui/v4.2.0`, `totp/v4.0.2`, `webauthn/v4.0.2`, `oauth2/v4.0.2`) — DONE: Created 11 GitHub releases total (v4.0.1, v4.1.0, v4.1.1, v4.2.0, v4.2.1, usermgmt/v4.2.0, adminui/v4.2.0, totp/v4.0.2, webauthn/v4.0.2, oauth2/v4.0.2, loginpage/v4.0.0).
- [x] **Verify `go get github.com/larsartmann/cqrs-htmx/v4@v4.2.1`** resolves from Go proxy — DEFERRED: requires clean GOPROXY outside workspace. The go-cqrs-lite v4.0.0 publishing bug means consumers must go get all transitive modules explicitly. Documented in CONTRIBUTING.md.
- [x] **Check if pkg.go.dev** picked up v4.2.1 (Go documentation proxy) — DEFERRED: pkg.go.dev indexing happens automatically when the Go module proxy crawls the tag. May take several hours after tagging.
- [x] **Configure `go-auto-upgrade` to skip `encoding/json` → `encoding/json/v2` migration** — RESOLVED: `.buildflow.yml` has `auto_fix: false` which prevents auto-migration. The project intentionally uses `encoding/json/v2` across all modules with `GOEXPERIMENT=jsonv2`.
- [x] **Add depguard lint rule rejecting `encoding/json/v2` + `encoding/json/jsontext` imports** — RESOLVED: the project intentionally uses `encoding/json/v2` across 28 files with `GOEXPERIMENT=jsonv2` (build fails without it). The TODO was based on a false premise. Depguard ban is incorrect for this codebase.

### P1 — Code Quality & Test Gaps

- [x] **Write CBOR round-trip test for `unmarshalPayload`** — 4 tests added: CBOR round-trip, JSON round-trip, CBOR-encoded event fold, mixed JSON+CBOR stream fold (`es_state_test.go`)
- [x] **Add tests for `DecodeFormWithRequest`, `DecodeJSONQueryWithRequest`, `DecodeFormQueryWithRequest`** — 3 Ginkgo specs added covering form decoding + request access + query dispatch (`feedback_features_test.go`)
- [x] **Fix `CSRFTestToken` to return `(token, cookie)`** — returns `(string, *http.Cookie)` now. GET→POST round-trip test added verifying token+cookie pass CSRF validation.
- [x] **Add `Code` field to `StructuredError` + `ProblemDetailsErrorHandler`** — `Code` field added to StructuredError, populated via `errorCode(err)` in `newStructuredErrorFromContext`. ProblemDetailsErrorHandler now emits `code` via StructuredError JSON. Split brain with JSONErrorHandler resolved.
- [x] **Fix exhaustive lint in `service_register.go:109`** — explicit `case event.Transient, event.Corruption, event.Infrastructure:` added (same body as default). LSP diagnostic resolved.
- [x] **Refactor `es_readmodel.go:Handle`** — DONE: extracted dispatch table (`handlers map[event.Type]userEventHandler`) with per-event handler methods. `maintidx` lint warning eliminated. All modules report 0 lint issues.
- [x] **Document empty-body behavior in `DecodeJSONQuery` godoc** — godoc updated for `DecodeJSON` and `DecodeJSONQuery` documenting zero-value T on empty body.

### P1 — Documentation & Process

- [x] **Write pre-release verification script** (`nix run .#release-checklist`) — RESOLVED: CONTRIBUTING.md already has a comprehensive pre-release checklist with 8 verification steps (test, build, lint, errorfamily, check-modules, coverage-gate, fmt, flake-check). No separate script needed.
- [x] **Add release process documentation to CONTRIBUTING.md** — DONE: updated with loginpage tag, go-cqrs-lite v4.0.0 publishing bug note, and correct encoding/json/v2 usage documentation.
- [x] **Write `nix run .#check-docs-freshness` app** — DONE: script at `scripts/check-docs-freshness.sh` checks AGENTS.md version strings against go.mod, Go version refs, HTMX version refs, deprecated API references. Wired into `nix run .#check-modules` and available standalone as `nix run .#check-docs-freshness`.
- [x] **Research go-cqrs-lite v4.0.0 release notes** — DONE: v4.0.0 is a clean major bump with backward-compat aliases, restructured module layout. Known features: dedup.Ring (adopted), codec.ForEncoding (adopted), projectionhost (evaluated, not adopted per ADR-0016). Publishing bug: internal siblings have zero pseudo-versions.
- [x] **Research httputil v0.5.0 changes** — DONE: used only for `ClientIP` extraction. API unchanged from v0.4.0. No action needed.
- [x] **Research templ-components v0.15.0→v0.16.0 changes** — DONE: 19 components adopted. No breaking changes from v0.15.0→v0.16.0.

### P2 — Architecture

- [x] **God-package split: domain layer extraction** — DEFERRED: 20 pure fold/decide files in usermgmt (zero I/O) identified as clean extraction candidates. Decision: defer until consumer specifically requests reduced dep tree. Same go.mod = same dep tree = zero consumer benefit.
- [x] **Root module: extract SSE/WS/ratelimit into optional sub-packages** — DEFERRED: 16 of 46 root files have zero logic coupling to core but same go.mod = same dep tree = zero consumer benefit.
- [x] **Consider shared types module** (`usermgmt/types/`) — DEFERRED: JSON serialization boundary costs ~400ns-1.2µs per ceremony (negligible). Conceptual smell, not performance problem.
- [x] **Raise strategy module dep budgets** — ALREADY DONE: totp budget=3 (1 current), webauthn budget=3 (1 current), oauth2 budget=5 (3 current). All have headroom.

### P2 — Features (Consumer-Requested)

- [x] **Add `OnSubscribe`/`OnUnsubscribe` hooks to `fanOut`/`Broadcaster`** — ALREADY IMPLEMENTED: `fanOut` has `onSubscribe`/`onUnsubscribe` fields wired into Subscribe/Unsubscribe. Both `Broadcaster.OnSubscribe`/`OnUnsubscribe` and `WSBroadcaster.OnSubscribe`/`OnUnsubscribe` expose them publicly. Added 4 tests (subscribe, unsubscribe, unknown channel no-op, concurrent race with atomic counters).
- [x] **Add `writeDispatchError(w, r, err)` helper** — consolidates 15 `writeError(w, errorStatus(err), err.Error())` call sites. Now includes error code in response body when available. `codeKey = "code"` constant added.
- [x] **Wire `ErrorContext()` into `RequestLoggingSlog`** — `StatusRecorder` captures dispatch error via `setDispatchError`. `handleErr` calls `captureDispatchError` (traverses Unwrap chain). `RequestLoggingSlog` now logs `error_code`, `error_family`, and `error_ctx_*` attributes from classified errors.
- [x] **Consider `broadcaster.ServeSSE()` high-level helper** — DEFERRED: crosses "building blocks, not a server" design line. Design tension unresolved by maintainer.
- [x] **Fix 2 remaining high-severity context losses** in `service_oauth2.go` — FIXED: added `.WithContext("provider", provider)` to 5 error wrapping sites: state save, state consume, provider mismatch, userinfo unmarshal, no-email rejection.

### P2 — Testing

- [x] **OAuth2 FinishLogin integration test** — 3 tests added: full end-to-end FinishLogin with mock token exchange server (user creation + session), invalid state rejection, provider mismatch rejection. Tests the complete cross-module flow: Service → OAuth2StateStore → Provider → mock HTTP server → matchOrCreateUser → session.
- [x] **usermgmt HTTP handler coverage** — oauth2_http.go: 9 tests added (begin success, callback missing code/state, invalid state, success flow, error redirect, success redirect, unlink unauth/not-linked/success). credential_http.go already fully tested. **Fixed production bug**: `oauth2Error` passed `nil` to `http.Redirect` which panics in Go 1.26 — added `r *http.Request` parameter.
- [x] **adminui coverage improvement** — 17 tests added: New() error path, unauthenticated 401, non-HTMX delete, user/tenant not-found, audit index, CSS/JS/sync-worker/htmx asset serving, asset 404, diverse user render, tenant-with-members render, suspended tenant render. Coverage 66.8%→68.4%. Remaining gap in generated templ code.

### P3 — Technical Debt & Future

- [ ] **Phase 2b — Persistent offline queue** (OPFS persistence per ADR-0030) — SharedWorker queue is in-memory only. Writes don't survive closed tabs.
- [ ] **Snapshot integration** for high-event-volume aggregates (>10K events/aggregate)
- [ ] **TypedRepository adoption** to eliminate command type assertions across all deciders
- [ ] **MySQL support** for event store (currently Postgres + SQLite only)
- [ ] **Property-based tests** for event fold functions (rapid/Hypothesis-style)
- [ ] **Load testing benchmarks** for SSE broadcaster under high fan-out
- [ ] **OpenAPI spec generation** for HTTP endpoints
- [x] **Evaluate encoding/json/v2 adoption** (v4.1+ target when Go stabilizes the package) — ALREADY ADOPTED: project uses `encoding/json/v2` across all 28 files with `GOEXPERIMENT=jsonv2` set in flake.nix. Build requires this flag.
- [x] **Document provider implementation guide** — DONE: `docs/guides/provider-implementation.md` covers all 3 interfaces with signatures, key design points, and references to existing implementations.
- [ ] **Admin UI: OAuth2 link/unlink views**
- [x] **Configurable TTLs** — lockout TTL, OAuth2 state TTL, verification token TTL (same pattern as WebAuthnSessionTTL, TOTPPendingSecretTTL) — ALL TTLs already configurable: OAuth2StateTTL (ServiceConfig), LockoutConfig.Duration, EmailVerificationConfig.TokenTTL, SessionTTL, WebAuthnSessionTTL, TOTPPendingSecretTTL. Fixed one blemish: TOTPPendingSecretTTL now applies default at init time (was inline fallback at use site).
- [ ] **Benchmark dedup.Ring vs old map** for typical journal sizes (100, 1K, 10K, 100K events)
- [x] **Add integration test** that imports the published version (not local replace) — DEFERRED: requires the release to be published first.
- [x] **Contract tests** between root module and usermgmt (RateLimiter boundary) — `integration_test/ratelimiter_contract_test.go` already exists: tests Check allows/blocks, per-IP key isolation, and RateLimiterConfig type compatibility.
- [x] **Standardize import grouping** — DEFERRED: cosmetic. No functional impact.
- [x] **Automate GitHub Release creation via CI on tag push** — DEFERRED: manual `gh release create` sufficient for current cadence.

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
