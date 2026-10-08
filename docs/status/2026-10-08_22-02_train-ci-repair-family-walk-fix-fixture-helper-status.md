# Status — patch train shipped, self-inflicted CI lag repaired, family-walk false positive fixed at the source

**Session window:** 2026-10-08 ~19:50–22:05 CEST · **HEAD at write time:** `07b2b54a` (daemon sweep of the fixture-helper edit; 1 commit ahead of origin) · **CI:** master GREEN 18/18 on `254c3919` (the train + repair push) · **Load at write time:** ~19–36 all session (bench-spike honestly refused) · **Disk:** `/mnt/buildcache` 92% (19G free; no fallback needed)

---

## a) FULLY DONE (this session)

### 1. P1: the 2026-10-07 untagged-debt patch train — shipped end to end

- **Debt verification first:** scanned every tagged module's delta vs its latest tag. Found EXACTLY the row's three modules (identity-model `v4.12.1..HEAD` 5 files, dashboardui `v4.13.1..HEAD` 69 files — the M17–M26 SUPERB surface was riding untagged, setup `v4.14.1..HEAD` 8 files). usermgmt v4.14.2 + adminui v4.12.3 turned out to be ALREADY published (my first scan used stale baselines — see d2). loginpage/datastar/health/auditlog/systemadapter: zero drift.
- **Hermetic pre-tag proof:** `GOWORK=off` build + vet + test green for all three modules against published pins (the gotcha-30 root-tag-gap catch; setup's login-CSP test passes against published root v4.13.3).
- **Receipts (CHANGELOG-before-tag):** identity-model `[v4.12.2]` section (fold `slices.Clone` modernization + fold-aliasing invariant doc + `TestUserID_ParseBrandPrefix_Policy` D13 pins + id/v4 v4.7.0→v4.7.2, snapshot/storage-memory v4.6.2, cbor v2.9.6 alignment); dashboardui `[Unreleased]` renamed to `[v4.13.2]` (content already covered M17–M26 + the asset-serve/overview dedup — verified the 69-file delta against the section); root CHANGELOG § Fixed gained the setup v4.14.2 receipt (setup keeps no module CHANGELOG by repo structure) INCLUDING the retro receipt for usermgmt v4.14.2, which shipped receipt-less on 2026-10-07 (its `[Unreleased]` read "nothing yet").
- **Truth-pass on the receipt:** my first draft described a "bus-recovery drain fix" — WRONG. Diffing `setup/v4.14.1..HEAD` showed the real content: watermill `EventRecovery` middleware on the default bus (panicking subscriber → logged + Infrastructure-family error, opt-out via `Config.EventBus`) and the login page's inline scripts now carrying the CSP nonce (`NonceFromRequest`). Also fixed on sight: setup's recovery comment claimed "Corruption error" while the code wraps `WrapInfrastructure` — comment corrected (and em-dash removed), hermetic build re-proven. Receipt rewritten to the verified content.
- **Wave-ordered tags + full consumer alignment:** `verify-tag.sh --dry-run` → `--push` for identity-model/v4.12.2, then `bump-dep.sh 'larsartmann/cqrs-htmx/identity-model/v4$' v4.12.2` (11 modules PASS: hermetic tidy + `go mod verify` + build + vet each), cache refresh, commit; same for dashboardui/v4.13.2 (setup, integration_test, e2e/server) and setup/v4.14.2 (async-startup-demo, setup-demo — these two my own dependency map had missed; the train gate caught them, see d4). `check-release-train --strict-lag 0`: **0 unpublished / 0 lag at 842 requires after every wave**. Master pushed through the strict pre-push gates at `f1b20f8e`.
- **Verification:** per-module `GOWORK=off go build/vet/test` green pre-tag (incl. the suite for all three), train gate green, push gates green.

### 2. P2: httputil doc fix released as v1.4.2

- Read the repo's state (HEAD = `a891f0c`, the 7-line `KeyExtractorFromRemoteAddr` caveat, plus two daemon commits: `server_timing/LICENSE` + a flake vendorHash line). Verified the caveat's actual wording against the diff before receipting it (my first receipt phrasing invented a "proxy pooling" claim the doc does not make — corrected to the real semantics: per-TCP-connection keying, churn = fresh buckets, per-request connections effectively unlimited, pointer to `KeyExtractorFromClientIP`).
- Renamed `[Unreleased]` → `[1.4.2]` (ships the whole backlog: the Metrics validate-and-log panic fix, zero-value MiddlewareStack pin, doc truth-pass batch, the caveat, LICENSE) + fresh `[Unreleased]`. Committed, tagged `v1.4.2` (annotated), pushed; **proxy-resolve verified live** (`go list -m github.com/larsartmann/httputil@v1.4.2` → resolves).

### 3. The self-inflicted CI red — root-caused and repaired

- CI on `f1b20f8e` went RED (module-architecture → "Release train check (blocking)"): **my own httputil v1.4.2 release** (20:55) lagged every workspace module pinned v1.4.1 before cqrs-htmx master pushed at 20:59. The local pre-push gates passed only because the 15-min `/tmp/cqrs-htmx-tag-cache` hadn't seen the fresh cross-repo tag — a NEW class: gotcha-27's TTL race, but cross-repo (dependency release racing the dependent's push).
- First repair attempt (`bump-dep.sh 'larsartmann/httputil$' v1.4.2`) was **aborted by the R18 family-consumability pre-flight with a FALSE POSITIVE**: it demanded `httputil/server_timing` at v1.4.2, but that submodule is on its own v1.0.x train.
- **Fixed the checker at the source** (`scripts/checks/check-family-release-consumable.sh`): the family walk now demands each submodule's OWN required version — read verbatim from the parent's require line — instead of the parent's tag. The templ-components v1.20.0 poison class is still caught (placeholder versions are carried verbatim into the fetch, which fails; submodule-content placeholders now label the submodule's own train). Fixture self-test extended with three offline `file://`-stub-proxy cases (multi-train pass / missing-version fail / deep-placeholder catch): **12/12 green**. Two fix iterations were needed (see d5/d6).
- Live proof: pre-flight `httputil@v1.4.2` rc=0 (was abort) → `bump-dep` swept 22 modules to v1.4.2 (all PASS) → train 0/0 → commits `23b54cc2` (tool fix) + `254c3919` (wave 4) pushed through strict pre-push → **CI green** (watched to `completed/success`).

### 4. P2: SUPERB bookkeeping batch (TODO row) — closed

- All **15 consumed research ideas struck** in `docs/research/2026-10-06_dashboardui-metaengine-system-improvements.md` with per-idea outcome notes + executing-session pointers (38→M21 Routes(), 42→M20 variadic Autodetect, 75→obsolete-on-verification (demo EXISTS — verified), 76/77/78→02-57 doc fixes, 263/265/274/275→M26 telemetry probes, 281/301→M22 systemadapter options, 293/294/308→M20/M25 duck-typed seam/systembridge).
- `02-57` report **§g annotated** with a dated `> ANNOTATED 2026-10-08` blockquote recording the applied defaults: (1) the push window opened root-tag-first (v4.13.2 assets → v4.13.3 CSP → today's train), (2) the foreign asset-API session landed and is published, (3) the post-daemon compile guard stayed manual.

### 5. TODO_LIST hygiene — 4 rows closed with receipts

- Patch-train row (DONE — this session, with the corrected module inventory), SUPERB bookkeeping row (DONE), Docs & memory micro-batch row (CLOSED — all sub-items were done 2026-10-01), CHANGELOG-receipt-convention row (DECIDED 2026-10-05 A8.3, commit `64b113b6` — verified the decision exists in AGENTS gotcha 20 before striking).

---

## b) PARTIALLY DONE

### 1. dashboardui test-fixture config-variant conversion (P2 row)

- **Analysis complete:** `mustTestDashboardWithConfig` already exists (`handlers_write_test.go:493`); classified every candidate site — sse_replay ×5 (EventBus/heartbeat variants, no Mount), handlers_security ×13 (XSS seeds ×3, SeekableJournal stats/pagination ×2, ReadOnly ×1, pure-vanilla asset/404 ×3, middleware-only ×2), dashboard_test ×12 (pure-vanilla Mount ×3, seeded Mount ×5, variant-store Mount ×2, no-Mount probes ×3). Deviating-by-design classes (handlers_coverage stub-journal, csp/fmt dashboards) stay custom per the row.
- **DONE:** `mustTestDashboardWithConfig` now auto-fills the vanilla store wiring when a Config sets neither source (deviation-only Configs), `memorystorage` import added, hermetic vet green. The daemon swept this edit as `07b2b54a` (unpushed).
- **REMAINING:** add `mustTestDashboardMuxWithConfig` (Mount-twin returning `(*Dashboard, *http.ServeMux)`); convert the ~24 classified sites; drop the resulting dead imports; run the dashboardui suite in workspace AND hermetic mode; commit.

### 2. TODO_LIST row hygiene for THIS session's work

- The httputil-release row and the dashboardui fixture row are not yet struck (the release itself is done; the row strike is pending the fixture completion so one commit can close both honestly).

---

## c) NOT STARTED (all still open in TODO_LIST)

1. A012×4 inspection → per-finding verdicts into the residual-triage doc.
2. v5 cut runbook skeleton (plan M15) from `docs/guides/v5-removal-inventory.md`.
3. BuildFlow failing-step-name capture (plan M13) → agents-notes.
4. branching-flow per-module candidate-count proof (plan R12/T8) — gate must print counts, fail on zero.
5. branching-flow analyzer-subset tuning (plan R15/T14) — two-run stability + FP measurement.
6. loginpage test-depth debt (login.js zero tests; property test + goldens + Playwright WebAuthn E2E).
7. Credential 4× duplication decision memo (feeds OQ28).
8. Battery remainder: `.#test-all`, `.#coverage-gate` re-run (2026-10-04/06 wave paths unmeasured), bench-spike (REFUSED this session — load 36–86; needs a quiet window).
9. GCL legs: (b) cqrs-lint/coverage runs + the captured 11 lint findings, (e) `fail()`/`Close()` teardown unify — cross-repo.
10. CHANGELOG receipt for the family-walk checker fix (`23b54cc2`) — CI-behavior gates get receipts per gotcha 20; NOT yet written. Should ride the next docs commit.

---

## d) TOTALLY FUCKED UP (radical honesty)

1. **Receipt written from memory of the TODO row, not from the diff.** The first root-CHANGELOG setup-v4.14.2 entry claimed a "bus-recovery drain fix"; the actual delta was EventRecovery middleware + CSP nonce passthrough. Caught because I diffed before committing — but the draft should never have existed. The same pass caught a comment/code family mismatch ("Corruption" vs `WrapInfrastructure`) which I fixed in setup.go.
2. **First module scan used stale tag baselines** (assumed usermgmt v4.14.1 / adminui v4.12.2 were latest) and briefly reported 5 untagged modules; a tag listing corrected it to the row's 3. Cost: one wasted scan cycle, zero damage.
3. **My own dependency map missed the setup consumers** (grep pattern missed indented/single-line require forms + examples), so the first "final" train check came back with 2 lag entries. The gate caught what my map missed — the gate worked as designed; I nearly trusted a hand-rolled scan over it.
4. **Amend vs daemon race:** the corrected-receipt amend went through the full BuildFlow pre-commit (90s under load 36) and FAILED on the 12 known environment-class steps (license-check unpassable class, go-generate/govulncheck timeout-killed) — then the daemon swept the staged content into `03f9c7cb` anyway, so the amend degraded to a message-only rewrite (`d382f19f`). Net effect fine, but I should have gone straight to the documented `--no-verify` fallback.
5. **Fixture regression used single-line require form**, which the family walk's block-require grep (pre-existing scope) never matches — both new cases false-passed until I switched the fixture to the block form the real httputil go.mod uses.
6. **`IFS= read -r path subver` disables field splitting** — the version never landed in `subver` and the walk kept demanding the parent's version. Classic; should have unit-tested the parsing line before the full self-test run. Fixed with plain `read -r path subver`.
7. **CI watcher v1 polled short SHAs** (`headSha=="f1b20f8e"` vs the API's full 40-char) → 11 empty polls before I killed it and re-watched with the full SHA.
8. **Wave-1 history split into three commits** (`aa69eef0` + daemon tails `deeb31f7`/`2a8af1d7`) because the daemon raced the `git add -A`. I chose not to rebase mid-stack under a live daemon — correct call, but the wave would have been one clean commit with `bump-dep --commit`.

---

## e) WHAT WE SHOULD IMPROVE

1. **Receipts from diffs, always:** the verify-before-writing habit caught both fabrications this session (setup receipt, httputil caveat phrasing). Make it mechanical: no receipt text without a `git diff <oldtag>..HEAD` read in the same breath.
2. **New gotcha candidate (cross-repo fresh-tag race):** releasing a DEPENDENCY (httputil) can lag the workspace between its proxy publication and any cqrs-htmx push — the pre-push gate reads a 15-min cache and green-lights a state CI then rejects. Candidate: `check-release-train --refresh-cache` inside the pre-push hook, or a documented "after any family dependency release, refresh before push" step in the release playbook.
3. **bump-dep `--commit` by default for train waves:** avoids the daemon-race split history (d8) and its message is already rc-checked.
4. **The family-walk grep only matches block requires** — single-line `require x v1` submodules are silently skipped. Pre-existing scope; either extend the grep or document the limit in the checker header.
5. **BuildFlow pre-commit under load is a 90-second tax that deterministically fails** (12 env-class steps). Either budget-tune the failing steps in `.buildflow.yml` or codify "docs/tooling commits → `--no-verify` with the step list" so sessions stop paying the tax.
6. **Fixture self-tests should stub the proxy via `file://`** (as now done in cases 8–10) — the network-path logic of gates was previously untested offline; this pattern is reusable for verify-tag/bump-dep self-tests.
7. **Trust gates over hand-rolled scans** (d3): my ad-hoc dependency map was wrong in exactly the way the runbook warns; the map is a convenience, the train gate is the truth.

---

## f) NEXT (up to 50, impact-ordered)

**Immediate tails of this session**
1. Finish the dashboardui fixture conversion (add `mustTestDashboardMuxWithConfig`, convert ~24 sites, both-mode tests, commit; then strike the TODO row).
2. CHANGELOG receipt for the family-walk checker fix (CI-behavior gate, gotcha 20).
3. Strike the httputil-release TODO row (work done; receipt + release + CI-green evidence in hand).
4. Push `07b2b54a` + the fixture commit; watch CI.
5. Add the cross-repo fresh-tag race to AGENTS gotcha 27 (or a new gotcha) + release-playbook §3a note.
6. Amend `docs/analysis`-style docs: record the family-walk multi-train rule in `docs/runbooks/dependency-train-bump.md` (R18 pre-flight section).
7. A012×4 inspection → residual-triage verdicts (bounded analysis).
8. v5 cut runbook skeleton (M15) — pure docs from the removal inventory.
9. BuildFlow failing-step capture (M13) — dry-run + tee, append the named steps to agents-notes.
10. branching-flow per-module candidate-count proof (R12/T8) — checker prints counts, fails on zero, fixture case, flake app wiring, atomic checklist.
11. branching-flow analyzer-subset measurement (R15/T14) — two-run diff + FP rates from triage-decisions, narrow-or-not verdict.
12. Credential 4× duplication memo (R11/T7) — read-only analysis, input to OQ28.
13. loginpage test-depth: Base64URL property test + serializeAssertion/serializeAttestation goldens via `node --test` (the Playwright WebAuthn E2E is the bigger half — maybe phase it).
14. Battery legs at a quiet window: `.#test-all`, `.#coverage-gate` re-run (wave paths unmeasured), bench-spike only when load < threshold.
15. GCL legs (b): run cqrs-lint + coverage on the M04–M10 surface (read-only; capture verdicts).
16. GCL legs (e) + the 11 lint findings — needs owner go-ahead (see g1).
17. server_timing LICENSE: decide whether to cut server_timing v1.0.2 so the proxy copy carries the license (see g2).

**From TODO_LIST (unchanged, still open)**
18. Wire `check-cqrs-lint` into CI (blocked: nested-module tag decision D6).
19. Fleet cqrs-lint binary swap (D4; local verification done 2026-10-01, steps in packet §8).
20. D13 owner tick (memo + pins are closed; KEEP-generic recommendation stands).
21. PapDashboard reply send (owner channel).
22. D1–D12 decision-table ticks as the owner rules on them.
23. SidebarNav revisit when templ-components ships a dark-token shell (criterion 1 re-check on next UI change).
24. DataStar Tier 4 (demand-gated; no signal yet).
25. BuildFlow `go-version-auto-configure` re-enable after BF1+BF2 upstream.
26. GCL pre-commit "Doc-only" misclassification repro (M7).
27. scorecard timing variance env investigation (M22, low).
28. VCS-cache junk-vs-corruption classification (03-45 table item 22).
29. Foreign-session lint debts: handlers_events cyclop, accent_color mnd (03-45 items 21).
30. httputil/server_timing proxy checksum anomaly in BuildFlow env (03-45 item 20).

**Housekeeping / candidates surfaced this session**
31. TODO_LIST header "Modules: 28" + coverage/lint lines are stale vs today's train — next docs-health pass should refresh the header block (22:05 state: CI green on `254c3919`, tags +3).
32. `docs/status/` this report should be harvested by the next docs-health round (annotations + archive per convention).
33. Consider a `check-release-train --refresh-cache` step in the pre-push hook (see e2) — small, mechanical, kills the fresh-tag class permanently.
34. `mustTestDashboardWithConfig` auto-fill semantics deserve a line in dashboardui's test README/AGENTS section if one exists (fixture contract discoverability).
35. The two mid-stack daemon commits (`deeb31f7`, `2a8af1d7`) still carry heuristic messages — note for the next rebase-free history pass (or accept; they are unambiguous bumps).
36. Loginpage coverage-gate re-pin (quiet-window micro from the episode-4 tail).
37. Roadmap: capture the "multi-train family" lesson for templ-components-style releases in the family-release-consumable README section.
38. `.buildflow.yml`: evaluate budget/timeout floors for go-generate/govulncheck under load (e5) — needs a load-representative measurement run.
39. Confirm the ship-state of `docs/planning/2026-10-08` SUPERB "train trust & cadence" plan (`93230e16`, another session) — cross-link today's outcomes if it tracks train cadence.
40. CHANGELOG [Unreleased] "Fixed" section: today's entries (train receipt, checker fix once receipted) are already correctly placed — verify at next docs-health sweep that nothing else claims the same receipt.

---

## g) QUESTIONS (cannot figure out myself)

1. **GCL authorization:** the go-cqrs-lite repo holds the remaining hardening legs — (b) cqrs-lint/coverage gate runs, (e) the `fail()`/`Close()` teardown unify, and 11 golangci findings (gci ×10, gocognit ×1). Cross-repo DOCS were explicitly owner-gated in the TODO, but code fixes happened in the M-wave. Do I have the go-ahead to edit + commit (daemon rules) in go-cqrs-lite for (e) + the lint findings, or are those still foreign-session territory?
2. **server_timing release:** the LICENSE file for `httputil/server_timing` is committed but its proxy copy (v1.0.1) predates it. Cut `server_timing/v1.0.2` so license scanners resolve it from the proxy, or leave the submodule untouched until its next real change?
3. **BuildFlow pre-commit policy:** the 12 deterministic env-class failures (license-check unpassable class; go-generate/govulncheck timeout-killed under load ~36) cost ~90s per commit before the documented `--no-verify` fallback. Should I tune budgets/skips in `.buildflow.yml` (affects every session, and gotcha 35 warns about config churn), or is `--no-verify`-with-step-names the standing answer?

---

_Annotated per status convention: absence of a strike/marker = open. Evidence hashes inline. Written 2026-10-08 22:05 CEST._
