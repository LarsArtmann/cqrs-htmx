# cqrs-lint Rebuild Triage — Root Cause Fixed At The Linter Source, Gate Green

> ANNOTATED 2026-10-01 (docs-health round 13): **SUPERSEDED-BY [`2026-09-30_08-03_cqrs-lint-session-forgotten-items-audit.md`](archived/2026-09-30_08-03_cqrs-lint-session-forgotten-items-audit.md)** (its own header says so) — this is the mid-session triage snapshot; the 07-19 report is the close-of-fix and 08-03 adds the forgotten-items audit. Known-open items here (system rebuild, E005/V007/A016/V006 residual triage, P009 codec decision, `--fail-on-stale-suppressions`) are tracked: TODO_LIST P2/P3 + ROADMAP OQ25. Archived per the tail convention.

**Session:** 2026-09-29 late → 2026-09-30 01:5x CEST
**Scope:** Single-thread session — triage of the failing `check-cqrs-lint` gate after the 2026-09-29 binary rebuild (upstream `3756eb4`), escalated by the owner's "you took the easy way out" challenge into fixing the root causes instead of suppressing them. Two repos touched: this one and `~/projects/go-cqrs-lite` (`cmd/cqrs-lint`, the linter's actual home).
**Trigger:** Manual `GOTOOLCHAIN=auto cqrs-lint` run (workspace mode) exiting 1 with an ERROR-severity C017 finding, 21 C040 warnings, and one stale-suppression warning.

---

## Executive summary

The rebuilt cqrs-lint binary turned the ROOT recursive-walk run red. The easy path (config-disable + suppress everything) was taken first and then reversed after the owner's challenge. Final state:

1. **C040's 21 phantoms were root-caused to a collector gap in the linter itself and FIXED AT SOURCE** in `go-cqrs-lite/cmd/cqrs-lint`: `event.New`/`NewEvent`/`catalog.Event` arguments were resolved from string literals only, so this repo's alias-based emissions (identity-model consts → usermgmt `var` re-exports, the "Go has no const aliases" pattern) and `EventCatalog.Register(EventMetadata{Type: string(const)})` declarations were invisible. The collector now follows const AND var-alias chains and reads metadata-style registrations. No config exemption exists or is needed.
2. **The C009 panics were eliminated properly** — adminui + dashboardui asset handlers now read embedded assets once at `New` with error propagation and unit-tested error paths, replacing both compile-time-unreachable panics and their suppressions.
3. **C042 was verified against the demo code** (fresh streams; version 0 IS the correct new-stream contract) and suppressed with that verified reason; **the deprecated `event.AggregateID()`** call in e2e became `StreamID()` (deprecation confirmed in go-cqrs-lite source).
4. **The severity model was corrected in the record**: exit codes fire on ERROR-severity only, `--strict` gates package LOAD errors, the preset `min_severity` is a display floor — the earlier "strict fails at WARNING+" claim was wrong.
5. The gate is green throughout the handoff window; a controlled prefix-build comparison proves the linter fix adds zero new warnings anywhere.

## a) FULLY DONE

1. **C040 fixed at the linter source** (`~/projects/go-cqrs-lite/cmd/cqrs-lint`, committed by that repo's daemon): (a) `scanCallExpr` records unresolvable event-type args as pending refs; (b) `scanConstDecl` additionally records const→const alias expressions; (c) NEW `scanVarAliasDecl` records cross-package `var` re-exports (the cqrs-htmx identity-model↔usermgmt pattern); (d) NEW `ResolveEmittedEventTypeConsts` post-pass expands alias chains (memoized, cycle-guarded, depth-bounded) and resolves pending refs into `EventTypesEmitted`/`EventTypesInCatalog`; (e) NEW EventMetadata-style `Register(CompositeLit{Type: ...})` collection (literal, const ref, or `string(const)`). All four post-pass call sites wired.
2. **Linter regression tests added** (scanner_test.go): const-identifier emission, type-inherited alias chain, cross-file selector alias, catalog const arg, unresolvable-const negative, alias-cycle safety, var-alias emission, same-package-var negative, EventMetadata Register. Full `cmd/cqrs-lint` module suite: BUILD RC=0, TEST RC=0.
3. **Real-repo verification with a locally built fixed binary**: root recursive walk + all 13 per-module runs, C040 ENABLED → **0 C040 findings, 0 errors, RC=0 everywhere**.
4. **Zero-new-warnings proof**: a control binary built from the pre-fix commit (`ad1dcd46a` worktree) produces IDENTICAL warning sets on identity-model (16× E005 + 1× V006) and usermgmt (41× V007 + 1× A016) — the residual warnings are the newer linter source's pre-existing behavior, not this session's regression. (First "baseline" attempt was contaminated — the daemon had already committed the fix — caught by verifying `git status`/stash state; redone properly via worktree.)
5. **C009 fixed by design, not suppression**: `newAssetHandler(fsys, name, contentType) (http.Handler, error)` reads at construction; `dashboardui.New`/`adminui.New` propagate the error; routes use the stored handlers. Missing-asset unit tests added (fstest.MapFS negative + real-embed positive); one existing adminui test updated to the new signature; both module suites green.
6. **C042 verified then suppressed**: all three `examples/dashboard-demo` Save sites write `Version(1)` events at `expectedVersion=0` to freshly created streams — the rule's "optimistic concurrency bypassed" premise is false there; suppressed with the verified reason, matching the demo's established per-finding style.
7. **`event.AggregateID()` deprecation confirmed** (go-cqrs-lite `event/v3_compat_aliases.go`: "Deprecated: use id.StreamID") and the e2e call site fixed to `StreamID()` (identical value per the compat tests).
8. **Severity semantics corrected in the record** (AGENTS.md gotcha 13 + CHANGELOG): exit code = ERROR findings; `--strict` = load errors; `min_severity` = display floor. Empirically pinned by an RC=0 run with 21 warnings present.
9. ~~**Gate green through the handoff window**: `nix run .#check-cqrs-lint` (system binary, C040 enabled in config) → all 14 modules pass; the old binary's phantom C040 warnings are non-failing by the corrected semantics. Both status gates pass on the report.~~ done (verified standing through the 08-03 handoff; docs-health pass 2026-10-01).
10. **Docs paid**: CHANGELOG entry rewritten to final truth; TODO_LIST triage item replaced by the bounded handoff item (system rebuild → verify C040 silent); AGENTS.md quick-ref row + gotcha 13 updated; ROADMAP OQ 25 opened for the P009 []byte/JSON codec question; `.cqrs-lint.json` carries the fix story in place of the never-actually-needed exemption.

## b) PARTIALLY DONE

1. **Playwright/e2e spec runs** — the Go test suites for both touched UI modules, the workspace build, the e2e server build, and vet all pass, but the browser-truth Playwright specs were not executed this session (the e2e/server edits were comment- and one-field-rename-only; risk near-zero, verification absent).
2. **Residual linter warning classes** — identity-model 16× E005 ("register a handler" on pure-domain command types) + usermgmt 41× V007 (`stack.Materialize` v5 migration advisory, maps to ROADMAP OQ 11/ADR-0051) + 1× A016 (idempotency middleware advisory) + 4× V006 (the documented per-module-train false positive). Verified pre-existing (identical on the pre-fix control), non-gating, but untriaged: suppress-with-reason vs fix vs accept is open.
3. **Manual-run cleanliness until the system rebuild** — with the still-installed pre-fix binary, manual root walks print 21 stale C040 warnings (non-failing). The fixed binary exists at `/tmp/cqrs-lint-fixed`; the system rebuild is the owner's move.
4. **A013 (pointer-embedding) in the 7 example modules** — dispositioned "accepted for the frozen library API" but the examples-as-dogfooders option was never evaluated against `BasicCommand`'s receiver shapes.

## c) NOT STARTED (observed, routed or deferred)

1. **System cqrs-lint binary rebuild** — TODO_LIST item with exact verification step; blocked only on an owner rebuild.
2. **E005/V007/A016/V006 residual triage** — noted in the TODO item as separate work; nothing started.
3. **P009 codec decision** — routed to ROADMAP OQ 25 (storage-format decision: CBOR vs non-binary field shapes); decision itself untouched.
4. **`--fail-on-stale-suppressions` adoption** — still unwired; today's stale C017 was found by a human, not a gate.
5. **Pre-existing, re-observed:** `check-cqrs-lint` CI wiring (blocked on a Go-installable distribution); manual-run recipe block for AGENTS.md.

## d) TOTALLY FUCKED UP / near-misses (honest ledger)

Nothing shipped broken. The session's real stumbles, in descending severity:

1. **The first C040 fix was WRONG and the unit tests hid it.** The initial collector change (const decls only) passed every new unit test yet left 21/21 phantoms on the real repo — because the initial root-cause pass pattern-matched on identity-model's `constants.go` (typed consts) and never read `es_constants.go`, where the actual emission-facing aliases live as `var`. Only instrumented debug runs against the real tree surfaced the var-alias shape. Lesson recorded in §e: a green fixture proves the fixture's shape, not the repo's.
2. **The "baseline" control was contaminated on first attempt** — `git stash push` silently stashed nothing (the go-cqrs-lite daemon had already committed the fix), so the first "pre-fix" binary included the fix. Caught immediately by verifying stash/worktree state; redone with a worktree at the true pre-fix commit.
3. **An unverified claim shipped mid-session**: "strict fails at WARNING+" was written into AGENTS/CHANGELOG/report from one ambiguous observation and stood until the fixed-binary run exited 0 with 21 warnings. Corrected everywhere; the source-level semantics are now the pinned truth.
4. **Process slips recovered same-session:** 3-of-6 edit batch failures (files not View-read first), one stale old_string on the AGENTS row, one rg `-r` flag misused (mangled a search's output), two background jobs racing tree edits (baselines happened to be valid — luck, not process), and `/tmp` log files vanishing under concurrent sessions (switched to shell-variable capture).

## e) WHAT WE SHOULD IMPROVE

1. **Verify fixes against the real surface, not just fixtures.** The unit-only green + real-repo red gap (item d1) is the session's core lesson: before believing a root cause, read every file in the failing surface, not just the one that fits the hypothesis.
2. **Make suppression staleness loud.** `--fail-on-stale-suppressions` remains the mechanical answer to the C017 class.
3. **Treat daemon-committed trees as the norm.** Check `git status`/`git log` BEFORE any stash/checkout-style isolation — the daemon had already committed the "uncommitted" fix.
4. **Capture long outputs in shell variables, not /tmp files** under concurrent sessions (two log-file reads failed this session).
5. **Claims about tool semantics go in only with source-level or controlled-experiment evidence** — both the strict-severity claim and the first root cause failed that bar and both cost a correction cycle.
6. **Diff rule sets across linter updates** (`cqrs-lint rules` old vs new) so new warning classes (V007/A016 etc.) are triaged deliberately, not discovered as surprise noise.

## f) Next things to get done (brainstorm — the earlier 40-item list stands, with these deltas; ROADMAP fuel, not a commitment list)

**Resolved from the earlier list this session:** upstream C040 report (fixed at source instead), re-enable C040 + drop exemption (done — none exists), P009 routing (ROADMAP OQ 25), locate cqrs-lint source (found), C042 verification, AggregateID verification, module test runs.

**New/renewed top items:**

1. System rebuild → verify `cqrs-lint --strict --verbose .` shows zero C040 on a root walk (TODO_LIST item, exact step recorded).
2. Triage identity-model 16× E005 (likely a missing cross-module fail-open for pure-domain command modules — candidate linter improvement in `~/projects/go-cqrs-lite`).
3. Triage usermgmt 41× V007 against ADR-0051's migration posture (suppress-with-reason vs batch-migrate `stack.Materialize`).
4. Suppress or accept A016 (idempotency advisory) + the 4× V006 lockstep warnings per the documented per-module-train policy.
5. Run the e2e Playwright suite once over this session's UI-module changes.
6. Diff `cqrs-lint rules` output between binary `3756eb4` and the current source to enumerate every rule that changed.
7. Wire `--fail-on-stale-suppressions` into the flake gate (after checking interaction with `--strict`).
8. Sweep ALL inline `//cqrs-lint:ignore` comments: assert each still fires under the current source; delete the rest.
9. Add the copy-paste manual-run command block (env + invocation) to AGENTS.md gotcha 13.
10. Decide A013 for examples (value-embed `BasicCommand` in demos if receiver shapes allow).
11. Upstream-ask: `min_severity`/strict/exit-code semantics in the linter README (this session had to read `run.go` to learn them).
12. Evaluate `cqrs-lint scorecard` + `--group-by module` for triage ergonomics.
13. Consider a repo-pinned cqrs-lint build (flake input on go-cqrs-lite) so gate provenance stops depending on system-profile rebuild timing.
14. Re-verify the corrected severity-semantics facts on the next binary update.
15. agents-notes narrative: the easy-way-out correction arc — first root cause wrong, unit tests green, real walk red, var-alias discovery, contaminated baseline, prefix-worktree proof.

(The remaining items from the earlier 40 — config-inheritance verification, JSONC fixture gate, exclude-key semantics, CI lane for e2e/examples at INFO threshold, docs cross-links, coverage-gate re-run, treefmt over markdown — stand unchanged.)

## g) Questions I can NOT figure out myself

1. **When does the system profile rebuild, and should the flake pin cqrs-lint?** Until then the repo is green but manual runs show stale C040 noise. If rebuild timing is owner-controlled, say the word; if you'd rather the gate stop depending on the system binary entirely, I can wire a flake-local cqrs-lint build from `~/projects/go-cqrs-lite`.
2. **A013 appetite:** should examples adopt value-embedded `BasicCommand` (dogfooding the rule) or is the pointer-embedding pattern canonical everywhere and worth suppressing in examples?
3. **Advisory-noise appetite:** the newer linter source surfaces E005×16 / V007×41 / A016×1 / V006×4 as non-gating warnings. Suppress-with-reason for clean verbose runs, triage individually, or leave visible as standing nudges?

---

## Verification evidence appendix

| Claim                                    | Evidence                                                                                                                                                                                                                                   |
| ---------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| C040 root cause: literal-only collectors | `scanner_calls.go` pre-fix used `StringLit(call.Args[0])` for event.New/NewEvent/catalog.Event; `es_constants.go` declares emission aliases as `var` (gotcha-15 pattern); `catalog` registrations use `EventMetadata{Type: string(const)}` |
| Fix works with rule ENABLED              | Fixed binary root walk: `C040=0 C038=0 WARN=0 ERR=0 RC=0`; all 13 per-module runs RC=0, C040=0                                                                                                                                             |
| Fix adds zero new warnings               | Pre-fix commit `ad1dcd46a` worktree build vs fixed binary: identical E005=16 (identity-model), V007=41 + A016=1 (usermgmt)                                                                                                                 |
| Linter suite                             | `cmd/cqrs-lint`: BUILD RC=0, TEST RC=0 (9 new regression tests included)                                                                                                                                                                   |
| C009 refactor                            | `newAssetHandler` error propagation in both modules; dashboardui suite RC=0, adminui suite RC=0; missing-asset unit tests (fstest.MapFS)                                                                                                   |
| C042 verified                            | `examples/dashboard-demo` seeds: `aggID := id.NewStreamID()` per item, first write `Version(1)`/expected `Version(0)` — correct new-stream contract                                                                                        |
| AggregateID deprecated                   | go-cqrs-lite `event/v3_compat_aliases.go:17` "Deprecated: use id.StreamID"                                                                                                                                                                 |
| Builds                                   | Workspace `go build ./...` RC=0; standalone e2e/server build RC=0                                                                                                                                                                          |
| Gate green (handoff window)              | `nix run .#check-cqrs-lint` (system binary): all 14 modules pass                                                                                                                                                                           |
| Severity semantics                       | `run.go`: exit on ERROR findings; `--strict`→`isStrictMode` gates LoadErrors; `resolveMinSeverity` = display floor; empirically RC=0 with 21 warnings                                                                                      |
| Status gates                             | `check-status-annotations.sh` RC=0; `check-status-rows.py` RC=0                                                                                                                                                                            |
| Code commits                             | cqrs-htmx: daemon commits through the session (`436e821b`, `fef0da50`, later sweeps); go-cqrs-lite: daemon commits `775091d1e`/`b2583c334`/`a94aaccf6`/`b63fd8ad2` carry the linter fix                                                    |
