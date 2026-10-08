# Status — fixture consolidation, upstream wave alignment, branching-flow ratchet hardening, CI reds repaired

**Session window:** 2026-10-08 ~22:10–2026-10-09 00:06 CEST · **HEAD at write time:** `0017cc35` (pushed; CI watch in flight) · **Machine:** load swung 6 → 30 → 67 (another session running gates concurrently all night; bench-spike + race-test legs honestly refused) · **CI:** the 4739234b run red'd on Shellcheck (foreign self-test SC2030/31) — fixed and pushed; definitive watch on `0017cc35`

---

## a) FULLY DONE (this session)

### 1. dashboardui test-fixture consolidation (P2 row closed)

`mustTestDashboardWithConfig` moved to `testsetup_test.go` with the auto-fill rule (Config sets neither EventSource nor Journal → one fresh in-memory store serves both, so a test's Config states only what it deviates by); new `mustTestDashboardMuxWithConfig` adds the `/dashboard/` mount; `newTestDashboardMux` became its zero-arg vanilla case. **29 near-vanilla sites converted** (sse_replay ×4, handlers_security ×12, dashboard_test ×13); construction-failure tests (SSESubscribeAllFails, RequiresAtLeastOneInterface) and stub-journal/stream-reader fixtures stay custom on purpose. Suite green in workspace AND `GOWORK=off` hermetic modes; lint 0. Commit `70b1289e` (recomposed from two mid-edit daemon sweeps — the daemon raced every phase tonight).

### 2. Upstream wave alignment BEFORE the push — the fresh-tag race, avoided this time

`check-release-train --refresh-cache` revealed templ-components **v1.21.0** + go-retry **v0.8.0** published upstream AFTER master's last green CI run — pushing without alignment would have red'd CI exactly like the morning incident (the gotcha-27(e) class, lag direction). Executed the documented sequence: R18 pre-flight clean on both tags → read both CHANGELOGs from the module proxy (v1.21.0's consumer-relevant change: the **omit-empty nonce rule** — inline scripts no longer emit `nonce=""`; nonce-carrying renders byte-identical) → templ-components prefix sweep (26 files) + go-retry sweep → **dashboardui's two page goldens regenerated** (the diff was EXACTLY the 8 `<script nonce="">` → `<script>` changes, verified line-by-line before regenerating — the goldens pinned the old empty-attribute form) → **both CSS bundles rebuilt in the same change** per the family-bump rule; class-set gate green (1016 tokens exact). Commits `cc3e7f0a` + `43efc564` (recomposed after the daemon raced bump-dep's commit).

### 3. CI reds on the pushed tree — both repaired at root cause

- **checks job: Docs freshness** — the templ-components adoption section still claimed "uniform at v1.20.1" after the sweep; the gate did exactly its job. Claim updated to v1.21.0 + adoption receipt (`bfb391c5`).
- **checks job: Shellcheck** — the CONCURRENT session's new `train-preflight.sh` self-test carried 42 SC2030/SC2031 hits (their fixture exports env vars inside `( cd )` subshells intentionally — the false-positive-by-construction shape). They fixed the checker script itself in `d9a1f6b5` but not the self-test; I added the file-level `# shellcheck disable=SC2030,SC2031` with the reason recorded, shellcheck clean, self-test 6/6 (`0017cc35`). Verified their file was quiet 14+ minutes and the tree clean before touching a foreign file.

### 4. branching-flow: zero-candidate guard + counts (R12/T8), baseline re-pin 666→229, keep-all-14 measurement (R15/T14)

- **Guard:** the checker now prints the tool's Baseline counts in EVERY verdict line and REFUSES rc=0 runs that detect ZERO candidates against a non-empty baseline (`+0 added, -N removed, ~0 modified, =0 unchanged` = analyzer misfire wearing a green coat). +2 fixture cases; self-test 12 green; verified against the real binary (`baeffce1` + `c5d223de`).
- **Re-pin:** the local ratchet was silently RED — 59 new findings had accumulated since 2026-10-05 (gate is local-only; CI self-skips; nobody had run it). The 2026-10-05→08 churn (M17–M26 capability seams, dedup rounds, fold modernization, templ-components alignment) dissolved **−449** findings; the +59 new ones are the same primitive-string branding-advice class already rejected in the ledger — accepted under existing verdicts, not re-litigated. Baseline 666 → **229**; ledger amendment row written; gate green (`+0 added, ~205 modified, =19 unchanged`).
- **Subset measurement:** signal fraction ~2.6% across all 14 analyzers; verdict **KEEP ALL 14** (the ratchet already neutralizes the noise; DUPLICATE_TYPE and FLAG_PARAM each produced one real fix — removing their tripwires buys nothing). Three-day churn 663→666→229 recorded as baseline-volatility context. `ab23e247`, README § Analyzer-subset measurement.

### 5. A012×4 inspection (plan M12) — re-derived at 0, retired with evidence

The 2026-10-01 4-finding inventory does NOT regenerate under the current system binary (swept all 13 gate modules) — the third confirmed cross-binary inventory flip (after B024, sentinel_concrete_type). Code-level inspection done anyway: **all four folds already handle their deletion event** — `FoldUser`←`EventUserDeleted` (`identity-model/fold.go:138`), `FoldMembership`←`EventMemberRemoved` (:255), `FoldTenant`←`EventTenantDeleted` (:300), `FoldBot`←`EventBotDeleted` (:341) — with `Deleted` state + `IsActive` guards in place since `3e6580af` (2026-07-23), predating the inventory. The old binary's ×4 were collector blind spots. Verdicts in the residual-triage doc; TODO row struck.

### 6. M15 + M13 verified/closed; docs bookkeeping

- **M15 (v5 cut runbook skeleton):** ALREADY LANDED 2026-10-05 by a concurrent session — `docs/runbooks/release-v5-cut.md` is the complete ready artifact (six-wave deletion plan, per-class consumer notes, standing + v5-specific gate checklist, OQ11 precondition). Row struck with the verification receipt.
- **M13 (BuildFlow failing-step capture):** the named-steps section existed (2026-10-04→05); appended the 2026-10-08 addendum — three live failures under load 19–36, the concurrent session's hook-budget fix `7505cb85` (13/13 sweep commits bypassed the gate that day), markdown-only commits now pass rc=0 live-verified.
- Root CHANGELOG § Fixed gained the family-walk-fix receipt (`23b54cc2`, CI-behavior gate); TODO rows struck: httputil release, fixtures conversion, A012, M15, R12/T8, R15/T14; header restamped. Cross-repo fresh-tag race documented in AGENTS gotcha 27(e) + release-playbook §3a + the multi-train-family rule + block-require scope limit in `docs/runbooks/dependency-train-bump.md` (`b2a12066`).

---

## b) NOT DONE (honest refusals + blocked)

1. **Bench-spike + `.#test-all` race battery + coverage-gate re-run:** load swung 6→30→67 all night (the concurrent session's gate runs). A machine-pinned bench baseline under load is a false measurement (gotcha: never re-pin under load); race tests under load 67 flake on timing. Deferred to a quiet window — the main `.#test` battery DID run green earlier at load ~7 before the push.
2. **The 3 owner questions from the 22-02 report remain unanswered** (unanswered ≠ forgotten): (1) go-cqrs-lite edit authorization for GCL legs (b-remainder)+(e)+11 lint findings; (2) cut `server_timing/v1.0.2` for the LICENSE; (3) BuildFlow pre-commit policy — NOTE: lane (3) was partially addressed tonight by the concurrent session's `7505cb85` (honest hook budgets); `.buildflow.yml` skip-list tuning is still open.
3. **Concurrent-session in flight:** `train-preflight.sh` + flake app + CI step + playbook docs landed mid-session (their session). The pre-push `--refresh-cache` candidate from my 22-02 report §e2 appears to be what they built. Their AGENTS.md edits and mine coexist; no collisions after the self-test fix above.

## c) NEXT (impact-ordered)

1. Confirm CI green on `0017cc35` (watch in flight at write time).
2. Battery legs at a quiet window: `.#test-all`, coverage-gate re-run (the 10-04/06 wave paths are still unmeasured), bench-spike when load < ~4.
3. The 3 owner-decision lanes above.
4. `docs/feedback/new/` sweep if anything landed; docs-health tail annotation for the 22-02 + this report when their items age out.

---

## g) QUESTIONS

Same three as the 22-02 report (§g), unchanged — no new blockers discovered this session.
