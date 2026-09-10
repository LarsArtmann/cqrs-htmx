# Status Report — DataStar Dual-Frontend Rollout Execution (Tiers 1-3)

**Date:** 2026-09-07 21:59 CEST
**Session:** Full Execution Mode of the v2 rollout plan
(`docs/planning/2026-09-07_17-18_datastar-dual-frontend-rollout.html`)
**Tree:** clean, **20 commits ahead of origin/master** (NOT pushed)
**Scope:** Tiers 1-3 of the plan (M0-M8) executed; M9/M10 (gate sweep,
CHANGELOG+push) NOT started; Tier 4 (M11-M16) NOT started.

---

> **ANNOTATED 2026-09-09 (docs-health):** Tiers 1-3 shipped and M9/M10 executed by the 23:36 session (setup/v4.10.0 tagged+pushed). Tier 4 remains demand-gated (TODO_LIST P2 DataStar item; ADR-0050). ARCHIVED.

## a) FULLY DONE (verified green at time of work)

| #  | Item                                                                                                    | Evidence                                                                                                                                                                                                                                                                                                                                                                                                                  |
| -- | ------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1  | **D2 plan source committed** + 2 drive-by example lint fixes (`a1875b23`)                               | observability-demo `SecurityHeadersMiddleware`→`httputil.SecurityHeaders`; middleware-showcase `httputil.ETag`→`etag.New`. Both hermetic build+lint = 0 issues                                                                                                                                                                                                                                                            |
| 2  | **M0 demand validation**                                                                                | GitHub: 3 issues, 0 DataStar, discussions disabled. **15 consumer projects surveyed: 6 integrate go-datastar + cqrs-htmx manually** (InboxClean ×2, KeyHolderAI, bank-sync, crush-daily, file-and-image-renamer); crush-daily hand-rolls a 100+ LOC DataStar SSE bridge duplicating transport-layer logic                                                                                                                 |
| 3  | **M4 ADR-0050** `docs/adr/0050-dual-frontend-protocol-strategy.md`                                      | Route-split (Model A) accepted; Model B rejected; Model C demand-gated; setup gains direct opt-in `datastar/v4` dep; `/ds/events` session-gated 401; `/sse` open decision sequenced by adoption (WithSSEFilter composes below the wire-format split). INDEX.md + research report backlinked. **Discovered the plan's "ADR-0049" label was a numbering error (0049 = layer-split ADR)** — plan annotated non-destructively |
| 4  | **M3 route-split recipe** in `docs/guides/fullstack-wiring.md`                                          | Recipe + hub-sharing sample + ADR-0050 cross-link; consumer-side session gating documented truthfully (setup's `requireSession` is unexported)                                                                                                                                                                                                                                                                            |
| 5  | **M3/F13 contract test** `integration_test/datastar_route_split_test.go`                                | Compiles AND passes: one broadcast on root Broadcaster reaches a DataStar subscriber through the shared hub; script mount 200                                                                                                                                                                                                                                                                                             |
| 6  | **M1 setup implementation**                                                                             | `Config.DataStarPath` + `Config.DataStarScriptPath` (default `/datastar.js`, `"-"` opts out); `Bundle.DataStarBroadcaster` via `NewBroadcasterFromHub`; hub+bridge built when EITHER feed set; mount guards; full path validation. `setup/go.mod` + `datastar/v4 v4.9.0`. Hermetic build+vet+lint(0)+suite green; **coverage 86.3% (gate 80)**                                                                            |
| 7  | **M2 setup tests** (7 tests, `setup/setup_datastar_test.go`)                                            | default-off 404s, script 200/ETag/304, script opt-out, 401 gating + `text/event-stream`, shared-hub SignalsPatch delivery, ds-only-no-/sse, bus-published domain events reaching the feed. Suite green, lint 0                                                                                                                                                                                                            |
| 8  | **M5 setup-demo dual-transport showcase**                                                               | `DataStarPath` enabled; `POST /broadcast` fans out ONE action to both transports (raw event + SignalsPatch); `/ds-demo` client page; README.md created; e2e extended (unauth 401s, script 200, dual frame assertions). Tests 3× green, lint 0                                                                                                                                                                             |
| 9  | **M6 "Using with setup"** in `docs/guides/datastar-integration.md` + cross-link in `datastar/README.md` | Snippet matches shipped API                                                                                                                                                                                                                                                                                                                                                                                               |
| 10 | **M7 guide cross-links** in `docs/guides/sse-and-datastar.md`                                           | Setup option referenced from transport-choice section + See Also updated                                                                                                                                                                                                                                                                                                                                                  |
| 11 | **M8 living docs sync**                                                                                 | TODO_LIST: rollout item rewritten (Tiers 1-3 SHIPPED, Tier 4 remaining, correct ADR-0050 refs); stale "Re-investigate datastar/go-sse" item marked RESOLVED (ADR-0049). ROADMAP: routing note + per-row Tier-4 gating + new "setup DataStar option **Done**" row. FEATURES: datastar section updated                                                                                                                      |

All content is committed (verified in history) — but see (d): mostly via daemon
heuristic commits, not my detailed messages.

---

## b) PARTIALLY DONE

| Item                               | State                                                                                                                                           | Missing                                                                                                                                          |
| ---------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------ |
| **M9 gate sweep**                  | Per-module verification done for every touched module (setup, setup-demo, integration_test, 2 examples: hermetic build/vet/test/lint all green) | The FULL canonical sweep (`nix run .#build/.#test/.#lint/.#coverage-gate/.#check-cqrs-lint`) was never run once, end-to-end, over the final tree |
| **setup-demo hermetic resolution** | Works via `TEMPORARY dev-replace => ../../setup` (sanctioned pattern, removal-condition comment, check-replace-directives green)                | Replace must be stripped when setup publishes the feature — tracked in go.mod comment                                                            |
| **ADR numbering correction**       | Plan carries a top annotation (every "ADR-0049" = ADR-0050)                                                                                     | The plan HTML body still says ADR-0049 in ~10 places (point-in-time artifact; deliberate non-destructive choice)                                 |

---

## c) NOT STARTED

- **M10**: CHANGELOG `[Unreleased]` entry; final detailed commit; **push** (20 local commits sit unpushed).
- **M11** dashboardui signal-patch spike; **M12** dashboardui DataStar variant; **M13** loginpage signal forms; **M14** offline-sync evaluation; **M15** adminui variant design doc; **M16** release train (setup carries the new API — needs a family train to publish).

Tier 4 is demand-gated per ADR-0050 §Decision 1. M0 evidence supports
integration-friction demand (6/15 consumers) but NOT panel-variant demand.

---

## d) TOTALLY FUCKED UP (honest list)

1. **Final tool-call batch was malformed garbage** (a broken `multiedit[...]` array followed by hundreds of junk `bash` invocations) — user cancelled it. This aborted the M9 gate sweep mid-flight. Root cause: I emitted a corrupted tool call and then a runaway batch instead of one clean command.
2. **Lost the commit-message war to the auto-git daemon, 4+ times**: the pre-commit hook kept failing (phantom golangci on dead-GOCACHE mount; later real SA1019s which I fixed), and while my `git commit` ran the daemon swept my staged/working files into `chore: auto-commit N changed file(s) (heuristic)` commits. Roughly a dozen of my milestone changes (ADR-0050, M1+M2 implementation, M5-M8 docs) live in heuristic commits — content verified intact, but history readability for this whole wave is poor. Only 3 commits carry my detailed messages.
3. **Plan mislabeled the ADR number** ("ADR-0049" throughout) — my own error from the planning session; discovered only at execution time. Fixed by annotation, but it should never have happened (the ADR existed since 2026-08-30).
4. **Three of my own test/code bugs found and fixed during the session** (net-positive, but they were mine):
   - `WithModeAppend` used as a constant in samples (it's a function).
   - 404 assertions masked by the login page's `/` catch-all (fixed with `DisableLogin`).
   - setup-demo dual-transport test race: goroutine-owned `defer cancel()` killed the DataStar stream before frame delivery; DataStar stream flushes only on first frame (no connected event) — restructured with channel-passed result + main-owned cancel.
   - First-cut validation bug: two unset optional paths compared as colliding ("" vs "") — caught by the existing appkit test.
5. **Pre-existing doc inaccuracy noticed, not fixed**: `sse-and-datastar.md` CORS sample references unexported `bundle.sseHandler()`.

---

## e) WHAT WE SHOULD IMPROVE

1. **Commit hygiene under the daemon**: stage + commit IMMEDIATELY after each verified milestone, one file-set at a time; consider `--no-verify` earlier when the phantom-hook class is already documented and content is verified — long hook runs (250s) are exactly the daemon's sweep window.
2. **Hook reliability**: the pre-commit golangci failures on example modules (dead `/mnt/buildcache` GOCACHE inside buildflow) burn 4+ minutes per attempt and misfire on lint-clean modules. The documented workaround exists; a structural fix (repoint buildflow's cache or pre-warm) is worth a session.
3. **ADR numbering check**: before writing plans that reference "ADR-NNNN", `ls docs/adr/ | tail` — this cost an annotation pass.
4. **DataStar adapter flush behavior**: `datastar.Broadcaster.ServeHTTP` writes headers but only flushes on first event; HTTP clients block on connect until a frame arrives. Consider a `connected` event (like `/sse` does) or documenting the behavior — real consumers hit this too.
5. **My tool-call discipline**: one clean, small tool call at a time; never emit batched filler.

---

## f) NEXT UP TO 50 (ranked)

**Finish the shipped tier (blocking):**

1. M9: full `nix run .#build` (all modules, hermetic)
2. M9: full `nix run .#test`
3. M9: `nix run .#lint` (15 modules) + `.#check-cqrs-lint`
4. M9: `nix run .#coverage-gate` (setup must hold ≥80 with new code)
5. M9: `bash scripts/check-go-toolchain.sh` + `nix run .#check-release-train` (expect advisory lag only)
6. M10: CHANGELOG `[Unreleased]` entry (ADR-0050, setup option, recipe, demo, guides)
7. M10: final `git status` review + detailed commit (or accept daemon state and note it)
8. M10: push master + verify remote (USER GATE — see question 3)
9. Re-verify no `setup/coverage.out` or stray artifacts got committed by daemon sweeps
10. Confirm daemon didn't commit anything unrelated inside my file-sets (audit the 20 unpushed commits once)

**Tier 4 evaluation work (docs-only, cheap, demand-independent):**
11. M14: offline-sync evaluation report (sync-worker.js vs DataStar retry/replay) → route findings to ROADMAP
12. M15: adminui DataStar variant design doc (21 hx-* attrs → patch mapping, re-estimate vs ROADMAP 6h) + demand-gate decision
13. M11: dashboardui signal-patch spike (projection health as SignalsPatch; payload comparison vs polling) + go/no-go

**Tier 4 build work (demand-gated per ADR-0050):**
14. M12: dashboardui-datastar variant module (only on M11 go + demand)
15. M13: loginpage signal-based form validation (demand-gated)
16. M16: release train — cut `setup/v4` next tag via `scripts/verify-tag.sh` (carries DataStarPath); strip setup-demo's TEMPORARY replace in the same change
17. Post-train: strip the dev-replace, `check-version-drift --strict`, CHANGELOG version cut

**Quality debt surfaced this session:**
18. Fix `sse-and-datastar.md` CORS sample (`bundle.sseHandler()` unexported → use `bundle.Mount` + wrap handler)
19. Consider `connected` flush event or doc note in datastar Broadcaster.ServeHTTP
20. The 6 dual-integration consumers (InboxClean, KeyHolderAI, bank-sync, crush-daily, file-and-image-renamer) could migrate to `setup.Config.DataStarPath` or the documented recipe — worth an announcement/CHANGELOG note
21. Pre-commit hook: repoint buildflow GOCACHE off the dead mount (structural fix for the phantom golangci class)
22. e2e suite: add a setup-demo browser-level `/ds-demo` Playwright check (current e2e is Go-level only)
23. Plan HTML v2: consider a v2.1 with the ADR-0050 renumber applied textually (or leave as annotated point-in-time — my recommendation: leave)
24. Verify FEATURES.md setup-module section gains a "DataStar feed" row (I updated the datastar section; setup section row may be missing)
25. AGENTS.md: add setup DataStar option to the module description line (Key Patterns)

**Larger follow-ons (from TODO_LIST, untouched this session):**
26. ADR-001 appkit fold-in (b)-(f) — now unblocked by ADR-0050 landing
27. `/sse` endpoint-shape decision (user's call — feeds both transports symmetrically now)
28. bench-spike re-evaluation on idle machine (deferred under load)
29. CI flip of check-release-train to blocking (green advisory week done?)
30. Issue #13 (panic recovery slog attributes) follow-up verification

---

## g) QUESTIONS (cannot figure out myself)

1. **Push authorization now or after M9/M10?** 20 commits sit unpushed (incl. daemon heuristic ones and the whole Tier 1-3 wave). Earlier blanket push authorization existed for the plan docs; does it extend to this code wave — push immediately, or only after the full gate sweep + CHANGELOG?
2. **Tier 4: execute or park?** ADR-0050 gates M11-M15 on consumer demand. M0 showed real integration friction (6/15 consumers) but zero explicit panel-variant requests. Do you want the cheap docs-only evaluations (M14 offline-sync eval, M15 adminui design doc, M11 dashboardui spike) run now, or parked until a consumer actually asks?
3. **Release timing for `setup`**: the new `Config.DataStarPath` API is invisible to hermetic consumers until setup's next tag. Cut a small family train (e.g. setup v4.9.1/v4.10.0) soon, or batch with the appkit fold-in (ADR-001) into one train? Tag cutting is user-gated.

---

_Report written after session interruption; execution halted at M9 start. Waiting for instructions._
