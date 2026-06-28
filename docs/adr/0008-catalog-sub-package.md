# ADR 0008: Catalog Sub-Package for API Documentation

**Date:** 2026-06-17
**Status:** SUPERSEDED by ADR 0015 (2026-06-28)

> **Superseded.** The `catalog/` module was merged upstream into
> [`go-cqrs-lite/catalog/v3`](https://github.com/LarsArtmann/go-cqrs-lite/tree/master/catalog)
> (v3.2.0). The single-service Builder facade now lives at `catalog/v3/simple`,
> and the standalone HTTP handlers (D2, Health, EventCatalog) live at
> `catalog/v3/docserver`. The redundant `OpenAPIHandler`/`AsyncAPIHandler` were
> removed in favor of the richer upstream `docserver.DocsServer` (which adds
> HTML UIs and static assets). The `cqrs-htmx/catalog/` module is deleted.
>
> See ADR 0015 for the reversal rationale. The text below is the original
> decision, preserved for history.

---

## Context

cqrs-htmx consumers build CQRS applications with commands, queries, and events. Every consumer needs API documentation — OpenAPI specs for REST endpoints, AsyncAPI for event flows, architecture diagrams for onboarding. Currently, there's no way to generate this from Go types without manual maintenance.

go-cqrs-lite includes a `catalog/` module that auto-derives JSON schemas from Go structs via reflection and exports to four formats (OpenAPI, AsyncAPI, D2, EventCatalog). The question was whether and how to integrate it into cqrs-htmx.

## Decision

Create a separate `catalog/` sub-package (5th Go module) that bridges go-cqrs-lite's catalog module with cqrs-htmx patterns.

### Design Choices

1. **Separate Go module** (`github.com/larsartmann/cqrs-htmx/catalog/v2`), not a sub-directory of root
   - Consumers who don't want documentation generation pay zero cost
   - The `go-faster/yaml` dependency stays out of root and usermgmt

2. **No dependency on cqrs-htmx root or usermgmt**
   - The catalog sub-package is a thin wrapper over `catalog.Builder`
   - Consumers pass service name/version as strings; no import of `App` type
   - This avoids circular dependencies and keeps the module lightweight

3. **Generic standalone functions instead of generic methods**
   - Go doesn't allow type parameters on methods: `func (b *Builder) Command[T]()` is illegal
   - Solution: package-level functions `Command[T](b *Builder, id, ...)` (same pattern as `command.RegisterTyped[T]`)

4. **Explicit type registration instead of auto-discovery**
   - The `command.Dispatcher` and `query.Dispatcher` have no enumeration API
   - True auto-discovery would require upstream changes to go-cqrs-lite
   - Explicit registration is actually better: the consumer controls what appears in docs

5. **Validation in Build()**
   - `Build()` panics on validation failures (duplicate IDs, empty names)
   - `BuildValid()` returns violations for non-panic usage

## Consequences

### Positive

- **One-time registration → four export formats**: OpenAPI, AsyncAPI, D2, EventCatalog
- **Zero dep creep**: Root and usermgmt modules unaffected
- **Opt-in**: Consumers `go get` only if they want docs
- **Schema auto-derivation**: Struct tags (`json`, `doc`, `format`, `enum`) become JSON Schema
- **HTTP handlers**: One-liner to serve docs endpoints

### Negative

- **Explicit registration required**: Can't auto-discover from dispatcher state
- **Extra module to maintain**: go.mod, go.sum, tests for the catalog sub-package

### Mitigation

- The registration code is minimal (one line per message type)
- The usermgmt pre-built catalog is documented as a pattern in the README rather than a module (avoids adding catalog deps to usermgmt)

## Alternatives Considered

### A: Direct dependency in root module

Rejected — would force `go-faster/yaml` on every consumer, even those who don't want docs.

### B: Auto-discovery via dispatcher enumeration

Rejected — requires upstream go-cqrs-lite changes (exposing registered handler types). Future enhancement.

### C: Pre-built usermgmt catalog as a 6th module

Rejected — too much module overhead for a convenience wrapper. The registration pattern is documented in the catalog README instead.
