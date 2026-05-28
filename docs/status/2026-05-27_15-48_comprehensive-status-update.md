# Comprehensive Status Report — 2026-05-27_15-48

**Date:** 2026-05-27_15-48 | **Branch:** master | **Commit:** 52b8d81
**Coverage:** 96.5% root, 90.5% usermgmt | **Lint:** 0 issues | **Tests:** All pass (race-safe)

---

## a) FULLY DONE

### Today's Bug Fix Session (11 commits)

| #   | Commit    | Fix                                                                              | Impact                                            |
| --- | --------- | -------------------------------------------------------------------------------- | ------------------------------------------------- |
| 1   | `00b7187` | GetUser/UpdateRoles: pass domain errors through instead of wrapping as Transient | **Bug: 500 → 404 for missing users**              |
| 2   | `994e9e0` | integration_test: go mod tidy                                                    | Build fix                                         |
| 3   | `2d872bc` | Rate limiter: fresh heap entry on TTL re-check, prevent hot-key eviction         | **Bug: active keys evicted prematurely**          |
| 4   | `4f653b4` | CSRFTokenHXHeaders: json.Marshal instead of string concat                        | **Bug: malformed JSON on tokens with `"` or `\`** |
| 5   | `fe9f8a1` | Store: User.Clone() defensive copies from FindByID/FindByEmail                   | **Correctness: store isolation**                  |
| 6   | `14b4478` | SessionMiddleware: log auth failures at Debug level                              | **Observability**                                 |
| 7   | `485dfd9` | Register: log rollback errors instead of `_ =`                                   | **Correctness: visibility into partial failures** |
| 8   | `2262d8e` | Extract validatePassword shared function (DRY)                                   | Code quality                                      |
| 9   | `f34b81e` | Authz.Apply: add-before-remove ordering                                          | **Security: prevents permission gaps**            |
| 10  | `ac80978` | Rate limiter heap consistency fix + CSRF JSON fix (combined)                     | **Correctness**                                   |
| 11  | `52b8d81` | WriteJSON: buffer before WriteHeader                                             | **Correctness: no committed empty 200s**          |

### Previous Sessions (also done)

- Documentation overhaul (README, SECURITY, CONTRIBUTING, CHANGELOG, AGENTS, TODO_LIST, FEATURES, ROADMAP)
- gorilla/csrf → justinas/nosurf migration
- cockroachdb/errors → go-error-family migration
- All 168 original TODO items from the project's history

---

## b) PARTIALLY DONE

Nothing partially done — all committed work is complete and verified.

---

## c) NOT STARTED

### Lower-priority improvements from brutal self-review:

| #   | Item                                                                   | Impact | Effort | Risk             |
| --- | ---------------------------------------------------------------------- | ------ | ------ | ---------------- |
| 1   | Replace hand-rolled `Chain` with `justinas/alice`                      | Low    | 30min  | Adds dep         |
| 2   | Replace `decodeFormValues` JSON round-trip with `gorilla/schema`       | Medium | 1h     | Breaking         |
| 3   | Deduplicate `handleCommandDispatch`/`handleQueryDispatch` into generic | Medium | 2h     | Complexity       |
| 4   | Deduplicate `Command()`/`Query()` on App                               | Medium | 1h     | Complexity       |
| 5   | Replace `StatusRecorder` with `httpsnoop`                              | Low    | 30min  | Adds dep         |
| 6   | Add clock abstraction for testable time logic                          | Medium | 2h     | Test-only        |
| 7   | Group `handlerConfig` 14-field god struct into sub-structs             | Low    | 1h     | Refactor         |
| 8   | Reduce `any` usage in public API                                       | Low    | 2h     | Breaking         |
| 9   | Hash session tokens before storage                                     | Medium | 1h     | Breaking         |
| 10  | Rate limit on registration endpoint                                    | Medium | 1h     | New feature      |
| 11  | Fix `HandlerConfig.Secure` zero-value trap (use `*bool`)               | Medium | 30min  | Breaking         |
| 12  | Fix `Response.JSON` to propagate marshal errors                        | Low    | 30min  | API change       |
| 13  | Add structured logging for dispatch failures                           | Medium | 1h     | New feature      |
| 14  | Return value types from store (or read-only interface)                 | Medium | 30min  | Done via Clone   |
| 15  | Add `List`/`FindAll` to `UserStore` interface                          | Low    | 30min  | Interface change |
| 16  | Fix `CSRFConfig.Validate()` — either enforce or remove it              | Low    | 15min  | Dead code        |
| 17  | Log correlation ID parse failures in middleware                        | Low    | 15min  | Observability    |
| 18  | Upgrade usermgmt to go-cqrs-lite v1.5.1                                | Low    | 30min  | Dep alignment    |
| 19  | SQL store backend for usermgmt                                         | Medium | 4h     | New module       |
| 20  | OpenTelemetry integration                                              | Medium | 4h     | New feature      |

---

## d) TOTALLY FUCKED UP

Nothing. The codebase is in good shape:

- All 4 modules build and pass tests with race detector
- 0 lint issues across all modules
- Coverage 96.5% root, 90.5% usermgmt (slight decrease from new defensive code branches)
- All bugs found in brutal self-review have been fixed or documented as deferred

---

## e) WHAT WE SHOULD IMPROVE

### Code Architecture

1. **`handleCommandDispatch`/`handleQueryDispatch` duplication** — ~70 lines duplicated. Maintenance hazard: fix a bug in one, forget the other. But unifying adds generics complexity.
2. **`handlerConfig` god struct** — 14 fields mixing auth, decode, response, CSRF, rate limiting. Should be grouped.
3. **`any` overuse** — `Enforce(rvals ...any)`, `RenderFunc(result any)`, `TriggerWithDetail(detail any)` defeat compile-time safety.
4. **No clock abstraction** — `time.Now()` in ~15 places makes time-dependent logic untestable without hacks.

### Type Safety

5. **`HandlerConfig.Secure` zero-value trap** — `HandlerConfig{}` sets `Secure=false`. Should use `*bool` or default to true.
6. **Store returns pointers** — Fixed via Clone(), but the interface still returns `*User` which is a footgun for future implementations.
7. **Session tokens stored plaintext** — Memory dump = valid tokens. Should hash like passwords.

### Library Usage

8. **Hand-rolled `Chain`** — `justinas/alice` does the same thing, battle-tested.
9. **`decodeFormValues` JSON round-trip** — `gorilla/schema` is purpose-built for form→struct decoding.
10. **`StatusRecorder`** — `httpsnoop` or `chi/middleware.WrapResponseWriter` cover all `http.ResponseWriter` interfaces.

---

## f) Top 25 Things We Should Get Done Next

Sorted by impact × effort (highest first):

| #   | Task                                                                                 | Impact | Effort | Category      |
| --- | ------------------------------------------------------------------------------------ | ------ | ------ | ------------- |
| 1   | Fix `HandlerConfig.Secure` zero-value trap                                           | High   | 30min  | Security      |
| 2   | Rate limit on registration endpoint                                                  | Medium | 1h     | Security      |
| 3   | Add structured dispatch failure logging                                              | Medium | 1h     | Observability |
| 4   | Log correlation ID parse failures in middleware                                      | Low    | 15min  | Observability |
| 5   | Fix `CSRFConfig.Validate()` — enforce or remove                                      | Low    | 15min  | Dead code     |
| 6   | Fix `Response.JSON` to propagate marshal errors                                      | Low    | 30min  | Correctness   |
| 7   | Replace `Chain` with `justinas/alice`                                                | Low    | 30min  | Quality       |
| 8   | Replace `StatusRecorder` with `httpsnoop`                                            | Low    | 30min  | Quality       |
| 9   | Upgrade usermgmt to go-cqrs-lite v1.5.1                                              | Low    | 30min  | Deps          |
| 10  | Hash session tokens before storage                                                   | Medium | 1h     | Security      |
| 11  | Replace `decodeFormValues` with `gorilla/schema`                                     | Medium | 1h     | Perf/Quality  |
| 12  | Add clock abstraction for testable time logic                                        | Medium | 2h     | Testability   |
| 13  | Deduplicate `Command()`/`Query()` on App                                             | Medium | 1h     | Architecture  |
| 14  | Deduplicate `handleCommandDispatch`/`handleQueryDispatch`                            | Medium | 2h     | Architecture  |
| 15  | Group `handlerConfig` into sub-structs                                               | Low    | 1h     | Architecture  |
| 16  | Reduce `any` usage in public API                                                     | Low    | 2h     | Type safety   |
| 17  | Add `List`/`FindAll` to `UserStore` interface                                        | Low    | 30min  | Completeness  |
| 18  | OpenTelemetry integration                                                            | Medium | 4h     | Observability |
| 19  | SQL store backend for usermgmt                                                       | Medium | 4h     | Storage       |
| 20  | Prometheus metrics middleware                                                        | Medium | 2h     | Observability |
| 21  | JWT/OIDC integration helpers                                                         | Medium | 3h     | Auth          |
| 22  | Redis session store                                                                  | Medium | 3h     | Storage       |
| 23  | `App.RecoveryMiddleware()` naming collision (two functions named RecoveryMiddleware) | Low    | 15min  | API clarity   |
| 24  | `User.Roles` vs Casbin roles dual-write problem                                      | Medium | 2h     | Correctness   |
| 25  | Password complexity requirements beyond length                                       | Low    | 30min  | Security      |

---

## g) Top #1 Question I Cannot Figure Out Myself

**Should we deduplicate `handleCommandDispatch`/`handleQueryDispatch`?**

These two functions are ~70 lines of structural twins. Both: (1) build dispatch context, (2) nil-check decoder, (3) decode request, (4) apply timeout, (5) dispatch via command vs query dispatcher, (6) apply response. The only real differences are the dispatch method called and the error sentinels used.

A generic `dispatch[Req any]` would eliminate duplication but:

- Makes the hot path harder to read for contributors
- Generics + `command.Command`/`query.Query` interfaces add cognitive load
- A bug fix in the generic could break both paths simultaneously

**My recommendation**: Do it. The maintenance hazard of "fix one, forget the other" outweighs the readability cost. Use clear naming and comments. But I want your explicit go/no-go before committing to this scope.
