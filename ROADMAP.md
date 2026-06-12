# Roadmap — cqrs-htmx

**Updated:** 2026-06-08

## Current State

- **Version:** v2.1.0
- **Coverage:** 96.9% root, 91.1% usermgmt
- **Lint:** 0 issues
- **Dependencies:** go-cqrs-lite v2.2.0 (all modules), justinas/nosurf, go-error-family v0.3.0, larsartmann/httputil
- **Test suite:** 464+ specs, race-safe, fuzz tests, benchmarks

---

## v1.1.0 — Production Hardening

_Focus: Align modules, stabilize deps, prepare for broader adoption._

| Area   | Item                                                                        | Priority | Status             |
| ------ | --------------------------------------------------------------------------- | -------- | ------------------ |
| SSE/WS | SSE streaming (Broadcaster, reconnection, CQRS bridge)                      | High     | Done (v2.1.0)      |
| SSE/WS | WebSocket message parser, typed generics, OOB HTML                          | High     | Done (v2.1.0)      |
| HTMX   | Embedded HTMX JS (v2.0.9) with ETag/caching                                 | Medium   | Done (v2.1.0)      |
| Deps   | Upgrade all modules to go-cqrs-lite v2.2.0                                  | High     | Done               |
| Deps   | Remove CatalogEntries (dead upstream code in v2)                            | High     | Done               |
| Deps   | Adopt v2 typed dispatch (`RegisterTyped`/`DispatchTyped`)                   | Medium   | Open               |
| Deps   | Adopt v2 `PaginatedResult[T]` for query handlers                            | Low      | Done               |
| Types  | BrandNamer for root module marker types (`userMarker`, `correlationMarker`) | Medium   | Blocked (upstream) |
| Docs   | Add comprehensive godoc package examples with runnable snippets             | Medium   | Open               |
| Docs   | README.md refresh — reflect nosurf, go-error-family, httputil migration     | Medium   | Done               |
| Test   | Expand integration_test module to cover more cross-module bridges           | Low      | Open               |
| Perf   | Profile hot paths (dispatch, decode) for allocation reduction               | Low      | Open               |

---

## v2.2.0 — SQL Store Backend

_Focus: Persistent storage for usermgmt beyond in-memory._

| Area    | Item                                                                    | Priority | Status              |
| ------- | ----------------------------------------------------------------------- | -------- | ------------------- |
| Store   | PostgreSQL store for `UserStore` interface                              | High     | Planned             |
| Store   | PostgreSQL store for `SessionStore` interface                           | High     | Planned             |
| Types   | Numeric branded IDs (`brandid.ID[Brand, int64]`) for auto-increment PKs | High     | Pattern in ADR 0003 |
| Migrate | Database migration tooling (goose, golang-migrate, or gnorm)            | Medium   | Planned             |
| Test    | Integration tests against real PostgreSQL                               | Medium   | Planned             |

---

## v3.0.0 — Observability & Extensibility

_Focus: Production-grade observability and plugin ecosystem._

| Area          | Item                                                          | Priority | Status  |
| ------------- | ------------------------------------------------------------- | -------- | ------- |
| Observability | OpenTelemetry middleware (tracing via lifecycle hooks)        | High     | Planned |
| Observability | Prometheus metrics middleware (dispatch latency, error rates) | Medium   | Planned |
| Auth          | JWT/OIDC integration helpers                                  | Medium   | Planned |
| Store         | Redis session store for distributed deployments               | Medium   | Planned |
| Store         | BadgerDB embedded store alternative                           | Low      | Planned |

---

## Not Planned

These are explicitly out of scope for this library:

- **WebSocket upgrade logic** — Consumers should use dedicated libraries (gorilla/websocket, coder/websocket, etc.). The library provides protocol helpers (`WSMessage`, `WSOOBHTML`) only.
- **ORM integration** — Store interfaces are intentionally simple; consumers provide their own implementations
- **Template engine support beyond templ** — The `TemplComponent` duck-typing pattern covers any `Render(ctx, w) error` interface
