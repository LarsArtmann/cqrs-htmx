# Status Report — cqrs-htmx

**Date:** 2026-05-21 00:24 CEST
**Session:** Dependency upgrade to latest go-cqrs-lite + comprehensive project status

---

## Executive Summary

The project is in **excellent health**. All dependencies are now on latest versions, both modules build clean, all tests pass, coverage is strong (95.9% root / 92.1% usermgmt), and zero lint issues remain. This session focused on upgrading `go-cqrs-lite/core` v1.2.0 → **v1.4.0** and `go-branded-id` v0.1.0 → **v0.3.0** with thorough breakage analysis. No breaking changes were found — the upgrade was zero-cost.

---

## a) FULLY DONE

### This Session (2026-05-21)

| #   | Item                                  | Details                                                                                                                                                                                                                                                                                          |
| --- | ------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| 1   | **go-cqrs-lite/core v1.2.0 → v1.4.0** | Root module upgraded. Two major versions of new features now available: `CatalogDispatcher` mixin, `TypedHandler[T]`/`RegisterTyped[T]`, `NewEvents`/`DecodePayloads` batch helpers, `Publisher`/`Subscriber` ISP interfaces, `SnapshotStrategy` extraction, `WalkMessages`, and many bug fixes. |
| 2   | **go-branded-id v0.1.0 → v0.3.0**     | Both root and usermgmt modules upgraded. v0.3.0 adds `BrandNamer` interface (brand-aware `String()`), `ValidateID`, `GoString` improvements. No breakage — `userBrand` uses `.Get()` at Casbin boundaries, not `.String()`.                                                                      |
| 3   | **Breaking change analysis**          | Thoroughly audited all `.String()` usage across both modules. Confirmed zero impact: go-cqrs-lite marker types don't implement `BrandNamer`, and usermgmt uses `.Get()` at all Casbin boundaries.                                                                                                |
| 4   | **AGENTS.md updated**                 | Dependency table and key decisions updated with version references and v1.4.0 feature notes.                                                                                                                                                                                                     |

### Project-Wide (Cumulative)

| Category                   | Status                                                                                                       |
| -------------------------- | ------------------------------------------------------------------------------------------------------------ |
| **App Builder**            | FULLY_FUNCTIONAL — `New(Config)` with validation, lifecycle hooks, timeout, per-App LoginRedirect            |
| **Command/Query Dispatch** | FULLY_FUNCTIONAL — Generic decoders (JSON, Form, Query variants), HTMX-aware response builder                |
| **Authorization**          | FULLY_FUNCTIONAL — Casbin v3 Enforcer interface, `Authorize`, `Enforce`, `AuthorizeMiddleware`               |
| **User Identity**          | FULLY_FUNCTIONAL — Strongly-typed `UserID`, `CorrelationID`, `RequestID` (all ULID-backed branded types)     |
| **HTMX Integration**       | FULLY_FUNCTIONAL — Request context, response builder, notifications, swap strategies, all header constants   |
| **Error Classification**   | FULLY_FUNCTIONAL — `sync.Once` registration, family → HTTP status mapping, custom error handlers             |
| **CSRF Protection**        | FULLY_FUNCTIONAL — gorilla/csrf v1.7.3, double-submit cookie, per-handler opt-in, plaintext HTTP auto-detect |
| **Security Headers**       | FULLY_FUNCTIONAL — Configurable CSP, HSTS, X-Frame-Options, Referrer-Policy, Permissions-Policy              |
| **Rate Limiting**          | FULLY_FUNCTIONAL — Token-bucket per key, MaxKeys cap, Retry-After header                                     |
| **Request Logging**        | FULLY_FUNCTIONAL — Default + JSON formatters, correlation/user/request ID capture                            |
| **usermgmt Submodule**     | FULLY_FUNCTIONAL — Branded UserID, RBAC, sessions, password auth, account lockout, HTTP handlers             |
| **Test Coverage**          | Root: 95.9% (289+ specs), usermgmt: 92.1%                                                                    |
| **Lint**                   | 0 issues (golangci-lint clean)                                                                               |
| **Benchmarks**             | 16 sub-benchmarks                                                                                            |
| **Godoc**                  | 9 Example functions + ~70 usermgmt symbols documented                                                        |

---

## b) PARTIALLY DONE

| #   | Item                                      | What's Done                                         | What's Missing                                                                                                                                                                                        |
| --- | ----------------------------------------- | --------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | **go-cqrs-lite v1.4.0 feature adoption**  | Dependency upgraded, builds and tests pass          | Haven't adopted new v1.4.0 APIs in consumer code: `CatalogDispatcher` catalog introspection, `TypedHandler[T]`/`RegisterTyped[T]`, `Publisher`/`Subscriber` ISP split, `DecodePayloads` batch helpers |
| 2   | **go-branded-id v0.3.0 feature adoption** | Dependency upgraded                                 | Haven't adopted `ValidateID`, `BrandNamer` (root module markers could get names), `GoString` improvements                                                                                             |
| 3   | **usermgmt → cqrs-htmx type bridge**      | `UserIDFromRequest()` returns `string` via `.Get()` | Type split remains: `usermgmt.UserID` (string-backed) vs `cqrshtmx.UserID` (ULID-backed) are incompatible types                                                                                       |

---

## c) NOT STARTED

| #   | Item                                           | Priority | Notes                                                                                                                                    |
| --- | ---------------------------------------------- | -------- | ---------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | **Adopt `CatalogDispatcher` mixin**            | Medium   | go-cqrs-lite v1.4.0 exposes `CatalogEntries()` on both dispatchers — could expose catalog metadata through cqrs-htmx's App for auto-docs |
| 2   | **Adopt `Publisher`/`Subscriber` ISP**         | Low      | v1.4.0 splits `event.Bus` into sub-interfaces. No impact yet since cqrs-htmx doesn't interact with event bus directly                    |
| 3   | **Adopt `TypedHandler[T]` for query dispatch** | Medium   | v1.4.0 adds `query.RegisterTyped[T]` and `query.DispatchTyped[T]` — could make query dispatch type-safe instead of `any`                 |
| 4   | **Adopt `DecodePayloads` batch helper**        | Low      | Only relevant if cqrs-htmx ever handles event replay/payload decoding                                                                    |
| 5   | **Adopt `BrandNamer` for root module markers** | Low      | `userMarker`, `correlationMarker`, `requestMarker` could get `Name()` methods for debug-visible `String()` output                        |
| 6   | **Adopt `ValidateID` from go-branded-id**      | Low      | Could replace manual `parseID` generic with `brandid.ValidateID`                                                                         |
| 7   | **Integration tests between root + usermgmt**  | High     | Full register → cqrs dispatch with user context flow — the TODO list's only remaining high-priority test gap                             |
| 8   | **Resolve UserID type split**                  | Medium   | Decide: merge `usermgmt.UserID` into `cqrshtmx.UserID`, or formalize the bridge pattern                                                  |
| 9   | **Rate limiter eviction O(n) → O(log n)**      | Low      | `evictOldestIfAtCapacity()` does linear scan; min-heap or LRU would be better for large key spaces                                       |
| 10  | **CI/CD pipeline**                             | Medium   | `.github/workflows/ci.yml` exists but hasn't been verified with the dependency upgrades                                                  |

---

## d) TOTALLY FUCKED UP

**Nothing.** Zero issues. The project is clean:

- Build: **PASS** (both modules)
- Tests: **PASS** (root + usermgmt, all specs green)
- Lint: **0 issues**
- Coverage: **95.9% / 92.1%**
- Dependencies: **All on latest versions**

The only known wart is a stale LSP warning (SA1012 in `usermgmt/coverage_test.go:206`) — a pre-existing issue in a test file, not production code.

---

## e) WHAT WE SHOULD IMPROVE

### High Impact

1. **Integration tests between root module and usermgmt** — The two modules have never been tested together end-to-end. A `register → login → cqrs dispatch with user context` flow test would catch bridge bugs that unit tests miss.
2. **Adopt `TypedHandler[T]` for query results** — `query.Dispatch` returns `(any, error)` which loses type safety. `DispatchTyped[T]` from v1.4.0 could make this compile-time safe.
3. **Expose `CatalogEntries()` from App** — v1.4.0's `CatalogDispatcher` mixin provides catalog metadata. cqrs-htmx's `App` wraps both dispatchers — exposing this would enable auto-generated API docs.

### Medium Impact

4. **Resolve the UserID type split** — Two incompatible `UserID` types (`string`-backed vs `ULID`-backed) create friction at the boundary. Either unify them or formalize the conversion pattern.
5. **usermgmt coverage gaps** — `policyWrapErr` (0%), `generateToken` (75%), `EnforceEx` (75%) are the weakest spots. Pushing to 95%+ would strengthen the safety net.
6. **Root module coverage gaps** — `Hijack` (0%), `sameSite` (66.7%), `fieldName` (66.7%), `csrfTokenFromRequest` (66.7%) in CSRF-related code.
7. **CI pipeline validation** — Ensure GitHub Actions CI passes with the new dependency versions.

### Low Impact

8. **Adopt `BrandNamer` for all marker types** — `userMarker`, `correlationMarker`, `requestMarker` could get `Name()` methods for better debug output when printing IDs.
9. **Adopt `ValidateID` from go-branded-id** — Replace manual `parseID` generic helper.
10. **Rate limiter eviction** — O(n) → O(log n) for large key spaces.

---

## f) Top 25 Things We Should Get Done Next

| #   | Priority | Item                                                                | Effort | Impact |
| --- | -------- | ------------------------------------------------------------------- | ------ | ------ |
| 1   | P0       | **Integration test: root + usermgmt E2E flow**                      | M      | High   |
| 2   | P1       | **Adopt `TypedHandler[T]`/`DispatchTyped[T]` for query dispatch**   | S      | High   |
| 3   | P1       | **Expose `CatalogEntries()` on App for API doc introspection**      | S      | Medium |
| 4   | P1       | **Resolve `usermgmt.UserID` vs `cqrshtmx.UserID` type split**       | M      | Medium |
| 5   | P1       | **Validate CI pipeline with new deps**                              | S      | Medium |
| 6   | P2       | **Adopt `Publisher`/`Subscriber` ISP in any event bus interaction** | S      | Medium |
| 7   | P2       | **Add `BrandNamer` to root module marker types**                    | S      | Low    |
| 8   | P2       | **Usermgmt: test `policyWrapErr` (0% coverage)**                    | S      | Medium |
| 9   | P2       | **Usermgmt: test `generateToken` error path (75%)**                 | S      | Low    |
| 10  | P2       | **Root: test `Hijack()` method (0% coverage)**                      | S      | Low    |
| 11  | P2       | **Root: test `sameSite` CSRF helper (66.7%)**                       | S      | Low    |
| 12  | P2       | **Root: test `fieldName` CSRF helper (66.7%)**                      | S      | Low    |
| 13  | P2       | **Root: test `csrfTokenFromRequest` (66.7%)**                       | S      | Low    |
| 14  | P3       | **Adopt `ValidateID` from go-branded-id**                           | S      | Low    |
| 15  | P3       | **Rate limiter eviction O(n) → min-heap**                           | M      | Low    |
| 16  | P3       | **Root: test `Push` logging helper (66.7%)**                        | S      | Low    |
| 17  | P3       | **Root: test `sanitizeRedirectURL` (75%)**                          | S      | Low    |
| 18  | P3       | **Usermgmt: test `RolesForUser` error path (75%)**                  | S      | Low    |
| 19  | P3       | **Usermgmt: test `ImplicitRolesForUser` error path (75%)**          | S      | Low    |
| 20  | P3       | **Usermgmt: test `handleLogout` (77.8%)**                           | S      | Low    |
| 21  | P3       | **Usermgmt: test `handleMe` (80%)**                                 | S      | Low    |
| 22  | P3       | **Usermgmt: test `handleLogin` (80%)**                              | S      | Low    |
| 23  | P3       | **Usermgmt: test `handleRegister` (87.5%)**                         | S      | Low    |
| 24  | P3       | **Evaluate `brandid.ID[Brand, int64]` for numeric IDs**             | S      | Low    |
| 25  | P3       | **Fuzz tests for CSRF token validation**                            | M      | Low    |

---

## g) Top #1 Question I Cannot Figure Out Myself

**Should `usermgmt.UserID` be unified with `cqrshtmx.UserID`?**

The current split:

- `usermgmt.UserID` = `brandid.ID[userBrand, string]` (string-backed, arbitrary values like `"lock_user1_user2"`)
- `cqrshtmx.UserID` = `id.Of[userMarker]` = `brandid.ID[userMarker, ulid.ULID]` (ULID-backed, strict format)

These are fundamentally different types — one accepts any string, the other only valid ULIDs. Unifying them would force usermgmt to require ULIDs for user IDs, which may break consumers using non-ULID identifiers (e.g., UUIDs, integer sequences, composite keys). But keeping them split means two incompatible `UserID` types in the same ecosystem with a bridge function (`UserIDFromRequest`) converting between them.

**Question for you:** Is usermgmt intended to be a standalone module (accepting any string ID) or always paired with cqrs-htmx (where ULID-backed IDs are the norm)? The answer determines whether we unify or formalize the bridge.

---

## Dependency Versions (Current)

| Dependency            | Version    | Status  |
| --------------------- | ---------- | ------- |
| `go-cqrs-lite/core`   | **v1.4.0** | Latest  |
| `go-branded-id`       | **v0.3.0** | Latest  |
| `go-error-family`     | **v0.1.1** | Latest  |
| `casbin/casbin/v3`    | v3.10.0    | Current |
| `cockroachdb/errors`  | v1.13.0    | Current |
| `gorilla/csrf`        | v1.7.3     | Current |
| `golang.org/x/time`   | v0.15.0    | Current |
| `golang.org/x/crypto` | v0.51.0    | Current |
| `onsi/ginkgo/v2`      | v2.29.0    | Current |
| `onsi/gomega`         | v1.41.0    | Current |

## Metrics

| Metric             | Root   | usermgmt |
| ------------------ | ------ | -------- |
| Coverage           | 95.9%  | 92.1%    |
| Test files         | 20     | 7        |
| Production files   | 17     | 9        |
| Total lines (prod) | ~3,300 | ~1,800   |
| Benchmarks         | 16     | 0        |
| Lint issues        | 0      | 0        |

## Files Changed This Session

| File                                | Change                                                               |
| ----------------------------------- | -------------------------------------------------------------------- |
| `go.mod`                            | `go-cqrs-lite/core` v1.2.0 → v1.4.0, `go-branded-id` v0.1.0 → v0.3.0 |
| `go.sum`                            | Updated checksums                                                    |
| `usermgmt/go.mod`                   | `go-branded-id` v0.1.0 → v0.3.0 (promoted to direct dep)             |
| `usermgmt/go.sum`                   | Updated checksums                                                    |
| `AGENTS.md`                         | Version references + v1.4.0 feature notes                            |
| `docs/status/2026-05-20_13-40_*.md` | Formatting cleanup (prior session)                                   |
