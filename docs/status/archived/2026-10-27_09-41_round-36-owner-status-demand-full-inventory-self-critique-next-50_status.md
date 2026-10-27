# Round 36 — Owner status demand: full inventory of the W12→W17 session, self-critique, next 50

> **ANNOTATED 2026-10-27** — ~~this report~~ **SUPERSEDED** by [`2026-10-27_13-59_round-37-docs-health-2026-0-corpus-audit-living-docs-truth-pass_status.md`](2026-10-27_13-59_round-37-docs-health-2026-0-corpus-audit-living-docs-truth-pass_status.md), which round-37 declares itself the round-36 continuation: it executed the round-36 section-f harvest into TODO_LIST/ROADMAP rows, carries the remaining section-f tails verbatim, and carries round-36's three open questions on its own section-g (branching-flow hold, upstream-filings hold, release train). Round-36's push hold is now resolved - ebfcd5e2 was pushed and CI run 40512643594 came back green.. Strike depth:
> supersession verdict (round-18 precedent for series-internal reports); nothing here is a
> unique open item. **ARCHIVED 2026-10-27 (round-39 (2026-10-27, issue/PR sweep)).**

**Date:** 2026-10-27 09:41 CET · **Reporter:** module-lane session (round 35 continuation) · **Head at report:** `820b01cd` (master GREEN, CI 40439932229 all jobs success) · **Scope:** this session's run only, as instructed · **Box load at session end:** 100–165 throughout (drove verification strategy)

## a) FULLY DONE (this session)

1. **W12 wrap closed** — CI green through every push; `verify-remote-tags` 179/179 peel identically; book reconciliation `f6dec11d..HEAD` vs receipts consistent.
2. **W14.3 re-verified → root-caused → fixed → CI healed.** The round-34 "unverified, probably load-140 environmental" verdict was WRONG: `fakeQueryJournal.ReadQueriesFrom` ignored cursor AND limit, so `loadAllQueries` appended a full batch forever. The local 611s timeout (wall-clock `go test` default = a HANG, not slowness) and both CI runner shutdowns (exit 143, the OOM class) were this one bug. Fix: real cursor semantics in the fake + fail-fast stall guards in both export walkers (`*_cursor_stalled` infrastructure errors) + regression tests. CI lint reds fixed same change (wrapcheck ×2, wsl ×2). Master green at `05c7347b` (run 40398258036).
3. **W14.4** — the three core journal walkers (`LoadTailEvents`, `LoadEventByID` scan, `FindEventNeighbors`) stall-guarded (error / error / degrade-to-no-navigation) with stalling-journal tests; snapshot detail 404/500 split (not-found → missing page, store failure → 500, no more masking outages as "No snapshot found"); delete treats already-gone as idempotent success; rendered-page boost-inheritance guard walks every GET route and fails on unpinned ajax elements (the 2026-10-18 blanking class). dashboardui suite+lint green hermetically.
4. **W14.5** — `biome.json` REMOVED (verdict: nothing consumes it — proven against treefmt, BuildFlow runs, CI, scripts; devShell binary + `.buildflow.yml` claim removed with it); `check-root-files` allowlist gate + baseline (31 files at landing) + 8-case self-test, wired into check-modules + CI; `ack.go` broadcast-peek audit = safe; dprint tombstone confirmed accurate; ~280MB regenerable binaries trashed (never `rm`).
5. **W14.6** — SSE guide `WithRequestFilter` + `TailReader`/`ReadLastN` recipes; README SSE rows + actor row (auth-audit trio); actor guide ↔ usermgmt README cross-links; `SSEOptions.Options()` conversion test pins the RequestFilter branch. Root suite+lint green.
6. **W15.2/W15.3** — `check-skill-claims.sh` (221 symbols; caught REAL drift: `cqrshtmx.ErrValidation` never existed — fixed in SKILL.md + README; the root actor bridge is `ParseActorID`/`ActorIDFromUser`, not `NewActorID`); `check-skill-snippets.sh` (40 code fences parse; `//doc:api-tour` marker convention for API tours; 29 reference fences skipped); `check-readme-splitbrain.sh` (module READMEs, per-doc allowlist for auditlog's intentional shadowing). Three gates + three self-tests in check-modules + CI.
7. **W15.4/W15.5 canaries a–g** — (a) `tidy-sweep.sh`; (b) fixture-poison canary (8 fixtures keep their markers, clean-go-mod stays clean); (c) BuildFlow skip_steps canary (7 documented skips); (d) coverage fail-loud; (e) `docs/runbooks/toolchain-floor.md`; (f) `.codespellrc` + clean pass over living docs; (g) coverage thresholds single-sourced in `scripts/checks/coverage-thresholds.tsv` (flake app + all 10 CI steps read one file).
8. **W17 verdicts + sweep + closure docs** — 8/8 parked-row verdicts in place (7 pre-recorded 2026-10-25, row-43's added); TODO swept 62→58 rows (4 completed rows removed, 3 updated), hygiene gate green; round-35 receipt written (round-34 archived, tail kept at 2); CURRENT_FOCUS rewritten; AGENTS truth pass (coverage row → TSV; new-gates row; the awk-procsub host quirk).
9. **Battery of record GREEN** — CI runs 40438109268 and 40439932229: ALL 16 jobs success (test, module-architecture, checks with every new gate, 11-module lint matrix, build, security, mod-tidy). Two self-inflicted reds owned and fixed in-flight (see §d), final state green.

## b) PARTIALLY DONE

1. **Skill-maintenance row** — the three gates landed; the row's remainder (authored evals + skill description optimization pass + deep re-audit of core-api/realtime/usermgmt references) is open.
2. **dashboardui 09-48 row** — everything landed except: page-size selector off-menu option; browser-level e2e smoke. Both ride the next dashboardui change.
3. **w15.1 art-dupl tails** — verdict = HELD per §g1 (the named extraction targets predate the W11 file splits; re-identification + baseline refresh is one act). I did NOT pre-stage the read-only re-identification analysis that would make the owner tick one-click — defensible, but it was possible.
4. **Local battery legs** — honestly refused at load 100–165 (`.#test`/`.#lint` fleet/`.#coverage-gate`/`.#check-modules` composite not run locally); CI is the record. Note: the check-modules excuse was the weakest — most of its stages are offline fixture self-tests that do not care about load.
5. **tidy-sweep.sh** — smoke-tested on ONE module (auditlog); never a full 29-module sweep (it is a tool, not a gate; nothing runs it automatically).

## c) NOT STARTED

1. **w15.6 upstream ask drafts** — HELD per §g2 default (package with the S55 packet).
2. **Release train** for loginpage `NoWebAuthn` + dashboardui fixes — §g3 question open; session default taken: batch into the next train.
3. **Branching-flow baseline refresh** — HELD per §g1 (+46/−40/~111 drift vs baseline, mostly the W11 file splits).
4. **S55 packet chains** (cqrs-lint fleet swap → CI wiring; ReadLastN upstream) — owner-gated.
5. The authored skill evals / description pass / reference re-audit (see b1).
6. bench-spike (W16's load-gated leg — sibling lane, restated for completeness).

## d) TOTALLY FUCKED UP (honest ledger — all caught, all fixed, none hidden)

1. **I shipped a CI-wired script tested only for its FAILURE modes.** `check-coverage-threshold.sh` was verified on: missing row, missing profile, real local run from repo root — but never in the CI step's SHAPE (cwd=root, profile in a subdir). Coverage profiles carry module-relative source paths; `go tool cover` from the root could not resolve them. The usermgmt leg went red on the battery-of-record run. Fix: measure from the profile's directory (`91c6aca2`).
2. **I generated the root-files baseline before staging the file that changed the set.** `.codespellrc` was untracked when `--update-baseline` ran → baseline=31; once committed, tracked set=32 → the gate flagged ITS OWN AUTHOR at HEAD. The gate worked exactly as designed; I was the drift. Fix: re-pin at 32 (`839b5a44`+`91c6aca2`). Two CI cycles burned by d1+d2 together.
3. **Marker-insertion cascade** — inserted 9 `//doc:api-tour` markers by line number against numbers that shifted under my own sequential `sed`s; had to revert all 9 and redo content-anchored (python). Should have anchored by content on the first attempt.
4. **The awk-backtick-in-procsub failure is workaround'd, not diagnosed.** Empirically: literal backticks in an awk program inside `<(...)` silently feed the loop EOF on this host; `"\140"` octal escapes work. I first misattributed it to mvdan/sh, then found `/bin/bash` doesn't exist here and left the AGENTS note honestly vague ("under this host's `bash`"). Root cause unidentified.
5. **Round-34's misdiagnosis was inherited, then corrected too slowly for comfort** — I initially continued the "environmental/load" framing; the wall-clock-timeout-means-hang insight only landed when I read the CI runner-shutdown logs properly. The correction is now institutional (round-35 §c post-mortem).
6. **Edit discipline slips** — 3 "must read before editing" rejections this session (flake.nix mod-time, README, actor guide); one truncated/broken first write of `tidy-sweep.sh` (caught on read-back); one bogus `'false'` pattern line left in the fixtures canary (caught on re-read).
7. **Composite `check-modules` never ran locally once this session** — I wired six new stages into it and proved them individually + via `nix eval` + CI, but the local composite pass (cheap, mostly offline fixtures) was skipped under the load blanket.
8. **The `.codespellrc` `skip=` line is unvalidated** — the living-docs pass ran on an explicit file list; whether the skip patterns parse/apply was never tested.
9. **Daemon-split history** — W14.3's fix and parts of W14.4/W15 live under heuristic auto-commit titles with my proper commit carrying only the tail of each wave (verified content-complete each time, but the ledger requires checking three commits to see one logical change).

## e) WHAT WE SHOULD IMPROVE (durable, not one-off)

1. **CI-shape rehearsal for every CI-wired script**: before push, run the EXACT step (cwd + env + default-path resolution) locally once — failure-mode tests do not cover happy-path shape. This single ritual would have prevented both §d1 and §d2.
2. **Baselines are generated AFTER staging, verified at the exact push tree** (`gate --strict` on a clean tree right before `git push`) — the root-files incident is the template for why.
3. **Content-anchored multi-edits in docs** (match on text, never on line numbers) whenever more than one insertion targets a file.
4. **Run the composite `check-modules` locally at least once per session** that touches its stage list — it is mostly offline fixtures; load is a weak excuse for THAT gate specifically.
5. **Diagnose the procsub/backtick host behavior properly** (isolated minimal repro against the actual shell binary) and record the root cause; the octal workaround stands meanwhile.
6. **Pre-stage owner-gated analysis**: when a verdict is HELD on an owner tick, do the read-only preparation anyway (e.g., the art-dupl re-identification table) so the tick converts to execution in one step.
7. **Beat the daemon on the first attempt** where it matters: verify → stage → commit in one call, and re-verify the daemon's capture (`git show --stat` + status) after every background verification window.
8. **The `//doc:api-tour` convention needs one line in the skill-authoring docs** so future fence authors know the parse gate and the marker exist (currently only in the gate + AGENTS row).

## f) Up to 50 things next (ordered by impact; ★ = unblocks on an §g answer)

1. ★ §g1: authorize branching-flow baseline refresh (owner).
2. ★ §g2: ratify upstream-filings hold + w15.6 drafts (owner).
3. ★ §g3: release train now vs batch — loginpage `NoWebAuthn` + dashboardui fixes are unreleased (owner).
4. Pre-stage the art-dupl re-identification table (read-only) for the §g1 tick.
5. Branching-flow refresh + re-triage as ONE act once §g1 lands.
6. The two deferred extractions (re-identified post-W11 shapes) + foreign `handlers_audit` clone groups + golden page tests for extracted templ cells.
7. cqrs-lint fleet swap to binary 4.16.1 (S55 D6 chain) once owner ticks.
8. `check-cqrs-lint` CI wiring (needs the go-cqrs-lite nested-module tag, S55 §1).
9. ReadLastN upstream filing + `SQLEventStore` DESC LIMIT n (S55 U2).
10. Wire the REAL cqrs-lint walk into check-modules (round-34 §e proposal — currently self-test only).
11. Dashboardui: page-size selector off-menu option.
12. Dashboardui: browser-level e2e smoke (Playwright, one happy-path per panel).
13. Skill authored evals + description optimization pass (skill-creator).
14. Deep re-audit of core-api/realtime/usermgmt reference skills (post-drift-fix).
15. Run `tidy-sweep.sh` over all 29 modules once at low load; record candidates.
16. Validate `.codespellrc` skip patterns actually apply; add a codespell stage to CI if cheap.
17. Diagnose the awk-backtick/procsub root cause; sharpen the AGENTS quirk note.
18. Load-aware verification runbook: codify "CI-shape rehearsal" + "which gates are load-insensitive" into docs/runbooks/.
19. Document the `//doc:api-tour` marker convention where fence authors will see it.
20. Add a self-test for `check-coverage-threshold.sh` (happy path in a temp module dir — the exact shape that failed in CI).
21. W16 sibling-lane legs: BuildFlow nix-fmt source verification + bench-spike at load < 6.
22. CHANGELOG entries for the dashboardui fixes (W14.2/3/4) ahead of the next train.
23. loginpage CHANGELOG check (NoWebAuthn row exists; confirm AutoRedirect×Sessions note).
24. Release-notes draft for the next train (loginpage + dashboardui).
25. `.#coverage-gate` local re-run at low load to corroborate the CI battery.
26. Fleet `.#lint` local re-run at low load (same rationale).
27. Review the six new gates' CI runtime after a few runs (cheap now, watch for drift).
28. The w15.6 draft texts themselves (content exists in archived reports; assemble under docs/proposals/upstream/ once unheld).
29. S55 packet §3 watch: BuildFlow BF1 fixture-tree pruning (upstream).
30. GCL follow-through legs (b-remainder cqrs-lint/coverage + e teardown twin) — owner-gated via S55 §4.
31. dashboardui DLQ/snapshots/projections export (research finding #33, still open).
32. dashboardui table sort/filter generalization (finding #26).
33. Timestamp consistency pass across dashboardui tables (finding #49).
34. DataStar Tier-4: keep watching for the demand signal (no action).
35. SidebarNav: re-check on the next templ-components release for a zero-JS drawer.
36. Consider a `check-verify-tag-fixtures-untidy` negative-control self-test fixture (currently validated against the real tree only).
37. Root-files gate: consider a `--why` field per baseline entry (free-text justification column).
38. Coverage TSV: add a CI guard that every module WITH a test step has a threshold row (inverse drift).
39. Skill-snippets gate v2: build-level compile for fences that import cqrs-htmx (quickstart-gate pattern), parse stays the floor.
40. The 611s-class lesson: add "wall-clock timeout = suspect a hang first" to the testing section of AGENTS gotchas (candidate gotcha 42).
41. Session-start rc guard: surface the current box load in the session-start runbook so verification posture is chosen deliberately.
42. examples/sync-demo: refresh walkthrough if the SSE guide recipes changed its shape (they did not, verify once).
43. `docs/guides/frontend-sync.md`: cross-link the new WithRequestFilter section where per-request filtering is relevant.
44. Audit remaining root-module `_reexport.go` deprecated aliases for v5 removal tracking (README table is the map; confirm it matches).
45. Add the round-35 receipt link to the wave plan's W17 row (plan-vs-receipt cross-link).
46. TODO_LIST row count pressure: 58 rows — run docs-health VERIFY on the P2 section next session.
47. Consider archiving round-35 → round-36 NOW (done as part of this report's ritual) — keep tail ≤ 2.
48. Trash-ritual cadence: schedule the ignored-artifact sweep (quarterly-ish note in the runbook).
49. Investigate the gh logs 403 (token/admin rights anomaly mid-session) — if it recurs, the CI-watch runbook needs the annotations-API fallback documented (this session improvised it).
50. Close the wave: once the sibling lane closes, write the plan's terminal DONE line + final wave receipt.

## g) Questions I CANNOT answer myself (3)

1. **Branching-flow baseline refresh (§g1):** authorize the refresh (+46/−40/~111 drift, dominated by the W11 file splits moving findings wholesale)? Default if silent: keep holding — check-modules' branching-flow stage stays red locally (CI-exempt), and the art-dupl tails stay parked with it.
2. **Upstream filings (§g2):** ratify holding w15.6 (the 5 upstream ask drafts: P15 classifier, event.New encoding, cqrs-lint suppression forms, go-output note, BuildFlow pair) inside the S55 packet instead of filing now? Default if silent: hold.
3. **Release timing (§g3):** loginpage `NoWebAuthn` (new feature) and the dashboardui fixes (rune-safety, export completeness, snapshot honesty, stall guards) are all committed but UNRELEASED — cut a train now, or keep the session default (batch into the next family train)? If "now": name the modules to tag.

---
*Report basis: this session's run only (round 35 continuation, 2026-10-26 21:15 → 2026-10-27 02:40, plus this morning's review). Now WAITING FOR INSTRUCTIONS.*
