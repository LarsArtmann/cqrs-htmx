# Code Quality Scan — cqrs-htmx

**Date:** 2026-05-19_22-39

## Build & Lint

| Check       | Result |
|-------------|--------|
| `go build ./...` | PASS |
| `go vet ./...` | PASS |
| `golangci-lint run` | 0 issues |
| `go test -race ./...` | 289 specs PASS |
| Coverage | 94.7% |

## usermgmt submodule

| Check       | Result |
|-------------|--------|
| `go build ./...` | PASS |
| `go vet ./...` | PASS |
| `go test -race ./...` | PASS |

## Code Duplication Analysis

### High Priority

| # | Duplication | Sites | Recommendation |
|---|------------|-------|----------------|
| 1 | HTMX accessor 8-function pattern (`htmx.go:98-170`) | 8 functions | Generic `htmxField[T]` helper |
| 2 | DecodeJSON/Query + DecodeForm/Query (`options.go:66-107`) | 4 functions | Generic decoder setter |
| 3 | ValidateCommand/Query (`options.go:201-250`) | 2 functions | Generic validator wrapper |
| 4 | Notification funcs across 3 types (`notify.go` + `response.go`) | 12 functions | Single impl, delegate |

### Medium Priority

| # | Duplication | Sites | Recommendation |
|---|------------|-------|----------------|
| 5 | Parse/New/Must/Context triplets for 3 IDs (`context.go`) | ~18 functions | Go generics `parseID[T]` |
| 6 | Logging context extraction (`logging.go:33-163`) | 3 blocks | Shared `appendContextAttrs` helper |
| 7 | Error handler with redirect (`errors.go:127-183`) | 2 functions | Shared core |
| 8 | handleCommand/QueryDispatch (`handler.go:46-168`) | 2 functions | Shared dispatch pipeline |

### Low Priority

| # | Duplication | Sites | Recommendation |
|---|------------|-------|----------------|
| 9 | Dispatch error wrapping (`handler.go`) | 3 lines | Helper func |
| 10 | Login redirect default (`errors.go` + `authz.go`) | 3 lines | Helper func |
| 11 | usermgmt policy add/remove (`usermgmt/authz.go:222-252`) | 4 functions | `modifyPolicy` helper |

## Summary

- **0 build errors**, **0 lint issues**, **0 vet issues**
- **11 duplication patterns** identified (4 high, 4 medium, 3 low priority)
- Coverage is strong at 94.7% — gaps are edge cases and interface methods
- No banned dependencies detected
