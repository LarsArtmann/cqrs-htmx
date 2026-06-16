# Status Report — 2026-06-17 01:16

## TODO Completion Session: Verification, TOTP, Import/Export, Read Model Hardening

---

## Summary

Completed the remaining TODO items from the hardening session: `UserReadModel.AllUsers()` + export encapsulation, background eviction for pending TOTP and verification tokens, structured logging for TOTP/email verification, and HTTP handlers for email verification, TOTP, and user import/export. Also fixed an incomplete `Clone()` deep-copy (credential inner slices). **All 3 modules pass with race detector. Usmermgmt coverage: 83.5%. Lint: 0 issues in both root and usermgmt.**

---

## a) FULLY DONE

### TODO Items Completed

- ✅ **Fix `Clone()` to deep-copy credentials** — Added `WebAuthnCredential.Clone()` and used it in `User.Clone()` so ID, PublicKey, Transports, and AAGUID are independent allocations. Prevents accidental shared mutable state when returning users from the read model.
- ✅ **Add `AllUsers()` to `UserReadModel` and fix import/export encapsulation** — New `AllUsers()` returns a sorted (CreatedAt, then ID), deep-copied slice. `exportAllUsers()` no longer reaches through `readModel.mu`/`readModel.users` directly.
- ✅ **CSV export includes `totp_enabled`** — ExportUsersToCSV header and rows now include the TOTP flag, using `strconv.FormatBool` to avoid magic string constants.
- ✅ **Wire background eviction for `pendingTOTP` and `verificationTokens`** — Both stores now have `startEviction()` goroutines matching the WebAuthn session store pattern. `Service.Stop()` stops all three goroutines and is idempotent.
- ✅ **Add structured logging to TOTP and email verification methods** — `EnableTOTP`, `VerifyTOTPSetup`, `VerifyTOTP`, `DisableTOTP`, `SendVerificationEmail`, and `VerifyEmail` now emit `usermgmt:` auth events via `s.logAuth()`, including failure reasons.
- ✅ **Add HTTP handlers for email verification, TOTP, import/export** — New routes registered under `AuthHandler.RegisterRoutes()`:
  - `POST /auth/email/verify/send`
  - `POST /auth/email/verify`
  - `POST /auth/totp/setup`
  - `POST /auth/totp/setup/verify`
  - `POST /auth/totp/verify`
  - `POST /auth/totp/disable`
  - `GET /auth/export?format=json|csv`
  - `POST /auth/import?format=json|csv`
- ✅ **Error status mapping** — `errorStatus()` now maps TOTP/email errors to appropriate HTTP statuses (409 Conflict, 400 Bad Request, 401 Unauthorized, 503 Service Unavailable).
- ✅ **Tests added** — 19 new tests across `readmodel_test.go`, `eviction_test.go`, and `verification_totp_http_test.go` covering AllUsers, credential deep-copy, CSV header, eviction, stop idempotency, and all new HTTP endpoints.

### Metrics

| Metric               | Before Session | After Session |
| -------------------- | -------------- | ------------- |
| usermgmt tests       | ~570           | ~589          |
| usermgmt coverage    | ~84%           | 83.5%         |
| Root lint issues     | 0              | 0             |
| Usmermgmt lint issues| 0              | 0             |
| New .go files        | —              | 2             |
| New test files       | —              | 3             |

---

## b) PARTIALLY DONE

| Item | Status | Gap |
| ---- | ------ | --- |
| **Coverage** | 83.5% | Slightly down because new HTTP handler code and import/export paths added ~250 LOC. Coverage will rise as more error branches are tested. |
| **HTTP handler authorization** | Requires middleware | New import/export endpoints require authentication via `NewSessionMiddleware`. No built-in admin/role check — consumers must layer their own authorization (e.g., Casbin middleware). |

---

## c) NOT STARTED

1. **OAuth2/OIDC provider integration** — Social login as alternative to WebAuthn.
2. **Event schema versioning upcasters** — `SchemaVersion` is set but no upcaster machinery exists yet.
3. **Pluggable Redis session store** — SKIPPED per prior decision.
4. **Admin user management UI (templ + HTMX)** — Management interface.
5. **Passwordless account recovery flow** — Email-based recovery for lost credentials.
6. **Audit log HTTP endpoint / persistence** — Currently in-memory only.
7. **Distributed rate limiter (Redis-backed)** — For multi-instance deployments.
8. **SQL event store: Postgres test suite** — Currently only SQLite tested.

---

## d) TOTALLY FUCKED UP

**Nothing is broken.** All 3 modules build. All tests pass with race detector. No panics in production code.

One note:

- **`Clone()` was incomplete** — The previous hardening session fixed `TOTPSecret` deep-copy but left credential inner slices aliased. This session completed the fix. No security incident; the read model always returned clones, but the clones were shallow for credential byte/string slices.

---

## e) WHAT WE SHOULD IMPROVE

### High Priority

1. **Coverage push for new HTTP handlers** — Test error branches (invalid body, missing auth, TOTP not configured, invalid codes, expired setup) to bring coverage back above 84%.
2. **Role-based authorization for import/export** — The new endpoints are admin-ish operations. Consider a built-in `Admin` role check or a configurable `AuthorizeFunc` in `HandlerConfig`.
3. **Update AGENTS.md / CHANGELOG.md / FEATURES.md** — Document the new verification, TOTP, and import/export capabilities.

### Medium Priority

4. **Rate limiting for new endpoints** — Import/export and TOTP endpoints could be rate-limited to prevent abuse.
5. **TOTP backup codes** — Provide single-use backup codes during TOTP setup for account recovery.
6. **Email verification callback URL** — Currently `SendVerificationEmailFunc` receives a raw token. Consider a helper that builds the full verification URL.
7. **Export pagination** — For large user bases, export should support pagination or streaming.

### Low Priority

8. **Audit log persistence** — Write audit entries to SQL instead of memory.
9. **Audit log HTTP endpoint** — Allow admins to query audit entries.
10. **Distributed session store** — Redis-backed `SessionStore`.
11. **Benchmarks for import/export** — Measure CSV/JSON throughput.
12. **Admin UI** — templ + HTMX user management interface.

---

## f) Top 25 Things to Get Done Next

| # | Task | Impact | Effort |
| --- | ------------------------------------------------------------------------ | ------ | ------ |
| 1 | Coverage tests for new HTTP handler error branches | Medium | 30m |
| 2 | Role-based authorization for import/export endpoints | High | 45m |
| 3 | Update AGENTS.md with new features | Medium | 15m |
| 4 | Update CHANGELOG.md with verification/TOTP/import/export | Medium | 15m |
| 5 | Update FEATURES.md feature inventory | Medium | 15m |
| 6 | TOTP backup codes | Medium | 60m |
| 7 | Rate limiting for TOTP/import/export endpoints | Medium | 30m |
| 8 | Email verification URL builder helper | Low | 15m |
| 9 | OAuth2/OIDC provider integration | High | 120m |
| 10 | Event schema upcaster machinery | Medium | 60m |
| 11 | Audit log SQL persistence | Medium | 60m |
| 12 | Audit log HTTP endpoint | Low | 30m |
| 13 | SQL event store: Postgres tests + connection pooling | High | 60m |
| 14 | Passwordless account recovery flow | High | 90m |
| 15 | Distributed Redis rate limiter | Low | 60m |
| 16 | Redis session store | Low | 45m |
| 17 | Export pagination / streaming | Low | 45m |
| 18 | WebAuthn integration test coverage to 70%+ | High | 120m |
| 19 | Admin UI (templ + HTMX) | Medium | 120m |
| 20 | Benchmarks for import/export | Low | 20m |
| 21 | CI: add Postgres service for SQL store tests | Medium | 30m |
| 22 | Update TODO_LIST.md / close completed items | Low | 10m |
| 23 | Refactor `verificationTokenStore` TTL to be configurable per-token | Low | 15m |
| 24 | Add OpenAPI/Swagger docs for auth endpoints | Low | 90m |
| 25 | Security review of new endpoints (CSP, CSRF, authZ) | High | 60m |

---

## g) Top #1 Question

**Should import/export endpoints have built-in role-based authorization, or should that remain the consumer's responsibility?**

The new `/auth/import` and `/auth/export` endpoints require authentication (via `NewSessionMiddleware`) but do not enforce any role. Options:

1. **Consumer responsibility** — Keep AuthHandler focused on auth. Consumers wrap the mux with Casbin/root authorization middleware. Preserves current design and clean boundaries.
2. **Built-in admin role check** — Add a simple `RequireRole(RoleAdmin)` helper or `HandlerConfig.AdminRoles` slice. Easier out-of-the-box but couples usermgmt to role semantics.
3. **Configurable authorization callback** — `HandlerConfig.Authorize func(user *User, r *http.Request) bool` lets consumers inject policy without usermgmt dictating roles.

I lean toward option 3: add an optional `Authorize` callback to `HandlerConfig` for the import/export endpoints, defaulting to allowing any authenticated user (current behavior). This keeps the library flexible while making secure defaults easy.

---

## Git History (this session)

```
c412fc9 feat(usermgmt): add HTTP handlers for verification, TOTP, and import/export
a3aee73 feat(usermgmt): encapsulate read model, add eviction, structured logging
4400d1e fix(usermgmt): fix 5 critical bugs in TOTP/email verification/export
```

### Files Changed

**Modified (10):**

- `usermgmt/credential.go` — Added `WebAuthnCredential.Clone()`.
- `usermgmt/credential_http.go` — Use `statusRemoved` constant.
- `usermgmt/email_verification.go` — Structured logging; eviction goroutine; use `statusVerified` constant.
- `usermgmt/es_readmodel.go` — Added `AllUsers()` with sorting and deep-copy.
- `usermgmt/http.go` — Status constants; `RegisterVerificationTOTPRoutes` wiring; extended `errorStatus()`.
- `usermgmt/import_export.go` — Use `AllUsers()`; add `totp_enabled` CSV column; `strconv.FormatBool`.
- `usermgmt/service_core.go` — Start/stop eviction goroutines for verification and pending TOTP.
- `usermgmt/totp.go` — Structured logging; pending TOTP eviction; use status constants.
- `usermgmt/user.go` — Deep-copy credentials via `WebAuthnCredential.Clone()`.
- `usermgmt/webauthn_http.go` — Use `statusRegistered` constant.

**New (5):**

- `usermgmt/verification_totp_http.go` — New HTTP handlers for verification, TOTP, import/export.
- `usermgmt/verification_totp_http_test.go` — Handler tests.
- `usermgmt/eviction_test.go` — Eviction and `Stop()` tests.
- `usermgmt/readmodel_test.go` — `AllUsers()`, credential clone, CSV header tests.
