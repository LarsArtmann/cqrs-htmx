# Project: cqrs-htmx

> **THIS IS A LIBRARY/SDK — NOT AN APPLICATION.**
>
> Consumers import this package into THEIR projects.
> There is no "main app."

A Go library that makes it very easy to use go-cqrs-lite with HTMX, templ, and Casbin authorization.

## Quick Reference

| Item     | Value                                                                             |
| -------- | --------------------------------------------------------------------------------- |
| Language | Go 1.26.3                                                                         |
| Module   | github.com/larsartmann/cqrs-htmx                                                  |
| Test     | `GOWORK=off GONOSUMCHECK='github.com/larsartmann/*' go test ./... -count=1 -race` |
| Build    | `GOWORK=off GONOSUMCHECK='github.com/larsartmann/*' go build ./...`               |
| Lint     | `golangci-lint run`                                                               |
| Coverage | 96.9% root, 91.1% usermgmt (390+ tests)                                           |

## Architecture

```
cqrs-htmx/
├── app.go            # App builder, Config, Command(), Query(), enrichUserID(), catalog entries
├── handler.go        # handleCommandDispatch(), handleQueryDispatch()
├── options.go        # HandlerOption, decoders, Render/RenderTempl, validation, authMode enum
├── response.go       # HTMX response builder (fluent API) + notification methods
├── authz.go          # Enforcer interface, Authorize, Enforce, AuthorizeMiddleware
├── context.go        # UserID/CorrelationID/RequestID types, Parse*/MustParse*, context helpers
├── errors.go         # Error → HTTP status mapping, sentinels, LoginRedirect (go-error-family)
├── htmx.go           # HTMXRequest struct, accessors, context storage, RenderPartial
├── notify.go         # Notification HandlerOptions + NotifyWithEvent builder
├── middleware.go      # HTTP middleware (HTMXMiddleware, ContextEnrichmentMiddleware, Chain)
├── csrf.go           # CSRF middleware, CSRFConfig, context helpers (justinas/nosurf)
├── csrf_handler.go   # CSRFProtect (per-handler CSRF HandlerOption)
├── csrf_helpers.go   # CSRFTokenHTMLMeta, CSRFTokenHXHeaders, CSRFTokenFormField
├── decoder.go        # Body reading, form/JSON decoding, MaxBodySize enforcement
├── httputil.go       # WriteJSON, ClientIP (delegates to larsartmann/httputil)
├── logging.go        # RequestLogging, RequestLoggingSlog, formatters
├── ratelimit.go      # RateLimiterMiddleware, per-key token bucket, min-heap eviction
├── security.go       # SecurityHeadersMiddleware, SecurityHeadersConfig, RecommendedCSP/HSTS
├── recovery.go       # RecoveryMiddleware, App.RecoveryMiddleware() — panic recovery
├── usermgmt/         # User management submodule (RBAC, sessions, password auth)
│   ├── go.mod        # Independent Go module
│   ├── id.go         # Branded UserID type (go-branded-id), NewUserID constructor
│   ├── authz.go      # Authz wrapper around Casbin (RBAC with domains), AsEnforcer adapter
│   ├── service.go    # Service (register, login, logout, authenticate, changePassword, updateRoles)
│   ├── user.go       # User/Session types, bcrypt password hashing
│   ├── store.go      # In-memory UserStore/SessionStore with email index, atomic Create
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

| Dependency               | Purpose              | Used in          |
| ------------------------ | -------------------- | ---------------- |
| go-cqrs-lite/core v1.5.1 | CQRS dispatch        | All modules      |
| casbin/casbin/v3         | Authorization        | Root, usermgmt   |
| justinas/nosurf v1.2.0   | CSRF protection      | Root             |
| go-error-family v0.1.1   | Error classification | Root             |
| larsartmann/httputil     | ClientIP extraction  | Root             |
| go-branded-id            | Branded types        | usermgmt         |
| golang.org/x/crypto      | bcrypt               | usermgmt         |
| golang.org/x/time        | Rate limiting        | Root             |
| onsi/ginkgo/v2 + gomega  | BDD test framework   | All test modules |

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

- **go-error-family**: Replaced `cockroachdb/errors` for error classification. `sync.Once` lazy-registers sentinels
- **Error → HTTP mapping**: `MapError` classifies errors into families (Rejection, NotFound, Conflict, etc.) → HTTP status
- **HTMX-aware errors**: All error handlers check for HTMX requests; auth errors use HX-Redirect
- **Request ID in errors**: `Config.IncludeRequestIDInErrors` auto-selects request-ID-aware error handlers
- **text/plain default**: `DefaultErrorHandlerWithRedirect` uses text/plain (no HTML escaping needed)

### CSRF

- **justinas/nosurf**: Replaced gorilla/csrf. Simpler API, no secret management needed
- **Custom header/field translation**: `translateCSRFHeaders` maps consumer-defined names to nosurf defaults
- **Per-handler CSRF**: `CSRFProtect(cfg)` caches nosurf instance per handler config

### Type Safety

- **Strongly-typed IDs**: `UserID`, `CorrelationID`, `RequestID` are all ULID-backed branded types via `go-cqrs-lite/pkg/id`
- **usermgmt UserID**: Separate branded type via `go-branded-id` (string-backed, not ULID). Bridge: `.Get()` for cross-module conversion
- **Context key sentinels**: Empty-struct types (`userIDKey{}`, `correlationIDKey{}`, `htmxKey{}`) — collision-free

### Dispatch

- **Timeout wraps dispatch only**: `Config.Timeout` applies to `Dispatch`, not decode/auth — intentional separation
- **Lifecycle hooks**: `BeforeDispatchHook`/`AfterDispatchHook` for tracing, logging, metrics
- **Validation order**: `ValidateCommand`/`ValidateQuery` must come AFTER decoder in HandlerOption list

## Key Gotchas

### Module & Build

1. **GOWORK=off required**: `go.work` covers root + usermgmt + integration_test. `GOWORK=off` needed for CI/commands using per-module go.mod
2. **Module path casing**: go-cqrs-lite uses lowercase `github.com/larsartmann/go-cqrs-lite` (not `LarsArtmann`)
3. **go-cqrs-lite version alignment**: All 4 modules now use v1.5.1-pre. datastar-demo was migrated from `command.Core`/`query.Core` → `command.BasicCommand`/`query.BasicQuery`
4. **golangci-lint v2 format**: `.golangci.yml` uses `version: "2"`. Exclusions under `linters.exclusions.rules`, NOT `issues.exclude-rules`
5. **LSP vs CLI discrepancy**: LSP shows ~31 stale warnings; CLI reports 0 — unresolved LSP cache issue

### Type System

6. **UserID breaking change**: `WithUserID`/`UserIDFromContext` use `id.UserID` (ULID-backed). `UserIDExtractor` still returns `string`. Use `ParseUserID`/`MustParseUserID`
7. **CorrelationID breaking change**: Branded `id.CorrelationID`. Non-ULID header values silently dropped by middleware
8. **usermgmt UserID is different type**: `usermgmt.UserID = brandid.ID[userBrand, string]` — NOT the same as root `id.UserID`. Cross-module bridge uses `.Get()` (not `.String()` which includes brand prefix)
9. **GroupPolicy.User/Domain remain string**: Casbin boundary types. Intentional.

### HTTP & Security

10. **CSRF middleware ordering**: `Chain(CSRFMiddleware, HTMXMiddleware, app.Middleware())` — CSRF first, then HTMX, then enrichment
11. **nosurf custom headers**: Must use `translateCSRFHeaders` to map consumer header/field names to nosurf defaults
12. **HX-Redirect sanitization**: `Response.Redirect()` sanitizes URLs for both HTMX and non-HTMX requests
13. **HandlerConfig.Secure zero-value**: `Secure` defaults to `true` when NO config passed, but `HandlerConfig{}` (zero-value) overrides to false. Set explicitly in production
14. **Max password length 128**: Enforced in `RegisterRequest.Validate()` and `Service.ChangePassword()`. Prevents bcrypt CPU abuse

### Error Handling

15. **Error wrapping format**: Use `fmt.Errorf("%w: ...", sentinel, err)` for sentinel wraps. go-error-family for classification
16. **Middleware silently drops invalid IDs**: Parse failures → ID silently dropped → auth fails downstream with `ErrUnauthorized`
17. **DefaultMaxBodySize**: 10 MB when both `Config.MaxBodySize` and per-handler `WithMaxBodySize` are zero

### Testing

18. **Flaky test anti-pattern**: Never use `time.After` + `select` for timeouts — use `<-ctx.Done()` blocking
19. **Test type consolidation**: All BDD types use `bdd` prefix (e.g., `bddCreateUserCmd`)
20. **Benchmark/example lint exclusions**: `.golangci.yml` relaxes `intrange`, `noctx`, `nilnil` for benchmark/example test files only
21. **usermgmt golines**: `.golangci.yml` uses golines formatter (100-char default). Long signatures must be split

## Test Commands

**Note:** `GOWORK=off` is required so each module uses its own go.mod (not the workspace-level resolution).

```bash
# All tests (root)
GOWORK=off GONOSUMCHECK='github.com/larsartmann/*' go test ./... -count=1

# With verbose output
GOWORK=off GONOSUMCHECK='github.com/larsartmann/*' go test ./... -count=1 -v

# Race detector
GOWORK=off GONOSUMCHECK='github.com/larsartmann/*' go test ./... -count=1 -race

# Coverage
GOWORK=off GONOSUMCHECK='github.com/larsartmann/*' go test ./... -count=1 -coverprofile=coverage.out
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
