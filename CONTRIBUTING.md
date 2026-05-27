# Contributing to cqrs-htmx

Thank you for contributing! This guide covers everything you need to write code that passes review on the first try.

## Prerequisites

- **Go 1.26+**
- **golangci-lint v2** — install from [golangci-lint](https://golangci-lint.run/usage/install/)
- **Ginkgo/Gomega** CLI (optional — `go test` works directly)

## Quick Start

```bash
# Build (GOWORK=off uses each module's own go.mod)
GOWORK=off GONOSUMCHECK='github.com/larsartmann/*' go build ./...

# Test (always with -count=1 to disable cache)
GOWORK=off GONOSUMCHECK='github.com/larsartmann/*' go test ./... -count=1 -race

# Usermgmt submodule (separate Go module)
cd usermgmt && GOWORK=off GONOSUMCHECK='github.com/larsartmann/*' go test ./... -count=1 -race

# Integration tests (separate Go module)
cd integration_test && GOWORK=off GONOSUMCHECK='github.com/larsartmann/*' go test ./... -count=1 -race

# Lint
golangci-lint run
```

All three must pass with zero errors before submitting.

## Architecture

This is a **library/SDK**, not an application. There is no `main` package. Consumers import `github.com/larsartmann/cqrs-htmx` into their projects.

The project uses a **multi-module Go workspace** with 4 modules:

| Module        | Path                        | Go Module                                   |
| ------------- | --------------------------- | ------------------------------------------- |
| Root          | `./`                        | `github.com/larsartmann/cqrs-htmx`          |
| Usermgmt      | `./usermgmt/`               | `github.com/larsartmann/cqrs-htmx/usermgmt` |
| Integration   | `./integration_test/`       | separate test module                        |
| Datastar Demo | `./examples/datastar-demo/` | example app                                 |

```
cqrs-htmx/
├── app.go         # App builder, Config, Command(), Query(), enrichUserID()
├── handler.go     # handleCommandDispatch(), handleQueryDispatch()
├── options.go     # HandlerOption, decoders, Render/RenderTempl, validation
├── response.go    # HTMX response builder (fluent API) + notification methods
├── authz.go       # Enforcer interface, Authorize, Enforce, AuthorizeMiddleware
├── context.go     # UserID type, context enrichment (UserID → CQRS metadata)
├── errors.go      # Error → HTTP status mapping (go-error-family), sentinels, error handlers
├── htmx.go        # HTMXRequest struct, accessors, context storage, RenderPartial
├── notify.go      # Notification HandlerOptions + NotifyWithEvent builder
├── middleware.go   # HTTP middleware (HTMXMiddleware, ContextEnrichmentMiddleware, Chain)
├── csrf.go            # CSRFMiddleware, CSRFConfig, token context helpers (justinas/nosurf)
├── csrf_handler.go    # CSRFProtect (per-handler CSRF)
├── csrf_helpers.go    # CSRFTokenHTMLMeta, CSRFTokenHXHeaders, CSRFTokenFormField
├── decoder.go         # Body reading, form/JSON decoding, MaxBodySize
├── httputil.go         # WriteJSON, ClientIP (delegates to larsartmann/httputil)
├── logging.go          # RequestLogging, RequestLoggingSlog, StatusRecorder
├── ratelimit.go        # RateLimiterMiddleware, token bucket, min-heap eviction
├── security.go         # SecurityHeadersMiddleware, SecurityHeadersConfig
├── recovery.go         # RecoveryMiddleware, panic recovery with stack logging
├── usermgmt/      # User management submodule (RBAC, sessions, password auth)
├── integration_test/ # Cross-module integration tests
└── examples/datastar-demo/ # Real-time CQRS + Datastar SSE example
```

## Code Style

### Error Wrapping

Always use `fmt.Errorf` with `%w` for sentinel wrapping:

```go
// Correct
return fmt.Errorf("%w: %s: %w", ErrDispatchFailed, cmdType, err)

// Wrong — never use %s on sentinels
return errors.Wrapf(err, "dispatch failed: %s", cmdType)
```

Note: Both root and `usermgmt/` use `go-error-family` (via `go-cqrs-lite/core/event`) for error classification. Use `fmt.Errorf` with `%w` for wrapping.

HTMX header values use constants defined in `htmx.go`. Use `HeaderTrue` (exported) instead of `"true"`.

### Decoder Symmetry

Maintain the `DecodeJSON`/`DecodeJSONQuery` and `DecodeForm`/`DecodeFormQuery` pairs. New decoders should follow the same pattern.

### Validation Order

`ValidateCommand`/`ValidateQuery` wrap the decoder. They must be applied **after** the decoder option in the `HandlerOption` list.

### Composition Over Inheritance

No base classes, no deep hierarchies. Use function options (`HandlerOption`), interfaces (`Enforcer`, `TemplComponent`), and dependency injection.

## Testing

### Framework

We use [Ginkgo v2](https://onsi.github.io/ginkgo/) + [Gomega](https://onsi.github.io/gomega/) for all tests.

### Test File Conventions

- Test types use `bdd` prefix: `bddCreateUserCmd`, `bddTemplComponent`
- Shared test helpers live in `testing_test.go`
- Each feature area gets its own test file: `timeout_test.go`, `validation_test.go`, etc.

### Test Patterns

```go
// Use Ginkgo describe/it structure
var _ = Describe("Feature Name", func() {
    It("does something specific", func() {
        // Arrange
        app, err := cqrshtmx.New(cqrshtmx.Config{...})
        Expect(err).NotTo(HaveOccurred())

        // Act
        handler := app.Command("CreateUser", decodeCreateUserJSON())
        r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{}`))
        w := httptest.NewRecorder()
        handler(w, r)

        // Assert
        Expect(w.Code).To(Equal(http.StatusNoContent))
    })
})
```

### Timing in Tests

Never use `time.After` + `select` for timeout/cancellation tests. Use deterministic `<-ctx.Done()` blocking:

```go
// Correct — deterministic
_ = disp.Register("CreateUser", func(ctx context.Context, _ command.Command) error {
    <-ctx.Done()
    return ctx.Err()
})

// Wrong — flaky, race-dependent
_ = disp.Register("SlowCommand", func(ctx context.Context, _ command.Command) error {
    select {
    case <-time.After(5 * time.Second):
        return nil
    case <-ctx.Done():
        return ctx.Err()
    }
})
```

### Handler Type Names

When registering test handlers, use the same command/query type name that the decoder produces. Mismatched names cause "handler not found" errors that can mask test failures.

### Benchmarks

Add benchmarks for performance-sensitive code. Follow the existing pattern in `benchmark_test.go`. Note: `intrange`, `noctx`, and `nilnil` linters are relaxed for benchmark/example test files via `.golangci.yml` exclusions.

### Godoc Examples

Add `Example*` functions in `example_test.go` for public APIs. These appear on pkg.go/dev and serve as executable documentation.

## Lint Configuration

The project uses `.golangci.yml` v2 format with a strict linter set. Key points:

- **`exhaustruct` exclusions**: `Config` and `handlerConfig` are partially populated — excluded from exhaustive struct checks
- **`gocritic`**: Only `ifElseChain` is disabled
- **Test file relaxations**: `revive:dot-imports` and `wrapcheck` are disabled for `*_test.go` files
- **Benchmark/example relaxations**: `intrange`, `noctx`, `nilnil` disabled for `(benchmark|example)_test.go`
- **Global variables**: `gochecknoglobals` excludes `errors.go` and `notify.go` (sentinel errors and notification defaults)

### Known LSP Issue

The `golangci_lint_ls` LSP may show ~23 stale warnings that `golangci-lint run` (CLI) does not report. Always verify with the CLI — the LSP cache is unreliable.

## Pull Request Checklist

- [ ] `GOWORK=off go build ./...` passes
- [ ] `GOWORK=off go test ./... -count=1 -race` passes (root, usermgmt, integration_test)
- [ ] `golangci-lint run` reports 0 issues
- [ ] New code has tests (aim for 90%+ coverage)
- [ ] New public APIs have godoc comments
- [ ] No hardcoded HTMX header strings — use constants
- [ ] Error wrapping uses `fmt.Errorf("%w: ...")` — not `errors.Wrapf`
- [ ] `AGENTS.md` updated if adding new features, gotchas, or conventions
- [ ] Run tests in all modules: root, usermgmt, integration_test
