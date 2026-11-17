# catalog-demo

A standalone, runnable example showing end-to-end API documentation generation
via [`go-cqrs-lite/catalog/v3`](https://github.com/LarsArtmann/go-cqrs-lite/tree/master/catalog):
build an API catalog from Go types and serve it as live OpenAPI, AsyncAPI, and
D2 documentation.

## What it demonstrates

- **Schema reflection from struct tags** — `doc`, `json`, `enum`, `example`,
  `required` are all derived automatically (see `main.go`).
- **All four export formats** — OpenAPI 3.0.3, AsyncAPI 3.0.0, D2 diagrams, and
  the EventCatalog MDX file tree.
- **JSON and YAML output** — the `docserver.DocsServer` exposes both
  `OpenAPISpec()` (JSON) and `OpenAPISpecYAML()` (YAML) handlers.
- **Build-time doc generation** — EventCatalog files are written to disk on
  startup (pass `-eventcatalog ""` to skip).
- **Health check** — `docserver.HealthCheckHandler` reports catalog service count.

## Run it

```bash
cd examples/catalog-demo
GOWORK=off go run .

# then visit:
#   http://localhost:8080/openapi.json    — OpenAPI 3.0.3 spec
#   http://localhost:8080/asyncapi.json   — AsyncAPI 3.0.0 spec
#   http://localhost:8080/diagram.d2      — D2 architecture diagram
#   http://localhost:8080/health          — {"services":1,"status":"healthy"}
```

Flags: `-addr :8080` (listen address), `-eventcatalog ./eventcatalog` (output
dir; empty disables file generation).

## How it maps to your own service

`buildCatalog()` in `main.go` is the whole integration — roughly 30 lines. Swap
in your own command/event/query types (with struct tags) and you get a live,
always-accurate API catalog with zero hand-written docs.

## The executable order saga (GFL, 2026-11-15)

The order service this demo documents is no longer docs-only:
[`ordersaga/`](ordersaga/) is a real event-sourced fixture — one Order
decider, three commands (`order.create`, `order.cancel`,
`payment.settle`), and one ADR-0154 bus deriver (settlement derives from
`order.created`, never dispatched by hand) — described by the GFL spec
[`ordersaga/order-saga.gfl`](ordersaga/order-saga.gfl) and compiled into an
executable suite ([`ordersaga/order-saga_test.go`](ordersaga/order-saga_test.go),
generated — DO NOT EDIT): 5 scenarios covering the full saga chain, both
rejection paths, and two read-model assertions, green under `-race`.

### The committed EventCatalog tree changed hands

|                      | before                                                                                     | after                                                                                                                                  |
| -------------------- | ------------------------------------------------------------------------------------------ | -------------------------------------------------------------------------------------------------------------------------------------- |
| `eventcatalog/` tree | generated at runtime by the catalog/v4 library from Go struct tags (untracked, gitignored) | generated from `ordersaga/order-saga.gfl` via `gfl export eventcatalog --bindings` (committed: 3 services, 2 events, 5 scenario flows) |
| source of truth      | Go types in `main.go`                                                                      | the GFL spec — the same file that generates the executable suite and feeds the drift gate                                              |
| runtime export       | default `-eventcatalog ./eventcatalog`                                                     | default `-eventcatalog ./eventcatalog-runtime` (gitignored); the library export still serves the CI architecture-gate                  |

The scenario flows exist because the bindings resolve every step: prose in,
flow documentation out, from one artifact.

### Reproduce the whole verification loop

```bash
# from the cqrs-htmx workspace root; gfl built from ../graph-features-language

# 1. validate the spec against the event universe (drift gate demo)
gfl validate examples/catalog-demo/ordersaga/order-saga.gfl \
  --events-file examples/catalog-demo/ordersaga/order-saga-events.json

# 2. compile the prose into an executable suite
gfl generate tests examples/catalog-demo/ordersaga/order-saga.gfl \
  -o examples/catalog-demo/ordersaga -p ordersaga_test \
  --domain 'ordersaga.Domain()' \
  --import github.com/larsartmann/cqrs-htmx/examples/catalog-demo/ordersaga \
  --bindings examples/catalog-demo/ordersaga/gfl.bindings.json

# 3. run it against the real deciders
GOEXPERIMENT=jsonv2 go test ./examples/catalog-demo/ordersaga/... -race -count=1

# 4. regenerate the committed EventCatalog tree (the docs face of the same spec)
gfl export eventcatalog examples/catalog-demo/ordersaga/order-saga.gfl \
  --bindings examples/catalog-demo/ordersaga/gfl.bindings.json \
  -o examples/catalog-demo/eventcatalog
```

`order-saga-events.json` is pinned to `ordersaga.Domain().Events` by
`events_sync_test.go`; the fixture itself is proven by a direct chain test
(no GFL, no generator) in the same file.
