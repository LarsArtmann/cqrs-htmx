# g1/g2/g3 execution + go-cqrs-lite CI triage + verification battery (11-10 session)

**Session:** 2026-11-18 ~05:35–11:10 CET · **Scope:** resume the 22-25 train-closure report's §g owner questions (g1/g2/g3) with the owner's blanket "execute the whole list" directive; triage the go-cqrs-lite master CI red window discovered at state verification; run the standard receipt battery. Three repos touched: go-cqrs-lite (2 fixes pushed), cqrs-htmx (gate + docs + baseline), BuildFlow (doctor fix committed, not pushed). Active sibling sessions observed throughout (overnight CI-triage + fleet-alignment reports, bump-dep sweeps landing mid-session as 53-file daemon blobs).

## a) Fully done

1. **go-cqrs-lite CI red triage (the 3-day red window).** 15+ consecutive `ci.yml` failures on master since the `/v4` reorganization commit `0e4fbb3d2`. Root cause: three jobs `cd` into depth-3 module dirs (`cmd/cqrs-lint/v4`, `cmd/cqrs-upgrade/v4`, `stack/duckdb/v4`) but climb only two levels, so their example/bench paths resolved under `cmd/` or `stack/` where nothing exists. Worst part: the **cqrs-lint-examples job has been a FALSE GREEN since the move** — its `../../example/*/` glob matched nothing, the literal-pattern iteration errored, and `|| true` swallowed the error, so "V007 must be silent in every example" has verified zero examples. Fixed all three (`../../../` paths) plus an empty-glob guard that fails the job loudly (`::error::example glob matched nothing; check the relative path`). Verified every chain locally (9 examples now actually match). Committed **159818b9a**, pushed. Nightly Benchmarks calibration-drift red = ubuntu runner migration (image annotation names it) — separate lane, not triaged here.
2. **g2 decided + executed: `release.yml` tag trigger FIXED upstream.** The trigger only matched `v*` and one-level `*/v*`; GitHub tag patterns never let `*` cross `/`, so all six v4.22.x dual-shape deep tags (`cmd/cqrs-lint/v4/v4.22.2` + compat short shape) fired ZERO Release runs and the GH releases had to be hand-created. Added per-depth patterns `*/*/v*` + `*/*/v*/v*` with the trap documented in-file; the resolve step already handled deep tag names correctly (verified: `cmd/cqrs-lint/v4/v4.22.2` → MOD `cmd/cqrs-lint/v4`), so auto-GH-release is restored for every future deep-module train. actionlint clean. Committed **30d8c6c39**, pushed.
3. **g1 decided + executed: gate candidates 14 → 26.** Every module with a `.cqrs-lint.json` is now a gate candidate — `examples/*` ×10, `e2e/server`, `gfl-usermgmt` joined the 14 library modules. All 12 were verified clean per-module BEFORE joining (`--scope=module --strict --fail-on-stale-suppressions`: rc=0, zero findings, no load errors — catalog-demo included). Gate green **26/26 candidates, zero findings**; self-test green; shellcheck green.
4. **The "15 workspace-walk findings" question resolved by measurement.** Re-running the root workspace walk surfaces **109 findings**, not 15 — and per-module walks surface zero. The reconciliation: the workspace walk applies ONLY the root config (module configs are not merged there by design), so its output is the PRE-TRIAGE view; every class it shows (C044 ×50, E004 ×22, E023 ×11, C009, the e2e batch-size advisory, catalog-demo demo-code) already carries a dated verdict in its owning module's config from the same-day 2026-11-17 triage wave (verified at source: e2e/server disables C044+F005+the batch-size rule with reasons; usermgmt/gfl carry the ActorID form-consistent verdict; catalog-demo the demo-code set). Adjudicated **diagnostic-only** — recorded in the gate script, AGENTS.md, and the skill: the gate and the BuildFlow toolsdk, both per-module with ancestor-merged configs, are the only enforcement surfaces.
5. **g3 decided + executed: BuildFlow `env/binary-freshness` made content-aware.** Exact build-commit==HEAD identity was the only green state, so dogfood mode (blocking) was permanently red on a daemon repo and rebuilding chased HEAD forever. New rule: a binary whose commit is an **ancestor of HEAD whose delta touches no compiled surface** (Go sources other than tests, `go.mod`/`go.sum`, `vendor/`) counts as fresh; a **code-bearing delta stays stale** (dogfood FAIL kept — the 2026-09-02 incident class stays covered); a **non-ancestor commit stays stale and now says so** in the message; merge-base exit 128 (unknown revision — fake hash, rebase discard, foreign checkout) classifies as not-an-ancestor; probe failures fail closed. Implemented in `discovery/doctor/checks/binary_freshness.go` + 4 new tests and a filter unit test (`binary_freshness_1_test.go`); full discovery suite green; gofmt clean. Committed **c830330e3** (NOT pushed — rule 11 + the hazard in §g3).
6. **g3 receipt captured.** Rebuilt + `.#reinstall`ed the profile binary (now `c830330`, `buildflow version` verified); dogfood doctor: **✓ env/binary-freshness "binary is current (built at HEAD c830330)"** — the permanent-red dogfood case is dead. Fleet-mode doctor on cqrs-htmx: the advisory now correctly SELF-EXPLAINS with the new non-ancestor note (see 7).
7. **Dual-checkout hazard discovered and recorded.** The fleet doctor run exposed TWO BuildFlow checkouts on this host: `/home/lars/projects/buildflow` (lowercase, this session's tree, ahead 11: the v4.22.x pin train + vendor hashes + the g3 fix) and `/home/lars/projects/BuildFlow` (capitalized — the name fleet sibling-discovery targets, ahead 4: the gomod-freshness testdata exclusion + erraudit-gate work), with **diverged unpushed histories** (different merge-bases with origin). Consequences recorded: fleet freshness compares against the capital tree (lowercase-built binaries read as foreign), and fleet `gomod-freshness` still flags the 7 verify-tag fixtures because the lowercase-built binary lacks the capital tree's exclusion. TODO_LIST row added; NOT reconciled here (neither tree is mine to merge or push).
8. **docs-gates annotate repaired to PASS:** the two rule-F "space stripped around inline code" findings (the `.\`#reinstall\`ed` adjacency class — one pre-existing from the train-day AGENTS receipt, one mine) fixed by rewording to "reinstalled via `.#reinstall`". Battery now 3/5 PASS (annotate, routing, cadence).
9. **branching-flow +1 adjudicated and re-pinned.** The composite surfaced +1 NEW finding: `COMPOSITION_mixin` on `examples/admin-demo/openapi.go:41` — `directoryUser` shape-shares 2 fields with `usermgmt.ImportUser` (the overnight B14 admin-demo QUERY/openapi wave's code; this session authored none of it). Accepted under the STANDING mixins reject verdict (flat wire DTOs are the published format) + the 2026-10-13 demo-noise class; baseline re-pinned 211 → 212, amendment row in `triage-decisions.md`, hand-commit **6b091f5e**; gate + fixture self-test green after.
10. **Receipts:** `check-upstream-treadmill`: 20 repos · 81 prefixes · **worst lag 1** (`go-etag/entitytag` v0.6.1 → v0.6.2 — a NEW upstream tag since the 11-15 sweep) · 0 disagreements (advisory). `/tmp` scratch cleaned (f18 list already gone; two `bf-lint*.log` trashed). TODO_LIST gained the f20 release-dance-helper design row.

## b) Partially done

1. **`check-modules` full composite: no clean receipt this session.** First run 104/106 (branching-flow + one unnamed second); reruns DEGRADED to 6/106 then 8/106 — the sibling sessions' bump-dep sweeps landed mid-gate as daemon blobs (a 53-file commit during one run), so the composite raced a moving tree. Individually verified green this session: branching-flow (post-re-pin), cqrs-lint 26/26, shellcheck, root-files, treadmill, the fixture self-tests that ran. Red individually: **route-postures — dashboardui manifest pins** (`TestRoutes_MatchRegisteredMuxPatterns`): triage test was still RUNNING at report time; the same-commit rule (fix the manifest or update the golden WITH the route change) applies, and the B-wave sibling owns recent dashboardui changes — coordinate before touching.
2. **docs-gates battery 3/5.** `freshness` RED: driven by the overnight session's newest report (`2026-11-18_06-53_master-ci-red-window...`) whose §f items are unharvested — that report's closure step, not this session's (its items: P04 README license, P13 bridge guide, bump-dep retry — none from this session's report; the 22-25 report's §f items are annotated DONE/RESOLVED or TODO-rowed here). `audit-all` RED: **300 unharvested vs pinned 268** — the known, TODO-tracked owner decision (drain by harvesting or owner re-pin), volume growth from the new reports.
3. **CI watch:** both fixes pushed, but the matrix (~2h) had only reached in-progress/queued at report time — no green verdict yet. Note: the examples loop now lints 9 REAL examples for the first time since the `/v4` move, so genuinely new findings may surface.
4. **f20 release-dance helper:** TODO row + design-note pointer only; the design note is not written.

## c) Not started

- §f7 fleet phantom-class survey (BuildFlow+cqrs-lint over every `~/projects` go.work repo).
- §f8 BuildFlow release cut carrying the doctor fixes (blocked on the dual-checkout reconciliation).
- §f11 golines survival check of the five inline `//cqrs-lint:ignore` directives.
- §f12 port the per-module cqrs-lint + scope-flag notes into `how-to-golang`/`linter-building` skills.
- §f13 E005 cross-module registration decision upstream.
- §f14 ADR-0059 CBOR extension backlog row (CredentialRemoved/CredentialUsed/BotRegistered/TOTPEnabled payloads).
- §f15 the erraudit-style findings-ratchet design note.
- §f16 BuildFlow `system/io-history-ratchet` warning (195 files vs median 10) + the 1.02 GB state DB (VACUUM candidate).
- §f19 `buildflow timings` re-baseline on an idle machine.
- The `entitytag` v0.6.2 consumer sweep (new treadmill tail).
- All carried TODO_LIST P1/P2 waves (fleet presentation plan K1, direct-imports W1–W3, M2M W1/W2, dispatch-boundary consumer adoption, and the rest).

## d) Totally fucked up

- **Nothing destructive.** No data loss, no wrong pushes (the two go-cqrs-lite pushes are the CI-fix path itself; BuildFlow stayed unpushed deliberately), no reverted work, no stomped sibling files (verified at every edit: the files I touched were stable in HEAD; the B-wave's WIP files were never opened for edit).
- **The one real process failure:** I ran the 106-gate composite three times against a demonstrably live-mutating tree — receipts degraded 2 → 6 → 8 fails from concurrency, costing ~30 minutes and producing noise instead of a receipt. The signals were visible in advance (fresh `/tmp/bump-dep-a-*.log` files, daemon blobs landing between my own commands, sibling mtime churn). The individual-stage verification pattern I used for branching-flow was right; I should have applied it to the whole battery the moment the first degradation appeared.
- **Minor:** one `multiedit` on AGENTS.md bounced (file modified since read — the session-start snapshot was stale); the view-before-edit rule caught it, one round trip paid.

## e) What we should improve

1. **Quiet-tree precondition for composite receipts:** before any multi-minute gate, check sibling activity (fresh `/tmp` artifacts, `git status` churn rate, recent daemon commits). If the tree is hot, verify per-STAGE and capture the composite only in a settled window.
2. **Degraded reruns should shrink, not repeat:** when run N fails M stages, rerun only those M stages standalone (the composite's value is the aggregate receipt, not the retry loop).
3. **Run `docs-gates annotate` immediately after doc edits** — it costs 400 ms and catches the rule-F class (my own edit introduced one of the two findings; the other was inherited from the train-day text that was never battery-checked).
4. **Adjudication template worked — keep it:** measure the finding at source → check the ledger for a standing verdict → accept under existing class with a dated amendment + same-commit baseline re-pin → gate + self-test green. No new verdict classes invented.
5. **CI watch belongs in `push-and-watch`** (the fleet app with workflow filtering + failed-job triage), not manual `gh run list` polling.
6. **The false-green examples loop is a fleet-wide lesson:** a loop that can match zero iterations and still exit 0 verifies nothing — the guard (`[ -d "$ex" ] || fail`) belongs on every glob-driven CI job in every repo. Candidate gotcha.
7. **Baseline re-pins by sessions that authored none of the code** (this session for branching-flow, like the 2026-10-12 precedent) must keep saying so in the ledger row — it keeps adjudication ownership honest.

## f) Next queue (ordered, newest-first for this session's items)

1. Triage the route-postures dashboardui manifest-pins red (test was running at report time) — fix manifest or golden in the same commit as the route change; coordinate with the B-wave session if the routes are theirs.
2. Capture a QUIET `check-modules` composite receipt (all 106) once the sibling bump-dep waves settle.
3. Collect the CI verdict on `159818b9a`/`30d8c6c39` (push-and-watch); triage anything the now-actually-running examples loop finds.
4. Harvest the overnight reports' (06-53, 07-01) §f into TODO_LIST — this greens the freshness gate.
5. OWNER: the audit-all pin decision (300 vs 268) — drain by harvesting or re-pin (existing TODO row).
6. OWNER: BuildFlow dual-checkout reconciliation (TODO row) — after which §f8 (release cut carrying the g3 fix + the capital tree's gomod exclusion) unblocks.
7. Sweep `entitytag` v0.6.2 into the consumer modules (treadmill tail).
8. §f11 golines survival of the five inline `//cqrs-lint:ignore` directives.
9. §f12 port per-module cqrs-lint + `--scope=module` notes into `how-to-golang`/`linter-building`.
10. §f13 E005 cross-module registration decision upstream.
11. §f14 ADR-0059 CBOR extension backlog TODO row.
12. §f15 erraudit-style findings-ratchet design note (escape-hatch ADR while the fail-on-any-finding posture is fresh).
13. §f16 BuildFlow io-history-ratchet warning + 1.02 GB state DB VACUUM.
14. §f19 `buildflow timings` re-baseline on an idle machine.
15. f20 release-dance helper design note (TODO row exists).
16. Candidate gotcha 62/63: the zero-iteration glob false-green class (this session's ci.yml find) — write it into `docs/gotchas.md` + index.
17. K1 (fleet plan): oauth2 `v4.13.1` hotfix-vs-train decision and tag.
18. Direct-imports W1–W3 (v5 window waves).
19. M2M hardening W1/W2 (bot durable grants, realtime feeds, audit events) after the owner-decisions row.
20. Dispatch-boundary (L5) consumer adoption waves + the upstream go-cqrs-lite ask.
21. The `ReadLastN` SQL upstream ask (S55 packet U2, owner tick).
22. The remaining firefox cross-engine divergence (e) in `sync.spec.ts`.
23. Hermetic suites for `examples/samber-do-demo` + `loginpage` (00-15 §c4 inheritance).
24. ExtraNav owner decisions (a) Href scheme rejection, (b) `aria-current` a11y.
25. Fleet phantom-class survey (§f7) once BuildFlow's release path is decided.
26. `docs-gates audit-all` drain-or-repin execution (after the owner decision).
27. erraudit blocking-promotion window closes 2026-12-16 — keep the advisory month green (TOTAL 352).
28. metaengine-family train watch (v4.26.0+ nested-dir bumps when tagged).
29. templ-components popover PR #101 merge watch → retire adminui's vendored corrected layer on the next family train.
30. cqrs-lint Go-installable distribution (the gate is local-only until then; TODO P2/P3).
31. BuildFlow `FileContract.ConfigFiles` upstream ask (retire `BUILDFLOW_NO_RESULT_CACHE=1`, §f6).
32. The stale-proxy-resolution gotcha candidate (45, line 45 row) formalization.
33. e2e battery timing-budget measurement (CI delta vs 2.7 min local).
34. `check-e2e-ports` promote-or-drop decision.
35. e2e README serial-context note.
36. Cached-revision-≠-expected warning in the `e2e-firefox` flake app.
37. `gomod-freshness` fixture-flag re-verification AFTER the capital-tree binary ships the exclusion.
38. Nightly bench calibration drift: re-pin or accept after the ubuntu-26 migration settles.
39. Nightly "Module sums" go.mod-tidiness red triage (go-cqrs-lite nightly, 2026-11-18).
40. The fleet helper for release dances (§f20 execution after the design note).
41. `buildflow doctor` fleet phantom survey receipts per repo (§f7 detail).
42. Port one-line per-module cqrs-lint semantics into the sibling fleet skills (§f12 detail).
43. Verify the canonical-sed e2e fix under the real runner path shape (06-53 report §f34, needs their push).
44. Guard the daemon commit path against binary blobs (06-53 report §f: the 42 MB `server` binary class).

## g) Three questions I cannot answer myself

1. **BuildFlow dual-checkout:** which tree is canonical — lowercase `buildflow` (this session's, ahead 11, carries the v4.22.x pin train + the g3 doctor fix) or capitalized `BuildFlow` (the fleet-discovery name, ahead 4, carries the gomod testdata exclusion)? May I push lowercase master so capital can fast-forward, or are you reconciling manually? This blocks fleet freshness truth, the §f8 release cut, and every fleet repo's gomod-freshness verdict.
2. **route-postures dashboardui red:** are the dashboardui route changes behind the manifest-pins mismatch the B-wave session's in-flight work (I wait for their window), or settled code I should fix now under the same-commit rule (manifest/golden together)?
3. **The now-actually-linting examples loop + entitytag v0.6.2:** if the CI examples loop surfaces genuinely new findings (it has linted zero examples since the `/v4` move), do you want them triaged in this window — and should the `entitytag` v0.6.2 tail be swept as its own mini-train now or bundled into the next scheduled train?

---

*Receipts: go-cqrs-lite 159818b9a + 30d8c6c39 (pushed); BuildFlow c830330e3 (unpushed) + reinstalled binary c830330 + dogfood doctor ✓; cqrs-htmx gate 26/26 zero findings + self-test + shellcheck green, branching-flow re-pin 6b091f5e, treadmill worst-lag-1 advisory, battery 3/5 with the two reds attributed above. The 22-25 report's §f/§g carry the per-item DONE/RESOLVED annotations.*
