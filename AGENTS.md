# Project: cqrs-htmx

> **THIS IS A LIBRARY/SDK — NOT AN APPLICATION.**
>
> Consumers import this package into THEIR projects.
> There is no "main app."

A Go library that makes it very easy to use go-cqrs-lite with HTMX, templ, and Casbin authorization.

## Quick Reference

| Item     | Value                                                                             |
| -------- | --------------------------------------------------------------------------------- |
| Language | Go 1.26                                                                           |
| Module   | github.com/larsartmann/cqrs-htmx                                                  |
| Test     | `GOWORK=off GONOSUMCHECK='github.com/larsartmann/*' go test ./... -count=1 -race` |
| Build    | `GOWORK=off GONOSUMCHECK='github.com/larsartmann/*' go build ./...`               |
| Lint     | `golangci-lint run`                                                               |
| Coverage | 96.6% root, 91.3% usermgmt (340+ tests)                                              |

## Architecture

```
cqrs-htmx/
├── app.go         # App builder, Config, Command(), Query(), enrichUserID(), CommandCatalogEntries(), QueryCatalogEntries()
├── handler.go     # handleCommandDispatch(), handleQueryDispatch()
├── options.go     # HandlerOption, decoders, Render/RenderTempl, validation, authMode enum, authorization logic
├── response.go    # HTMX response builder (fluent API) + notification methods
├── authz.go       # Enforcer interface, Authorize, Enforce, AuthorizeMiddleware
├── context.go     # UserID type, ParseUserID/MustParseUserID, context enrichment (UserID → CQRS metadata)
├── errors.go      # CQRS error → HTTP status mapping, sentinels, LoginRedirect
├── htmx.go        # HTMXRequest struct, accessors, context storage, RenderPartial
├── notify.go      # Notification HandlerOptions + NotifyWithEvent builder
├── middleware.go   # HTTP middleware (HTMXMiddleware, ContextEnrichmentMiddleware, Chain)
├── csrf.go        # CSRF token generation, CSRFMiddleware, CSRFProtect, context helpers
├── decoder.go     # Body reading, form/JSON decoding, MaxBodySize enforcement
├── logging.go     # RequestLogging, RequestLoggingSlog, DefaultLogFormatter, JSONLogFormatter
├── ratelimit.go   # RateLimiterMiddleware, per-key token bucket, TTL eviction, hooks
├── security.go    # SecurityHeadersMiddleware, SecurityHeadersConfig builder
├── usermgmt/      # User management submodule (RBAC, sessions, password auth)
│   ├── go.mod     # Independent Go module
│   ├── id.go      # Branded UserID type (go-branded-id), NewUserID constructor
│   ├── authz.go   # Authz wrapper around Casbin (RBAC with domains), AsEnforcer adapter
│   ├── service.go # Service (register, login, logout, authenticate, changePassword, updateRoles)
│   ├── user.go    # User/Session types, bcrypt password hashing (SetPasswordWithCost)
│   ├── store.go   # In-memory UserStore/SessionStore with email index, atomic Create
│   ├── http.go    # AuthHandlers (HTTP routes), SessionMiddleware
│   ├── middleware.go # User context helpers, UserIDFromRequest bridge
│   ├── lockout.go # AccountLockout (configurable max attempts + duration)
│   └── errors.go  # Sentinel errors (cockroachdb/errors)
```

## Key Decisions

- **Framework-agnostic**: Works with `net/http`, Gin, Chi, etc.
- **Enforcer interface**: `authz.go` defines `Enforcer` interface matching Casbin's `Enforce(...any) (bool, error)` — `*casbin.Enforcer` satisfies it automatically, but consumers can provide mock/fake enforcers
- **Casbin v3**: Uses `casbin/casbin/v3` for authorization
- **go-cqrs-lite/core v1.4.0**: Depends on command, query, event, pkg/id packages. v1.4.0 adds `CatalogDispatcher` mixin, `TypedHandler[T]`/`RegisterTyped[T]`, `NewEvents`/`DecodePayloads` batch helpers, `Publisher`/`Subscriber` ISP interfaces
- **Error classification**: `sync.Once` lazy-registers all sentinels (not `init()`)
- **HTMX-aware by default**: All error handling and responses check for HTMX requests
- **User identity propagation**: `UserIDExtractor` → context → event metadata
- **Strongly-typed UserID**: `WithUserID(ctx, UserID)` / `UserIDFromContext(ctx) UserID` — context stores `id.UserID` (ULID-backed branded type); `ParseUserID` / `MustParseUserID` helpers exported; `UserIDExtractor` still returns `string` (consumer extracts from JWT/session), middleware parses to `id.UserID`. **Breaking change**: context values are now strongly typed.
- **Strongly-typed CorrelationID**: `WithCorrelationID(ctx, CorrelationID)` / `CorrelationIDFromContext(ctx) CorrelationID` — context stores `id.CorrelationID` (ULID-backed). `ParseCorrelationID` / `MustParseCorrelationID` / `NewCorrelationID` helpers exported. Auto-extracted from `X-Correlation-ID` header by `ContextEnrichmentMiddleware` which silently drops non-ULID values. **Breaking change**: was raw `string`, now branded type.
- **templ duck-typing**: `TemplComponent` interface matches `templ.Component` without importing templ
- **HTMXRequest context**: `HTMXMiddleware` parses headers once, stores in context for downstream use
- **Notifications**: Standard `{level, message}` trigger pattern for HTMX client-side events; `NotifyWithEvent` builder for custom event names
- **Decoder symmetry**: `DecodeJSON`/`DecodeJSONQuery` and `DecodeForm`/`DecodeFormQuery` pairs for both commands and queries
- **Ginkgo/Gomega**: Test framework per project standards
- **Test type consolidation**: All BDD test types use `bdd` prefix (e.g., `bddCreateUserCmd`, `bddTemplComponent`)
- **Error context wrapping**: All authorization errors (`ErrForbidden`, `ErrEnforcerNil`, `ErrUnauthorized` with Authorize) include resource/action for debugging
- **Timeout support**: `Config.Timeout` wraps only the `Dispatch` call (not decode/auth). Zero or negative = no timeout. Uses single `timeoutCtx()` helper on App
- **Validation HandlerOptions**: `ValidateCommand`/`ValidateQuery` wrap the decoder — short-circuit on decode errors, wrap validation errors with `ErrValidationFailed` (400 Rejection)
- **Lifecycle hooks**: `BeforeDispatchHook`/`AfterDispatchHook` on Config for tracing, logging, metrics around dispatch
- **NotificationLevel type**: `LevelSuccess`, `LevelError`, `LevelWarning`, `LevelInfo` typed constants replace magic strings in notifications
- **JSONErrorHandlerWithRedirect**: JSON error handler that respects per-App `Config.LoginRedirect` (unlike `JSONErrorHandler` which was hardcoded)
- **Authorization config**: `authMode` enum (`authNone`, `authRequired`, `authAuthorized`) replaces `authorize bool` + `requireAuth bool` — impossible states are now unrepresentable
- **Internal sentinels**: `errCommandsNil`, `errQueriesNil`, `errDecoderMissing` are unexported — consumers get HTTP responses, not CQRS errors
- **No deprecated exports**: `DefaultNotificationEvent` removed (was race-risk deprecated var)
- **usermgmt branded UserID**: `usermgmt/id.go` defines `UserID = brandid.ID[userBrand, string]` via `go-branded-id` — all user ID fields/params in the submodule are strongly typed; `.String()` converts at Casbin boundaries; `NewUserID(s)` constructor for tests. **Breaking change**: `User.ID`, `Session.UserID`, `RegisterRequest.ID`, and all service/store/authz method params that took `string` now take `UserID`
- **usermgmt context.Context**: All Service methods take `context.Context` as first param — enables future cancellation, tracing, logging
- **usermgmt input validation**: `RegisterRequest.Validate()` and `LoginRequest.Validate()` check email format, password length (8+), required fields
- **usermgmt atomic Create**: `UserStore.Create()` checks email uniqueness atomically (fixes TOCTOU)
- **usermgmt email index**: `InMemoryUserStore` has `emails map[string]string` for O(1) `FindByEmail`
- **usermgmt EnforceAny/AsEnforcer**: Bridge to `cqrshtmx.Enforcer` interface without importing parent module
- **usermgmt RawEnforcer removed**: Use `AsEnforcer()` instead — prevents leaking casbin internals
- **usermgmt cockroachdb/errors**: Consistent error wrapping with root project
- **usermgmt atomic UpdateRoles**: Uses `Authz.Apply(PolicyUpdate{...})` instead of individual remove/add
- **usermgmt structured logging**: `ServiceConfig.Logger` (defaults to `slog.Default()`) — logs failed logins, role updates
- **usermgmt ChangePassword**: `Service.ChangePassword(ctx, userID, oldPassword, newPassword)` — validates old password, minimum length
- **usermgmt account lockout**: `ServiceConfig.Lockout` — configurable max attempts + duration, returns `ErrAccountLocked` (429)
- **usermgmt immutable bcryptCost**: `ServiceConfig.BcryptCost` replaces mutable global variable
- **CatalogEntries exposure**: `App.CommandCatalogEntries()` and `App.QueryCatalogEntries()` delegate to the embedded `dispatcher.CatalogDispatcher` in go-cqrs-lite v1.4.0. Returns `nil` if the respective dispatcher is not configured. Uses `//nolint:staticcheck` because the upstream `CatalogMeta` type is deprecated in favor of the zero-cost catalog API
- **Zero lint warnings**: All pre-existing lint warnings (gochecknoglobals, noctx, prealloc, unparam) fixed. Pre-commit hook should pass without `--no-verify`
- **usermgmt unified error import**: `usermgmt/http.go` uses `cockroachdb/errors` consistently (not `std/errors`) — prevents split brain with `errors.Is` across the submodule

## Dependencies

| Dependency         | Purpose                   |
| ------------------ | ------------------------- |
| go-cqrs-lite/core  | CQRS dispatch (v1.4.0)    |
| casbin/casbin/v3   | Authorization             |
| cockroachdb/errors | Error handling            |
| gorilla/csrf       | CSRF protection (v1.7.3+) |
| go-branded-id      | Branded types (usermgmt)  |

## Key Gotchas

1. **Module path casing**: go-cqrs-lite uses lowercase `github.com/larsartmann/go-cqrs-lite` (not `LarsArtmann`)
2. **Error wrapping**: Always use `fmt.Errorf("%w: ...", sentinel, err)` — never `errors.Wrapf` with `%s` on sentinels
3. **UserIDExtractor dedup**: Handlers check context first — if middleware already set user ID, extraction is skipped
4. **TemplComponent duck-typing**: No import of `a-h/templ` — local interface matches `Render(ctx, w) error`
5. **headerTrue constant**: Defined in `htmx.go`, used everywhere (including `response.go`) — never hardcode `"true"`
6. **golangci-lint v2 format**: `.golangci.yml` uses `version: "2"` format. Exclusions go under `linters.exclusions.rules`, NOT `issues.exclude-rules` (v1 format silently fails)
7. **gocritic disabled-checks**: Only `ifElseChain` needs explicit disable; `dupImport`, `octalLiteral`, `whyNoLint` are already disabled by default
8. **LSP vs CLI discrepancy**: The LSP (`golangci_lint_ls`) shows ~31 stale warnings that `golangci-lint run` (CLI) does not report — this is an unresolved LSP cache issue, not a real lint problem
9. **Per-App LoginRedirect**: `Config.LoginRedirect` is threaded into the error handler at `New()` time — if no custom `ErrorHandler` is set, the default handler captures the resolved loginRedirect in a closure
10. **Enforcer interface**: `*casbin.Enforcer` satisfies `Enforcer` interface automatically. Custom implementations must implement `Enforce(rvals ...any) (bool, error)`
11. **DefaultNotificationEvent removed**: Was deprecated exported var with race risk. Internal `defaultNotificationEvent` constant used instead. Use `NotifyWithEvent` builder for custom event names
12. **text/plain error handler**: `DefaultErrorHandlerWithRedirect` writes plain text without HTML escaping — `text/plain` Content-Type prevents browser HTML rendering, and escaping distorts error messages
13. **AuthorizeMiddleware backward compat**: `loginRedirect` is variadic (optional 4th arg) for backward compatibility — `AuthorizeMiddleware(e, res, act, extractor)` still works
14. **pkg/errors transitive dep**: `github.com/pkg/errors` is an indirect dep via `cockroachdb/errors` — cannot remove, but it's not directly used
15. **Strongly-typed UserID**: `WithUserID` / `UserIDFromContext` now use `id.UserID` (ULID-backed). `UserIDExtractor` still returns `string`; middleware parses to `UserID`. **Consumer breaking change**: callers passing string literals or invalid ULIDs will see parse failures — use `MustParseUserID` in tests or `ParseUserID` in production
16. **Strongly-typed CorrelationID**: `WithCorrelationID` / `CorrelationIDFromContext` now accept/return `id.CorrelationID` (ULID-backed). `ContextEnrichmentMiddleware` silently drops non-ULID correlation IDs from headers. **Consumer breaking change**: callers passing raw strings or non-ULID values will see compile errors — use `MustParseCorrelationID` in tests, `NewCorrelationID()` to generate, or `ParseCorrelationID` in production
17. **Context keys are empty-struct sentinel types**: `contextKey string` replaced with private `userIDKey{}`, `correlationIDKey{}`, `htmxKey{}` — standard Go pattern for collision-free context values across packages
18. **Middleware silently drops invalid IDs**: `ContextEnrichmentMiddleware` and `App.enrichUserID` parse the extractor's string output to `UserID` — if parsing fails (not a valid ULID), the ID is silently dropped. Auth will fail downstream with `ErrUnauthorized`, not an explicit parse error
19. **Middleware also silently drops invalid correlation IDs**: `ContextEnrichmentMiddleware` calls `ParseCorrelationID` on the `X-Correlation-ID` header — if parsing fails (not a valid ULID), the correlation ID is silently dropped. The `EventOptionsFromContext` function checks `.IsZero()` and silently skips invalid/zero CorrelationID values
20. **Timeout wraps dispatch only**: `Config.Timeout` applies only to the `Dispatch` call in `handler.go`, not to decode or auth — this is intentional; decode/auth should not be time-bounded by the handler timeout
21. **Validation order matters**: `ValidateCommand`/`ValidateQuery` must be applied AFTER the decoder option (e.g., `DecodeJSON`) in the `HandlerOption` list — they wrap the existing decoder, so a nil decoder means validation is silently skipped
22. **Flaky test anti-pattern**: Never use `time.After` + `select` for timeout tests — use `<-ctx.Done()` blocking instead. Also ensure command/query type names in test handlers match decoder output names exactly
23. **Benchmark/example lint exclusions**: `.golangci.yml` has `linters.exclusions.rules` for `(benchmark|example)_test\.go$` files — `intrange`, `noctx`, `nilnil` are relaxed for these files only; production code has no exclusions
24. **GOWORK=off required for CI/local testing**: The project's `go.work` covers root + usermgmt + integration_test. `GOWORK=off` is needed for commands that should use each module's own go.mod (CI, scripts). `go.work` takes precedence locally and provides `replace` semantics automatically
25. **Rate limiter MaxKeys cap**: `MaxKeys int` on `RateLimiterConfig` caps tracked keys. When exceeded, oldest entry evicted via O(log n) min-heap (`container/heap`). Zero = no cap (backward compatible)
26. **CSRF Protection**: Uses `gorilla/csrf` internally. `CSRFProtect(cfg)` caches the middleware instance per handler (not per request). Secure defaults: `SameSite=Lax`, `HttpOnly=false` (required for JS double-submit). **Secure flag must be explicitly set**
27. **Middleware ordering**: `Chain(CSRFMiddleware, HTMXMiddleware, app.Middleware())` is the recommended order — CSRF first (sets cookie + context), then HTMX parsing, then user enrichment
28. **CSRF token format**: gorilla/csrf uses masked tokens (XOR with one-time pad). The cookie contains the session token; the header/form must contain the masked token. Always use `CSRFTokenFromContext()` or template helpers (`CSRFTokenHTMLMeta`, `CSRFTokenHXHeaders`) to obtain the token for the frontend — never read the raw cookie value
29. **gorilla/csrf Secret**: Requires a 32-byte key. `CSRFConfig.Secret` is padded to 32 bytes if shorter; if empty, a random key is generated per `CSRFMiddleware()` call (not persisted across restarts). For production, always provide a stable 32-byte secret
30. **usermgmt submodule**: `usermgmt/` has its own `go.mod` and is an independent Go module. Tests run separately with `cd usermgmt && go test ./...`. Uses bcrypt cost 12 in production (cost 4 in tests via `main_test.go`). Casbin model uses `some(where (p.eft == allow)) && !some(where (p.eft == deny))` — casbin v3 only recognizes this exact ordering, NOT the reverse
31. **gorilla/csrf v1.7.3 Origin/Referer enforcement**: v1.7.3 defaults to HTTPS mode, enforcing strict Origin/Referer checks even for non-TLS requests. `CSRFMiddleware` and `executeCSRFValidation` automatically detect non-TLS requests (`r.TLS == nil`) and mark them as plaintext via `csrf.PlaintextHTTPRequest`, disabling the strict checks for HTTP deployments. No consumer action needed
32. **usermgmt branded UserID**: `usermgmt.User` and `usermgmt.Session` fields (`ID`, `UserID`) are branded types (`usermgmt.UserID = brandid.ID[userBrand, string]`), not raw strings. `UserIDFromRequest` still returns `string` for cqrs-htmx compatibility (converts via `.String()`). Use `NewUserID(s)` in tests. `GroupPolicy.User` and `Domain` remain `string` (Casbin boundary)
33. **usermgmt SessionMaxAge bug (fixed)**: `NewAuthHandlers` previously did not copy `SessionMaxAge` from `HandlerConfig` — always defaulted to 86400 regardless of the provided value. Fixed in 2026-05-20 session
34. **usermgmt coverage**: 95.6% as of 2026-05-20 (up from 85%). All authz, store, http, and UserID type paths tested
35. **usermgmt `GroupPolicy.User`/`Domain` remain `string`**: These are Casbin boundary types. `UserID.String()` converts at the boundary. Intentional design decision — Casbin is an external system
36. **CatalogMeta deprecation**: `command.CatalogMeta` and `query.CatalogMeta` are deprecated in go-cqrs-lite v1.4.0 in favor of the zero-cost catalog API. `App.CommandCatalogEntries()` and `App.QueryCatalogEntries()` wrap these with `//nolint:staticcheck` since the `CatalogEntries()` method on `Dispatcher` is not deprecated — only the metadata struct is
37. **Zero lint warnings**: `golangci-lint run` reports 0 issues. All pre-existing warnings fixed including revive missing docs, errcheck, forcetypeassert, recvcheck, exhaustruct, noctx
38. **Rate limiter uses min-heap eviction**: `evictionHeap` (container/heap) replaces O(n) linear scan with O(log n) eviction. Heap entries track `lastUsed` time; stale entries are checked at heap root before full scan
39. **RateLimiterConfig signedness unified**: `perKeyLimiter.burst` and `perKeyLimiter.maxKeys` changed from `int` to `uint` to match config fields. Conversion to `int` only at `rate.NewLimiter` boundary
40. **Usermgmt HTTP timeout**: `HandlerConfig.Timeout` adds optional `context.WithTimeout` to `handleAuthEndpoint` and `handleLogout`. Zero means no timeout (backward compatible)
41. **CSRFConfig.Secure warning**: `CSRFMiddleware` emits `slog.Warn` when `Secure=false` to alert developers in development
42. **Integration test module**: `integration_test/` is a third Go module that imports both root and usermgmt. Tests `AsEnforcer()` bridge and cross-module `UserID` conversion via `.Get()` (not `.String()` which includes brand prefix). Uses `replace` directives for local development; also included in `go.work`
43. **Root coverage 97.0%** (up from 96.1%), **usermgmt coverage 91.2%**
44. **usermgmt BrandNamer**: `userBrand` implements `BrandNamer` with `Name() string { return "User" }`. `.String()` returns "User:ULID", `.Get()` returns raw ULID. Cross-module bridges MUST use `.Get()` for cqrshtmx `ParseUserID` compatibility
45. **go.work includes integration_test**: `go.work` lists root, usermgmt, and integration_test. datastar-demo is NOT included (it doesn't import sibling modules). integration_test keeps its `replace` directives for `GOWORK=off` CI usage
46. **datastar-demo is standalone**: `examples/datastar-demo/` has its own go.mod but does NOT import the cqrs-htmx root library. It's a go-cqrs-lite + datastar SSE example. Uses go-cqrs-lite/core v1.5.0 (root uses v1.4.0)
47. **CI tests all 4 modules**: CI builds root, usermgmt, integration_test, and datastar-demo. Tests run for root, usermgmt, and integration_test (datastar-demo is a main package with no tests)
48. **Validate uses pointer receiver**: `RegisterRequest.Validate()` and `LoginRequest.Validate()` use `*T` receivers — trimmed Email/DisplayName are persisted in-place to the caller's struct. Was a value receiver bug, fixed 2026-05-22
49. **Max password length 128**: `maxPasswordLength = 128` enforced in both `RegisterRequest.Validate()` and `Service.ChangePassword()`. Prevents bcrypt CPU abuse (bcrypt only uses first 72 bytes anyway)
50. **ErrUserIDExists sentinel**: `store.Create` returns typed `ErrUserIDExists` (not untyped `errors.Newf`). Mapped to HTTP 404 in `errorStatus`. Consistent with `ErrEmailExists` pattern
51. **HandlerConfig.Timeout propagation**: `NewAuthHandler` copies `cfg[0].Timeout` to the handler config. Was a bug — timeout from config was silently dropped before 2026-05-22 fix
52. **HandlerConfig.Secure zero-value caveat**: `Secure` defaults to `true` when NO config is passed, but `HandlerConfig{}` (zero-value Secure=false) overrides it to false. Always set `Secure: true` explicitly in production. Documented on `HandlerConfig` type
53. **WriteJSON returns error**: `WriteJSON` returns `error` from `json.Encoder.Encode` (was silently swallowed). Callers should check the return value
54. **usermgmt coverage 91.3%** (up from 91.2%), **root coverage 96.6%**

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
