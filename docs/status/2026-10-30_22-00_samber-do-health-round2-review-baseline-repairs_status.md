# Status: samber/do × Health Round-2 Review, Baseline Repairs, Gate Sweeps

> **Date:** 2026-10-30 22:00 CET
> **Scope:** this session only — the round-2 deep review of samber/do v2 ×
> health checks, the corpus/tail baseline repairs it surfaced, and the gates
> actually run. No repo areas beyond that were touched or triaged.
> **Session commit:** `c49eaf3a` (daemon: report + round-1 annotation +
> CORPUS_BASELINE repair + treefmt table re-padding).

## 0. One-paragraph summary

Executed the round-2 deep review answering "are we using samber/do v2 ×
health checks superbly / is the architecture proper SO/composable/resilient/
self-health": verified every round-1 roadmap item against source (all landed
or owner-gated), re-ran the DO-1→DO-6 sweep (clean), re-ran all four touched
test suites under `-race` (health 100.0%, auditlog 100.0%), verified version
currency against upstream tags, and published
`docs/research/2026-10-30_samber-do-v2-health-checks-deep-review-round-2.md`
(verdict **9/10, up from 8/10**). En route: found and fixed a committed
merge-conflict corruption in `docs/CORPUS_BASELINE`, discovered the
docs-tail ratchet already red before the session (4 > 3), and surfaced six
new findings (F1-F6), of which zero are executed — all ticketed in the
report's roadmap.

---

## a) FULLY DONE

| # | Item | Receipt |
| - | ---- | ------- |
| 1 | Round-2 deep review report (346 lines) — both questions answered with source-verified evidence | `docs/research/2026-10-30_samber-do-v2-health-checks-deep-review-round-2.md`, committed `c49eaf3a` |
| 2 | Round-1 roadmap disposition verified item-by-item (8/8: landed or explicitly owner-gated — nothing silently rotted) | report §2 receipt table; every claim checked in code, not changelogs |
| 3 | Round-1 report annotated per docs/research append-only series convention (superseded-by pointer) | `2026-10-08_...md` outcome note, +7 lines |
| 4 | DO-1→DO-6 anti-pattern sweep: fully clean with per-rule receipts (zero Must* in non-test, Override only in tests, no package-level injectors, self-contained shutdowns) | report §4; `rg` evidence re-scanned 2026-10-30 |
| 5 | Test suites re-run green under `-race`: health **100.0%**, auditlog **100.0%**, samber-do-demo, integration health/do/docs seams | `go test` outputs this session |
| 6 | Version currency verified: samber/do v2.1.0 = upstream-latest v2 tag (ls-remote); go-health v0.8.1 + samber-do-auditlog v0.11.0 treadmill-aligned; go-health v0.8.1 API claims (ChainRecorders, ErrRecorderNotApplicable, NewWithDetailedCheck, DetailedHealthRecorder) verified in module-cache source | treadmill run + cache reads |
| 7 | `docs/CORPUS_BASELINE` repaired — it was committed containing **unresolved merge-conflict markers** (daemon commit `158e86f6`), which broke the blocking `--ratchet` parse for every check-modules/CI run | resolved to measured count 35220; `--ratchet` green |
| 8 | Gates green after changes: corpus ratchet (35220=35220), docs links (919 OK), docs freshness, `nix fmt` (treefmt canonical; 3 files table re-padded) | gate outputs this session |
| 9 | Six new findings documented + ranked (F1-F6) with a 6-item action roadmap in the round-2 report | report §7, §10 |

## b) PARTIALLY DONE

| # | Item | Done | Missing |
| - | ---- | ---- | ------- |
| 1 | Worker-state/error-code unification (R1 roadmap 5 / T12) | worker states **exported** from root (`ProjectionStatus*` consts + `ProjectionDrainReady`); health's private string mirror is dead | error codes still split (`projection.*` vs `cqrshtmx.health.*`); T12 unexecuted |
| 2 | Plain-`Run` drain story (T13) | docs posture landed (`setup/README` §Drain posture: RunWithAppkit is production) | `Bundle.MarkDraining()` still owner-gated; `run.go` unchanged |
| 3 | Deprecated-alias truth (F2) | found: `auditlog/README.md:46` + health README surface-map row still teach deprecated `health.RecorderChain` | examples not updated, deprecation decision not made |
| 4 | InvokeAs wording alignment (F3) | found: gotcha 37 + `main.go:8` claim patterns the demo code doesn't use | three texts, two patterns — none corrected |
| 5 | Vacuous-readiness disclosure (F4) | found: `DisableService` + empty HealthChecks = constant-200 `/health` | README clause / New-time warning missing |
| 6 | go-health-dashboard bump (F5) | LAG 1 detected (v0.12.1 → v0.12.2) | bump not executed (family-wave gotcha 38 applies) |
| 7 | DO analyzer mechanization (F6) | gap documented: DO-1→DO-6 enforced by review only; doanalyzerv2 not wired | gate not built (blocked on installable binary) |
| 8 | HARVEST of report roadmap into TODO_LIST | report §10 written as HARVEST input | TODO_LIST.md NOT updated this session — loop not closed |

## c) NOT STARTED

- T12 execution (train + CHANGELOG receipt) — untouched
- T13 execution (MarkDraining or close-as-won't-fix) — untouched
- F2/F3/F4 doc fixes — untouched
- go-health-dashboard v0.12.2 bump — untouched
- doanalyzerv2 gate — untouched (owner-gated)
- Full `nix run .#check-modules` battery — this session ran only scoped gates (corpus, tail, links, freshness, fmt, treadmill) + touched-module tests
- Upstream filing of the `do.InvokeAs` interface-scan contract — queued since round 1 (owner decision: samber/do issue vs internal rule), still queued

## d) TOTALLY FUCKED UP

1. **`docs/CORPUS_BASELINE` was committed broken** (merge-conflict markers: `<<<<<<< Updated upstream 34777 ======= 34124 >>>>>>> Stashed changes`). The blocking corpus ratchet — wired into check-modules AND CI — would exit 1 "must contain a single integer" on every run. Pre-existing damage (daemon stash-pop residue committed in `158e86f6`); found by accident mid-session because I happened to `cat` the file. **Fixed this session** (measured 35220), but the class is the risk: a one-number gate file corrupted and committed without any gate screaming until run.
2. **Docs-tail ratchet already red before this session**: 4 live status reports vs baseline 3 — check-modules/CI would fail on it too. Being fixed right after this report (archive ritual), but it sat red through round 44's close-out.
3. **My own process misses (honest):**
   - Ran `nix fmt` **before** loading the buildflow skill (skill-order violation; outcome still correct — repo AGENTS.md says treefmt/`nix fmt` is canonical — but the order was wrong).
   - Triggered the `rg -rn` replace-trap **again** (hook block; the hook itself counted it as the *third strike* of this exact trap on this host).
   - Two wasted tool cycles: module-cache `rg` with a glob path (IO error ×1), wrong flag `--update` for the corpus gate (usage error ×1).
   - **Forgot the memory-maintenance rule at discovery time**: AGENTS.md's health-module bullet still cites only the 2026-10-08 review; the round-2 report should be cited there too. Not done yet.
   - Didn't run HARVEST before yielding — the round-2 roadmap lives in a timestamped file, which the docs-health skill explicitly calls "entombed" until harvested.

## e) WHAT WE SHOULD IMPROVE

1. **Baseline/integrity files need corruption resistance** — the corpus gate caught the malformed baseline only when run; nothing prevented the corrupted file from being committed. A conflict-marker lint over tracked files (or a session-start check) kills the class, not the instance.
2. **Tail discipline**: adding a status report must mechanically pair with the archive ritual in the same session — the red tail survived a full close-out round unnoticed.
3. **Error-code governance**: root itself now mixes three prefix styles (`projection.*`, `cqrshtmx.readiness.*`, `cqrshtmx.sse.*`); T12 should be scoped as one naming decision, not a two-file patch.
4. **Doc deprecation hygiene**: a deprecation whose own READMEs don't follow it (F2) will never die — deprecations need an examples-migrated-in-same-change rule.
5. **Failure-path tests for the demo**: the demo suite pins all the pass paths (real verdicts, latch, dashboard content) but has no test injecting a failed critical projection → `/readyz` 503. The resilience claim rests on go-health semantics, not on a repo test.
6. **Recorder failure-mode consistency**: `NewProbe(nil)` returns an error; `Recorder(nil)` panics. Both documented, but two failure modes for sibling constructors is a smell worth one decision.
7. **Skill ordering discipline**: load buildflow (and any gate-adjacent skill) before touching formatters/gates, and re-check the `rg -r` trap before any `rg` with flags beyond `-n`.

## f) UP TO 50 THINGS TO GET DONE NEXT

Grouped, impact-ordered. Items 1-11 are session-grounded execution; 12-50 are the adjacent backlog this session's evidence supports.

**P0 — decisions that unblock trains**
| # | Thing | Why now |
| - | ----- | ------- |
| 1 | Decide T12 scope: only the health-vs-root pair, or all root prefixes (`projection.*` / `cqrshtmx.readiness.*` / `cqrshtmx.sse.*`) in one naming packet | every alert on health codes waits on it; wire-visible → needs train slot |
| 2 | Cut the next health module train carrying the `RecorderChain` deprecation + duration_ns receipts | integration_test now asserts PUBLISHED behavior; the chain proof is already landed in-repo |
| 3 | Decide `health.RecorderChain` fate: migrate docs to `gohealth.ChainRecorders` (recommended) or undeprecate as the family-readable alias | kills the deprecation split-brain (F2) |
| 4 | Decide T13: implement `Bundle.MarkDraining()` or close as won't-fix (RunWithAppkit = production drain path) | open owner-gated item since round 1 |
| 5 | Decide the `do.InvokeAs` interface-scan upstream filing (samber/do issue vs internal rule only) | queued since round 1; gotcha 37 depends on the outcome |

**P1 — XS fixes, this week**
| # | Thing | Effort |
| - | ----- | ------ |
| 6 | Update `auditlog/README.md:46` example to the chosen chain form | XS |
| 7 | Update health README surface-map row 3 to the same form | XS |
| 8 | Fix gotcha 37 parenthetical: demo uses type-keyed `do.As` + single-satisfier rule, not `AsNamed` | XS |
| 9 | Fix `examples/samber-do-demo/main.go:8` package comment ("ProvideNamed" claim is stale) | XS |
| 10 | Document `DisableService` + empty `HealthChecks` = constant-200 `/health` (README row or New-time warning) | XS |
| 11 | Cite the round-2 review from AGENTS.md's health bullet (memory maintenance, forgotten this session) | XS |
| 12 | HARVEST round-2 report §10 + this report's (f) into TODO_LIST/ROADMAP (docs-health HARVEST) | S |
| 13 | Bump go-health-dashboard v0.12.1 → v0.12.2 as a family wave (gotcha 38 rules; class-set check if classes drift) | S |
| 14 | Run the full `nix run .#check-modules` battery (this session ran scoped gates only) | M |

**P2 — resilience/test depth**
| # | Thing | Effort |
| - | ----- | ------ |
| 15 | Demo failure-path test: failed critical projection → `/readyz` 503 (resilience claim currently untested in-repo) | S |
| 16 | Setup test pinning the `DisableService`-constant-200 behavior (whatever F4 decides) | XS |
| 17 | Decide `Recorder(nil)` panic vs `(recorder, error)` — unify sibling constructor failure modes | S |
| 18 | Document/carry per-service durations for injector checks (currently `Duration: 0` — samber/do reports outcomes only) | S |
| 19 | Wire-format map doc: one table mapping all 7 health surfaces' response schemas + status vocabularies (the self-health 4.0 gap) | M |
| 20 | Map `App.HealthHandler`'s `ok`/503 vocabulary onto the readiness vocabulary or document why it differs | XS |

**P3 — gates/tooling**
| # | Thing | Effort |
| - | ----- | ------ |
| 21 | Conflict-marker lint over tracked files (baseline-file class fix) | S |
| 22 | Pair every new `docs/status/` report with the archive ritual mechanically (session checklist) | XS |
| 23 | Wire doanalyzerv2 into check-modules/CI once branching-flow ships an installable binary (owner-gated) | M |
| 24 | Decide `check-upstream-treadmill` `--strict-lag` policy for CI (currently advisory; dashboard LAG 1 sailed through round 44) | XS |
| 25 | Corpus gate: print per-directory breakdown by default when it fails (DX) | XS |

**P4 — upstream / ecosystem**
| # | Thing | Effort |
| - | ----- | ------ |
| 26 | Verify-then-file the go-health RegisterRoutes method-less README caveat upstream (round-1 F11 note candidate; verify-before-filing applies) | M |
| 27 | Confirm go-health-dashboard v0.12.2 changelog before bumping (no blind bumps) | XS |
| 28 | Watch samber/do for v2.1.x (the InvokeAs scan contract may get fixed upstream — then gotcha 37 needs an update) | ongoing |

**P5 — docs hygiene (routine)**
| # | Thing | Effort |
| - | ----- | ------ |
| 29 | Archive round-44 status reports when this round's cycle closes (tail budget stays ≤3) | XS |
| 30 | Annotate the round-44 close-out self-review with the two pre-existing reds it missed (corpus conflict, tail 4>3) — ANNOTATE mode, non-destructive | XS |
| 31 | Re-run `check-docs-links`/`freshness` after items 6-11 land (they touch 4 docs) | XS |
| 32 | Keep `e2e/server` static `/health` marked deliberate (comment cross-ref exists; nothing to do — listed so it isn't "fixed" by mistake) | — |
| 33 | Verify round-2 report's remaining roadmap assumptions after T12 decision (the report's §10 item 1 cites exact files) | XS |
| 34 | Consider a CHANGELOG entry template for wire-visible error-code changes (T12 will need it) | XS |
| 35 | Audit whether any other single-integer gate files exist with the same corruption risk (tail baseline, coverage thresholds TSV are structured — verify) | XS |
| 36 | Session-start guard idea: `git grep '<<<<<<<' -- docs/ scripts/` one-liner in the runbook | XS |
| 37 | Move the round-1 InvokeAs repro snippet into a runnable `scripts/selftests/` fixture (currently prose in a report) | S |
| 38 | Consider exporting a root helper so `health.checkError`'s transient-vs-infrastructure mapping and readiness stay type-linked if states change | S |
| 39 | Health README: document that `WithHealthRecorder` in `NewProbe` opts are refused at Start (already there) — verify wording survives the go-health v0.9 wave | XS |
| 40 | Add the round-2 verdict (9/10) to the research README series table if the series becomes load-bearing | XS |
| 41 | Audit integration_test's "PUBLISHED behavior only" comment after the next health train — the duration chain assertions may graduate | XS |
| 42 | Check whether `dashboardui`'s five-state vocabulary deserves a row in the eventual wire-format map (item 19) | XS |
| 43 | bench-spike: confirm no bench paths touched this session (none were — skip unless the demo changes) | — |
| 44 | After T12 lands: grep docs for the old code strings and update (alerting runbooks) | S |
| 45 | After the health train: re-run integration_test GOWORK=off locally once to confirm the published-tag resolution | S |
| 46 | Consider `gohealth.WithEvaluationHook` + flightrecorder wiring as an example (README documents it; no runnable proof) | M |
| 47 | Decide whether `do.Package` grouping belongs in the demo once a second provider cluster appears (YAGNI today) | — |
| 48 | Keep the demo's `httputil.NewServer` usage aligned when the SSE-timeout story changes upstream | — |
| 49 | Re-verify gotcha 44 erraudit posture still current if the health module gains non-test code | XS |
| 50 | Close the loop on items 3-5 decisions by recording them in `docs/decisions/` (the open-decisions file pattern) | XS |

*(Items marked "—" are watch-only; they exist so nobody "fixes" deliberate state.)*

## g) QUESTIONS I CANNOT FIGURE OUT MYSELF

1. **T12 compatibility envelope:** when we unify projection-health error codes, which *consuming* systems (your alerting rules, dashboards, runbooks, downstream repos) match on the current `cqrshtmx.health.*` / `projection.*` strings — i.e., how breaking can the next minor train actually be, and which namespace do you want to be the survivor?
2. **`health.RecorderChain` fate:** migrate all docs to `gohealth.ChainRecorders` and keep the bridge alias deprecated (my recommendation — one canonical name, upstream-aligned), or undeprecate the alias as the bridge's family-readable composition name (bridge ergonomics, two names forever)?
3. **T13 drain posture:** is `RunWithAppkit`-as-the-production-drain-path the *permanent* answer (close T13 as won't-fix), or do you want `Bundle.MarkDraining()` so plain `Run`/`RunHandler` also flip readiness 503 before shutdown?

**AWAITING INSTRUCTIONS.**
