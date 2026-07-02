# ADR 0035: Auth Strategy Extraction — TOTP, WebAuthn, OAuth2 as Independent Modules

**Date:** 2026-07-02
**Status:** Accepted

## Context

The v3 `usermgmt` module was a god-package (~11K LOC, 84 files) that pulled in
**every** auth-related dependency as a direct or transitive cost:

| Dependency            | Why it was in core                     | Consumers who needed it |
| --------------------- | -------------------------------------- | ----------------------- |
| `go-webauthn`         | WebAuthn ceremony code + adapter       | Only passkey apps       |
| `pquerna/otp`         | TOTP secret gen + validation           | Only TOTP MFA apps      |
| `golang.org/x/oauth2` | OAuth2 authorization code flow         | Only social-login apps  |
| `coreos/go-oidc/v3`   | OIDC discovery + ID token verification | Only OIDC apps          |
| `go-jose/go-jose/v4`  | JWT signing (transitive from go-oidc)  | Only OIDC apps          |

A consumer who only wanted basic user management + session auth (no WebAuthn,
no OAuth2, no TOTP) still pulled all five libraries into their binary.

The [2026-07-01 Sollbruchstellen analysis](../modularization/2026-07-01_SOLLBRUCHSTELLEN.html)
identified auth strategy extraction as the **1% effort → 51% impact** change:
extracting these three strategies behind interfaces would eliminate 5 direct
dependencies and all their transitives, with minimal API surface change.

v3 already defined the interfaces (`TOTPProvider`, `WebAuthnProvider`,
`OAuth2Provider` in `auth_interfaces.go`) as purely additive seams. The
extraction was the v4-breaking step: moving implementations behind those
interfaces and into separate Go modules.

## Decision

Extract TOTP, WebAuthn, and OAuth2 into three independent Go modules. Core
`usermgmt` retains **zero** auth-related dependencies.

### Module structure

```
usermgmt/              # core — zero auth deps
usermgmt/totp/         # pquerna/otp
usermgmt/webauthn/     # go-webauthn
usermgmt/oauth2/       # oauth2 + oidc (+ go-jose transitive)
```

### Interface design: primitive types + structural typing

The interfaces use **only primitive types** (`[]byte`, `string`) so the
implementation modules do not import core `usermgmt`:

```go
// usermgmt/auth_interfaces.go
type WebAuthnProvider interface {
    BeginRegistration(ctx, userJSON []byte) (options, sessionData []byte, err error)
    FinishRegistration(ctx, userJSON, body, sessionData []byte) (credentialJSON []byte, err error)
    BeginLogin(ctx, userJSON []byte) (options, sessionData []byte, err error)
    FinishLogin(ctx, userJSON, body, sessionData []byte) error
}
```

The provider modules satisfy these interfaces via **structural typing** — they
never import `usermgmt`. Compile-time satisfaction is verified in
`integration_test/auth_interface_assert_test.go`.

### JSON serialization boundary

Core serializes user data to JSON (`userJSON`) and passes it to the provider.
The provider deserializes, runs the ceremony, and returns results as JSON.
This boundary is what enables module separation:

```
Service → marshal(user) → userJSON []byte → Provider.BeginRegistration()
                                                    ↓
Provider → unmarshal(userJSON) → go-webauthn ceremony → marshal(credential) → credentialJSON []byte
                                                    ↓
Service ← unmarshal(credentialJSON) → AddCredential command
```

The JSON wire format between core and providers is an internal contract (not
part of the public API). The field names match the `credentialCore` struct's
JSON tags.

### Consumer injection pattern

Consumers import only what they need:

```go
import (
    "github.com/larsartmann/cqrs-htmx/usermgmt/v4"
    "github.com/larsartmann/cqrs-htmx/usermgmt/webauthn/v4"  // only if passkeys
    // does NOT import oauth2 or totp
)

svc, _ := usermgmt.NewService(usermgmt.ServiceConfig{
    WebAuthn: must(webauthn.New(webauthn.Config{
        RPID: "example.com", RPOrigins: []string{"https://example.com"},
    })),
})
```

### Breaking changes (v3 → v4)

| v3 API                                               | v4 API                                               |
| ---------------------------------------------------- | ---------------------------------------------------- |
| `ServiceConfig.WebAuthnConfig`                       | `ServiceConfig.WebAuthn` (inject `WebAuthnProvider`) |
| `ServiceConfig.TOTPConfig`                           | `ServiceConfig.TOTP` (inject `TOTPProvider`)         |
| `ServiceConfig.OAuth2Providers`                      | `ServiceConfig.OAuth2` (inject `OAuth2Provider`)     |
| `WebAuthnSessionStore` uses `*webauthn.SessionData`  | Uses `[]byte`                                        |
| `OAuth2StateStore.Save(provider, pkceVerifier, ttl)` | `Save(state, provider, pkceVerifier, ttl)`           |
| `BeginRegistrationResponse.Options` typed            | `json.RawMessage` (same wire format)                 |

See `docs/migrations/v3-to-v4.md` for complete before/after examples.

## Consequences

### Positive

- **Zero auth deps in core usermgmt**: `go mod graph` in usermgmt shows no
  webauthn, oauth2, oidc, jose, or pquerna dependencies
- **Pay-per-use**: consumers who only need passkeys don't pull TOTP or OAuth2
- **Testable in isolation**: each sub-module has its own focused test suite
  (W3C spec vectors for WebAuthn, mock OIDC provider for OAuth2)
- **Clear module boundaries**: each strategy is a focused, independent module

### Negative

- **JSON serialization boundary overhead**: core marshals user data, provider
  unmarshals, runs ceremony, marshals result, core unmarshals. For ceremonies
  (not hot paths), this is negligible — benchmarked at **~400ns** (0 creds)
  to **~1.2µs** (2 creds), 3-4 allocs per call. See
  `usermgmt/webauthn_benchmark_test.go`.
- **New untested code path**: the marshal/unmarshal boundary is new code.
  Mitigated by provider tests (W3C vectors, mock OIDC provider), fuzz tests
  (`FuzzMarshalWebAuthnUser`, `FuzzParseUser`, `FuzzParseSession`), and
  cross-module integration tests in `integration_test/`.
- **Consumer must wire providers**: one extra import + constructor call per
  strategy. The migration guide documents the pattern.
- **OAuth2StateStore signature change**: the most disruptive API change.
  Custom implementations (e.g., Redis-backed) must add the `state` parameter.

### Neutral

- **Structural typing**: providers don't import core, so the interface
  satisfaction is checked in `integration_test`, not in the sub-module itself.
  This is the same pattern as `Enforcer` (casbin) and `TemplComponent` (templ).

## Testing

Each sub-module has its own test suite:

- **TOTP** (`provider_test.go`): generate/validate round-trip, default config
- **WebAuthn** (`provider_test.go`): full ceremony with W3C spec test vectors
  (registration → login), wrong challenge rejection, expired session, credential
  conversion round-trip, transport conversion, user adapter
- **OAuth2** (`provider_test.go`): config validation (6 cases), PKCE URL
  generation, pure OAuth2 flow with mock server, OIDC flow with real JWT
  signing/verification, GitHub login fallback, error cases
- **integration_test**: compile-time interface satisfaction assertions for
  all three providers
