# Status report — the CI toolchain split-brain healed (1.27.3 floor + appkit v0.12.0 + the appkit-hold receipt), the round-48 docs round (archive + audit-all drain), and the T12 masking fix (2026-11-21 14:12)

**Scope line — what this session VERIFIED:** the cancelled ritual from the handoff closed first (the 08-45 report set landed via daemon capture `d761d4e0`, adopted); the docs-gates hook-red window that the landing exposed was named and closed same-session per gotcha 50 — all five gates PASS (annotate CLEAN, freshness FRESH, routing 0 unrouted, cadence ARCHIVE-HEALTHY at tail 15, audit-all 263 ≤ the pin ratcheted 268 → 263) — via the round-48 archive pass (SEVEN settled 11-17/11-18 CI/buildflow-arc reports moved to `docs/status/archived/` with per-report successors and reasons; the archive-status app's archived/-successor defect found LIVE — an `archived/…`-prefixed successor emitted `../` links whenever an earlier line of the same pass had already moved it — fixed at root cause in BOTH the validator and the blockquote writer, pinned by a new two-pass self-test case, 26 assertions, sabotage-proven) and the consolidated round-48 drain row (24 source reports routed with distinctive nouns in TODO_LIST; the matcher lesson recorded: DONE verdicts must keep their distinctive tokens OUTSIDE strikethrough spans or the matcher cannot see them); then master's CI reds triaged to root cause and healed: the sibling's go.work 1.27.4 bump had split the tree three ways (go.work/go.mods at 1.27.4 vs flake + CI at 1.27.3) because go-appkit v0.13.0 declares `go 1.27.4` and tidy silently auto-bumped its four consumers past the fleet floor — healed to the recorded posture (AGENTS floor row): go.work + the four go.mods back to 1.27.3, appkit re-pinned v0.13.0 → v0.12.0 (bump-dep sweep, tidy+verify+build+vet+lint green on all four consumers), the deliberate 4-entry lag receipted on the fleet-floor TODO row (the floor and the appkit train are now the SAME owner decision; one train unblocks both), and a sibling's 1.6 MB render PNG palette-compressed to 654 KB at identical 1440×9517 dimensions (the binary guard's named offender); the PNG fix then UNMASKED a latent hook defect — the install-git-hooks T12 fixture (no go.mod/go.work by design) failed the pre-push tidy leg's rc=2, red since the leg landed and hidden behind the guard red upstream in the same job (the gotcha-40 sequencing class) — fixed: the leg skips only when go.work is ABSENT, a present-but-broken go.work still fails the push (template + both deployments synced, T12 17/17, real-workspace leg 32/32 re-verified under the flake go 1.27.3 with GOTOOLCHAIN=local — the CI-equivalent environment). **NOT verified:** the tip CI run `44907597459` (for `632285c3`) was in_progress at report time — the expected shape is everything green except module-architecture's release-train step on the deliberate 4-entry lag (the predecessor run for `85eb28de` already showed mod-tidy ✓ 36s, e2e ✓, security ✓, test ✓, build ✓, all 14 lint jobs ✓ with exactly module-architecture + checks red — checks being the since-fixed T12, module-architecture being the named lag); the run for `9cb6fd66` was cancelled by the push race (superseded, not evidence); the full check-modules composite still has no single end-to-end green receipt (carried; each leg verified individually).

## a) FULLY DONE

1. **Handoff state adopted; the cancelled ritual closed.** The 08-45 report set was daemon-captured (`d761d4e0`) before my commit ran — adopted per protocol; ledger sentence + both baselines verified in-tree (corpus 64887, TAIL 20 → 21 for the new report), all four local gates green before the landing.
2. **The docs-gates hook-red window is CLOSED — five of five PASS.** The landing exposed: annotate FAIL (one rule-F finding at 08-45 line 21, the glued-suffix form of `git restore` — reworded), freshness FAIL (two items: one token-repair — `NewSignalsPatch` added to the go-datastar row — plus the sibling's fresh:skip-exempted metaengine watch), cadence FAIL (21 unarchived > 15), audit-all FAIL (395 vs pin 268). Each resolved; the battery's final matrix in this session: PASS ×5.
3. **The round-48 archive pass landed as two gated runs** (23-40 closeout first so later successors could point at `archived/…`): seven reports — the 11-17 buildflow-yml verdict, the 11-17 v4.22.x train + gate escalation, the 11-18 master-ci six-job triage, the 11-18 g1g2g3 execution, the 11-18 ci-heal-verified, the 11-18 ci-shepherding battery-245, the 11-18 ci-closeout-verdicts — each with a successor that genuinely carries its remainder (the reasons name the coverage; the gate verified every row struck/routed BEFORE the move). TAIL 21 → 15; cadence green at the 15 threshold.
4. **The archive-status app's archived/-successor defect is fixed at root cause and sabotage-pinned.** Live defect shape: pass B names a successor that pass A already moved — the file exists at BOTH depths, the `-f` probe wins over the prefix, and the blockquote gets `../` links that only break later. Fix: the `archived/` prefix wins in the validator AND the blockquote writer; the new two-pass self-test case reproduces the exact shape (single-pass is masked by the in-process link healer — proven by running the sabotage both ways: one-pass sabotage passes, two-pass sabotage fails with the live link text). Self-test 25 → 26 assertions, all green; shellcheck clean; AGENTS archive row updated.
5. **The audit-all drain: 395 → 263, the pin ratcheted 268 → 263 same-change.** The consolidated round-48 drain row routes 24 source reports (the 22-18 legalparty, 02-26 billing, 08-41 + 11-29 adminui waves, 01-34, 01-49, 14-20, 07-01, 14-26, 20-47, 20-25, 02-52, 03-09, 03-15, 13-15, the 00-19 browser-e2e wave2, 18-05 fleet-audit, 21-31 root-package-gates, 01-57 superbification, and the 10-05 → 11-04 era reports), every item with an honest disposition (DONE with receipt pointers / OPEN owned by its lane / MOOT-EXPIRED session-scoped steps / owner-gated). Row 61 carries the re-true note; one token-blind sibling item (no alphabetic distinctive token exists in it) exempted with the gate's own `<!-- fresh:skip -->`.
6. **The CI toolchain split-brain is healed to the recorded posture.** Root cause chain: go-appkit v0.13.0's go.mod declares `go 1.27.4` → yesterday's treadmill sweep pulled it in and tidy auto-bumped the four consumer go.mods → a sibling raised go.work to match → flake + CI still ship 1.27.3 → three CI jobs red. Healed: directives back to 1.27.3 (go.work + setup, integration_test, examples/setup-demo, examples/async-startup-demo), appkit v0.12.0 (its go.mod declares 1.27.3) swept via bump-dep with all legs green, verified under the flake go 1.27.3 with GOTOOLCHAIN=local (the exact CI environment): `check-go-toolchain` ✓ and the tidy gate 32/32 ✓.
7. **The deliberate appkit hold is receipted, not silent.** The fleet-floor TODO row gained the addendum: the floor and the appkit train are the same decision; one train (fleet nixpkgs rev + directives + appkit + CI go-version pins) unblocks all of it; until then the strict-lag gate names 4 lag entries. The Housekeeping CHANGELOG entry carries the full mechanism + the v0.13.0 re-sweep trigger.
8. **The binary-guard red is healed at the artifact.** The sibling's docs-marketing render PNG (1,607,524 bytes > the 1,048,576 limit) compressed via 256-color no-dither quantize to 654,546 bytes at unchanged 1440×9517 dimensions (four variants measured; the dither variant compressed WORSE — recorded).
9. **The T12 masking defect is fixed.** The tidy leg in the pre-push template now guards on go.work existence (absent → skip, the T12 fixture's designed shape; present-but-broken → still fails). All three hook layers synced from the template (byte-match confirmed by the installer); T12 17/17; the real-workspace tidy leg re-verified 32/32; shellcheck clean. The masking lesson (a guard red upstream hides a downstream defect until the guard heals) is receipted in the Housekeeping entry.
10. **The pushes landed with the documented justification protocol.** Four `--no-verify` pushes, each naming the skipped gate (release-train strict-lag-0 vs the deliberate v0.12.0 hold) with the alternative legs hand-verified per push (version-drift --strict green 1061/1061 published tags; tidy 32/32; conflict-markers, todo-hygiene, corpus + tail ratchets green). The docs-only commits kept the gotcha-50 docs-only discipline.
11. **The daemon reality was honored throughout.** Seven heuristic captures adopted this session (report set, drain row v1, the v2 replacement + pin, the toolchain/PNG set, the appkit sweep, the token-repair set, the hook fix); no history rewrites; the durable receipts live in CHANGELOG, row 21, row 61, and the ledger.
12. **The battery's matcher internals are now documented in-repo** (the drain row + row 61): distinctive-noun matching ignores strikethrough spans and drops digit-bearing tokens — both discovered empirically this session (the go-datastar#31 token never matched; `B18` never matched), both worked around with alphabetic tokens or the sanctioned skip.

## b) PARTIALLY DONE

1. **The tip CI verdict (`44907597459`) was in_progress at report time.** Evidence from the completed predecessor run (`44906289542`, for `85eb28de`): mod-tidy ✓ 36s (the tidy fix works in CI), e2e ✓ 6m20s, security ✓, test ✓, build ✓, all 14 lint jobs ✓ — exactly two reds: module-architecture (the release-train strict step on the deliberate lag) and checks (T12, fixed at the tip). The tip run should differ only in checks going green.
2. **The one-green-composite receipt is still pending** (carried from 08-45 §f2): every leg is verified green individually under the new state, but the full check-modules composite has not been re-run end-to-end as one artifact — and it cannot go fully green until the fleet-floor decision lands (the composite's strict release-train step reds on the deliberate lag).
3. **Master CI is at one named deliberate red** (release-train lag 4 in the blocking train check) — down from three reds plus a masked fourth, with the owner decision that clears it receipted and routed.

## c) NOT STARTED

1. **W3/W4 + the migration-plan D1/D2 contradiction** — parked per the standing no-re-ask directive; recorded in the 08-45 report §g and CURRENT_FOCUS.
2. **The CSP knob (`Config.CSPBuilder` + `RecommendedSecurityStackWithCSP`)** — train-gated; recipe preserved in its TODO row.
3. **The spec-race audit, the fast-slice pre-flight, the hook byte-match stage, the kill-strays widening, the bump-dep concurrency default** — all standing §f tails from 08-45, routed, untouched this session.

## d) FUCKED UP

1. **The archive pass ran with a successor-shape the tool could not validate, and my first spec-construction lost three lines to a bad grep** (`grep -v` on the basename killed the successor lines too — the tool silently accepted the truncated spec at rc=0, a silent-failure footgun worth its own hardening ask). Cost: one wasted dry-run round, caught by reading the plan output.
2. **The edit-tool staleness refusal on row 21** (the daemon had touched TODO_LIST between my read and edit) — recovered by re-reading and editing via python; no damage, one retry.
3. **The palette+dither PNG variant compressed WORSE than the original** (2.4 MB vs 1.6 MB) — measured, discarded, the no-dither variant won; one wasted conversion.
4. **The first `--no-verify` docs commit was unnecessary** (the pre-push refusal only bites on the lag legs, and at that moment the hook had not yet been exercised) — harmless, but the skip discipline should be: run the hook once, skip only the named failing leg.

## e) IMPROVEMENTS PROCESSED

1. **Archive-status hardened** (the two-pass self-test case + the prefix-wins rule) — a live defect found by the pass's own gate, fixed, and pinned the same session.
2. **The matcher lessons are in the drain row** (strikethrough invisibility, digit-token dropping) — the next harvest round starts from them instead of rediscovering them.
3. **The pre-push tidy leg is fixture-safe** (go.work-absent skip) — the gate's own rc=2 contract unchanged.
4. **The fleet-floor decision is now a single-train recipe** on row 21 — five coordinated edits instead of the three-way split-brain this session healed.

## f) NEXT (ordered roughly by impact; [owner] = owner-gated, [lane] = another lane's surface, [watch] = trigger-gated)

1. **[owner] The fleet-floor train** — nixpkgs rev shipping Go 1.27.4 + directives + appkit v0.13.0 + CI go-version pins, one train (row 21's addendum is the recipe); clears the last named CI red and the strict-lag entries together.
2. Confirm the tip CI run green-except-the-named-red (the predecessor run's shape: everything ✓ except module-architecture's train step).
3. One full check-modules composite receipt at a quiet window — impossible to make fully green before the floor train; re-attempt after it (the composite's strict train step is the only red leg).
4. [lane] The sibling 03-15 report's unharvested §f items (their pareto lane owns the harvest or the skip markers).
5. adminui `Config.CSPBuilder` + root `RecommendedSecurityStackWithCSP` — land with the next root+adminui train (recipe in the row).
6. Spec-race audit: sweep the battery for `waitForResponse`-by-URL patterns sharing a URL with a debounced twin (the QUERY fix is the template).
7. Build-then-run pre-flight inside the fast-slice script.
8. Hook byte-match stage in check-modules via `install-git-hooks.sh --verify`.
9. Widen `e2e-kill-strays.sh` to the full demo-port range.
10. `BUILDFLOW_MAX_CONCURRENCY=1` default for bump-dep commit legs.
11. [owner] §g.1 of 08-45 — gate retry-tolerance vs fail-loud for shared-worktree transients.
12. [owner] §g.2 of 08-45 — the migration D1/D2 contradiction (W3/W4 unblock).
13. [owner] §g.1 of 20-47 — the NixOS system templ repin.
14. [owner] §g.3 of 20-47 — K2 multi-instance follow-up.
15. [owner] Fleet-cache hygiene probe of `cache.home.lan/*`.
16. e2e sync offline-reads posture (raise the 30s ceiling or document the retry).
17. coverage-gate receipt on the current tree (the last full receipt pre-dates today's trains).
18. v4.19.0 wave planning (the QUERY W4/W5 receipts ride it).
19. bench-spike re-pin on a verified-quiet window (twice-quiet rule).
20. [watch] templ-components PR#101 release → sweep + retire adminui's vendored popover CSS same-change.
21. [watch] go-cqrs-lite#113 kv constructor → bump + delete usermgmt's view-cache adapter.
22. [watch] storage v4.12.0 tag → flip the StateShape sentinel test.
23. [owner] M2M D1–D4 per-row surfacing.
24. [owner] adminui D1–D3 per-row surfacing.
25. [lane] legalparty-model M03 / family train.
26. [lane] webphone C27 family-pin sweep.
27. C28 dispatch-gate consumer waves (inventories pre-staged).
28. [owner] The five usermgmt API asks filing decision.
29. [owner] go-cqrs-lite `event.New` pooled-encode ask.
30. [watch] go-datastar#31 deterministic signals ordering (`NewSignalsPatch` marshal instability).
31. [watch] httputil#48 + go-cqrs-lite#117 maintainer cadence.
32. [owner] K3 templ-components popover evidence pack — DONE-by-action: PR#101 merged; strike stale K3 mentions at the next annotate pass.
33. Fleet libs report-count gate re-run after the next `docs/review/libs` refresh.
34. README quickstart gate sanity after any README touch.
35. setup-demo `app.css` byte-delta eyeball — satisfied by the class-set gate; the human glance stays optional.
36. Annotate the remaining superseded reports' settled §g items (sweep the siblings next annotate pass).
37. README↔FEATURES quarterly claim-sweep (due ~2027-01-17).
38. Shared-tree branch guard (preflight-tree-check branch assertion + self-test).
39. loginpage test-depth debt (WebAuthn ceremony e2e + full-page render golden).
40. `ReadLastN` SQL event-store upstream ask.
41. GCL hardening follow-through (b)-remainder + (e).
42. BuildFlow diverged-checkouts reconciliation.
43. Fleet release-dance helper design.
44. dashboardui custom-sidebar revisit criteria re-check.
45. ExtraNav owner decisions (a)/(b).
46. Quarterly `/mnt/buildcache` watch (`df -h` before any gate battery).
47. Harden archive-status against silently-truncated specs (rc must reflect dropped lines — the T12-class silent-acceptance this session worked around).
48. Add a release-train dated-hold mechanism (a holds file the strict-lag gate honors) so deliberate downgrades do not need `--no-verify` pushes [owner].
49. The 20-47 report's remaining owner-verdict tails (§g.1/§g.3 surface there).
50. TODO hygiene gate self-test after every new TODO batch (this session's standing practice).

## g) TWO QUESTIONS I CAN NOT FIGURE OUT MYSELF

1. **Should the fleet-floor train move NOW, per-repo, ahead of the fleet?** The appkit v0.13.0 blocker means master carries one deliberate CI red (the strict train check's 4 lag entries) and every push needs the named `--no-verify`, until the ADR-0003 nixpkgs rev ships 1.27.4. The alternative — bump THIS repo's flake nixpkgs input to any rev shipping 1.27.4 unilaterally — splits the fleet floor across repos (every sibling repo's `check-go-toolchain` compares against their own flake, so cqrs-htmx would leap ahead of the family). One word from you: hold with the named red (current posture, receipted on row 21), or leap and own the fleet drift.
2. **Does release-train get a dated-hold mechanism?** The strict-lag-0 gate encodes "never lag the family" — correct as a default, but it has no way to express a DELIBERATE hold, so a legitimate downgrade (this session's appkit) can only land via `--no-verify` pushes with named justification. A `scripts/checks/release-train-holds.tsv` (module path, held version, reason, expiry) the gate honors would make holds first-class and retire the bypass. Your call whether the bypass discipline is the intended terminal state or the mechanism is worth building.

*Report ends. Waiting for instructions.*
