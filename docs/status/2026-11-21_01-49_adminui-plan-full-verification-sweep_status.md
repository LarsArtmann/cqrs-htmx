# Status: adminui SUPERB plan — full verification sweep, drift repair, and sibling-lane triage

**Date:** 2026-11-21 01:49 CET
**Session scope:** verify every receipt of [`docs/planning/2026-11-14_08-45_SUPERB-adminui-tenant-user-management-next-wave-pareto-plan.md`](../planning/2026-11-14_08-45_SUPERB-adminui-tenant-user-management-next-wave-pareto-plan.md) (M00–M17, all logged DONE 2026-11-14) against the current tree — six days of sibling waves (K2 idempotency, templ-1118, v1.27.2 family train, legalparty-model, fleet CI heals) had landed since the log was written — repair what this lane owns, and name precisely what belongs to siblings and the owner.
**Predecessor:** the plan's own §9 execution log (final aggregate row, 2026-11-14) and the fleet lane's 01:34 verdict-wave report (its "still-open e2e green proof" is quoted verbatim below).

---

## a) FULLY DONE (verified, with receipts)

1. **Full-plan verification sweep: every M00–M17 receipt re-verified green.** Static receipts confirmed on the tree: `roleCheckboxGroup` (components.templ + both surfaces), `edge_states_test.go` (5 tests), `admin-tenant-access.spec.ts`, `admin-a11y.spec.ts`, `admin-screenshots.spec.ts` + the 8 committed PNGs, `handler_import.go` + `import_export_test.go` (literal routes, 4 MiB cap, bounded error panel), `lastSeenIndex` (`handler_users.go:35`), `attribution.go`/`addedByIndex` (`handler_tenants.go:100`, `handler_members.go:92`), tenant switcher + `tenant_switcher_test.go`, `tenant_polish_test.go`, gotcha 60 (`docs/gotchas.md:243`), README import/export section, the D1–D3 decision sheet, the M14 archive receipts, the corrected `AddMember` doc comment. The §9 row [`2026-11-20/21 FULL-PLAN VERIFICATION SWEEP`] is the receipt of record.
2. **`check-go-toolchain` GREEN** — go.work floor 1.27.3 ≤ toolchain 1.27.3.
3. **`check-release-train --refresh-cache --strict-lag 0` GREEN** — 0 unpublished, 0 train lag across 1049 internal requires (re-verified twice this session: once at 20:4x, once by the fleet lane's treadmill receipt).
4. **Both CSS class-set gates GREEN** — adminui 1075 tokens + dashboardui 1051 tokens, both bundles canonical (form + content gates).
5. **`check-cqrs-lint` 26/26 GREEN after repairing the one drift finding.** The K2 idempotency wave (2026-11-19, sibling lane) introduced the only finding in the whole workspace: `usermgmt/command_idempotency.go:91` C026 "Idempotency TTL mismatch". Triaged at source and ruled a FALSE POSITIVE: the argument is the sweep interval (`idempotencyMemoryStoreSweep`), the store has no TTL constructor parameter (upstream signature `NewMemoryStore(sweepInterval time.Duration)` verified in the module cache), and the replay TTL constant `DefaultCommandIdempotencyTTL` IS used at the `CheckAndRecord` claim 13 lines below. Inline suppression carries exactly that receipt (own-line form, survives golines 120-col — the trailing form would have been rewrapped, gotcha 7 class). Gate re-run rc=0; no stale-suppression findings.
6. **`check-erraudit-strict` advisory GREEN, TOTAL=364 reproduced ×2 on an idle machine.** First reading (TOTAL=348, exit 1) was taken while coverage-gate + test batteries + check-modules ran concurrently — load-induced noise, discarded after the two clean reproductions. AGENTS row trued same-change: growth past the 352 receipt RESTARTS the advisory window, now closing 2026-12-21.
7. **`coverage-gate` PASSED, all 16 modules above threshold** (tail: datastar 100/90, huma 95.6/90, setup 93.7/80, systemadapter 89.5/70, health/auditlog 100/90, gfl-usermgmt 92.9/85).
8. **adminui test batteries GREEN both legs** — hermetic (`GOWORK=off`, 7.4s) AND workspace (7.3s); usermgmt hermetic GREEN (19.5s).
9. **e2e chromium: the admin spec family GREEN** (behavior + tenant-access + a11y) inside a 125-passed run over the current tree. The 2 failures are the fleet lane's DECLARED open proof (see §d).
10. **docs-gates `annotate` CLEAN** (215 files, 1508 struck, 0 findings) after fixing the one rule-F violation this lane introduced (see §d item 3 for the fix), and `check-docs-freshness` (links) PASSED.
11. **All three docs ratchets GREEN same-change**: corpus ledger sentence appended to the manifest + `CORPUS_BASELINE` re-trued to the measured 63,819; `check-docs-corpus-budget --ratchet`, `check-docs-tail-budget --ratchet`, `check-conflict-markers` (2035 files) all ✓.
12. **Freshness harvest for the then-newest report (20-25 round-2 CI-heal)**: the three unharvested items it flagged are now routed — TODO_LIST P2 row (gosec.sarif artifact from run 44776015013; dgraph NixOS-VM-test connection-refused localhost:9080) and a session-start-runbook paragraph (the sibling-lane input check, §f25 healed in place: `nix run` consumes the working tree's flake.lock, so a green local run under modified flake inputs is not a CI-parity receipt).
13. **CHANGELOG rule-F reword** — "`MemoryStoreOption`s" → "`MemoryStoreOption` values" (the annotate gate's only finding; the battery's rule F bans alnum touching a backtick, plurals included).
14. **TODO_LIST hygiene respected** — one additive P2 row, no `[x]` rows, no section imbalance introduced.

## b) PARTIALLY DONE

1. **`check-modules` composite: the two reds this lane owned are repaired, but no full green re-run was captured.** My first aggregate read 3 of 106 FAILED: (a) `erraudit-strict-advisory` — load-induced (see §a6), green idle ×2 after; (b) `corpus-budget-ratchet` — the sibling sessions' growth had outrun their own true-ups; repaired by this session's ledger + re-true (§a11); (c) `branching-flow` — NOT repaired by choice (§c2). The tree never settled long enough for an honest full-composite green: three sibling sessions + the daemon landed work continuously through the sweep (20:26 → 01:37).
2. **e2e full-suite green: admin family proven, suite overall 125/127.** The 2 remaining failures are sibling-owned (§d1/§d2). Firefox advisory battery NOT re-run this session — the M05 verdict (66 protocol errors, Playwright↔Nix-firefox mismatch, zero product assertions) stands as recorded 2026-11-14.
3. **docs-gates battery: 3 of 5 green at write time** (routing PASS, annotate CLEAN, freshness-links script PASSED); freshness (harvest matcher) now points at the NEWEST report — the sibling 22-18 legalparty execution report, whose §f is their close-out; `cadence` (16 live vs threshold 15) and `audit-all` (owner-gated drain-or-re-pin) stay red for their owners. This report adds one live report with `TAIL_BASELINE` 15 → 16 same-change.

## c) NOT STARTED (deliberate, with reasons)

1. **M16 D1–D3 follow-on waves (impersonation / sessions / bots / rename)** — owner-gated, unanswered per CURRENT_FOCUS 2026-11-19 ("per-row M2M D1–D4 / adminui D1–D3 — the packets stay parked; each decision surfaces individually through the native decision tool"). Nothing to do without the tick.
2. **branching-flow +11 re-pin** — the findings are the LIVE legalparty/fleet lanes' unreviewed in-flight code (the count grew +2 → +11 while this session watched). Pinning another session's unreviewed work is the recorded anti-pattern (TODO_LIST art-dupl row: "THEIR settle pass re-pins"). Left deliberately.
3. **Archive pass for the cadence red** — `docs/status/` + baselines were being actively re-trued by the sibling lanes (README.md and TAIL_BASELINE written minutes before each of my probes). The archive ritual on a live sibling surface is the gotcha-47 class. Left to the next docs-health round.
4. **go-cqrs-lite CI latent work (gosec.sarif fetch, dgraph VM-test root-cause)** — upstream work, owner-gated filing; harvested to TODO_LIST P2 (§a12), not executed.
5. **`TestSnapshotShape_SQLStoreDropsStampToday` workspace-mode flip** — proxy-verified storage v4.12.0 still does not exist upstream (latest v4.11.1, 2026-10-17); the TODO row's explicit instruction is DO NOT flip until the tag ships. The workspace `-race` red remains the sentinel working as designed; hermetic leg green.

## d) TOTALLY FUCKED UP (this session's own failures)

1. **Hours of wall-clock drift inside the sweep.** The session started 2026-11-20 ~20:26 and its gates finished ~01:37 — three sibling sessions and the daemon moved the tree repeatedly mid-verification (HEAD advanced through the legalparty report, the treadmill receipt, the oauth2 v4.13.1 hotfix, the v1.27.2 family train, the ADR-0049 accent adaptation). Every long gate's result had to be re-derived against the new tree state; the plan §9 row and this report pin the final tree. Cost: several re-runs, one discarded erraudit reading. No work was lost (gotcha 47 did not bite this time), but the session ran far hotter than the task needed.
2. **Trusted a load-contended gate reading once.** The first `check-erraudit-strict` reading (TOTAL=348, exit 1) raced coverage-gate + two test batteries; had it been recorded as the receipt, the AGENTS row would now carry a fabricated 348. Caught by the ×2 idle reproduction discipline — which should have been the FIRST step, not the recovery.
3. **Launched the e2e run without reading the flake app's arg semantics.** `nix run .#e2e -- tests/admin-*.spec.ts` was intended as a 3-file filtered run; the app ran a materially larger set (125 tests). The outcome was still the right evidence (admin family green, suite state sampled), but the scope was luck, not intent.
4. **First `cqrs-lint` gate reading was incomplete.** The initial background run's `tail -6` showed only the module list and looked green-ish; the FAIL verdict surfaced only on the synchronous re-run with grep. The gate's finding-shape output (gotcha 61) deserves an always-capture-to-file discipline from the first invocation.
5. **Interpretation of "THE WHOLE TODO LIST" was assumed, not confirmed.** This lane read it as "the pasted adminui plan, verified end to end" — the repo's TODO_LIST.md itself contains mostly owner-gated, parked, or cross-repo rows that repo policy forbids starting without ticks. The reading was correct per policy, but stating it up front would have saved ambiguity.
6. **No fresh status report until this one.** The runbook's finish checklist wants the receipt surfaces landed before the session ends; the plan §9 row carried the receipts for hours before this report existed.

## e) WHAT WE SHOULD IMPROVE

1. **Run `PREFLIGHT_EXPECTED_BRANCH=master bash scripts/tools/preflight-tree-check.sh` at session START, not ad-hoc** — it names dirty-tree/foreign-in-flight states mechanically instead of the manual ps/mtime probing this session improvised (twice).
2. **Never run advisory gates concurrently with heavy batteries.** erraudit, cqrs-lint, and the docs battery are cheap and fast — sequence them in a quiet window and reproduce ×2 before recording any total.
3. **One writer per worktree needs teeth in shared files.** TODO_LIST.md, CHANGELOG.md, README.md, and the baselines were all co-edited with live siblings this session; the daemon absorbed everything correctly, but each shared-file edit was a silent race. The preflight tool warns; it does not serialize.
4. **Read flake app wrappers before assuming CLI passthrough** (one `rg` on the app's `text` would have settled the e2e scope question).
5. **The erraudit window-restart semantics deserve a checklist line**: any receipt above the last one restarts the window — recording a total without idle reproduction can silently burn a month of blocking-promotion progress.
6. **The corpus ledger "measure-once, sentence LAST" discipline worked** — keep the order: ledger sentence inline → measure → pin → gate. Doing the pin first is the +N-unledgered class the manifest documents four times.

## f) UP TO 50 THINGS WE SHOULD GET DONE NEXT

**This lane's direct tails:**
1. Capture a full `check-modules` composite green on a settled tree (the §b1 gap) — requires the sibling lanes to settle first.
2. Re-pin `branching-flow` +11 after the legalparty/fleet lanes' settle pass (their §f20/§g3; the amendment rows belong in `docs/analysis/triage-decisions.md`).
3. Heal the e2e suite to full green: the fleet lane's two open reds (§d refs below) + `check-e2e-ran-count.sh` on a fresh report.
4. Re-run the firefox advisory battery after the fleet lane's CSP/console-guard fix lands (M05 posture: advisory until 3 clean runs).
5. Archive round: drain `docs/status/` to ≤15 live (cadence red at 16, this report makes 17 — the next docs-health round must archive BEFORE adding anything).
6. Re-check `docs-gates freshness` after the legalparty lane harvests its own §f (their 22-18 report is the newest; my 3 items from the 20-25 report are already routed).
7. Record the D1–D3 answers here + TODO_LIST when the owner ticks arrive (per-row native decision tool per CURRENT_FOCUS).

**Owner-gated (blocked, listed so the costs stay visible):**
8. D1 impersonation panel surface + guardrails (published API already exposes `BeginImpersonation`/`EndImpersonation` — plan erratum).
9. D2 batch of four usermgmt asks (bot listing, batched membership counts, tenant rename, session listing/revocation) — one tag train.
10. D3 next-wave priority call (bots vs sessions+impersonation vs import-export depth vs hardening).
11. M2M D1–D4 ticks (the W1/W2 waves stay parked).
12. audit-all drain-or-re-pin (419 unharvested vs pin 268; policy forbids raising without an owner tick).
13. `ReadLastN` SQL-store upstream ask + S55 packet chain (D6: tag → swap → CI gate).
14. samber-do `InvokeAs` + go-health `RegisterRoutes` upstream filings (verify-before-filing gates).
15. architecture-gate secret configuration (drop the `continue-on-error`) — owner call.
16. Repository branch-protection decision (master currently unprotected).
17. ExtraNav (a)/(b) owner decisions (URL-scheme rejection; `aria-current="page"`).
18. authztest external-consumer waves (C27 leftovers: Swetty unlocatable, crm row stale).
19. StateShape sentinel flip + storage pin sweep + go-cqrs-lite#117 strike — waits on storage v4.12.0.

**Fleet-lane coordination (theirs, watched):**
20. The two e2e reds: `loginpage-golden` (golden drift) and `setup-bundle` console guard (CSP inline-style block on the login page — `style-src` nonce violation) — the 01:34 fleet report names this as their open proof.
21. Their branching-flow +11 adjudication (§c2).
22. Their legalparty §f harvest (freshness driver).
23. Their cadence/archive round (§c3).
24. The oauth2 v4.13.1 hotfix tag already landed (e3f937a6 in integration_test) — verify the fleet P1 row's remaining verify-lines.
25. templ-components PR-#101 release watch → family train → retire adminui's vendored popover CSS layer.
26. erraudit window: hold TOTAL ≤ 364 through 2026-12-21 or the window restarts again.

**Small, unblocked, worth doing in any quiet window:**
27. `check-e2e-ran-count.sh` self-test fixture refresh if the spec count moves (125 now).
28. Sweep the session-start runbook's new sibling-lane paragraph for link freshness after the next runbook edit.
29. Verify the K2-era `DefaultCommandIdempotencyTTL` (10 min) against the admin-demo ack recipe docs (one read, one true if stale).
30. Confirm `usermgmt/CHANGELOG.md` [Unreleased] carries the C026 suppression receipt (in-code reason exists; the CHANGELOG line is optional posture).
31. adminui axe coverage for audit/dashboard/import pages (harvested §f19 tail, still open in TODO_LIST).
32. Candidate gotcha 62 (stale-proxy demo binaries) formalization (harvested §f23 tail).
33. e2e cached-revision warning in the `e2e-firefox` flake app (TODO row (3), still open).
34. serial-context note in `e2e/README` (TODO row (5), still open).
35. Battery timing budget measurement (TODO row (6)).
36. `check-e2e-ports` promote-or-drop decision (TODO row (7)).
37. sync.spec (e) firefox SharedWorker investigation (the one open divergence).
38. loginpage Playwright WebAuthn ceremony E2E + full-page render golden (test-depth debt).
39. dashboardui page-size selector + browser-level smoke (harvested tails).
40. `BuildFlow` checkout reconciliation watch (cross-repo, owner-gated).
41. Fleet release-dance helper design note (§f20 of the 22-25 report).
42. `ephemeral-pg.sh` cd-guard + script sweep (go-cqrs-lite §f13/§f14, upstream).
43. README-links contract decision (§f7/§f8, upstream, owner-gated).
44. `shfmt` tug-of-war repoint (§f29, fleet).
45. otel Dependabot alerts manual-close evidence (§f28).
46. CI paths-filter cost review for go-cqrs-lite (§f30, after their green lands).
47. Quarterly README↔FEATURES claim-sweep (next due ~2027-01-17 — early only if a drift is spotted).
48. `nolint-audit` CI wiring (round-42 tail, owner-gated posture).
49. Skill-maintenance gates: authored evals + description optimization (harvested tail).
50. Crush-config `references/lessons.md`: the load-contended-advisory-gate lesson (cross-project; edit there, not here).

## g) THREE QUESTIONS I CANNOT FIGURE OUT MYSELF

1. **D1–D3 (adminui next wave)** — impersonation behind opt-in config / batching the four usermgmt asks into one tag train / priority order: the packets are pre-staged with recommendations, and nothing gated can start without your ticks. Do you want any of the three answered this week, or do they stay parked past the erraudit window (2026-12-21)?
2. **The two open e2e reds** — `loginpage-golden` and the `setup-bundle` CSP console guard are named "still-open" by the fleet lane's 01:34 commit, but their session's last visible action was 90+ minutes ago. Should THIS lane take the fix over (it touches loginpage/setup — anti-goal 9 territory for the adminui plan, but not for a fresh mandate), or keep waiting for the fleet lane to land its proof?
3. **audit-all at 419 unharvested vs pin 268** — policy forbids raising the pin without an owner tick, and archiving provably drains nothing. Do you authorize a drain pass (annotate items inside archived files, each with evidence — hours of mechanical work), or re-pin to the measured truth with a dated owner reason?

---

## Verification receipts (commands, this session)

| Receipt | Command | Verdict |
| --- | --- | --- |
| Toolchain | `nix run .#check-go-toolchain` | ✓ 1.27.3 ≤ 1.27.3 |
| Train | `nix run .#check-release-train -- --refresh-cache --strict-lag 0` | ✓ 0 unpublished, 0 lag |
| CSS ×2 | `nix run .#check-css-bundle-classes` + `.#check-css-bundles` | ✓ 1075+1051 exact, 2 canonical |
| cqrs-lint | `nix run .#check-cqrs-lint` | rc=1 (C026 FP) → suppressed → rc=0, 26 modules |
| erraudit | `nix run .#check-erraudit-strict` ×2 idle | TOTAL=364, rc=0 both |
| Coverage | `nix run .#coverage-gate` | PASSED, 16/16 above threshold |
| adminui | `GOWORK=off go test ./...` + workspace `go test ./...` | ok 7.4s / ok 7.3s |
| usermgmt | hermetic `go test ./...` | ok 19.5s (workspace `-race` = documented sentinel) |
| e2e | `nix run .#e2e -- tests/admin-*.spec.ts` | 125 passed, 2 failed (fleet lane's open proof) |
| Docs ratchets | `check-docs-corpus-budget --ratchet` + tail + conflict-markers | ✓ / ✓ / ✓ (2035 files) |
| docs-gates | `annotate .` + `check-docs-freshness.sh` | CLEAN / PASSED |
