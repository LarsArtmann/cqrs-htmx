# catalog/v4 integration map — who embeds go-cqrs-lite/catalog, and why setup does not

> **Point-in-time snapshot (2026-10-12).** Answers the question: *"Do cqrs-htmx/setup
> and go-appkit have `/home/lars/projects/go-cqrs-lite/catalog` embedded? If not, why not?"*
> Verification commands at the bottom re-derive every claim.

## TL;DR

| Surface | `catalog/v4` dependency? | Detail |
| --- | --- | --- |
| cqrs-htmx `setup` | **No** | Serves the root module's own `EventCatalog` instead (see below) |
| cqrs-htmx `integration_test` | Yes (v4.7.1) | Contract tests wiring `simple` + `docserver` over a live App |
| cqrs-htmx `examples/catalog-demo` | Yes (v4.7.1) | Executable guide for OpenAPI/AsyncAPI/D2/EventCatalog generation |
| go-appkit `docs` | **Yes** (v4.7.1) | `NewCatalogBuilder` + `RegisterDocs` wrap `catalog/docserver` |
| go-appkit `integration` | Yes (test-only) | `docs_composition_test.go` imports catalog + docserver |
| go-appkit `cqrs` | **No** | Runtime CQRS wrapper; docs concern lives in the dedicated `docs` module |

**"Embedded" caveat:** Go's `embed` cannot cross module or repository boundaries at
all, so no project can literally embed `/home/lars/projects/go-cqrs-lite/catalog`.
What exists is (a) the catalog module's own embedded UI assets
(`//go:embed static/*` in `go-cqrs-lite/catalog/docserver/embed.go`), consumed via
import, and (b) plain module dependencies at v4.7.1 everywhere listed above.

## Why cqrs-htmx/setup does not depend on catalog/v4

1. **It ships a different, lighter catalog concept.** Setup serves the root
   module's "Published Language" event catalog —
   `cqrshtmx.EventCatalogHandler(usermgmt.DefaultEventCatalog())`
   (`setup/machine.go`, implemented in `event_catalog_handler.go:45` +
   `usermgmt/es_event_catalog.go:13`): a dependency-free immutable-JSON surface
   for the 21 identity events with a 1-year cache and FNV-1a ETag.
   See `docs/guides/event-catalog-guide.md`.

2. **Two different "catalogs" — do not conflate.**
   - cqrs-htmx `EventCatalog` = schema surface for the identity domain events.
   - `go-cqrs-lite/catalog/v4` = the heavier doc-generation suite (OpenAPI,
     AsyncAPI 3.0, D2 diagrams, EventCatalog MDX from Go CQRS types), dragging
     templ, go-faster/yaml, go-snaps into the graph.

3. **Deliberate opt-in composition, not an oversight.** Wiring catalog onto a
   cqrs-htmx app is consumer-side: `simple.New(app.ServiceName(), …)` +
   `docserver`, demonstrated by `examples/catalog-demo/main.go` and pinned by
   `integration_test/catalog_test.go` (and the usermgmt DTO recipe mirror in
   `integration_test/usermgmt_catalog_test.go`). The library principle — never
   enforce defaults consumers might disagree with — keeps the heavy dependency
   out of the `setup` bundle while the seam stays proven.

## Why go-appkit/cqrs does not depend on catalog/v4

Separation of concerns by module split: auto-documentation lives in the
dedicated opt-in `docs` module (go-appkit AGENTS.md: "docs — opt-in
auto-documentation via catalog/v4"), so the runtime CQRS wrapper keeps its
dependency surface on the system-engine stack. The `integration` module's usage
is test-only (composition proof).

## If setup should offer catalog

The seam exists — the App introspection used by `integration_test/catalog_test.go`
is exactly what a `Config.DocsPath`-style option would build on (mounting
docserver next to the machine endpoints). That would be a feature decision
(ROADMAP material), not a missing integration.

## Verification (re-derive the table)

```sh
rg -l 'go-cqrs-lite/catalog' ~/projects/cqrs-htmx --glob 'go.mod'
rg -l 'go-cqrs-lite/catalog' ~/projects/go-appkit --glob 'go.mod'
rg -n 'go:embed' ~/projects/go-cqrs-lite/catalog
rg -n 'EventCatalogHandler' ~/projects/cqrs-htmx/setup ~/projects/cqrs-htmx/event_catalog_handler.go
```

## Outcome (2026-10-12, later the same day — append-only)

- **Versions advanced:** the 10-sweep family alignment swept `catalog/v4`
  v4.8.0 onto both cqrs-htmx consumers (`integration_test` +
  `examples/catalog-demo`); push `291b2c51`, strict train gate 866 requires
  0/0/0. The v4.7.1 pins in the TL;DR above are the morning snapshot; the
  go-appkit `docs` + `integration` consumers remain at v4.7.1 pending
  go-appkit's own train (owner-gated lane).
- **Disambiguation landed where consumers look:** the two-catalogs distinction
  is now documented in `docs/guides/event-catalog-guide.md` (Serving note +
  See Also), `setup/README.md` (EventCatalogPath naming note), and the
  catalog-demo doc comment — killing the exact conflation that prompted this
  map.
- **DocsPath stays a decision, not a feature:** the seam design + go/no-go
  demand-signal criteria were consolidated into the ROADMAP data-mesh
  Approach-C row (no separate proposal artifact); implementation remains
  owner-gated.
