# Status Report: P2 Backlog Execution + v4.8.x Family Alignment Train

**Date:** 2026-08-30 08:47 CEST
**Session scope:** (1) Execute the non-gated P2 backlog from `docs/status/2026-08-29_23-05_pareto-master-plan-execution-complete.md` §f; (2) execute the user-approved gated family train per `docs/runbooks/release-next-train-prep.md`.
**Result:** 12 commits pushed to `origin/master` (`845b48ba..bacbea2b`), 7 tags pushed, tree clean, all runnable gates green on HEAD. The 275-file format-sweep question DISSOLVED (root-caused to a config delta). The strict version-drift floor for THIS repo is reached; remaining axes are upstream-blocked.
**Machine state during run:** `/mnt/buildcache` still dead (documented); `/tmp` healthy (~30G free, guarded by the new preflight). A concurrent session was building heavily (load 8.7) during the first bench runs — two bench gate runs were garbage before a clean green run.

---

## a) FULLY DONE

### P2 backlog (all non-gated items from the prior report's §f)

- **Low-disk fail-fast in the shared Go cache guard** (`043a409d`): `scripts/lib/go-cache-env.sh` checks free space on the resolved `GOCACHE` filesystem and fails up front with a what/why/fix message; `GO_CACHE_MIN_FREE_MB` (default 2048, 0 disables) tunes it. Verified: 4 functional cases (default pass, tiny-threshold fail, disabled, garbage-value fallback) + shellcheck clean. AGENTS.md gained a "Go caches" quick-ref row (prior §f.25).
- **Per-benchmark thresholds in bench-spike** (`80109cb3`): `BENCH_THRESHOLD_<NAME>` (bench name uppercased, non-alnum → `_`, suffix match works, e.g. `BENCH_THRESHOLD_JSON_ROUNDTRIP=15`); non-numeric values exit 2 naming the variable. Verified end-to-end: default green run, suffix override isolating json-roundtrip (`gate -100%` on it, global 10% on the others), exact-form override, invalid-value exit. Documented in `docs/benchmarks/README.md` + AGENTS.md Bench row (prior §f.20).
- **`check-phantom-version` → `check-require-tags` rename** (`28f1d3cf`): inline flake logic extracted to `scripts/check-require-tags.sh`; canonical app renamed, old name kept as a deprecated alias that warns and runs the same script. Verified behavior-identical (zero-pseudo leg green; chained strict drift red as documented). AGENTS.md + TODO_LIST mentions updated (prior §f.18).
- **CI wiring** (`65adf543`): `module-architecture` job runs the Go-toolchain gate (blocking) and check-release-train (advisory `|| true` with the auth caveat comment); `checks` job carries the bench-spike local-only annotation (prior §f.21 + §f.34). actionlint clean.
- **`go.work.sum` actually untracked** (`e603bb0d`): it was in `.gitignore` all along but still indexed, so the daemon kept absorbing drift (most recently into `0e2b0aa3`). `git rm --cached` + workspace build verified green with the file regenerating locally (prior §f.22).
- **treefmt golines ROOT-CAUSED and enabled** (`0d66097b`): the 2026-08-29 275-file churn was treefmt's golines default max-len 100 vs the golangci gate's 120 (`.golangci.yml: settings.golines.max-len`). Aligned via `settings.formatter.golines.options = ["--max-len" "120"]`; first `nix fmt` changed only 4 Go files (examples/ + oauth2 — outside the golangci fixer's reach), second run = no-op; oauth2 build/vet/test + both touched examples green (prior §f.14 + §f.15).
- **shfmt + shellcheck enabled** (`a7539c81`): all 6 findings fixed first (dead `SKIP_MODULES` in dep-budgets, unused `usermgmt_ver` in docs-freshness, never-read `drift_found` in version-drift, SC2155 local-assign split in replace-directives, SC1091 directive on the lib source, sed→parameter-expansion), then treefmt programs enabled; scripts normalized to shfmt 2-space (18 files, +626/−628 pure layout). Verified: shellcheck 0 findings across all scripts, `nix fmt` idempotent, every touched gate re-run functionally (dep-budgets/docs-freshness/replace-directives green; version-drift advisory 0 / strict 1) (prior §f.16).
- **check-modules stage status recorded** (`b15b140d`): full chain run end-to-end — isolation ✅ (27 modules), dep-budgets ✅, go-toolchain ✅, drift --strict ✗ (documented), then release-train ✅ / replace-directives ✅ / docs-freshness ✅ / docs-links ✅ run individually (chain aborts at drift) (prior §f.35 + §f.36).
- **Docs gates on the new/edited docs**: freshness ✅, links ✅ (210 links) (prior §f.36).
- **Status-report bookkeeping**: the prior session's report was already committed+pushed by the daemon as `845b48ba` before this session started (my commit+push tasks were already done).

### v4.8.x family alignment train (user-approved via question prompt)

- **Tags cut + pushed (all signed, verified via `git tag -v` / `git ls-remote`)**: `usermgmt/v4.8.1` (Journal()/EventBus() accessors, verified present at the tagged commit via `git grep` on HEAD), first-ever `usermgmt/totp/v4.8.1` + `usermgmt/oauth2/v4.8.1` (closing the phantom-require family checklist), `dashboardui/v4.8.2` (SSEMaxReplay), `setup/v4.8.3`. Runbook §1b/§1d were already satisfied — `dashboardui/v4.8.1` and `datastar/v4.8.1` were found already pushed (stale precondition).
- **All 5 DEV-ONLY replaces stripped**: setup's 2 (`../usermgmt`, `../dashboardui`) + setup-demo's 3 (`../../setup`, `../../usermgmt`, `../../dashboardui`); both modules verified hermetically (tidy/build/vet; setup also full test, 8.1s suite green) resolving everything from freshly pushed proxy tags.
- **admin-demo totp pin bumped** v4.7.0 → v4.8.1 (runbook §3), hermetic build/vet green.
- **Family require alignment**: 7 modules bumped to usermgmt v4.8.1 (integration_test, admin-demo, samber-do-demo, system-demo, adminui, loginpage, systemadapter) + setup-demo/setup/dashboardui consumers at dashboardui v4.8.2 → **usermgmt and dashboardui axes fully uniform** (prior "align refs to the family version" from TODO P3 drift item).
- **Final gate state at `bacbea2b`**: isolation ✅ (27 modules GOWORK=off), test ✅, lint ✅ (0 issues), check-release-train ✅ (0 unpublished, 1 replace-exempted — systemadapter's documented TEMPORARY metaengine replace — 366 train-lag advisories), toolchain ✅, bench-spike ✅ (green earlier this session; request paths untouched since), docs gates ✅, `nix flake check --no-build` ✅.
- **Docs updated**: CHANGELOG train entry, runbook marked EXECUTED with deltas, TODO_LIST (decisions item resolved, drift axes rewritten, header), AGENTS.md (replaces-inventory gotcha rewritten post-train + new stale-tag lesson gotcha).

---

## b) PARTIALLY DONE

1. **Strict version-drift: reduced to the achievable floor, still RED.** All cqrs-htmx family axes are now uniform, but 5 go-cqrs-lite sibling axes (`go-codec`, `event`, `metadata`, `metaengine`, `record`) remain split between the 20 standard modules and the 2 modules (systemadapter, examples/system-demo) carrying the documented TEMPORARY metaengine local-master replace. Blocker: go-cqrs-lite upstream tags (`metaengine/v4 v4.12.0+`) — other repo, user's call there. Effort to finish after upstream: M.
2. **check-modules chain still aborts at the drift stage**, so release-train/replaces/docs stages have no chain coverage (I ran them individually). A `--report` (run-all-stages) mode would fix the blind spot. Effort: M.
3. **CI release-train check is advisory only** (`|| true`). Blocking flip needs one green advisory week AND confirmation of repo visibility (see g1). Effort: S once unblocked.
4. **pkg.go.dev verification**: proxy resolution is proven (hermetic builds downloaded setup/v4.8.3, usermgmt/totp/oauth2 v4.8.1 from the proxy during this session), but the pkg.go.dev pages themselves were not checked. Effort: S.
5. **Bench baseline is now ~20% stale**: the quiet-machine green run measured all three sub-benches 18–23% FASTER than the pinned baseline. The gate is green but its regression sensitivity is effectively halved until a re-pin. Re-pin is a one command, but the policy (when to re-pin, whether machine speed drift should reset the baseline) is undecided — see g3.
6. **Prior report's 3 open questions**: Q1 (tag version) RESOLVED — executed at v4.8.1 per the runbook the user approved; Q2 (format-sweep timing) DISSOLVED — the sweep collapsed to 4 files and landed; Q3 (repo visibility) INFERRED public from CI evidence (no auth/secret wiring, plain `go build` of larsartmann modules) but UNCONFIRMED — hence still g1.
7. **Runbook §4's "drift must turn GREEN"** was not achieved literally — the reachable floor (cqrs-htmx axes uniform) is what shipped; full green is upstream-gated.
8. **docs-health HARVEST of this report's §f** into TODO_LIST/ROADMAP not yet run (the session's major outcomes ARE already in TODO_LIST; the 50 items below are not).

## c) NOT STARTED

1. **Full coordinated family version sweep** (prior §f.9): cut the coordinated tag set (root + remaining submodules at one family version) — the runbook the user approved covered the blocking tags + strips + alignment of the drift axes; the wider uniform-version cut (root v4.8.x, adminui, identity-model, loginpage, webauthn, health, auditlog, datastar cross-refs) was NOT part of it and remains (366 train-lag advisories).
2. **go-cqrs-lite upstream tags** (metaengine v4.12.0+, projectionadapter v4.5.0, sqliteengine v4.0.2 re-validation) — other repo, user-gated; unblocks the drift floor.
3. **systemadapter first tag** — blocked on 2.
4. **Retraction of the poisoned `setup/v4.8.1` + `setup/v4.8.2`** (go mod edit -retract landing in a future setup release; upstream precedent: storage/v4.7.0). Noted in CHANGELOG, not executed.
5. **go-appkit wave push + ADR-001 fold-in** — user gate in the go-appkit repo; untouched.
6. **CI blocking flip for check-release-train** — pending g1 + green week.
7. **Tag cache for ls-remote** (prior §f.24) + **pre-commit train gate** (§f.23) — deliberately deferred (30s network in every commit without the cache is bad UX); untouched.
8. **`/sse` authz posture decision** (session-gating vs `sse.ReplayFiltered` stream-type filter) — the one remaining open decision in TODO P2; untouched.
9. **datastar/go-sse architecture ADR** — carried P3, untouched.
10. **v4 branch binary purge + setup-demo 27MB blob purge** — force-push class, user-gated, untouched.
11. **`/mnt/buildcache` hardware** — external, untouched.
12. **daemon-glitched commit message** (`传闻` artifact inside pushed `845b48ba`) — noticed, recorded here; fixing requires history rewrite (forbidden without approval). Cosmetic only.
13. **test-flake / test-fuzz / e2e (bun+playwright) suites post-train** — not re-run this session (module content on the tagged paths is unchanged from the green 2026-08-29 sweep, but the version-reshuffled graph has not had the full auxiliary suites against it).

## d) TOTALLY FUCKED UP

1. **Two poisoned published tags: `setup/v4.8.1` and `setup/v4.8.2`** (my doing, this session). Severity: does not block workspace dev; breaks hermetic consumers pinning exactly those versions. Root causes — TWO distinct mistakes chained: (a) I tagged `setup/v4.8.2` while the go.mod fix (dashboardui v4.8.2 require) was still UNCOMMITTED — tags point at commits, not the working tree; (b) the underlying `dashboardui/v4.8.1` require pointed at a tag that never had SSEMaxReplay. Mitigation: superseded by `setup/v4.8.3` (verified via `git show setup/v4.8.3:setup/go.mod` BEFORE push this time); retraction pending (c4). The verified-then-push step must have happened for v4.8.1/v4.8.2 too — it only happened after the first failure.
2. **Trusted a stale runbook precondition without content verification**: the runbook (written yesterday) said dashboardui/v4.8.1 was "cut locally" carrying SSEMaxReplay. It was actually pushed at a stale pre-SSEMaxReplay commit (`ca7b7ba0`). I discovered this via a hermetic build failure, not by pre-verifying with `git show` — the exact discipline the v4.8.0 session used ("verified via git show") and that I skipped. Cost: the whole v4.8.1 setup/dashboardui tag ladder.
3. **Malformed question-tool call**: I sent a placeholder free-text question ("Placeholder") in the batch. The user answered "Do not get it" — a direct symptom of my malformed UX — and I proceeded without acknowledging or re-asking cleanly. The two real questions (repo visibility; retraction appetite) therefore remain unasked (folded into g).
4. **Alignment loop reported 3 spurious FAILs**: my log-filename used the module path verbatim (`/tmp/align-examples/admin-demo.log` — slash creates a missing directory), so results were lost and misreported. Re-ran all 3 successfully; wasted a cycle.
5. **Ran the first bench gate under concurrent heavy load** (another session building, load 8.7): got a garbage ±25% failure before checking `uptime`. This is the SAME mistake the prior session documented in its §d.8 — repeated once before I checked. The failing runs did double as override-path verification, but that was luck, not plan.
6. **AGENTS.md table edit split a row**: my edit anchored on an unpadded substring and injected a new row mid-line (the table rows carry alignment padding). Caught by immediate view, repaired next edit. Exact-match discipline slipped on a doc file where I assumed padding.
7. **Introduced a trailing space** after `)` in check-dep-budgets.sh during the dead-variable removal (my new_string had `)` + newline). Caught via `cat -A`, repaired.
8. **Cross-file multiedit misfire**: an edit intended for check-version-drift.sh (a `drift_found` noqa comment) was sent in the check-replace-directives.sh batch — it failed harmlessly there, and the approach itself was abandoned (full removal was correct). Two sloppy steps in one edit call.
9. **CHANGELOG history incoherence (minor)**: the cache-guard CHANGELOG entry landed in the bench commit (`80109cb3`) instead of the cache-guard commit (`043a409d`) — I batched it to avoid a fiddly amend; the entry describes a feature from the previous commit.
10. **git tag -v output ordering**: minor — verified signatures post-push rather than pre-push; pre-push verification would have caught nothing extra here (signature ≠ content), but `git show <tag>:<path>` verification is the load-bearing check and I only institutionalized it after mistake 1.

## e) WHAT WE SHOULD IMPROVE

1. **Institutionalize the tag protocol as a script, not a habit**: `verify-tag.sh <module> <version>` that (a) fails if the module dir has uncommitted changes, (b) tags, (c) runs `git show <tag>:<mod>/go.mod` content assertions, (d) pushes. Both release incidents across two sessions are this one missing guard. Impact: high — each miss cost a poisoned published version; Fix: small script + runbook template update.
2. **Never trust a runbook precondition table older than the session**: two of this runbook's preconditions were wrong within 24 hours (tags pushed elsewhere). Re-derive from `git ls-remote`/`git show` every time; the runbook should say "verify, do not trust" at the top. Impact: medium; Fix: one header line + my own discipline.
3. **Check load before ANY timing-sensitive gate** (bench): second session in a row burned runs on concurrent-build noise. Fix: `uptime` guard inside the bench-spike app itself (fail or warn above a load threshold) instead of relying on the operator remembering.
4. **Question hygiene**: one well-formed batch; never placeholder questions; if the user's answer indicates confusion ("Do not get it"), acknowledge and re-ask cleanly instead of silently proceeding. Impact: the two real decision questions went unanswered.
5. **CHANGELOG entries ride with their feature commits** — self-contained history beats amend-avoidance. Impact: low but recurring.
6. **Multiedit discipline**: View the target region in its CURRENT state before batching (two round trips lost to file-not-read/mtime-changed this session); never include an edit for file B in file A's batch.
7. **Loop hygiene in shell**: sanitize path-derived filenames (`tr '/' '_'`) or use mktemp; this bug class produced false FAILs mid-alignment.
8. **check-modules `--report` mode**: the chain's abort-at-first-red means 4 stages have no chain coverage and must be remembered manually. A run-all/summarize mode makes the floor state visible in one command (feeds TODO_LIST updates too).
9. **Retraction playbook**: document `go mod edit -retract` in the release runbook template (upstream precedent: storage/v4.7.0) so poisoned tags get retracted mechanically, not "noted in CHANGELOG".
10. **Bench baseline policy**: pin a rule — e.g. "re-pin when the machine's quiet-state medians drift >10% from the baseline OR after any machine change; record load context in the baseline header" — instead of ad-hoc judgment each session (see g3).

## f) 50 THINGS TO GET DONE NEXT (Pareto-ordered; brainstorm, HARVEST-routes to TODO_LIST/ROADMAP)

**P1 — train follow-through (this week)**

1. Confirm `larsartmann/*` repo visibility (public vs private) — unblocks the CI blocking flip (see g1). Impact: High | Effort: S | Decision.
2. Verify pkg.go.dev resolves all 7 new tags (usermgmt/totp/oauth2 v4.8.1, setup v4.8.1–3, dashboardui v4.8.2). Impact: High | Effort: S | Documentation.
3. Add retract directives for `setup/v4.8.1` + `v4.8.2` (land in the next setup release; storage/v4.7.0 precedent). Impact: Medium | Effort: S | Cleanup.
4. Flip `check-release-train` to CI blocking after one green advisory week. Impact: High | Effort: S | Quality.
5. go-cqrs-lite (OTHER repo, user gate): tag `metaengine/v4 v4.12.0+` + re-validate the projectionadapter/sqliteengine §3 mappings against the ADR-0128-extraction tree. Impact: Critical (unblocks drift-green + systemadapter tag) | Effort: M | Feature.
6. Strip systemadapter + examples/system-demo TEMPORARY metaengine replaces after 5; strict drift turns fully green for the first time since 2026-08-15. Impact: High | Effort: M | Cleanup.
7. Cut systemadapter's first tag (post-6). Impact: High | Effort: M | Feature.
8. Family train-lag sweep: align the remaining 366 lag refs (root v4.8.1 consumers, adminui/identity-model/loginpage refs in examples, catalog/snapshot/storage stragglers) using the v4.8.0 textual-drift pass pattern; collapse check-release-train lag 366 → near-0. Impact: High | Effort: M | Cleanup.
9. Cut the coordinated family tag set (root + submodules at one family version, runbook §4 pattern of the v4.8.0 train) after 8 (GATED: user). Impact: High | Effort: M | Feature.
10. Re-run `nix run .#check-modules` end-to-end post-8; record the (first-ever fully green?) stage table in TODO_LIST. Impact: Medium | Effort: M | Quality.

**P2 — gate/tooling hardening (no user gate)**

11. Tag cache for ls-remote (TTL + refresh on `git fetch --tags`) — check-release-train ~30s → ~1s. Impact: Medium | Effort: M | Feature.
12. Pre-commit train gate (only after 11). Impact: Medium | Effort: S | Quality.
13. `check-release-train --json` machine-readable output for train planning. Impact: Medium | Effort: M | Feature.
14. `check-release-train --strict-lag N` mode (fail on lag beyond N post-train). Impact: Medium | Effort: S | Quality.
15. `check-modules --report` mode: run all stages, summarize red/green (kills the abort-at-drift blind spot). Impact: Medium | Effort: M | Quality.
16. In-app load guard for bench-spike (warn/fail when load average exceeds a threshold; refuse garbage comparisons). Impact: Medium | Effort: S | Quality.
17. Re-pin the bench baseline per the policy decided in g3 (machine now ~20% faster than the pin). Impact: Medium | Effort: S | Cleanup.
18. Extend go-cache-env preflight to check the GOMODCACHE filesystem too (currently GOCACHE only). Impact: Low | Effort: S | Quality.
19. Add CI assertion that `go.work.sum` stays untracked (`git diff --exit-code -- go.work.sum` in mod-tidy job). Impact: Low | Effort: S | Quality.
20. Verify batch-release.sh still functions after the shfmt reformat (dry-run). Impact: Low | Effort: S | Quality.
21. Document the poisoned-tag ladder (setup v4.8.1→v4.8.3) in the release runbook template + the verify-tag protocol from e1. Impact: High | Effort: S | Documentation.
22. Write `scripts/verify-tag.sh` (e1: uncommitted-changes guard → tag → `git show <tag>:<mod>/go.mod` assertions → push). Impact: High | Effort: M | Feature.
23. Confirm buildflow pre-commit formatters and the new treefmt programs don't double-format staged scripts; document the division of labor. Impact: Low | Effort: S | Documentation.
24. Wire check-require-tags' zero-pseudo leg into CI's module-architecture job as the canonical step (replacing the inline grep copy). Impact: Low | Effort: S | Quality.
25. docs-health HARVEST: route this §f list into TODO_LIST/ROADMAP. Impact: Medium | Effort: S | Documentation.

**P3 — quality/docs**

26. Document ServiceConfig precedence in `docs/guides/fullstack-wiring.md` (carried from prior §f.48). Impact: Medium | Effort: S | Documentation.
27. Contract tests for consumer-supplied `LockoutStore`/`SessionStore` via the ServiceConfig hatch (prior §f.47). Impact: Medium | Effort: M | Feature.
28. `TestNew_ServiceConfig_WithSQLReadModels`-style persistence integration test (prior §f.27). Impact: Medium | Effort: M | Feature.
29. samber-do-demo: assert the Hello command end-to-end over the mux (prior §f.28). Impact: Low | Effort: S | Feature.
30. setup-demo: read an actual SSE frame after broadcast instead of status codes only (prior §f.29). Impact: Medium | Effort: M | Feature.
31. Run `.#test-flake` + `.#test-fuzz` + the e2e (bun/playwright) suite once against the post-train graph (untouched this session). Impact: Medium | Effort: M | Quality.
32. Stale-doc sweep: any remaining "totp pinned to v4.7.0" / "cut locally, NOT pushed" mentions in AGENTS.md/guides after the train. Impact: Low | Effort: S | Documentation.
33. Move the CHANGELOG train entry into a proper version section at the next release cut. Impact: Low | Effort: S | Documentation.
34. datastar/go-sse architecture ADR (carried P3, weeks old). Impact: Medium | Effort: M | Documentation.
35. `/sse` authz posture: decide session-gating vs `sse.ReplayFiltered` stream filter before the endpoint shape is published (user decision). Impact: High | Effort: M | Decision.
36. Add the verify-tag protocol (e1) as a docs-health/AGENTS checkable rule (like the no-[x] convention) so future sessions inherit it. Impact: High | Effort: S | Documentation.
37. `check-release-train`: cache-friendly structure (separate tag-fetch from classification) to prep for 11. Impact: Low | Effort: S | Quality.
38. Consider renaming the remaining misnamed gate: check-version-drift's strict leg vs check-release-train overlap — clarify headers or merge. Impact: Low | Effort: S | Documentation.
39. Post-train consumer smoke: one `GOWORK=off` build of each example resolving ONLY from pushed tags (setup-demo done this session; the other 10 examples not). Impact: Medium | Effort: M | Quality.
40. Record the 2026-08-30 stage-floor table in the runbook post-execution notes (drift axes enumeration with module attribution). Impact: Low | Effort: S | Documentation.

**P4 — pre-existing debt (scheduled/user-gated)**

41. v4 branch binary purge (~27.7MB, filter-repo + force-push; user gate). Impact: Low | Effort: L | Cleanup.
42. setup-demo 27MB blob purge from pushed master history (force-push; user gate). Impact: Low | Effort: L | Cleanup.
43. `/mnt/buildcache` hardware decision (replace vs abandon the documented /tmp workaround). Impact: Medium | Effort: — | Decision.
44. cqrs-lint Go-installable distribution → unblocks check-cqrs-lint in CI. Impact: Medium | Effort: L | Feature.
45. Upstream ask: decouple go-cqrs-lite `stack/v4` root package from `metaengine/v4` (kills a transitive require). Impact: Low | Effort: L | Feature (upstream).
46. Go-based markdown link checker evaluation (goldmark) vs the awk checker. Impact: Low | Effort: M | Quality.
47. `WithOpenAPI` collector (metadata stored, nothing reads it yet). Impact: Low | Effort: M | Feature.
48. v5 removal bundle inventory: sweep the deprecated re-export layers (`sse_event.go`, `csrf_reexport.go`, `Raw()` accessors) into one checklist. Impact: Medium | Effort: M | Cleanup.
49. `listing`-based SSE filtering (`sse.ReplayFiltered`) spike to inform 35. Impact: Medium | Effort: M | Feature.
50. Schedule the next status-report + HARVEST session after the NEXT train (post-9) so TODO_LIST reflects the post-family-version world. Impact: Medium | Effort: S | Documentation.

## g) QUESTIONS I CANNOT ANSWER MYSELF

1. **Are the `larsartmann/*` repos public or private?** What I tried: CI workflow inspection — `ci.yml` builds larsartmann modules with plain `go build` and NO auth/secret/token steps (only `GONOSUMCHECK` env), which only works for public repos; local pushes use SSH so they prove nothing about HTTPS/Actions. The evidence says public, but I cannot confirm, and the answer decides whether `check-release-train`'s CI advisory output is even meaningful (on private repos it self-downgrades to WARN always) and when it can go blocking.
2. **Retract the poisoned `setup/v4.8.1` + `setup/v4.8.2` tags?** I know the mechanics (`go mod edit -retract` in setup's go.mod landing with the next setup release; upstream precedent: go-cqrs-lite retracted broken storage/v4.7.0). The decision is risk-appetite: retraction keeps naive consumers off broken versions but adds release overhead; leaving them superseded-but-published costs nothing today. Your call.
3. **Bench baseline re-pin policy?** The pinned baseline (`setup-baseline.raw.txt`) now sits ~20% ABOVE this machine's quiet-state medians (measured this session), so the 10% regression gate effectively tolerates real regressions up to ~30% vs current speed until a re-pin. Options: (a) re-pin now — restores sensitivity, erases the headroom; (b) keep stale — conservative gate; (c) adopt a written policy (re-pin when quiet-state medians drift >10% or after machine changes, with load context recorded in the baseline header). I can implement any of them; the risk appetite is yours.

---

**Verification summary for this report:** HEAD `bacbea2b` = origin/master, tree clean. Gates at HEAD: isolation ✅ (27 modules), test ✅, lint ✅ (0 issues/15 modules), check-release-train ✅ (0 unpublished / 1 replace-exempted / 366 lag), check-go-toolchain ✅, bench-spike ✅ (green at the 80109cb3 state; no request-path changes since), docs-freshness ✅, docs-links ✅ (210 links), `nix flake check --no-build` ✅. Strict version-drift: red at the upstream-blocked floor (5 go-cqrs-lite axes, 2 modules). Tags pushed this session: usermgmt/v4.8.1, usermgmt/totp/v4.8.1, usermgmt/oauth2/v4.8.1, dashboardui/v4.8.2, setup/v4.8.1+2 (poisoned, superseded), setup/v4.8.3.
