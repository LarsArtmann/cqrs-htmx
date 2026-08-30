# Status Report — bench-spike tooling, workspace repair, and the setup.Config collapse design

**Timestamp:** 2026-08-29 20:54 CEST
**Repo:** cqrs-htmx @ `282fcd49` (release v4.8.1), tree clean
**Session:** two working sessions on 2026-08-29 (tooling/repair session, then design session), spanning the v4.8.1 release cut by the concurrent session in between

---

## Context you need first

Two sessions ran concurrently on this tree (documented repo hazard). Between my two working blocks, the concurrent session cut and pushed **v4.8.1** (`release(cqrs-htmx): v4.8.1 - Go 1.26.7 upgrade and dependency consolidation`), which absorbed all of my uncommitted work (flake.nix, go.work, module repairs, docs). Tree is clean at HEAD. Tags `v4.8.1`, `dashboardui/v4.8.1`, `datastar/v4.8.1` are on origin.

---

## a) What is FULLY done

1. **Broken workspace repaired (root cause ×2).** (1) `go.work` declared `go 1.26.6` while root `go.mod` required 1.26.7 — every workspace-mode command failed. Fixed via `go work use` (also deduped a duplicated `id/v4` replace line). (2) The v4.8.0 family train never tagged `usermgmt/totp` (remote max `v4.7.0`), but examples/admin-demo's require had been bumped to the never-published `totp/v4.8.0` — a phantom version in any workspace member poisons the ENTIRE workspace module graph ("unknown revision" errors on unrelated imports). Pinned admin-demo to the published v4.7.0 (content-identical: zero commits touched totp since that tag).
2. **The phantom-version poisoning mechanism documented** — one bad require anywhere = the whole workspace graph fails to load. Recorded in CHANGELOG + TODO_LIST as a process rule for release trains.
3. **admin-demo `go.sum` caught up** on the go-sse v0.5.1 bump it had missed.
4. **`nix run .#bench-spike` shipped** (P2 TODO item closed): runs the canonical `BenchmarkSpikeBaselineVsAppkit` bench (default 5×2s), prints the benchstat baseline→current table, gates each benchmark's median ns/op at >5% regression (exit 1), guards environments (goos/goarch/pkg/cpu mismatch → exit 2 with re-pin instructions), `--save-baseline [path]` to re-pin, env overrides `BENCH_COUNT`/`BENCHTIME`/`BENCH_SPIKE_THRESHOLD`.
5. **benchstat installed** (second P2 item closed): nixpkgs does not package it — built in-flake from `golang.org/x/perf@19be9d8e6c70` (2026-08-25) as `.#benchstat` and added to the devShell. Real hashes bootstrapped via `nix build` error messages.
6. **New raw baseline pinned and gated green**: `docs/benchmarks/setup-baseline.raw.txt` on the AMD Ryzen AI MAX+ 395 (baseline-httputil median ≈38.8µs/op, appkit ≈44.6µs, allocs gap 61→90 — same +30-alloc story as the Xeon artifact, absolute numbers not comparable across CPUs, which is exactly why the env guard exists). Gate OK-path verified live (appkit −4.4%, httputil +0.7%).
7. **Gate-failure paths verified**: synthetic +5% regression → exit 1; `BENCH_SPIKE_THRESHOLD=-1` end-to-end → exit 1 with per-benchmark REGRESSION lines; CPU mismatch guard fires.
8. **Parsing bug found by synthetic test and fixed**: the median-extraction awk used `i < NF`, so a bench line ENDING in `ns/op` (output without `-benchmem`) never matched and would have produced 0.0 medians. The real run only worked because `B/op` follows. Fixed to `i <= NF`; verified against the built program.
9. **Hermetic repairs (same class, four modules)**: setup (stale sums for the sibling-appkit charmbracelet tree), dashboardui (`go` directive → 1.26.7), systemadapter (go-cqrs-lite event v4.8.0 / metadata v4.6.0 / record v4.4.0 floor-raises forced by its documented TEMPORARY metaengine replace). All `GOWORK=off` build + vet green.
10. **Isolation gate self-heals dead caches**: `scripts/check-module-isolation.sh` gained the /tmp cache fallback (the dead `/mnt/buildcache` disk made the gate report uniform cache-init failures masquerading as 25 module failures). Mirrors the guard in the bench-spike app.
11. **metaengine "stale require" mystery solved** (former P3 item): it is NOT stale — usermgmt imports `stack/v4`, and stack v4.3.0's root package (`options.go`, `bundle.go`) imports metaengine packages. The `// indirect` require is load-bearing; `go mod tidy` is correct to keep it. The old TODO premise ("no Go file under usermgmt imports metaengine") was true but irrelevant — dependency-side imports are the edge. Item re-scoped as an upstream ask (decouple stack's option/bundle layer from metaengine).
12. **Toolchain security debt verified closed**: `govulncheck ./...` on root under the devShell's go1.26.7 = "No vulnerabilities found" (GO-2026-6090, GO-2026-5972 resolved). flake.lock now carries a nixpkgs with go 1.26.7; devShell confirmed on go1.26.7.
13. **Docs per convention**: CHANGELOG [Unreleased] Added (bench-spike + benchstat) and Fixed (workspace unbreak, setup sums, dashboardui/systemadapter repairs + gate self-heal, toolchain closure) entries written; TODO_LIST updated (bench items removed, metaengine item re-scoped, drift-splits inventory added, totp/oauth2 train reminder added). My CHANGELOG entries survived the v4.8.1 absorption (verified).
14. **Gates verified at session end**: workspace `go build ./...` + `go vet ./...` green; `check-module-isolation` green (all 27 modules hermetic); dep budgets, replace-directives, docs-links, docs-freshness green; `nix fmt` clean; `nix flake check --no-build` passes.

## b) What is PARTIALLY done

1. **P1 "Collapse setup's mirrored ServiceConfig into one source of truth"** — analysis and design ~90%, implementation 0%. Full field inventory done: setup.Config mirrors 10 of ServiceConfig's ~30 fields; `buildService` passes them through plus a hardcoded `AuditLog: usermgmt.NewAuditLog()`; bundle resolves stores _before_ NewService today but the adoption path proves `svc.Journal()/EventBus()` sourcing works, so the flattened path can be unified onto it. Two designs on the table: **(A) embed `usermgmt.ServiceConfig`** — true collapse, but breaks every consumer literal (`Config{SessionTTL: x}` → `Config{ServiceConfig: usermgmt.ServiceConfig{SessionTTL: x}}`) for zero behavioral gain; **(B′) non-breaking pointer escape hatch `ServiceConfig *usermgmt.ServiceConfig`** — mirrors the existing `Config.Service` adoption vocabulary, crisp precedence (Service > ServiceConfig > flattened fields), conflict validation in the style of `validateAdoptedService`, new ServiceConfig capabilities (SecurityHooks, CheckpointStore, Lockout, TokenPepper, MaxUsers, DrainTimeout, …) reachable from day 0, flattened fields stay as documented convenience until v5. My recommendation is B′; the literal-breaking A belongs in v5.
2. **v4.8.x family train** — root/dashboardui/datastar at v4.8.1 pushed; **`usermgmt/totp` and `usermgmt/oauth2` still have no v4.8.x tags** (totp remote max v4.7.0, oauth2 v4.6.x). The train gap that poisoned the workspace in the first place is still open; admin-demo remains deliberately pinned to totp v4.7.0 until the tag exists.
3. **`check-modules` strict version-drift stage is red** — pre-existing transition splits, inventoried in TODO_LIST: `cqrs-htmx/usermgmt/v4` v4.8.0 vs v4.8.1 (setup only), `record/v4` v4.3.0 (15 modules) vs v4.4.0 (systemadapter, forced by the TEMPORARY metaengine replace), `metaengine/v4` local-master (same replace). Isolation/budgets/replaces/docs stages around it are green. All splits dissolve in the next train.
4. **Dead-disk workaround consistency** — the /tmp cache fallback now exists in three places (AGENTS.md manual workaround, bench-spike app, isolation script) but is copy-pasted, not shared. Works; duplication is debt.

## c) What is NOT done (not started)

1. The P1 implementation itself (see b1) — design session was interrupted by this report.
2. Composition-seams example (P2): extend `examples/setup-demo/` with a `Config.Service` adoption path, `SSEPath` live-feed wiring, a `bundle.Broadcaster` consumer; stop discarding `*cqrshtmx.App` in `examples/samber-do-demo/main.go:57` (`_ = app`).
3. Non-trivial-handler sub-benchmark for `BenchmarkSpikeBaselineVsAppkit` (P3) — note it renames benchmark paths, so the bench-spike baseline must be re-pinned in the same change (my gate fails loudly with re-pin instructions if done separately, which is the intended UX).
4. `golines` integration into treefmt (P3).
5. Stale TODO_LIST housekeeping: the "Bump Go toolchain … BLOCKED on nixpkgs" item (line 31) is back in the file even though it is DONE and verified — my removal was lost when the concurrent session's release absorbed their version of the file (see e2).
6. AGENTS.md Quick Reference still documents the old bench invocation instead of `nix run .#bench-spike`; coverage/lint numbers still dated 2026-08-16.
7. Setup README does not yet document (the not-yet-built) ServiceConfig escape hatch.
8. check-cqrs-lint in CI (blocked: Nix-only binary), datastar/go-sse ADR decision, Go markdown link checker — unchanged.
9. Force-push items (v4 branch blobs, setup-demo binary in pushed master history) — user gate, unchanged.
10. Full test suite + race + coverage + lint sweeps were NOT re-run after the v4.8.1 dependency moves (I verified build/vet/isolation only) — the "all gates green" claims in TODO header still date to 2026-08-16.

## d) What is TOTALLY fucked up

1. **Nothing user-facing is broken.** Workspace, hermetic builds, and all local gates except the (documented, pre-existing) drift stage are green. But three process failures deserve honesty:
2. **I wrote fabricated placeholder hashes into flake.nix for one edit** (invented sha256 strings for the benchstat src/vendorHash) before replacing them with the real ones from `nix build` errors. Nothing shipped wrong — `nix build` catches wrong hashes deterministically and the final values are tool-derived — but inventing values at all violates verify-before-encode. Should have used `lib.fakeHash` as the bootstrap sentinel from the start.
3. **My TODO_LIST edit was silently lost.** The toolchain-item removal I made did not survive: the concurrent session's release commit carried their version of the file, resurrecting the stale item. Collaborative-editing race on a shared dirty tree (AGENTS.md documents the hazard; I was reminded of it the hard way). Un-re-detected, this class of loss is how "done" items rot back into "open".
4. **I hit documented gotchas I'd already been warned about**: pipe-swallowed exit codes (`go get … | tail` reported TIDY: 0 on a failed tidy — the AGENTS.md PIPESTATUS trap) cost a round trip, and `go work use -remove` (flag doesn't exist) was guessed instead of checked. Both recovered quickly, but both were preventable by reading my own repo's gotcha list first.

## e) What we should improve (process fixes)

1. **Release-train checklist must include `usermgmt/totp` + `usermgmt/oauth2`** — two trains in a row have now skipped them. Suggested: a `scripts/check-release-train.sh` that diffs the family tag list against go.work module list and fails if any workspace module lacks the train tag (mechanical, would have caught both incidents).
2. **Concurrent-session doc-loss hazard needs a convention**: before committing docs that another session may also touch, re-grep your own edits (`git diff -- TODO_LIST.md | grep <my-marker>`) and re-apply losses. Better: the auto-git daemon should be suspended during multi-session work, or sessions should touch disjoint files by agreement.
3. **Phantom-version detection ran too late**: `check-phantom-version` exists as a CI app, but the totp v4.8.0 phantom was committed and released before anything flagged it. Worth investigating why the gate didn't catch it (exemption for replace-satisfied requires? not run on that commit? CI advisory-only?) and whether the strict local gate (`check-modules`) should run it on every commit rather than in CI.
4. **Machine-specific baseline artifacts in git are a quiet footgun**: `setup-baseline.raw.txt` is pinned to the AMD box; running the gate on the Xeon machine correctly fails with re-pin instructions, but that means the committed baseline is only valid on one machine. Either document that loudly in `docs/benchmarks/` or gitignore the raw file and make re-pinning part of the workflow.
5. **Extract the cache-fallback guard** into a sourced snippet (e.g. `scripts/lib/go-cache-env.sh`) used by isolation script + bench-spike + any new gate, instead of three copies.
6. **Run the full suite after dependency waves**, not just build/vet: the v4.8.1 wave re-tidied half the repo; tests/race/coverage have not been re-executed since.

## f) Up to 50 next things (Pareto-ordered within tiers)

**Do next (high impact, unblocked, local):**

1. Implement P1 design B′: `setup.Config.ServiceConfig *usermgmt.ServiceConfig` escape hatch + precedence validation + buildService unification onto `svc.Journal()/EventBus()` sourcing (design ready, see b1).
2. Tests for it: override path builds a capability flattened fields can't express (TokenPepper or MaxUsers), Service+ServiceConfig rejection, ServiceConfig+flattened-field conflict rejection, AuditLog default preserved in override mode.
3. Composition-seams example (P2 item): setup-demo adoption path + SSEPath + Broadcaster consumer — can showcase the new ServiceConfig override in the same change.
4. Fix `_ = app` in `examples/samber-do-demo/main.go:57`.
5. Non-trivial-handler sub-benchmark + re-pin `setup-baseline.raw.txt` in one commit.
6. Remove the stale resurrected toolchain item from TODO_LIST (line 31).
7. Verify whether the sibling `metaengine/planner.go:137` `newIdempotencyTracker()` break still exists (workspace builds are green now — the item may be stale); update the TODO item with findings.
8. Document `nix run .#bench-spike` in AGENTS.md Quick Reference (replace the raw invocation) and setup/README#benchmarks.
9. Investigate why check-phantom-version didn't flag admin-demo's totp v4.8.0 before release (e3).
10. Build the train-completeness checker (e1): go.work modules vs family tags, fail on missing.
11. Full gate sweep re-run post-v4.8.1: `nix run .#test`, test-race, coverage + coverage-gate, lint (15 modules).
12. Try stripping setup's dev-replaces (`../usermgmt`, `../`, `../dashboardui`) now that v4.8.1 tags are pushed — hermetic verify per module; reduces the replace pile.
13. Refresh TODO header numbers (coverage/lint dates) after 11.
14. `docs/benchmarks/README` note: raw sidecar vs markdown artifact, machine-specificity, re-pin flow.
15. Extract shared cache-guard snippet (e5).
16. golines into treefmt (P3 item; check `pkgs.golines` availability first).
17. AGENTS.md: record the phantom-poisoning mechanism + train checklist rule (permanent memory).
18. setup README + doc.go: ServiceConfig escape hatch section (after 1).
19. root README quick-start: one line on the escape hatch for advanced consumers.
20. Consider wiring bench-spike into CI as advisory-only (numbers are machine-dependent; probably keep local-only — document the decision either way).

**User-gated (need your call/push):**
21. Cut + push `usermgmt/totp/v4.8.1` and `usermgmt/oauth2/v4.8.1` (second consecutive train to miss them); then bump admin-demo back to the family version and re-pin totp onto the train.
22. go-cqrs-lite upstream: tag `metaengine/projectionadapter/v4 v4.5.0` + `metaengine/sqliteengine/v4 v4.0.2` (runbook §3) → then `systemadapter/v4.8.x` train + strip its metaengine replaces (and examples/system-demo's).
23. go-cqrs-lite upstream: decouple `stack/v4` root package from metaengine (my re-scoped ask) — then usermgmt's `// indirect` require finally drops.
24. Adopt appkit as setup server layer (ADR-001 fold-in, six sub-decisions a–f) — blocked on go-appkit push.
25. `/sse` authz posture decision (session-gating vs stream-type filter).
26. Force-push rewrites: v4 branch blob strip; setup-demo 27 MB blob purge from pushed master.
27. `/mnt/buildcache` hardware attention or a permanent new home for the cache env vars.

**Later / low priority:**
28. Go-based markdown link checker (goldmark).
29. check-cqrs-lint CI distribution (Go-installable cqrs-lint or Nix CI runner).
30. datastar/go-sse architecture ADR.
31. Raise identity-model coverage (75.5% vs gate 70) or lower the gate honestly.
32. adminui coverage margin is thin (68.5% vs gate 66) — add tests before it flakes red.
33. Dedupe SSE/transport examples between docs/guides and doc.go.
34. go.work.sum drift-commit hygiene item (line 34).
35. Consider a `go.work` directive-vs-toolchain gate (go.work `go` ≤ nixpkgs go version — would have caught yesterday's 1.26.6/1.26.7 mismatch before any build).
36. Audit remaining `// Deprecated:` re-export shims (sse_event.go, sse_store.go, security.go, ratelimit/server_timing re-exports) for v5 removal bundling.
37. Sweep stale module counts in docs ("26 modules" vs actual go.work use list — verify actual count).
38. setup-demo: MaxUsers single-user deployment showcase (pairs with 3).
39. Consider `Bundle.Addr()` exposure (appkit fold-in item f) independent of the appkit adoption.
40. Document Bundle.Handler vs Mount vs Run decision table in setup README.
41. Prewarm script relevance check after cache-guard changes (scripts/prewarm-gocache.sh).
42. Treefmt: add shfmt/shellcheck for scripts/*.sh (today only BuildFlow covers them).
43. Review `.golangci.yml` `run.go` floors across module-level configs after future toolchain moves (they lag behind root).
44. Consider benchstat HTML output option for the comparison report (`-format`) in bench-spike verbose mode.
45. AGENTS.md memory: bench-baseline machine-pinning policy.
46. examples/: retire or refresh middleware-showcase against current httputil API drift (verify it still demonstrates 8/8).
47. Sweep for remaining `_ = app`-style discarded-return smells across examples.
48. Add `doc.go` example for ServiceConfig override (godoc visibility).
49. Decide fate of `setup-baseline-2026-08-17.txt` (Xeon artifact) — archive marker vs deletion once raw sidecar is canonical.
50. Re-run `check-docs-freshness` after the doc batch (18/19/44) lands.

## g) Questions I can't figure out myself (max 3)

1. **P1 shape:** Do you want the non-breaking escape hatch (`ServiceConfig *usermgmt.ServiceConfig`, my recommendation — mirror stays until v5) or the full collapse now (embed `usermgmt.ServiceConfig`, breaking every setup.Config literal in consumer code, tests, and docs)? Embedding is the only way to make the mirror _disappear_ today; the hatch only stops it from _growing_.
2. **totp/oauth2 train gap:** Should I cut `usermgmt/totp/v4.8.1` + `usermgmt/oauth2/v4.8.1` locally on current master (you push, per the usual tag gate), or do you want them folded into the next coordinated train? After they exist I'd bump admin-demo's totp require back to the family version.
3. **Baseline ownership:** The raw bench baseline is machine-specific (AMD pin). Do you benchmark on this machine routinely (keep the committed baseline), or do you move between machines (then I'd gitignore the raw file and make `--save-baseline` an explicit local step)?

---

**Verification snapshot at report time:** HEAD `282fcd49` clean; workspace build+vet green; isolation gate green (27 modules); budgets/replaces/links/freshness green; drift stage red on documented pre-existing splits; bench-spike gate green vs the AMD-pinned raw baseline; govulncheck clean.
