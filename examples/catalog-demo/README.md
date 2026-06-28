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
