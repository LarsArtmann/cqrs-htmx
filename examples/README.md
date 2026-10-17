# Examples

Runnable, tested programs. Each is an independent Go module — `go run .` from
its directory (`GOEXPERIMENT=jsonv2` is required).

| Example                                                | Shows                                                                         | Port  |
| ------------------------------------------------------ | ----------------------------------------------------------------------------- | ----- |
| [`basic`](basic/README.md)                             | Minimal CQRS + HTMX + SSE app with actor metadata and typed endpoints         | 8096  |
| [`setup-demo`](setup-demo/README.md)                   | The one-call full stack (`setup.New` → `bundle.Run`)                          | 8099  |
| [`admin-demo`](admin-demo/README.md)                   | Admin panel showcase with all identity operations                             | 18930 |
| [`dashboard-demo`](dashboard-demo/README.md)           | Event-store observability dashboard                                           | 8098  |
| [`sync-demo`](sync-demo/README.md)                     | Offline sync protocol end-to-end (pull, batched push, offline reads, OpenAPI) | 8098  |
| [`datastar-demo`](datastar-demo/README.md)             | Real-time todo app via the DataStar adapter (SSE + signals)                   | 8095  |
| [`middleware-demo`](middleware-demo/README.md)         | Dispatch middleware composition (retry, circuit breaker, metrics, tracing)    | 8098  |
| [`middleware-showcase`](middleware-showcase/README.md) | All httputil HTTP middleware composed into one validated stack                | 8099  |
| [`observability-demo`](observability-demo/README.md)   | OTel tracing + Prometheus metrics proofs                                      | 8099  |
| [`async-startup-demo`](async-startup-demo/README.md)   | Zero-downtime restarts: bind HTTP while projections replay                    | 8098  |
| [`samber-do-demo`](samber-do-demo/README.md)           | samber/do v2 dependency injection best practice                               | 8098  |
| [`system-demo`](system-demo/README.md)                 | go-cqrs-lite `system.New` integration via systemadapter (console demo)        | —     |
| [`catalog-demo`](catalog-demo/README.md)               | Catalog doc-server example                                                    | 8080  |

Ports are defaults from each `main.go`; several demos intentionally share a
port (they are alternatives, not companions). On this fleet, 8080 and 8099 are
commonly occupied by local services — pass a different addr when they collide.
