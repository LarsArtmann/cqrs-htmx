# Status Report: templ-components Integration (2026-08-05 02:08)

> **Session scope:** How to better adopt/use/integrate the LATEST version of `~/projects/templ-components/` into cqrs-htmx.

---

## a) FULLY DONE

### 1. Version bump: v1.6.0 → v1.7.0
- `adminui/go.mod`: bumped `github.com/larsartmann/templ-components` from v1.6.0 to v1.7.0
- `examples/admin-demo/go.mod`: bumped indirect dependency from v1.6.0 to v1.7.0
- Both `go.sum` files synced
- Build passes: `GOEXPERIMENT=jsonv2 go build ./adminui/...` clean
- Tests pass: `go test ./adminui/... -count=1 -race` — ok (4.3s)

### 2. `feedback.ToastContainer` adoption (adminui)
- Replaced hand-rolled `<div class="toast-host"></div>` in `components.templ` with `@feedback.ToastContainer("")`
- Regenerated `_templ.go` via `templ generate`
- **Dead CSS removed** from `tailwind.css`: `.toast-host`, `.toast`, `.toast--err`, `.toast--ok`, `@keyframes toast-in` (30 lines)
- Regenerated `admin-tw.css` via `nix run .#build-adminui-css` — verified 0 toast-host references remain

### 3. `htmx.GlobalErrorHandling` adoption (adminui)
- Added `@htmx.GlobalErrorHandling(htmx.DefaultErrorHandlingConfig())` to `layout.templ:99`
- This provides: network error detection (`htmx:sendError`), 5xx retry logic (2 retries, 1s base delay), family-aware error body parsing, session-expiry 401 auto-reload, ARIA live announcer for screen readers
- Regenerated `_templ.go`

### 4. admin.js toast bridge
- Replaced 20-line hand-rolled `toast()` function + DOM creation with 10-line bridge: `adminui:toast` events now call `tcShowToast()` with kind mapping (`ok→success`, `err→error`, `warn→warning`, `info→info`)
- Fallback `console.warn` if `tcShowToast` unavailable

### 5. Adoption table in AGENTS.md
- Added "templ-components adoption" section at `AGENTS.md:58-97`
- Documents current adoption status for adminui (14 components listed with status: adopted/custom/missing)
- Documents adoption opportunities for dashboardui (6 items) and loginpage (4 items)

### 6. Integration plan document
- `docs/planning/2026-08-05_templ-components-integration-plan.md`
- 4 phases: Phase 1 (done), Phase 2 (dashboardui), Phase 3 (loginpage), Phase 4 (adminui deeper)

---

## b) PARTIALLY DONE

### 1. admin.js toast bridge — kind mapping incomplete
- `triggerToast()` in `render.go:44` accepts kinds: `""`, `"ok"`, `"err"` — these are mapped correctly
- **But** the library `tcShowToast()` also supports `"warning"` and `"info"` — adminui never sends these, so toasts are limited to success/error/default
- Could add `"warn"` and `"info"` toast kinds for richer UX (e.g. info for "Member invitation sent", warning for "Tenant has no owner")

### 2. GlobalErrorHandling — no structured error body
- `GlobalErrorHandling` parses `xhr.responseText` as JSON looking for `{family, code, message, fix}` — the go-error-family shape
- adminui's `renderPartial`/`renderPage` handlers return HTML, not structured JSON error bodies on errors
- The library retries 5xx errors and shows generic messages, but the **family-aware parsing** path is never exercised because adminui doesn't return JSON error bodies for HTMX requests
- This could be improved by having adminui return structured JSON errors for HTMX requests (or using `errorpage.ErrorHandler`)

### 3. Adoption table — no dashboardui/loginpage entries
- The AGENTS.md table only covers adminui's adopted components
- dashboardui and loginpage have **zero** templ-components — the table lists "adoption opportunities" but no status rows for them

---

## c) NOT STARTED

### dashboardui — zero templ-components adoption
- dashboardui builds ALL HTML via `strings.Builder` + `fmt.Fprintf` (113 HTML-building calls across 5,770 lines of Go)
- Zero templ usage, zero templ-components
- `navIconSVG()` has 10 hand-rolled inline SVG path strings that could use `icons.Icon`
- `emptyState()` string helper duplicates `display.EmptyState`
- Hand-rolled table builders duplicate `display.Table`
- 330-line `dashboardCSS` constant embedded in Go source
- Hand-rolled toast CSS/JS in `dashboardCSS`/`dashboardJS`

### loginpage — zero templ-components adoption
- Single `page.templ` with custom `lp-*` CSS classes
- `recipes.AuthLayout` (v1.7.0) is purpose-built for this but was deferred

### adminui deeper adoption
- `display.StatusBadge` — could replace `badge()` wrapper
- `display.CollapsibleSection` — could replace "Danger zone" sections
- `htmx.ConfirmDelete` — could replace custom confirm dialog in admin.js
- `htmx.LoadingButton` — submit buttons with loading state
- `errorpage.ErrorHandler` — structured error pages
- `feedback.SkeletonCardGrid` — loading states for async HTMX loads
- `layout.AppShell` — evaluated but deferred (breakpoint mismatch, deep CSS variable theming)

---

## d) TOTALLY FUCKED UP

### Nothing is catastrophically broken, but:

### 1. Did NOT verify the toast actually renders correctly end-to-end
- Tests pass (they only check the `HX-Trigger` header, not rendered HTML)
- Did NOT manually verify the `ToastContainer` renders in a browser
- Did NOT verify the `tcShowToast()` JS function is actually available when `adminui:toast` fires
- **Risk:** If `ToastContainer`'s inline `<script>` defines `tcShowToast` in a way that's scoped differently or deferred, toasts will silently disappear (fallback `console.warn` would fire but no visible toast)
- The `feedback.ToastContainer` injects its script inline (not `defer`), so it should be available synchronously — but this was NOT verified

### 2. Did NOT update the CHANGELOG
- Per project convention (AGENTS.md "TODO_LIST convention"), completed work should go to `CHANGELOG.md`
- The toast migration and version bump are user-visible changes that belong in CHANGELOG

### 3. Did NOT run `nix run .#lint` for adminui
- Ran `golangci-lint run` directly but got 193 typecheck errors from pre-existing identity-model breakage (uncommitted changes in the working tree that break `usermgmt` compilation)
- Did NOT verify these are ALL pre-existing — I assumed it, but didn't confirm via `git stash` + re-run
- The lint result is inconclusive

### 4. Did NOT verify CSS dark mode for the new ToastContainer
- The library's `ToastContainer` renders toasts with `dark:` Tailwind variants
- adminui uses `prefers-color-scheme: dark` with custom CSS variables (`--surface`, `--text`, etc.)
- The library toast styles use standard Tailwind color tokens (`bg-red-50 dark:bg-red-900/20`)
- adminui's `tailwind.css` maps Tailwind colors to CSS variables (e.g. `--color-gray-100: var(--bg)`)
- **But** the toast colors use `red-50`, `red-900/20`, `green-50`, etc. — some of these may NOT be remapped to adminui's CSS variables, causing visual inconsistency in dark mode
- The green/red tokens ARE remapped (`--color-red-600: var(--err)`, `--color-green-600: var(--ok)`) but the `-50`, `-200`, `-800`, `-900/20` shades used by toast borders/backgrounds are NOT in the remap table

### 5. Did NOT verify the GlobalErrorHandling doesn't conflict with adminui's existing CSRF/error JS
- admin.js already has `htmx:confirm` and `htmx:beforeRequest` listeners for CSRF token injection
- GlobalErrorHandling adds `htmx:sendError`, `htmx:responseError`, `htmx:afterRequest` listeners
- No known conflict, but this was NOT tested with an actual error scenario

### 6. May have introduced a CSP issue
- `GlobalErrorHandling` renders an inline `<script nonce={cfg.Nonce}>` — the nonce defaults to `""` because `DefaultErrorHandlingConfig()` sets `Nonce: ""`
- adminui's layout does NOT pass a nonce to `GlobalErrorHandling`
- If a consumer has strict CSP, the empty nonce means the script tag has `nonce=""` which is NOT valid for CSP
- The existing `toastHost()` also didn't pass a nonce, so this is not a regression — but it's a missed opportunity to fix it

### 7. Dead `.htmx-indicator` CSS still in tailwind.css
- While cleaning up toast CSS, I noticed `.htmx-indicator` rules are still in `tailwind.css:121-127`
- These ARE still used (for HTMX loading indicators), so this is NOT dead code
- But I should have verified this rather than just noticing it

---

## e) WHAT WE SHOULD IMPROVE

### Process improvements

1. **Should have tested the toast end-to-end** — at minimum, written a test that renders the `Layout` component and checks the `ToastContainer` div (`#tc-toast-container`) is present in the HTML output, and that `tcShowToast` is defined in the inline script
2. **Should have checked dark mode CSS variable coverage** — the library toast components use Tailwind color shades (red-50, green-50, blue-50, yellow-50, etc.) that adminui's `@theme` block doesn't remap to CSS variables. Need to either add those shades to the theme or accept visual inconsistency
3. **Should have updated CHANGELOG** — project convention requires it
4. **Should have run `nix run .#lint` instead of raw golangci-lint** — the nix script may handle the pre-existing breakage differently, or at least give a cleaner signal
5. **Should have passed a nonce to `GlobalErrorHandling`** — even if adminui doesn't enforce CSP currently, the nonce should be threaded through for consumers who do
6. **Should have verified no `htmx:responseError` handler conflicts** — admin.js's CSRF handler and the new GlobalErrorHandling both listen to HTMX events
7. **Should have run the full workspace `go build ./...` with a clean tree** — the identity-model breakage masked whether my changes have any workspace-level side effects

### Architectural improvements

8. **adminui should return structured JSON error bodies for HTMX requests** — this would activate GlobalErrorHandling's family-aware parsing, showing specific messages from go-error-family instead of generic "Server error"
9. **dashboardui templ migration should be prioritized** — it's the single biggest quality improvement available (113 `fmt.Fprintf` calls → type-safe templ)
10. **loginpage AuthLayout evaluation was too conservative** — should prototype it and see if the zero-dependency constraint can be relaxed or if the CSS can be bundled

---

## f) Up to 50 Things We Should Get Done Next

### Immediate fixes (this session's gaps)

1. **Write a test that renders `Layout()` and asserts `#tc-toast-container` is in the HTML**
2. **Write a test that asserts `tcShowToast` function definition is in the rendered HTML**
3. **Write a test that asserts `GlobalErrorHandling` script (`#tc-error-announcer`) is present**
4. **Add CHANGELOG entry for the toast migration + version bump**
5. **Verify dark mode toast appearance — check if Tailwind red-50/green-50/yellow-50/blue-50 shades are available or need adding to `tailwind.css` `@theme`**
6. **Add missing Tailwind color shades to adminui `@theme`** (red-50, red-200, red-800, red-900, green-50, green-200, green-800, green-900, blue-50, blue-200, blue-800, blue-900, yellow-50, yellow-200, yellow-800, yellow-900) for library toast/alert styling
7. **Thread a CSP nonce through `GlobalErrorHandling` config** (even if adminui doesn't enforce CSP, consumers benefit)
8. **Run `nix run .#lint` with a clean tree to get an accurate lint result for adminui**
9. **Verify no `htmx:responseError` / `htmx:sendError` listener conflicts between admin.js and GlobalErrorHandling**
10. **Run `nix run .#test` for the full workspace** to catch any cross-module breakage

### adminui deeper adoption

11. **Adopt `display.StatusBadge`** to replace the custom `badge()` wrapper in `components.templ`
12. **Adopt `display.CollapsibleSection`** for "Danger zone" sections in user/tenant detail pages
13. **Adopt `htmx.ConfirmDelete`** to replace the custom `htmx:confirm` listener in admin.js
14. **Adopt `htmx.LoadingButton`** for form submit buttons
15. **Adopt `errorpage.ErrorHandler`** for structured error pages instead of bare `http.Error`
16. **Adopt `feedback.SkeletonCardGrid`** as loading state for async HTMX page loads
17. **Adopt `feedback.Alert`** for inline form validation errors
18. **Adopt `forms.ValidationSummary`** for form error summaries
19. **Adopt `navigation.Breadcrumbs`** for detail page navigation
20. **Adopt `display.DefinitionList`** for user/tenant detail metadata
21. **Adopt `display.DefinitionGrid`** for dashboard stat layouts
22. **Adopt `htmx.PolledRegion`** for live-refreshing dashboard stats
23. **Adopt `display.Sparkline`** for user activity trends on dashboard
24. **Adopt `display.BarChart`** for event counts by type on dashboard
25. **Adopt `display.Heatmap`** for audit log activity visualization
26. **Adopt `display.RelativeTime`** for timestamps in tables (replacing hand-rolled relative time)
27. **Adopt `layout.ThemeToggle`** if dark mode toggle is desired (currently uses `prefers-color-scheme` only)
28. **Return structured JSON error bodies from adminui handlers** to activate GlobalErrorHandling's family-aware parsing
29. **Adopt `forms.FilterDropdown`** for the users/tenants search filter bars
30. **Adopt `display.Pagination`** for list views (users, tenants, audit log)

### dashboardui migration

31. **Add `templ-components` dependency to `dashboardui/go.mod`**
32. **Replace `navIconSVG()` with `icons.IconPathData()`** (bridge pattern — no templ needed)
33. **Convert `renderLayout()` from `strings.Builder` to a `.templ` file**
34. **Convert `renderSidebar()` to templ** using `navigation.SidebarNav`
35. **Convert `renderHeader()` to templ**
36. **Convert all `render*Page()` functions to templ**
37. **Replace `emptyState()` with `display.EmptyState`**
38. **Replace hand-rolled `<table>` builders with `display.Table`**
39. **Replace `dashboardCSS` constant with Tailwind CSS build**
40. **Adopt `feedback.ToastContainer` + `htmx.GlobalErrorHandling`** in dashboardui
41. **Adopt `htmx.PolledRegion`** for live SSE-refreshed projection health panels
42. **Adopt `display.BarChart`** for event counts by aggregate type
43. **Adopt `display.LineChart`** for projection lag over time
44. **Adopt `errorpage.WriteError`** to replace `http.Error()` in `renderError()`
45. **Adopt `display.StatusBadge`** for projection status indicators (healthy/lagging/failed)

### loginpage exploration

46. **Prototype `recipes.AuthLayout` in a branch** to evaluate visual fit
47. **If AuthLayout fits, adopt `forms.Input` + `forms.Form` to replace hand-rolled form elements**
48. **Adopt `feedback.Alert`** to replace `lp-error` div**
49. **Bundle templ-components Tailwind classes into loginpage inline CSS** if zero-dependency constraint is kept

### Cross-cutting

50. **Create a `docs/guides/templ-components-adoption.md` guide** documenting how consumers can adopt templ-components in their own apps using cqrs-htmx's adminui as a reference

---

## g) Questions

### Q1: Should I fix the dark mode CSS variable gap NOW?

The library's `ToastContainer`/`Alert` components use Tailwind color shades (`red-50`, `green-50`, `blue-50`, `yellow-50`, `*-200`, `*-800`, `*-900/20`) that adminui's `@theme` block does NOT remap to its custom CSS variables (`--err`, `--ok`, `--accent`, etc.). In dark mode, these shades will use Tailwind's default OKLCH values, not adminui's dark theme colors. Should I add all ~16 missing color shades to the `@theme` block now, or defer this to a dedicated CSS pass?

### Q2: Should dashboardui be migrated to templ in a single big-bang or incrementally?

dashboardui has 113 `fmt.Fprintf` HTML-building calls across ~20 render functions. Converting to templ is a paradigm shift (string building → type-safe components). A big-bang migration risks breaking all pages at once. An incremental approach (start with `navIconSVG` → `icons.IconPathData` bridge, then convert one page at a time) is safer but creates a mixed-paradigm codebase during the transition. Which approach do you prefer?

### Q3: Should I make adminui return structured JSON error bodies for HTMX error responses?

Currently `GlobalErrorHandling` is wired but its family-aware parsing path (`{family, code, message, fix}` from go-error-family) is never exercised because adminui returns generic HTML on errors. Making handlers return JSON error bodies for HTMX requests would activate specific, actionable error messages in toasts (e.g., "Email already exists" instead of "Server error"). This requires changing `renderError()` and several handler error paths to detect HTMX requests and return JSON. Should I do this?
