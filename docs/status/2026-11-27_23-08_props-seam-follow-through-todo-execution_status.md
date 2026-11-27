# Status — the props-seam follow-through and the TODO burn-down, interrupted by the usage limit

**Date:** 2026-11-27 23:08 CET · **Branch:** `master` · **HEAD:** `27a83188`
(`git rev-parse --short HEAD`, run at report time; the auto-commit daemon commits
continuously, so HEAD moves after this line — the five files still uncommitted at
that moment, among them the branching-flow baseline, were committed by the daemon
and the gate re-read green before this report was saved)

**Scope line.** VERIFIED at report time, by command, at or near the pinned HEAD:
the root module's whole suite with the race detector
(`GOEXPERIMENT=jsonv2 go test ./... -count=1 -race` → every package `ok` after one
re-run, see d6) and with coverage (`go test ./... -coverprofile` → 91.9% merged
profile, bar 90); the seam specs (`--ginkgo.focus='QuerySet|Views|OOB|LoadScope'` →
78 of 581 specs, 0 failed); `golangci-lint run ./` on root (0 issues); the WHOLE
hermetic lint sweep (`bash scripts/checks/check-lint-sweep.sh` → 17 of 17 gated
modules clean under `GOWORK=off`, including the three modules this session
touched); the usermgmt module under `-race` (ok, 24.5 s) and its consumers
(`go test ./systemadapter/... ./adminui/... ./gfl-usermgmt/... ./identity-model/...
./legalparty-model/...` → all `ok`); the adminui tenants tests including a sabotage
run that proves the new order test fails on the old logic; `check-module-layers`,
`check-reexport-policy --strict` (48 files / 444 declarations), `check-cqrs-lint`
(all modules strict, zero findings), `check-skill-claims` (253 symbols),
`check-skill-snippets` (51 fences), `check-readme-splitbrain`,
`check-command-bijection` (27/27), `check-go-mod-tidy` (32 modules),
`check-module-isolation` (every module builds and vets standalone),
`check-release-train --strict-lag 0` (1,062 requires, 0 unpublished, 0 train lag),
`check-version-drift --strict` (no drift, all requires resolve to published tags),
`check-adminui-surface`, `check-permission-surface`, `check-conflict-markers`
(2,194 files), `check-docs-links` (2,150 links), `check-todo-hygiene`,
`check-large-files --all`, `check-root-files --strict`, `check-root-size` and
`check-root-clusters` (after the re-pins named in a20); and the five docs-gates
verbs run singly for their numbers. CI was read through `gh run list` / `gh run
view` for the two commits named in d4.

NOT VERIFIED, and why: the full 127-test chromium battery (not run — the machine
sat at load average 200 for most of the session; only the default load was 5–18 at
report time); `nix run .#check-modules` as a composite (its legs were run
individually, never as one command at the final HEAD); `.#coverage-gate`,
`.#bench-spike`, `.#check-erraudit-strict` (not run); the `.#lint` flake app (the
direct `golangci-lint` sweep was run instead); the benchmarks' nanoseconds (one
smoke run at `-benchtime=300ms` under load — the allocation counts are exact, the
nanoseconds are indicative); the docs-gates battery as a whole after this report
was written (four verbs were re-run singly: audit-all 438 unharvested against 263,
cadence 30 live reports, annotate CLEAN after its three findings were fixed,
freshness FRESH with this report in place); the usermgmt
email change's coverage BEFORE the change (the subagent that made it died before
reporting; after is 83.2% against a bar of 74); the adminui module under `-race`
(it ran without the race detector, plus the tenants subset); anything about
multi-replica behaviour of the new email lock (it is a single-instance guard by
design and untested across processes).

Everything below is limited to this session's own work and what it noticed on the
way, as instructed.

---

## a) FULLY DONE

| # | Item | Receipt |
| --- | --- | --- |
| a1 | HTMX response options apply on a QuerySet route (the previous report's real defect d1): `Trigger`, `TriggerWithDetail`, `Notify*`, `PushURL`, `Redirect` are no longer accepted and dropped | `respondLoaded` in `query_set.go` mirrors `App.Query` (headers first, then the renderer; a non-HTMX redirect stays terminal). 4 specs: trigger + push-url with the body still rendered, redirect precedence for a non-HTMX client (303, renderer never called), `HX-Redirect` for an HTMX client, and the same without an App |
| a2 | `OnLoad(func(r *http.Request, report LoadReport))` — the success-path hook | Fires for every load that did not abort (also a degraded one, after `OnLoadError`); never for an aborted one. 4 specs |
| a3 | An unroutable fragment is refused BEFORE any loader runs, and the rejection lists what would have matched | `viewSet.requireRoutable` runs as `handlerConfig.beforeLoad`; spec proves zero loader runs; the message reads `no view matches target "#nowhere" and no Default region is declared (routable: page, region #item-list)` |
| a4 | The apply adapter never skips silently | A nil result applies the zero value of `V` (pointer-typed and interface-typed loaders behave the same); any other mismatch is a classified error `cqrshtmx.queryset.apply_failed`. 3 white-box tests in `query_set_internal_test.go` |
| a5 | The untested corners named in the previous report | Specs for: a loader's own `context.Canceled` (reported, not swallowed), the App timeout, `WithTimeout` (a deadline actually reaches the loader), `WithOpenAPI` (the route is collected), a two-failure report in declaration order, and the default WARN line naming every degraded loader |
| a6 | A crash bug in the previous wave is fixed: a zero-value `View[T]{}` passed declaration as a Page with a nil component and panicked at render | The zero `viewKind` is now invalid; `Views` refuses a zero-value view, any view without a component, and a `Page` or `OOB` view marked `Default`, collecting every problem into one panic. 6 new specs |
| a7 | Root lint is truly clean | `golangci-lint run ./` → 0 issues. The previous session's "0 issues" receipt was wrong: `declareViews` (cyclomatic 14 of 12) and `loaderRun` (exhaustruct) were live findings, and CI's `lint (.)` job was red on them at `b0a0e013` |
| a8 | Benchmarks for the seam, and the single-render proof | `benchmark_views_test.go`: 3 benchmarks, 12 sub-benchmarks, `b.Loop` + `ReportAllocs`. A page or region render costs 5 allocs/op — identical to the `RenderPartialOrFull` branch it replaces; the scheduler's floor is 8 allocs for one trivial loader; one out-of-band view adds 8. A spec counts `Render` calls: the primary view once, each out-of-band view once, never the page on a fragment |
| a9 | The wire bytes are pinned | `TestViews_Golden` + `testdata/views/` (page, boosted navigation, history restore, region with an out-of-band fragment), regenerable with `-update` |
| a10 | The guide is rewritten to match the code | `docs/guides/query-set-to-templ.md`: the `nav` typo is gone (the "before" now shows the component loading data, which was the point), plus the response-options row, `OnLoad`, the free refusal, the scope/shape agreement, a cost table, and the reading list no longer cites a browser spec that does not exist |
| a11 | ADR-0068 carries its three same-day decisions | Amendment section + four new rejected alternatives: response parity, two load hooks, and the seam staying in root for v4 (measured surface: 20 exported declarations in two files, 1,027 lines with docs and internals) |
| a12 | adminui: each tenant's member count is read once per request under the members sort | Was N + page reads (2N on one page); now N. `TestTenantsIndex_MemberCountReads` counts through a small `Handler.memberCount` seam: 24 → 12 reads for twelve tenants, 33 → 23 for twenty-three at ten per page |
| a13 | adminui: the tenants list pages in a stable order again | Since 2026-11-24 only `sort=members` sorted; the default name sort was skipped and the read model returns map order, so every page was a random window. Sabotage run (guard restored for one test run): page 1 showed `tenant-03, 06, 07, 11, 12, 15` and page 2 repeated `03, 06, 07, 11, 12, 15`. `TestTenantsIndex_NameSortIsDeterministic` and the rewritten `TestTenantsList_SortByMembers` (it now reads the table fragment — the full page's name-sorted switcher had masked the bug) both fail on the old logic and pass on the fix |
| a14 | The two hermetic lint reds are gone, and with them CI's `lint (identity-model)` | `identity-model`: `FoldUser` cyclomatic 34 → 18 as a dispatch table over per-event helpers, behaviour pinned by a 12,000-fold differential run against the old function (module coverage 77.7% → 87.0%). `legalparty-model`: 11 → 0 findings. Lint sweep 17/17 |
| a15 | TODO row 98 — five verified email bugs in usermgmt are fixed | A blank address on `ChangeEmail` emitted an `EmailChanged` carrying `""`, and because `UserState.Exists()` is `Email != ""` every later command answered `*.not_found`. Now: the change-email and register deciders refuse blank addresses (`usermgmt.change_email.email_required`, `usermgmt.register.email_required`); `Service.ChangeEmail` parses the address and answers `usermgmt.email_exists` when a different user holds it; `ValidateCommand` no longer skips an empty email; Register, ChangeEmail, the JSON import and the OAuth2 first-login hold a striped per-address lock across their uniqueness check and dispatch (a probe had produced 5–6 duplicate users per 24 racing registrations). 18 new tests incl. `-race` concurrency tests; consumers green |
| a16 | TODO row 63 — the two inherited module suites ran hermetically with the race detector | `examples/samber-do-demo` ok 4.3 s, `loginpage` ok 4.1 s (`GOWORK=off go test ./... -count=1 -race`) |
| a17 | `transport`'s heartbeat test no longer bets on one 50 ms nap | It failed once under load average 200 (`expected at least 1 heartbeat comment frame, got 0`), passed 3/3 on re-run, and now retries the stream window up to five times; 5/5 with `-race` |
| a18 | CHANGELOG receipts for everything above | Root `[Unreleased]` (one Added, four Fixed entries) + the usermgmt, adminui and identity-model module changelogs |
| a19 | The docs-gates `annotate` gate is green | Its three rule-F spacing findings (one in the previous report of this session, two in other sessions' docs) reworded; re-read: `docs-annotate: 249 file(s), 1526 struck item(s), 0 finding(s)` — CLEAN. `freshness` reads FRESH with this report in place |
| a20 | Four gates this session had turned red, or found red, are healed | `root-size` re-pinned 9,825 → 10,080 LOC (the seam's follow-through: `OnLoad`, the pre-load check, the apply adapter's error path, the declaration split — the decision to keep it in root is ADR-0068 amendment 3); `root-clusters` regenerated (✓ 57 files / 11 clusters); `root-files` green again — `.docs-gates-preflight.json`, a docs-gates tool cache the daemon committed on 2026-11-22, is untracked and gitignored (34 files = baseline; this was CI's red `module-architecture` job); `check-large-files --all` green — `docs/review/libs/index.render.png` re-encoded to an 8-bit palette, 1,473,160 → 469,743 bytes (this was CI's red `checks` job since the 11-22 render) |
| a21 | Three net-new design-analyser findings fixed or adjudicated | Two `FLAG_PARAM` on the legalparty close helper fixed at source (`closeRequest.requireCloseable(lifecycle)`); one `PHANTOM-COLLISION` accepted as the third `ParseEmail` call site of an already-baselined class. Baseline 248 → 249 with a triage row; the gate re-read after the daemon's commit: OK (+0 added, −0 removed, ~204 modified, =38 unchanged) |
| a22 | 24 new Ginkgo specs in the three seam files | 73 now (49 at the previous report); the suite runs 581 specs |

## b) PARTIALLY DONE

| # | Item | What is missing |
| --- | --- | --- |
| b1 | **The usermgmt email work was finished by a subagent that died before reporting.** It hit the account's five-hour usage limit mid-verification. I re-verified everything myself at 22:50: `-race` ok, consumers ok — and found 6 lint findings in its new test code (one cyclomatic, four unchecked returns, one cognitive complexity 49), which I fixed (lint 0) | No coverage-before number; the lock-order reasoning exists only as code comments plus one concurrency test, not as the agent's written argument; the multi-replica limit is documented but untested by construction |
| b2 | **Per-function coverage in the two seam files**: 38 of 47 functions at 100% | Below: `QuerySetHandler` 76.9% (its render-error and refusal branches are only exercised through the App), `loadAndRespond` 81.8%, `LoadQuery` 83.3% (the nil-dispatch and nil-build guards), `respondLoaded` 85.7%, `viewSet.render` 87.5% (the out-of-band render-error branch), `admit` 85.7%, and the two `String()` methods at 66.7% (their unreachable default arms) |
| b3 | **Documentation that should have moved with the code did not.** The CHANGELOGs are written; the guide and the ADR are current | NOT written: the AGENTS.md "Props seam" Key Patterns block (the previous report's f2 — still the biggest doc gap); the skill (the view-sets section in `SKILL.md` and `references/core-api.md` do not know `OnLoad`, the response parity or the free refusal); FEATURES.md; the `docs/guides/fullstack-wiring.md` row; the amendment to gotcha 30 (an example must never call an unreleased root symbol — check `examples/*/go.mod` and `check-module-isolation --examples` BEFORE writing example code) |
| b4 | **docs-gates: one of the three red gates is closed, two are untouched.** `annotate` reads CLEAN. `audit-all`: 438 unharvested open items against `audit_max: 263` (204 reports replayed). `cadence`: 31 live reports (threshold 15), oldest 8 days (threshold 21) | An archive round (`nix run .#archive-status`, dry run first, `TAIL_BASELINE` down in the same change) and the drain-or-re-pin decision on audit-all |
| b5 | **TODO_LIST.md was not edited this session.** Reading alone showed at least four stale rows: 52 (the `go mod tidy -diff` pre-push leg — the hook has had that leg since 2026-11-21), 63 (done, a16), the F40/F50 halves of row 89 (done per plan M13/M14 on 2026-10-06), and 187 (the README TOC revisit fell due 2026-11-17) | A HARVEST pass: retire those with evidence, route this report's §f |
| b6 | **The `index.render.png` fix changes the bytes of an artefact whose owner eyeball is still queued** (TODO row 35, item B10.4) | One human look at the re-encoded file; a lossless re-encode was tried first and stays above the limit (1,448,558 bytes) |
| b7 | **History quality.** Everything is on `master` and pushed, as ~25 daemon commits titled `chore: auto-commit N changed file(s) (heuristic)`; 63 files, +3,698 / −414 since the session's first commit | No reviewed commit message for any of it; one of the daemon commits (`719c66f8`) captured a throwaway differential test file that was deleted again in the next |
| b8 | **The new benchmarks are not in the bench-spike gate** | A pinned baseline needs an idle machine and a deliberate addition to the gate's bench list |

## c) NOT STARTED

| # | Item | Why it matters |
| --- | --- | --- |
| c1 | Root tag `v4.20.0`, then the `examples/basic` conversion and its three browser specs (previous report f11/f12) | Everything in the seam is still consumable by nobody outside the root package; owner-gated |
| c2 | The fleet nixpkgs floor decision (ADR-0003 rev to go ≥ 1.27.4, previous f14) | The standing red in AGENTS.md / CURRENT_FOCUS.md — but see f19: the strict train gate read 0 lag at HEAD tonight |
| c3 | Full chromium battery, composite `check-modules`, `.#coverage-gate` | The three whole-tree receipts this session never produced |
| c4 | AGENTS.md, the skill, FEATURES.md, fullstack-wiring, the gotcha amendment, the plan's §12 execution log | See b3 |
| c5 | HARVEST of both reports' §f into TODO_LIST / ROADMAP (previous f50) | See b6 |
| c6 | The archive round and the audit-all decision | See b5 |
| c7 | TODO rows 53, 54, 84 (the fast e2e slice script, the sync offline-reads flake posture, the e2e stale-port guard), 104, 105 (the CSP-knob guide and skill recipe), 107 (five gate micro-items), 143 (`nolint-audit` CI wiring), 149 (templ formatter-to-generator chain), 186 (`RequireSession` export decision) | Read and triaged, not executed |
| c8 | adminui adoption of the seam (previous f33/f34) and the batched membership-count ask (previous report f27) | Still blocked on a batched read; filing is owner-gated |
| c9 | ROADMAP rows for the seam's deferred tail (previous f35–f48) and for two ideas I would now reject with a reason: a typed swap-strategy set for `OOB` (htmx swap values are an open grammar with modifiers — a closed set would refuse valid values) and dropping the out-of-band detection buffer (it buys rule parity with `OOBTempl`) | Nothing is recorded; the reasoning would be lost |
| c10 | ROADMAP rows for the email probe's decision items: the avatar feature, branded-email constructor parameters, a database unique index on email, re-keying `UserState.Exists()` onto a registered flag, and the second email normalisation in the OAuth2 path | Each needs an owner decision; none is written down outside the probe's output |
| c11 | Adminui's two `AllTenants()` scans per request (`tenantsIndex` and `tenantSwitchOptions` via `h.page`) | Noticed by the subagent, not touched |
| c12 | `TOTPDisabled` is the one payload-bearing user event whose payload is never decoded; `Credentials` / `ExternalAccounts` appends can write into the caller's spare capacity | Pre-existing, noticed by the identity-model subagent, untouched |
| c13 | An idle-machine benchmark re-measure and a benchstat comparison | The guide's cost table says "indicative" and means it |
| c14 | CURRENT_FOCUS.md still describes the 2026-11-21 family train | Every session starts there |

## d) TOTALLY FUCKED UP

**d1 — the session stood still for four hours.** The usermgmt subagent hit the
five-hour usage limit at about 18:48 and failed; the orchestrator stopped with it;
nothing moved until the owner's prompt at 22:47. The work in the tree was fine
(the daemon had committed it) but nobody could see that, and the report the owner
had been waiting for did not exist for four hours.

**d2 — a behavior change to a published module sat lint-red on `master` for about
four hours.** The usermgmt email change went in as daemon commits between roughly
18:10 and 18:48; its new test code carried 6 hermetic lint findings; the daemon
pushes; CI's `lint (usermgmt)` job failed on `c03cd564` (read through `gh run
view`). Fixed at 22:55 (lint 0). CI also showed what this session DID heal: at
`b0a0e013` (before it) the red jobs were `checks`, `lint (.)`, `lint (identity-model)`
and `module-architecture`; at `c03cd564` only `checks`, `lint (usermgmt)` and
`module-architecture` — `lint (.)` and `lint (identity-model)` are gone, and the
other two are addressed in a20. The next CI run has not been read.

**d3 — the previous wave of this same day shipped a request-time panic and a
false gate receipt.** The zero-value `View` crash (a6) was in the code I wrote this
morning; it surfaced only because the "0 issues" lint receipt in the 16-39 report
was not true (a7). A receipt copied from memory instead of from command output at
HEAD is worse than no receipt.

**d4 — a three-day-old pagination bug in adminui was invisible to its own test.**
Since 2026-11-24 the tenants list paged randomly (a13). `TestTenantsList_SortByMembers`
asserted on the full page, where the tenant switcher's links come first and are
name-sorted whatever the table does. The test was green, the list was broken. Not
caused by this session; found because the one-pass fix forced a close read.

**d5 — two near-misses of my own, caught on re-read and not by a test.** My first
`importUsers` refactor appended an empty error string on the "address already
taken" path, which would have changed the import result; and my first restructure
of the legalparty close helper re-introduced the very duplication the fix was for
(twice: two identical struct literals, then the two decider bodies). Both were
caught by running lint and reading the diff, neither by the suite.

**d6 — one race-suite run failed on a pre-existing timing test.**
`TestServeDomainEvents_HeartbeatEmission` returned zero heartbeats once at load
average 200 (a17). Re-run: green. It is hardened now, but the class — a fixed
sleep that assumes the goroutine gets scheduled — may exist elsewhere; I did not
audit for it.

## e) WHAT WE SHOULD IMPROVE

**What did you forget? (honest list, in descending consequence)**

1. **AGENTS.md and the skill, again.** The previous report's number-one lesson
   was "the wave's pattern belongs in AGENTS.md"; this session added `OnLoad`, the
   response parity and the free refusal and still did not write the block or touch
   the skill. CHANGELOG entries were written 20 minutes before this report, after
   the interrupt, not with the change.
2. **A subagent that dies must be treated as unfinished, immediately.** The
   usermgmt agent's work was committed and pushed by the daemon while its
   verification (lint, race, consumers) had not run. I verified it four hours
   later because the owner prompted. Lint-last ordering inside a subagent means an
   interrupted agent leaves lint-red code on `master`.
3. **Gates were healed at the end, not as the code changed.** `root-size`,
   `root-clusters` and `branching-flow` went red when I grew the seam and were
   re-pinned only while writing this report.
4. **The composite receipts were skipped again**: no `check-modules`, no browser
   battery, no coverage gate — the same three "NOT VERIFIED" lines as the previous
   report.
5. **TODO_LIST.md was not touched** although four stale rows were identified by
   reading.

**What could you have done better?**

- **Batch the work per change: code, specs, lint, race, CHANGELOG, the two docs
  that every session loads.** This session batched code and specs, then stopped.
- **Run the race suite and the lint sweep after every subagent report**, before
  starting the next task — the six usermgmt findings would have surfaced at 18:10.
- **Verify a claim before relying on it.** The strict release-train gate reads
  0 train lag tonight, while AGENTS.md and CURRENT_FOCUS.md still describe the
  four-entry appkit hold as the standing red. I noticed it and recorded it (f19)
  instead of chasing it.
- **Keep the fix proportional.** The legalparty close helper went through three
  shapes to satisfy two analysers; the end state is right, the path was long.
- **Scope the PNG fix deliberately.** Re-encoding a "human-eyeballed render
  receipt" to an 8-bit palette is defensible (a crop was inspected, a lossless
  re-encode does not fit) but it is a judgement the owner should confirm (b6).

**What could you still improve?**

- Write the AGENTS.md block and the skill section first thing, before any code.
- Add a per-session closing ritual to the runbook: lint sweep, race suite on
  touched modules, consumers, `root-size`/`root-clusters`/`branching-flow`,
  CHANGELOG, then the report.
- Teach `check-large-files.sh` and `check-root-files.sh --strict` to run locally
  in `check-modules` if they do not already (both were CI-only reds here).
- Give the daemon's heuristic commits a readable alternative at phase boundaries
  (question 1).

## f) Up to 40 things we should get done next

Ordered by impact per unit of effort. `[P0]` = before building further on the
seam; `[P1]` = next session; `[P2]` = soon; `[P3]` = backlog.

| # | Task | Why / done-when |
| --- | --- | --- |
| f1 | `[P0]` AGENTS.md "Props seam" Key Patterns block | The five rules (two invariants, out-of-band on fragments only, fail-loud selection that costs no read, response parity, loader-agnostic loaders) are in the loaded context |
| f2 | `[P0]` Skill: `OnLoad`, response parity, the free refusal, and an `OnLoad` example fence in `SKILL.md` + `references/core-api.md` | `check-skill-claims` and `check-skill-snippets` green; the skill matches the code |
| f3 | `[P0]` FEATURES.md rows for `OnLoad` and the view-declaration validation | The inventory is honest |
| f4 | `[P0]` Re-run `docs-gates run .` with this report in place | `freshness` stays PASS; `annotate` reads PASS after the three fixes |
| f5 | `[P0]` Read the next CI run on `master` | `lint (usermgmt)`, `checks` (large-file guard) and `module-architecture` (root files) are green; if not, name the step |
| f6 | `[P0]` The docs-gates battery once as a whole command, with this report in place | Four verbs were read singly; one matrix is the receipt |
| f7 | `[P0]` Gotcha 30 amendment: the unreleased-symbol pre-flight for examples | `cat examples/X/go.mod` + `check-module-isolation --examples` BEFORE writing example code; two sessions lost time to it |
| f8 | `[P1]` Composite `nix run .#check-modules` at the final HEAD, reds recorded per leg | One composite receipt instead of per-leg claims |
| f9 | `[P1]` Full chromium battery on the now-quiet machine | 127 tests, one verdict |
| f10 | `[P1]` Root tag `v4.20.0`, then re-apply the `examples/basic` conversion and the three browser specs from plan §12 | Browser-level proof of the seam (owner-gated) |
| f11 | `[P1]` Close the per-function coverage gaps in the seam files (b2) | `QuerySetHandler`'s refusal and render-error branches through the bare handler; `LoadQuery`'s nil guards |
| f12 | `[P1]` HARVEST both reports of 2026-11-27 into TODO_LIST / ROADMAP; retire rows 52, 63, the F40/F50 halves of 89, and decide 187 | `check-todo-hygiene` green; no stale row left |
| f13 | `[P1]` Archive round: 31 live reports against a threshold of 15 | `nix run .#archive-status -- <spec.tsv>` dry run first, `TAIL_BASELINE` lowered in the same change, `cadence` PASS |
| f14 | `[P1]` audit-all: drain or re-pin 438 against 263 | Owner decision recorded either way |
| f15 | `[P1]` Annotate the 16-39 report's §f items this session closed | Inline `done at` markers; audit-all shrinks by ~25 items |
| f16 | `[P1]` Plan §12: add the follow-through execution log | The plan of record stops at the first pass |
| f17 | `[P1]` ROADMAP: ADR-0068 amendment 3 and the two rejected seam ideas with reasons (c9) | The reasoning survives |
| f18 | `[P1]` ROADMAP: the email probe's five decision items (c10) | Each becomes a visible owner decision |
| f19 | `[P1]` Trued the standing-red claims: is the four-entry appkit hold over? | The strict train gate read 0 lag; AGENTS.md / CURRENT_FOCUS.md say otherwise — verify, then true both |
| f20 | `[P1]` Rewrite CURRENT_FOCUS.md | It describes 2026-11-21 |
| f21 | `[P1]` TODO rows 104 + 105: the CSP-knob guide section and skill recipe | Two doc rows read and not executed |
| f22 | `[P1]` TODO row 107: the five gate micro-items | Each one-liner-class, batchable |
| f23 | `[P1]` TODO row 186: `RequireSession` decision | ADR-0062 forbids the re-export; document setup as the home |
| f24 | `[P1]` Idle-machine benchmark re-measure with benchstat; trued the guide's cost table | `-benchtime=2s -count=5`, load under 6 |
| f25 | `[P1]` Owner eyeball of the re-encoded `index.render.png` (b6) | One look; revert to a Git-LFS route if the palette is not acceptable |
| f26 | `[P2]` TODO rows 53, 54, 84: the fast e2e slice script, the sync flake posture, the e2e stale-port guard | The e2e tooling tail |
| f27 | `[P2]` TODO rows 143 + 149: `nolint-audit` CI wiring, the templ formatter-to-generator chain | Two tooling rows |
| f28 | `[P2]` adminui: pass the loaded tenants into the page shell (c11) | One `AllTenants()` scan per request |
| f29 | `[P2]` adminui under `-race`, whole module | Ran without the race detector this session |
| f30 | `[P2]` `.#coverage-gate` and `.#check-erraudit-strict` at the final HEAD | Two standing receipts |
| f31 | `[P2]` legalparty: the stale `country.go` lint exclusion and `run.go 1.27.2` in its lint config | Config hygiene |
| f32 | `[P2]` legalparty: the LEI path still returns canonical errors unwrapped while the register path wraps | One rule for both identifier paths |
| f33 | `[P2]` usermgmt: `ValidateCommand` for the external-account link when an email is present | Product decision first |
| f34 | `[P2]` Audit fixed-sleep timing tests for the heartbeat shape (d6) | A list of candidates, then harden or leave |
| f35 | `[P2]` The seam's deferred tail into ROADMAP (previous f35–f48) | Per-view props, streaming, cache keys, the introspection accessor |
| f36 | `[P3]` identity-model: `TOTPDisabled` payload decoding, the append aliasing (c12) | Two recorded decisions |
| f37 | `[P3]` bench-spike baseline re-pin on an idle machine, with the new benchmarks added deliberately | Standing row |
| f38 | `[P3]` `docs/guides/fullstack-wiring.md`: the seam in the decision table | One more discovery path |
| f39 | `[P3]` Teach `check-modules` to run the large-file and root-files guards locally | Two CI-only reds would have been visible |

## g) Questions I cannot answer myself

1. **Commit policy.** Everything from this session is on `master` and pushed as
   about 25 daemon commits titled `chore: auto-commit N changed file(s)
   (heuristic)` — 63 files, about 3,700 lines, with no reviewed commit message.
   The harness forbids me to commit without an explicit "commit". Do you want
   phase-boundary commits with detailed messages to be authorized for sessions
   like this one, or is the daemon's history acceptable as is?
2. **`Service.ChangeEmail` changed meaning for consumers.** It now refuses an
   invalid or blank address, refuses an address another user holds, and stores
   the address lower-cased (exactly as `Register` always did). That is a
   behavior change in a published module. Ship it as is in the next usermgmt
   minor, or should `ChangeEmail` keep the caller's letter case and only
   validate?
3. **Root `v4.20.0`.** The strict release-train gate read 1,062 requires, 0
   unpublished, 0 train lag tonight — the four-entry appkit hold that AGENTS.md
   calls the standing red is not in its output. May I cut the tag on the current
   floor (it unblocks the `examples/basic` conversion and the browser proof),
   or do you want the fleet floor moved first?
