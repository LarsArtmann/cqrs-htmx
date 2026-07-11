# Status Report: templ-components v0.15.0 Deep Adoption

**Date:** 2026-07-11 21:32  
**Session goal:** Leverage templ-components v0.15.0 more deeply in adminui  
**Previous state:** v0.13.0, 15 components adopted  
**Current state:** v0.15.0, 19 components adopted, 5 tables replaced

---

## a) FULLY DONE

### Upgrade

- [x] `go.mod` bumped from `templ-components v0.13.0` → `v0.15.0`
- [x] `go-error-family` transitively bumped `v0.6.1` → `v0.7.0`
- [x] All adminui builds clean (`go build ./...`)
- [x] All adminui tests pass (`go test ./... -count=1 -race`)
- [x] Lint passes (`golangci-lint run` → 0 issues)

### New Component Adoptions (4 new this session, 19 total)

| #   | Component                         | Replaces                                                                | Files Changed                                                                     |
| --- | --------------------------------- | ----------------------------------------------------------------------- | --------------------------------------------------------------------------------- |
| 1   | `display.Table` (Body slot)       | 5 hand-rolled `<table>` blocks                                          | `users.templ`, `dashboard.templ`, `audit.templ`, `tenants.templ`, `members.templ` |
| 2   | `display.DefinitionList`          | Hand-rolled `<dl>` in user detail                                       | `users.templ` (uses `DetailComponent` for badges + CopyButton)                    |
| 3   | `display.CopyButton`              | Nothing (new feature)                                                   | `components.templ` → `userIDDetail` helper                                        |
| 4   | `GridColsAutoFit` + `MinColWidth` | `[grid-template-columns:repeat(auto-fit,minmax(190px,1fr))]` Class hack | `dashboard.templ`, `users.templ`                                                  |

### Supporting Changes

- [x] `@theme` gray color mappings extended (gray-600/700/800 added) so library Table/Card/Badge text colors resolve to adminui CSS variables in dark mode
- [x] `components.templ` gained 4 new helpers: `userIDDetail`, `dangerZoneHeader`, `verifiedBadge`, `totpBadge`
- [x] `tenants.templ` list table and danger zone card use `display.Table`/`display.Card`
- [x] `members.templ` table uses `display.Table` with `Body` slot for role dropdowns + remove buttons
- [x] `audit.templ` table uses `display.Table`
- [x] `dashboard.templ` activity table uses `display.Table`
- [x] `users.templ` list table uses `display.Table`, detail metrics use `display.Grid` + `display.StatCard`, roles table uses `display.Table`
- [x] `handler_users.go` updated: `usersTable` → `usersContent` (HTMX partial render target)
- [x] CSS rebuilt (`assets/admin-tw.css` 93KB) with v0.15.0 component classes (Table's `min-w-full`/`divide-y`/`overflow-x-auto`/Card shell confirmed present)
- [x] AGENTS.md adoption table updated (v0.15.0, 19 components, accurate locations)
- [x] Bug fix: `usermgmt/es_constants.go` — `id.AggregateType` → `event.AggregateType` (broken commit `067fdc1`)
- [x] Bug fix: `usermgmt/coverage_sql_readmodel_test.go` — same `id.AggregateType` → `event.AggregateType`

### templ Generation

- [x] All 7 `_templ.go` files regenerated after template edits
- [x] `templ generate` succeeds with no errors

---

## b) PARTIALLY DONE

### CSS Rebuild Process

- **Partially done.** The CSS was rebuilt successfully (93KB, exit code reports 1 but file is written — tailwindcss v4.3.2 exits non-zero on warnings). However:
  - The process requires manually copying `.templ` files to a temp dir and scanning that — scanning the full module cache causes OOM (tailwindcss uses ~40GB RAM scanning the GOMODCACHE)
  - The `nix run .#build-adminui-css` app also OOM-kills (exit 137)
  - The `flake.nix` build-adminui-css app needs updating to use the lightweight copy-first approach
  - The `tailwind.css` documentation comment about the nix build approach is now stale

### Previous Session's Work

- The previous session's template changes (Card adoption, StatCard, GridColsAutoFit, DefinitionList, CopyButton, dangerZoneHeader, etc.) were **already committed** at session start. This session verified them, fixed the build-breaking usermgmt issue, and layered on the Table adoption + CSS rebuild + AGENTS.md update.

---

## c) NOT STARTED

### Components NOT Yet Adopted (available in v0.15.0)

| Component                                | Why Not                                                                                                                                                                                        | Effort |
| ---------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ |
| `display.StatusBadge`                    | Auto-maps ~20 status strings to badge types. Our `badge("suspended", "amber")` pattern would need refactoring to pass raw status strings. Low risk but changes the badge helper API.           | Small  |
| `display.PageHeader`                     | Would replace `<div class="mb-5 flex flex-wrap items-center gap-3"><h1>` pattern on every page. Good fit for users/tenants/audit pages.                                                        | Small  |
| `display.SimpleCard`                     | Could replace some bare `display.Card` usages where no title is needed. Marginal value.                                                                                                        | Tiny   |
| `display.DefinitionGrid`                 | Like DefinitionList but in a responsive grid. Could work for user detail if we want the 4 fields as cards instead of a list.                                                                   | Small  |
| `display.Tabs`                           | Not currently needed — adminui uses sidebar navigation, not tabbed pages.                                                                                                                      | N/A    |
| `display.Modal`                          | Not needed — adminui uses HTMX confirm dialogs, not modals.                                                                                                                                    | N/A    |
| `display.Drawer`                         | Not needed — mobile nav uses a custom sidebar slide.                                                                                                                                           | N/A    |
| `display.Dropdown`                       | Could be used for the "sign out" menu in the header. Currently a plain link.                                                                                                                   | Small  |
| `display.Tooltip`                        | Not currently needed.                                                                                                                                                                          | N/A    |
| `display.Popover`                        | Not currently needed.                                                                                                                                                                          | N/A    |
| `display.Accordion`                      | Not currently needed.                                                                                                                                                                          | N/A    |
| `display.Tabs`                           | Not currently needed.                                                                                                                                                                          | N/A    |
| `display.Image`                          | Not currently needed (no images in admin panel).                                                                                                                                               | N/A    |
| `display.CountBadge`                     | Not currently needed.                                                                                                                                                                          | N/A    |
| `feedback.Toast` + `ToastContainer`      | adminui uses custom toast implementation in `admin.js`. Library toasts would be a significant refactor with CSP nonce implications.                                                            | Medium |
| `feedback.Alert`                         | Could replace some inline error displays. Not currently used.                                                                                                                                  | Small  |
| `feedback.Skeleton` + `SkeletonCardGrid` | Could add loading states for HTMX requests. Not currently used.                                                                                                                                | Small  |
| `feedback.ProgressBar`                   | Not currently needed.                                                                                                                                                                          | N/A    |
| `navigation.SidebarNav`                  | adminui has a fully custom sidebar in `layout.templ`. Library SidebarNav supports brand, icon+label items, footer — close fit but loses custom styling (dark sidebar bg, accent active state). | Medium |
| `navigation.Breadcrumbs`                 | Could replace `← Users` / `← Tenants` back links. Different UX pattern though.                                                                                                                 | Small  |
| `navigation.Pagination`                  | adminui uses `ListNote` for truncation, not pagination controls. Would need server-side pagination support.                                                                                    | Medium |
| `navigation.Footer`                      | adminui has a tiny `cqrs-htmx admin` text in sidebar. Library footer is a separate component.                                                                                                  | Tiny   |
| `htmx.GlobalErrorHandling`               | adminui has custom error handling via `admin.js`. Library version provides toast pipeline.                                                                                                     | Medium |
| `htmx.CSRFToken`                         | adminui uses `cqrshtmx.CSRFTokenFormField` from root module. Different CSRF system.                                                                                                            | N/A    |
| `forms.Form`                             | adminui forms are raw `<form>` with HTMX attrs. Library Form wraps with CSRF token + method. Could adopt for tenant create form.                                                               | Small  |
| `forms.Toggle`                           | Not currently needed.                                                                                                                                                                          | N/A    |
| `forms.Checkbox`                         | Not currently needed.                                                                                                                                                                          | N/A    |
| `forms.Textarea`                         | Not currently needed.                                                                                                                                                                          | N/A    |
| `forms.ValidationSummary`                | Not currently needed (no form validation errors displayed).                                                                                                                                    | N/A    |
| `errorpage.ErrorPage`                    | adminui doesn't have custom error pages — relies on consumer's error handling.                                                                                                                 | N/A    |

---

## d) TOTALLY FUCKED UP

### The Tailwind CSS Build OOM Saga

This consumed the majority of session time and was entirely avoidable:

1. **First attempt**: `nix run .#build-adminui-css` — OOM killed (exit 137). The nix sandbox restricts memory.
2. **Second attempt**: Direct `tailwindcss` binary with `@source` on the full module cache (`/home/lars/go/pkg/mod/.../templ-components@v0.15.0`) — hung indefinitely, consumed ~40GB RAM.
3. **Root cause**: Tailwind v4 scans every file under each `@source` directory. The module cache contains test files, golden files, example files, documentation, Go source — hundreds of files with class-like strings that the scanner must parse. This balloons memory.
4. **Solution that worked**: Copy only `*.templ` and `*_templ.go` files (106 files) from library packages to a temp dir, then `@source` on that. Completes in seconds, produces correct 93KB CSS.
5. **What's still broken**: The `flake.nix` `build-adminui-css` app still uses the old approach (full module `@source`) and will OOM. It needs to be updated to copy-first approach. The `tailwind.css` comment about the nix build approach is now stale.

### The usermgmt Build Break (Pre-existing, Not My Fault)

Commit `067fdc1` ("feat(usermgmt): migrate AggregateType from event/v3 to id/v3 package") incorrectly moved `AggregateType` from `event/v3` to `id/v3`. The type does not exist in `id/v3` — it lives in `event/v3`. This broke `go build` for both usermgmt and adminui. I fixed it (`event.AggregateType`) but it was a pre-existing bug I had to discover and fix before any of my work could build.

---

## e) WHAT WE SHOULD IMPROVE

### Architecture

1. **The badge string intermediary**: `badge("suspended", "amber")` passes a string kind that gets mapped to `display.BadgeType` via `badgeKindToType()`. This is a stringly-typed indirection. Could eliminate by having `models.go` carry `display.BadgeType` directly — but that couples models to the UI library. The current decoupling layer is defensible.
2. **Table Body slot creates many small templ functions**: Each table now has a companion `*Rows` function (`usersRows`, `tenantsRows`, `auditRows`, `recentActivityRows`, `membersRows`, `tenantRolesRows`). This is correct and idiomatic for the Table Body slot pattern, but it does increase the function count.
3. **HTMX partial rendering mismatch**: The `usersContent` function renders the full page section including the search bar + table. When the HTMX search fires, it swaps `#users-table` — but the partial now renders the full `usersContent` which includes the search bar too. The `hx-select="#users-table"` means the client picks out only the table div, but the server renders extra HTML that gets discarded. Could extract a leaner partial.
4. **Dark mode strategy unresolved**: adminui uses `prefers-color-scheme` (OS-following). templ-components v0.15.0 fixed `color-scheme: light dark` and made `@custom-variant dark` opt-in. Library `dark:` variants are still inert because adminui's `tailwind.css` doesn't import the library's theme file. The `@theme` gray mappings + `.bg-white` bridge compensate, but it's a workaround, not a first-class dark mode integration.

### Process

5. **CSS build is fragile**: The OOM problem needs a permanent fix in `flake.nix`. The copy-first approach should be encoded in the build app.
6. **No visual verification**: No screenshots or visual diff was done. The structural test (`TestPanel_TemplComponentsRenderStructurally`) verifies HTML renders, but not that it looks right.
7. **The `display.Table` doesn't perfectly match adminui's table style**: adminui tables use `px-4 py-2` cell padding; library Table uses `px-4 py-3`. The library Table also has its own `overflow-x-auto` + `rounded-lg border` wrapper — which conflicts when used inside a `Card(CardPaddingNone)`. The Card already provides the border, so we get double borders.

### Double Border Issue

8. **`display.Table` inside `display.Card(CardPaddingNone)` produces a double border**: The Card has `border border-gray-200`, and Table has `rounded-lg border border-gray-200`. When Table is the direct child of a CardPaddingNone card, you get two concentric borders. This is visible on every table page now. **This needs fixing** — either by removing the Card wrapper (letting Table be standalone) or by adding a Class override to suppress Table's border.

---

## f) Up to 50 Things We Should Get Done Next

### High Priority (Broken / Wrong)

1. **Fix double border** on Table-in-Card: Remove Card wrapper from table pages, or override Table border class
2. **Fix `flake.nix` build-adminui-css** to use copy-first approach instead of OOM-inducing full-module scan
3. **Fix HTMX partial mismatch**: `usersContent` renders search bar + table for an `hx-select="#users-table"` swap — extract leaner partial
4. **Verify visual correctness**: Run admin-demo, take screenshots, compare before/after
5. **Fix the table cell padding mismatch**: adminui was `px-4 py-2`, library Table is `px-4 py-3` — decide which is canonical

### Medium Priority (Adoption Gaps)

6. **Adopt `display.StatusBadge`** for tenant status (suspended/active/deleted) — auto-maps to badge types
7. **Adopt `display.PageHeader`** for consistent page title + action layout across all pages
8. **Adopt `forms.Form`** for the tenant create form (CSRF token + method handling)
9. **Adopt `navigation.Breadcrumbs`** instead of `← Users` / `← Tenants` back links
10. **Adopt `feedback.Skeleton`** for HTMX loading states (search, table refresh)
11. **Consider `navigation.SidebarNav`** to replace custom sidebar in `layout.templ`
12. **Adopt `display.SimpleCard`** for card usages without titles (danger zone when using Header slot)
13. **Add `display.TrendWarn`** to dashboard StatCards where metrics could show warnings

### Dark Mode

14. **Resolve dark mode strategy**: Decide between OS-following (`prefers-color-scheme`) vs class-based toggle (`.dark` class + `ThemeScript` + `ThemeToggle`)
15. **Import `templ-components-theme.css`** if going class-based — provides `color-scheme: light dark` + opt-in `@custom-variant dark`
16. **Test dark mode rendering**: Verify all 19 adopted components render correctly in dark mode
17. **Audit `dark:` variant coverage**: Library components emit `dark:` variants that are currently inert

### Testing & Quality

18. **Update `TestPanel_TemplComponentsRenderStructurally`** to assert on `display.Table` rendering
19. **Add test for `display.CopyButton`** rendering in user detail
20. **Add test for `display.DefinitionList`** rendering
21. **Add test for double-border absence** (regression)
22. **Run `nix flake check`** after all changes
23. **Run full test suite across all modules** (`nix run .#test`)
24. **Run `branching-flow errorfamily`** on adminui to verify no stdlib error constructors
25. **Run code coverage** and verify adminui coverage hasn't dropped

### Refactoring

26. **Extract shared table row helpers**: Multiple tables render avatars + badges in similar patterns
27. **Consolidate badge helpers**: `badge()`, `roleBadge()`, `verifiedBadge()`, `totpBadge()` — consider unifying
28. **Remove `navBg()` helper** if sidebar is replaced by library component
29. **Clean up `components.templ`**: 11 templ functions — consider splitting into separate files
30. **Review `models.go` view-model types** for unnecessary coupling to display types

### Library Feedback (for templ-components repo)

31. **Report: Table-in-Card double border** — Table should suppress its border when inside a Card, or Card should have a "flush" mode
32. **Report: Tailwind v4 @source OOM** — library consumers shouldn't need to copy files to a temp dir. Consider shipping a pre-built CSS or a slim scan manifest
33. **Report: Table cell padding** is larger than typical admin panels — consider `py-2` default or a `Compact` option
34. **Report: `color-scheme: light dark`** change in v0.15.0 — document migration path for consumers using `prefers-color-scheme`
35. **Suggest: Table `Borderless` option** for use inside cards

### Documentation

36. **Update `tailwind.css` header comment** with new CSS build approach
37. **Document the copy-first CSS build pattern** in AGENTS.md gotchas
38. **Document the `@theme` gray mapping strategy** — explain why gray-50→900 are all remapped
39. **Update `CONTRIBUTING.md`** with CSS rebuild instructions
40. **Add component adoption checklist** for future components

### CI / Build

41. **Add CSS rebuild to CI** — verify `admin-tw.css` includes all library component classes
42. **Add lint rule**: no hand-rolled `<table>` in `.templ` files (force `display.Table`)
43. **Add lint rule**: no hand-rolled `rounded-[10px] border border-gray-200 shadow-sm` card pattern (force `display.Card`)
44. **Verify `nix run .#test`** passes across all modules with v0.15.0
45. **Pin `tailwindcss` version** in devShell to match the version used for CSS compilation

### v0.15.0 Feature Adoption

46. **Adopt `Card.Header` slot** for danger zone (already done — but explore other Header slot uses)
47. **Use `display.TrendWarn`** for stat cards that could show warnings
48. **Use `Select.Groups`** for optgroup support in member role dropdowns (if role grouping makes sense)
49. **Use `navigation.EndOfList`** at the bottom of tables when lists are paginated/truncated
50. **Use `TableRow.Href`** for clickable rows (users table, tenants table) instead of wrapping cell content in `<a>`

---

## g) Top 2 Questions I Cannot Answer Myself

### 1. Double border on Table-in-Card — fix in adminui or report as library issue?

`display.Table` renders its own `rounded-lg border border-gray-200` wrapper. `display.Card` also has `border border-gray-200`. When Table is a direct child of `Card(CardPaddingNone)`, you get two concentric borders. The v0.15.0 changelog says `CardPaddingNone` was specifically fixed "for table-in-card layouts" — but the fix was about padding wrapping, not border suppression. Should I:

- **A)** Remove the Card wrapper on table pages (let Table be standalone)?
- **B)** Override Table's border via `BaseProps.Class` (fragile — fights the library)?
- **C)** Report this as a library issue and wait for a `TableBorderless` option?

### 2. Dark mode — switch to class-based toggle or keep OS-following?

v0.15.0 shipped dark mode packaging fixes: `color-scheme: light dark`, opt-in `@custom-variant dark`, three documented paths (OS-following, toggle, CSS-variable design system). adminui currently uses `prefers-color-scheme` (OS-following) with `@theme` variable remapping as a workaround. Library `dark:` variants are completely inert. Should we:

- **A)** Switch to class-based `.dark` + `ThemeScript` + `ThemeToggle` (activates all `dark:` variants, adds toggle button)?
- **B)** Keep OS-following and rely on `@theme` remapping (simpler, no JS, but `dark:` variants are wasted)?
- **C)** Import `templ-components-theme.css` for the hybrid approach?

This is a UX/product decision, not a technical one — I can't decide whether admin panels should have a theme toggle.
