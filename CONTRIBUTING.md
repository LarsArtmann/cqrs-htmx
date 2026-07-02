# Contributing to cqrs-htmx

Thank you for contributing! This guide covers everything you need to write code that passes review on the first try.

## Prerequisites

- **Go 1.26+**
- **Nix** (recommended) — provides `go`, `gopls`, `golangci-lint`, `templ` CLI, `d2` via `nix develop`
- **golangci-lint v2** — install standalone from [golangci-lint](https://golangci-lint.run/usage/install/) if not using Nix
- **templ CLI v0.3.x** — only needed when editing `adminui/*.templ` files

## Quick Start

### Via Nix (preferred — handles GOWORK, env vars, module isolation)

```bash
nix develop                           # Enter dev shell (go, gopls, golangci-lint, templ)
nix run .#build                       # Build all modules
nix run .#test                        # Test all modules with -race
nix run .#lint                        # Lint root + usermgmt + adminui
nix run .#errorfamily                 # Verify zero stdlib error constructors
nix run .#coverage                    # Coverage report (root + usermgmt)
nix run .#coverage-gate               # Fail if coverage drops below threshold
nix fmt                               # Format all Go + Nix files
nix flake check                       # Verify formatting, devShells, apps
```

### Manual (GOWORK=off for submodules)

Root module runs in workspace mode (uses `go.work`). Submodules need `GOWORK=off` so each uses its own `go.mod`.

```bash
# Root
GONOSUMCHECK='github.com/larsartmann/*' go build ./...
GONOSUMCHECK='github.com/larsartmann/*' go test ./... -count=1 -race

# Usermgmt submodule (separate Go module)
cd usermgmt && GOWORK=off GONOSUMCHECK='github.com/larsartmann/*' go test ./... -count=1 -race

# Admin UI submodule
cd adminui && GOWORK=off GONOSUMCHECK='github.com/larsartmann/*' go test ./... -count=1 -race

# Integration tests (separate Go module)
cd integration_test && GOWORK=off GONOSUMCHECK='github.com/larsartmann/*' go test ./... -count=1 -race

# Lint
golangci-lint run
```

All tests in all modules must pass with zero errors before submitting.

## Architecture

This is a **library/SDK**, not an application. There is no `main` package. Consumers import `github.com/larsartmann/cqrs-htmx/v4` into their projects.

The project uses a **multi-module Go workspace** with 11 modules:

| Module            | Path                        | Go Module                                               | Tests |
| ----------------- | --------------------------- | ------------------------------------------------------- | ----- |
| Root              | `./`                        | `github.com/larsartmann/cqrs-htmx/v4`                   | Yes   |
| Usermgmt          | `./usermgmt/`               | `github.com/larsartmann/cqrs-htmx/usermgmt/v4`          | Yes   |
| Usermgmt/TOTP     | `./usermgmt/totp/`          | `github.com/larsartmann/cqrs-htmx/usermgmt/totp/v4`     | Yes   |
| Usermgmt/WebAuthn | `./usermgmt/webauthn/`      | `github.com/larsartmann/cqrs-htmx/usermgmt/webauthn/v4` | Yes   |
| Usermgmt/OAuth2   | `./usermgmt/oauth2/`        | `github.com/larsartmann/cqrs-htmx/usermgmt/oauth2/v4`   | Yes   |
| Admin UI          | `./adminui/`                | `github.com/larsartmann/cqrs-htmx/adminui/v4`           | Yes   |
| Integration Test  | `./integration_test/`       | separate test module                                    | Yes   |
| Basic Example     | `./examples/basic/`         | example app                                             | No    |
| Datastar Demo     | `./examples/datastar-demo/` | example app (Datastar SSE)                              | No    |
| Catalog Demo      | `./examples/catalog-demo/`  | example app (go-cqrs-lite catalog)                      | No    |
| Admin Demo        | `./examples/admin-demo/`    | runnable admin panel showcase                           | No    |

**Auth sub-modules** (totp, webauthn, oauth2) use **structural typing** — they implement core interfaces via primitive types (`[]byte`, `string`) without importing core `usermgmt`. This keeps auth dependencies (pquerna/otp, go-webauthn, oauth2/oidc) out of core. Each has its own `.golangci.yml`.

**Dependency direction:** Root → usermgmt: zero imports. Usermgmt → root: YES (imports `RateLimiter`). Auth sub-modules → usermgmt: ZERO (structural typing via interfaces). Adminui → root + usermgmt. Nothing depends on adminui.

## Key Conventions

### Error Handling — NO stdlib error constructors

**BANNED in non-test code:** `errors.New`, `fmt.Errorf` (as error), `errors.Join`. Enforced by `nix run .#errorfamily` (`branching-flow errorfamily .` — must report 0).

Use `go-error-family` constructors re-exported via `go-cqrs-lite/event/v3`:

```go
// Rejections (400) — invalid user input, parse failures
return event.NewRejection("myapp.bad_input", "field X is required")

// Conflicts (409) — state conflict (duplicate, already exists)
return event.NewConflict("myapp.duplicate", "email already registered")

// Transient (503) — retryable system/external failure
return event.WrapTransient(err, "myapp.store_error", "DB write failed")

// Wrapping with the inner error's own family (preserves domain classification)
return event.Wrapf(err, event.Classify(err), "myapp.dispatch_failed", "handler error")
```

**Exception:** `fmt.Sprintf` is fine for building message _strings_ (not error objects).

### Validation Order

`ValidateCommand`/`ValidateQuery` wrap the decoder. They must be applied **after** the decoder option in the `HandlerOption` list.

### Composition Over Inheritance

No base classes, no deep hierarchies. Use function options (`HandlerOption`), interfaces (`Enforcer`, `TemplComponent`), and dependency injection.

### HTMX Constants

Use `HeaderTrue` (exported) instead of string literals `"true"` for HTMX header values. Constants are defined in `htmx.go`.

### Templ Codegen (adminui only)

After editing `adminui/*.templ` files, regenerate and commit both sources:

```bash
cd adminui && templ generate
```

The generated `*_templ.go` files are committed so consumers run no codegen. The import style (single-line vs grouped) is normalized by gofmt — don't manually adjust import formatting in generated files.

## Testing

### Framework

Tests use the **standard `testing` package** with table-driven tests. The usermgmt domain layer also uses the **scenario/v3 BDD DSL** (`scenario.Given().When().Then()`) for decider tests.

Ginkgo/Gomega are available as a dependency but the primary test style is standard Go testing.

### Test File Conventions

- Test types use `bdd` prefix where BDD helpers are involved: `bddCreateUserCmd`
- Each feature area gets its own test file: `timeout_test.go`, `validation_test.go`, etc.
- Scenario-based decider tests live in `es_scenario_test.go`

### Test Patterns

```go
// Standard table-driven test
func TestHandler_ReturnsNoContent(t *testing.T) {
    t.Parallel()

    app, err := cqrshtmx.New(cqrshtmx.Config{...})
    if err != nil {
        t.Fatalf("create app: %v", err)
    }

    handler := app.Command("CreateUser", decodeCreateUserJSON())
    r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{}`))
    w := httptest.NewRecorder()
    handler(w, r)

    if w.Code != http.StatusNoContent {
        t.Errorf("status = %d, want %d", w.Code, http.StatusNoContent)
    }
}

// scenario/v3 BDD DSL for decider logic
scenario.Given[*RegisterUserCmd, UserState](t, foldUser, UserState{}).
    When(cmd, decide).
    Then(eventUserRegistered)
```

### Timing in Tests

Never use `time.After` + `select` for timeout/cancellation tests. Use deterministic `<-ctx.Done()` blocking:

```go
// Correct — deterministic
_ = disp.Register("CreateUser", func(ctx context.Context, _ command.Command) error {
    <-ctx.Done()
    return ctx.Err()
})
```

### Benchmarks

Add benchmarks for performance-sensitive code. Note: `intrange`, `noctx`, and `nilnil` linters are relaxed for benchmark/example test files via `.golangci.yml` exclusions.

### Godoc Examples

Add `Example*` functions in `example*_test.go` for public APIs. These appear on pkg.go/dev and serve as executable documentation.

## Lint Configuration

The project uses `.golangci.yml` v2 format with a strict linter set. Key points:

- **`exhaustruct` exclusions**: `Config` and `handlerConfig` are partially populated — excluded from exhaustive struct checks
- **golines formatter**: 100-char line limit (auto-formatted)
- **Test file relaxations**: `revive:dot-imports` and `wrapcheck` are disabled for `*_test.go` files
- **Benchmark/example relaxations**: `intrange`, `noctx`, `nilnil` disabled for `(benchmark|example)_test.go`
- **Global variables**: `gochecknoglobals` excludes sentinel error files and notification defaults

### LSP vs CLI

The `golangci_lint_ls` LSP may show stale warnings that `golangci-lint run` (CLI) does not report. Always verify with the CLI — the LSP cache is unreliable. The CLI is authoritative.

## Pull Request Checklist

- [ ] `nix run .#build` passes (all modules)
- [ ] `nix run .#test` passes with `-race` (all modules)
- [ ] `nix run .#lint` reports 0 issues
- [ ] `nix run .#errorfamily` reports 0 violations
- [ ] `nix fmt` — 0 files changed
- [ ] New code has tests (aim for 90%+ coverage)
- [ ] New public APIs have godoc comments
- [ ] No stdlib error constructors — use `event.New*/Wrap*`
- [ ] No hardcoded HTMX header strings — use constants
- [ ] `AGENTS.md` updated if adding new features, gotchas, or conventions
- [ ] `*_templ.go` regenerated and committed if `*.templ` files changed
