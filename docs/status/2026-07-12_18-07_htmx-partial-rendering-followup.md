# Status Report: HTMX Partial Rendering — Follow-up Session

**Date:** 2026-07-12 18:07
**Session Goal:** Complete all follow-up tasks from the partial rendering feature session (17:02)

---

## a) FULLY DONE (Verified)

### Code Changes

1. **Edge case tests added** (`partial_test.go`, +49 lines, 4 new tests)
   - `RenderPartialOrFull[T]` with nil result → returns error (non-200)
   - `RenderTemplComponent` error propagation → returns the Render error directly
   - `OOBHTML` with empty id → produces valid `<div id="" hx-swap-oob="true">...`
   - `OOBHTML` with empty html → produces valid `<div id="slot" hx-swap-oob="true"></div>`
   - Added `errTemplComponent` test double (always fails to render) to avoid polluting the shared `bddTemplComponent` with an `err` field (which would trigger `exhaustruct` on every existing usage)

2. **adminui refactored** (`handler_users.go`, `render.go`)
   - `handler_users.go:16`: `r.Header.Get("HX-Request") == "true"` → `cqrshtmx.RenderPartial(r)` (semantically correct — this IS a partial-vs-full branching)
   - `render.go:65`: `r.Header.Get("HX-Request") == "true"` → `cqrshtmx.IsHTMXRequest(r)` (correct — this is a redirect helper, not partial rendering)
   - Both files gained `cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"` import

3. **Benchmarks added** (`benchmark_htmx_test.go`, +82 lines, 8 sub-benchmarks)
   - `BenchmarkRenderPartial`: HTMXRequest (2005 ns), HTMXHistoryRestore (2025 ns), NonHTMX (1853 ns)
   - `BenchmarkRenderTemplComponent`: Partial (548 ns), Full (452 ns)
   - `BenchmarkOOBHTML`: DefaultSwap (115 ns), CustomSwap (119 ns), Passthrough (6 ns)
   - Added `benchComponent` type and `sink` global (with `nolint:gochecknoglobals`)

### Documentation

4. **CHANGELOG.md** — `[Unreleased]` section with `### Added` (5 functions + adminui refactor + benchmarks) and `### Changed` (WSOOBHTML delegate)

5. **README.md** — New "Partial Rendering (HTMX)" section (59 lines) with 5 code examples: typed templ, non-generic, standalone, custom predicate, OOB swaps

6. **AGENTS.md** — Three updates:
   - File map: `partial.go` added, `options_render.go` description updated, `ws.go` description updated
   - Key Decisions: new bullet point for partial rendering helpers with ADR-0037 reference

7. **ADR-0037** (`docs/adr/0037-partial-rendering-helpers.md`, new file) — Full design document: context, decision (3-layer architecture), Content-Type design decision, 4 alternatives considered with rejection rationale, consequences

8. **ADR INDEX** (`docs/adr/INDEX.md`) — ADR-0037 entry added

9. **SKILL.md** (`.agents/skills/cqrs-htmx/SKILL.md`) — Partial rendering section with code examples, Content-Type note, ADR reference

10. **core-api.md** (`.agents/skills/cqrs-htmx/references/core-api.md`) — 3 HandlerOptions added to Response section, 2 standalone helpers added to the standalone section

### Verification (ALL PASS)

| Check                           | Command                        | Result            |
| ------------------------------- | ------------------------------ | ----------------- |
| Root build                      | `go build ./...`               | PASS              |
| Root tests (679 specs, `-race`) | `go test ./... -count=1 -race` | PASS              |
| Root lint                       | `golangci-lint run ./...`      | 0 issues          |
| Errorfamily                     | `branching-flow errorfamily .` | Clean             |
| Coverage                        | `go test -coverprofile`        | 93.7%             |
| adminui build                   | `go build ./...` (GOWORK=off)  | PASS              |
| adminui tests (`-race`)         | `go test ./... -count=1 -race` | PASS              |
| Module isolation                | `nix run .#check-modules`      | All checks passed |
| Flake check                     | `nix flake check`              | All checks passed |
| Formatting                      | `nix fmt`                      | 0 files changed   |

---

## b) PARTIALLY DONE

### Nothing

All tasks from the follow-up plan were fully completed and verified.

---

## c) NOT STARTED

### Identified but Deferred (from previous session's status report)

1. **`RenderPartialOrFull` for `App.Command`** — Currently only works on query handlers (`handler.go` calls `cfg.render`). Commands don't call `cfg.render`. This is architecturally correct (commands don't return data to render) but limits the API surface. Not addressed.

2. **`Cache-Control: no-store` as a HandlerOption** — adminui sets `Cache-Control: no-store` on every render. This pattern is common enough to warrant a reusable `HandlerOption`. Not addressed.

3. **loginpage module integration** — The loginpage module wasn't checked for partial rendering opportunities. It renders a single standalone page, so it likely doesn't need these helpers, but this wasn't verified.

4. **examples/basic demonstration** — `examples/basic/main.go` could demonstrate `RenderPartialOrFull`. Not updated.

5. **`RenderTemplComponent` naming** — Potential confusion with `RenderTempl` (one is a `HandlerOption`, the other is a standalone function). Identified in the previous session but not resolved. The names are technically correct (`RenderTemplComponent` renders a `TemplComponent`, `RenderTempl` is a `HandlerOption` that sets up templ rendering) but the proximity is confusing.

---

## d) TOTALLY FUCKED UP

### Nothing catastrophic, but several mistakes worth calling out:

1. **`bddTemplComponent` field addition → exhaustruct explosion** — I added an `err` field to the shared `bddTemplComponent` struct to support error testing. This immediately broke `exhaustruct` lint on ALL 10+ existing usages across 4 test files. Fix: reverted the shared struct, created a local `errTemplComponent` type instead. **Lesson: never add fields to shared test doubles without checking exhaustruct impact.**

2. **Missing `HX-Request` header on error test** — The first `RenderTemplComponent` error test didn't set `HX-Request: true`, so the full component (which didn't have the error) was rendered instead of the partial (which did). Test failed. Fix: added the header. **Lesson: when testing partial-vs-full, always verify which branch is actually exercised.**

3. **Import management whack-a-mole** — Removed `errors` import when adding `context`/`io`, then had to re-add it because `errTemplComponent` uses `errors.New`. Should have planned the imports before editing. **Lesson: think about all imports before making structural changes.**

4. **Didn't investigate stale LSP diagnostic** — The `golangci_lint_ls` typecheck warning on `partial_test.go:13` (`expected declaration, found ')'`) appeared throughout the session. I dismissed it as "stale IDE diagnostics" without verifying WHY the LSP can't parse a file that `go vet` and `golangci-lint` both accept. This could indicate a real LSP configuration issue that affects developer experience.

5. **Coverage dipped and I didn't flag it** — Coverage went from 94.2% (baseline) → 93.6% (previous session) → 93.7% (this session). The new edge case tests recovered 0.1%, but we're still below baseline. The new production code (`partial.go`, new functions in `options_render.go`) isn't fully covered — the error paths in `RenderPartialOrFull[T]` (type assertion failure) and `RenderTemplComponent` (Render error) now have tests but the happy paths rely on existing test infrastructure.

---

## e) WHAT WE SHOULD IMPROVE

### Process Improvements

1. **Always check exhaustruct before adding struct fields** — The `bddTemplComponent` mistake was preventable. Run `golangci-lint run` after EVERY structural change, not just at the end.

2. **Investigate LSP diagnostics, don't dismiss them** — The stale typecheck warning on `partial_test.go` might indicate the LSP is using a different Go version or build constraint. This affects every developer using this project with an LSP. Should be investigated.

3. **Run `nix fmt` BEFORE verification, not after** — I ran formatting as an afterthought. It should be part of the pre-commit verification loop. (It passed with 0 changes, but that's luck.)

4. **Test the error path FIRST** — When writing tests for new functions, write the error/edge case tests alongside the happy path tests, not as a follow-up. The initial session shipped 14 tests but missed 4 edge cases that this session had to add.

5. **Document the Content-Type contract in the function signature** — The Content-Type decision (HTML helpers set it, generic don't) is correct but subtle. It's documented in ADR-0037 and the doc comments, but a consumer reading just the function signature won't know. Consider adding a `Content-Type` note to the godoc examples.

### Code Quality

6. **`RenderTemplComponent` vs `RenderTempl` naming** — These names are too similar for functions with different semantics (standalone function vs HandlerOption). Consider renaming `RenderTemplComponent` to `RenderPartialOrFullStandalone` or `RenderTemplDirect` to make the distinction obvious.

7. **`RenderIf` doesn't set Content-Type** — This is correct (the user's RenderFunc owns it), but it's a potential footgun. A consumer using `RenderIf` for HTML rendering might forget to set Content-Type. Consider adding a note to the doc comment.

8. **No integration test for adminui refactor** — The adminui changes (`RenderPartial(r)` replacing raw header check) were verified by the existing adminui test suite, but there's no specific test that verifies "the library's `RenderPartial` works correctly inside adminui's handler structure." The existing `seed_render_test.go` covers it indirectly.

9. **`errTemplComponent` should be in `testing_types_test.go`** — I put it in `partial_test.go` because it's only used there. But if future tests need an error component, they'll have to duplicate it. Should be moved to the shared test types file. (Deferred — YAGNI for now.)

10. **Benchmark `sink` global is a code smell** — The `nolint:gochecknoglobals` directive works but it's a linter suppression. Consider using `runtime.KeepAlive` or a package-level test struct instead. (Standard Go benchmark practice, but worth noting.)

---

## f) Up to 50 Things We Should Get Done Next

### P0 — High Impact

1. Commit all changes (11 modified files + 1 new ADR)
2. Run `nix run .#test` (full multi-module suite across root + usermgmt + integration_test + adminui)
3. Run `nix run .#coverage-gate` (enforces coverage thresholds)
4. Run `nix run .#lint` (multi-module lint)
5. Delete or fix `adminui/coverage_gaps3_test.go` (untracked, uses `templ-components/display` import path — may be wrong package path)
6. Investigate the LSP typecheck false-positive on `partial_test.go:13`
7. Add `RenderPartialOrFull` usage to `examples/basic/main.go` as a demonstration
8. Consider renaming `RenderTemplComponent` to avoid confusion with `RenderTempl`

### P1 — Medium Impact

9. Add `Cache-Control: no-store` as a reusable `HandlerOption` (extracted from adminui pattern)
10. Write integration test that exercises `RenderPartialOrFull` end-to-end through the full App.Query pipeline
11. Add a `RenderPartialOrFullSimple(w, r, partial, full)` variant that takes `TemplComponent` values directly (no mapper functions) for the common case where the caller already has the components
12. Consider `RenderPartialOrFullResult[T]` that takes `func(T) (TemplComponent, error)` for mappers that can fail
13. Add OOBHTML test for HTML containing `hx-swap-oob` in a non-attribute position (e.g. inside text content) — verify passthrough detection is attribute-based, not substring-based
14. Document the `RenderPartial(r)` vs `IsHTMXRequest(r)` distinction more prominently in README (currently only in SKILL.md gotchas)
15. Add a `HandlerOption` that sets `Cache-Control` header (general purpose, not just for partial rendering)
16. Move `errTemplComponent` to `testing_types_test.go` for reuse
17. Add benchmark for `RenderPartialOrFull[T]` (the generic HandlerOption path, not just `RenderTemplComponent`)
18. Check if `loginpage` module has any partial rendering opportunities
19. Consider `RenderHTMLPartialOrFull(partialHTML, fullHTML string)` for non-templ HTML string consumers
20. Add test verifying `RenderPartialOrFull` works with `App.Query` when the query returns a pointer type (`*User` not `User`)

### P2 — Lower Impact

21. Add godoc examples (`ExampleRenderPartialOrFull`, `ExampleRenderIf`, etc.) as runnable tests
22. Create a partial rendering cheat-sheet diagram (D2 or ASCII) showing the decision tree
23. Consider extracting `RenderPartial` check into a standalone `ShouldRenderPartial(r)` that's more discoverable
24. Add `OOBHTMLMulti(map[string]string)` for batch OOB updates
25. Consider `OOBHTMLWithSwap(id, html, swap, target)` for `hx-swap-oob` with target selector
26. Add test for `OOBHTML` with special characters in id (e.g. `id="my-id"`)
27. Add test for `OOBHTML` with very large HTML payloads
28. Consider `RenderPartialOrFull` integration with HTMX `HX-Reselect` / `HX-Reswap` headers
29. Document recommended HTMX `hx-target` / `hx-select` patterns that pair with `RenderPartialOrFull`
30. Add CSS example showing how to structure partial/full templates with shared styles
31. Consider `RenderModal(modal TemplComponent)` HandlerOption for HTMX modal patterns
32. Add test for concurrent `RenderTemplComponent` calls (thread safety)
33. Consider `RenderPartialOrFull` with context-aware mappers (`func(ctx.Context, T) TemplComponent`)
34. Add `RenderPartialOrFull` to the `integration_test` module for cross-module verification
35. Consider `App.QueryPartial(name, decoder, partialMapper, fullMapper)` as a shorthand
36. Add `HandlerOption` that sets `Vary: HX-Request` for cache correctness
37. Consider `RenderFragment(component)` as a simpler name for partial-only rendering (no full-page fallback)
38. Document interaction between `RenderPartialOrFull` and `HX-Boosted` (boosted requests are NOT partial)
39. Add test matrix: all combinations of HX-Request/HX-Boosted/HX-History-Restore-Request
40. Consider `RenderPartialOrFull` for `App.Command` that renders on success (some commands return data)

### P3 — Nice to Have

41. Create a templ component library snippet showing partial rendering patterns
42. Add `RenderPartialOrFull` to the `admin-demo` example for live demonstration
43. Consider `RenderSSEPartial(component)` that writes a partial as an SSE event
44. Add `RenderPartialOrFull` benchmark with realistic data (not just `benchComponent`)
45. Consider `CacheControl(d)` as a general-purpose `HandlerOption` for cache headers
46. Add `ETag(hash)` HandlerOption for client-side caching of rendered partials
47. Consider `RenderPartialOrFull` returning `ETag` header based on result hash for conditional requests
48. Document `RenderPartialOrFull` interaction with `HX-Boost` ( boosted requests want full pages)
49. Add a "Partial Rendering Patterns" section to the wiki/docs
50. Consider publishing a blog post / example repo demonstrating the partial rendering helpers

---

## g) Top 2 Questions I Cannot Answer Myself

### Question 1: Should `RenderPartialOrFull` set `Cache-Control: no-store` automatically?

**Context:** adminui sets `Cache-Control: no-store` on every render (both `renderPage` and `renderPartial`). But the library's `RenderPartialOrFull` doesn't set any cache headers. This means a consumer using `RenderPartialOrFull` without adding their own `Cache-Control` header could get cached partial HTML served for full-page requests (or vice versa).

**Why I can't decide:** This touches the "library principle" (never enforce defaults consumers might disagree with). Some consumers WANT caching for full-page renders. But stale cached partials are a real correctness issue with HTMX. Should the library set `no-store` by default for partial renders only? Or should it set `Vary: HX-Request`? Or leave it entirely to the consumer?

### Question 2: Should we rename `RenderTemplComponent` to avoid confusion with `RenderTempl`?

**Context:** `RenderTempl(component)` is a `HandlerOption` that sets up a static templ render for a CQRS handler. `RenderTemplComponent(w, r, partial, full)` is a standalone function that does partial-vs-full selection for non-CQRS routes. The names are different but too similar — a developer scanning an import list or autocomplete will be confused.

**Why I can't decide:** `RenderTemplComponent` is the most accurate name (it renders a `TemplComponent`). But `RenderTemplPartialOrFull` is too long. `RenderTemplDirect` loses the partial-vs-full semantics. `RenderPartialOrFullStandalone` is descriptive but verbose. The naming decision affects the public API and should be made by the person who owns the library's API surface — not by me guessing what reads best.
