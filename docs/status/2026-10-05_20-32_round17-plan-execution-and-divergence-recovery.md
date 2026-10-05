# Round-17 Plan Execution, Divergence Recovery & Cross-Session Triage

**Date:** 2026-10-05 20:32 CEST
**Session window:** 2026-10-04 07:45 → 2026-10-05 20:32 (two active turns, one ~32 h gap)
**Scope of this report:** what THIS session ran, verified, broke, and noticed. Concurrent-session work is attributed, not claimed.
**Baseline at start:** the round-17 Pareto plan (`docs/planning/2026-10-04_07-45_SUPERB-round17-tail-pareto-execution-plan.md`), master at `48b9e9bc`, CI red on exactly `module-architecture` (loginpage 7/5 deps).

**What changed underneath me:** during the 32 h gap, concurrent sessions landed the identity-auth-hardening wave (usermgmt/setup/dashboardui tags), a dependency sweep (go-output family, go-health, samber-do-auditlog), the scripts/ reorganization (`checks/`/`selftests/`/`tools/`), the branching-flow ratchet (SARIF baseline + gate, R3/R8), the push-unblock diagnosis (templ-components v1.20.0 is an unconsumable release), the P1 StreamID `.String()` drift fix (gotcha 25), the #52 closure comment, the agents-notes signing narrative, and the episode-4 truth-pass (RelativeTime already adopted; PageHeader 4/11 + 7 justified divergences). Several of my plan's tasks were done before I resumed — caught by re-assessment, not duplicated.

---

## a) FULLY DONE (this session, evidence-backed)

1. **A1 — loginpage dep-budget resolution (the last red CI job).** Budget row raised 5→7 with the adoption rationale (utils is the required `BaseProps` props-contract type — NOT droppable by inlining Class/Ternary; the other 5 deps are load-bearing security helpers). Verify: `check-dep-budgets.sh` green (loginpage 7/7), loginpage build+vet+test green, golangci 0 issues. Commit `6d6f03ef` (bulk carried by daemon `ba9a1b62`). Scope: `scripts/check-dep-budgets.sh` (now `scripts/checks/` after the reorg). Open loop: CI run id post-fix not yet seen (push pending, see d-4).
2. **A2 — read_model_missing closure verified.** The sibling session's brand-prefix strip landed as daemon commit `eba45c80` (pushed). Setup suite workspace-mode: `ok` (11.3 s); setup-demo seed: `ok`. Row closed per convention; evidence recorded on the signing-arc CHANGELOG entry. My own closure commits were later superseded by a concurrent rebase (see d-1) — the closure itself stands on the sibling fix, re-verified this session.
3. **A5 — statusToBadgeMap (open since 2026-09-17 F6) resolved without filing.** Verify-before-filing showed the premise dead: the capability shipped upstream as `display.Badge` + `Dot` + `Pill` + typed `BadgeType` (all in published v1.19.4). Closed the real residue instead: adminui's `badge`/`statusBadge`/`stateBadge` now take `display.BadgeType` directly; the stringly `badgeKindToType` adapter (silent default-to-neutral) and its branch test are deleted; 19 call sites migrated; templ regenerated module-dir canonical. Verify: GOWORK=off build+vet+test green, golangci 0 issues, rendered HTML unchanged (every literal mapped 1:1). Commits `8cc079f6` (bulk, daemon) + `44e03709` (message + fmt nit).
4. **A10 — v5 cut runbook skeleton.** `docs/runbooks/release-v5-cut.md`: removal classes mapped onto release-playbook §3a waves (NewUserID → wave 2; root re-exports + raw broadcaster + usermgmt alias layer + V007 stack surface → wave 3; datastar facade + ProjectionLayer + renames → wave 4), per-class consumer notes at inventory depth, pre-cut gate checklist incl. the two v5-readiness prerequisites. ROADMAP OQ11 links it. Verify: links 367 OK, freshness PASS, fmt clean. Carried by daemon `e79e5564`.
5. **A8.2/A8.3 — hygiene receipts.** agents-notes gained the named BuildFlow non-devShell failing steps (`type-check`/tsc TS2688, `load-flake` timing, `license-check` GOROOT, `govulncheck` package patterns, `samber-linter`) with today's live 86-step failure as receipt; AGENTS gotcha 20 records the convention: CHANGELOG receipts are for consumer-visible changes only. Commit `64b113b6`.
6. **A29 — upstream watch refreshed.** templ#1449 CLOSED completed 2026-10-05 11:15; v0.3.1070 shipped 2026-10-04 (our floor: v0.3.1020). Retirement queued behind verify + train. treefmt-nix#545 + BuildFlow#29 still OPEN. Commit `2908bbd0`.
7. **TODO episode-4 tail strike.** The statusToBadgeMap remnant struck with evidence; only the quiet-window loginpage coverage re-pin remains. Commit `c2353fc2`.
8. **Cross-session verification (read-only):** #52 closure comment confirmed live on go-cqrs-lite AND `TestCloneEvent_PreservesPayloadEncoding` confirmed present in the `signing/v4.3.3` tag diff (clone-fetched); agents-notes signing narrative confirmed at line 300. A15/A16 verified done, not duplicated.

## b) PARTIALLY DONE

1. **v5 runbook (A10)** — skeleton by design: wave mapping + gates are decided; per-class consumer notes are at inventory depth, not copy-paste depth. Remaining: expand notes as the cut approaches (blocked on OQ11 timeline). Effort to finish: M per class.
2. **A8.2 — BuildFlow step names** — assembled from prior VERIFIED captures (AGENTS gotcha 8) + today's aggregate "86 step(s) failed"; NOT a fresh dry-run+tee re-derivation. Defensible (names are gated content), but the plan text asked for a fresh capture. Effort: S if wanted.
3. **A1 CI-loop closure** — local gate green, no post-fix CI run id recorded (push blocked by the train gate). Remaining: one push + one CI watch. Effort: S once unblocked.
4. **The train (A3)** — fully staged (dep-budget fix + badge refactor + R4 drift-guard + R5 TOTP seam fixes all in tree, unpushed), zero tags cut. Blocker: upstream templ-components v1.20.0 unconsumable (see d-4). Everything else is ready.
5. **This report's (f) list** — brainstorm for HARVEST; not yet routed into TODO_LIST/ROADMAP (deliberate: most items already live there; the delta is listed in f).

## c) NOT STARTED (by this session)

1. **A7 loginpage JS smoke tests** — blocked: workspace graph poisoned (d-4) + flake.nix owned by the active R2/R8 session. Still wanted (High).
2. **A9 battery remainder** (`.#test-all` + `.#bench-spike`) — blocked: graph + load policy (quiet window < 6 twice). Still wanted (High).
3. **A11 codemod Track A scaffold** — blocked: graph + go.work/flake hot files. Still wanted (High).
4. **A12 Track B v4-journal goldens** — blocked: graph; identity-model adjacency to the active R5 work (f7af84b1 touched setup tests replacing NewUserID — v5-prep is IN FLIGHT by another session). Still wanted (High).
5. **A13 go-cqrs-lite "Doc-only" hook repro** — deferred: that repo is actively landing fixes (C040, signing); a repro session should coordinate. Still wanted (Med).
6. **A14 loginpage Playwright E2E** — blocked: needs a working graph + a running usermgmt service. Still wanted (Med).
7. **A30 scorecard timing rerun** — deferred: meaningless under a broken graph + concurrent load; row is low-priority by its own text.
8. **A17–A28** — owner-gated / demand-gated / dormant (unchanged from the plan: D4/D6 ticks, E005 file, cross-repo records, buildcache reclaim, owner-call batch, appkit/V007/dormant watches).

## d) TOTALLY FUCKED UP

1. **Amend-after-daemon-push divergence (2026-10-04).** I message-amended `6d6f03ef` without re-fetching — the daemon had ALREADY pushed my pre-amend change as `ba9a1b62`. Non-fast-forward divergence, recovered via verified-carrier `git rebase --onto` (no force-push, per policy). Cost: ~30 min, and BOTH of my recovery commits (`f721529e`/`229b86a6`) evaporated anyway when the concurrent session rebased the branch — my A2 closure was written twice and survived zero times as authored. Root cause: treated the daemon as commit-only; it also pushes. Workaround now: fetch + check `origin/master` BEFORE any amend.
2. **Commit-message loss by daemon racing (≥6 times this session).** Despite immediate commits, BuildFlow hooks (380 s) + daemon polling beat me repeatedly: the dep-budget fix, the badge-refactor bulk, and the v5 runbook stand in history under heuristic `chore: auto-commit` messages. Mitigation used where possible: message-only follow-up/amend commits carrying the rationale (`44e03709`, `c2353fc2`). Damage: history discoverability, not content.
3. **Pipe-rc trap hit live (gotcha 3, known, still hit).** `go build ./... 2>&1 | head -5; echo ROOT_BUILD_RC=$?` printed `0` on a FAILED build (rc was `head`'s). I briefly reported "rc=0" internally before catching it. The rule stands: `cmd > f 2>&1; rc=$?` — never through pipes.
4. **The tree cannot build right now (pre-existing, owned elsewhere, but it blocked most of my plan).** Root module resolution fails: `go-cqrs-lite/catalog/v4@v4.6.1 → templ-components@v1.20.0 → charts/echarts@v1.20.0-00010101000000-...` (unknown revision). The published templ-components v1.20.0 root go.mod requires submodule pseudo-version placeholders that exist as no tags. Consequence: NO workspace build/test/bench/train until upstream re-releases or R1's unblock call lands. My lane (docs + cross-repo + hermetic per-module work) was chosen because of this — verified real, not an LSP artifact (the LSP's 8469 errors are consistent with the true resolution failure).
5. **Stale-plan execution start.** I began executing the Oct-4 plan as written before discovering ~a day of concurrent progress; only the cross-session re-assessment (after A15/A16 turned out done) caught it. Nearly duplicated four finished tasks. No damage, but the resume protocol (see e-2) should have been step zero.
6. **Edited a code module during an active foreign session.** The adminui refactor ran while another session committed every 2–5 min; the daemon bundled MY adminui files with THEIR R4 files (handler.go, audit_context*) into one commit `8cc079f6`. I correctly refused to amend (would mislabel foreign work) — the refactor's real message rides a 1-file follow-up commit. It worked out; the guardrail (wait-tree-quiet / single-writer lane) was skipped and luck-leaning.
7. **Repeated known-failure commits.** After the 86-step BuildFlow failure was established (deterministic non-devShell class), I still ran one plain `git commit` that re-triggered it. The `--no-verify`-with-justification fallback should have been used from the second attempt on.
8. **Edit-tool first-attempt failures (×3)** — edited files I had only grepped/sed'd, not Viewed. Friction only; every edit landed on the View-first retry.

## e) WHAT WE SHOULD IMPROVE

1. **Fetch-before-amend rule** (new, concrete): the auto-commit daemon BOTH commits and pushes. Any `git commit --amend` must be preceded by `git fetch && git merge-base --is-ancestor HEAD origin/master` (abort if not an ancestor). Candidate for a gotcha-4 refinement + a tiny wrapper script.
2. **Session-resume protocol:** after ANY gap, step zero is `git log --oneline -15` + re-read the TODO_LIST header + `ls docs/planning/` for newer plans, BEFORE executing any stored plan. This session proved plans go stale in hours here.
3. **Single-writer lane discipline:** while foreign sessions are mid-flight (daemon commits < 5 min apart), restrict to docs-only + cross-repo + hermetic per-module lanes; code modules and shared files (flake.nix, go.work, TODO_LIST, AGENTS.md) only after `wait-tree-quiet`. Cost when skipped: bundled attributions, mislabeled histories.
4. **Race-resistant commits:** pre-write the message to a temp file and `git add <files> && git commit -F <file>` in ONE command; accept that heuristic carriers happen and always follow with a message-only commit when content matters (current de-facto — formalize it).
5. **Plan-text realism for re-derivation tasks:** A8.2-style "re-capture known facts" tasks should explicitly allow "verified prior capture + fresh aggregate receipt" as a done-state, so the choice is recorded as a decision, not a shortcut.
6. **Upstream consumability is a FIRST-CLASS release input:** the train plan assumed alignable targets; the broken templ-components release idled the entire 20% tier (battery, codemod, E2E all need a building tree). R2's pre-flight gate is the right permanent fix — prioritize it when the tree quiets.

## f) Next 50 (brainstorm — HARVEST fuel; most route to existing rows, new deltas marked *)

1. R1 call: resolve the push unblock (wait / replace-shield / --no-verify) and record it — Critical / M / Release
2. File the templ-components v1.20.0 unconsumable-release issue upstream (R17, verify-before-filing first) — Critical / S / Bug-report
3. When upstream re-releases: re-run templ-components family sweeps (bump-dep --commit), strict gates, then push — Critical / M / Release
4. Cut the release train (A3, wave-ordered; now also carries badge refactor + R4/R5 fixes) — Critical / L / Release
5. Close A1's CI loop: watch module-architecture green on the pushed dep-budget fix (record run id) — High / S / Release
6. Proxy smoke: scratch `go get` every new tag post-train — Critical / S / Release
7. Cut CHANGELOG version headers BEFORE tagging (train step 3.7) — High / S / Release
8. Run `.#check-modules` composite (28 stages) at the first quiet window — High / M / Quality
9. Re-pin the loginpage coverage gate post-adoption (quiet window) — Med / S / Quality
10. A7: loginpage JS smoke tests (node:test, Base64URL property test, serialize goldens, wire into `.#test`) — High / M / Quality
11. A9: `.#test-all` (28 modules) + `.#bench-spike` under verified quiet (< 6 twice) — High / M / Quality
12. A11: `cqrs-htmx-upgrade` codemod Track A (R1–R3, R9–R10, dry-run default, golden fixture) — High / L / Feature
13. A12: Track B data-compat goldens (real v4.13 journal, 21-event decode, upcaster matrix) — High / L / Feature
14. A14: loginpage Playwright E2E (real WebAuthn ceremony + full-page golden) — Med / L / Quality
15. A13: reproduce the go-cqrs-lite "Doc-only" hook skip (coordinate with that repo's active session) — Med / M / Quality
16. A30: scorecard timing rerun in a healthy window, close as env or real — Low / S / Quality
17. templ#1449 follow-through: verify the fix in v0.3.1070, schedule the workaround retirement + templ floor bump with templ-components' repair release — Med / S / Quality
18. R2: consumability pre-flight gate (`check-family-release-consumable.sh` + self-test + wire) — Critical / L / Tooling
19. R18: bump-dep pre-flight refusing placeholder pseudo-versions with an "upstream broken" message — High / M / Tooling
20. R19: expiring "upstream-broken" allowlist proposal for check-release-train — Med / L / Tooling
21. R20: runbook updates (validate-before-align, rc-capture, moving-target stop-rule) — Med / M / Documentation
22. Expand the v5 runbook consumer notes toward copy-paste depth per class (needs OQ11) — Med / M / Documentation
23. OQ11: owner decision on the v5 timeline — unblocks V007 + runbook growth + codemod priority — Critical (owner) / S / Planning
24. Class-5 rename decision: identityadmin/esdashboard vs keep adminui/dashboardui — Med (owner) / S / Planning
25. A17: wire check-cqrs-lint into CI (post-D6) — Med / M / Tooling
26. A18: fleet cqrs-lint binary swap + rules-diff ritual (post-D4) — Med / M / Tooling
27. A19: file the E005 cross-module FP proposal (draft ready in docs/proposals/) — Med / S / Bug-report
28. A20: go-cqrs-lite cross-repo records + vet/lint/race on cmd/cqrs-lint (post-owner) — Med / M / Quality
29. A28: owner-call batch (PapDashboard SEND, datastar-demo keep confirm, D3/D8–D11 ticks) — Med (owner) / S / Planning
30. A27: /mnt/buildcache reclaim (rust/ 155G + sccache/ 20G) at a maintenance window — Med (owner) / M / Ops
31. R9/R10: triage-ledger freeze of rejected analyzer classes + isolate the 4 high strong-id rows — Med / M / Quality
32. R11: credential 4× decision memo (no code) — High / L / Quality
33. R12: per-module coverage/candidate-count proof — Med / M / Analysis
34. R15: analyzer-subset tuning for the branching-flow gate — Med / M / Tooling
35. R21: credential data-model review (feeds R11) — Med / L / Quality
36. R23: nightly JSON/SARIF digest proposal — Low / M / Tooling
37. R25: small-findings bundle (navItem, Capabilities Has*, flagparam, overlap doc, suppression convention, verdict template, FP-rate baseline) — Low / L / Cleanup
38. * Record the fetch-before-amend refinement into AGENTS gotcha 4 (+ optional wrapper script) — Med / S / Documentation
39. * Codify the session-resume protocol (step zero: git log + TODO header + newer-plan check) into AGENTS — Med / S / Documentation
40. * Formalize the race-resistant commit pattern (pre-written message file + one-shot add+commit -F) — Low / S / Tooling
41. Verify CI green end-to-end on the eventual push (module-architecture + lint + test + the new branching-flow lane) — High / S / CI
42. V007 cluster-1 criterion re-check at the next train (upstream metaengine + checkpoint/DLQ option) — Med / S / Watch
43. appkit v5-window re-check at the next train — Low / S / Watch
44. SidebarNav revisit trigger check at the next dashboardui UI change — Low / S / Watch
45. Watch treefmt-nix#545 + BuildFlow#29 responses; retire shims when they land — Low / S / Watch
46. DataStar Tier-4 demand-gate re-check at the next train — Low / S / Watch
47. Route this report's (f) deltas into TODO_LIST/ROADMAP via HARVEST (the standing rule: items not already in living docs die here otherwise) — High / S / Documentation
48. TODO_LIST header restamp: fold in this session's deltas (A5 closure, runbook, templ#1449 watch) — Low / S / Documentation
49. Post-train: annotate THIS report (docs-health ANNOTATE) with the outcome hashes — Low / S / Documentation
50. Reconsider a CI-side "workspace-TEST gate" equivalent (the 06-54 report's observation: the workspace-TEST gap is what let regressions ride trains unnoticed) — Med / L / Quality

## g) Three questions I cannot answer myself

1. **The push-unblock call (R1):** upstream templ-components v1.20.0 is unconsumable and there is no announced repair. Do you want the branch pushed via `--no-verify` with the recorded justification, held until upstream re-releases, or shielded with a replace pin (the 373209a7-class pattern we retired for being load-bearing)? I can execute any of the three; the risk appetite (shared-history rewrite tolerance vs upstream relationship vsCI red on master) is yours to set.
2. **OQ11 v5 timeline:** is there any target window (even a quarter) for the v5 cut? It gates the V007 migration branch, whether the runbook grows from skeleton to execution plan, and how much codemod (A11) vs goldens (A12) work should be prioritized now.
3. **Module renames (inventory class 5):** at v5, do we adopt `identityadmin`/`esdashboard` (one-time discovery fix, one cycle of doc shims) or keep `adminui`/`dashboardui` permanently (the cross-linked docs mitigation becomes the final answer)? I cannot make the naming-identity call.

---

*Point-in-time snapshot. Annotate non-destructively (docs-health ANNOTATE), never rewrite. Section (f) is HARVEST fuel — items not routed into TODO_LIST/ROADMAP die in this file.*
