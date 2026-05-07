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
| Test     | `GONOSUMCHECK='github.com/larsartmann/*' go test ./... -count=1 -race` |
| Build    | `GONOSUMCHECK='github.com/larsartmann/*' go build ./...`               |
| Lint     | `golangci-lint run`                                                    |
| Coverage | 95.7% (150+ tests)                                                     |

## Architecture

```
cqrs-htmx/
├── app.go         # App builder, Config, Command(), Query(), enrichUserID()
├── handler.go     # handleCommandDispatch(), handleQueryDispatch()
├── options.go     # HandlerOption, decoders, Render/RenderTempl, authz helpers
├── response.go    # HTMX response builder (fluent API) + notification methods
├── authz.go       # Enforcer interface, Authorize, Enforce, AuthorizeMiddleware
├── context.go     # Context enrichment (user ID → CQRS metadata)
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
- **templ duck-typing**: `TemplComponent` interface matches `templ.Component` without importing templ
- **HTMXRequest context**: `HTMXMiddleware` parses headers once, stores in context for downstream use
- **Notifications**: Standard `{level, message}` trigger pattern for HTMX client-side events; `NotifyWithEvent` builder for custom event names
- **Decoder symmetry**: `DecodeJSON`/`DecodeJSONQuery` and `DecodeForm`/`DecodeFormQuery` pairs for both commands and queries
- **Ginkgo/Gomega**: Test framework per project standards
- **Test type consolidation**: All BDD test types use `bdd` prefix (e.g., `bddCreateUserCmd`, `bddTemplComponent`)
- **Error context wrapping**: All authorization errors (`ErrForbidden`, `ErrEnforcerNil`, `ErrUnauthorized` with Authorize) include resource/action for debugging
- **AuthorizeMiddleware HTMX-aware**: Uses `DefaultErrorHandlerWithRedirect` for consistent auth error handling with HTMX redirect support; optional `loginRedirect` parameter

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
9. **Dead sentinels**: `ErrNoUserID` and `ErrRendererMissing` are exported but never returned by any code path — removal deferred to v2 as a breaking change
10. **Per-App LoginRedirect**: `Config.LoginRedirect` is threaded into the error handler at `New()` time — if no custom `ErrorHandler` is set, the default handler captures the resolved loginRedirect in a closure
11. **Enforcer interface**: `*casbin.Enforcer` satisfies `Enforcer` interface automatically. Custom implementations must implement `Enforce(rvals ...any) (bool, error)`
12. **DefaultNotificationEvent**: Exported var is deprecated (was mutable, race risk). Internal code uses unexported `defaultNotificationEvent` constant. Use `NotifyWithEvent` builder for custom event names
13. **text/plain error handler**: `DefaultErrorHandlerWithRedirect` writes plain text without HTML escaping — `text/plain` Content-Type prevents browser HTML rendering, and escaping distorts error messages
14. **AuthorizeMiddleware backward compat**: `loginRedirect` is variadic (optional 4th arg) for backward compatibility — `AuthorizeMiddleware(e, res, act, extractor)` still works
15. **pkg/errors transitive dep**: `github.com/pkg/errors` is an indirect dep via `cockroachdb/errors` — cannot remove, but it's not directly used

## Test Commands

```bash
# All tests
GONOSUMCHECK='github.com/larsartmann/*' GOFLAGS=-insecure go test ./... -count=1

# With verbose output
GONOSUMCHECK='github.com/larsartmann/*' GOFLAGS=-insecure go test ./... -count=1 -v

# Race detector
GONOSUMCHECK='github.com/larsartmann/*' GOFLAGS=-insecure go test ./... -count=1 -race

# Coverage
GONOSUMCHECK='github.com/larsartmann/*' GOFLAGS=-insecure go test ./... -count=1 -coverprofile=coverage.out
```
