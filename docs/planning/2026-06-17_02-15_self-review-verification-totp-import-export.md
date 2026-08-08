# Brutal Self-Review + Pareto Plan: Verification, TOTP, Import/Export

**Date:** 2026-06-17 02:15\
**Scope:** `usermgmt` submodule after verification/TOTP/import-export hardening session\
**Build/Test State:** All modules pass `go build ./...`, `go test ./... -race`, and `nix run .#lint`. `gopls` diagnostics are stale/cached and do **not** reflect the actual compiler state.

---

## Brutal Self-Review

### 1. What did we forget?

- **Authorization on import/export endpoints.** `/auth/import` and `/auth/export` only check that the caller is authenticated. Any logged-in user can bulk-export every user's email/roles and bulk-create accounts. There is no admin or permission gate.
- **Rate limiting on new endpoints.** Import, TOTP setup/verify, and email verification endpoints have no rate limiting beyond the existing per-IP fixed-window limiter on `/auth/register`.
- **TOTP disable confirmation.** `/auth/totp/disable` disables MFA with only a session cookie. A hijacked session can strip the second factor without proving possession.
- **Input validation on bulk import.** `ImportUser` accepts raw strings with no email format, length, or display-name checks.
- **TOTP secret retention in the event store.** `TOTPEnabled` events store the raw secret in the immutable event journal. Even after `DisableTOTP`, the secret remains in historic events forever.
- **Update root `TODO_LIST.md` / `FEATURES.md` / `CHANGELOG.md`.** The TODO list still shows coverage as 86.2% and omits the security caveats of the new endpoints.

### 2. What is something stupid that we do anyway?

- **Hand-rolled TOTP implementation.** We wrote our own RFC 6239 generator/validator when `github.com/pquerna/otp/totp` is mature, audited, and reduces the chance of subtle timing or counter bugs.
- **Duplicate rate-limiting code.** `registrationRateLimiter` in `usermgmt/http.go` is a second, simpler fixed-window limiter that ignores the sophisticated token-bucket limiter already built in the root module (`ratelimit_middleware.go`).
- **Duplicated context-timeout boilerplate.** Every handler repeats the same `ctx, cancel := r.Context(), func() {}` / `if h.timeout > 0 { ... }` block.
- **Exporting `TOTPSecret` is impossible today, but `ExportUser` still has no guarantee that future developers won't accidentally add it.** The type has no compile-time guard.

### 3. What could we have done better?

- Extracted a typed `Email` value object **before** adding import/export, so validation would be centralized and impossible to bypass.
- Designed import/export as admin-only operations from the start, with explicit `Authorize` calls or a configurable `ImportExportAuthorizer` interface.
- Used the root module's `RateLimiterMiddleware` for consistent behavior across the library.
- Written negative-path tests for invalid TOTP codes, import errors, and authorization failures **with** the handlers.

### 4. What could we still improve?

- Replace the custom TOTP implementation with `pquerna/otp/totp`.
- Introduce an `Email` branded/validated type and use it in `User`, `RegisterRequest`, `ImportUser`, `ExportUser`.
- Centralize handler timeout/context helper in `AuthHandler`.
- Reuse root token-bucket rate limiting for registration, import, and TOTP endpoints.
- Add authorization checks on import/export and require TOTP re-auth to disable TOTP.
- Encrypt TOTP secrets at rest in event payloads (or at least document the tradeoff).
- Improve test coverage for the new HTTP handlers' error branches.

### 5. Did we lie to the user?

No. The features work, tests pass, and the documented behavior matches the code. However, the docs imply these features are production-ready without calling out the missing authorization/rate-limiting guardrails. That is a lie by omission.

### 6. How can we be less stupid?

- Stop writing security-critical crypto when an established library exists.
- Stop duplicating infrastructure code (rate limiting, timeouts) when the root module already has it.
- Add authorization and rate-limiting **before** declaring an HTTP endpoint "fully functional."
- Use strong types for domain concepts (email, roles) so invalid data cannot be constructed.

### 7. Ghost systems / split brains?

- **Split brain:** `ExportFormat` is used for both import and export. The name implies export-only; a future reader will be surprised it drives import parsing too.
- **Ghost system:** `EventHandler` backward-compat bridge in `service_core.go` is wired but receives minimal test coverage for the new TOTP/email-verification events.
- **Ghost system:** `UserState.Exists()` checks `Email != ""`. This is a hidden invariant that breaks if we ever allow empty emails. A typed `Email` would make this impossible.

### 8. Are we focused on scope creep?

No. The current request is to review and plan, not to keep adding features. The planned work below is strictly hardening and consolidation.

### 9. Did we remove something useful?

No. The passwordless/event-sourced migration removed password handling, which was intentional and correct.

### 10. How are we doing on tests?

- Coverage is high, but the new handler tests (`verification_totp_http_test.go`) are mostly happy-path.
- Missing: invalid TOTP code, TOTP setup expiration, import validation failures, import/export authorization denial, CSV edge cases (wrong headers, empty rows).
- `gopls` stale diagnostics are noisy but do not affect CI.

---

## Pareto Breakdown

### The 1% that delivers ~51% of the value

1. Add admin authorization to `/auth/import` and `/auth/export` (or configurable authorizer).
2. Add rate limiting to `/auth/import`, `/auth/totp/*`, and `/auth/email/verify/*`.

→ These two changes eliminate the most severe production risks.

### The 4% that delivers ~64% of the value

3. Extract `AuthHandler.withTimeout` helper and remove duplicated timeout boilerplate.
4. Add input validation to `ImportUser` (email format, max length, display name length).
5. Replace `registrationRateLimiter` with the root module's token-bucket rate limiter.
6. Add negative-path tests for TOTP/import handlers.

### The 20% that delivers ~80% of the value

7. Replace hand-rolled TOTP with `github.com/pquerna/otp/totp`.
8. Require TOTP re-verification before disabling TOTP.
9. Introduce a validated `Email` type and propagate it through `User`, `RegisterRequest`, `ImportUser`, `ExportUser`.
10. Rename `ExportFormat` → `UserDataFormat` to reflect dual use.
11. Update `TODO_LIST.md`, `FEATURES.md`, `CHANGELOG.md`, and `usermgmt/AGENTS.md` with the new endpoints and security caveats.

### Everything else (still valuable, lower immediate impact)

12. Encrypt TOTP secrets in event payloads at rest.
13. Add `ServiceConfig.ImportExportAuthorizer` interface for custom authorization logic.
14. Add property-based tests for `foldUser` TOTP/email-verification state transitions.
15. Consolidate `UserID` ↔ `AggregateID` conversion into a single strongly-typed helper.

---

## Comprehensive Execution Plan (Medium Granularity)

Sorted by **impact / effort / risk reduction**. Estimated times are 30–100 min.

| #  | Task                                                            | Effort | Impact | Module(s)      |
| -- | --------------------------------------------------------------- | ------ | ------ | -------------- |
| 1  | Add authorization gate to import/export HTTP endpoints          | 30 min | High   | usermgmt       |
| 2  | Add rate limiting to import/TOTP/verification endpoints         | 45 min | High   | usermgmt, root |
| 3  | Extract `AuthHandler.withTimeout` helper and refactor handlers  | 30 min | Med    | usermgmt       |
| 4  | Add validation to `ImportUser` and wire into import paths       | 45 min | Med    | usermgmt       |
| 5  | Replace custom `registrationRateLimiter` with root token-bucket | 60 min | Med    | usermgmt       |
| 6  | Add negative-path tests for TOTP/import/verification handlers   | 60 min | Med    | usermgmt       |
| 7  | Replace hand-rolled TOTP with `pquerna/otp/totp`                | 75 min | High   | usermgmt       |
| 8  | Require TOTP code to disable TOTP                               | 45 min | High   | usermgmt       |
| 9  | Introduce validated `Email` type and propagate                  | 90 min | Med    | usermgmt       |
| 10 | Rename `ExportFormat` → `UserDataFormat`                        | 30 min | Low    | usermgmt       |
| 11 | Update project docs (TODO, FEATURES, CHANGELOG, AGENTS)         | 45 min | Low    | root, usermgmt |
| 12 | Document TOTP secret at-rest retention tradeoff                 | 15 min | Low    | usermgmt       |
| 13 | Add `ImportExportAuthorizer` interface for custom authorization | 60 min | Low    | usermgmt       |
| 14 | Property-based tests for foldUser TOTP/email transitions        | 60 min | Low    | usermgmt       |

---

## Detailed Execution Plan (Fine Granularity, ≤15 min each)

### P0 — Critical Security (do first)

| #  | Task (≤15 min)                                                  | Why it matters                          |
| -- | --------------------------------------------------------------- | --------------------------------------- |
| 1a | Add `ErrForbidden` response when non-admin calls `/auth/export` | Prevents data leakage                   |
| 1b | Add same admin check to `/auth/import`                          | Prevents account-creation abuse         |
| 1c | Add tests for import/export authorization denial                | Locks in the guard                      |
| 2a | Add `HandlerConfig.ImportRateLimit` + limiter wiring            | Prevents import abuse                   |
| 2b | Add `HandlerConfig.TOTPRateLimit` + limiter wiring              | Prevents TOTP brute-force               |
| 2c | Add `HandlerConfig.VerificationRateLimit` + limiter wiring      | Prevents email-verification token abuse |
| 2d | Add handler tests for 429 responses                             | Verification                            |

### P1 — Code Quality & Reuse

| #  | Task (≤15 min)                                           | Why it matters                           |
| -- | -------------------------------------------------------- | ---------------------------------------- |
| 3a | Create `AuthHandler.withTimeout(r)` helper               | Removes duplication in 8+ handlers       |
| 3b | Replace all handler timeout blocks with helper           | Consistent, testable                     |
| 4a | Add `ImportUser.Validate()` with email regex/length      | Invalid imports fail early               |
| 4b | Wire validation into `ImportUsersFromJSON/CSV`           | No bypass path                           |
| 4c | Add import validation tests                              | Coverage                                 |
| 5a | Delete `registrationRateLimiter` and config              | Remove duplication                       |
| 5b | Import root `RateLimiterMiddleware` for `/auth/register` | Use existing token-bucket implementation |
| 5c | Update registration rate-limit tests                     | Match new behavior                       |

### P2 — TOTP Hardening

| #  | Task (≤15 min)                                             | Why it matters            |
| -- | ---------------------------------------------------------- | ------------------------- |
| 7a | Add `github.com/pquerna/otp/totp` to `usermgmt/go.mod`     | Battle-tested library     |
| 7b | Replace `generateTOTPSecret/validateTOTP/generateTOTPCode` | Less custom crypto        |
| 7c | Preserve event-sourced contract (`TOTPEnabled` secret)     | Backward compat           |
| 7d | Add TOTP library tests                                     | Confirm RFC 6238 behavior |
| 8a | Add `DisableTOTP` service method signature requiring code  | MFA strip protection      |
| 8b | Update HTTP handler to require code                        | API contract              |
| 8c | Add disable-without-code and disable-with-wrong-code tests | Coverage                  |

### P3 — Type Model & Naming

| #   | Task (≤15 min)                                              | Why it matters                   |
| --- | ----------------------------------------------------------- | -------------------------------- |
| 9a  | Create `Email` type with `ParseEmail` / `MustParseEmail`    | Strong types, central validation |
| 9b  | Use `Email` in `User`, `RegisterRequest`, `ImportUser`, etc | Impossible invalid states        |
| 9c  | Update fold/read model to use `Email`                       | Consistency                      |
| 9d  | Update tests for `Email` type                               | Compilation + behavior           |
| 10a | Rename `ExportFormat` → `UserDataFormat` and constants      | Honest naming                    |
| 10b | Update all references and tests                             | Compilation                      |

### P4 — Documentation & Polish

| #   | Task (≤15 min)                                             | Why it matters                    |
| --- | ---------------------------------------------------------- | --------------------------------- |
| 11a | Update `TODO_LIST.md` coverage figure and new tasks        | Accurate status                   |
| 11b | Update `FEATURES.md` with endpoint security caveats        | Honest inventory                  |
| 11c | Update `CHANGELOG.md` with verification/TOTP/import-export | Release notes                     |
| 11d | Create `usermgmt/AGENTS.md` with module-specific gotchas   | Future AI context                 |
| 12a | Add ADR or comment documenting TOTP secret retention       | Security transparency             |
| 13a | Define `ImportExportAuthorizer` interface                  | Extensibility                     |
| 13b | Wire optional authorizer into import/export handlers       | Custom RBAC                       |
| 14a | Add rapid/property tests for TOTP/email fold transitions   | Confidence in event-sourced state |

---

## D2 Execution Graph

```d2
direction: right

security: {
  label: P0 Critical Security
  style.fill: "#ffcccc"

  auth: Admin auth on import/export
  rate: Rate limit new endpoints

  auth -> rate
}

quality: {
  label: P1 Code Quality / Reuse
  style.fill: "#ffffcc"

  timeout: Extract withTimeout helper
  validate: Validate ImportUser
  ratelimit: Reuse root rate limiter

  timeout -> validate
  validate -> ratelimit
}

totp: {
  label: P2 TOTP Hardening
  style.fill: "#ccffcc"

  lib: Use pquerna/otp/totp
  disable: Require code to disable

  lib -> disable
}

types: {
  label: P3 Type Model
  style.fill: "#cce5ff"

  email: Introduce Email type
  format: Rename ExportFormat

  email -> format
}

docs: {
  label: P4 Documentation
  style.fill: "#e5ccff"

  todo: Update TODO/FEATURES/CHANGELOG
  agents: Create usermgmt/AGENTS.md
  adr: Document TOTP secret retention
}

security -> quality -> totp -> types -> docs
```

---

## Notes & Risks

- **gopls diagnostics are stale.** The source of truth is `go build ./...`, `go test ./... -race`, and `nix run .#lint`. Do not chase `gopls` errors.
- **TOTP secret in events is intentional today** (needed for read model projection), but retaining it forever is a security liability. Short-term: document. Long-term: encrypt at rest or store outside the event journal.
- **Authorization model:** `usermgmt` already uses Casbin via `Authz`. The simplest fix is to add a policy check like `s.authz.Enforce(user, "import-export", "admin")` (or domain variant). A configurable `ImportExportAuthorizer` interface can be added later for consumers that do not use Casbin.
- **Rate limiting:** The root module exposes `RateLimiterMiddleware` and `KeyExtractorFromClientIP`. The usermgmt `registrationRateLimiter` should be removed and replaced with root middleware. For per-endpoint limiting inside `AuthHandler`, we can instantiate root `RateLimiter` structs directly.
- **TOTP library:** `github.com/pquerna/otp/totp` is compatible with RFC 6238 and generates `otpauth://` URIs. It will not increase the dependency footprint significantly.

---

## Recommended Approval Path

1. Approve **P0 + P1** as the first batch (security + cleanup).
2. After P0/P1 merge, approve **P2 + P3** (TOTP hardening + type model).
3. Finally approve **P4** (docs + extensibility).
