# Status Report: templ-components Adoption + Self-Review Cleanup

**Date:** 2026-07-11 10:39
**Session scope:** Adopt templ-components in adminui, write consumer feedback, brutal self-review, fix all identified issues
**Working tree:** Clean, all pushed to origin

---

## a) FULLY DONE

### templ-components adoption (15 components adopted, was 1)

| Component              | Sites | Replaced                                                                           |
| ---------------------- | ----- | ---------------------------------------------------------------------------------- |
| `icons.IconPathData`   | ~15   | Pre-existing (gateway adoption)                                                    |
| `display.Avatar`       | 3     | Hand-rolled initials circles in layout + users                                     |
| `display.Badge`        | ~20   | `badge()` + `roleBadge()` + `badgeColor()` + `badgeColors` map + `roleBadgeKind()` |
| `display.Button`       | 9     | All hand-rolled `<button>` and `<a>` elements across tenants/members/users         |
| `display.Card`         | 2     | Card wrapper divs in dashboard + audit pages                                       |
| `display.Grid`         | 1     | Dashboard stat grid                                                                |
| `display.StatCard`     | 1     | `statCardView()` hand-rolled card                                                  |
| `display.EmptyState`   | 6     | `empty()` hand-rolled component                                                    |
| `display.RelativeTime` | 3     | `relTime()` 19-line switch/case formatter                                          |
| `display.ListNote`     | 3     | `listNote()` hand-rolled component                                                 |
| `forms.Input`          | 4     | Search box, tenant form fields, member email                                       |
| `forms.Select`         | 2     | Role dropdowns in members page                                                     |
| `feedback.Spinner`     | 0     | Was dead code — deleted entirely                                                   |

### Infrastructure

- CSS bridge: `.bg-white { background-color: var(--surface) }` in `tailwind.css` for theme compatibility
- Dynamic `@source` resolution in `nix run .#build-adminui-css` (resolves GOMODCACHE via `go list -m`)
- `tailwind.css` recompiled with templ-components class scanning
- Dead code eliminated: `relTime()`, `badgeColor()`, `badgeColors`, `roleBadgeKind()`, `selectedAttr()`, hand-rolled spinner SVG

### Self-review cleanup (4 commits)

- Removed dead `spinner()` component (commit `9a9d772`)
- Added structural smoke test `TestPanel_TemplComponentsRenderStructurally` (commit `45c91c1`)
- Updated AGENTS.md with full adoption table + CSS bridge docs (commit `6b4b8e1`)
- Replaced deprecated `GONOSUMCHECK` with `GOPRIVATE` across 6 files (commit `7d0ad08`)

### Consumer feedback report

- Written to `/home/lars/projects/templ-components/docs/feedback/2026-07-10_cqrs-htmx-consumer-feedback.md`
- 6 prioritized recommendations for the templ-components library

### Pre-existing build fixes (unblocked compilation)

- `render.go`: json v2 → stdlib `encoding/json` + `json.RawMessage`
- `import_export.go`: `enc.SetIndent` → `jsontext.WithIndent`
- `oauth2/provider.go`: `json.Number` → `jsontext.Value`

---

## b) PARTIALLY DONE

### CSS dark mode compatibility

- **Done:** `.bg-white` bridge handles surface colors in dark mode
- **Not done:** Library `dark:` variants are inert (adminui uses `prefers-color-scheme`, not `.dark` class). Badge type colors (green/blue/red/amber) work fine in both modes, but any component that relies on `dark:` for non-surface styling won't dark-mode-correctly. This is documented but not fixed.

### Structural test coverage

- **Done:** Smoke test verifies Avatar (rounded-full + bg-blue-600), Select (`<select name=role>`), Input (type=search name=q), Button (text content), EmptyState (title)
- **Not done:** No test for Card, Grid, StatCard, RelativeTime, ListNote, Badge rendering specifically

---

## c) NOT STARTED

1. **Adopt `display.DefinitionList`** for user detail `<dl>` — evaluated, decided poor fit (mixed text + conditional badge content)
2. **Adopt `navigation.SidebarNav`** for the layout sidebar — identified as opportunity, not attempted
3. **Eliminate `badgeKindToType` string intermediary** — use `display.BadgeType` directly in view models
4. **Consolidate `icon()` vs `icons.Icon()`** — two ways to render icons exist (adminui wrapper vs library component)
5. **Visual verification** — no browser/screenshot verification was done; all testing is programmatic

---

## d) TOTALLY FUCKED UP

### Hardcoded `@source` path (FIXED but was a critical mistake)

I added `@source "/home/lars/go/pkg/mod/github.com/larsartmann/templ-components@v0.13.0"` to `tailwind.css`. This was a machine-specific absolute path that would silently produce broken CSS on any other developer's machine. No error — just missing utility classes. Fixed by making the nix build app inject the `@source` dynamically via `go list -m`.

### Did not investigate flake.nix changes

During the first session, `flake.nix` changes appeared (GONOSUMCHECK → GOPRIVATE, maintainers field). I did not make these changes intentionally and should have investigated immediately. Turned out to be from a prior commit — not harmful, but I failed to understand my own working tree.

### Search input icon overlay fragility

The user search box has a positioned icon overlay (`<span class="pointer-events-none absolute left-2.5...">`) that sits on top of `forms.Input`. It works today because `FormFieldWrapper` renders no wrapper div when `Label` is empty. But if anyone adds a `Label` to that Input, the form height grows and the icon misaligns silently.

---

## e) WHAT WE SHOULD IMPROVE

### Architecture

1. **Use `display.BadgeType` in view models** instead of `string` badge kinds + `badgeKindToType()` adapter — make impossible states unrepresentable
2. **Consolidate `icon()` into `icons.Icon()`** — two icon rendering paths is a minor split brain
3. **Adopt `navigation.SidebarNav`** for layout sidebar — eliminates 25 lines of hand-rolled nav

### Testing

4. **Add visual regression testing** — the structural smoke test catches missing elements but not visual breakage
5. **Test dark mode rendering** — no test verifies that components render correctly in `prefers-color-scheme: dark`

### Process

6. **Commit after each component adoption** — I batched Avatar+Button+Select+Input in one commit. Cleaner history = easier rollback.
7. **Investigate unexpected working tree changes immediately** — don't proceed with work when `git diff` shows changes you didn't make

### templ-components library feedback

8. **CSS-variable-based surface tokens** instead of hardcoded `bg-white` — eliminates fragile bridge
9. **Document `BaseProps.Class` struct literal gotcha** — promoted fields can't be set in struct literals
10. **Add `GridColsAutoFit`** — dashboard auto-fit/minmax pattern has no typed enum

---

## f) Up to 50 things to get done next

### P0 — Correctness & Stability

1. Fix search input icon overlay to not break when Label is added to forms.Input
2. Add `data-turbo="false"` or equivalent to prevent HTMX/turbo conflicts if consumer uses both
3. Verify admin-demo example renders correctly in a browser (never visually tested)
4. Run `nix run .#build-adminui-css` from a clean nix shell to verify dynamic @source works
5. Audit all remaining hand-rolled `style="background:var(--accent)"` patterns for dark mode correctness

### P1 — Architecture & Type Safety

6. Replace `string` badge kinds in `models.go` with `display.BadgeType` — eliminate `badgeKindToType()`
7. Replace `string` icon names with `icons.Name` typed constants in `models.go`
8. Consolidate `icon()` helper into `icons.Icon()` call (or document why the wrapper exists)
9. Adopt `navigation.SidebarNav` for layout sidebar
10. Adopt `navigation.Pagination` for user/tenant list pagination
11. Adopt `htmx.ConfirmDelete` for delete confirmation patterns
12. Adopt `htmx.CSRFToken` for CSRF meta tag rendering
13. Evaluate `display.Tabs` for tenant detail page sections
14. Evaluate `display.PageHeader` for page title blocks
15. Evaluate `navigation.Breadcrumbs` for back navigation links
16. Remove `navBg()` helper — replace with CSS-only active state

### P2 — Testing

17. Add structural test for `display.Card` rendering (border, header, body structure)
18. Add structural test for `display.Grid` rendering (grid class, column count)
19. Add structural test for `display.StatCard` rendering (value, label, icon)
20. Add structural test for `display.RelativeTime` rendering (`<time datetime>` element)
21. Add test for `forms.Select` option selected state correctness
22. Add test for `display.Button` with `Href` rendering as `<a>` vs `<button>`
23. Add dark mode rendering test (assert no hardcoded white backgrounds)
24. Add test for HTMX attribute passthrough on `display.Button` (`hx-post`, `hx-confirm`)
25. Add fuzz test for `initials()` function (unicode, empty, special chars)

### P3 — Documentation

26. Write migration guide for consumers upgrading from pre-templ-components adminui
27. Document the CSS bridge pattern in adminui README
28. Update adminui CHANGELOG.md with templ-components adoption
29. Document `build-adminui-css` workflow (when to recompile, how @source works)
30. Update templ-components SKILL.md with cqrs-htmx as a known consumer

### P4 — templ-components library improvements (cross-repo)

31. Fix hardcoded `bg-white` → use CSS custom property surface token
32. Document `BaseProps.Class` struct literal gotcha in consumer guide
33. Add `GridColsAutoFit` or `MinColWidth` to GridProps
34. Add `TitleClass`/`HeaderClass` to CardProps
35. Document dark mode strategy implications for `prefers-color-scheme` consumers
36. Add `DefaultGridProps()` constructor (consistency with other components)
37. Document Go module cache `@source` path pattern in tailwind-v4-adoption-guide.md
38. Consider `Header templ.Component` slot on CardProps (like Footer)

### P5 — Code Quality

39. Run `errorfamily` check on adminui (`branching-flow errorfamily .`)
40. Run full module isolation check (`nix run .#check-modules`)
41. Audit adminui dependency budget after adding templ-components
42. Check if `templ-components` pulls in any banned dependencies
43. Run `nix flake check` to verify formatting + devShells + apps
44. Verify `GOEXPERIMENT=jsonv2` is consistently set in all build paths
45. Clean up the json v2 migration remnants in usermgmt (compile errors we fixed)

### P6 — Future Consideration

46. Evaluate adopting `templ-components` `errorpage` package for admin error pages
47. Evaluate adopting `feedback.ToastContainer` + `feedback.Toast` (currently hand-rolled)
48. Evaluate adopting `htmx.GlobalErrorHandling` for HTMX error pipeline
49. Consider extracting the CSS theme bridge into a shared `theme-bridge.css` import
50. Consider whether adminui should ship its own `DefaultPageProps` that disables CDN htmx

---

## g) Top 2 Questions

### Q1: Should we fully adopt the `.dark` class strategy for dark mode?

adminui uses `@media (prefers-color-scheme: dark)`. templ-components uses `@custom-variant dark (&:where(.dark, .dark *))`. Currently all library `dark:` variants are inert in adminui. The `.bg-white` CSS bridge handles surface colors, but it's fragile. Full dark mode support would require:

- Adding `ThemeScript` + `ThemeToggle` from templ-components
- Switching from `prefers-color-scheme` to `.dark` class toggling
- Adding `.dark` class to `<html>` based on system preference or user toggle

This is a design philosophy question: should adminui follow the system preference silently (current) or offer a user-toggleable theme? I can't answer this without knowing the product direction.

### Q2: Should `badgeKindToType()` be eliminated in favor of typed `display.BadgeType` in models?

The current flow is: template calls `@badge("active", "green")` → `badgeKindToType("green")` → `display.BadgeType(BadgeSuccess)` → `display.Badge`. The string `"green"` is an intermediary that could be eliminated if `models.go` used `display.BadgeType` directly. But this couples the view model layer to the UI library. The alternative is keeping the string intermediary and accepting the mapping function as the price of decoupling. Which direction do you prefer?
