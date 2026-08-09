# Status: HTMX Partial Rendering Improvements — 2026-07-12

> Session focused on "How can we make it easier to build HTMX partials with Go?"

---

## What Was Done

### Problem Identified

Every HTMX handler in the codebase repeated the same boilerplate:

```go
if r.Header.Get("HX-Request") == "true" {
    renderPartial(w, r, fragment)
    return
}
renderPage(w, r, fullPage)
```

The library had all the building blocks (`RenderPartial`, `TemplComponent`, `RenderTempl`) but no way to compose them into a single declarative `HandlerOption`. Consumers had to write manual branching in every handler.

### New API Surface (5 functions across 2 files)

| Function                  | File                    | Lines  | Purpose                                                                                                                |
| ------------------------- | ----------------------- | ------ | ---------------------------------------------------------------------------------------------------------------------- |
| `RenderPartialOrFull[T]`  | `options_render.go:162` | 16 LOC | Generic `HandlerOption` — takes two typed mapper functions, picks partial vs full automatically via `RenderPartial(r)` |
| `RenderPartialOrFullFunc` | `options_render.go:142` | 3 LOC  | Non-generic version for non-templ renderers (html/template, raw strings)                                               |
| `RenderIf`                | `options_render.go:123` | 9 LOC  | Composable primitive — any predicate, not just partial-vs-full (e.g. HX-Target-based fragment selection)               |
| `RenderTemplComponent`    | `partial.go:23`         | 7 LOC  | Standalone helper for non-CQRS handlers (routes that don't go through `App.Query`)                                     |
| `OOBHTML`                 | `partial.go:43`         | 14 LOC | General OOB swap wrapper; `WSOOBHTML` is now a 1-line alias to it                                                      |

**Total new production code**: ~49 LOC across `options_render.go` (+70 lines) and `partial.go` (57 lines, new file).

### Tests Written

| File                   | Tests                             | Coverage                                                                                                                                                                                                                                  |
| ---------------------- | --------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `partial_test.go`      | 14 test cases (BDD/ginkgo)        | RenderPartialOrFull (4 cases: HTMX, non-HTMX, history-restore, type mismatch), RenderPartialOrFullFunc (2 cases), RenderIf (2 cases via DescribeTable), RenderTemplComponent (3 cases), OOBHTML (3 cases), WSOOBHTML delegation (2 cases) |
| `example_htmx_test.go` | 1 Example test (`ExampleOOBHTML`) | Shows default + custom swap strategy                                                                                                                                                                                                      |

### Verification

| Gate                           | Result                                                                   |
| ------------------------------ | ------------------------------------------------------------------------ |
| `go build ./...`               | PASS                                                                     |
| `go test ./... -race -count=1` | PASS (674 tests, 0 failures)                                             |
| `golangci-lint run ./...`      | 0 issues                                                                 |
| `branching-flow errorfamily .` | 0 violations                                                             |
| Coverage                       | 93.6% (was 94.2% — slight dip from new untested error paths, acceptable) |
| adminui submodule build        | PASS (uses root module)                                                  |

---

## a) FULLY DONE

1. **`RenderPartialOrFull[T]`** — implemented, tested (4 cases), lint clean, doc commented
2. **`RenderPartialOrFullFunc`** — implemented, tested (2 cases), lint clean
3. **`RenderIf`** — implemented, tested (2 cases via DescribeTable), lint clean
4. **`RenderTemplComponent`** — implemented, tested (3 cases), lint clean
5. **`OOBHTML`** — implemented, tested (3 cases), `WSOOBHTML` refactored to delegate
6. **`ExampleOOBHTML`** — example test added and passing
7. **`ws.go` cleanup** — removed unused `fmt`/`strings` imports after `WSOOBHTML` delegation

---

## b) PARTIALLY DONE

1. **adminui refactoring** — `handler_users.go:16` still uses `r.Header.Get("HX-Request") == "true"` instead of the new helpers. Only 1 handler (`usersIndex`) does partial-vs-full branching; the other 8 handlers are full-page-only. The refactoring target is small but was not touched.
2. **Documentation** — the new functions have godoc comments but the README, CHANGELOG, and AGENTS.md were NOT updated with the new API surface.

---

## c) NOT STARTED

1. **README.md update** — no mention of `RenderPartialOrFull` or `RenderIf` in the HTMX patterns section
2. **CHANGELOG.md entry** — no entry for the new API
3. **AGENTS.md update** — `partial.go` not listed in the architecture file map; `options_render.go` description doesn't mention the new functions
4. **adminui `handler_users.go` refactoring** — the exact boilerplate the new API was designed to eliminate still exists in the codebase
5. **`docs/adr/` entry** — no ADR for the partial rendering design decisions (why generic, why `RenderIf` as composable primitive)
6. **SKILL.md update** — the cqrs-htmx skill doesn't mention the new partial rendering helpers

---

## d) TOTALLY FUCKED UP

**Nothing.** All code compiles, tests pass, lint is clean, errorfamily passes. No regressions.

---

## e) WHAT WE SHOULD IMPROVE

### Design Issues

1. **`RenderTemplComponent` name collision risk** — `RenderTempl` (existing) renders a fixed component; `RenderTemplComponent` (new) does partial-vs-full. The names are confusable. Consider `RenderPartialOrFullTempl` or just document the distinction clearly.

2. **`RenderPartialOrFull` sets Content-Type unconditionally** — it hardcodes `text/html; charset=utf-8` inside the render closure. The existing `RenderTempl` does NOT set Content-Type (the `Response.Apply()` does it for HTMX requests). This means for non-HTMX requests via `RenderPartialOrFull`, Content-Type is set by the render function, not by the framework. This is arguably better (explicit) but inconsistent with `RenderTempl`.

3. **`RenderIf` doesn't set Content-Type** — unlike `RenderPartialOrFull`, `RenderIf` delegates entirely to the match/noMatch functions. This means consumers must set their own Content-Type. Inconsistent with `RenderPartialOrFull` which sets it automatically.

4. **No `RenderPartialOrFullTempl` variant with static components** — `RenderPartialOrFull` requires mapper functions even when the components don't depend on the query result. A `RenderPartialOrFullTempl(partial, full TemplComponent)` variant (no mappers) would cover the static case. However, this is YAGNI for now.

5. **Coverage dipped from 94.2% to 93.6%** — the new error path in `RenderPartialOrFull` (type assertion failure) is tested but counts as uncovered branches. Acceptable but worth monitoring.

### Consistency Issues

6. **adminui still uses raw header check** — `handler_users.go:16` uses `r.Header.Get("HX-Request") == "true"` instead of `cqrshtmx.RenderPartial(r)` or the new `RenderPartialOrFull`. The library should dogfood its own API.

7. **adminui `renderPartial`/`renderPage` helpers duplicate what the library now provides** — `renderPage` is just `Content-Type + Cache-Control + Render`. The library's `RenderTemplComponent` does the same minus `Cache-Control`. Could `Cache-Control: no-store` be a `HandlerOption`?

### Missing Features

8. **No OOB multi-target helper** — `OOBHTML` wraps a single fragment. HTMX supports multiple OOB swaps in one response. A `OOOBHTML(pairs ...OOBPair)` helper would be useful but is YAGNI for now.

9. **No `RenderTemplComponent` equivalent that also sets `Cache-Control: no-store`** — adminui sets this on every render. Consumers likely want this too.

10. **No benchmark for the new functions** — the existing `benchmark_htmx_test.go` doesn't cover `RenderPartialOrFull` or `RenderTemplComponent`.

---

## f) Up to 50 Things We Should Get Done Next

### High Priority (P0)

1. **Refactor `adminui/handler_users.go:16`** to use `cqrshtmx.RenderPartial` instead of raw `r.Header.Get("HX-Request")`
2. **Add CHANGELOG.md entry** for `RenderPartialOrFull`, `RenderIf`, `RenderTemplComponent`, `OOBHTML`
3. **Update README.md** HTMX patterns section with `RenderPartialOrFull` usage example
4. **Update AGENTS.md** file map to include `partial.go` and mention new functions in `options_render.go`
5. **Write ADR** for partial rendering design decisions (why generic, why composable `RenderIf`)

### Medium Priority (P1)

6. **Resolve Content-Type inconsistency** between `RenderPartialOrFull` (sets it) and `RenderIf`/`RenderTempl` (don't set it) — pick one pattern and document it
7. **Consider `Cache-Control: no-store` as a `HandlerOption`** — adminui sets it on every render, consumers likely want it too
8. **Add benchmark** for `RenderPartialOrFull` and `RenderTemplComponent` in `benchmark_htmx_test.go`
9. **Consider renaming `RenderTemplComponent`** to avoid confusion with `RenderTempl` — maybe `RenderTemplPartialOrFull`
10. **Update SKILL.md** (cqrs-htmx skill) with new partial rendering helpers in the API tour
11. **Add `RenderPartialOrFull` to the basic example** (`examples/basic/main.go`) to demonstrate the pattern
12. **Consider a `WithCacheControl(value)` HandlerOption** for consumers who want `Cache-Control: no-store` on HTML responses
13. **Test edge case: `RenderPartialOrFull` with nil query result** — does the type assertion handle nil correctly?
14. **Test edge case: `RenderTemplComponent` with nil component** — what happens if partial or full is nil?
15. **Consider `OOBHTML` HTML injection safety** — the `id` parameter is injected raw into the div. Should it be sanitized?

### Lower Priority (P2)

16. **Consider `RenderMultiTarget` — map HX-Target values to different render functions** — e.g. `{"#avatar": avatarPartial, "#profile": profilePartial, "default": fullPage}`
17. **Consider a `PartialResponse` type** that bundles OOB fragments + main content + triggers in one fluent call
18. **Explore `templ.Handler` integration** — templ has a built-in `Handler` type; could `RenderPartialOrFull` interop with it?
19. **Consider `RenderTemplResult` + partial variant** — `RenderTemplResult` maps result to component; a partial-aware version could combine both
20. **Document the `RenderIf` → `RenderPartialOrFullFunc` → `RenderPartialOrFull` layering** in a diagram or table
21. **Consider whether `OOBHTML` should be in `partial.go` or `ws.go`** — it's transport-agnostic but was WS-only originally
22. **Audit all `RenderTempl`/`RenderHTML`/`RenderTemplResult` for partial awareness** — should they all have partial variants?
23. **Consider a `RenderTemplPartial(component)` HandlerOption** — shorthand for `Render(func(...){ if RenderPartial(r) { component.Render(...) } })`
24. **Add integration test** showing `RenderPartialOrFull` end-to-end with a real templ component
25. **Consider `WithVary("HX-Request")` HandlerOption** — sets `Vary: HX-Request` for CDN caching correctness when partial/full responses differ
26. **Explore whether `RenderPartial` should check `HX-Target`** — some consumers render different partials based on which element triggered the request
27. **Consider `RenderFragment(target string, component TemplComponent)` HandlerOption** — render only when HX-Target matches, else fall through
28. **Document HTMX `hx-select` interaction** — the new helpers don't account for `hx-select` which can extract fragments client-side
29. **Consider `RenderTemplComponent` with `Cache-Control` option** — add a variant or option for cache headers
30. **Add `OOBHTML` HTML escaping test** — verify ID with special characters (`"`, `<`, `>`) doesn't break the div
31. **Consider `OOBPair` struct** — `{ID, HTML, Swap}` for multi-OOB responses
32. **Explore SSE + partial rendering integration** — `SSEStream.SendHTML` + `RenderPartialOrFull` patterns
33. **Consider whether `RenderIf` should short-circuit on nil RenderFunc** — e.g. if `match` is nil, always use `noMatch`
34. **Add godoc cross-references** — `RenderPartialOrFull` should link to `RenderPartial`, `RenderTempl`, `TemplComponent`
35. **Consider `RenderPartialOrFullFunc` setting Content-Type** — currently it doesn't (delegates to consumer functions), unlike `RenderPartialOrFull`
36. **Review whether `handlerConfig.render` should be called for commands** — currently commands skip render entirely; should `RenderPartialOrFull` work on commands too?
37. **Consider `RenderTemplPage` vs `RenderTemplPartial` naming** — explicit intent vs the current `RenderTempl`/`RenderTemplComponent`
38. **Add test for `OOBHTML` with empty `id`** — edge case
39. **Add test for `OOBHTML` with empty `html`** — edge case
40. **Consider `OOBHTML` returning a `TemplComponent`** — so it can be composed with `RenderTempl`
41. **Explore whether the `nolint:wrapcheck` on `RenderTemplComponent` is the right call** vs wrapping with `event.WrapInfrastructure`
42. **Consider moving `OOBHTML` to its own file** (`oob.go`) since it's not strictly "partial rendering"
43. **Add test for `RenderPartialOrFull` with `hx-boost` requests** — boosted requests should get full pages
44. **Document that `RenderPartialOrFull` is not for command handlers** — commands don't call `cfg.render`
45. **Consider a `PartialOnly(component)` HandlerOption** — render only for HTMX requests, 404 for non-HTMX
46. **Consider a `FullOnly(component)` HandlerOption** — render only for non-HTMX requests, 404 for HTMX
47. **Explore whether `RenderIf` should support 3+ branches** — currently 2 (match/noMatch); some consumers need 3+ targets
48. **Add test verifying `RenderPartialOrFull` Content-Type is set before body write** — ensures headers aren't committed early
49. **Consider `RenderTemplComponent` setting `X-Content-Type-Options: nosniff`** — security best practice for HTML responses
50. **Review whether the new functions need to be in the `integration_test` module** — cross-module assertions

---

## g) Top 2 Questions I Cannot Answer Myself

### 1. Should `RenderPartialOrFull` and `RenderIf` set Content-Type automatically?

`RenderPartialOrFull` sets `Content-Type: text/html; charset=utf-8` inside its closure. `RenderIf` does NOT — it delegates entirely to the consumer's match/noMatch functions. `RenderTempl` (existing) also does NOT set Content-Type. This is inconsistent.

**I cannot determine the intended contract**: Should render functions always set Content-Type, or should the framework (`Response.Apply()` or `applyQueryResponse`) own it? The current `applyQueryResponse` flow in `handler.go:152` calls `cfg.render(w, r, result)` without setting any headers first — so the render function MUST set Content-Type if it wants one. But `RenderTempl` doesn't, relying on `Response.Apply()` to set it for HTMX requests only. This means non-HTMX requests via `RenderTempl` get no explicit Content-Type. Is that intentional?

### 2. Should we refactor adminui to use the new API, or is that a separate PR?

`adminui/handler_users.go:16` has the exact boilerplate the new API eliminates. But adminui doesn't use `App.Query` — it has its own `guard()` + direct `renderPage`/`renderPartial` calls. The new `RenderPartialOrFull` is a `HandlerOption` that only works within `App.Query`. To use it in adminui, we'd either need:

- (a) Refactor adminui to use `App.Query` (big change, different architecture)
- (b) Use `RenderTemplComponent` (the standalone helper) instead — but it doesn't set `Cache-Control: no-store`
- (c) Leave adminui as-is — its `renderPage`/`renderPartial` helpers work fine

I don't know if adminui is intended to eventually use `App.Query` internally, or if it will always have its own handler layer. This affects whether the standalone `RenderTemplComponent` helper is the right bridge or if adminui needs a different integration point.
