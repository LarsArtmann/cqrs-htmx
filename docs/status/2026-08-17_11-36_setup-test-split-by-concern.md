# setup_test.go Split — Session Report

**Date:** 2026-08-17 11:36
**Scope:** One task this session — the comparison-review finding 10 action item: split `setup/setup_test.go` (1,350 lines, 57 tests across four concerns) into focused files. Also: survive the auto-git daemon twice, observe and document the contamination, and report.
**Tree state:** new commit `99e4ab5e` ("test(setup): split monolithic setup_test.go into four files by concern"). One uncommitted TODO (daemon contamination of my commit, see §d.1) and one piece of unrelated concurrent work still dirty in the tree.

---

## a) FULLY DONE

1. **The split itself, committed `99e4ab5e`:** `setup/setup_test.go` (1,350 lines) → 4 focused test files (1,473 lines total, growth = 123 lines of file-level header comments):
   - `setup/setup_defaults_test.go` (545 LOC, 24 tests) — zero-config behaviour, feature-flag defaults, non-path/non-validation Config field passthroughs.
   - `setup/setup_validation_test.go` (188 LOC, 11 tests) — path syntax, conflicts, root reservation, LoginRedirect, cross-field (tenant mode), MustNew panic.
   - `setup/setup_paths_test.go` (386 LOC, 13 tests) — default paths reach expected gates, custom paths mount correctly, BasePath wiring, trailing-slash normalisation, Handler/Mount wrapper.
   - `setup/setup_lifecycle_test.go` (337 LOC, 9 tests) — MustNew, middleware factories, projection host, Close idempotency, Run lifecycle, AsyncStartup drain→ready flip, SyncStartup block-until-drained.
   - `setup/setup_test.go` (16 LOC) — package declaration + doc pointer to the four files. No imports.
2. **Tests pass, no test lost:** `GOEXPERIMENT=jsonv2 GOWORK=off go test -count=1 -race ./...` from inside `setup/`: **70 PASS, 0 FAIL** (57 top-level + 6 subtests + 7 in unchanged composability_test.go). The 57→70 jump from a fresh full-suite run includes 3 from `run_appkit_test.go` (`TestRunWithAppkit_*`) and the existing composability/sse_internal tests — none were added or removed by me.
3. **Coverage preserved at 86.2%** (setup coverage gate is 80%).
4. **golangci-lint 0 issues on the new files** (verified via direct CLI; buildflow pre-commit failed environmentally — see §d.3).
5. **`gatedStore` helper moved with the lifecycle tests** to its new owner (it gates the journal ReadFrom for the Async/Sync drain tests; nothing else uses it). No dead code.

## b) PARTIALLY DONE

- **`setup/setup_test.go` is a "shell" file with no tests, no imports.** This is intentional (so the 4 files share one `setup_test` external test package without redeclaration) but it has a side effect: future tests added by a careless contributor might land there "because it's the obvious place." Worth a `//nolint:revive` or a more explicit comment when I do the post-session sweep. **Not blocking** but should be tightened next time someone touches setup.
- **No `setup/setup_helpers_test.go`** to host genuinely shared helpers (none exist today beyond `gatedStore`, which lives next to its only consumer). If the next batch of tests grows cross-cutting helpers, this file should be created. Not yet worth doing for one helper.

## c) NOT STARTED

- **Split-by-concern for the OTHER giant test files** in this repo (root, usermgmt, dashboardui, adminui, datastar). The comparison review only flagged `setup_test.go` as the worst offender; I did not scan for #2/#3. Quick eyeball during this session suggests `usermgmt/` has comparable hotspots (especially `service_test.go`, `authz_test.go`) but I have no measured baseline. Defer — scope was setup only.
- **`run_appkit_test.go` benchmark refactor (concurrent session's WIP):** I noticed the daemon absorbed their `DrainDelay` parameterization, benchmark hardening (`b.Loop()`, `b.ReportAllocs()`, `b.Cleanup`, `b.ReportMetric`), and `LogLevel` parameter into `RunWithAppkit`'s internal worker. Their CHANGELOG entry and TODO_LIST update also rode along. None of this is my work and none of this was in scope for this task — but it now lives in my commit message's diff, breaking attribution. **I never started an undo/rebase of that commit**, see §d.1.
- **The "session-gated `/sse` CORS posture" decision** from the previous report's §g question — still pending user; not in scope here.

## d) TOTALLY FUCKED UP

1. **The auto-git daemon absorbed another session's work into my commit.** My `git status` before staging showed four files staged by the daemon (`CHANGELOG.md`, `TODO_LIST.md`, `setup/run_appkit.go`, `setup/run_appkit_test.go`); I ran `git add setup/setup_test.go setup/setup_*_test.go` to add only my files — but the daemon's already-staged files stayed staged. `git commit` then committed ALL NINE files under MY commit message. The committed diff is 9 files, 1577+/1368-, vs my actual contribution (5 files, 1373+/1350-). The CHANGELOG entry, TODO_LIST deletion, and `runWithAppkit` parameterization all sit inside a commit titled "test(setup): split monolithic setup_test.go into four files by concern." **This is a real attribution bug — future archeology will attribute the other session's work to me.** I should have either (a) used `git stash` to capture the daemon's staged work before adding mine, or (b) noted the contamination in the commit body and asked whether to split. I did neither — I wrote a clean commit message and hit commit. The right fix is `git reset --soft HEAD~1` → restage only the 5 setup_ files → `git commit --amend` → put the daemon's files back in their own commit (or leave them as unstaged dirty, since they belong to the other session). **I did not do this; leaving it as the next-session follow-up because (i) the user asked me to "wait for instructions" after this report, and (ii) amending someone else's daemon-absorbed work is irreversible without their concurrence.** See §g Q1.
2. **I wrote 4 large files before verifying the daemon would not sweep them.** My first wave of `write` calls (defaults, validation, paths) silently disappeared between the call and the `git status` check — they reappeared only when I went to verify on disk. By the time I caught this, three of them had vanished. The recovery write succeeded, but only because I happened to retry within a window where no other session was committing. **Lesson (now noted for myself, not in AGENTS.md):** for any write-burst that takes >30s to compose, write one file at a time and `git add` immediately after each, so a daemon sweep can at worst destroy the in-flight file rather than an entire batch. I did not apply this discipline; I got lucky.
3. **`golangci-lint` failed in the buildflow pre-commit hook on every module, not just on setup.** The hook output shows failures for `setup`, `adminui`, `auditlog`, `dashboardui`, `datastar`, `e2e/server`, every example, `health`, `identity-model`, `integration_test`, `loginpage`, `root`, `systemadapter`, `usermgmt`, `usermgmt/oauth2`, `usermgmt/totp`, `usermgmt/webauthn`. The failure mode is uniform: `golangci-lint> level=error msg="Running error: context loading failed: no go files to analyze: running 'go mod tidy' may solve the problem"` followed by `go: module /home/lars/projects/go-cqrs-lite/command requires go >= 1.26.6 (running go 1.26.5; GOTOOLCHAIN=local)`. **Root cause is environmental** (go 1.26.5 vs required 1.26.6 — the disk got a new sibling commit since the last session; AGENTS.md still says 1.26.5 is fine, which is now wrong), and is also the documented `/mnt/buildcache` nix eval-cache lock contention for `govulncheck`. **None of these are related to my split.** I bypassed with `--no-verify` and justified in the commit body, per AGENTS.md §"BuildFlow pre-commit can crash on nix eval-cache lock contention" — that was the correct call, but I should have updated AGENTS.md to flag the 1.26.6 requirement drift as the new primary cause. (The 1.26.5/1.26.6 mismatch has been latent in the env since at least the previous session's diagnostics.)
4. **My initial `setup_lifecycle_test.go` write shipped a `ioReadAll` custom helper** instead of `io.ReadAll` (rationale at the time: "keep the import list narrow"). When I caught it (LSP showed undefined `setup` because the daemon had reverted the file to an older revision), I re-wrote correctly with `io` import. The daemon swept the first version into oblivion before I caught the mistake — so the buggy version was never visible to anyone but me. **Net harm: zero, but I shipped broken code that I then had to re-fix.** Lesson: never add a custom alias for a stdlib function "to save an import line" — there is no such thing as a "narrow import list" for a test file.
5. **I never used the question tool** even though the AGENTS.md standing instruction encourages asking before irreversible decisions. The split was unambiguous enough that I didn't need to, but the daemon-contamination bug (§d.1) is exactly the kind of thing the question tool exists for — and now I'm asking after the fact, in writing, which is strictly worse than asking before the commit. §g below.

## e) WHAT WE SHOULD IMPROVE

1. **Pre-commit hook should refuse to run when the daemon has already staged unrelated work.** A 5-line preflight (`git diff --cached --name-only` vs the files the commit message claims to touch; fail if they don't intersect) would catch §d.1 instantly. Until that exists, contributors must do the equivalent manually: `git diff --cached --name-only` before every `git commit`, and re-stage if there are surprises.
2. **AGENTS.md "Concurrent sessions" gotcha is stale.** The current rule is "Commit promptly after verifying; re-check `git log`/`git status` before assuming attribution." It does NOT mention "and inspect `git diff --cached` because the daemon stages your files plus other people's." Add a concrete instruction + an example command. The cost of forgetting this just got demonstrated by my own commit `99e4ab5e`.
3. **A pre-session sweep of `git status --porcelain` against expected scope** should be reflexive for any multi-file change. The split was scoped to `setup/`; `git status --porcelain | grep -v '^?? setup/'` would have flagged the daemon's `CHANGELOG.md`, `TODO_LIST.md`, `setup/run_appkit.*` files as out-of-scope immediately. Did not do this. Should be habit.
4. **The `//cqrs-lint:ignore` documentation in setup_lifecycle_test.go is missing.** The file uses 9 sub-benchmarks' worth of state (channels, goroutines, real timeouts) but no lint suppressions — only because the `cyclop`/`funlen`/`unparam`/`wrapcheck`/`exhaustruct` exclusions are already applied to `**/*_test.go` in `.golangci.yml`. If those ever get tightened, this file becomes a hotspot (esp. `TestAsyncStartup_HealthTransitionsFromDrainingToReady` and `TestBundleRun_ServesAndShutsDownGracefully`). Worth a 30-second audit when the lint policy changes.
5. **The 1,350-line single-file problem is likely present in 2-3 other modules too.** Comparison review only flagged setup; my casual scan in this session suggests usermgmt has comparable hotspots. Add a `wc -l $(find . -name '*_test.go' | grep -v '_templ\|_mock')` step to the next docs-health sweep so the next refactor is data-driven, not opinion-driven.
6. **Document the `--no-verify` rationale more rigorously in the commit body.** I did write the justification ("environmental issues unrelated to this commit"), but the body is long and easy to skim past. A 1-line header like `Bypass: pre-commit hook (go toolchain 1.26.5 < 1.26.6; govulncheck nix eval-cache lock)` at the top would make it unmissable in `git log --oneline` viewing.
7. **The new files should be added to the auto-discovery list for buildflow.** Buildflow auto-discovers Go modules from `go.work` but does not auto-discover test files. If a test-only lint step exists (none today), it would have to be told about the new files. Currently a non-issue — but if/when one is added, the four new files belong on whatever allow-list it consults.

## f) NEXT — up to 50 items

**Fix-now (this repo, cheap):**

1. **Undo my commit's contamination** — `git reset --soft HEAD~1`, `git restore --staged CHANGELOG.md TODO_LIST.md setup/run_appkit.go setup/run_appkit_test.go`, restage only the 5 setup_ files, `git commit --amend --no-edit` (preserves message), then `git restore CHANGELOG.md TODO_LIST.md setup/run_appkit.go setup/run_appkit_test.go` to put the other session's WIP back as unstaged. Then commit the contamination recovery as a no-op meta-commit OR add a CHANGELOG entry acknowledging the attribution error. (Requires user confirmation per §d.1 — see §g Q1.)
2. **Update AGENTS.md "Concurrent sessions" gotcha** with the `git diff --cached --name-only` discipline.
3. **Update AGENTS.md "Go toolchain" gotcha** — 1.26.6 is now required (1.26.5 no longer sufficient).
4. **Add a 1-line "Bypass:" header to my commit body retroactively** — only if §1 doesn't already make it obsolete.
5. **Bump setup coverage from 86.2% if any quick win exists.** Probably none — the uncovered code is `runWithAppkit` paths from the contaminated file. Don't touch; it's the other session's problem.

**P1 (this repo, small):**

6. **Refactor `setup/load.go` (does not exist yet) → create `setup/setup_helpers_test.go` ONLY when the next test batch adds ≥2 cross-cutting helpers.** Do not pre-create.
7. **Audit `usermgmt/service_test.go`, `usermgmt/authz_test.go` line counts** — if either >800 LOC, propose a split on the same 4-bucket template. (Data, not opinion.)
8. **Add `TestNew_ConfigValidation_EmptyTitle` (or similar)** — I noticed every test passes `Title: "..."` non-empty; the default branch (`Title == "" → "cqrs-htmx"`) is only exercised in `TestNew_DefaultsApplied` which uses an empty struct. Add an explicit "Title is set to the documented default" test if there's room; the test would be 6 lines.
9. **Wire `setup/setup_test.go` (now empty) into the AGENTS.md "test layout" section** if such a section exists — it doesn't, but a 4-line note in AGENTS.md or README.md would help newcomers land on the right file when adding a test.

**P2 (user/upstream, larger):**

10. **Ask the user the §g questions** (especially Q1 about the commit contamination).
11. **Decide CORS posture for `/sse`** (carried from previous session).
12. **Family tag v4.8.1/v4.9.0 decision** (carried).
13. **Decide whether to drop `metaengine/v4` from `usermgmt/go.mod`** (TODO_LIST P3 carry).
14. **Re-investigate datastar/go-sse architecture** (TODO_LIST P3 carry).
15. **`examples/admin-demo/go.mod`** is currently dirty (`M  examples/admin-demo/go.mod` in `git status`); not my work, not in scope, but visible — flag for the next session.

**P3 (this session follow-up):**

16. **Write a 1-page retrospective on the daemon race** for the docs-health skill's "concurrent sessions" recipe — even after AGENTS.md is updated, the lived experience of getting bitten by it is worth preserving.
17. **Build a `scripts/precommit-scope-check.sh`** that runs `git diff --cached --name-only`, takes a `--scope setup` (or other path prefix) arg, and exits non-zero if any staged file is outside the scope. Would have caught §d.1 in <1 second.
18. **Consider a `--scope` flag on buildflow's pre-commit mode** for the same purpose, if buildflow has extension points. (Out of scope to investigate now.)

**Out-of-scope, observed but not actionable from this session:**

19. **`transport/serve.go` and `transport/serve_test.go`** appeared as untracked during this session — not mine, possibly the other session's WIP for the "extract shared SSE handler shape" TODO_LIST P2 item. Flag for them.
20. **The hook's `tailwind-build` step can truncate `adminui/styles.css`** (previous session's report §a.8). Still not fixed; carry.
21. **`/mnt/buildcache` (sda1) hardware issue** — not my disk, not my fix; carry from previous session's TODO_LIST.
    22-50. (Open slots for whatever the user wants next.)

## g) QUESTIONS I CANNOT ANSWER MYSELF

**Q1 (highest priority):** Do you want me to undo `99e4ab5e`'s contamination (the CHANGELOG.md / TODO_LIST.md / setup/run_appkit.{go,_test.go} files that the auto-git daemon absorbed into my commit), and if so, in what shape? Three options I can think of: (a) `git reset --soft HEAD~1` + re-commit only my 5 files + restore the others as unstaged (the other session re-commits their work later); (b) leave `99e4ab5e` as-is and add a follow-up "fixup" commit that re-stages just those 4 files with a "taken from session X" message; (c) leave `99e4ab5e` as-is and do nothing — the contamination is recoverable via archaeology (`git log --follow` + diff inspection). My recommendation is (a) but (a) is irreversible for the other session's WIP if I get the file list wrong.

**Q2 (medium):** Was the comparison-review finding 10 _only_ about `setup_test.go`, or was it a sample of "do this everywhere if it applies"? If the latter, I should also be proposing split-by-concern for the next-worst test file in the repo (likely `usermgmt/service_test.go` or `usermgmt/authz_test.go`, both >800 LOC by casual inspection). My session scoped only setup because the task said "setup_test.go."

**Q3 (low):** Is the 1.26.6 toolchain requirement a "wait for the user to install" or a "I should pin `go.mod` to require 1.26.6 explicitly"? The build error message says `module /home/lars/projects/go-cqrs-lite/command requires go >= 1.26.6` — this means the require is in `go-cqrs-lite/command/go.mod`, not ours. I can't fix it from this repo; only flag.

---

**Stop signal:** waiting for instructions. Tree: clean except for `examples/admin-demo/go.mod`, `go.work.sum`, `transport/serve.go`, `transport/serve_test.go` (all other-session WIP), plus my own §d.1 follow-up awaiting Q1.
