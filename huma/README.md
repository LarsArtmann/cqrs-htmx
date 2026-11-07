# cqrs-htmx/huma — Huma v2 Adapter

Optional [Huma v2](https://huma.rocks/) adapter module for [cqrs-htmx](https://github.com/larsartmann/cqrs-htmx).

## Why?

Huma owns a typed HTTP contract: it registers operations, decodes and validates
the request, chooses the status code, and generates the OpenAPI 3.1 document
from your Go types. cqrs-htmx's `App.Command`/`App.Query` return
`http.HandlerFunc` values, and an `http.HandlerFunc` cannot participate in that
contract — so an OpenAPI-first app that also uses CQRS ends up with two
disconnected worlds: Huma handlers re-implementing authorization and error
mapping per operation, while the cqrs-htmx pipeline serves nothing.

This module is the bridge. It runs the pipeline stages the endpoint builder runs
— authorize, validate, guard, dispatch, map error — inside a Huma handler, and
answers failures with the same RFC 9457 problem document the HTTP endpoints
emit. Huma still owns decode, input-schema validation, status codes and OpenAPI
generation: the module adds no fields to the generated spec and no behaviour of
its own.

The root module gains nothing: this is a separate Go module with its own
`go.mod`, so `github.com/danielgtaylor/huma/v2` never enters the dependency
graph of a consumer that does not import this package.

## Install

```bash
go get github.com/larsartmann/cqrs-htmx/huma/v4
```

## Quick Start

```go
import (
	humav2 "github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	cqrsadapter "github.com/larsartmann/cqrs-htmx/huma/v4"
)

// 1. One adapter per app: the dispatchers your endpoints already use, plus the
//    same authorization engine (any cqrshtmx.PermissionChecker, e.g. svc.Authz()).
pipe, err := cqrsadapter.New(cqrsadapter.Config{
	Commands: cmdDisp,
	Queries:  qryDisp,
	Checker:  svc.Authz(),
	Timeout:  5 * time.Second,
})
if err != nil {
	panic(err)
}

// 2. Register a command operation. Huma decodes and validates the input; the
//    adapter authorizes, validates the command, dispatches, and maps errors.
api := humago.New(mux, humav2.DefaultConfig("Visits API", "1.0.0"))

cqrsadapter.RegisterCommand(api, pipe,
	humav2.Operation{OperationID: "create-visit", Method: "POST", Path: "/visits", DefaultStatus: 201},
	func(ctx context.Context, in *CreateVisitInput) (command.Command, error) {
		return newVisitCmd(in.Body.URL), nil
	},
	cqrsadapter.Authorize("visits", "create"),
	cqrsadapter.ValidateCommand(validateVisit),
)

// 3. Register a query operation. respond turns the dispatch result into the
//    operation's typed output, so Huma still generates the response schema.
cqrsadapter.RegisterQuery(api, pipe,
	humav2.Operation{OperationID: "list-visits", Method: "GET", Path: "/visits"},
	func(ctx context.Context, in *ListVisitsInput) (query.Query, error) {
		return newListVisitsQuery(in.Cursor), nil
	},
	func(ctx context.Context, in *ListVisitsInput, result any) (*ListVisitsOutput, error) {
		return &ListVisitsOutput{Body: result.([]Visit)}, nil
	},
	cqrsadapter.Authorize("visits", "list"),
)

// 4. Mount cqrs-htmx's enrichment middleware so the pipeline sees the caller.
srv := app.Middleware()(mux)
```

An anonymous caller answers 401, a policy denial 403, a validator failure 400, a
conflict 409, a broken policy engine 503 — the same mapping the route-based
pipeline produces, because both go through `httperr.MapError`.

## API

### Registration

| Function                                                      | Purpose                                                                                        |
| ------------------------------------------------------------- | ---------------------------------------------------------------------------------------------- |
| `RegisterCommand[I](api, pipe, op, build, opts...)`           | Dispatch a command built from the typed input; success answers `op.DefaultStatus` with no body |
| `RegisterQuery[I, O](api, pipe, op, build, respond, opts...)` | Dispatch a query and turn its result into the typed output                                     |

Both run the same stage order: **authorize → build → validate → guard →
dispatch**. Registering with a nil API, adapter or function panics at startup,
the way `App.Command("")` does.

### Options

| Option                                                 | Meaning                                                                                  |
| ------------------------------------------------------ | ---------------------------------------------------------------------------------------- |
| `Authorize(resource, action)`                          | Require `action` on `resource` in the wildcard domain (`*`)                              |
| `AuthorizeIn(domain, resource, action)`                | Tenant-scoped permission question                                                        |
| `AuthorizeOn(func(ctx, *I) (cqrshtmx.Request, error))` | Derive the whole question from the typed input (path params, tenant header, resource ID) |
| `RequireAuth()`                                        | Require an authenticated caller without asking the policy engine                         |
| `ValidateCommand(fn)` / `ValidateQuery(fn)`            | Reject invalid commands/queries with 400 and the shared `cqrshtmx.validate.failed` code  |
| `Guard(func(ctx, any) error)`                          | Ownership checks, API keys, feature flags — runs after build+validate, before dispatch   |
| `WithTimeout(d)`                                       | Per-operation dispatch timeout, overriding `Config.Timeout`                              |

### Stages as callables

A handler shape the helpers do not cover (custom output types, response
headers, a second dispatch) can call the stages directly:

| Method                           | Purpose                                                                                                                    |
| -------------------------------- | -------------------------------------------------------------------------------------------------------------------------- |
| `pipe.Authorize(ctx, req)`       | Ask one authorization question; a denial is classified 401/403, an engine failure is an error (503), never a silent denial |
| `pipe.DispatchCommand(ctx, cmd)` | Enrich from ctx, apply the timeout, dispatch                                                                               |
| `pipe.DispatchQuery(ctx, qry)`   | Query-side mirror                                                                                                          |
| `MapError(ctx, err)`             | Convert any error into the `application/problem+json` body Huma writes                                                     |

### The error document

`MapError` returns a `*ProblemError`, which embeds `httperr.StructuredError` —
the same document the route-based error handler writes:

```json
{
	"type": "conflict",
	"title": "Conflict",
	"status": 409,
	"detail": "visit already recorded",
	"code": "visit.exists",
	"instance": "01JBREQID...",
	"message": "…",
	"why": "…",
	"fix": "…"
}
```

Server-fault detail and context are redacted for a public audience (raw detail
stays in the server log), and `code` is the same machine-readable code a JSON
endpoint would emit, so a client branches on one vocabulary regardless of which
transport served the request.

An error that already implements `huma.StatusError` (`huma.Error422…`, a custom
type) passes through untouched: if your handler chose the response, the adapter
does not relabel it.

## What the adapter does not do

- **Decode.** Huma does. There are no `DecodeJSON` options here.
- **Render.** Huma writes the typed output; `respond` only maps the result.
- **HTMX.** A JSON API has no HTMX headers; use `App.Command` for HTML surfaces.
- **Run `BeforeDispatch`/`AfterDispatch`.** Those hooks take an `*http.Request`
  that a Huma handler never receives. Use Huma middleware for tracing and
  metrics instead.
- **QUERY (RFC 10008) operations.** huma v2 (verified at v2.40.0, the latest)
  panics with `unknown method QUERY` at registration — its OpenAPI Path Item
  builder knows the eight standard methods only. Until upstream adds a `query`
  case, serve QUERY routes through the route builder next to the Huma API
  (`app.Query(...)` + `MethodOverrideMiddleware`; guide:
  `docs/guides/http-query-method.md`). The verdict is pinned by
  `TestQueryMethod_HumaRegistrationPanicsUntilUpstreamSupportsIt`, which turns
  red the day upstream ships support.

## Wiring with setup/v4

The adapter needs the dispatchers themselves. `setup.New` builds them inside
`*usermgmt.Service` and deliberately does not expose them (`Service.Dispatcher()`
is unexported by design), so a pure `setup` bundle has nothing to hand the
adapter. It fits the other shape: an app that owns its domain dispatchers next
to a `setup` bundle (identity), or a consumer on the "root App only" wiring in
`docs/guides/fullstack-wiring.md`. Identity work then goes through
`Service` methods and your domain commands go through the adapter — one
authorization engine (`svc.Authz()`), one error vocabulary.

## License

MIT — see [LICENSE](LICENSE).
