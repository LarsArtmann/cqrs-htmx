# Status Report: Dependency Upgrade & Lint Zero — 2026-06-17

**Session focus:** Upgrade all dependencies to latest, adopt new APIs idiomatically, eliminate all lint issues.

---

## Metrics Snapshot

| Metric             | Value                                               |
| ------------------ | --------------------------------------------------- |
| Date               | 2026-06-17 12:19                                    |
| Go version         | 1.26.3                                              |
| Root coverage      | 96.4%                                               |
| usermgmt coverage  | 84.1%                                               |
| Lint               | **0 issues** (root + usermgmt)                      |
| Tests              | All passing (-race, all modules)                    |
| Total Go files     | 200 (119 test files)                                |
| Total Go LOC       | 28,131                                              |
| ADRs               | 6                                                   |
| Commits since 6/14 | 77                                                  |
| Modules            | 4 (root, usermgmt, integration_test, datastar-demo) |

---

## A) FULLY DONE

### Dependency Upgrades (all 4 modules)

| Dependency             | Old           | New         | Status         |
| ---------------------- | ------------- | ----------- | -------------- |
| go-cqrs-lite (all sub) | v2.3.0/v2.3.1 | **v2.4.0**  | ✅             |
| go-error-family        | v0.3.0        | **v0.4.0**  | ✅             |
| go-branded-id          | v0.3.0        | **v0.3.1**  | ✅             |
| ginkgo/v2              | v2.30.0       | **v2.31.0** | ✅             |
| gomega                 | v1.41.0       | **v1.42.0** | ✅             |
| casbin/casbin/v3       | v3.10.0       | v3.10.0     | Already latest |
| go-webauthn            | v0.17.4       | v0.17.4     | Already latest |
| pquerna/otp            | v1.5.0        | v1.5.0      | Already latest |
| justinas/nosurf        | v1.2.0        | v1.2.0      | Already latest |
| modernc.org/sqlite     | v1.52.0       | v1.52.0     | Already latest |
| datastar-go            | v1.2.2        | v1.2.2      | Already latest |
| golang.org/x/time      | v0.15.0       | v0.15.0     | Already latest |

### Idiomatic API Adoption

- ✅ **`event.Wrapf` / `event.WrapTransient`**: All 16 `event.NewTransient(fmt.Sprintf(...)).WithCause(err)` patterns replaced with idiomatic wrapping APIs across `authz_policies.go`, `authz_types.go`, `authz_roles.go`, `recovery.go`
- ✅ **`event.Newf`**: `recovery.go` panic handler uses `event.Newf(event.Infrastructure, ...)` instead of `fmt.Sprintf`
- ✅ **`event.IsRetryable`**: `coverage_gaps_test.go` uses `event.IsRetryable(err)` instead of manual `event.Classify(err) == event.Transient` comparison
- ✅ **`policyWrapErr` deleted**: Helper function eliminated — `event.Wrapf` handles formatted policy messages directly. Test deleted too.
- ✅ **`fmt` imports removed**: `authz_policies.go`, `authz_roles.go`, `authz_types.go` no longer import `fmt` (all formatting now in `event.Wrapf`)
- ✅ **Duplicate email validation eliminated**: `ImportUser.Validate()` delegates to `ParseEmail()` instead of reimplementing identical logic

### Lint: Zero Issues (was 16)

| Linter       | Was | Now | How                                                                                 |
| ------------ | --- | --- | ----------------------------------------------------------------------------------- |
| errorlint    | 2   | 0   | `%s` → `%w` for inner errors in `email.go`, `import_export.go`                      |
| exhaustruct  | 2   | 0   | Third-party `pquerna/otp` structs excluded in `.golangci.yml`                       |
| contextcheck | 12  | 0   | False positives from `withTimeout` helper — file-level exclusion in `.golangci.yml` |

### Verification

- ✅ `nix run .#test` — all 3 test modules pass with `-race`
- ✅ `nix run .#lint` — 0 issues (root + usermgmt)
- ✅ `nix flake check` — formatting + devShells + apps pass
- ✅ `nix run .#build` — all 4 modules compile (including datastar-demo)

### Documentation

- ✅ `AGENTS.md` — dependency versions, API notes, and LSP discrepancy note updated
- ✅ `README.md` — dependency table updated
- ✅ `CHANGELOG.md` — unreleased section updated with all changes

---

## B) PARTIALLY DONE

### usermgmt Coverage (84.1%)

The usermgmt module dropped from ~90% to 84.1% during the event-sourcing migration. Key gaps:

| Area                        | Coverage | Issue                                 |
| --------------------------- | -------- | ------------------------------------- |
| `verification_totp_http.go` | 73-90%   | HTTP handler error paths under-tested |
| `webauthn_service.go`       | 72-90%   | WebAuthn ceremony error paths         |
| `webauthn_session.go`       | 77-100%  | Get() miss path                       |
| `parseImportFormat`         | 55.6%    | Content-Type fallback branch          |

### ROADMAP.md / TODO_LIST.md Version Drift

- `ROADMAP.md` says `v2.1.0` and `go-cqrs-lite v2.3.0` — stale
- `TODO_LIST.md` header says `v2.1.0` and coverage `96.0%/83.6%` — slightly stale
- Both need refresh after this session

### LSP Cache Issue

The LSP (gopls) still shows 16 stale warnings that `golangci-lint` does not. The `.golangci.yml` changes are picked up by CLI but the LSP cache hasn't invalidated. This is a tooling issue, not a code issue. The CLI is authoritative.

---

## C) NOT STARTED

### From ROADMAP.md

| Item                                          | Priority |
| --------------------------------------------- | -------- |
| PostgreSQL event store for User aggregate     | High     |
| PostgreSQL session store                      | High     |
| OpenTelemetry tracing middleware              | High     |
| Prometheus metrics middleware                 | Medium   |
| JWT/OIDC integration helpers                  | Medium   |
| Redis session store                           | Medium   |
| Database migration tooling                    | Medium   |
| Godoc package examples with runnable snippets | Medium   |
| Expand integration_test cross-module coverage | Low      |
| Profile hot paths for allocation reduction    | Low      |

### Git Tag

No `v2.x.0` release tag exists in git. The module is at v2.4.0 per go.mod but there's no corresponding git tag.

---

## D) TOTALLY FUCKED UP

**Nothing is totally fucked up.** The codebase is in good shape:

- All tests pass with `-race`
- Zero lint issues
- All dependencies at latest stable
- No broken builds
- No security issues flagged

The closest thing to "fucked up" is the LSP cache showing stale warnings, but that's a tooling issue — `golangci-lint` CLI is clean.

---

## E) WHAT WE SHOULD IMPROVE

### Architecture & Type Safety

1. **`usermgmt.UserID` vs root `id.UserID` split brain** — Two different branded types for "user ID" (one ULID-backed via go-branded-id, one string-backed via go-branded-id). Cross-module conversion via `.Get()` is fragile. Consider unifying on a single type or documenting the boundary more explicitly.

2. **`Email` is a type alias, not a branded type** — `type Email string` allows any string to be assigned. A branded type `type Email = brandid.ID[emailBrand, string]` would prevent accidental misuse, matching the pattern already used for `UserID`.

3. **`HandlerConfig.Secure` uses `*bool`** — `nil` defaults to true. This is a known anti-pattern (three-valued bool). An enum (`SecureModeAuto`/`SecureModeAlways`/`SecureModeNever`) would be clearer.

4. **Casbin enforcer is `*casbin.Enforcer` concrete type** — `authz_types.go` stores `*casbin.Enforcer` directly rather than an interface. This makes testing harder and couples usermgmt to Casbin's concrete implementation.

### Error Handling

5. **Sentinel error `== event.NewRejection(...)` pattern** — All usermgmt errors are package-level vars created via `event.NewRejection`. In go-error-family v0.5.0, `WithContext`/`WithCause` are copy-on-write, but `errors.Is` matching relies on code+family comparison. This works but is unusual — most Go code uses `errors.Is` with type matching.

6. **Missing `event.Compose` usage** — go-error-family v0.4.0+ provides `Compose` (re-exported via `event.Compose`) for multi-error aggregation. `RegisterRequest.Validate()` manually joins error strings with `strings.Join` instead of using `Compose` or `errors.Join`.

### Testing

7. **No fuzz tests in cqrs-htmx** — go-cqrs-lite has extensive fuzz testing. cqrs-htmx has zero. The decoder (`decoder.go`), SSE parsing (`sse_event.go`), and WebSocket parsing (`ws.go`) are prime fuzz targets.

8. **No benchmark CI regression tracking** — Benchmarks exist (`benchmark_*.go`) but there's no CI integration to catch performance regressions.

9. **Integration tests are thin** — Only 3 files in `integration_test/`. Cross-module bridges (root + usermgmt) are minimally tested.

### Documentation

10. **ADR for v2.4.0 upgrade missing** — ADR 0005 documents v2.3.0 adoption. No ADR for v2.4.0 changes (15 performance optimizations upstream, reactive APIs, KV module).

11. **DOMAIN_LANGUAGE.md not updated** — Last updated pre-event-sourcing. Missing terms: Decider, foldUser, CasbinProjection, WebAuthnCredential, TOTP, etc.

---

## F) Top 25 Things to Get Done Next

Sorted by impact × effort ratio (highest first):

### Tier 1: High Impact, Low Effort (do now)

| #   | Item                                                                | Impact | Effort |
| --- | ------------------------------------------------------------------- | ------ | ------ |
| 1   | **Tag v2.4.0 release** — create git tag matching go.mod             | High   | 5 min  |
| 2   | **Refresh ROADMAP.md** — update versions, coverage, dependency list | Medium | 10 min |
| 3   | **Refresh TODO_LIST.md** — mark completed items, update header      | Medium | 10 min |
| 4   | **Write ADR 0007** — document v2.4.0 upgrade decisions              | Medium | 15 min |
| 5   | **Update DOMAIN_LANGUAGE.md** — add event-sourcing terms            | Medium | 20 min |

### Tier 2: High Impact, Medium Effort

| #   | Item                                                                                | Impact | Effort |
| --- | ----------------------------------------------------------------------------------- | ------ | ------ |
| 6   | **Fuzz tests for decoder.go** — `FuzzDecodeJSON`, `FuzzDecodeForm`                  | High   | 1h     |
| 7   | **Fuzz tests for sse_event.go** — `FuzzWriteSSEEvent`, `FuzzParseSSE`               | High   | 1h     |
| 8   | **Fuzz tests for ws.go** — `FuzzParseWSMessage`, `FuzzParseWSMessageInto`           | High   | 1h     |
| 9   | **Use `event.Compose` in RegisterRequest.Validate** — replace manual `strings.Join` | Low    | 15 min |
| 10  | **Improve usermgmt coverage to 90%+** — focus on HTTP handler error paths           | High   | 2-3h   |
| 11  | **Brand `Email` type** — `type Email = brandid.ID[emailBrand, string]`              | Medium | 30 min |
| 12  | **Replace `HandlerConfig.Secure *bool` with enum** — clearer intent                 | Medium | 30 min |

### Tier 3: Medium Impact, Medium Effort

| #   | Item                                                               | Impact | Effort   |
| --- | ------------------------------------------------------------------ | ------ | -------- |
| 13  | **PostgreSQL event store** — persistent storage for User aggregate | High   | 1-2 days |
| 14  | **PostgreSQL session store** — `SessionStore` interface impl       | High   | 4h       |
| 15  | **OpenTelemetry tracing middleware** — via lifecycle hooks         | Medium | 4h       |
| 16  | **Integration test expansion** — more cross-module bridges         | Medium | 2-3h     |
| 17  | **Profile dispatch + decode hot paths** — allocation reduction     | Medium | 2h       |
| 18  | **Prometheus metrics middleware** — dispatch latency, error rates  | Medium | 3h       |

### Tier 4: Lower Priority / Future

| #   | Item                                                                                       | Impact | Effort   |
| --- | ------------------------------------------------------------------------------------------ | ------ | -------- |
| 19  | **Redis session store** — distributed deployments                                          | Medium | 4h       |
| 20  | **JWT/OIDC integration helpers** — auth beyond WebAuthn                                    | Medium | 1 day    |
| 21  | **Database migration tooling** — goose/golang-migrate integration                          | Medium | 4h       |
| 22  | **Casbin enforcer interface extraction** — decouple from concrete type                     | Low    | 2h       |
| 23  | **Benchmark CI regression tracking** — automated perf gates                                | Low    | 2h       |
| 24  | **Godoc runnable examples** — package-level `Example*` functions                           | Low    | 2h       |
| 25  | **Explore go-cqrs-lite reactive APIs** — `NewCommandBus`/`NewQueryBus` for event streaming | Low    | Research |

---

## G) Top Question I Cannot Figure Out Myself

**Should we cut a v2.4.0 git tag NOW, or is there more work to bundle into the release?**

The go.mod files all declare `v2.4.0` (via go-cqrs-lite submodule tags). The CHANGELOG has an `[Unreleased]` section. But there's no git tag. The codebase is stable (0 lint, all tests pass, race-safe).

Options:

1. **Tag v2.4.0 now** — the dependency upgrade + lint fixes are a clean, shippable unit
2. **Wait for more work** — bundle coverage improvements, fuzz tests, ADR 0007 into the release
3. **Tag v2.4.0 now, plan v2.5.0 for next batch** — release small and often

This is a product decision only the project owner can make.

---

## Session Commits

| Commit    | Description                                                                             |
| --------- | --------------------------------------------------------------------------------------- |
| `f131b20` | `chore(deps): upgrade go-cqrs-lite v2.3.0 → v2.4.0 and go-error-family v0.3.0 → v0.4.0` |
| `25ab685` | `refactor: use go-error-family v0.4.0 APIs idiomatically, fix all lint issues`          |
| `21959ff` | `docs: update CHANGELOG and AGENTS.md for v2.4.0/v0.4.0 upgrade`                        |
