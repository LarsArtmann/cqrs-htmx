# SUPERB Round-19 execution: tag-diff ritual, guards (R2/R18/R20, M19), D13 pins, battery triage

> STATUS REPORT · 2026-10-07 02:09 CEST · executes `docs/planning/2026-10-07_00-43_SUPERB-round19-release-path-verification-and-trust-surface-pareto-plan.md`
> Chain: round-18 docs-health (00-37) → this session. A concurrent session raced the same plan's dashboardui/GCL lanes (its reports: `2026-10-07_00-15` m06-f22-closure + `2026-10-07_01-08` m17-m19-versionz-audit-actor-config-validate) — lane discipline held; every red this session produced is attributable to their in-flight files, not mine.

## Executive summary

Executed the round-19 plan's unowned lanes: **M5** (the gotcha-24 tag-diff debt — all four sweep bumps audited from tag clones, ZERO gotcha-24-class drift found), **M17** (R2 consumability guard built + verified LIVE against the historical poison), **M18** (R18 bump-dep pre-flight + rc-checked commit; R20 runbook updates), **M19** (lint matrix fail-fast:false, mechanized env-leak audit — which found and fixed a REAL leak — and the ci-parity battery), **M22** (D13: the prefix-strip's coverage hole closed with 6 policy pins + a decision memo), plus the templ-drift CI repair. **M2 battery: partially green — every red is in the concurrent session's mid-rename files** (adminui/dashboardui reference a root asset API captured mid-rename; dashboardui+systemadapter cqrs-lint findings in their new files); root + all my lanes green. CI on pushed `3abc804e` was triaged: checks=templ codegen drift (FIXED this session), lint=5 dashboardui findings in their active lane.

## a) FULLY DONE

### M5 — gotcha-24 tag-diff ritual (the encoding-critical debt)

All four diffs audited from tag-range clones in /tmp (evidence-grade, not changelog-level):

| Bump                             | Verdict                                                                                                                                                                                                                                                                                                                                                                                                   |
| -------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| go-codec v0.3.0→v0.3.1           | **0 Go source files changed** — docs/flake/go.mod housekeeping only (`go 1.27` directive, error-family v0.11.0, gomega v1.44.0). The gotcha-24 encoding mechanism did NOT occur.                                                                                                                                                                                                                          |
| go-webauthn v0.18.1→v0.18.2      | Semantic but **default-safe**: new `validateSessionChallenge` backstop (rejects short/invalid stored challenges — ours are library-generated 64-byte raw-base64url; only an error-path constructs `SessionData{}`), opt-in `WithLoginAuthorizeUVInitialization`, UV-flag advance now RP-authorized with the default path byte-identical to v0.18.1, ML-DSA additive. No API breaks (compile+tests green). |
| templ-components v1.19.4→v1.20.1 | v1.20.1 = 1-line version heal. v1.20.0 added utils/wire (+628 new pkg), button/input token changes, ARIA feedback roles, PolledRegion `Now`, and the `Pad`→`NoPad` API swap (no cqrs-htmx consumer used `Pad`). Class tokens captured by the CSS class-set gate.                                                                                                                                          |
| go-flightrecorder v0.2.0→v0.2.1  | Internal-only: `captureOnce` dedup + `atomic.Pointer[sync.Once]` (Reset re-arm under in-flight async capture). No exported changes.                                                                                                                                                                                                                                                                       |

**No upstream filings** — zero surprises of the gotcha-24 class. TODO row struck with the full verdicts.

### M17 (R2) — family-release consumability guard

`scripts/checks/check-family-release-consumable.sh`: fetches `<module>@<version>.mod` from the proxy (curl), rejects `-00010101000000` placeholder requires, then walks every same-family submodule require the same way. rc: 0 consumable / 1 unconsumable+unpublished / 2 tool-failure. **Verified LIVE**: v1.20.1 → consumable (4 submodule walks clean); **v1.20.0 → correctly named the poison with all 4 offending placeholder lines quoted** — the exact historical fault, reproduced and detected. Offline fixture self-test 6/6 (`--check-gomod` mode). Wired atomically: flake app (`.#check-family-release-consumable`, carries `"$@"`), check-modules stage + self-test stage, CI self-test step (live check stays a bump-time tool — CI has the offline coverage).

### M18 (R18+R20) — bump-dep hardening + runbooks

- **Pre-flight (R18):** bump-dep now validates EVERY module it is about to bump BEFORE the first mutation (unique module paths from the match set → R2 checker); aborts with "upstream release … is unconsumable (placeholder sub-requires)"; proxy-unreachable (rc 2) degrades to a WARNING + continue (per-module builds remain the backstop). `BUMP_DEP_NO_NETWORK=1` skips for fixtures. Fixture cases: poisoned target aborts with go.mod untouched (file:// proxy, offline-deterministic) + NO_NETWORK skip — suite 5/5.
- **`--commit` rc-check (M18.4):** commit rc captured; non-zero reports "the sweep commit did NOT land; changes remain STAGED" + the daemon-absorption caution instead of the old unconditional success line. Success path prints the `--refresh-cache` fresh-tag-TTL reminder (gotcha 27).
- **Runbooks (R20):** `dependency-train-bump.md` gained validate-before-align (with the manual one-off command) + the moving-target stop-rule; rc-capture discipline was already present. `release-playbook.md` §3a gained the per-wave fresh-tag TTL dance (tag → `--refresh-cache` → re-run the gated commit) + CHANGELOG-before-tag (the identity-model v4.12.0 stub class).

### M19 — CI robustness

- **Lint job → `fail-fast: false` matrix** (11 module entries; 138 lines of sequential steps collapsed). One red module no longer hides the rest (the gotcha-27c class). YAML validated.
- **Env-leak audit mechanized:** `check-selftest-env-leaks.sh` — every self-tested checker that reads `${CI}` must strip CI (`env -u CI`) in its self-test; missing self-tests surface as INFO debt. **Found a REAL leak:** `test-check-release-train.sh` exercised the CI-advisory branch on GitHub runners instead of local-strict — fixed with `env -u CI` (suite re-verified 3/3). Fixture self-test 4/4. Wired: flake app + check-modules stages + CI steps.
- **`ci-parity-check.sh`** (gotcha-27c as one command): CI=true fixture self-tests → per-go.mod `tidy -diff` (testdata fixtures excluded) → lint → test → check-modules; stage-restricting args + `PARITY_SKIP`. Flake app `.#ci-parity-check`. No CI step by design (it would recursively run CI inside CI). Smoke run: **stage 1 green — ALL fixture self-tests pass under CI=true** (the leak fix holds suite-wide).

### M22 (D13) — ParseUserID prefix policy

- **22.1 (coverage confirmation): `eba45c80` added NO tests** — the brand-prefix strip had zero coverage. Closed: `TestUserID_ParseBrandPrefix_Policy` (6 subtests, green) pins known-brand strip, unknown-brand strip (generic policy), multi-colon rejection, non-ULID tail, brand-only, and the leading-colon quirk.
- **22.3 (memo):** `docs/drafts/2026-10-07-d13-parseuserid-prefix-policy-memo.md` — recommendation KEEP generic (the strip cannot forge an identity; it must still strict-parse as ULID; the real hazard is gotcha 16's `NewUserID` hash path). Owner call outstanding; the pins make a policy flip test-visible.

### CI triage on pushed `3abc804e` (run 37543524226: checks+lint red, 5 jobs green)

- **checks red = templ codegen drift**: the `events.templ` edit on master removed a line without an in-module `templ generate`; committed `_templ.go` carried stale line numbers. **FIXED this session**: regenerated in-module (canonical form), line-number-only diff, build green, committed (`486ffac6` lineage; daemon-absorbed into `182b553c`). Rides the next push.
- **lint red = 5 dashboardui findings** (cyclop `handlers_events.go`, dupl ×2 `handlers_audit.go`, globals+mnd `accent_color.go`) — all in the concurrent session's active refactor lane (their own report Q2 marks them hands-off-foreign); they were actively editing `assets_test.go` for the wsl piece during this session.

## b) PARTIALLY DONE

### M2 — post-wave battery (blocked by foreign mid-rename, not by my lanes)

| Leg                       | State                                                                                                                                                                                                                                                                                                                                         |
| ------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| erraudit-inventory --gate | **GREEN: TOTAL=0** over all 28 modules                                                                                                                                                                                                                                                                                                        |
| test-all                  | rc=1 at **adminui**: `undefined: cqrshtmx.ServeAsset` — the concurrent session's root asset API is mid-rename (`asset_serve.go` currently holds a garbage `func n(...)`) and adminui/dashboardui consumers still call the old name. Root module tests: green. Their own 01:08 report documents this exact daemon-captured-intermediate class. |
| check-cqrs-lint           | rc=1: findings in **dashboardui + systemadapter** — their in-flight files (new code awaiting suppressions/finish). All other 13 gate modules green.                                                                                                                                                                                           |
| coverage-gate             | rc=1 as a cascade of the adminui/dashboardui build breakage; every PRINTED threshold passed (root 94.8/90, identity-model 77.1/70, usermgmt 85.3/74, totp 88.2/80, webauthn 89.2/80, oauth2 89.2/80). No re-pin warranted on this evidence.                                                                                                   |
| parity battery            | stage 1 green (see M19); stages 2+ interrupted by an edit-while-running self-inflicted syntax error, re-queued.                                                                                                                                                                                                                               |

**Owed:** full battery re-run at a quiet tree (the foreign session's rename lands).

## c) VERIFIED-ELSEWHERE (concurrent session's lanes, this session's confirmation)

- **M6/D12 streamid guard**: fully wired (checker + self-test + flake app + check-modules + CI + AGENTS gotcha-25 receipt). Verified green on-tree: 13 display sites / 8 files, all pinned. The guard's landing was accompanied by the two demo identity fixes (datastar-demo `todoID`, dashboard-demo `userId` payload → `.Get()`, hermetic builds proven).
- **M3/F22 (GCL)**: confirmed CLOSED — `adttest.RunPaginationConformance` matrix live for memory/sqlite/bbolt/pebble/badger; its first run found the real sqlite MapScan tiebreak bug; CHANGELOG receipted.
- **M4 (GCL)**: `sort_paginate.go` doc already current; lint + full-suite `-race` batteries LAUNCHED this session (results pending at write time).

## d) NOT DONE (explicitly out of this session's lanes)

- M7–M10 dashboardui trust surface (ETags/versionz/actor/Config.Validate) — DONE by the concurrent session per its 01:08 report; not re-verified here beyond build/lint observation.
- M11–M15, M20–M21, M23–M27 — unstarted (owner batches, loginpage depth, embed story, v5 window).
- The D13/R11 owner calls themselves — memos are inputs, decisions stay with the owner.

## e) Incidents & lessons

1. **Edit-while-running shell scripts self-destruct**: my mid-flight edit of `ci-parity-check.sh` syntax-errored the RUNNING copy (bash reads incrementally). Kill + re-run after edits; never touch a script a background job is executing.
2. **`stage_wanted` counted the name in `$@`** — the "restrict to one stage" flag silently ran ALL stages. Fixed with `shift` (`$@` includes `$1` in bash functions).
3. **sed BRE `\(\)` trap again**: `\(` is grouping, not literal — `aggID.String()` → `aggID.Get()()` on the first demo fix; caught immediately by reading the line back.
4. **mvdan/sh tail-quArg**: `tail -5 file` fails ("option used in invalid context"); use `tail -n 5`.
5. **LSP phantom reds, third confirmation**: 9 dashboardui compile errors in project diagnostics while `go vet` was green at the same tree — gotcha 14 holds.
6. **Daemon sweep mixing lanes is now the norm**: my M17/M18/M19 files landed inside `4e16a0e3` together with the foreign session's systemadapter M13/M14 files. Attribution is reconstructed here; amend-for-readability was attempted twice and abandoned both times (pre-commit could not pass mid-churn — correct behavior, cosmetic message not worth `--no-verify`).

## f) Files & commits (this session)

- Scripts/gates: `check-family-release-consumable.sh` + self-test; `check-selftest-env-leaks.sh` + self-test; `ci-parity-check.sh`; `bump-dep.sh` (pre-flight + rc-checked commit); `test-bump-dep.sh` (+2 fixtures); `test-check-release-train.sh` (env-leak fix); `check-streamid-identity.sh` + self-test (verified, built by the concurrent session).
- CI/flake: `ci.yml` (lint matrix, 3 new step pairs), `flake.nix` (3 apps, 4 stage entries).
- Code: `examples/datastar-demo/domain_cqrs.go` + `examples/dashboard-demo/main.go` (`.String()`→`.Get()` identity fixes); `dashboardui/events_templ.go` (drift regen); `identity-model/model_test.go` (D13 pins).
- Docs: TODO_LIST (M5 strike, more pending below), `docs/drafts/2026-10-07-d13-parseuserid-prefix-policy-memo.md`, `dependency-train-bump.md`, `release-playbook.md` §3a, AGENTS gotcha-25 receipt.
- Key commits: `486ffac6` (AGENTS receipt), `182b553c` (templ fix + M5 strike, daemon), `4e16a0e3` (M17/M18/M19 files, daemon-swept with foreign systemadapter work).

## g) Next

1. At quiet tree: re-run `nix run .#test-all` + `.#check-cqrs-lint` + `.#coverage-gate` + `.#ci-parity-check` — expect green on all my lanes; strike the battery row with full receipts.
2. Push when the foreign session's rename lands (carries the templ-drift CI fix + M5/M17/M18/M19/M22 work); watch the fresh-run for the lint-matrix's first multi-red surface.
3. Owner calls queued: D13 (memo ready), R11 credential memo (unstarted), PapDashboard nudge.
4. GCL: read the launched lint + `-race` results into the M4 row; `fail()`/`Close()` unify still open.
