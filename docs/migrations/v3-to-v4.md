# v3 → v4 Migration Guide

> **Status:** Complete — all three auth strategies (TOTP, WebAuthn, OAuth2) extracted behind interfaces

## Breaking Changes

### 1. Module Paths: `/v3` → `/v4`

All import paths change:

```diff
- import "github.com/larsartmann/cqrs-htmx/v3"
+ import "github.com/larsartmann/cqrs-htmx/v4"

- import "github.com/larsartmann/cqrs-htmx/usermgmt/v3"
+ import "github.com/larsartmann/cqrs-htmx/usermgmt/v4"

- import "github.com/larsartmann/cqrs-htmx/adminui/v3"
+ import "github.com/larsartmann/cqrs-htmx/adminui/v4"
```

Run `go mod edit -module` on your module, then find-replace import paths.

### 2. TOTP Configuration (DONE)

The `TOTPConfig` field on `ServiceConfig` is replaced by the `TOTP` interface field.

**Before (v3):**

```go
import "github.com/larsartmann/cqrs-htmx/usermgmt/v3"

svc, _ := usermgmt.NewService(usermgmt.ServiceConfig{
    TOTPConfig: &usermgmt.TOTPConfig{
        Issuer: "MyApp",
        Window: 1,
    },
})
```

**After (v4):**

```go
import (
    "github.com/larsartmann/cqrs-htmx/usermgmt/v4"
    usermgmttotp "github.com/larsartmann/cqrs-htmx/usermgmt/totp/v4"
)

svc, _ := usermgmt.NewService(usermgmt.ServiceConfig{
    TOTP: usermgmttotp.New(usermgmttotp.Config{
        Issuer: "MyApp",
        Window: 1,
    }),
})
```

**What changed:**

- `pquerna/otp` is no longer a dependency of core `usermgmt`
- Consumers who don't need TOTP skip the `usermgmt/totp` import entirely
- `TOTPConfig` type removed from core (now `totp.Config` in the totp module)
- `TOTPVerifier` interface removed (was a ghost — never wired)
- New `TOTPProvider` interface in core (crypto-only: `GenerateSecret`, `ValidateCode`)
- `TOTPSetupResponse` stays in core (return type of `Service.EnableTOTP`)

### 3. WebAuthn Configuration (DONE)

The `WebAuthnConfig` field on `ServiceConfig` is replaced by the `WebAuthn` interface field.

**Before (v3):**

```go
import "github.com/larsartmann/cqrs-htmx/usermgmt/v3"

svc, _ := usermgmt.NewService(usermgmt.ServiceConfig{
    WebAuthnConfig: &usermgmt.WebAuthnConfig{
        RPID:          "example.com",
        RPDisplayName: "My App",
        RPOrigins:     []string{"https://example.com"},
    },
})
```

**After (v4):**

```go
import (
    "github.com/larsartmann/cqrs-htmx/usermgmt/v4"
    usermgmtwa "github.com/larsartmann/cqrs-htmx/usermgmt/webauthn/v4"
)

wa, _ := usermgmtwa.New(usermgmtwa.Config{
    RPID:          "example.com",
    RPDisplayName: "My App",
    RPOrigins:     []string{"https://example.com"},
})
svc, _ := usermgmt.NewService(usermgmt.ServiceConfig{
    WebAuthn: wa,
})
```

**What changed:**

- `go-webauthn/webauthn` is no longer a dependency of core `usermgmt`
- Consumers who don't need passkeys skip the `usermgmt/webauthn` import entirely
- `WebAuthnConfig` type removed from core (now `webauthn.Config` in the webauthn module)
- New `WebAuthnProvider` interface in core (ceremony delegation via `[]byte` JSON)
- `WebAuthnSessionStore` interface changed: `*webauthn.SessionData` → `[]byte`
- `BeginRegistrationResponse.Options` and `BeginLoginResponse.Options` changed to `json.RawMessage`
  (transparent to consumers — same wire format)
- In-memory session store uses TTL-based eviction (5 min default) instead of go-webauthn's `Expires` field

### 4. OAuth2 Configuration (DONE)

The `OAuth2Config` field on `ServiceConfig` is replaced by the `OAuth2` interface field.

**Before (v3):**

```go
import "github.com/larsartmann/cqrs-htmx/usermgmt/v3"

svc, _ := usermgmt.NewService(usermgmt.ServiceConfig{
    OAuth2Config: &usermgmt.OAuth2Config{
        Providers: map[string]usermgmt.OAuth2ProviderConfig{
            "google": {ClientID: "...", ClientSecret: "...", RedirectURL: "...", IssuerURL: "https://accounts.google.com"},
        },
    },
})
```

**After (v4):**

```go
import (
    "context"

    "github.com/larsartmann/cqrs-htmx/usermgmt/v4"
    usermgmtoauth2 "github.com/larsartmann/cqrs-htmx/usermgmt/oauth2/v4"
)

oauthProv, _ := usermgmtoauth2.New(context.Background(), usermgmtoauth2.Config{
    Providers: map[string]usermgmtoauth2.ProviderConfig{
        "google": {ClientID: "...", ClientSecret: "...", RedirectURL: "...", IssuerURL: "https://accounts.google.com"},
    },
})
svc, _ := usermgmt.NewService(usermgmt.ServiceConfig{
    OAuth2: oauthProv,
})
```

**What changed:**

- `golang.org/x/oauth2`, `coreos/go-oidc`, `go-jose` are no longer dependencies of core `usermgmt`
- Consumers who don't need OAuth2 skip the `usermgmt/oauth2` import entirely
- `OAuth2Config` and `OAuth2ProviderConfig` types removed from core (now `oauth2.Config` and `oauth2.ProviderConfig`)
- New `OAuth2Provider` interface in core (PKCE generation + token exchange delegation)
- `OAuth2UserInfo` exported in core (Service deserializes provider's JSON return)
- `OAuth2StateStore.Save` signature changed: now accepts pre-generated state token
  (`Save(state, provider, pkceVerifier, ttl) error` instead of `Save(provider, pkceVerifier, ttl) (state, error)`)
- OIDC discovery happens at `oauth2.New()` time (moved from Service.initOAuth2 to provider constructor)

## Migration Steps

1. Update all import paths `/v3` → `/v4`
2. If using TOTP: add `usermgmt/totp/v4` dependency, update `ServiceConfig`
3. If using WebAuthn: add `usermgmt/webauthn/v4` dependency, construct `webauthn.Provider`, inject as `ServiceConfig.WebAuthn`
4. If using OAuth2: add `usermgmt/oauth2/v4` dependency, construct `oauth2.Provider`, inject as `ServiceConfig.OAuth2`
5. If using a custom `OAuth2StateStore`: update `Save` signature
6. If using a custom `WebAuthnSessionStore`: update to `[]byte` data type
7. Run `go mod tidy`

## Dependency Impact

| Dependency  | v3 (core usermgmt) | v4 (core usermgmt) | v4 (sub-module only) |
| ----------- | ------------------ | ------------------ | -------------------- |
| pquerna/otp | ✅ direct          | ❌ removed         | ✅ `usermgmt/totp`   |
| go-webauthn | ✅ direct          | ❌ removed         | ✅ `usermgmt/webauthn` |
| oauth2      | ✅ direct          | ❌ removed         | ✅ `usermgmt/oauth2` |
| go-oidc     | ✅ direct          | ❌ removed         | ✅ `usermgmt/oauth2` |
| go-jose     | transitive         | ❌ removed         | ✅ `usermgmt/oauth2` (transitive) |

Core `usermgmt` has **ZERO** auth-strategy dependencies. Consumers import only the auth strategies they need.
