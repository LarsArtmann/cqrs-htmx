# Project: cqrs-htmx

> **THIS IS A LIBRARY/SDK — NOT AN APPLICATION.**
>
> Consumers import this package into THEIR projects.
> There is no "main app."

A Go library that makes it very easy to use go-cqrs-lite with HTMX, templ, and Casbin authorization.

## Quick Reference

| Item     | Value                                                                                      |
| -------- | ------------------------------------------------------------------------------------------ |
| Language | Go 1.26.3                                                                                  |
| Module   | github.com/larsartmann/cqrs-htmx                                                           |
| Test     | `nix run .#test` or `GONOSUMCHECK='github.com/larsartmann/*' go test ./... -count=1 -race` |
| Build    | `nix run .#build` or `GONOSUMCHECK='github.com/larsartmann/*' go build ./...`              |
| Lint     | `nix run .#lint` or `golangci-lint run`                                                    |
| Coverage | `nix run .#coverage`                                                                       |
| Fmt      | `nix fmt`                                                                                  |
| Flake    | `nix flake check` (formatting + devShells + apps)                                          |
| DevShell | `nix develop` (go, gopls, golangci-lint)                                                   |
| Coverage | 96.0%+ root, 90.0% usermgmt (570+ tests)                                                   |

## Architecture

```
cqrs-htmx/
├── app.go            # App builder, Config, Command(), Query(), enrichUserID()
├── handler.go        # handleCommandDispatch(), handleQueryDispatch()
├── options.go        # HandlerOption, decoders, Render/RenderTempl, validation, authMode enum
├── response.go       # HTMX response builder (fluent API) + notification methods
├── authz.go          # Enforcer interface, Authorize, Enforce, AuthorizeMiddleware
├── context.go        # UserID/CorrelationID/RequestID types, Parse*/MustParse*, context helpers
├── errors.go         # Error → HTTP status mapping, sentinels, LoginRedirect (go-error-family)
├── htmx.go           # HTMXRequest struct, accessors, context storage, RenderPartial
├── htmx_embed.go     # Embedded HTMX v2.0.9 minified JS (go:embed)
├── htmx_serve.go     # HTMXScriptHandler, HTMXVersion, HTMXScriptTag
├── notify.go         # Notification HandlerOptions + NotifyWithEvent builder
├── middleware.go      # HTTP middleware (HTMXMiddleware, ContextEnrichmentMiddleware, Chain)
├── csrf.go           # CSRF middleware, CSRFConfig, context helpers (justinas/nosurf)
├── csrf_handler.go   # CSRFProtect (per-handler CSRF HandlerOption)
├── csrf_helpers.go   # CSRFTokenHTMLMeta, CSRFTokenHXHeaders, CSRFTokenFormField
├── decoder.go        # Body reading, form/JSON decoding, MaxBodySize enforcement
├── httputil.go       # WriteJSON, ClientIP (delegates to larsartmann/httputil)
├── logging.go        # RequestLogging, RequestLoggingSlog, formatters
├── sse.go            # SSE: SSEEvent, WriteSSEEvent, SSEStream, Broadcaster (O(1) unsubscribe), SSEEventStore, ReplayEvents, LastEventIDFromRequest, BroadcastOnSuccess/BroadcastOnSuccessFunc
├── ws.go             # WebSocket protocol helpers: WSMessage, ParseWSMessage, ParseWSMessageInto[T], WSOOBHTML
├── ratelimit.go      # RateLimiterMiddleware, per-key token bucket, min-heap eviction
├── security.go       # SecurityHeadersMiddleware, SecurityHeadersConfig, RecommendedCSP/HSTS
├── recovery.go       # RecoveryMiddleware (package-level), App.RecoverHandler() — panic recovery
├── usermgmt/         # User management submodule (RBAC, sessions, password auth)
│   ├── go.mod        # Independent Go module
│   ├── id.go         # Branded UserID type (go-branded-id), NewUserID constructor
│   ├── authz.go      # Authz wrapper around Casbin (RBAC with domains), AsEnforcer adapter
│   ├── service.go    # Service (register, login, logout, authenticate, changePassword, updateRoles)
│   ├── user.go       # User/Session types, bcrypt, domain methods (SetRoles, ChangePassword, AddRole, RemoveRole)
│   ├── store.go      # In-memory UserStore/SessionStore with email index, atomic Create (pure persistence, no timestamps)
│   ├── http.go       # AuthHandlers (HTTP routes), SessionMiddleware
│   ├── middleware.go  # User context helpers, UserIDFromRequest bridge
│   ├── lockout.go    # AccountLockout (configurable max attempts + duration)
│   └── errors.go     # Sentinel errors
├── integration_test/ # Cross-module integration tests (3rd Go module)
└── examples/
    └── datastar-demo/ # Standalone go-cqrs-lite + datastar SSE example (4th Go module)
```

### Module Layout

| Module           | go.mod                                              | Tests | Notes                             |
| ---------------- | --------------------------------------------------- | ----- | --------------------------------- |
| Root             | `github.com/larsartmann/cqrs-htmx`                  | Yes   | Core library                      |
| usermgmt         | `github.com/larsartmann/cqrs-htmx/usermgmt`         | Yes   | Independent submodule             |
| integration_test | `github.com/larsartmann/cqrs-htmx/integration_test` | Yes   | Tests cross-module bridges        |
| datastar-demo    | `examples/datastar-demo/`                           | No    | Standalone example (main package) |

## Dependencies

| Dependency              | Purpose                   | Used in          |
| ----------------------- | ------------------------- | ---------------- |
| go-cqrs-lite v2.3.0     | CQRS dispatch, pagination | All modules      |
| casbin/casbin/v3        | Authorization             | Root, usermgmt   |
| justinas/nosurf v1.2.0  | CSRF protection           | Root             |
| go-error-family v0.3.0  | Error classification      | Root             |
| larsartmann/httputil    | ClientIP extraction       | Root             |
| go-branded-id           | Branded types             | usermgmt         |
| golang.org/x/crypto     | bcrypt                    | usermgmt         |
| golang.org/x/time       | Rate limiting             | Root             |
| onsi/ginkgo/v2 + gomega | BDD test framework        | All test modules |

## Key Decisions

### Architecture

- **Framework-agnostic**: Works with `net/http`, Gin, Chi, etc. — no router dependency
- **Enforcer interface**: `authz.go` defines `Enforcer{ Enforce(...any) (bool, error) }` — `*casbin.Enforcer` satisfies it automatically; enables mock/fake enforcers
- **templ duck-typing**: `TemplComponent` interface matches `templ.Component` without importing templ
- **authMode enum**: `authNone`/`authRequired`/`authAuthorized` — impossible states unrepresentable
- **Library principle**: Never enforce defaults that consumers might disagree with (no mandatory CSP/HSTS, no mandatory CSRF)
- **Root module is intentionally a single flat package**: 19 files form a cohesive "HTMX-aware CQRS HTTP integration" library. The errors↔response↔csrf cycle prevents further splitting. Sub-package extraction would harm consumer UX. See `docs/modularization/PROPOSAL.md` for full analysis
- **Root ↔ usermgmt: zero mutual imports**: Clean module boundary. Cross-module bridging happens in `integration_test/` only

### Error Handling

- **go-error-family v0.3.0**: Replaced `cockroachdb/errors` for error classification. `sync.Once` lazy-registers sentinels. Re-exported via `event/v2` package
- **Error → HTTP mapping**: `MapError` classifies errors into families (Rejection, NotFound, Conflict, etc.) → HTTP status
- **HTMX-aware errors**: All error handlers check for HTMX requests; auth errors use HX-Redirect
- **Request ID in errors**: `Config.IncludeRequestIDInErrors` auto-selects request-ID-aware error handlers
- **text/plain default**: `DefaultErrorHandlerWithRedirect` uses text/plain (no HTML escaping needed)

### CSRF

- **justinas/nosurf**: Replaced gorilla/csrf. Simpler API, no secret management needed
- **Custom header/field translation**: `translateCSRFHeaders` maps consumer-defined names to nosurf defaults
- **Per-handler CSRF**: `CSRFProtect(cfg)` caches nosurf instance per handler config
- **CSRFConfig.Validate()**: Called automatically by `CSRFMiddleware` — logs errors for SameSite=None+Secure=false and unsafe TrustedOrigins

### Type Safety

- **Strongly-typed IDs**: `UserID`, `CorrelationID`, `RequestID` are all ULID-backed branded types via `go-cqrs-lite/pkg/id`
- **usermgmt UserID**: Separate branded type via `go-branded-id` (string-backed, not ULID). Bridge: `.Get()` for cross-module conversion
- **Context key sentinels**: Empty-struct types (`userIDKey{}`, `correlationIDKey{}`, `htmxKey{}`) — collision-free

### Dispatch

- **Timeout wraps dispatch only**: `Config.Timeout` applies to `Dispatch`, not decode/auth — intentional separation
- **Lifecycle hooks**: `BeforeDispatchHook`/`AfterDispatchHook` for tracing, logging, metrics
- **Validation order**: `ValidateCommand`/`ValidateQuery` must come AFTER decoder in HandlerOption list
- **Dispatch error logging**: `handleErr` logs method, path, and error at warn level before calling error handler
- **Response.JSON error handling**: Returns HTTP 500 on json.Marshal failure instead of empty body
- **usermgmt writeJSON**: Buffers before WriteHeader so encode failures don't commit success status
- **Empty type validation (v2.3.0)**: `App.Command("")` and `App.Query("")` panic — uses `command.Type.IsZero()`/`query.Type.IsZero()` from go-cqrs-lite v2.3.0
- **Deadline propagation (v2.3.0)**: `EventOptionsFromContext` now propagates context deadline via `event.FromContext` — downstream events inherit request timeouts
- **TypedHandler (v2.3.0)**: `command.RegisterTyped[T]` eliminates manual type assertions in handlers — demonstrated in `examples/datastar-demo/`

### Recovery

- **Package-level RecoveryMiddleware**: Uses `DefaultErrorHandler` for panics
- **App.RecoverHandler()**: Uses the App's configured error handler (renamed from RecoveryMiddleware to avoid naming collision)

### Embedded HTMX JS

- **HTMX v2.0.9 minified**: Embedded via `//go:embed htmx.min.js` in `htmx_embed.go` (~49KB). No CDN dependency — consumers serve it from their Go binary
- **HTMXScriptHandler()**: Returns `http.Handler` with correct Content-Type, ETag, Cache-Control (1 year, immutable). Supports GET/HEAD, returns 405 for others. 304 Not Modified via If-None-Match
- **HTMXVersion()**: Returns `"2.0.9"` — useful for cache-busting query params
- **HTMXScriptTag(path)**: Returns `<script src="path"></script>` — convenience for templ/templates
- **Opt-in**: Consumers mount it at any path they choose. Not wired into any middleware or handler by default

### SSE (Server-Sent Events)

- **No external dependencies**: SSE is pure HTTP (text/event-stream), no new imports required (uses `reflect` for channel identity in Broadcaster)
- **SSEEvent**: Protocol-level type with Event, Data, ID, Retry fields. Multi-line data auto-split per SSE spec. CRLF normalized to LF
- **SSEStream**: Sets correct headers (Content-Type, Cache-Control, Connection). Flushes after each Send. Context-aware (cancelled when client disconnects). `LastEventID()` for reconnection
- **Broadcaster**: Thread-safe fan-out using buffered channels (64 capacity). O(1) Unsubscribe via `uintptr` channel identity. Non-blocking broadcast — slow consumers have events dropped. Subscribe/Unsubscribe with channel close
- **Reconnection**: `LastEventIDFromRequest()` parses `Last-Event-ID` header. `SSEEventStore` interface for event replay. `ReplayEvents()` sends missed events to stream
- **CQRS bridge**: `BroadcastOnSuccess(eventName, data)` and `BroadcastOnSuccessFunc(fn)` create `AfterDispatchHook` that broadcasts SSE events on successful dispatch
- **Not tied to App**: SSE building blocks work independently. Consumers wire Broadcaster → SSEStream in their own HTTP handlers

### WebSocket

- **No WebSocket library dependency**: Protocol helpers only (message format, OOB HTML). Consumers choose their own WebSocket library
- **WSMessage/ParseWSMessage**: Parses incoming HTMX WebSocket JSON — extracts HEADERS separately from body fields. `StringBody` helper for typed access
- **ParseWSMessageInto[T]**: Generic typed parser — deserializes body fields into a typed struct T while separating HEADERS. Compile-time safe alternative to `ParseWSMessage`
- **WSOOBHTML**: Wraps HTML with hx-swap-oob attributes for HTMX OOB swap. Uses existing `SwapStrategy` type. Passthrough when HTML already contains hx-swap-oob
- **Decision documented in**: `docs/adr/0004-sse-websocket-support.md`

### Pagination (go-cqrs-lite v2.3.0)

- **DecodePagination(r)**: Extracts `page`/`page_size` from query params, delegates to `query.NewPagination` for defaults and validation
- **RenderPaginatedJSON[T]()**: HandlerOption that renders `query.PaginatedResult[T]` as JSON with 200 OK. Type-safe via generic parameter
- **go-cqrs-lite PaginatedResult[T]**: `Data`, `TotalCount`, `Page`, `PageSize`, `TotalPages`, `HasNext()`, `HasPrev()`

### Domain Model (usermgmt)

- **Rich User entity**: `User` has behavior methods — `SetRoles(roles)`, `ChangePassword(old, new, cost)`, `SetEmail(email)`, `SetDisplayName(name)`, `AddRole`, `RemoveRole`, `HasRole`, `SetPassword`, `CheckPassword`, `IsPasswordSet`. Service layer never directly mutates `user.Roles`, `user.Email`, `user.DisplayName`, or `user.UpdatedAt`
- **Service delegates to domain**: `Service.UpdateRoles` → `user.SetRoles()`, `Service.ChangePassword` → `user.ChangePassword()`. No read-modify-save pattern at the service level
- **Timestamp ownership**: `UpdatedAt` is set by `User.touch()` helper, called from all mutation domain methods. `InMemoryUserStore.Save`/`Create` are pure persistence — no timestamp side-effects
- **Validation co-location**: `validatePassword()` and password constants live in `user.go` alongside `ChangePassword` that uses them. Request validation (`RegisterRequest.Validate`, `LoginRequest.Validate`) stays in `service.go`

## Key Gotchas

### Module & Build

1. **GOWORK=off required**: `go.work` covers root + usermgmt + integration_test. `GOWORK=off` needed for CI/commands using per-module go.mod
2. **Module path casing**: go-cqrs-lite uses lowercase `github.com/larsartmann/go-cqrs-lite` (not `LarsArtmann`)
3. **go-cqrs-lite v2.3.0**: Per-module tags (`command/v2.3.0`, `event/v2.3.0`, etc.) now published. All `go.mod` files declare `v2.3.0`. No replace directives needed
4. **Removed APIs in v2.3.0**: `query.MustNew`, `command.MustNew`, `id.MustParse[T]` removed — use `query.New()`, `command.New()`, `id.Parse[T]()` with error check instead. Our `MustParseUserID`/`MustParseCorrelationID`/`MustParseRequestID` are local wrappers around `Parse`
5. **golangci-lint v2 format**: `.golangci.yml` uses `version: "2"`. Exclusions under `linters.exclusions.rules`, NOT `issues.exclude-rules`
6. **LSP vs CLI discrepancy**: LSP shows ~31 stale warnings; CLI reports 0 — unresolved LSP cache issue
7. **flake.nix uses flake-parts + treefmt**: Nix formatting via `nix fmt` (treefmt with nixfmt + gofmt). No package builds in nix due to private Go deps — use `nix run .#build`/`nix run .#test` apps instead

### Type System

6. **UserID breaking change**: `WithUserID`/`UserIDFromContext` use `id.UserID` (ULID-backed). `UserIDExtractor` still returns `string`. Use `ParseUserID`/`MustParseUserID`
7. **CorrelationID breaking change**: Branded `id.CorrelationID`. Non-ULID header values silently dropped by middleware
8. **usermgmt UserID is different type**: `usermgmt.UserID = brandid.ID[userBrand, string]` — NOT the same as root `id.UserID`. Cross-module bridge uses `.Get()` (not `.String()` which includes brand prefix)
9. **GroupPolicy.User/Domain remain string**: Casbin boundary types. Intentional.

### HTTP & Security

10. **CSRF middleware ordering**: `Chain(CSRFMiddleware, HTMXMiddleware, app.Middleware())` — CSRF first, then HTMX, then enrichment
11. **nosurf custom headers**: Must use `translateCSRFHeaders` to map consumer header/field names to nosurf defaults
12. **HX-Redirect sanitization**: `Response.Redirect()` sanitizes URLs for both HTMX and non-HTMX requests
13. **HandlerConfig.Secure uses \*bool**: `Secure` is `*bool` in usermgmt — nil defaults to true. Use `new(bool)` (false) or a local variable (true) to explicitly set. The zero-value `HandlerConfig{}` is safe
14. **Max password length 128**: Enforced in `RegisterRequest.Validate()` and `Service.ChangePassword()`. Prevents bcrypt CPU abuse

### Error Handling

15. **Error wrapping format**: Use `fmt.Errorf("%w: ...", sentinel, err)` for sentinel wraps. go-error-family for classification
16. **Middleware logs invalid IDs**: Correlation ID parse failures logged at debug level. Request ID parse failures silently generate a new ID
17. **DefaultMaxBodySize**: 10 MB when both `Config.MaxBodySize` and per-handler `WithMaxBodySize` are zero

### Testing

18. **Flaky test anti-pattern**: Never use `time.After` + `select` for timeouts — use `<-ctx.Done()` blocking
19. **Test type consolidation**: All BDD types use `bdd` prefix (e.g., `bddCreateUserCmd`)
20. **Benchmark/example lint exclusions**: `.golangci.yml` relaxes `intrange`, `noctx`, `nilnil` for benchmark/example test files only
21. **usermgmt golines**: `.golangci.yml` uses golines formatter (100-char default). Long signatures must be split

## Test Commands

### Via Nix (preferred)

```bash
nix run .#test       # All tests across all modules (with -race)
nix run .#build      # Build all modules
nix run .#lint       # Lint root + usermgmt
nix run .#coverage   # Coverage report for root + usermgmt
nix fmt              # Format all Go + Nix files
nix flake check      # Verify formatting, devShells, and apps
nix develop          # Enter dev shell (go, gopls, golangci-lint)
```

### Manual (GOWORK=off for individual modules)

Root module runs in workspace mode (uses go.work). Submodules need `GOWORK=off` so each uses its own go.mod.

The `flake.nix` provides per-module nix apps that set `GOWORK=off` automatically:

- `nix run .#test-root` — run root tests in isolation
- `nix run .#test-usermgmt` — run usermgmt tests in isolation
- `nix run .#test-integration` — run integration_test in isolation
- `nix run .#build-datastar-demo` — build the example

The multi-module `nix run .#test` and `nix run .#build` apps run all four modules in sequence.

```bash
# All tests (root)
GONOSUMCHECK='github.com/larsartmann/*' go test ./... -count=1

# With verbose output
GONOSUMCHECK='github.com/larsartmann/*' go test ./... -count=1 -v

# Race detector
GONOSUMCHECK='github.com/larsartmann/*' go test ./... -count=1 -race

# Coverage
GONOSUMCHECK='github.com/larsartmann/*' go test ./... -count=1 -coverprofile=coverage.out
```

### usermgmt submodule

```bash
# All tests
cd usermgmt && GOWORK=off GONOSUMCHECK='github.com/larsartmann/*' go test ./... -count=1 -race

# Build
cd usermgmt && GOWORK=off GONOSUMCHECK='github.com/larsartmann/*' go build ./...
```

### integration_test module

```bash
# All tests
cd integration_test && GOWORK=off GONOSUMCHECK='github.com/larsartmann/*' go test ./... -count=1 -race
```

### datastar-demo example

```bash
# Build only (no tests — it's a main package)
cd examples/datastar-demo && GOWORK=off go build ./...
```
