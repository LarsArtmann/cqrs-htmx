# Round-13 Plan — T06 Close + Battery Tail (2026-10-01 15:55 CEST)

> **ANNOTATED 2026-10-04 (docs-health round 17):** §d2 (T08) resolved to its local half the same evening (rebuilt binary: 0×C040 phantoms, 14/14 gate modules — fleet swap owner-pending, TODO D4); §d3 (T09) closed 2026-10-01 evening (3 filed / 2 retired); §e2 rawIDToken executed per the recorded decision (TODO D3 tick pending). Still open (routed): §d1 bench-spike (verified-quiet window), §d4 the next release train carrying the erraudit + loginpage cluster (deferred deliberately in the round-15 reports), §d5 the /mnt/buildcache reclaim (D5, urgency down at 72%). Archived this pass.

> **Session:** continuation of `docs/planning/2026-10-01_06-47_pareto-round13-superb-execution-plan.md` after the W0–W3 (10:30) and round-14 (12:30) reports. Owner directive: execute-and-verify one step at a time. Branch `master`, tree clean, all work committed (daemon-carried heuristic commits throughout — contents verified per commit).

## a) Fully done

| Item | Evidence |
|---|---|
| **T06 — the erraudit `context_loss` program CLOSED** (the plan's declared completion criterion, first half) | Current buildflow binary's inventory: **34 sites** (usermgmt 18 — es_readmodel 1, sql_readmodel 1, sql_readmodel_extra 6, totp 1, webauthn_service 7, service_oauth2_extracted 2; dashboardui 10; identity-model authz_roles 3; oauth2 provider 3). 29 fixed with family-preserving context (`Wrapf(err, Classify(err), …)` / `WithContextAny` / `WithContext`), 5 suppressed-with-reason: the rawIDToken trio (live credential — never echoed; the W0–W3 §g1 decision, now executed), 2 helper-mediated bare returns (context attached inside `decodePayload`/`deleteViewOnTombstone`), 1 success-path return the detector misreads. **0 erraudit criticals across all 28 modules** (per-module `buildflow -s erraudit@<m>` sweeps, before AND after the formatter reflowed a file under the directives). Four modules: build+vet+`-race` tests+`golangci-lint` all rc=0. `//nolint:erraudit` semantics source-verified in erraudit's suppression engine BEFORE relying on them: tool name + `legacyerrors` analyzer name both honored, comma lists work, a directive covers its own line + the statement-start line (which is why golines reflowing `provider.go` kept every suppression live — verified empirically). |
| **The 61-vs-54 count discrepancy resolved by supersession** | The 61/55/25 numbers came from a since-deleted /tmp CLI extractor paired against an older erraudit SDK; the current in-process SDK flags MORE (it found webauthn_service ×7 the old inventory never listed). The gate's own per-module counts are the only comparable series; the historical numbers are recorded as superseded in TODO_LIST + AGENTS gotcha 8. |
| **T03/T04 battery — e2e leg DONE** | Playwright suite **70/70 passed (rc=0, 1.7m)** incl. axe sweeps, screenshot captures (light/dark/mobile × 10 pages), and the offline-sync specs. The snapshot-sentinel change (`nil,nil` → `snapshot.ErrSnapshotNotFound` in the e2e server) is proven by the green run over the `/snapshots` pages. 9 refreshed screenshots committed (`9d7b7799`) — pixel drift confined to events/overview/projections. |
| **T22(i) — ExtraMiddleware pin under RunWithAppkit** | `TestRunWithAppkit_ExtraMiddlewareComposesInsideSecurity` (setup/run_appkit_test.go): real appkit listener, nil handler (the path only RunWithAppkit owns — it builds Mount+Middleware itself), asserts the consumer extras' header AND the security layer reach responses. First probe used `/health/ready` — served by appkit OUTSIDE the bundle chain (headers absent) — corrected to a bundle-owned route; the test now documents that boundary implicitly. setup `-race` suite rc=0; CHANGELOG [Unreleased] Added. |
| **T19(a) — agents-notes narratives** | Two dated histories assembled from the archived source reports (no invention): the cqrs-lint three-pass arc (easy-way green → root-cause fix at the linter source → self-audit catching the fix repo's own docs debt) and the three-tools-vs-nixpkgs-default-Go story (one floor re-pin, three silent-assumption failures, three shims with queued upstream asks). Docs micro-batch item (a) struck in TODO_LIST. |
| **T26 — v5-window watches** | appkit master re-verified live (`048137e90`, v0.7.0 tag present; adoption stays v5-window per ADR-0052); ProjectionLayer v5-removal inventory re-confirmed accurate; DataStar Tier-4 demand gate re-verified (no new evidence); **buildcache watch: intraday 58% → 98% swing (5.5G free)** — annotated on the 08-30 decision file + TODO hardware-watch line; `df -h` before any gate battery is the standing rule. |
| **W10 — check-modules composite re-run discharged** | `nix run .#check-modules` **rc=0, all 27 stages green** (including both self-tests verified standalone) — the composite re-run owed since W0–W3, executed under load 80+ and still green. |

## b) Explicitly not done (with reasons)

- **Bench-spike: honest refusal** (OQ16 posture). Load 79–86 across the session window (parallel flake/nixpkgs session); the gate requires < 6 verified twice. Refusal recorded in the TODO battery item; retry at the next quiet window.
- **T20 planning-corpus triage: already completed** by the parallel docs-health round-14 pass (upstream-asks draft archived, index + distribution draft annotated, three owner-gated decisions confirmed leave-in-place, HTML keep-in-place decision recorded in `docs/status/README.md`). No duplicate work performed; verified by reading the round-14 annotations.
- **Owner-gated/blocked items unchanged:** T08 (system cqrs-lint rebuild — binary still pre-fix), T09 (5 upstream asks — filing authorization pending), T10/T16 (blocked on T08), T11/T12 (cross-repo owner gate), T18(g) feedback-inbox checker (owner disposition), T23–T25.

## c) Fucked up (honest incident log — all recovered)

1. **Daemon races on every commit** (5 of 5 commits landed as heuristic `chore: auto-commit` carriers while the 73–152s BuildFlow pre-commit ran): each carrier was `git show`-verified before building on it; zero content lost. The pre-commit hook PASSED on the two code commits (no `--no-verify` needed — the erraudit fixes removed the findings layer that kept it red).
2. **The T22(i) test first asserted against `/health/ready`** — appkit serves it outside the bundle chain, so the extra-middleware header was (correctly) absent. Root-caused in one red run, probe moved to a bundle-owned route. The lesson is the test's value: the appkit/bundle boundary is real and now implicitly documented.
3. **A thinking artifact leaked into the TODO_LIST tombstone** (a mid-sentence "wait, corrected below") — caught on immediate re-read and rewritten via script before commit.

## d) Next tasks (ordered)

1. **Bench-spike** at the next verified-quiet window (load < 6 twice) — the last battery item.
~~2. **T08** (owner): rebuild the system cqrs-lint binary → C040 root-walk verification → retires the gotcha-13 caveat and unblocks T10/T16.~~ done 2026-10-01 evening LOCAL HALF (rebuilt binary from b06ac8add5e8: 0×C040, 14/14 gate modules); fleet swap owner-pending (TODO D4)
~~3. **T09** (owner authorization): file the 5 fleet upstream asks (drafts ready per the plan's §2 row).~~ done 2026-10-01 evening (3 FILED: treefmt-nix#545, a-h/templ#1449, BuildFlow#29; 2 RETIRED with evidence at the verify-before-filing gates)
4. **Next train** carries the erraudit context fixes (published module code) + everything in CHANGELOG [Unreleased].
5. **Watch the disk**: 5.5G free on /mnt/buildcache — the /tmp fallback layout is the expected next step; reclaim decision (rust/ 155G + sccache/ 20G) stays with the owner.

## e) Questions I cannot answer myself

1. **Disk reclaim authorization:** /mnt/buildcache is at 98% (5.5G free). The two reclaim candidates (rust/ 155G, sccache/ 20G) are other ecosystems' caches — deletion needs your call (the 08-30 decision doc carries the analysis).
~~2. **rawIDToken suppression executed as decided** (W0–W3 §g1): confirm-and-close, or do you want the three sites re-shaped differently?~~ executed 2026-10-01 (suppress-with-reason, LIVE-SECRET class); owner tick = TODO D3
