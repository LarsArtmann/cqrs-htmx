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

## Projection readiness (health checks)

`*Service` implements `ProjectionStatuses()` — the worker state of every
registered projection (`idle`, `running`, `backoff`, `draining`, `live`,
`stopped`, `failed`). Wire it into a readiness endpoint so a projection
still replaying its journal, or one that exhausted its restart budget,
fails the probe instead of quietly serving a degraded read model:

```go
mux.Handle("GET /readyz", cqrshtmx.ReadinessHandler(
	cqrshtmx.ProjectionReadinessCheck(svc),
))
```

`ServiceConfig.OnProjectionFailed` is the log-side twin: the func is
called when a worker exhausts its restart budget. Neither is a
constructor error by design — startup drain failures surface through
`ProjectionStatuses()` and this hook, never from `NewService`.

## Published language (25 events / 21 commands)

The identity domain's events (`identity-model` constants; `DefaultEventCatalog()`
serves every schema — see `docs/guides/event-catalog-guide.md`):

| Aggregate       | Events                                                                                                                                                                                                 |
| --------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| User            | `UserRegistered`, `RolesUpdated`, `EmailChanged`, `DisplayNameChanged`, `UserDeleted`, `EmailVerified`, `TOTPEnabled`, `TOTPDisabled`, `ExternalAccountLinked`, `ExternalAccountUnlinked`              |
| User credential | `CredentialAdded`, `CredentialRemoved`, **`CredentialUsed`** (assertion state after each login: signature counter + UP/UV/backup flags — the clone-detection high-water mark, identity-model v4.14.0+) |
| Auth audit      | `LoginSucceeded`, `LoginFailed`, `RegistrationClosed` (own Auth-typed streams, best-effort append)                                                                                                     |
| Membership      | `MemberAdded`, `MemberRolesChanged`, `MemberRemoved`                                                                                                                                                   |
| Tenant          | `TenantCreated`, `TenantSuspended`, `TenantReactivated`, `TenantDeleted`                                                                                                                               |
| Bot             | `BotRegistered`, `BotDeleted`                                                                                                                                                                          |

Commands (21): `RegisterUser`, `ChangeEmail`, `ChangeDisplayName`, `DeleteUser`,
`AddCredential`, `RemoveCredential`, `RecordCredentialUse`, `VerifyEmail`,
`EnableTOTP`, `DisableTOTP`, `LinkExternalAccount`, `UnlinkExternalAccount`,
`AddMember`, `UpdateMemberRoles`, `RemoveMember`, `CreateTenant`,
`SuspendTenant`, `ReactivateTenant`, `DeleteTenant`, `RegisterBot`, `DeleteBot`.
All 21 are registered both on the usermgmt dispatcher and in
`systemadapter.DomainConfig()` for `system.New()` consumers.

## Consuming domain events

There is no callback API — the legacy `EventHandler` was removed. Consumers
subscribe to the service's event bus directly (imports:
`github.com/larsartmann/go-cqrs-lite/event/v4` as `event`,
`github.com/larsartmann/cqrs-htmx/identity-model/v4` as `identitymodel`):

```go
svc, _ := usermgmt.NewService(usermgmt.ServiceConfig{ /* ... */ })
defer svc.Close()

_ = svc.EventBus().Subscribe(identitymodel.EventCredentialUsed,
    func(_ context.Context, evt event.Event) error {
        var p identitymodel.CredentialUsedPayload
        _ = json.Unmarshal(evt.Payload(), &p) // SignCount, UserPresent, UserVerified...
        return nil
    })

// or receive every identity event:
_ = svc.EventBus().SubscribeAll(func(_ context.Context, evt event.Event) error {
    logger.Info("identity event", "type", evt.Type(), "stream", evt.StreamID())
    return nil
})
```

- **Where events come from:** every dispatched command appends its events to
  the journal and publishes them on the bus (`Service.EventBus()`; bring your
  own via `ServiceConfig.EventBus`, default is watermill-backed). Handler
  ordering across subscribers is not guaranteed — treat handlers as
  projections, not orchestrations.
- **Payload types:** every event's payload struct is re-exported from this
  package (`usermgmt.CredentialUsedPayload`, `usermgmt.UserDeletedPayload`, …)
  and aliased 1:1 to `identity-model`, the domain source of truth. The full
  schema set is served by `DefaultEventCatalog()`.
- **Replay instead of live-only:** for rebuildable state prefer projecting
  from the journal (`Service.Journal()`, or the declarative
  `systemadapter` path) — the bus is at-least-once and in-memory by default;
  the journal is the durable record.
- **Relationship to the audit trail:** the built-in audit log is itself a
  projection of these events (see `Service.AuditLog()`); you are consuming
  the same published language it folds. Auth-audit events (`LoginSucceeded`,
  `LoginFailed`, `RegistrationClosed`) ride their own Auth-typed streams and
  are appended best-effort. The full actor/causation/correlation model —
  auto-wired by `NewService` onto every dispatch — is the
  [actor-and-audit-trail guide](../docs/guides/actor-and-audit-trail.md).

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
