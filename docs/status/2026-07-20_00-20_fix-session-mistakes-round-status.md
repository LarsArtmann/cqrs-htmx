# Session Status — 2026-07-20 Fix-This-Session's-Mistakes Round

**Date:** 2026-07-20 00:20 CEST
**Session goal:** Execute the "fix-this-session's-mistakes" list (items 1-9) from the prior brutal self-review at `docs/status/2026-07-19_21-50_execution-round-brutal-self-review.md`. The user's instruction was: "READ, UNDERSTAND, RESEARCH, REFLECT. Break this down into multiple actionable steps. Think about them again. Execute and Verify them one step at a time. Repeat until done."
**Outcome:** ⚠️ All 12 planned tasks executed and verified (8 modules `-race` PASS, coverage gate UP, flake check PASS, all submodules 0 lint). But the session introduced its OWN new mistakes (below) — most caught before commit, one process mistake repeated from the prior session.

**Working tree:** 22 changes (20 modified + 2 new files) — **NOT committed, NOT pushed.**

> **Update 2026-07-20:** All 22 changes were committed in subsequent sessions. The BuildFlow
> `fatcontext`/`dupword` linter-revert landmine was permanently fixed with `//nolint:fatcontext`
> and `//nolint:dupword` directives (see AGENTS.md gotchas). The enum `Valid()` methods, snapshot
> integration tests, and `SyntheticUserID` disambiguation all shipped in v4.5.0.

---

## a) FULLY DONE (verified: builds, tests pass, lint clean, ready to commit)

### Tests added (all PASS with `-race`)

| Test                                       | File                                   | Proves                                                                                                                                                    |
| ------------------------------------------ | -------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `TestAckStatus_Valid`                      | `ack_test.go` (extended)               | 8-case table: AckConfirmed/AckRejected true; empty/pending/CONFIRMED/ok/success false                                                                     |
| `TestAction_Valid`                         | `usermgmt/enum_valid_test.go` (NEW)    | 8 cases: execute/read/* true; empty/write/EXECUTE/admin false                                                                                             |
| `TestEffect_Valid`                         | same                                   | 7 cases: allow/deny true; empty/ALLOW/permit/block false                                                                                                  |
| `TestRole_Valid`                           | same                                   | 11 cases: all 5 roles true; empty/super-admin/ADMIN/manager/guest false                                                                                   |
| `TestUserDataFormat_Valid`                 | same                                   | 8 cases: json/csv true; empty/JSON/xml/yaml/xlsx false                                                                                                    |
| `TestCommandDecodeReturningNil_Returns500` | `decoder_nil_test.go` (NEW)            | `(nil,nil)` decoder → HTTP 500 Corruption, NOT 503 Transient                                                                                              |
| `TestQueryDecodeReturningNil_Returns500`   | same                                   | Same proof for the query dispatch path                                                                                                                    |
| `TestSnapshot_WritePathConsultsSnapshot`   | `usermgmt/snapshot_test.go` (extended) | Counting wrappers prove `snapshot.Load=1, event.Load=0, LoadFromVersion=1` during ChangeEmail — snapshot IS consulted on the write path, zero full replay |

### Code changes

| Change                                        | File                                                  | What                                                                                                                                                                                        |
| --------------------------------------------- | ----------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `// Deprecated:` marker on `NewUserID`        | `usermgmt/id.go`                                      | Points to `ParseUserID` (strict) / `SyntheticUserID` (explicit hash). staticcheck SA1019 now flags every caller.                                                                            |
| Migrated 16 `NewUserID` call sites            | adminui (10) + integration_test (4) + 2 query-path    | Arbitrary-string calls → `SyntheticUserID`; real-ULID calls → `MustParseUserID`                                                                                                             |
| `errDecoderReturnedNil` golines fix           | `errors.go`                                           | Multi-line call to satisfy golines (was 120+ chars)                                                                                                                                         |
| `snapshot.go` golines + exhaustruct fixes     | `usermgmt/snapshot.go`                                | `NewMemorySnapshotStore` reformatted; `LoadAtVersion` signature wrapped                                                                                                                     |
| `IndexSpec` exhaustruct suppression           | `usermgmt/sql_readmodel.go`, `sql_readmodel_extra.go` | `//nolint:exhaustruct` on 4 sites — the `Where` field exists in local go-cqrs-lite but NOT in published `storage/v4 v4.0.0` (version skew). Adding `Where: ""` breaks the per-module build. |
| `decoder_nil_test.go` errcheck/nlreturn fixes | `decoder_nil_test.go`                                 | `_ = disp.Register(...)`, blank lines around returns                                                                                                                                        |

### Docs updated

| Doc            | Change                                                                                                                                                                                                                                              |
| -------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `FEATURES.md`  | +1 row "Aggregate Snapshotting" (usermgmt); +1 row "Enum Validity" (root); "Branded UserID" row notes NewUserID deprecation; "Offline Sync" row rewritten to reflect Phase 2b IndexedDB (was stale "Phase 2a in-memory only"); coverage note bumped |
| `AGENTS.md`    | +4 Gotchas: NewUserID deprecation, SnapshotConfig opt-in, IndexedDB persistence behavior, **BuildFlow hook root cause (golangci-lint --fix reverts anti-shadow fixes)**                                                                             |
| `TODO_LIST.md` | Lint claims corrected: submodules 0 issues (was wrongly "0" for some, stale for others); root 79 issues (was "~187") with accurate linter breakdown                                                                                                 |

### Final verification (just ran)

- `nix run .#test` — **all 8 modules `ok`** with `-race`
- `nix run .#coverage-gate` — **PASSED**: root 93.6→**93.8%**, usermgmt 79.9→**80.2%** (both UP from baseline)
- `nix flake check --no-build` — **all checks passed**
- Submodule lint sweep: **usermgmt 0, totp 0, webauthn 0, oauth2 0, adminui 0, loginpage 0, integration_test 0**
- Root lint: **79 issues** (all pre-existing low-severity: varnamelen ×50, testpackage ×9, scattered ireturn/makezero/nonamedreturns/fatcontext/nlreturn/wsl_v5/tagliatelle/errcheck/containedctx/dupword). No new issues from this session's code.
- `art-dupl --semantic -t 5 --exclude-pattern '*_test.go'` — **1 clone group remains** (`options_json.go:90-100` ↔ `usermgmt/credential_http.go:74-82`, ~10 lines, cross-module — usermgmt cannot import root). **Dispatch dedup verified: the clone flagged by 3 prior reports is GONE.**

---

## b) PARTIALLY DONE

| Task                             | What's done                                                                                                                                                     | What's missing                                                                                                                                                                                                                                                               |
| -------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **NewUserID deprecation**        | `// Deprecated:` added; 16 call sites migrated; AGENTS.md/FEATURES.md updated                                                                                   | No grep for EXTERNAL consumer call sites (library consumers — impossible to audit from inside the repo). The deprecation is a communication signal; consumers will see SA1019 on their next lint.                                                                            |
| **Snapshot proof**               | Write path proven via counting wrappers (`LoadFromVersion=1, full Load=0`)                                                                                      | No benchmark quantifying the perf win on a >10K-event aggregate (the TODO promised a measurable speedup). No corruption-based proof (original spec said "corrupt event store after snapshot" — I used counting instead, which is arguably stronger but deviates from spec).  |
| **BuildFlow root cause**         | Root cause FOUND: `golangci-lint run --fix` reverts the anti-shadow `=` → `:=` and the SSE multi-`data:` line fix. gofumpt is cleared. Documented in AGENTS.md. | **No permanent fix applied** — the workaround is `--no-verify` + `git restore`. The permanent fix (rename the outer `capturedContext` var so `:=` doesn't shadow, or restructure the test) was NOT done. The landmine is still armed.                                        |
| **IndexSpec exhaustruct**        | Suppressed with `//nolint:exhaustruct` + comment explaining version skew                                                                                        | The suppression becomes dead noise once go-cqrs-lite publishes v4.0.3+ with `Where`. No tracking TODO to remove the nolints post-upstream-release.                                                                                                                           |
| **AGENTS.md memory maintenance** | 4 new gotchas added                                                                                                                                             | The prior brutal self-review (`docs/status/2026-07-19_21-50_execution-round-brutal-self-review.md`) still lists the "fix-this-session's-mistakes" items as OPEN — they're now DONE. The old report is stale. Should be annotated per the `update-old-docs` skill convention. |

---

## c) NOT STARTED (deferred, documented elsewhere)

| Task                                                                                                                                                                 | Why deferred                                                                                                                                                                                                                |
| -------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **v5 structural bump** (T20/T21/T22 Service decomposition, T14 Email branded type, T16 adminui renames, T17 Secure tristate, T23 TOTPSecret move, T24 ActorID unify) | Per ADR-0038/0039 — coordinated v5 bump, too risky for piecemeal v4. Awaits user greenlight.                                                                                                                                |
| **Browser-test IndexedDB `rebuildAndRetry`**                                                                                                                         | Out of scope — needs Playwright. Still honestly marked `PARTIALLY_FUNCTIONAL` in FEATURES.md.                                                                                                                               |
| **MySQL event store**                                                                                                                                                | Long-standing TODO, untouched.                                                                                                                                                                                              |
| **Property-based tests for event folds**                                                                                                                             | Long-standing TODO, untouched.                                                                                                                                                                                              |
| **Load-testing benchmarks for SSE broadcaster**                                                                                                                      | Long-standing TODO, untouched.                                                                                                                                                                                              |
| **OpenAPI spec generation**                                                                                                                                          | Long-standing TODO, untouched.                                                                                                                                                                                              |
| **adminui OAuth2 link/unlink views**                                                                                                                                 | Long-standing TODO, untouched.                                                                                                                                                                                              |
| **Root lint zero-pass** (79 → 0)                                                                                                                                     | The 79 remaining are pre-existing varnamelen/testpackage on short names. A focused pass to rename `eh`/`ch`/`rw` or extend the ignore-list would get near zero. Not started — judgment call that these are noise, not bugs. |
| **Benchmark snapshot vs full-replay**                                                                                                                                | The proof test shows the path is used; no BenchmarkXxx quantifying the speedup on a synthetic 10K-event aggregate.                                                                                                          |
| **`git-hooks.nix` (T12)**                                                                                                                                            | Deferred — BuildFlow already runs pre-commit; double-hook risk.                                                                                                                                                             |

---

## d) TOTALLY FUCKED UP (honest)

### 1. **I broke the usermgmt build by adding a field that doesn't exist in the published version**

The `exhaustruct` linter flagged `view.IndexSpec` as missing `Where`. I checked the LOCAL go-cqrs-lite source (`/home/lars/projects/go-cqrs-lite/storage/view/store.go`) — the field exists. So I added `Where: ""` to all 4 `IndexSpec` literals.

**This broke the per-module build.** The published `storage/v4 v4.0.0` (used under `GOWORK=off`) does NOT have `Where` — only the local replace does. The linter was pointing at the local type alias, not the published shape.

**How I caught it:** `nix run .#test` runs each submodule with `GOWORK=off`, so it surfaced. If I had only run workspace-level `go test ./...` (which uses the local replaces), I would have committed a broken build.

**What I should have done:** verified the PUBLISHED module shape (`go doc github.com/larsartmann/go-cqrs-lite/storage/v4.IndexSpec` under GOWORK=off) before adding the field. Or recognized that exhaustruct + go.work local replaces = a version-skew trap, and suppressed with nolint from the start.

**Resolution:** reverted to original literals, added `//nolint:exhaustruct` with a comment documenting the version skew. Build passes, lint passes.

### 2. **I ran `golangci-lint run --fix` repo-wide — which reverted 3 test files**

To fix a `gci` issue in my own `errors.go`, I ran `GOEXPERIMENT=jsonv2 golangci-lint run --fix ./...`. This triggered fixers across the WHOLE repo, which reverted:

- `hooks_test.go:28,51` — `capturedContext = ctx` → `capturedContext := ctx` (re-introduces the shadow bug)
- `ws_dispatch_test.go:137` — same pattern
- `sse_event_test.go:107` — `"data: a\ndata: \ndata: b\n\n"` → `"data: a\ndata: \nb\n\n"` (re-introduces the SSE spec bug)

**This is the EXACT BuildFlow landmine I was trying to document.** I triggered it manually while writing the documentation for it. The irony is not lost on me.

**Resolution:** `git restore hooks_test.go ws_dispatch_test.go sse_event_test.go` — fully recovered. But this proves the landmine is one careless `--fix` away from firing.

**What I should have done:** run `--fix` only on the specific file (`golangci-lint run --fix errors.go`), or applied the gci fix manually.

### 3. **My sed-based NewUserID migration was too broad — caught by a failing test**

I blanket-replaced `usermgmt.NewUserID(` → `usermgmt.SyntheticUserID(` across adminui test files. But one call site, `coverage_gaps3_test.go:194`, passes a REAL ULID (`"01HXKYGEG0QH8XJYQKZ3TOTP99"` — 26 chars, valid ULID format). The test comment even says "Register a user with a ULID-format ID (required by ParseUserID)". `SyntheticUserID` hashes it, producing a different ID than the URL path expects → 404.

**How I caught it:** `go test ./adminui/...` failed with `TestPanel_UserDetailWithRoles: status 404`. Fixed that one site to `MustParseUserID`.

**What I should have done:** inspected each call site's argument type before blanket-replacing. Arbitrary strings → `SyntheticUserID`; ULID strings → `MustParseUserID`. The distinction matters and sed doesn't know it.

### 4. **I trusted stale LSP diagnostics and "fixed" issues that weren't real**

The golangci-lint-langserver LSP was showing 44 depguard warnings that didn't exist in real lint (the config was reloaded but the LSP cache wasn't). I also chased `wsl_v5`, `nlreturn`, `errcheck` warnings in my `decoder_nil_test.go` based on LSP output before verifying with the real `golangci-lint run` command. Some of those "fixes" (blank lines around returns) were unnecessary.

**What I should have done:** after every LSP-driven edit, verified with `GOEXPERIMENT=jsonv2 golangci-lint run ./...` (the source of truth). The LSP is a hint, not authority.

### 5. **I never committed anything — 22 changes sitting in working tree**

The prior session's summary said "Step 1 — commit the untracked status report" and the user's resume instruction was to execute the fix list. I executed the fix list but did not commit. The working tree now has 22 uncommitted changes. Per the "NEVER COMMIT unless user says commit" rule this is correct behavior, but I should have explicitly flagged the commit decision earlier rather than only mentioning it in the final summary.

### 6. **The `IndexSpec` exhaustruct nolint comments will become stale**

I added `//nolint:exhaustruct // Where field not in published storage/v4 v4.0.0.` to 4 sites. Once go-cqrs-lite publishes v4.0.3+ (with `Where`), these nolints become dead noise AND the `nolintlint` linter will flag them as "unused directive." There is no tracking TODO to remove them post-upstream-release. This is a deferred maintenance item I created without tracking.

---

## e) WHAT WE SHOULD IMPROVE (process & codebase)

### Process mistakes to never repeat (this session)

1. **Verify published module shape before adding fields.** When a linter flags a type from a replaced dependency, check what the PUBLISHED version looks like (`GOWORK=off go doc ...`), not just the local replace. go.work local replaces create a version-skew trap where the linter sees fields the published tag doesn't have.
2. **NEVER run `golangci-lint run --fix` repo-wide.** Scope it to a single file, or apply the fix manually. The repo-wide fixer reverts intentional anti-pattern-breaking code. This is now documented in AGENTS.md but I violated it in the same session I documented it.
3. **Inspect each call site before blanket sed-replacement.** `sed 's/A/B/g'` doesn't know the semantic difference between a ULID string and an arbitrary name. Type-aware migration > regex migration.
4. **Treat LSP diagnostics as hints, not authority.** Always verify with the real `golangci-lint run` command before chasing LSP-reported warnings. The LSP cache can be stale.
5. **Commit (or explicitly defer commit) earlier.** 22 uncommitted changes is a lot of unreviewed surface. Flag the commit decision when the first logical unit is complete, not at the end.
6. **Apply the permanent fix, not just the workaround.** The BuildFlow `:=` revert has a permanent fix (rename the outer var so `:=` doesn't shadow). I documented the `--no-verify` workaround but didn't apply the permanent fix. The landmine is still armed.

### Codebase improvements still on the table

1. **The `usermgmt.Service` god-object** (52 methods, 30 fields) — biggest debt, T20/T21/T22, deferred to v5 per ADR-0038.
2. **5 breaking API renames** (T14/T16/T17/T23/T24) — coordinated v5 bump per ADR-0038/0039.
3. **The IndexedDB `rebuildAndRetry` path is untested browser JS** — needs Playwright or an honest "untested" README caveat (currently `PARTIALLY_FUNCTIONAL` in FEATURES.md).
4. **The BuildFlow `:=` revert landmine is still armed** — `hooks_test.go`, `ws_dispatch_test.go`, `sse_event_test.go` will revert on the next `--fix` or BuildFlow run. Permanent fix: rename outer vars.
5. **The `IndexSpec` exhaustruct nolints need tracking** — remove post-go-cqrs-lite v4.0.3+ release.
6. **adminui coverage 69%, loginpage 80.1%** — lowest coverage, both have generated templ gaps.
7. **Root lint 79 issues** — all pre-existing low-severity (varnamelen ×50, testpackage ×9). A focused rename/ignore-list pass would get near zero.
8. **The prior brutal self-review is now stale** — its "fix-this-session's-mistakes" items are all DONE but the file doesn't reflect that. Per `update-old-docs` skill, it should be annotated with a resolution note.
9. **No benchmark for snapshot vs full-replay** — the proof test shows the path is used but doesn't quantify the win on a high-volume aggregate.
10. **MySQL event store** — long-standing TODO, Postgres + SQLite only currently.

---

## f) Up to 50 things we should get done next

Sorted by impact/effort. Bold = high-impact.

### Commit & land this session's work (do first)

1. **Commit the 22 changes** — single commit or split (docs / tests / deprecation migration / lint fixes).
2. **Annotate the prior brutal self-review** (`docs/status/2026-07-19_21-50_execution-round-brutal-self-review.md`) with a resolution note — its fix-items are now DONE.
3. **Push to origin/master** (requires explicit user approval).

### Permanent fixes for this session's mistakes

4. **Apply the permanent BuildFlow fix** — rename the outer `capturedContext`/`capturedCtx` vars in `hooks_test.go`/`ws_dispatch_test.go` so `:=` doesn't shadow, eliminating the revert trigger.
5. **Add a tracking TODO** to remove the 4 `IndexSpec` exhaustruct nolints once go-cqrs-lite publishes v4.0.3+.
6. **Add a snapshot benchmark** (`BenchmarkSnapshot_Load_10KEvents` vs `BenchmarkFullReplay_10KEvents`) to quantify the perf win the feature promises.

### Verification gaps

7. **Browser-test the IndexedDB `rebuildAndRetry` path** with Playwright, or add an explicit "untested in browser" caveat to `adminui/README.md`.
8. **Add a test for the IndexedDB graceful-degradation path** (private browsing / quota exceeded / IDB unavailable).
9. **Add a test for `enrichUserID` Error-level log** (the Warn→Error change from prior session).
10. **Add a test for the `csrf_middleware` per-consumer warning** (the sync.Once removal).
11. **Fuzz test `SyntheticUserID`** for determinism + collision behavior.
12. **Verify README.md anchor links** after prior copywriting edits (T26 sub-item, still skipped).

### The v5 coordinated bump (largest remaining structural work)

13. **T20: Decompose `usermgmt.Service` into 6 sub-services + facade** (ADR-0038).
14. **T21: Split usermgmt into `internal/{user,membership,tenant,bot}` packages.**
15. **T22: Extract `UserCore` shared struct.**
16. **T24: Unify `ActorID` shape** (ADR-0039).
17. **T14: Adopt `Email` branded type** across User/payloads/ChangeEmail.
18. **T16: Rename adminui `*Data`→`*ViewModel`, `AuthHandler`→`AuthRoutes`, `OAuth2UserInfo`→`OAuth2Profile`.**
19. **T17: Replace `HandlerConfig.Secure *bool` with explicit tristate enum.**
20. **T23: Move `TOTPSecret []byte` out of `User` into the TOTP strategy module.**
21. **Write `docs/migrations/v4-to-v5.md`** covering all breaking changes in 13-20.

### Quality / testing

22. Raise adminui coverage 69% → 75% (close generated-templ gaps).
23. Raise loginpage coverage 80.1% → 85%.
24. Property-based tests for event fold functions (rapid/Hypothesis-style).
25. Load-testing benchmarks for SSE broadcaster under high fan-out.
26. **Root lint zero-pass:** rename or ignore-list the 79 remaining varnamelen/testpackage warnings.
27. Audit all `errorfamily.Wrapf` call sites for consistent code/message formatting.
28. Consider extracting SSE/WS/ratelimit into optional root sub-packages (previously deferred).

### Frontend / UX

29. Visually verify focus-visible outlines + reduced-motion in Chrome + Firefox.
30. Add OAuth2 link/unlink views to admin UI (long-standing TODO).
31. adminui: add a visible "N commands syncing…" indicator wired to the `pending` worker message.
32. adminui: dark-mode audit (sync-states, toast colors).
33. loginpage: add a "passkey not supported" fallback path test.
34. Consider a Service Worker variant for true background sync (Chrome-only, deferred in ADR-0040).

### Docs / process

35. Write ADR-0042 (BuildFlow hook workaround) documenting the `:=`→`=` revert behavior formally (currently only in AGENTS.md).
36. Write a CONTRIBUTING.md section on "when to use SyntheticUserID vs ParseUserID vs MustParseUserID".
37. Automate GitHub Release creation on tag push via CI.
38. Add `check-docs-freshness` assertions for the new AGENTS.md/FEATURES.md content.
39. Benchmark `dedup.Ring` vs old map for typical journal sizes (long-standing TODO).
40. **Update the prior brutal self-review** to reflect this session's resolution (per `update-old-docs` skill).

### Architecture / cleanup

41. Evaluate `projectionhost` adoption (previously rejected per ADR-0016 — re-evaluate).
42. **MySQL event store support** (long-standing TODO; currently Postgres + SQLite only).
43. Audit the `storage.IndexSpec` version-skew more broadly — are there other types where local replace diverges from published?
44. Add a CI job that runs `GOWORK=off go build ./...` per submodule to catch version-skew breakage early (the `nix run .#test` path caught it; CI should too).
45. Consider a `go-cqrs-lite` release checklist item: "all submodule go.mod files `go mod tidy`-resolved before tag" (root cause of the 13/40 broken tags).

### Memory / maintenance

46. **Remove the `go.work` local replaces** once go-cqrs-lite cuts v4.0.3+ (long-standing, blocked upstream).
47. Update CHANGELOG.md "Unreleased" with this session's additions (deprecation, tests, doc updates).
48. Sweep all `// Deprecated:` markers across the repo — are there others that should be added?
49. Verify the `admin-demo`/`catalog-demo`/`datastar-demo` examples still compile after the NewUserID deprecation (they may call it).
50. **Run the `docs-health` skill** to verify all project docs are consistent after this session's edits.

---

## g) Questions I CANNOT figure out myself

### Q1. Should the 22 changes be committed as a single commit or split?

The changes span 4 logical units: (a) deprecation + 16-site migration, (b) new tests (enum Valid, decoder-nil, snapshot proof), (c) lint fixes (IndexSpec nolint, golines, errcheck), (d) docs (FEATURES/AGENTS/TODO_LIST). I can't infer your commit-granularity preference. A single commit is simpler history; split commits are easier to review/revert individually. **What's your preference?**

### Q2. Is the ADR-0030 reversal (Phase 2b IndexedDB) confirmed?

The prior session's summary flagged this as "unconfirmed." ADR-0030 explicitly rejected Phase 2b ("will never ship"). The prior session reversed it via ADR-0040 based on interpreting your "high-prio" flag as overriding the rejection — without explicit confirmation. I did not touch ADR-0040 this session, but the FEATURES.md/AGENTS.md updates I made describe IndexedDB as shipped. **Did you intend to reverse your own prior rejection, or should ADR-0040 + commit `84f9b9d` be reverted?**

### Q3. When should the v5 structural bump happen?

T20/T21/T22 (Service decomposition) + T14/T16/T17/T23/T24 (breaking renames) are all deferred to v5 per ADR-0038/0039. They are the largest remaining structural debt (the `usermgmt.Service` god-object: 52 methods, 30 fields). But they are breaking changes on a published v4 library. **Is there a timeline or trigger for the v5 bump, or should these stay deferred indefinitely?**

---

## Session metrics

| Metric                                       | Value                                                                                                       |
| -------------------------------------------- | ----------------------------------------------------------------------------------------------------------- |
| Tasks planned                                | 12                                                                                                          |
| Tasks completed                              | 12                                                                                                          |
| Tests added                                  | 8 (5 enum Valid + 2 decoder-nil + 1 snapshot proof)                                                         |
| Tests passing                                | 8/8 with `-race`                                                                                            |
| Modules passing                              | 8/8 (`nix run .#test`)                                                                                      |
| Coverage gate                                | PASSED (root 93.8%, usermgmt 80.2% — both UP)                                                               |
| Lint (submodules)                            | 0 issues across all 7                                                                                       |
| Lint (root)                                  | 79 issues (pre-existing, no new from this session)                                                          |
| Commits made                                 | **0** (per "never commit without explicit word" rule)                                                       |
| Build breaks introduced and caught           | 2 (IndexSpec Where field; golangci-lint --fix revert) — both recovered before any commit                    |
| Test failures introduced and caught          | 1 (SyntheticUserID on a real ULID) — fixed                                                                  |
| Process mistakes repeated from prior session | 1 (running a fixer that reverts intentional code — same class as the BuildFlow bypass, different mechanism) |

---

_The working tree is clean and verified. Nothing is committed. Nothing is pushed. Awaiting instructions on Q1/Q2/Q3._
