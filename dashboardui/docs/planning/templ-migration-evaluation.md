# Templ Migration Evaluation — dashboardui

## Context

dashboardui renders all HTML via `strings.Builder` with `html.EscapeString` (aliased as `esc()`). Every page is a Go function that builds HTML string fragments. This document evaluates whether migrating to [templ](https://templ.guide) (a-h/templ) would improve the codebase.

## Current Approach

### Pattern

```go
func (d *Dashboard) renderEvents(p pageData, events []event.Event, ...) string {
    return d.renderLayout(p, func() string {
        var b strings.Builder
        fmt.Fprintf(&b, `<table class="data-table"><thead>...`)
        for _, evt := range events {
            fmt.Fprintf(&rows, `<tr><td>%s</td>...`, esc(string(evt.Type())))
        }
        b.WriteString(rows.String())
        b.WriteString(`</tbody></table>`)
        return b.String()
    })
}
```

### Stats

- **HTML-generating files**: 12 (handlers_events.go, handlers_audit.go, handlers_projections.go, handlers_timetravel.go, handlers_snapshots.go, handlers_dlq.go, handler_overview.go, layout.go, pagination.go, render.go, sort.go, export.go)
- **Total lines**: ~2,800
- **Template fragments**: ~30 distinct render functions
- **XSS protection**: Manual `esc()` calls on every interpolated value
- **No template files**: Zero `.templ` or `.html` template files

## Evaluation

### Benefits of Templ

| Benefit                   | Impact | Notes                                                           |
| ------------------------- | ------ | --------------------------------------------------------------- |
| **Type-safe HTML**        | High   | Compile-time guarantee that HTML is well-formed                 |
| **Automatic escaping**    | High   | Eliminates manual `esc()` calls, removes XSS risk surface       |
| **Component composition** | Medium | Layout/table/card components become reusable                    |
| **Better tooling**        | Medium | LSP support, syntax highlighting, formatting                    |
| **Reduced boilerplate**   | Medium | No more `fmt.Fprintf(&b, ...)` string building                  |
| **Tests on output**       | Low    | Current tests check string contains — templ doesn't change this |

### Costs of Migration

| Cost                       | Impact | Notes                                                                    |
| -------------------------- | ------ | ------------------------------------------------------------------------ |
| **New build dependency**   | Medium | templ codegen required; `_templ.go` committed (existing pattern in repo) |
| **Large migration effort** | High   | ~2,800 lines across 12 files, ~30 render functions                       |
| **Module isolation**       | Low    | dashboardui/v4 is a submodule; templ dep stays local                     |
| **Risk of regressions**    | Medium | Every page changes; need thorough testing                                |
| **Learning curve**         | Low    | Templ syntax is straightforward for Go developers                        |

### Risks

1. **XSS regression**: Current code manually escapes. A templ migration must verify every interpolation is auto-escaped. Templ does this by default, but the migration itself is the risk window.

2. **Conditional rendering complexity**: The dashboard has extensive capability-based conditional rendering (`if d.caps.EventSource`, `if !p.ReadOnly`). This maps to templ conditionals but adds verbosity.

3. **Performance**: Templ generates Go code that writes directly to `io.Writer`. The current `strings.Builder` approach also writes to a buffer. Performance should be equivalent or slightly better (no intermediate string).

4. **CSP compatibility**: The daemon already moved inline JS to CSP-safe patterns. Templ doesn't affect JS handling — it's purely HTML generation.

### Decision Matrix

| Criterion             | Stay with strings.Builder         | Migrate to Templ        |
| --------------------- | --------------------------------- | ----------------------- |
| XSS safety            | Manual (error-prone)              | Automatic               |
| Maintainability       | Low (string concatenation)        | High (typed components) |
| Build complexity      | Zero                              | Adds templ codegen      |
| Migration cost        | Zero                              | ~2,800 lines            |
| Consistency with repo | Inconsistent (adminui uses templ) | Consistent              |
| Performance           | Good                              | Equivalent              |

## Recommendation

**Defer migration.** The current approach works, tests pass, and the dashboard is functionally complete. A templ migration would improve type safety and reduce XSS risk, but:

1. The dashboard is a **read-only observability tool** — it doesn't handle user input directly (no forms except reset/replay actions which already have CSRF protection).
2. The manual `esc()` pattern is consistent and well-tested.
3. Migration effort (~2,800 lines) is better spent on features.

**Revisit when**: adding complex forms, if XSS vulnerability is found, or if the codebase grows beyond ~5,000 lines of HTML generation.

## Migration Plan (if pursued later)

1. Start with leaf components: `statCard`, `metaRow`, `emptyState`, `renderPagination`
2. Migrate one page at a time: overview → events → commands → projections → etc.
3. Keep `renderLayout` as the wrapper — convert inner content to templ
4. Add `// templ: skip` equivalent for test-only paths
5. Verify every test still passes at each step

## Alternative: html/template

Go's stdlib `html/template` provides auto-escaping without external deps. However:

- It's string-based (no type safety on template data)
- Template parsing adds startup cost
- Less ergonomic than templ for component composition

**Not recommended** — templ is strictly better than html/template for this use case.
