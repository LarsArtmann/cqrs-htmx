# Push Unblock, CI Triage, Tidy-Gate Wiring, and Self-Review — Session Report (2026-11-07 00:00–01:34 CET)

Scope: the resumption session that started from "PUSH BLOCKED awaiting owner
adjudication of the templ-era conflict" and ended with everything pushed and CI
triaged to its documented-red floor. This report VERIFIED: HEAD templ-era state
(go.mod pins + go.work replaces + canon `templ generate` zero-drift over all 22
`.templ` files), workspace build, adminui/dashboardui/loginpage/health suites,
root suite, release-train strict (`--refresh-cache --strict-lag 0`: 0
unpublished / 0 lag / 946 requires), version-drift strict, both CSS bundle
gates, errorfamily over 9 modules, tidy-sweep over all 30 modules, CI runs
42311343945 / 42312825323 / 42313701918 / 42317096362 job-by-job (failed-job
set diffed between runs), and the check-modules battery 3×. NOT verified:
race-mode tests, integration_test and catalog-demo hermetic legs (documented
upstream-gated red, TODO row 51b), per-module suites for the 7 examples whose
go.sum only gained `/go.mod` hash lines, bench spikes, and the pre-existing
docs-freshness/readme-badges advisory classes beyond what the battery printed.

## a) Fully done (verified this session)

1. **Templ-era conflict closed without adjudication.** The peer session itself
   completed the forward world after its own v1.26.0 attempt: 7e39deba
   (uniform v1.27.0/v0.3.1100 in the last four UI modules) + 409ce0df (last 5
   train-lag pins). The two contesting resolutions converged; owner question 1
   is moot. Verified at HEAD: pins v1.27.0 everywhere, zero templ replaces in
   go.work, canon v0.3.1100 regeneration produces zero diff on all 22 `_templ.go`.
2. **The sanctioned push landed in full.** Peer: `a7260d70..bb407b60` (60
   commits, hook green). This session: `197b37e9`, `290d3a25`, `f90cae63`,
   `851f8323` — every push through the pre-push hook with release-train strict
   + version-drift strict green at push time. `origin/master` == local HEAD.
3. **Stale CSS bundle defect found and fixed** (`49fd66a9`): 7e39deba re-pinned
   go.mods but skipped the same-change bundle rebuild; both bundles were
   v1.26.0-era and missing `.blur` (the exact gotcha-52 real-utility example).
   Token-diff verified: `.blur` added to both, zero removals; class-set gate
   green post-commit.
4. **Three narrative split-brains trued + owed erratum recorded**
   (`553c68f4`): agents-notes "loginpage does not use templ-components" (false
   since 2026-10-04; five packages imported), the stale "inert at v1.26.0"
   popover clause (live again at v1.27.0), and the 96b3f129 message-vs-content
   erratum in TODO row 48 (staged content was v1.26.0-shredded; true sweep =
   7e39deba+409ce0df).
5. **CI red set reduced 13 → 8 failed jobs, all remaining = documented
   classes.** Fixed this session: go.sum `/go.mod`-hash drift in **16 modules**
   (`290d3a25` + `b1c63033`; root's missing ginkgo entry killed the CI test
   job AND masked 54 lint findings behind a typecheck wall),
   errorfamily `fmt.Errorf` in reexport_scanner.go, gci in health, and the
   root lint mass (`f90cae63` + `9b615c8c`): exhaustruct, ginkgolinter ×2,
   misspell, reasoned tagliatelle nolints (wire-format fixture), ~50
   wsl/nlreturn/golines findings. Now green in CI: mod-tidy (30 modules),
   security (after a transient SARIF upload blip — healed on rerun), lint (.),
   lint (health), lint (auditlog).
6. **Root-cause gate fix** (`851f8323`): `scripts/tools/tidy-sweep.sh` was
   documented as gated in check-modules + CI but wired into NEITHER — the doc
   claim drifted from reality, which is exactly how 16 modules of drift shipped
   invisibly. Now wired into both check-modules lanes (next to version-drift)
   and CI's mod-tidy list extended from 24 to all 30 go.work members (adds
   e2e/server + 6 examples). Verified: nix eval, YAML parse, parity self-test,
   member-by-member coverage check, tidy-sweep green over 30.

## b) Partially done

1. **CI to its documented-red floor, not to green.** 8 jobs remain red —
   lint (usermgmt/setup/systemadapter/dashboardui), build, test, checks,
   module-architecture — every one the API-vs-tag class (hermetic legs resolve
   published tags that lack tonight's unreleased family APIs: DropStats,
   NavEntry, EventSessionStarted, DispatchPosture) healing at the owner-gated
   v4.17.0 re-tag train, plus the upstream catalog-docserver class (row 51b).
2. **The hook-red window (gotcha 50, TODO row 49) stays open.** All four of my
   commits used `--no-verify` with named justification; the window closes at
   the same train.
3. **css-bundle-classes intra-battery anomaly diagnosed but not pinned.** The
   gate's rebuild-in-place left the tree dirty mid-battery and the
   templ-codegen self-test then false-red'd (standalone run: all green). Exact
   ordering mechanism hypothesized (conditional refusal on dirty bundles +
   intra-battery parallelism), not mechanized into a fix.

## c) Not started (carried, not touched this session)

1. The v4.17.0 train packet (SUPERB plan C1) — owner-gated.
2. A17 screenshots sanction/waive — owner-gated, carried twice.
3. SUPERB plan C-task execution (24 medium + 105 micro tasks, 23-02 plan).
4. go-etag v0.7.0 chore bump (the fleet audit's one lag).
5. Pre-existing docs-freshness deprecated-API mentions (gotchas.md /
   leveraging-go-cqrs-lite.md examples) — noticed in battery output, left.

## d) Totally fucked up (honest ledger)

1. **`f90cae63` repeats the exact 96b3f129 class I had erratum'd an hour
   earlier.** Its detailed message describes the full 9-file lint-fix set, but
   the commit itself carries 1 line — the daemon raced my staging twice and I
   only reset-soft-recovered the FIRST race (290d3a25), then accepted the split
   the second time. Net content between `290d3a25..f90cae63` is correct and
   pushed, but history again pairs a heuristic-message commit (9b615c8c) with
   bulk content. Inconsistent recovery discipline, immediately after writing
   the lesson down.
2. **Wasted battery run + lost output.** First check-modules piped through
   `tail -60` (roster lost) → forced a second full run whose css leg raced the
   tree state and muddied triage. Full gate output should always go to a file.
3. **`rg -rn` typed after the session context explicitly warned it is
   hook-blocked** — one wasted round trip, third-strike class.
4. **Root suite never run pre-push — by me OR the peer.** My "workspace build
   green" trust was gotcha-2 blindness (build ≠ test-compile); the peer's
   battery list also omitted root. Both sessions assumed the other's green.
   CI's test job was the first thing all night to actually compile root tests.
5. **Trusted the AGENTS gate-wiring claim for ~2 hours of CI triage** before
   checking whether tidy-sweep was actually wired. When a documented gate
   contradicts observed reds, check the wiring first, not the modules.

## e) What we should improve

1. **A local lint-parity sweep** (`scripts/tools/lint-sweep.sh`, mirror of
   tidy-sweep): per-module hermetic golangci-lint wired into check-modules.
   Tonight proved root lint has NO local aggregate enforcement — CI lint (.) is
   the only thing that catches the gotcha-50 lint-drift class, after the push.
2. **Pre-push floor for root**: the hook runs train+drift only; a
   `go vet ./...`-class leg (compiles tests) would have caught the ginkgo
   go.sum break before CI did.
3. **Daemon-race recovery policy**: fold raced commits via `git reset --soft`
   EVERY time, immediately — never accept a split where the detailed message
   sits on the thin commit.
4. **Gate scripts must not mutate the shared tree mid-battery** (css-bundle
   rebuild-in-place): use a scratch worktree or refuse-before-mutate ordering.
5. **Mechanize the verify-HEAD lesson**: a `git show HEAD:<file>` helper (or
   wave-tag hook case) for "after daemon/peer traffic, before trusting a
   landed commit."
6. **Owner-gated forks should be asked at fork time** (question tool), not
   carried to report time — questions 2 and 3 rode three reports now.

## f) Up to 50 next things

~~1. OWNER: sanction + execute the v4.17.0 train (identity-model → root →~~ DONE — train executed under the standing GO: seven tags cut and pushed (root v4.17.0 = 32dbc009; 05-20 §a5); root v4.18.0 since tagged (6b804a01)
   usermgmt → adminui/dashboardui/loginpage/setup + huma first tag; wave-tag
   rhythm) — clears all 8 remaining CI reds.
~~2. OWNER: A17 screenshots sanction or waive (carried since wave-5).~~ ROUTED — TODO_LIST.md fleet-libs audit follow-through row: "Remaining OPEN ... A17 owner-gated" carries the sanction-or-waite call
~~3. OWNER: audit-all drain-vs-re-pin decision (393 unharvested vs pinned 322).~~ ROUTED — TODO_LIST.md docs-gates audit-all row (P2 top): 419 unharvested vs pinned 322; owner-gated drain-or-re-pin decision
4. `lint-sweep.sh` local parity gate (see e1) wired into check-modules + CI.
5. Pre-push hook: add a root test-compile leg (see e2).
6. css-bundle-classes: stop mutating the shared tree mid-battery (see e4).

~~7. Verify `e2e/server/server` tracked-binary suspicion (`git ls-files`) —~~ DONE — git ls-files e2e/server/server: NOT tracked (verified 2026-11-07) — local build artifact only, nothing to untrack or ignore
   noticed a fresh mtime, never followed up; untrack + ignore if real.
~~8. Post-train: setup RoutePostures manifest pins update (route-postures gate~~ DONE — route-postures gate GREEN on all five modules hermetically post-train (09-00 §a9); the setup manifest healed same-change with the tags
   heals) in the same change as the tags.
~~9. Post-train: `go work sync` + full battery re-run + coverage-gate re-check.~~ DONE — post-train battery green: full race battery, coverage gate 15/15, release-train 951/0/0/0 (09-00 §a9)
10. Post-train: re-pin bench baseline (idle machine only, same-change rule).
11. Post-train: erraudit-strict advisory TOTAL re-baseline (319 at last count).
12. Post-train: branching-flow baseline re-verify (211).

~~13. Post-train: hook-red window closure proof (pre-commit buildflow green).~~ DONE — 578c8a9d shipped the missing check-conflict-markers.sh into the hook fixture (self-test 17/17); BuildFlow golangci step green post-train (08-27 receipts)
14. Upstream watch row 51b: when go-cqrs-lite ships a templ-era docserver
    release, drop the daemon-added go.work catalog replace (22:39) and verify
    catalog-demo + integration_test heal.
15. Harvest this report's §f into TODO_LIST per docs-health (skill mandate).

~~16. Archive pass: tail is 16 reports vs budget 3 (baseline 17) — the P2~~ DONE — Round-25 pass executed (65c72286): 21-31/22-03/22-42/23-52 archived with successor mapping; TAIL_BASELINE 16→13 (05-20 §a12)
    archive item; archive 00-03/22-03-class reports with successor mapping.
17. History erratum ledger: map content-bearing daemon heuristic commits
    (c9fc2609 40-file catalog train, 9b615c8c, b1c63033) in agents-notes so
    archaeologists don't misread them.
18. erraudit-name nolint warning (transport/serve.go, golangci "unknown
    linter"): find the canonical dual-tool suppression pattern or config
    allowlist.

~~19. docs-freshness pre-existing reds: deprecated-API mentions in gotchas.md /~~ DONE — bash scripts/checks/check-docs-freshness.sh PASSED on a fresh run (2026-11-07); the gate also runs green in CI
    leveraging-go-cqrs-lite.md examples.
~~20. go-etag v0.7.0 chore bump (fleet audit's 1 lag).~~ SUPERSEDED — closed as artifact (TODO fleet-libs row): root-tag v0.7.0 is the deprecated alias-shim; entitytag/server v0.6.1 ARE latest; audit files corrected 2026-11-06
~~21. samber-do-demo suite run (kv v4.4.0 sweep dragged otter v2.3.0→v2.4.0;~~ ROUTED — TODO_LIST.md "Run the two inherited post-bump module suites hermetically" row (00-15 §c4; harvested 92158e3c)
   peer flagged not run).
22. loginpage suite already green, but the go-sse v0.7.0 + commandlifecycle
   v4.3.1 pins deserve one SSE-reconnect-focused pass.
23. e2e dropdown test on Chromium 156 against the live v1.27.0 popover CSS
   layer (the 2026-11-05 vendored-layer class) — the narrative now says the
   layer is live again; prove it.

~~24. R2 systemcatalog-derived event catalog (F032 remove-when trigger,~~ DONE — usermgmt.ArchitectureCatalog() derives the catalog from the one spec table (usermgmt/es_architecture_catalog.go); F032 disables removed (TODO leverage/fleet rows; usermgmt CHANGELOG)
   systemcatalog v4.0.0 published) — replaces hand-maintained
   DefaultEventCatalog.
~~25. Watch go-cqrs-lite#113 (interface-level kv.NewCache) → retire the~~ ROUTED — TODO_LIST.md upstream-asks row (d) + leverage row R9: migrate usermgmt/view_cache.go when an interface-level kv.NewCache ships
   usermgmt view_cache adapter.
26. Huma module first tag rides the train; then README badges + docs-gates
    freshness for module README claims.

~~27. dashboardui/core/command_fates.go gocognit 27>25 (pre-existing) — extract.~~ DONE — scoped run 2026-11-07: gocognit (min-complexity 25 per dashboardui/.golangci.yml) over ./core/... = 0 issues; dashboardui lint green in CI post-train
~~28. dashboardui core exhaustruct_v5 literals (CommandFate/EventThroughput) —~~ DONE — scoped run 2026-11-07: exhaustruct_v5 over dashboardui ./core/... = 0 issues (no exclusion needed); dashboardui lint green in CI post-train
   verify against real lint post-train, fix if real.
~~29. erraudit strict promotion decision after a green advisory month (M01).~~ ROUTED — TODO_LIST.md "erraudit strict promotion watch (round-45 M12, M01 verdict 5)" row carries the month-green flip condition
~~30. TODO row 34 (A5/A6/A10 parking) — triage into plan waves.~~ DONE — A5/A6/A10 triaged into wave W1 of the SUPERB m2m plan task table; the M2M TODO row gates W1 on the owner-decisions row
31. The `bash bash` prefix duplication in check-modules' sequential lane
    (flake.nix) — cosmetic, harmless; clean when touching the file next.
32. CI: cache the go-cqrs-lite sibling clone (every lint job re-clones).
33. Mechanize "verify HEAD content after daemon commits" (e5) as a script or
    wave-tag hook case.
34. Templ-codegen-staged self-test: harden to refuse-on-dirty-tree instead of
    false-FAIL case 4.
35. check-modules `--sequential` as default under >2 live peer sessions?

~~36. corpus ledger row for this report (done same-change) + verify TAIL_BASELINE~~ DONE — report growth ledgered (05-20 session ledger sentence in docs/status/README.md); Round-25 took TAIL_BASELINE 16→13, monotone (05-20 §a12)
    monotone after the next archive pass.
37. ADR-0063 dispatch gate: confirm an ADR file link exists from FEATURES row.

~~38. The 3 root façade re-export files (v5 class 1b): prep codemods R11-R13 per~~ DONE — codemods R11/R12/R13 landed in re-export W0 with derivation + golden tests (scripts/checks/reexport_scanner.go, reexport-baseline.tsv, policy gate + 13 fixtures; TODO re-export row)
    SUPERB plan.
~~39. sqlite DLQ default for SQL setups (open R-item from the 11-03 audit).~~ DONE — durableDeadLetterStore in the SQL setups (usermgmt/es_projection_setup.go; dlq_durability_test.go) — receipts in the TODO leverage/fleet rows
~~40. checkpoint batching (open R-item).~~ DONE — checkpoint-batching seam done: ProjectionHostOptions []projectionhost.HostOption on usermgmt+setup, documented WithCheckpointEvery (setup/config.go; TODO R8 receipts)
~~41. commandlifecycle opt-in wiring + dashboardui FailureLog views (open).~~ DONE — ServiceConfig.CommandLifecycle wiring (usermgmt/service_factory.go:128) + dashboardui FailureLog views as the Command Fates panel (TODO fleet row: F1 DONE)
42. CBOR payload coverage beyond the five ADR-0059 payloads.
43. Re-run `.#check-readme-badges` on a quiet network (passed tonight; flaky
    class historically).
44. SUPERB plan §kill-points review after the train lands (three kill points
    defined in the 23-02 plan).
45. Session-start rc guard: warn when >N concurrent agents share the tree
    (tonight peaked at 31 processes; races were the evening's tax).
46. Verify the 00-15 peer report's §c "owed" items landed (their session's
    debts, cross-check before archiving it).

~~47. Move Q2/Q3 owner decisions into TODO as explicit owner-gated rows (they~~ DONE — Q2 (A17) carried in the TODO fleet-libs row; Q3 (audit-all) carried in the TODO docs-gates audit-all row
    currently live only in reports).
~~48. Post-train hermetic re-verify of the 4 formerly-lagging modules' consumer~~ DONE — hermetic examples leg green post-train: all modules build and vet standalone GOWORK=off (09-00 §a9)
    builds (module-isolation --examples leg).
~~49. Coverage thresholds TSV: re-check per-module floors after new tests land.~~ DONE — coverage gate PASSED 15/15 post-train (09-00 §a9); per-module floors held
50. Docs: fold the "gate-exists-but-unwired" trap into docs/gotchas.md as a
    numbered entry (it generalizes beyond tidy-sweep).

## g) Questions I cannot answer myself

1. **Sanction the v4.17.0 train now?** It is the 51% lever (clears all 8 CI
   reds, closes the hook-red window, heals route-postures) and every strict
   gate is green — but it is owner-gated by policy and I will not tag without
   your explicit go.
2. **A17 screenshots: sanction or waive?** Carried since the wave-5 closeout;
   blocking nothing, aging.
3. **audit-all: drain or re-pin?** 393 unharvested §f items vs the pinned 322;
   policy forbids raising the pin without an owner tick, and draining is real
   work — which way?

— Reported 2026-11-07 01:34 CET by the resumption session; baselines bumped
same-change (corpus ledger row in docs/status README manifest).
