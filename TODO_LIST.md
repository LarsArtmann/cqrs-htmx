# TODO List — cqrs-htmx

**Updated:** 2026-05-27 | **Coverage:** 96.9% root, 91.1% usermgmt | **Lint:** 0 issues

## Status Legend

- [ ] OPEN
- [x] DONE
- [~] PARTIALLY DONE
- [-] NOT APPLICABLE / BLOCKED

---

## Open Items

### Upstream-Blocked

- [ ] **BrandNamer for root module marker types** — BLOCKED: upstream `go-cqrs-lite/core/pkg/id` marker types (`userMarker`, `correlationMarker`) are unexported. Requires upstream change to expose them or provide BrandNamer integration.

### Future Enhancements (Not Started)

- [ ] **Upgrade usermgmt to go-cqrs-lite/core v1.5.1** — Root module is on v1.5.1 pre-release; usermgmt still on v1.5.0. Align versions when v1.5.1 is formally released.
- [ ] **SQL store backend for usermgmt** — Pattern documented in ADR 0003 (numeric IDs via `brandid.ID[Brand, int64]`). Not yet implemented.
- [ ] **OpenTelemetry integration** — Lifecycle hooks (`BeforeDispatchHook`/`AfterDispatchHook`) enable tracing. No official OTel middleware yet.
- [ ] **WebSocket/SSE helpers** — NOT_PLANNED per FEATURES.md. Would require significant API surface.

---

## Completed (2026-05-07 → 2026-05-27)

_168 items completed. See [CHANGELOG.md](CHANGELOG.md) and [git log](https://github.com/larsartmann/cqrs-htmx/commits/master) for full history._

### Highlights by Session

| Session     | Key Accomplishments                                                          |
| ----------- | ---------------------------------------------------------------------------- |
| 2026-05-07  | Initial lint zero (103→0), test coverage 93.5%                               |
| 2026-05-16  | v1.0.0 release: lifecycle hooks, validation, timeout, benchmarks             |
| 2026-05-19  | CSRF protection (gorilla/csrf), error context, deduplication                 |
| 2026-05-20  | Branded UserID migration, SessionMaxAge fix, usermgmt 85%→95.6%              |
| 2026-05-21  | CatalogEntries exposure, CI fix, lint elimination, error wrapping            |
| 2026-05-22  | Integration tests, O(log n) eviction, HTTP timeout, fuzz tests               |
| 2026-05-23  | Mock stores, coverage 88.6%→91%, go-cqrs-lite v1.5.0 upgrade                 |
| 2026-05-24  | Perf optimizations (7 alloc reductions), security hardening                  |
| 2026-05-25+ | gorilla/csrf→nosurf, cockroachdb/errors→go-error-family, httputil delegation |
| 2026-05-27  | RecoveryMiddleware, RenderJSON, request ID in errors, benchmarks             |
