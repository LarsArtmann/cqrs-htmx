# Status: adminui superbification — 118-idea brainstorm, Pareto plan, waves 1–6 landed, wave 7 mid-flight

**Session window:** 2026-11-06 ~00:40 → 01:57 CET · **Author:** this session (Crush), adminui lane
**Mission (open):** "How can we make adminui/ SUPERB — brainstorm 100+ ideas, break into steps, execute and verify one at a time, repeat until done."
**Verdict:** the brainstorm and the plan are COMPLETE and delivered (118 ideas / 14 angles / Pareto-tiered, HTML report written and validity-checked); execution waves 1–6 are CODE-COMPLETE AND TEST-GREEN in the working tree (18 adminui files, +2,961/−150 lines, daemon-committed in 13 heuristic commits); **wave 7 is a RED BUILD** (`icons.ChartBar` undefined, one line) and the gate battery (lint / cqrs-lint / route-postures / css-bundles / coverage / check-modules) has NOT been run on the finished waves — that verification gap is the honest headline. Docs (README/CHANGELOG/demo) not yet updated.

## a) FULLY DONE (receipts)

1. **Research pass complete.** adminui fully read (routes/handlers/templ/tests/config); the `*Service` capability surface inventoried — **25 exported capabilities had no panel surface** (the brainstorm's fuel); prior plans/TODO/status folded in (ExtraNav #21, session-lifecycle labels #30, BroadcasterHealth A2/B15, A8 dogfood); gate mechanics mapped before editing (route-manifest golden, coverage floor 66%, `nix run .#gen`, CSS bundle gates, corpus/tail ratchets).
2. **The brainstorm: 118 ideas across 14 angles** (missing CRUD, audit superpowers, dashboard enrichment, users/tenants polish, config/composability, honest-UI/realtime, a11y, performance, code quality, docs, security, creative, ops/integrations). Delivered inside the plan report, idea-numbered, impact/effort-tagged.
3. **The plan:** `docs/planning/2026-11-06_00-55-SUPERB-adminui-superbification-pareto-plan.html` (76 KB, self-contained Bauhaus dark; 1%/4%/20% Pareto tiers; 27-task comprehensive plan; micro-steps; verification battery; errata; D2 execution graph rendered and inlined as SVG). **Structural validity machine-checked**: exactly one `</head>` / one `<body>`, zero unclosed tags, title replaced, every CSS class used exists in the template's stylesheet. (The HTML parser warnings were self-closing-SVG artifacts, not defects.)
4. **Wave 1 — user lifecycle (the 1% tier).** `GET /users/new` + `POST /users` (create via `Register`, session discarded honestly, passwordless hint in-form), `POST /users/{id}/display-name`, `POST /users/{id}/email`. Route table rows with Why; manifest golden updated in-change; handlers follow the tenantCreate error pattern.
5. **Wave 2 — platform roles.** `POST /users/{id}/roles` + `/roles/revoke` through the DURABLE `GrantRoles`/`RevokeRoles` path (ADR-0057; never `AddGroupPolicy`); Platform-access card (checkboxes pre-set from current wildcard roles, Revoke all); **self-lockout warning** when the target is the session user; **HasAdminGrant("\*") dashboard banner** mirroring setup.New's boot diagnostic.
6. **Wave 3 — credentials.** `POST /users/{id}/credentials/{credential}/delete` (base64url path segment); Passkeys card (name / added / backup-state badges) with **escalating confirm texts** (last-passkey + no-provider = "locked out entirely"); `POST /users/{id}/resend-verification` with a graceful not-configured error path.
7. **Wave 4 — audit superpowers.** `audit_labels.go`: one label+severity map (16 actions, human labels, badge tones: destructive=error, refused login=warning, sessions=quiet) reused by audit page, dashboard, and timeline; **omnibox filter** (action/email/aggregate/label substring) with HTMX partial swap and honest filtered pagination; login_failed `Detail` rendered; absolute-timestamp hover titles; **per-user Activity timeline card** on user detail (≤8 recent, "View all N" deep-link into the filtered audit page).
8. **Wave 5 — dashboard enrichment.** Adoption stats (Verified %, MFA %, Suspended tenants — computed, not decorative, "—" for empty sets); recent-activity rows **link to the entity**; the no-admin-grant banner (disappears once a grant exists — proven by test).
9. **Wave 6 — users list.** Status filter chips (Verified/Unverified/MFA on/Has provider; `aria-pressed`; compose with search; chips preserve `q` in both toggle directions; unknown filter values refused); linked-**provider badges** in the status column.
10. **A gotcha-25-class bug fixed before it shipped:** every identity use of `AuditEntry.AggregateID` now goes through `.Get()` (bare string) — `String()` is the StreamMarker display form and produced wrong hrefs/matches in 4 sites (dashboard link, audit table, timeline match, filter). Caught by the new tests, not by review.
11. **Tests: 21 new tests / 3 updated files, ALL GREEN** as of the last full run (pre-wave-7): `user_lifecycle_test.go` (14), `audit_superpowers_test.go` (7), `users_filter_test.go` (4) + manifest golden (+8 rows) and flow coverage. Full-module suite `ok ... 6.5s`.

## b) PARTIALLY DONE

1. **Wave 7 (composability) — RED BUILD, ~80% coded.** Done: `Config.ExtraNav []NavEntry` (+ validation), `NavEntry{Label, Href-verbatim, Icon}`, `Config.DashboardURL` (cross-link "Event store" nav entry), `buildNav(config)` rewrite, `navItem.Verbatim`, `sidebarItems` honoring it. **Broken:** `handler.go:94` references `icons.ChartBar`, which does not exist in templ-components v1.27.0 (my unverified-symbol mistake — verify-before-filing applies to my own code too). **Missing:** the ChartBar→valid-icon fix, templ regen, wave-7 tests, `setup.Config` passthrough (`AdminExtraNav` / `AdminDashboardURL`).
2. **Verification battery NOT run on the finished waves.** Only `go build` + `go test` + `go vet` were run per-wave. NOT yet run: `nix run .#lint`, `.#check-cqrs-lint`, `.#check-route-postures`, `.#check-css-bundles` + `.#check-css-bundle-classes` (**new Tailwind classes were added — amber banner, chips, hover:underline — the bundle MUST be rebuilt or classes ship missing**), coverage threshold (66% floor), `.#check-modules`.
3. **Docs not yet updated:** adminui README (What-you-get + config table + create-user passwordless note), CHANGELOG `[Unreleased]` entries, `examples/admin-demo` showcase of the new flows, plan-report status column.
4. **Working tree carries 3 files from a PARALLEL session** (`sse_broadcaster.go` modified, `sse_drop_stats.go`/`_test.go` untracked) — present since session start, not touched by me, left for their owner.

## c) NOT STARTED (parked in the plan's 20%/tail tiers with owners/conditions)

1. Import/export users (#9, #10); Bots page (#11, #12 — blocked on a FindByOwner accessor); impersonation (#51 — ADR + banner + audit needed); tenant search/filter (#39).
2. BroadcasterHealth sync indicator (fleet-libs A2/B15, wave 2 of that plan); SSE nudge for polled regions (#31); offline-queue depth badge (#54).
3. Keyboard shortcuts (#89), command palette (#90), view transitions (#91), density toggle (#92), print stylesheet (#93).
4. A11y pass: skip-link (#57), aria-current (#58), captions (#59), focus-visible audit (#60), reduced-motion (#61); form error-summary UX depth (#63).
5. Pagination windowing (#21), sort columns (#34), row-count selector (#35); benchmark spike (#65); ETag partials (#66).
6. Consumer Layout override (#47 — v5-shaped, ADR first); page visibility toggles (#46); i18n seam; footer/brand config (#48, #49).
7. TODO_LIST harvest of parked ideas; plan-report status update; wave-tag/release-train decision.

## d) TOTALLY FUCKED UP (all self-caught; all but one fixed in-session)

1. **`icons.ChartBar` does not exist** — used without verifying the constant; wave 7 ships a red build. The exact class I enforce on others (verify symbols before use) and skipped myself. CURRENTLY UNFIXED.
2. **Gotcha-25 identity bug in MY new code** — 4 sites used `AggregateID.String()` (display form) where identity was needed: entity links would have 404'd, the timeline never matched, the aggregate filter never matched. The repo has a gate, a numbered gotcha, AND an AGENTS row about this; I still wrote it wrong first. Tests caught it.
3. **`dashboardData.BasePath` never set** despite adding the field (multiedit targeted the wrong file; the "applied" edit landed in the templ, not the handler) — rendered `/users/...` links missing `/admin`. Caught by test, fixed.
4. Smaller self-inflicted wounds, all fixed: literal `\n` pasted into a templ Go-string; an inverted old/new edit; a stray `</div>` in platformAccessCard; invented test helpers (`VerifyEmailWithTokenForTest`, `streamIDOf`) that don't exist; a redundant toast before `writeActionError` (which toasts itself).
5. **Process miss:** I batched verification at wave ends instead of gofmt+build after each edit batch — most of §d would have been caught at edit time.

## e) WHAT TO IMPROVE (my own craft, next session)

1. Verify third-party symbols (icons.Name constants, component props) with `go doc`/grep BEFORE writing them — zero excuses, it's a 5-second check.
2. Per-edit-batch `gofmt` + `go build`; per-wave gate slice (`check-route-postures`, css-class check) immediately, not at the end.
3. When a multiedit "partially applies", STOP and re-read both files before continuing — two of §d's bugs were mis-targeted edits discovered late.
4. Write the manifest-golden change and the route-table row in the SAME edit — they are one fact in two places.
5. Run the HTML splice through the parser at WRITE time (I only validated it when asked).

## f) NEXT THINGS (35, ordered)

1. Fix `icons.ChartBar` → a verified v1.27.0 constant (grep icons package first).
2. `nix run .#gen`; full adminui `go test`.
3. Wave-7 tests: ExtraNav validation (empty label/href rejected), verbatim href rendering, DashboardURL nav row absent when empty.
4. `setup.Config`: `AdminExtraNav`/`AdminDashboardURL` passthrough + setup test.
5. `nix run .#build-adminui-css`; then `.#check-css-bundles` + `.#check-css-bundle-classes` (new classes must land in the SAME change).
6. `nix run .#lint` (fix everything in the 18 changed files).
7. `nix run .#check-cqrs-lint` (must-style helper rules on new code).
8. `nix run .#check-route-postures` (manifest golden green across modules).
9. Coverage: `bash scripts/checks/check-coverage-threshold.sh adminui` (floor 66%).
10. `nix run .#check-modules` (the aggregate).
11. adminui CHANGELOG `[Unreleased]`: waves 1–7 entries.
12. README: What-you-get list + config table (ExtraNav, DashboardURL) + create-user passwordless honesty note.
13. admin-demo: exercise create-user → grant roles → audit filter → timeline; confirm tenant-admin panel still renders.
14. `check-session-route-wrappers` + `check-streamid-identity` (should be untouched-green; run to prove).
15. Update plan HTML: mark waves 1–6 done, wave-7 remainder, link this report.
16. TODO_LIST harvest: park the plan's 20%/tail items as rows (todo-hygiene gate after).
17. Decide + execute the wave-tag/release-train step (see §g1).
18. BroadcasterHealth beside the sync indicator (fleet-libs A2/B15 — the recorded follow-through).
19. Import users (CSV/JSON upload + honest ImportResult failure list).
20. Export users (CSV/JSON download endpoints).
21. Tenant list search (FilterInput, same pattern as users).
22. Users list sort (created/email/display name).
23. Pagination windowing on all list pages (library Pagination window prop — upstream check).
24. Audit CSV export of the filtered view.
25. Per-tenant audit timeline on tenant detail.
26. "Needs attention" dashboard panel (suspended tenants, unverified admins).
27. Dashboard quick actions row (create user / create tenant).
28. Bots read-model accessor upstream ask (FindByOwner) → Bots page + bot actor names in audit Who column.
29. Impersonation ADR + config-gated flow (reason modal, banner, session lifecycle audit).
30. Keyboard shortcuts (g u/g t/g a, `/` focuses search) — nonce'd, progressive.
31. A11y pass: skip-link, aria-current, captions, focus-visible, reduced-motion.
32. Render benchmark: 200-user page (bench-spike pattern).
33. SSE control-plane nudge for polled regions (no payload contract).
34. Consumer Layout override ADR (v5 material, cluster with dashboardui's seam).
35. Re-run `docs-gates` battery + freshness (this report is the newest; the next session must be covered by it).

## g) QUESTIONS (3 — cannot decide from the repo alone)

1. **Release train:** waves 1–7 are consumer-visible feature additions to adminui (`+` fields are backward-compatible). Cut them as ONE adminui minor train when the battery is green (wave-tag → bump consumers), or hold for a v4.17.0 batch together with the parked import/export + bots work? The train rules are strict, but batch composition is an owner call.
2. **Wave-8 priority:** import/export (#9/#10), bots (#11, needs an upstream accessor ask), or impersonation (#51, security-sensitive)? They serve different consumer personas (migration vs automation vs support) — I cannot read demand from this repo.
3. **Plan artifact format:** the pareto-planning skill mandates the HTML report (done), but EVERY prior plan in `docs/planning/` is Markdown. Keep the HTML (skill-native, richer) and accept the format outlier, or should I cut a Markdown twin/replace so the planning corpus stays homogeneous? Affects whether docs-freshness/links gates need a new pinned entry.

---
*Point-in-time snapshot. Living truth: TODO_LIST.md / FEATURES.md / CURRENT_FOCUS.md. The wave-7 build break and the un-run gate battery are the two facts a successor session must absorb first.*
