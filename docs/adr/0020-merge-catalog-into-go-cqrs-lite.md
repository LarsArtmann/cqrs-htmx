# ADR 0020: Merge catalog/ into go-cqrs-lite

**Date:** 2026-06-28
**Status:** ACCEPTED
**Supersedes:** ADR 0008

## Context

ADR 0008 created a `catalog/` sub-package (5th Go module) inside cqrs-htmx to
provide API documentation generation — a thin wrapper over
`go-cqrs-lite/catalog` with a streamlined single-service Builder facade and
standalone HTTP handlers (D2, Health, EventCatalog, OpenAPI, AsyncAPI).

On review, the module was in the wrong home:

1. **The name lied.** The package was named `cataloghtmx`, but it had zero
   dependency on cqrs-htmx root or usermgmt. Its `go.mod` depended only on
   `go-cqrs-lite/catalog/v3` and `go-error-family`. A package whose name
   advertises a framework it doesn't import is a misnomer.

2. **The precedent was already set.** `go-cqrs-lite/catalog` already shipped a
   `docserver/` sub-package with HTTP serving (OpenAPI/AsyncAPI JSON+YAML+HTML
   UIs, static assets, `RegisterRoutes`). HTTP docs tooling already lived
   upstream. The cqrs-htmx module was a convenience layer over its own exports.

3. **The OpenAPI/AsyncAPI handlers were strictly redundant.** Upstream
   `docserver.DocsServer` is richer (HTML UIs, static assets, YAML+JSON). The
   cqrs-htmx versions were a strict subset.

4. **The genuinely unique value was small and generic** — a single-service
   Builder facade, a D2-over-HTTP handler, a health check, and an EventCatalog
   file-generation helper. None of these are HTMX-specific.

## Decision

Merge the catalog module into `go-cqrs-lite/catalog/v3` (released as v3.2.0)
and delete `cqrs-htmx/catalog/`.

### Where things landed

| What                                                                                               | New home                                                                            |
| -------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------- |
| Single-service Builder facade (`New`, `Command[T]`, `Query[T]`, `Event[T]`, `Build`, `BuildValid`) | `catalog/v3/simple`                                                                 |
| `D2Handler`, `HealthCheckHandler`, `GenerateEventCatalog`                                          | `catalog/v3/docserver` (standalone handlers)                                        |
| `OpenAPIHandler`, `AsyncAPIHandler`                                                                | **Deleted** — use upstream `docserver.DocsServer` (richer: HTML UIs, static assets) |

### Simplifications during the merge

- Replaced the hand-rolled 35-line `toKebab` with upstream's
  `internal/caseutil.ToKebab` (already importable from within the module).
- Removed double error-wrapping in `GenerateEventCatalog` — upstream
  `eventcatalog.Export` already classifies as Infrastructure via go-error-family.

## Consequences

### Positive

- **Single home for catalog tooling.** "Where do I serve my catalog docs?" has
  one answer, not two.
- **No misnamed package.** No more `cataloghtmx` package with zero HTMX content.
- **Less code to maintain.** -2180 lines from cqrs-htmx; the redundant handlers
  are gone, not moved.
- **Richer docs for consumers.** Consumers now get HTML UIs (Scalar, AsyncAPI
  React) via `DocsServer` — the deleted handlers never offered these.

### Negative

- **Breaking change.** `github.com/larsartmann/cqrs-htmx/catalog/v3` is deleted.
  Consumers must migrate to `go-cqrs-lite/catalog/v3` (`simple` + `docserver`).
  The migration is mechanical (documented in the CHANGELOG).

## Alternatives Considered

### A: Keep both modules, deduplicate the overlap

Rejected — maintaining two homes for catalog tooling guarantees drift and
confusion. The split brain is worse than either single home.

### B: Move only the unique handlers upstream, keep the facade in cqrs-htmx

Rejected — the facade has no HTMX dependency either. Keeping it in cqrs-htmx
perpetuates the misnaming. Moving it upstream keeps all catalog ergonomics in
one importable module.
