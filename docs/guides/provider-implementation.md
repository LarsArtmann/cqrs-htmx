# Provider Implementation Guide

cqrs-htmx uses **structural typing** for auth strategy interfaces. Your provider implementation doesn't need to import the `usermgmt` package — it just needs to match the method signatures. This keeps auth strategy dependencies out of core `usermgmt`.

## Interface Overview

All three interfaces live in `usermgmt/auth_interfaces.go` and use only primitive types (`[]byte`, `string`) at the boundary.

### TOTPProvider

```go
type TOTPProvider interface {
    GenerateSecret(email string) (rawSecret, base32Secret, otpauthURI string, err error)
    ValidateCode(secret, code string) bool
}
```

**Implementation steps:**

1. Import your TOTP library (e.g., `pquerna/otp/totp`)
2. Create a struct with any config you need (period, digits, issuer)
3. Implement both methods using primitive types only
4. Pass an instance to `ServiceConfig.TOTP`

**Reference:** `usermgmt/totp/provider.go` — 60 LOC using `pquerna/otp`

### WebAuthnProvider

```go
type WebAuthnProvider interface {
    BeginRegistration(ctx context.Context, userJSON []byte) (options, sessionData []byte, err error)
    FinishRegistration(ctx context.Context, userJSON, body, sessionData []byte) (credentialJSON []byte, err error)
    BeginLogin(ctx context.Context, userJSON []byte) (options, sessionData []byte, err error)
    FinishLogin(ctx context.Context, userJSON, body, sessionData []byte) (credentialJSON []byte, err error)
}
```

**Key design points:**

- `userJSON`/`body`/`sessionData` are `[]byte` (JSON-serialized) — you handle serialization internally; `BeginLogin` receives the user's allowed credentials inside `userJSON`
- `WebAuthnCredentialDescriptor` is a simple struct with `ID []byte` and `Type string`
- `credentialJSON` (registration AND login) is the full credential state as JSON — on login it must carry the assertion-observed state (signature counter + user-present/user-verified/backup flags), because `usermgmt` dispatches `RecordCredentialUse` with it: the stored counter becomes the high-water mark the next assertion must advance past (WebAuthn L3 §7.2 — this is what makes clone detection live), and a regression is rejected as a conflict
- Credentials created before the sign-counter persistence (identity-model v4.14.0) carry the registration-time state — `UserPresent`/`UserVerified` start `false` until the credential's next login records an assertion
- The `usermgmt` package handles storing credentials as events

**Reference:** `usermgmt/webauthn/provider.go` — ~300 LOC using `go-webauthn`

### OAuth2Provider

```go
type OAuth2Provider interface {
    BeginLogin(ctx context.Context, provider, state string) (redirectURL, pkceVerifier string, err error)
    FinishLogin(ctx context.Context, providerName, code, pkceVerifier string) (userInfoJSON []byte, err error)
}
```

**Key design points:**

- `userInfoJSON` must be JSON-serialized `OAuth2UserInfo` struct (subject, email, display_name, email_verified)
- PKCE (S256) is mandatory — the verifier is passed back to FinishLogin
- Multiple providers (Google, GitHub, etc.) can be supported in one implementation
- The `usermgmt` package handles user matching/registration based on provider+subject and email

**Reference:** `usermgmt/oauth2/provider.go` — ~250 LOC using `golang.org/x/oauth2` + `coreos/go-oidc`

## Compile-time Assertions

The `integration_test` module contains compile-time interface assertions:

```go
var _ usermgmt.OAuth2Provider = (*oauth2.Provider)(nil)
var _ usermgmt.TOTPProvider = (*totp.Provider)(nil)
var _ usermgmt.WebAuthnProvider = (*webauthn.Provider)(nil)
```

Add your provider to `integration_test/auth_interface_assert_test.go` to verify the structural match.

## Multi-Provider Setup

```go
svc, _ := usermgmt.NewService(usermgmt.ServiceConfig{
    TOTP:     myTOTPProvider{},
    WebAuthn: myWebAuthnProvider{...},
    OAuth2:   myOAuth2Provider{...},
})
```

Each strategy is opt-in — pass `nil` (or omit) for strategies you don't need. The `Service` methods (`HasTOTP`, `HasWebAuthn`, `HasOAuth2`) reflect which strategies are active.
