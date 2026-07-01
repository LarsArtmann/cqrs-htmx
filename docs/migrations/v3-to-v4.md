# v3 → v4 Migration Guide

> **Status:** In progress — TOTP extraction complete, WebAuthn and OAuth2 pending

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

### 3. WebAuthn Configuration (PLANNED — not yet implemented)

Will follow the same pattern: `WebAuthnConfig` → `WebAuthn` interface field, with
implementation in `usermgmt/webauthn/v4`.

### 4. OAuth2 Configuration (PLANNED — not yet implemented)

Will follow the same pattern: `OAuth2Config` → `OAuth2` interface field, with
implementation in `usermgmt/oauth2/v4`.

## Migration Steps

1. Update all import paths `/v3` → `/v4`
2. If using TOTP: add `usermgmt/totp/v4` dependency, update `ServiceConfig`
3. If using WebAuthn: (no change yet — still uses `WebAuthnConfig`)
4. If using OAuth2: (no change yet — still uses `OAuth2Config`)
5. Run `go mod tidy`

## Dependency Impact

| Dependency | v3 (core usermgmt) | v4 (core usermgmt) | v4 (totp module only) |
|---|---|---|---|
| pquerna/otp | ✅ direct | ❌ removed | ✅ direct |
| go-webauthn | ✅ direct | ✅ direct (pending) | N/A |
| oauth2 | ✅ direct | ✅ direct (pending) | N/A |
| go-oidc | ✅ direct | ✅ direct (pending) | N/A |

After all three extractions, core `usermgmt` will have ZERO auth-strategy dependencies.
