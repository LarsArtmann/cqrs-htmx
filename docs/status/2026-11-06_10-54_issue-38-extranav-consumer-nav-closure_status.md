# Status — issue #38 closed in code: `ExtraNav` made usable by real consumers, and the three holes the first pass left

**Date:** 2026-11-06, 09:15–11:05 CET
**Repo:** `/home/lars/projects/cqrs-htmx`, branch `master`. The auto-commit daemon committed this session's work incrementally (heuristic commits, 09:3x–10:5x), so the narrative below cites files and measurements rather than one commit hash.
**Scope of this session:** read and execute [cqrs-htmx#38](https://github.com/LarsArtmann/cqrs-htmx/issues/38) ("Config knob to append consumer nav entries (ExtraNav) — nav is capability-internal today"). The core knob had already landed in `b3ce8c52` (2026-11-05 22:38) but was **unreleased and unusable from two real consumer seats**; this session closed those holes, proved them, and closed the issue.
**One-sentence verdict:** the feature now works for a consumer who mounts via `setup` and for one who renders `PageMeta.Nav` in their own shell, with the injection surface pinned by test and the duplicate-shape debt retired; what remains is owner-gated (a `dashboardui`/`setup` tag) plus three a11y/validation decisions I deliberately did not take alone.

---

## Context correction (read this first)

The issue, read at face value, looked unimplemented. It was **implemented but incomplete**, and the difference mattered:

| Claim in the issue | Reality at session start |
| --- | --- |
| `buildNav(caps)` → unexported `pageData.Nav`, no consumer seam | `Config.ExtraNav []NavEntry{Label, Href, Icon}`, `extraNavItems()`, `validateExtraNav()`, `navItem.Verbatim`, `navHref()` — all present, tested, and committed (`b3ce8c52`) |
| "zero behavior change when unset" | True, goldens pin it |
| — | **`PageMeta.Nav`'s `NavLink` dropped `Verbatim`, while its doc claimed "Href is relative to BasePath"** — a custom layout following the doc prefixes BasePath onto a sibling-surface href and 404s |
| — | **`setup.Config` had `AdminExtraNav` but no `DashboardExtraNav`** — a bundle consumer (the mounting story in the issue) could not reach the knob at all |
| — | **Verbatim entries could never be `Active`** — `page()` compared raw hrefs, so an in-panel consumer entry stayed inert |
| — | `dashboardui/v4.16.0` was tagged **2026-11-05 04:11**, the knob landed **22:38** — unpublished |

So this session was not "write the knob", it was "make the shipped knob honest, reachable, and verifiable".

---

## a) FULLY DONE

### a1. `PageMeta.Nav` now tells the truth (`NavLink.Verbatim`)

`dashboardui/layoutfunc.go`:

- `NavLink.Verbatim bool` — true for consumer `ExtraNav` entries, false for every capability-derived item.
- `PageMeta.Nav` doc rewritten: "Check `NavLink.Verbatim` before prefixing `Href` with `BasePath`".
- `LayoutFunc`'s contract bullet rewritten — it previously said "Nav Hrefs are relative to meta.BasePath", which was false the moment `ExtraNav` existed.
- `meta()` collapsed to `slices.Clone(p.Nav)` (the two types are now one; see a4), so a consumer shell cannot mutate dashboard state.

This is the fix that turned a silent 404 into a documented contract. It is the class the repo calls a split brain: the same concept (`verbatim or not`) existed on the internal type and was dropped at the exported boundary.

### a2. Verbatim entries get `Active` when they point at an in-panel page

`dashboardui/layout.go` gained `navItemActive(basePath, item, active)`: built-ins match `active` directly; verbatim entries match `active` **or** `basePath+active`, so a consumer entry built from `Config.BasePath` highlights exactly like the built-in one, while a sibling surface or absolute URL never does. `page()` uses it (`dashboardui/dashboard.go:112`).

Pinned by `TestNavItemActive` (6 rows: built-in match/miss, verbatim with/without prefix, sibling surface, absolute URL) and by `TestExtraNav_InternalHrefHighlightsWhenActive` (real HTTP render through a custom layout: the `/obs/events` extra entry is `Active`, the `/event-catalog` one is not).

### a3. `setup.Config.DashboardExtraNav` — the bundle seat can finally use the knob

`setup/config.go:447` declares `DashboardExtraNav []dashboardui.NavEntry`; `setup/setup.go:416` wires it into `buildDashboardConfig`. Documentation states the verbatim contract and that entries also reach `PageMeta.Nav` when `DashboardLayout` is set.

Proof is end-to-end, not config-level: `setup/dashboard_extra_nav_test.go` builds a real bundle, registers a user, mounts the dashboard at `/dash`, GETs `/dash/` with the session cookie and asserts the sidebar contains `href="/event-catalog"` and **not** `href="/dash/event-catalog"`. That is literally the issue's motivating case ("a surface we mount next door").

Plus `setup/setup_validation_test.go`: `TestNew_DashboardExtraNav_ReachesDashboard` (config lands in `bundle.Dashboard.Config().ExtraNav`) and `TestNew_DashboardExtraNav_RejectsEmptyHref` (the passthrough reaches `dashboardui`'s validation instead of being silently dropped).

### a4. The duplicate nav shape is gone (and one lint class with it)

Adding `Verbatim` to `NavLink` made it field-for-field identical to the internal `navItem`, which `branching-flow` immediately reported as a NEW `DUPLICATE_STRUCT` (both directions). The honest answer was not a suppression: the two types **were** the same concept.

- `navItem` deleted; `pageData.Nav`, `Dashboard.nav`, `buildNav`, `extraNavItems`, `navHref`, `navItemActive`, `page()` all speak `NavLink`.
- `overview_page_test.go` updated (`[]navItem` → `[]NavLink`).
- `exhaustruct_v5` then demanded exhaustive literals — satisfied explicitly (`Active: false, Verbatim: true|false`), which doubles as documentation that `Active` is stamped per request, not at build time.

Net: one exported type instead of a public twin plus a private duplicate, and the analyzer's duplicate class retired rather than frozen.

### a5. The injection surface is pinned

`TestExtraNav_EscapesLabelAndHref` feeds a hostile label (`<img src=x onerror=...>`) and href (`/x" onmouseover="alert(1)`) through the real render path. I probed the actual output first (a temporary test) and then asserted on it verbatim:

```
<a href="/x&#34; onmouseover=&#34;alert(1)" class="nav-link">
```

Quotes escape, no attribute break-out, label renders as `&lt;img`. The test asserts all three facts (no live `onmouseover="alert(1)"`, no raw `<img`, the exact escaped href present) so an escaping regression fails loudly instead of passing silently.

### a6. Documentation, in the four places a consumer actually looks

| Surface | Change |
| --- | --- |
| `dashboardui/README.md` § Extra Nav Entries | +12 lines: `NavLink.Verbatim`, the Active rule for in-panel entries, the `setup` passthrough, sibling-seam pointer to `AdminExtraNav` |
| `dashboardui/README.md` § Integration Modes | the `meta.Nav` contract bullet now names `Verbatim` and links `#extra-nav-entries` |
| `dashboardui/CHANGELOG.md` `[Unreleased]` | full `Config.ExtraNav` + `NavEntry` entry (there was none — the feature was undocumented in the changelog) |
| `setup/README.md` config table | `DashboardExtraNav` row next to `DashboardLayout`, verbatim contract + `PageMeta.Nav` note |
| `AGENTS.md` dashboardui bullet | "Consumer nav (cqrs-htmx#38)" clause appended to the integration-modes paragraph |
| `FEATURES.md` Integration Modes row | the same clause, so the inventory matches the tree |

### a7. Release + ratchet bookkeeping (the part that keeps the next session honest)

- `TODO_LIST.md` row 23 (owner-gated next train): the unpublished-surface list now names dashboardui's `ExtraNav`/`NavEntry`, `NavLink.Verbatim`, and `setup.Config.DashboardExtraNav`, with the explicit consequence (dashboardui/setup hermetic legs stay red until the tags land).
- `TODO_LIST.md` row 87 (art-dupl / branching-flow refresh, owner-held): recorded today's ratchet reading `+7 added, −3 removed, ~168 modified` and attributed it — 2 findings are my anchor churn under *frozen* verdicts (`COMPOSITION_mixin` + `DUPLICATE_TYPE` moved `config.go:321` → `:310`), 5 are the concurrent `huma/` session's uncommitted package. The baseline is **not** re-pinned, because pinning would freeze another session's unreviewed findings; their settle pass re-pins.

### a8. Issue #38 closed with evidence, in the right register

`gh issue close 38 --reason completed` with a three-sentence comment naming the knob, the two test files, the verbatim contract, and the unpublished-tag reality. Drafted against the `github-voice` skill and passed its checker: `0 FAIL, 0 WARN`.

### a9. Verification battery actually run

| Leg | Result |
| --- | --- |
| `go build ./dashboardui/... ./setup/...` (workspace) | rc=0 |
| `go vet` same targets | rc=0 |
| `go test ./dashboardui/... ./setup/... -count=1` | `ok` dashboardui (0.51s), `ok` core (0.01s), `ok` setup (9.7s…15.3s across runs) |
| `-race` same targets | `ok` dashboardui (1.76s), `ok` core (1.06s), `ok` setup (9.67s) |
| `gofmt -l dashboardui setup` | clean |
| coverage (per-module script, profile regenerated) | dashboardui **73.6%** vs floor 60; setup **92.8%** vs floor 80 |
| `golangci-lint` 2.15 **workspace-mode** | zero findings in any file I touched (after fixing the 2 `exhaustruct_v5` misses I introduced) |
| docs/skill/tree gates | `readme-splitbrain`, `skill-claims`, `skill-snippets`, `todo-hygiene`, `docs-corpus-budget`, `docs-tail-budget`, `root-files`, `module-layers`, `conflict-markers` — all rc=0 |
| `nix run .#check-modules` | **87 PASS / 7 FAIL of 94** — attribution table below |
| `nix run .#check-branching-flow` | FAILED, +7 / −3 / ~168 — decomposed and booked (a7) |

Composite attribution (none of the 7 is new work of mine, one carries a line of mine):

| Stage | Cause | Mine? |
| --- | --- | --- |
| `module-isolation`, `module-isolation-examples` | hermetic (`GOWORK=off`) builds against **published** tags: setup/usermgmt/systemadapter need `DropStats`, adminui/dashboardui `NavEntry`, identity-model session events; integration_test needs the catalog release | no (pre-existing, TODO row 23) |
| `workspace-build` | go-cqrs-lite `catalog/v4@v4.12.0` docserver ships templ-poisoned generated code (`templ.JoinURLErrs`, `ResolveAttributeValue`) | no (upstream) |
| `streamid-identity` | `dashboardui/core/command_fates.go:198` uses `StreamID().String()` in an identity position | no (concurrent command-fates/huma session) |
| `branching-flow` | 2 anchor-churn findings (mine, frozen verdicts) + 5 in the sibling's `huma/` package | partly (churn only) |
| `route-postures`, `route-postures-self-test` | setup's hermetic manifest run; `setup/config.go:447 dashboardui.NavEntry` is one line of that error list | no (the field is the tag gap; adminui/DropStats already in the same list) |

---

## b) PARTIALLY DONE

1. **Lint coverage of my own code is indirect.** The devshell exports `GOWORK=off` (`flake.nix:59`), so the canonical `nix run .#lint` lints each module **hermetically against published tags** — it cannot compile `dashboardui`'s or `setup`'s current tree until the train lands. I worked around it with `nix develop . -c bash -c 'export GOWORK=$PWD/go.work; cd <module> && golangci-lint run'`, which is the mode that actually sees my code but is *not* a repo gate. Result: my code is clean, but the CI-shaped lint verdict for these two modules stays red on the tag gap alone.
2. **Coverage verified per module, not through `.#coverage-gate`.** I regenerated `dashboardui/coverage.out` and ran the shared per-module script (73.6% / 92.8%); I did **not** run the 15-module app, because it is load-sensitive (gotcha 41) and a concurrent session was mid-flight. Thresholds were already single-sourced, so the numbers are trustworthy; the composite run just wasn't made.
3. **`branching-flow` ratchet is red and deliberately not re-pinned.** Documented in TODO row 87 with per-finding ownership. It stays red until the `huma` session's settle pass — no action from this seat that wouldn't sweep another session's work.
4. **Issue closure is "implemented on master", not "released".** The comment says so explicitly ("Unpublished until the next dashboardui tag"). Publishing needs an owner-gated push (`wave-tag`), which I must not do.
5. **The `NavEntry`↔`NavLink` mixin finding remains** (3 shared fields) — a frozen verdict from the 2026-11-06 triage row, reported from `config.go:310`. I left it adjudicated-as-is rather than restructuring the public `NavEntry` shape to silence an analyzer heuristic.
6. **`setup` CHANGELOG does not exist.** Setup's user-facing surface lives in `setup/README.md` only, so my `DashboardExtraNav` docs land there. If setup ever gains a changelog, this entry has to be back-filled.

---

## c) NOT STARTED (deliberate)

1. **Tags.** `dashboardui` (ExtraNav) and `setup` (passthrough) are unpublished: the family train is owner-gated and would publish three other sessions' in-flight API at the same time (TODO row 23 lists root `DropStats` + the whole dispatch-boundary API, identity-model session events, adminui's wave).
2. **An `examples/` demo of the knob.** The issue's next-list item #39 suggested one; the examples hermetic leg resolves **published** tags and is already red (`setup-demo` needs `AdminDashboardURL`, the root `DropStats` family). Adding a demo now would add a *new* uncompilable example to an already-red leg — noise, not signal.
3. **`aria-current="page"` on active nav links.** Neither the built-in nor the extra links carry it (pre-existing a11y gap). Fixing it changes every sidebar golden and is a UI-wide decision, not part of #38.
4. **URL-scheme validation for `Href` (`javascript:`, `data:`).** The library validates `AccentColor` against CSS injection because it *interpolates* it; an href is escaped by templ, so this is a policy question about defence-in-depth for a developer-supplied field — asked as question #2 instead of decided unilaterally.
5. **Skill documentation.** `.agents/skills/cqrs-htmx/SKILL.md` has no dashboard Config field inventory (so no stale claim; `check-skill-claims` passes), but it also does not teach `ExtraNav`. Not wrong, just incomplete.
6. **Docs-health archive round.** The live tail is 14 reports across two days of concurrent sessions; archiving requires judging *their* reports' completion, which is not mine to do.
7. **A release-notes draft** collecting the dashboardui [Unreleased] section into `docs/proposals/`. Blocked on the same tag decision.

---

## d) TOTALLY FUCKED UP

1. **I added a public field before running the shape analyzer.** `NavLink.Verbatim` made `NavLink` a field-for-field twin of `navItem`; one `nix run .#check-branching-flow` *before* the edit would have told me the duplicate class existed and pointed straight at consolidation. Instead I shipped the field, discovered it from the gate afterwards, and did a second larger refactor (deleting `navItem`) that moved SARIF anchors and added 2 churn findings on top of the sibling's 5. Correct order: gate → shape → implement.
2. **I wrote the XSS test blind and broke the build twice.** `extra_nav_test.go` first failed to parse (`missing ',' before newline`, then `expected declaration, found '}'`) because I hand-wrote an HTML-escaping assertion without knowing the escaping form. Only then did I probe the real markup with a throwaway test and write exact assertions. Probing first would have cost one step instead of four.
3. **My first verbatim assertion loop condemned the extra entry itself** — it iterated every nav item asserting `!Verbatim`, including the consumer entry under test. Caught on the first run, fixed by skipping the entry's href, but it is the third time this session that an assertion was written against an imagined output instead of the real one.
4. **I ran the wrong `golangci-lint`.** Bare `golangci-lint run` is 2.14 ambient; the repo pins 2.15 everywhere. I read 13 findings as real (varnamelen, staticcheck, wsl, exhaustruct) before remembering the version-drift class. Wasted cycle, and the findings were pure noise.
5. **I trusted the coverage script's stale profile.** `check-coverage-threshold.sh dashboardui` reported `0.0% (threshold 60%)` from an August-era `coverage.out` referencing a deleted `format.go`. That is the documented trap (TODO §e6) and it still cost me a diagnosis cycle; the fix is to regenerate the profile first (which I then did).
6. **I closed the issue before running the composite gate.** At closure time the claims were unit-verified only; `check-modules` ran afterwards. Nothing in the comment turned out wrong, but the order should be gates → evidence → closure, and I had the tooling to do it that way.
7. **I reasoned about a working tree that no longer existed.** The auto-commit daemon had already committed most of my edits; my first `git status`/`git diff` reading was against a tree that did not match what I had just written. No damage, but it cost a confusing minute — the gotcha-51 discipline (verify HEAD before concluding anything about the tree) applies to my own edits too.

---

## e) WHAT WE SHOULD IMPROVE

1. **Gate-before-shape for type changes.** Any new/changed exported field or struct in `dashboardui`/`root` should be preceded by `nix run .#check-branching-flow` and `nix run .#check-reexport-policy`, not followed by it. One command would have merged two refactors into one.
2. **Probe, then assert, for rendered HTML.** A three-line throwaway test that logs the real markup beats three guessing iterations. Consider a repo convention: HTML assertions cite output captured from a probe.
3. **A workspace-mode lint entry point.** Authors currently cannot lint their own new code through any gate until tags publish; `nix run .#lint-workspace` (same config, `GOWORK` set) would close a real blind spot that this session had to work around by hand.
4. **Kill the stale-profile class properly** (TODO §e6): the per-module coverage script should regenerate or ignore a profile older than the newest source file. It has now cost two sessions.
5. **Record the ambient-binary traps in the AGENTS quick reference.** `golangci-lint` on PATH is 2.14 (wrong), `cqrs-lint` must run via the flake app, `branching-flow` needs the go floor. One table row per tool, one line each.
6. **Make the release-train TODO row scannable.** Row 23 is now a ~5 KB paragraph mixing five modules' unpublished API; a small table (module | unpublished symbol | consuming module | gate it reds) would let the next session decide in seconds.
7. **Give the docs ledger a same-change check.** The `10-32` huma report raised `CORPUS_BASELINE`/`TAIL_BASELINE` without a ledger sentence (50,841 → 51,547 unrecorded). Either a gate that fails a baseline bump without a ledger mention, or a docs-health checklist step.
8. **Decide the a11y/correctness backlog on `NavEntry` deliberately:** `aria-current`, dangerous-scheme rejection, and whether `Icon` should be required (an unset icon silently becomes `Link`). Three small decisions that keep accruing.
9. **Narrow the "written status report" default.** Concurrent sessions now produce 14 live reports; the tail gate's real budget is 3. A "small change → CHANGELOG + TODO row, no report" rule would keep the corpus from growing 200 lines per feature.

---

## f) NEXT — up to 50, ordered

**Unblockers first (owner-gated; nothing below works without them)**

1. Answer the train question (g1) so root `DropStats` + dispatch-boundary API can be tagged.
2. `nix run .#train-preflight` (S52: every stage, or a recorded reason).
3. `nix run .#wave-tag -- . v4.17.0` (root: `DropStats`, `NewDropStats`, `NewBroadcaster(opts)`, dispatch-boundary API).
4. `bump-dep` sweep for root consumers, then `wave-tag usermgmt` (session-audit decorator + `command_postures`).
5. `wave-tag adminui` (NavEntry/ExtraNav/DashboardURL + superbification wave).
6. `wave-tag dashboardui` — this session's `ExtraNav` + `NavLink.Verbatim`, and the honest changelog entry goes public with it.
7. `wave-tag setup` (`AdminExtraNav`, `AdminDashboardURL`, `DashboardExtraNav`).
8. `nix run .#push-and-watch` + `.#verify-remote-tags` after the train.
9. Re-run `nix run .#check-cqrs-lint` (hermetic, unblocked by 3–7) and close TODO row 41-class residue.
10. Re-run `nix run .#check-modules`; expect `module-isolation*`, `route-postures*` to flip green on the tag alone.

**Immediately after the train (cheap, high value)**

11. `examples/setup-demo`: show `DashboardExtraNav` — one sibling-surface link, one README line (deferred only because the leg was red).
12. `examples/admin-demo`: keep the admin equivalent visible in the same demo so the pair reads as one feature.
13. Release-notes draft under `docs/proposals/` harvesting dashboardui + setup `[Unreleased]`.
14. FEATURES.md: give `ExtraNav` its own row instead of a clause inside Integration Modes.
15. `dashboardui/README.md`: a one-line snippet for the custom-layout seat (`meta.Nav` + `Verbatim`), not just the bundle seat.
16. `setup/README.md`: a "consumer nav" paragraph in the mounting section pointing at both knobs.

**Code quality / gates**

17. `nix run .#lint-workspace` (or a `GOWORK=` override app) so authors can lint unpublished code (e4).
18. Coverage script: ignore profiles older than the newest source file (e4, TODO §e6).
19. `check-docs-corpus-budget`: fail a baseline bump whose delta has no ledger sentence.
20. `branching-flow`: the sibling's `huma/` + `command_fates`/`streamid-identity` findings need their owner's settle pass, then one re-pin.
21. Replace the two remaining `dashboardui/core/command_fates.go` identity-position `String()` calls with `.Get()` (owner: command-fates session; flagged here with the exact line).
22. Consider a `sidebar` extraction inside `dashboardui` (nav build + render + active) now that `NavLink` is the single type — it is a clean, testable unit.
23. `exhaustruct_v5` ignore-list review: `paginationState`/`eventFilter`/`routeSpec` are pinned; the nav types now need no exception, keep it that way.
24. Add an `ExtraNav` fixture to the golden suite so a future refactor cannot silently drop the extra link from the built-in sidebar.
25. Benchmark the nav build (`buildNav` + `extraNavItems` + clone) if anyone reports per-request cost — it is O(nav) and allocation-light; likely unnecessary (recorded as a deliberate skip).

**Decisions worth making (not code)**

26. `aria-current="page"` on active nav links (g3).
27. Dangerous URL schemes in `ExtraNav.Href` (g2).
28. Should `NavEntry.Icon` be required instead of defaulting to `icons.Link`? Silently substituting an icon is a small honesty problem.
29. Should `ExtraNav` support ordering relative to built-ins (e.g. `Before`/`After` a named item) or is append-only enough? Append-only is what shipped; confirm it is what consumers want.
30. Should `ExtraNav` entries be filterable per-role (an admin-only sibling surface)? Today a consumer must add it in `Config`, which is process-wide.

**Documentation debt observed this session**

31. `.agents/skills/cqrs-htmx/SKILL.md`: teach `ExtraNav` + `PageMeta.Nav` where the dashboard mounting path is described.
32. `docs/guides/` has no dashboard-embedding guide; the README carries it. If one is written, `Verbatim` must be in it.
33. `AGENTS.md` dashboardui bullet is now four sentences long; consider moving the config inventory to the module README and keeping a pointer.
34. TODO row 23 (release train): convert to the table shape — module | unpublished symbol | consumer | gate it reds (from e6).
35. Docs coastal cleanup: the 14-report tail needs an archive round (not mine to judge, but it is the next docs-health pass's first job).

**Session hygiene**

36. Batch HTML assertions behind a probe helper (`renderNavForTest(t, cfg) string`) so future nav tests do not each rebuild the mux.
37. Add a `TestNavLink_VerbatimContract` compile-level doc test? The field doc + README + PageMeta doc already triangulate; probably enough.
38. Record in AGENTS: devshell `GOWORK=off` means `nix develop -c golangci-lint` is hermetic (e5).
39. Make `setup` own a CHANGELOG (b6), or state explicitly in `CONTRIBUTING` that setup has none.
40. When the daemon commits mid-edit, `git log --since=10min` at session start (gotcha 51) — restate for one's own files.

---

## g) QUESTIONS (cannot answer from this seat)

1. **The train.** The unpublished surface now spans five modules and four sessions (root `DropStats` + dispatch-boundary, identity-model session events, usermgmt, adminui, dashboardui `ExtraNav`, setup passthrough). Do I prepare and cut that train now (preflight → `wave-tag` root → sweep), or does another session own the next train and I hold? This decides whether #38's consumers can actually `go get` it this week.
2. **Href scheme policy.** Should `dashboardui`/`adminui` `NavEntry.Href` reject `javascript:`/`data:`/`vbscript:` at `Validate`, the way `AccentColor` is validated because it is interpolated, or is templ's attribute escaping the intended and sufficient boundary?
3. **`aria-current`.** Should active nav links (built-in *and* extra) carry `aria-current="page"`? It is a genuine a11y improvement, but it changes every sidebar golden and both UI modules, so I want it named as a decision rather than smuggled in under #38.
