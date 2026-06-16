# Status Report — 2026-06-16

## Module: usermgmt — Passwordless Event-Sourced CQRS

---

## a) FULLY DONE

### Event-Sourced Architecture (commits 1dd18a4 → 2411fd9)

- ✅ User aggregate fully event-sourced via go-cqrs-lite Decider pattern
- ✅ 7 event types: UserRegistered, RolesUpdated, EmailChanged, DisplayNameChanged, UserDeleted, CredentialAdded, CredentialRemoved
- ✅ 7 command types with pure decide functions + guards
- ✅ UserState + foldUser() pure function (all event types covered)
- ✅ UserReadModel projection (FindByID, FindByEmail, Count)
- ✅ CasbinProjection — derives all policies from events, uses only public Authz methods
- ✅ Projection Runner with journal replay + live subscription
- ✅ Read-your-writes consistency via MemoryBus synchronous delivery
- ✅ DefaultEventSourcedSetup() for turn-key infrastructure

### Passwordless Authentication (commits da73483 → b542ecb)

- ✅ ALL password code removed (bcrypt, PasswordHash, ChangePassword, validatePassword, etc.)
- ✅ ALL User mutation methods removed (SetRoles, SetEmail, SetPassword, touch, etc.)
- ✅ WebAuthnCredential type with full event/command pipeline
- ✅ go-webauthn/webauthn v0.17.4 integrated
- ✅ webauthnUser adapter (domain User → webauthn.User interface)
- ✅ webauthnSessionStore (in-memory challenge store with TTL)
- ✅ Service.BeginRegistration / FinishRegistration / BeginLogin / FinishLogin
- ✅ HTTP endpoints: POST /auth/webauthn/{register,login}/{begin,finish}
- ✅ doc.go with comprehensive package documentation
- ✅ 175 tests passing with race detector
- ✅ Coverage: 79.2% (up from 69.1%)

### Cleanup (commit e722698)

- ✅ Replaced hand-rolled bytesEqual with bytes.Equal
- ✅ Removed dead defaultWebAuthnSessionTTL constant
- ✅ CasbinProjection no longer accesses unexported Authz.enforcer
- ✅ Added Authz.RemoveAllRolesForUser public method
- ✅ Removed dead NewCasbinProjectionWithNewAuthz and CasbinProjection.Authz()

### Bug Fixes (commit 232c64a)

- ✅ Fixed SignCount silent data loss in FinishLogin (removed dead mutation of clone)

---

## b) PARTIALLY DONE

| Item                  | Status                     | Gap                                                                                                                                                      |
| --------------------- | -------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Test coverage**     | 79.2%                      | 106 functions at 0% — mostly constructors/accessors on es_commands.go (21), http.go errorStatus paths (10), user.go methods (9), webauthn_adapter.go (7) |
| **Integration tests** | Basic                      | No end-to-end WebAuthn ceremony test (requires browser/virtual authenticator)                                                                            |
| **AGENTS.md**         | Updated for event-sourcing | Not yet updated for passwordless/WebAuthn                                                                                                                |
| **CHANGELOG.md**      | Has event-sourcing entry   | Missing passwordless removal + WebAuthn entry                                                                                                            |

---

## c) NOT STARTED

1. **OAuth2/OIDC provider integration** — user requested this as alternative to WebAuthn
2. **AGENTS.md update for passwordless architecture** — file listing, key decisions, gotchas
3. **CHANGELOG.md passwordless entry** — BREAKING: password removal, WebAuthn addition
4. **example_test.go update** — still references old API patterns
5. **AccountLockout integration with WebAuthn** — lockout is defined but not wired into BeginLogin
6. **WebAuthn session store TTL enforcement** — SessionData.Expires is checked but sessions never proactively cleaned up
7. **Email uniqueness in event store** — currently pre-checked via read model (TOCTOU race), not enforced at the event level

---

## d) TOTALLY FUCKED UP

Nothing is broken. All tests pass. All modules build. No panics, no race conditions.

**However, these design smells remain:**

1. **webauthn_http.go reads body twice** — reads all bytes, then tries JSON unmarshal, then resets r.Body. This works but is fragile if go-webauthn changes how it reads the request.
2. **es_commands.go has 21 uncovered functions** — these are all trivial Type()/AggregateID() accessors and constructors. Testing them individually would be busywork.
3. **Authz struct accesses enforcer directly in several places** — the enforcer field is unexported but methods within the same package can access it. Not a real leak since it's the same package, but CasbinProjection now goes through public methods while other code doesn't.

---

## e) WHAT WE SHOULD IMPROVE

### Architecture

1. **OAuth2/OIDC provider** — Add a pluggable identity provider interface for social login (Google, GitHub, etc.)
2. **Event schema versioning** — Add SchemaVersion field to events for future migrations
3. **Persistent stores** — In-memory store/bus/checkpoint are not production-ready. Need SQL/Postgres implementations.
4. **WebAuthn session cleanup** — Add periodic eviction of expired challenge sessions

### Code Quality

5. **Remove golangci-lint suppressions** for es\_\*.go files — the cyclop/funlen exclusions are necessary for event-sourcing patterns (large switch statements), but should be reviewed
6. **Add integration test** that dispatches a command and verifies projection state without the ceremony (already partially done in es_wiring_test.go)
7. **User.MarshalJSON** — currently exposes credential_count but not credential details. Consider a separate API for credential listing.

### Testing

8. **Virtual authenticator testing** — Use Chrome DevTools Protocol or a test-only authenticator to test full WebAuthn ceremonies
9. **Property-based testing** for foldUser — folding the same events should always produce the same state

---

## f) Top 25 Things to Get Done Next

| #   | Task                                                                       | Impact | Effort |
| --- | -------------------------------------------------------------------------- | ------ | ------ |
| 1   | Push to remote                                                             | High   | 1m     |
| 2   | Update AGENTS.md for passwordless architecture                             | High   | 15m    |
| 3   | Update CHANGELOG.md with passwordless BREAKING entry                       | High   | 10m    |
| 4   | Wire AccountLockout into BeginLogin flow                                   | Medium | 15m    |
| 5   | Add OAuth2/OIDC provider interface                                         | High   | 60m    |
| 6   | Add tests for es_commands.go constructors (batch test)                     | Low    | 10m    |
| 7   | Add tests for user.go Clone/HasRole/HasCredential/MarshalJSON              | Medium | 10m    |
| 8   | Add tests for store.go InMemorySessionStore edge cases                     | Low    | 10m    |
| 9   | Add test for Authz.RemoveAllRolesForUser                                   | Medium | 10m    |
| 10  | Update example_test.go for passwordless API                                | Medium | 15m    |
| 11  | Add webauthn_adapter.go nil-transport edge case tests                      | Low    | 5m     |
| 12  | Refactor webauthn_http.go body parsing to be less fragile                  | Medium | 15m    |
| 13  | Add SQL store implementation (event.Store + SessionStore)                  | High   | 120m   |
| 14  | Add event schema versioning                                                | Medium | 30m    |
| 15  | Add projection consistency integration test                                | Medium | 20m    |
| 16  | Add fuzz test for foldUser (property: idempotent fold)                     | Low    | 15m    |
| 17  | Update docs/DOMAIN_LANGUAGE.md for passwordless terms                      | Low    | 10m    |
| 18  | Add WebAuthn session expiry eviction goroutine                             | Low    | 15m    |
| 19  | Add POST /auth/credentials DELETE endpoint for credential removal via HTTP | Medium | 15m    |
| 20  | Add GET /auth/credentials endpoint for listing user's credentials          | Medium | 10m    |
| 21  | Add rate limiting to WebAuthn endpoints                                    | Medium | 15m    |
| 22  | Add CSRF protection to WebAuthn POST endpoints                             | Medium | 10m    |
| 23  | Consider adding `github.com/mark3labs/mcp-go` for MCP integration          | Low    | 30m    |
| 24  | Add README.md update for passwordless quickstart                           | Medium | 20m    |
| 25  | Add CI pipeline for multi-module testing                                   | Medium | 30m    |

---

## g) Top #1 Question

**How should OAuth2/OIDC integrate with the event-sourced model?**

Options:

1. **OAuth identity as a credential type** — Store OAuth subject ID + provider as a WebAuthnCredential-like type. Login dispatches no command, just queries read model + creates session.
2. **Separate IdentityProvider aggregate** — OAuth providers are their own event stream with their own link table to users.
3. **OAuth as a session source only** — Don't store OAuth state in the event store at all. OAuth login creates a session directly, like a magic link.

I lean toward option 1 (consistent with how WebAuthn credentials work), but I'm unsure whether the OAuth provider's `sub` claim should be the credential ID or a separate field.

---

## Metrics Summary

| Metric                   | Value                                  |
| ------------------------ | -------------------------------------- |
| Total .go files          | 60                                     |
| Test files               | 29                                     |
| Total LOC                | 6,439                                  |
| Passing tests            | 175                                    |
| Failing tests            | 0                                      |
| Coverage                 | 79.2%                                  |
| 0% functions             | 106                                    |
| Modules building         | 3/3 (root, usermgmt, integration_test) |
| Race detector            | Clean                                  |
| Pre-commit checks        | 25/25 passing                          |
| Git commits this session | 7                                      |
