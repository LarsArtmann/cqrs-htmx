# Project: cqrs-htmx

> **THIS IS A LIBRARY/SDK — NOT AN APPLICATION.**
>
> Consumers import this package into THEIR projects.
> There is no "main app."

A Go library that makes it very easy to use go-cqrs-lite with HTMX, templ, and Casbin authorization.

## Quick Reference

| Item     | Value                                                                  |
| -------- | ---------------------------------------------------------------------- |
| Language | Go 1.26                                                                |
| Module   | github.com/larsartmann/cqrs-htmx                                       |
| Test     | `GOWORK=off GONOSUMCHECK='github.com/larsartmann/*' go test ./... -count=1 -race` |
| Build    | `GOWORK=off GONOSUMCHECK='github.com/larsartmann/*' go build ./...`               |
| Lint     | `golangci-lint run`                                                    |
| Coverage | 95.7% (150+ tests)                                                     |

## Architecture

```
cqrs-htmx/
├── app.go         # App builder, Config, Command(), Query(), enrichUserID()
├── handler.go     # handleCommandDispatch(), handleQueryDispatch()
├── options.go     # HandlerOption, decoders, Render/RenderTempl, validation, authMode enum, authorization logic
├── response.go    # HTMX response builder (fluent API) + notification methods
├── authz.go       # Enforcer interface, Authorize, Enforce, AuthorizeMiddleware
├── context.go     # UserID type, ParseUserID/MustParseUserID, context enrichment (UserID → CQRS metadata)
├── errors.go      # CQRS error → HTTP status mapping, sentinels, LoginRedirect
├── htmx.go        # HTMXRequest struct, accessors, context storage, RenderPartial
├── notify.go      # Notification HandlerOptions + NotifyWithEvent builder
├── middleware.go   # HTTP middleware (HTMXMiddleware, ContextEnrichmentMiddleware, Chain)
```

## Key Decisions

- **Framework-agnostic**: Works with `net/http`, Gin, Chi, etc.
- **Enforcer interface**: `authz.go` defines `Enforcer` interface matching Casbin's `Enforce(...any) (bool, error)` — `*casbin.Enforcer` satisfies it automatically, but consumers can provide mock/fake enforcers
- **Casbin v3**: Uses `casbin/casbin/v3` for authorization
- **go-cqrs-lite/core**: Depends on command, query, event, pkg/id packages
- **Error classification**: `sync.Once` lazy-registers all sentinels (not `init()`)
- **HTMX-aware by default**: All error handling and responses check for HTMX requests
- **User identity propagation**: `UserIDExtractor` → context → event metadata
- **Strongly-typed UserID**: `WithUserID(ctx, UserID)` / `UserIDFromContext(ctx) UserID` — context stores `id.UserID` (ULID-backed branded type); `ParseUserID` / `MustParseUserID` helpers exported; `UserIDExtractor` still returns `string` (consumer extracts from JWT/session), middleware parses to `id.UserID`. **Breaking change**: context values are now strongly typed.
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

## Dependencies

| Dependency         | Purpose        |
| ------------------ | -------------- |
| go-cqrs-lite/core  | CQRS dispatch  |
| casbin/casbin/v3   | Authorization  |
| cockroachdb/errors | Error handling |

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
16. **Middleware silently drops invalid IDs**: `ContextEnrichmentMiddleware` and `App.enrichUserID` parse the extractor's string output to `UserID` — if parsing fails (not a valid ULID), the ID is silently dropped. Auth will fail downstream with `ErrUnauthorized`, not an explicit parse error
17. **Timeout wraps dispatch only**: `Config.Timeout` applies only to the `Dispatch` call in `handler.go`, not to decode or auth — this is intentional; decode/auth should not be time-bounded by the handler timeout
18. **Validation order matters**: `ValidateCommand`/`ValidateQuery` must be applied AFTER the decoder option (e.g., `DecodeJSON`) in the `HandlerOption` list — they wrap the existing decoder, so a nil decoder means validation is silently skipped
19. **Flaky test anti-pattern**: Never use `time.After` + `select` for timeout tests — use `<-ctx.Done()` blocking instead. Also ensure command/query type names in test handlers match decoder output names exactly
20. **Benchmark/example lint exclusions**: `.golangci.yml` has `linters.exclusions.rules` for `(benchmark|example)_test\.go$` files — `intrange`, `noctx`, `nilnil` are relaxed for these files only; production code has no exclusions
21. **GOWORK=off required**: A parent `go.work` exists at `../go.work` that doesn't include this module. All `go test`/`go build` commands need `GOWORK=off` or they fail with "directory prefix does not contain modules listed in go.work"

## Test Commands

**Note:** `GOWORK=off` is required because a parent `go.work` exists that doesn't include this module.

```bash
# All tests
GOWORK=off GONOSUMCHECK='github.com/larsartmann/*' GOFLAGS=-insecure go test ./... -count=1

# With verbose output
GOWORK=off GONOSUMCHECK='github.com/larsartmann/*' GOFLAGS=-insecure go test ./... -count=1 -v

# Race detector
GOWORK=off GONOSUMCHECK='github.com/larsartmann/*' GOFLAGS=-insecure go test ./... -count=1 -race

# Coverage
GOWORK=off GONOSUMCHECK='github.com/larsartmann/*' GOFLAGS=-insecure go test ./... -count=1 -coverprofile=coverage.out
```
