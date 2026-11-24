# Status: adminui ↔ dashboardui integration — smart cross-link defaults

**Date:** 2026-11-24 06:42 CET
**Scope:** this session only — the adminui/dashboardui "don't work together" diagnosis and the smart-defaults fix built from it. No unrelated research.
**Head:** `e8ddc7f8` (auto-commit daemon; all session files committed).

## Verification scope (honest line)

**VERIFIED with commands this session:**
- `go build ./...` — dashboardui ✓, setup ✓, full go.work workspace ✓
- `go test ./... -count=1 -race` — dashboardui full suite ✓ (2.1s), setup full suite ✓ (11.3s, re-run 18.0s after refactor)
- `golangci-lint run` — dashboardui **0 issues**, setup **0 issues** (after fixing one golines + one cyclop finding)
- `nix run .#check-module-layers` — "OK: layering holds for 4 packages"
- New tests: 4 dashboardui nav tests + 7 setup cross-link tests, all green in the full suites
- Accent/theme parity: confirmed by code read (setup propagates ONE `AccentColor` to both panels; theme syncs via shared `localStorage 'theme'` through templ-components `ThemeScript`)

**NOT verified (and why):**
- E2E browser battery (chromium/firefox) — not run; the nav entry renders only when `AdminURL` is set and e2e seeds don't set it, so zero render delta expected; goldens in the unit suite pin the built-in sidebar and passed
- `nix run .#check-modules` composite (docs freshness, corpus budget, route-postures, hermetic isolation, …) — not run this session
- Coverage gate (`nix run .#coverage-gate`) — not re-run; coverage was 73.4% (>60) before this session and the added code is test-covered
- adminui module suite — module source untouched (only its *observed default* via setup changed)
- Renamer consumer behavior — still on old published pins; nothing to verify there yet

## The mandate and the diagnosis it answers

The user rejected my "the library principle says don't integrate" framing: **"never enforce defaults consumers might disagree with" does NOT mean "do not build good defaults and integrate things smartly BY DEFAULT with a possible way out if needed."** Correct. The session then built exactly that.

Root causes found last turn (evidence-backed, still true):
1. adminui rides `*usermgmt.Service` (identity domain); dashboardui rides go-cqrs-lite introspection (observability domain); zero cross-imports.
2. Cross-links were one-way: adminui had `Config.DashboardURL` → "Event store" entry; dashboardui had NO reverse link.
3. Both shells are different anatomies (adminui = templ-components `layout.Base`; dashboardui = bespoke boost shell) — the "two apps bolted together" feel.
4. Duplicated grammar documented "by design" (accent_color.go in both, comment: "the two UI modules share no module boundary").
5. Theme and accent were ALREADY unified under setup — parity confirmed, not a problem.

## What was delivered (all committed by the daemon)

1. **dashboardui `Config.AdminURL`** (dashboardui/config.go, dashboard.go): set → one "Identity admin" sidebar entry (`icons.Users`, verbatim href, `Active:false`), appended AFTER `ExtraNav` so consumer ordering stays stable — the exact mirror of adminui's `DashboardURL` contract. Empty → nothing (default).
2. **setup smart defaults** (setup/config.go `resolvePanelCrossLinks` + `DashboardAdminURL` field; setup/setup.go `buildDashboardConfig` passthrough): when both panels mount, each sidebar gets one entry for the other BY DEFAULT (admin → dashboard mount path, dashboard → admin mount path). `"-"` = explicit opt-out (the HealthPath convention). Disabled sibling → no default link. Custom value → verbatim. Applied after `normalizeRoutePaths` so hrefs carry the same trailing-slash form as the mount paths.
3. **Tests:** dashboardui — entry shape/verbatim/label/icon, empty-URL absence, ExtraNav-before-cross-link ordering, rendered built-in sidebar assertions. setup — `withDefaults` unit rows (default-on, custom paths, "-" opt-out, disabled sibling, custom-value-wins) + e2e wiring proofs through `bundle.Admin.Config().DashboardURL` / `bundle.Dashboard.Config().AdminURL`.
4. **Docs:** dashboardui README (Config list line + § Extra Nav Entries paragraph), dashboardui CHANGELOG `[Unreleased] ### Added` entry, setup README config table (`AdminDashboardURL` row re-defaulted; new `DashboardAdminURL` row).
5. **Parity checks (no change needed):** accent propagates from setup to both panels; theme already shared.

## What did I FORGET? What could be BETTER? (self-critique)

**Forgot / skipped:**
- **Repo AGENTS.md not updated** — the cross-link smart default is durable context; the route-posture/key-patterns bullet should name it. Not done.
- **adminui CHANGELOG/README silent** — adminui source is unchanged, but its *observed default behavior* under setup changed (an "Event store" entry now appears without opting in). A one-line adminui CHANGELOG note would have been honest bookkeeping. Not done.
- **Layout-mode test for AdminURL** — `ExtraNav` has `TestExtraNav_FlowsToPageMeta`; the new cross-link has no equivalent proving it reaches `PageMeta.Nav` under a consumer shell (it flows — `d.nav` feeds `meta()` — but it is asserted nowhere).
- **E2E not run** — discipline from the previous session (real-browser proof) not applied here; justified by "entry only renders when configured", but unproven in a browser.
- **Full gates not run** — `check-modules`, coverage gate, route-postures composite skipped; targeted checks only.

**Could have done better:**
- The setup e2e tests assert `Config()` accessors, not rendered HTML on the admin side (dashboard-side rendering IS covered by dashboardui tests). A rendered-sidebar assertion through `bundle.Handler()` with a super-admin session would be the stronger pin.
- `resolvePanelCrossLinks` could validate that non-"-" custom values look like paths/URLs (fail-fast), instead of trusting verbatim (a typo silently renders a dead link).
- I defaulted both directions independently; a single `PanelCrossLinks bool`-style master switch (with per-side override) might be a simpler mental model than two string knobs with a "-" sentinel. The chosen form mirrors the existing `AdminDashboardURL` precedent, but the alternative deserved a beat of thought.

**Could still improve (product-level):**
- Cross-link labels are hardcoded ("Event store" / "Identity admin") — no custom-copy knob.
- The real "feel like one product" work — one shared page anatomy — remains untouched (ADR-scale, v5-class).

## a) FULLY DONE

- dashboardui `Config.AdminURL` + "Identity admin" nav entry, full contract mirror of adminui's `DashboardURL` — built + tested (4 tests) + lint-clean.
- setup smart defaults (`resolvePanelCrossLinks`, `DashboardAdminURL` config field, `buildDashboardConfig` passthrough) — built + tested (7 tests) + lint-clean.
- Accent/theme parity verification — confirmed already unified; no change needed (this closed the suspected divergence cheaply).
- dashboardui README + CHANGELOG, setup README table — updated.
- Both module suites green with `-race`; workspace build green; `check-module-layers` green.

## b) PARTIALLY DONE

- **The integration itself:** navigation-wise the pair is now one click apart BY DEFAULT; visually they remain two different shells (bespoke dashboardui boost shell vs templ-components `layout.Base`). Nav integration done; chrome unification not attempted.
- **adminui documentation:** module unchanged by design, but the setup-level default change is undocumented in adminui's own CHANGELOG.
- **Consumer benefit:** renamer (and any consumer) gets this only after the next release train + pin bumps. Code done, delivery not.
- **Repo AGENTS.md:** not yet carrying the new default (durable-context gap).

## c) NOT STARTED

- Release train (dashboardui + setup + root pins) and consumer bumps (renamer CSS rebuild etc.).
- E2E spec for the cross-link (needs an e2e server config that sets `AdminURL`).
- `check-modules` composite + coverage gate re-run on this tree.
- HARVEST of section (f) items into TODO_LIST/ROADMAP (this report is the input; per skill contract the loop stays open until harvested).
- Repo AGENTS.md entry.
- adminui CHANGELOG note.
- Both prior status reports' §f item harvests (2026-11-24 06:09 report also still unharvested per owner instruction to WAIT).

## d) TOTALLY FUCKED UP

Nothing this session broke remains broken. Honest near-misses and hazards:
- **Two lint findings shipped before being caught** (golines long line in a test; `withDefaults` cyclop 15 > max 12). Both fixed: scoped `nix fmt` (gotcha 53 style) and an extraction to `resolvePanelCrossLinks` (which also improved the code — the helper now carries the rationale doc comment). Lesson: I should have designed the helper first; the complexity gate caught me one step later than it should have.
- **Foreign uncommitted WIP in the tree at report time:** `usermgmt/es_constants.go`, `usermgmt/es_event_catalog.go`, `usermgmt/es_projection_health_test.go` — NOT my files, NOT touched (gotcha 47 discipline: another session's in-flight work, likely the identity-model merge-events feature continuing). Flagged for the owner; the same two-agents-one-tree hazard as before.
- **LSP phantom diagnostics persisted all session** (5 stale errors in dashboardui timetravel files — known gotcha 14 class). Builds/tests were ground truth throughout; nothing actual behind them.

## e) WHAT WE SHOULD IMPROVE

1. Smart defaults exist only at the setup composition root; manual multi-panel mounters must wire two strings by hand. Fine for now; a doc paragraph in both module READMEs ("mounting manually? set AdminURL/DashboardURL") would help.
2. No fail-fast validation of cross-link values (verbatim trust + setup-level "-" resolution only).
3. Labels not configurable (i18n/custom copy).
4. AdminURL lacks the Layout-mode (`PageMeta.Nav`) pin its sibling `ExtraNav` has.
5. Admin-side rendered-HTML pin through the bundle (stronger than Config() assertions).
6. No e2e proof of the pair being one click apart.
7. Shell unification — the deeper fix — needs an ADR before anyone "just unifies" them (dashboardui's shell exists for boost/partial reasons discovered the hard way on 2026-10-18).
8. The duplicated accent grammar ("lives in both by design") is now the ONLY acknowledged duplication left in the pair — natural templ-components extraction candidate (cross-repo PR).

## f) Up to 50 things we should get done next

Brainstorm, not commitment — most non-top items are ROADMAP fuel (docs-health HARVEST applies routing rigor).

**Train-blocking / delivery (highest impact):**
1. Cut the release train so consumers get: time-travel real-data fix (prev session) + cross-link smart defaults (this session). ⬅ owner decision pending since 06:09 report
2. Bump renamer consumer pins after the train; rebuild its Tailwind CSS same-change.
3. Run `train-preflight` (quiescence + lint + hermetic battery + strict release-train + e2e battery as declared gate).
4. Run the FULL e2e battery (chromium blocking + firefox advisory) before the train — this session skipped it.
5. Run `nix run .#check-modules` composite on this tree before the train.
6. Run coverage gate for dashboardui + setup after the additions.
7. Post-push: `push-and-watch` + `verify-remote-tags` per the release playbook.

**Carryovers from the 06:09 report (still unanswered):**
8. Owner decision: foreign merge-events session — finish the feature here or hand it back (deciders/commands/projections/tests for the 5 merge/rename events are missing).
9. Owner decision: payload column on the aggregates timeline page (time-travel has it; aggregate timeline doesn't — asymmetry).
10. Harvest BOTH status reports' §f items into TODO_LIST/ROADMAP (docs-health HARVEST; TODO_LIST.md has been waiting since 06:09).

**Test/doc hardening for THIS session's work:**
11. Add `TestAdminURL_FlowsToPageMeta` (Layout-mode pin, mirroring ExtraNav's).
12. Add rendered-HTML admin-side cross-link proof through `bundle.Handler()` with a super-admin session.
13. Add e2e spec asserting the pair is one click apart (requires an e2e server setting `AdminURL`).
14. Validate cross-link values fail-fast in the modules (startsWithSlash / scheme / reject bare garbage).
15. Decide + document the tenant-admin-mode interaction (cross-link renders in ModeTenantAdmin too — intended? untested).
16. Note the setup-level default change in adminui CHANGELOG (observed-behavior change bookkeeping).
17. Add the cross-link default to repo AGENTS.md (route-posture/key-patterns bullet).
18. Document manual-mount wiring in both module READMEs ("not using setup? set the two knobs").
19. Consider custom cross-link labels (config knobs) — only if a consumer asks; otherwise YAGNI-reject explicitly.
20. Screenshot refresh showing the cross-linked sidebars (docs/screenshots — note byte-nondeterminism gotcha).
21. docs/agents-notes.md entry: why the "-" sentinel, why setup-level, why not module auto-detect.

**Hygiene batch from the previous session (still open):**
22. Reword time-travel INDEX subtitle ("see the state at each step" still overpromises).
23. Remove dead `data-slider-display` fallback in dashboardui layout_assets.go JS.
24. Add gotcha entries: paste-corruption token class; templ output-form trio (`&#34;` / bare bool attrs / display-form StreamID).
25. Pin foreign `UserState.IsMerged` with a unit test (guard the unblocked WIP from regression).

**Panel-pair roadmap (the deeper integration):**
26. Write ADR: shell unification options (shared anatomy vs consumer-owned chrome only) — v5-class decision.
27. Extract accent-color grammar into templ-components (kills the documented "lives in both by design" duplication; cross-repo PR).
28. Audit the remaining `audit.templ` name collision (adminui AuditLog vs dashboardui event audit view — same name, different domains; rename one?).
29. Consider a shared nav-item vocabulary module (NavLink shapes already nearly identical).
30. setup README: prose section "The panel pair" describing complementary roles + new defaults.
31. examples/setup-demo: assert/show both cross-links in a demo scenario.
32. integration_test: cross-link integration assertion (module-level e2e-lite).
33. Route-postures: add a guard test that nav cross-links can never become routes (keep the manifest clean).
34. Dark-mode + mobile visual check of "Identity admin" entry (unit goldens cover desktop HTML only).
35. Active-state semantics under custom layout for verbatim entries — pin like ExtraNav's contract.

**Fleet/library upkeep visible from this session:**
36. go-appkit v0.13.0 hold receipt (go-1.27.4 floor) — the only live fleet lag.
37. templ-components PR #101 popover-layer: merged upstream per open items — schedule the vendored adminui layer retirement on the next family train.
38. docs-gates `audit-all` red (393 vs pinned 322) — drain-or-re-pin decision sits in TODO_LIST P2.
39. erraudit blocking-promotion window (restarted at 364, closes 2026-12-21) — keep the advisory month green.
40. libs-report refresh for the cqrs-htmx row after the train (86 count / ~94 effective).

**Session-craft improvements:**
41. Design helpers before hitting complexity gates (cyclop caught me one step late).
42. Run the module composite once per session, not only targeted gates.
43. When a user challenges a stated principle, re-derive the design from the mandate, not from the old framing (this session's best move — do it first next time).
44. Capture "foreign files present" checks at session start (tree hygiene probe), not only when noticed at report time.
45. Add the AdminURL knob to the cqrs-htmx SKILL.md config notes so future sessions know it exists.

**Backlog adjacency (parked, not urgent):**
46. Time-travel payload perf: per-row `payloadBytes` render cost unbenchmarked.
47. Collapsed payload bytes still ship in DOM (page bloat) — consider on-demand fetch behind `hx-get`.
48. Back-button semantics after `hx-sync="this:replace"` — unexercised in e2e.
49. Firefox leg for the time-travel slider (one of the 5 known divergences is scoped off; verify the rest).
50. Renamer: decide whether adminui belongs there at all (needs a usermgmt identity stack — product question, see §g Q3).

## g) Questions I cannot figure out myself

1. **Foreign merge-events session:** `usermgmt/es_*` files are modified in the tree RIGHT NOW (uncommitted) and identity-model/fold.go carries the half-landed merge WIP from earlier today. Is that other session still active — should I leave everything alone, or is it dead and I should finish the merge feature (deciders/commands/projections/tests) on top of what's there?
2. **Release train now?** Two feature waves are parked unpublished (time-travel real data + panel cross-link defaults). Cut the train this session so the renamer consumer gets both, or review the UX in a browser first?
3. **Renamer product scope:** is the admin panel (identity ops) ever wanted in file-and-image-renamer — which would mean adding the usermgmt identity stack there — or is it dashboard-only forever? (This decides whether the new cross-link defaults ever fire in your most-watched consumer, and whether the hardcoded "Identity admin" label needs custom copy.)
