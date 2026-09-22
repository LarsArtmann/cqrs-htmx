# Round 11 Status Report — gitignore-gap healing, live gate catch, sixth wave

**Generated:** 2026-09-23 01:36 CEST · **Session window:** ~00:05–01:36 CEST (round 11, immediately after round 10's handoff)
**Scope:** THIS session only. Point-in-time snapshot, append-only per docs/status convention. (User requested `.md`; the status-report skill's HTML default was overridden by explicit instruction.)

> **ANNOTATED placeholder:** rows/items below carry no resolution markers at write time; absence of a marker IS the open signal.

**Headlines:** The inherited 84085c35 CI red was a `.gitignore` negation gap (not a code bug) — fixed and proven from a `git write-tree` archive. The new CSS-bundle sanity gate — shipped THIS session — caught a live corruption in CI within the hour (the buildflow hook's pretty rewrite riding a daemon commit through my own push). Two more upstream waves swept (go-health-dashboard v0.10.0 by the concurrent session, v0.10.1 by me). Full Playwright suite 60/60. Origin green at `d6132cf3` (run 35797581704), tree clean, 0 unpushed.

---

## a) FULLY DONE

1. **84085c35 CI red root-caused and closed.** The dashboardui templ migration committed 10 `.templ` sources but the root `.gitignore`'s `*_templ.go` rule (negations existed only for `!adminui/` and `!loginpage/`) silently swallowed every generated file. Local builds passed (files on disk); CI checkouts failed with `undefined: overviewPage/badge/errorShell…`. Fixed in `a1cfa41f`: `!dashboardui/*_templ.go` + the 10 files committed, verified byte-identical to fresh `templ generate`, build+tests green from a `git write-tree`/`git archive` of the index — exactly what CI checks out.
2. **Committed-drift regeneration.** A later `.templ` edit (`aggregates_templ.go` mirror missing) regenerated and committed (`f969ced7`); now that the mirror is tracked, `check-codegen` catches this class (while ignored, the same gate false-greened — git diff never sees untracked files).
3. **Fifth wave verified (concurrent session's sweep).** The pre-push release-train gate fired on go-health-dashboard v0.10.0 — diagnosis showed the concurrent session had already swept it (`4573f169`); I verified all three consumers (build+test green), re-gated strict, pushed. Division of labor worked exactly as the protocol intends.
4. **Round-10 HARVEST completed.** TODO_LIST rewritten (4 done items removed with CHANGELOG witnesses, 6 new items routed, consumer-eye item updated to v1.19.2, gate-integrity item extended with §f18–20); ROADMAP OQ20 recorded (strict-lag policy); CHANGELOG [Unreleased] gained Added/Fixed/Verified entries for the whole round-10 + round-11 arc. Daemon split the batch across two commits (content verified complete). Both status gates green.
5. **agents-notes narrative shipped** (`0c0b6ae4`): "The five-wave evening" — external-wave diagnosis method, the two bump-dep prefix traps, the gitignore-negation gap with the write-tree proof recipe, the committed-drift gate activation, the quantified daemon-race pattern, the concurrent-session division-of-labor rule.
6. **e2e DLQ count-notice assertions** (`4dc205c7`): the e2e server seeds two dead-letter entries directly into the in-memory DeadLetterStore (deterministic — a failing-handler seed would race retry exhaustion); two new specs assert the exact pluralized text (`Showing 2 items.`), the named `role=status` live region, and the index projection link. **Full Playwright suite: 60 passed** (57/57 precedent + my 2 + 1). Screenshots refreshed in the same run — the first captures since the full-templ migration; dead-letters pages now show the seeded rows.
7. **Daemon-race runbook shipped** (`20e2b518`): `docs/runbooks/daemon-commit-races.md` — HEAD-lock recovery (check `git log` before retrying; the daemon likely committed your staged set), shredded-commit verification rules (amend only immediately-local tips with no foreign mixing), mid-batch add splits, prevention table cross-linking the preflight/wait-tree-quiet helpers. Decision recorded: separate runbook (recovery narrative ≠ tool usage).
8. **CSS-bundle sanity gate shipped — atomic 6-piece** (`9fd6400d`+`9644e7ca`): checker (1-line minified + tailwind banner + 30KB floor + 4 canary utilities per bundle), 11-case fixture self-test, flake apps (`.#check-css-bundles` / `.#test-css-bundles`), check-modules stages (both lists), CI steps, AGENTS command docs.
9. **Consumer-eye verification of templ-components v1.192 complete** (`456d85a3`): a throwaway module outside the workspace `go get`s the published root tag from the proxy and compiles + renders Badge/StatCard/EmptyState with content assertions. Bonus discovery: the family is a single root module carrying all packages — `templ-components/<pkg>@v1.19.x` submodule requires are a stale split-era trap (recorded in CHANGELOG).
10. **Sixth wave swept + coverage re-verified.** go-health-dashboard v0.10.1 (published ~1h after v0.10.0): exact-anchor recipe sweep, 3 consumers tidy+build+vet+test green, strict gate 816/0/0. Coverage gate re-run at load 10.95: **15/15 modules above thresholds** (setup 86.9%, systemadapter 91.9%, health 100%, auditlog 100%, dashboardui/core 88.8%).
11. **loginpage adoption pass — survey complete, decision routed** (`f5112e89`): full adoption is NOT a mechanical swap (new templ-components require breaks the zero-dep README contract; needs compiled+embedded bundle + builder + gate wiring; replaces the custom-brand design). Routed to ROADMAP OQ21 with revisit criteria — the repo's established owner-call pattern.
12. **Live corruption caught and healed.** CI red on run 35796632818: the new CSS gate fired — both bundles in the pushed tree were the buildflow hook's pretty rewrites (4244/4092 lines; daemon-batched `1b3c0cf9` with the `styles.css` orphan + flake.lock — the full hook signature). Canonical minified bundles rebuilt (`17655d54`, class sets identical), producer identified and recorded in the hunt item.

---

## b) PARTIALLY DONE

1. **CSS-bundle producer hunt → now a PREVENTION task.** Detection solved (the gate fired live, correctly). Prevention open: the buildflow hook's tailwind step still rewrites bundles without `--minify` on any hooked commit; fix at the source (`.buildflow.yml` mirror/exclude or upstream ask) not done. Interim rule recorded: re-run flake builders after any hooked commit touching UI modules.
2. **Pre-push gate coverage.** The pre-push hook runs release-train + version-drift only — the new CSS gate is in check-modules + CI but NOT pre-push, which is exactly how the pretty bundles rode my push red (see d-1). Wiring it in is a one-line hook change… not done yet.
3. **dashboardui session report harvest coverage.** I imported its top follow-throughs (Grid, page goldens, error-shell survey) as one TODO bundle, but its Next-50 remainder (Sparkline, per-page `<title>` e2e, axe sweep, BuildFlow tsc investigation, i18n stance, export buttons, etc.) lives only in that timestamped report. If its session has ended, those are entombed; if it continues, they're its backlog. Needs one attribution check + harvest-or-confirm pass.

---

## c) NOT STARTED

1. **AGENTS.md distillations of tonight's two structural lessons:** (a) per-module allowlists in global ignore rules are a trap for the NEXT module joining the gated class (grep the negation list on adoption); (b) templ-components is a single root module — submodule-path requires are a stale trap. Both live only in agents-notes/CHANGELOG; AGENTS (the living core) doesn't carry them.
2. **docs/status/README.md index + count refresh** for the three new reports (round-10 HTML, dashboardui migration, this one).
3. **dependabot grouping config** (round-10 §f9 — carried, untouched this session).
4. **Codegen-guard fixer-class fixture** (does `check-codegen` fire on a formatter-shaped rewrite? — carried).
5. **Bench-spike quiet-window attempt** — correctly not attempted (load 10.95 > limit 8 at the only quiet-ish moment).

---

## d) TOTALLY FUCKED UP (all self-inflicted, all recovered — the honest list)

1. **I pushed a red tip (2a2e2368) and MY OWN gate caught it.** The daemon batched the concurrent session's hook-corrupted pretty bundles (`1b3c0cf9`) between my gate-green commit and my push; the pre-push hook doesn't run the CSS gate, so it rode through and CI went red on run 35796632818. Root cause is dually mine: (a) I designed the gate's wiring and chose 4 of 5 enforcement surfaces, missing pre-push; (b) I pushed docs-only changes without re-running cheap gates on a tree a concurrent session was actively mutating. Silver lining: first live fire of the gate, producer identified, healed in one cycle — but the red was avoidable.
2. **`git stash push --staged` chained into a verification command** (masked by `|| true`) stashed my own 11-file staged fix mid-flight. Recovered immediately (`stash pop` + re-stage), but it was a near-miss race window on the session's critical fix. Never chain conditional git mutations into verification commands.
3. **e2e environment fumble (2 wasted rounds).** Ran `bun x playwright` bare first — browser launch failed (`libglib-2.0.so.0`, the NixOS FHS class), then downloaded a browser that couldn't run, before reading the flake app that already solves this (`E2E_BROWSER_PATH` nix chromium + `PLAYWRIGHT_BROWSERS_PATH`). The runner doc existed; I should have read it before hand-rolling the invocation.
4. **Consumer-eye submodule-path detour.** `go get templ-components/display@v1.19.2` etc. — stale split-era paths; the repo's own imports (display as a package of the root require) were visible before I wrote the throwaway. The detour produced the trap-discovery, but reading go.mod first was cheaper.
5. **Edit-tool anchor fumbles (3 rounds).** CHANGELOG multiedit: 2 of 3 edits failed on ambiguous `### Added`/`### Fixed` anchors (repeated across version sections — I knew the structure). TODO_LIST watch-item removal: failed twice on exact-match, fell back to sed. Fixture newline-escaping: tr→sed→python3 iterations. All recovered; each cost a round-trip that exact-anchoring-from-the-start would have saved.
6. **Daemon lost ~5 races despite phase-boundary discipline** (11-file staged fix grabbed mid-buildflow-hook; docs batch split in two; harvest split across commits). Content survived every time (verified per `git show --stat`) — the pattern is now quantified in the runbook — but my amend attempts remain theater while the daemon re-races in seconds.

---

## e) WHAT WE SHOULD IMPROVE

1. **Wire `check-css-bundles` into the pre-push hook** — the exact gap that reddened 2a2e2368. One line; do it before the next push cycle.
2. **Fix the buildflow CSS step at the source** (emit `--minify` or exclude `assets/*-tw.css`) — detection is not prevention; every hooked commit can still corrupt until this lands.
3. **Gate-design checklist: enumerate ALL enforcement surfaces explicitly** (pre-commit, pre-push, check-modules, CI, README) and record a deliberate include/exclude decision per surface — my implicit "CI+check-modules is enough" decision failed within the hour.
4. **Pre-push minimum bar on shared trees:** when a concurrent session is active, run the cheap repo-state gates (CSS bundles, docs-links, status gates — seconds) before every push, not just the hook's two.
5. **Read the existing runner (flake app / justfile / README) before hand-rolling any tool invocation** — the e2e environment was already solved; I rediscovered it the expensive way.
6. **Anchor edits uniquely from the start** in files with repeated section headers (CHANGELOG version sections) — include a section-unique line in every old_string.
7. **Distill the two structural lessons into AGENTS.md** (gitignore allowlist trap; templ-components root-module layout) — agents-notes is the archive, AGENTS is what the next session actually loads.
8. **HARVEST trigger discipline:** every session that ends with a status report should also confirm the PREVIOUS session's report was harvested (the dashboardui report's Next-50 is the live example of near-entombment).

---

## f) NEXT (up to 50, ordered by priority)

| # | Pri | Task | Status |
|---|-----|------|--------|
| 1 | P1 | Wire `check-css-bundles` into the pre-push hook (the 2a2e2368 gap) | Open |
| 2 | P1 | BuildFlow CSS step prevention: `.buildflow.yml` mirror/exclude or upstream ask | Open |
| 3 | P1 | Watch for waves 7+: six upstream publishes in ~3.5h tonight; each reds master until swept (OQ20 decides the posture) | Open |
| 4 | P2 | OQ20 decision: strict-lag 0 vs bounded tolerance + scheduled alignment job | Open (owner) |
| 5 | P2 | OQ21 decision: loginpage templ-components adoption vs zero-dep posture | Open (owner) |
| 6 | P2 | Attribution check: is the dashboardui session done? If yes, HARVEST its Next-50 remainder | Open |
| 7 | P2 | docs/status/README.md index entries + counts for the 3 new reports | Open |
| 8 | P2 | AGENTS.md distillations: gitignore per-module-allowlist trap; templ-components root-module layout trap | Open |
| 9 | P2 | dependabot grouping config (stop solo actions-bump reds mid-train) | Open |
| 10 | P2 | Codegen guard: fixture proving it fires on fixer-shaped (formatting-only) rewrites | Open |
| 11 | P2 | Gate-integrity spot checks post-bumps (verify-tag exemptions, GOWORK pin, both-repos push, vcs-cache placement, go.work.sum tidy, e2e lane coverage) | Open |
| 12 | P2 | Post-adoption exclusion-claims sweep vs v1.19.2 (+ NoThemeScript note check) | Open |
| 13 | P2 | templ-components upstream asks: CopyButton `var(--text)` override; ListNote X–Y range variant | Open |
| 14 | P2 | dashboardui: adopt `htmx.PolledRegion` for the projection-health region | Open |
| 15 | P2 | dashboardui follow-through bundle: `display.Grid` adoption; page-level goldens; error-shell unification survey | Open |
| 16 | P2 | CopyButton contrast visual check (adminui tables) | Open |
| 17 | P2 | Screen-reader check of the DLQ count note (role=status announce semantics) | Open |
| 18 | P2 | Count-notice consistency sweep (audit/commands/queries pages) — design call | Open |
| 19 | P2 | Dark-mode gray-800/900 pin re-check post-bump (adminui + dashboardui) | Open |
| 20 | P2 | Human glance at the refreshed dashboard screenshots (templ-migration DOM deltas; nobody has looked) | Open |
| 21 | P3 | FEATURES.md rows for the round-8 + round-11 gates (CSS bundle gate included) | Open |
| 22 | P3 | bench-spike quiet-window attempt + OQ16 automate-or-retire decision | Open |
| 23 | P3 | Orphan `dashboardui/styles.css` churn-rule re-check post-migration | Open |
| 24 | P3 | verify-tag fixture drift exemption vs v1.19.2 fixtures | Open |
| 25 | P3 | docs/status README link check post-rename | Open |
| 26 | P3 | HARVEST this report's §f into TODO_LIST/ROADMAP (next session) | Open |
| 27 | P3 | OQ15: release-notes posture (GitHub Releases vs tags+CHANGELOG) | Open (owner) |
| 28 | P3 | OQ18: push/sync policy under concurrent sessions — codify in gotcha 4 | Open (owner) |
| 29 | P3 | OQ19: heavy-gate policy (full sweep vs targeted+CI per family bump) | Open (owner) |
| 30 | P3 | OQ5: membership payload ActorID format decision | Open (owner) |
| 31 | P3 | check-cqrs-lint CI gating (blocked: Nix-only distribution) | Open (blocked) |
| 32 | P3 | /mnt/buildcache fill-drain watch (recurring; `df -h` first) | Open |
| 33 | P3 | V007 cluster-1 migration (68 SQLViewStore findings; gated on metaengine layout-planning) | Open (blocked) |
| 34 | P3 | e2e: full axe-core sweep of all templ pages (beyond landmarks/labels) | Open |
| 35 | P3 | e2e: per-page `<title>` uniqueness check (HTMX boost relies on it) | Open |
| 36 | P3 | Investigate BuildFlow `type-check`/`tsc` ambient failure (burns 60s per pre-commit) | Open |
| 37 | P3 | nix eval-cache SQLite busy errors (pre-commit log) — tuning or relocation | Open |
| 38 | P3 | `templ generate --watch` guidance in devShell docs (LSP noise) | Open |
| 39 | P3 | Hybrid-adoption guide HISTORICAL banner (dashboardui migrated; guide is legacy) | Open |
| 40 | P3 | Sweep docs for stale "dashboardui strings.Builder" claims | Open |
| 41 | P3 | dashboardui minor-bump decision (templ migration: now vs next family train) | Open (owner) |
| 42 | P3 | go-health-dashboard v0.10.x adoption notes in the next family-train notes | Open |
| 43 | P3 | Check go-health-dashboard v0.10.x for health-bridge APIs dashboardui should surface | Open |
| 44 | P3 | Benchmark templ render vs response streaming for the largest page (DLQ detail) | Open |
| 45 | P3 | Move `streamListPageConfig`-style page configs into templ components with props | Open |
| 46 | P3 | `display.Sparkline` evaluation for projection trends (trivially adoptable now) | Open |
| 47 | P3 | SSE-partial ETag for the polled projection-health partial (caching) | Open |
| 48 | P3 | Golden-update workflow doc for dashboardui CONTRIBUTING | Open |
| 49 | P3 | DLQ bulk-replay progress feedback (synchronous replay looks hung) | Open |
| 50 | P3 | Event-detail next/prev keyboard navigation (match time-travel arrows) | Open |

---

## g) QUESTIONS I CANNOT ANSWER MYSELF

1. **(OQ20, urgent tonight)** Should the release-train gate stay `--strict-lag 0` — every upstream publish reds master until a consumer sweep lands, six waves in 3.5 hours tonight — or move to a bounded tolerance (e.g. `--strict-lag 3`) plus a scheduled alignment job? This is a policy call on what the gate is FOR: zero-tolerance staleness signal vs stable green with bounded lag. I can implement either; I cannot choose the tradeoff for you.
2. **(OQ21)** loginpage: adopt templ-components (ends its zero-dependency "self-contained" differentiator, gains fleet design consistency, costs a compiled embedded bundle + builder + gate wiring + custom-brand design replacement) — or keep hand-rolled forever? Revisit triggers are recorded; the taste call is yours.
3. **(pre-push posture, new from d-1)** Should the pre-push hook run the FULL `check-modules --report` (~2–5 min per push) instead of just release-train + version-drift? Tonight's red would have been caught at the boundary — but every push (including the daemon's era) pays the latency. Alternatively: cheap repo-state gates only (CSS bundles + docs-links + status gates, ~seconds). Which bar do you want at the push boundary?

---

*Evidence: CI runs 35792905362 / 35795038829 / 35795170263 / 35797136646 / 35797581704 (all green), 35796632818 (red — CSS gate live catch). Commits this session: a1cfa41f f969ced7 a2001079 (daemon-split w/ c3d9f962) 0c0b6ae4 4dc205c7 20e2b518 9644e7ca (daemon-split w/ 9fd6400d) 456d85a3 f5112e89 3b52222a 17655d54 2a2e2368 d6132cf3. Playwright 60/60; coverage 15/15; train gate 816/0/0.*
