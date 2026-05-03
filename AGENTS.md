# Project: cqrs-htmx

> **THIS IS A LIBRARY/SDK — NOT AN APPLICATION.**
>
> Consumers import this package into THEIR projects.
> There is no "main app."

A Go library that makes it very easy to use go-cqrs-lite with HTMX and Casbin authorization.

## Quick Reference

| Item      | Value                                    |
| --------- | ---------------------------------------- |
| Language  | Go 1.26                                  |
| Module    | github.com/larsartmann/cqrs-htmx         |
| Test      | `go test ./... -count=1`                 |
| Build     | `go build ./...`                         |
| Lint      | `golangci-lint run`                      |

## Architecture

```
cqrs-htmx/
├── app.go         # App builder, Config, Command(), Query()
├── handler.go     # HandlerOption, decoders, dispatch logic
├── response.go    # HTMX response builder (fluent API)
├── authz.go       # Casbin authorization integration
├── context.go     # Context enrichment (user ID → CQRS metadata)
├── errors.go      # CQRS error → HTTP status mapping, sentinels
├── htmx.go        # HTMX request detection, swap strategies
├── middleware.go   # HTTP middleware (context enrichment, chain)
```

## Key Decisions

- **Framework-agnostic**: Works with `net/http`, Gin, Chi, etc.
- **Casbin v3**: Uses `casbin/casbin/v3` for authorization
- **go-cqrs-lite/core**: Depends on command, query, event, pkg/id packages
- **CQRS error classification**: Registers custom sentinels with `event.RegisterClassification` in `init()`
- **HTMX-aware by default**: All error handling and responses check for HTMX requests
- **User identity propagation**: `UserIDExtractor` → context → event metadata
- **Ginkgo/Gomega**: Test framework per project standards

## Dependencies

| Dependency                  | Purpose           |
| --------------------------- | ----------------- |
| go-cqrs-lite/core           | CQRS dispatch     |
| casbin/casbin/v3            | Authorization     |
| cockroachdb/errors          | Error handling    |

## Test Commands

```bash
# All tests
GONOSUMCHECK='github.com/larsartmann/*' go test ./... -count=1

# With verbose output
GONOSUMCHECK='github.com/larsartmann/*' go test ./... -count=1 -v

# Race detector
GONOSUMCHECK='github.com/larsartmann/*' go test ./... -count=1 -race
```
