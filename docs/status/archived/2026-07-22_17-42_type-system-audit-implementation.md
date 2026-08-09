# Status: Type-System Audit Implementation — 2026-07-22

> **Session goal:** Implement the remaining "DO IT" items from the type-system audit
> (`docs/brainstorming/cqrs-htmx-type-system-audit.html`).
> **Outcome:** 4 of 5 items shipped and tested. 1 item (micro-types) not started. ~~1 lint regression introduced and not fully resolved.~~
> **Update 2026-07-28:** The lint regression (a `gci` false positive on `options_decode.go`) was
> resolved in the followup sessions the same day (`18-36`, `19-05`). All typed handler items
> shipped in v4.5.0 as `FULLY_FUNCTIONAL`.

---

## What This Session Did

The session resumed from an interrupted previous session. The prior session had:

- Analyzed all 5 files in `docs/brainstorming/*` to determine which proposals were already implemented.
- Completed: generic context keys, `DecodeAndValidateJSON[T]`, generic `Store[T, ID]`.
- Left broken: `CommandTyped`/`QueryTyped` (written as methods, which Go rejects).

This session:

1. Fixed the compilation errors from the previous session.
2. Added comprehensive tests for typed handlers.
3. Ran the full test suite.
4. Updated FEATURES.md and CHANGELOG.md.
5. Fixed some new lint issues introduced by the changes.

---

## a) FULLY DONE

### 1. Generic Context Keys (`context.go`)

- **Status:** SHIPPED, committed, tested.
- Replaced sentinel struct keys (`userIDKey{}`, `correlationIDKey{}`, etc.) with a generic `contextKey[T]` helper.
- Internal refactor; public API (`UserIDFromContext`, `WithUserID`, etc.) unchanged.
- Added `//nolint:gochecknoglobals` for the key instances (standard Go pattern for context keys).

### 2. `DecodeAndValidateJSON[T]` + Typed Decoders (`options_decode.go`)

- **Status:** SHIPPED, committed, tested.
- `Validatable` interface constraint.
- 4 validating decoders: `DecodeAndValidateJSON[T]`, `DecodeAndValidateJSONQuery[T]`, `DecodeAndValidateForm[T]`, `DecodeAndValidateFormQuery[T]`.
- 4 typed decoders: `DecodeJSONTyped[Q]`, `DecodeFormTyped[Q]`, `DecodeJSONQueryTyped[Q]`, `DecodeFormQueryTyped[Q]`.
- Validation errors wrapped with `errorfamily.Wrapf(err, event.Rejection, ...)` per the project's no-stdlib-error rule.

### 3. Generic `Store[T, ID]` (`usermgmt/store.go`)

- **Status:** SHIPPED, committed, tested.
- `Store[T, ID]` interface with `Get`, `Save`, `Delete`, `All` methods.
- `InMemoryStore[T, ID]` implementation with `idOf` extractor function.
- Tests in `usermgmt/store_test.go`.

### 4. `CommandTyped[Q]` / `QueryTyped[Q, R]` (`app.go`, `handler.go`)

- **Status:** SHIPPED, committed, tested.
- Package-level generic functions (Go forbids generic methods on receiver types).
- `CommandTyped[Q command.Command](a *App, cmdType command.Type, opts ...HandlerOption) http.HandlerFunc`
- `QueryTyped[Q query.Query, R any](a *App, qryType query.Type, opts ...HandlerOption) http.HandlerFunc`
- Both share the full dispatch pipeline (authz, CSRF, method guard, timeout, error handling, hooks) via the refactored standalone `dispatchRequest[Q, R]` function.
- Type mismatches surfaced as 400 Bad Request (not panics).
- Tests in `typed_handlers_test.go`: 4 BDD specs covering success paths and type-mismatch error paths.

### 5. `dispatchRequest` Refactor (`handler.go`)

- **Status:** SHIPPED, committed, tested.
- Extracted from a method to a standalone generic function `dispatchRequest[Q, R any]`.
- Both typed and untyped handlers now share the same pipeline.
- Fixed `v == nil` → `any(v) == nil` for generic type comparison.

### 6. Documentation Updates

- **Status:** SHIPPED, committed.
- FEATURES.md: Added "Typed Command/Query Dispatch" row and "Typed Decoding" row.
- CHANGELOG.md: Added entry under `[Unreleased] > Added`.
- Updated date to 2026-07-22.

### 7. Full Test Suite

- **Status:** PASSING (all 8 modules).
- `nix run .#test` — all green: root (4s), usermgmt (21s), totp (1s), webauthn (1s), oauth2 (1s), adminui (4s), loginpage (1s), integration_test (1s).

---

## b) PARTIALLY DONE

### Lint Cleanup

- **Before this session:** 111 lint issues in root module (from the initial lint run that included the broken typed handlers).
- **After this session:** ~76 lint issues in root module.
- **What was fixed:** `gochecknoglobals` (5→0 via nolint), `nolintlint` (2→0 by removing stale `//nolint:contextcheck`), `wrapcheck` (4→0 via `errorfamily.Wrapf`), `canonicalheader` (24→0 — these were pre-existing but apparently auto-fixed by `nix fmt`), `nlreturn` (1→0).
- **What remains:** 1 `gci` formatting issue on `options_decode.go:97` that appears to be a false positive (gci standalone shows no diff; `golangci-lint fmt --diff` shows no changes). The remaining ~75 issues are ALL pre-existing (`varnamelen`, `testpackage`, `ireturn`, `makezero`, `nonamedreturns`, `tagliatelle`, `containedctx`, `testableexamples`).

---

## c) NOT STARTED

### Micro-types (from type-system audit)

- The audit marked this as "DO IT" but it was the lowest-priority item.
- Not started; no design work done.
- Would involve extracting small focused types (e.g., `Email`, `Password`) from larger structs.

### OPFS/sqlite-wasm (from offline-first research)

- Not part of this session's scope.
- The SharedWorker + IndexedDB queue exists; OPFS is the next evolution.

### Honest UI State Machine (from offline-first research)

- Not part of this session's scope.

---

## d) TOTALLY FUCKED UP

### `gofumpt -w` Corrupted adminui Assets

- Running `GOEXPERIMENT=jsonv2 gofumpt -w` on `options_decode.go` reformatted `.js` and `.templ` files across adminui, deleting ~960 lines of JavaScript and corrupting templ output.
- **Impact:** Required `git restore` to undo. Lost the manual formatting fix to `options_decode.go` in the process.
- **Root cause:** `gofumpt -w` with a file path doesn't respect `.golangci.yml` exclusions; it processes all Go files in the directory tree.
- **Lesson:** NEVER run `gofumpt -w` directly on a single file in this project — it cascades to sibling files. Use `nix fmt` (which respects exclusions) or `golangci-lint fmt`.

### gci Lint Issue (Unresolved)

- `options_decode.go` reports a `gci` formatting issue that no tool can reproduce or fix.
- `gci diff` (standalone) shows no diff.
- `golangci-lint fmt --diff` shows no changes.
- Likely a stale cache or gci/golangci-lint integration bug.
- **Severity:** Cosmetic; doesn't affect build or tests.

### Commit Hygiene

- The session produced 17 commits (from the prior + this session) with generic auto-generated messages like "refactor(core): restructure application initialization and context handling" that don't explain WHY changes were made.
- Per the project's commit quality rules, these should have been squashed or amended with descriptive messages.

---

## e) WHAT WE SHOULD IMPROVE

1. **Never run `gofumpt -w` directly** — it ignores `.golangci.yml` exclusions and corrupts non-Go files in the directory. Use `nix fmt`.
2. **Verify lint changes are actually fixable before attempting** — the `gci` false positive wasted time.
3. **The `CommandTyped`/`QueryTyped` API ergonomics** — they're package-level functions, not methods, so the call site is `cqrshtmx.CommandTyped[Q](app, ...)` instead of `app.CommandTyped[Q](...)`. This is a Go language limitation but it's a worse developer experience. Consider whether a builder pattern or option-based approach would be more ergonomic.
4. **Test coverage for typed decoders** — `DecodeFormTyped`, `DecodeFormQueryTyped`, `DecodeAndValidateForm`, `DecodeAndValidateFormQuery` have no direct tests (only `DecodeJSONTyped` and `DecodeJSONQueryTyped` are tested via the typed handler tests).
5. **The `dispatchRequest` generic function uses `any(v) == nil`** for the nil check — this works but is non-obvious. A comment explaining why would help.
6. **CHANGELOG entry should mention the `dispatchRequest` refactor** — it's a significant internal change that affects anyone who has forked the handler pipeline.
7. **AGENTS.md should document the `CommandTyped`/`QueryTyped` functions** — the Gotchas section should mention that Go doesn't allow generic methods, so these are package-level functions.

---

## f) Next 50 Things To Do

### High Priority (Type-System Audit Completion)

1. Implement micro-types (the last "DO IT" item from the audit).
2. Add tests for `DecodeFormTyped[Q]` and `DecodeFormQueryTyped[Q]`.
3. Add tests for `DecodeAndValidateForm[T]` and `DecodeAndValidateFormQuery[T]`.
4. Add tests for `DecodeAndValidateJSONQuery[T]` with an invalid body.
5. Resolve the `gci` false positive on `options_decode.go` (try clearing golangci-lint cache).

### Medium Priority (API Polish)

6. Add `CommandTyped`/`QueryTyped` examples to `examples/basic/`.
7. Document typed handlers in the cqrs-htmx skill (`SKILL.md` or `references/core-api.md`).
8. Update AGENTS.md with the generic-method limitation note.
9. Consider whether `CommandTyped`/`QueryTyped` should also support `DecodeFormTyped` out of the box (currently they work but aren't documented together).
10. Add a benchmark comparing typed vs untyped dispatch overhead.

### Lint Debt (Pre-Existing, Not Introduced This Session)

11. Fix `varnamelen` warnings (50 issues — rename short variables to longer names).
12. Fix `testpackage` warnings (9 test files using `package cqrshtmx` instead of `cqrshtmx_test`).
13. Fix `ireturn` warnings (4 functions returning interfaces).
14. Fix `makezero` warnings (4 slices with non-zero initial length).
15. Fix `nonamedreturns` warnings (4 named returns).
16. Fix `tagliatelle` warnings (2 JSON tags using snake_case instead of camelCase).
17. Fix `containedctx` warning (1 struct containing `context.Context`).
18. Fix `testableexamples` warning (1 example missing output).

### Architecture / Design

19. Evaluate whether `Result[T]` should be reconsidered (was skipped as "CONSIDER").
20. Consider whether the generic `Store[T, ID]` should be used inside usermgmt repositories.
21. Evaluate OPFS/sqlite-wasm for the offline queue (from offline-first research).
22. Design the honest UI state machine for offline sync.
23. Consider whether `dispatchRequest` should be exported for consumers who want to build custom dispatch pipelines.

### Testing

24. Add integration test for `CommandTyped` with CSRF middleware.
25. Add integration test for `QueryTyped` with `RenderJSON[R]` + pagination.
26. Add test for typed handler with `Authorize` option.
27. Add test for typed handler with `RequestGuard` option.
28. Add test for typed handler with `WithTimeout` option.
29. Add test for `InMemoryStore` with concurrent access (race detector).
30. Add fuzz test for `DecodeJSONTyped` with malformed JSON.

### Documentation

31. Update `docs/brainstorming/cqrs-htmx-type-system-audit.html` to mark implemented items.
32. Add ADR for the generic-method workaround decision.
33. Update `CONTRIBUTING.md` with the `gofumpt -w` warning.
34. Update `docs/migrations/` if typed handlers change the recommended handler pattern.
35. Add typed handler usage to the cheat sheet in SKILL.md.

### Code Quality

36. Add `// why` comment on the `any(v) == nil` check in `dispatchRequest`.
37. Consider extracting the type-assertion pattern in `handleCommandTypedDispatch` into a shared helper.
38. Verify `nil` command/query from typed decoders is handled correctly (the `any(v) == nil` guard).
39. Consider whether `DecodeJSONTyped[Q]()` should validate the decoded command's `Type()` matches the registered type.
40. Add Go doc examples (`ExampleCommandTyped`, `ExampleQueryTyped`).

### Offline / Sync (Separate Track)

41. Evaluate `sync-worker.js` error handling robustness.
42. Add browser-based E2E test for the offline queue.
43. Consider Web Locks API for cross-tab coordination.
44. Add retry backoff configuration.
45. Add dead-command surfacing in admin UI.

### Miscellaneous

46. Squash the 17 session commits into a clean history.
47. Run `nix flake check` to verify flake health.
48. Run coverage gate (`nix run .#coverage-gate`) to verify coverage didn't drop.
49. Consider adding `CommandTypedMust` / `QueryTypedMust` convenience functions (panic on error).
50. Review whether the standalone `dispatchRequest` function should be unexported or available for testing.

---

## g) Questions I Cannot Answer Myself

1. **Should `CommandTyped`/`QueryTyped` use a different API shape?** The current `cqrshtmx.CommandTyped[Q](app, type, opts...)` is functional but breaks the `app.Command(type, opts...)` pattern. An alternative is `app.Command(type, opts...)` with a `WithTypeSafety[Q]()` option, but Go's generic constraints make this awkward. Is the package-level function acceptable, or should we explore a builder-pattern alternative?

2. **Should the `gci` lint warning on `options_decode.go` block commits?** It appears to be a false positive (gci standalone shows no diff, `golangci-lint fmt` shows no changes). The pre-commit hook runs `buildflow` which may or may not catch this. Should we add a `//nolint:gci` directive or investigate the root cause?

3. **Should we squash the 17 session commits before pushing?** The current commit messages are auto-generated and don't follow the project's quality bar (they don't explain WHY). The alternative is leaving them as-is since the tree is clean and tests pass.
