# cqrs-htmx — Code Quality Scan

**Date:** 2026-05-07 | **Coverage:** 92.8% | **Tests:** 121 passing (race-safe)

## Summary

| Metric      | Status           |
| ----------- | ---------------- |
| Build       | PASS             |
| Tests       | PASS (race-safe) |
| Coverage    | 92.8%            |
| Lint Issues | 102              |

## Sorted Issues by Priority

### P0 — Must Fix (Production Safety)

| #   | File         | Line | Linter             | Issue                                                                                        |
| --- | ------------ | ---- | ------------------ | -------------------------------------------------------------------------------------------- |
| 1   | `errors.go`  | 97   | gosec G705         | XSS via taint analysis — `w.Write([]byte(err.Error()))` writes unsanitized error to response |
| 2   | `handler.go` | 11   | cyclop             | `handleCommandDispatch` complexity=11, max=10                                                |
| 3   | `handler.go` | 44   | revive empty-block | Empty `else if` block — dead code path                                                       |
| 4   | `options.go` | 224  | wrapcheck          | `json.Marshal` error returned unwrapped                                                      |
| 5   | `options.go` | 227  | wrapcheck          | `json.Unmarshal` error returned unwrapped                                                    |

### P1 — Should Fix (Code Quality)

| #   | File        | Line | Linter           | Issue                                                                  |
| --- | ----------- | ---- | ---------------- | ---------------------------------------------------------------------- |
| 6   | `app.go`    | 131  | exhaustruct      | `handlerConfig{}` — zero-value init is intentional but linter flags it |
| 7   | `htmx.go`   | 71   | goconst          | `"true"` string literal has 8 occurrences — extract to constant        |
| 8   | `htmx.go`   | 39   | revive exported  | Swap strategy const block missing doc comment                          |
| 9   | `notify.go` | 5    | gochecknoglobals | `NotificationEvent` is a mutable global                                |
| 10  | `errors.go` | 82   | gochecknoglobals | `LoginRedirect` is a mutable global                                    |
| 11  | `errors.go` | 10   | gochecknoinits   | `init()` for error classification registration                         |

### P2 — Nice to Have (Lint Hygiene)

| #      | File               | Lines   | Linter       | Issue                                                   |
| ------ | ------------------ | ------- | ------------ | ------------------------------------------------------- |
| 12     | `coverage_test.go` | 429-505 | dupl         | Notification test boilerplate duplicated 3x             |
| 13     | `app_test.go`      | 181     | goconst      | Test JSON body `"email":"test@example.com"` repeated 3x |
| 14     | `suite_test.go`    | 10      | paralleltest | Missing `t.Parallel()` call                             |
| 15-64  | `*_test.go`        | various | exhaustruct  | 37 test Config/struct literals missing optional fields  |
| 65-114 | `*_test.go`        | various | revive       | 50 unused-parameter/dot-import warnings in tests        |

## Detailed Analysis

### 1. XSS Vulnerability (errors.go:97)

`DefaultErrorHandler` writes `err.Error()` directly to the response body. If error messages contain user-controlled data, this is a reflected XSS vector. Should sanitize or use `text/plain` content-type consistently (already sets it, but browsers may still interpret HTML in some contexts).

### 2. Cyclomatic Complexity (handler.go:11)

`handleCommandDispatch` has 11 branches. The empty `else if` block at line 44 is a dead code smell — the redirect was already written by `applyHTMXResponse`, so this branch does nothing. Refactor by extracting response-finalization logic.

### 3. Unwrapped Errors (options.go:224-227)

`decodeFormValues` returns raw `json.Marshal`/`json.Unmarshal` errors without wrapping. Should wrap with `fmt.Errorf` for error chain traceability.

### 4. Test Duplication (coverage_test.go:429-505)

Four notification tests (NotifySuccess/Error/Warning/Info) share identical boilerplate. Extract a table-driven test or helper function.
