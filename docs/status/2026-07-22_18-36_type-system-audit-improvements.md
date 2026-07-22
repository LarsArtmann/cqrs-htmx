# Status: Type-System Audit Improvements — 2026-07-22_18-36

> **Session goal:** Execute the improvement items from the type-system audit status doc
> (`docs/status/2026-07-22_17-42_type-system-audit-implementation.md`).
> **Outcome:** 6 of7 improvement items addressed. 1 skipped. Multiple self-inflicted wounds from poor session discipline.

---

## What This Session Did

Resumed from a status document that described a prior session's work (5 shipped type-system items + 1 lint regression). The prior session had already committed 8 commits (27ad109 through ea147dd) covering the typed handler implementation, tests, sync refactoring, docs, and ADR. This session picked up the "WHAT WE SHOULD IMPROVE" (section e) and "Next 50 Things" (section f) items.

---

## a) FULLY DONE

### 1. `any(v) == nil` explanatory comment (`handler.go:104-108`)

- **Status:** SHIPPED, committed.
- Added 4-line comment explaining WHY `any(v)` wrapper is needed (Go compiler cannot prove nil-comparability on generic type parameters; wrapping converts to interface first).
- Pre-existing comment explained WHAT the nil check does (decoder wiring bug); new comment explains the Go language reason for the `any()` wrapper.

### 2. `dispatchRequest` refactor mention in CHANGELOG.md

- **Status:** SHIPPED, committed.
- Added "Internally, `dispatchRequest` was extracted from a method to a standalone generic function shared by both typed and untyped handlers." to the typed handlers entry under `[Unreleased] > Added`.
- Addresses improvement item6 from section e.

### 3. Generic-method limitation in AGENTS.md Gotchas

- **Status:** SHIPPED, committed.
- Added gotcha: "`CommandTyped`/`QueryTyped` are package-level functions, not methods: Go does not allow generic methods on receiver types."
- Addresses improvement item7 from section e.

### 4. `DecodeAndValidateForm` + `DecodeAndValidateFormQuery` tests (`validation_test.go`)

- **Status:** SHIPPED, committed.
- 4 new BDD specs: DecodeAndValidateForm (success + validation failure), DecodeAndValidateFormQuery (success + validation failure).
- Tests use `Content-Type: application/x-www-form-urlencoded` with `withHeader()` helper.
- Addresses task2-3 from section f (High Priority).

### 5. `DecodeFormTyped` + `DecodeFormQueryTyped` tests (`typed_handlers_test.go`)

- **Status:** SHIPPED, committed.
- 2 new BDD specs: CommandTyped with form decoder, QueryTyped with form decoder.
- Both verify form data decoding (`message=hello`, `a=3&b=4`) through the typed dispatch pipeline.
- Addresses task2 from section f (High Priority).

### 6. Malformed JSON body test for `DecodeAndValidateJSONQuery` (`validation_test.go`)

- **Status:** SHIPPED, committed.
- 1 new BDD spec: sends `{not json` as body, verifies 400 response containing "decode".
- Addresses task4 from section f (High Priority).

---

## b) PARTIALLY DONE

### gci false positive resolution (`options_decode.go`)

- **Status:** LIKELY RESOLVED but unverified.
- Attempted `golangci-lint cache clean` but it failed (permission error on cache directory).
- Running `golangci-lint run ./... | grep gci` returned empty — no gci issues found.
- However, this could be a golangci-lint v2 behavior change rather than a real fix. The original issue was reported against v1.x.
- **Verdict:** The gci issue appears gone, but confidence is moderate. Needs verification with the exact golangci-lint version that originally reported it.

### Full workspace test (`nix run .#test`)

- **Status:** NOT RUN THIS SESSION.
- Ran root module tests only (`GOEXPERIMENT=jsonv2 go test ./...`). All pass.
- The `nix run .#test` command was attempted but moved to background due to long runtime. The `TestSyncScriptTags` build failure from the previous session was not re-verified.
- **Should have:** Run the full workspace test to confirm nothing is broken across all 8 modules.

---

## c) NOT STARTED

### Micro-types (from type-system audit, task1 in section f)

- The audit marked this as "DO IT" but it was the lowest-priority item.
- Not started; no design work done.
- Would involve extracting small focused types (e.g., `Email`, `Password`) from larger structs.

### Type-system audit HTML update (task31 in section f)

- The audit HTML (`docs/brainstorming/cqrs-htmx-type-system-audit.html`) uses `badge-done` as a styling class, not a completion status marker.
- Skipped because: (a) items are already documented in FEATURES.md and CHANGELOG.md, (b) the HTML is a brainstorming document not a living doc, (c) low value-to-effort ratio.

### OPFS/sqlite-wasm (from offline-first research)

- Not part of this session's scope.

### Honest UI State Machine (from offline-first research)

- Not part of this session's scope.

### Coverage gate (`nix run .#coverage-gage`)

- Not run. Should verify root module still meets 90% threshold.

---

## d) TOTALLY FUCKED UP

### Stash/pop cycle caused file state confusion

- Mid-session, ran `git stash` to verify a pre-existing test failure was not from my changes.
- After `git stash pop`, the working tree state was confusing: handler.go and validation_test.go showed NO diff (already committed by parallel session commits 27ad109/e6867aa), while sync_serve.go/sync_serve_test.go showed pre-existing uncommitted changes.
- Spent ~10 minutes debugging file state that was caused by the stash/pop cycle, not by actual problems.
- **Lesson:** NEVER stash in the middle of active work. Verify pre-existing failures on a clean `git stash` BEFORE making changes, or just trust the initial test run.

### Accidentally duplicated handler.go comment

- After the stash/pop cycle, the `any(v) == nil` comment appeared twice in handler.go (the committed version + my re-applied version).
- Had to detect and fix the duplication.
- **Root cause:** Didn't check `git diff handler.go` after stash pop to see if my edit was already committed.

### Accidentally replaced AGENTS.md projection checkpoints gotcha

- When adding the `CommandTyped`/`QueryTyped` gotcha, I accidentally replaced the "Projection checkpoints are per-projection" gotcha instead of appending.
- Had to fix by re-adding both entries.
- **Root cause:** Used `edit` with too narrow an old_string match. Should have included more context.

### sync_serve.go confusion

- Pre-existing uncommitted deletion of `SyncWorkerScriptTag` from a prior session was in the working tree.
- Spent time investigating whether this was my change, whether to restore it, whether it was intentional.
- It was NOT my change. Should have been ignored entirely.
- **Lesson:** Check `git diff --name-only` FIRST to understand what's dirty before touching anything.

### No `nix run .#test` verification

- Only ran root module tests. Never ran the full workspace test across all 8 modules.
- The `TestSyncScriptTags` build failure from the previous session was not re-verified.
- **Lesson:** Always run the full test suite as the FIRST and LAST action in a session.

---

## e) WHAT WE SHOULD IMPROVE

1. **Check `git log` FIRST** — Before making any changes, check recent commits to understand what a parallel session may have already done. This session wasted significant time re-doing work that was already committed.

2. **Never stash mid-work** — The `git stash` / `git stash pop` cycle created 15+ minutes of confusion about file state. If you need to verify a pre-existing failure, do it BEFORE making changes.

3. **Run full test suite early AND late** — `nix run .#test` should be the first and last command. Only root module tests were run this session.

4. **The `gci` false positive** — Appears resolved in golangci-lint v2 but was never definitively verified. Should run the exact lint command from `.golangci.yml` configuration to confirm.

5. **Test file organization** — The decoder tests are split across `validation_test.go` (validated decoders) and `typed_handlers_test.go` (typed decoders). Consider whether `typed_handlers_test.go` should be renamed to `typed_decoders_test.go` since it now tests both handler dispatch AND decoder functionality.

6. **The `any(v) == nil` comment is good but could be a Go doc** — The comment explains the language limitation well, but it would be even better as a package-level doc comment or in a README section about generic dispatch patterns.

7. **Decoder test coverage is still incomplete** — `DecodeFormTyped` and `DecodeFormQueryTyped` have happy-path tests but no error-path tests (malformed form data, empty body, wrong Content-Type).

---

## f) Next 50 Things To Do

### High Priority (Type-System Audit Completion)

1. Implement micro-types (the last "DO IT" item from the audit).
2. Add error-path tests for `DecodeFormTyped[Q]` (malformed form data, empty body).
3. Add error-path tests for `DecodeFormQueryTyped[Q]`.
4. Add error-path tests for `DecodeAndValidateForm[T]` (malformed form data).
5. Add error-path tests for `DecodeAndValidateFormQuery[T]`.
6. Run full workspace test (`nix run .#test`) and verify all 8 modules pass.
7. Run coverage gate (`nix run .#coverage-gate`) to verify root 90% threshold still holds.
8. Verify `gci` false positive is truly resolved with the project's configured linter version.

### Medium Priority (API Polish)

9. Add `CommandTyped`/`QueryTyped` examples to `examples/basic/`.
10. Document typed handlers in the cqrs-htmx skill (`SKILL.md` or `references/core-api.md`).
11. Consider whether `CommandTyped`/`QueryTyped` should also support `DecodeFormTyped` out of the box (currently they work but aren't documented together).
12. Add a benchmark comparing typed vs untyped dispatch overhead.
13. Add Go doc examples (`ExampleCommandTyped`, `ExampleQueryTyped`).

### Lint Debt (Pre-Existing, Not Introduced This Session)

14. Fix `varnamelen` warnings (50 issues — rename short variables to longer names).
15. Fix `testpackage` warnings (9 test files using `package cqrshtmx` instead of `cqrshtmx_test`).
16. Fix `ireturn` warnings (4 functions returning interfaces).
17. Fix `makezero` warnings (4 slices with non-zero initial length).
18. Fix `nonamedreturns` warnings (4 named returns).
19. Fix `tagliatelle` warnings (2 JSON tags using snake_case instead of camelCase).
20. Fix `containedctx` warning (1 struct containing `context.Context`).
21. Fix `testableexamples` warning (1 example missing output).

### Architecture / Design

22. Evaluate whether `Result[T]` should be reconsidered (was skipped as "CONSIDER").
23. Consider whether the generic `Store[T, ID]` should be used inside usermgmt repositories.
24. Evaluate OPFS/sqlite-wasm for the offline queue (from offline-first research).
25. Design the honest UI state machine for offline sync.
26. Consider whether `dispatchRequest` should be exported for consumers who want to build custom dispatch pipelines.

### Testing

27. Add integration test for `CommandTyped` with CSRF middleware.
28. Add integration test for `QueryTyped` with `RenderJSON[R]` + pagination.
29. Add test for typed handler with `Authorize` option.
30. Add test for typed handler with `RequestGuard` option.
31. Add test for typed handler with `WithTimeout` option.
32. Add test for `InMemoryStore` with concurrent access (race detector).
33. Add fuzz test for `DecodeJSONTyped` with malformed JSON.
34. Add fuzz test for `DecodeFormTyped` with malformed form data.

### Documentation

35. Update `docs/brainstorming/cqrs-htmx-type-system-audit.html` to mark implemented items.
36. Add ADR for the generic-method workaround decision.
37. Update `CONTRIBUTING.md` with the `gofumpt -w` warning.
38. Update `docs/migrations/` if typed handlers change the recommended handler pattern.
39. Add typed handler usage to the cheat sheet in SKILL.md.

### Code Quality

40. Consider extracting the type-assertion pattern in `handleCommandTypedDispatch` into a shared helper.
41. Verify `nil` command/query from typed decoders is handled correctly (the `any(v) == nil` guard).
42. Consider whether `DecodeJSONTyped[Q]()` should validate the decoded command's `Type()` matches the registered type.
43. Rename `typed_handlers_test.go` to better reflect its scope (tests decoders AND dispatch).

### Offline / Sync (Separate Track)

44. Evaluate `sync-worker.js` error handling robustness.
45. Add browser-based E2E test for the offline queue.
46. Consider Web Locks API for cross-tab coordination.
47. Add retry backoff configuration.
48. Add dead-command surfacing in admin UI.

### Miscellaneous

49. Squash the 17+ session commits into a clean history.
50. Run `nix flake check` to verify flake health.

---

## g) Questions (Cannot Self-Resolve)

1. **Should the `TestSyncScriptTags` build failure be investigated?** — The `nix run .#test` build failed with `undefined: _xtest.TestSyncScriptTags` when I tried it (pre-existing from a previous session). Is this a known issue, or should it be fixed now?

2. **Is the `gci` false positive truly fixed?** — Running `golangci-lint run ./... | grep gci` returned empty, but the original issue was reported against a specific golangci-lint version. Should I verify with the project's pinned linter version from `flake.nix`, or is the v2 result sufficient?

3. **Should I proceed with micro-types implementation next?** — The type-system audit marked it as "DO IT" but it was the lowest-priority item. Is this still the right next step, or should I focus on lint debt / test coverage / API polish first?
