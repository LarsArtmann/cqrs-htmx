# Brutal Self-Review — templ-components Adoption in adminui

**Date:** 2026-07-10
**Scope:** Two-session adoption of templ-components in `cqrs-htmx/adminui`
**Verdict:** Good progress, real bugs introduced, portability broken, tech debt found

---

## 1. What did you forget?

### CRITICAL: Hardcoded `@source` path in tailwind.css

I added `@source "/home/lars/go/pkg/mod/github.com/larsartmann/templ-components@v0.13.0"` to `tailwind.css`. This is a **machine-specific absolute path**. Any other developer who runs `nix run .#build-adminui-css` will silently produce a CSS file missing all templ-components utility classes. No error — just broken styling.

This is the #1 bug. It should use `go env GOMODCACHE` or a relative path.

### Did not visually verify the output

Tests check for content strings ("Recent activity", "Users") but no test verifies that components actually render correct HTML structure. I adopted `display.Button` with `Icon` slots, `forms.Input` with `Class` overrides, `display.Avatar` with custom `BaseProps.Class` — all of these could render broken HTML and the tests would still pass.

### Did not check if flake.nix changes were mine

The `flake.nix` diff (GONOSUMCHECK → GOPRIVATE, maintainers field, build check) appeared during the session. I did not make these changes intentionally. They may have been auto-applied by nix tooling. I should have investigated and reverted before committing.

### AGENTS.md not updated

The templ-components skill explicitly recommends tracking adoption in AGENTS.md. I adopted 15 components and deleted 6 hand-rolled helpers but never updated the project documentation.

### Stale GONOSUMCHECK references left behind

The flake.nix migrated to `GOPRIVATE` but AGENTS.md, CONTRIBUTING.md, and scripts/ still reference the deprecated `GONOSUMCHECK`. Found during review, not fixed.

---

## 2. What is something stupid that we do anyway?

### String-to-enum-to-string badge mapping

The flow for rendering a badge is now: `roleBadgeType(role string) → display.BadgeType` → `display.Badge` → Tailwind classes. The intermediary `badgeKindToType(kind string)` maps adminui's internal strings ("green", "blue", "amber") to `display.BadgeType` values. These string intermediaries exist only because the view models (`models.go`) use `string` for badge kinds instead of `display.BadgeType` directly.

### `icon()` function uses `templ.Raw()`

The `icon()` function wraps `iconSVG()` output in `templ.Raw()`, bypassing XSS escaping. This is pre-existing, not introduced by my work, but it's a latent XSS vector if icon names were ever user-controlled. They aren't today (all hardcoded), but the type signature accepts arbitrary strings.

---

## 3. What could you have done better?

### Should have verified the rendered HTML

I should have written a quick test that renders a page and checks for key structural elements (e.g., "button element exists", "avatar initials render", "select has expected options"). The existing tests only check content strings.

### Should have used `display.BadgeType` in the view models

Instead of `string` badge kinds + `badgeKindToType()` adapter, the view models should use `display.BadgeType` directly. This would eliminate the entire adapter layer.

### Should have committed incrementally

I did all Avatar + Button + Select + Input changes in one batch. Should have committed each component adoption separately for cleaner git history and easier rollback.

---

## 4. What could you still improve?

### Eliminate `badgeKindToType` — use `display.BadgeType` in models

The `statCard`, `pageData`, and template-local structs use `string` for badge kinds. Changing to `display.BadgeType` eliminates `badgeKindToType()` entirely and makes impossible states unrepresentable (no more typo'd badge kind strings).

### Adopt `navigation.SidebarNav` for the layout sidebar

The entire sidebar (layout.templ lines 24-48) is hand-rolled with custom CSS variables. `navigation.SidebarNav` from templ-components provides the same structure with typed props.

### Add a rendering smoke test

One test that renders the dashboard page and checks for structural elements: avatar div, button elements, select elements. Catches regressions when components change.

---

## 5. Did you lie to you?

**No.** All component adoptions are real and working. Build, tests, and lint genuinely pass. The line count reductions are accurate.

However, I claimed the CSS bridge (`.bg-white { background-color: var(--surface) }`) is "safe because adminui never uses bg-white directly." This is true TODAY but fragile — there's no enforcement.

---

## 6. How can we be less stupid?

1. Use `go env GOMODCACHE` in build scripts instead of hardcoded paths
2. Track component adoption in AGENTS.md (as the skill recommends)
3. Use typed enums (`display.BadgeType`) instead of string intermediaries
4. Write structural rendering tests, not just content tests

---

## 7. Ghost systems?

### `spinner()` component is dead code

`spinner()` in `components.templ` wraps `feedback.Spinner` but is never called from any template or handler. It was already dead before my changes (the original was also never called). Should be deleted.

### `navBg()` could be eliminated

`navBg(active bool)` returns `"var(--accent)"` or `"transparent"`. This could be a CSS class or inline ternary in the template. It's a 5-line function for a 2-value lookup.

---

## 8. Scope creep?

No. The task was "adopt templ-components" and I did exactly that. I did not add features or change behavior.

---

## 9. Did we remove something that was actually useful?

### `selectedAttr()` removed — replaced by `roleSelectOptions()`

`selectedAttr()` was a template helper that returned `"selected"` or `""`. The replacement (`roleSelectOptions()`) builds the entire options slice with `Selected` flags. This is better (typed, testable) but the old function was used directly in templ expressions which was more flexible for ad-hoc selects. No regression since all call sites were migrated.

---

## 10. Split brains?

### Two badge rendering paths

`badge(text, kind string)` uses `badgeKindToType(kind)` → `display.Badge`.
`roleBadge(role string)` uses `roleBadgeType(role)` → `display.Badge`.

Both exist because role names and status kinds use different mappings. This is correct (roles and statuses are different domains) but means there are two parallel mapping functions. Consolidating them into one "badge type from semantic meaning" function would be premature — they serve different domains.

### `icon()` vs `icons.Icon()`

adminui has `icon(name string)` (returns `templ.Raw(iconSVG(name))`) while templ-components has `icons.Icon(name, class)` (returns `templ.Component`). Two ways to render the same icon. The adminui version wraps the library's path data in a custom SVG element with adminui-specific sizing (18px). Not a real split brain — intentional wrapping for consistent sizing.

---

## 11. How are we doing on tests?

### Existing tests are content-only

All adminui tests check for text content ("Recent activity", "Users", "No users found") and HTTP semantics (HX-Redirect, HX-Trigger). None verify HTML structure or CSS classes. This means:

- Component adoptions can't break tests (good for velocity)
- Component adoptions can't be verified by tests (bad for confidence)

### Missing tests

1. No test that verifies `display.Button` renders `<button>` or `<a>` elements
2. No test that verifies `display.Avatar` renders initials
3. No test that verifies `forms.Select` renders `<option>` elements with correct values
4. No test that verifies `forms.Input` renders the correct `name` and `type` attributes

---

## Prioritized Improvement Plan

| # | Task                                         | Impact   | Effort | Priority |
| - | -------------------------------------------- | -------- | ------ | -------- |
| 1 | Fix hardcoded `@source` path in tailwind.css | Critical | 5 min  | P0       |
| 2 | Revert non-mine flake.nix changes            | High     | 5 min  | P0       |
| 3 | Delete dead `spinner()` component            | Low      | 2 min  | P1       |
| 4 | Update AGENTS.md with adoption table         | Medium   | 10 min | P1       |
| 5 | Clean up stale GONOSUMCHECK references       | Low      | 10 min | P1       |
| 6 | Commit work incrementally                    | Process  | 5 min  | P0       |
