# Modularization Execution Plan: cqrs-htmx

> **Status:** Phase 5 — Ready for execution
> **Date:** 2026-05-14
> **Based on:** PROPOSAL.md (revised after self-review)

---

## Summary

Extract `htmx.go` and `response.go` into a standalone `cqrs-htmx/htmx` sub-module with zero external dependencies. Root module re-exports all symbols for backward compatibility. This is the only clean extraction — `authz` and `middleware` are too coupled to the handler config pattern.

---

## Task List

### Tier 1: Foundational (1% → 51% impact)

| #   | Task                                              | Dependencies | Effort | Verification                                                                                                                                   | Rollback                                     |
| --- | ------------------------------------------------- | ------------ | ------ | ---------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------- |
| T1  | **Create `htmx/` directory and `go.mod`**         | None         | 5 min  | `go build ./htmx/...` compiles empty package                                                                                                   | `rm -rf htmx/`                               |
| T2  | **Move `htmx.go` to `htmx/` package**             | T1           | 15 min | `go build ./htmx/...` — package compiles with all HTMX types/functions exported                                                                | `git checkout -- htmx.go`                    |
| T3  | **Move `response.go` to `htmx/` package**         | T2           | 15 min | `go build ./htmx/...` — Response builder compiles (depends on htmx.go types internally)                                                        | `git checkout -- response.go`                |
| T4  | **Resolve `defaultNotificationEvent` dependency** | T3           | 10 min | Move constant from `notify.go` to `htmx/response.go` (or `htmx/htmx.go`). Keep deprecated `DefaultNotificationEvent` var in root as re-export. | `git checkout -- notify.go htmx/response.go` |

### Tier 2: High Leverage (4% → 64% impact)

| #   | Task                                          | Dependencies | Effort | Verification                                                                                                          | Rollback                                 |
| --- | --------------------------------------------- | ------------ | ------ | --------------------------------------------------------------------------------------------------------------------- | ---------------------------------------- |
| T5  | **Create `go.work` file**                     | T1           | 5 min  | `go work sync` succeeds                                                                                               | `rm go.work`                             |
| T6  | **Update root module to import from `htmx/`** | T2, T3, T4   | 30 min | `go build ./...` at root — all files compile, imports resolve                                                         | `git checkout -- go.mod go.sum *.go`     |
| T7  | **Add re-exports to root package**            | T6           | 20 min | Existing test suite compiles with zero changes — `cqrshtmx.IsHTMXRequest`, `cqrshtmx.NewResponse`, etc. still resolve | `git checkout -- *.go`                   |
| T8  | **Export `headerRedirect` constant**          | T2           | 5 min  | `errors.go` can reference `htmx.HeaderRedirect` (or similar). Root `errors.go` compiles.                              | `git checkout -- errors.go htmx/htmx.go` |

### Tier 3: Broad Value (20% → 80% impact)

| #   | Task                                    | Dependencies | Effort | Verification                                                                  | Rollback                                                |
| --- | --------------------------------------- | ------------ | ------ | ----------------------------------------------------------------------------- | ------------------------------------------------------- |
| T9  | **Migrate HTMX tests to `htmx/`**       | T6, T7       | 30 min | `go test ./htmx/...` passes — all HTMX + Response tests green                 | `git checkout -- htmx/`                                 |
| T10 | **Verify root test suite still passes** | T7, T9       | 10 min | `go test ./...` — all 137 specs pass (or adjusted count after test migration) | Fix forward or `git reset --soft`                       |
| T11 | **Run `go mod tidy` on all modules**    | T6, T5       | 5 min  | `go mod tidy` changes nothing in any module                                   | `git checkout -- go.mod go.sum htmx/go.mod htmx/go.sum` |
| T12 | **Run full lint suite**                 | T10          | 5 min  | `golangci-lint run` reports 0 issues                                          | Fix forward                                             |

### Tier 4: Polish

| #   | Task                      | Dependencies       | Effort | Verification                                                                          | Rollback                      |
| --- | ------------------------- | ------------------ | ------ | ------------------------------------------------------------------------------------- | ----------------------------- |
| T13 | **Update README.md**      | T10                | 10 min | README shows both import paths: `cqrs-htmx` (full) and `cqrs-htmx/htmx` (lightweight) | `git checkout -- README.md`   |
| T14 | **Update AGENTS.md**      | T10                | 10 min | AGENTS.md reflects new module structure and build commands                            | `git checkout -- AGENTS.md`   |
| T15 | **Update FEATURES.md**    | T10                | 5 min  | Features doc mentions htmx sub-module availability                                    | `git checkout -- FEATURES.md` |
| T16 | **Commit modularization** | T12, T13, T14, T15 | 5 min  | Clean git history with descriptive commit message                                     | N/A                           |

---

## Dependency Graph (Execution Order)

```
T1 ──→ T2 ──→ T3 ──→ T4
  │            │
  │            └──→ T8
  │
  └──→ T5

T2 + T3 + T4 + T8 ──→ T6 ──→ T7

T6 + T7 ──→ T9 ──→ T10 ──→ T11 ──→ T12 ──→ T13 + T14 + T15 ──→ T16
```

## Execution Order (Linear)

1. T1: Create `htmx/` directory and `go.mod`
2. T2: Move `htmx.go` to `htmx/` package
3. T3: Move `response.go` to `htmx/` package
4. T4: Resolve `defaultNotificationEvent` dependency
5. T5: Create `go.work` file
6. T8: Export `headerRedirect` constant
7. T6: Update root module to import from `htmx/`
8. T7: Add re-exports to root package
9. T9: Migrate HTMX tests to `htmx/`
10. T10: Verify root test suite still passes
11. T11: Run `go mod tidy` on all modules
12. T12: Run full lint suite
13. T13: Update README.md
14. T14: Update AGENTS.md
15. T15: Update FEATURES.md
16. T16: Commit modularization

---

## Per-Task Detail

### T1: Create `htmx/` directory and `go.mod`

```bash
mkdir -p htmx
```

Create `htmx/go.mod`:

```
module github.com/larsartmann/cqrs-htmx/htmx

go 1.26.2
```

Verify: `cd htmx && go build ./...` (empty package, should succeed)

### T2: Move `htmx.go` to `htmx/` package

1. Move file: `git mv htmx.go htmx/htmx.go`
2. Change package declaration from `package cqrshtmx` to `package htmx`
3. Export `headerTrue`, `parseHTMXRequest`, `htmxContextKey`, `htmxKey` if needed by other modules, OR keep them unexported if only used internally
4. Export all HTMX header constants that `response.go` needs (they're in the same package now, so no change needed)

Verify: `cd htmx && go build ./...`

### T3: Move `response.go` to `htmx/` package

1. Move file: `git mv response.go htmx/response.go`
2. Change package declaration from `package cqrshtmx` to `package htmx`
3. Move `defaultNotificationEvent` constant to `htmx/` (from `notify.go`) — it's used by `response.go`'s `triggerNotification` method
4. Unexported helpers (`setTriggerHeader`, `setTriggerWithDetail`) stay in the same package — no change

Verify: `cd htmx && go build ./...`

### T4: Resolve `defaultNotificationEvent` dependency

`response.go:130` references `defaultNotificationEvent` which is defined in `notify.go`.

**Solution**: Define the constant in `htmx/` package (where `response.go` lives). Keep the deprecated `DefaultNotificationEvent` var in root `notify.go` as a re-export:

```go
// notify.go (root) — keep deprecated var for backward compat
var DefaultNotificationEvent = htmx.DefaultNotificationEvent
```

Actually, the constant is `defaultNotificationEvent` (unexported). Better: define the actual string in `htmx/` and have root's `notify.go` reference it via the exported package.

**Simplest approach**: Move `defaultNotificationEvent` constant to `htmx/response.go` (same package). In root's `notify.go`, replace the constant reference with the string literal `"showMessage"` (it's just a string) or import from `htmx` package.

### T5: Create `go.work` file

```go
go 1.26.2

use (
    .
    ./htmx
)
```

Verify: `go work sync`

### T8: Export `headerRedirect` constant

`errors.go:105` uses `headerRedirect` (unexported, defined in `htmx.go`). After moving `htmx.go` to `htmx/`, root can't access it.

**Options**:

1. Export as `HeaderRedirect` in `htmx` package → root uses `htmx.HeaderRedirect`
2. Define the string `"HX-Redirect"` directly in `errors.go`

**Recommendation**: Option 1 — export it. It's a well-known HTMX header name, useful for consumers too.

Verify: `errors.go` compiles with `htmx.HeaderRedirect`

### T6: Update root module to import from `htmx/`

1. Update `go.mod` to add `require github.com/larsartmann/cqrs-htmx/htmx v0.0.0`
2. Update all files that reference moved symbols:
   - `errors.go` → import `htmx` for `IsHTMXRequest`, `HeaderRedirect`
   - `options.go` → import `htmx` for `NewResponse`, header constants
   - `handler.go` → import `htmx` for response types (via `options.go` chain)
   - `middleware.go` → import `htmx` for `parseHTMXRequest`, `WithHTMX`
   - `app.go` → import `htmx` indirectly through other files
3. Run `go mod tidy`

Verify: `go build ./...` at root

### T7: Add re-exports to root package

Create a `reexports.go` file (or add to existing files) that re-exports all symbols moved to `htmx/`:

```go
package cqrshtmx

import (
    "github.com/larsartmann/cqrs-htmx/htmx"
)

type HTMXRequest = htmx.HTMXRequest
type SwapStrategy = htmx.SwapStrategy
type Response = htmx.Response

var (
    IsHTMXRequest    = htmx.IsHTMXRequest
    IsBoosted        = htmx.IsBoosted
    // ... etc
)
```

Note: Go type aliases work for types. For functions, use var assignments. For constants, re-declare them.

Verify: All existing tests compile with zero changes to test files.

### T9: Migrate HTMX tests to `htmx/`

1. Create `htmx/suite_test.go` with Ginkgo test runner
2. Move HTMX-specific tests from `htmx_test.go` to `htmx/htmx_test.go`
3. Move Response-specific tests from `htmx_test.go` to `htmx/response_test.go`
4. Update package to `htmx_test`
5. Localize any test helpers needed

Verify: `go test ./htmx/...` passes

### T10: Verify root test suite still passes

```bash
GONOSUMCHECK='github.com/larsartmann/*' GOFLAGS=-insecure go test ./... -count=1
```

Verify: All specs pass (count may be slightly lower after test migration)

### T11–T12: Tidy and lint

```bash
go mod tidy
go work sync
golangci-lint run
```

### T13–T16: Documentation and commit

Update all documentation files to reflect the new structure. Single commit with detailed message.

---

## Risk Mitigation

| Risk                                                | Mitigation                                                                  |
| --------------------------------------------------- | --------------------------------------------------------------------------- |
| Re-export syntax doesn't work for some symbol types | Test compilation early in T7; use type aliases for types, var for functions |
| `go.work` breaks consumer builds                    | `go.work` is ignored by consumers; only affects local development           |
| Test migration breaks coverage                      | Verify coverage percentage after T10; if lower, investigate                 |
| Constants duplicated between root and htmx          | Use exported constants from htmx package in root — single source of truth   |

---

## Future Work (Out of Scope)

These are deferred based on the self-review findings in PROPOSAL.md §6:

1. **`authz/` module extraction** — Requires abstracting `handlerConfig` so auth options don't depend on it directly
2. **`middleware/` module extraction** — Requires `context.go` and `authz.go` types to be in shared modules first
3. **Package-only split** (sub-packages without separate go.mod) — Lower-risk intermediate step for API organization
4. **Independent versioning** — Start with shared versioning; independent tags when sub-modules stabilize
