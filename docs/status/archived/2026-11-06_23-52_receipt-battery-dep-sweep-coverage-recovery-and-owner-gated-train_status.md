# Status: Receipt Battery Complete — Dependency Sweep to Zero Lag, Coverage Recovered, Split Brain Fixed, Train Still Owner-Gated

> **ANNOTATED 2026-11-07** — ~~this report~~ **SUPERSEDED** by [`2026-11-07_00-15_templ-era-conflict-train-lag-zero-and-push-unblock_status.md`](../2026-11-07_00-15_templ-era-conflict-train-lag-zero-and-push-unblock_status.md), which Its §g1 (train) and §g3 (push) were answered by execution (the v4.17.0 train + the 60-commit push); §g2 audit-all rides TODO row 21 canonically; §g3 cqrs-lint posture stays owner-gated. Strike depth:
> supersession verdict (round-18 precedent for series-internal reports); nothing here is a
> unique open item. **ARCHIVED 2026-11-07 (train-execution).**

**Written:** 2026-11-06 23:52 CET · **Author:** Crush (receipt-battery session, continuation of the 21-31 root-package-gates session)
**Repo:** `/home/lars/projects/cqrs-htmx`, branch `master` · **HEAD at write time:** `409ce0df` · **53+ commits ahead of origin, unpushed**
**Session window:** 22:00 → 23:52 CET (single session, one sibling lane active throughout — its sweep landed as real commits at the end, `409ce0df` closing the last 5 train-lag pins)
**Plan context:** carried handoff from the 21-31 session (plan of record `docs/planning/2026-11-06_09-48_SUPERB-train-first-recovery-and-gate-debt-burndown-pareto-plan.md`); this session executed that plan's receipt battery, the archive round, and everything ownable short of the owner-gated train and push.

**TL;DR:** Every item ownable without the owner's tick is done and verified: the docs-gates battery is wired everywhere with a 9/9 self-test; six reports archived (tail 14, cadence green); six lagged go-cqrs-lite pins swept to zero treadmill lag across 26 consumer modules; root coverage recovered above its 90% floor with real tests; a stale templ-components pin (v1.26.0) bumped to the v1.27.0 train with same-change CSS rebuilds; a genuine doc-vs-code split brain in `WithSyncPullLimit` found by writing tests and fixed at the doc; branching-flow drift adjudicated and re-pinned; cqrs-lint's 4-module red root-caused as the API-vs-tag class (not binary drift — my first diagnosis was wrong, see §d). The release train and the push remain owner-gated; the train now also heals seven red legs (route-postures ×2, cqrs-lint 4-module leg, module-isolation ×2, coverage unmeasured ×4).

---

## §a — Fully done (receipts)

1. **C21 docs-gates battery — COMPLETE.** `scripts/selftests/test-docs-gates-battery.sh` (9 stub cases: green matrix, advisory WARNING, strict exit 1, broken-battery exit 2, crash not promotable under strict, honest SKIP, non-executable binary, unknown arg, `--json` passthrough) 9/9 green + shellcheck-clean. Wired into: flake apps `check-docs-gates-battery`/`test-check-docs-gates-battery`, two `check-modules` stages, CI-parity table, advisory `ci.yml` step with the membership comment (CI-only + advisory because `audit_max` is local policy and the binary is not a flake input). Verified from every surface: `nix-instantiate --parse`, YAML parse (9 jobs), both nix apps, real battery through the app. The parity self-test caught my stage env-prefix bug (§d1) — fixed same session.
2. **Round-24 archive pass.** Six settled 2026-11-05/06 reports archived with specific successors and non-generic reasons (22-31→03-19, 23-57→10-54, 00-27→22-03, 05-24→13-24, 07-24→08-42, 08-51→21-31) via `nix run .#archive-status -- --apply --update-baseline` (dry run first). 9 links re-based across 7 files (incl. ADR-0063 + two planning docs). `TAIL_BASELINE` 20 → 14, cadence green. Dry-run validated before apply; apply rc 0.
3. **Annotate gate CLEAN.** Fixed the rule-F violation in my own 21-31 report line 38 (the id.CommandID plural touching the backtick → "values of type"); then annotated §f items 3–6 and 8–11 (this session's completed work) inline with strikethrough + receipts, per the ANNOTATE ritual. `docs-gates annotate` → 159 files / 571 struck / 0 findings.
4. **Six go-cqrs-lite pins swept to zero treadmill lag.** catalog v4.13.0, id v4.7.5, metaengine v4.22.0, metaengine/sqliteengine v4.7.1, system v4.17.0, systemscenario v4.3.0 — via per-consumer `go get` + `go mod tidy` with `go.work` renamed (26 consumer modules, zero GET/TIDY failures). `check-upstream-treadmill --strict-lag 0` → **worst lag 0** (one kv DISAGREEMENT warning remains — warning-only). Version greps verified post-sweep: 4/4, 26/26, 5/5, 2/2, 4/4, 1/1 new pins, 0 residual.
5. **Workspace build + vet green post-sweep;** targeted `go test` green on systemadapter, usermgmt, dashboardui, examples/system-demo.
6. **Branching-flow re-pinned under existing verdicts.** Gate verdict +17 added / −13 removed / ~158 re-attributed: es_casbin_projection PHANTOM-COLLISION ×11 + http.go ×1 (frozen casbin-slot branding-advice class), huma PHANTOM-TRANSPOSE ×2, `commandOptionApplier`/`queryOptionApplier` DUPLICATE_TYPE ×5 (the AGENTS-documented structural-interface seam), adminui mixins ×2, setup large-struct ×1. Baseline re-pinned 203 → 210; amendment appended to `docs/analysis/triage-decisions.md` (2026-10-12 provisional-acceptance precedent cited; this session authored none of the underlying code). Gate **OK — no new findings** after the baseline commit.
7. **reexport-policy adjudicated.** `dashboardui/core_bridge.go` alias growth 17 → 19 (`+commandFate` fleet-plan A7 `a48cf1d8`, `+eventThroughput` wave-5 `3db33a89`) proven package-internal facade plumbing over the extracted core package (gotcha-48 mandated shape; used only by dashboardui's own templ pages — zero consumer surface). Baseline row updated with the receipt; `--strict` green.
8. **Root coverage leg GREEN — 89.6% → 90.4% (floor 90).** Three test additions: `coverage_delegation_test.go` (6 tests: NotificationLevel.String, delegatingWriter.Unwrap through http.ResponseController, ParseSSEEventID family incl. newline rejection, WriteSSEEvent wire format + SetSSEHeaders, WithMaxReplay first-connect clamp, WithSyncPullLimit default/override), `htmx/request_coverage_test.go` (Ginkgo: ParseRequest all headers, RenderPartial both branches, context round-trip, package-level accessors context-vs-header-fallback), httperr `structured_test.go` append (String=JSON identity, Unwrap cause traversal via errors.Is, nil-err zero values, NewStructuredErrorForAudienceWithContext, WithHTTPStatus wrapper chain + nil passthrough). Aggregate 91.6% measured mid-work.
9. **templ-components v1.26.0 → v1.27.0** (root, adminui, dashboardui — the only three stale holders, verified by survey; latest tag confirmed v1.27.0 via ls-remote). CSS bundles rebuilt in the SAME change (`build-adminui-css` + `build-dashboardui-css`); `check-css-bundles` + `check-css-bundle-classes` green (class sets exact, 1057 tokens); adminui + dashboardui tests green post-bump, page goldens stable (no drift to adjudicate). This fixed adminui's hermetic test run (its v1.26.0 generated code calls the templ-removed `ResolveAttributeValue`).
10. **WithSyncPullLimit doc-vs-code split brain fixed.** The comment claimed the option was "the ceiling when ?limit= is present" — the handler (`parseSyncPullQuery`) treats explicit `?limit=` as a pure override clamped only by the global `MaxSyncPullLimit`; the option is the default only. Doc corrected to the actual contract; the test pins default=2 applied without `?limit=` and explicit `?limit=4` overriding. Found by writing the coverage test — the test failed against the doc, not the code.
11. **webauthn hermetic go.sum completed.** My sweep left `fxamacker/cbor`/`x/sys` go.mod-file hash gaps that hermetic (GOWORK=off) builds tripped over (`train-consumers-hermetic` leg). `GOWORK=off go mod tidy` in `usermgmt/webauthn` → hermetic build OK. Committed in `95c3f173`.
12. **Session commit `95c3f173`** (scoped paths, gotcha-50 `--no-verify` under the sibling's hook-red window, every file hand-verified, window documented in the message body). The daemon carried the rest of the session's artifacts.
13. **TODO_LIST trued (hygiene clean, 78 rows, 0 dead):** row 21 receipt-battery sentence (treadmill 0, erraudit 319, branching re-pin, cqrs-lint 10/14, battery 4/5, archive round), row 46 audit-all re-measure 419 with the archiving-drains-nothing confirmation, row 49 cqrs-lint re-red receipt with the correct API-vs-tag root cause + ambient-toolchain trap + binary-drift facts.
14. **Corpus ledger discipline held:** round-24 archive sentence + receipt-session true-up sentence appended inline to the `docs/status/README.md` ledger paragraph; `CORPUS_BASELINE` trued twice same-change (53667, then 53940 after the sibling's pin); ratchet green throughout.
15. **Canonical cqrs-lint binary rebuilt** from the published `cmd/cqrs-lint/v4.20.0` tag per the documented git-archive + ldflags recipe (`dev (commit: 18eeb6f)`) and used for the comparative analysis that disproved my first root-cause theory (§d5) — the recipe itself is re-verified working.
16. **Final composite residual fully classed** (12/100 red at the last run, every leg attributed): API-vs-tag (module-isolation ×2, train-consumers-hermetic, route-postures ×2, cqrs-lint 4-module leg, coverage-unmeasured ×4) — heals at the train; catalog docserver poison (workspace-build in examples/catalog-demo) — upstream-gated; erraudit/loginpage — sibling mid-flight churn, since landed.

## §b — Partially done

1. **Coverage gate overall:** root + adminui legs now pass (90.4%, 67.4%); **usermgmt, dashboardui, setup, systemadapter remain unmeasured** — hermetic runs fail on the unreleased root/UI APIs (DropStats/NewBroadcaster/Nav). Train-gated, not test debt.
2. **check-modules end-to-end:** baseline run (13/100 red) + final run (12/100 red) both recorded and fully triaged, but **no clean-window all-green run was achieved** — the sibling's sweep was mid-flight during both. The class list is complete; a quiet-window re-run is the remaining step (the owner's `train-preflight` re-runs everything anyway).
3. **docs-gates battery: 4/5 PASS** — `audit-all` (419 vs pin 322) is the open leg; the drain-or-re-pin is owner question 2.
4. **cqrs-lint: 10/14 green** — the 4 reds are the same API-vs-tag class; the advisory pile (identity-model E005 ×17 + C013 ×2 + A009, huma A025/A009, new sibling C044 sites in dashboardui/events.go) exits rc 0 and is untriaged per-site.
5. **The 21-31 report's §f:** items 3–11 annotated done with receipts; items 1 (train), 2 (push), 12–14+ (post-train sequence) remain open by design.
6. **erraudit:** TOTAL=319 measured (303 at 10-31) but the per-module growth (usermgmt 27, setup 8, examples…) is unadjudicated — advisory posture makes this non-blocking.
7. **The freshness harvest step from the handoff** became moot when freshness went PASS (the sibling's harvest + my archive drained it) — verified twice, not worked around.
8. **TODO row 46 (audit-all):** re-measured and trued, but the drain itself (≈97 items to annotate) not started.

## §c — Not started (owner-gated or queued behind the train)

1. **The release train** (identity-model → root → usermgmt → adminui+dashboardui → setup, sweeps between steps) — owner question 1; now also the fix for seven red legs.
2. **The push** (53+ local commits incl. the sibling's work) — owner question 3.
3. **audit-all drain** to the 322 pin (or the owner-approved re-pin) — owner question 2.
4. **AGENTS.md trues** (stale shellcheck count, the "all five gates PASS" claim, missing concurrency rule, missing go.work-deletion gotcha) — sibling-held file.
5. **FEATURES.md ADR-0062/0063 rows** — sibling-held file.
6. **Post-train sequence:** consumer require sweep, `bump-dep` go-appkit 4/4 PASS re-run, `verify-remote-tags`, `push-and-watch`, route-postures/cqrs-lint/isolation/coverage heal verification.
7. **bench-spike re-pin** on a genuinely idle machine (refused three times historically under load).
8. **Upstream cqrs-lint filing candidates** surfaced this session: E005 cross-module handler resolution (identity-model's commands register in usermgmt; the bijection gate proves registration), V003's systemcatalog "v4.3.x exists" claim (ls-remote shows only v4.0.0) — content not drafted.
9. **cqrs-lint distribution** (the standing TODO P2/P3) — the ~/go/bin shadowing problem this session hit argues for it.
10. **humane E005/A-rules disposition** for the advisory pile (disable-with-reason per module vs upstream fix) — a posture decision needing the owner's review pattern from M10.

## §d — Totally fucked up (self-attributed failures, this session)

1. **The docs-gates-battery stage line shipped broken once.** I wrote `bash DOCS_GATES_NO_NIX=1 bash scripts/...` into the check-modules stage list; the runner cannot parse env prefixes ("DOCS_GATES_NO_NIX=1: command not found") and the parity self-test flagged the mismatch. The parity gate caught it exactly as designed, but the broken form briefly existed on the CI-parity surface. Fix: plain forms both sides.
2. **The self-test's PATH-restricted cases hid `bash` itself** (rc 127) — two red runs before the absolute-path fix. I restricted PATH without checking which of my own invocations depended on it.
3. **I ran `check-cqrs-lint.sh` directly with the ambient toolchain and trusted the all-red result.** The script exports `GOWORK=off` + `GOTOOLCHAIN=local` and its own header says bare-bash runs fail all 14 modules with load errors — I read neither before running, mis-diagnosed "binary drift" as the root cause of the 4-module red, built the canonical tag binary, ran comparative analyses, and started drafting an E005 disable — all before discovering the flake-app run showed only 4 reds, all API-vs-tag. The canonical binary still earned its keep (it disproved the drift theory and verified the recipe), but the diagnosis order was backwards: **read the gate's contract first, theorize second.**
4. **My first root-cause claim (binary drift) was wrong and I wrote it down before correcting it** — the correction now lives in row 49, but the wrong version existed in my working notes and one ledger sentence draft. The discipline I preach ("a verdict counts only when you read which gate produced it") extends to root causes: name the mechanism before writing it.
5. **The bump-dep sweeps were no-ops and I almost logged them as the fix.** "no go.mod requires a module matching…" meant the lagged pins are indirect MVS floors bump-dep skips by design; I had to re-derive the real consumer map and sweep with `go get`. The first treadmill re-run after bump-dep would have been a false green if I had trusted the "ALL SIX SWEPTS" line.
6. **My go.work rename dance collided with the daemon:** the deletion got committed to HEAD (`c9fc2609`), my restore came back as a staged re-add, and the workspace walked around for ~30 minutes with HEAD lacking a tracked file the gotcha index explicitly says is tracked (gotcha 11). Consequences rippled into two confusing gate runs. The rename trick needs a daemon-safe protocol (§e6).
7. **I started the full composite while the sibling had uncommitted in-flight docs** — the corpus ratchet leg red'd from the race and one baseline run's signal was partially noise. Load/collision discipline: check `git status` immediately before the long gate, not two minutes earlier.
8. **I ran the composite TWICE in one background command** (a `grep -c` pass plus a second full run for the FAIL list) — ~25 extra minutes of machine time on an already-loaded box and mangled output. One run, tee to a file, parse the file.
9. **Three build-failure cycles writing the coverage tests** (`http FlusherOf` — a nonexistent API I invented; `sse.Event.Type` instead of `.Event`; a branded-type conversion the compiler rightly rejects). The go-sse v0.7.0 source was one grep away the whole time; I wrote against memory instead of reading the actual Event shape first.
10. **My first commit attempt hit the pre-commit hook red** (14 steps failed on the sibling's in-flight tree) — I had not checked hook state before attempting, though the carried handoff explicitly warns to check hook/hook-path state first. Recovered correctly (scoped `--no-verify`, gotcha-50 protocol, window named in the message), but the check should have been proactive.
11. **audit-all growth attribution was arithmetic-by-vibes** ("419 = 409 + my 50 §f + sibling"): direction right, itemization never done. Row 46 records the number, not a proof.
12. **The "5-case self-test" I promised the §f item shipped as 9 cases** — over-delivery is not a failure, but the plan-vs-actual drift went unrecorded until the annotation; receipts should state deltas, not just totals.

## §e — What we should improve

1. **Read a gate's env/toolchain contract before invoking it directly.** Every script in `scripts/checks/` should get a one-line "invoke via: `nix run .#<app>`" banner at the top — the cqrs-lint trap cost an hour and produced a wrong published-ish diagnosis.
2. **Record the walk mode with every gate verdict** (workspace vs GOWORK=off hermetic; flake-app toolchain vs ambient). A verdict without its mode is half a verdict — this session produced three.
3. **Pin the check-modules stage-line format in a fixture self-test** (no env prefixes, `bash <cmd>` shape) so the next gate author cannot repeat §d1 — the parity self-test caught it, but only after the fact.
4. **Document in the treadmill row that `bump-dep` sweeps direct requires only** — indirect MVS floors need `go get`+tidy per consumer; a TODO one-liner would have saved the entire no-op sweep detour.
5. **Daemon-safe go.work protocol:** the rename dance needs a handshake (a lockfile the daemon respects, or `git stash push go.work` instead of `mv`) — the committed deletion (§d6) is the third go.work-collision incident this week.
6. **Heavy-job coordination:** my composite ran while a sibling sweep was live (bump-dep printed collision warnings both directions). A shared lock file or a pre-flight `pgrep` gate for multi-minute jobs would stop interleaved gates shredding each other's signal.
7. **Pre-commit state check before the first commit of any session:** `core.hooksPath`, which steps will run, and whether a sibling's in-flight tree will red tree-wide steps — the gotcha-50 exception is for documented windows, not for surprises.
8. **Read the dependency's actual source before writing tests against it** (three build cycles, §d9).
9. **The canonical cqrs-lint binary should be flake-pinned** (or at least repo-path-pinned): a ~/go/bin install any session can shadow is the standing drift source (4.16 → 4.19.1 → 4.20.0-tag → 4.20.1-master in three days). The 11-05 recipe + this session's rebuild both exist because of it.
10. **erraudit deltas per module** (growth attribution), not just the TOTAL — the total moved 303 → 319 and the "who" is a guess.
11. **The E005 finding class deserves an upstream filing draft** (cross-module handler resolution, bijection gate as the compensating-control evidence) — file or fix, never absorb (buildflow loop rule 7).
12. **The sibling's 23:0x archive pass and mine raced on the same ledger paragraph** — both landed, but the paragraph is becoming a write-hot single line; a structured manifest table (per-pass rows) would stop the inline-append collision class entirely.

## §f — Next things to get done (≤50, ordered by leverage)

1. **Owner decision Q1 (train authorization under concurrent writers)** — unblocks items 2–14.
2. **Owner decision Q2 (audit-all drain vs re-pin)** — unblocks items 15–17.
3. **Owner decision Q3 (push authorization + sibling-held doc edits)** — unblocks items 18–20.
4. Run `nix run .#train-preflight` in a quiet window (mandatory since S52; record skip reasons per stage).
5. Tag `identity-model` first (`verify-tag` → push → tag+push), then root, then usermgmt, then adminui+dashboardui, then setup — the documented fire sequence with `--refresh-cache` after fresh tags.
6. Sweep consumer requires after each tag (`bump-dep '<module>$' <ver> --commit`).
7. Verify with `check-release-train --refresh-cache --strict-lag 0` + `verify-remote-tags` (peeled-ref audit).
8. Post-train: re-run route-postures (expect green — the manifest pins resolve), cqrs-lint (expect 14/14), module-isolation ×2 + train-consumers-hermetic (expect green), coverage-gate (expect all 15 measured).
9. Post-train: re-run `check-modules` end-to-end in the quiet window and drive the residual to the documented catalog-poison-only class.
10. `bump-dep` go-appkit sweep expecting 4/4 PASS (row 21's named receipt).
11. `verify-remote-tags` over the new tags; `push-and-watch` filtered CI watch.
12. Push the 53+ commits (after Q3) — includes my `95c3f173` and the sibling's landed train-lag pins (`409ce0df`).
13. Watch the CI `checks` job green; confirm the new docs-gates battery advisory step self-skips as designed on runners.
14. File the upstream go-cqrs-lite dispatch-middleware ask (owner-gated, row 21).
15. audit-all drain, batch 1: annotate the 21-31 report's remaining open §f items (12–50 minus the owner-gated ones) with dispositions — the largest single-report contributor.
16. audit-all drain, batch 2: the 09-06 and 09-38 reports' §f lists (50 items each per their headers).
17. audit-all drain, batch 3: archived-file items (the tool replays them; each needs its own inline evidence).
18. After the drain: lower `.docs-gates.json` `audit_max` toward the measured residue — never raise (policy).
19. Decide archived-replay semantics with the owner (excluding archived/ from audit-all would change the pin's meaning — Q2 sub-question).
20. Edit AGENTS.md after Q3: true the shellcheck count, retire the "all five gates PASS" claim, add the concurrent-writer rule, add the go.work-deletion gotcha (gotcha 11 conflict + this session's c9fc2609 incident).
21. Add the two missing FEATURES.md rows (ADR-0062 direct-imports, ADR-0063 dispatch-boundary) after Q3.
22. Draft the E005 upstream filing (cross-module handler resolution; evidence: identity-model E005 ×17 exit-0 advisory + `check-command-bijection` green + usermgmt tests green; the 11-05 triage's "green 14/14" does not reproduce at its own commit — include that).
23. Draft the V003 false-data filing (systemcatalog "v4.3.x" claim vs ls-remote v4.0.0-only; binary version that emits it noted).
24. Decide E005 posture with the owner: upstream fix vs disable-with-reason in identity-model (+ the strategy modules) with the bijection gate as compensating control — mirror the F032/M10 pattern.
25. Pin cqrs-lint in the flake (or a repo path) to end ~/go/bin shadowing; record the 4.20.1-master V003 false-data receipt in the AGENTS cqrs-lint row when done.
26. Extend `check-modules-parity` fixture to pin the stage-line format (no env prefixes) — prevents §d1 class permanently.
27. Add the treadmill-row note: `bump-dep` sweeps direct requires only; indirect floors via `go get`+tidy.
28. Daemon-safe go.work protocol: implement the lockfile (or stash-based) handshake; add a gotcha entry for the committed-deletion incident.
29. Split the README corpus ledger paragraph into a structured per-pass table (§e12) — one row per pass, no more inline appends.
30. Gate the next C044 sites: the sibling's new dashboardui/events.go filter + overview cursor sites need the same display-consistency review the 11-05 triage did (they exit 0 — advisory, but the class review is the standing control).
31. Review identity-model C013 ×2 (CBOR epoch on `SessionStartedPayload.StartedAt` / time.Time fields): adopt `event.Instant` or document the wire-format reason — the CBOR advisory has a concrete suggested remedy.
32. Decide the `WithSyncPullLimit` ceiling question: an explicit `WithSyncPullCeiling` option is a real API addition (v5 candidate) — the doc now says override-only; make that intentional in ADR form or leave it.
33. Update the frontend-sync guide's pull-limit wording to match the corrected `WithSyncPullLimit` contract (guide §pull).
34. Re-run `docs-gates run .` post-push and record the matrix in the next report (freshness/annotate/routing/cadence expected green; audit-all per Q2).
35. Close the gotcha-50 window: once the sibling's sweep is fully landed and the hook goes green, verify a normal (non-`--no-verify`) commit passes; note the closure in TODO.
36. Give TODO row 49 a post-train verify line (cqrs-lint 14/14 expectation) so the next session can close it mechanically.
37. Re-run `check-css-bundle-classes` after the sibling's next templ-components-family bump — the class-set gate fires on their render changes, not mine.
38. TOTP hydrate note check: confirm no SQL read model was rebuilt this session (none was — the `RebuildProjection` TOTP caveat stays dormant).
39. Bench-spike re-pin on an idle machine (load < 6, twice-quiet rule) — the legacy remaining leg from row 67.
40. `nix run .#check-docs-tail-budget --ratchet` after every new report until the tail returns to 3 (the archive cadence should reclaim the budget within two rounds).
41. Keep the kv DISAGREEMENT warning on the radar (v4.3.5 ×11 vs v4.4.0 ×1) — warning-only, but it is the next treadmill item after the train.
42. Sweep `integration_test`'s projectionadapter local replace (gotcha 36 un-hermetic class) — still the only tracked filesystem replace; needs the upstream tag wave to retire.
43. After the upstream catalog release with regenerated docserver templ (post-v4.13.0): bump `examples/catalog-demo` + `integration_test`, expect the workspace-build + vet legs to close — the last upstream-gated class.
44. Verify the CI `mod-tidy`/`security`/`checks` jobs on the pushed HEAD (Dependabot graph runs aside).
45. Split-brain sweep for the huma module's A025/A009 advisories (new module, two exit-0 advisories — adopt or suppress with reason while the module is fresh).
46. Add the `humane`/adminui ExtraNav render-preset coverage the wave-5 report promised (sibling lane, not started per their report).
47. Prune stale TODO rows that rows 21/46/49 receipts superseded (row 21's "First wave" clauses now executed — true them to DONE inline next docs-health pass).
48. Next docs-health pass: archive the settled 2026-11-06 evening set (22-42, 21-31 when its owner-gated residue is decided) — tail back toward 3.
49. Keep `/tmp/cqlbin`'s canonical binary recipe in AGENTS (it exists; the row references the recipe — add the "verify with `cqrs-lint version` reads `dev (commit: 18eeb6f)`" step as STEP 0 of triage).
50. Genuinely wait for the owner's answers to §g before items 1–3 — the remaining work is sequenced behind them by design.

## §g — Questions only the owner can answer

1. **Train authorization under concurrent writers.** The tree now has 53+ unpushed commits including the sibling's landed train-lag pins (`409ce0df`), the sibling's sweep has settled into real commits, and the build/vet workspace state is green — but the auto-commit daemon still commits every few minutes, and a bad tag is irreversible (module-proxy poisoning). May I run the release train (§f items 4–8) while the sibling lane is active, or is single-writer a hard precondition — and if concurrent is acceptable, may I use the documented fire sequence without a further checkpoint?
2. **audit-all endgame.** The replay count is 419 against the pinned 322 and archived files replay too (so archiving drains nothing — verified twice). Do you want the multi-session drain (annotate ≈97 open items each with its own evidence, §f 15–18), an owner-approved re-pin to the measured residue, or a semantics change (stop replaying archived reports) — and if the last, should the pin move with it?
3. **cqrs-lint posture and binary custody.** (a) May I file the E005 cross-module resolution defect upstream (identity-model's 17 advisory findings with the bijection gate as counter-evidence), or do you prefer disable-with-reason per module first and upstream second? (b) May I pin the cqrs-lint binary in the flake to end the ~/go/bin shadowing (the 4.20.1 master build currently installed emits a factually false V003), or is the system-installed binary a deliberate rolling posture?

---

*Receipts for every claim above live in the annotated §f rows of the 21-31 report, TODO_LIST rows 21/46/49, `docs/analysis/triage-decisions.md` (2026-11-06 22:24 amendment), the corpus ledger in `docs/status/README.md`, and commit `95c3f173`.*
