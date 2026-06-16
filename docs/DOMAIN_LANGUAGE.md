# Domain Language

A **Ubiquitous Language** for `cqrs-htmx` — shared across Customer, Product Owner, Developer, and AI.
Inspired by Domain-Driven Design (DDD).

Every term below should mean the **same thing** to everyone who reads it.

## Glossary

| Term          | Definition                                                                      | Context               |
| ------------- | ------------------------------------------------------------------------------- | --------------------- |
| Aggregate     | A cluster of domain objects treated as a single unit for data consistency       | Event-sourced CQRS    |
| Authz         | Authorization engine wrapping Casbin RBAC with domains                          | usermgmt module       |
| Casbin        | External authorization library providing RBAC with domain support               | Authorization         |
| Command       | An intent to change state, dispatched to a handler that produces events         | CQRS write side       |
| Credential    | A WebAuthn passkey registered to a user for passwordless authentication         | Authentication        |
| CQRS          | Command Query Responsibility Segregation — separate write and read models       | Architecture pattern  |
| Decider       | Pure function that validates a command against current state and emits events   | Event sourcing        |
| Enforcer      | Interface satisfied by Casbin; the authorization decision point                 | Authorization         |
| Event         | An immutable record of something that happened in the past                      | Event sourcing        |
| Event Store   | Append-only persistence for events on an aggregate stream                       | Event sourcing        |
| Fold          | Pure function that reconstructs aggregate state by replaying events             | Event sourcing        |
| HandlerOption | Functional option pattern for configuring CQRS HTTP handlers                    | cqrs-htmx root module |
| HTMX          | HTML-over-the-wire library for dynamic web pages without writing JavaScript     | Frontend              |
| Projection    | A read model built by subscribing to events and updating a materialized view    | CQRS read side        |
| Read Model    | A query-optimized view of current state, derived from the event stream          | CQRS read side        |
| Role          | A named permission group (admin, user, viewer, owner) assigned within a domain  | RBAC authorization    |
| Service       | The application service orchestrating commands, queries, and session management | usermgmt module       |
| Session       | An ephemeral authentication artifact (token + expiry) created after login       | Authentication        |
| Templ         | Go HTML templating engine with type-safe compile-checked templates              | Frontend              |
| UserID        | A branded string type uniquely identifying a user                               | usermgmt module       |
| WebAuthn      | W3C standard for passwordless authentication using passkeys/FIDO2               | Authentication        |

## Entities

Objects with identity and lifecycle.

| Term               | Definition                                                           | Context                |
| ------------------ | -------------------------------------------------------------------- | ---------------------- |
| User               | A registered person with email, roles, credentials, and display name | User aggregate root    |
| WebAuthnCredential | A registered passkey with public key, transports, and backup state   | Part of User aggregate |

## Value Objects

Immutable objects defined by attributes.

| Term        | Definition                                                | Context        |
| ----------- | --------------------------------------------------------- | -------------- |
| UserID      | Branded string uniquely identifying a user                | Identity       |
| Session     | Token + expiry + user ID, immutable after creation        | Authentication |
| Role        | Named permission group string (admin, user, viewer)       | RBAC           |
| Policy      | An RBAC rule: subject + domain + object + action + effect | Authorization  |
| GroupPolicy | A role assignment: subject + role + domain                | Authorization  |

## Events

Immutable records of domain changes.

| Event              | What Happened                                         | Aggregate |
| ------------------ | ----------------------------------------------------- | --------- |
| UserRegistered     | A new user account was created with email and roles   | User      |
| RolesUpdated       | The user's roles were changed in a specific domain    | User      |
| EmailChanged       | The user's email address was changed                  | User      |
| DisplayNameChanged | The user's display name was changed                   | User      |
| UserDeleted        | The user was deleted (tombstone — no further changes) | User      |
| CredentialAdded    | A WebAuthn credential was registered to the user      | User      |
| CredentialRemoved  | A WebAuthn credential was removed from the user       | User      |

## Commands

Intents that trigger state changes.

| Command           | What It Does                                        | Events Produced    |
| ----------------- | --------------------------------------------------- | ------------------ |
| RegisterUser      | Create a new user account (email only, no password) | UserRegistered     |
| UpdateRoles       | Change the user's role assignments in a domain      | RolesUpdated       |
| ChangeEmail       | Change the user's email address                     | EmailChanged       |
| ChangeDisplayName | Change the user's display name                      | DisplayNameChanged |
| DeleteUser        | Soft-delete the user (tombstone)                    | UserDeleted        |
| AddCredential     | Register a WebAuthn passkey to the user             | CredentialAdded    |
| RemoveCredential  | Remove a WebAuthn passkey from the user             | CredentialRemoved  |

## Bounded Contexts

| Context         | Responsibility                                          | Module           |
| --------------- | ------------------------------------------------------- | ---------------- |
| User Management | User lifecycle, authentication, authorization, sessions | usermgmt         |
| CQRS-HTMX Core  | HTTP handler integration, response building, middleware | root module      |
| Authorization   | RBAC policy management, enforcement, role hierarchy     | usermgmt (Authz) |
