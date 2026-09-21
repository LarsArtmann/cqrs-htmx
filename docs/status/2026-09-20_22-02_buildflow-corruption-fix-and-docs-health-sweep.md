# Status Report — BuildFlow Fixture-Corruption Fix + Docs-Health Sweep (in progress)

**Date:** 2026-09-20 22:02 CEST
**Session scope:** Resume the 2026-09-20 docs-health AUDIT: diagnose + repair a BuildFlow hook that corrupted the tree, consolidate the `docs/status` archive split brain, and start the annotate-and-archive sweep over the 36 unarchived reports. Report written on request; execution **paused** pending further instructions.
**Tree state:** clean at `d4c7d34e` (daemon); all work below is committed.

> **ANNOTATED 2026-09-21** (docs-health sweep A5/A6): the BuildFlow repair and archive merge held; the annotate sweep this report started has since completed.
> - **§a:** a1–a14 verified — `go-version-auto-configure` still skipped in `.buildflow.yml`, fixtures intact (`scripts/test-verify-tag.sh` green), `docs/status/archive/` gone, `archived/` now 402 files.
> - **§b:** b1 sweep → DONE (A1–A4); b2 README stale → DONE (A5); b3 completeness gates → DONE (A6, `scripts/check-status-annotations.sh` + adjudicated row gate); b4 CHANGELOG → B1; b5 AUDIT report → A7 (this session); b6 living-doc repairs verified.
> - **§c:** all items DONE or routed — README rewrite (A5), legacy-subset decision (q2 → date-scoped LEGACY-EXEMPT, documented), gates (A6), AUDIT report (A7), planning archive (A8), DOMAIN_LANGUAGE (A9), AGENTS size (A10), docs-lint gate (B2), C/D tiers active; the one genuine row-format deviation (`2026-08-05_11-46`, 2 rows) normalized in A6.
> - **§d / §e:** incidents + lessons — historical record (the BuildFlow corruption is the canonical AGENTS.md entry).
> - **§f:** f1–f7 and f9–f10 DONE or routed above; f8 (tooling into the repo) partially DONE — the annotation gate now lives in `scripts/`; f11–f15 routed (B2, A9, C/D tiers; heuristic commit history accepted, no daemon pause).
> - **§g:** q1 annotation scope → **date-scoped convention** (`docs/status/README.md`); q2 legacy 92 files → **LEGACY-EXEMPT**; q3 daemon attribution → **accept heuristic history** (documented; no pause or wrapper).

---

## Headline

A **repo-corrupting BuildFlow step** was found, root-caused from its vendored source, repaired, and permanently neutralized — plus its sibling config bug (a user `exclude:` list silently replaced BuildFlow's defaults). Separately, the `docs/status` **archive split brain is fixed** (92 files merged, `archive/` gone), and the annotate+archive sweep is **2 of 36 reports done**.

Two independent defects were found, both with the same blast radius: silent loss of the verify-tag regression guard.

---

## a) FULLY DONE

### 1. BuildFlow corruption — diagnosed, repaired, neutralized

| #   | What                                                                                                                                                                                                                                                                                                                                                            | Evidence                                                                                     |
| --- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------- |
| a1  | **Corruption identified.** Commit `27057e5c` (auto-commit daemon, 14 files) downgraded the deliberate `go 1.27.1` floor to `go 1.27` in `go.work` + 4 module go.mods, and **deleted every `require` line** from 7 `scripts/testdata/verify-tag/*/go.mod` fixtures (incl. 107 lines from `real-setup-v4.8.1-poisoned`).                                       | `git show 27057e5c --stat`; diff read in full                                                       |
| a2  | **Root cause proven from vendored source.** The step `go-version-auto-configure` (BuildFlow vendor `larsartmann/go-version-auto-configure`) hardcodes a major.minor policy (`RuleGoDirectivePatchForm`) and, after `go mod edit`, runs `ensureTidyStable` → **`go mod tidy`**. The fixture modules have no `.go` files, so tidy removed all requires; the step then re-read the directive, saw success, and reported "36 fixed". Its walker skips only `vendor/node_modules/.git/result` — **not `testdata`**. | `pkg/surface/discover.go:18` (`skippedDirs`), `pkg/fix/fix.go:220-286` (`applyOne`/`ensureTidyStable`) |
| a3  | **Impact quantified.** `scripts/test-verify-tag.sh` (12 cases) was passing **vacuously** — the 5 sensitive cases had nothing left to reject. The guard protecting tag hygiene was silently dead.                                                                                                                                                                     | Restored fixture set → suite back to 12/12 (see a5)                                                 |
| a4  | **Tree restored** from the pre-corruption commit. 12 files: `go.work`, `integration_test/go.mod`, `usermgmt/{oauth2,totp,webauthn}/go.mod`, and all 7 fixtures.                                                                                                                                                                                                   | daemon `6a5a0f6c` (12 files)                                                                        |
| a5  | **Regression suite re-verified green: 12 passed / 0 failed.**                                                                                                                                                                                                                                                                                                   | `bash scripts/test-verify-tag.sh` → 12/12                                                           |
| a6  | **Step neutralized.** `go-version-auto-configure` added to `.buildflow.yml` `skip_steps` with a documented re-enable condition (upstream must prune `testdata` from its walk AND expose a go-floor policy knob).                                                                                                                                                 | `.buildflow.yml`; commit `234a105a`                                                                 |
| a7  | **Proven inert.** A real `buildflow --build-mode pre-commit --staged-only` run leaves all go directives at `1.27.1` and every fixture byte-identical; dry-run prints `go-version-auto-configure (skipped via skip_steps config)`.                                                                                                                                    | `git status` clean post-run; sha256 fixture diff empty                                              |
| a8  | **Sibling config bug fixed: `exclude:` REPLACES BuildFlow's defaults** (verified via `buildflow config view` — it showed only our 3 entries). `.buildflow.yml` now mirrors `DefaultExcludePatterns()` explicitly (34 patterns) plus the repo's `*.html`/`*.min.js`/`*.sh`, so `*_templ.go`, vendor, testdata and lock files stay excluded from formatters.       | `buildflow config view` → 37 excludes; `buildflow config validate` → all checks passed            |
| a9  | **Gotcha recorded** in `AGENTS.md` (root cause, recovery recipe, `env -u GOTOOLCHAIN` commit note) and a re-enable tracking item added to `TODO_LIST.md`.                                                                                                                                                                                                       | daemon `9ea1fa81` (AGENTS.md + TODO_LIST.md)                                                        |

**Note on commits:** my deliberate commit attempt lost the daemon race; the content landed as heuristic chunks `6a5a0f6c` / `234a105a` / `9ea1fa81`. The exact documented failure mode (AGENTS.md "Commit at phase boundaries, never at the end").

### 2. Codegen integrity restored

| #   | What                                                                                                                                                                                                            | Evidence                                                                   |
| --- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------- |
| a10 | The earlier `27057e5c` had collapsed `adminui/audit.templ`'s import block but not regenerated `audit_templ.go`. The generated file was brought back in sync (import collapse + `FileName`/Line/Col shifts).        | `git show 234a105a -- adminui/audit_templ.go`; `nix run .#check-codegen` → **Codegen drift check PASSED** |

### 3. docs/status archive split brain — RESOLVED

| #   | What                                                                                                                                                                                            | Evidence                                                              |
| --- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------- |
| a11 | **Merged `docs/status/archive/` (92 files, 2026-05-03 → 2026-06-14) into `docs/status/archived/` (273, 2026-06-16 → 2026-09-07)** — they were complementary halves, not duplicates (0 filename collisions). `git mv` used throughout; `archive/` removed. | `git status` 92 renames staged; `ls docs/status/archived/*.md` → **365** |
| a12 | **Stale link repaired:** the one repo reference to the old path (`docs/research/2026-08-02_datastar-integration-analysis.md:185`) now points at `archived/`; target verified to exist.                                                                          | `edit` applied; `ls` target OK                                          |

### 4. Annotate + archive sweep — STARTED (2 of 36)

| #   | What                                                                                                                                                                                                                                                        | Evidence                                                       |
| --- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------- |
| a13 | **09-09 train-bump report**: 14 table rows struck (`~~…~~ done at …`) across §c and §f + dated `> ANNOTATED 2026-09-20` block enumerating per-section verdicts.                                                                                                | file in `archived/`; `grep -c '~~'` = 14                       |
| a14 | **09-09 docs-health-sweep report**: 19 rows struck (§b/§f) + block; the report's own open question §g1 ("ratify block-only or convert to literal strikethrough?") is **answered: inline strikethrough ratified** by the user's directive, and the pass converts it. | file in `archived/`; `grep -c '~~'` = 19                       |

---

## b) PARTIALLY DONE

1. ~~**Annotate + archive sweep — 2 of 36 reports.** The remaining 34 unarchived reports (2026-09-10 → 2026-09-20) are unannotated and unarchived. `docs/status/*.md` still holds **37 reports** (README convention: 3).~~ — done (verified 2026-09-21)
2. ~~**`docs/status/README.md` is stale.** It still claims "273 annotated + archived reports (2026-06-16 → 2026-09-07)" and "3 unarchived + 273 archived" (now 367 archived / 37 unarchived), and does not document the legacy pre-2026-06-16 unannotated subset from the merge.~~ — done (verified 2026-09-21)
3. ~~**`docs-health` completeness gates not run.**~~ — done: A6 runs both; the presence gate now lives in `scripts/check-status-annotations.sh` (date-scoped to the 2026-09-09 convention era) and the row gate is adjudicated (8 mixed-table files, 0 PARTIAL rows).
4. ~~**CHANGELOG entry not written** for this session (BuildFlow fix + archive merge + README).~~ — done (verified 2026-09-21)
5. ~~**AUDIT health report not printed** (Accuracy + Fitness scores, per-doc table).~~ — done (verified 2026-09-21)
6. ~~**docs-health living-doc repairs** (README `/v4` bug, FEATURES/ROADMAP/AGENTS/TODO_LIST refresh) were completed earlier in this session and committed; they are **not** re-verified here beyond the checks in §Verification.~~ — done (verified 2026-09-21)

---

## c) NOT STARTED

- **34 remaining reports**: annotate (inline strikethrough + block) and `git mv` to `archived/`; then reduce `docs/status/` to the 3 most recent.
- **`docs/status/README.md` rewrite**: counts (367 archived), date range (2026-05-03 → 2026-09-07), merged-layout table, and a note that pre-2026-06-16 files are legacy-unannotated.
- **Legacy-unannotated subset decision**: the 92 merged files (2026-05 → 2026-06-14) carry no annotation; decide annotate-as-batch vs declare-legacy in the README.
- **Completeness gates** (`grep -rLn '~~'`, `check-rows.py`) and any offender fixes.
- **AUDIT health report** (inline print, not filed).
- **The `docs/planning/…SUPERB-plan.md` downstream phases**: C1–C10 (code/verify backlog) and D1–D8 (upstream/tooling/hygiene) — untouched.
- **AGENTS.md size question** (120 KB; skill rubric calls >100 KB "Broken") — decision pending.
- **`docs/DOMAIN_LANGUAGE.md` freshness pass** (plan A9).
- **docs-lint gate** rejecting `/v4`-less `larsartmann/cqrs-htmx/*` imports (plan B2).

---

## d) TOTALLY FUCKED UP (honest)

| #   | Incident                                                                                                                                                                                                                                                                                                                                                                                              | Root cause                                                                                                                                                | Mitigation / lesson                                                                                                                                                                                                                        |
| --- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| d1  | **The corruption was introduced by MY commit attempt.** Running `git commit` for a plan file triggered the hook, which rewrote 14 files; the auto-commit daemon then committed the damage as `27057e5c`. I did not read the hook's output before the daemon acted.                                                                                                                                        | I did not expect a content-scoped BuildFlow step to rewrite go.mod/go.work during a commit that only staged a `.md`; the hook failed (exit 1) so I assumed "no commit = no changes". The step mutates files **before** the workflow verdict. | Never trust "commit failed" to mean "tree untouched" when buildflow ran. After any failed hook run, `git status` immediately. Root cause is now documented + the step is skipped.                                                            |
| d2  | **I restored files from the parent commit with `git restore`, then staged them with `git add -A`** — which also staged a stray scratch file (`TODO_LIST.md.baktest`) I had created moments earlier with a careless `cp`. It briefly appeared in the index.                                                                                                                                             | Sloppy scratch-file hygiene in the repo root.                                                                                                             | Fixed (unstaged; file removed). Lesson: create scratch files only under `/tmp`, never in the repo root.                                                                                                                                    |
| d3  | **Lost the deliberate commit race again** — the BuildFlow config fix + gotcha landed as `chore: auto-commit` chunks (`6a5a0f6c`, `234a105a`, `9ea1fa81`) instead of narrative commits. This is the **third+ recurrence** of the AGENTS.md phase-boundary gotcha.                                                                                                                                       | The daemon polls faster than my verify→commit cadence; I batched verification before committing.                                                          | The rule is documented and still violated. Real fix is mechanical (commit immediately after each file/phase, or a daemon pause during agent sessions) — raised as §g.                                                                        |
| d4  | **Two `annotate-rows.py` invocations failed on the wrong file shape** (`00-42` report's §f is a *prose* numbered list, not a table). The tool failed atomically (0 rows written) — which is correct behavior — but I had to stop and re-read the tooling to discover `annotate-prose.py`.                                                                                                                | I assumed table shape from the previous file without grepping the actual section format (the exact mistake the 09-09 report itself recorded).             | Inspect the section shape first (grep), then choose `annotate-rows` vs `annotate-prose`. Cost: ~2 wasted calls. The tool's atomic refusal prevented damage.                                                                                  |
| d5  | **I nearly over-reached on scope.** Mid-sweep I considered converting the *entire* 365-file historical corpus (~500+ lines) to literal strikethrough, which is an L-effort, high-Verschlimmbesserung task the repo README explicitly warns against.                                                                                                                                                     | Enthusiasm to satisfy the literal wording of the skill/user directive without a cost/value check.                                                          | Scoped back to the 36-report unarchived tail (the actual debt) + a README policy note for the legacy corpus; flagged as §g for the user to ratify the corpus-wide scope.                                                                    |

---

## e) WHAT WE SHOULD IMPROVE

1. **Treat "hook failed" as "inspect the tree, now."** BuildFlow repairs run before the verdict; a non-zero exit does not imply a clean working tree. Cheap guard: `git status --short` after every failed commit.
2. **Neutralize destructive auto-fixers at the config layer, not by vigilance.** The fix here (skip + restore default excludes) is durable; the lesson generalizes — a formatter/auto-configure step that can write go.mod/go.work is a supply-chain hazard for fixture corpora.
3. **Finish the annotate sweep with the right tool per shape** (`annotate-rows.py` for tables, `annotate-prose.py` for prose lists), grepping the section first. Both fail atomically; dry-run first.
4. **Keep the archive a single canonical dir.** `archive/` vs `archived/` was a two-directory split brain for weeks. A `check-docs-freshness` assertion that there is exactly one `docs/status/archived*` dir would catch the next one.
5. **Update directory-level meta files in the same pass as bulk moves** — `docs/status/README.md` went stale again (third recorded time). Add a count-drift assertion (documented count vs `find | wc -l`) to `check-docs-freshness`.
6. **Decide the corpus-wide annotation standard once** (block+inline vs literal strikethrough on every line) and document it in `docs/status/README.md`, so future sweeps don't relitigate it.
7. **Commit at phase boundaries, mechanically** — three sessions have now documented and then violated this rule. A wrapper/daemon-pause is overdue.

---

## f) TOP NEXT TASKS (ranked, harvest input)

| #  | Task                                                                                                                                                     | Impact | Effort | Category      |
| -- | -------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ | ------ | ------------- |
| 1  | Finish annotate+archive for the 34 remaining unarchived reports (use rows vs prose per shape; block + inline strikethrough; `git mv` each)                | High   | L      | Documentation |
| 2  | Update `docs/status/README.md`: counts (367 archived / 3 kept), date range, merged layout, legacy-subset note, annotation-standard section                 | High   | S      | Documentation |
| 3  | Run docs-health completeness gates (`grep -rLn '~~'`, `check-rows.py`) over the newly annotated set; fix offenders                                        | High   | S      | Documentation |
| 4  | CHANGELOG `[Unreleased]` entry for this session (BuildFlow fix, archive merge, README `/v4` repair)                                                      | Med    | S      | Documentation |
| 5  | Print the AUDIT health report (Accuracy + Fitness, per-doc table, visible math) inline                                                                   | Med    | S      | Documentation |
| 6  | Decide + document the annotation standard for the legacy corpus (92 merged files) — annotate-as-batch vs declare-legacy                                   | Med    | S      | Documentation |
| 7  | Add `check-docs-freshness` assertions: single `archived/` dir + documented-count-vs-actual within N                                                       | Med    | M      | Tooling       |
| 8  | Wire the sweep tooling into the repo (`scripts/docs/`) so the next docs-health pass is one command, not throwaway Python (the 09-09 report's §e3 ask)     | Med    | M      | Tooling       |
| 9  | Report the upstream bugs: (a) `go-version-auto-configure` walks into `testdata` and tidies fixture modules; (b) BuildFlow `exclude:` replaces defaults     | Med    | S      | Upstream      |
| 10 | Decide the AGENTS.md size plan (currently 120 KB; rubric "Broken >100 KB")                                                                               | Med    | M      | Documentation |
| 11 | docs-lint gate: reject `/v4`-less `larsartmann/cqrs-htmx/*` import paths in living docs (prevents the README bug class)                                  | Med    | M      | Tooling       |
| 12 | Verify `docs/DOMAIN_LANGUAGE.md` against code (plan A9)                                                                                                  | Low    | M      | Documentation |
| 13 | Execute the SUPERB plan's C-tier (bench idle re-run, V007 cluster 1/2, templ-components v1.19 prep)                                                       | High   | L      | Code          |
| 14 | Execute the SUPERB plan's D-tier (upstream filings, gate hardening, blob-purge prep, example smoke tests, ROADMAP triage)                                 | Med    | L      | Mixed         |
| 15 | Authored commit messages: get the config fix into a narrative commit (or accept heuristic history and stop relitigating)                                 | Low    | S      | Process       |

---

## g) QUESTIONS I CANNOT FIGURE OUT MYSELF

1. **Annotation scope for the historical corpus.** The 365-file `archived/` corpus predominantly uses the *blockquote + `✅`* convention (only ~25 files carry `~~`). Do you want (a) the literal-strikethrough standard applied to the **unarchived tail only** (my current scope — 36 reports), (b) the whole 365-file corpus converted (~500+ lines; L-effort, real Verschlimmbesserung risk on 2-month-old reports), or (c) ratify the house block convention and keep literal strikethrough only for new/active reports? I executed (a) and stopped here.
2. **Legacy subset (92 files, 2026-05 → 2026-06-14).** These merged files carry no annotation at all. Annotate them as a batch, or declare them "legacy pre-convention, retained as history" in `docs/status/README.md`?
3. **Daemon + commit attribution.** The phase-boundary rule has now been documented and violated three sessions running. Do you want (a) a daemon pause during agent sessions, (b) accept heuristic history (my pragmatic default), or (c) a `scripts/commit-phase.sh` wrapper enforced by the hook?

---

## Verification snapshot (2026-09-20 ~22:00 CEST)

| Check                                                             | Result                                                                     |
| ----------------------------------------------------------------- | -------------------------------------------------------------------------- |
| Go directives (modules / fixtures)                                | ✅ 28 × `1.27.1` / 7 × `1.26.7` (fixtures intentional)                     |
| `scripts/test-verify-tag.sh`                                      | ✅ 12 passed / 0 failed                                                    |
| `buildflow` dry-run: `go-version-auto-configure`                   | ✅ `skipped via skip_steps config`; no fixture/directive churn             |
| `buildflow config validate`                                       | ✅ all checks passed (37 exclude patterns)                                 |
| `nix run .#check-codegen`                                         | ✅ Codegen drift check PASSED                                              |
| `docs/status/archive/`                                            | ✅ gone; `archived/` = 367 files                                           |
| Working tree                                                      | ✅ clean at `d4c7d34e`                                                     |
| NOT re-run this segment                                           | coverage-gate (15/15 green earlier today), lint (0/15 earlier), race/fuzz/flake/e2e/bench |

---

_Paused for instructions. The next natural step is task f1 (finish the annotate+archive sweep) or f2/f3/f4 (close the docs loop), then the SUPERB plan's C/D tiers._
