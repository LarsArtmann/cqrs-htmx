# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

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
