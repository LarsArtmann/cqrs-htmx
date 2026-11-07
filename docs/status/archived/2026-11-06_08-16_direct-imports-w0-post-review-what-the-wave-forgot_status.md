# 2026-11-06 08:16 — Direct Imports Only: W0 Post-Review Status (what the wave forgot)

> **ANNOTATED 2026-11-06** — ~~this report~~ **SUPERSEDED** by [`2026-11-06_08-51_direct-imports-w0-completion-pass-and-what-the-first-report-got-wrong_status.md`](2026-11-06_08-51_direct-imports-w0-completion-pass-and-what-the-first-report-got-wrong_status.md), which executes the three gaps it named (doc sweep, root test repoint, ADR class drift) and records the composite-battery attribution it declared unproven. Strike depth:
> supersession verdict (round-18 precedent for series-internal reports); nothing here is a
> unique open item. **ARCHIVED 2026-11-06 (round-23 (2026-11-06)).**

> **Scope.** A second, deliberately narrow pass over the same session, triggered by the question
> "what did you forget?". It is NOT a re-statement of
> `docs/status/2026-11-06_08-04_direct-imports-only-w0-reexport-facade-eviction_status.md`
> (the W0 execution report, 397 lines, committed) — that file carries the full evidence table, the
> six attested failures and the 50-item backlog. This file carries what that report MISSED and what
> changed after it.
>
> **Verified by execution in this pass:** `grep -rn` over every markdown file for the deprecated
> façade symbols (33 code-fence occurrences remain across 14 files, listed in §b1);
> `bash scripts/checks/check-skill-claims.sh`; `check-skill-snippets.sh`; `check-readme-splitbrain.sh`;
> `check-docs-links.sh` (1316 links) — all green AFTER the two corrections in §a1.
>
> **NOT verified — explicitly:** nothing was compiled or tested in this pass (no `go build`, no
> `go test`, no new lint run); the full `nix run .#check-modules` aggregate still has not been run
> (`§b3`); no consumer repository was inspected; no push was attempted.
>
> **Working-tree impact:** 2 files edited (prose only: `AGENTS.md` one clause, `.agents/skills/cqrs-htmx/SKILL.md`
> one numbered rule), 1 new status report, plus the baselines/ledger this report requires.
> **Still not pushed.**

---

## a) FULLY DONE

### a1. Two authoritative docs asserted the OPPOSITE of the policy this wave shipped — found and fixed

This is the answer to "what did you forget". W0 deprecated 77 façade aliases and shipped a gate, and
then left the two files every session reads FIRST saying the aliases are the API:

- `AGENTS.md` §Architecture said: *"Root keeps the published API through
  `htmx_reexport.go`/`eventcatalog_reexport.go`/`httperr_reexport.go` … so `cqrshtmx.HTMXTarget` and
  `htmx.Target` are the identical function value … public API unchanged."* That is now false in the
  way that matters: it tells the next session the façade is the contract.
- `.agents/skills/cqrs-htmx/SKILL.md` rule 1 said: *"`cqrshtmx` is still the supported surface."*

Both are rewritten to state the ADR-0062 position (the subpackage is the surface; the aliases are
deprecated with a `// Deprecated:` marker naming the direct import; the gate refuses a new alias
file), with the gate command and the ADR path inlined. `check-skill-claims`,
`check-skill-snippets`, `check-readme-splitbrain` and `check-docs-links` are green afterwards.

Notably: **no gate in this repo could have caught this.** `check-skill-claims` verifies that the
symbols a doc names still EXIST — the aliases do exist. The staleness was semantic.

### a2. The gap quantified rather than guessed

Rather than "the docs probably need a sweep", the actual residue is measured: 33 occurrences of a
deprecated façade symbol inside markdown code fences, in 14 files (§b1). That number is what makes
the follow-up schedulable, and it is the same instrument-first habit that made the 47/413/239 counts
possible in the 08:04 report.

### a3. W0 itself is unchanged and still green

ADR-0062, the scanner, the frozen 47-row baseline, the three-rule gate, the 13-case self-test, the
flake/CI wiring, the 77 deprecations, the 47 repointed call sites, the R11/R12/R13 codemod rules and
the migration guide are all committed and all still pass (`check-reexport-policy --strict`, its
self-test 13/13, `check-module-layers` 9/9, shellcheck over 129 scripts). Detail:
`docs/status/2026-11-06_08-04_…`.

### a4. Committed and baselined in the same change

The report, the two doc corrections, `docs/status/TAIL_BASELINE` (10 → 11) and
`docs/CORPUS_BASELINE` move together, with the ledger sentence appended to
`docs/status/README.md` — the repo's own rule for deliberate corpus growth.

---

## b) PARTIALLY DONE

### b1. 33 markdown occurrences still TEACH the deprecated aliases

Measured with `grep -rn` over `*.md`, excluding the files this wave already rewrote
(`docs/adr/0062-*`, `docs/migrations/v4-to-v5.md`, `docs/guides/v5-removal-inventory.md`,
`CHANGELOG.md`, `docs/status/`, `docs/planning/`):

| File | Occurrences |
| ---- | ----------- |
| `.agents/skills/cqrs-htmx/SKILL.md` | 16 (the Path 0 / Path A / Path C code fences and the discoverability notes) |
| `.agents/skills/cqrs-htmx/references/core-api.md` | 9 |
| `docs/guides/event-catalog-guide.md` | 7 |
| `README.md` | 2 |
| `AGENTS.md` | 1 (the Quick Reference table's `CSP`-adjacent note, still a live example) |
| `docs/guides/production-readiness.md`, `leveraging-httputil.md`, `gotchas.md`, `agents-notes.md`, `docs/adr/0034-…`, `docs/reviews/book-insights-…`, `docs/research/2026-10-12_…`, `docs/analysis/dedup-acceptance.md` | 1 each |

Every one of them still COMPILES, so no gate reds — that is why it survived. It is teaching drift,
not a break.

### b2. In-repo examples still call the aliases

`examples/`, `e2e/`, `setup/`, `dashboardui/`, `integration_test/` are unchanged (29 codemod-planned
edits, dry run only). Deliberate: W1.2/W1.6 repoint them through the codemod, one module per commit.

### b3. The aggregate battery still has not run

`nix run .#check-modules` (≈90 stages), `-race`, `.#coverage-gate`, `.#check-cqrs-lint`,
`.#check-erraudit-strict`, `.#check-branching-flow` — none executed this session. Each affected
stage was run individually and is green, which is evidence, but "the aggregate passes" remains an
inference.

### b4. Root's `_test.go` files still use the aliases

In-package, so staticcheck emits nothing, but the "zero self-use" claim holds for production files
only — as the 08:04 report states.

### b5. `api_surface_compat_test.go` untouched

Correctly deferred to W1.4: it pins the aliases, so it must be retired together with the files.

---

## c) NOT STARTED

- **Push.** 12 commits ahead of `origin/master`, all of W0 included. Boosted risk: the sibling
  session's in-flight work is inside several of those commits (see §d3), so a push publishes work
  I have not reviewed.
- **W1 / W2 / W3 / W4** — the v5 window has not been opened; no deletions, no extractions.
- **The five `exempt:pending` dispositions** (`authztest`, `dashboardui/core_bridge.go`,
  `datastar/patch.go`, `health/probe.go`, `usermgmt/sql_event_store.go`) — frozen, undecided.
- **The doc sweep in §b1** — not started; 33 occurrences.
- **Any tag or release.**

---

## d) TOTALLY FUCKED UP!

### d1. I never pushed, and I reported it as a neutral fact instead of a failure

The session's standing mandate included commit **and push**. W0 is 12 commits deep and entirely
local. Two pushes ago I even documented the exact blocker (`GIT_CONFIG_GLOBAL=/dev/null`, the stale
`go-appkit` rewrite) and then did not use it. "Nothing is pushed" appeared in the 08:04 report as a
status line, not as an open failure — that is the report doing the reader a disservice.

### d2. I left the two most-read files contradicting the policy I had just built

§a1. The gate suite is silent on semantic staleness, so the contradiction would have sat in
`AGENTS.md` until some future session re-created the façade — the precise failure mode ADR-0062
exists to prevent. I found it only because the owner asked "what did you forget", which is the
cheapest possible signal and I should have run it myself.

### d3. The auto-commit daemon still owns most of this session's history

Now 9 of 12 commits are `chore: auto-commit N changed file(s) (heuristic)`, and several bundle a
sibling session's files (`docs/adr/0063-*`, `authz_dispatch*.go`, `identity-model/authz_matcher_test.go`,
`authztest/command_suite*.go`, `.agents/skills/cqrs-htmx/*`, `docs/guides/authorization-system.md`,
and two untracked `scripts/checks/check-policy-rows.sh` / `scripts/selftests/test-check-policy-rows.sh`
that are not mine). Two `git commit` attempts died on `cannot lock ref 'HEAD'`, losing the messages
the owner explicitly asked for.

### d4. The plan's "zero blocker" row was reasoned, not measured

`authz_contract.go` uses five `httperr` symbols as same-package identifiers, so the `authz`
extraction carries a real `authz → httperr → htmx → internal/hx` edge. I propagated the plan's claim
into an ADR-adjacent statement before probing it. Recorded in plan §11.

### d5. The ADR's own numbers were briefly wrong

418/162 shipped before the plumbing-alias deletion changed them to 413/239. Fixed in place; the
corrected figures are the ones that survived.

### d6. Three mechanical self-inflicted do-overs

The `// Deprecated:` generator produced 77 `gocritic deprecatedComment` violations before I added the
required blank comment line; shellcheck SC1091 took three attempts before I read the repo's own
`# shellcheck disable=SC1091` convention; the codemod's new rules exposed a latent empty-qualifier
bug (`.New()` instead of `eventcatalog.New()`) that only the new tests caught.

### d7. I wrote a 50-item backlog and then ignored its own item 12

The 08:04 report lists "update SKILL.md §Subpackages … its point 1 is now FALSE" as §f item 12 and
then stopped. Writing the follow-up into a list is not the same as doing a two-sentence fix that
prevents the next session from repeating the mistake.

---

## e) WHAT WE SHOULD IMPROVE

### e1. Make "what did you forget" a mechanical step, not a reflection prompt

Concretely: after any change that deprecates or renames a symbol, grep every markdown file for that
symbol and classify each hit as *compiles-anyway example* vs *claim of fact*. The distinction in
§b1 is the whole value: examples can wait, claims cannot.

### e2. A policy change is not done until the canonical context files agree

`AGENTS.md` and the project skill are loaded first by every future session. They outrank a 400-line
report nobody re-reads. Any wave that changes a rule must edit those two files in the same commit.

### e3. Push is part of the wave

Finish the wave on the remote, not in the working tree. The host blocker is known and one env
override wide; treating it as a reason to defer is how a local-only branch becomes a merge incident
in a repo where another session shares the tree.

### e4. Cross-reference reports instead of re-writing them

This file deliberately does not repeat the 397-line execution report. Two reports answering the same
question differently is the split-brain class the repo already gates elsewhere.

### e5. Quiesce the daemon before any commit whose message matters

Check `git log --oneline -3` immediately before staging; if an auto-commit landed in the last minutes,
either wait or move to a worktree (`git worktree add ../cqrs-htmx-<thread> -b <branch>`).

### e6. Lint after the FIRST file of any bulk text transform

The gocritic class (d6) would have cost one file instead of 77 declarations.

---

## f) Up to 50 things we should get done next

> Brainstorm, not commitment. Items marked **[carried]** come from
> `docs/status/2026-11-06_08-04_…` and are restated only where they are still open.

### Tier 1 — close this session's own holes

1. ~~**Push** with `GIT_CONFIG_GLOBAL=/dev/null git push origin master`, after deciding the sibling~~ done — master pushed - the whole stack on origin, strict-lag 0 (CHANGELOG 2026-11-04 receipt; v4.18.0 bump 2c077269)
   ~~commit question (§g1). [carried, escalated]~~
2. ~~Sweep the 33 teaching occurrences in §b1, worst first: SKILL.md (16), `references/core-api.md`~~ done — completion pass swept 51 occurrences across skill + 9 guides; 10 deliberate contrasts remain (TODO_LIST W0 row)
   ~~(9), `event-catalog-guide.md` (7), README (2), then the eight single-hit files.~~
3. ~~Switch the skill's Path 0 / Path A / Path C fences to direct imports and re-run~~ done — skill fences switched; check-skill-snippets 43/43 + skill-claims 218 green (TODO_LIST W0 receipts)
   ~~`check-skill-snippets` + `check-skill-claims` + `.#check-readme-quickstart`.~~
4. ~~Same for `README.md`'s two occurrences, then run `.#check-readme-quickstart` (it compiles the~~ done — README quick-start fixed; check-readme-quickstart compiles (TODO_LIST W0 row)
   ~~fence in a throwaway consumer module).~~
5. ~~Add a `docs/guides/event-catalog-guide.md` note that `cqrshtmx.NewEventCatalog` is now~~ done — event-catalog-guide teaches eventcatalog.New + notes the deprecated aliases (guide lines 92, 176-177)
   ~~`eventcatalog.New` and deprecated.~~
6. ~~Run `nix run .#check-modules` and record the aggregate result. [carried]~~ done — check-modules battery receipts 2026-11-06 (TODO_LIST 09-48 row; W0 green list)
7. ~~Run `nix run .#test` with `-race` for root + `htmx` + `httperr` + `eventcatalog`. [carried]~~ done — root -race batteries green (root CHANGELOG v4.18.0)
8. ~~Run `.#coverage-gate`, `.#check-cqrs-lint`, `.#check-erraudit-strict`, `.#check-branching-flow`. [carried]~~ resolved — routed: TODO_LIST Plan-of-record 09-48 row - first wave carries the coverage-gate run; erraudit 319 / branching 210 / cqrs-lint 10/14 receipted
9. ~~Repoint root's `_test.go` files off the aliases. [carried]~~ done at `4380de92 - root 25 _test.go files repointed by dogfooding the codemod (TODO_LIST W0 row)`
10. Decide whether a semantic-staleness gate is buildable: "a file named `*_reexport.go` may not be
    called the supported surface in any markdown file". Cheap shell implementation; only worth it if
    item 2 shows the drift recurs.

### Tier 2 — finish the policy paperwork

11. ~~Update the plan header (`Status:` still reads "proposed — W0 is executable today"). [new]~~ done — plan Status header now reads accepted - W0 DELIVERED 2026-11-06; W1-W4 gated on the v5 window
12. ~~Add the `exempt:pending` class to ADR-0062's exemption table (it documents four; the baseline~~ done — ADR-0062 exemption table documents all five classes incl. exempt:pending (ADR line 63)
    ~~uses five). [carried, narrowed]~~
13. `docs/gotchas.md`: a gotcha for the empty-qualifier codemod class (`.New()`), if a second
    instance ever appears — do not pad the list speculatively. [new]
14. Record in ADR-0062 that a façade's deprecation must land in the same commit as the AGENTS.md and
    skill edits (the §d2 lesson, mechanised as convention). [new]
15. ~~`docs/guides/leveraging-httputil.md` + `production-readiness.md`: they also mention the aliases;~~ done — both guides swept by the completion pass - only deliberate contrasts remain (TODO_LIST W0 row)
    ~~add a one-line "deprecated, import httputil" pointer rather than rewriting the examples. [new]~~

### Tier 3 — the v5 window (W1)

16. ~~Write the single-window checklist and link it from `VERSIONING.md`. [carried]~~ resolved — routed: TODO_LIST direct-imports row - W1-W3 ride the single v5 window
17. ~~`grep -rn` every `/v4` consumer path outside `go.work`. [carried]~~ resolved — routed: TODO_LIST direct-imports row - W1-W3 ride the single v5 window
18. ~~`-apply` the codemod on `examples/`, then `e2e/`, `setup/`, `dashboardui/`, `integration_test/` —~~ resolved — routed: TODO_LIST direct-imports row - W1-W3 ride the single v5 window
    one module per commit. [carried]
19. ~~Re-run the dry run and prove 0 remaining R11/R12/R13 edits. [carried]~~ resolved — routed: TODO_LIST direct-imports row - W1-W3 ride the single v5 window
20. ~~Delete the three façade files; let the gate print the shrink note; lower the baseline. [carried]~~ resolved — routed: TODO_LIST direct-imports row - W1-W3 ride the single v5 window
21. ~~Retire the alias half of `api_surface_compat_test.go`. [carried]~~ resolved — routed: TODO_LIST direct-imports row - W1-W3 ride the single v5 window
22. ~~Delete the six deprecated root httputil/go-sse alias files plus the now-unused requires. [carried]~~ resolved — routed: TODO_LIST direct-imports row - W1-W3 ride the single v5 window
23. ~~Delete the 176 `usermgmt` → `identity-model` aliases, keep the constructor wrappers. [carried]~~ resolved — routed: TODO_LIST direct-imports row - W1-W3 ride the single v5 window
24. ~~Re-baseline `root-files.baseline` (9 files leave root) and the coverage thresholds. [carried]~~ resolved — routed: TODO_LIST direct-imports row - W1-W3 ride the single v5 window
25. ~~Decide the five `exempt:pending` files, one written reason each. [carried]~~ resolved — routed: TODO_LIST direct-imports row - W1-W3 ride the single v5 window
26. ~~Treat `datastar/patch.go` (63 aliases) as its own ADR. [carried]~~ resolved — routed: TODO_LIST direct-imports row - W1-W3 ride the single v5 window
27. ~~Treat `dashboardui/core_bridge.go` (17) as "the `core/` package is the import" and deprecate the~~ resolved — routed: TODO_LIST direct-imports row - W1-W3 ride the single v5 window
    bridge. [carried]

### Tier 4 — extractions (W2/W3)

28. ~~Measure `authz_contract.go`'s real dependency set before moving it (the §d4 lesson). [carried]~~ resolved — routed: TODO_LIST direct-imports row - W1-W3 ride the single v5 window
29. ~~Decide the denial-vocabulary home (`authz/` vs `httperr`) — gates the whole `authz` move. [carried]~~ resolved — routed: TODO_LIST direct-imports row - W1-W3 ride the single v5 window
30. ~~Extract `authz/` + repoint `authz.go`, `authztest`, `adminui`, `dashboardui`, `setup`. [carried]~~ resolved — routed: TODO_LIST direct-imports row - W1-W3 ride the single v5 window
31. ~~Add the `authz` layer-gate row + planted-violation fixture. [carried]~~ resolved — routed: TODO_LIST direct-imports row - W1-W3 ride the single v5 window
32. ~~Decide the readiness home; then extract `readiness/` and delete `health`'s root edge. [carried]~~ resolved — routed: TODO_LIST direct-imports row - W1-W3 ride the single v5 window
33. ~~Extract `sync/` with `App.SyncPushHandler` reduced to a delegate. [carried]~~ resolved — routed: TODO_LIST direct-imports row - W1-W3 ride the single v5 window
34. ~~Extract `realtime/`; settle `AfterDispatchHook` ownership. [carried]~~ resolved — routed: TODO_LIST direct-imports row - W1-W3 ride the single v5 window
35. ~~ADR-0038 type map, then move `HandlerOption`/`handlerConfig`/`App` into `internal/handler`. [carried]~~ resolved — routed: TODO_LIST direct-imports row - W1-W3 ride the single v5 window

### Tier 5 — repo hygiene this session kept tripping over

36. Make the daemon's message name the changed paths, so history stays readable. [carried]
37. Find out whether the daemon can yield to an in-flight `git commit` (two lock races). [carried]
38. Lead `docs/runbooks/session-start.md` with the one-session-per-worktree rule. [carried]
39. Fix the 33 pre-existing root lint findings (`wsl_v5` 27, `nlreturn` 3, `tagliatelle` 2,
    `exhaustruct_v5` 1) so `--new-from-rev` stops being the only honest signal. [carried]
40. ~~Resolve the two `gopls go mod tidy` warnings (`dashboardui` commandlifecycle, `usermgmt` catalog). [carried]~~ done — the two gopls tidy warnings fixed (TODO_LIST dispatch row, Landed list)
41. Add `reexport-policy` to the pre-push hook's legs so a push cannot be greener than CI on it. [carried]
42. Pin the ambient `golangci-lint` (2.14.0) to the flake's 2.15.0 story, or record the delta. [carried]
43. ~~Close the W0 row in `TODO_LIST.md` into `CHANGELOG.md` once W1 opens the v5 window. [carried]~~ resolved — routed: TODO_LIST direct-imports row - W0 receipts recorded; closure into CHANGELOG rides the W1 v5 window
44. Decide whether `docs/status` reports should be prettier-formatted (they are currently emitted as
    0 files, i.e. excluded — confirm that is deliberate and documented). [new]
45. ~~Re-read `docs/planning/2026-11-06_07-16_…` §1.2 and correct any remaining row that the execution~~ done — plan section 1.2 carries the re-measured 54/8,547 correction (plan header note)
    ~~disproved (only the authz row is known so far). [new]~~

### Tier 6 — smaller observations

46. The `docs/status` ledger sentence for this pass names ~113 lines of "sibling growth" — the same
    imprecision as before; keep the convention, do not chase the line. [new]
47. `scripts/checks/reexport_scanner.go` has no test of its own Go-AST path (fixtures use pre-baked
    TSVs). Add one tiny fixture module. [carried]
48. The gate's rule 2 accepts a deprecation that merely CONTAINS the package name; a stricter form
    (must appear in the `Deprecated:` paragraph specifically) was considered and not implemented. [new]
49. ~~`authztest/authztest.go`'s five `cqrshtmx.Posture` aliases are `exempt:pending`; W2's `authz`~~ resolved — routed: TODO_LIST direct-imports row - W2 authz extraction carries the authztest baseline edit
    extraction changes their target, so they will need a baseline edit in that commit. [new]
50. Nothing in this wave touched `go.mod`/`go.sum` — zero new dependencies, zero new modules. Worth
    stating in the release note when v5 is cut. [new]

---

## g) Three questions I cannot answer myself

**g1. Should I push the 12 local commits, knowing several contain another session's unreviewed work?**
`GIT_CONFIG_GLOBAL=/dev/null git push origin master` is the command. The content includes their
`docs/adr/0063-*`, `authz_dispatch*.go`, `identity-model/authz_matcher_test.go`,
`authztest/command_suite*.go`, the skill/reference edits and two brand-new untracked scripts
(`scripts/checks/check-policy-rows.sh` + its self-test) that are still uncommitted. PUBLISHING their
half-finished work is their decision, not mine — and the alternative (reset/rebase to separate my
commits) would modify their in-flight history, which I will not do unasked.

**g2. Does the 33-occurrence doc sweep belong in W0 or in the v5 window?**
Every occurrence still compiles, so no gate reds and nothing breaks between now and v5. Doing it now
keeps the documentation honest but touches 14 files and the skill's tutorial fences in a wave whose
stated scope was the policy + the gate; deferring means the repo teaches a deprecated API for the
whole v4 remainder. I have no way to judge which cost you care about more.

**g3. Should I remove the stale host git rewrite, or keep using the env override?**
`url./home/lars/projects/go-appkit.insteadof https://github.com/larsartmann/go-appkit` in your global
git config makes the pre-push version-drift leg fail on 4 phantom missing tags. Removing it fixes
every future push; keeping it means every push needs `GIT_CONFIG_GLOBAL=/dev/null`. It is your
machine configuration — I will not edit it without an explicit yes.
