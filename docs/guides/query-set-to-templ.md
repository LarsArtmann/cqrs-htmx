# Query set → Props → View set

> The one-page guide to the props seam. Decision record:
> [`docs/adr/0068-props-seam-query-set-to-view-set.md`](../adr/0068-props-seam-query-set-to-view-set.md).
> Runnable proof: the root package's own specs (`views_test.go`,
> `query_set_test.go`) exercise the whole HTTP path — selection matrix,
> parallel loaders, PageOnly scoping, ETag 304. The `examples/basic`
> conversion and its browser gate land with the next root tag: examples must
> resolve PUBLISHED tags (gotcha 30/36), so a demo using unreleased symbols is
> red by construction until `v4.20.0` exists.

## The problem in one picture

templ composes on **Props**. A real HTMX page is three views over ONE view
model:

```
browser visit  ──► Page   (shell, nav, session, meta, full data)
hx-get swap    ──► Region (just the fragment the target needs)
rides along    ──► OOB    (fragments that update elsewhere)
```

Before the seam, the library's unit was one query → one component. That left
three options per page, all bad: a god query, hand-assembled props inside the
handler, or a second query just for the sidebar. The seam replaces all three:

```
query set   →   Props struct   →   view set (Page | Region | OOB)
(lawful)        (the only seam)      (the request selects one)
```

## Two invariants

1. **A query never returns a templ component.** Loaders produce data; the
   props struct holds it.
2. **A component never dispatches.** Views are pure presentation — given
   `Props`, they draw.

## The 20-line version

```go
type tenantsProps struct {
    Rows   []Tenant
    Counts map[identitymodel.TenantID]int
    Nav    Nav                        // PageOnly: a region swap skips it
}

set := cqrshtmx.NewQuerySet(
    // A dispatched query keeps its query/<Name> posture.
    cqrshtmx.LoadQuery("ListTenants", cqrshtmx.FromDispatcher(qryDisp),
        func(ctx context.Context) (query.Query, error) { return &listTenantsQuery{}, nil },
        func(p *tenantsProps, rows []Tenant) { p.Rows = rows }),
    // A plain function loader needs no dispatcher at all.
    cqrshtmx.Load("TenantMemberCounts", countMembers,
        func(p *tenantsProps, counts map[identitymodel.TenantID]int) { p.Counts = counts }).Optional(),
    cqrshtmx.Load("PageNav", loadNav,
        func(p *tenantsProps, nav Nav) { p.Nav = nav }).PageOnly(),
)

mux.Handle("GET /tenants", app.QuerySet(set,
    cqrshtmx.Views(
        cqrshtmx.Region("#tenant-rows", tenantsRows),          // hx-get hx-target="#tenant-rows"
        cqrshtmx.Region("#tenant-search", tenantsSearch).Default(),
        cqrshtmx.Page(tenantsPage),                            // direct visit / hx-boost
        cqrshtmx.OOB("tenant-count", tenantCount),             // rides region responses
    ),
    cqrshtmx.OnLoadError(func(r *http.Request, rep cqrshtmx.LoadReport) {
        slog.Warn("tenant counts degraded", "failures", len(rep.Failures))
    }),
))
```

## Selection rules (deterministic, request-derived)

| Request                                                   | Rendered                |
| --------------------------------------------------------- | ----------------------- |
| plain GET (browser visit)                                 | `Page`                  |
| `HX-Request` + `HX-Boosted` (htmx boost navigation)       | `Page`                  |
| `HX-Request` + `HX-History-Restore-Request`               | `Page`                  |
| `HX-Request` + `HX-Target` matching a `Region` selector   | that `Region`           |
| `HX-Request` + unmatched `HX-Target` + a `Default` region | the `Default` region    |
| `HX-Request` + unmatched `HX-Target`, no `Default`        | `400` `unroutable_view` |

The selector comparison tolerates one optional leading `#` on either side,
because htmx sends the raw element id while documentation usually writes
`#id`.

**Why the last row is an error and not a fallback:** a silent full-page
fallback makes htmx swap a whole document into a fragment target. That is the
2026-10-18 blanking class from the other direction, and it looks like a
design bug rather than a misdeclaration. If your page really wants "any
partial target gets the list", say so with `.Default()`.

## Loading rules

- **Parallel, then one apply pass.** Every needed loader runs in its own
  goroutine; the props struct is written by a single goroutine after the
  join. No field can race, and `PageOnly` skipping needs no coordination.
- **Fail loud, degrade explicitly.** A non-optional failure aborts the
  request. The reported error is the first failing loader in **declaration
  order** (completion order would be a flake), and the fail-fast cancellation
  of siblings never wins the report.
- **`Optional()`** is the only degradation path: the page renders with the
  zero field, and `LoadReport`/`OnLoadError` make it visible. Without a hook
  the failures are logged at WARN with loader names.
- **`PageOnly()`** skips a loader for targeted HTMX requests. The rule is
  exported as `LoadScopeFor(r)` so you can test your cost model directly.

## Composing with the rest of the library

| Composition   | How                                                                                                                                                                           |
| ------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| ETags         | `cqrshtmx.ETagResults()` after the render option: the tag hashes the rendered bytes, so 304 works for all three shapes.                                                       |
| Authorization | The `Authorize`/`AuthorizeIn`/`RequireAuth` options run before any loader. Dispatched loaders additionally answer to `query/<Name>` through the dispatcher's gate (ADR-0063). |
| Timeout       | `cqrshtmx.WithTimeout(d)` (or `Config.Timeout`) bounds the whole load phase.                                                                                                  |
| Bare mux      | `cqrshtmx.QuerySetHandler(set, opts...)` — same rules, no `App`, no pipeline; your middleware owns auth.                                                                      |
| Errors        | A fatal load error flows through the normal `App` error path (family → status), and layout/flash handling stays where it is.                                                  |

## Migrating a handler

Before:

```go
mux.Handle("GET /tenants", app.Query("ListTenants", decoder,
    cqrshtmx.RenderPartialOrFull(
        func(rows []Tenant) cqrshtmx.TemplComponent { return tenantsRows(rows) },
        func(rows []Tenant) cqrshtmx.TemplComponent { return tenantsPage(rows, nav) },
    )))
```

After: the same two components become `Region` and `Page`; the props struct
absorbs `nav`; anything the region cannot use becomes `PageOnly`. If that
conversion needs more handler code than it deletes, do not convert — the
degenerate helpers stay supported on purpose (ADR-0062 forbids deprecating
them for no gain).

## Deliberate non-goals

- No `view/` subpackage yet (it would need a `DispatchFunc` seam to avoid the
  module-layers back-edge; revisit past ~200 LOC).
- No per-view props types, no shell-first streaming, no per-page cache keys.
- No templ dependency: views are built from the structural `TemplComponent`.

## Reading list

- ADR-0068 — the decision and the rejected alternatives.
- `query_set_test.go` — a dispatched loader, a function loader and a
  `PageOnly` loader in one set (the shape `examples/basic` will adopt at the
  next root tag).
- `e2e/tests/root-htmx.spec.ts` — the browser proof (region shape,
  `PageOnly` scoping, history restore, boosted navigation).
