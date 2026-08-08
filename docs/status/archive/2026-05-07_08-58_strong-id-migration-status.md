# Status Report — cqrs-htmx Strong ID Migration

**Date:** 2026-05-07 08:58 | **Session:** Strong ID + Cleanup

---

## Executive Summary

Migrated user identity from primitive `string` to strongly-typed `id.UserID` (ULID-backed branded type from `go-branded-id`). Removed dead sentinels. All 148 tests pass, 0 lint issues, 95.5% coverage.

---

## Metrics

| Metric      | Before Session | After Session | Delta                                        |
| ----------- | -------------- | ------------- | -------------------------------------------- |
| Test specs  | 147            | 148           | +1                                           |
| Coverage    | 95.7%          | 95.5%         | -0.2% (new exported `NewUserID()` uncovered) |
| Lint issues | 0              | 0             | —                                            |
| Prod files  | 10             | 10            | —                                            |
| Total lines | 3,793          | 3,827         | +34                                          |
| Sentinels   | 9              | 7             | -2 (removed dead)                            |

---

## a) FULLY DONE

### 1. Strongly-typed UserID ✅

- **Commit:** `764f70c`
- `WithUserID(ctx, string)` → `WithUserID(ctx, UserID)` where `type UserID = id.UserID`
- `UserIDFromContext(ctx) string` → `UserIDFromContext(ctx) UserID`
- Added `NewUserID()`, `ParseUserID(s)`, `MustParseUserID(s)` as package exports
- `EventOptionsFromContext` simplified — no re-parsing, direct `IsZero()` check
- `ContextEnrichmentMiddleware` parses string → `UserID`, silently drops invalid ULIDs
- `App.enrichUserID` parses string → `UserID`, uses `IsZero()` check
- `executeAuthorization` uses `userID.String()` for Casbin `Enforce`
- All 13 test files updated with valid ULIDs and typed assertions

### 2. Enforce(nil) error context fix ✅

- **Commit:** `764f70c`
- `authz.go:43` — was `fmt.Errorf("%w: %s/%s", ErrEnforcerNil, resource, action)` (missing `subject`)
- Now `fmt.Errorf("%w: subject=%s resource=%s action=%s", ErrEnforcerNil, subject, resource, action)`
- Consistent with `ErrForbidden` and Casbin error messages in same file

### 3. Dead sentinels removed ✅

- **Commit:** `3ec00f8`
- Removed `ErrNoUserID` and `ErrRendererMissing` — exported but never returned by any code path, never checked by any test
- Removed their `RegisterClassification` calls from `registerErrorClassifications()`

### 4. Middleware invalid ID test ✅

- **Commit:** `e576b30`
- Verifies `ContextEnrichmentMiddleware` silently drops unparseable user IDs (e.g., `"not-a-ulid"`)
- Context left without user ID, auth fails downstream with `ErrUnauthorized`

### 5. Documentation updated ✅

- **Commit:** `7612b1b`
- AGENTS.md: removed gotcha #9 (dead sentinels gone), renumbered, added gotcha #16 (middleware ID drop), updated architecture table
- CHANGELOG.md: added `NewUserID/ParseUserID/MustParseUserID` to Added, breaking change to Changed, sentinel removal to Removed, `Enforce(nil)` fix to Fixed
- README.md: context propagation example uses `MustParseUserID`

---

## b) PARTIALLY DONE

### 1. Strong ID analysis — 1 of 2 violations fixed

- ✅ `UserID` in `context.go` — migrated to `id.UserID`
- ⏭️ `TriggerID` in `HTMXRequest` — **deliberately skipped** (see section d)

### 2. TODO_LIST.md — stale

- Still lists dead sentinel removal as TODO (`- [ ] Remove dead sentinels`)
- Still lists `95.7% test coverage` — now 95.5%
- Metrics table in FEATURES.md still says `92.6%` and `137 specs` — very stale

---

## c) NOT STARTED

### From TODO_LIST.md P1–P4 (unchanged from previous session):

| #  | Item                                                                     | Priority |
| -- | ------------------------------------------------------------------------ | -------- |
| 1  | Export `HeaderTrue` or provide test helper (tests hardcode `"true"` 34×) | P1       |
| 2  | Dispatch lifecycle hooks (`OnBeforeDispatch`/`OnAfterDispatch`)          | P3       |
| 3  | Request validation middleware                                            | P3       |
| 4  | JSON error response option                                               | P3       |
| 5  | Correlation ID propagation                                               | P3       |
| 6  | Timeout propagation                                                      | P3       |
| 7  | Godoc examples                                                           | P4       |
| 8  | CONTRIBUTING.md                                                          | P4       |
| 9  | golangci-lint CI/CD                                                      | P4       |
| 10 | Benchmark tests                                                          | P4       |

---

## d) TOTALLY FUCKED UP / DELIBERATELY SKIPPED

### 1. `TriggerID` branded type — Skipped with justification

The analysis tool flagged `HTMXRequest.TriggerID string` as a "strong ID violation". This is a **false positive**:

- `TriggerID` is an HTML element ID from the `HX-Trigger` request header
- Values are arbitrary strings like `"submit-btn"`, `"modal"`, `"my-button"`
- `go-branded-id` is for ULID-backed domain entity identifiers
- Branding it as `id.ID[TriggerBrand, string]` would force consumers to parse DOM IDs through validation — wrong abstraction entirely
- **Decision:** Keep as `string`. The analysis tool doesn't distinguish domain IDs from header values.

### 2. `go-branded-id` as direct dependency — Skipped

- Currently `// indirect` via `go-cqrs-lite/core/pkg/id`
- We access it through `type UserID = id.UserID` where `id.UserID = cbid.ID[userMarker, ulid.ULID]`
- Adding a direct import would be artificial — the type alias chain is correct
- `go mod tidy` confirms `// indirect` is the right status

### 3. Coverage regression 95.7% → 95.5%

- New exported function `NewUserID()` has 0% coverage — it's a trivial wrapper around `id.NewUserID()`
- Net coverage dropped 0.2% despite adding a test
- Not a real quality issue — one trivial wrapper uncovered

---

## e) WHAT WE SHOULD IMPROVE

### Critical

1. **TODO_LIST.md and FEATURES.md are stale** — they reference old metrics, dead sentinels as TODO, outdated coverage numbers
2. **`NewUserID()` has 0% coverage** — trivial fix, add one test call

### Architecture

3. **`handleQueryDispatch` at 72.7% coverage** — lowest function coverage in the codebase, multiple uncovered branches
4. **`decodeFormValues` at 72.7%** — form decoding paths undertested
5. **`decodeFormBody` at 80.0%** — error branches not covered

### Design

6. **Middleware silently drops invalid IDs** — intentional but could surprise consumers. Consider logging or an error callback.
7. **`ParseUserID` is a thin wrapper** — wraps `id.ParseUserID` just to satisfy `wrapcheck`. Consider `//nolint:wrapcheck` on the alias instead.
8. **`UserIDExtractor` still returns `string`** — the boundary between "consumer extracts string from JWT" and "library parses to UserID" could be cleaner. Consider accepting `UserID` directly in a future API.

### Test Quality

9. **Test enforcer uses hardcoded ULID subjects** — `adminUserID` and `viewerUserID` are valid ULIDs used as Casbin policy subjects. This is correct but unusual — most Casbin setups use role names, not user IDs, as subjects. Tests might not reflect real-world usage patterns.
10. **`coverage_test.go` is 565 lines** — largest file, mix of coverage and behavioral tests. Could be split.

---

## f) Top 25 Things to Do Next

Sorted by impact × effort (Pareto order):

### High Impact, Low Effort

1. **Fix TODO_LIST.md** — Mark dead sentinels as done, update metrics
2. **Fix FEATURES.md** — Update metrics table (coverage, specs), update User Identity Propagation description
3. **Add `NewUserID()` test** — Single test to cover 0% → 100%, bring coverage back to 95.7%+
4. **Test `Enforce` casbin error path** — `authz.go:47` (enforcer returns error) at 87.5%, add one test
5. **Test `handleQueryDispatch` uncovered branches** — 72.7% → higher, multiple query handler paths untested

### High Impact, Medium Effort

6. **Export `HeaderTrue` or add test helper** — 34 test hardcodes of `"true"`, P1 TODO item
7. **Cover `decodeFormValues` error paths** — 72.7%, form query parameter parsing undertested
8. **Cover `decodeFormBody` error paths** — 80.0%, form body parsing undertested
9. **Cover `setTriggerWithDetail` branches** — 88.2%, JSON detail serialization
10. **Cover `MapError` remaining branches** — 93.3%, some error families untested

### Medium Impact, Medium Effort

11. **Add correlation ID propagation** — `WithCorrelationID`/`CorrelationIDFromContext`, high value for distributed tracing
12. **Add JSON error response option** — `JSONErrorHandler` alternative to plain text
13. **Split `coverage_test.go`** — 565 lines is too large, separate behavioral from pure coverage tests
14. **Add dispatch lifecycle hooks** — `OnBeforeDispatch`/`OnAfterDispatch` for observability
15. **Add timeout propagation** — Context deadline from request, prevent runaway dispatches

### Medium Impact, Higher Effort

16. **Request validation middleware** — Schema validation in decode pipeline
17. **Godoc examples** — `SwapStrategy`, `Config`, `Response`, `HTMXRequest`
18. **Benchmark tests** — `MapError`, `parseHTMXRequest`, `HTMXMiddleware`
19. **CONTRIBUTING.md** — Document lint config, test patterns, naming conventions
20. **golangci-lint CI/CD** — GitHub Actions enforcement

### Lower Impact (Polish)

21. **Consider `//nolint:wrapcheck` on `ParseUserID`** — Thin wrapper, wrapping adds noise not value
22. **Add logging/callback for dropped invalid IDs** — Middleware silently drops, consumers may want visibility
23. **Document `.golangci.yml` decisions** — Inline comments explaining exclusions
24. **Review `UserIDExtractor` API for v2** — Could accept `UserID` directly instead of `string`
25. **Consider `go-cqrs-lite` version bump** — Check if newer version available with improvements

---

## g) Top Question I Cannot Figure Out

**Should `UserIDExtractor` remain `func(*http.Request) string` or become `func(*http.Request) (UserID, error)`?**

Arguments for keeping `string`:

- Consumers extract from JWT claims, session cookies, headers — all string-based
- Simpler API, no error handling burden on consumers
- Library handles parsing centrally (single validation point)

Arguments for changing to `(UserID, error)`:

- Fail-fast: consumer knows immediately if their extraction produces invalid IDs
- Type-safe: consumer must consciously produce a valid UserID
- Consistent: entire internal pipeline uses typed IDs, why leave the boundary as string?

**I cannot decide this without understanding how consumers actually extract user IDs.** If they always have ULIDs from JWT `sub` claims, the typed API is better. If they sometimes have opaque tokens, role names, or non-ULID identifiers, the string API is necessary.

---

## Commit Log (This Session)

```
7612b1b docs: update AGENTS.md and CHANGELOG for dead sentinel removal
e576b30 test: add coverage for middleware dropping unparseable user IDs
3ec00f8 refactor: remove dead sentinels ErrNoUserID and ErrRendererMissing
764f70c feat: strongly-typed UserID (ULID-backed branded type)
```

## Files Changed (This Session)

```
14 files changed, 157 insertions(+), 89 deletions(-)
```

## Health Check

| Check                 | Status          |
| --------------------- | --------------- |
| `go test ./... -race` | ✅ 148/148 pass |
| `golangci-lint run`   | ✅ 0 issues     |
| `go build ./...`      | ✅ clean        |
| Coverage              | 95.5%           |
| Banned deps           | 0               |
