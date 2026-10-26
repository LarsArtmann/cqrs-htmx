# Round 31 — W04/W05/W06 landed, CI green through d4a65b0f; waiting on owner

> **ANNOTATED 2026-10-25** — ~~this report~~ **SUPERSEDED** by [`2026-10-25_19-27_round-32-train-lag-fix-w07-schema-stamps-receipts_status.md`](2026-10-25_19-27_round-32-train-lag-fix-w07-schema-stamps-receipts_status.md), which Round-31 session report superseded by round-32 (W04-W06 receipts live in module CHANGELOGs; the 3 owner questions carried forward verbatim in section g). Strike depth:
> supersession verdict (round-18 precedent for series-internal reports); nothing here is a
> unique open item. **ARCHIVED 2026-10-25 (round-23 (2026-10-25)).**

**Date:** 2026-10-25 18:28 CET · **Session:** round 31 (this file's author)
**Plan of record:** `docs/planning/2026-10-25_10-42_SUPERB-wave-d-completion-code-fidelity-to-closure-pareto-plan.md`
**Prior report:** `docs/status/2026-10-25_16-45_ci-inventory-closed-w04-half-executed_status.md` (superseded by this one, archived with it)

---

## a) FULLY DONE (verified green)

1. **Red-CI triage for `9bccf0cd`** (run 40208754369, FAILURE): root-caused to the corpus ratchet — the sibling's baseline bump (32281) missed the archived-README net growth (+2). Fixed by baseline bump `f1fb20f9` (32352).
2. **Second red caught and fixed before it spread:** `f1fb20f9` itself failed the **tail ratchet** (3 live reports vs baseline 2 — my round-30 report was banked by the daemon without an archive counterpart). Fixed by the archive ritual (`5bc6f833`: archived the 15-22 W02/W03 report under the round-30 successor, corpus → 32357). **CI green from `8101189f` onward.**
3. **W04 — S38 a–g ALL LANDED** (identity-model; hermetic build+vet+test+lint 0 issues; downstream usermgmt 17.5s suite green; workspace build green; CI `8101189f` SUCCESS):
   - S38b-fold: `ParseTenantID` wired into `FoldMembership` — ambiguous tenant references rejected at replay (Rejection family).
   - S38c: additive `UserState.IsActive` / `BotState.IsActive` (Exists semantics frozen; Membership documented as needing no split).
   - S38d: `NewAuthz` skips g2 seeding under custom models without a g2 section (extracted `seedDefaultRoleHierarchy`; cyclop back under 12).
   - S38e: EmailVerification/TOTP/OAuth `NotConfigured` → Rejection (400-class), family+IsRetryable pinned.
   - S38f: 7 command slice accessors return `slices.Clone` (mutation probes green).
   - S38a/g tests: ULID-ordering (5ms-apart strict increase), parse accept/reject vectors, fold rejection, liveness matrices.
   - identity-model CHANGELOG receipt (daemon split it from code into `8101189f` — same push).
4. **W05 — S42 systemadapter declarative fidelity LANDED** (`688f7683`; lint 0, suite green; CI green as of d4a65b0f tip):
   - S42a/b: `Enforce` now answers what legacy Casbin answers (only admin/super_admin; the invented viewer→read/user→write/owner≡admin ladder removed). **Live parity test** drives BOTH paths per role×action (5 roles × 3 actions) and asserts equality — the old widening test ("viewer should be allowed to view") flipped to the deny it always should have been.
   - S42b: audit Action vocabulary = `usermgmt.AuditAction*` constants (change_email→email_changed etc.); LoginSucceeded/LoginFailed(+reason Detail)/RegistrationClosed folds added; external-account labels pinned as documented declarative-only constants.
   - S42c: `AuditEntryView` gains `ActorID` (kind:raw from record metadata) + `Detail`; unit test pins the fold's data path; honest limit documented (declarative dispatch can't attach actors until upstream `system.Execute` propagates context — v5 ask).
5. **W06 — S46+S49 consumer seams LANDED** (`d4a65b0f`; CI SUCCESS):
   - S46a: `TestSetup_DocsHandler_SessionGatedMount` — end-to-end through `setup.New` + `bundle.Handler`: 401 unauthenticated, 200+marker with a register-issued session; mismatch (DocsPath without DocsHandler) rejected at New. integration_test now consumes setup/v4 (proxy-pinned, GOWORK=off).
   - S46b: `TestHealth_RecorderChain_MergesProjectionsAndAuditPlugin` — the "one probe" chain proof with the real audit plugin; nil-skip semantics pinned.
   - S49a/b: **verdict tests landed.** Memory store round-trips StateShape (green). SQL store (storage v4.11.1) **drops the stamp** — confirmed at source (INSERT/SELECT omit the column) AND empirically; pinned as `TestS49_SQLSnapshotStore_DropsStateShape_UpstreamGap` with the flip instruction. Upstream ask routed to the W13 packet.
   - Full integration_test suite green (1.1s), lint 0 after gci/--fix pass.
6. **Train alignment:** pre-push release-train gate blocked the W06 push (setup v4.14.4 pinned vs v4.15.0 published — my pin was stale on arrival). `bump-dep.sh` sweep to v4.15.0 across all 3 consumers, strict train check rc=0, pushed through the gates (`d4a65b0f`).
7. Todo list restored (19 items) and kept current through the session.

## b) PARTIALLY DONE

- **W07 (S58 WithSchemaVersion(1))**: research started — spec row located (TODO_LIST:88: 21 idm constructors + 8 e2e/demo sites, "stamp now so the first real schema change is upcastable"; health-train rider row TODO_LIST:28 = delete status mirrors + root require bump — **NOTE: the RecorderChain integration_test half of that rider is now DONE by W06**). No edits made.
- **Release-train question**: identity-model/usermgmt/systemadapter all carry uncommitted-to-tag Unreleased changes (W04/W05 + round-30's S39). No tag cut decision taken (question 3 below).

## c) NOT STARTED

W08 (verify sibling's S45/S52/S53/S54 receipts), W09 (sibling lane), W10 (loginpage Playwright WebAuthn E2E + golden), W11 (five file splits: setup/config.go 1186, systemadapter/declarations.go ~700, usermgmt/service_core.go 576, dashboardui/layout.go 518, e2e/server/main.go 487), W12 CHECKPOINT (full battery 4× rc=0 + push + CI watch + `.#verify-remote-tags` + book reconciliation vs sibling), W13 (owner packet), W14 (backlog I), W15 (backlog II incl. the S49 upstream draft), W16 (BuildFlow nix-fmt + bench), W17 (8 parked-row verdicts + FINAL battery + receipt superseding this one + CURRENT_FOCUS rewrite).

## d) TOTALLY FUCKED UP (own goals, each cost a cycle)

1. **`f1fb20f9` pushed RED on the tail ratchet** — I bumped the corpus baseline but never ran `check-docs-tail-budget --ratchet` (only `--strict`, which passes at budget 3 while the ratchet baseline is 2). One avoidable CI failure on a gate class I had JUST been burned by. Root cause: gate-checking piecemeal instead of running the full docs-gate pair.
2. **Verdict-from-piped-rc trap (again):** read grep's exit 1 as a usermgmt test failure; wasted a full 17.5s suite re-run to discover it was green. The session summary explicitly warns "verdicts only from unpiped rc or PIPESTATUS[0]" and I still got it on the first read.
3. **Stale dep pin on arrival:** chose setup v4.14.4 without checking the newest tag; v4.15.0 was already published → pre-push release-train block → extra bump sweep + commit + push cycle. (`git tag --list | sort -V | tail -1` is one command.)
4. **Test-file package sloppiness in systemadapter:** wrote `package systemadapter` into a file needing the EXTERNAL package's harness (startDeclarativeSystem/eventually/must), plus an invented `RoleMember` constant and a `must` signature mismatch — three compile-fix iterations that 30 seconds of reading the neighbor file's package line would have avoided. Split into internal+external test files (correct end state).
5. **Daemon raced commits twice** (W04 CHANGELOG, W06 code): receipts landed in daemon commits separate from the code. Same push/train, so the CHANGELOG-same-commit discipline held at the push level, but not at the commit level. Not fully preventable (daemon windows), but the `git add && git commit` chain in ONE bash call still lost twice — worth a post-edit immediate-commit habit instead of verify-then-commit.

## e) WHAT WE SHOULD IMPROVE

1. **Mechanize the docs-gate pair:** never push anything touching `docs/` without BOTH `check-docs-corpus-budget --ratchet` AND `check-docs-tail-budget --ratchet`. Two commands, zero exceptions — both this session's red CIs were this class.
2. **Newest-tag check before any pin:** one-liner before `go get`/bump-dep. The train gate catches it at push time — CI-parity is working as designed — but pre-checking saves the cycle.
3. **Read the neighbor test file's package clause + helper signatures before writing a new test file** (the mixed internal/external package layout in this repo makes this non-optional).
4. **The corpus baseline is a seesaw when reports + archives land together:** archive adds ~5 lines (blockquote+manifest) while removing 0 (file moves, lines counted under docs/ regardless). Every report commit should compute: bump = report_lines + 5·archives. Write it down once in the runbook instead of rediscovering it.
5. **Watch CI with branch-list polling** (`gh run list --branch master --limit 1`) right after push; `--commit <sha>` filtering races registration twice per session.
6. systemadapter's `system.Execute` actor gap (S42c limit) belongs in the W13 upstream-asks packet alongside S49's SQL StateShape column — two concrete, evidence-backed asks now exist; draft both in one sitting.

## f) NEXT UP TO 50 (ordered; plan-of-record numbering)

1. Watch/confirm CI stays green on this report's push.
2. W07.1: stamp `event.WithSchemaVersion(1)` on the 21 identity-model payload constructors.
3. W07.2: stamp the 8 e2e/demo event sites (build-gated).
4. W07.3: delete `statusLive/statusStopped/statusFailed` mirrors in health/probe.go; consume `cqrshtmx.ProjectionStatus*`; bump health's root require.
5. W07.4: integration_test half — ALREADY DONE in W06 (RecorderChain proof); close the rider row with a note.
6. W07.5: idm + health CHANGELOG receipts; hermetic suites + lint.
7. W08.1: verify sibling's S45 (EventHandler section + idm README) exists and matches code.
8. W08.2: verify S52 mandate row, S53 credential memo, S54 v5 skeleton.
9. W08.3: docs gates (links/freshness/corpus/tail) rc=0 + AGENTS quick-ref truth-pass on sibling rows.
10. W09 (sibling lane — skip if sibling active): check-upstream-treadmill.sh + wave-tag.sh + self-tests + flake wiring.
11. W10.1: Playwright virtual-authenticator harness for loginpage.
12. W10.2: register+login E2E through the rendered page.
13. W10.3: nonce-insensitive full-page render golden (stable IDs).
14. W11.1: split setup/config.go (1186 lines) by concern.
15. W11.2: split systemadapter/declarations.go (query/fold files).
16. W11.3: split usermgmt/service_core.go (576).
17. W11.4: dashboardui/layout.go assets refile + e2e/server/main.go split.
18. W11 gate: build + `check-large-files` + per-module suites + goldens.
19. W12: full battery `.#test` rc=0.
20. W12: `.#lint` rc=0.
21. W12: `.#coverage-gate` rc=0.
22. W12: `.#check-modules -- --report` rc=0 (≤2 attempts).
23. W12: push + `gh run watch` + `.#verify-remote-tags` + ahead-0.
24. W12: `git log --stat` book reconciliation vs sibling's 15-16 receipt.
25. W13: owner packet D1–D17 answered-or-recommended.
26. W13: upstream asks — ADD the S49 SQL StateShape column ask (evidence: the verdict test).
27. W13: upstream asks — ADD the systemadapter `system.Execute` context/actor ask.
28. W13: S56 watch table (buildcache, fleet asks, scorecard, catalog consumers).
29. W14: loginpage AutoRedirect follow-through (NoWebAuthn asymmetry memo, AuthPrefix validation, guide, FEATURES rows).
30. W14: dashboardui 09-48 tails (rune-safe truncate/initials, PathEscape, data-base-path, audit streaming, neighbour early-stop, snapshot 404/500, boost guard).
31. W14: root declutter (biome.json prove-or-remove, allowlist sketch).
32. W14: sweep-plan M18/M20/M21 + RequireSession export-or-document.
33. W15: art-dupl r3 accept-baseline decision + 2 extractions.
34. W15: check-skill-claims.sh skeleton.
35. W15: snippet compile gate + README split-brain scan.
36. W15: tooling canaries a–g.
37. W15: upstream ask drafts batch (now incl. items 26–27).
38. W16: BuildFlow nix-fmt loop source-verified + upstream filing per verify-before-filing.
39. W16: bench-spike ONLY at load < 6 (honest refusal otherwise).
40. W17: 8 parked-row verdicts (billing S00, DataStar Tier-4, ReadLastN, SidebarNav, BuildFlow BF1, cqrs-lint fleet swap, check-cqrs-lint CI, GCL).
41. W17: TODO promote/unpark pass.
42. W17: FINAL battery + push + CI watch.
43. W17: final receipt superseding THIS file (archive ritual + both baselines; tail back to ≤2).
44. W17: CURRENT_FOCUS rewrite.
45. Standing: pull + `git log --stat` before each milestone (sibling interleave).
46. Standing: hermetic per-module build+test+LINT before any "landed" claim.
47. Standing: re-verify every file:line in-source before editing (plan guardrail 6).
48. Standing: receipts land in the same push as published-module code.
49. Standing: unpiped rc only for verdicts; PIPESTATUS for piped.
50. Standing: no gate weakened, ever — CI-parity pre-push is the backstop.

## g) QUESTIONS FOR THE OWNER (cannot be resolved from the repo)

1. **Release-train timing:** identity-model (S38), systemadapter (S42), and usermgmt (S39, round-30) all carry substantial Unreleased changes, and W07 will add more to identity-model. Cut the release train NOW to bank W04/W05 before W07 piles on (cleaner release notes, smaller blast radius), or continue stacking the wave and tag once at W12 (fewer trains, one big delta)? The plan is silent; gotcha 8 only forbids tagging with uncommitted edits.
2. **S49 + S42c upstream filings:** both upstream asks are evidence-backed and test-pinned (SQL snapshot StateShape column; `system.Execute` context/actor propagation). File both on go-cqrs-lite as soon as you tick the S55/U2 approval, or hold everything for the W13 packet so you review the wording first?
3. **The three standing defaults I applied** (from round-30's unanswered questions): AccentColor reject-at-`New` KEPT; OIDC nonce KEPT additive-permanent; S38c Exists/IsActive split KEPT additive. All three are shipped, tested, and CHANGELOGged. Ratify all three, or name any to revert (reverts are cheap now, expensive after the train).
