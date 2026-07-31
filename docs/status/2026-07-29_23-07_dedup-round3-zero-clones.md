# Deduplication Round 3: Zero Clones Achieved

**Date:** 2026-07-29 23:07
**Session scope:** `art-dupl --type-aware --sort total-tokens -t 3` → eliminate all clone groups → verify

---

## a) FULLY DONE

### 3 clone groups eliminated (0 remaining)

| #   | Clone                                                                                           | Files                                                                                                              | Fix                                                                                                                                                                                                                                                                                                                                                                           |
| --- | ----------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | `streamPathValues(r)` called redundantly before `loadStreamFromRequest`                         | `dashboardui/handlers_aggregates.go`, `dashboardui/handlers_timetravel.go`                                         | Deleted `streamPathValues` function entirely. Derived `streamType`/`streamID` from the `id.StreamRef` already returned by `loadStreamFromRequest`. Extracted `streamTitlePath(ref)` helper to centralize the `"type/truncated-id"` title construction.                                                                                                                        |
| 2   | Duplicate `readBody` + error-wrap in `decodeJSONBody`/`decodeFormBody`                          | `decoder.go` (root)                                                                                                | Extracted generic `readBodyForDecode[T any](r, maxBodySize, errCode)` helper that encapsulates the read + zero-value + error-wrap pattern. Both decoders now call it with their unique error code.                                                                                                                                                                            |
| 3   | Duplicate `requireUser` (package func) vs `h.currentUser` (method) + repeated `PathValue` guard | `usermgmt/http.go`, `usermgmt/credential_http.go`, `usermgmt/oauth2_http.go`, `usermgmt/verification_totp_http.go` | Deleted `requireUser` package function (was byte-identical to `h.currentUser`). All 3 call sites migrated to `h.currentUser`. Extracted `requirePathValue(w, r, key, errMsg)` for the 3 repeated `r.PathValue + empty check + writeError` pattern. Further extracted `h.currentUserWithPathValue(w, r, key, errMsg)` for the 2 handlers needing both auth + path-value guard. |

### Verification

- `art-dupl --type-aware --sort total-tokens -t 3` → **0 clone groups** (was 3)
- `go build ./...` → clean
- `go test` root, dashboardui, usermgmt → all PASS (without `-race`)
- `gofmt -l` on all 8 changed files → clean (after fixing one missed blank line)

### Commits (auto-commit daemon)

- `ebe9c58` — refactor(dashboardui): restructure handlers and decoder for improved consistency
- `b2a6040` — refactor(handlers): standardize HTTP handler responses across dashboard and user management

---

## b) PARTIALLY DONE

### Full workspace test suite

Ran root, dashboardui, and usermgmt individually (the 3 affected modules). Did NOT run all 15 workspace modules (adminui, loginpage, dashboardui, identity-model, totp, webauthn, oauth2, integration_test, examples/*). The changes are internal to the 3 modules tested, but downstream consumers (adminui, loginpage, dashboardui) depend on usermgmt and should be verified.

### Lint

Did NOT run `golangci-lint run` after changes. The pre-commit hook runs `buildflow` which includes linting, but I did not manually verify lint passes on all changed modules. This is a gap — new helpers like `requirePathValue` and `currentUserWithPathValue` may trigger linter rules (e.g., `wrapcheck`, `exhaustruct`).

---

## c) NOT STARTED

- CHANGELOG.md entry for this dedup session (project convention: completed work → CHANGELOG, not TODO_LIST)
- `dedup-acceptance.md` — not needed since we reached zero clones, but the skill recommends it when accepting clones
- Coverage gate verification (`nix run .#coverage-gate`) — not run
- Pre-existing data race investigation (see section d)

---

## d) TOTALLY FUCKED UP

### gofmt violation I introduced and missed

I added `streamTitlePath` to `dashboardui/handlers.go` without a trailing blank line before the next function's doc comment. `gofmt -l` caught it — but only when the user asked for a status report, not after my edit. **I should have run `gofmt -l` or `nix fmt` immediately after every edit.** Fixed at report time.

### Did not run `golangci-lint` at all

The skill says "Run tests after every change." I ran `go test` but never `golangci-lint run`. For a project with 0 lint issues and strict coverage gates, this is inexcusable. The pre-commit hook may have caught issues, but I should not rely on it.

### Pre-existing data race noticed but ignored

`dashboardui/sse_replay_test.go:182` has a data race: `httptest.ResponseRecorder` is accessed from both the test goroutine (`buf.String()`) and the SSE handler goroutine (`buf.Write` via `WriteHeartbeat`). This is a **pre-existing bug** unrelated to my changes, but I flagged it and moved on without investigating. It only manifests with `-race`. This means `-race` is broken for dashboardui.

---

## e) WHAT WE SHOULD IMPROVE

### Process improvements (my own performance)

1. **Run `gofmt -l` after EVERY edit, not just at the end.** This was a stupid miss.
2. **Run `golangci-lint run` after changes, not just `go test`.** Lint is part of the quality gate.
3. **Run the FULL workspace test suite, not just the modules I touched.** Downstream consumers depend on usermgmt.
4. **Update CHANGELOG.md immediately when work is complete**, per project convention.

### Code improvements noticed during this session

5. **Pre-existing data race in `dashboardui/sse_replay_test.go:182`** — `httptest.ResponseRecorder` is not thread-safe; the SSE heartbeat goroutine writes to it concurrently with the test reading `buf.String()`. This breaks `-race` for the entire dashboardui module.
6. **78+ `gopls infertypeargs` warnings across the workspace** — unnecessary explicit type arguments on generic function calls (e.g., `CommandTyped[Q](...)` where Go can infer `[Q]`). These are cosmetic but noisy.
7. **`gopls stdversion` warnings** — 20+ files reference `json.Marshal`/`json.Unmarshal` from `encoding/json/v2` which gopls flags as requiring go1.27, but the project uses go1.26.5 with `GOEXPERIMENT=jsonv2`. These warnings are expected/false-positive given the experiment flag, but they add noise.
8. **`requirePathValue` is a package-level function in `usermgmt/http.go`** while `currentUser` is a method on `*AuthHandler`. Consider whether `requirePathValue` should also be a method for consistency, or whether it's fine as a standalone utility (current design is fine — it doesn't need receiver state).

---

## f) Up to 50 Things to Get Done Next

### Immediate (this session's debt)

1. Run `golangci-lint run` on root, usermgmt, and dashboardui modules
2. Run full workspace test suite (`go test ./...` across all 15 modules)
3. Add CHANGELOG.md entry for dedup round 3
4. Verify coverage gates still pass (`nix run .#coverage-gate`)

### Data race fix

5. Fix `dashboardui/sse_replay_test.go:182` data race — use a thread-safe response recorder or synchronize goroutine access
6. Re-enable `-race` for dashboardui tests after fix
7. Investigate whether the SSE heartbeat goroutine should be stopped/canceled before the test reads the response body

### Lint cleanup

8. Resolve the 78+ `gopls infertypeargs` warnings (remove unnecessary type args)
9. Suppress or document the `gopls stdversion` json.v2 false positives if possible
10. Run `nix run .#lint` across all 15 modules to confirm 0 issues maintained

### Dedup hardening

11. Add a CI/pre-commit hook that runs `art-dupl -t 3` and fails on new clones
12. Run `art-dupl` with threshold `-t 5` (the default) to see if there are deeper clones worth addressing
13. Run `art-dupl` with `--include-generated` to inspect templ/sqlc-generated code for duplication
14. Check if the new helpers (`readBodyForDecode`, `requirePathValue`, `currentUserWithPathValue`) are tested with unit tests
15. Add unit tests for `requirePathValue` (empty value, non-empty value, error message)
16. Add unit tests for `streamTitlePath` (truncation behavior)

### Architecture / code quality

17. Consider extracting a `dashboardui/handler_helpers.go` for `streamRefFromRequest`, `loadStreamFromRequest`, `streamTitlePath`, `latestVersion` — they're scattered in `handlers.go`
18. Consider whether `currentUserWithPathValue` is over-abstraction (3 returns, used by 2 callers) vs the explicit two-guard pattern
19. Review whether `handleOAuth2Callback` (line 44) should also use `requirePathValue` — it uses `h.oauth2Error` instead of `writeError`, so it was correctly excluded, but document why
20. Review whether `handleOAuth2Begin` (line 19) should use `requirePathValue` — it doesn't call `h.currentUser` first, so the pattern differs slightly

### Broader workspace health

21. Run `art-dupl` per-module (not workspace-wide) to find module-internal clones at lower thresholds
22. Check identity-model module for domain-type duplication
23. Check adminui/loginpage for shared UI handler patterns that could be extracted
24. Review the 15-module workspace for cross-module import duplication
25. Verify all `go.mod` files are tidy (`go mod tidy` per module with `GOWORK=off`)
26. Check if go-cqrs-lite local replaces can be removed yet (AGENTS.md says 13 of ~40 tags still broken)
27. Update AGENTS.md if any conventions changed (new helper patterns added)

### Testing

28. Add BDD tests for the OAuth2 unlink flow (uses `currentUserWithPathValue`)
29. Add BDD tests for credential deletion flow (uses `currentUserWithPathValue`)
30. Add integration test verifying 401 is returned when no user in context for all auth-guarded endpoints
31. Add integration test verifying 400 is returned when path values are empty for all `requirePathValue` endpoints
32. Test the `readBodyForDecode` helper directly (body read error, max body size, empty body)

### Documentation

33. Document the `requirePathValue` / `currentUserWithPathValue` pattern in a guide
34. Update `docs/guides/` if any new handler patterns were introduced
35. Consider adding a "handler conventions" section to AGENTS.md documenting the guard helpers

### Pre-existing issues noticed

36. Fix the `httptest.ResponseRecorder` thread-safety issue in SSE tests
37. Review all dashboardui SSE test files for the same ResponseRecorder pattern
38. Consider using `httptest.NewServer` instead of `ResponseRecorder` for SSE tests (real HTTP server is thread-safe)
39. Audit all test files that spawn goroutines writing to `http.ResponseWriter`

### Dependency / build

40. Verify `GOEXPERIMENT=jsonv2` is documented in all relevant places
41. Check if any new deps were introduced (shouldn't be — pure refactoring)
42. Run `nix flake check` to verify flake integrity
43. Run `nix build` to verify Nix build path
44. Verify `buildflow` pre-commit hook passes on the changes

### Monitoring / CI

45. Add `art-dupl` to CI as a quality gate
46. Add coverage gate enforcement for the new helpers
47. Consider a `justfile` or `flake.nix` target for `art-dupl` (if not already present)
48. Review whether the `gopls infertypeargs` warnings should be auto-fixed with a tool
49. Add a pre-push hook that runs the full workspace test suite
50. Review whether the auto-commit daemon's commit messages (`refactor(handlers): standardize HTTP handler responses`) accurately describe the dedup work — they're generic

---

## g) Questions I CANNOT Answer Myself

1. **Should I fix the pre-existing `dashboardui/sse_replay_test.go` data race?** It's not related to dedup, but it breaks `-race` for the entire module. It's a real concurrency bug in the test (not production code), but fixing it requires understanding the SSE handler test architecture. Should I investigate, or is this owned/tracked elsewhere?

2. **Should the `requirePathValue` helper live in `usermgmt/http.go` or in the root module (`cqrs-htmx`)?** It's a generic HTTP utility (read path value, validate non-empty, write 400). The root module has other HTTP helpers. But moving it to root means usermgmt depends on root for it, and the root module currently has zero imports of usermgmt (clean boundary per AGENTS.md). Should I keep it in usermgmt, or promote it to root?

3. **Should I update CHANGELOG.md for this session, or does the auto-commit daemon's commit messages suffice?** The project convention says completed work goes to CHANGELOG (append-only, per-version). But the auto-commit daemon already created 2 commits with descriptive messages. Is there a version bump planned, or should I add an "Unreleased" entry?

---

## Resolution (2026-07-31)

| Item | Resolution |
| ---- | ---------- |
| 0 clone groups at threshold 3 | **Done** — confirmed. Round 4 (threshold 2) also reached 0. |
| CHANGELOG entry | **Done** — dedup work documented in v4.6.0 and `[Unreleased]`. |
| Lint at 0 issues across all modules | **Done** (2026-07-28). |
| `dashboardui/sse_replay_test.go:182` data race | Still open — TODO_LIST P2. Breaks `-race` for dashboardui. |
| `decoder.go:22` unparam | Still open — TODO_LIST P2. `readBodyForDecode` always returns zero-value T. |
| Canonical nix gates | **Blocked** by httputil v0.8.0 — TODO_LIST P1. |
