# State of Affairs: the system/metaengine Migration Thread + Wave Posture (2-Week Delta)

> **ANNOTATED 2026-10-25** — ~~this report~~ **SUPERSEDED** by [`2026-10-12_05-23_train-lag-unblock-atomic-wave-ci-repair.md`](2026-10-12_05-23_train-lag-unblock-atomic-wave-ci-repair.md), which master-red repaired next morning (binaries untracked, coverage 90.6); the migration-plan, V007, and OQ14 threads are TODO/ROADMAP rows. Strike depth:
> supersession verdict (round-18 precedent for series-internal reports); nothing here is a
> unique open item. **ARCHIVED 2026-10-25 (round-23 (2026-10-24)).**

**Date:** 2026-10-11 19:46 CEST
**Scope:** read-only state-of-affairs session answering the operator's "what's the state of affairs since we last talked" — the last talk was the interrupted **"100% command + event-sourced via `system` + `metaengine`, backwards-safe migration plan"** session (~2026-10-04: 3 of 4 research legs completed, plan never written, session cut off). This session: verified the two-week delta across both repos, discovered master CI RED on a push that raced the 18-40 report, and harvested the vanished plan deliverable into TODO_LIST. No code mutations.
**Format note:** `.md` per the operator's standing instruction (same override as the 10-10 21-09, 18-28, and 18-40 reports; the status-report skill's HTML default is overridden, flagged per contract).
**Tree at writing:** clean, HEAD `864d3f21` (daemon capture of the 18-40 report), exactly **1 unpushed commit** — the 19-commit pile was pushed at 18:36 CEST, turning master CI red (see §d-1).

---

## The headline (one paragraph)

The migration thread moved **further in the last week than in the month before it**: the ADR-0051 checkpoint/DLQ pin that blocked the declarative `system.New` path is dissolved (adapter options + upstream ADR-0142/0149 durable checkpoints), the full 20-command usermgmt train now runs through `system.New` under the new `systemscenario` BDD harness, `dashboardui.FromSystem` gives system-first consumers the observability panels, and recommended deployment presets landed. The one remaining hard gate for "100%" — V007 cluster 1 (usermgmt's 68 `storage.SQLViewStore` findings) — was re-checked **today** and is still unmet upstream (no declarative Hydrator equivalent). Meanwhile the release train staged a second 14-tag wave (root re-versioned v4.14.1→**v4.15.0**) that is NOT cut, and **master CI is red right now** from the pushed pile (two committed binaries + a root coverage regression).

## a) FULLY DONE

| Item | Evidence |
| --- | --- |
| **systemadapter declarative path completed — the ADR-0051 pin is dissolved.** `DomainConfig(opts ...DomainOption)` now accepts `WithCheckpointStore` / `WithDeadLetterStore` / `WithHostOptions` (`systemadapter/options.go`), `system.New` itself defaults to engine-backed durable checkpoints (`system_checkpoints` Map collection, ADR-0142) when nothing is injected, `RecommendedMemoryDeployment()` / `RecommendedSQLiteDeployment()` presets ship (`systemadapter/presets.go`), and a restart-resume proof pins checkpoint semantics (`domain_config_restart_test.go`). The guide limitation section was rewritten as "Custom checkpoint / dead-letter stores" (`docs/guides/declarative-projections.md:91-123`). This was the documented blocker for retiring `NewProjectionLayer` on the declarative path. | M20–M26 execution, `docs/status/2026-10-07_02-57_SUPERB-m20-m27-session-m20-m26-done-battery-80pct-foreign-interference.md` §a; source reads this session |
| **The full usermgmt command train now runs through `system.New`** — systemadapter's test suite migrated to go-cqrs-lite's new `systemscenario` BDD harness (ADR-0153): `declarative_harness_pilot_test.go` + `declarative_harness_usertrain_test.go`. The gotcha-36 un-hermetic dev-replace is gone: `systemscenario/v4 v4.0.0` published from the proxy, requires resolved, `.#test-all` green **twice** (both Go floors). | `749ddbb5`, `e4f9784f`; TODO header 2026-10-11 entry; AGENTS gotcha 36 RESOLVED addendum |
| **`dashboardui.FromSystem` duck-typed bridge + `systembridge` telemetry phase 1** — a `*system.System` can now feed the event-store observability dashboard without any usermgmt wiring. This is the dashboardui half of OQ 14 (`setup.NewFromSystem`); the identity-bundle half stays open. | `dashboardui/system_bridge.go`, `dashboardui/systembridge/telemetry.go`; M20–M26 receipts |
| **Upstream family aligned + tag-diff audited:** `system v4.11.0→v4.12.0` (additive: Clock seams, `ResetProjection`, projection wiring — audited CLEAN, no gotcha-24 unlabeled-migration class), `scheduling v4.7.0` (ADR-0154 `WithAsyncDispatch`), `go-health v0.6.1`; strict release-train gate **865/0/0/0** including the new `examples/sync-demo` go.work member. | TODO row 55 (P18 receipts, 2026-10-11); `9fc3aeae`, `639e5958`, `3bedf6e8` |
| **Frontend Sync Protocol (ADR-0056) shipped and hardened:** root **v4.14.0 published** this morning (14-tag wave, consumers swept `cdf84122`), carrying `SyncPullHandler`/`SyncPushHandler`/sync assets v1.5.0. The new e2e battery (73/73) then caught two real v1.5.0 asset defects (cursor/backendId never persisted) — fixed with mutation-proven node:test suite (`ff36c975`, `8c29f147`, `bd38452c`). | CHANGELOG `[v4.14.0]`/`[v4.14.1]`; pushed history |
| **Fleet security floor raised Go 1.27.1 → 1.27.2** (stdlib net/http vulns GO-2026-6603/6611/6613/6617) with the flake goPkg overrideAttrs pin + golangci v2.14.0 CI export-data fix; full battery verified on both floors. | `3e2bf224`, `d06b17d8`; CI green run 38110505538 at that HEAD |
| **v5 codemod Track A: `cmd/cqrs-htmx-upgrade`** — dry-run-by-default CLI rewriting consumers off v5-removal APIs (R1 httputil re-exports, R2 go-sse shims, R3 broadcaster hub migration, R9/R10 usermgmt aliases), symbol tables **derived at run time** from the consumer's resolved module graph so it cannot drift. 10/10 tests. Unreleased — rides the staged wave's successor. | CHANGELOG `[Unreleased]`; `0f5e2bc9` era commits |
| **Cross-repo hardening M01–M26 executed** (the 308-idea study → 2026-10-06 Pareto plan): dashboardui exploit trio verified DONE-BY-DRIFT with hostile-payload tests pinned; metaengine `SortKeyCursor` delegated by all nine engine MapScans + cross-engine pagination conformance matrix; `system.New` fail-path teardown + goleak, timer lifecycle, bus publish-target validation, unknown-role rejection. | plan annotation blockquote (127 strikes); TODO row 22 |
| **Watch/upstream verdicts (P22/P23, both sessions corroborating):** V007 criterion re-checked today — still unmet; templ#1449 fix merged but **no release carries it** (retirement = first post-10-05 templ release); BuildFlow#29 closed-completed; treefmt-nix#545 still open; buildcache healthiest since the watch opened (59% used). | TODO rows 55/69/87/94 annotations; 18-28 §a, 18-40 §a |

## b) PARTIALLY DONE

| Item | State |
| --- | --- |
| **The staged second 14-tag wave** | Receipts cut (`a4f28aae`), all 14 tags verified absent local+remote (re-verified this session), sweep order + verify-tag choreography planned (18-40 §f-1), root re-versioned to **v4.15.0** (additive `AddSyncOpenAPI` family ⇒ minor, oauth2 precedent). NOT cut — blocked on (1) owner answers g-2/g-3 (18-40 report), (2) **master red repair first** (§d-1 — a red master cannot cut a clean wave). Effort to finish: S once reds are fixed + owner answers. |
| **Post-change verification battery** | `.#test-all` green ×2, fleet lint 0, strict train 865/0/0/0 — but **`.#coverage-gate` was NOT re-run** after the sibling's P11/P12 root additions and CI now reds on exactly that leg (root 88.2% < 90%, §d-1). Bench-spike still honestly refused at load 72–77 (twice-quiet rule). |
| **V007 cluster 1 — the "100%" gate** | Still gated (re-checked 2026-10-11 P22): upstream has `ResetProjection` but **no declarative Hydrator equivalent**; the 68 SQLViewStore findings stay on the legacy path. Everything else the interrupted session flagged as blocking has since dissolved. |
| **OQ 14 `setup.NewFromSystem()`** | Open since 2026-08-09. dashboardui half now answered by `FromSystem`; the identity-bundle bridge (system-first consumers adopting setup's panels/service) remains an owner build-or-reject call with no consumer demand. |
| **Round-21 archive** | Honest slice only (2 files); 27 live reports vs budget 3 — round-22 full ANNOTATE sweep owed. |

## c) NOT STARTED

- **The migration plan document itself** — the deliverable of the interrupted session, vanished for a week+: no doc in `docs/planning/` or `docs/migrations/`, no TODO row (until this session's harvest). Its research is 3/4 complete (systemadapter adoption map ✅, go-cqrs-lite system/metaengine surface ✅, migration-conventions ✅; **missing: the usermgmt internal-wiring deep-dive**). Still wanted — and cheaper now: two of its three identified blockers (checkpoint/DLQ, presets) landed since. **Harvested to TODO_LIST this session.**
- **P13** (v5 Track B data-compat goldens) and **P14** (loginpage Playwright E2E) — deferred by honest capacity call; ownership contested between the two sessions (18-40 §g-1).
- **P8/P16** (go-cqrs-lite gate compliance + teardown twin) — blocked on the foreign session's settlement. **P15** repro is DONE out-of-tree (the doc-only classifier hole is fail-open by construction: a failed `git diff` + `\|\| true` ⇒ "nothing staged"); the upstream-ask doc is drafted but unwritten to `docs/proposals/upstream/`.
- **Toolchain-floor runbook page** (06-39 e-1) — carried over a third consecutive session.

## d) TOTALLY FUCKED UP!

1. **Master CI is RED right now (run 38156167640, failure, pushed 18:36 CEST) — two independent causes, both from the pushed 19-commit pile:**
   - **`checks` job — large-file guard:** two daemon-captured `go build` outputs were COMMITTED and are now in pushed history: `examples/sync-demo/sync-demo` (12.1 MB, `f3de578e`) and root `cqrs-htmx-upgrade` (3.7 MB, `0f5e2bc9`). CI log: `FAIL: cqrs-htmx-upgrade is 3717771 bytes (limit 1048576)` + `FAIL: examples/sync-demo/sync-demo is 12126726 bytes`. The 18-40 session DISCOVERED these at 18:40 and queued cleanup as wave step 0 — but the pile had already been pushed at 18:36, four minutes earlier; its report's "nothing pushed by either session today after `48b118e0`" was stale at publication. Severity: blocks the wave cut; severity for history: the blobs are permanent in history (mitigation: remove going forward; do NOT rewrite pushed history). Fix: `git rm --cached` both + `.gitignore` entries, one commit.
   - **`test` job — root coverage gate:** `Root coverage: 88.2%` < 90% threshold. The sibling's P11/P12 root surface (sync OpenAPI metadata + builder primitives) landed without a coverage-gate leg in the final battery. Fix: tests for the new surface or an explicit threshold adjudication — not silent.
2. **The interrupted session's deliverable silently evaporated.** A full research investment (3 agent legs) produced zero artifact: no plan doc, no TODO row, nothing — for 7+ days. This is the third recorded instance of the "answer-then-capture failure" class (18-28 §d-5 being the second). This report closes it by harvesting the plan as a TODO row.
3. **The "wave blocked on GCL dedup" misdiagnosis cost most of a day** (owned in full by 18-40 §d-1: a BLOCKED claim inherited without evidence; the lint stage is hermetic and was never red). Corrections to the three blocker-carrying docs are still owed (18-40 §b-3).

## e) WHAT WE SHOULD IMPROVE!

1. **The pre-push battery must include `.#coverage-gate` whenever the root package surface grows.** Today's `test`-job red is exactly the skipped leg: the 18-28 battery ran tests + vet + lint + docs + train (all green) but not coverage — while adding ~2 files of new root API. Concrete fix: add a coverage-gate line to the wave-cut checklist (release-playbook §1 train-preflight already runs the hermetic battery; teach it or the checklist to refuse a root-surface-growing push without a coverage verdict).
2. **Reports claiming push/tree state must re-verify at publication time.** The 18-40 report asserted "nothing pushed today" about a push that had landed 4 minutes earlier — the same freshness class as its own d-2 resume-ritual finding, mirrored at close-out. One `git log origin/master..HEAD --oneline \| wc -l` before writing the "Tree at writing" line makes every report's anchor true.
3. **Endorse (second occurrence) the 18-40 e-1 rule: BLOCKED claims carry the same evidence bar as DONE claims.** Two sessions, two days, one misdiagnosis each inherited at face value. This is now a pattern — the AGENTS-note candidate from 18-40 e-2 ("flake goApp lint/test are HERMETIC; gopls workspace diagnostics and script comments are not gate truth") should land with the next AGENTS touch.

## f) Top things to get done next (ranked by impact)

| # | Task | Impact | Effort | Status |
| --- | --- | --- | --- | --- |
| 1 | **Repair master red (i):** `git rm --cached examples/sync-demo/sync-demo cqrs-htmx-upgrade` + `.gitignore` entries; never build binaries into the tree | Critical | S | routed (18-40 f-1 step 0) |
| 2 | **Repair master red (ii):** root coverage 88.2%→≥90% — tests for the sync-OpenAPI/builder surface, or explicit threshold adjudication | Critical | M | **NEW — harvested** |
| 3 | **Cut the 14-tag wave post-repair** with root **v4.15.0** re-version (owner g-2/g-3 answers; CHANGELOG restructure; sweep + strict gate) | Critical | M | routed (18-40 f-1) |
| 4 | **Write the backwards-safe system/metaengine migration plan** (`docs/migrations/` v4-to-v5-adjacent shape; complete the missing usermgmt-wiring research leg; gates: V007 criterion, OQ 14, experimental labels) | High | M–L | **NEW — harvested** |
| 5 | **P5 tail:** `RecorderChain` integration_test half after health v4.13.0 publishes in the wave (gotcha 37 clears) | High | S | routed |
| 6 | **Correct the three blocker-carrying docs** + the stale `train-preflight.sh:24` comment (one docs+script commit) | Med | S | routed (18-40 f-4) |
| 7 | **Dedup sweep when GCL's held tag drops:** bump require + adapt `usermgmt/es_dedup_benchmark_test.go:29` to explicit instantiation in the same sweep | Med | S | routed (18-40 f-8) |
| 8 | **jsonv2 map-order ETag churn:** `Spec.JSON()` deterministic-emit option or construction-time canonicalization — restart-stable `OpenAPISpecHandler` ETags | Med | S | **NEW — harvested** (18-28 §d-5 filed it nowhere) |
| 9 | **Round-22 annotate sweep** (27 live reports, budget 3) + P15 upstream-ask doc to `docs/proposals/upstream/` | Med | M | routed (18-40 f-3/f-9) |
| 10 | **P13/P14** (v5 Track B goldens; loginpage Playwright E2E) — resolve ownership first (18-40 g-1) | Med | L | routed, contested |
| 11 | **P20 owner-call batch** (PapDashboard SEND, datastar-demo keep, E005 filing, GCL records, T12/T13, OQ28, OQ11, module-rename names) | High (owner) | S | routed (18-40 f-5) |
| 12 | **Bench-spike** at a verified-quiet window (load < 6 twice) | Med | S | routed |
| 13 | **Toolchain-floor runbook page** | Low | S | carried ×3 |
| 14 | **Annotation-gate README fixture case + JS static gate** (18-28 §d-6 / 18-40 f-15) | Low | S | routed |

## g) Top question I can NOT figure out myself

**The wave sequence, given the new red master:** both prior sessions claim the wave cut (18-40 §g-1), two owner calls are unanswered (confirm root **v4.15.0**; cut now vs hold for GCL's dedup tag — with "hold" now strictly worse since the pile is already pushed and red), and the repair (f-1/f-2) must precede any cut. Confirm the sequence — *repair reds → re-version v4.15.0 → cut* — and who executes it (this session on your word, or the claimed sibling queue). Secondary (answerable here, no urgency): whether you want the deferred 100% migration plan (f-4) written before or after the wave.

---

**Truth anchors:** CI run 38156167640 (`checks` + `test` failed; log lines quoted in §d-1; `gh run view` this session); `git log origin/master..HEAD` = 1 commit; origin/master at `4e666ed2` (18:30:45); binaries on disk `ls -la` (12,126,726 + 3,717,771 bytes); all 14 staged tags absent local (`git tag -l`) and remote (`git ls-remote`); systemadapter surface read from source (`options.go`, `presets.go`, `domain_config_restart_test.go`, `declarative_harness_*_test.go`); go-cqrs-lite tags `systemscenario/v4.0.0`, `system/v4.12.0`, `scheduling/v4.7.0` present; ADRs 0152–0157 exist in `../go-cqrs-lite/docs/adr/`; V007 re-check = TODO row 87 (P22, 2026-10-11); prior-session claims cross-read from `docs/status/2026-10-11_{10-58,18-28,18-40}_*.md`.
