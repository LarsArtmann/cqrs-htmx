# templ-components Integration Plan

> How to better adopt `github.com/larsartmann/templ-components` v1.7.0 across cqrs-htmx.
>
> Created: 2026-08-05. Status: Phase 1 complete, Phases 2-3 planned.

## Current state

| Module        | templ-components | Version | Approach                                           |
| ------------- | ----------------- | ------- | -------------------------------------------------- |
| adminui       | yes               | v1.7.0  | templ + templ-components (display, forms, icons, utils, feedback, htmx) |
| dashboardui   | no                | -       | `strings.Builder` HTML (113 fmt.Fprintf calls)     |
| loginpage     | no                | -       | Single hand-rolled `.templ` with custom `lp-*` CSS |
| admin-demo    | indirect          | v1.7.0  | Pulls adminui transitively                         |

## Phase 1 — Completed (2026-08-05)

1. **Version bump v1.6.0 → v1.7.0** (adminui + admin-demo)
2. **Adoption table** in root `AGENTS.md` for discoverability
3. **`feedback.ToastContainer`** replaced hand-rolled `<div class="toast-host">` — now uses the library's accessible toast container with `aria-live`, proper toast types, and dismiss buttons
4. **`htmx.GlobalErrorHandling`** added to adminui layout — provides:
   - Network error detection (`htmx:sendError`) → toast
   - HTTP error handling (`htmx:responseError`) with retry logic for 5xx
   - Family-aware error body parsing (rejection/conflict/transient/corruption/infrastructure)
   - Session-expiry detection (401 → auto-reload)
   - ARIA announcer for screen readers
5. **admin.js toast bridge** — `adminui:toast` HX-Trigger events now call `tcShowToast()` with kind mapping (`ok→success`, `err→error`)
6. **Dead CSS cleanup** — removed `.toast-host`, `.toast`, `.toast--err`, `.toast--ok`, `@keyframes toast-in` from `tailwind.css`

## Phase 2 — dashboardui adoption (medium effort, high impact)

dashboardui builds ALL HTML via `strings.Builder` with `fmt.Fprintf`. 113 HTML-building calls across 5,770 lines. Zero templ usage. This is the biggest opportunity.

### Step 2a: Add templ-components icons (low effort, immediate value)

Replace the hand-rolled `navIconSVG()` function (10 hard-coded inline SVG path strings in `layout.go:142-167`) with `icons.Icon(name, class)` from templ-components.

**Challenge:** dashboardui uses `strings.Builder`, not templ. The icons package returns `templ.Component`, not `string`. Two approaches:
- **Option A:** Convert dashboardui's render functions to templ (the right long-term answer)
- **Option B:** Call `icons.IconPathData(name)` to get raw SVG path data as `[]string`, then build the SVG string manually (bridge pattern)

**Recommendation:** Option B as a stepping stone, Option A as the goal.

### Step 2b: Convert render functions to templ (high effort, transformative)

Convert `renderLayout`, `renderSidebar`, `renderHeader`, and all `render*Page` functions from `strings.Builder` to `.templ` files. This enables:
- Type-safe HTML generation (no more `esc()` calls or `fmt.Fprintf` format string bugs)
- Direct use of all templ-components (display.Table, display.Card, feedback.ToastContainer, etc.)
- Better maintainability (templ syntax highlighting, formatting, LSP support)

**Estimated scope:** ~20 render functions across layout.go, handlers_*.go, render.go.

### Step 2c: Adopt specific components

After converting to templ (or as part of it):

| Current hand-rolled code            | Replace with                          |
| ----------------------------------- | ------------------------------------- |
| `emptyState()` string helper        | `display.EmptyState`                  |
| `<table>` string builders           | `display.Table` with `TableProps`     |
| `navIconSVG()` switch               | `icons.Icon`                          |
| `dashboardCSS` const (330 lines)    | Tailwind CSS (via templ-components)   |
| `dashboardJS` toast listener        | `htmx.GlobalErrorHandling`            |
| `renderToastContainer()`            | `feedback.ToastContainer`             |
| `renderError()` using `http.Error`  | `errorpage.WriteError`                |

### Step 2d: Adopt v1.7.0 chart components

dashboardui shows projection health, event counts, DLQ sizes — all prime candidates for:
- `display.BarChart` — event counts by type
- `display.Sparkline` — projection lag trends
- `display.Heatmap` — event activity by time/hour

## Phase 3 — loginpage adoption (low effort, high impact)

loginpage is a single `page.templ` with hand-rolled `lp-*` CSS. The new `recipes.AuthLayout` (v1.7.0) is purpose-built for this.

### Step 3a: Adopt `recipes.AuthLayout`

`AuthLayout` provides a split-screen authentication layout with:
- Card panel (for the login form)
- Branding panel (for logo/tagline)
- Reversed layout option
- Panel features list

This would replace the hand-rolled `lp-container`, `lp-card`, `lp-header` structure.

### Step 3b: Adopt form components

Replace hand-rolled `<input>`, `<label>`, `<form>` elements with:
- `forms.Input` — typed input with label, error, help text
- `forms.Form` — form wrapper with CSRF token
- `display.Button` — styled button/link

### Step 3c: Adopt feedback components

Replace `lp-error` div with `feedback.Alert` or `forms.ValidationSummary`.

### Challenge

loginpage has a unique constraint: it must work without external CSS dependencies (it embeds its own CSS inline). templ-components requires Tailwind CSS to be loaded. Two approaches:
- **Option A:** Bundle the templ-components Tailwind classes into loginpage's inline CSS
- **Option B:** Accept the external CSS dependency (add a `<link>` tag)
- **Option C:** Keep loginpage as-is (it's deliberately self-contained)

**Recommendation:** Option C for now. loginpage is intentionally zero-dependency. The AuthLayout pattern is better suited for consumer apps that already use templ-components.

## Phase 4 — adminui deeper adoption (optional)

### Components not yet adopted that exist in the library:

| Component                     | Use case in adminui                                   |
| ----------------------------- | ----------------------------------------------------- |
| `display.StatusBadge`         | Replace `badge()` wrapper with auto-mapping StatusBadge |
| `display.CollapsibleSection`  | "Danger zone" sections in user/tenant detail pages    |
| `navigation.Pagination`       | Replace hand-rolled pagination if present             |
| `feedback.SkeletonCardGrid`   | Loading state for async HTMX page loads               |
| `htmx.ConfirmDelete`          | Replace custom confirm dialog in admin.js             |
| `htmx.LoadingButton`          | Submit buttons with loading state                     |
| `errorpage.ErrorHandler`      | Structured error pages for unhandled errors           |

### AppShell consideration (deferred)

`layout.AppShell` was evaluated but deferred because:
- adminui's layout uses `max-md:` breakpoint (768px); AppShell uses `lg:` (1024px)
- adminui uses custom CSS variables (`--sidebar-bg`, `--surface`, `--accent`) deeply integrated with the theme
- adminui has SSE sync bar, custom mobile hamburger with admin.js integration
- Forcing AppShell would require either custom CSS overrides or accepting visual regressions
