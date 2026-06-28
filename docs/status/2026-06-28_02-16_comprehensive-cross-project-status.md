# Status Update — 2026-06-28 02:16

**Scope:** templ-components (v0.4.0+) + cqrs-htmx/adminui (v3.1.0)
**Session span:** 2026-06-27 → 2026-06-28 (multi-session, ~6 hours of work)

---

## A) FULLY DONE

### templ-components

| Item                           | Evidence                                                                      |
| ------------------------------ | ----------------------------------------------------------------------------- |
| CHANGELOG v0.4.0 entry         | CHANGELOG.md — documents 4 components, 3 icons, Tailwind v4 adoption          |
| FEATURES.md at v0.4.0          | 73 components, 101 icons, 51 generated files — all counts verified            |
| README accuracy                | Forms count 12→16, ecosystem cross-links (GOTH stack)                         |
| AGENTS.md corrections          | 46→51 generated files, SanitizeID lie fixed, new conventions documented       |
| CONTEXT.md + TODO_LIST.md      | All metrics refreshed to v0.4.0 numbers                                       |
| ButtonHTMLType enum            | `ButtonProps.Type` typed (was raw `string`), 4 tests, backward-compatible     |
| FormMethod validation          | `formMethod()` normalizer with GET fallback, 5 tests                          |
| AvatarStatus invisible-dot fix | Only `online`/`offline` render a dot, 2 regression tests                      |
| `utils.Version` + drift guard  | Single source of truth, test asserts match with CHANGELOG                     |
| Stale colors.css refs fixed    | Self-contradicting status report annotated                                    |
| `go get @v0.4.0` verified      | Compiles and runs from clean external project                                 |
| `admin-tw.css` embed fix       | Ghost artifact now embedded, served, linked, tested                           |
| Dead CSS stripped              | admin.css: 706→105 lines (601 lines removed)                                  |
| admin.js selector bugs fixed   | `.admin-sidebar`, `.admin-toggle`, `.admin-scrim`, `.toast-host` all restored |
| badgeColor() theme-aware       | Uses CSS variables instead of hardcoded hex                                   |
| All TODOs resolved             | 0 actionable open items remaining (2 deferred to v1.0, documented)            |

### cqrs-htmx

| Item                                                 | Evidence                                                                              |
| ---------------------------------------------------- | ------------------------------------------------------------------------------------- |
| All 7 adminui `.templ` files migrated to Tailwind v4 | layout, dashboard, audit, members, tenants, users, components                         |
| templ-components icons adopted                       | 58 lines of hardcoded SVG paths → `icons.IconPathData()`                              |
| `DomainsForUser` returns `[]TenantID`                | Was `[]string`, now type-safe                                                         |
| 6 stale TODOs marked done                            | foldUser, BotState.OwnerID, actorKindFromString, NewActorID, TenantState, LastEventID |
| 6 remaining TODOs investigated + resolved            | All verified with code-level evidence (see section C)                                 |

---

## B) PARTIALLY DONE

| Item                                             | Status                | What's missing                                                                                                                                                                                                                                                 |
| ------------------------------------------------ | --------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| adminui Tailwind migration                       | ~90%                  | admin.css still loaded (as token layer — provides CSS variables). admin-tw.css compiled but minified to 1 line. 3 admin.css class refs remain in layout.templ (`.admin-sidebar`, `.admin-toggle`, `.admin-scrim` — these are queried by admin.js, not styling) |
| Component adoption (templ-components in adminui) | ~15%                  | Only icons adopted. Badge, Button, Card, Table, Form controls still use Tailwind utility classes directly                                                                                                                                                      |
| awesome-templ / templ.guide listing              | Entry prepared        | Needs manual PR submission to external repos                                                                                                                                                                                                                   |
| Snapshot → golden test conversion                | 12 golden files exist | 60+ assertion-based tests remain as-is (work fine)                                                                                                                                                                                                             |

---

## C) NOT STARTED (Explicitly Deferred with Justification)

| Item                                      | Why deferred                                                                                           |
| ----------------------------------------- | ------------------------------------------------------------------------------------------------------ |
| `Validate() error` on props structs       | Changes silent-fallback design philosophy. Needs per-component methods (73 components). v1.0 scope.    |
| Move test helpers to `internal/testutil/` | Breaking change for 70 test files + external consumers. v1.0 scope.                                    |
| ActorID context typing (root vs usermgmt) | Cross-module type coupling. Root can't import usermgmt. Needs shared identity module.                  |
| Email branded type in all domain structs  | Would change JSON event serialization for stored events. Needs upcaster migration. Next major version. |
| WebAuthn `*http.Request` extraction       | go-webauthn library API requires `*http.Request` directly. Blocked by upstream.                        |
| Snapshot integration                      | Premature optimization. Only beneficial at >10K events per aggregate.                                  |
| Goreleaser                                | N/A — library, not binary application.                                                                 |
| `go:generate stringer`                    | Not feasible — stringer doesn't support string-backed enums.                                           |
| go.work modularization                    | Not beneficial — single-module library.                                                                |
| Documentation site                        | pkg.go.dev already serves API docs.                                                                    |
| A11y audit automation                     | Component library has no live DOM. Defer to consumer E2E tests.                                        |

---

## D) TOTALLY FUCKED UP (and fixed)

| Bug                                       | Impact                                                                   | How it was caught                                 | Fix                                                     |
| ----------------------------------------- | ------------------------------------------------------------------------ | ------------------------------------------------- | ------------------------------------------------------- |
| `admin-tw.css` never embedded             | Tailwind CSS invisible to running app. "Infrastructure ready" was a lie. | Brutal self-review — checked `go:embed` directive | Added to embed, route, layout link, test                |
| `.admin-sidebar` → `id="admin-sidebar"`   | Mobile sidebar toggle completely broken. admin.js queries by class.      | Self-review — traced admin.js selectors           | Restored class, removed inline onclick                  |
| `.admin-toggle` removed                   | Hamburger button invisible to admin.js event delegation                  | Same review pass                                  | Restored class                                          |
| `.admin-scrim` → `id="admin-scrim"`       | Backdrop click doesn't close mobile sidebar                              | Same review pass                                  | Restored class                                          |
| `.toast-host` → `id="toast-host"`         | Toast notification system completely broken                              | Same review pass                                  | Restored class                                          |
| `badgeColor()` hardcoded hex              | Badges ignore theme changes (dark mode, accent override)                 | Self-review — checked CSS variable usage          | Changed to `var(--info)`, `var(--ok)`, etc.             |
| 601 lines of dead CSS                     | admin.css had all old component classes but no `.templ` file used them   | Verified via grep — zero class refs               | Stripped to 105 lines (tokens + htmx-indicator + toast) |
| CHANGELOG said "Nothing yet" after v0.4.0 | Shipped a release with no changelog entry                                | Self-review — read CHANGELOG top                  | Added full v0.4.0 entry                                 |
| FEATURES.md frozen at v0.3.0              | Every metric wrong (69 vs 73 components, 99 vs 101 icons)                | Self-review — cross-checked counts                | Overhauled to v0.4.0                                    |
| AGENTS.md SanitizeID claim                | Said "not used internally" — it IS used in radio_go.go:55                | Self-review sub-agent                             | Corrected                                               |
| AGENTS.md generated file count            | Said 46, actual 51                                                       | Direct count                                      | Corrected                                               |
| README forms count                        | Said 12, actual 16 (contradicted its own total of 73)                    | Cross-check arithmetic                            | Fixed                                                   |
| Demo hardcoded version                    | `v0.4.0` string would rot on next release                                | Self-review                                       | `utils.Version` + drift-guard test                      |

---

## E) WHAT WE SHOULD IMPROVE

### Architecture

1. **The four-doc count split brain**: Component/icon/coverage counts are hand-maintained in README, FEATURES, AGENTS, and CONTEXT. They drift every release. A code-generated stats file or test assertion would eliminate this permanently.
2. **admin.css token migration to tailwind.css**: The CSS variables (`--accent`, `--surface`, etc.) still live in admin.css. They should move into `tailwind.css`'s `@theme` block so there's one stylesheet, not two.
3. **Dark mode strategy**: admin.css uses `@media (prefers-color-scheme)` while tailwind.css uses `.dark` class variant. Pick one (the `.dark` class is better — user-toggleable).
4. **Component adoption in adminui**: Currently using raw Tailwind classes for badges/buttons/cards. Could replace with `display.Badge`, `display.Button`, `display.Card` from templ-components for consistency.

### Type Safety

5. **`Validate() error` opt-in**: Instead of replacing silent fallbacks, add an optional `Validate()` method that consumers can call during development/debugging to catch typos. Non-breaking.
6. **Branded Email type for new fields**: Can't change existing event payloads, but new structs should use `Email` from the start.

### Testing

7. **Golden file tests for all new components**: Policy — every new component gets a golden test. Old tests stay as-is.
8. **Integration test: adminui with real browser**: The selector bug would have been caught by a Playwright/Selenium test that clicks the hamburger button.

### Process

9. **Tag v0.5.0**: Post-v0.4.0 changes (ButtonHTMLType, FormMethod, AvatarStatus, Version, bug fixes) need a release tag.
10. **Pre-release checklist**: A script that verifies CHANGELOG ↔ FEATURES ↔ README ↔ AGENTS counts match before tagging.

---

## F) TOP 25 THINGS TO DO NEXT

Sorted by impact ÷ effort.

| #   | Task                                                                          | Repo | Effort | Impact   |
| --- | ----------------------------------------------------------------------------- | ---- | ------ | -------- |
| 1   | Tag v0.5.0 (post-v0.4.0 changes shipped but untagged)                         | T    | 5m     | **High** |
| 2   | Move CSS variables from admin.css → tailwind.css `@theme` (single stylesheet) | C    | 20m    | **High** |
| 3   | Pick `.dark` class dark-mode strategy, remove `prefers-color-scheme`          | C    | 15m    | **High** |
| 4   | Add pre-release count-verification test (components ↔ README ↔ FEATURES)      | T    | 15m    | **High** |
| 5   | Replace adminui badges with `display.Badge` component                         | C    | 20m    | Med      |
| 6   | Replace adminui buttons with `display.Button` component                       | C    | 25m    | Med      |
| 7   | Replace adminui cards with `display.Card` / `StatCard`                        | C    | 30m    | Med      |
| 8   | Replace adminui tables with `display.Table` component                         | C    | 25m    | Med      |
| 9   | Submit awesome-templ PR (entry content prepared)                              | T    | 10m    | Med      |
| 10  | Submit templ.guide PR (entry content prepared)                                | T    | 10m    | Med      |
| 11  | Add `Validate() error` as opt-in dev tool on 3 pilot components               | T    | 30m    | Med      |
| 12  | Delete admin.css entirely (after tokens move to tailwind.css)                 | C    | 5m     | Med      |
| 13  | Add Playwright E2E test for adminui (catch selector bugs)                     | C    | 45m    | **High** |
| 14  | Adopt `forms.Input` in adminui (replaces raw `<input>` Tailwind classes)      | C    | 20m    | Low      |
| 15  | Adopt `forms.Select` in adminui                                               | C    | 15m    | Low      |
| 16  | Add `display.PageHeader` to adminui pages (consistent page titles)            | C    | 15m    | Low      |
| 17  | Add `navigation.SidebarNav` to replace hand-built sidebar                     | C    | 25m    | Med      |
| 18  | Add `display.EmptyState` to adminui empty states                              | C    | 10m    | Low      |
| 19  | Expand demo to showcase remaining ~25 components                              | T    | 90m    | Low      |
| 20  | Design shared identity module for ActorID (root + usermgmt)                   | C    | 60m    | Med      |
| 21  | Add Email branded type to new (non-event) structs                             | C    | 20m    | Low      |
| 22  | Write v1.0 API freeze document (formal ADR)                                   | T    | 30m    | Med      |
| 23  | Add adminui handler test for admin-tw.css content (not just status code)      | C    | 10m    | Low      |
| 24  | Create `CONTRIBUTING.md` section on "how to add a new component"              | T    | 20m    | Low      |
| 25  | Evaluate `go-webauthn` fork for `*http.Request` removal                       | C    | 60m    | Med      |

---

## G) TOP QUESTION I CAN'T FIGURE OUT MYSELF

**#1: Should adminui fully adopt templ-components components (Badge, Button, Card, Table, etc.), or stay with raw Tailwind utilities?**

Arguments for adoption:

- Single source of truth for styling
- Consistent with the library's purpose
- Less code in adminui

Arguments against:

- adminui needs fine-grained control (inline styles for CSS variables like `style="background:var(--accent)"`)
- templ-components don't support CSS variable theming (they emit hardcoded Tailwind classes like `bg-blue-600`)
- The components might not match adminui's exact design (rounded corners, shadows, spacing)
- Adoption would add a tight coupling between adminui and templ-components release cycle

**The core tension**: templ-components uses Tailwind class strings (`bg-blue-600`) that consumers override via `@theme { --color-blue-600: #custom; }`. But adminui uses CSS custom properties directly (`var(--accent)`). These are two different theming strategies. Adopting templ-components would force adminui to switch to the `@theme` override approach, abandoning `var(--accent)` inline styles.

**I need your call**: Do we commit to the `@theme` override strategy across all projects, or do we keep adminui's CSS-variable approach and only adopt templ-components for non-styled parts (icons, which we already did)?
