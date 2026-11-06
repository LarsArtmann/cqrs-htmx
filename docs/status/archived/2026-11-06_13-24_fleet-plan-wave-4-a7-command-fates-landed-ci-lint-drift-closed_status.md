# Status: Fleet-plan wave 4 — A7 command fates LANDED end to end, dashboardui's CI-lint drift window closed, owner directive unparked the adminui leg

> **ANNOTATED 2026-11-06** — ~~this report~~ **SUPERSEDED** by [`2026-11-06_22-42_fleet-plan-wave-5-completion-a22-a23-a24-sa1019-sweep-changelog-debt-a2-mid-flight_status.md`](../2026-11-06_22-42_fleet-plan-wave-5-completion-a22-a23-a24-sa1019-sweep-changelog-debt-a2-mid-flight_status.md), which wave-4 closeout state fully superseded by the 22-42 completion report (A7 receipted, fleet plan closed). Strike depth:
> supersession verdict (round-18 precedent for series-internal reports); nothing here is a
> unique open item. **ARCHIVED 2026-11-06 (round-23 (2026-11-06)).**

**Session window:** 2026-11-06 ~13:00 → ~13:30 CET (resumption of the 05:00–05:30 wave-4 session)
**Plan of record:** `docs/planning/2026-11-06_00-37_SUPERB-fleet-libs-audit-follow-through-pareto-plan.md` (A7 was in_progress with core done)
**Author:** this session (Crush). Intermediate sibling sessions (07:16–11:49: direct-imports W0, dispatch gate, huma, ExtraNav, issue #25, m2m) advanced the tree between my 05:30 pause and this resumption — their lanes untouched.
**Verdict:** A7 ✅ complete (core + UI + tests + docs + gates). Dashboardui lint 11 findings → 0 (a CI-red window from the ADR-0062 deprecation + tonight's A2 leg, closed). All other list items untouched this window; owner said "the WHOLE TODO list" — the adminui leg is unparked and is next.

---

## a) FULLY DONE (receipts)

1. **A7 — dashboardui Command Fates panel, complete.** On the prior session's green core layer (`core/command_fates.go` + 6 tests), this session landed the whole UI layer: `handlers_command_fates.go` (window handler + zero-time/attempt label helpers), `commandfates.templ` (page + rows; fate badges colored by severity, errorfamily badge `rejection/user.email_taken`, EventID drill-down links into `/events/{id}`), the `GET /command-fates` route row + `panelCommandFates` const (routes.go, gated `Capabilities.HasEventRead` — EventSource is stream-by-ref and cannot scan lifecycle streams), the nav entry (`icons.ExclamationTriangle`, config.go), `EmptyStateCommandFates` + honest opt-in copy naming `ServiceConfig.CommandLifecycle`, `fateBadgeType` (badges.go), the `commandFate` alias + `loadCommandFates` wrapper (core_bridge.go). Tests: 4 new main-package tests (rows render with family badge + links; empty-state copy; route gated OFF for EventSource-only configs via the ROUTE MANIFEST — the mux 200s on "/" fallback so mux-404 is the wrong instrument; `fateBadgeType` matrix) + the 2 existing route pins trued (13 panels, journal-only 11 routes + allowlist). Generated `_templ.go` with the **pinned** v0.3.1100 generator (`nix build .#templGen`). Verification: full dashboardui suite green, golangci 2.15.0 **0 issues**, coverage 73.4% (threshold 60%), `check-css-bundle-classes` green after rebuilding `dashboard-tw.css` same-change (the new `.max-w-64` utility). Docs: README capability-table rows + a "Command Fates" section; CHANGELOG [Unreleased] receipt; **gotcha 55** (CBOR default encoding) in `docs/gotchas.md` + the AGENTS index row + the trap count trued 54 → 55.
2. **Dashboardui CI-lint drift window closed (11 findings → 0).** The real gate (`golangci-lint run ./...`, same v2.15.0 + config CI's matrix uses) was RED: 6× SA1019 from root's ADR-0062 deprecation landing (`cqrshtmx.IsHTMXRequest`/`HTMXScriptHandler` in dashboard.go, handler.go, render.go, routes.go, handlers_coverage_ext_test.go ×2) + nlreturn (handler_overview.go) + wsl_v5 (overview_sse_hub_test.go) from tonight's A2 leg + 3 findings in my new core file (exhaustive: added the `codec.EncodingRaw` case mirroring `core/payload.go`; exhaustruct: `core\.CommandFate$` ignore-pattern beside the existing precedent; gocognit 27: extracted `keepFates`), later + cyclop (buildNav 13>12 after my nav entry — refactored to table-driven gated entries) and 2 gci (gofmt). All sites repointed to `htmx.IsRequest`/`htmx.ScriptHandler` (the ADR-0062 direct-import direction).
3. **A26 unit 2 of 2–3:** the 18-15 round-46 report archived same-change as this report's tail ritual (successor: the 02-44 waves-1-2 execution report; round-46 receipts are fully absorbed into AGENTS/TODO/CHANGELOG).

## b) IN PROGRESS (paused mid-flight)

Nothing — A7 closed at a clean boundary. Next up per the list: **A8** (examples/basic partial-render showcase).

## c) BLOCKED / VERIFIED-BROKEN (owned by whom)

- **Q1 (identity-model v4.16.0 tag):** still owner-gated; the concurrent session-audit lane stays theirs. Not on this list.
- **Q2 (A17 render verification):** chromium sanction/waiver still unanswered. Not on this list.
- **A25:** still gated on the upstream go-cqrs-lite catalog release. Not on this list.
- **NEW FLAG — possible tree-wide lint-red window from ADR-0062:** the same SA1019 class I fixed in dashboardui likely fires in OTHER modules still consuming the deprecated root re-exports (adminui, loginpage, setup, usermgmt, examples). NOT verified this session — `nix run .#lint` runs the root module only; CI's matrix runs per-module. First action of the next battery: sweep each module for the deprecated symbols. If red, the fix is the same mechanical repoint.

## d) DECISIONS (autonomous) + SELF-ATTRIBUTED FAILURES

- **Owner directive read:** "NOW GET SHIT DONE! The WHOLE TODO LIST!" unparks the A2-adminui leg (standing Q3 → proceed with rebase care). Q1/Q2/A25 remain parked (not on the list). Before touching adminui: re-verify the concurrent lane is quiet.
- **Route gating is proven via the manifest, not the mux:** `Dashboard.Mount`'s "/" overview fallback makes unregistered paths 200, so "gated off" assertions belong on `Routes()` rows (the route-postures gate's own instrument).
- Failures, honestly: (1) ran the ambient `templ` v0.3.1114 BEFORE checking the flake's pinned v0.3.1100 — gotcha 10 repeated verbatim; caught by diffing (only version-stamp churn), reverted, regenerated with `.#templGen`; (2) an edit to config.go dropped the `if caps.EventSource {` line — caught by immediate re-view, repaired; (3) wrote the gated-off test against `mustTestDashboardMuxWithConfig` without reading the helper first (it auto-wires a journal; then `New` rejected a store-less config) — two wasted iterations; (4) trusted the session summary's "TAIL_BASELINE=8" instead of re-measuring — the tree raced to 18 while I was away. Always re-measure, never re-trust.

## e) VERIFICATIONS

`go test ./...` dashboardui (main + core) green · `golangci-lint run ./...` **0 issues** · coverage gate 73.4% ≥ 60 · `check-css-bundle-classes` 2 bundles exact · manifest pins trued (13 panels / 11 journal-only routes) · `go mod tidy` clean (`commandlifecycle/v4` direct) · templ stamps at v0.3.1100.

## f) NEXT (the remaining list, unchanged otherwise)

A8 (examples/basic partial-render page) → A13 (audit-log `-race` + BasicCommand embeds) → A16 (engine presets) → A18–A24 → A26 remainder (22-31) → CHANGELOG debt (usermgmt/systemadapter A1, dashboardui A2, health A3, A27-check) → A2-adminui leg (unparked) → Final battery (check-modules + the §c lint sweep + libs/index.html go-etag row + plan Table A trues + `/tmp` litter + badges check).

## g) QUESTIONS (carried, unchanged)

1. identity-model v4.16.0: cut the tag when their lane is quiet, or leave it to the concurrent session?
2. A17: sanction a nix-profile chromium screenshot path, or waive to structural verification?
3. (answered by directive) A2-adminui leg: proceeding — confirm if you'd rather still wait for their wave.
