# Domain Language

A **Ubiquitous Language** for `cqrs-htmx` — shared across Customer, Product Owner, Developer, and AI.
Inspired by Domain-Driven Design (DDD).

Every term below should mean the **same thing** to everyone who reads it.

## Glossary

| Term               | Definition                                                                                                                                                 | Context                                   |
| ------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------- |
| AccountLockout     | Brute-force protection that blocks authentication after N failed attempts                                                                                  | Authentication                            |
| Actor              | A kind-discriminated identity: User, Bot, System, Service, or Unknown (used in authorization + audit)                                                      | Identity model (ADR-0111)                 |
| ActorID            | A value identifying an Actor — carries kind (user/bot/system/service/unknown) + raw ID string                                                              | Identity model (ADR-0111)                 |
| Aggregate          | A cluster of domain objects treated as a single unit for data consistency                                                                                  | Event-sourced CQRS                        |
| AuditLog           | A projection that records all user events as queryable audit entries                                                                                       | Compliance / Security                     |
| Authz              | Authorization engine wrapping Casbin RBAC with domains                                                                                                     | usermgmt module                           |
| Bot                | A non-human actor with an API token, registered to an owner for automated access                                                                           | Identity model (ADR 0015)                 |
| BotID              | A branded string uniquely identifying a bot                                                                                                                | Identity model                            |
| Broadcaster        | The SSE fan-out hub (`*sse.Broadcaster[sse.Event]`); `Hub()` is the canonical shareable object; datastar re-exports the upstream go-datastar/broadcast hub | cqrs-htmx root + datastar                 |
| Casbin             | External authorization library providing RBAC with domain support                                                                                          | Authorization                             |
| CasbinProjection   | A projection that derives all Casbin policies from user events                                                                                             | Event sourcing                            |
| Ceremony           | A WebAuthn protocol flow (registration or login) split into a begin + finish exchange                                                                      | Authentication                            |
| Command            | An intent to change state, dispatched to a handler that produces events                                                                                    | CQRS write side                           |
| Credential         | A WebAuthn passkey registered to a user for passwordless authentication                                                                                    | Authentication                            |
| CQRS               | Command Query Responsibility Segregation — separate write and read models                                                                                  | Architecture pattern                      |
| Decider            | Pure function that validates a command against current state and emits events                                                                              | Event sourcing                            |
| DashboardUI        | Ready-made CQRS/ES observability dashboard module — event browser, projection health, time-travel inspector, SSE live updates                              | dashboardui module                        |
| EmailVerification  | Token-based confirmation that a user controls an email address (single-use, TTL)                                                                           | Authentication                            |
| Enforcer           | Interface satisfied by Casbin; the authorization decision point                                                                                            | Authorization                             |
| Event              | An immutable record of something that happened in the past                                                                                                 | Event sourcing                            |
| Event Store        | Append-only persistence for events on an aggregate stream                                                                                                  | Event sourcing                            |
| EventCatalog       | Immutable JSON catalog of event schemas (type, description, payload fields); served as Published Language for projection builders                          | cqrs-htmx root module                     |
| ExternalAccount    | An OAuth2/OIDC provider account (provider + subject) linked to a User                                                                                      | OAuth2 integration (ADR 0014)             |
| Fold               | Pure function that reconstructs aggregate state by replaying events                                                                                        | Event sourcing                            |
| FoldUser           | The concrete fold function that reconstructs `UserState` from the User event stream (siblings: `FoldMembership`, `FoldTenant`, `FoldBot`)                  | Event sourcing                            |
| HandlerOption      | Functional option pattern for configuring CQRS HTTP handlers                                                                                               | cqrs-htmx root module                     |
| HTMX               | HTML-over-the-wire library for dynamic web pages without writing JavaScript                                                                                | Frontend                                  |
| HTMXRedirect       | Helper that emits an HTMX-aware redirect (`HX-Redirect` for HTMX clients, standard `http.Redirect` otherwise)                                              | cqrs-htmx root module                     |
| Impersonation      | A SuperAdmin acting on behalf of another user with an auditable session origin                                                                             | Identity model (ADR 0015)                 |
| EventBus           | The event pub/sub backend (`event.Bus`; default `watermill.EventBus`) that blocks publishers until handlers complete (read-your-writes)                    | Event sourcing                            |
| JournalSSEStore    | SSE replay store that reads missed events from a journal on reconnect, using `Last-Event-ID` as a cursor                                                   | cqrs-htmx root module                     |
| Membership         | A grant of roles to an Actor within a Tenant — the RBAC link between actor and tenant                                                                      | Identity model (ADR 0015)                 |
| Passkey            | A WebAuthn credential (FIDO2) bound to a device, enabling passwordless login                                                                               | Authentication                            |
| Projection         | A read model built by subscribing to events and updating a materialized view                                                                               | CQRS read side                            |
| ProjectionStatus   | Live health snapshot of a projection (name, state, lag, last-event, error); served as JSON via `ProjectionStatusHandler`                                   | cqrs-htmx root module                     |
| Read Model         | A query-optimized view of current state, derived from the event stream                                                                                     | CQRS read side                            |
| Role               | A named permission group (SuperAdmin, Admin, User, Viewer, Owner) assigned within a domain                                                                 | RBAC authorization                        |
| SafeRedirectPath   | Normalizes untrusted redirect paths to site-relative URLs as an open-redirect guard                                                                        | cqrs-htmx root module                     |
| SSE Reconnect      | Browser reconnects to the SSE endpoint; the server replays missed events from the journal using `Last-Event-ID`                                            | dashboardui module                        |
| Service            | The application service orchestrating commands, queries, and session management                                                                            | usermgmt module                           |
| Session            | An ephemeral authentication artifact (token + expiry) created after login                                                                                  | Authentication                            |
| SessionOrigin      | The cause of a session: a `DirectLogin` or `Impersonation` value (a sealed interface, not an enum)                                                         | Authentication                            |
| SnapshotConfig     | Opt-in configuration for aggregate snapshotting (Store + Codec + Strategy); zero-value = full-replay mode                                                  | Event sourcing (ADR 0041)                 |
| SQLEventStore      | Persistent `event.Store` for PostgreSQL, SQLite, and MySQL with optimistic concurrency                                                                     | Persistence                               |
| Scoped Feed        | A shared `/sse` stream filtered by a predicate (`setup.Config.SSEFilter`) — scopes live events AND journal replay, fail-closed                             | setup / transport (v4.12.0)               |
| Templ              | Go HTML templating engine with type-safe compile-checked templates                                                                                         | Frontend                                  |
| Tenant             | An organizational boundary for multi-tenancy — contains members with roles                                                                                 | Identity model (ADR 0015)                 |
| TenantID           | A branded string uniquely identifying a tenant                                                                                                             | Identity model                            |
| Tombstone          | A soft-delete marker event signaling an aggregate is logically deleted                                                                                     | Event sourcing                            |
| TOTP               | Time-based One-Time Password (RFC 6238) — a 6-digit second-factor code                                                                                     | Multi-factor auth                         |
| TOTPProvider       | Interface for TOTP secret generation + code validation; implemented by usermgmt/totp/v4                                                                    | Auth strategy (ADR 0035)                  |
| UserID             | A branded string type uniquely identifying a user                                                                                                          | identity-model (aliased in root/usermgmt) |
| WebAuthn           | W3C standard for passwordless authentication using passkeys/FIDO2                                                                                          | Authentication                            |
| WebAuthnProvider   | Interface for WebAuthn ceremony delegation via []byte JSON; implemented by usermgmt/webauthn/v4                                                            | Auth strategy (ADR 0035)                  |
| WebAuthnSessionTTL | Configurable TTL for WebAuthn challenge sessions (ServiceConfig.WebAuthnSessionTTL)                                                                        | Authentication                            |
| OAuth2Provider     | Interface for OAuth2/OIDC login flow (PKCE + token exchange); implemented by usermgmt/oauth2/v4                                                            | Auth strategy (ADR 0035)                  |
| LegalParty         | A party with legal personality — registered name, address, legal form, jurisdiction, tax identifiers — that contracts bind and tax documents address; closed but never deleted (retention) | legalparty-model (issue #45)              |
| LegalFormClass     | The five-value personality classifier: natural_person, legal_entity, partnership, public_body, non_profit — peers, never a bool; decides register-number and document requirements | legalparty-model                          |
| LegalFormCode      | The ISO 20275 ELF code when a registry speaks one (GLEIF carries it); natural persons carry none and use OtherLegalForm instead                            | legalparty-model                          |
| LEI                | Legal Entity Identifier (ISO 17442): 20-character, mod-97-checksummed global ID for legal entities; natural persons cannot hold one                        | legalparty-model                          |
| RegisterRef        | Opaque pointer to a commercial/public register entry (registry code + number), e.g. an OpenCorporates jurisdiction code plus company number                | legalparty-model                          |
| VATID              | Value-added tax identifier in canonical prefixed EU form (DE validates the ISO 7064 Mod 11,10 checksum); deliberately a different type from TaxNumber      | legalparty-model                          |
| TaxNumber          | National direct-tax number (Steuernummer, UTR); carries no country prefix, so a prefixed VAT shape is rejected instead of silently accepted                | legalparty-model                          |
| RegistryFacts      | Provenance-stamped snapshot of what an external registry says about a party; applied to the aggregate only through an explicit profile update, never silently | legalparty-model                        |
| LEIResolver        | The registry port resolving a LEI into RegistryFacts; implemented by gleif (api.gleif.org, unauthenticated, CC BY 4.0 attribution)                         | legalparty-model                          |
| RegisterResolver   | The registry port resolving a RegisterRef into RegistryFacts; implemented by opencorporates (X-API-TOKEN auth, ODbL attribution mandatory)                 | legalparty-model                          |
| BillTo             | The per-tenant link naming which LegalParty receives the tenant's invoices; one stream per tenant, keyed by the tenant ID (issue #45) | legalparty-model |
| BillToRedirected   | The event recording that a tenant's bills now go to another legal party; the previous party stays journaled for retention  | legalparty-model |

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

| Term              | Definition                                                              | Context        |
| ----------------- | ----------------------------------------------------------------------- | -------------- |
| UserID            | Branded string uniquely identifying a user                              | Identity       |
| Email             | Validated email address (`ParseEmail`/`MustParseEmail`)                 | Identity       |
| Session           | Token + user + actor + origin + timestamps, immutable after creation    | Authentication |
| Role              | Named permission group string (super_admin, admin, user, viewer, owner) | RBAC           |
| Policy            | An RBAC rule: subject + domain + object + action + effect               | Authorization  |
| GroupPolicy       | A role assignment: subject + role + domain                              | Authorization  |
| VerificationToken | Single-use secret proving email ownership, with a TTL                   | Authentication |

## Events

Immutable records of domain changes.

| Event                   | What Happened                                                                                                              | Aggregate  |
| ----------------------- | -------------------------------------------------------------------------------------------------------------------------- | ---------- |
| UserRegistered          | A new user account was created with email and roles                                                                        | User       |
| RolesUpdated            | The user's roles were changed in a specific domain (**legacy** — superseded by MemberRolesChanged, kept for replay compat) | User       |
| EmailChanged            | The user's email address was changed                                                                                       | User       |
| EmailVerified           | The user proved ownership of their email via token                                                                         | User       |
| DisplayNameChanged      | The user's display name was changed                                                                                        | User       |
| UserDeleted             | The user was deleted (tombstone — no further changes)                                                                      | User       |
| CredentialAdded         | A WebAuthn credential was registered to the user                                                                           | User       |
| CredentialRemoved       | A WebAuthn credential was removed from the user                                                                            | User       |
| TOTPEnabled             | TOTP multi-factor authentication was activated                                                                             | User       |
| TOTPDisabled            | TOTP multi-factor authentication was deactivated                                                                           | User       |
| ExternalAccountLinked   | An OAuth2/OIDC provider account was linked to the user                                                                     | User       |
| ExternalAccountUnlinked | An OAuth2/OIDC provider account was unlinked                                                                               | User       |
| MemberAdded             | An actor was granted a role in a tenant                                                                                    | Membership |
| MemberRolesChanged      | An actor's roles within a tenant were changed                                                                              | Membership |
| MemberRemoved           | An actor's membership in a tenant was revoked                                                                              | Membership |
| TenantCreated           | A new tenant was created                                                                                                   | Tenant     |
| TenantSuspended         | A tenant was suspended (inactive but not deleted)                                                                          | Tenant     |
| TenantReactivated       | A suspended tenant was reactivated                                                                                         | Tenant     |
| TenantDeleted           | A tenant was deleted (tombstone)                                                                                           | Tenant     |
| BotRegistered           | A bot was registered with an API token                                                                                     | Bot        |
| BotDeleted              | A bot was deleted (tombstone)                                                                                              | Bot        |
| LoginSucceeded          | An authentication attempt succeeded (auth-audit trail, 2026-10-19)                                                         | Auth audit |
| LoginFailed             | An authentication attempt failed, with reason (auth-audit trail)                                                           | Auth audit |
| RegistrationClosed      | A registration ceremony window was closed (auth-audit trail)                                                               | Auth audit |
| UserMerged              | The survivor side of a user merge: assets adopted, losers recorded on the survivor's stream                                  | User       |
| UserMergedInto          | The loser tombstone: this user stream was merged into a survivor and is terminal                                             | User       |
| TenantRenamed           | A tenant's display name changed; the ID (stream, Casbin domain, every tenant-scoped key) never moves                          | Tenant     |
| TenantMerged            | The survivor side of a tenant merge: the absorbed tenants recorded on the survivor's stream                                   | Tenant     |
| TenantMergedInto        | The loser tombstone: this tenant stream was merged into a survivor and is terminal                                            | Tenant     |
| BotTransferred          | A bot moved to a new owner, keeping its ID and token hash                                                                    | Bot        |
| BillToOpened            | A tenant's billing link was created, naming the first legal party bills go to                                                | BillTo     |
| BillToRedirected        | A tenant's bills now go to another legal party; the previous party stays journaled                                           | BillTo     |
| BillToClosed            | A tenant's billing link closed (terminal, retained)                                                                          | BillTo     |

## Commands

Intents that trigger state changes.

| Command               | What It Does                                                  | Events Produced         | Aggregate  |
| --------------------- | ------------------------------------------------------------- | ----------------------- | ---------- |
| RegisterUser          | Create a new user account (email only, no password)           | UserRegistered          | User       |
| UpdateRoles           | **Legacy** — removed. Use UpdateMemberRoles instead           | RolesUpdated (legacy)   | User       |
| ChangeEmail           | Change the user's email address                               | EmailChanged            | User       |
| VerifyEmail           | Confirm email ownership with a verification token             | EmailVerified           | User       |
| ChangeDisplayName     | Change the user's display name                                | DisplayNameChanged      | User       |
| DeleteUser            | Soft-delete the user (tombstone; cascades memberships + bots) | UserDeleted             | User       |
| AddCredential         | Register a WebAuthn passkey to the user                       | CredentialAdded         | User       |
| RemoveCredential      | Remove a WebAuthn passkey from the user                       | CredentialRemoved       | User       |
| EnableTOTP            | Activate TOTP multi-factor authentication                     | TOTPEnabled             | User       |
| DisableTOTP           | Deactivate TOTP (requires a valid code)                       | TOTPDisabled            | User       |
| LinkExternalAccount   | Link an OAuth2/OIDC provider account to the user              | ExternalAccountLinked   | User       |
| UnlinkExternalAccount | Unlink an OAuth2/OIDC provider account                        | ExternalAccountUnlinked | User       |
| AddMember             | Grant an actor a role within a tenant                         | MemberAdded             | Membership |
| UpdateMemberRoles     | Change an actor's roles within a tenant                       | MemberRolesChanged      | Membership |
| RemoveMember          | Revoke an actor's membership in a tenant                      | MemberRemoved           | Membership |
| CreateTenant          | Create a new tenant                                           | TenantCreated           | Tenant     |
| SuspendTenant         | Suspend a tenant (inactive but not deleted)                   | TenantSuspended         | Tenant     |
| ReactivateTenant      | Reactivate a suspended tenant                                 | TenantReactivated       | Tenant     |
| DeleteTenant          | Delete a tenant (tombstone; cascades memberships)             | TenantDeleted           | Tenant     |
| RegisterBot           | Register a bot with an API token for automated access         | BotRegistered           | Bot        |
| DeleteBot             | Delete a bot (tombstone)                                      | BotDeleted              | Bot        |
| MergeUsers            | Fold duplicate users into a survivor (adopt assets, record)   | CredentialAdded/ExternalAccountLinked/TOTPEnabled/EmailVerified/UserMerged | User |
| MergeUserInto         | Tombstone a duplicate user as merged into a survivor (last)   | UserMergedInto          | User       |
| RenameTenant          | Change a tenant's display name in place                       | TenantRenamed           | Tenant     |
| MergeTenants          | Record absorbed tenants on the surviving tenant               | TenantMerged            | Tenant     |
| MergeTenantInto       | Tombstone a tenant as merged into a survivor (last)           | TenantMergedInto        | Tenant     |
| TransferBot           | Move a bot to a new owner (token and ID unchanged)            | BotTransferred          | Bot        |
| OpenBillTo            | Open a tenant's bill-to link                                  | BillToOpened            | BillTo     |
| RedirectBillTo        | Point a tenant's bills at another legal party                 | BillToRedirected        | BillTo     |
| CloseBillTo           | Close a tenant's bill-to link (terminal)                      | BillToClosed            | BillTo     |

## Bounded Contexts

| Context                 | Responsibility                                                                    | Module           |
| ----------------------- | --------------------------------------------------------------------------------- | ---------------- |
| User Management         | User lifecycle, authentication, authorization, sessions                           | usermgmt         |
| Identity Model          | Pure domain types — IDs, events, commands, fold functions, Authz engine           | identity-model   |
| CQRS-HTMX Core          | HTTP handler integration, response building, middleware                           | root module      |
| Dashboard Observability | CQRS/ES event browser, projection health, time-travel inspector, SSE live updates | dashboardui      |
| Authorization           | RBAC policy management, enforcement, role hierarchy                               | usermgmt (Authz) |
| MFA                     | TOTP second-factor enrollment and verification                                    | usermgmt         |
| Auditing                | Immutable record of user actions derived from events                              | usermgmt         |
| Legal Party             | Billing legal-party domain: LegalParty aggregate (opened/updated/closed, never deleted), legal-form and tax-identifier types, GLEIF/OpenCorporates registry enrichment | legalparty-model |
