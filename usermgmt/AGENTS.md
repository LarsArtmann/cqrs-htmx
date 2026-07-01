# usermgmt Submodule

> Independent Go module: `github.com/larsartmann/cqrs-htmx/usermgmt/v4`
> Zero imports from root module. Cross-module bridging happens in `integration_test/` only.

## Quick Reference

| Item     | Value                                                                                            |
| -------- | ------------------------------------------------------------------------------------------------ |
| Test     | `cd usermgmt && GOWORK=off GONOSUMCHECK='github.com/larsartmann/*' go test ./... -count=1 -race` |
| Build    | `cd usermgmt && GOWORK=off GONOSUMCHECK='github.com/larsartmann/*' go build ./...`               |
| Coverage | 83.6%                                                                                            |

## Architecture

Event-sourced CQRS using go-cqrs-lite Decider pattern. All state changes are events.
`UserState` is the aggregate state, folded from events via `foldUser()`.

### Event Types (10)

`UserRegistered`, `RolesUpdated`, `EmailChanged`, `DisplayNameChanged`, `UserDeleted`, `CredentialAdded`, `CredentialRemoved`, `EmailVerified`, `TOTPEnabled`, `TOTPDisabled`

### Command Types (10)

`RegisterUser`, `UpdateRoles`, `ChangeEmail`, `ChangeDisplayName`, `DeleteUser`, `AddCredential`, `RemoveCredential`, `VerifyEmail`, `EnableTOTP`, `DisableTOTP`

## Key Gotchas

### Module Boundary

1. **GOWORK=off required** — usermgmt has its own `go.mod`. Workspace mode (`go.work`) won't work for per-module commands.
2. **Cannot import root module** — `github.com/larsartmann/cqrs-htmx` is NOT a dependency. Rate limiting, SSE, etc. are reimplemented locally.
3. **Module path has /v2** — `github.com/larsartmann/cqrs-htmx/usermgmt/v4`.

### Event Sourcing

4. **TOTP secrets stored in events** — `TOTPEnabledPayload.Secret` is persisted to the immutable event journal. Even after `TOTPDisabled`, the secret remains in historic events. This is a known tradeoff documented in `docs/adr/`.
5. **Email change resets verification + TOTP** — `EmailChanged` event sets `EmailVerified=false` and `TOTPEnabled=false` in the fold. Users must re-verify and re-setup TOTP after email change.
6. **UserState.Exists() checks Email != ""** — This is a hidden invariant. An event-sourced user with an empty email would not "exist."
7. **foldUser returns full state copies** — Each case constructs a new `UserState` with all fields. This is intentional for event sourcing purity but verbose.

### TOTP

8. **pquerna/otp/totp library** — Replaced hand-rolled RFC 6238 with `github.com/pquerna/otp/totp` v1.5.0. Secret generation, validation, and QR URI generation all use the library.
9. **DisableTOTP requires a code** — Prevents MFA stripping via session hijack. The HTTP handler reads `{"code":"..."}` from the request body.
10. **Pending secrets are ephemeral** — In-memory `pendingTOTPStore` with 5-minute TTL and background eviction. Lost on process restart (users must re-setup).

### Authorization & Rate Limiting

11. **Import/export require admin** — `ImportExportAuthorizer` defaults to `RequireAdminRole`. Set `HandlerConfig.ImportExportAuthorizer` to customize.
12. **Rate limiting per-endpoint** — `HandlerConfig.ImportRateLimit`, `TOTPRateLimit`, `VerificationRateLimit`. All use `RegistrationRateLimitConfig` type. Disabled by default (zero value).
13. **UpdateRoles revokes sessions** — `Service.UpdateRoles` calls `DeleteByUserID` after dispatching. Tests must create a new session after UpdateRoles.

### Types

14. **UserID is ULID-backed** — Alias of `id.UserID` from `go-cqrs-lite/id/v3` (ADR-0018). Backed by `ulid.ULID`, unified with root module's `cqrshtmx.UserID`. Use `.Get().String()` for string conversion at boundaries (SQL, Casbin, logging). `NewUserID(s)` accepts any string (valid ULIDs pass through, non-ULIDs are deterministically hashed). `MustParseUserID(s)` panics on invalid ULID.
15. **Email value type** — `type Email string` with `ParseEmail`/`MustParseEmail`. Used in `ExportUser`. Not propagated to `User`/`UserState`/event payloads (they stay `string` for backward compat).
16. **UserDataFormat** (renamed from ExportFormat) — Used for both import and export. Constants: `UserDataFormatJSON`, `UserDataFormatCSV`.

### Testing

17. **TOTP test codes are time-dependent** — Use `currentTOTPCode(t, secret)` which calls `totp.GenerateCode`. For deterministic invalid codes, generate from a far-past time: `totp.GenerateCode(b32Secret, time.Now().Add(-100 * TOTPTimeStep))`.
18. **setupAuthenticatedHandler grants admin** — The test helper in `verification_totp_http_test.go` registers a user and then grants admin role + creates a fresh session. Tests for non-admin scenarios must create their own setup.

## Commands

- See parent project cqrs-htmx for full build commands
- `go test ./...` for unit tests
- `golangci-lint run` for linting
