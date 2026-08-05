# Status Report — dashboardui Core Extraction & Phase 2 Validation

**Date:** 2026-08-05 06:32
**Session scope:** Execute `docs/planning/2026-08-05_05-25_dashboardui-3-layer-decomposition.md` Phase 2
**Overall status:** Phase 2 COMPLETE. Build green, tests green, lint clean. Several gaps remain.

---

## A) FULLY DONE

| # | Item | Evidence |
|---|------|----------|
| 1 | **Build fix: `export.go` broken** | `jsontext.Encoder.SetEscapeHTML` does not exist. Replaced with `jsontext.NewEncoder(w, jsontext.WithIndent("  "))`. Build was BROKEN on arrival — the "all 96 tests pass" claim from the prior session was stale or was tested before the json/v2 migration commit. |
| 2 | **Core package unit tests** | 7 test files (1,196 LOC), 158 test assertions. Covers capabilities, pagination, events, overview, format, payload, DLQ. Committed as `0b5ac53b`. |
| 3 | **CHANGELOG.md updated** | 3 new entries: core tests, CSP-safe rendering, JSON export encoder fix. |
| 4 | **AGENTS.md updated** | Fixed wrong description (said "templ + HTMX" — dashboardui does NOT use templ). Added core/ architecture info and CSP-safe note. |
| 5 | **README.md updated** | Added `core/` sub-package usage section with import example. |
| 6 | **Lint clean** | 0 issues on `dashboardui/...` after exhaustruct exclusions for `core.Config`, `core.Capabilities`, and new fake types. |
| 7 | **Full workspace build** | `go build ./...` passes clean. |
| 8 | **CSP safety (prior session, verified)** | 0 inline `onsubmit` handlers remain (only in a comment). Inline toast script moved to external JS. `data-confirm` delegated listener active. |

---

## B) PARTIALLY DONE

| # | Item | What's done | What's missing |
|---|------|-------------|----------------|
| 1 | **Core test coverage** | 67.2% — decent for pure-data functions | `ListStreamsPaged` is **0% covered** (no test at all). `DLQProjectionLinks` is 16.7% (only nil-host path tested). `FetchOverview` is 48.9% (no ProjectionHost health classification path). `ProjectionStats` is 25% (only nil-host tested — needs a real `projectionhost.Host` which requires a journal + checkpoint store to construct). |
| 2 | **Error path testing** | `LoadEventByID` has not-found and no-source tests | `LoadRecentEvents`/`LoadFilteredEvents` error paths (readErr, allErr on fakes) NOT tested. `DefaultPayloadRenderer` CBOR path NOT tested. `FetchOverview` error paths (StreamReader error, journal read error) NOT tested. |
| 3 | **golangci-lint exclusions** | Added core types and fakes | The fakes are duplicated — the parent `dashboardui` package has `fakeSeekableJournal`, `fakeEventByIDLoader`, etc. in `handlers_coverage_ext_test.go`, and the `core` package has its own copies in `fakes_test.go`. Not a bug (test packages are isolated), but it's duplication. |

---

## C) NOT STARTED

| # | Item | Planning doc ref |
|---|------|------------------|
| 1 | **Phase 3: `panels/` package scaffold** | Task 3.1 — create `panels/` with go.mod, PanelOpts, doc.go |
| 2 | **Phase 3: DLQ panel templ port** | Task 3.2 — smallest panel, most duplicated by DiscordSync |
| 3 | **Phase 3: Projections panel templ port** | Task 3.4 — highest operational value |
| 4 | **Phase 3: Golden-file rendering tests** | Tasks 3.3, 3.5 |
| 5 | **Phase 4: Events browser, command audit, time-travel, aggregates, snapshots, overview templ ports** | Tasks 4.1-4.7 |
| 6 | **Phase 5: Standalone layout → templ** | Tasks 5.1-5.5 |
| 7 | **Phase 6: DiscordSync wiring** | Tasks 6.1-6.7 — separate repo |
| 8 | **Coverage gate enforcement** | The dashboardui coverage gate (84.0%/60) does NOT include `core/` — the flake.nix coverage app would need updating to test the core sub-package separately. Not done. |

---

## D) TOTALLY FUCKED UP

| # | What | Impact | Severity |
|---|------|--------|----------|
| 1 | **Build was broken on arrival** | The prior session claimed "all 96 tests pass" and "full workspace builds clean" but `export.go:62` called `encoder.SetEscapeHTML(true)` which does not exist on `jsontext.Encoder`. The build was broken. Either the prior session tested before the json/v2 migration commit (`775b5d1c`), or the auto-commit daemon committed the broken file after the session ended. I caught this immediately with `go build`. | **High** — the first thing I did was fix it, but anyone pulling master between the prior session and this one got a broken build. |
| 2 | **AGENTS.md said "templ + HTMX" for dashboardui** | dashboardui does NOT use templ — it uses `strings.Builder` + `fmt.Fprintf`. The root AGENTS.md line 33 was factually wrong. I corrected it, but it was wrong for the entire prior session and nobody noticed. | **Medium** — misleading to any session that reads AGENTS.md to understand the codebase. |
| 3 | **gopls stale diagnostics (50+ DuplicateDecl errors)** | The prior session documented this as a "known gotcha" — gopls shows 50 false DuplicateDecl errors in `core_bridge.go` and `handler_overview.go`. The errors are stale (go build succeeds). But they pollute the LSP diagnostics and make it hard to spot REAL errors. Restarting gopls did not fix them. | **Low** — cosmetic, but annoying for any session relying on LSP diagnostics. Root cause is likely gopls not handling Go type aliases that alias across packages. |

---

## E) WHAT WE SHOULD IMPROVE

### Process improvements

1. **Always run `go build ./...` FIRST, before trusting any "done" claim.** The prior session's status report said tests pass, but the build was broken. I caught it in 2 seconds. This should be the first tool call of every session.

2. **gopls aliases issue needs root-causing.** 50 false-positive DuplicateDecl errors are a productivity killer. The type aliases in `core_bridge.go` are valid Go — gopls should handle them. Either file a gopls bug or restructure the bridge to avoid aliases (use wrapper types instead). The `handler_overview.go` false errors suggest gopls is seeing phantom declarations that `go build` doesn't.

3. **Test fakes are duplicated.** `fakeSeekableJournal`, `fakeEventByIDLoader`, `fakeDeadLetterStore`, `fakeStreamReader` exist in BOTH `dashboardui/handlers_coverage_ext_test.go` AND `dashboardui/core/fakes_test.go`. Consider extracting to a shared `testutil` package or a `core/testhelpers` sub-package.

4. **Coverage gate gap.** The flake.nix coverage gate enforces 84.0% on dashboardui but does NOT include the new `core/` sub-package (67.2%). Either add core to the gate or document it as intentionally un-gated.

### Code quality improvements

5. **`ListStreamsPaged` is completely untested (0% coverage).** It's the only function with zero coverage. It needs a `fakeStreamReader` that returns a real `*listing.Page` with items and HasMore.

6. **`FetchOverview` ProjectionHost path is untested (48.9% coverage).** The health classification (Healthy/Degraded/Unhealthy) logic is the most complex branch in core and has no test. Needs a mock `*projectionhost.Host` which requires constructing one via `projectionhost.New(journal, cpStore, ...)`.

7. **`DLQProjectionLinks` with a real ProjectionHost is untested (16.7%).** Only the nil-host early return is tested. The DeadLetterStore path and the error-counter fallback path need tests.

8. **CBOR payload rendering is untested.** `DefaultPayloadRenderer.Render` with `codec.EncodingCBOR` has no test — only JSON, raw, and empty are covered.

9. **`contains` and `indexOf` helper functions in `pagination_test.go`** are hand-rolled reimplementations of `strings.Contains`. Should use `strings.Contains` directly.

10. **`export.go` nolint cleanup was messy.** I tried `//nolint:errchkjson` (wrong linter name), then `//nolint:errcheck` (unused — errcheck doesn't flag `_ =` assignments), then removed it entirely. The `MarshalEncode` return value is intentionally ignored for dynamic export data. A `//nolint:errchkjson` with a correct reason would be cleaner, but the linter doesn't flag it anyway so removing was fine.

---

## F) Up to 50 Things to Get Done Next

### Immediate (this session could have done but didn't)

1. Test `ListStreamsPaged` with a populated `fakeStreamReader` (0% → ~90%)
2. Test `FetchOverview` with a real `projectionhost.Host` (health classification path)
3. Test `DLQProjectionLinks` with ProjectionHost + DeadLetterStore (the actual logic)
4. Test `DefaultPayloadRenderer` CBOR encoding path
5. Test `LoadRecentEvents` / `LoadFilteredEvents` error paths (readErr, allErr)
6. Test `FindEventNeighbors` with `Journal` fallback (currently only SeekableJournal)
7. Replace `contains`/`indexOf` helpers with `strings.Contains`
8. Extract shared test fakes to avoid duplication between packages

### Phase 3: Panels Proof of Concept

9. Create `dashboardui/panels/` sub-package with `go.mod`
10. Define `PanelOpts` struct (BasePath, AccentColor, CSRFToken, ReadOnly)
11. Write `panels/doc.go` explaining the layer
12. Port DLQ panel from `handlers_dlq.go` to `panels/dlq.templ`
13. Write golden-file rendering test for DLQ panel
14. Port Projections panel from `handlers_projections.go` to `panels/projections.templ`
15. Write golden-file rendering test for Projections panel
16. Verify panels package has zero root-module dependency
17. Add panels to `go.work`
18. Add panels to flake.nix module lists (test, lint, coverage, build apps)
19. Add panels to `.github/workflows/ci.yml`

### Phase 4: Full Panels Migration

20. Port Command Audit panel to templ
21. Port Events Browser panel to templ (largest, ~480 LOC)
22. Port Time-Travel panel to templ
23. Port Aggregates Browser panel to templ
24. Port Snapshots panel to templ
25. Port Overview panel to templ (composes others)
26. Write golden-file tests for all ported panels
27. Port detail views (event detail, command detail, query detail, projection detail)

### Phase 5: Standalone Rebuild

28. Port standalone layout (sidebar, header, CSS) to templ
29. Rewrite standalone handlers to use core + panels
30. Delete all `fmt.Fprintf` rendering code
31. Delete embedded `dashboardCSS` constant (335 lines)
32. Delete `core_bridge.go` (no longer needed once main package imports core directly)
33. Full test suite green on standalone rebuild

### Infrastructure & Quality

34. Root-cause the gopls DuplicateDecl false positives (file gopls issue or restructure)
35. Add `core/` to flake.nix coverage gate
36. Add `core/` to `.golangci.yml` coverage enforcement
37. Run `nix run .#test` to verify all 19 modules still pass
38. Run `nix run .#lint` to verify all 11 lint-checked modules pass
39. Run `nix run .#coverage` to verify coverage gates
40. Run `nix run .#check-templates` to verify SQL setup files
41. Update planning doc to mark Phase 2 as COMPLETE
42. Add gopls stale-diagnostics note to dashboardui-specific AGENTS.md or gotchas
43. Consider a `core/testhelpers` sub-package for shared fakes

### DiscordSync (separate repo)

44. Delete DiscordSync's custom `dlq.templ` + handler (~570 LOC of duplication)
45. Import `dashboardui/panels` in DiscordSync
46. Wire `SeekableJournal` for event journal browser
47. Wire `EventSource` for time-travel
48. Run DiscordSync test suite
49. Verify CSP headers pass with the new panels
50. Document the DiscordSync integration pattern as a guide

---

## G) Questions I Cannot Answer Myself

### 1. Should I fix the gopls false positives by restructuring `core_bridge.go`?

The 50 DuplicateDecl errors are stale gopls diagnostics — `go build` succeeds. They exist because gopls may not handle type aliases that cross package boundaries well (`type Capabilities = core.Capabilities` in one file, while another file in the same package has the original declaration path cached). Two options:

- **Option A:** File a gopls bug and leave the aliases (they're correct Go)
- **Option B:** Replace aliases with wrapper types (more code, but no false errors)

I can't decide this because I don't know if you've seen this pattern in other modules or if there's a known gopls workaround.

### 2. Should the coverage gate include `core/`?

The flake.nix coverage-gate app enforces 84.0%/60 on dashboardui but doesn't test `core/` separately. Adding it would require adding `core` to the module list in the coverage-gate app AND setting an appropriate threshold (67.2% now, but should be higher after the missing tests). I don't know what threshold you'd want or if you even want to gate the sub-package separately.

### 3. Is the `export.go` fix correct for your use case?

I removed `SetEscapeHTML(true)` entirely (the method doesn't exist on `jsontext.Encoder`). The usermgmt export pattern just uses `jsontext.NewEncoder(w, jsontext.WithIndent("  "))` with no HTML escaping. For `application/json` file downloads this is fine. But if any consumer was relying on HTML-escaped JSON output from the `?format=json` endpoint, they'd get different output now. I can't verify this without knowing your consumer expectations.
