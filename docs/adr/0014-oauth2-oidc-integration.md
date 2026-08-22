# ADR 0014: OAuth2/OIDC Integration

**Status:** Accepted\
**Date:** 2026-06-18\
**Deciders:** larsartmann\
**Supersedes:** —\
**Amended by:** —

---

## Context

cqrs-htmx provides passwordless authentication via WebAuthn. Many users expect OAuth2/OIDC login ("Sign in with Google", "Sign in with GitHub") as an alternative. We needed to add external identity provider support while maintaining our event-sourced CQRS architecture and avoiding password storage entirely.

## Decision

Add OAuth2/OIDC login as a ceremony that produces events on the User aggregate, mirroring the WebAuthn pattern.

### Architecture

```
Provider → BeginOAuthLogin (state + PKCE) → Browser redirect → Provider
→ Callback → FinishOAuthLogin (exchange + userinfo) → Email match
→ (Existing user: link account) | (New user: auto-register + link)
→ Create session → Set cookie
```

### New events

- `ExternalAccountLinked` — adds an `ExternalAccount` to `UserState`
- `ExternalAccountUnlinked` — removes an `ExternalAccount` from `UserState`

### New commands

- `LinkExternalAccount`
- `UnlinkExternalAccount`

### Libraries

| Library                        | Purpose                                         | Rationale                                                 |
| ------------------------------ | ----------------------------------------------- | --------------------------------------------------------- |
| `golang.org/x/oauth2`          | OAuth2 authorization code flow with PKCE        | Stdlib-adjacent, Go team maintained, trivial PKCE support |
| `github.com/coreos/go-oidc/v3` | OIDC provider discovery + ID token verification | Red Hat maintained, RFC-compliant JWT verification        |

Both are hard dependencies in the `usermgmt` module, following the precedent set by `go-webauthn`.

### Provider configuration

```go
type OAuth2ProviderConfig struct {
    ClientID     string
    ClientSecret string
    RedirectURL  string
    Scopes       []string
    IssuerURL    string // Enables OIDC discovery + ID token verification
    // For pure OAuth2 (no OIDC):
    AuthURL     string
    TokenURL    string
    UserInfoURL string
}
```

When `IssuerURL` is set, the library discovers endpoints via OIDC `.well-known/openid-configuration` and verifies ID tokens. When empty, explicit endpoints are used and the UserInfo endpoint fetches user details.

### Security decisions

1. **PKCE (S256) on every flow** — `oauth2.GenerateVerifier()` + `oauth2.S256ChallengeOption()` + `oauth2.VerifierOption()`. Prevents authorization code interception.
2. **CSRF state tokens** — 32-byte random base64url tokens, 10-minute TTL, one-time use. Stored in-memory with background eviction (mirrors WebAuthn challenge store pattern).
3. **Email is auto-trusted from OAuth providers** — Google/GitHub/Microsoft verify emails during their own registration. We set `EmailVerified = true` and skip our own verification flow.
4. **Last-auth-method guard** — `UnlinkExternalAccount` is rejected if the user has zero WebAuthn credentials AND zero other external accounts. Prevents account lockout.

### Email matching

```
FinishOAuthLogin:
  user = FindByEmail(providerEmail)
  if found:
    LinkExternalAccount(user, provider, subject)
    return user
  else:
    RegisterUser(email=providerEmail, displayName=providerName)
    LinkExternalAccount(newUser, provider, subject)
    VerifyEmail() // provider already verified it
    return newUser
```

### Type model

```
ExternalAccount {
  Provider    string    // "google", "github"
  Subject     string    // provider's unique user ID
  Email       string    // email from provider (may differ from User.Email)
  DisplayName string
  LinkedAt    time.Time
}
```

The `Provider+Subject` pair is the unique deduplication key. A given provider subject can only be linked to one user at a time.

### foldUser refactor prerequisite

Before adding `ExternalAccounts []ExternalAccount` to `UserState`, the `foldUser` function was refactored from "full struct rebuild per case" to "shallow copy + mutate selected fields". This made adding the new field O(1) instead of touching every existing case.

## Consequences

### Positive

- Users can authenticate without registering a passkey first — lowers friction.
- Multi-provider support: one user can link Google + GitHub simultaneously.
- Auto-registration means no separate "sign up" vs "sign in" flows for OAuth users.
- All state changes are events — replay, audit, and projection work unchanged.

### Negative

- Two new hard dependencies (`oauth2`, `go-oidc`) in the `usermgmt` module.
- OIDC discovery makes HTTP calls at `NewService` startup — slow/broken discovery blocks service creation.
- Email matching creates an implicit trust relationship: if an attacker compromises an OAuth provider account with the same email as an existing user, they gain access. Mitigation: providers verify emails (especially Google Workspace / GitHub verified domains).

### Alternatives considered

1. **Build a separate OAuth2 microservice** — Rejected: would require cross-service coordination for CQRS events, breaking the event-sourced boundary.
2. **Use an OAuth2 proxy (e.g., Vouch, OAuth2 Proxy)** — Rejected: adds deployment complexity and doesn't integrate with our session/authz model.
3. **Soft dependency (build tags)** — Rejected: would require consumers to opt-in at compile time, adding complexity for a stdlib-adjacent dependency.
4. **OIDC only, no pure OAuth2** — Rejected: GitHub doesn't support OIDC for OAuth apps, only for GitHub Apps. Pure OAuth2 userinfo endpoint is required.

## Downstream consumers

- **dnsblockd** (2026-08-21, `v4.7.0`) — first recorded consumer of the standalone `usermgmt/oauth2/v4` sub-module outside the cqrs-htmx tree. Uses the provider for dashboard single sign-on against Pocket ID (authorization-code + PKCE S256, confidential client). The consumer owns sessions/states/cookies/audit itself and treats the library purely as the protocol engine (discovery, PKCE, exchange, ID-token verification, claim extraction) — confirming the sub-module fits the "embedded appliance" shape with no usermgmt-core coupling. Integration notes live in dnsblockd `AGENTS.md` ("Dashboard Auth") and `docs/status/2026-08-21_22-32_OIDC-SSO-CQRS-HTMX-LIBRARY-ADOPTION.md`.

### Open upstream considerations (from that adoption)

1. ~~`ProviderConfig.Validate` requires `ClientSecret` — public/PKCE-only clients are structurally unsupported.~~ **RESOLVED 2026-08-22** via `ClientType` (`ClientTypePublic`, default `ClientTypeConfidential`): the secret requirement is conditional, PKCE S256 stays mandatory. Filed 2026-08-22: [#8](https://github.com/LarsArtmann/cqrs-htmx/issues/8).
2. ~~Only `sub`/`email`/`email_verified`/`name` are extracted from the ID token; consumers wanting `preferred_username` or the raw token must fork `FinishLogin`.~~ **RESOLVED 2026-08-22**: `preferred_username` is now extracted (ID token + UserInfo, `login` fallback) and `FinishLoginWithToken` exposes the verified raw ID token. Filed 2026-08-22: [#9](https://github.com/LarsArtmann/cqrs-htmx/issues/9).
