# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

### Changed

- **Rebuilt on the templ-components design system.** The hand-rolled `lp-*`
  CSS card is gone: the page now renders via `layout.Base`, `forms.Input`,
  `display.Button`, and `feedback.Alert`, and every class is a Tailwind v4
  utility compiled by the consumer (`@source` this package — see README
  "Styling"). `Config.CSSPath` now defaults to `/app.css` and is effectively
  required. `assets/login.css` is deleted; `assets/login.js` remains embedded
  and its class hooks changed (`hidden` instead of `lp-hidden`, an injected
  spinner instead of the `.lp-btn-loading` CSS class).

### Added

- **CSP-nonce support** — `Config.NonceFromRequest` renders the inline config
  JSON and WebAuthn scripts with the consumer's per-request nonce (sanitized
  to base64url before embedding), for consumers whose CSP has no
  `'unsafe-inline'` for `script-src`. Closes the 2026-10-04 review follow-up (b).
- **Server-driven theming** — `Config.Theme` / `Config.ThemeFromRequest`
  resolve the color scheme per request and render it SSR-first (body class,
  `NoThemeScript`), keeping cookie-driven consumers the single source of truth.
- **`Config.RegisterFirst`** — renders the registration section on load, for
  consumers exposing `/register` as a distinct route (the JS toggle links
  still switch sections).

### Fixed

- **Hardened config embedding against HTML injection** (defense-in-depth; all inputs are consumer-controlled `Config` fields, not remote input):
  - The client config JSON (contains `CredentialName`, `AuthPrefix`) is now escaped before being embedded in the `application/json` script block — `encoding/json/v2` does not HTML-escape, so a value containing `</script>` could previously break out of the block.
  - `Config.AccentColor` is now validated (`New`/`NewPageData` fail fast) — it is embedded unescaped in the inline `<style>` block and the SVG favicon, so markup/quote characters were previously injectable.
  - `Config.CSSPath` is now validated to be a root-relative path or absolute http(s) URL, rejecting executable schemes like `javascript:`.
- Corrected the `Config.OAuth2Buttons` field doc: a nil/empty slice **auto-detects** configured providers (the previous comment claimed an empty slice hides all buttons — it never did).

## [v4.11.0] - 2026-09-19

### Changed

- **Coordinated family-train cut (v4.11.0).** No loginpage code changes since v4.10.0 (verified by tree diff): internal requires (root, usermgmt, identity-model) move to v4.11.0 and dependencies ride the Go 1.27.1 fleet floor (`go-etag v0.4.0` requires Go ≥ 1.27.1). Consumer action: build with Go ≥ 1.27.1.

## [v4.8.0] – [v4.10.0] — 2026-08-14 → 2026-09-10

- Coordinated lockstep bumps riding the family release trains (root v4.8.0–v4.10.0, usermgmt v4.8.0–v4.10.0). No loginpage-specific feature entries were recorded in this file for those trains; the historical detail lives in the root `CHANGELOG.md` sections for those versions.

## [v4.7.0] - 2026-08-07

### Added

- HTMX partial rendering support.
- Updated dependencies (root v4.7.0, usermgmt v4.7.0).

## [v4.6.1] - 2026-07-27

### Changed

- **Lockstep version bump** with root `cqrs-htmx/v4` v4.6.1 — no code changes in this sub-module; the bump keeps the lockstep release consistent. go-cqrs-lite `v4.1.0` → `v4.2.0` (command, event, id, idempotency, query); go-branded-id `v0.3.2` → `v0.5.0`.

## [v4.6.0] - 2026-07-27

### Changed

- **Lockstep version bump** with root `cqrs-htmx/v4` v4.6.0 — no code changes in this sub-module; the bump keeps the lockstep release consistent. go-cqrs-lite sub-module version refs aligned to v4.1.0; go-error-family bumped to v0.10.0.

## [v4.5.0] - 2026-07-24

### Changed

- **Dependency tidy**: aligned go-cqrs-lite and sibling module version refs across the workspace.

## [v4.4.0] - 2026-07-23

### Changed

- **httputil upgrade** to v0.6.0 — adapted to go-cqrs-lite stack/sqlopt split.
- **identity-model extraction**: identity-model extracted as a standalone module; loginpage now depends on it transitively via usermgmt.
- **go-cqrs-lite schema bump** to v4.0.3.

## [v4.3.0] - 2026-07-12

### Changed

- **go-cqrs-lite v3 → v4 migration**: all module paths migrated from `/v3` to `/v4`; vendored eventtest removed.

## [v4.0.0] - 2026-07-12

### Added — loginpage extracted as independent module

- **New module**: `github.com/larsartmann/cqrs-htmx/loginpage/v4`
- Self-contained passwordless WebAuthn login page (templ + HTMX).
- OAuth2 button support with auto-discovery — providers are rendered automatically when configured on the usermgmt service.
- Server-side ID generation for WebAuthn registration ceremonies.
- Graceful no-auth fallback (renders a message when no auth providers are configured).
- Depends on `cqrs-htmx/v4` and `cqrs-htmx/usermgmt/v4` — renders a ready-made login UI that mounts alongside any usermgmt-backed application.
