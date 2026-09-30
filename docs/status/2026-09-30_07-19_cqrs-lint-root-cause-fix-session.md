# cqrs-lint Session — Easy-Way Triage Corrected Into Root-Cause Fixes

**Session:** 2026-09-29 ~23:00 → 2026-09-30 02:00 CEST (report finalized 07:19)
**Repos touched:** `cqrs-htmx` (this repo) + `go-cqrs-lite` (`cmd/cqrs-lint` — the linter's actual home, discovered mid-session)
**Arc:** Two passes. Pass 1 triaged the failing cqrs-lint gate with suppressions and a config disable. The owner challenged it ("you took the easy way out"), and Pass 2 reversed the paper-overs and fixed the actual root causes — including a fix INSIDE the linter itself.

---

## Executive summary

A rebuilt cqrs-lint binary (upstream `3756eb4`, 2026-09-29 05:53) turned the gate's ROOT recursive-walk run red: one ERROR (a C017 whose line the rebuild silently re-attributed, staling its inline suppression), 21 C040 phantom warnings, C009/A023/F001/A003/D008/P006 on deliberate patterns. Pass 1 made everything green with suppressions + a C040 config disable. Pass 2 (after the owner's challenge) did it properly:

- **C040 root-caused to the linter's collectors and fixed at source**: `event.New`/`NewEvent`/`catalog.Event` args were resolved from string literals only — this repo's emissions go through alias chains (identity-model consts → usermgmt **`var`** re-exports, the "Go has no const aliases" pattern) and its catalog declarations through `EventCatalog.Register(EventMetadata{Type: string(const)})`, both invisible. The collector now follows const AND var-alias chains and reads metadata-style registrations. 9 regression tests; full linter suite green; **0 C040 findings with the rule ENABLED** on the root walk + all 13 modules; a pre-fix-commit worktree build proves zero new warnings anywhere.
- **C009 panics eliminated by design** (startup-time error propagation + tested error paths, both UI modules).
- **C042 verified against code** (fresh streams — version 0 is the correct new-stream contract) and suppressed with that evidence.
- **Deprecated `event.AggregateID()` call fixed** to `StreamID()` (deprecation confirmed in go-cqrs-lite source).
- **Severity semantics corrected in the record** (exit = ERROR only; `--strict` = load errors; `min_severity` = display floor — the earlier "strict fails at WARNING+" claim was wrong).
- Gate green throughout; both status gates pass; docs (CHANGELOG/AGENTS/TODO/ROADMAP/config) all carry the final truth.

## a) FULLY DONE

1. **Gate failure diagnosed** — rebuilt binary re-attributed findings to different lines (moving-linter regression, not code regression); baseline captured: ROOT run FAIL, 13/14 per-module runs pass.
2. **Stale C017 suppression fixed** — moved to the construction site the rebuilt binary attributes to (e2e/server main.go deadStore line); stale warning eliminated.
3. **C040 root cause found and fixed in the linter** (`go-cqrs-lite/cmd/cqrs-lint`): pending-ref recording for unresolvable emission/catalog args; const-decl alias capture; NEW var-alias capture (`scanVarAliasDecl`, cross-package selectors only — same-package vars stay unrecorded, FN-only); NEW `ResolveEmittedEventTypeConsts` post-pass (memoized, cycle-guarded, depth-bounded chain expansion); NEW EventMetadata-style `Register({Type: ...})` collection (literal / const ref / `string(const)`); wired at all 4 post-pass call sites.
4. **9 linter regression tests** (const ident, alias chain, cross-file selector alias, catalog const, unresolvable negative, alias cycle, var alias, same-package-var negative, EventMetadata Register) — full `cmd/cqrs-lint` module suite BUILD+TEST RC=0.
5. **Real-repo verification**: fixed binary, C040 ENABLED → 0 findings, root walk + all 13 modules, RC=0.
6. **Zero-delta proof**: worktree build at pre-fix commit `ad1dcd46a` → identical residual warning sets (identity-model 16× E005 + 1× V006; usermgmt 41× V007 + 1× A016; dashboardui/datastar V006) — the fix regresses nothing.
7. **Contaminated-baseline mistake caught and redone** — first "pre-fix" build included the fix (daemon had already committed it); caught by verifying stash/worktree state; redone via true pre-fix worktree.
8. **C009 fixed properly**: `newAssetHandler(fsys, name, contentType) (http.Handler, error)` reads embedded assets once at `New`; error propagates from `dashboardui.New`/`adminui.New`; routes use stored handlers; missing-asset unit tests (fstest.MapFS negative + real-embed positive); one existing adminui test updated; both module suites green; both C009 suppressions deleted.
9. **C042 verified then suppressed**: all three demo Save sites are first writes on freshly created streams — rule premise false there; suppressed with the verified reason (demo's established style).
10. **`AggregateID` deprecation verified in source** (`event/v3_compat_aliases.go`: "Deprecated: use id.StreamID") and the e2e call fixed to `StreamID()`.
11. **Severity record corrected everywhere** (AGENTS gotcha 13, CHANGELOG, status reports): exit code fires on ERROR findings only; `--strict` hard-fails on package LOAD errors; preset `min_severity` is a display floor. Pinned by source read + an empirical RC=0-with-21-warnings run.
12. **Gate green through the handoff window**: `nix run .#check-cqrs-lint` 14/14 with the OLD system binary and C040 re-enabled in config (phantom warnings are non-failing); user's exact invocation exits 0.
13. **Docs paid in full**: CHANGELOG entry rewritten to final truth; TODO_LIST item replaced with the bounded handoff (system rebuild → verify C040 silent) including the residual-warning inventory; AGENTS.md quick-ref row + gotcha 13 carry the corrected semantics and the linter-fix story; ROADMAP OQ 25 opened (P009 []byte/JSON codec decision); `.cqrs-lint.json` documents the fix and the handoff in place of the never-needed exemption.
14. **Build/test battery**: workspace `go build ./...` RC=0; standalone e2e/server build RC=0 (root workspace patterns don't reach `./e2e/...`); dashboardui + adminui suites RC=0; go vet on touched packages RC=0; gofmt clean.
15. **Status-report hygiene**: both repo gates (`check-status-annotations.sh`, `check-status-rows.py`) pass.

## b) PARTIALLY DONE

1. **Playwright e2e specs never ran.** Go suites, builds, and vet all pass, but the browser-truth specs over the touched UI modules and the e2e server were not executed. The e2e/server changes were one deprecated-method rename plus comments; the UI changes are startup-time only — risk near-zero, verification absent.
2. **Residual warning classes untriaged** (verified pre-existing, non-gating): identity-model 16× E005 (wants handlers for pure-domain command types — smells like a missing cross-module fail-open, candidate linter improvement), usermgmt 41× V007 (`stack.Materialize` v5 advisory — maps to ROADMAP OQ 11/ADR-0051), 1× A016 (idempotency middleware advisory), 4× V006 (the documented per-module-train lockstep false positive). Suppress/fix/accept undecided.
3. **Manual-run cleanliness pending the system rebuild** — PATH's cqrs-lint is still the pre-fix binary; manual root walks print 21 stale C040 warnings (non-failing) until it rebuilds from the fixed go-cqrs-lite state. Fixed binary staged at `/tmp/cqrs-lint-fixed`.
4. **A013 in the 7 example modules** — "accepted" for the frozen library API, but the examples-as-dogfooders option (value-embedding `BasicCommand`) was never evaluated against the actual receiver shapes.
5. **Info-class dispositions** — A013×24/P009×3/P008/D013 remain visible, not suppressed (deliberate: gate unaffected, signal preserved); C042 moved from "accepted on assumption" to "suppressed with verified evidence".

## c) NOT STARTED (observed, routed or deferred)

1. **System cqrs-lint binary rebuild + verification** — TODO_LIST item with the exact verification command; owner-side action.
2. **E005/V007/A016/V006 triage** — inventoried in the TODO item; nothing started. The E005 half is likely another linter fix (fail-open for pure-domain modules), the V007 half maps to ADR-0051's existing migration posture.
3. **P009 codec decision** — routed to ROADMAP OQ 25 (CBOR vs non-binary field shapes for three `[]byte`-carrying payloads); the decision itself untouched.
4. **`--fail-on-stale-suppressions` adoption** — the mechanical answer to the C017 class; still unwired anywhere.
5. **`cqrs-lint rules` diff across the binary update** — new rules (V007/A016 etc.) were discovered as surprise noise, not triaged deliberately.
6. **Pre-existing, re-observed:** `check-cqrs-lint` CI wiring (blocked on a Go-installable distribution); copy-paste manual-run recipe for AGENTS.md; config-inheritance verification per module; JSONC-config fixture gate; `exclude` key semantics investigation.

## d) TOTALLY FUCKED UP / near-misses (the honest ledger)

Nothing shipped broken. The session's real failures, descending:

1. **The first C040 fix was WRONG and green unit tests hid it.** The initial collector change (const decls only) passed every new test while leaving 21/21 phantoms on the real repo — the initial root-cause pass pattern-matched identity-model's `constants.go` (typed consts) and never read `es_constants.go`, where the emission-facing aliases live as `var`. Only instrumented debug runs against the real tree surfaced it. This is the session's most important failure: a root cause derived from the first file that fit, not from every file in the failing surface.
2. **The owner was right the first time.** Pass 1 suppressed a rule whose phantom-ness I had verified only indirectly (spot-checks), config-disabled it, and called the triage done. The C009 "unreachable panic" suppressions papered over a duplicated, panic-based design in two modules. Both were easy-way outcomes; both reversed at real cost (the correction arc roughly doubled the session's length).
3. **An unverified claim shipped mid-session** — "strict fails at WARNING+" entered AGENTS/CHANGELOG/the first status report from one ambiguous observation and survived until a controlled run disproved it. Two correction cycles across three documents.
4. **The contaminated baseline** — `git stash push` silently stashed nothing (the daemon had committed the fix first), so the first control build measured the fix against itself. Caught by post-stash `git status` verification; redone with a worktree at the true pre-fix commit.
5. **Process slips, recovered same-session:** 3-of-6 edit failures (bash-read files aren't edit-authorized), one stale old_string on the AGENTS row, an rg `-r` flag misuse that mangled a search, two background jobs overlapping tree edits (baselines valid by luck), `/tmp` log files vanishing under concurrent sessions (switched to shell-variable capture), and a failed `write` that would have clobbered an existing `dashboardui/assets_test.go` (the tool's read-check saved an existing test file from deletion — the guard worked as designed).

## e) WHAT WE SHOULD IMPROVE

1. **Derive root causes from the full failing surface.** Read every file the finding touches before naming the mechanism; a hypothesis that fits one file is a hypothesis, not a cause. (d1's lesson, now the session's core one.)
2. **Verify fixes on the real surface, not just fixtures.** Unit-green + real-red is a fixture-shape gap; the first verification after any collector/linter change must be the actual repo walk.
3. **Suppress only with code-read evidence; prefer design fixes for design smells.** "Unreachable" panics in duplicated library code deserved the error-propagation refactor, not comments. The two-pass structure of this session is the cost of getting that ordering wrong.
4. **Make suppression staleness loud** — `--fail-on-stale-suppressions` in the gate; today's stale C017 was found by a human, not a machine.
5. **Treat daemon-committed trees as the norm**: check `git status`/`git log` before any stash/isolation move.
6. **Capture long tool output in shell variables, not /tmp files**, under concurrent sessions.
7. **Claims about tool semantics require source-level or controlled evidence** before entering durable docs — both corrections this session trace to skipping that.
8. **Diff rule sets across linter updates** so new warning classes arrive as triage items, not surprise noise.
9. **Difficult-feedback handling worked and should repeat**: the owner's challenge was actionable because it named a pattern, not a detail; reversing course mid-session cost less than shipping the easy way.

## f) Up to 50 things to get done next (brainstorm, rough impact order; ROADMAP fuel, not commitments)

**Direct handoffs from this session:**

1. Rebuild the system cqrs-lint binary; verify `cqrs-lint --strict --verbose .` shows zero C040 on a root walk (TODO_LIST item, exact command recorded).
2. Triage identity-model 16× E005 — likely a missing fail-open for pure-domain command modules; candidate fix in `go-cqrs-lite/cmd/cqrs-lint` (same collector area as the C040 fix).
3. Triage usermgmt 41× V007 against ADR-0051/OQ 11: batch-migrate `stack.Materialize` or suppress-with-reason per call site.
4. Decide A016 (idempotency advisory) and the 4× V006 (documented lockstep FP) — suppress, fix, or accept.
5. Run the e2e Playwright suite once over this session's UI-module and e2e-server changes.
6. Diff `cqrs-lint rules` output: binary `3756eb4` vs current source — enumerate every added/changed rule deliberately.
7. Wire `--fail-on-stale-suppressions` into the flake gate (after checking interaction with `--strict` and the V006 exemption policy).
8. Sweep ALL inline `//cqrs-lint:ignore` comments repo-wide: assert each still fires under the current source; delete the stale ones.
9. Add the copy-paste manual-run command block (env + invocation + expected exit) to AGENTS.md gotcha 13.
10. Decide A013 for examples: value-embed `BasicCommand` in the 7 demo commands if receiver shapes allow (dogfood the rule), else suppress in examples only.
11. Consider a flake-pinned cqrs-lint build from `~/projects/go-cqrs-lite` so gate provenance stops depending on system-profile rebuild timing.
12. Re-verify the corrected severity semantics on the next binary update (they were wrong once already).
13. Write the agents-notes narrative: the two-pass correction arc — wrong first root cause, green fixtures vs red real walk, var-alias discovery, contaminated baseline, prefix-worktree proof.
14. Upstream-doc ask (go-cqrs-lint): document strict/min-severity/exit-code semantics — this session read `run.go` to learn them.
15. Evaluate `cqrs-lint scorecard` + `--group-by module` for future triage ergonomics.
16. Update the 00-41 status report's commit pointer once the daemon sweeps it (it was still modified-uncommitted at session close).

**P009/codec track:**
17. Decide ROADMAP OQ 25: CBOR for `CredentialRemoved`/`TOTPEnabled`/`BotRegistered` payloads vs non-binary field shapes; if CBOR, plan the mixed-stream rollout (`event.WithCodec` per event type).
18. Measure the real payload-size delta on the three event types before deciding (the 35% figure is the linter's, not measured here).

**Gate/CI hardening:**
19. Get cqrs-lint a Go-installable distribution (pre-existing blocker) → wire `check-cqrs-lint` into CI (TODO `[~]` item).
20. Consider cqrs-lint in the pre-push hook next to release-train + version-drift (post-distribution; watch duration).
21. Verify config inheritance per module empirically (`cqrs-lint doctor` per gate module) — AGENTS asserts it from docs.
22. Add a fixture self-test that JSONC comments in `.cqrs-lint.json` parse (today's config relied on it working).
23. Investigate the `exclude` config key's substring + inheritance semantics (skipped as risky this session); if safe, excluding e2e/examples from root walks may beat per-finding suppressions.
24. Consider per-module `.cqrs-lint.json` presets (identity-model/usermgmt currently inherit root).
25. Add a CI lane running cqrs-lint on e2e/examples at INFO threshold — demo rot visible without failing the library gate.
26. Record the strict/min-severity/exit-code semantics table in the linter's README/RULES.md.

**Example/demo hygiene:**
27. `system-demo` `waitForView`: check whether go-cqrs-lite exposes a channel-based drain/wait replacing the poll (P006's actual suggestion).
28. Audit every `store.Save(..., event.Version(0))` call site repo-wide for the new-stream contract (C042-class sweep beyond the demo).
29. `dashboard-demo`: consider teaching real load-then-save optimistic concurrency in a second seed pass (demo quality, not a bug).

**Docs/knowledge:**
30. Cross-link CHANGELOG entry ↔ this report ↔ the 00-41 report ↔ the TODO handoff item so the next docs-health sweep routes cleanly.
31. Split cqrs-lint knowledge into `docs/guides/cqrs-lint.md` if gotcha 13 grows further (distilled rules stay in AGENTS; recipes + narratives move out).
32. Register both of today's status reports for the next archive sweep per `docs/status/README.md`.
33. Record the var-alias re-export pattern (gotcha 15) as the canonical cross-module event-vocabulary shape in `docs/DOMAIN_LANGUAGE.md` if it isn't already.
34. Annotate the 00-41 report as superseded-by this one (inline note per the annotation convention) once verified.

**Linter upstream (go-cqrs-lite/cmd/cqrs-lint):**
35. Land the E005 cross-module fail-open improvement (if triage item 2 confirms).
36. Consider a `--check-suppressions` mode: fail when an inline suppression no longer matches any finding (mechanizes item 8).
37. Stable finding attribution across rebuilds (the C017 stale-suppression class) — canonical positions or a suppression-migration hint.
38. Document the collector's resolution chain (typed consts → const aliases → var re-exports → conversions) in the analyzer's README.
39. Property/fuzz test the alias-chain resolver (cycles, depth, shadowing) — it now walks user-authored const graphs.
40. Benchmark the post-pass on a large multi-module tree (the walk analyzed 256 files; cost unknown but bounded).

**Verification-debt sweep:**
41. `nix run .#test` once over the fleet (only targeted module suites ran this session).
42. `nix run .#coverage-gate` (untouched this session; cheap drift check).
43. treefmt/prettier pass over the session's markdown (CHANGELOG/TODO/AGENTS/ROADMAP/two status reports).
44. Re-run `GOWORK=off go vet` per touched module (gotcha 2's hermetic-check habit).
45. Verify the daemon swept every session edit (spot-audit the last 8 heuristic commits against the intended file set).
46. Confirm `/tmp/cqrs-lint-fixed` is disposable (rebuildable from go-cqrs-lite HEAD) before any /tmp cleanup loses it.
47. Re-run both status gates after the daemon sweeps the two report files.
48. Check whether e2e specs assert the DLQ `StreamID` values that the `AggregateID` → `StreamID` rename touches (values identical per compat tests; assertion text may reference the old method name in comments).
49. Review whether adminui's `admin.js`/`admin-tw.css` dual resolution at `New` should share one ETag or per-asset ETags (current behavior preserved; worth a conscious decision).
50. Close the loop on the owner's challenge: a short retro note in agents-notes on when suppression is the right tool (deliberate patterns, evidence-cited) vs when it is the easy way (unverified premises, design smells).

## g) Questions I can NOT figure out myself

1. **When will the system profile rebuild so `cqrs-lint` on PATH picks up the go-cqrs-lite fix — and do you want the gate to pin its own linter build instead** (flake input on `~/projects/go-cqrs-lite`), so gate provenance never depends on rebuild timing again?
2. **A013 policy:** should the 7 example modules value-embed `BasicCommand` (dogfooding the rule, needs a receiver-shape check) or is pointer-embedding canonical repo-wide and worth suppressing in examples only?
3. **Advisory-noise appetite:** E005×16 / V007×41 / A016×1 / V006×4 are non-gating warnings under the newer linter source. Suppress-with-reason for clean verbose runs, fix at the linter source (E005), batch-migrate (V007), or leave visible as standing nudges? (V007's answer interacts with ADR-0051/OQ 11.)

---

## Verification evidence appendix

| Claim                                                  | Evidence                                                                                                                                                                                                             |
| ------------------------------------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| C040 root cause: literal-only collectors + var aliases | Pre-fix `scanner_calls.go` used `StringLit(call.Args[0])`; `usermgmt/es_constants.go` declares emission aliases as `var`; catalog registrations use `EventMetadata{Type: string(const)}`                             |
| Fix works, rule enabled                                | Fixed binary root walk: C040=0 C038=0 WARN=0 ERR=0 RC=0; all 13 per-module runs RC=0, C040=0                                                                                                                         |
| Zero new warnings                                      | Worktree at pre-fix commit `ad1dcd46a`: identical E005=16 (identity-model), V007=41 + A016=1 (usermgmt)                                                                                                              |
| Linter suite                                           | `cmd/cqrs-lint` BUILD RC=0, TEST RC=0; 9 new scanner regression tests                                                                                                                                                |
| C009 refactor                                          | dashboardui + adminui suites RC=0; fstest.MapFS missing-asset tests; no `assetHandler` references remain                                                                                                             |
| C042 verified                                          | `examples/dashboard-demo`: `aggID := id.NewStreamID()` per item; `Version(1)` event at expected `Version(0)`                                                                                                         |
| AggregateID deprecated                                 | go-cqrs-lite `event/v3_compat_aliases.go:17`                                                                                                                                                                         |
| Builds                                                 | Workspace build RC=0; e2e/server standalone build RC=0; vet RC=0                                                                                                                                                     |
| Gate (handoff window)                                  | `nix run .#check-cqrs-lint` (old system binary, C040 enabled): all 14 pass                                                                                                                                           |
| User invocation                                        | `GOTOOLCHAIN=auto cqrs-lint`: RC=0 (21 stale C040 warnings printed, non-failing, until rebuild)                                                                                                                      |
| Severity semantics                                     | `run.go`: exit on ERROR; `isStrictMode` gates LoadErrors; `resolveMinSeverity` = display floor; empirical RC=0 with 21 warnings                                                                                      |
| Status gates                                           | Gate 1 RC=0, Gate 2 RC=0 (re-run after both reports)                                                                                                                                                                 |
| Commits                                                | cqrs-htmx daemon sweeps (`436e821b`, `fef0da50`, `0b9898a9`, `354f70f2`, `ed0d45b8`, `a35f1b81`, `5114a4b4`…); go-cqrs-lite daemon commits (`775091d1e`, `b2583c334`, `a94aaccf6`, `b63fd8ad2`) carry the linter fix |
| Known-open                                             | Playwright specs not run; 00-41 report still modified-uncommitted at close                                                                                                                                           |
