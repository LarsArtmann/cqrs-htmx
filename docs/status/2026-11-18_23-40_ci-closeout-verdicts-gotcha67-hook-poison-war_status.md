# Status report — CI close-out verdicts, gotcha 67, the hook-poison templ war (2026-11-18 23:40)

**Scope line — what this report VERIFIED:** CI run `44397985522` on `5321c847` (gh run view: all 21 jobs, success), CI run `44411702775` on `468641b1` (gh run watch --exit-status, rc 0), the push `5321c847..468641b1` (pre-push receipt: release-train strict + version-drift strict green), the canonical templ form (final `nix run .#gen` printed updates=0, tree clean), the local doc gates (corpus ratchet at 62,612, tail budget, todo hygiene 86 rows 0 violations, docs freshness PASS, docs-gates annotate CLEAN at 0 findings), and the templ-components release state (gh pr view 101: MERGED 2026-11-17; gh release list: v1.27.0 from 2026-11-04 predates the merge). **What it did NOT verify:** the erraudit strict TOTAL (advisory sweep deferred — 7 sibling crush sessions alive, load ~52 on 32 cores, gotcha 51), a local `check-modules` composite run (deferred same reason; CI's checks job carries both verdicts green on HEAD), the identity of whichever sibling session's hook re-poisoned the templ tree the second time (log evidence proves the mechanism, not the actor), the e2e battery (not rerun — nothing e2e-relevant changed this session), and PR #101's merge content at byte level (state checked, diff not re-read).

## a) FULLY DONE

1. **The open CI verification closed ALL GREEN.** Run `44397985522` on `5321c847`: e2e, build, test, mod-tidy, every lint leg, module-architecture, security, checks — all success; only the self-skipping architecture gates skipped. This settles the format war by evidence: the codegen gate greens on the `nix run .#gen` (generated-AND-formatted) form, so the 22-05 report's canonical-form question needs no owner input and no gate alignment.
2. **Docs trued on the verdict.** TODO_LIST CI-watch row removed per the retirement rule (its watched run `44397175316` was in fact cancelled by the push race — the verdict arrived on the later run); CURRENT_FOCUS flipped to no-active-thread/owner-wait; CHANGELOG Housekeeping receipt landed; the 22-05 report's §f.1–4 annotated done/moot/resolved and its receipts line trued.
3. **Gotcha 67 landed with full wiring.** The two-canonical-forms trap (bare `templ generate` vs the gen app), the `updates=N`-counts-touches-not-bytes rule, and the twice-regenerate diagnosis — entry in `docs/gotchas.md`, AGENTS index row 67, count trued 66→67, a producer clause appended to gotcha 10, and the harness cheatsheet's templ section repointed from the raw pinned-binary invocation to `nix run .#gen`.
4. **A fourth templ producer was discovered live and documented same-session (gotcha 67 rule d).** The BuildFlow pre-commit hook's `go-generate` step regenerated the 7 adminui `_templ.go` files with its BUNDLED generator (v0.3.1123) while go.mod pins v0.3.1100 — the daemon captured the drifted bytes as a heuristic commit, and a sibling hook re-poisoned the tree a second time minutes later. Both heals via `nix run .#gen`; the final run prints updates=0 (canonical, twice-regenerate-verified). The drift survives treefmt, so formatting never masks it.
5. **Corpus ratchet discipline held.** Ledger sentence + `CORPUS_BASELINE` 62,603 → 62,612 (+9 net: gotcha 67 block +4, cheatsheet repoint +4, ledger line +1; verified by git diff --numstat), ratchet green; tail held at 13 (no new report until this one).
6. **The annotate gate repaired to CLEAN.** Two findings fixed on sight: the migration plan's struck W1.3/W1.4 item carried no parseable verdict marker (the resolution said "executed", which is not in the marker vocabulary — now "done" adjacent to the strike), and the 11-10 report's own g3 receipt line carried the exact backtick-adjacency violation that line documents fixing elsewhere. Gate now 0 findings over 209 files.
7. **Push completed with CI-parity.** `5321c847..468641b1`, 4 commits (docs truing; baseline; templ canonical heal + gotchas; annotate fixes), pre-push release-train strict + version-drift strict green, 1047 internal requires all published.
8. **The push's own CI run verified green.** `44411702775` on `468641b1`: SUCCESS, watched to completion.
9. **templ-components release watch trued.** PR #101 is MERGED (2026-11-17) but the latest release (v1.27.0, 2026-11-04) predates the merge — adminui's vendored popover CSS layer stays until a release carries it. Watch row stands.
10. **Todo list maintained to close-out state** (14 completed, 1 parked on owner gates).

## b) PARTIALLY DONE

1. **erraudit advisory TOTAL re-true — deferred, with a deadline.** The AGENTS receipt still says TOTAL=352 (window closes 2026-12-16). CI's checks job (which runs the erraudit advisory leg) is green on HEAD, so the gate substance is covered; the bookkeeping number is not refreshed. Deferred deliberately under 7 live sibling sessions (gotcha 51).
2. **Local `check-modules` composite — deferred.** CI green on HEAD covers the substance; the local/CI parity honesty run did not happen this session.
3. **The hook-poison mitigation — documented but not fixed at source.** Gotcha 67 rule (d) records the mechanism and the heal recipe; the hook still runs a mismatched generator over the three codegen modules, so EVERY sibling hook run can re-dirty the tree (it happened again before session end; the final gen run healed it). Hook edits are owner-gated (gotcha 65).

## c) NOT STARTED (all previously gated; nothing new opened)

1. M2M W1/W2 waves — gated on the D1–D4 owner ticks (~2 days of work pre-staged in the packet).
2. Adminui panel waves (impersonation surface, sessions, rename) — gated on D1–D3 ticks.
3. K2 idempotency default-mount wave — gated on one "GO" tick (packet pre-staged, ~2–4 h).
4. §g.3 cache-poisoner attribution (who provisioned chromium-1255 on Nov 17 ~19:25) — owner knowledge.
5. W3/W4 deep migration waves — gated on D1/D2 recording.
6. adminui vendored popover-CSS retirement — gated on a templ-components release carrying PR #101.
7. CI security-job `go-version` revert to `go-version-file` — gated on the ADR-0003 fleet nixpkgs rev shipping Go ≥ 1.27.4.
8. B18 — trigger-gated on the next templ-components/go-cqrs-lite family train.
9. docs-gates audit-all drain-or-repin decision (393 vs the 322 pin) — parked in TODO P2.
10. A hook-side templ fix (scope `go-generate` out of the codegen modules, or pin its generator) — NEW decision opened by tonight's incident; owner-gated (§g.1).

## d) TOTALLY FUCKED UP (own failures this session, named)

1. **Two lost ref races on hand-commits.** The amend ("fold the baseline into the same commit") died with `cannot lock ref 'HEAD'` while the 82-second hook ran; an earlier hand-commit was likewise overtaken. Work converged (the daemon captured everything), but two descriptive commit messages were lost to heuristic titles, and the corpus baseline landed in a commit adjacent to — not inside — the growth it pins (a letter-violation of the same-commit rule; the pushed pair is net-green, and CI verified the end state).
2. **First edit batch: 3 of 8 edits bounced on stale read-state.** The previous session had modified `docs/gotchas.md`, `AGENTS.md`, and the status README minutes before this session started; I edited from the session-start snapshot instead of re-viewing the targets first. Cost one full re-read-and-reapply round.
3. **A fabricated old_string.** The gotcha-67 entry edit assumed a bold-boundary (`**`) that does not exist in the file, because I had only seen the line through a truncated (character-cut) read. Two edit rounds failed before the exact substring was taken from a full read.
4. **A false-alarm verification round.** I grepped the anchor form (`gotcha-67`, hyphenated) to confirm prose references written as `gotcha 67` (spaced) and briefly concluded the edits were missing. Both spellings should have been grepped in one command; the panic cost one diagnostic cycle.
5. **A wasted 108-second full-hook run.** The daemon committed the staged files seconds before my `git commit` executed; the hook ran to completion under SQLITE_BUSY contention (siblings' concurrent BuildFlow runs) and exited 1 for nothing. A `git status` check between add and commit would have skipped it.
6. **Ledger estimates written before measuring.** The placeholder ledger guessed +6/+2 per file; the real split was +4/+4, caught only by a numstat re-derivation. The measure-first discipline was applied one step late (the final numbers are honest, but the detour was avoidable).

## e) WHAT WE SHOULD IMPROVE

1. **Race-proof the commit ritual.** Under a live daemon, either run `wait-tree-quiet` then a single-call add+commit, or accept the daemon as the sanctioned committer for mechanical captures and hand-commit only at true phase boundaries — the hybrid (hand-commit a daemon-raceable tree) loses both ways, proven twice tonight.
2. **Fix the templ poison at source (owner-gated).** Scope the hook's `go-generate` step away from adminui/loginpage/dashboardui, or pin its generator to go.mod's templ version — otherwise every session's hook run re-opens gotcha 67.
3. **Give the gen app a `--check` (no-write) mode**, then wire a codegen canonical-form leg into check-modules — gates and sessions could verify canonical form without mutating, and origin would self-verify on every CI run.
4. **Batch-read-then-batch-edit.** Refresh view state on ALL edit targets in one parallel batch before the first edit; never mix stale-snapshot edits with fresh-file edits in one batch.
5. **Never build old_string from truncated reads** — full-line reads (awk, not character-cut) for any edit target.
6. **Grep both spellings when verifying cross-references** (anchor hyphen form and prose space form) in a single command.
7. **Check `git status` immediately before `git commit`** in daemon windows; skip the hook run entirely if the daemon already captured.
8. **Schedule the quiet-window battery instead of deferring opportunistically.** The erraudit receipt has a real deadline (2026-12-16 window); put a date on the sweep rather than waiting for load to drop by luck.

## f) Things to get done next (owner/unlock items first; most of the tail is brainstorm fuel for HARVEST routing)

1. **OWNER — §g.1 of this report: pick the hook-side templ fix** (scope `go-generate` out of the codegen modules / pin its generator / accept rule (d) + heals as the steady state).
2. **OWNER — §g.2 ticks (M2M D1–D4 + adminui D1–D3)**: one word per row converts the pre-staged waves (~2 days).
3. **OWNER — K2 "GO" tick**: executes the pre-staged idempotency default-mount wave (~2–4 h).
4. **OWNER — §g.3 attribution**: who provisioned the poisoned chromium-1255 cache (Nov 17 ~19:25), or approve writing the provisioning ritual unattributed.
5. **OWNER — §g.3 of this report: CI cost policy** — path-filter docs-only pushes or keep the full matrix on every push.
6. Quiet window: erraudit strict sweep, refresh the AGENTS TOTAL receipt (was 352; blocking-promotion window closes 2026-12-16).
7. Quiet window: one local `check-modules` composite run for local/CI parity honesty.
8. Implement the gen-app `--check` mode (improvement 3) — small, unblocks a canonical-form CI leg.
9. Wire the codegen canonical-form leg into check-modules + CI once 8 exists (origin self-verifies the templ form on every run).
10. On the next templ-components release carrying PR #101: retire adminui's vendored popover-CSS layer same-change (CSS bundle rebuilds + goldens per the family-train ritual).
11. Archive pass: the live tail grows to 14 with this report — the next docs-health round must drain the settled 2026-11-17/18 set back toward the budget of 3.
12. CI security-job `go-version` revert to `go-version-file` when the fleet nixpkgs rev ships Go ≥ 1.27.4 (verify the pin sites in the workflow first).
13. B18 execution when the next family train fires (trigger-gated).
14. audit-all drain-or-repin decision (393 items vs the 322 pin) — TODO P2, untouched.
15. Screenshot drift eyeball: the regenerated dashboard screenshots from `043f388c` need one human check against the goldens.
16. Reconcile or formally park the dual-BuildFlow-checkout hazard (lowercase vs capitalized trees with diverged unpushed histories — recorded by the 11-10 session, neither tree mergeable from here).
17. e2e flake app: add the cached-revision-≠-expected chromium warning (open since the 11-08 wave).
18. e2e README: the serial-context note (open since the same wave).
19. e2e battery timing budget: compare the CI e2e job duration against the 2.7-minute local figure with the healed runner.
20. check-e2e-ports: promote-or-drop decision after the e2e plan receipt (open item 7 of that plan).
21. On an M2M tick, execute W1 per the packet unlock list (token prefix+checksum, required expiry, auth-audit events, scope-policy projection, four invariant tests).
22. On an adminui tick: impersonation panel behind opt-in config, sessions surface, rename field-set sign-off first.
23. On a K2 tick: ServiceConfig default-mount seam + TTL constants + three tests + docs + CHANGELOG + the libs-report row true.
24. W3/W4 deep waves once D1/D2 are recorded.
25. Decide the file-size-check posture: 99 advisory findings (test files over 350 lines) re-print in every hook run — ratchet, verdict, or suppress.
26. Decide the module-inventory posture: 27 go.work modules without dependabot entries — pick coverage intent per module or accept-and-suppress.
27. dependabot-auto-configure cap finding (31 Go modules, generation capped at 20) — raise the cap upstream or record the accepted state.
28. README.md freshness: flagged unmodified for 264+ hours by the hook's age check — refresh or ratchet the check.
29. BuildFlow state DB at 1.03 GB — run the one-line VACUUM at a quiet window.
30. io-history-ratchet warning (a hook run hashed 38 files vs median 7) — one profiled run to find the whole-tree scanner.
31. The unused `STEP_USES_SECRET_RE` in `scripts/checks/check-pr-secrets.py` (vulture finding) — one-line cleanup next time scripts/ is touched.
32. The stale vendorHash preflight warning re-prints every hook run (gotcha 43 documents it as a benchstat non-fix) — consider a `.buildflow.yml` suppression note so it stops screaming.
33. Candidate: a scripted twice-regenerate self-test for gotcha 67 (gen → bare gen → gen, assert the flip signature) — decide if the recipe deserves mechanization.
34. Candidate: post-daemon-capture reconciliation in session rituals — if a heuristic capture straddles a docs edit and its baseline, re-pin immediately (this session's letter-violation class).
35. Candidate: a weekly digest of daemon-authored commits into the agents-notes (heuristic titles bury real work; eight commits tonight carry the session's substance).
36. On the next sibling re-poison: skip the hand-commit entirely, heal via the gen app, and let the daemon capture the canonical bytes (cheaper than racing).
37. Keep the pre-push release-train cache in mind for the next train day: fresh-tag TTL means the first strict run after tagging needs the refresh-cache flag.
38. ROADMAP fuel: a `gen --check`-based pre-push leg would close the "push can still be redder than the composite" gap for the codegen dimension (the hook currently covers release-train + version-drift only).

## g) Questions for the owner (3)

1. **The hook's `go-generate` step re-poisoned the templ tree twice tonight (bundled generator v0.3.1123 vs go.mod v0.3.1100; drift survives formatting; the daemon captures the bad bytes as commits). Hook edits are owner-gated (gotcha 65). Which fix do you want:** (a) scope the step out of adminui/loginpage/dashboardui in `.buildflow.yml` (my recommendation — smallest, stops the class), (b) pin the hook's generator to go.mod's templ version, or (c) leave it and run on gotcha 67 rule (d) + gen-app heals?
2. **Quiet-window priority:** the erraudit receipt (TOTAL=352) has a real deadline (blocking-promotion window closes 2026-12-16) and the local `check-modules` parity run is also owed — but 7 sibling sessions are live. Do you want me to burn a semi-quiet window now (roughly 30–60 minutes of per-module lint under load ~52 on 32 cores), or hold both until you call a quiet window, and if so, when?
3. **CI cost policy:** tonight two full-matrix runs (21 jobs each) were triggered by docs-only deltas. Path-filtering docs-only pushes would save ~20 runner-minutes each, but it trades against the discipline that a push is never greener than CI (the ratchets live in the checks job). Filter or keep?

---

*Receipts: CI runs `44397985522` (5321c847, success, 21 jobs) and `44411702775` (468641b1, success, watched rc 0); push `5321c847..468641b1` (4 commits, pre-push strict green); `nix run .#gen` final updates=0, tree clean; corpus 62,603 → 62,612 (ratchet green, +9 ledgered); annotate CLEAN (209 files, 0 findings); todo hygiene 86 rows 0 violations; docs freshness PASS; templ-components PR #101 MERGED 2026-11-17, latest release v1.27.0 (2026-11-04) predates it; hook-poison incidents 2 (heals 2, both via the gen app).*
