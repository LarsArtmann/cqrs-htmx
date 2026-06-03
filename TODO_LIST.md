# TODO List — cqrs-htmx

**Updated:** 2026-06-02 | **Coverage:** 96.9% root, 91.1% usermgmt | **Lint:** 0 issues

## Status Legend

- [ ] OPEN
- [x] DONE
- [~] PARTIALLY DONE
- [-] NOT APPLICABLE / BLOCKED

---

## Open Items

### Security & Correctness (Pre-v1.1.0)

- [x] **Fix rate limiter unbounded heap growth** — Fixed: limiterEntry now stores heapRef back-pointer. Refresh uses heap.Fix for in-place updates instead of pushing duplicate entries. No more ghost entries.
- [ ] **Fix CSRF proxy bypass** — `r.TLS == nil` check trusts all HTTP proxies. Add `TrustedProxies []string` config and IP-based trust check. (Large effort, needs design)
- [x] **Fix Response.Status() fluent chain** — Fixed: Status() stores code in Response.statusCode. Apply() writes it at the end. Fluent chains like Status(201).Redirect("/x").Apply() now work.
- [x] **Add tests for nil-enforcer + query nil check** — Tests already existed in coverage_test.go (verified). Added ErrEnforcerNotInitialized sentinel to all Authz methods for defensive nil-checking.
- [x] **Add Login error classification tests** — TestService_Login_StoreError added. Verifies store errors return transient, not ErrInvalidCredentials.
- [x] **Add UpdateRoles rollback tests** — TestService_UpdateRoles_AuthzFailurePreservesUser added. Verifies user roles remain unchanged when Casbin Apply fails. Also fixed UpdateRoles ordering (Casbin before user save).

### Upstream-Blocked

- [ ] **BrandNamer for root module marker types** — BLOCKED: upstream `go-cqrs-lite/core/pkg/id` marker types (`userMarker`, `correlationMarker`) are unexported. Requires upstream change to expose them or provide BrandNamer integration.
- [x] **Remove local replace directives** — go-cqrs-lite v2.0.0 tags are published upstream. All go-cqrs-lite replace directives removed from all 4 go.mod files. Only `integration_test` retains cqrs-htmx local replaces (library not yet published).

### Future Enhancements (Not Started)

- [x] **Upgrade to go-cqrs-lite v2.0.0** — All 4 modules migrated to v2 import paths (`/v2` suffix). CatalogEntries removed (dead upstream code). go-error-family v0.3.0. Replace directives removed (v2.0.0 tags published).
- [ ] **SQL store backend for usermgmt** — Pattern documented in ADR 0003 (numeric IDs via `brandid.ID[Brand, int64]`). Not yet implemented.
- [ ] **OpenTelemetry integration** — Lifecycle hooks (`BeforeDispatchHook`/`AfterDispatchHook`) enable tracing. Upstream v2 has generic OTel middleware in `middleware/` module.
- [ ] **Adopt v2 typed dispatch** — `command.RegisterTyped[T]`/`query.RegisterTyped[T]`/`query.DispatchTyped[T]` eliminate manual type assertions.
- [ ] **Adopt PaginatedResult[T]** — `query.PaginatedResult[T]` provides built-in pagination for query handlers.
- [ ] **Reactive event streams** — `event.EventBus` + `FilterEventType` + `ScanState` for real-time SSE/HTMX out-of-band updates.

---

## Completed (2026-05-07 → 2026-05-27)

_168 items completed. See [CHANGELOG.md](CHANGELOG.md) and [git log](https://github.com/larsartmann/cqrs-htmx/commits/master) for full history._

### Highlights by Session

| Session     | Key Accomplishments                                                                                                                                                                                           |
| ----------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 2026-05-07  | Initial lint zero (103→0), test coverage 93.5%                                                                                                                                                                |
| 2026-05-16  | v1.0.0 release: lifecycle hooks, validation, timeout, benchmarks                                                                                                                                              |
| 2026-05-19  | CSRF protection (gorilla/csrf), error context, deduplication                                                                                                                                                  |
| 2026-05-20  | Branded UserID migration, SessionMaxAge fix, usermgmt 85%→95.6%                                                                                                                                               |
| 2026-05-21  | CatalogEntries exposure, CI fix, lint elimination, error wrapping                                                                                                                                             |
| 2026-05-22  | Integration tests, O(log n) eviction, HTTP timeout, fuzz tests                                                                                                                                                |
| 2026-05-23  | Mock stores, coverage 88.6%→91%, go-cqrs-lite v1.5.0 upgrade                                                                                                                                                  |
| 2026-05-24  | Perf optimizations (7 alloc reductions), security hardening                                                                                                                                                   |
| 2026-05-25+ | gorilla/csrf→nosurf, cockroachdb/errors→go-error-family, httputil delegation                                                                                                                                  |
| 2026-05-27  | RecoveryMiddleware, RenderJSON, request ID in errors, benchmarks                                                                                                                                              |
| 2026-05-27b | 10 bug fixes: GetUser 404, rate limiter TTL, CSRF JSON, store copies, authz ordering, WriteJSON buffer, password DRY, rollback logging, SessionMiddleware logging                                             |
| 2026-05-27c | HandlerConfig.Secure \*bool, CSRFConfig.Validate(), Response.JSON 500, correlation ID logging, RecoverHandler rename, go-cqrs-lite v1.6.0, dispatch logging, usermgmt writeJSON buffer, tests                 |
| 2026-05-28  | Domain model enrichment: SetRoles, ChangePassword, SetEmail, SetDisplayName, IsPasswordSet, touch(). Domain events: 4 event types with optional EventHandler. Fuzz + benchmarks. CRUD eliminated.             |
| 2026-06-02  | v2.0.0 migration (42 files). Pre-release fixes: nil-enforcer bypass, query nil panic, Login error classification, UpdateRoles ordering, store clone, query param logging removed, defaultLoginRedirect const. |
