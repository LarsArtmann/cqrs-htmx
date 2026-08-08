# Comprehensive Status Report — cqrs-htmx

**Date:** 2026-05-22 22:17 CEST
**Session:** Bug fixes, security hardening, lint cleanup, deep reflection

---

## TL;DR

| Metric            | Before Session     | After Session                                                                       |
| ----------------- | ------------------ | ----------------------------------------------------------------------------------- |
| Root coverage     | 96.6%              | 96.6%                                                                               |
| Usermgmt coverage | 91.2%              | 91.3%                                                                               |
| Root lint         | 0 issues           | 0 issues                                                                            |
| Usermgmt lint     | Not run separately | **165 issues** (tests: exhaustruct, paralleltest, goconst, noctx, gosec, wrapcheck) |
| Integration tests | 4 passing          | 4 passing                                                                           |
| Race detector     | Clean              | Clean                                                                               |
| Git status        | Clean              | 13 files modified, uncommitted                                                      |

---

## A) FULLY DONE

### 1. Bug Fixes (This Session)

- **Validate() trim bug**: `RegisterRequest.Validate()` and `LoginRequest.Validate()` used value receivers — `strings.TrimSpace` only modified a copy. Fixed to pointer receivers so trimmed values persist. 3 tests added proving email/display-name trimming works end-to-end.
- **HandlerConfig.Timeout not propagated**: `NewAuthHandler` never copied `cfg[0].Timeout` into the handler — timeouts from config were silently dropped. Fixed. Test added.
- **HandlerConfig.Secure zero-value caveat documented**: `Secure` defaults to `true` only when NO config is provided. Passing `HandlerConfig{}` without `Secure: true` results in `false`. Added explicit documentation on `HandlerConfig` type.

### 2. Security Hardening (This Session)

- **Max password length (128 chars)**: Added `maxPasswordLength = 128` to `RegisterRequest.Validate()` and `Service.ChangePassword()`. Prevents bcrypt CPU abuse. 2 tests added.
- **ErrUserIDExists sentinel**: Replaced untyped `errors.Newf(...)` in `store.Create` with typed sentinel `ErrUserIDExists`, mapped to HTTP 404 in `errorStatus`. Updated existing tests to use `errors.Is`. Added to `TestErrorStatus` table.

### 3. Code Quality (This Session)

- **WriteJSON returns error**: Root `WriteJSON` now returns error from `json.Encoder.Encode` (was silently swallowed). Properly wrapped with `fmt.Errorf` for wrapcheck compliance.
- **bdd_test.go writestring warning fixed**: Replaced inefficient string concatenation inside `strings.Builder.WriteString` with multiple `WriteString` calls.
- **coverage_test.go SA1012 fixed**: Changed `nil` context to `var nilCtx context.Context` to satisfy staticcheck.
- **Documentation improvements**: Added proxy trust warning to `ClientIP`, "not for production" warnings to `InMemoryUserStore`/`InMemorySessionStore`, unbounded growth warning to `AccountLockout`, mutation warning to `Service.Authz()`, zero-value caveat to `HandlerConfig`.
- **AGENTS.md updated**: 8 new gotchas (#48–#54) documenting all findings.

### 4. From Prior Sessions (Still Valid)

All previous work remains intact: CSRF protection, rate limiting, security headers, request logging, strong types (UserID, CorrelationID, RequestID), branded IDs in usermgmt, integration tests, datastar-demo.

---

## B) PARTIALLY DONE

### 1. Usermgmt Lint — 165 Issues

The root module has 0 lint issues. Usermgmt was never run through the full golangci-lint config because it has no `.golangci.yml`. The 165 issues break down as:

| Category     | Count | Severity                          |
| ------------ | ----- | --------------------------------- |
| paralleltest | 50    | Low (test quality)                |
| exhaustruct  | 47    | Low (test patterns)               |
| wrapcheck    | 20    | Medium (production code)          |
| goconst      | 15    | Low (test magic strings)          |
| noctx        | 10    | Medium (test quality)             |
| gosec        | 9     | Low (test-only cookie usage)      |
| errcheck     | 4     | Medium (production code)          |
| revive       | 3     | Medium (missing docs, unused ctx) |
| unparam      | 2     | Low                               |
| Others       | 5     | Low                               |

The **production code** issues that matter:

- `http.go:201` — `json.Encoder.Encode` return value not checked (errcheck)
- `authz.go` — 20 wrapcheck issues (unwrapped casbin errors)
- `service.go` — 5 wrapcheck issues (unwrapped interface method errors)
- `service.go:141,204` — unused `ctx context.Context` parameters (revive)
- `service.go:14` — gci formatting
- `http.go:82,101` — wrapcheck on json.Unmarshal
- `user.go:95,143` — wrapcheck on json.Marshal, rand.Read

The **test code** issues are mostly exhaustruct (partial structs in tests), paralleltest (missing `t.Parallel()`), goconst (magic test strings like `"secret12"`, `"a@b.com"`, `"session_token"`).

---

## C) NOT STARTED — From This Session's Reflection

These are identified but NOT yet implemented, sorted by **impact × effort**:

### HIGH IMPACT

1. **usermgmt `.golangci.yml`** — Create a usermgmt-specific config with appropriate exclusions for test patterns. The 165 issues need categorization: which are real production issues vs test-only noise.

2. **usermgmt `writeJSON` errcheck** — `http.go:201` silently discards `json.Encoder.Encode` error. Same pattern fixed in root module this session.

3. **usermgmt context.Context unused** — `Register`, `Login`, `Authenticate`, `Logout`, `UpdateRoles`, `ChangePassword`, `GetUser` all take `context.Context` but ignore it (`_` or unused). No cancellation, timeout, or tracing propagation.

4. **usermgmt `authz.go` wrapcheck** — 20 unwrapped casbin error returns. These should be wrapped with `fmt.Errorf("%w", ...)` or `errors.Wrapf` for consistent error chains.

5. **usermgmt `Session.Valid` double-checks expiration** — `Authenticate()` checks `IsExpired()` first, then `Valid()` checks again. If the clock advances between checks, `Valid()` returns `false` (not expired message) for what should be an expired session. Semantic inconsistency.

6. **usermgmt `SessionStore.EvictExpired()`** — No expired session cleanup mechanism exists. Sessions grow unbounded in the in-memory store.

7. **usermgmt `Authz.Apply` non-atomic** — Sequential removal/addition of policies. If a mid-operation failure occurs, policy state is partially updated. Doc says "atomically" but it isn't.

8. **usermgmt `Register` partial-state leak** — If `AddGroupPolicy` or `sessions.Create` fails after `users.Create` succeeds, the user is persisted but incomplete. No compensating transaction.

9. **usermgmt timeout only wraps `process`, not body read** — `handleAuthEndpoint` creates timeout context AFTER reading the request body. Slow clients bypass the timeout during body upload.

### MEDIUM IMPACT

10. **usermgmt `contextKey` uses `string` not `struct{}`** — `middleware.go:9` uses `type contextKey string`. Root package correctly uses `struct{}`. Inconsistent pattern.

11. **usermgmt `UserStore.Save` O(n) email index** — Iterates ALL email entries to find old email. Should use reverse mapping.

12. **usermgmt `NewUser` defaults to `RoleViewer` but `Register` adds `RoleUser`** — User gets `[viewer, user]` but the viewer role is never explicitly granted. Confusing.

13. **usermgmt `AccountLockout` unbounded maps** — Unlike root's rate limiter with TTL eviction and MaxKeys, lockout has no eviction for non-locked entries. Unbounded growth.

14. **root `enrichUserID` silently swallows extractor errors** — Misconfigured extractors result in silent unauthenticated requests.

15. **root `Response.Redirect` doesn't sanitize for HTMX** — `HX-Redirect` header is set without `sanitizeRedirectURL` check (only non-HTMX path is sanitized).

16. **root `decodeFormValues` round-trips through JSON** — `decoder.go` marshals form values to JSON then unmarshals. Consider `gorilla/schema` for proper form decoding with better error messages.

17. **root `CommandCatalogEntries`/`QueryCatalogEntries` wrap deprecated API** — `CatalogMeta` is deprecated in go-cqrs-lite v1.4.0. Should migrate to `catalog` API or mark these methods deprecated too.

18. **root LSP stale warnings** — `httputil_test.go:20,33` show errcheck warnings for `WriteJSON` even though the tests now use `Expect(...).To(Succeed())`. This is an LSP cache issue, not a real problem.

### LOW IMPACT

19. **usermgmt test constants** — `"secret12"` (16 uses), `"a@b.com"` (12 uses), `"session_token"` (11 uses) should be package-level test constants.

20. **usermgmt test `t.Parallel()`** — 50 test functions missing `t.Parallel()`.

21. **usermgmt gosec test cookie warnings** — 9 test-only `http.Cookie` without Secure/HttpOnly flags (safe in tests, but noisy).

22. **usermgmt `forcetypeassert` in service_test.go** — `svc.sessions.(*InMemorySessionStore)` should check the assertion.

23. **root `JSONLogFormatter` uses `http.StatusText` not numeric code** — Makes log parsing harder. The slog version correctly uses `slog.Int`.

24. **root `CSRFConfig.secret()` pads short secrets with zeros** — Significantly weakens HMAC for secrets < 32 bytes. `Validate()` only warns about empty secrets.

25. **root `StatusRecorder.Push` wraps error** — Changes error chain, breaks `errors.Is()` matching.

---

## D) TOTALLY FUCKED UP — Nothing!

All tests pass, all modules build, race detector clean, root lint is 0.

---

## E) WHAT WE SHOULD IMPROVE

### Architecture

1. **usermgmt needs its own `.golangci.yml`** — Currently inherits root config which is too strict for test patterns. Need per-module exclusions.
2. **Context.Context is a lie** — Every service method takes `ctx` but ignores it. This is worse than not having the parameter — it gives false confidence.
3. **Error wrapping inconsistency in usermgmt** — `authz.go` has 20 unwrapped casbin errors. Root package has 0. Inconsistent within the project.

### Type Model

4. **`contextKey` inconsistency** — Root uses `struct{}`, usermgmt uses `string`. Should be consistent.
5. **`Role` type could be a typed string enum** — Currently `type Role string` with loose constants. Could use `go-sumtype` or similar for exhaustiveness checking.
6. **`Session.Valid` has double responsibility** — Checks both expiration AND token equality. Should be split into `IsExpired()` (already exists) and `TokenMatches(token string) bool`.

### Library Usage

7. **Form decoding** — `decoder.go`'s JSON round-trip is fragile. `gorilla/schema` is purpose-built for this and handles type coercion better.
8. **Password hashing** — Could use `golang.org/x/crypto/bcrypt` with context-aware wrapping for cancellation support (bcrypt blocks for ~250ms at cost 12).

---

## F) TOP 25 THINGS TO DO NEXT

Sorted by **impact / effort** (highest ROI first):

| #  | Task                                                                                                    | Impact | Effort | Category     |
| -- | ------------------------------------------------------------------------------------------------------- | ------ | ------ | ------------ |
| 1  | Create `usermgmt/.golangci.yml` with appropriate exclusions                                             | High   | Low    | Lint         |
| 2  | Fix usermgmt production code lint: `writeJSON` errcheck                                                 | High   | Low    | Bug          |
| 3  | Fix usermgmt production code lint: `authz.go` wrapcheck (20 issues)                                     | Medium | Low    | Quality      |
| 4  | Fix usermgmt production code lint: `service.go` wrapcheck (5 issues)                                    | Medium | Low    | Quality      |
| 5  | Fix usermgmt production code lint: `service.go` unused ctx → rename to `_`                              | Medium | Low    | Quality      |
| 6  | Fix usermgmt production code lint: `http.go` wrapcheck (json.Unmarshal)                                 | Medium | Low    | Quality      |
| 7  | Fix usermgmt production code lint: `user.go` wrapcheck (json.Marshal, rand.Read)                        | Medium | Low    | Quality      |
| 8  | Fix usermgmt production code lint: gci formatting on service.go                                         | Low    | Low    | Quality      |
| 9  | Fix usermgmt test constants: extract `"secret12"`, `"a@b.com"`, `"session_token"`                       | Low    | Low    | Quality      |
| 10 | Fix usermgmt test: exhaustruct exclusions in `.golangci.yml`                                            | Low    | Low    | Lint         |
| 11 | Fix usermgmt test: `t.Parallel()` on all test functions                                                 | Low    | Medium | Quality      |
| 12 | Fix usermgmt test: gosec cookie warnings (add nolint comments)                                          | Low    | Low    | Quality      |
| 13 | Fix usermgmt test: noctx warnings (NewRequestWithContext)                                               | Low    | Low    | Quality      |
| 14 | Fix usermgmt test: nolintlint unused directive                                                          | Low    | Low    | Quality      |
| 15 | Fix usermgmt test: unparam warnings                                                                     | Low    | Low    | Quality      |
| 16 | Fix usermgmt test: forcetypeassert check                                                                | Low    | Low    | Quality      |
| 17 | Split `Session.Valid` → `IsExpired()` + `TokenMatches()`                                                | Medium | Low    | Architecture |
| 18 | Add `InMemorySessionStore.EvictExpired()` method                                                        | Medium | Low    | Feature      |
| 19 | Change `usermgmt/contextKey` from `string` to `struct{}`                                                | Low    | Low    | Consistency  |
| 20 | Thread `context.Context` through usermgmt service methods (phase 1: rename `_` to `ctx` where possible) | Medium | Medium | Architecture |
| 21 | Add `AccountLockout` TTL eviction for non-locked entries                                                | Medium | Medium | Feature      |
| 22 | Add compensating transaction for `Register` partial failures                                            | Medium | Medium | Correctness  |
| 23 | Move timeout context creation before body read in `handleAuthEndpoint`                                  | Medium | Low    | Security     |
| 24 | Update `Authz.Apply` doc to reflect non-atomic behavior                                                 | Low    | Low    | Docs         |
| 25 | Evaluate `gorilla/schema` for form decoding                                                             | Medium | Medium | Library      |

---

## G) TOP #1 QUESTION

**How should we handle `context.Context` in usermgmt service methods?**

Currently all methods take `context.Context` but ignore it. The right fix depends on intent:

- **Option A**: Remove `ctx` from methods that can't use it (e.g., `Logout`, `GetUser`, `Authenticate`). Breaking API change but honest.
- **Option B**: Thread `ctx` through to store operations for future cancellation support. Non-breaking but requires store interface changes.
- **Option C**: Rename to `_` for now, add `//nolint:revive` comments, and plan real context support for v2.

Recommendation: **Option B** for `FindByID`, `FindByEmail`, `Create`, `Save` on stores. These are the natural points where a SQL backend would use `ctx`. For bcrypt operations, wrapping in a goroutine with ctx.Done() is complex and may not be worth it — bcrypt's CPU-bound nature makes cancellation tricky.

---

## Test Results

```
=== LINT === 0 issues (root)
=== ROOT BUILD === ok
=== ROOT TEST === ok (race clean)
=== USERMGMT BUILD === ok
=== USERMGMT TEST === ok (race clean)
=== INTEGRATION TEST === ok (race clean)
=== DATSTAR BUILD === ok
=== ALL PASSED ===
```
