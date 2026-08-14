# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

### Added

- **Registration cap (`ServiceConfig.MaxUsers`)**: when greater than zero, no new users can be created once the read model reaches the cap — `Service.Register` returns `ErrRegistrationClosed` (HTTP 403). Zero (default) keeps unlimited registration.
- **OAuth2 auto-provisioning gated by `MaxUsers`**: first-login user creation in `OAuth2Service.matchOrCreateUser` enforces the same cap (error code `usermgmt.oauth2.registration_closed`, same 403). Existing users keep working: matches by external account and by email link as before, repeat logins are never blocked.
- Handler tests for the 403 mapping of both paths (`TestHandlers_Register_MaxUsersReached_Returns403`, `TestHandler_OAuth2Callback_RegistrationClosed_Returns403`) and a mixed concurrency regression test proving exactly `MaxUsers` users are created under parallel Register + first-login load.

### Changed

- **Registration check is serialized**: `Service` and `OAuth2Service` share one registration mutex held across the count-check-then-dispatch window, closing the TOCTOU race where two concurrent registrations could both pass the advisory count check (effective for in-process synchronous projections, which is the shipped setup).

### Breaking

- `NewOAuth2Service` signature changed: it now takes `maxUsers int` and `registrationMu *sync.Mutex` (pass `ServiceConfig.MaxUsers` and the Service's registration lock). Consumers that only build the service via `NewService` are unaffected.

## [v4.7.0] - 2026-08-07

### Added

- State cache wired in all aggregate repositories via `decider.WithStateCache` (O(total events) → O(new events) per Execute).
- Lockout eviction wired into `NewService` via `wireLockoutEviction()` (prevents unbounded memory growth).
- `OnProjectionFailed` callback on `EventSourcedConfig`/`ServiceConfig` for terminal projection worker failures.
- MySQL event-store dialect, read model constructors, and setup template.
- `ReadinessHandler` + `DebugHandler` composite health checks (root module).
- UserDelete cascades memberships and bots (prevents orphaned API tokens).
- Benchmark for state cache cold-vs-warm path (13.7x speedup).

## [v4.6.0] - 2026-07-26

### Fixed

- **ErrorFamily compliance** (`store.go`): migrated `errors.New("entity already exists")` to `errorfamily.NewConflict`. `nix run .#errorfamily` now reports 0 violations for usermgmt.

### Changed

- **Dedup sweep**: shared helpers extracted (see root CHANGELOG `[v4.6.0]` Changed section for the full list).

## [v4.5.0] - 2026-07-24

### Added

- **Projection health monitoring** (`es_projection_health.go`): `ProjectionStatuses()` method on both `EventSourcedSetup` and `Service` returns live projection status (name, state, lag, last-event, error). Enables ops dashboards and alerting via `cqrshtmx.ProjectionStatusHandler`.
- **Projection rebuild** (`es_projection_health.go`): `RebuildProjection(ctx, name)` on both `EventSourcedSetup` and `Service`. Stops host, resets named projection checkpoint + read-model, creates fresh host, replays entire journal. Preserves read-your-writes by blocking until all workers reach live state.
- **Event catalog** (`es_event_catalog.go`): `DefaultEventCatalog()` pre-populates all 21 event types across User (12), Membership (3), Tenant (4), Bot (2) with descriptions and payload field metadata. `EventCatalog()` method on both setup structs.
- **Upcaster support** (`upcaster.go`): Thin wrapper around identity-model's upcaster registry for event payload version migration.

### Changed

- **identity-model integration**: All domain type definitions replaced with type aliases to identity-model. identity-model is now the single source of truth for domain types, fold logic, and constants. Command dispatch uses accessor methods (`c.Email()`, `c.Roles()`).
- **Projection host factory deduplicated**: Extracted shared `startProjectionHost` helper used by both `StartProjections` and `RebuildProjection`.
- **Service field renamed**: `projectionListField` → `projections` for consistency.
- **go-cqrs-lite v4.0.x dependency alignment**: Updated all go-cqrs-lite module references.

## [v4.2.0] - 2026-07-08

### Added

- **dedup.Ring O(1) memory** (`es_projection_setup.go`): Replaced unbounded `map[id.EventID]struct{}` with `dedup.Ring` (1024 entries, ~90KB fixed) for replay→live dedup. Memory is now O(1) regardless of journal size — a 1M-event journal no longer loads 1M IDs permanently into memory.
- **CBOR codec support** (`es_readmodel.go`): `unmarshalPayload` now resolves codec per-event via `codec.ForEncoding(evt.Encoding())` instead of hardcoded `json.Unmarshal`. Consumers who set `event.DefaultCodec = codec.CBORCodec{}` get transparent CBOR support. Mixed JSON+CBOR event streams decode correctly.

### Changed

- **go-error-family direct import**: Migrated from transitive dependency (via go-cqrs-lite event/v3) to direct import. All error constructors now use `errorfamily.New*`/`Wrap*` directly. Error contexts enriched with domain-specific identifiers (user IDs, provider names, credential IDs).
- **go-cqrs-lite v3.5.0 → v3.7.4**: Aligned all go-cqrs-lite modules (decider, projection, stack, storage, watermill, listing, scheduling, scenario, stack/sqlite, stack/postgres). Adopted dedup.Ring and codec.ForEncoding from v3.7.0+.

### Fixed

- **go-cqrs-lite version drift**: Several modules were pinned at v3.7.0 while others were at v3.7.4. All now aligned to latest available tags.

## [v4.1.1] - 2026-07-04

### Changed

- **httputil v0.3.0 → v0.4.0**: Transitive dependency bump. No API or behavior change for usermgmt consumers.

## [v4.0.1] - 2026-07-02

### Added

- **Configurable TOTP pending-secret TTL** (`ServiceConfig.TOTPPendingSecretTTL`): Was hardcoded to 5 minutes, now configurable. Defaults to 5 minutes when ≤ 0.
- **Configurable WebAuthn session TTL** (`ServiceConfig.WebAuthnSessionTTL`): Was hardcoded to 5 minutes, now configurable.
- **Coverage tests** for parse helpers (`ParseUserID`, `MustParseUserID`, `ParseActorID`, `MustParseEmail`).
- **Fuzz tests** on JSON serialization boundary (`FuzzMarshalWebAuthnUser`, `FuzzParseUser`, `FuzzParseSession`).

## [v4.0.0] - 2026-07-02

### Changed — Passwordless Event-Sourced CQRS

- **BREAKING**: ALL password code removed. No bcrypt, no PasswordHash, no ChangePassword, no validatePassword, no LoginRequest/LoginResponse, no Service.Login, no Service.ChangePassword.
- **BREAKING**: `User` struct no longer has `PasswordHash` field. Replaced by `Credentials []WebAuthnCredential`.
- **BREAKING**: `RegisterRequest` no longer has `Password` field. Registration is email-only.
- **BREAKING**: `ServiceConfig.BcryptCost` field removed. Use `ServiceConfig.WebAuthn` (the `WebAuthnProvider` interface) instead.
- **BREAKING**: `golang.org/x/crypto` dependency removed. WebAuthn deps moved to the optional `usermgmt/webauthn` sub-module.
- **BREAKING**: User aggregate is now fully event-sourced using go-cqrs-lite's Decider pattern.
- **BREAKING**: `UserStore` interface and `InMemoryUserStore` REMOVED — replaced by `UserReadModel` projection.
- **BREAKING**: `ServiceConfig.UserStore` field removed. Use `ServiceConfig.EventStore` and `ServiceConfig.EventBus` instead.
- **BREAKING**: User mutation methods (`SetRoles`, `SetEmail`, `SetDisplayName`, `AddRole`, `RemoveRole`, `SetPassword`, `CheckPassword`, `touch`) ALL removed.
- Added WebAuthn/Passkey authentication via go-webauthn v0.17.4
- Added 7 event types: `UserRegistered`, `RolesUpdated`, `EmailChanged`, `DisplayNameChanged`, `UserDeleted`, `CredentialAdded`, `CredentialRemoved`
- Added 7 command types: `RegisterUser`, `UpdateRoles`, `ChangeEmail`, `ChangeDisplayName`, `DeleteUser`, `AddCredential`, `RemoveCredential`
- Added `WebAuthnCredential` type, `WebAuthnProvider` interface for ceremony delegation
- Added `Service.BeginRegistration` / `FinishRegistration` / `BeginLogin` / `FinishLogin`
- Added HTTP endpoints: `POST /auth/webauthn/{register,login}/{begin,finish}`
- Added `Service.AddCredential`, `Service.RemoveCredential`, `Service.ChangeEmail`, `Service.ChangeDisplayName`, `Service.DeleteUser`
- Added `UserReadModel` projection with email index for O(1) lookups
- Added `CasbinProjection` — Casbin policies fully derived from events
- Added `Authz.RemoveAllRolesForUser` for clean user deletion
- Added `DefaultEventSourcedSetup()`, `EventSourcedConfig`, `UserDecider()`, `RegisterCommands()`
- Added new errors: `ErrNoCredentials`, `ErrWebAuthnNotConfigured`, `ErrSessionDataNotFound`
- Added comprehensive `doc.go` with usage examples
- `Service.Register` now pre-checks email uniqueness via read model before dispatching
- `Service.DeleteUser` revokes all user sessions for security
- Read-your-writes consistency via `watermill.EventBus` with `BlockPublishUntilSubscriberAck: true`
- Old `EventHandler` callback preserved for backward compatibility (bridged from bus events)
- See `docs/adr/0006-event-sourced-user-aggregate.md` for full decision record

### Migration Guide

**Before (CRUD):**

```go
svc, _ := usermgmt.NewService(usermgmt.ServiceConfig{
    UserStore: usermgmt.NewInMemoryUserStore(),
})
```

**After (Event-Sourced):**

```go
svc, _ := usermgmt.NewService(usermgmt.ServiceConfig{}) // defaults to in-memory event store + bus
```

Service method signatures for surviving methods (`Register`, `UpdateRoles`, `GetUser`, `Authenticate`) are unchanged. The following methods were **removed**: `Login`, `ChangePassword` — replaced by WebAuthn ceremonies (`BeginRegistration`/`FinishRegistration`/`BeginLogin`/`FinishLogin`).

## [2.0.0] - 2026-05-27

### Changed

- Upgraded to go-cqrs-lite v2.2.0 with `/v2` import paths
- Branded `UserID` via go-branded-id (`brandid.ID[userBrand, string]`)
- Domain model enrichment: `SetRoles`, `ChangePassword`, `SetEmail`, `SetDisplayName`, `AddRole`, `RemoveRole`, `IsPasswordSet`
- Domain events: `UserRegisteredEvent`, `UserLoggedInEvent`, `PasswordChangedEvent`, `RolesUpdatedEvent`
- Error handling migrated to go-error-family v0.3.0

## [0.1.0] - 2026-01-01

### Added

- Initial release
