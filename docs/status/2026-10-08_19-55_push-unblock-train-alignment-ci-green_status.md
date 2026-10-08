# Push-Unblock: Release-Train Alignment to Zero Lag, Two Pre-Existing CI Reds Fixed at Root Cause, Root v4.13.3 Published — Master CI Green

> Session snapshot: 2026-10-08, ~17:35–19:55 CEST. Operator context: `git sync` from the
> 192.168.1.150 session was blocked by the pre-push release-train gate (33 train-lag
> findings); the fix recipe (15 `bump-dep` sweeps) stalled after sweep 1 on the dirty-tree
> guard. Four other Crush sessions were live on this tree throughout (gotcha-4
> concurrency applied constantly — attribution checked before every tree action).

**Outcome:** push unblocked and landed; master CI **success** (run
[37818614080](https://github.com/LarsArtmann/cqrs-htmx/actions/runs/37818614080) @
`f49fb815`). Along the way two pre-existing CI reds were fixed at root cause, one latent
hermetic-only test red was fixed by publishing root `v4.13.3`, and one fixer/gate split
brain in the status-row tooling was closed permanently.

---

## a) FULLY DONE (with evidence)

| # | Work | Evidence |
| - | ---- | -------- |
| A1 | Release-train alignment to zero lag: 13 families swept in-session (kv v4.3.4, listing v4.4.4, metaengine/projectionadapter v4.5.3, metaengine/sqliteengine v4.5.2, metaengine v4.17.0, scenario v4.4.3, scheduling v4.6.2, signing v4.4.0, snapshot v4.6.2, stack v4.5.0, storage/memory v4.6.2, storage v4.10.5, watermill v4.6.5 — the last already current, no-op) | `nix run .#check-release-train --strict-lag 0` green: 842 requires, 0 unpublished, 0 lag; sweeps at `98f5ea15`…`cb2570bd` |
| A2 | encryption (user's stalled sweep) + id (concurrent session) confirmed landed and counted | `0292a66d`, `652426b5`; re-measured 33→27 findings before starting |
| A3 | Hermetic go.sum residue fix (auditlog `go-runewidth` v0.0.30 hash) — the CI `mod-tidy` red | `e9a9e19b`; hermetic `go mod tidy -diff` loop 28/28 clean |
| A4 | Status-row gate red fixed at root cause: fixer/gate split brain (fixer required a well-formed separator row; the gate flags rows in any pipe-block) — `normalize-status-rows.py` now mirrors `table_blocks`; 2 new regression cases in the fixture self-test | `673e0605`; self-test 9/9 green; gate green across 467 files (43 deliberately-mixed tables untouched) |
| A5 | 40 aborted-annotation rows behind malformed 2-dash separators normalized per the documented whole-row-strike policy | `673e0605`; `check-status-rows.py` rc=0 |
| A6 | CHANGELOG receipt for the battery (gotcha-20 rule: gate/tooling changes are receipt-worthy) | `673e0605` |
| A7 | Root-tag-gap test-form fixed by publishing: root `v4.13.3` cut AT the commit carrying the CSP `img-src` behavior, CHANGELOG-before-tag | `verify-tag.sh . v4.13.3 --push` rc=0, post-push ls-remote verified; receipt `66163813` |
| A8 | All root consumers re-pinned to v4.13.3 (bump-dep exact-anchor sweep) | `56242288`; strict gate re-green after |
| A9 | Full hermetic battery green: `nix run .#test` (flake goApp exports `GOWORK=off` — CI-parity for the per-submodule test jobs), including the previously-failing `TestLoginPage_InlineScriptsCarryCSPNonce` | test3 log rc=0, all modules ok |
| A10 | Lint gate green (repo-wide golangci, 0 issues) | battery log `lint_rc=0` |
| A11 | integration_test CI-submodule replication green (hermetic build+vet+test with the 5 bumped families riding, incl. signing v4.4.0 — the gotcha-24 red class) | 1.1s suite, rc=0 |
| A12 | AGENTS gotcha 30 amended: behavior/test variant of the root-tag-gap class + the fact that `nix run .#test` IS hermetic | `a0f3e5a3` |
| A13 | Daemon-captured gate litter (`.go.work.check-workspace-build`, +95 lines committed mid-run by pma) untracked again | `f49fb815` |
| A14 | Push landed through the green pre-push gate (release-train strict + version-drift strict); CI watched to completion | `5c97ca1c..88c44ae0`, then `88c44ae0..f49fb815`; CI run 37818614080 = success |
| A15 | Foreign concurrent-session diff (prometheus transitive bump in two examples, 4 files) correctly NOT reverted — waited for tree-quiet, the other actor landed it as `5c97ca1c` | `wait-tree-quiet` rc=0 (90s stable window) |
| A16 | Receipt count nit fixed on sight (15 families total: 2 pre-landed + 13 in-session, not "13") | `136059ac` |

## b) PARTIALLY DONE

| # | Work | Done | Missing |
| - | ---- | ---- | ------- |
| B1 | Status-row hygiene in the 7 archived tables with the bare-`~~` + shifted-cells shape | Rows are gate-consistent (whole-row strike); PARTIAL failures gone | The underlying cell CONTENT is still shifted (row number in the Task column, Impact/Effort/Category misaligned) from the original aborted annotations — mechanical policy deliberately did not attempt semantic repair; a human/annotator pass over those 7 files is owed |
| B2 | `nix run .#check-modules` local replication | Every CI-wired stage green, incl. docs-freshness with the new CHANGELOG/AGENTS text | The local-only `branching-flow` ratchet fails: +59 findings vs the committed baseline — all attributable to the prior session's unpushed CSP-hardening Go code (`recommended_middleware.go` +14, `setup.go` +7, two new test files ~+225 lines). Deliberately NOT fixed: gotcha 26 requires adjudication via `docs/analysis/triage-decisions.md` before a re-pin, and blind-refactoring another session's fresh security code mid-train was the wrong move. CI self-skips this gate, so push/CI were unaffected |
| B3 | vendorHash.nix staleness (buildflow nix-checker flagged it after the go.sum churn) | Followed gotcha 31: did NOT edit the hash off the checker finding | Never ran the failing derivation build to learn whether it is genuinely stale or the known misattribution (gotcha 31 says the checker false-positives on repo-wide go.sum churn; the actual build is the only truth). No `nix build` output this session depends on it, and CI runs no nix builds — but the question is open |
| B4 | Pre-commit hook ergonomics during the train | Documented `--no-verify` fallback used with justification per sweep; content verified by bump-dep's hermetic gate each time | The hook itself still burns ~2 min per attempt on the documented noise class (samber-linter deadlines, test-compile 800ms kills, 60s budget exceeded at 1m55s) and then fails — 13 wasted hook runs happened; nothing upstream/in-config changed |

## c) NOT STARTED (surfaced this session, no work begun)

- Adjudication pass for the +59 branching-flow findings (B2).
- Semantic repair of the 7 malformed archived tables (B1).
- vendorHash.nix truth-check via the actual derivation build (B3).
- Anything about the go-cqrs-lite upstream side: 13+ family versions published within ~24h of the 2026-10-07 alignment — whether that burst was intentional and whether wave tags carry the gotcha-24 risk again was NOT investigated (out of scope per consumer-side session).
- TODO_LIST.md updates: none of this session's surfaced work has been written into TODO_LIST yet (repo convention: short-term actionable items belong there; this report is the capture point).

## d) TOTALLY FUCKED UP

Nothing shipped broken — final state is pushed and CI-green, and no foreign work was
damaged. The honest list of self-inflicted detours and near-misses:

| # | What | Cost | Lesson (now recorded where) |
| - | ---- | ---- | --------------------------- |
| D1 | The new CSP test failure was very nearly misdiagnosed as a timing flake: it passed in isolation AND with `-race` full-package — because those runs were workspace-mode. Two wasted hypotheses before checking flake.nix line 46 (`export GOWORK=off`) | ~15 min + one full extra battery run | The battery's hermeticity is the FIRST thing to check when local-vs-battery results disagree; now in AGENTS gotcha 30 |
| D2 | First sweep commit (kv) chained a NORMAL `git commit` — burned ~2 min on a pre-commit hook doomed to fail on the documented samber-linter/test-compile noise class, and lost the tree to the daemon's heuristic commit, needing an amend | ~3 min + a messy history beat | After the first documented-class hook failure, chain `--no-verify` + justification immediately (gotcha 8's fallback exists precisely for this) |
| D3 | No baseline gate pass before mutating the tree (go-ecosystem-upgrade skill Phase 1 explicitly demands it). The row-gate red and branching-flow red were both pre-existing but I discovered them only mid-verification and had to prove pre-existence via git archaeology (`24dfae16` pushed, `gh run list` already red) | ~10 min of forensics + a moment of genuine "did I break this?" risk | Baseline: run `check-modules` + `.#test` BEFORE the first sweep next train |
| D4 | `nix develop -c shellcheck` — shellcheck is not on the devShell PATH by that name; wasted call. treefmt (`nix run .#fmt -- <file>`) is the wired path | one round trip | Formatter/linter invocation always via the flake apps |
| D5 | The daemon captured the workspace-build gate's temp file as a +95-line addition at the exact moment I was pushing — my deletion landed only as a follow-up commit. The ROOT cause (gate writes a litter-able temp file at repo root) is still unfixed; pma can re-capture it any future run | one churn commit; recurring risk | Gate temp files belong in /tmp or .git/ — open task below |

## e) WHAT WE SHOULD IMPROVE

1. **Baseline-first discipline for trains**: run the full gate battery before the first
   mutation, so pre-existing reds are catalogued (and exempted) before they can be
   misattributed — this session's biggest avoidable time sink.
2. **Kill the commit-race pattern**: bump-dep should own its commit (rc-checked) with the
   proper message, instead of the daemon-vs-me race + amend dance that happened 3 times.
3. **Make the pre-commit hook honest about its noise**: the documented failure class
   (samber-linter ~90%, test-compile 800ms kills) fails every sweep commit and trains
   everyone to reach for `--no-verify`. Either scope those steps out of pre-commit mode,
   raise the 60s budget, or teach the hook to skip-and-warn on the known classes.
4. **Fixer/gate contract drift is a recurring class** (this is the second split brain:
   checker↔fixer here, anchoring cqrs-lint↔walk-mode before). The status-row pair
   duplicates `is_separator`/`classify_row` logic in two files; extract a shared module
   or add an equivalence meta-test so the next drift is a test failure, not a mystery.
5. **Hermeticity must be stated where it bites**: "workspace green ≠ battery green" cost
   this session a misdiagnosis. Gotcha 2 (false-green of root-run `./...`) now has its
   sibling recorded in gotcha 30, but both should cross-link from one "resolution modes"
   section.
6. **Root-tag-gap needs a mechanical pre-publish catch**: the release-train gate checks
   version publication, not symbol/behavior containment (gotcha 30, twice now). A
   `GOWORK=off go test ./...` scoped to modules whose go.mod changed in the train — run
   BEFORE `verify-tag --push` — would have caught the CSP gap before the tag existed
   (here the tag happened to be cut after the test was already red locally, but the
   sequencing was luck, not process).
7. **Train-lag is regenerating faster than it is cleared**: 33 findings re-accumulated
   within ~24h of the 2026-10-07 alignment. The reactive push-blocked flow works but
   burns an afternoon per cycle; a standing cadence or early-warning would flatten it.
8. **Report hygiene in-flight**: receipts written mid-train should be re-read at train
   end (the "13 families" nit shipped and needed a follow-up commit).

## f) UP TO 50 THINGS WE SHOULD GET DONE NEXT

**Branching-flow / analysis debt**

1. Adjudicate the +59 branching-flow findings on the CSP-hardening code: fix the findings or re-pin the baseline per `docs/analysis/README.md`; amend `docs/analysis/triage-decisions.md` (gotcha 26).
2. After adjudication, re-run `nix run .#check-branching-flow` to a local green and note the pin delta.
3. Extract the duplicated `is_separator`/`classify_row` logic shared by `check-status-rows.py` and `normalize-status-rows.py` into one module (or add an equivalence meta-self-test) — kill the drift class at the code level.
4. Add a fixture to the row-gate self-test for the bare-`~~`-first-cell + shifted-cells shape (the exact 40-row class) so both tools stay pinned to it.
5. File the fixer/gate table-detection contract upstream in the docs-health skill assets if the same pair exists there (the skill ships its own `check-rows.py` authoring aid).

**Root-tag-gap hardening (gotcha 30)**

6. Add a pre-publish step to the release checklist: `GOWORK=off go test ./...` for every module whose go.mod changed in the train, BEFORE `verify-tag --push`.
7. Grep the workspace for other tests asserting `Content-Security-Policy` shapes and confirm none still assume the pre-img-src policy.
8. Cross-link gotcha 2 and gotcha 30 into one "resolution modes" note (workspace vs hermetic, who exports GOWORK=off, what each gate actually runs).
9. Consider a `verify-tag.sh --check-consumers-hermetic` dry-run flag that surfaces symbol/behavior containment before the tag is cut.

**Daemon / hook ergonomics**

10. Teach `bump-dep.sh --commit` to rc-check the commit and print an honest result (gotcha 27 says it historically didn't; verify current behavior, fix if still silent).
11. Give bump-dep a `--message` template so sweeps commit with readable messages without the daemon race.
12. Teach pma (auto-commit daemon) to recognize bump-dep-shaped diffs (go.mod/go.sum-only, single family) and emit a family-named message instead of "N changed file(s)".
13. Scope `samber-linter` out of BuildFlow's pre-commit build mode (documented ~90% failure rate) or raise its per-module timeout; it failed every one of this session's 13 hook attempts.
14. Fix `test-compile`'s 800ms kill under pre-commit parallelism (raise the timeout or drop the step from pre-commit mode).
15. Raise or make honest the pre-commit 60s performance budget (actual run: 1m55s; "budget exceeded" on every commit is noise fatigue).
16. Add the NEW noise shapes observed this session (test-compile `transient:tool.timeout` at 800ms, go-humanize-linter cancels) to the gotcha-8 class list in `.buildflow.yml` comments / runbooks.
17. Consider a hook-side known-noise detector: if only documented-class steps failed, warn + pass instead of forcing `--no-verify`.
18. Make the workspace-build gate write `.go.work.check-workspace-build` under /tmp or `.git/` so pma can never capture it (D5 root cause).
19. Audit for other gate litter files tracked at repo root (`git ls-files | grep -iE 'tmp|workspace-build|fsprobe'`).
20. Document in AGENTS gotcha 4 the amend-daemon-commit flow as the DEFAULT (it worked cleanly 3 times this session; it is currently buried in gotcha 27 prose).

**Train process**

21. Decide a standing train-lag cadence: nightly `bump-dep` dry-run report (CI cron or local app) vs the reactive push-blocked flow; surface lag before it hits the pre-push gate.
22. Ask upstream (go-cqrs-lite) whether the ~24h 13-version burst ending in v4.17.0/v4.10.5 etc. was intentional, and whether wave tags re-carry the gotcha-24 CloneEvent/encoding risk class.
23. Next multi-family train: use the recipe's same-version prefix-sweep merging (samber-do-demo needed 10 single sweeps; a merged prefix sweep would have cut 9 commits).
24. Verify the signing v4.4.0 line actually carries the WithEncoding-preserving CloneEvent fix in its tagged source (gotcha 24's lesson was that wave tags can lie about content; integration_test passes hermetically, but a tag-level source check closes the loop).
25. Re-check the release-train gate's tag-cache TTL interplay: after cutting root v4.13.3 mid-session, the very next commit needed `--refresh-cache` (gotcha 27) — consider auto-refresh inside bump-dep when it just pushed a tag.
26. Rotate root CHANGELOG's `[Unreleased]` block: it still carries entries already shipped in v4.13.0–v4.13.3 — cut real version sections so pkg.go.dev-era release notes stop accreting stale content.
27. Decide whether the repo wants GitHub Releases per tag (release notes are currently CHANGELOG-only).
28. Revisit whether security-header behavior changes (CSP directives) should ride minor bumps (v4.14.0) instead of patch trains (v4.13.3) — repo precedent carries new API in patches, but loosening a security default is consumer-visible policy.
29. Add `docs/guides/release-playbook.md` §3a note: mid-train daemon absorption of sweep commits is EXPECTED; amend, don't re-commit (this session's working pattern, currently tribal knowledge).

**Docs/status hygiene**

30. Semantic repair pass over the 7 archived tables with shifted cells (B1) — restore row-number/Effort/Category alignment per the original reports' intent, or annotate the shift as deliberate.
31. Spot-audit the SUPERB-hardening archived report's 13 struck rows (normalized this session) against actual code state — the strikes were mechanical, not verified (annotation integrity, gotcha 19 spirit).
32. Harvest this report's open items into TODO_LIST.md per the docs-health doctrine (this session wrote none of them there).
33. Add the long-form narrative of this push-unblock episode to `docs/agents-notes.md` (per AGENTS.md: dated histories live there, this file is the snapshot).

**Verification debt**

34. vendorHash.nix truth-check (B3): run the flagged derivation build; re-derive only off a real `got:` hash; if it is the known gotcha-31 misattribution, note it and silence the class.
35. Run `nix run .#check-cqrs-lint` explicitly post-train (check-modules green implies it, but the module-walk + root-walk duality of gotchas 28/29 deserves one explicit confirmation after dependency churn).
36. Re-run `nix run .#erraudit-inventory --gate` after the family bumps (upstream error shapes can change findings; check-modules covered the self-test, the real inventory is one command).
37. Confirm `nix run .#coverage-gate` thresholds still hold after the bumps (15 modules gated; not run this session).
38. One `nix flake check` pass to confirm no flake-level regression from the go.sum churn (vendorHash class, B3's cousin).
39. Verify go.work.sum is still untracked and contains no session drift (`git status` says clean — record the check as a standing post-train step).

**Small code/tooling improvements observed in passing**

40. `check-status-rows.py`'s "deliberately-mixed tables" count could name the files (it reports a count + first example only).
41. bump-dep output prints "next: check-release-train --refresh-cache --strict-lag 0, then commit" — add the amend-if-daemon-raced hint (gotcha 27 wording) to that line.
42. `wait-tree-quiet` + `preflight-tree-check` should be one preflight command for trains (quiet-tree + surprise-check in a single invocation; this session used them separately).
43. The lint battery and test battery ran as one chained background command — consider a `nix run .#ci-parity` meta-app that runs lint+test+strict-train in sequence with one rc, for one-command pre-push replication (gotcha 27c's one-pass recipe, productized).
44. `gh run watch` on the new HEAD worked well — consider documenting it in the release playbook §4 as the post-push step.
45. The CHANGELOG "Fixed" entry for this battery is one paragraph doing three jobs (train + tidy + row gate) — future receipts: one change-class per entry.

**Watch items (no action without a trigger)**

46. The 4-file prometheus/client_golang v1.25.0 transitive bump from the concurrent session (`5c97ca1c`) — already pushed, verified green; only watch that it was deliberate on their side.
47. The `adminui`-class `// indirect` require pruning the concurrent session did (33-file commit `652426b5`) resolved several lag findings by REMOVAL rather than bump — confirm the release-train gate treats removal-vs-bump equivalently going forward (it did: 846→842 requires).
48. CI's ubuntu-label migration notice (runner images issue, Oct 19 2026) — the fleet's pinned actions may need a label review before that date.
49. buildflow `dependabot-auto-configure` INFO (28 modules > 20-entry cap) remains a documented accept — revisit only if the generator cap is lifted upstream.
50. The five concurrent Crush sessions on this tree — if this train had been cut while another session was mid-release (gotcha 4's worst case), the wave ordering could have interleaved. Consider a soft lock file or a `wait-tree-quiet` gate at train START, not just before push.

## g) QUESTIONS I CANNOT ANSWER MYSELF

1. **Branching-flow +59**: should the CSP-hardening middleware code be *fixed* to satisfy the design-smells analyzers, or do you want the baseline *re-pinned* as accepted with a triage note? (Owner taste on fresh security code; gotcha 26 routes this through you.)
2. **Versioning policy**: is a security-header behavior change (CSP gains `img-src 'self' data:`) acceptable on a PATCH train (root v4.13.3, matching repo precedent), or should such changes ride minor bumps going forward?
3. **Train cadence**: do you want a standing early-warning for train lag (e.g., a nightly bump-dep dry-run report), or is the current reactive push-blocked-then-align flow acceptable at ~one alignment afternoon per upstream burst?

---

*Snapshot ends. Living state: `FEATURES.md`, `TODO_LIST.md`, `AGENTS.md`, `README.md`.*
