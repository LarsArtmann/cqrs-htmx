# Domain Language

A **Ubiquitous Language** for `cqrs-htmx` — shared across Customer, Product Owner, Developer, and AI.
Inspired by Domain-Driven Design (DDD).

Every term below should mean the **same thing** to everyone who reads it.

## Glossary

| Term              | Definition                                                                                 | Context                       |
| ----------------- | ------------------------------------------------------------------------------------------ | ----------------------------- |
| AccountLockout    | Brute-force protection that blocks authentication after N failed attempts                  | Authentication                |
| Actor             | A kind-discriminated identity: either a User or a Bot (used in authorization + audit)      | Identity model (ADR 0015)     |
| ActorID           | A value identifying an Actor — carries kind (user/bot) + raw ID string                     | Identity model                |
| Aggregate         | A cluster of domain objects treated as a single unit for data consistency                  | Event-sourced CQRS            |
| AuditLog          | A projection that records all user events as queryable audit entries                       | Compliance / Security         |
| Authz             | Authorization engine wrapping Casbin RBAC with domains                                     | usermgmt module               |
| Bot               | A non-human actor with an API token, registered to an owner for automated access           | Identity model (ADR 0015)     |
| BotID             | A branded string uniquely identifying a bot                                                | Identity model                |
| Casbin            | External authorization library providing RBAC with domain support                          | Authorization                 |
| CasbinProjection  | A projection that derives all Casbin policies from user events                             | Event sourcing                |
| Ceremony          | A WebAuthn protocol flow (registration or login) split into a begin + finish exchange      | Authentication                |
| Command           | An intent to change state, dispatched to a handler that produces events                    | CQRS write side               |
| Credential        | A WebAuthn passkey registered to a user for passwordless authentication                    | Authentication                |
| CQRS              | Command Query Responsibility Segregation — separate write and read models                  | Architecture pattern          |
| Decider           | Pure function that validates a command against current state and emits events              | Event sourcing                |
| EmailVerification | Token-based confirmation that a user controls an email address (single-use, TTL)           | Authentication                |
| Enforcer          | Interface satisfied by Casbin; the authorization decision point                            | Authorization                 |
| Event             | An immutable record of something that happened in the past                                 | Event sourcing                |
| Event Store       | Append-only persistence for events on an aggregate stream                                  | Event sourcing                |
| ExternalAccount   | An OAuth2/OIDC provider account (provider + subject) linked to a User                      | OAuth2 integration (ADR 0014) |
| Fold              | Pure function that reconstructs aggregate state by replaying events                        | Event sourcing                |
| foldUser          | The concrete fold function that reconstructs `UserState` from the User event stream        | Event sourcing                |
| HandlerOption     | Functional option pattern for configuring CQRS HTTP handlers                               | cqrs-htmx root module         |
| HTMX              | HTML-over-the-wire library for dynamic web pages without writing JavaScript                | Frontend                      |
| Impersonation     | A SuperAdmin acting on behalf of another user with an auditable session origin             | Identity model (ADR 0015)     |
| MemoryBus         | In-memory event bus that blocks publishers until handlers complete (read-your-writes)      | Event sourcing                |
| Membership        | A grant of roles to an Actor within a Tenant — the RBAC link between actor and tenant      | Identity model (ADR 0015)     |
| Passkey           | A WebAuthn credential (FIDO2) bound to a device, enabling passwordless login               | Authentication                |
| Projection        | A read model built by subscribing to events and updating a materialized view               | CQRS read side                |
| Read Model        | A query-optimized view of current state, derived from the event stream                     | CQRS read side                |
| Role              | A named permission group (SuperAdmin, Admin, User, Viewer, Owner) assigned within a domain | RBAC authorization            |
| Service           | The application service orchestrating commands, queries, and session management            | usermgmt module               |
| Session           | An ephemeral authentication artifact (token + expiry) created after login                  | Authentication                |
| SessionOrigin     | The cause of a session: DirectLogin, Impersonation, or OAuth2                              | Authentication                |
| SQLEventStore     | Persistent `event.Store` for PostgreSQL, SQLite, and MySQL with optimistic concurrency     | Persistence                   |
| Templ             | Go HTML templating engine with type-safe compile-checked templates                         | Frontend                      |
| Tenant            | An organizational boundary for multi-tenancy — contains members with roles                 | Identity model (ADR 0015)     |
| TenantID          | A branded string uniquely identifying a tenant                                             | Identity model                |
| Tombstone         | A soft-delete marker event signaling an aggregate is logically deleted                     | Event sourcing                |
| TOTP              | Time-based One-Time Password (RFC 6238) — a 6-digit second-factor code                     | Multi-factor auth             |
| UserID            | A branded ULID type uniquely identifying a user                                            | usermgmt module               |
| WebAuthn          | W3C standard for passwordless authentication using passkeys/FIDO2                          | Authentication                |

## Entities

Objects with identity and lifecycle.

| Term               | Definition                                                                    | Context                              |
| ------------------ | ----------------------------------------------------------------------------- | ------------------------------------ |
| User               | A registered person with email, roles, credentials, and display name          | User aggregate root                  |
| Tenant             | An organizational unit for multi-tenancy with create/suspend/delete lifecycle | Tenant aggregate root (ADR 0015)     |
| Bot                | A non-human actor with API token, owner, and scopes                           | Bot aggregate root (ADR 0015)        |
| Membership         | A role grant linking an Actor to a Tenant                                     | Membership aggregate root (ADR 0015) |
| WebAuthnCredential | A registered passkey with public key, transports, and backup state            | Part of User aggregate               |

## Value Objects

Immutable objects defined by attributes.

| Term              | Definition                                                | Context        |
| ----------------- | --------------------------------------------------------- | -------------- |
| UserID            | Branded string uniquely identifying a user                | Identity       |
| Email             | Validated email address (`ParseEmail`/`MustParseEmail`)   | Identity       |
| Session           | Token + expiry + user ID, immutable after creation        | Authentication |
| Role              | Named permission group string (admin, user, viewer)       | RBAC           |
| Policy            | An RBAC rule: subject + domain + object + action + effect | Authorization  |
| GroupPolicy       | A role assignment: subject + role + domain                | Authorization  |
| VerificationToken | Single-use secret proving email ownership, with a TTL     | Authentication |

## Events

Immutable records of domain changes.

| Event              | What Happened                                         | Aggregate |
| ------------------ | ----------------------------------------------------- | --------- |
| UserRegistered     | A new user account was created with email and roles   | User      |
| RolesUpdated       | The user's roles were changed in a specific domain    | User      |
| EmailChanged       | The user's email address was changed                  | User      |
| EmailVerified      | The user proved ownership of their email via token    | User      |
| DisplayNameChanged | The user's display name was changed                   | User      |
| UserDeleted        | The user was deleted (tombstone — no further changes) | User      |
| CredentialAdded    | A WebAuthn credential was registered to the user      | User      |
| CredentialRemoved  | A WebAuthn credential was removed from the user       | User      |
| TOTPEnabled        | TOTP multi-factor authentication was activated        | User      |
| TOTPDisabled       | TOTP multi-factor authentication was deactivated      | User      |

## Commands

Intents that trigger state changes.

| Command           | What It Does                                        | Events Produced    |
| ----------------- | --------------------------------------------------- | ------------------ |
| RegisterUser      | Create a new user account (email only, no password) | UserRegistered     |
| UpdateRoles       | Change the user's role assignments in a domain      | RolesUpdated       |
| ChangeEmail       | Change the user's email address                     | EmailChanged       |
| VerifyEmail       | Confirm email ownership with a verification token   | EmailVerified      |
| ChangeDisplayName | Change the user's display name                      | DisplayNameChanged |
| DeleteUser        | Soft-delete the user (tombstone)                    | UserDeleted        |
| AddCredential     | Register a WebAuthn passkey to the user             | CredentialAdded    |
| RemoveCredential  | Remove a WebAuthn passkey from the user             | CredentialRemoved  |
| EnableTOTP        | Activate TOTP multi-factor authentication           | TOTPEnabled        |
| DisableTOTP       | Deactivate TOTP (requires a valid code)             | TOTPDisabled       |

## Bounded Contexts

| Context         | Responsibility                                          | Module           |
| --------------- | ------------------------------------------------------- | ---------------- |
| User Management | User lifecycle, authentication, authorization, sessions | usermgmt         |
| CQRS-HTMX Core  | HTTP handler integration, response building, middleware | root module      |
| Authorization   | RBAC policy management, enforcement, role hierarchy     | usermgmt (Authz) |
| MFA             | TOTP second-factor enrollment and verification          | usermgmt         |
| Auditing        | Immutable record of user actions derived from events    | usermgmt         |
