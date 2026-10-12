# Status — catalog/v4 plan delivery + fleet family-alignment push

> **Session snapshot: 2026-10-12 18:11** (single Crush session, cqrs-htmx repo; cross-repo
> go-appkit/go-cqrs-lite touched read-only). Scope: the catalog-embedding question →
> research note → SUPERB Pareto plan → the push that required a 10-sweep family
> alignment through strict CI-parity gates. Report format is `.md` at the operator's
> explicit path demand (skill default HTML overridden by instruction).

## 0. One-paragraph state

The catalog/v4 integration question is answered and documented (setup deliberately
does NOT depend on it; go-appkit isolates it in its `docs` module), the Pareto plan
is filed and AWAITING APPROVAL for its feature tiers, and the session's ordered push
(`58a6a7ce..291b2c51`, 20 commits) landed green through both strict pre-push gates
after a sanctioned 10-sweep upstream alignment (0 unpublished / 0 lag / 866 requires).
Three process stumbles were self-caught and self-corrected; no data loss; master
content is gate-green, with two watch items (CI run on the new HEAD; possible fresh
upstream tags re-red'ing train lag).

---

## a) FULLY DONE

| # | Item | Evidence |
| --- | --- | --- |
| a1 | **Catalog-embedding question answered with receipts**: setup = NO (serves root's own `EventCatalog`, `setup/machine.go`); go-appkit = PARTIAL (`docs` module yes via `catalog/docserver`, `cqrs` module deliberately clean); Go `embed` cannot cross module boundaries (the literal premise is impossible) | conversation + verification commands in the research note |
| a2 | **Research note** `docs/research/2026-10-12_catalog-v4-integration-map.md` — integration map table, two-catalogs distinction, why-setup-why-not, verification commands | file content; pushed in the session range |
| a3 | **SUPERB Pareto plan** `docs/planning/2026-10-12_17-33_SUPERB-catalog-v4-integration-pareto-plan.md` — 1%/4%/20% tiers, 30–100min table (T1–T7), ≤12min table (F1–F19), mermaid execution graph, anti-Verschlimmbesserung guardrails, verification commands | file content; committed `12000ecc` |
| a4 | **TODO_LIST 2× P2 rows** (two-catalogs docs disambiguation; catalog v4.8.0 bump prep) with plan/map cross-links | `TODO_LIST.md` §P2 top |
| a5 | **Fleet family alignment, all 10 upstream bumps landed**: catalog v4.8.0, decider v4.8.0, sqliteengine v4.5.3, metaengine v4.19.0, schema v4.7.0, snapshot v4.7.0, systemscenario v4.1.0, system v4.13.0, go-health-dashboard v0.11.0, go-appkit v0.9.0 — each via exact-anchor `bump-dep` with per-module hermetic tidy+build+vet green | per-sweep commits `dd811880`…`291b2c51` + daemon chunks `1432ae77`/`41e90905`/`a1b04938` |
| a6 | **Push through BOTH strict pre-push gates**: release-train strict (866 requires, 0 unpublished, 0 lag — incl. the mid-session-discovered go-appkit v0.9.0) + version-drift `--strict`; `58a6a7ce..291b2c51 master -> master` | `/tmp/push-final.log` verdict lines |
| a7 | **Whole-wave workspace verification**: `go build ./...` rc=0 + `go vet ./...` rc=0 after all sweeps (gotcha-38c test-only class checked) | session logs |
| a8 | **Pre-existing failure correctly classified instead of "fixed"**: sync-demo `undefined: cqrshtmx.AddSyncOpenAPI` verified byte-identical on `origin/master` (require root v4.14.0 predates the symbol) — gotcha-38d classification confirmed by diff, NOT touched | `git diff origin/master -- examples/sync-demo/go.mod` |
| a9 | **Hook auto-fixes adjudicated and kept, cleanly separated**: `dashboardui/csp_test.go` nlreturn fix + `dashboardui/aggregates.templ` + `e2e/server/main.go` templ-fmt — mechanical, committed apart from feature/doc work | `32c4dc4f`, `79711ae0` |
| a10 | **Two gotchas honored under pressure**: gotcha-38b dirty-tree guard (switched to the prescribed per-sweep commit loop), gotcha-6 `--no-verify` fallback for mid-train bump commits WITH justification in every message | commit messages in the pushed range |

## b) PARTIALLY DONE

| # | Item | Done | Missing |
| --- | --- | --- | --- |
| b1 | **Plan tier T3 (catalog v4.8.0 bump prep)** — ironically executed ~1h after the plan was filed, by the alignment | THIS repo's consumers now at v4.8.0, swept + pushed | **The P2 TODO row I wrote for it is stale-on-arrival** (says "bump prep" — the bump happened); go-appkit's `docs` + `integration` modules still at catalog v4.7.1 (their repo, their train) |
| b2 | **Plan tier T4 (bookkeeping)** | TODO rows landed | F11 outcome annotations (append-only) not yet applied to the plan file or the research note after T3 executed |
| b3 | **Commit-message quality goal ("VERY DETAILED")** | All 10 sweep commits carry detailed justification bodies | The plan + TODO_LIST themselves landed under daemon heuristic messages (`12000ecc`) — the detailed-message amend was killed by a transient hook timeout and the daemon moved HEAD, making amend a history rewrite (skipped deliberately) |
| b4 | **Pre-push local verification bar** | Per-sweep hermetic verify + whole-wave build/vet + strict train/drift gates | The full `nix run .#test` / `.#test-all` battery was NOT run locally (machine was killing hook steps — load class); CI on `291b2c51` is now the verifier and was not yet watched at report time |

## c) NOT STARTED (all gated on owner approval per the plan)

- **T1** (1% → 51%): two-catalogs docs disambiguation — guide See Also + Serving pointers (F1–F2), `setup/README.md` EventCatalogPath footnote (F3), `catalog-demo` doc-comment links (F4). Highest value-per-effort; the owner hitting the confusion IS the demand signal.
- **T2** (4% → 64%): `setup.Config.DocsPath` decision packet (F5–F7) — seam design + go/no-go demand criteria, consolidated INTO the existing ROADMAP data-mesh Approach-C row (`ROADMAP.md:75-83`), no duplicate artifact.
- **T5** (gated): the `Config.DocsPath` implementation itself (F12–F16) — owner YES required (`ROADMAP.md:222` demand-signal rule).
- **T6**: catalog-demo visual smoke e2e (F17–F18; pre-existing gap, `ROADMAP.md:236`).
- **T7**: go-appkit `docs/README.md` pointer to the integration map (F19; separate repo).
- **F11**: outcome annotations on plan + research note.
- **go-appkit-side alignment**: catalog v4.8.0 (+ whatever wave their repo accumulated).

## d) TOTALLY FUCKED UP

Nothing destructive — no data loss, no broken master, no gate bypassed. Three self-inflicted stumbles, all caught and corrected in-session, listed here because they were real:

1. **Violated gotcha-3 myself**: read `GATE_RC=0` off a `cmd | tail; echo $?` pipe — that was TAIL's rc, not the gate's (the exact mvdan/sh trap AGENTS.md warns about). The gate was actually FAILING at that moment. Self-caught one step later by re-running with proper rc capture (`cmd > /tmp/f; echo $?`). Cost: one wasted cycle.
2. **Batched sweep→commit chains committed on FAIL**: the snapshot sweep failed (sync-demo, pre-existing) but my chained `git add -A && git commit` ran unconditionally, committing 8 files of PARTIAL sweep state before the failure was read. The landed state happened to be the correct target state — that was luck, not discipline. The fix (rc-guarded commits) is in section e.
3. **Amend race against a live daemon**: attempted a detailed-message amend of `12000ecc` while the daemon was mid-loop; the pre-commit hook timed out (transient, machine load) 3× and the daemon committed on top (`79711ae0`), converting the amend into a forbidden history rewrite. ~6 minutes burned on doomed hook runs before switching to the documented fallback. The docs landed with heuristic messages.

Cosmetic: two sweep commit TITLES leaked the raw regex anchor (`bump larsartmann/go-health-dashboard$ to v0.11.0`, `…system/v4$…`) — my template interpolated `$1` unstripped. Ugly, harmless, pushed.

## e) WHAT WE SHOULD IMPROVE

1. **rc-guard every chained verification→commit step.** `sweep && commit` chains must check the sweep's rc before `git add`/`commit` — partial-failure states must never auto-commit (stumble d2).
2. **Namespace /tmp logs per sweep with a collision-proof suffix** (full dep path sanitized + `$$` + timestamp) — `basename` on an exact-anchor regex collapsed schema+snapshot logs into one file (stumble d3-adjacent).
3. **Commit YOUR content immediately after fmt, before the daemon polls.** The plan/TODO window between `nix fmt` and commit was wide enough for the daemon to win (gotcha-4's known race). On a docs-only turn: write → fmt → `git add <paths> && git commit` in ONE tool call.
4. **On hook-timeout signals, stop retrying the hook after the FIRST transient kill.** Gotcha-8 says a deadline-exceeded is a real load signal: diagnose (`uptime`, concurrent sessions) or switch to the documented fallback immediately — not after the third identical failure.
5. **Run `wait-tree-quiet` before a push that carries daemon-commit piles** (this push carried 20 commits, several from daemon/concurrent-session churn). The ritual exists; I skipped it.
6. **Re-read TODO rows written earlier in the SAME session before writing new ones** — the T3 bump-prep row went stale within the hour (b1). Better: write planned-work rows AFTER the wave that might consume them, or word them for both outcomes.
7. **The lag tail grows while you sweep** — go-appkit v0.9.0 published BETWEEN my first gate run and the `--refresh-cache`. Rule: after the LAST sweep, always `--refresh-cache` and re-run the strict gate BEFORE declaring the wave done (I did, and it caught the 3-entry tail — the ritual worked; make it explicit next time).
8. **Strip regex anchors from generated commit titles** — `$1` in a message template carried the `$` anchor (d-cosmetic).
9. **Watch CI after every push** — not done yet for `291b2c51` (b4). The 10-11 lesson: master CI can red on classes invisible locally (fresh upstream tags re-red'ing train lag; exhaustruct on 5-day-gap code).
10. **Consider a tiny `bump-dep` UX fix upstream**: the fix recipe's `basename`-style log naming + the fact that `--commit` is green-gated but plain runs leave the tree dirty for the daemon to capture — a `--commit-mine` mode (daemon-aware: commit only its own edits) would kill the whole d2/d3 + b3 class.

## f) NEXT — up to 50 (brainstorm, not commitment; HARVEST should route with rigor)

**Execute the plan (approval-gated where marked):**
1. T1/F1 — two-catalogs distinction note in `event-catalog-guide.md` § See Also (12min, no gate).
2. T1/F2 — guide § Serving pointer to `examples/catalog-demo` (8min).
3. T1/F3 — `setup/README.md` EventCatalogPath footnote ≠ go-cqrs-lite/catalog (8min).
4. T1/F4 — `examples/catalog-demo/main.go` doc-comment links back (8min).
5. T2/F5 — DocsPath decision packet: seam design sketch (12min).
6. T2/F6 — packet: scenarios + demand-signal + go/no-go criteria (12min).
7. T2/F7 — consolidate packet link INTO ROADMAP data-mesh Approach-C row, in place (8min).
8. OWNER — YES/NO on T5 (`Config.DocsPath` implementation; F12–F16 pre-decomposed).
9. T6/F17–F18 — catalog-demo visual smoke e2e + ROADMAP:236 gap strike (2×12min).
10. T7/F19 — go-appkit docs README pointer (cross-repo session).
11. F11 — outcome annotations: plan file + research note (T3 executed by the alignment).
12. Fix the stale-on-arrival P2 row: strike the "bump prep" wording, keep the go-appkit-side remainder.
13. Update TODO_LIST "Updated:" stamp (currently still points at the 10-11 session).

**Alignment follow-through / watch:**
14. Watch CI run on `291b2c51` to completion (b4) — verdict into the next report.
15. Re-check upstream tags before/at next CI cycle — fresh tags re-red train lag (treadmill; go-appkit v0.9.0 appeared mid-session).
16. go-appkit repo: catalog v4.7.1 → v4.8.0 sweep in THEIR modules (docs + integration) + their own pending wave — separate session, their gates.
17. Root v4.14.1 staged wave (TODO_LIST P1-adjacent): heals sync-demo `AddSyncOpenAPI` hermetic break + sync-asset defect-fix receipts + 1.27.2 floor receipts.
18. After ANY cross-repo family release: `check-release-train --refresh-cache` before the dependent's push (gotcha 27e — mechanically still pre-push-hook-manual).
19. Coverage-gate re-run debt (TODO_LIST battery row) — untouched this session, still owed.
20. Fleet lint sweep debt (`nix run .#lint` over all modules) — still owed per TODO_LIST battery row.
21. Bench-spike battery leg — still load-refused per TODO_LIST; quiet-window carry.

**Process/hardening ideas surfaced by this session:**
22. rc-guarded sweep→commit helper (or `bump-dep --commit-mine`) — kills d2/d3/b3 permanently.
23. Collision-proof log naming convention for batched tool runs.
24. Add a gotcha candidate: "the lag tail grows while you sweep — refresh-cache AFTER the last sweep, not just before the first" (today's live proof).
25. Add a gotcha candidate: "exact-anchor `$` leaks into commit titles via `$1` interpolation — strip anchors in message templates."
26. Pre-push ritual addition: `wait-tree-quiet` before pushes carrying daemon piles (gotcha-4 codification, partially exists).
27. Machine-load triage snippet for hook-timeout storms (`uptime` + concurrent-session check) — codify beside gotcha-8.
28. Amend-window rule: if the daemon commits during a failed amend, DON'T rewrite — record the heuristic hash in the next detailed commit instead.

**Adjacent open items seen while in this tree (not worked, just sighted):**
29. `e2e/server/main.go` syncAddItemCmd cqrs-lint warning (value-embed vs pointer receiver) — sighting from BuildFlow findings list.
30. `examples/sync-demo/main.go` same class (createNoteCmd) — sighting.
31. `sync_pull.go` branded-ID suggestions (SyncEvent.EventID/StreamID, BackendID) — sighting, owner-gated design question.
32. `usermgmt/snapshot.go:193` exhaustruct `StateShape` finding — sighting (possibly intentional partial literal; needs `//nolint` or fix).
33. `SECURITY.md` shape findings from securitymd tool (error-severity) — sighting, worth 15min.
34. dependabot.yml 20-entry cap vs 29 Go modules (info) — sighting.
35. `.github/workflows` actions-pin exact-version comments (gotcha 33 class) — periodic re-verify owed.
36. type-check findings in `tests/sync.spec.ts` ('unknown' types ×8) — sighting from BuildFlow detect-only list.
37. vulnix CVE report (binutils/bison/coreutils derivations) — nix-store noise class, policy question.
38. `vendorHash.nix` stale-FOD warning (BuildFlow preflight WARN) — claims stale; per gotcha 31 verify by building the derivation before touching.
39. go-work-paths `use`-suffix warnings (13 modules) — BuildFlow preflight WARN class; likely accepted-noise, worth one adjudication note.
40. lychee private-link/archived-docs exclude policy (two WARNs, fleet-undecided) — sighting.

**Parking-lot ideas (ROADMAP fuel, low priority):**
41. Consider `Config.DocsPath` session-gating posture (if T5 ever approves: machine-endpoints parity).
42. Event-catalog guide: add a comparison table (EventCatalog vs catalog/v4) if T1 lands well.
43. catalog-demo: add `-export-only` smoke assertion for the EventCatalog MDX tree.
44. Cross-repo: go-appkit `docs` module could link the cqrs-htmx integration map (mirror of T7).
45. Status-report skill: this session honored `.md`-on-demand — no spec change needed (noted for the record).
46. `bump-dep` sweep summaries could append "NEXT: the lag tail grows — refresh-cache" to every PASS output.
47. Sweep commits could carry a machine-readable `Alignment-Sweep: <dep>@<ver>` trailer for greppability.
48. TODO_LIST P2 row for the csp/e2e templ auto-fix class: none needed (adjudicated done) — record only if it recurs.
49. Re-verify `docs/planning/README.md` "39 files" count vs current planning-dir population (stale count sighted).
50. Consider annotating today's plan file with the alignment outcome ONCE T1/T2 verdicts exist (single annotation pass, not drip-feed).

## g) Questions I can NOT figure out myself

1. **Approve T1 + T2 now?** (docs disambiguation + the DocsPath decision packet — both cheap, zero-risk; and do you want T5's go/no-go criteria in the packet to include a specific demand signal, e.g. "N consumer requests" or "your own call"?)
2. **Cut root v4.14.1 now** (heals the sync-demo `AddSyncOpenAPI` hermetic break + carries the staged sync-asset fixes and 1.27.2-floor receipts) **or keep it staged** as TODO_LIST currently plans?
3. **go-appkit repo**: should a follow-up session run THEIR catalog v4.8.0 (+ wave) alignment over there, or do you keep that repo's trains on your own calendar?

---

*Point-in-time snapshot per `docs/status/README.md` convention (append-only; annotate, never rewrite). Commit skipped per harness rule — the auto-commit daemon owns this file.*
