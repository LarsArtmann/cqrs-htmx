# Status Report — Train-Lag Unblock, Atomic Wave Consumption, CI-Repair Session

**Date:** 2026-10-12 05:23 CEST
**Session scope:** unblock the interrupted `git sync` (strict release-train lag), consume the go-cqrs-lite reentrancy wave, repair the CI reds it surfaced.
**End state:** master = origin/master at `58a6a7ce` · CI **green, all 17 jobs** (first green since `48b118e0`; the four runs before this session were red) · strict release-train gate green · full local battery green.

---

## 0. The three framing questions

### What did you forget?

1. **The pre-push CI-parity tail.** I mirrored the release-train + version-drift + lint + test gates locally but did **not** run `nix run .#coverage-gate`, `check-large-files --all`, or `.#test-all` before push #1. CI's two failures (root coverage 88.2% < 90%; the 3.7MB `cqrs-htmx-upgrade` binary) were both locally predictable — the repo even ships `.#train-preflight` for exactly this and I did not use it.
2. **That committed artifacts hide from `git status`.** Both stray binaries (sync-demo's and the root codemod binary) were *already tracked* — a clean tree is not a clean tree. A one-shot `git ls-files` size/binary scan at session start would have surfaced both before any push.
3. **bump-dep's operational limits** — I ran three chained sweeps before reading the failure-handling side of the script (green-gated `--commit` can never fire mid-wave; dirty-tree guard aborts chained runs). Cost: one wasted cycle and a daemon-captured partial sweep.
4. **git-town state hygiene** — the interrupted sync left pending state ("run git town continue") that I only cleared after push #2. Harmless here, but it should have been step one.

### What could you have done better?

1. **Commit at phase boundaries (gotcha 4)** — the daemon beat me to every commit (6+ heuristic chunks: `7f5a8740`, `0b66e6ac`, `11707fd9`, `7ce19a94`, `f10d3e20`, `dd0ec5f7`, `a08f449f`, `58a6a7ce`). The wave's history is readable only through the lens of this report. I chose not to rebase under a live daemon — defensible, but the debt is real.
2. **Script correctness first time.** My wave-finish sweep script had two mvdan/sh rc traps (grep `&&` short-circuit counted as failure; `cmd | tee; $?` reports tee's rc) — both are *documented* in AGENTS.md gotcha 3, and both produced misleading FAIL/ABSENCE output I then had to re-verify by hand.
3. **Verify each wave member's diff, not just event's.** I deep-read only event v4.13.1→v4.14.0; projectionhost/watermill/metaengine got stat-level review plus the test battery. The battery is the mechanical catch, but the CHANGELOG's "deltas are additive" claim rests on one module's inspection.
4. **Fixer damage watch.** The pre-commit hook's BuildFlow run mutated `e2e/server/main.go` (wsl-style blank-line inserts) during a commit attempt at 50-failed-steps scale. I caught and reverted it, but I should have snapshotted `git diff --stat` before every hook commit run.

### What could I still improve?

- Build margin, not just threshold: root coverage is 90.6% against a 90 floor — 13 statements of headroom (see f-9/f-29/f-41…45).
- Turn this session's one-off scripts into repo tools (f-5, f-19).
- Push the mechanical lessons (gotcha 38 landed; the runbook amendment and CI-parity preflight extension have not).

---

## a) FULLY DONE

| # | Work | Evidence |
|---|------|----------|
| 1 | Release-train lag resolved: event/v4 v4.13.1→v4.14.0 (24 modules), projectionhost/v4 v4.5.3→v4.6.0 (14), watermill/v4 v4.6.5→v4.7.0 (12), metaengine/v4 v4.17.0→v4.18.0 (4); testdata fixtures untouched | absence assertion clean; strict train gate rc=0 post-push |
| 2 | Atomic-wave coupling diagnosed and handled (event v4.14.0 ⟷ dedup v4.3.0 generic `Ring[K]` ⟷ projectionhost v4.6.0 ⟷ watermill v4.7.0 ⟷ metaengine v4.18.0) | hermetic builds green across all 29 modules post-wave |
| 3 | Test-only API drift fixed: `usermgmt/es_dedup_benchmark_test.go:29` → `dedup.NewRing[string](...)` | usermgmt build+vet rc=0 |
| 4 | Union-graph go.sum residue repaired in 5 modules (usermgmt/oauth2, usermgmt/totp, usermgmt/webauthn, examples/catalog-demo, auditlog) via hermetic `go mod tidy -diff` sweep | second tidy-diff pass clean, candidate count printed (29) |
| 5 | Full local verification battery green: `.#test` (hermetic, race), `.#lint`, `.#check-modules`, strict `check-release-train --refresh-cache` | all rc=0 |
| 6 | **CI green on pushed master `58a6a7ce`: all 17 jobs ✓** (security, mod-tidy, checks, module-architecture, test, build, 10× lint) — first green since `48b118e0` | `gh run view 38198663907` |
| 7 | CI red #1 fixed: 3.7MB `cqrs-htmx-upgrade` binary untracked + `/cqrs-htmx-upgrade` gitignored (daemon had captured it 2026-10-11, `22bd3663`) | `check-large-files --all` green locally; CI checks ✓ |
| 8 | CI red #2 fixed: root coverage 88.2% → **90.6%** via real codemod CLI tests (`cmd/cqrs-htmx-upgrade/cli_test.go`: file-walk, dry/apply engine, report modes, file-class lists; runCLI test with the established live-graph skip guard) | `go tool cover -func` total 90.6%; CI test ✓ |
| 9 | Second stray binary (examples/sync-demo) untracked + gitignored (plus missing system-demo entry) | tracked-file scan clean |
| 10 | Branching-flow ratchet integrity kept: baseline re-pinned 237→248 (+29/−20/~199 re-attributed) with honest triage-decisions amendment (provisional under existing reject classes, pending owner adjudication); gate green | `check-branching-flow` rc=0 |
| 11 | CHANGELOG receipt for the upstream wave (consumer-visible go.mod changes) | CHANGELOG Unreleased § Changed |
| 12 | AGENTS.md gotcha 38 written (atomic-wave lesson: bump-dep green-gate, daemon capture, vet-only test drift, classify-residuals-against-origin rule) | AGENTS.md |
| 13 | Pre-existing sync-demo root-tag-gap classified, not "fixed": `AddSyncOpenAPI` usage landed in `af1af8c8` (on origin/master), require v4.14.0 predates the symbol; CHANGELOG already pins it to "ride the next root tag" | merge-base proof; only `.#test-all` sees it (examples excluded from `.#test`) |
| 14 | git-town interrupted-sync state cleared (`git town continue` → "Everything up-to-date"); pre-push wait-tree-quiet observed before both pushes | push #1 `4d659bda`, push #2 `58a6a7ce` |
| 15 | Moving-target stop-rule checked: ls-remote confirms the four pinned versions are still the latest published go-cqrs-lite tags | event v4.14.0 / projectionhost v4.6.0 / watermill v4.7.0 / metaengine v4.18.0 |

## b) PARTIALLY DONE

| # | Work | What's missing |
|---|------|----------------|
| 1 | Coverage margin restored | 90.6% vs 90 floor = 13 statements of headroom; `fatalf`/`main` and `deriveMappings` error paths still 0%/22.7% |
| 2 | Train history readability | the wave landed as 6+ daemon heuristic commits; messages not amended (chose not to rebase under a live daemon) |
| 3 | Hook-failure documentation | this session's non-devShell failures (golangci-lint, govulncheck, license-check; 9→19→50 steps as disk filled) extend gotcha 8's list but the new failure modes (fixer damage on `e2e/server/main.go`, disk-pressure step-count explosion) are only in this report |
| 4 | Disk management | switched this session to /tmp caches when /mnt/buildcache hit 100%, but reclaimed nothing on the shared disk |
| 5 | Wave verification depth | event's diff deep-read; projectionhost/watermill/metaengine verified by battery + stat only |

## c) NOT STARTED

| # | Work | Note |
|---|------|------|
| 1 | The owner-approved **14-tag internal wave** (root v4.14.1, usermgmt v4.14.3, identity-model v4.12.3, totp/webauthn v4.12.1, oauth2 v4.13.0, datastar/auditlog v4.12.2, loginpage v4.12.3, dashboardui v4.13.3, adminui v4.12.4, systemadapter v4.12.4, setup v4.14.3, health v4.13.0) | CHANGELOG entries are prepared; zero tags exist locally or remotely; e2e 73/73 battery is its documented gate. Deliberately not executed unrequested — it pushes 14 tags |
| 2 | sync-demo require bump to the next root tag (closes the hermetic gap) | blocked on #1 |
| 3 | TODO_LIST/ROADMAP harvest of this report's section (f) | docs-health HARVEST, after owner triage |
| 4 | `docs/agents-notes.md` narrative for this session | dated war stories belong there per AGENTS.md convention |
| 5 | bench-spike re-verification post-wave | dedup Ring went generic; perf-class change possible (gate would catch, not run this session) |

## d) TOTALLY FUCKED UP

| # | Item | Damage |
|---|------|--------|
| 1 | **CI was red on master for ~a day before this session** — runs `4b4c12aa`, `4e666ed2`, `4d659bda` all failed (large-file guard + coverage); the blocking causes (binary 10-11, coverage drop from the 10-11 feature commits) sat on origin with nobody pushing or watching | fixed this session, but the pattern — red master behind a blocked local gate — is the exact "a push can never be greener than CI" failure the pre-push hook exists to prevent, and it still happened one level up |
| 2 | **The other session's dedup sweep left the workspace hermetically red mid-wave** — dedup v4.3.0 (generic Ring) with old projectionhost/metaengine tags fails hermetic build/vet in ~9 modules; workspace mode masked it completely | my push gate caught it (design working); but red trees should never sit between sessions |
| 3 | **Daemon-vs-hook churn during the train**: my in-process commits were blocked three times by the pre-commit hook (9/19/50 failed steps — non-devShell class, aggravated by disk-full), and each failure ended in a daemon heuristic capture; at 50-steps scale the hook's fixer also mutated `e2e/server/main.go` (reverted) | history readability debt + a live fixer-damage landmine |
| 4 | **/mnt/buildcache at 100%** — my heavy gates (test-all, coverage) contributed; go-cache-env warns but still exports the full disk | risk of corrupted caches / phantom go.sum failures for every concurrent session |

## e) WHAT WE SHOULD IMPROVE

1. **Pre-push CI parity must be one command.** `.#train-preflight` exists but doesn't include coverage-gate, `check-large-files --all`, or the examples set. Extend it so "mirror CI" is mechanical, not memory.
2. **Committed-artifact scanning.** A clean `git status` says nothing about tracked junk; CI's `--all` mode is the only thing that caught two binaries. Local default should match CI, or a session-start scan should run.
3. **Atomic wave support in bump-dep.** Green-gated per-sweep commits are wrong for coupled waves; a `--wave` mode (multi-pattern, cluster-wide green gate, one commit) removes both the daemon-capture and dirty-tree-guard failure paths this session hit.
4. **Daemon-commit amend ritual.** Gotcha 27 already recommends amending daemon messages for train readability; make it a named, safe-window step of the release checklist instead of an ad-hoc judgment call under a live daemon.
5. **go-cache-env fallback.** Low-disk currently warns and still exports the full-disk cache; auto-fallback to /tmp (or hard-fail) would have saved a failed verification round.
6. **Red-master watchdog.** CI red for a day on origin while local work continued — a session-start `gh run list` check (like the tree preflight) would surface "master is red, here's why" before work begins.
7. **LSP diagnostics remain unreliable post-edit** (three phantom wsl/modernize warnings persisted after golangci-lint reported 0 issues) — gotcha 14 holds; do not create work items from LSP output.

## f) TOP 50 THINGS TO GET DONE NEXT

*(brainstorm, not commitment — most beyond the top ~12 are ROADMAP fuel for docs-health HARVEST routing)*

| # | Item | Impact |
|---|------|--------|
| 1 | Cut the prepared 14-tag internal wave (owner-approved cadence, e2e 73/73 gate) | publishes the sync-OpenAPI/codemod/asset-fix surface |
| 2 | Bump sync-demo's root require to the new root tag (closes the last hermetic gap) | makes `.#test-all` fully green |
| 3 | Extend `.#train-preflight` with coverage-gate + `check-large-files --all` + `.#test-all` (full CI parity) | prevents every class of post-push CI surprise |
| 4 | Session-start CI status check (gh run list) alongside tree preflight | surfaces red-master before work begins |
| 5 | bump-dep `--wave` mode: multi-module atomic sweep, cluster green gate, single commit | kills the mid-wave daemon-capture class |
| 6 | Local `check-large-files` default parity with CI `--all` (or `.#check-binaries` app) | catches tracked binaries locally |
| 7 | Reclaim /mnt/buildcache (gotcha 12: rust/ 155G + sccache/ 20G dominate; Go caches ~12G) | un-blocks shared-disk sessions |
| 8 | go-cache-env: auto-fallback to /tmp on low disk (currently warn-only) | no more failed rounds on a full disk |
| 9 | Raise root coverage margin: `main`/`fatalf` via subprocess-exec tests | ~8 stmts + real CLI-path confidence |
| 10 | Cover `deriveMappings`/`moduleDir` error paths | codemod robustness + margin |
| 11 | Cover 0% Bucket-B utilities: sse_event.go (5 funcs), redirect.go (2), errors/structured_error `Error()` (3), `ReplayEvents`, `EventsAfter`, `GenerateUserID` | ~40 stmts; cheap table tests |
| 12 | Add the atomic-wave procedure to `docs/runbooks/dependency-train-bump.md` (manual-finish loop, green-gate limits, daemon interplay) | next wave is faster |
| 13 | Mechanize gotcha 27e: refresh tag cache inside the pre-push hook (flagged "pending evaluation" since 10-08) | kills the cross-repo TTL race class |
| 14 | Amend/reword this train's daemon heuristic commits in a quiescent window (squash `58a6a7ce`+`a08f449f`+`dd0ec5f7`+`f10d3e20` story into readable messages) | train-readable history |
| 15 | Generalize the wave-finisher script into `scripts/tools/wave-finish.sh` (correct rc bookkeeping baked in) | reusable for the next atomic wave |
| 16 | Single-source CI coverage thresholds (ci.yml hardcodes 90/74; flake coverage-gate hardcodes its own lists — split brain) | one number, two readers today |
| 17 | Decide cmd/ exclusion policy for root coverage (CLI entry shells vs library code) | stops codemod growth from eating library margin |
| 18 | Pre-commit guard: refuse staging files >1MB without explicit allowlist | mechanical catch for daemon binary captures |
| 19 | Sweep the repo for other tracked build artifacts (`git ls-files` size/type audit) | two found this session; are there more? |
| 20 | Wire the examples/e2e hermetic set into CI's test job (or a dedicated job) — the sync-demo gap survived only because CI doesn't build examples | closes gotcha 30's test variant permanently |
| 21 | Extend `check-train-consumers-hermetic` to cover examples/e2e dirs | same class, pre-publish |
| 22 | File the BuildFlow fixer-damage upstream issue (wsl-style inserts into treefmt-clean files during multi-failure runs; verify at source first per verify-before-filing) | the 50-step run mutated e2e/server/main.go |
| 23 | Document the non-devShell hook-failure escalation (9→19→50 steps under disk pressure) in gotcha 8 | triage speed next time |
| 24 | Write the `docs/agents-notes.md` narrative for this session | dated history convention |
| 25 | HARVEST this report's section (f) into TODO_LIST/ROADMAP (docs-health) | the report must not entomb its own actions |
| 26 | Re-pin bench baseline post-wave (dedup Ring generic change may shift the dedup benches; `.#bench-spike` gate) | perf-regression proof |
| 27 | Audit remaining go.work replaces for load-bearing-ness post-wave | dead replaces mislead hermetic reasoning |
| 28 | Verify the "Upstream: consumed…" CHANGELOG entry gets folded into the 14-tag wave's version section at cut time | receipts must ride versions, not Unreleased forever |
| 29 | codemod: cover printReport/manual-only path variants already written but re-check under `-v` flag combinations | report is the product |
| 30 | Add a permanent absence checker for non-instantiated `dedup.NewRing(` (the vet catch is incidental today) | mechanical guard for the generic-migration class |
| 31 | shellcheck custom rule (or review checklist) for the `cmd \| tee; $?` rc-eating pattern in scripts/ | gotcha 3 keeps biting hand-rolled scripts |
| 32 | Audit module-local .gitignore consistency (system-demo has one; others rely on root list) | one policy, one place |
| 33 | Run the e2e sync battery (73/73) locally once post-wave as an explicit re-confirmation | it gates the pending wave |
| 34 | Evaluate renovate/dependabot-style detection for upstream go-cqrs-lite train drops | train lag gets caught in hours, not at push |
| 35 | Improve `nix run .#test` vs `.#test-all` naming/docs (the exclusion of e2e/examples is a recurring trap) | gotcha 2-adjacent clarity |
| 36 | Check whether CI's "Dependency Graph" go_modules jobs should be scoped (they fan out per module and drown the run list) | signal/noise in gh run list |
| 37 | gopls/golangci_ls: investigate persistent phantom warnings after fixes (LSP cache?) | restore trust in diagnostics |
| 38 | Consider `go test -shuffle=on` for the root module suite (order coupling check) | flake resistance |
| 39 | Add `check-large-files` fixture case for "tracked binary added by daemon" | self-test coverage for the class |
| 40 | cli_test.go: make `writeFile` MkdirAll-aware (fixture friction this session hit) | small DX win |
| 41 | Document git-town interplay: interrupted sync + manual push → `git town continue` is the finisher (agents-notes) | next interrupted sync is unscary |
| 42 | Sweep status reports: annotate superseded ones per docs-health ANNOTATE mode (this report supersedes none, but the backlog check is due) | docs hygiene |
| 43 | Verify `GOEXPERIMENT=jsonv2` propagation into the codemod's `go list` subprocess (moduleDir inherits env — confirm under exotic shells) | codemod portability |
| 44 | Add release-checklist step: "ls-remote moving-target re-check immediately before tag" (was manual this session) | stop-rule mechanization |
| 45 | Normalize this status report's tables (`.#normalize-status-rows`) | gate 2 hygiene |
| 46 | Explore parallelizing `forEachGoModule` test loop (test-all wall-clock) | faster feedback |
| 47 | Add codemod fixture for R9/R10 end-to-end through `run()` (currently planFile-level) | engine-level rule coverage |
| 48 | Track dedup generic-migration upstream note (which go-cqrs-lite train made Ring generic) in the CHANGELOG receipt | archaeology for consumers |
| 49 | Consider a `.#ci-parity` flake app that maps 1:1 to the CI job matrix | the "mirror CI" discipline, mechanized |
| 50 | Revisit whether the pre-push hook should also run `check-large-files --all` (CI parity at the last gate) | binaries die at push, not in CI |

## g) QUESTIONS I CANNOT FIGURE OUT MYSELF

1. **Cut the 14-tag wave now?** The CHANGELOG, cadence, and gate (e2e 73/73) are prepared and owner-approved, but executing it means pushing 14 tags from this session. Do you want me to run that train now — or is another session/pipeline the designated train driver?
2. **Coverage policy for the CLI package:** is the 90% root floor meant to include `cmd/cqrs-htmx-upgrade` (I raised real coverage to comply), or should CLI entry shells be excluded from the root count so library margin doesn't subsidize codemod growth?
3. **Daemon-commit history:** do you want this train's ~6 heuristic commits reworded/squashed in a quiescent window for train readability, or is heuristic attribution acceptable and history should never be rewritten once pushed?

---

*Prepared per the status-report skill; format is `.md` at the user's explicit path demand (HTML is the skill canonical — flagged per spec). Section (f) is HARVEST-ready. Waiting for instructions.*
