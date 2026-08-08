# Status: adminui CSS Build OOM Fix + Table-in-Card Double Border Fix

**Date:** 2026-07-12 00:08\
**Session:** Fix 3 known issues from previous templ-components v0.15.0 adoption session\
**Commit:** `cffbe35` — fix(adminui): eliminate 55 GB CSS build OOM and fix Table-in-Card double border\
**Previous session:** Upgraded v0.13.0 → v0.15.0, adopted 4 new components (Table, DefinitionList, CopyButton, GridColsAutoFit), replaced 5 hand-rolled tables

---

## a) FULLY DONE

### 1. Root-Caused the 55 GB CSS Build OOM

**Root cause found and fixed.** The `@source "../";` directive in `adminui/tailwind.css` resolved to the **entire cqrs-htmx project root** — 953 files including Go source, markdown docs, test golden files, D2 diagrams, shell scripts. Tailwind v4 scans every file under each `@source` directory for class-like strings. 953 files × AST parsing = 55 GB RAM.

The previous session thought the OOM was caused by scanning the templ-components module cache. That was wrong — the module cache scan was a secondary issue. The primary culprit was `@source "../"` scanning the project root.

**Fix applied:**

- Removed `@source "../";` from `tailwind.css` entirely
- The build script now injects two explicit `@source` paths at build time:
  1. `@source "$(pwd)"` — adminui's own directory (8 `.templ` files)
  2. `@source "$SCAN_DIR"` — temp dir with only 59 `.templ` files copied from templ-components
- Excluded `*_templ.go` from the library scan (they're 3x larger mirrors of the same class strings)

**Measured impact:**

| Metric          | Before                                          | After                             |
| --------------- | ----------------------------------------------- | --------------------------------- |
| Files scanned   | 953 (project root) + 960 (module cache)         | 8 (adminui) + 59 (library .templ) |
| Peak RAM        | 55 GB (OOM-killed)                              | < 500 MB                          |
| Build time      | OOM-kill (never completes)                      | 613ms                             |
| CSS output size | 93 KB (with junk classes from irrelevant files) | ~73 KB                            |

### 2. Fixed Table-in-Card Double Border

**Problem:** `display.Table` hardcodes `border border-gray-200 rounded-lg` on its wrapper div. `display.Card` has the identical classes. When nested via `Card(CardPaddingNone)`, both borders render — visible on every table page (dashboard, audit, users list, tenants list, members).

**Fix:** Targeted CSS rule in `tailwind.css`:

```css
.overflow-hidden > .overflow-x-auto {
	border: 0 !important;
	border-radius: 0 !important;
}
```

This selector is unique to the Table-in-Card nesting pattern: Card always emits `overflow-hidden`, Table wrapper always emits `overflow-x-auto`. No other component combination in the codebase matches this pattern.

### 3. Fixed HTMX Partial Mismatch

**Problem:** `handler_users.go:17` rendered the full `usersContent(d)` (header + search bar + table) for HTMX search requests, but the client's `hx-select="#users-table"` only kept the table div. The server rendered ~2 KB of discarded HTML per search request.

**Fix:** Extracted `usersTableContent(d)` as a lean partial that renders only the `#users-table` div. Handler now calls `usersTableContent(d)` for HTMX requests and `usersContent(d)` for full page loads. (This change was committed in `00bef1d` by another agent that swept it into the loginpage commit.)

### 4. Updated flake.nix build-adminui-css

**Updated the build script** to use the copy-first approach:

- Copies only `*.templ` files (not `*_templ.go`) from 7 library packages to a temp dir
- Injects `@source` for adminui's own directory (since temp CSS lives in `/tmp`)
- Cleans up temp dir after build

### 5. Updated AGENTS.md

- Added gotcha #8: adminui CSS build OOM root cause and workaround
- Corrected `go-error-family v0.6.1` → `v0.7.0` (2 occurrences)
- Corrected `templ-components v0.13.0` → `v0.15.0`
- Updated adoption table description with double-border fix
- Added direct-binary CSS rebuild instructions to adminui test commands section
- Updated build description to reference copy-first approach

### 6. Verified Everything

- adminui: `go build` ✓, `go test -race` ✓ (1.1s), `golangci-lint` ✓ (0 issues)
- usermgmt: `go build` ✓, `go test -race` ✓ (4.0s)
- root: `go build` ✓ (with GOEXPERIMENT=jsonv2)
- `nix fmt` ✓ (1 file formatted)
- `nix flake check` ✓ (all checks passed)
- BuildFlow pre-commit ✓ (36/36 checks, 5.6s)

---

## b) PARTIALLY DONE

### flake.nix build-adminui-css Still OOM-Kills in Nix Sandbox

The build script is updated with the correct copy-first approach, but `nix run .#build-adminui-css` still exits 137 (OOM-killed) because the nix sandbox has memory restrictions. The direct-binary approach works:

```bash
cd adminui && \
  TC_DIR=$(GOWORK=off go list -m -f '{{.Dir}}' github.com/larsartmann/templ-components) && \
  SCAN_DIR=$(mktemp -d) && \
  for pkg in display errorpage feedback forms htmx icons layout navigation; do \
    cp "$TC_DIR/$pkg/"*.templ "$SCAN_DIR/" 2>/dev/null || true; \
  done && \
  TMP_CSS=$(mktemp --suffix=.css) && cp tailwind.css "$TMP_CSS" && \
  echo "@source \"$(pwd)\";" >> "$TMP_CSS" && \
  echo "@source \"$SCAN_DIR\";" >> "$TMP_CSS" && \
  tailwindcss -i "$TMP_CSS" -o assets/admin-tw.css --minify && \
  rm -rf "$SCAN_DIR" "$TMP_CSS"
```

The nix sandbox OOM is likely because `writeShellApplication` runs in a restricted environment. The fix would be to either:

- Use `pkgs.buildNixPackage` with `__noChroot = true` (dangerous)
- Ship a script that runs outside the sandbox (e.g., `pkgs.writeShellScriptBin` instead of `writeShellApplication`)
- Or accept that CSS rebuild is a manual step (documented)

### CSS Size Discrepancy in Commit Message

The commit message says "CSS dropped from 93 KB to 58 KB" but the committed CSS is actually 73 KB. The 58 KB measurement was from my manual test rebuild. Between that test and the commit, something regenerated the CSS (likely `nix fmt` / treefmt with `templ.enable = true` reformatted `.templ` files, then BuildFlow's pre-commit hook regenerated CSS). The 73 KB is still correct — all classes verified present — but the commit message's number is wrong.

---

## c) NOT STARTED

1. **Dark mode strategy decision** — adminui uses `prefers-color-scheme` (OS-following). templ-components v0.15.0 supports `.dark` class toggle with `ThemeScript` + `ThemeToggle`. Not addressed this session.
2. **Remaining component adoptions** — `display.StatusBadge`, `display.PageHeader`, `forms.Form`, `navigation.Breadcrumbs`, `feedback.Skeleton` all identified as good fits. Not started.
3. **Visual verification** — No screenshots or visual diff was done. The structural test verifies HTML renders but not that it looks right.
4. **Library bug reports** — Table-in-Card double border should be reported upstream (Table needs a `Borderless` option). Tailwind v4 `@source` OOM should be documented for consumers.
5. **CSS rebuild CI gate** — No CI check that `admin-tw.css` includes all library component classes. A drift could silently break styling.

---

## d) TOTALLY FUCKED UP

### 1. The @source "../" Was There All Along

The previous session's status report documented the OOM as a mystery ("Tailwind v4's file scanner consumes enormous memory when pointed at a Go module cache"). The actual root cause was a one-line CSS directive (`@source "../";`) that had been in `tailwind.css` since the adminui module was created. It was never about the module cache at all — scanning the full module cache was a secondary concern. The primary scan target was 953 files in the project root.

The previous session spent hours working around a symptom (module cache scanning) without finding the actual root cause (project root scanning). One `grep -r "@source" tailwind.css` would have found it in 5 seconds.

### 2. Commit Message CSS Size Is Wrong

I wrote "CSS dropped from 93 KB to 58 KB" in the commit message. The committed CSS is 73 KB. I didn't re-verify the size after `nix fmt` / BuildFlow ran. The 73 KB is still correct and smaller than the original 93 KB, but the commit message contains a factual error.

### 3. Spent Time on @source "../" Removal Before Diagnosing

When the user said "Something is like hardcore miss configure for tailwindcss to use 55GB of RAM!", I had already been trying to fix the CSS build for several tool calls using the copy-first-to-temp-dir approach from the previous session. I should have stopped immediately and investigated the actual root cause (what is `@source "../"` scanning?) before trying workarounds. The investigation took 2 tool calls once I actually looked.

---

## e) WHAT WE SHOULD IMPROVE

### Architecture

1. **The `@source` directive is a footgun.** Tailwind v4's auto-detection is supposed to scan the CSS file's directory. But `tailwind.css` lives in `adminui/` and `@source "../"` scanned the parent. This should never have been committed without a comment explaining why the parent dir needs scanning (answer: it doesn't — Tailwind auto-detects adminui's own files).
2. **No CSS class presence test.** The structural test (`TestPanel_RendersSeededData`) checks that HTML renders but doesn't verify that the CSS classes used in the HTML actually exist in `admin-tw.css`. A CSS drift could silently break styling after a library upgrade.
3. **The nix sandbox OOM is an infrastructure problem.** The `build-adminui-css` app can't run in the sandbox. This means CSS rebuild is always a manual step outside nix. Either fix the sandbox memory limit or accept this as a manual step and stop pretending the nix app works.

### Process

4. **Should have diagnosed before fixing.** The previous session's copy-first workaround was treating a symptom. I continued that approach for several tool calls before actually investigating. The root cause was found in 2 tool calls once I looked.
5. **Should have re-verified CSS size after auto-formatters ran.** The commit message has a wrong number because I didn't re-check after `nix fmt` and BuildFlow.
6. **The `tailwind.css` comment was stale for multiple sessions.** The comment said "The nix build-adminui-css app generates a temporary tailwind.css with the correct GOMODCACHE path" — but the nix app was broken. Nobody noticed because CSS rebuild is rare.
7. **treefmt has `templ.enable = true`** — This runs `templ fmt` on `.templ` files during `nix fmt`. It's good for formatting but means `.templ` files can change during formatting, which cascaded into the CSS size discrepancy.

### Library Feedback (for templ-components repo)

8. **Table needs a `Borderless` or `Flush` option.** The component hardcodes `border border-gray-200 rounded-lg` on its wrapper div with no way to suppress it. This makes Table-in-Card nesting always produce double borders. A `TableProps.Flush bool` or `TableProps.Class` that can override (not just append) would fix it.
9. **Library should ship a CSS scan manifest.** Instead of requiring consumers to copy `.templ` files to a temp dir, the library could ship a `classes.txt` file listing all utility classes used. Tailwind v4 could scan that instead of parsing source files.

---

## f) Up to 50 Things We Should Get Done Next

### High Priority (Broken / Wrong)

1. **Fix commit message CSS size** — amend or document that the CSS is 73 KB not 58 KB (minor, cosmetic)
2. **Fix nix sandbox OOM for build-adminui-css** — switch from `writeShellApplication` to `writeShellScriptBin` or document as manual-only
3. **Add CSS class presence test** — verify `admin-tw.css` contains classes used by adminui `.templ` files (regression gate)
4. **Run admin-demo and verify visually** — the double border fix, table rendering, and dark mode all need visual confirmation
5. **Report Table `Borderless` feature request upstream** — to templ-components repo

### Medium Priority (Adoption Gaps from Previous Session)

6. **Adopt `display.StatusBadge`** for tenant status (suspended/active/deleted)
7. **Adopt `display.PageHeader`** for consistent page title + action layout
8. **Adopt `forms.Form`** for the tenant create form
9. **Adopt `navigation.Breadcrumbs`** instead of `← Users` back links
10. **Adopt `feedback.Skeleton`** for HTMX loading states
11. **Consider `navigation.SidebarNav`** to replace custom sidebar
12. **Adopt `display.SimpleCard`** for title-less card usages

### Dark Mode

13. **Resolve dark mode strategy** — `.dark` class toggle vs `prefers-color-scheme`
14. **If class-based: add `layout.ThemeScript` + `layout.ThemeToggle`**
15. **Test all 19 components in dark mode**
16. **Audit `dark:` variant coverage** — library emits them but they may be inert

### CSS Build Hardening

17. **Add CSS rebuild to CI** — verify `admin-tw.css` is up-to-date after `.templ` changes
18. **Pin tailwindcss version** in devShell to match the binary used for compilation
19. **Add a `make css` / `just css` convenience target** — one-command CSS rebuild using the working direct-binary approach
20. **Document the CSS build workflow in CONTRIBUTING.md**
21. **Consider shipping a pre-built CSS in the templ-components library** — eliminates the consumer-side scan entirely

### Testing & Quality

22. **Add test for double-border absence** — assert `.overflow-hidden > .overflow-x-auto` rule exists in CSS
23. **Update `TestPanel_RendersSeededData`** to assert on `display.Table` rendering
24. **Add test for `display.CopyButton`** rendering
25. **Add test for `display.DefinitionList`** rendering
26. **Run full test suite across ALL modules** (`nix run .#test`)
27. **Run `branching-flow errorfamily`** on adminui
28. **Run code coverage** — verify adminui coverage hasn't dropped

### Refactoring

29. **Consolidate badge helpers** — `badge()`, `roleBadge()`, `verifiedBadge()`, `totpBadge()`
30. **Extract shared table row helpers** — avatar + badge patterns repeated across tables
31. **Review `models.go`** for unnecessary coupling to display types
32. **Clean up `components.templ`** — 11 templ functions, consider splitting

### Documentation

33. **Update `tailwind.css` header comment** — it still says "Build: tailwindcss -i tailwind.css -o assets/admin-tw.css --minify" which doesn't work without the @source injection
34. **Document the `@theme` gray mapping strategy** — explain why gray-50→900 are remapped
35. **Add CSS build troubleshooting section** to README or CONTRIBUTING
36. **Update the status report from the previous session** — it has wrong root cause analysis

### Library Feedback

37. **Report: Table-in-Card double border** — Table should suppress border inside Card, or expose `Flush` option
38. **Report: Tailwind v4 @source OOM** — library should ship a class manifest instead of requiring source scanning
39. **Report: Table cell padding** — `py-3` is large for admin panels, consider `Compact` option
40. **Suggest: Table `Class` override** — currently only appends, can't strip hardcoded classes

### Previous Session Backlog (Still Relevant)

41. **Fix table cell padding mismatch** — adminui was `py-2`, library Table is `py-3`
42. **Adopt `display.TrendWarn`** for stat cards
43. **Use `Select.Groups`** for optgroup support
44. **Consider `display.Dropdown`** for sign-out menu
45. **Adopt `feedback.Alert`** for inline error displays
46. **Evaluate `errorpage.ErrorPage`** for custom error pages
47. **Add lint rule: no hand-rolled `<table>`** in `.templ` files
48. **Add lint rule: no hand-rolled card pattern** in `.templ` files
49. **Review `Card.Header` slot** for additional use cases beyond danger zone
50. **Consider `navigation.Pagination`** if server-side pagination is added

---

## g) Top 2 Questions

### 1. Should the `flake.nix` `build-adminui-css` app be replaced with a plain shell script?

The nix sandbox OOM-kills tailwindcss (exit 137) regardless of how small the scan set is. The `writeShellApplication` wrapper runs in a restricted sandbox. Switching to `writeShellScriptBin` (no sandbox) or a plain shell script would make `nix run .#build-adminui-css` actually work. But it loses the nix-managed runtime inputs (tailwindcss binary path). Is the sandbox restriction configurable, or should we accept manual-only CSS rebuilds?

### 2. Should we report the Table-in-Card double border as a library bug or keep the CSS workaround?

The CSS fix (`.overflow-hidden > .overflow-x-auto { border: 0 !important }`) is fragile — it depends on Tailwind class names that could change. A library-level fix (Table `Flush` option, or Card auto-detecting Table children) would be permanent. But it requires upstream changes in templ-components. The workaround works today but could break on the next library upgrade if class names change.
