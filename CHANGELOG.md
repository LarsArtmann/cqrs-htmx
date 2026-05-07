# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

### Added

- BDD test suite using Ginkgo/Gomega (`bdd_test.go`)
- `DecodeFormQuery` handler option for query parameter form decoding (symmetry with `DecodeForm`)
- `docs/` directory with architecture reviews, planning docs, and status reports

### Changed

- Extract helper functions (`hasNoResponse`, `hasMinimalResponse`, `decodeJSONBody`, `decodeRequest`, `decodeFormBody`, `notifyOption`, `triggerNotification`) to reduce duplication
- Extract notification helpers to dedicated `notify.go` module
- Consolidate duplicate test types across test files into shared helpers
- Remove local-path `replace` directives from `go.mod` — resolve from GitHub

### Fixed

- Use `headerRedirect` constant instead of hardcoded `"HX-Redirect"` string in `DefaultErrorHandlerWithRedirect`
- Thread `Config.LoginRedirect` into per-App error handler (was dead code — `New()` now creates a closure that captures the resolved loginRedirect)
- Use `headerTrue` constant in `Response.Refresh()` instead of hardcoded `"true"`
- Fix README compile-breaking example (`cqrshtmx.LoginRedirect` → `Config.LoginRedirect`)
- Fix error wrapping: `errors.Wrapf` with `%s` on sentinels → `fmt.Errorf("%w: ...")` throughout

### Removed

- Remove dead `enrichContext()` no-op stub
- Remove redundant gocritic `disabled-checks` entries (`dupImport`, `octalLiteral`, `whyNoLint` — already disabled by default)
- Remove unused `io` and `event` imports from test files

## [0.2.0] - 2026-05-07

### Added

- Eliminate all 103 golangci-lint issues → 0 issues, project is lint-clean
- `.golangci.yml` v2 format with proper exclusion rules
- Comprehensive test coverage at 93.5% (138 specs)

## [0.1.0] - 2026-05-04

### Added

- **App builder**: `App` struct with `Config`, `Command()`, `Query()`, per-App `ErrorHandler` and `LoginRedirect`
- **CQRS dispatch**: `handleCommandDispatch()` and `handleQueryDispatch()` with automatic error handling
- **Handler options**: `DecodeJSON`, `DecodeForm`, `Render`, `RenderTempl`, `RenderTemplResult`, `Authorize`, `Enforce`, `UserIDExtractor`
- **HTMX response builder**: Fluent API with `Response` struct — `StatusCode()`, `Header()`, `Redirect()`, `Refresh()`, `Retarget()`, `Reswap()`, `Trigger()`, `TriggerAfterSettle()`, `TriggerAfterSwap()`
- **HTMX middleware**: `HTMXMiddleware` parses `HX-*` headers once, stores `HTMXRequest` in context
- **HTMX context accessors**: `IsHTMXRequest()`, `GetHTMXPrompt()`, `GetHTMXTarget()`, `GetHTMXTrigger()`, `GetHTMXTriggerName()`, `RenderPartial()`
- **Notification system**: `NotifySuccess`, `NotifyError`, `NotifyWarning`, `NotifyInfo` — standard `{level, message}` trigger pattern via `notify.go`
- **Casbin authorization**: `Authorize()`, `Enforce()`, `AuthorizeMiddleware()` using `casbin/casbin/v3`
- **Context enrichment**: `ContextEnrichmentMiddleware` + `UserIDExtractor` → context → event metadata
- **Error classification**: CQRS error → HTTP status mapping with `RegisterClassification`, sentinel errors (`ErrUnauthorized`, `ErrForbidden`, `ErrNotFound`, `ErrBadRequest`, `ErrConflict`, `ErrInternal`), and `LoginRedirect` support
- **templ integration**: `TemplComponent` duck-typed interface (no `a-h/templ` import dependency) with `RenderTempl` and `RenderTemplResult` options
- **Middleware chain**: `Chain()` utility for composing `net/http` middleware
- **Git Town integration**: `git-town.toml` configuration

### Changed

- Deduplicate `UserIDExtractor` calls — handlers check context first, skip if middleware already set user ID
