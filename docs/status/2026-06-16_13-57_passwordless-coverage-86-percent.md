# Status Report — 2026-06-16 13:57

## Passwordless Event-Sourced CQRS — usermgmt Module

---

## Summary

The usermgmt module has been transformed from a CRUD store with bcrypt passwords into a **fully event-sourced, passwordless, WebAuthn-based** authentication system. All tests pass with race detector. Coverage is 86%. Zero functions at 0%.

---

## a) FULLY DONE

### Event-Sourced Architecture
- ✅ User aggregate fully event-sourced via go-cqrs-lite Decider pattern
- ✅ 7 event types: UserRegistered, RolesUpdated, EmailChanged, DisplayNameChanged, UserDeleted, CredentialAdded, CredentialRemoved
- ✅ 7 command types with pure decide functions + guards
- ✅ UserState + foldUser() pure function
- ✅ UserReadModel projection (FindByID, FindByEmail, Count)
- ✅ CasbinProjection — derives all policies from events via public Authz methods only
- ✅ Projection Runner with journal replay + live subscription
- ✅ Read-your-writes consistency via MemoryBus
- ✅ DefaultEventSourcedSetup() for turn-key infrastructure

### Passwordless Authentication
- ✅ ALL password code removed (bcrypt, PasswordHash, ChangePassword, validatePassword, etc.)
- ✅ ALL User mutation methods removed (SetRoles, SetEmail, SetPassword, touch, etc.)
- ✅ go-webauthn/webauthn v0.17.4 integrated
- ✅ WebAuthnCredential type with full event/command pipeline
- ✅ Service.BeginRegistration / FinishRegistration / BeginLogin / FinishLogin
- ✅ HTTP endpoints: POST /auth/webauthn/{register,login}/{begin,finish}
- ✅ In-memory challenge store (webauthnSessionStore) with TTL-based expiry
- ✅ webauthnUser adapter (domain User → webauthn.User interface)

### Code Quality
- ✅ bytes.Equal used (not hand-rolled)
- ✅ CasbinProjection uses only public Authz methods (no enforcer field leaks)
- ✅ Authz.RemoveAllRolesForUser added for clean user deletion
- ✅ Dead code removed (NewCasbinProjectionWithNewAuthz, CasbinProjection.Authz, defaultWebAuthnSessionTTL)
- ✅ SignCount data loss bug fixed (removed dead mutation of clone in FinishLogin)

### Test Coverage
- ✅ 195 passing tests (0 failing)
- ✅ 86.0% statement coverage
- ✅ Zero functions at 0% coverage
- ✅ All tests pass with race detector
- ✅ All 3 modules pass (root, usermgmt, integration_test)

### Documentation
- ✅ doc.go with comprehensive package documentation + usage examples
- ✅ AGENTS.md fully updated (file tree, dependencies, domain model section)
- ✅ CHANGELOG.md comprehensive BREAKING entry
- ✅ ADR 0006 (event-sourced user aggregate)

---

## b) PARTIALLY DONE

| Item | Status | Gap |
|------|--------|-----|
| **Coverage** | 86.0% | ~30 functions between 66-90% — mostly Casbin error paths + decide function error branches |
| **Authz error path coverage** | ~83% | Casbin API call errors (not nil-enforcer) are not tested — requires injecting a broken enforcer |
| **CasbinProjection.Handle** | 71.4% | EventTypes() for credential events + decode error paths untested |
| **NewCasbinProjection** | 66.7% | nil-authz error path untested |

---

## c) NOT STARTED

1. **OAuth2/OIDC provider integration** — user requested this as alternative to WebAuthn
2. **AccountLockout wired into BeginLogin** — lockout code exists and is tested but not called in the WebAuthn login flow
3. **example_test.go update** — still references old API patterns (no password field, but may have stale imports)
4. **SQL persistent stores** — in-memory only; need Postgres/SQLite event store + session store
5. **WebAuthn session cleanup goroutine** — sessions check expiry on Get but are never proactively evicted
6. **Event schema versioning** — no SchemaVersion field on events for future migrations
7. **Credential listing HTTP endpoint** — no GET /auth/credentials route
8. **Credential removal HTTP endpoint** — no DELETE /auth/credentials/{id} route
9. **CSRF protection on WebAuthn POST endpoints** — not wired
10. **Rate limiting on WebAuthn endpoints** — not wired

---

## d) TOTALLY FUCKED UP

**Nothing is broken.** All tests pass. All modules build. No race conditions. No panics.

One design concern:
- **webauthn_http.go reads request body twice** — reads all bytes, then tries JSON unmarshal, then resets r.Body. Works but is fragile if go-webauthn changes its parsing approach. Should be refactored to parse metadata from headers or query params instead.

---

## e) WHAT WE SHOULD IMPROVE

### High Priority
1. **Wire AccountLockout into BeginLogin** — 5 attempts then lockout is tested but never called
2. **OAuth2/OIDC integration** — social login as alternative to WebAuthn
3. **SQL event store** — production-ready persistence

### Medium Priority
4. **CasbinProjection event type coverage** — add CredentialAdded/Removed to EventTypes()
5. **WebAuthn session proactive eviction** — background goroutine or time-based cleanup
6. **Refactor webauthn_http.go body parsing** — use query params for user_id/credential_name instead of JSON body wrapping
7. **Test Casbin API error paths** — inject mock enforcer that returns errors
8. **example_test.go** — update for passwordless API

### Low Priority
9. **Event schema versioning** — add version field for future migrations
10. **Credential management HTTP endpoints** — list + delete credentials via HTTP
11. **Property-based testing** for foldUser
12. **Fuzz tests** for WebAuthn ceremony inputs

---

## f) Top 25 Things to Get Done Next

| # | Task | Impact | Effort |
|---|------|--------|--------|
| 1 | Wire AccountLockout into BeginLogin | High | 15m |
| 2 | Add OAuth2/OIDC provider interface | High | 60m |
| 3 | SQL event store implementation | High | 120m |
| 4 | CasbinProjection: add CredentialAdded/Removed to EventTypes() | Medium | 10m |
| 5 | Test CasbinProjection nil-authz error path | Medium | 5m |
| 6 | Test decide function error branches (event.NewEvent failures) | Low | 15m |
| 7 | Refactor webauthn_http.go to not read body twice | Medium | 20m |
| 8 | Add GET /auth/credentials endpoint | Medium | 15m |
| 9 | Add DELETE /auth/credentials/{id} endpoint | Medium | 15m |
| 10 | Update example_test.go | Medium | 15m |
| 11 | WebAuthn session proactive eviction goroutine | Low | 15m |
| 12 | Test Authz error paths with broken enforcer | Medium | 30m |
| 13 | CSRF protection on WebAuthn endpoints | Medium | 15m |
| 14 | Rate limiting on WebAuthn endpoints | Medium | 15m |
| 15 | Event schema versioning | Medium | 30m |
| 16 | Property-based test for foldUser | Low | 15m |
| 17 | Fuzz test for RegisterRequest.Validate | Low | 10m |
| 18 | Update docs/DOMAIN_LANGUAGE.md | Low | 10m |
| 19 | README.md passwordless quickstart | Medium | 20m |
| 20 | CI pipeline for multi-module testing | Medium | 30m |
| 21 | Integration test: full WebAuthn flow via virtual authenticator | Medium | 60m |
| 22 | Add Service.Authz() accessor for consumer-side auth checks | Low | 5m |
| 23 | Consider pluggable session store interface for Redis | Medium | 30m |
| 24 | Add benchmarks for WebAuthn ceremony methods | Low | 15m |
| 25 | Add structured logging to WebAuthn ceremonies | Low | 10m |

---

## g) Top #1 Question

**How should OAuth2/OIDC integrate with the event-sourced User aggregate?**

The WebAuthn path is clear: credentials are stored as events on the User stream. But OAuth2 identity is different — it's a link to an external provider, not a local credential.

Options:
1. **OAuth as a credential type** — Store `{provider, subject_id}` alongside WebAuthn credentials in the same Credentials list. Login looks up by provider+subject, creates session. Simple, consistent.
2. **Separate Identity aggregate** — OAuth providers get their own event stream with link events. More complex but allows multiple providers per user with independent lifecycle.
3. **OAuth as session-only** — Don't store OAuth state at all. Each OAuth login creates a session directly. User must exist first (registered via email). Simplest but no persistent link.

I lean toward option 1 for consistency with WebAuthn, but the question is whether `WebAuthnCredential` should be renamed to just `Credential` and made polymorphic, or if we need a separate `OAuthCredential` type.

---

## Metrics

| Metric | Value |
|--------|-------|
| Total .go files | 62 |
| Test files | 31 |
| Total LOC | 6,779 |
| Passing tests | 195 |
| Failing tests | 0 |
| Statement coverage | 86.0% |
| Functions at 0% | 0 |
| Modules building | 3/3 (root, usermgmt, integration_test) |
| Race detector | Clean |
| Git commits (this session) | 12 |
| Pre-commit checks | 25/25 passing |

### Git History (this session)

```
d56cdbf test(usermgmt): comprehensive coverage push — 79%→86%, zero 0% functions
fc3ecaf docs: update AGENTS.md and CHANGELOG for passwordless WebAuthn architecture
149fa46 docs: comprehensive status report for passwordless event-sourced usermgmt
232c64a test(usermgmt): fix SignCount bug, add service+handler tests — 69%→79% coverage
e722698 fix(usermgmt): remove dead code and fix abstraction leaks
b542ecb feat(usermgmt): integrate go-webauthn for passwordless authentication
da73483 feat(usermgmt): remove ALL password code — passwordless event-sourced domain
093a229 docs: update AGENTS.md and CHANGELOG for event-sourced architecture
2411fd9 feat(usermgmt): complete event-sourcing migration — decider, projections, read model
1dd18a4 feat(usermgmt): migrate User aggregate from CRUD to full event sourcing
```
