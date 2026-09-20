# Status Report: 2026-05-21 16:42 — Post-Deduplication & Error Context Session

**Branch:** `master` | **Base commit:** `e517f7a` | **Coverage:** 96.1% (root), 91.7% (usermgmt) | **Tests:** 110 passing (98 root + 12 usermgmt) | **Lint:** 8 issues (4 pre-existing)

---

## a) FULLY DONE ✅

### 1. Code Deduplication (art-dupl threshold 40): 13 → 1 clone groups

Eliminated 12 of 13 semantic clone groups:

| Change                                           | Files                                                   | Impact                                         |
| ------------------------------------------------ | ------------------------------------------------------- | ---------------------------------------------- |
| `encodeJSONResult` now sets `Content-Type`       | `testing_test.go`, `bdd_test.go`, `integration_test.go` | Replaced 3 identical inline render closures    |
| `assertHTMXErrorRedirect` helper                 | `testing_test.go`, `coverage_test.go`, `errors_test.go` | Extracted repeated error-redirect-test pattern |
| `newCommandApp()` reuse                          | `integration_test.go`                                   | Replaced 2 identical `BeforeEach` blocks       |
| `csrfTokenOnceHandler`/`csrfTokenCaptureHandler` | `csrf_test.go`                                          | Extracted 3 inline handler closures            |
| `tracingMW` parameterized helper                 | `middleware_test.go`                                    | Replaced 2 identical middleware definitions    |
| `trackAfterDispatch` helper                      | `hooks_test.go`                                         | DRYed up AfterDispatch test setup              |
| `testRemoteAddr` constant                        | `httputil_test.go`                                      | Extracted repeated string, fixed goconst lint  |
| `DescribeTable` for ClientIP                     | `httputil_test.go`                                      | Consolidated 3 identical test structures       |
| Removed duplicated correlation ID tests          | `hooks_test.go`                                         | Already covered by `context_test.go`           |
| Removed duplicated EventOptions tests            | `bdd_test.go`                                           | Already covered by `context_test.go`           |
| `decodeGetUserJSONQuery()` reuse                 | `app_test.go`                                           | Replaced 2 identical inline decoder closures   |
| Merged identical test cases                      | `middleware_test.go`                                    | "empty extractor" == "unparseable IDs"         |

**Remaining 1 clone:** Standard Go benchmark boilerplate (`for i := 0; i < b.N; i++ { ... }`) — false positive, not extractable.

### 2. Error Context Enrichment (pre-existing uncommitted work)

| File                | Change                                                                                          |
| ------------------- | ----------------------------------------------------------------------------------------------- |
| `decoder.go`        | All decode errors now include `maxBodySize` in message (`maxBodySize=%d: ...`)                  |
| `usermgmt/authz.go` | `EnforceEx`, `RolesForUser`, `ImplicitRolesForUser` errors include `domain`/`sub`/`obj` context |
| `usermgmt/user.go`  | `SetPasswordWithCost` error includes `cost` value, switched from `errors.Wrapf` to `fmt.Errorf` |

### 3. Build & Test Status

- ✅ `go build ./...` — clean
- ✅ `go vet ./...` — clean
- ✅ `go test ./... -count=1 -race` — 110/110 passing
- ✅ `go test ./... -cover` — 96.1% root, 91.7% usermgmt

---

## b) PARTIALLY DONE 🔧

### 1. Lint Clean-up (8 remaining issues)

Pre-existing issues NOT introduced in this session:

| Linter     | File                         | Issue                                                 |
| ---------- | ---------------------------- | ----------------------------------------------------- |
| `errcheck` | `httputil.go:15`             | `json.NewEncoder(w).Encode(v)` return value unchecked |
| `golines`  | `decoder.go:21`              | Line too long (introduced by error context work)      |
| `golines`  | `csrf_test.go:70`            | Line too long (introduced by dedup work)              |
| `noctx`    | `testing_test.go:280`        | `httptest.NewRequest` without context                 |
| `revive`   | `logging.go:150,158,213,219` | 4 exported `StatusRecorder` methods missing comments  |

**2 of these were introduced in this session** (`golines` in `decoder.go` and `csrf_test.go`) — need formatting fix.

### 2. usermgmt Coverage (91.7%)

Down from 95.6% — likely due to the new error context paths needing test coverage for the enriched error messages.

---

## c) NOT STARTED ⏳

1. **golangci-lint auto-fix** for the 2 `golines` issues (just formatting)
2. **errcheck fix** in `httputil.go` — handle `json.Encoder.Encode` error return
3. **noctx fix** in `testing_test.go` — use `NewRequestWithContext`
4. **StatusRecorder method comments** — add doc comments to 4 exported methods
5. **usermgmt coverage recovery** — test new error context paths
6. **docs/status/2026-05-21_02-00** — only formatting changes, needs content review

---

## d) TOTALLY FUCKED UP 💥

**Nothing is broken.** All tests pass, build is clean, race detector clean.

**Concerns:**

- The `decoder.go` error context changes add `maxBodySize` to error messages but these are user-facing errors — the format string `maxBodySize=%d: ...` is a debugging aid, not a user-friendly message. This may leak implementation details.
- The `golines` formatting issues are self-inflicted from this session's edits.

---

## e) WHAT WE SHOULD IMPROVE 📈

### Architecture & Code Quality

1. **Error messages in decoder.go** — `maxBodySize=%d: ...` format is developer-oriented. Consider structured error fields instead of string formatting for machine-parseable errors.
2. **StatusRecorder lacks http.Flusher/http.Pusher/http.Hijacker comments** — these are interface compliance methods that deserve documentation explaining _why_ they exist.
3. **httputil.go WriteJSON** — silently swallows `json.Encoder.Encode` error. Should return error or at minimum log it.
4. **usermgmt coverage regression** — 91.7% is below the project standard. The new error context wrapping in `authz.go` and `user.go` needs test coverage.
5. **Test helper sprawl** — `testing_test.go` has grown to 284 lines with many helpers. Consider if some (like `csrfTokenOnceHandler`) should be in their respective test files instead.

### Process

6. **Pre-commit hook should catch golines issues** — the formatter should auto-fix line length before commit.
7. **AGENTS.md coverage claim** says 95.7% but actual is 96.1% (root) / 91.7% (usermgmt). Should update.
8. **Stale LSP warnings** — AGENTS.md documents ~31 stale LSP warnings, but CLI `golangci-lint run` shows only 8 real issues. This discrepancy should be re-verified and AGENTS.md updated.

---

## f) Top 25 Things We Should Get Done Next

### Priority 1: Fix Session-Introduced Issues (5-10 min each)

| # | Task                                                               | Impact                               |
| - | ------------------------------------------------------------------ | ------------------------------------ |
| 1 | Fix `golines` formatting in `decoder.go` and `csrf_test.go`        | Zero lint warnings from this session |
| 2 | Fix `noctx` in `testing_test.go:280` — use `NewRequestWithContext` | 1 less lint warning                  |
| 3 | Fix `errcheck` in `httputil.go:15` — handle Encode error           | 1 less lint warning                  |
| 4 | Add doc comments to 4 `StatusRecorder` exported methods            | 4 less lint warnings                 |
| 5 | Run `gofumpt`/`goimports` on all changed files                     | Formatting consistency               |

### Priority 2: Test Coverage Recovery (30-60 min)

| #  | Task                                                              | Impact                                |
| -- | ----------------------------------------------------------------- | ------------------------------------- |
| 6  | Test `decoder.go` error context format strings                    | Verify new error messages are correct |
| 7  | Test `usermgmt/authz.go` error context wrapping                   | Recover usermgmt coverage to 95%+     |
| 8  | Test `usermgmt/user.go` `SetPasswordWithCost` error includes cost | Verify new error format               |
| 9  | Add edge case test for maxBodySize=0 in decoder                   | Defensive coverage                    |
| 10 | Verify all error context changes don't break `errors.Is` chains   | Regression prevention                 |

### Priority 3: Documentation & Memory (15-30 min)

| #  | Task                                                                  | Impact           |
| -- | --------------------------------------------------------------------- | ---------------- |
| 11 | Update AGENTS.md coverage claim (95.7% → 96.1%/91.7%)                 | Accurate memory  |
| 12 | Update AGENTS.md with deduplication results                           | Accurate memory  |
| 13 | Update AGENTS.md lint status (was "7 warnings fixed" → "8 remaining") | Accurate memory  |
| 14 | Update AGENTS.md with error context enrichment decision               | Future reference |
| 15 | Review `docs/status/2026-05-21_02-00` content changes                 | Ensure accuracy  |

### Priority 4: Structural Improvements (1-2 hours)

| #  | Task                                                                        | Impact                     |
| -- | --------------------------------------------------------------------------- | -------------------------- |
| 16 | Extract `csrf_test.go` helpers to `csrf_helpers_test.go`                    | Better test organization   |
| 17 | Consider `httputil.go` WriteJSON returning error instead of void            | API correctness            |
| 18 | Add integration test for full error context propagation end-to-end          | Confidence in error chains |
| 19 | Review all `fmt.Errorf` wraps for double-wrapping risk                      | Error chain hygiene        |
| 20 | Verify `errors.Is(err, ErrDecodeFailed)` still works after context wrapping | Regression test            |

### Priority 5: Nice-to-Have (Backlog)

| #  | Task                                                         | Impact                               |
| -- | ------------------------------------------------------------ | ------------------------------------ |
| 21 | Run art-dupl at threshold 20 to catch smaller clones         | Zero tolerance                       |
| 22 | Add `//nolint` directives for acceptable lint warnings       | Explicit documentation of exceptions |
| 23 | Benchmark the error context string formatting overhead       | Performance awareness                |
| 24 | Consider structured error types instead of formatted strings | Machine-parseable errors             |
| 25 | Pre-commit hook integration for art-dupl                     | Continuous dedup monitoring          |

---

## g) Top #1 Question I Cannot Figure Out Myself 🤔

**Should `decoder.go` error messages include `maxBodySize` in the format string, or should this be a structured error field?**

The current approach (`fmt.Errorf("maxBodySize=%d: %w", maxBodySize, ...)`) bakes implementation details into the error message. This is great for debugging but:

- It changes the error message format, potentially breaking consumers who match on error strings
- It makes the error message harder to parse programmatically
- The `maxBodySize` is a config value, not per-request context — logging it on every decode failure is redundant

**Recommendation:** Keep `maxBodySize` only in the _first_ error (body read failure) where it matters for diagnosis, but not in the decode/parse errors where it's noise. Or better: use a structured error type with a `MaxBodySize` field.

---

## Metrics Summary

| Metric                  | Value            | Trend                         |
| ----------------------- | ---------------- | ----------------------------- |
| Clone groups (t=40)     | 1                | ↓ from 13                     |
| Root test coverage      | 96.1%            | ↑ from 95.7%                  |
| usermgmt coverage       | 91.7%            | ↓ from 95.6%                  |
| Total tests passing     | 110              | → stable                      |
| Lint issues             | 8                | ↑ from 0 (new golines)        |
| Build                   | ✅ clean         | → stable                      |
| Race detector           | ✅ clean         | → stable                      |
| Production code changes | 3 files          | decoder.go, authz.go, user.go |
| Test code changes       | 9 files          | DRY improvements              |
| Net lines changed       | -70 (137+, 207-) | Cleaner codebase              |

---

_Generated by Crush — 2026-05-21 16:42_
