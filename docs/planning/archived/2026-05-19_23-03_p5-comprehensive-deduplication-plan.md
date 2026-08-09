# Comprehensive Execution Plan — P5 Deduplication & Quality

**Date:** 2026-05-19_23-03
**Scope:** 13 open TODOs from P5 comprehensive review
**Constraint:** Each task ≤ 12 minutes

---

## Pareto Breakdown

### 1% → 51% of Result (Correctness + Production Safety)

Tasks 1–4 fix real bugs or production risks. Without these, deduplication is polishing broken glass.

### 4% → 64% of Result (High-Impact Deduplication)

Tasks 5–14 eliminate the worst duplication. 22 functions collapse to ~5 generics + thin wrappers.

### 20% → 80% of Result (Medium Deduplication + Polish)

Tasks 15–23 clean up remaining duplication and file organization.

---

## Task Table

| #  | Tier    | Task                                                                                                              | Files                                     | Effort | Impact               | Customer Value        |
| -- | ------- | ----------------------------------------------------------------------------------------------------------------- | ----------------------------------------- | ------ | -------------------- | --------------------- |
| 1  | **1%**  | Fix context mismatch: pass `ctx` from `handleQueryDispatch` to `applyQueryResponse`                               | `handler.go:112-167`                      | 5min   | Critical correctness | Bug prevention        |
| 2  | **1%**  | Add test for context timeout during query render error                                                            | `handler_test.go` or `timeout_test.go`    | 10min  | High                 | Regression protection |
| 3  | **1%**  | Add `MaxKeys` to `RateLimiterConfig` + eviction when exceeded                                                     | `ratelimit.go`                            | 10min  | High                 | Production safety     |
| 4  | **1%**  | Add test for rate limiter max-keys eviction                                                                       | `ratelimit_test.go`                       | 10min  | High                 | Regression protection |
| 5  | **4%**  | Extract generic `htmxBoolField(r, extract func, header) bool` + `htmxStringField(r, extract func, header) string` | `htmx.go`                                 | 8min   | High                 | 8→2 functions         |
| 6  | **4%**  | Rewrite 8 HTMX accessor functions to delegate to generic helpers                                                  | `htmx.go`                                 | 8min   | High                 | Locality              |
| 7  | **4%**  | Run tests to verify HTMX accessor refactor                                                                        | —                                         | 3min   | Medium               | Verification          |
| 8  | **4%**  | Extract `decodeAndSet[T, R](bodyDec, mapper, setter)` generic decoder                                             | `options.go`                              | 10min  | High                 | 4→1 + 4 wrappers      |
| 9  | **4%**  | Rewrite 4 Decode functions to delegate to generic                                                                 | `options.go`                              | 8min   | High                 | Locality              |
| 10 | **4%**  | Run tests to verify decoder refactor                                                                              | —                                         | 3min   | Medium               | Verification          |
| 11 | **4%**  | Extract `validateDispatch[T](decoder, validator, setter)` generic                                                 | `options.go`                              | 10min  | High                 | 2→1 + 2 wrappers      |
| 12 | **4%**  | Rewrite ValidateCommand/Query to delegate to generic                                                              | `options.go`                              | 5min   | High                 | Locality              |
| 13 | **4%**  | Run tests to verify validation refactor                                                                           | —                                         | 3min   | Medium               | Verification          |
| 14 | **4%**  | Unify `notifyOption` and `triggerNotification` into single `notify(level, msg, event)`                            | `notify.go` + `response.go`               | 10min  | High                 | 2 impls → 1           |
| 15 | **20%** | Extract `contextFields(r) map[string]string` from logging formatters                                              | `logging.go`                              | 8min   | Medium               | 3 blocks → 1          |
| 16 | **20%** | Rewrite 3 logging formatters to use `contextFields`                                                               | `logging.go`                              | 10min  | Medium               | Locality              |
| 17 | **20%** | Run tests to verify logging refactor                                                                              | —                                         | 3min   | Medium               | Verification          |
| 18 | **20%** | Extract `handleErrorCore(w, r, err, loginRedirect, writeBody)` from error handlers                                | `errors.go`                               | 10min  | Medium               | 2→1 + 2 wrappers      |
| 19 | **20%** | Rewrite both error handlers to delegate to core                                                                   | `errors.go`                               | 8min   | Medium               | Locality              |
| 20 | **20%** | Run tests to verify error handler refactor                                                                        | —                                         | 3min   | Medium               | Verification          |
| 21 | **20%** | Extract `parseID[T](s string, parse func(string) (T, error), label string) (T, error)` generic                    | `context.go`                              | 8min   | Medium               | 3→1                   |
| 22 | **20%** | Rewrite 3 Parse functions to delegate to generic                                                                  | `context.go`                              | 5min   | Medium               | Locality              |
| 23 | **20%** | Split `csrf.go`: move template helpers to `csrf_helpers.go`                                                       | `csrf.go` → `csrf.go` + `csrf_helpers.go` | 10min  | Low                  | File readability      |
| 24 | **20%** | Fix nil context in usermgmt tests                                                                                 | `usermgmt/handler_test.go`                | 5min   | Low                  | Lint clean            |
| 25 | **20%** | Run full test suite (both modules) + lint                                                                         | —                                         | 5min   | Critical             | Final verification    |

---

## Dependency Graph

```
Task 1 ──→ Task 2
Task 3 ──→ Task 4
Task 5 ──→ Task 6 ──→ Task 7
Task 8 ──→ Task 9 ──→ Task 10
Task 11 → Task 12 → Task 13
Task 14 (standalone)
Task 15 → Task 16 → Task 17
Task 18 → Task 19 → Task 20
Task 21 → Task 22
Task 23 (standalone)
Task 24 (standalone)
Task 25 (after all)
```

## Execution Order

1. **Batch 1 (Correctness):** Tasks 1–4 — fix bugs first
2. **Batch 2 (HTMX dedup):** Tasks 5–7 — simplest refactor, proves pattern
3. **Batch 3 (Decoder dedup):** Tasks 8–10 — same pattern, different files
4. **Batch 4 (Validation dedup):** Tasks 11–13 — builds on decoder pattern
5. **Batch 5 (Notification):** Task 14 — independent, simple unification
6. **Batch 6 (Logging):** Tasks 15–17 — extract shared helper
7. **Batch 7 (Error handlers):** Tasks 18–20 — shared core
8. **Batch 8 (Parse IDs):** Tasks 21–22 — generic parse helper
9. **Batch 9 (Polish):** Tasks 23–24 — file split + test fix
10. **Batch 10 (Verify):** Task 25 — full suite

## Metrics

- **25 tasks** total
- **Average effort:** ~7 min/task
- **Total estimated effort:** ~185 min (~3 hours)
- **Functions eliminated:** ~25 functions collapsed to ~8 generics + thin wrappers
- **Files modified:** 7 production files, 2 test files
- **Net LOC change:** ~-80 lines (removing duplication)

---

## D2 Execution Graph

```d2
direction: down

title: P5 Execution Plan — Dependency Graph

batch1: {
  shape: rectangle
  style: {fill: "#FFCDD2"}
  label: "Batch 1: Correctness"
  t1: "1. Fix ctx mismatch" -> t2: "2. Test ctx timeout"
  t3: "3. Rate limiter MaxKeys" -> t4: "4. Test MaxKeys eviction"
}

batch2: {
  shape: rectangle
  style: {fill: "#C8E6C9"}
  label: "Batch 2: HTMX Accessors"
  t5: "5. Generic htmxField" -> t6: "6. Rewrite 8 accessors" -> t7: "7. Verify"
}

batch3: {
  shape: rectangle
  style: {fill: "#BBDEFB"}
  label: "Batch 3: Decoders"
  t8: "8. Generic decodeAndSet" -> t9: "9. Rewrite 4 decoders" -> t10: "10. Verify"
}

batch4: {
  shape: rectangle
  style: {fill: "#D1C4E9"}
  label: "Batch 4: Validation"
  t11: "11. Generic validate" -> t12: "12. Rewrite 2 validators" -> t13: "13. Verify"
}

batch5: {
  shape: rectangle
  style: {fill: "#FFF9C4"}
  label: "Batch 5: Notification"
  t14: "14. Unify notify impl"
}

batch6: {
  shape: rectangle
  style: {fill: "#FFE0B2"}
  label: "Batch 6: Logging"
  t15: "15. Extract contextFields" -> t16: "16. Rewrite 3 formatters" -> t17: "17. Verify"
}

batch7: {
  shape: rectangle
  style: {fill: "#B2DFDB"}
  label: "Batch 7: Error Handlers"
  t18: "18. Extract handleErrorCore" -> t19: "19. Rewrite 2 handlers" -> t20: "20. Verify"
}

batch8: {
  shape: rectangle
  style: {fill: "#F0F4C3"}
  label: "Batch 8: Parse IDs"
  t21: "21. Generic parseID" -> t22: "22. Rewrite 3 parsers"
}

batch9: {
  shape: rectangle
  style: {fill: "#D7CCC8"}
  label: "Batch 9: Polish"
  t23: "23. Split csrf.go"
  t24: "24. Fix nil context"
}

batch10: {
  shape: rectangle
  style: {fill: "#F8BBD0"}
  label: "Batch 10: Final"
  t25: "25. Full test suite + lint"
}

batch1 -> batch2 -> batch3 -> batch4 -> batch5 -> batch6 -> batch7 -> batch8 -> batch9 -> batch10
```
