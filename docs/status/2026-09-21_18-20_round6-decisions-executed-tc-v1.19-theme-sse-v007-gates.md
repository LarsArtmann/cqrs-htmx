# Round-6 Status — Design-Tree Interview Executed: templ-components v1.19.0 Cut, Theme B, /sse Option B, V007 ADR; Gate Tail In Flight

**Session:** 2026-09-21 ~15:50–18:20 CEST (single session, resumed from the 14:03 docs-health sweep)
**Mode:** paste_2 design-tree interview → user decisions locked → full execution of the plan of record
**Plan of record (user-confirmed):** 1) templ-components v1.19 train → 2) adminui theme Option B → 3) setup `/sse` filter option → 4) V007 spike → 5) cut cqrs-htmx **v4.12.0**; force-push history rewrites WAIVED; `stack/v4` to be REMOVED in go-cqrs-lite v5.

---

## a) FULLY DONE (this session, all verified)

### Interview (design tree, 3 rounds + confirm)
1. All decisions settled via the native questions tool: objective=clear gated tail; theme=**B**; `/sse`=**B** (scoped option); git=**push master + waive force-push rewrites**; workstreams authorized = templ-components fix + V007 spike; `stack` removed in v5 (user's words: "metaengine + system/ in go-cqrs-lite is the new way"); order=tc→theme→sse→V007; release=v4.12.0 after batch.
2. **cqrs-htmx master pushed** (29 round-5 commits, `0187f46c..e441963f`); force-push backlog routed to ROADMAP "Not Planned" (TODO_LIST items removed, ROADMAP entry added).
3. **Decision recorded in go-cqrs-lite** ROADMAP §v5 Unification: stack/v4 removed entirely in v5; the old "decouple from metaengine" ask superseded (sibling-repo edit, daemon-absorbed).

### templ-components v1.19.0 — authored, verified, tagged, PUSHED
4. **Two real popover-positioner defects fixed** in `display/shared.go`: (1) `toggle` listener registered without capture (toggle doesn't bubble → panels at viewport corner); (2) NEW, discovered via the visual suite's nondeterminism: a panel already open when the script executes never got positioned — added attach-time self-heal (`[popover]:popover-open[data-tc-anchor]` positioned at script exec; also fixes HTMX-swapped content).
5. **`DropdownProps.Trigger` slot added** (custom trigger content inside a functionally-wired button — keeps `popovertarget`/`aria-haspopup`/`data-dropdown-trigger`/`aria-label`, no default classes) + 2 subtests + goldens + FEATURES row + CHANGELOG.
6. Found cqrs-htmx's "v1.19 = sharp cards" heads-up was **STALE** (sharp cards shipped in v1.18.1, CHANGELOG line 406) — corrected in cqrs-htmx AGENTS.md.
7. Repo plumbing fixed in templ-components: `cmd/tc` scaffolder `_sources` re-sync; go-directive tug-of-war resolved (tidy canonicalizes `1.26.0`, buildflow's `go-version-auto-configure` normalizes to bare `1.26` → workspace skew poisoned its own go-lanes) — directives pinned at **1.26.0** across root/website/visualtest/go.work AND `go-version-auto-configure` skipped in `.buildflow.yml` (same documented class as the `go-structure-linter` skip, TODO #231).
8. Full verification: per-module suites, `nix run .#lint` 0 issues, `nix run .#visual` full suite GREEN (open-state goldens unchanged — they were already anchored renders), `ci-repro.sh --lint --website` **VERDICT: PASS**. Master pushed; **v1.19.0 cut** via `release.sh` (bumped requires, stripped replaces, one release commit `53861ccf`) — 7 tags pushed and verified via ls-remote.

### cqrs-htmx adopts v1.19.0
9. All 12 consuming `go.mod`s bumped (hermetic tidy+build+vet green per module); BOTH CSS bundles rebuilt in the same change (adminui changed; dashboardui byte-identical).
10. adminui's vendored capture-phase positioner **DELETED** from `admin.js`; identity menu now renders avatar+email INSIDE the trigger via the `Trigger` slot.
11. **Two real bugs found & fixed on the way:**
    - e2e pagination failure (65 rows vs 50): the devShell exports `GOWORK=off`, so the admin-demo webServer silently compiled against **published** adminui (pagination is unreleased). `playwright.config.ts` webServers now pin an absolute `GOWORK` (config-dir-relative `path.resolve`) — shell-independent.
    - After removing the workaround the menu anchored at 0,0: the library's inline positioner script was CSP-blocked — the Dropdown call never passed `Nonce`. Fixed: `BaseProps{ID, Nonce: p.Nonce}` (gotcha recorded in AGENTS adoption table).
12. e2e: admin-behavior 9/9, then FULL suite **57/57**.

### adminui theme Option B (M089 — owner-approved)
13. Implemented with the LIBRARY's own components (less code than the spike's hand-rolled variant): `layout.ThemeScript` in `headExtras` (pre-paint `.dark` + `color-scheme` from `localStorage 'theme'`/system pref) + `layout.ThemeToggle` in `topBar` (`role="switch"`, aria sync).
14. `tailwind.css`: `@custom-variant dark (&:where(.dark, .dark *))` (all `dark:` utilities follow the class) + dark token block moved to `html.dark` (media-query fallback removed; no-JS degrades to light; gray-800/900 re-pin rides along).
15. New e2e gate test: toggle flips `.dark`, syncs `aria-checked`, persists across reload — **admin-behavior 10/10**; screenshots **37/37 total, ZERO visual drift** (auto mode byte-identical, exactly as the spike predicted). CHANGELOG + AGENTS + TODO updated.

### setup `/sse` filter option (endpoint-shape decision RESOLVED)
16. `setup.Config.SSEFilter func(sse.Event) bool` threads `transport.WithSSEFilter` into the SSEPath handler — live + replay scoped, fail-closed; nil keeps the documented full-feed contract; DataStar feed unfiltered. go-sse became a direct setup dep.
17. `TestBundle_SSEHandlerFilterScopesReplay` green; full setup suite green. Decision one-pager marked RESOLVED (Option B); recipe added to `docs/guides/sse-and-datastar.md` §Scoped Feeds; TODO item updated.

### V007 spike (clusters 2+3) — done with a pivot
18. Worktree `../cqrs-htmx-v007` (branch `v007-spike`), merged to master (`99be42f7`).
19. **Key finding: the 2026-09-10 spike plan's premise aged out.** `NewEventSourcedSetup` is ALREADY stack-free; the remaining surface is (a) `MaterializeProjection` — zero in-repo consumers, (b) the `//go:build ignore` SQL setup templates whose value IS ~200 lines of preset wiring (pragmas/schema/durability) whose v5 replacement is the systemadapter declarative path (equivalence-tested), (c) the `Bundle` field.
20. Executed the established v5-removal pattern instead of a rewrite: `// Deprecated:` markers (adapter family + `eventSourcedSetupCore.Bundle`), deprecation notes in the 3 templates, **ADR-0051** (incl. the cluster-1 go/no-go criterion: metaengine layout-planning must cover secondary-index semantics + declarative hydration), v5-removal-inventory §5b, ADR INDEX row.
21. Verified in the worktree: usermgmt suite green, lint 0 issues, `check-templates` green.

### Gate ladder (so far)
22. `.#build` ✓ · `.#test` ✓ (0 fails) · `.#test-race` ✓ · `.#coverage-gate` ✓ · `.#check-codegen` ✓ · `.#check-templates` ✓ · `.#lint` ✓ 0 issues/15 after fixing one real pre-existing finding on sight: `HeaderClientID = "X-Client-ID"` → canonical `"X-Client-Id"` (round-5 leftover; wire-identical).
23. `check-modules`: broken markdown link fixed (archived status file move — planning doc repointed to `../status/archived/…`); V006 false-positives suppressed in datastar + systemadapter (documented per-module-train rationale) and root's suppression re-anchored for cqrs-lint ≥4.11 (tool drift; second anchor comment added).

---

## b) PARTIALLY DONE

1. **`check-modules` still red** — dependency budget: root 20/19, datastar 7/6 "OVER BUDGET". Mid-flight discovery at interruption time: `scripts/check-dep-budgets.sh` counts **tab-indented comment lines** inside require blocks as deps — my `//cqrs-lint:ignore(V006)` suppressions (own line, tab-indented, no `// indirect` marker) each added +1. The "over budget" is almost certainly the comment artifact, NOT real dep growth (root's direct block counts 18; datastar 6). FIX PENDING: exclude comment lines in the awk or un-indent the suppressions.
2. **`check-cqrs-lint` still red** — three disentangled causes: (a) REAL V006 in datastar/systemadapter → FIXED; (b) root V006 re-anchored by the drifted PATH tool (4.11.2 vs documented 4.8.1) → FIXED; (c) remaining ~25 warnings = the AGENTS-documented pre-existing tool-drift triage set (C009 `panic()` in dashboardui/assets.go, e2e/server A023 custom snapshot store, P008 polling, …) — NOT triaged this session (predates it); (d) the nix gate itself flaked across runs (health/auditlog failed unreproducibly in run 1; all-modules failed in run 2 — writeShellApplication PATH / nix eval-cache contention, the documented flake class). Manual loop with correct env: root fails only on (c).
3. **v4.12.0 family train NOT cut** (was next after the gates): no verify-tag run, no tags, master not re-pushed since the phase merges.
4. Docs tail: CHANGELOG entries exist for tc-train/theme/sse; **missing**: V007 ADR+deprecations entry, canonical-header fix, V006 suppression notes, and the AGENTS cqrs-lint row (tool drifted 4.8.1→4.11.2, new anchoring, remaining warning set).
5. bench-spike: not attempted this session (machine-gated; load was ~10–20 vs limit 8 all day).

---

## c) NOT STARTED

1. v4.12.0 train mechanics: `scripts/verify-tag.sh` ×15 modules, family require bumps, tag push, release notes.
2. `.#test-fuzz`, `.#test-flake`, `nix flake check --no-build` on the final tree.
3. Full e2e rerun on the FINAL tree (post theme+/sse+V007 merges only admin-behavior+screenshots reran; sync/dashboard/axe/axe-probe not rerun — nothing touches them, but the discipline says rerun).
4. cqrs-lint warning triage (~25 findings) and/or pinning the gate's tool version.
5. AGENTS.md cqrs-lint row refresh; `datastar-demo` rebrand (explicitly NOT authorized); cqrs-lint Go distribution (NOT authorized).

---

## d) TOTALLY FUCKED UP (all caught + recovered in-session; nothing shipped broken)

1. **Golden mass-rewrite with wrong fonts**: ran `go test -update` on visualtest WITHOUT the flake app's pinned `FONTCONFIG_FILE` → rewrote dozens of goldens with fallback-font rendering. Recovered via `git restore`; the correct path (`nix run .#visual -- -update`) was documented IN THE FLAKE — should have read it before the manual run.
2. **Stray scaffolder file**: blind `cp display/shared.go cmd/tc/_sources/display/` created a file that never existed there (the `_sources` mirror is `.templ`-only). Removed after the drift-guard test pointed it out.
3. **Rejected edits / wasted cycles**: `.buildflow.yml` edit before View (tool refused); worktree `add` without `-b` (branch didn't exist); a heredoc-python edit attempt on templ-components CHANGELOG replaced by proper edits.
4. **e2e triple-miss**: (1) plain shell → chromium missing `libglib`; (2) devShell → `GOWORK=off` silently testing PUBLISHED modules (the big one — root-caused via an A/B worktree test at origin/master + direct HTTP probes); (3) relative `GOWORK` rejected ("not an absolute path"). Three attempts where a "what does the webServer env actually resolve?" check up front would have been one.
5. **Test-first type error**: asserted `e.Name` on `sse.Event` (field is `Event`; predicate fixed to Data-envelope matching). Should have `go doc`'d the type first.
6. **Awkward TODO wording** on the theme item ("rename the spike's description?") — self-corrected to a proper resolved-style entry.
7. **THE ONE STILL OPEN (see b/1):** my tab-indented `//cqrs-lint:ignore` comments inside require blocks inflate `check-dep-budgets.sh` counts — introduced while fixing V006, noticed only when the gate flagged root/datastar over budget. Not yet fixed at report time.

---

## e) WHAT WE SHOULD IMPROVE (structural, from this session's pain)

1. **`check-dep-budgets.sh` must ignore comment lines** (awk: skip `^\t*//`) — any future in-block suppression or comment re-breaks the gate.
2. **Pin the gate's cqrs-lint version** (nix runtimeInputs), not ambient PATH — the 4.8.1→4.11.2 drift re-anchored suppressions and added findings mid-session; gates must be reproducible.
3. **`check-cqrs-lint` flakiness**: the writeShellApplication has no runtimeInputs for cqrs-lint → resolves via ambient PATH inside a hardened script; combine with (2).
4. **e2e webServer env contract**: now encoded in the config (absolute GOWORK) — consider the same pinning for `E2E_BROWSER_PATH` (nix chromium) so the suite stops depending on system glib drift.
5. **Golden-update runbook**: the font pin requirement is documented in the flake but not in AGENTS.md's Bench/Gates rows — one line would have saved the d/1 detour.
6. **buildflow `go-version-auto-configure`** fights `go mod tidy` (bare-minor vs `.0` canonicalization) — reported in the templ-components skip comment; worth an upstream BuildFlow issue.
7. Stop-loss discipline: the dep-budget + cqrs-lint gate tails consumed the last stretch; a pre-train "gate dry-run" earlier would have surfaced them before all four workstreams were merged.

---

## f) NEXT (ordered; ~28 items)

1. Fix `check-dep-budgets.sh` to skip comment lines (or un-indent the three V006 suppressions) → rerun `.#check-modules`.
2. Decide cqrs-lint posture (see question 1) → make `.#check-cqrs-lint` green or document the accepted state.
3. `.#test-fuzz`, `.#test-flake`, `nix flake check --no-build`.
4. Full e2e suite on the final tree.
5. Attempt `.#bench-spike` (machine-gated; if load > 8 → document the refusal, proceed).
6. CHANGELOG: V007 ADR/deprecations entry + canonical header fix + V006 notes.
7. AGENTS.md: cqrs-lint row refresh (4.11.2, new anchor, remaining set) + e2e GOWORK gotcha + theme row already done; add dep-budget/comment gotcha.
8. TODO_LIST truth pass: mark V007 spike done (ADR-0051), header note.
9. Cut **v4.12.0**: family require bumps → per-module verify-tag (order: identity-model → usermgmt family → root → UIs → setup → systemadapter/health/auditlog) → push master + tags.
10. Post-train: `check-release-train --refresh-cache`, CI green check, `docs/status` train report.
11. If bench tripped: re-pin baseline per policy (same change).
12. Cleanup: remove `../cqrs-htmx-v007` worktree + branch; remove `/tmp/cqrs-head` worktree + symlink; kill round-5's `admin-demo-p3` (:18932) if still running.
13. Templ-components follow-up: file the BuildFlow version-normalizer conflict upstream (larsartmann/buildflow).
14. Triaging backlog (post-train): C009 panic in dashboardui/assets.go, e2e/server A023 custom snapshot store, P008 polling findings.
15. cqrs-lint strict CI gate still blocked on Go-installable distribution (P3, unauthorized).
16. datastar-demo KEEP-but-rebrand execution (unauthorized this round).
17. ListNote/Grid/CopyButton upstream asks (a–c) — still open by design.
18. Command-audit tail item (5): upstream `requestContextEnricher` to go-cqrs-lite event/ (next train).
19. systemadapter declarative ExternalAccountLink fold cleanup (tracked bug).
20. `Volume > 0` regression test + provenance tuning (residual micro-ideas).
21. e2e admin-panel SSE spec, catalog-demo visual smoke, samber-do SSE e2e (examples/tests micro-ideas).
22. `docs/planning/2026-08-30_cqrs-lint-go-distribution-draft.md` — awaiting owner.
23. V007 cluster 1 (68 SQLViewStore findings) — gated per ADR-0051 criterion; re-check metaengine layout-planning API each go-cqrs-lite train.
24. go-cqrs-lite: execute the stack-removal v5 plan when the v5 train starts (decision recorded).
25. templ-components: visual suite `TestSiteSalesCopyButton` clipboard flake (failed once in visual2, passed since) — worth a retry-wrapper.
26. adminui: consider hiding the theme toggle on very small screens (product micro-call).
27. AGENTS.md line-count warn (387 > 377 in templ-components buildflow) — trim.
28. Sweep `docs/status` archive links after every archive move (the broken-link class) — maybe make docs-health annotate move-aware.

---

## g) QUESTIONS FOR THE USER (cannot figure these out myself)

1. **cqrs-lint posture:** the ambient tool drifted 4.8.1 → 4.11.2 and now fires ~25 pre-existing warnings (documented "pending triage" since 2026-09-17: C009 `panic()` in dashboardui/assets.go, e2e/server A023 custom snapshot store, P008 polling, …). Triage them all NOW (blocks the local gate, ~1 session), or pin the gate tool to 4.8.1 in nix and keep the findings as the documented backlog?
2. **Dep-budget comment artifact:** fix `check-dep-budgets.sh` (skip comment lines — touches shared gate tooling, right fix), or un-indent my three `//cqrs-lint:ignore(V006)` go.mod suppressions (no leading tab — workaround, keeps the script untouched)?
3. **Theme toggle on mobile:** it is currently always visible in the header (small ghost icon button). Keep as-is, or hide below `sm` like the email text?

---

## Session state at report time

- Tree: all work daemon-absorbed on master (spike merge `99be42f7` in); working tree has the uncommitted gate fixes (header constant, V006 suppressions, link fix, budget investigation pending).
- Gates red: `check-modules` (dep-budget comment artifact) + `check-cqrs-lint` (pre-existing triage set + flaky env).
- Worktrees: `../cqrs-htmx-v007` (merged, removable), `/tmp/cqrs-head` (debug, removable). Stale: round-5 demo server on :18932.
- templ-components: master + v1.19.0 tags PUSHED and verified; go-cqrs-lite ROADMAP edit daemon-absorbed.
