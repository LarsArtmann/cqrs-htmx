# Round-5 Status — Phase 4 COMPLETE, All Gates Green, TODO_LIST Truth Restored

**Session:** 2026-09-20 ~13:30–15:00 CEST (resumed after the 13:22 pause; user re-issued "keep going until everything works")
**Standing order:** "GET SHIT DONE! The WHOLE TODO LIST! DO NOT STOP UNTIL THE ENTIRE LIST IS FINISHED and VERIFIED"
**Plan:** `docs/planning/2026-09-20_09-05_all-todos-pareto-comprehensive-execution-plan.html` + adminui micro-plan M001–M101

---

## a) FULLY DONE (this session, all verified)

### Phase 4.1 gate (M078–M084) — CLOSED GREEN
1. Codegen 0-drift (after daemon absorbed the regenerated `_templ.go` set — check-codegen is git-diff based, so it reds until the daemon commits).
2. Lint 0 issues; hermetic tests PASS after fixing the third confirm test (it queried LIST pages; the destructive buttons live on DETAIL pages — rewritten against `svc.Register` + `mustCreateTenant` fixtures; also `tenant.ID.Get()` returns string, no `.String()`).
3. CSS rebuilt; `margin: auto` added to the vendored dialog block (**real bug**: Tailwind preflight zeroes margin, pinning native `<dialog>` to the viewport's top-left — the modal would have shipped corner-pinned).
4. **Real PRE-EXISTING bug found + fixed: adminui's CSRF injection was a no-op.** `htmx.config.headers` is not an htmx v2 option — every htmx BUTTON POST 403'd under CSRF middleware (forms were safe via hidden input; no browser gate had ever clicked a write button before). Fixed on the sanctioned `htmx:configRequest` hook; verified live (header sent, delete 200, redirect).
5. verify gate **45/45 PASS** (was 33): modal opens on Delete, title/body filled, Esc/Cancel/backdrop cancel without any request, Confirm issues the POST and the user is gone (q=team60 empty).
6. 20 re-shots adjudicated with a **noise-baseline method** (second same-code run = AE 0 on time-free pages): every p3-prev→p3 delta is the Created column's relative time text ("just now" vs "N minutes ago") shifting auto-table column edges. HTML diff (old binary via worktree): byte-identical modulo the invisible closed dialog + csrf meta value. Modal causes ZERO visual regression. + 2 open-modal shots (light: 50% dim backdrop dominant; dark: red Confirm button visible).

### Phase 4.2 (M085–M087) — user menu — CLOSED
7. Header identity cluster → library `display.Dropdown` (native Popover API; email trigger + avatar sibling — `DropdownProps` has no custom-trigger slot, recorded as upstream candidate; mobile keeps the plain sign-out link). Deterministic `ID: "admin-user-menu"` (EnsureID randomizes un-pinned IDs).
8. **Second upstream bug found + worked around: templ-components v1.18.1's popover positioner never runs.** It registers `document.addEventListener("toggle", …)` WITHOUT capture; the popover toggle event does not bubble (verified empirically: bubble 0, capture 1) — every library Dropdown/Popover opens at the viewport corner. admin.js now carries a vendored capture-phase positioner (mirror of the library's algorithm). To be reported upstream (TODO_LIST upstream-asks (e)).
9. Popover animation CSS vendored (same pattern as the dialog block). Tests PASS (dropdown markup, popover attrs, 2 sign-out hrefs, no inline handlers); live flow verified (opens anchored at x:1032, light-dismiss, sign-out navigates); light+dark open-menu shots.

### Phase 4.3 (M088) — theme spike — DOC WRITTEN, M089 GATED
10. `docs/research/2026-09-20_theme-toggle-strategy-spike.md`: Option A (class-strategy `dark:` migration — full rewrite of the token architecture) vs **Option B (recommended: `data-theme` toggle over the existing token flip, ~30 lines)** vs defer. M089 stays a user sign-off gate (plan guardrail) — question in §g.

### Phase 4.4 (M090–M093) — PolledRegion — CLOSED
11. `dashboardStatsRegion` wraps the stats grid in `htmx.PolledRegion` (30s, ShowTimestamp); `GET /partials/stats` (session-gated) re-renders the REGION itself (outerHTML contract — bare children would kill polling). Poll-vs-SSE decision documented inline (SSE = control plane, never stats). One handler-structure bug during refactor (stray brace) caught by build + fixed.
12. Tests PASS (partial fragment + trigger attrs, 401 gate, dashboard wiring); live-verified: 2 polls fired in 31s.

### Phase 4.5 (M094–M096) — dark sweep — CLOSED, ZERO DRIFT
13. Scripted sweep (computed white-bg detection under dark scheme) across all 10 pages: **CLEAN everywhere** — the token architecture holds; no bridge extensions needed (M095 vacuous by success).

### Phase 4.6 (M097) — polish — CLOSED
14. Danger-zone eyebrow ("IRREVERSIBLE ACTIONS" + title). **Real find: `.admin-toggle` (mobile hamburger) had NO styling at all** — hit target, hover, and focus ring added (display scoped inside a max-lg media query: unlayered CSS beats Tailwind's layered `hidden` utility and would have leaked the hamburger onto desktop). Focus-ring audit: hand-rolled buttons already use the library focus-visible pattern. Spacing: no change (zero drift; "revert if worse" cuts against churn).

### Phase 4.7 (M098–M100) — final gates + docs — CLOSED
15. `nix run .#build` ✓ · `.#test` ✓ (all modules) · `.#lint` ✓ 0 issues · `.#coverage-gate` ✓ (**adminui 70.1%**, gate ≥66) · `.#check-codegen` ✓ · `.#check-templates` ✓ · `.#check-modules` ✓ (after raising the root dep budget 18→19 — the morning sweep's DecodePayload work made `go-codec` a legitimate direct dep; budget comment updated).
16. bench-spike **REFUSED (6th documented refusal)**: load 19.9 vs limit 8; setup bench paths untouched (only adminui + admin-demo changed).
17. CHANGELOG `[Unreleased]`: adminui Phase 3 + Phase 4 entries + the round-2 bundle (client metadata, API exposure, SSE hardening) that had never been changelogged.
18. AGENTS.md adoption table: +`navigation.Pagination`, +`forms.FilterInput`, +`display.Modal`, +`display.Dropdown`, +`htmx.PolledRegion` rows with their gotchas.

### Carried items — CLOSED
19. **e2e spec check-in:** `e2e/tests/admin-behavior.spec.ts` — 9 Playwright tests (pagination counts/clamping/compose, partial-swap contract, spinner, full modal flow incl. CSRF'd delete, dropdown anchor + light-dismiss, polled region + partial, favicon 204). **9/9 PASS** via `bun x playwright test tests/admin-behavior.spec.ts`. Playwright config gained a second webServer (admin-demo on :18930; both servers now pin `GOTOOLCHAIN=go1.27.1` — even e2e/server needed it outside the devShell). Two config lessons: webServer `cwd` is relative to the CONFIG dir (`../examples/admin-demo`, not `../../`), and a wrong cwd surfaces as `spawn /bin/sh ENOENT`.
20. **TODO_LIST truth pass:** retired routed-gaps (all landed), hygiene micro-pack (samber-linter CLOSED — repro passes, 0 findings, no exclusion), findings-gate STALE item (`.buildflow.yml` has been `fail_on: critical` since 2026-09-18 — verified), OTel meta-note, adminui program item (COMPLETE). Upstream-asks (2) resolved (`cqrs-upgrade --workspace` shipped; used today). New adminui theme-toggle ❓ item carries the M089 gate. Header line rewritten for rounds 1–4.
21. **V007 re-surface:** `cqrs-upgrade -dry-run --workspace` from the sibling — ONE command (the 22-manual-run class is dead): **78 findings** (68 layout-planning / 6 auto-projection / 4 system.New; per-identifier counting now). P3 item updated.
22. **appkit ADR-001 (b)–(f) assessment** (recorded in the P3 item): (b) fold-into-RunHandler defers to v5 (silently changes the default path); (c) /health coexistence is by design (`HealthPath: "-"` opt-out) — document, don't automate; (d) chain dedup NOT safe (bundle owns CSP nonces, appkit doesn't); (e) logging isolated by `LogLevelError` in the bench — marginal seam; (f) Addr() on consumer demand (racy post-Start).

---

## b) PARTIALLY DONE (nothing from this round's scope)

None — every phase and carried item that did not require user input is complete.

## c) NOT STARTED (require the user; see §g)

1. M089 theme sign-off → then Option B implementation (~30 lines).
2. M101 final push authorization (commits are daemon-absorbed; nothing pushed).

## d) MISTAKES MADE (all caught + fixed in-session)

1. First confirm-test draft asserted against LIST pages (buttons live on DETAIL pages) — ground-truth-before-assertion violated, then honored.
2. `tenant.ID.Get().String()` — Get() already returns string.
3. handler_dashboard refactor left a stray brace (audit append inside the else branch) — caught by build immediately.
4. multiedit on TODO_LIST accidentally deleted the DataStar Tier-4 item (it shared the old_string block) — restored on sight.
5. Playwright webServer cwd `../../examples/admin-demo` (one `../` too many) → misleading `spawn /bin/sh ENOENT`; the first admin-spec run also exposed that e2e/server needs the GOTOOLCHAIN pin outside the devShell.
6. My verify script's "user gone" check was initially vacuous (page-1 table can't contain team60 anyway) — replaced with a q=team60 empty-table assertion.

## e) IMPROVEMENTS FOR NEXT TIME

1. The noise-baseline method (second same-code shot run) should be the FIRST move of any screenshot adjudication — it converts "probably timestamps" into proof.
2. Worktree-build the previous commit to diff served HTML when pixels differ but the cause is unclear — resolved the table-column shift question in one step.
3. The adminui behavior gate is now checked in; new adminui UI work should extend `admin-behavior.spec.ts` instead of /tmp scripts.

## f) SESSION STATE

- Demo server for the /tmp harness still running on **:18932** (`/tmp/adminui-baseline/admin-demo-p3`); the e2e spec self-starts its own on :18930. Kill either freely.
- All work daemon-absorbed (no manual commits, per rule). Tree clean; every gate re-verified after the last doc edit.

## g) QUESTIONS FOR THE USER (the only blockers left)

1. **Theme (M089):** implement Option B (token-flip + `data-theme` toggle, ~30 lines, recommended), Option A (class-strategy migration), or defer? Doc: `docs/research/2026-09-20_theme-toggle-strategy-spike.md`.
2. **Push (M101):** push master? (No push was made; force-push items from round 3 also still need explicit approval or a permanent waiver decision.)
3. **Upstream filings:** shall I file the two templ-components bugs (popover positioner toggle-no-capture; optionally the DropdownProps trigger slot + the earlier ListNote/Grid/CopyButton asks) via the jj-fork-PR workflow?
