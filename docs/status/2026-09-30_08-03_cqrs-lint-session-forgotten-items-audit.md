# cqrs-lint Session Complete — With Forgotten-Items Audit

**Session:** 2026-09-29 ~23:00 → 2026-09-30 08:03 CEST (three passes: easy-way triage → root-cause fixes → "forgot anything?" self-audit)
**Repos touched:** `cqrs-htmx` (this repo) + `go-cqrs-lite` (`cmd/cqrs-lint` — the linter's actual home)
**Report supersedes:** `2026-09-30_00-41_cqrs-lint-rebuild-triage-gate-green.md` (mid-session) and `2026-09-30_07-19_cqrs-lint-root-cause-fix-session.md` (close-of-fix) — both now swept into git; this one adds the self-audit's findings.

---

## Executive summary

The rebuilt cqrs-lint binary (upstream `3756eb4`) turned the gate's ROOT recursive-walk run red. Pass 1 made it green the easy way (suppressions + a C040 config disable); the owner's "easy way out" challenge triggered Pass 2, which fixed the real causes: the C040 phantom class was a **collector gap in the linter itself** (string-literal-only event-type resolution vs this repo's const/var-alias emission chains and `EventMetadata` catalog registrations) — fixed at source in `go-cqrs-lite/cmd/cqrs-lint` with 9 regression tests and verified 0-findings-with-rule-enabled on the full walk; the C009 panics became tested startup-time error propagation; C042 was verified against code before suppression; a deprecated `event.AggregateID()` call was fixed; and the severity-semantics record was corrected after an empirical disproof. Pass 3 (the "forgot anything?" audit) then found what Pass 2's own close-out missed: **the linter fix's home repo got code but not its documentation conventions, and the new ~150 lines were never vetted, linted, or race-tested by that repo's standards.** Those four gaps are this report's headline open work. Gate green throughout; both repos' trees fully swept by their daemons.

## a) FULLY DONE

1. **Gate failure diagnosed**: rebuilt binary re-attributed findings to different lines (moving-linter regression); baseline ROOT FAIL / 13 per-module passes captured.
2. **Stale C017 suppression fixed** — moved to the construction site the rebuilt binary attributes to (e2e/server `deadStore` line); stale warning gone.
3. **C040 fixed at the linter source** (`go-cqrs-lite/cmd/cqrs-lint`, committed by its daemon): pending-ref recording for unresolvable `event.New`/`NewEvent`/`catalog.Event` args; const-alias capture; NEW `scanVarAliasDecl` (cross-package `var` re-exports only — same-package vars excluded, FN-only tradeoff); NEW `ResolveEmittedEventTypeConsts` post-pass (memoized, cycle-guarded, depth-bounded chain expansion); NEW EventMetadata-style `Register({Type: …})` collection; wired at all 4 post-pass call sites.
4. **9 linter regression tests**; full `cmd/cqrs-lint` module suite BUILD+TEST RC=0.
5. **Real-repo verification with the fixed binary, C040 ENABLED**: 0 findings on the root walk + all 13 per-module runs, RC=0.
6. **Zero-delta proof**: worktree build at true pre-fix commit `ad1dcd46a` → identical residual warning sets (identity-model 16× E005 + 1× V006; usermgmt 41× V007 + 1× A016) — fix regresses nothing.
7. **Contaminated-baseline mistake caught and redone** (daemon had committed the fix before the stash; true pre-fix worktree used instead).
8. **C009 eliminated by design**: `newAssetHandler(fsys, name, contentType) (http.Handler, error)` reads at `New` with error propagation in both adminui and dashboardui; missing-asset unit tests (fstest.MapFS negative + real-embed positive); existing adminui test updated; both module suites green; both suppressions deleted.
9. **C042 verified against code then suppressed**: all three demo Save sites are first writes on freshly created streams (`Version(1)` events at expected `Version(0)`) — rule premise false; suppressed with the verified reason.
10. **`event.AggregateID()` deprecation confirmed in source** (`event/v3_compat_aliases.go`) and the e2e call fixed to `StreamID()`.
11. **Severity semantics corrected in the record** (AGENTS gotcha 13, CHANGELOG, both status reports): exit = ERROR findings only; `--strict` = load errors; `min_severity` = display floor; pinned by source read + RC=0-with-21-warnings empirics.
12. **Gate green through the handoff window**: `nix run .#check-cqrs-lint` 14/14 with the OLD system binary and C040 enabled in config; user's exact invocation exits 0.
13. **This repo's docs paid**: CHANGELOG entry (final truth), TODO_LIST handoff item (system rebuild → verify C040 silent, with residual-warning inventory), AGENTS.md quick-ref row + gotcha 13, ROADMAP OQ 25 (P009 codec question), `.cqrs-lint.json` fix story.
14. **Build/test battery**: workspace build RC=0; standalone e2e/server build RC=0; dashboardui + adminui suites RC=0; vet on touched packages RC=0; gofmt clean.
15. **Both status reports written and swept**; both repo status gates pass (`check-status-annotations.sh`, `check-status-rows.py` RC=0).
16. **The "forgot anything?" self-audit executed** — produced this report's four new open items (§c1–4) instead of letting them surface later.

## b) PARTIALLY DONE

1. **go-cqrs-lite documentation debt (NEW, from the self-audit)** — the linter fix exists there only as daemon `chore:` commits. No CHANGELOG entry, no AGENTS.md/TODO_LIST note in that repo. A significant collector change is invisible to its own conventions.
2. **go-cqrs-lite verification gaps (NEW)** — `cmd/cqrs-lint` got build + plain test suite (RC=0) but **no `-race` run, no `go vet`, and no golangci-lint** by that repo's standards; ~150 new lines are unlinted upstream.
3. **RULES.md drift upstream (NEW)** — the C040 entry describes catalog coverage via `catalog.Event` only; metadata-style `Register(EventMetadata{Type: …})` now counts too.
4. **Playwright e2e specs never ran** — Go suites, builds, vet pass; browser-truth specs over the touched UI modules/e2e server unexecuted (changes were one deprecated-method rename + comments + startup-time asset reads; risk near-zero, verification absent).
5. **Residual warning classes untriaged** (verified pre-existing via the prefix-worktree control, non-gating): identity-model 16× E005 + 1× V006; usermgmt 41× V007 + 1× A016; dashboardui/datastar V006.
6. **Manual-run cleanliness pending the system rebuild** — PATH's cqrs-lint still prints 21 stale C040 warnings (non-failing) on manual root walks; fixed binary staged at `/tmp/cqrs-lint-fixed`.
7. **A013 in the 7 example modules** — "accepted" without evaluating value-embedding against `BasicCommand`'s receiver shapes.

## c) NOT STARTED (observed, routed or deferred)

1. **go-cqrs-lite CHANGELOG + AGENTS/TODO entries for the linter fix** (owner approval pending — cross-repo docs work).
2. **`cmd/cqrs-lint` vet + golangci-lint + `-race` test run** by that repo's own gates.
3. **RULES.md C040 touch** (metadata-style catalog coverage).
4. **System cqrs-lint binary rebuild + post-rebuild verification** (TODO_LIST item, exact command recorded).
5. **E005/V007/A016/V006 residual triage** — E005 half is likely another linter fail-open fix; V007 half maps to ADR-0051/OQ 11.
6. **P009 codec decision** — routed to ROADMAP OQ 25; untouched.
7. **`--fail-on-stale-suppressions` adoption** — still unwired.
8. **`cqrs-lint rules` diff across the binary update** — new rules (V007/A016) surfaced as surprise noise.
9. **Pre-existing, re-observed:** `check-cqrs-lint` CI wiring (blocked on Go-installable distribution); manual-run recipe block for AGENTS.md; per-module config-inheritance verification; JSONC-config fixture gate; `exclude` key semantics investigation.

## d) TOTALLY FUCKED UP / near-misses (the honest ledger)

Nothing shipped broken. The failures, descending:

1. **The first C040 root cause was WRONG and green fixtures hid it** — pattern-matched `identity-model/constants.go` (typed consts), never read `usermgmt/es_constants.go` (the `var` aliases the emissions actually use). Unit tests passed; the real walk stayed 21/21 red. Instrumented debug runs against the real tree found it. A root cause derived from the first file that fits is a guess.
2. **Pass 1 was the easy way and the owner said so.** Suppressing a rule whose phantom-ness was only spot-verified, config-disabling it, and refactoring-averse C009 suppressions across two duplicated modules. The correction arc roughly doubled the session.
3. **The cross-repo close-out gap (found by the audit, not by me)** — work in `go-cqrs-lite` was declared done when its tests passed, without its docs conventions (CHANGELOG/AGENTS/TODO) or its lint/vet/-race gates. Two repos touched; only one repo's definition of "done" was applied.
4. **An unverified claim shipped mid-session** — "strict fails at WARNING+" entered three documents from one ambiguous observation; disproved later by RC=0-with-21-warnings; corrected everywhere.
5. **The contaminated baseline** — `git stash push` silently stashed nothing (daemon had committed first); first control build measured the fix against itself; caught by git-state verification; redone via pre-fix worktree.
6. **Process slips, recovered same-session:** 3-of-6 edit batch failures (bash-read files), one stale AGENTS old_string, an rg `-r` flag misuse mangling a search, two background jobs overlapping tree edits (valid by luck), `/tmp` logs vanishing under concurrent sessions (switched to variable capture), a `write` that would have clobbered an existing test file (tool guard saved it).

## e) WHAT WE SHOULD IMPROVE

1. **Two-repo rule: the touched repo's own "done" is the bar.** Code in a second repo implies that repo's CHANGELOG/AGENTS/TODO + lint/vet/race gates before verification is claimed. This is the audit's direct product and the session's newest standing rule.
2. **Derive root causes from the full failing surface** — read every file the finding touches before naming the mechanism (d1).
3. **Verify fixes on the real surface, not just fixtures** — unit-green + real-red is a fixture-shape gap.
4. **Evidence-gate durable claims** — both corrections (severity semantics, first root cause) trace to claims that outran their evidence.
5. **Suppress only with code-read evidence; prefer design fixes for design smells** — the C009 refactor is what "unreachable panic" should have gotten on day one.
6. **Make suppression staleness loud** (`--fail-on-stale-suppressions` in the gate).
7. **Diff rule sets across linter updates** (`cqrs-lint rules` old vs new) so new warning classes arrive as triage, not noise.
8. **Treat daemon-committed trees as the norm** — check git state before any stash/isolation move.
9. **Shell variables over /tmp files for long outputs** under concurrent sessions.
10. **Difficult feedback was the session's best input** — the owner's challenge named a pattern; reversing mid-session cost less than shipping the easy way. Keep that reversible.

## f) Up to 50 things to get done next (brainstorm, rough impact order; ROADMAP fuel, not commitments)

**NEW — from the forgotten-items audit (top priority):**
1. Write the go-cqrs-lite CHANGELOG entry for the cqrs-lint collector fix (var-alias + EventMetadata collection, 9 tests, verified 0 findings rule-enabled).
2. Add the go-cqrs-lite AGENTS.md/TODO_LIST note for the same (its own session-context file).
3. Run `go vet` + golangci-lint on `cmd/cqrs-lint` and fix anything the repo's own linters flag in the new code.
4. Re-run the `cmd/cqrs-lint` test suite with `-race` (repo standard).
5. Touch RULES.md C040: metadata-style `Register(EventMetadata{Type: …})` now counts as catalog coverage.
6. Confirm which commit range carries the fix upstream and whether go-cqrs-lite's own CI gates need the fix reflected anywhere (rule-count docs, doctor goldens).

**Handoffs from the fix (from prior list, re-ranked):**
7. Rebuild the system cqrs-lint binary; verify `cqrs-lint --strict --verbose .` shows zero C040 on a root walk (TODO_LIST item).
8. Triage identity-model 16× E005 — likely missing cross-module fail-open for pure-domain command modules; candidate second linter fix.
9. Triage usermgmt 41× V007 against ADR-0051/OQ 11 (batch-migrate `stack.Materialize` vs suppress-with-reason).
10. Decide A016 + 4× V006 (suppress/fix/accept per the documented per-module-train policy).
11. Run the e2e Playwright suite over this session's UI-module + e2e-server changes.
12. Diff `cqrs-lint rules` between binary `3756eb4` and current source; enumerate every added/changed rule.
13. Wire `--fail-on-stale-suppressions` into the flake gate (after interaction-check with `--strict`).
14. Sweep ALL inline `//cqrs-lint:ignore` comments: assert each fires; delete stale ones.
15. Add the manual-run command block (env + invocation + expected exit) to AGENTS.md gotcha 13.
16. Decide A013 for examples (value-embed `BasicCommand` in demos if receiver shapes allow).
17. Consider a flake-pinned cqrs-lint build from `~/projects/go-cqrs-lite` (gate provenance independent of system rebuilds).
18. Re-verify the corrected severity semantics on the next binary update.
19. agents-notes narrative: the three-pass arc — easy way, wrong root cause hidden by fixtures, var-alias discovery, contaminated baseline, prefix-worktree proof, audit-found docs debt.
20. Upstream-doc ask: document strict/min-severity/exit-code semantics in the linter README/RULES.md.
21. Evaluate `cqrs-lint scorecard` + `--group-by module` for triage ergonomics.

**P009/codec track:**
22. Decide ROADMAP OQ 25: CBOR for the three `[]byte`-carrying payloads vs non-binary field shapes.
23. Measure the actual payload-size delta before deciding (35% is the linter's figure, not measured).
24. If CBOR: plan the mixed-stream rollout (`event.WithCodec` per event type; reads already self-describing).

**Gate/CI hardening:**
25. Get cqrs-lint a Go-installable distribution (pre-existing blocker) → wire `check-cqrs-lint` into CI.
26. Consider cqrs-lint in the pre-push hook (post-distribution; watch duration).
27. Verify config inheritance per module empirically (`cqrs-lint doctor` per gate module).
28. Add a fixture self-test that JSONC comments in `.cqrs-lint.json` parse.
29. Investigate the `exclude` key's substring + inheritance semantics; if safe, exclude e2e/examples from root walks instead of per-finding suppressions.
30. Consider per-module `.cqrs-lint.json` presets (identity-model/usermgmt inherit root today).
31. CI lane running cqrs-lint on e2e/examples at INFO threshold.
32. Record the severity/exit-code semantics table in the linter's docs.

**Example/demo hygiene:**
33. `system-demo` `waitForView`: check for a channel-based drain/wait upstream (P006's actual suggestion).
34. Audit every `store.Save(..., event.Version(0))` repo-wide for the new-stream contract (C042-class sweep beyond the demo).
35. `dashboard-demo`: consider a second seed pass teaching load-then-save optimistic concurrency.

**Docs/knowledge:**
36. Cross-link CHANGELOG ↔ both status reports ↔ TODO handoff item for the next docs-health sweep.
37. Split cqrs-lint knowledge into `docs/guides/cqrs-lint.md` if gotcha 13 grows further.
38. Register today's three status reports for the next archive sweep.
39. Check `docs/DOMAIN_LANGUAGE.md` documents the var-alias re-export pattern (gotcha 15) as the cross-module event-vocabulary shape.
40. Annotate the two earlier reports as superseded-by this one (inline, per convention).
41. Verify the daemon swept the 00-41 report's final-truth rewrite (it was still modified at 07:19; clean by 08:03 — confirm content in the sweep commit).

**Linter upstream (beyond item 2):**
42. `--check-suppressions` mode: fail when an inline suppression matches nothing (mechanizes item 14).
43. Stable finding attribution across rebuilds (the C017 stale-suppression class).
44. Document the collector's resolution chain (typed consts → const aliases → var re-exports → string conversions) in the analyzer README.
45. Property/fuzz test the alias-chain resolver (cycles, depth, shadowing).
46. Benchmark the post-pass on a large multi-module tree.

**Verification-debt sweep:**
47. `nix run .#test` once over the fleet (only targeted module suites ran).
48. `nix run .#coverage-gate` drift check.
49. treefmt/prettier pass over the session's markdown (CHANGELOG/TODO/AGENTS/ROADMAP/three status reports).
50. Review whether adminui's two assets should share one ETag or carry per-asset ETags (current behavior preserved; make it a conscious decision).

## g) Questions I can NOT figure out myself

1. **Cross-repo docs approval:** may I write the go-cqrs-lite CHANGELOG/AGENTS/TODO entries for the linter fix (items 1–2), or do you keep that repo's docs under a different cadence/ownership?
2. **When does the system profile rebuild pick up the go-cqrs-lite fix — and do you want the gate to pin its own linter build** (flake input) so provenance stops depending on rebuild timing?
3. **A013 + advisory-noise policy:** value-embed `BasicCommand` in examples or suppress; and for E005×16/V007×41/A016/V006 — suppress-with-reason, fix at source (E005), batch-migrate (V007), or leave visible?

---

## Verification evidence appendix

| Claim | Evidence |
| --- | --- |
| C040 root cause | Pre-fix collectors used `StringLit` on arg0; `usermgmt/es_constants.go` var aliases; `EventMetadata{Type: string(const)}` registrations |
| Fix verified, rule enabled | Fixed binary root walk C040=0/WARN=0/ERR=0 RC=0; all 13 modules RC=0, C040=0 |
| Zero-delta | Pre-fix worktree `ad1dcd46a`: identical E005=16/V007=41/A016=1/V006 sets |
| Linter suite | BUILD RC=0, TEST RC=0 (no -race — open item 4); 9 new tests |
| C009 refactor | Both UI module suites RC=0; fstest.MapFS tests; no `assetHandler` refs remain |
| C042 verified | Demo seeds: fresh `id.NewStreamID()`, `Version(1)` at expected `Version(0)` |
| AggregateID deprecated | go-cqrs-lite `event/v3_compat_aliases.go:17` |
| Builds | Workspace + e2e standalone + vet RC=0 |
| Gate (handoff window) | `nix run .#check-cqrs-lint` all 14 pass (old system binary, C040 enabled) |
| User invocation | `GOTOOLCHAIN=auto cqrs-lint` RC=0 (21 stale C040 warnings, non-failing) |
| Severity semantics | `run.go` source + RC=0-with-21-warnings empirics |
| Status gates | Gate 1 RC=0, Gate 2 RC=0 |
| Sweep state | cqrs-htmx tree clean at 08:03 (both reports committed by daemon); go-cqrs-lite tree clean |
| Known-open | go-cqrs-lite docs + lint/vet/-race (§b1–3); Playwright; residual warning triage; system rebuild |
