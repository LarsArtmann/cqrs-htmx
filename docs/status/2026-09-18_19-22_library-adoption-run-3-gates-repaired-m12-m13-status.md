# Status — Sibling-Library-Adoption Execution, Run 3 (Gates Repaired + M12/M13 Landed)

**Generated:** 2026-09-18 19:22 CEST
**Session scope:** continued the Pareto plan (`docs/planning/2026-09-17_13-42_pareto-execution-plan-sibling-library-adoption.md`) after the two prior sibling programs (verification-battery, dashboardui-adoption run 2) finished and handed the tree back.
**Verified state at writing:** tree CLEAN, all session work committed (partly via daemon heuristic commits after two hook-failure races), workspace build green at `go 1.26.7`, pre-commit hook passes for clean diffs (proven on `935f4b36`), `nix run .#check-modules` GREEN, findings gate restored to `fail_on: critical`, dashboardui/setup/root test suites green on HEAD.

---

## a) FULLY DONE

| Task | Outcome | Verified by |
| --- | --- | --- |
| **State reconstruction + directive war forensics** | Found root go.mod at 1.26.7 after sibling revert `19a37e9f`; witnessed a 5th phantom bump to 1.27.1 mid-session (absorbed by daemon `e7056cc6`); restored (5th, `0b3facd6`). **Root cause documented in AGENTS.md and then CORRECTED**: go-appkit's root go.mod says `go 1.27.1` (its 8 submodules say 1.26.7), but the replace-based propagation vector is NOT the active one — a module-cache scan proved ZERO cqrs-htmx dependencies declare `go 1.27`, so the bumper is a 1.27.1-toolchain process (sibling sessions' go commands or gopls), outside repo control. | `git log -p go.mod`, module-cache rg scan, AGENTS.md update `307091b8` |
| **Workspace sanity** | `go build ./...` green at 1.26.7 immediately after each restore; the 59 phantom gopls diagnostics (examples "go mod tidy" complaints) identified as the documented stale-LSP class, never acted on. | CLI build, AGENTS.md LSP-unreliability rule |
| **check-modules → GREEN** | Was failing on 2 axes; both fixed: (1) setup dep budget 22→23 with justification comment (`go-cqrs-lite/command/v4` direct since `ServiceConfig.CommandMiddleware` threading); (2) completed the templ-components **v1.18.0 family drift sweep** left half-done by the concurrent wave: integration_test caught up (9 requires: templ-components ×5, go-health v0.2.0, go-health-dashboard v0.9.0, go-atomic-write v0.5.2, go-retry v0.7.1), usermgmt + systemadapter raised go-retry v0.5.0→v0.7.1. All 5 tags verified pushed via `git ls-remote` BEFORE sweeping. Gate re-run: all module architecture checks pass (275 markdown links verified too). | `nix run .#check-modules` exit 0; hermetic tidy+build+vet ×3 modules; commit `45d795b0` |
| **M07: buildflow binary refresh** | `buildflow upgrade` has no network (GitHub API blocked) → built from the local BuildFlow repo HEAD (`7d3d967`) into `/home/lars/go/bin/buildflow`, which PATH-shadows the nix-profile binary (verified: `~/go/bin` precedes `~/.nix-profile/bin`). Required downloading the go1.27.1 toolchain via the module proxy (BuildFlow's go.work needs ≥1.27.1). `buildflow doctor` binary-freshness: **✓ green** (was: "built at 42fd89b but HEAD is 7d3d967"). Also cleared 2 of 9 gomod-freshness warnings (hermetic tidy of examples/system-demo + setup; the 7 verify-tag fixtures stay deliberately untidy). | doctor before/after output; commit `731eef24` |
| **M08: findings gate RESTORED to `fail_on: critical`** | Two real root causes found and fixed: **(1) The suppression config was NEVER read by any session** — authored as `.go-structure-linter.yml` but the linter auto-discovers `.go-structure-linter.yaml`; the 51 "false positives" phantomed through every triage since 2026-09-17. Renamed (`git mv`), added the `go-version` suppression with the already-documented justification (flake owns toolchain policy; removal condition = TODO_LIST P2 decision landing). **(2) The stale binary** carried the old rule set (root-package-files ×50). Fresh binary + config = **5 findings total (2 warnings, 3 infos, 0 errors)**. Full fast battery: **236 steps, 0 failed, exit 0 at critical**. | `BUILDFLOW_NO_RESULT_CACHE=1 buildflow --build-mode fast` exit 0; `.buildflow.yml` comment block rewritten with history + caveats |
| **Hook repair — first normal commits of the day** | The single deterministic hook blocker: nix-fmt's shellcheck SC2164 (bare `cd`) in `check-command-bijection.sh:17` + `check-dep-budgets.sh:12/51` — "0/9 retries recovered, deterministic; fix the step". Fixed (`cd ... || exit 1`), budget script re-run green, **committed `935f4b36` THROUGH the hook (24s, no --no-verify)** — first clean hook commit of the day after every prior session used the documented fallback. | hook output "BuildFlow completed in 24s", COMMIT_EXIT=0 |
| **GOCACHE single-writer pin re-established** | The refreshed binary's CLI default (max_concurrency 4) overrides the file's `max_concurrency: 1` → the documented govalid/golangci GOCACHE race returned (2/2 fast runs failed on a random module's golangci step, each CLI-verified 0 issues; 1/1 green with `BUILDFLOW_MAX_CONCURRENCY=1`). Pinned via env in the hook template AND the active `.githooks/pre-commit` (reinstalled via `scripts/install-git-hooks.sh --force`, byte-verified). | /tmp/bffast{2,3}.log; hook template + installed hook diff |
| **M14: CLOSED AS REJECTED-WITH-EVIDENCE** | The plan's premise does not survive contact with the API: `ssetest.RequireDataJSON[T]` takes a wire-parsed `ssetest.Event` (Type/DataLines/ID/Retry — from reading a stream); our 12 hand-unmarshal sites are in-memory `json.Unmarshal(evt.Data)` on `sse.Event` (a Data **field**), and ssetest exports **no conversion constructor**. The only two hand-rolled wire parsers (`transport/serve_test.go:211`, `setup/sse_internal_test.go:187`) count heartbeat **comment frames**, which ssetest's reader deliberately ignores — adoption would delete the assertion. Zero test files import ssetest today. Forcing it would degrade every touched test. Honest negative result; doc note queued for M18. | ssetest v0.3.0 source (assert.go:86, event.go:24, reader.go:244 "SSE comment, ignore"); grep sweeps |
| **M12: dashboardui SSE-hub health card LANDED** | The observability dashboard finally shows the SSE hub itself: `core.HubHealth` DTO (Subscribers/BufferSize/Closed/Draining) on `core.Overview`; the dashboard bridge fills it from `broadcaster.Health()` (promoted method); `renderStatGrid` appends a conditional card — subscriber count in green, "draining" yellow, "closed" red, **absent entirely without EventBus** (journal-only dashboards unchanged). 3 new tests (subscriber-count render, closed-after-shutdown render, nil-absence); FULL dashboardui suite green; my files lint-clean (fixed my own gci/perfsprint/dupword findings immediately). | `go test . ./core/` exit 0 ×3 runs; `golangci-lint run` shows zero findings in my 3 files; commits `68081b16`/`2d79270e`/`73a8e438` (daemon-absorbed, content verified intact) |
| **M13: hub readiness check CODE DONE** | New `cqrshtmx.HubReadinessCheck(*Broadcaster)` in root `readiness.go`: ok while the hub accepts subscribers; fails with subscriber count + buffer size once closed or draining ("a dead feed answering 200 is worse" — the repo's fail-fast posture, matching `attachSSE`). Error constructors follow repo law (`errorfamily.Newf(event.Infrastructure, "cqrshtmx.sse.hub_*", ...)`). setup's `healthHandler()` now includes it whenever `Bundle.Broadcaster != nil` (rebuilt checks slice, nil-safe both ways). 3 root tests (healthy/fails-closed/JSON-body); setup temporary family dev-replace for root added **with removal condition** (same pattern as the existing usermgmt one — hermetic setup resolves published root which lacks the API). setup FULL suite green (12.4s), root hub checks green under race. | `go test` root+setup exit 0; hermetic tidy+build+vet setup; commits `7fd79677`/`baaaa2f9`/`dff58fad` (daemon-absorbed) |
| **Precise ownership of the new golangci surge** | The refreshed toolchain activated `exhaustruct_v5` (replacing deprecated `exhaustruct`) — the old `//nolint:exhaustruct` directives no longer match, surfacing pre-existing partial-struct findings: root ×4 (logging.go ×2, readiness.go:73, structured_error.go ×2) + dashboardui ×6 (paginationState/eventFilter in the sibling's handlers). Exactly ONE finding is mine: `varnamelen` at readiness.go:131 (`h := b.Health()`). All are warning-level — below the restored critical gate, but they make hook golangci STEPS exit-1. | /tmp/root-cli.log per-finding dump |

## b) PARTIALLY DONE

1. **M13 narrative commit lost to the daemon race — twice.** Both the M12 and M13 `git commit` attempts ran the (now slower, golangci-failing) hook while the daemon swept the staged files into heuristic commits (`68081b16`/`2d79270e`/`73a8e438`/`7fd79677`/`baaaa2f9`/`dff58fad`). Content verified intact per-file (`git log -- <file>` + `git show <rev>:<file>` spot-checks + post-absorption test runs), but the narrative history is archaeological again — the exact loss mode AGENTS.md warns about, hit despite the documented rule because the hook now runs 60-90s on failing diffs.
2. **M18 docs/memory wave** — not started beyond what AGENTS.md already got (root-cause + corrected vector analysis). Missing: CHANGELOG entries for everything above, HARVEST of the audit §f into TODO_LIST/ROADMAP, the M14-rejection note, cross-links to this report, M12/M13 adoption-table rows in AGENTS.md, and the three new tool-behavior gotchas (below).
3. **M09 go-mod findings** — narrowed but not done: the old ×26 double-count shrank to ×24 "mixed require blocks" warnings (gomod-check + go-mod-ignore-check report the same 24), all warning-level, below the critical gate. The vendor ×25 source resolved naturally (`examples/middleware-showcase/vendor/` was trashed by a sibling); the ×2 missing-submodule-replace findings did not reappear under the fresh binary. Normalization deferred to a quiet window.
4. **M10 phantom-`./v4` root cause** — materially advanced, not closed: the doctor's `workspace/go-work-paths` check naively compares `go.work` `use` dirs to go.mod module names ("./adminui implies module adminui… likely missing /v4" ×13) — a false-positive rule class, now with captured evidence text. The related new classes (below) belong in the same upstream report.
5. **Root go.mod directive ENDGAME** — restored to 1.26.7 (5th) and root-caused as far as the repo can see, but the war continues as long as any 1.27.1-toolchain session runs go commands here. The durable fix (coordinated bump or GOTOOLCHAIN pinning) is the open TODO_LIST P2 decision.

## c) NOT STARTED

1. **M18 docs wave** (see b2) — CHANGELOG, TODO_LIST/ROADMAP harvest, cross-links, annotations.
2. **F96 full verification battery** — `nix run .#build` / `.#test` (workspace mode), `.#coverage-gate` (dashboardui grew new code paths), `.#check-release-train -- --refresh-cache`, `bash scripts/check-go-toolchain.sh`, full `.#lint`.
3. **Push** — deferred per the standing default; note an auto-push mechanism has been observed pushing narrative commits earlier this week, so "defer push" is advisory, not enforced.
4. **M15–M17, M19–M27 tail** (etagclient recipe, If-Match recipes, benches, WPT corpus notes, datastar retry-consistency patch, family-audit scheduling) — untouched this run.
5. **Release train** — the tree now carries four unreleased API surfaces: `ServiceConfig.CommandMiddleware` (usermgmt), `HubReadinessCheck` (root), the SSE-hub card (dashboardui), drain+retry-hint (setup/transport). Two setup dev-replaces + verify-tag refusals make the strip order: tag usermgmt + root, then setup, then dashboardui per run-2's open Q2.
6. **exhaustruct_v5 surge triage** — 10 findings (4 root + 6 dashboardui) + likely more in examples (admin-demo/catalog-demo/dashboard-demo hook failures) need the fix-or-suppress-with-reason batch treatment, plus a decision on migrating `//nolint:exhaustruct` directives to `//nolint:exhaustruct_v5` (or the config's `exhaustruct` alias, if golangci supports redirecting).
7. **Upstream tool reports** — four evidence-backed BuildFlow/go-structure-linter findings are ready to file (see f37–f40).

## d) TOTALLY FUCKED UP

1. **I published a wrong conclusion and had to walk it back same-session.** My first AGENTS.md edit declared the directive-bump "vector now DEAD" based on the go-appkit replace removal; minutes later the 5th bump landed with no replace present. The corrected analysis (module-cache scan → no dep requires 1.27 → bumper is a foreign 1.27.1 toolchain process) is now in AGENTS.md, but the initial claim would have told the next session to stop checking `head -5 go.mod`. Root cause: I wrote the diagnosis before testing the disproof. Cost: one extra edit + commit; risk: a sibling session acting on the wrong "DEAD".
2. **The daemon race claimed two narrative commits in a row — the exact documented loss mode.** I staged 4-5 files, ran the hook-gated commit, and the daemon absorbed the files mid-hook (the hook now takes 60-90s on failing diffs — a new race window the old fast hook didn't have). Content survived and was verified, but the M12/M13 stories are heuristic-commit archaeology again. I re-learned a rule I already knew: on this tree, stage-and-commit is a race the moment the hook exceeds ~30s.
3. **I burned a round trip on a broken verification pattern.** After the budget edit, I "verified" with `rg '"setup"=23'` — missing the `["` brackets — got MISSING, and briefly concluded the edit was lost. The file was fine; my pattern was wrong. Test the pattern against the known-good line before believing a negative.
4. **I dismissed the LSP twice before it was right once.** The stale-LSP rule says never trust it; I applied that as "always ignore it" — but the `typecheck: undefined: event` warning on readiness.go was REAL (I had removed the event import while still referencing `event.Infrastructure`). The correct discipline is asymmetric: never ACT on LSP output, but always let the CLI adjudicate — and the CLI agreed with the LSP that time.
5. **Guessed a library constant instead of checking.** `display.StatToneAmber` doesn't exist (valid: Blue/Green/Yellow/Red/Purple) — the build caught it in one cycle, but the module cache was one `rg` away and I'd already read from it earlier that hour.
6. **`sed -i` on go.mod for the 5th restore** — worked, but bypassed read-before-edit discipline for speed on the single most fought-over line in the repo. No damage done (one-line, verified), but that's the file where a sloppy edit costs the whole workspace.
7. **Commit8's 3 failed hook steps went unread for ~25 minutes** — the report request arrived while the failure details were still in `/tmp/commit8.log`. Root-caused during this report's prep: root golangci exits 1 on the 5 (mostly pre-existing) exhaustruct_v5 warnings + my 1 varnamelen; examples/dashboard-demo similarly. The structural mismatch — hook STEPS fail on any finding while the findings GATE passes at critical — is now precisely characterized, but it should have been characterized the moment the commit failed.

## e) WHAT WE SHOULD IMPROVE

1. **Conclusions need a disproof pass before they're written down.** "Vector now DEAD" cost credibility; the corrected entry is better because it states what was tested (module-cache scan) and what can't be controlled. Rule: a root-cause claim ships with its falsification test.
2. **Daemon-race defense needs to be mechanical, not remembered.** For every hook-gated commit on this tree: write the message first, stage exactly the intended files, commit, and if the hook fails on steps that are provably false (CLI-verified) or on pre-existing findings, immediately `git status` → re-stage → `--no-verify` with justification rather than leaving staged files for the daemon. The 30-second hook of last week made the old habit safe; the 90-second failing hook makes it a race.
3. **Plan items that say "adopt X at N sites" must be API-checked at planning time.** M14 was scheduled (45 min est.) with zero verification that `RequireDataJSON` accepts what our tests hold. A 2-minute signature read would have reclassified it as "rejected" before it ever hit a session. The plan template should grow a "precondition: API shape verified" column.
4. **Check the module cache before guessing any library constant.** Every StatTone/constructor/parameter question this session was answered in seconds by `rg` on `~/go/pkg/mod/...` — and the one guess (StatToneAmber) was wrong.
5. **LSP policy should be "adjudicate, don't obey or ignore".** Both blind trust and blanket dismissal were wrong this session; the CLI is the only referee. Keep the rule as: LSP output is a hypothesis generator; `go build` is the verdict.
6. **Read hook failures at the moment they happen.** The three failed-step lists sat in logs while I moved on. The failure-class characterization (step-exit vs gate-threshold mismatch) that took 25 minutes post-hoc would have taken 2 minutes inline.
7. **Gate design insight worth encoding in AGENTS.md:** buildflow's per-module golangci STEPS are exit-1-on-any-finding, while the findings GATE is threshold-based (`fail_on`). Any module with a single warning therefore fails the hook forever, regardless of `fail_on`. The choices are: fix/suppress findings per module, or exclude the module from the hook's golangci fan-out — the gate threshold alone can never make those steps green.
8. **`//nolint` directives must track linter renames.** exhaustruct → exhaustruct_v5 silently detached every existing directive. When a golangci upgrade lands, sweep for renamed linters' nolint directives in the same change.

## f) NEXT (up to 50, Pareto-ordered within tiers)

**Resolve session debris first:**
1. Triage commit8's 3 failed hook steps to closure: fix my `varnamelen` (rename `h`), decide the exhaustruct_v5 response (below), CLI-verify examples/dashboard-demo, re-commit any narrative remnant with justification.
2. exhaustruct_v5 batch: fix the 4 root + 6 dashboardui partial-struct literals (they are real `exhaustruct` findings the old nolint used to suppress) OR migrate the directives to `//nolint:exhaustruct_v5` — one consistent policy, applied repo-wide, so hook golangci steps go green again.
3. M18: CHANGELOG entries — directive saga + drift sweep + gate restoration + M07 + M12 + M13 + M14-rejection (CHANGELOG is append-only per repo convention).
4. M18: HARVEST into TODO_LIST/ROADMAP — the four upstream tool reports, the exhaustruct_v5 policy, the step-vs-gate mismatch, M14's rejection note (so no future session re-plans it), the gauge-recipe pointer (consumer-side, per sse-and-datastar.md §Observability).
5. M18: AGENTS.md — add M12/M13 to the adoption tables; three new gotchas: (a) `.go-structure-linter.yaml` extension trap, (b) buildflow result cache replays findings after config/binary changes (use `BUILDFLOW_NO_RESULT_CACHE=1`), (c) hook golangci steps fail on warnings regardless of `fail_on`.
6. M18: annotate the two prior status reports + this one with outcomes (the ANNOTATE-not-rewrite convention).

**Full verification battery (F96):**
7. `nix run .#build` + `.#test` workspace-mode (prove the 1.26.7 posture end-to-end).
8. `nix run .#coverage-gate` — dashboardui gained code (hub card path); confirm the 85.2%/60 gate still holds.
9. `nix run .#check-release-train -- --refresh-cache` + `bash scripts/check-go-toolchain.sh`.
10. `nix run .#lint` all 15 modules; fold the exhaustruct_v5 surge into one triage pass (fix vs nolint-migration per module).
11. `nix run .#check-codegen` + `.#check-templates` (untouched this run, but the battery exists to prove that).

**Release + trains:**
12. Cut the family train carrying this week's API: usermgmt (CommandMiddleware) + root (HubReadinessCheck, conditional-GET fix, retry hint) first, then setup (drain + hub health) — then strip setup's two TEMPORARY dev-replaces and re-verify hermetically (the documented strip recipe: tidy+build+**vet** per module).
13. dashboardui release decision (run-2 g-Q2, still open): semantics + markup changed materially (stopped=healthy, all tables) — tag via `scripts/verify-tag.sh` or hold for the templ-components v1.18.0-train alignment.
14. After tags: verify `check-release-train` drops to 0 UNPUBLISHED (train-lag list will name the next alignment targets).

**Tool-behverity upstream reports (evidence in hand):**
15. BuildFlow: result cache replays findings after config/binary changes (keys cover files only) — repro: rename `.yml`→`.yaml`, finding persists until `BUILDFLOW_NO_RESULT_CACHE=1`.
16. BuildFlow: `max_concurrency` file value loses to the CLI default (4) — env var is the only effective pin; file key is dead in practice.
17. BuildFlow doctor: `workspace/go-work-paths` flags 13 standard `/v4` modules as "likely missing /v4 suffix" — naive dir-vs-module-name comparison.
18. go-structure-linter: wrong-extension config (`.yml` vs `.yaml`) is silently ignored — should warn; this hid a suppression config for a full day across sessions.
19. BuildFlow step/gate semantics doc: per-tool steps exit-1-on-findings vs threshold gate — document the "exclude from hook fan-out" escape hatch.

**Directive war endgame (needs Lars):**
20. Execute the TODO_LIST P2 decision: coordinated flake+go.work+27-module bump to 1.27.1, OR `GOTOOLCHAIN=go1.26.7` pinning in fleet shells/hooks — then fix go-appkit's root go.mod at source (its own submodules say 1.26.7).

**Run-2 handoff debt (still open, from the sibling's report):**
21. Playwright visual pass (light/dark/mobile) — the pixels have never been looked at.
22. Vacuous-assertion sweep (`strings.Contains` scoped to pages not elements) across adminui/setup/integration_test.
23. Fix `TestCSP_UnsafeInlineNotRequired` permanent SKIP (configure `NonceConfig.CSPBuilder` so the assertion path executes).
24. Rebuild the dashboardui benchmark baseline from `git show 81088b64:...` (not from memory); drop the dead `_ = ctx`.
25. M25 Playwright e2e specs (badges/stat cards, toasts/error pages, sortable tables) + axe sweep.
26. `nix fmt` treefmt sweep at a clean-tree boundary.
27. M09 normalization: `go-mod-normalize` over the 24 mixed-require-block warnings in a quiet window (verify dep budgets unchanged after).
28. samber-linter 88% hook failure rate (61/69) — root-cause or exclude with reason.
29. nix eval-cache SQLITE_BUSY under concurrent sessions — fleet-level mitigation (serialized eval cache or per-session cache dir).
30. Verify-tag fixtures: exclude `scripts/testdata/verify-tag/*` from the doctor's gomod-freshness warning (7 of the 9 "need tidy" modules are deliberate).
31. README (dashboardui): adoption-surface table + golden `-update` flow + benchmark pointer (run-2 f#21).
32. Doc guide: the hybrid-adoption pattern (`Component.Render(ctx, &b)`) so the next consumer module repeats it safely (run-2 f#22).
33. AGENTS.md: "nolint lines stay ≤120 chars incl. reason" gotcha (run-2 f#24).
34. Dashboardui release CHANGELOG cut from `[Unreleased]` once #13 is decided.
35. Upstream asks: ListNote range variant; Grid children-less render guidance (run-2 f#27/#28).
36. Dashboardui TODO_LIST entries for the exclusions (theme-toggle decision, SidebarNav revisit criteria).

**Original pareto tail (M15–M27 leftovers):**
37. M15: etagclient recipe + runnable proof.
38. M16: If-Match conditional-write recipes (docs + example).
39. M17: benchmark additions for the conditional-GET paths (hashTag vs etag middleware).
40. M19: dashboardui/health SSE fan-out gauge example built ON the recipe (consumer-side, uses `Bundle.Broadcaster.Hub()`; this is the sanctioned M13 remainder).
41. M20: e2e assertion hardening (the vacuous class, e2e flavor).
42. M21: upstream release tracking note for go-cqrs-lite projectionadapter/sqliteengine tags (the remaining systemadapter replace blockers).
43. M22: sse-and-datastar.md — add `HubReadinessCheck` to the guide's health wiring example.
44. M23: coverage: dashboardui hub-card path in the module's coverage budget math (new files shifted percentages).
45. M24: `event_catalog_handler`/`htmx_serve` conditional-GET spec harness — extract the reusable harness into a documented pattern (currently root-test-local).
46. M25: `conditional_get_spec_test.go` — consider promoting the harness to `ssetest` upstream (it is domain-generic).
47. M26: retry-hint constant: confirm `DefaultRetryHintMillis` is documented in the SSE guide's client-config section.
48. M27: roadmap fuel — WPT corpus eval notes; polled-panel 304 research (from the original 50-item list).
49. Family-audit scheduling: templ-components v1.19/go-sse v0.7 watch items into TODO_LIST.
50. Post-train hygiene: after #12's tags, re-run `check-release-train`, strip replaces, `GOWORK=off` tidy+build+vet per affected module, and update AGENTS.md version claims (the post-bump VERIFY ritual).

## g) QUESTIONS (cannot answer myself)

1. **The 1.27.1 war needs your verdict, not a sixth restore.** Five bumps, five restores; the bumper is a foreign 1.27.1-toolchain process I cannot control, and the flake/go.work both still say 1.26.7. Do you want the coordinated flake+go.work+27-module bump executed NOW (I will do it module-by-module with hermetic verification), or should I pin `GOTOOLCHAIN=go1.26.7` into the shared hook/env so foreign toolchains stop mutating the floor? My recommendation: the coordinated bump — the escape hatches (hermetic 1.27.1 builds) already prove the tree compiles either way, and the war costs every session time.
2. **Hook golangci steps vs the critical gate.** The hook's per-module golangci steps exit-1 on ANY finding, so the 10 exhaustruct_v5 warnings (4 root + 6 dashboardui, all warning-level, all now exposed because the v5 rename detached the old nolint directives) block narrative commits even though `fail_on: critical` passes them. My plan: migrate the directives + fix my one varnamelen, making the steps green. Alternative: exclude examples/* (and root?) from the hook's golangci fan-out, keeping them CI-only. Which policy do you want — I default to "fix + migrate directives" (keeps the hook honest) unless you prefer the CI-split.
3. **Release train now or after the dashboardui decision?** Four unreleased API surfaces are stacked (usermgmt CommandMiddleware, root HubReadinessCheck + conditional-GET + retry hint, setup drain + hub health, dashboardui hub card). Cutting usermgmt+root first would let setup strip its two dev-replaces and make every module hermetically self-consistent again. But run-2's open question (dashboardui semantics changed: stopped=healthy) gates the dashboardui tag specifically. Cut the non-dashboardui train now and leave dashboardui for its own decision, or hold everything for one coordinated cut?

---

**Awaiting instructions.**
