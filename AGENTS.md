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
| Coverage | 93.1% (136 tests)                                                      |

## Architecture

```
cqrs-htmx/
├── app.go         # App builder, Config, Command(), Query(), enrichUserID()
├── handler.go     # handleCommandDispatch(), handleQueryDispatch()
├── options.go     # HandlerOption, decoders, Render/RenderTempl, notifications, authz helpers
├── response.go    # HTMX response builder (fluent API) + notification methods
├── authz.go       # Casbin authorization (Authorize, Enforce, AuthorizeMiddleware)
├── context.go     # Context enrichment (user ID → CQRS metadata)
├── errors.go      # CQRS error → HTTP status mapping, sentinels, LoginRedirect
├── htmx.go        # HTMXRequest struct, accessors, context storage, RenderPartial
├── middleware.go   # HTTP middleware (HTMXMiddleware, ContextEnrichmentMiddleware, Chain)
```

## Key Decisions

- **Framework-agnostic**: Works with `net/http`, Gin, Chi, etc.
- **Casbin v3**: Uses `casbin/casbin/v3` for authorization
- **go-cqrs-lite/core**: Depends on command, query, event, pkg/id packages
- **CQRS error classification**: Registers custom sentinels with `event.RegisterClassification` in `init()`
- **HTMX-aware by default**: All error handling and responses check for HTMX requests
- **User identity propagation**: `UserIDExtractor` → context → event metadata
- **templ duck-typing**: `TemplComponent` interface matches `templ.Component` without importing templ
- **HTMXRequest context**: `HTMXMiddleware` parses headers once, stores in context for downstream use
- **Notifications**: Standard `{level, message}` trigger pattern for HTMX client-side events
- **Ginkgo/Gomega**: Test framework per project standards

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
