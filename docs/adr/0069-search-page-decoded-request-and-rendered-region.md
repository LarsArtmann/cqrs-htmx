# ADR 0069: A Search Page Is a Decoded Request and a Rendered Region

**Status:** Accepted
**Date:** 2026-11-28
**Related:** [ADR-0037](0037-partial-rendering-helpers.md) (render-helper content types),
[ADR-0064](0064-http-query-fallback-ladder.md) (the QUERY and POST twins), ADR-0068 (the
props seam), go-cqrs-lite ADR-0169 (the search contract, `search/v4`).

## Context

Every search page in this library and in its admin UI repeats the same four steps by hand:

1. read `q`, the facets, the sort and the page out of the request, each handler with its
   own set of reserved parameter names;
2. loop over a clone of the whole read model with `strings.Contains(strings.ToLower(...))`
   (`adminui/handler_users.go:348-364`, three more panels besides);
3. branch: partial for the htmx request, full page otherwise, with out-of-band count and
   pager envelopes glued in by hand;
4. read the urlencoded body a second time for the QUERY and POST twins (the 64 KiB reader is
   written twice).

The search box fires a request per keystroke and nothing cancels the stale ones. The e2e
suite documents the result as a flake (`e2e/tests/admin-behavior.spec.ts:239-244`).

Meanwhile the library already ships most of the glue and nobody uses it: `RenderPartialOrFull`,
`Trigger`, `PushURL`, `ETagResults`, the out-of-band envelopes.

## Decision

**A search page is one decoded request and one rendered region.**

- **In:** `DecodeSearch(r)` reads a `search.Request` from the URL: `q`, the sort, the page,
   and every other parameter as a facet, untouched. The reserved names are `q`, `page`,
   `page_size`, `sort` and `dir`. `DecodeSearchBody(r)` is the twin for the QUERY and POST
   methods and is the only urlencoded-body reader left. A query that is too long or invalid
   is answered as 400 in the library's error shape.
- **Search:** the handler calls the `search/v4` index. The library never matches text
  itself: an index built on `search/v4` replaces the inline loop in the same commit.
- **Out:** `RenderSearch` renders the results region as a partial for an htmx request and
  the full page otherwise, pushes the URL built from the request, sends the count and pager
  out of band, fires `HX-Trigger: search-results` with `{total, via}`, and applies the
  response ETag on GET. It composes the render helpers above; it does not add a second
  rendering path.

The UI side is `forms.SearchBox` and `display.SearchResults` of templ-components
(its ADR 0051).

## Consequences

- The admin UI's user, tenant, member and audit searches are the first consumers
  (ADR-0001 rule 1: the library that ships a feature migrates its own UI first).
- `FilterInput` is not removed; the search panels move to `SearchBox`.
- The guide's claim that `ETagResults` is not shipped is closed.
- A handler that keeps its own text matcher next to an index is a new split brain and is
  rejected in review.
