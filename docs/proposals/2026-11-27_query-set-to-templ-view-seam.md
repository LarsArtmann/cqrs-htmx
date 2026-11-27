# Design note: mapping `Query` / `QuerySet` onto templ

**Filed:** 2026-11-27 · **Status: DESIGN NOTE (exploration, nothing implemented, no owner tick needed yet)** · **Scope:** root module render surface (`options_render.go`, `options_types.go`, `partial.go`) plus a possible new root-module `view/` subpackage.

> Question that started this: *how can the concept of a `Query` or `QuerySet` map to templ in a smarter way architecturally?*
> Answer in one line: **stop mapping query → component, and map query set → Props → view set.**

## 1. The mismatch today

templ's unit of composition is **Props**: a component is a pure `func(Props) templ.Component`. This library's
unit of composition is **one query → one component**:

| Surface | Location | What it fixes the shape at |
| --- | --- | --- |
| `RenderTempl(component)` | `options_render.go:32` | one component, result ignored |
| `RenderTemplResult[T](mapper)` | `options_render.go:70` | one result type → one component |
| `RenderPartialOrFull[T](partial, full)` | `options_render.go:253` | two components over **the same** `T` |
| `RenderIf(check, match, noMatch)` | `options_render.go:213` | two components over the same result |
| `OOBHTML(id, fragment string, swap...)` | `partial.go:52` | OOB fragments are **strings**, not components |
| `QueryTyped[Q, R]` | `app.go:325` | result typed at dispatch, erased to `any` at render |

An HTMX page is not one component. It is three views over one view model:

1. the **full page** (document shell, nav, session/user, page meta, the region embedded),
2. the **region** (the swap target, a fragment),
3. zero or more **OOB fragments** (counters, badges, flash slots).

All three read the *same* facts, but not the same *subset*. So every real page today either grows a god
query that returns everything, or hand-assembles a props struct in the handler. The second is what the
repo actually does: `adminui.tenantsIndex` (`adminui/handler_tenants.go:19`) builds `tenantsListData`
from a tenant scan **plus** a per-tenant `TenantMembers` count query, with a comment that names the
missing batched query. `dashboard` (`adminui/handler_dashboard.go:27`) does the same across stats, audit
log, admin-grant probe, bus factor and window intelligence.

The result is that "what does this page need?" is not written down anywhere a machine can read. It lives
in the body of a handler, is untestable without a whole `Handler`, and repeats per route.

### Three leaks worth naming

- **View count is capped at two, with one props type.** `RenderPartialOrFull[T]` cannot express a third
  view (OOB), and forces the partial to receive the full page's props type even when it renders one table.
- **OOB is string-shaped.** A templ user must render `templ.Component` into a buffer and hand the bytes to
  `OOBHTML`, so the swap envelope escapes the type system exactly where templ is strongest.
- **The typed path is only typed on the way in.** `QueryTyped[Q, R]` gives compile-time typing at
  dispatch; the render option then receives `any` and type-asserts back (`RenderTemplResult` /
  `RenderPartialOrFull` return `unexpected_result_type` on mismatch). The binding between `R` and the
  component's props is a runtime agreement, not a compile-time one.

## 2. The seam worth building on

```
query set   ->   Props struct   ->   view set (Page | Region | OOB)
(lawful)        (the only seam)      (the request selects one)
```

Two invariants make this hold, and they are the part worth writing into an ADR:

1. **A query never returns a templ component**, and never knows a view exists. Read models stay
   presentation-free; the query set is still the authorization unit (`query/<Name>` posture at the
   dispatch gate) and still the ETag/catalog unit.
2. **A component never dispatches.** It receives props and renders. That is what makes a page renderable
   in a test with a literal props value, zero dispatchers, zero sessions.

Everything else follows: the props struct becomes the single place where "this page needs these N facts"
is declared, testable, and reusable across the full-page and region routes.

## 3. Three candidate shapes

| | Shape | What it fixes | Verdict |
| --- | --- | --- | --- |
| **A** | `QuerySet[Props]`: N typed loaders fanned out concurrently into one props struct, one error policy | god query, serial loaders, implicit page contract | **layer 2** |
| **B** | `View[Props]`: named views (Page / Region / OOB) over one props struct, request selects | view selection, OOB-as-templ, type binding | **start here** |
| **C** | one `PageModelQuery` returning the props struct | nothing new | **reject** |

**Why C is rejected.** Collapsing the page into a single query pushes presentation into the read side,
collapses `query/<Name>` posture into one giant object (one policy row for the whole page, and a dispatch
gate that cannot say "this caller may read tenants but not audit"), loses per-query ETag granularity and
per-query catalog rows, and makes the props shape a domain concern. A QuerySet is a *view* concern. The
domain should not learn what a sidebar is.

**Why B before A.** B is pure render surface: no new execution semantics, no dispatcher plumbing, no
concurrency or cancellation policy to design. It is also the half that unblocks the other half, because
A's output type (the props struct) is exactly B's input.

## 4. Recommended shape

```go
type tenantsProps struct {
    Nav    PageMeta
    Rows   []tenantRow
    Total  int
    Counts map[identitymodel.TenantID]int
}

// Declared once. Each loader is still a NAMED query, so query/<Name>
// posture, ETag granularity and catalog rows survive.
var tenants := cqrshtmx.QuerySet[tenantsProps]{
    cqrshtmx.Load("ListTenants", decodeTenants, into(func(p *tenantsProps, v []Tenant) { p.Rows, p.Total = v.Rows, v.Total })),
    cqrshtmx.Load("CountTenantMembers", decodeCounts, into(func(p *tenantsProps, v map[TenantID]int) { p.Counts = v })),
    cqrshtmx.Load("GetPageNav", nil, into(func(p *tenantsProps, v PageMeta) { p.Nav = v })).PageOnly(),
}

mux.Handle("GET /tenants", app.QuerySet(tenants,
    cqrshtmx.Views[tenantsProps](
        cqrshtmx.Region("#tenant-rows", tenantsRows),      // selected via htmx.Target(r)
        cqrshtmx.Page(tenantsPage),
        cqrshtmx.OOB("tenant-count", tenantCount),         // templ.Component, not a string
    )))
```

The pieces, in dependency order:

- **`cqrshtmx.OOB[T](id string, c func(T) TemplComponent)`**: the templ twin of `OOBHTML`. Renders the
  component into the same envelope (`<div id=... hx-swap-oob=...>`), no string round-trip, Content-Type
  already `text/html`.
- **`cqrshtmx.Views[T](views ...View[T])`**: a render option that selects one primary view and appends
  every OOB view. Selection order: `htmx.Target(r)` match on a `Region`, else `htmx.RenderPartial(r)`
  (history-restore included, per `htmx/htmx.go:170`) picks the first `Region`, else `Page`.
- **`Region(selector string, c func(T) TemplComponent)`**: keyed on `hx-target`, using the existing
  `htmx.Target(r)` (`htmx/htmx.go:180`). This is what makes one props type serve three swap shapes with
  no `if` in the handler.
- **`.PageOnly()`**: the HTMX cost fix. A 30s polled region must not re-run page-only loaders. Today
  `adminui.statsPartial` re-derives stats, audit tail and window intelligence on every poll because
  nothing declares which facts the region needs.

### View scoping is the load-bearing idea

`RenderPartialOrFull` already proved that two views over one result is the right ergonomic (it is what
`examples/basic` teaches at `examples/basic/main.go:391`). What it cannot say is *which facts each view
actually needs*, so the thrifty path is manual. A view set with scoped loaders makes the thrifty path the
declared path: a region request dispatches only region-scoped loaders, a full-page request dispatches
everything, and the difference is the declaration, not a hand-rolled branch.

## 5. Why a QuerySet belongs here and not upstream

`go-cqrs-lite/query/v4@v4.9.0` has no batch or group concept: `Dispatcher`, `RegisterTyped`, `Pagination`,
`PaginatedResult`, `PersistedQuery`. A QuerySet is not a missing CQRS primitive, it is a *presentation*
primitive: it exists because a page needs several read models, which is exactly the concern this repo
owns. Never propose it upstream; propose it in root (or its `view/` subpackage).

## 6. Placement and layering

A `view/` subpackage cannot import root: `nix run .#check-module-layers` fails any back-edge, and that
rule exists because the cycle already happened once (`internal/hx`, AGENTS §Architecture).

Two viable placements:

1. **Root package.** Simplest. `party`/`view` code sits beside `options_render.go` and can touch
   `handlerConfig.render` directly. Cost: root production files and LOC grow, and the root package keeps
   absorbing presentation concerns.
2. **New root-module subpackage `view/`, declared against a structural seam.** Define
   `type DispatchFunc func(ctx context.Context, q query.Query) (any, error)` plus a tiny `Decoder`
   interface, and the subpackage depends on nothing in-module. This is the same trick that keeps root
   Casbin-free: `PermissionChecker`/`Request`/`Decide` are primitive-shaped so `identitymodel.Authz`
   satisfies them structurally (ADR-0057). Root then supplies a two-line adapter from `*query.Dispatcher`.

Recommendation: option 2 if `QuerySet` survives prototyping, because it keeps the layering story intact
and makes the seam explicit. Start in root if the whole idea can be proven in under ~150 LOC.

## 7. Concerns the design must answer

| Concern | Position |
| --- | --- |
| **Concurrency** | Loaders fan out in parallel; a page is one screenful, so 3–6 loaders is the realistic count. Cancellation follows the request context. Order of writes into props is irrelevant because each loader owns distinct fields (`into()` writes). |
| **Error policy** | Fail-fast by default (first loader error aborts the render, mapped through `httperr.MapError`). Opt-in degradation (`Optional()`) for decorative panels: a failed counter renders nothing, the page survives. Never silently zero: an error that is swallowed must be visible in the response or the log. |
| **Authz** | Per loader, not per page. Each loader is a named query, so `query/<Name>` posture at the dispatch gate still applies, and a region's props cannot leak a fact the caller was denied. This is the main reason not to collapse into one god query. |
| **Caching / ETag** | `ETagResults()` (`options_render.go:136`) already hashes rendered bytes; it composes with `Views` unchanged. A later step can derive a cache key from the declaration ("this page depends on ListTenants + CountTenantMembers"), which is what makes a per-page cache honest instead of lucky. |
| **HTMX attribute inheritance** | Region selection must key on `htmx.Target(r)` and must not assume a fragment response is universally right: gotcha 55's class (boosted wrappers leak `hx-target`/`hx-select` onto every ajax element) means a self-fetching element inside a boosted shell needs `hx-target="this" hx-select="unset"` on its own attrs. `Views` cannot fix that from the server side; it can only stop *adding* to it by never emitting a full page for a region-keyed request. |
| **Partial vs full props subset** | Not in scope for v1. A single props type is good enough and keeps the seam simple; splitting per-view props types (formal "region props" structs) doubles the declarations for a small win. Revisit only if a page's full-page-only facts become expensive. |
| **Streaming** | Out of scope. templ renders into an `io.Writer`; a future `Views` could stream the page shell first and regions after, but that is a different design (and interacts with SSE/OOB, not with query mapping). |

## 8. Smallest first increment

Deliberately no execution semantics, no subpackage, no dispatcher changes:

1. `cqrshtmx.OOB[T](id string, c func(T) TemplComponent) RenderFunc`-shaped helper (+ the non-generic
   `OOBTempl(id string, c TemplComponent)`), closing the string round-trip.
2. `cqrshtmx.Views[T](views ...View[T]) HandlerOption` with `Page` / `Region` / `OOB` constructors and
   selection by `htmx.Target` → `htmx.RenderPartial` → `Page`.
3. Prove it in `examples/basic` by replacing the `/demo/items` pair (`examples/basic/main.go:394`) with
   one `Views` declaration and a templ OOB fragment.

Then, only if that feels right: `QuerySet` + `Load` + `PageOnly`, declared against `DispatchFunc`.

### Verification plan

- Golden byte tests for the selection matrix (page / region-by-target / region-by-partial / OOB append),
  in the style of the existing `partial_test.go` and the pinned SSE envelope tests.
- `check-module-layers` green after any subpackage lands; `check-reexport-policy --strict` if anything is
  aliased into root (prefer not to).
- An `adminui` conversion spike on `tenantsIndex` measured in LOC and query count before/after, because
  the winning argument for `PageOnly()` is the polled-stats path, and that argument should be a number,
  not a claim.
- Browser proof through the e2e battery for the region path: server-side unit tests cannot see the
  hx-target/hx-select inheritance class.

## 9. Non-goals

- No view model in the domain. `identity-model` stays presentation-free.
- No second templ wrapper type: keep the structural `TemplComponent` interface (`options_types.go:28`), so
  templ stays a consumer choice rather than a dependency of the seam.
- No replacement of `RenderTempl` / `RenderTemplResult` / `RenderPartialOrFull`. They are the degenerate
  1-query / 2-view cases of the same seam. Deprecate nothing until `Views` has a real consumer.
- No HTTP caching protocol work beyond composing with `ETagResults()`.

## 10. Open questions for the owner

1. **Placement.** Root package or a new `view/` subpackage declared against `DispatchFunc`? The subpackage
   is architecturally cleaner and buys the layering story; it costs an extra adapter and one more module
   surface to document.
2. **Naming.** `QuerySet` / `Props` / `Views` / `Region` / `PageOnly` is the working vocabulary. The
   domain-language glossary has no entry for a page-level fact set today, and this repo treats naming as
   architecture, so the term should be chosen once and added to `docs/DOMAIN_LANGUAGE.md`.
3. **Does `adminui` adopt it, or does it stay hand-assembled?** The panel is the largest existing
   consumer and the best proof, but converting it is a multi-wave job and the current shape is tested and
   shipped. A one-page spike (`tenantsIndex`) as evidence, then a decision.
4. **Error degradation policy.** Fail-fast page, or per-region degradation by default? Cheap to build
   either way, expensive to change once consumers rely on it.
