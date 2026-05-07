# Comprehensive Improvement Plan — cqrs-htmx

**Date:** 2026-05-07 | **Based on:** Full Code Review + Architecture Review + Quality Scan

## Pareto Breakdown

### 1% → 51% Impact (Do First)

| #   | Task                                             | Impact             | Effort | File         |
| --- | ------------------------------------------------ | ------------------ | ------ | ------------ |
| 1   | Fix XSS in DefaultErrorHandler                   | Security           | 15min  | `errors.go`  |
| 2   | Remove dead empty-block in handleCommandDispatch | Correctness        | 5min   | `handler.go` |
| 3   | Wrap decodeFormValues errors                     | Error traceability | 10min  | `options.go` |
| 4   | Reduce handleCommandDispatch complexity to ≤10   | Maintainability    | 20min  | `handler.go` |

### 4% → 64% Impact (Do Second)

| #   | Task                                           | Impact       | Effort | File                               |
| --- | ---------------------------------------------- | ------------ | ------ | ---------------------------------- |
| 5   | Add nolint directives for intentional patterns | Lint hygiene | 10min  | `app.go`, `errors.go`, `notify.go` |
| 6   | Extract `"true"` string constant in htmx.go    | Code quality | 5min   | `htmx.go`                          |
| 7   | Add SwapStrategy doc comment                   | Lint hygiene | 5min   | `htmx.go`                          |
| 8   | Deduplicate notification test boilerplate      | Test quality | 15min  | `coverage_test.go`                 |

### 20% → 80% Impact (Do Third)

| #   | Task                                     | Impact       | Effort | File(s)                                  |
| --- | ---------------------------------------- | ------------ | ------ | ---------------------------------------- |
| 9   | Move LoginRedirect to per-App config     | Architecture | 30min  | `app.go`, `errors.go`, `response.go`     |
| 10  | Move NotificationEvent to per-App config | Architecture | 30min  | `notify.go`, `response.go`, `options.go` |
| 11  | Extract Casbin interface for testability | Architecture | 20min  | `authz.go`                               |
| 12  | Add BDD tests for user-facing scenarios  | Testing      | 45min  | new `bdd_test.go`                        |

### Remaining (Nice to Have)

| #   | Task                              | Impact        | Effort | File(s)                       |
| --- | --------------------------------- | ------------- | ------ | ----------------------------- |
| 13  | Add dispatch lifecycle hooks      | Extensibility | 30min  | `options.go`, `handler.go`    |
| 14  | Add request validation middleware | Features      | 60min  | new file                      |
| 15  | Add observability/logging hooks   | Observability | 45min  | new file                      |
| 16  | Add correlation ID propagation    | Tracing       | 30min  | `context.go`, `middleware.go` |
| 17  | Add JSON error response option    | Features      | 20min  | `errors.go`                   |
| 18  | Add timeout propagation           | Reliability   | 20min  | `handler.go`                  |

## D2 Execution Graph

```d2
direction: right

title: Execution Plan — Dependency Graph

start: {shape: circle, label: "Start"}

phase1: {
  shape: rectangle
  style: {fill: "#C8E6C9"}

  fix_xss: "1. Fix XSS\n(errors.go)"
  remove_dead: "2. Remove dead block\n(handler.go)"
  wrap_errors: "3. Wrap errors\n(options.go)"
  reduce_complexity: "4. Reduce complexity\n(handler.go)"
}

phase2: {
  shape: rectangle
  style: {fill: "#FFF9C4"}

  nolint: "5. Add nolint directives"
  extract_const: "6. Extract 'true' const"
  doc_comments: "7. Add doc comments"
  dedup_tests: "8. Dedup test boilerplate"
}

phase3: {
  shape: rectangle
  style: {fill: "#FFE0B2"}

  login_redirect: "9. LoginRedirect per-App"
  notification_event: "10. NotificationEvent per-App"
  casbin_iface: "11. Casbin interface"
  bdd_tests: "12. BDD user scenarios"
}

phase4: {
  shape: rectangle
  style: {fill: "#E1BEE7"}

  lifecycle: "13. Dispatch hooks"
  validation: "14. Request validation"
  observability: "15. Observability"
  correlation: "16. Correlation ID"
  json_errors: "17. JSON error responses"
  timeouts: "18. Timeout propagation"
}

start -> phase1.fix_xss
phase1.fix_xss -> phase2.nolint
phase2.nolint -> phase3.login_redirect
phase3.login_redirect -> phase4.lifecycle
```

## Task Breakdown (15min each)

| #   | Task                                                    | Files                | Dependencies |
| --- | ------------------------------------------------------- | -------------------- | ------------ |
| 1a  | Sanitize error message in DefaultErrorHandler           | `errors.go`          | —            |
| 1b  | Add test for XSS-sanitized error                        | `errors_test.go`     | 1a           |
| 2a  | Remove empty else-if block                              | `handler.go`         | —            |
| 2b  | Verify tests still pass                                 | —                    | 2a           |
| 3a  | Wrap json.Marshal error in decodeFormValues             | `options.go`         | —            |
| 3b  | Wrap json.Unmarshal error in decodeFormValues           | `options.go`         | —            |
| 3c  | Add test for wrapped form decode errors                 | `coverage_test.go`   | 3a           |
| 4a  | Extract responseFinalization from handleCommandDispatch | `handler.go`         | 2a           |
| 4b  | Verify complexity ≤ 10                                  | —                    | 4a           |
| 5a  | Add nolint:exhaustruct to buildHandlerConfig            | `app.go`             | —            |
| 5b  | Add nolint:gochecknoinits to errors.go init             | `errors.go`          | —            |
| 5c  | Add nolint:gochecknoglobals to LoginRedirect            | `errors.go`          | 9            |
| 6a  | Extract headerValueTrue const in htmx.go                | `htmx.go`            | —            |
| 7a  | Add doc comment to SwapStrategy const block             | `htmx.go`            | —            |
| 8a  | Extract notification test helper                        | `coverage_test.go`   | —            |
| 8b  | Rewrite 4 notification tests as table-driven            | `coverage_test.go`   | 8a           |
| 9a  | Add loginRedirect field to App                          | `app.go`             | —            |
| 9b  | Remove global mutation in New()                         | `app.go`             | 9a           |
| 9c  | Pass loginRedirect to DefaultErrorHandler               | `errors.go`          | 9a           |
| 9d  | Update tests                                            | `*_test.go`          | 9b           |
| 10a | Add notificationEvent field to App                      | `app.go`             | —            |
| 10b | Thread notificationEvent through handlerConfig          | `options.go`         | 10a          |
| 10c | Update notify.go to use config                          | `notify.go`          | 10b          |
| 10d | Update tests                                            | `*_test.go`          | 10c          |
| 11a | Define Enforcer interface                               | `authz.go`           | —            |
| 11b | Update App and handlers to use interface                | `app.go`, `authz.go` | 11a          |
| 11c | Update tests to use mock                                | `*_test.go`          | 11b          |
| 12a | Write BDD test: "Consumer creates a command handler"    | new `bdd_test.go`    | —            |
| 12b | Write BDD test: "Consumer queries with authorization"   | `bdd_test.go`        | 12a          |
| 12c | Write BDD test: "Consumer renders templ components"     | `bdd_test.go`        | 12a          |
