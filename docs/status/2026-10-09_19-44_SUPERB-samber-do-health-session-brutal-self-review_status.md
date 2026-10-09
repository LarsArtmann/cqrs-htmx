# Status: samber/do × health SUPERB execution — full session self-review & comprehensive update

> **Date:** 2026-10-09 19:44 CEST
> **Scope:** this session only (resumed from the Phase-1 handoff ~17:20, closed at 19:44). Predecessors: review [`2026-10-08_samber-do-v2-health-checks-deep-review.md`](../research/2026-10-08_samber-do-v2-health-checks-deep-review.md) + Outcome Annex · plan [`2026-10-09_16-52`](../planning/2026-10-09_16-52_SUPERB-samber-do-health-truth-and-composability-pareto-plan.md) (annotated) · Phase-1 status [`17-19`](2026-10-09_17-19_SUPERB-execution-phase1-status.md) · execution status [`18-10`](2026-10-09_18-10_SUPERB-samber-do-health-execution-phases-2-5_status.md) (annotated).

## a) FULLY DONE (verified, this session)

| # | Item | Evidence |
| --- | --- | --- |
| 1 | T5: root `ProjectionStatus*` constants + `ProjectionDrainReady` + tests + CHANGELOG receipt | commit `0012e2dd`; root suite green, root lint 0 issues |
| 2 | T4: health `RecorderChain` + `DetailedHealthRecorder` durations (`duration_ns` on the wire) + `NewProbe` on `NewWithDetailedCheck` (implements #31); tests + CHANGELOG receipt | commit `f0e6f7e5`; health suite green `-race`, lint 0 issues, demo suite green |
| 3 | T6: integration proofs — lifecycle E2E (Start/RegisterRoutes//readyz 200//startupz latch) + auditlog plugin-is-recorder (compile-time assertion + real probe batch) | commit `8669f538`; hermetic (GOWORK=off = CI mode) test + lint green |
| 4 | T7: docs — health/README surface-map table + do extras (all API claims source-verified against go-health v0.5.1 / do v2.1.0), auditlog README literal RecorderChain snippet, setup/README cross-link, root README module tree entry; fixed wrong `/startedz` → `/startupz` | commit `ff1abd0b` |
| 5 | T8: setup/README drain posture — RunWithAppkit = production drain path (verified against `run_appkit.go` FIRST: ReadyCheck, DrainDelay, /health/live) | commit `ff1abd0b` |
| 6 | T9: harvest — TODO_LIST follow-through row, AGENTS health bullet + gotcha 37, plan annotated in place, research Outcome Annex with F12 repro embedded durably | commit `ff1abd0b` |
| 7 | F12 CORRECTED + reproduced: `do.InvokeAs` resolves by interface-scan over the service map → map-order nondeterministic with ≥2 interface-satisfying services (15–28/200 leak, single-goroutine, 5 runs); safe named-alias pattern verified 400/400 | research Outcome Annex; AGENTS gotcha 37 |
| 8 | Boot proofs (beyond plan): `buildRouter` extracted from main (single source of truth for the mux), `TestBoot_RouterBuilds`, `TestBoot_ProcessSmoke` (compiled binary on ephemeral port via new `PORT` env, /readyz polled to 200), README PORT hint | commit `b6277605`; demo suite green `-race`, lint 0 |
| 9 | Full hermetic battery `nix run .#test`: 13/14 modules ok — the 1 red is systemadapter's pre-adjudicated foreign replace (gotcha 36) | `/tmp/battery-test.log`; documented, not mine |
| 10 | Lint battery: all 4 of my modules 0 issues (root, health, integration hermetic, demo); the only red = usermgmt exhaustruct ×4 (foreign in-flight structs) | `/tmp/battery-lint.log` |
| 11 | PUSH (authorized): reconciled the mid-flight divergence (origin had my pre-amend T5 shape) via merge taking ours, post-merge tree verified byte-identical, strict pre-push gates green (841 requires published, 0 drift), pushed `ae7a25ee` | CI run 37957309392: every job my commits touch GREEN |
| 12 | CI verdict triaged: all 8 red jobs pre-existing foreign classes (systemadapter replace ×4, identity-model + usermgmt exhaustruct, dashboardui exhaustruct_v5 panic, govulncheck CVE policy); no NEW red introduced | annotated in 18-10 report §e + TODO_LIST push-window row superseded |
| 13 | Both status gates green after every report write (annotations + rows) | `check-status-annotations.sh` / `check-status-rows.py` rc=0 |

## b) PARTIALLY DONE

| Item | State | What's missing |
| --- | --- | --- |
| T6 f26 "run integration_test BOTH modes" | Hermetic mode green (the mode CI uses) | Workspace mode blocked all session by the sibling go-cqrs-lite `schema.EventSchema` in-flight breakage (10×45s retry loop stayed red); verify once their wave settles |
| Push closure | Everything through `ae7a25ee` pushed | The two final annotation commits (`22db1689` daemon-captured + `edd15cb6`) are STILL UNPUSHED — the push ran before the last annotations |
| Daemon-fold discipline | Folded 5/6 times (T5, T4, docs, boot, T6) | Missed the final one: `22db1689` (daemon capture of my annotation edits) sits below `edd15cb6` unfolded |
| f16/f25 (mirror deletion + chain proof in integration_test) | Correctly deferred with in-code comments + TODO rider | Lands at the health train (gotcha 30 — NOT a sloppiness, a sequenced deviation) |

## c) NOT STARTED (deliberate or owner-gated)

1. **T12** unify projection-health error codes (owner-gated, wire-visible).
2. **T13** `Bundle.MarkDraining()` vs docs posture (owner-gated).
3. **Upstream filings** — samber/do InvokeAs scan, go-health method-less RegisterRoutes (owner decision; repro + mechanism ready).
4. **crush-config `references/lessons.md`** cross-project entries (queued in TODO_LIST; another repo).
5. **`docs/agents-notes.md` narrative** for gotcha 37 — AGENTS footer requires every gotcha's root-cause story there; I added the gotcha but NOT the notes entry (missed until this review).
6. **T15** (setup go-health probe seam spike → likely ROADMAP) and **T17** (examples tagging policy) — plan "rest" tier, out of the authorized 20%.
7. **#31 issue comment** ("implemented, rides next health train") — not posted; owner may prefer closing at release.

## d) TOTALLY FUCKED UP (process stumbles, all recovered, zero lasting damage)

1. **The rebase attempt during push reconciliation** — `git rebase origin/master` conflicted on my own duplicate content and left my 6 commits UNAPPLIED mid-operation (HEAD == origin, my tree content gone from HEAD). Caught immediately via the identity-diff check (822 deletions visible), aborted cleanly, switched to merge-with-ours. Lesson: for same-content divergence, merge beats rebase; I should have predicted the per-commit replay conflict I explicitly called "expected clean".
2. **First `safeBuffer` in boot_test.go** — channel-based, and `String()` reset the channel (data loss on second call). Caught by self-review before ever running; replaced with a mutex buffer.
3. **First F12 repro reconstruction** — I initially rebuilt the handoff's claim with the wrong mechanism (named-alias path: 0/600 stale, refuting the claim as stated). Spent 3 iterations before isolating the real class (unrelated named override leaking into type-keyed InvokeAs scan). The handoff's /tmp repro itself was a mid-iteration husk that could NOT reproduce its own claim — worth remembering that handoff claims need re-verification, not just capture.
4. **Missed the final daemon fold** (`22db1689`) after folding meticulously all session — the daemon won the last race because I committed annotations without checking `git log` first.
5. **Initial misread of the CI watch output** — the watch's ✗ grep surfaced 3 jobs; the jobs API showed 8 reds. I almost reported "3 reds" — dug deeper only because the count looked low. Always pull the jobs list, not the watch tail.

## e) WHAT WE SHOULD IMPROVE (session-derived)

1. **Pushes should be the LAST action, after all annotations** — or annotations go in the pre-push commit. Pushing then writing more docs guarantees unpushed stragglers (happened: 2 commits stranded).
2. **A "concurrent-session pushed MY commits" runbook is missing** — I improvised merge-ours + identity-diff. This WILL recur (daemon + multi-session + shared remote); the identity-diff guard (`git diff pre post` must be empty) deserves a tiny script.
3. **The F12 capture proves handoff repros must be RE-RUN before being captured as durable truth** — the /tmp artifact contradicted its own claim; 20 minutes of empirical work turned a wrong mechanism note into a source-verified one (InvokeAs interface-scan). Without that, the upstream filing would have been wrong.
4. **Docs-first verification paid off every time** (run_appkit.go, NewChecks package, InjectorOpts field, DefaultRoutes names — caught `/startedz` bug) — keep source-verifying every README claim.
5. **The 10×45s sibling-retry loop was wasted compute** — hermetic mode made it moot for my redesigned tests; I should have switched to hermetic verification the moment the tests were redesigned for it (that insight came late).

## f) NEXT (up to 50, ordered)

**Immediate mechanical (minutes):**
1. Fold `22db1689` into `edd15cb6` (soft reset + recommit) and PUSH the annotations.
2. Write the `docs/agents-notes.md` gotcha-37 narrative (InvokeAs story + push-reconciliation story).
3. Verify integration_test in WORKSPACE mode once the go-cqrs-lite schema wave settles (the one unfinished verification leg).
4. Comment on #31: "implemented in ae7a25ee, ships on the next health train" (or leave for owner — see Q3).
5. Sweep `go mod tidy -diff` hermetic loop over my touched modules (belt-and-braces; CI already green on them).

**Health train rider (when the owner cuts it):**
6. Delete `statusLive/statusStopped/statusFailed` mirrors in `health/probe.go`; consume `cqrshtmx.ProjectionStatus*`.
7. Bump health's root require to the tag carrying the constants.
8. Add the deferred integration_test chain proof (`RecorderChain(Recorder(svc), auditPlugin)` against the real Service).
9. Root CHANGELOG [Unreleased] → release section; health receipt rides the same train.
10. Re-verify the auditlog README snippet compiles against the tagged module (it references `health.RecorderChain`).

**Owner decisions:**
11. T12 error-code unification (health `cqrshtmx.health.*` → root `projection.*`).
12. T13 `Bundle.MarkDraining()` vs RunWithAppkit docs posture.
13. Upstream filing: samber/do InvokeAs interface-scan nondeterminism (repro ready in the annex).
14. Upstream filing: go-health RegisterRoutes method-less patterns (F11 note).
15. crush-config lessons.md entries (cross-repo; F12/F11 classes).

**Standing foreign debt (not mine, tracked):**
16. GCL systemscenario wave publishes → drop systemadapter's filesystem replace (un-reds 4 CI jobs).
17. Train-lag 42 alignment (templ-components v1.21.0 + go-retry + httputil consumers) after the wave.
18. identity-model + usermgmt exhaustruct literals (their tasks 5-7) — their session.
19. dashboardui exhaustruct_v5 panic (their question 3).
20. govulncheck stdlib CVEs → fleet toolchain floor (their question 2).

**Quality polish (this session's leftovers, low priority):**
21. T15: evaluate a setup seam for serving go-health probes (spike → likely ROADMAP).
22. T17: examples-module tagging policy check + disposition.
23. Demo README: document `buildRouter` as the test seam in the design-decisions list.
24. Consider a tiny `scripts/tools/reconcile-diverged-master.sh` (merge-ours + identity-diff guard) from §e2.
25. Boot-smoke in CI: confirm `TestBoot_ProcessSmoke` runtime stays acceptable (~2s; it ran green in CI already).
26. health/README: add the dashboard `Accept: application/json` content-negotiation detail (minor).
27. Research annex: cross-link gotcha 37 back from the report (one line).
28. Rubric re-score propagation: TODO_LIST version note already carries 8→8.5; keep for the next full-code-review episode.

## g) QUESTIONS I CANNOT ANSWER MYSELF (max 3)

1. **Upstream filings (f-item 13/14):** file the samber/do `InvokeAs` interface-scan issue (repro + mechanism are ready and voice-checkable), note go-health's method-less `RegisterRoutes`, or keep both internal? Filing carries your name and implies a fix expectation upstream; I cannot judge your appetite for owning those threads.
2. **T12/T13 now or never-ish:** both are wire-visible/behavioral. Do you want T12 (error-code unification) and a verdict on T13 (`MarkDraining()` vs docs posture) bundled into the next health train, or decided separately after you've run the demo? The train bundling choice changes what the rider commits contain.
3. **Health train timing + #31:** cut the health/v4 + root tags NOW to land the mirror deletion, chain proof, and #31's duration_ns for consumers (a small train is hours of gates) — or hold until the GCL systemscenario wave + lag-42 alignment clears so one release window covers everything? And should I post the "implemented, rides the train" comment on #31 in the meantime?

*Point-in-time snapshot per [`docs/status/README.md`](README.md). Items resolved later get inline strikethrough + a dated `> ANNOTATED` blockquote, never a rewrite.*
