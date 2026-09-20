# Status Report — 2026-05-22 Session 2 (00:13)

## Build & Test

| Module                            | Status              | Coverage | Tests |
| --------------------------------- | ------------------- | -------- | ----- |
| Root (`cqrs-htmx`)                | ✅ PASS             | 97.0%    | ~300+ |
| Usermgmt (`usermgmt/`)            | ✅ PASS             | 91.2%    | ~120+ |
| Integration (`integration_test/`) | ✅ PASS             | N/A      | 4     |
| **Build**                         | ✅ `go build ./...` |          |       |
| **Lint**                          | ❌ 10 issues        |          |       |

## Git State

- Branch: `master`, 2 commits ahead of origin
- Working tree: clean
- Last commit: `cf86e40 docs: update AGENTS.md and TODO_LIST.md with 2026-05-22 session results`

---

## A) FULLY DONE ✅

| Task    | What                                                          | Impact                               |
| ------- | ------------------------------------------------------------- | ------------------------------------ |
| T01     | CI pipeline fixed (GOWORK=off, remove GOFLAGS, usermgmt jobs) | CI was completely broken             |
| T02     | Integration test module (`integration_test/`)                 | Cross-module bridge verified         |
| T03     | CSRF coverage (fieldName, sameSite, csrfTokenFromContext)     | Edge cases covered                   |
| T04     | Redirect URL coverage (data:, scheme-relative, unparseable)   | Security edge cases                  |
| T05     | Logging Push/Hijack coverage                                  | HTTP interface delegation            |
| T06-T08 | Usermgmt handler/authz/service coverage                       | +20 tests, 91.7→92.4%                |
| T09     | CSRFConfig.Secure runtime warning                             | Production safety alert              |
| T10     | policyWrapErr coverage (0→100%)                               | Formatter fully tested               |
| T13     | RateLimiterConfig signedness unified                          | `uint` everywhere internally         |
| T14     | Usermgmt HTTP timeout (`HandlerConfig.Timeout`)               | Configurable per-handler timeout     |
| T18     | Rate limiter eviction O(n) → O(log n) min-heap                | Performance fix for large key spaces |
| T19     | CSRF fuzz tests (`FuzzCSRFConfigValidation`)                  | Config safety under fuzz             |

## B) PARTIALLY DONE ⚠️

| Task          | What's done                    | What remains                                                                                                                   |
| ------------- | ------------------------------ | ------------------------------------------------------------------------------------------------------------------------------ |
| Coverage gaps | Root 97.0%, usermgmt 91.2%     | `sameSite` 66.7%, `csrfTokenFromRequest` 66.7%, `Hijack` 60%, `handleLogout` 64.3%, `Apply` 69.2%                              |
| Lint          | 0 real warnings before session | 10 new lint issues introduced by session's changes (revive comments, errcheck, forcetypeassert, recvcheck, noctx, exhaustruct) |

## C) NOT STARTED 📋

| Task              | Description                                                                         | Effort                |
| ----------------- | ----------------------------------------------------------------------------------- | --------------------- |
| T11               | TypedHandler[T] — API design decision                                               | HIGH (design)         |
| T21               | UserID type split resolution                                                        | HIGH (owner decision) |
| Magic strings     | Extract `"Bearer "`, `"session_token"`, log prefixes, password messages in usermgmt | LOW                   |
| `writeJSON` dedup | usermgmt has identical private `writeJSON` as root's exported `WriteJSON`           | LOW                   |

## D) TOTALLY FUCKED UP 💥

| Issue                           | Root Cause                                                                        | Fix                          |
| ------------------------------- | --------------------------------------------------------------------------------- | ---------------------------- |
| **10 lint warnings introduced** | New code (min-heap, mockPusher, logging tests) didn't respect existing lint rules | Fix all 10 (see E below)     |
| `evictionHeap` recvcheck        | Mix of pointer/non-pointer receivers on heap methods                              | Consistent pointer receivers |
| `forcetypeassert` x2            | Unchecked `heap.Pop().(*evictionEntry)` and `x.(*evictionEntry)`                  | Add safety checks            |
| `mockPusher` exhaustruct        | Missing embedded `http.ResponseWriter` field                                      | Add the field                |
| `noctx` in testing_test.go      | `httptest.NewRequest` without context                                             | Use `NewRequestWithContext`  |
| `WriteJSON` errcheck            | `json.NewEncoder(w).Encode(v)` return not checked                                 | Check error                  |
| 4x revive on StatusRecorder     | Missing doc comments on `WriteHeader`, `Push`, `Flush`, `Hijack`                  | Add godoc                    |
| Integration test `go.mod` stale | `go mod tidy` needed, `go-cqrs-lite/core` indirect                                | Run `go mod tidy`            |

## E) WHAT WE SHOULD IMPROVE 🔧

### Immediate (this session)

1. **Fix all 10 lint warnings** — These are regressions from our own changes
2. **Root coverage: `sameSite` 66.7%, `csrfTokenFromRequest` 66.7%, `Hijack` 60%** — Easy wins
3. **Usermgmt coverage: `handleLogout` 64.3%** — The lowest coverage function
4. **Extract magic strings in usermgmt** — `"Bearer "`, `"session_token"`, password messages
5. **Integration test go.mod tidy** — Stale deps

### Next session

6. **Usermgmt `Apply` coverage 69.2%** — Test remove+add group error paths
7. **`writeJSON` dedup** — usermgmt should use root's exported `WriteJSON`
8. **Root `Enforce` 87.5%** — Missing `enforcer == nil` path in tests
9. **Root `MapError` 93.3%** — Missing nil error + conflict/corruption/transient families
10. **Usermgmt `generateToken` 75%** — `rand.Read` error path

### Design decisions needed

11. **TypedHandler[T] API design** — Go methods can't have type params; needs top-level function
12. **UserID type split** — `usermgmt.UserID` (string-backed) vs `cqrshtmx.UserID` (ULID-backed)
13. **BrandNamer on upstream marker types** — go-cqrs-lite marker types are unexported

### Blocked

14. **Dependabot alerts** — `gh auth login` required
15. **Pre-commit hook** — `go-structure-linter` fails on flat package (intentional)

---

## F) Top 25 Next Actions (sorted by impact/effort)

| #  | Action                                           | Impact | Effort | Type         |
| -- | ------------------------------------------------ | ------ | ------ | ------------ |
| 1  | Fix 4 revive comments on StatusRecorder methods  | HIGH   | 5min   | Lint fix     |
| 2  | Fix `WriteJSON` errcheck                         | HIGH   | 2min   | Lint fix     |
| 3  | Fix `evictionHeap` recvcheck (pointer receivers) | HIGH   | 2min   | Lint fix     |
| 4  | Fix `forcetypeassert` x2 in ratelimit.go         | HIGH   | 3min   | Lint fix     |
| 5  | Fix `mockPusher` exhaustruct                     | HIGH   | 2min   | Lint fix     |
| 6  | Fix `noctx` in testing_test.go                   | HIGH   | 1min   | Lint fix     |
| 7  | Add `sameSite` coverage (66.7→100%)              | MED    | 10min  | Coverage     |
| 8  | Add `csrfTokenFromRequest` coverage (66.7→100%)  | MED    | 10min  | Coverage     |
| 9  | Add `Hijack` coverage (60→100%)                  | MED    | 10min  | Coverage     |
| 10 | Add `handleLogout` coverage (64.3→85%+)          | MED    | 15min  | Coverage     |
| 11 | Extract `"Bearer "` constant in usermgmt         | LOW    | 2min   | Cleanup      |
| 12 | Extract `"session_token"` constant in usermgmt   | LOW    | 2min   | Cleanup      |
| 13 | Extract password validation messages             | LOW    | 3min   | Cleanup      |
| 14 | Extract `"usermgmt:"` log prefix constant        | LOW    | 1min   | Cleanup      |
| 15 | Extract `maxDisplayNameLength` constant          | LOW    | 1min   | Cleanup      |
| 16 | Add `handleAuthEndpoint` timeout branch coverage | MED    | 15min  | Coverage     |
| 17 | Add `Apply` error path coverage                  | MED    | 15min  | Coverage     |
| 18 | Add `MapError` family branch coverage            | MED    | 10min  | Coverage     |
| 19 | Add `Enforce` nil enforcer test                  | MED    | 5min   | Coverage     |
| 20 | Add `generateToken` error path test              | LOW    | 5min   | Coverage     |
| 21 | `writeJSON` dedup (usermgmt → root's WriteJSON)  | MED    | 30min  | Architecture |
| 22 | Integration test go.mod tidy                     | LOW    | 1min   | Fix          |
| 23 | TypedHandler[T] API design proposal              | HIGH   | 60min  | Design       |
| 24 | UserID type split ADR update                     | MED    | 30min  | Design       |
| 25 | Dependabot alerts investigation                  | MED    | 5min   | Security     |

---

## G) Top Question for Owner 🎯

**The usermgmt `writeJSON` is an exact duplicate of root's exported `WriteJSON`.** Should usermgmt import and use `cqrshtmx.WriteJSON`? This would:

- Eliminate the duplicate
- Create a **circular dependency risk** (usermgmt → cqrs-htmx → usermgmt)
- Require usermgmt to depend on the root module

Currently usermgmt is independent (no root import). **Should usermgmt stay independent or should we couple them?** The tradeoff is: deduplication vs. module independence.
