# identity-model

> Pure Go domain model for event-sourced identity management.

Zero infrastructure dependencies. No HTTP frameworks, no SQL drivers, no web
servers. Casbin is the sole non-primitive dependency (first-class, ADR-0044 —
the Authz engine is domain logic).

## What This Is

A standalone Go module containing **only types** — the domain model for event-sourced
identity and access management:

> **When to import this directly:** you normally don't. `usermgmt` re-exports
> the payloads, commands, and folds, and that is the surface applications consume.
> Import `identity-model` directly when you are building a sibling adapter
> (another auth strategy, a projection sink, a codemod) or folding journals
> without the service.

- **Identifiers**: UserID, TenantID, BotID, ActorID (kind-discriminated)
- **Value Objects**: Email (validated/normalized), WebAuthnCredential, ExternalAccount
- **Aggregate States**: UserState, MembershipState, TenantState, BotState
- **Event Payloads**: All domain event payloads (UserRegistered, MemberAdded, etc.)
- **Commands**: All command structs with constructors (RegisterUser, AddMember, etc.)
- **Auth Types**: Role, Action, Effect, Policy, GroupPolicy, RBAC model config
- **Sessions**: Session, SessionOrigin, DirectLogin, Impersonation
- **Errors**: Domain error sentinels (classified, no HTTP status coupling)
- **Interfaces**: TOTPProvider, WebAuthnProvider, OAuth2Provider, store contracts
- **Fold Functions**: Pure event-sourcing state reconstruction (FoldUser, FoldTenant, etc.)
- **Crypto**: Token generation, HMAC hashing, constant-time verification

## Module Path

```
github.com/larsartmann/cqrs-htmx/identity-model/v4
```

## Usage

```go
import identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"

// Create a user ID
uid := identitymodel.GenerateUserID()

// Validate an email
email, err := identitymodel.ParseEmail("alice@example.com")

// Check a role
if identitymodel.RoleAdmin.Valid() {
    // ...
}

// Create a session
session, err := identitymodel.NewSession(uid, 24*time.Hour)
```

## Dependencies

Only Go ecosystem libraries for CQRS/event-sourcing primitives:

| Dependency              | Purpose                         |
| ----------------------- | ------------------------------- |
| go-cqrs-lite/id/v4      | UserID type                     |
| go-cqrs-lite/event/v4   | Event types, classification     |
| go-cqrs-lite/command/v4 | Command types                   |
| go-cqrs-lite/codec/v4   | Payload encoding/decoding       |
| go-branded-id           | Branded types (TenantID, BotID) |
| go-error-family         | Error classification            |
| oklog/ulid/v2           | ULID parsing                    |

**Not imported**: modernc.org/sqlite, database/sql, net/http, cqrs-htmx root.

**Imported by design**: casbin/v3 — the Authz engine is part of the domain
model (ADR-0044), not an infrastructure afterthought.

## Design Decisions

- **Errors without HTTP status**: Domain errors use `errorfamily` classification only.
  HTTP status mapping is the infrastructure layer's concern.
- **Exported embedding cores**: `CredentialCore` and `ExternalAccountCore` are
  embedded in their exported wrappers; their fields are promoted and accessible
  through `WebAuthnCredential`/`ExternalAccount`.
  Use `NewCredentialFromPayload` and `NewExternalAccount` for construction.
- **Event fold functions are pure**: `FoldUser`, `FoldMembership`, `FoldTenant`,
  `FoldBot` are pure functions with no side effects — perfect for testing and
  event-sourcing replay.
