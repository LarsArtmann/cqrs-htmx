# Status: Async Projection Startup — Session 3 (Post-Bugfix Verification)

**Date:** 2026-08-12 21:19
**Session scope:** Bug fix for nil-pointer test panic, lint fix, comprehensive verification, planning document, commit + push. This session followed two prior sessions where the feature was implemented and a status report was written.

> **CORRECTION** (2026-08-12 21:56): The "stale LSP" claim below in the Executive Summary was **WRONG**. The build IS broken — go-cqrs-lite master (`af4b60841`, committed 21:42) reverted the ADR-0111 API. All 4 core modules fail to compile (`event.WithActor`, `id.ActorID`, `Metadata().ActorID` all undefined). The tests that "passed" in this session likely ran against an earlier go-cqrs-lite state that was subsequently overwritten by the auto-git daemon. The coverage numbers (root 92.8%, usermgmt 81.2%, setup 87.9%) are **UNVERIFIABLE** until the build is fixed. The async startup feature code itself is correct — it was committed when the build worked. The break is in go-cqrs-lite, not in the feature code. See `docs/status/2026-08-12_21-54_docs-health-audit-self-review.md`.

---

## Executive Summary

The async projection startup feature is **shipped and verified**. All tests pass with race detector. All lint checks clean. All three affected modules (root, usermgmt, setup) are green. Coverage thresholds hold. Two bugs were found and fixed this session: a nil-pointer panic in the test's nil-provider case, and a gochecknoglobals lint finding.

**The critical lesson from this session:** The previous status report (session 2) claimed the build was broken by a pre-existing `event.WithActor` error. This was **stale LSP state** — the actual `go build` passes cleanly. I never ran `go build` until this session. This means the entire previous status report's "TOTALLY FUCKED UP" section was based on a hallucinated build break. The real lesson: **always verify with real commands, never trust LSP diagnostics for build-pass/fail decisions.**

---

## a) FULLY DONE

1. **Core mechanism:** `startProjectionHost(..., block bool, ...)` — skips `waitForDrain` when `block=false`. All 5 call sites updated.
2. **Config plumbing:** `AsyncStartup bool` on `ServiceConfig`, `EventSourcedConfig`, `setup.Config`. Zero-value = `false` = backward compatible. Forwarded end-to-end.
3. **Readiness gate:** `cqrshtmx.ProjectionReadinessCheck(provider)` — returns 503 for `idle`/`running`/`backoff`/`draining`/`failed`, 200 for `live`/`stopped`.
4. **Setup integration:** `healthHandler` uses `ProjectionReadinessCheck` instead of `failed`-only inline check.
5. **Tests — all pass:**
   - Root: `TestProjectionReadinessCheck` (9 table-driven subtests) + `TestProjectionReadinessCheck_FailedPriority` + `TestProjectionReadinessCheck_HandlerHTTP`
   - Usermgmt: `TestStartProjectionHost_AsyncSkipsDrain` (timing assertion) + `TestNewEventSourcedSetup_AsyncStartup` (config wiring + eventual readiness)
   - Setup: All 6 existing health tests still pass with the new `ProjectionReadinessCheck` (no behavioral regression in sync mode)
6. **Bug fix:** Nil-pointer panic in `TestProjectionReadinessCheck/nil_provider_passes` — typed-nil `*mockStatusProvider` becomes non-nil interface. Replaced fragile name-string check with explicit `nilProvider bool` struct field.
7. **Lint fix:** `//nolint:gochecknoglobals` on `projectionDrainReady` map — same pattern as `htmx_extensions.go`, `loginpage/config.go`.
8. **Race detector:** Full suites pass for root, usermgmt, setup (`-race` flag).
9. **Lint:** 0 issues on root, 0 issues on usermgmt, 0 issues on setup.
10. **Coverage verified:**
    - Root: 92.8% (gate is 90%) — `ProjectionReadinessCheck` function: **100% coverage**
    - Usermgmt: 81.2% (gate is 74%)
    - Setup: 87.9% (gate is 80%)
11. **Documentation:** Guide (`docs/guides/async-projection-startup.md`), CHANGELOG (3 entries), AGENTS.md (Key Patterns + guide count 20), planning doc with mermaid graph, feedback moved to `processed/`.
12. **Committed and pushed** to `master` (3 commits: feature + bugfix/lint + planning doc).

---

## b) PARTIALLY DONE

1. **Integration test for async mode 503→200 transition:** The unit tests verify `ProjectionReadinessCheck` logic and `AsyncStartup` config wiring separately, but no test starts a real HTTP server with `AsyncStartup=true` and verifies the `/health` endpoint transitions from 503 (draining) to 200 (ready) as projections catch up. This is the most important missing test.
2. **Backoff behavioral change documentation:** The `/health` endpoint now returns 503 during `backoff` (transient retry state), not just `failed`. This is a behavioral change from the old `failed`-only check. It is mentioned in the guide but not in the CHANGELOG as a potential breaking change for consumers who rely on `/health` always returning 200 during normal operation.
3. **`TestNew_AllConfigFields` in setup:** This test asserts all config fields are exercised. The new `AsyncStartup` field is NOT tested in this test. The test still passes because the field is optional (zero value = false), but it's a gap.

---

## c) NOT STARTED

1. **ADR-0048** — ← **OPEN** — see TODO_LIST P3.
2. **`examples/async-startup-demo/`** — ← **OPEN** — runnable example with reverse proxy config.
3. **Options B, C, D from feedback** — ← **OPEN** — see ROADMAP Operational Tooling Ideas (read-model hydrator, projection snapshots, SQLite CheckpointStore).
4. **`WaitForDrain(ctx)` method** — ← **OPEN** — see ROADMAP.
5. **Structured drain progress in readiness response** — ← **OPEN** — enhancement.
6. ~~**FEATURES.md update** — No entry for `AsyncStartup`.~~ **DONE** — added to FEATURES.md Root > Convenience in this docs-health session.
7. **`docs/guides/production-readiness.md` update** — ← **OPEN** — should mention `AsyncStartup` as a production checklist item.
8. **Cross-reference from `projection-health-monitoring.md`** — ← **OPEN** — see TODO_LIST P3.
9. ~~**`nix run .#coverage-gate`**~~ **Partially done** — raw `go test -cover` verified; full nix gate pending (see TODO_LIST P1).
10. ~~**`nix run .#check-cqrs-lint`**~~ **Partially done** — not run via nix; raw golangci-lint passed on all 3 affected modules.

---

## d) TOTALLY FUCKED UP

1. **Session 2's status report was fundamentally wrong about the build being broken.** The report claimed `event.WithActor` was undefined and the build was broken at HEAD. This was stale LSP state — I never ran `go build ./...` until session 3 (this session). The build has been clean the entire time. The status report's entire "TOTALLY FUCKED UP" section (item 1: "The build is broken and I used it as an excuse to not run tests") was describing a phantom problem. The REAL fuck-up was not running `go build` in session 1 or 2. **Root cause:** I trusted `<project_diagnostics>` from the LSP (gopls) output attached to every tool result instead of running the actual compiler. The LSP was showing stale state from before go-cqrs-lite was updated.
2. **The nil-pointer test bug should never have been written.** The fragile `if tt.statuses != nil || tt.name != "nil_provider_passes"` logic was a code smell from the start. A typed struct field (`nilProvider bool`) is the obviously correct pattern. I wrote the fragile version first and only fixed it when tests failed. Should have used the clean pattern from the beginning.
3. **Three commits for one logical change.** The auto-git daemon committed mid-session (commits `af59f3f7` and `e4b7e366`), then I added a third commit (`b9058c8c`) for the lint fix and planning doc. The commit history is messier than it should be — the feature, bugfix, and lint fix should ideally have been one squashed commit. This is a minor issue given the auto-git daemon context, but worth noting.

---

## e) WHAT WE SHOULD IMPROVE

1. **NEVER trust LSP diagnostics for build-pass/fail.** Always run `go build ./...` as the authoritative check. The LSP `<project_diagnostics>` are stale cache, not ground truth. This cost an entire session's worth of wrong status reporting.
2. **Run `go test` immediately after writing tests, not after writing all tests.** I wrote all tests then ran them together. If I had run the nil-provider test immediately after writing it, I would have caught the typed-nil bug instantly.
3. **The `ProjectionReadinessCheck` status strings are duplicated from `projectionhost.WorkerStatus` constants.** If go-cqrs-lite adds/renames a status, the readiness check silently breaks. Consider exporting constants or adding a compile-time check — but this requires importing `projectionhost` into the root module, which breaks the dependency boundary. Tradeoff is documented but not resolved.
4. **The `/health` behavioral change (backoff → 503) should be in the CHANGELOG as a potential breaking change**, not just in the guide. Consumers upgrading who rely on `/health` always returning 200 during normal operation may see intermittent 503s.
5. **`TestNew_AllConfigFields` in setup should exercise `AsyncStartup`.** The field exists but no test verifies it flows through `setup.New` → `usermgmt.NewService` → actual async behavior in the setup context.
6. **No integration test exists** that verifies the full async startup lifecycle: server starts → `/health` returns 503 → projections drain → `/health` returns 200. This is the single most valuable test that could be added.

---

## f) Up to 50 Things to Get Done Next

### Critical (correctness gaps)

1. Write integration test: HTTP server with `AsyncStartup=true` → `/health` 503→200 transition
2. Add `AsyncStartup` case to `TestNew_AllConfigFields` in `setup/setup_test.go`
3. Add CHANGELOG entry noting the `backoff` → 503 behavioral change (potential breaking change)
4. Test that `RebuildProjection` still works correctly when `AsyncStartup=true` (it passes `block=true` always — verify this is correct)

### High priority (documentation + completeness)

5. Write ADR-0048: Liveness/Readiness Decoupling
6. Add `AsyncStartup` to `FEATURES.md`
7. Update `docs/guides/production-readiness.md` checklist with `AsyncStartup` recommendation
8. Cross-reference from `docs/guides/projection-health-monitoring.md` to async startup guide
9. Run `nix run .#coverage-gate` (authoritative CI gate)
10. Run `nix run .#check-cqrs-lint` and add suppressions if needed
11. Run `nix run .#lint` (full 12-module lint via flake)

### Architecture / depth

12. Design `ReadModelHydrator` interface (Option B from feedback)
13. Implement SQLite `CheckpointStore` (Option D from feedback)
14. Design projection snapshots (Option C from feedback)
15. Add `WaitForDrain(ctx)` method on `Service`/`EventSourcedSetup`
16. Add structured drain progress to readiness response (JSON body)
17. Add `DrainProgress()` method for programmatic access to drain status
18. Consider `ReadyTimeout` — permanent 503 if drain exceeds N minutes

### Robustness

19. Add fuzz test for `ProjectionReadinessCheck` (random status combinations)
20. Add benchmark for `ProjectionReadinessCheck` (called on every `/health` request)
21. Test with SQL-backed read models + `AsyncStartup` + `CheckpointStore` together
22. Test crash-restart scenario: kill process mid-drain, verify clean restart
23. Verify `datastar` Broadcaster works correctly during async drain
24. Add metrics: drain duration, events processed during drain, drain completion timestamp

### Examples / ecosystem

25. Create `examples/async-startup-demo/` with Caddy/nginx config
26. Update `examples/basic/` to demonstrate `AsyncStartup`
27. Check if `systemadapter` module needs `AsyncStartup` support
28. Check if `dashboardui` should show drain state
29. Verify `e2e/server` Playwright tests aren't affected
30. Update `integration_test` module for new config field

### Cleanup / polish

31. Update CHANGELOG to note the `backoff` behavioral change explicitly
32. Verify `docs/guides/async-projection-startup.md` is in any index/table of contents
33. Run `nix fmt` on all changed files
34. Consider whether `AsyncStartup` should default to `true` in v5
35. Review whether `ProjectionReadinessCheck` should be composable with `OnProjectionFailed`
36. Add a `/drain-status` endpoint for fine-grained visibility
37. Consider connecting `DrainTimeout` to async mode (background warning goroutine)

### Testing depth

38. Test nil `OnProjectionFailed` callback with async startup
39. Test `ProjectionReadinessCheck` with concurrent `ProjectionStatuses()` calls
40. Test that writes work immediately with `AsyncStartup=true` (events append to journal)
41. Test that reads during drain return stale data (or error, depending on consumer choice)
42. Test with large event journal to verify drain actually completes in background

### Documentation

43. Update README.md if it mentions startup behavior
44. Add migration guide for sync→async (when to switch, what to watch for)
45. Document the status string contract (which strings mean what)
46. Add troubleshooting section: "server starts but /health stays 503"
47. Add troubleshooting section: "reads return stale data after async startup"

### Broader

48. Review if `samber-do-demo` or `fullstack-wiring` guide should mention `AsyncStartup`
49. Check if any external documentation (pkg.go.dev examples) needs updating
50. Consider a blog post or announcement about the zero-downtime restart capability

---

## g) Questions (cannot figure out myself)

1. **Should `AsyncStartup` default to `true` in a future major version (v5)?** The feedback recommends it as the production default, but changing the zero-value default from sync→async would be a behavioral breaking change for existing consumers who rely on read-your-writes-on-startup. This is a product/versioning decision — should we deprecate the sync default now and flip in v5, or keep sync as the default forever and just document async as recommended?

2. **Should `ProjectionReadinessCheck` treat `backoff` as ready (200) or not-ready (503)?** Currently it returns 503 for `backoff` — meaning if a projection hits a transient error and enters retry-backoff during normal operation, the health endpoint flips to 503. This is stricter than the old behavior (which only returned 503 for `failed`). Is this the right semantic? "Ready" could mean "fully caught up" (503 during backoff) or "healthy and retrying" (200 during backoff). This is a judgment call about what operators expect from a health check.

3. **Should the `ProjectionReadinessCheck` error response body be structured JSON with drain progress, or keep the current plain-text error string?** Currently it returns `"projections still draining: user-read-model, casbin-projection"`. A structured response would help operators and reverse-proxy health-check UIs, but changes the response body contract. The `ReadinessHandler` already wraps it in `{"status":"degraded","checks":{"projections":{"status":"fail","error":"..."}}}` JSON — is that sufficient, or should the inner error field also be structured?
