# OAuth2/OIDC Integration Plan — cqrs-htmx

**Created:** 2026-06-18 21:30 · **Granularity:** every task ≤ 12 min · **Mirrors:** existing WebAuthn pattern (ceremony → event → session)

## Design Summary

OAuth2/OIDC follows the same architecture as WebAuthn: a ceremony produces events that mutate the User aggregate. The key difference: OAuth uses redirect-based flows (not challenge-response), and accounts are matched by email (not credential ID).

**New events:** `ExternalAccountLinked`, `ExternalAccountUnlinked`
**New commands:** `LinkExternalAccount`, `UnlinkExternalAccount`
**Libraries:** `golang.org/x/oauth2` + `github.com/coreos/go-oidc/v3` (Go ecosystem standard, mirrors go-webauthn pattern)

**Security rules:**

1. Email from verified OAuth providers is auto-trusted (marks `EmailVerified = true`)
2. Unlinking the last authentication method (no WebAuthn creds + no other external accounts) is rejected
3. State tokens prevent CSRF (same pattern as WebAuthn challenge store)

---

## Tier Summary

| Tier   | Theme                                       | Tasks  | Est.  |
| ------ | ------------------------------------------- | ------ | ----- |
| **D1** | Domain layer (no deps needed)               | 12     | ~115m |
| **D2** | Ceremony + service layer (needs oauth2 lib) | 8      | ~90m  |
| **D3** | HTTP + tests                                | 6      | ~65m  |
| **D4** | Docs + integration tests                    | 3      | ~28m  |
|        | **TOTAL**                                   | **29** | ~5h   |

---

## Master Task Table (sorted by Score ↓, then Effort ↑)

| #  | Tier | Task                                                                                             | Impact | Value | Effort | Score  |
| -- | ---- | ------------------------------------------------------------------------------------------------ | :----: | :---: | :----: | :----: |
| 1  | D1   | **Refactor foldUser to use mutable update pattern** (eliminate 10× struct rebuild)               |   5    |   4   |  12m   | **20** |
| 2  | D1   | Add `ExternalAccounts []ExternalAccount` field to `UserState`                                    |   5    |   4   |  10m   | **20** |
| 3  | D1   | Define `ExternalAccount` value type + `Clone()` (mirrors `WebAuthnCredential`)                   |   4    |   4   |  10m   | **16** |
| 4  | D1   | Define `ExternalAccountLinkedPayload` + `ExternalAccountUnlinkedPayload`                         |   4    |   4   |   8m   | **16** |
| 5  | D1   | Add event constants + update `allUserEventTypes`                                                 |   4    |   4   |   5m   | **16** |
| 6  | D1   | `decideLinkExternalAccount` — guards: exists, not duplicate                                      |   4    |   4   |  10m   | **16** |
| 7  | D1   | `decideUnlinkExternalAccount` — guards: exists, link present, not last auth                      |   4    |   4   |  12m   | **16** |
| 8  | D1   | `foldUser`: handle `ExternalAccountLinked` event                                                 |   4    |   4   |   8m   | **16** |
| 9  | D1   | `foldUser`: handle `ExternalAccountUnlinked` event                                               |   4    |   4   |   8m   | **16** |
| 10 | D1   | Define `LinkExternalAccountCmd` + `UnlinkExternalAccountCmd`                                     |   3    |   4   |  10m   | **12** |
| 11 | D1   | `RegisterCommands`: wire both new commands via `RegisterTyped`                                   |   3    |   4   |  10m   | **12** |
| 12 | D1   | `UserReadModel`: project linked/unlinked events                                                  |   3    |   4   |  12m   | **12** |
| 13 | D1   | `CasbinProjection`: add event types for ordering (no-op handler)                                 |   2    |   3   |   5m   | **6**  |
| 14 | D1   | Domain unit tests: decide guards (duplicate link, last auth, not found)                          |   4    |   3   |  12m   | **12** |
| 15 | D1   | Domain unit tests: fold invariants (link adds, unlink removes, idempotent)                       |   4    |   3   |  12m   | **12** |
| 16 | D2   | Add `golang.org/x/oauth2` + `go-oidc/v3` deps to usermgmt/go.mod                                 |   3    |   4   |   5m   | **12** |
| 17 | D2   | Define `OAuth2ProviderConfig` + `OAuth2Config` (map of providers)                                |   4    |   4   |  10m   | **16** |
| 18 | D2   | `oauth2StateStore` — mirrors `webauthnSessionStore` (state tokens, CSRF)                         |   4    |   4   |  12m   | **16** |
| 19 | D2   | `Service.BeginOAuthLogin(provider) → redirectURL`                                                |   4    |   4   |  12m   | **16** |
| 20 | D2   | `Service.FinishOAuthLogin(provider, code, state) → (*User, *Session)`                            |   5    |   5   |  12m   | **25** |
| 21 | D2   | Email matching logic: find-by-email → link or auto-register                                      |   4    |   4   |  12m   | **16** |
| 22 | D2   | OIDC ID token verification helper (when provider supports OIDC)                                  |   3    |   3   |  12m   | **9**  |
| 23 | D3   | OAuth2 HTTP handlers: `GET /auth/oauth/{provider}/begin` + `GET /auth/oauth/{provider}/callback` |   4    |   4   |  12m   | **16** |
| 24 | D3   | Rate limiting on OAuth endpoints (mirror WebAuthn pattern)                                       |   3    |   3   |   8m   | **9**  |
| 25 | D3   | Session cookie on successful OAuth login (mirror `setSessionCookie`)                             |   3    |   3   |   5m   | **9**  |
| 26 | D3   | `User` struct: expose `ExternalAccounts` in JSON + `MarshalJSON`                                 |   3    |   3   |   8m   | **9**  |
| 27 | D3   | `Service.UnlinkExternalAccount(userID, provider)` — account management                           |   3    |   3   |  10m   | **9**  |
| 28 | D4   | Integration test: full OAuth flow with fake provider                                             |   4    |   4   |  12m   | **16** |
| 29 | D4   | ADR 0014: OAuth2/OIDC integration design decision                                                |   3    |   3   |  10m   | **9**  |

---

## Execution Order

**Phase 1 — Domain (no external deps, all testable now):**
Tasks 1→15. Refactor foldUser, add types/events/commands/deciders/fold/read-model, unit tests.

**Phase 2 — Ceremony (needs oauth2 lib):**
Tasks 16→22. Add deps, state store, service methods, email matching, OIDC verification.

**Phase 3 — HTTP + wiring:**
Tasks 23→27. HTTP handlers, rate limiting, session cookies, account management.

**Phase 4 — Tests + docs:**
Tasks 28→29. Integration test, ADR.

---

## Key Design Decisions (to document in ADR 0014)

1. **Hard dependency on `golang.org/x/oauth2`** — mirrors how `go-webauthn` is a hard dep. Stdlib-adjacent, maintained by Go team.
2. **Email-based matching** — OAuth providers verify emails; match incoming email to existing users. Auto-register if not found.
3. **Last-auth-method guard** — `UnlinkExternalAccount` rejected if user has 0 WebAuthn credentials and 0 other external accounts.
4. **State tokens in memory** — same pattern as WebAuthn challenge store. `oauth2StateStore` with 5-min eviction.
5. **Multi-provider** — `OAuth2Config` is a `map[string]OAuth2ProviderConfig` keyed by provider name ("google", "github", etc.).
6. **OIDC when available** — if provider configures an `IssuerURL`, use `go-oidc` for ID token verification. Falls back to userinfo endpoint for pure OAuth2 providers.
