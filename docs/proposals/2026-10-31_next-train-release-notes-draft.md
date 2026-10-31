# Next-train release notes — DRAFT (round-45 M21, 2026-10-31)

Source of truth: root `CHANGELOG.md` [Unreleased] + module CHANGELOG deltas.
Every named delta below cites its receipt (m21.2 check). NOT a release — the
draft ships with the next train (M27 verifies; tagging is a separate ritual).

## Root (next: v4.15.0 — minor; wire-visible changes)

- **T12 error-code namespace collapse** (`cqrshtmx.projection.*` family;
  root + health receipts in CHANGELOG [Unreleased]). CONSUMER ACTION: update
  alerts matching the old `projection.*` / `cqrshtmx.health.projection_*`
  code strings.
- **RFC 7807 `StructuredError.Context`** (machine context, <500 only;
  `omitempty` — context-less responses byte-identical).
- Family-wave pins: templ-components v1.26.0, go-codec v0.3.3,
  go-health-dashboard v0.12.2 (see family notes).

## health (next: v4.15.0 — minor)

- T12 bridge codes renamed (CHANGELOG receipt lives in root per repo
  practice); consumer-visible identical to root's T12 note.

## adminui (next: v4.13.0 — patch)

- No source delta this round (the S38e unlink test repair is TEST-only).
- Family pins ride (templ-components v1.26.0 + go-codec v0.3.3).

## dashboardui (next: v4.14.0 — patch; carries [Unreleased] W14 batch)

- The W14 honesty-and-hang batch (CHANGELOG [Unreleased]) + the amber
  warning-badge golden update (v1.26.0 semantic hue).

## loginpage (next: v4.13.0 — patch)

- No source delta; family pins ride.

## setup / integration_test / examples

- Family pins only; setup's DisableService constant-200 disclosure is docs.

## Family notes (upstream deltas adopted 2026-10-31, receipts in root CHANGELOG)

- templ-components v1.24.0→v1.26.0: RelativeTime `EmptyText`; five chart
  render fixes; amber warning hue (INTENTIONAL pixel diff); LoadingOverlay
  → `role="status"` + new `LoadingDialog`; native `<progress>`;
  popover-era overlays; `Toggle` `role="switch"`; ThemeToggle cross-tab.
- go-codec v0.3.3: decode-only — CBOR text strings decode into
  TextUnmarshaler types (pre-branding journal replay works again).
- go-health-dashboard v0.12.2: Table wrappers gain `role="region"`; no API.

## Pre-tag checklist (the ritual, not repeated here)

verify-tag guards → wave order → carrier re-tag report → treadmill
`--strict-lag 0` green at push time (gotcha 27/39).
