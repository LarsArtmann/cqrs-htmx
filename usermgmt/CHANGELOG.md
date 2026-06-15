# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased] — BREAKING

### Changed — Event-Sourced CQRS Architecture

- **BREAKING**: User aggregate is now fully event-sourced using go-cqrs-lite's Decider pattern
- **BREAKING**: `UserStore` interface and `InMemoryUserStore` REMOVED — replaced by `UserReadModel` projection
- **BREAKING**: `ServiceConfig.UserStore` field removed. Use `ServiceConfig.EventStore` and `ServiceConfig.EventBus` instead
- **BREAKING**: User mutation methods (`SetRoles`, `SetEmail`, `SetDisplayName`, `AddRole`, `RemoveRole`) are deprecated — state changes go through Service methods (dispatched as commands → events)
- Added 6 event types: `UserRegistered`, `PasswordChanged`, `RolesUpdated`, `EmailChanged`, `DisplayNameChanged`, `UserDeleted`
- Added 6 command types: `RegisterUser`, `ChangePassword`, `UpdateRoles`, `ChangeEmail`, `ChangeDisplayName`, `DeleteUser`
- Added `UserReadModel` projection (query side) with email index for O(1) lookups
- Added `CasbinProjection` — Casbin policies fully derived from events (single source of truth = event store)
- Added `Service.DeleteUser()`, `Service.ChangeEmail()`, `Service.ChangeDisplayName()` methods
- Added `DefaultEventSourcedSetup()`, `EventSourcedConfig`, `UserDecider()`, `RegisterCommands()` for advanced wiring
- `Service.Register` now pre-checks email uniqueness via read model before dispatching
- `Service.DeleteUser` revokes all user sessions for security
- Password hashing happens in Service layer (commands carry bcrypt hashes, not plaintext)
- Read-your-writes consistency via `MemoryBus` synchronous delivery
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

Service method signatures (`Register`, `Login`, `ChangePassword`, `UpdateRoles`, `GetUser`, `Authenticate`) are unchanged.

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
