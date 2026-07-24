# Build Repair Session: Zero Pseudo-Versions + carrierStatus Bug + go-sse Wiring

**Date:** 2026-07-23 21:00
**Session Type:** Build/test failure triage and repair
**Starting State:** 4 buildflow steps failing (go-fix, govalid-generate, hierarchical-errors, test-race)
**Ending State:** BuildFlow 38/39 passing (1 skipped via config)

---

## What Was Broken (3 Distinct Root Causes)

### 1. Broken Zero Pseudo-Versions in Workspace go.mod Files

Three workspace modules had sibling module references pointing to the broken
`v4.0.0-00010101000000-000000000000` zero pseudo-version:

- **`examples/admin-demo/go.mod`**: 4 broken refs (adminui/v4, usermgmt/totp/v4, usermgmt/v4, root v4)
- **`examples/basic/go.mod`**: 1 broken ref (root v4)
- **`integration_test/go.mod`**: 5 broken refs (oauth2/v4, totp/v4, usermgmt/v4, webauthn/v4, root v4)

These poisoned the entire workspace module graph — `go build ./...` failed
with `invalid version: unknown revision 000000000000` across every root
module file, because Go's workspace resolver tried to resolve the broken
version from the remote proxy.

**Fix:** `go mod edit -require=...@v4.4.0` for each broken ref, then `go mod tidy`.

### 2. carrierStatus Broken by go-error-family v0.8.0 Structural Typing

go-error-family v0.8.0 added `HTTPStatus() int` as a method on `*Error`
(returning the `httpStatus` field, default 0). This made every
`errorfamily.Error` structurally satisfy the `HTTPStatusCarrier` interface
defined in `errors_status.go`.

The old `carrierStatus()` function was:

```go
carrier, ok := errors.AsType[HTTPStatusCarrier](err)
status := carrier.HTTPStatus()
if validHTTPStatus(status) { return status, true }
return http.StatusInternalServerError, true  // <-- BUG: 0 is "valid" range check but wrong semantically
```

When `httpStatus=0` (meaning "use family default"), `validHTTPStatus(0)`
returned false, so the function returned **500 Internal Server Error**
instead of falling through to `errorfamily.Classify(err).HTTPStatus()`.

This caused **69 test failures** across root + usermgmt: every Rejection
(→400), Conflict (→409), and Transient (→503) error was incorrectly mapped
to 500.

**Fix:** Rewrote `carrierStatus` to walk the error chain, skipping carriers
with `HTTPStatus()==0` (no override) and returning the first non-zero
carrier status. This preserves the original `WithHTTPStatus` semantics
even when an `errorfamily.Error` wrapper sits on top of an
`httpStatusError`.

### 3. Missing go-sse Dependency for WIP SSE/WS Broadcaster Refactor

A previous session had started refactoring SSE/WS broadcasters to use the
new `github.com/larsartmann/go-sse` package. Files modified:

- `sse_broadcaster.go`: Now embeds `*sse.Broadcaster[sse.Event]` instead of `*fanOut[SSEEvent]`
- `ws_broadcaster.go`: Now embeds `*sse.Broadcaster[string]` instead of `*fanOut[string]`
- `sse_store.go`: Now delegates `ReplayEvents` to `sse.Replay`
- `sse_event.go`: New file with type aliases to go-sse
- `constants.go`: Removed `ContentTypeSSE` (now lives in go-sse)

But `go-sse` was never added to `go.mod` or `go.work`, so `loginpage` and
`integration_test` (which import root transitively) failed with
`no required module provides package github.com/larsartmann/go-sse`.

**Fix:** Added `replace github.com/larsartmann/go-sse => /home/lars/projects/go-sse`
to `go.work`, then `go get github.com/larsartmann/go-sse` in root, then
`go mod tidy` on loginpage and integration_test.

---

## a) FULLY DONE

- [x] Fixed 10 broken zero pseudo-versions across 3 workspace go.mod files
- [x] Ran `go mod tidy` on all affected modules (admin-demo, basic, integration_test)
- [x] Diagnosed and fixed `carrierStatus` chain-walking bug in `errors_status.go`
- [x] Verified all 69 previously-failing tests now pass (root + usermgmt)
- [x] Added `go-sse` replace directive to `go.work`
- [x] Added `go-sse` require to root `go.mod`
- [x] Ran `go mod tidy` on loginpage and integration_test for go-sse resolution
- [x] Verified all 12 workspace modules build cleanly (`go build ./...`)
- [x] Verified all 9 test modules pass with `-race` detector
- [x] Verified BuildFlow passes: 38/39 steps green (1 skipped via config)
- [x] All changes auto-committed by the pre-commit hook (4 commits)

## b) PARTIALLY DONE

- **Lint/typecheck in GOWORK=off mode:** The lint step (`nix run .#lint`) sets
  `GOWORK=off` and runs `golangci-lint` per-module against published tags. The
  root module lint shows a typecheck error: `event_store_sse_test.go:273:
undefined: id.StreamRef`. The local go-cqrs-lite has `StreamRef` (renamed from
  `AggregateRef`), but published `id/v4@v4.0.3` only has `AggregateRef`. This is
  a **workspace-vs-published-version mismatch** that tests don't catch (workspace
  mode uses local replaces). BuildFlow's own linter doesn't run `GOWORK=off`
  per-module golangci-lint, so it passed, but `nix run .#lint` would fail.

## c) NOT STARTED

- Nothing from this session's scope was left unstarted.

## d) TOTALLY FUCKED UP

- **Nothing was irreversibly damaged.** No data loss, no broken git history.
- **First fix attempt for carrierStatus was incomplete:** The initial fix
  (simple `if status == 0 { return 0, false }`) failed the usermgmt tests
  because it didn't walk the chain — an outer `errorfamily.NewRejection(...)`
  wrapper with `httpStatus=0` would shadow an inner `httpStatusError` with
  `httpStatus=409`. Caught immediately by running usermgmt tests, then fixed
  with the chain-walking approach.

## e) WHAT WE SHOULD IMPROVE

1. **Session startup protocol:** I should have run `git status` + `git diff`
   FIRST to understand the working tree state. There were already uncommitted
   WIP changes (sse_broadcaster.go, ws_broadcaster.go, sse_store.go,
   constants.go) that I discovered only incidentally when debugging the go-sse
   missing dependency. A full `git diff --stat HEAD` at the start would have
   revealed this immediately.

2. **go-sse is not published:** The `go.work` replace for go-sse points to
   `/home/lars/projects/go-sse` which is **not a git repository** (no `.git`).
   It has no tags and no go.sum. This means the workspace cannot build without
   the local directory. This should be published or vendored.

3. **AGENTS.md not updated:** The go-sse dependency and the carrierStatus
   chain-walking semantics are important gotchas that belong in AGENTS.md.
   Specifically: "go-error-family v0.8.0+ adds HTTPStatus() to *Error, making
   it structurally satisfy HTTPStatusCarrier — carrierStatus must skip zero
   values and walk the chain."

4. **Lint typecheck gap:** The lint target uses `GOWORK=off` which resolves
   against published tags, not local replaces. When local go-cqrs-lite has API
   changes (StreamRef rename), lint fails but tests pass. This is a known
   workspace-vs-published mismatch that needs either a tag release or lint
   tolerance.

5. **I didn't verify the go-sse refactor was complete:** The WIP changes to
   sse_broadcaster.go/ws_broadcaster.go removed the `fanOut` type entirely.
   I verified `fanOut` has no remaining references, but I didn't review
   whether the refactor was semantically correct (e.g., does
   `sse.Broadcaster` have the same backpressure/drop semantics as the old
   `fanOut`?).

6. **No unit test for carrierStatus specifically:** The fix was verified by
   integration tests (ExampleStructuredError, TestHandlers_Register_DuplicateEmail,
   etc.), but there's no isolated test that asserts the chain-walking behavior
   directly. A regression test like `TestCarrierStatus_WalksPastZeroCarrier`
   would prevent future breakage.

---

## f) Next Steps (Up to 50)

### Critical / Immediate

1. Publish `go-sse` as a git repo with at least one tag, or add it to go.work `use` block
2. Fix the lint typecheck: either tag go-cqrs-lite id/v4 with the StreamRef rename, or update test to use `AggregateRef`
3. Update AGENTS.md with go-sse dependency note and carrierStatus chain-walking gotcha
4. Add a dedicated unit test for `carrierStatus` chain-walking behavior
5. Run `nix run .#lint` to confirm the only remaining issue is the StreamRef typecheck

### go-sse Extraction

6. Review the SSE/WS broadcaster refactor for semantic equivalence with old fanOut
7. Check if `sse.Broadcaster` has configurable buffer capacity (old fanOut default was 64)
8. Verify `sse.Broadcaster.Subscribe/Unsubscribe` channel lifecycle matches old API
9. Check if `sse.Replay` has the same error wrapping as the old inline implementation
10. Remove dead code: old `fanOut` type and its tests if still present in git history
11. Add go-sse to the `go.work` `use()` block if it should be a workspace member

### Dependency Hygiene

12. Audit ALL workspace go.mod files for any remaining zero pseudo-versions
13. Add a CI check that rejects `00010101000000-000000000000` in any go.mod
14. Consider a `go work sync` to align all module versions
15. Check if `go.work.sum` needs updating after the go-sse addition
16. Verify `nix run .#coverage` and `nix run .#coverage-gate` still pass

### Testing

17. Run the full examples test suite (`examples/*/go test`)
18. Verify `nix run .#test` passes (it may run differently than per-module tests)
19. Add a test that verifies `WithHTTPStatus` works when wrapped by `errorfamily.NewRejection`
20. Add a test for MapError with nil error (should return 500)
21. Add a test for MapError with unknown error type (should return 503 via Transient default)

### Documentation

22. Document the go-sse extraction in a CHANGELOG entry
23. Update the architecture section in AGENTS.md to mention go-sse
24. Add go-sse to the dependency direction diagram
25. Create an ADR for the SSE/WS broadcaster extraction to go-sse

### Pre-existing Issues Noticed

26. The `id.StreamRef` vs `id.AggregateRef` rename needs to be resolved in lint mode
27. Check if other test files use `StreamRef` in ways that would break with published tags
28. The `go.work` replace comment block should mention go-sse as a new local replace
29. Verify the pre-commit hook (buildflow --build-mode pre-commit) still works with go-sse
30. Check if cqrs-lint needs updating for the go-sse package

### Broader Quality

31. Run `nix flake check` to verify the flake is healthy
32. Run the hierarchical-errors skill to check for error handling modernization
33. Review whether the carrierStatus fix should be upstreamed to go-error-family docs
34. Consider adding a `go mod verify` step to CI
35. Audit all `replace` directives in go.work for relevance (some may be stale)
36. Check if the datastar-demo example needs go-sse updates
37. Verify the admin-demo example builds standalone (GOWORK=off)
38. Check if the catalog-demo example is affected by any of these changes
39. Run the brutal-self-review skill on the carrierStatus fix
40. Consider whether the chain-walking in carrierStatus should be capped (depth limit)
41. Add a benchmark for the new chain-walking carrierStatus (old was O(1), new is O(n))
42. Check if `errors.AsType` already walks the chain (if so, manual walking is redundant)
43. Review the go-sse API surface for completeness vs the old cqrs-htmx SSE API
44. Check if SSEEvent alias is still needed or if consumers should import go-sse directly
45. Verify the sync-worker.js / sync-client.js are unaffected by the broadcaster refactor
46. Run `nix fmt` to ensure formatting is clean
47. Check if the coverage gate thresholds (root 90%, usermgmt 74%) are still met
48. Consider adding go-sse to the integration_test module explicitly
49. Review whether loginpage needs direct go-sse access or only transitive
50. Plan the go-sse v0.1.0 publication (git init, tag, module path finalization)

---

## g) Questions

1. **Should `go-sse` be a workspace member (added to `go.work` `use()` block) or
   stay as a replace-only dependency?** It currently lives at
   `/home/lars/projects/go-sse` with no `.git` directory. If it should be
   published, it needs `git init` + at least one tag. If it stays local-only,
   the replace is sufficient but CI/CD will break.

2. **The `id.StreamRef` vs `id.AggregateRef` lint failure: should I update the
   test files to use `AggregateRef` (the published name), or should we cut a
   new `id/v4` tag with the `StreamRef` rename?** The local go-cqrs-lite has
   `StreamRef` as the primary name with `AggregateRef` as a deprecated alias.
   Published `id/v4@v4.0.3` only has `AggregateRef`.

3. **The SSE/WS broadcaster refactor was uncommitted WIP when I started. Should
   I treat it as complete and ship it, or does it need a deeper review for
   semantic equivalence (backpressure, drop policy, buffer sizes) before the
   next release?** The old `fanOut` had explicit documentation about drop
   policy that the new `sse.Broadcaster` may or may not preserve.
