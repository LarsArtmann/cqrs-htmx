# usermgmt — Event-Sourced User Management for cqrs-htmx

## Description

Passwordless, event-sourced user management with WebAuthn/Passkey authentication,
RBAC authorization via Casbin, and opt-in event signing/encryption.

## Installation

```bash
go get github.com/larsartmann/cqrs-htmx/usermgmt/v4
```

## Event Signing & Encryption (Opt-in)

Signing and encryption are **zero-dependency by default** — consumers who don't
need crypto pull nothing extra. To enable, import `go-cqrs-lite/signing/v4` and/or
`go-cqrs-lite/encryption/v4` and wire them via `ServiceConfig`:

### Encryption-at-rest + bus-level signing (recommended)

```go
import (
    "github.com/larsartmann/go-cqrs-lite/encryption/v4"
    "github.com/larsartmann/go-cqrs-lite/signing/v4"
)

signer, _ := signing.NewHMAC(signingKey)       // 32+ bytes
cipher, _ := encryption.NewAES256GCM(encKey)   // exactly 32 bytes

svc, _ := usermgmt.NewService(usermgmt.ServiceConfig{
    StoreWrapper: func(s event.Store) (event.Store, error) {
        return encryption.NewEncryptedStore(s, cipher)
    },
    PublishMiddleware: []event.PublishMiddleware{signing.SignMiddleware(signer)},
    HandlerMiddleware: []event.Middleware{signing.RequireSignatureMiddleware(signer)},
})
```

### Asymmetric signing (Ed25519)

```go
pubKey, privKey, _ := signing.GenerateEd25519KeyPair()
signer, _ := signing.NewEd25519(privKey)
verifier, _ := signing.NewEd25519Verifier(pubKey)

svc, _ := usermgmt.NewService(usermgmt.ServiceConfig{
    PublishMiddleware: []event.PublishMiddleware{signing.SignMiddleware(signer)},
    HandlerMiddleware: []event.Middleware{signing.RequireSignatureMiddleware(verifier)},
})
```

See `docs/adr/0011-event-signing-encryption.md` for the full decision record.

## HTTP route posture

`AuthHandler.RegisterRoutes` (and the `Register*Routes` variants) enforce the
auth posture themselves — no external middleware required:

- **Public:** `POST /auth/register` (sets the session cookie), the WebAuthn
  _login_ ceremonies, `POST /auth/email/verify` (token is the credential),
  OAuth2 begin/callback, and logout.
- **Owner-session-gated:** the WebAuthn _enrollment_ ceremonies answer 401
  without a session and 403 when the target `user_id` is not the session user.
- **Session-required, self-wrapped:** `/auth/me`, credentials, TOTP,
  email-verify/send, import/export, and OAuth2 unlink run behind an enrich-only
  session pass — reachable with a valid cookie even on a bare mount, 401
  without one.

Full route table: the `RegisterRoutes` godoc and the root README. Decision
record: `docs/adr/0055-owner-session-gated-credential-ceremonies.md`.

## Published language (25 events / 21 commands)

The identity domain's events (`identity-model` constants; `DefaultEventCatalog()`
serves every schema — see `docs/guides/event-catalog-guide.md`):

| Aggregate | Events |
| --- | --- |
| User | `UserRegistered`, `RolesUpdated`, `EmailChanged`, `DisplayNameChanged`, `UserDeleted`, `EmailVerified`, `TOTPEnabled`, `TOTPDisabled`, `ExternalAccountLinked`, `ExternalAccountUnlinked` |
| User credential | `CredentialAdded`, `CredentialRemoved`, **`CredentialUsed`** (assertion state after each login: signature counter + UP/UV/backup flags — the clone-detection high-water mark, identity-model v4.14.0+) |
| Auth audit | `LoginSucceeded`, `LoginFailed`, `RegistrationClosed` (own Auth-typed streams, best-effort append) |
| Membership | `MemberAdded`, `MemberRolesChanged`, `MemberRemoved` |
| Tenant | `TenantCreated`, `TenantSuspended`, `TenantReactivated`, `TenantDeleted` |
| Bot | `BotRegistered`, `BotDeleted` |

Commands (21): `RegisterUser`, `ChangeEmail`, `ChangeDisplayName`, `DeleteUser`,
`AddCredential`, `RemoveCredential`, `RecordCredentialUse`, `VerifyEmail`,
`EnableTOTP`, `DisableTOTP`, `LinkExternalAccount`, `UnlinkExternalAccount`,
`AddMember`, `UpdateMemberRoles`, `RemoveMember`, `CreateTenant`,
`SuspendTenant`, `ReactivateTenant`, `DeleteTenant`, `RegisterBot`, `DeleteBot`.
All 21 are registered both on the usermgmt dispatcher and in
`systemadapter.DomainConfig()` for `system.New()` consumers.

## Development

```bash
# Build the project
nix run .#build

# Run tests
nix run .#test

# Run linter
nix run .#lint
```

## License

See [LICENSE](LICENSE) file for details.
