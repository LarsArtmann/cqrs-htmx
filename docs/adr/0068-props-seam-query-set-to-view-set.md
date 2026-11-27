# ADR 0068: The Props Seam — Query Set In, View Set Out

**Status:** Accepted
**Date:** 2026-11-27
**Driver (owner):** the design question "how can we make the concept of a
`Query`/`QuerySet` map to templ in a smarter way architecturally?" and the
comprehensive plan
[2026-11-27_15-24_SUPERB-query-set-to-templ-view-seam-pareto-plan.md](../planning/2026-11-27_15-24_SUPERB-query-set-to-templ-view-seam-pareto-plan.md)
(the thesis lives in
[the design note](../proposals/2026-11-27_query-set-to-templ-view-seam.md)).
**Related:** [ADR-0062](0062-direct-imports-only.md) (additive-only, no new
aliases), [ADR-0037](0037-partial-rendering-helpers.md) (render-helper content
types), [ADR-0057](0057-authorization-contract.md) (the one decision surface
the dispatched-loader adapter preserves).

## Context

templ's unit of composition is **Props**. This library's unit was **one query
→ one component**. A real HTMX page is three views over one view model:

| Shape | What it is | How it was written before |
| --- | --- | --- |
| Page | shell, nav, session, page meta, full data | `RenderPartialOrFull` full branch |
| Region | the fragment the request swaps | `RenderPartialOrFull` partial branch |
| OOB | fragments that update elsewhere | `OOBHTML(id, string)` glued into the partial by hand |

Three consequences, each visible in the shipped code:

1. **A page either grows a god query or hand-assembles props.** `adminui`
   does the latter everywhere: `tenantsIndex` reads
   `Service.AllTenants()` and then one `TenantMembers` read per visible
   tenant, and `handler_dashboard.go` re-derives every stat inside the
   region renderer because no loader layer exists to scope the work.
2. **Out-of-band updates are string-shaped.** `OOBHTML` takes HTML, so a
   templ component must be rendered into a string before it can ride along,
   and the wire bytes have no single source.
3. **The typed path was typed only on the way in.** `RenderTemplResult[T]`
   asserts the result type once; the shape decision was still an `if` in
   every handler.

The rejected alternative was one `PageModelQuery` returning a page struct:
that pushes presentation into the read side, collapses the per-query
authorization posture (`query/<Name>`) into a single policy row that cannot
say "may read tenants but not audit", and loses per-query ETag granularity
and event-catalog rows.

## Decision

**The props struct is the only seam: query set → Props → view set.**

- **In:** `QuerySet[T]` — a named set of `Loader[T]` values. Loaders run in
  parallel; the props struct is written by ONE goroutine after the join.
  Loaders are plain named functions (`LoadFunc[V]`), not queries: dispatched
  queries are one adapter (`LoadQuery` + `FromDispatcher`) that keeps the
  `query/<Name>` posture, so a consumer without a dispatcher uses the same
  API. `Optional()` is the only degradation path and it is visible through
  `LoadReport`/`OnLoadError`; `PageOnly()` marks loaders a region request
  does not need.
- **Out:** `Views[T](...)` — exactly one `Page`, any number of
  `Region(selector)` (at most one `.Default()`), any number of `OOB(id)`
  views. Selection is request-derived and deterministic: boosted navigation
  or history restore → Page; matching `HX-Target` → that region; partial
  with no match → the `Default` region, else an `unroutable_view`
  rejection; everything else → Page.
- **Invariants.** A query never returns a templ component; a component never
  dispatches. Both are review rules with mechanically checkable shapes.
- **No templ dependency.** Views are built from `TemplComponent` (structural
  `Render(ctx, io.Writer) error`), so templ stays a consumer choice.
- **OOB has one wire writer.** `OOBHTML` and `OOBTempl` both write through
  the same envelope function, so the string and component forms cannot
  drift. `OOB` views render only on fragment responses, so a direct visit
  can never leak an `hx-swap-oob` element into the document.

## Consequences

- The per-handler `if htmx.RenderPartial(r)` branch disappears from new
  code; `RenderPartialOrFull`/`RenderIf`/`OOBHTML` stay as the degenerate
  1-query / 2-view cases (no deprecation — ADR-0062).
- Failure policy is explicit and testable: first fatal loader in
  declaration order wins (never completion order), siblings are cancelled,
  optional failures degrade visibly.
- `LoadScopeFor(r)` is one exported rule shared by `Views` and `QuerySet`,
  so the cost model of page-only loaders is arguable (and testable) without
  a handler.
- Additive-only root API: next minor. Nothing is aliased; every new symbol
  is root-exported directly.
- What this deliberately does NOT do: no `view/` subpackage (a `DispatchFunc`
  seam would be needed to avoid the module-layers back-edge; revisit if the
  surface grows past ~200 LOC), no per-view props types, no shell-first
  streaming, no per-page cache keys, and no adminui panel-wide rewrite.

## Alternatives considered

| Alternative | Why rejected |
| --- | --- |
| One `PageModelQuery` per page | Presentation into the read side; posture collapse; one policy row for every fact; no per-query ETag or catalog row. |
| Query-only loaders (dispatched queries mandatory) | `adminui` — the repo's largest consumer — owns no dispatcher; the API would be demo-only. Function loaders plus a dispatch adapter serve both. |
| Silent full-page fallback for an unmatched fragment target | Reproduces the 2026-10-18 blanking class from the other direction: htmx would swap a whole document into a fragment target. Fail loud instead. |
| Deprecating `RenderPartialOrFull`/`OOBHTML` | They are the 1-query/2-view cases of the same seam; removing them buys nothing and breaks every consumer. |
