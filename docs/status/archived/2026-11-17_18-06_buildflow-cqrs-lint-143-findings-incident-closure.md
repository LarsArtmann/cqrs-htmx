# BuildFlow cqrs-lint 143-Findings Incident — Closure Status Report

> **ANNOTATED 2026-11-21** — ~~every open item in this report is resolved~~ **ARCHIVED 2026-11-21 (round-23 (2026-11-21)).**
> The next session in the same arc is [`2026-11-17_22-25_buildflow-cqrslint-train-v4-22-0-to-v4-22-2-and-gate-escalation.md`](2026-11-17_22-25_buildflow-cqrslint-train-v4-22-0-to-v4-22-2-and-gate-escalation.md), which incident closed per AGENTS; the train + gate escalation report is the successor.
> Nothing here is a live work item: it is kept as the session's evidence.

**Session:** 2026-11-17 ~15:53–18:06 CET · single-session scope: the `buildflow -s cqrs-lint --format finding` run that showed 143 findings behind a green canonical gate, plus the timing-regression and preflight-warning tails of the same run.

## Scope line

**Verified here (fresh commands/artifacts this session):**
- Canonical gate green: `nix run .#check-cqrs-lint` → "All modules pass cqrs-lint strict." 14/14, run twice (before AND after the config/suppression changes — the second run also proves zero stale suppressions).
- Finding decomposition: full 143-finding JSON extracted from `.crush/shell-output/output-4010040235.log`, classified by module × rule × severity (jq, counts in-text below).
- Fresh CLI walks: workspace root walk = 115 findings (`/tmp/root-walk.json`); hermetic per-module walks over 14 gate modules = 200 and 17 non-gate modules = 90 (rule×module tables captured in-session).
- go-cqrs-lite changes: `GOEXPERIMENT=jsonv2 go test ./cmd/cqrs-lint/v4/... -count=1` — full tree green on re-run (first run's FAIL was a parallel-load flake, re-run clean); 3 new regression pins pass by name.
- End-to-end BuildFlow verification with the shadow binary `/home/lars/go/bin/buildflow` (built via temporary go.mod replace to the local go-cqrs-lite, replace since DROPPED and go.mod/go.sum restored): `BUILDFLOW_NO_RESULT_CACHE=1 buildflow -s cqrs-lint --format finding` went **143 → 222 (intermediate, scope bug) → 327 (harness, scope+inheritance bug) → 180 (correct per-module truth) → 1** (V003, exit 0), and a final standard-cache run also returned total=1 with a correct cache miss→re-run.
- Doctor warnings cleared: `buildflow doctor` grep shows ✓ `workspace/go-work-paths`, ✓ `workspace/gomod-freshness`, ✓ `quality/lychee-archived-docs`.
- cqrs-htmx gates after the changes: `bash scripts/selftests/test-check-cqrs-lint.sh` green (twice, incl. after the gate-script edit), shellcheck on the edited script OK, `nix run .#check-root-files` ✓ 34 files matches baseline, `nix run .#check-todo-hygiene` ✓ clean (preamble 596), `go build ./...` OK, `gofmt -l` clean on all five touched Go files.
- ActorID form verification: `id/v4/actor_id.go:152` read directly — `String()` returns bare raw; `PrefixedString()` is the prefixed form. The C044 usermgmt/gfl sites are key/DTO-consistent; `sql_readmodel_extra.go:95`'s mixed-looking literal is actually uniform (both bare).

**NOT verified here (named gaps):**
- The V003 finding's substance: I never opened `systemadapter/go.mod:27` to name which go-cqrs-lite module lags behind which release — the "tracked family-train lag" claim comes from AGENTS.md context, not a fresh read.
- Byte-level parity toolsdk ≡ CLI-per-module: the 180-set was verified *coherent* (module distribution, rule classes, no duplication), not diff-equal to a summed CLI walk. The final residual of 1 makes this practically moot; as an engineering claim it is unproven.
- The shadow binary's freshness vs BuildFlow HEAD: the sibling session's commits (`7c417e72f`, `5921fcd61` — go_auto_upgrade_blame refactor) landed AFTER my last rebuild, so `/home/lars/go/bin/buildflow` likely predates them and doctor's binary-freshness advisory is probably firing again. Not re-checked.
- `nix run .#check-modules` full composite NOT re-run after the gate-script/config changes (time + concurrent-session load; the touched gates were run individually).
- Fleet-wide blast radius: no other LarsArtmann repo was surveyed for the same phantom class (any go.work repo + BuildFlow + cqrs-lint is exposed).
- `buildflow format` NOT run to re-verify formatter-fingerprint-drift verdicts (deliberate: gotcha 53 — a concurrent session is active in this tree; no tree-wide operations).
- golines survival of the 5 inline `//cqrs-lint:ignore` directives: safe by construction (full-line comments, gotcha-7 class is about trailing directives), not machine-verified.
- Timing-regression "benign" verdict rests on `buildflow timings --regressions` output (`VERSION 7e4ca72 !`, BASE N=170 cache-hits vs NEW N=5 real runs, clean `git status`) — the causal story is solid but no post-upgrade warm re-baseline was captured.

---

## a) FULLY DONE

1. **Canonical gate ground truth established** — `nix run .#check-cqrs-lint` green 14/14 strict, run twice (pre/post changes). This exposed the core trap: the gate is exit-code-only and pipes findings to `/dev/null`, so ~200 warning/info findings were invisible for weeks.
2. **143 findings fully classified** (module × rule × severity): C044×54, E004×22, E023×11, C009×10, F-series hygiene ×22, singles ×24; 14 landed on go.mod anchors (V-rules), spread over 63 files.
3. **Root cause 1 fixed at source (go-cqrs-lite toolspec)**: per-module hermetic walks (`GOWORK=off` injected via `hermeticEnv()`), per-module config resolution, merged findings, single suppression pass. Files: `cmd/cqrs-lint/v4/pkg/toolspec/toolspec.go` (detect/detectModule/repair/repairModule), `pkg/analyzer/loader.go` (`ModuleDirs`, `BuildContextForModule`, `buildScope`, env-threaded `loadFromDir`), `pkg/analyzer/project_config.go` (`LoadParentRulesConfig`).
4. **Root cause 1b fixed**: repo-root iteration swallowed the whole monorepo (union walk duplicated every nested module's findings under root config — usermgmt 21+44, dashboardui 37+35). Fixed by `scopeSingleModule`; regression-pinned.
5. **Config inheritance implemented** (`analyzer.LoadParentRulesConfig`): ancestors union `rules.disable` + `external-api-struct-prefixes`; includes the immediate parent — deliberately diverging from the CLI's ancestor walk, whose skip-immediate-parent behavior carries a cmdguard-copied comment and made depth-1 modules ignore root disables that depth-2 modules honored.
6. **Three regression pins added** (`toolspec_test.go`): per-module config scope (module b's disable silences only b), poisoned-go.work immunity (root go.work with hostile replace cannot distort hermetic loads), root-module scoping + parent inheritance (2 modules, 2 C003 sites → exactly 2 findings; root disable silences both). Shared fixture refactored into `writeConsumerModule`.
7. **End-to-end verification**: 143 → 1 through four measured intermediate states (222 = scoping bug, 327 = harness pre-inheritance, 180 = correct per-module truth pre-triage, 1 = post-triage). The final standard-cache run reproduces total=1.
8. **Consumer triage — 14 disable-with-reason `.cqrs-lint.json` configs** (dated 2026-11-17): root (+A009, F032), identity-model (+E005 ×17 structural domain-catalog, C013 ×2 v5-class), usermgmt (+E023 documented pre-metaengine class, C044 ActorID bare-raw, A016/F007/B024 library-principle middleware, F005, A027 process-global codec), dashboardui (+C044 templ-generated/display, A018), adminui (+A018), huma (+A025, F007), setup (NEW: E023/F003/F004/F028/B014/B029/B030/F023/F025/F032), 10 demo dirs, e2e/server, gfl-usermgmt.
9. **Five surgical inline suppressions** (rules stay live elsewhere): `options_validate.go:26` C009, `recommended_middleware.go:52` C009, `usermgmt/es_event_catalog_artifact.go:33` C009, `systemadapter/tenant_deletion_deriver.go` C027+C009 (ADR-0051 cascade bridge — disjoint consumers, documented), `transport/event_sse.go:40` C044 (golden-pinned envelope).
10. **Gate blindspot closed**: `scripts/checks/check-cqrs-lint.sh` now prints the failing module's findings (tail -60) instead of swallowing them; self-test + shellcheck green.
11. **Two BuildFlow doctor false positives fixed upstream**: `go-work-paths` suffix comparison (warned on every /v4 module with a backwards "missing /vN" hint; both layouts accepted; 4 test cases incl. genuine-mismatch retention), `gomod-freshness` testdata exclusion (gotcha-34 fixtures; internal collector test added). checks package green.
12. **lychee-archived-docs warning fixed**: new `lychee.toml` (excludes `docs/modularization/archived`, `docs/research/archived`) + `root-files.baseline` bump (clean one-line diff after fixing my own locale-sort mangle).
13. **BuildFlow hermeticity restored**: the temporary go-cqrs-lite replace dropped (`go mod edit -dropreplace` + tidy); go.mod/go.sum diffs are clean reversions. 37 unrelated replaces untouched; the concurrent session's files untouched.
14. **Timing-regression triage closed**: `buildflow timings --regressions` shows govalid upgraded 3 days ago (`VERSION 7e4ca72 !`; BASE N=170 cache-hit samples vs NEW N=5 real runs), tree clean, steps succeed → benign toolchain-drift artifact, no action needed.
15. **Docs closure**: AGENTS.md cqrs-lint row carries the incident-closure paragraph; TODO_LIST.md carries the P1 release-train tail row (hygiene gate green); CHANGELOG intentionally not touched (train convention).
16. **cqrs-lint full test tree green** after all changes (`go test ./cmd/cqrs-lint/v4/...`), and the canonical gate re-passed under the modified analyzer (CLI default path byte-identical: `scopeDiscover`).

## b) PARTIALLY DONE

1. **The release train for the toolspec fix** — code done and tested in `/home/lars/projects/go-cqrs-lite` (daemon-committed), but NOT tagged: no `cmd/cqrs-lint/v4.22.0`, BuildFlow still pins v4.21.0, so the nix-profile BuildFlow binary still carries the old workspace walk. Until the train, the shadow binary is the only honest instrument. **DONE 2026-11-17 evening:** `cmd/cqrs-lint/v4.22.0` tagged (dual shapes `cmd/cqrs-lint/v4/v4.22.0` + compat `cmd/cqrs-lint/v4.22.0`, both peeled to the api-golden commit; proxy smoke + clean-dir install probe green; CHANGELOG section cut; GH release with generated notes), BuildFlow pin bumped v4.21.0 → v4.22.0 + vendor hash + `nix build` + `.#reinstall` green.
2. **Shadow binary lifecycle** — `/home/lars/go/bin/buildflow` works and is PATH-first, but likely already stale vs the sibling session's BuildFlow commits (blame refactor landed after my last rebuild); deletion is scheduled only after the train + reinstall. **DONE 2026-11-17 evening:** shadow binary removed from `~/go/bin` after the reinstall; nix-profile binary is PATH-first again; `buildflow doctor` binary-freshness green after one rebuild-from-HEAD cycle (the vendorHash commit landed between build and check — the check is that honest).
3. **toolsdk/CLI parity** — the toolsdk now has inheritance; the CLI does not (its root walk still applies the root config globally; its ancestor walk still skips the immediate parent). Converging them was deliberately out of scope; the divergence is now documented rather than eliminated. **DECIDED 2026-11-17 evening: CLI semantics FROZEN — the toolsdk is the modern surface.** Rationale: the cqrs-htmx gate and BuildFlow (the two consumers that matter) both run per-module walks; converging the CLI root walk is an upstream go-cqrs-lite design change with wide blast radius and no fleet consumer asking for it; the E014 anchor-drift class stays resolved-by-disable per the v4.21 verdict. Convergence remains upstream-owned if a consumer ever needs it. **AMENDED same evening:** the escalation forced two upstream patch releases — v4.22.1 fixed the CLI ancestor walk (the immediate parent is now included; this item's second half — "its ancestor walk still skips the immediate parent" — is dead), and v4.22.2 shipped `--scope=module` as the OPT-IN per-module CLI semantics; the DEFAULT root walk stays frozen exactly as decided, the gate opts in, and gate/toolsdk now run one resolution semantics.
4. **V003 (the 1 residual finding)** — left visible on purpose (true signal), but unverified in substance (which module/version lags) and unactioned (pin bump rides the family train). **RESOLVED 2026-11-17 evening, and it was a FALSE POSITIVE:** the substance check found systemadapter pins `systemcatalog/v4 v4.0.0` and the proxy confirms v4.0.0 IS systemcatalog's own latest (exactly one tag exists); the "3 behind v4.3.x" verdict came from cqrs-lint's fleet-constant fallback (`latestKnownMinor = 3`), active because GOPRIVATE covers all larsartmann paths and bypassed the issue-#129 per-module fetch in every fleet run. Fixed at source in v4.22.0 (V003 emits only on verified proxy truth; mutation-pinned); post-train BuildFlow run = 0 findings. Genuine fleet lag stays covered by `check-upstream-treadmill`.
5. **Timing re-baseline** — diagnosed benign but no warm re-run captured; `timings --regressions` will carry the artifact until real-run samples accumulate.
6. **Doctor advisory hygiene** — three warnings cleared, but `system/io-history-ratchet` (73 files hashed vs median 1) was never investigated, and `formatter-fingerprint-drift` verdict re-verification (via `buildflow format`) was deferred with rationale, not done.
7. **CHANGELOG** — no `[Unreleased]` entry in cqrs-htmx for the configs/suppressions/gate-script change (the repo's train convention defers it; noting the gap honestly).

## c) NOT STARTED

1. Fleet survey: which other LarsArtmann repos carry the same BuildFlow phantom class (go.work + cqrs-lint) — zero repos checked.
2. `docs/gotchas.md` entry 61 (exit-code gate hiding findings behind `>/dev/null`) — not written; the incident lives in AGENTS.md + TODO_LIST only.
3. `.buildflow.yml` `env: GOEXPERIMENT: jsonv2` block (the documented HOW_TO_USE.md:392 pattern) — this repo relies on the ambient shell exporting it; a shell/CI without it would silently degrade cqrs-lint's package loading and nobody would know.
4. BuildFlow `FileContract.ConfigFiles` declaration for `.cqrs-lint.json`/`go.work` in the cqrs-lint Spec — today a config edit does NOT invalidate the result cache (the manual `BUILDFLOW_NO_RESULT_CACHE=1` dance remains necessary).
5. docs-gates battery re-run after the status report lands (freshness/annotate/routing).
6. cqrs-lint C044 ActorID boundary-awareness (the rule table itself promises "boundary-specific guidance" for ActorID — not yet implemented, so ActorID sites still fire per-module).
7. cqrs-lint E005 cross-module registration knowledge (workspace-registry option for domain-types modules).
8. ADR-0059 extension: the four JSON-codec `[]byte` payloads observed in the 115-era walk (CredentialRemoved/CredentialUsed/BotRegistered/TOTPEnabled) — P009 no longer fires per-module (codec-choice evidence lives cross-module), but the CBOR gap is real backlog.
9. cqrs-htmx CHANGELOG receipt for this triage wave.
10. `buildflow docs --check` / BuildFlow docs update for the cqrs-lint provider's new per-module semantics.

## d) TOTALLY FUCKED UP

Nothing in the final state — but three mid-session failures, self-caught and repaired, are owned here:

1. **I trusted AGENTS.md's "zero warnings" claim instead of verifying it first.** I built the "workspace walk phantom" theory on a gate claim that was — measured today — false as stated (the gate is exit-code-green WITH 200 warnings; "zero warnings" described a different era/binary). A single findings-mode CLI walk in minute one would have killed the wrong theory; instead it survived until the 20-vs-20 GOEXPERIMENT comparison falsified my first hypothesis and forced the recalibration. Cost: several analysis cycles and a wrong-fix risk that the empirical verification loop caught.
2. **I mangled `root-files.baseline` with a locale-dependent `sort`** — the file uses C-locale grouping (dotfiles, uppercase, lowercase) and ambient-locale sort interleaves them. Caught by diffing, restored my own edit, re-inserted surgically. Lesson applied: never re-sort a pinned file to add one line.
3. **I ran the 14-module CLI union walk with shell `&` parallelism and collected corrupted counts (85 vs the true 200)**, which briefly made the toolsdk look like a superset for the wrong reason. Caught by the count mismatch; redone sequentially. Also the first JSON extraction from the log truncated at the `}regression` mid-line (two parse errors before the boundary grep).
4. **Named honestly, not fucked up but unproven**: the V003 substance claim, the toolsdk≡CLI byte-parity claim, and the shadow-binary freshness are asserted from context, not fresh reads — all three are named in the scope line so no reader can mistake them for evidence.

## e) WHAT WE SHOULD IMPROVE

1. **Gates must not swallow their own evidence.** Any `>/dev/null 2>&1` gate whose verdict is richer than exit code is a lying instrument (this incident's core lesson — 200 findings invisible for weeks). Audit the other gate scripts in `scripts/checks/` for the same pattern.
2. **Verify the gate's claim, not the docs' claim.** AGENTS.md said "zero warnings"; the binary said 200. When a doc claim and a fresh measurement can be reconciled in one command, run the command first.
3. **Empirical falsification before theory-building.** The session converged only when every theory (phantom walk → GOEXPERIMENT → config scope → scoping) was pitted against a measured set. The right order is: measure first, theorize second — I did it backwards for the first hour.
4. **Cache-invalidation contracts should be declarative.** `BUILDFLOW_NO_RESULT_CACHE=1` is a tribal-knowledge workaround; config-file-aware cache keys (FileContract.ConfigFiles) are the fix.
5. **Walk modes should be ONE mode.** CLI root-union, CLI per-module, toolsdk per-module+inheritance are three resolution semantics; rules that depend on cross-module visibility (E005, E004, F032) behave differently in each. Converging on per-module+inheritance everywhere removes the "both cannot hold" class at its root (the E014 anchor-drift saga is the same disease).
6. **Per-module findings need a periodic visibility pass.** The exit-code gate was green for weeks while findings accumulated. A cheap advisory (`check-cqrs-lint --report` printing counts per module) or a gate warning-budget would prevent silent accretion.
7. **Concurrent-session discipline worked but cost care** — gotcha 47 is live in both trees right now (cqrs-htmx docs/corpus edits, BuildFlow blame refactor). Tree-wide operations (buildflow format, corpus re-pins) must keep waiting for a quiet window.
8. **Multi-repo fixes need the release-train loop closed fast** — the source fix is inert until tagged and pinned; the shadow binary is a crutch with its own staleness risk.

## f) Top 50 things to get done next

Ordered by dependency/impact; 1–8 are the incident's own tail, 9–20 are this session's discovered-but-unstarted items, 21–50 are the follow-on surface the session exposed.

1. Verify V003's substance: read `systemadapter/go.mod:27`, name the lagging module and version (10 min; converts the scope-line gap into a fact).
2. go-cqrs-lite: commit-sanity + release `cmd/cqrs-lint/v4.22.0` with the per-module toolspec fix (per its release playbook; train-preflight).
3. Bump BuildFlow's `go-cqrs-lite/cmd/cqrs-lint/v4` pin past the fix; `nix build . && nix run .#reinstall`.
4. Delete `/home/lars/go/bin/buildflow` after the reinstall; re-run `buildflow doctor` expecting binary-freshness ✓.
5. Verify the installed (nix-profile) binary reproduces total=1 on `BUILDFLOW_NO_RESULT_CACHE=1 buildflow -s cqrs-lint --format finding`.
6. Sweep the V003 pin (systemadapter's lagged go-cqrs-lite family member) via `wave-tag`/`bump-dep` on the same train; expect the step to reach 0 findings.
7. Run `nix run .#check-modules` end-to-end over the landed changes (gate script + 14 configs + inline suppressions).
8. Write the gotcha-61 entry ("an exit-code gate that pipes findings to /dev/null hides accretion — print evidence on failure") + index row.
9. Add `.buildflow.yml` `env: GOEXPERIMENT: jsonv2` (HOW_TO_USE.md:392 pattern) so cqrs-lint's analysis doesn't depend on the ambient shell.
10. BuildFlow: declare `.cqrs-lint.json` + `go.work` as `FileContract.ConfigFiles` in the cqrs-lint Spec (kills the `BUILDFLOW_NO_RESULT_CACHE` tribal dance).
11. Upstream cqrs-lint: implement C044 ActorID boundary-awareness (the rule table already promises it) so ActorID sites stop firing fleet-wide.
12. Upstream cqrs-lint: converge the CLI ancestor walk with `LoadParentRulesConfig` (include the immediate parent) + tests — retires the depth-1/depth-2 inconsistency.
13. Upstream cqrs-lint: decide the E005 cross-module story (workspace-registry option vs documented domain-module disable pattern) and encode the verdict in `docs/` + rule docs.
14. Reconcile the E014 disable: consider re-scoping from the root config's global disable to per-module disable-with-reason now that inheritance exists (the root-level disable's rationale comment still describes the two-walk-mode world).
15. Run the fleet phantom-class survey: for every go.work repo under `~/projects` with BuildFlow + cqrs-lint, run `BUILDFLOW_NO_RESULT_CACHE=1 buildflow -s cqrs-lint --format finding` with the fixed binary and triage residuals.
16. Fleet survey via `buildflow doctor --only go-work-paths --root ~/projects` and `--only gomod-freshness` — quantify how many repos carried the /v4 false positive and the testdata false positive.
17. Port the E005 disable pattern to other fleet domain-types modules the survey finds (browser-history et al.), each with its own dated reason.
18. cqrs-htmx CHANGELOG `[Unreleased]`: receipt row for the 14 configs + 5 suppressions + gate-script evidence fix.
19. Re-run `docs-gates` five-gate battery after this report lands (freshness, annotate, routing, cadence, audit-all owner-gated).
20. Investigate `system/io-history-ratchet` (73 files vs median 1): identify which memo layer stopped eliminating I/O, or record why it's noise.
21. Quiet-window `buildflow format` to re-verify formatter verdicts under the new fingerprints (oxfmt 0.75, templ-fmt 0.3.1100, buf 1.75, cargo 1.99) — blocked on gotcha 53 until the sibling session ends.
22. Capture a warm `buildflow timings` baseline so the govalid-generate regression artifact ages out of the comparison window.
23. Investigate govalid-generate's new 3–8s per-module cost (upgrade cost vs config drift vs GOEXPERIMENT recompiles) — decide whether a prewarm or exclusion is warranted.
24. Gate-script regression test: extend `test-check-cqrs-lint.sh` to assert the failure path PRINTS findings (currently asserts verdict only).
25. golines/format run over the five inline-suppression files to machine-verify the directives survive (gotcha-7 class).
26. Verify cqrs-lint's `--fail-on-stale-suppressions` interacts correctly with the NEW per-module config disables once rules get fixed upstream (a disable whose rule stops firing = gate error by design — schedule the prune pass).
27. BuildFlow docs: document the cqrs-lint provider's per-module semantics + the GOEXPERIMENT requirement in `docs/configuration.md` / provider docs.
28. BuildFlow: cut a release/tag carrying the two doctor fixes so fleet repos get them via the normal upgrade path.
29. Write the BuildFlow-side CHANGELOG/commit receipts per its conventions for workspace.go + gomod_freshness_1.go (+ the parity-audit test implications, if any).
30. Check whether `audit/lychee_exclude_parity_audit_test.go` (doctor/provider lockstep) needs a row for repo-level lychee.toml excludes like cqrs-htmx's new one.
31. Upstream cqrs-lint: add a CLI findings-mode summary to the gate story — e.g. `--min-severity warning --health-score` in the gate script's advisory tail, so accretion is visible without changing exit semantics.
32. Decide + document the canonical cqrs-lint walk mode for THIS repo's docs (AGENTS.md says "run only via the flake app" — now also true via fixed BuildFlow; update the row once the train lands).
33. Retire AGENTS.md's stale "zero warnings" phrasing in the cqrs-lint row (trued by this session's report; the row now carries the incident paragraph — sweep the older sentence for consistency).
34. ADR-0059 extension train: CBOR for CredentialRemoved/CredentialUsed/BotRegistered/TOTPEnabled (observed in the 115-era set; per-module walk no longer surfaces it — backlog row needed so it isn't lost).
35. E023 pressure: keep the ADR-0051/metaengine migration plan moving (usermgmt ×10 + setup ×1 sites are the standing class).
36. C013: schedule the `event.Instant` migration for SessionStarted/Ended payloads into the v4→v5 cutover doc.
37. `check-e2e-ran-count`/e2e battery: confirm no interaction with the new configs (none expected — configs are lint-time only; run once for the receipt).
38. Confirm steady-state result-cache behavior: a second `buildflow -s cqrs-lint` (no bypass) should HIT and still show total=1.
39. Evaluate `tool_options` support in the cqrs-lint Spec (e.g. `exclude_modules`) as the alternative to per-demo-module configs — design decision, owner-gated.
40. Sweep other fleet repos' `.buildflow.yml` for the missing `env: GOEXPERIMENT` on jsonv2 repos (same ambient-shell dependency cqrs-htmx has).
41. Read `docs/decisions/2026-10-27_open-decisions.md` §D lineage only if the report-format ritual changes again — skip; noted to prevent drift.
42. Consider an `erraudit`-style baseline/ratchet for cqrs-lint findings per module (the `check-cqrs-lint` gate could pin a findings-count baseline like the tail/corpus ratchets do) — prevents both silent accretion and noisy triage.
43. Skill upkeep: the `cqrs-htmx` skill and `how-to-golang`/`linter-building` skills don't mention the per-module cqrs-lint semantics — a one-line note prevents the next session from re-deriving the walk-mode story.
44. Verify the shadow binary vs sibling BuildFlow commits (rebuild from BuildFlow HEAD if the blame refactor is wanted in the verification binary) — cheap hygiene.
45. go-cqrs-lite: run its full gate battery (erraudit strict, branching-flow, coverage gates) over the analyzer/toolspec changes before tagging (item 2's precondition).
46. go-cqrs-lite CHANGELOG entry for the toolspec per-module fix (its conventions).
47. Add the incident's harness artifacts to the session receipt: the four measured states (222/327/180/1) are in this report; consider archiving `/tmp/bf-*.json` evidence into `docs/status/` attachments only if the owner wants the raw sets preserved (default: not needed).
48. Retire the `/tmp` harness (`/tmp/toolspec-harness`) — it documents the verification method; either promote it to a checked-in `toolspec` example/test or note it as disposable (default: disposable).
49. Review the other session's corpus edits (docs/CORPUS_BASELINE 61031→61035 + manifest rows) when it finishes — gotcha-47 courtesy diff-check of every daemon commit in the window.
50. Schedule the next archive round: the 11-14 pair of status reports is settled per the manifest and is the standing candidate; this report becomes the fifth live tail row (TAIL_BASELINE 4 → 5 same-change).

## g) Three questions I cannot figure out myself

1. **Release shape for the go-cqrs-lite toolspec fix:** tag a dedicated `cmd/cqrs-lint/v4.22.0` now (fastest path to deleting the `~/go/bin` shadow binary and un-poisoning every fleet repo's BuildFlow run), or fold it into the next family train you're already planning? I can execute either; the sequencing call (and whether the V003 systemadapter pin bump rides it) is yours. **DECIDED + EXECUTED 2026-11-17 evening:** dedicated tag, cut the same day (per-module truth wave) — a family train was disproportionate for a toolsdk-only fix and the incident's fix was inert until tagged. The V003 pin bump did NOT ride it: the finding proved to be a false positive (see §b4), so no systemadapter edit was needed.

2. **Gate policy escalation:** now that the residual is exactly 1 info finding, do you want `check-cqrs-lint` to escalate from exit-code-only to also FAILING on warning findings (a findings-honest ratchet, matching the AGENTS.md "zero warnings" claim it could never actually see), or keep warnings advisory forever? Escalating makes the incident class structurally impossible again; it also means every future info finding needs an immediate config decision. **DECIDED + SHIPPED 2026-11-17 evening: ESCALATED.** The gate fails on ANY finding (detected from the `<SEVERITY> file:line:col` output shape, not the binary's exit-code severity policy) and refuses cqrs-lint < v4.22.0 by version before running any module — the posture is only sound on the binary whose honest steady state is zero. Self-test extended to 9 cases with offline stub binaries (info-finding fails + evidence printed; benign banner passes; stale binary refused); shellcheck green; 14/14 modules green under the new posture. **Follow-through, same evening (two patch releases the escalation forced into the open):** v4.22.1 — the CLI ancestor walk skipped the IMMEDIATE PARENT on a false cmdguard assumption, so six depth-1 modules resurfaced root-config-silenced findings; fixed upstream (walker starts at the immediate parent; disable unions mean the fix can only silence, never create). v4.22.2 — `--scope=module` shipped (the opt-in per-module CLI semantics the toolsdk already had); the gate passes it per candidate, default walk stays frozen per the §g3 decision, final gate verdict TRUE-RC=0 across all 14 candidates with zero findings; gate floor moved to v4.22.2.

3. **Walk-mode convergence:** should the cqrs-lint CLI's root walk (`cqrs-lint .` at a monorepo root) also become per-module+inheritance — converging ALL walk modes on one resolution semantics and finally retiring the "inline waiver stale in one walk, load-bearing in the other" class (the E014 anchor-drift saga) — or do you want CLI semantics frozen for compatibility and the toolsdk as the only modern surface? **DECIDED 2026-11-17 evening: CLI FROZEN, toolsdk is the modern surface** (rationale at §b3). If a fleet consumer ever reports CLI root-walk findings, convergence rides an upstream go-cqrs-lite release with its own regression pins — not this repo's gate.

---

*Receipts for every verified claim are in the scope line; the /tmp evidence sets (`/tmp/bf-final2.json`, `/tmp/bf-final-set.txt`, `/tmp/cliwalks/`) are ephemeral. Session work spans three repos: cqrs-htmx (consumer triage + docs), go-cqrs-lite (toolspec/analyzer source fix), BuildFlow (two doctor fixes + hermeticity restore).*
