# go-cqrs-lite v2.0.0 Migration — Status Report

**Date:** 2026-06-02 13:52
**Session:** Migrate all 4 modules from go-cqrs-lite v1.7.1 → v2.0.0

---

## a) FULLY DONE

### Migration Core (all verified with tests + lint + nix check)

| Item                                                                                            | Status               |
| ----------------------------------------------------------------------------------------------- | -------------------- |
| Import paths updated (`go-cqrs-lite/{pkg}` → `go-cqrs-lite/{pkg}/v2`)                           | ✅ 28 Go files       |
| Dead catalog code removed (`CommandCatalogEntries`, `QueryCatalogEntries`, `dispatcher` import) | ✅ `app.go`          |
| CatalogEntries tests removed from `coverage_test.go`                                            | ✅ 4 test cases      |
| Root `go.mod` updated + replace directives                                                      | ✅                   |
| `usermgmt/go.mod` updated + replace directives                                                  | ✅                   |
| `integration_test/go.mod` updated + replace directives                                          | ✅                   |
| `datastar-demo/go.mod` updated + replace directives                                             | ✅                   |
| `go-error-family` v0.2.0 → v0.3.0                                                               | ✅                   |
| `go mod tidy` all 4 modules                                                                     | ✅                   |
| `go work sync`                                                                                  | ✅                   |
| `go build ./...` all 4 modules                                                                  | ✅                   |
| `go test ./... -race` root (390+ tests)                                                         | ✅ PASS              |
| `go test ./... -race` usermgmt                                                                  | ✅ PASS              |
| `go test ./... -race` integration_test                                                          | ✅ PASS              |
| `nix run .#lint`                                                                                | ✅ 0 issues          |
| `nix flake check`                                                                               | ✅ all checks passed |
| `AGENTS.md` updated with v2.0.0 info                                                            | ✅                   |

---

## b) PARTIALLY DONE

Nothing partially done — the migration itself is complete.

---

## c) NOT STARTED (post-migration improvements)

### Documentation cleanup needed

| Item                                                | Priority | Notes                                                                 |
| --------------------------------------------------- | -------- | --------------------------------------------------------------------- |
| Update `FEATURES.md` #34 Catalog Entries → REMOVED  | P1       | Still says FULLY_FUNCTIONAL                                           |
| Update `TODO_LIST.md` version references            | P2       | References v1.6.0                                                     |
| Update `ROADMAP.md` version references              | P2       | References v1.6.0                                                     |
| Update `docs/modularization/DEPENDENCY_GRAPH.md`    | P3       | References v0.1.1                                                     |
| Update `docs/modularization/PROPOSAL.md`            | P3       | References v0.1.1                                                     |
| Update `docs/modularization/EXECUTION_PLAN.md`      | P3       | References v0.1.1                                                     |
| Update `docs/modularization/RESEARCH-2026-05-27.md` | P3       | References v1.6.0                                                     |
| Root `go.sum` added to git tracking                 | P1       | New untracked file from go mod tidy                                   |
| Unused replace directives in go.mod                 | P2       | `snapshot/v2`, `memory/v2`, `schema/v2` in replace but not in require |

### go-cqrs-lite v2 feature adoption

| Item                                                         | Priority | Impact | Work   | Notes                                              |
| ------------------------------------------------------------ | -------- | ------ | ------ | -------------------------------------------------- |
| Use `command.RegisterTyped` for typed command handlers       | P2       | High   | Medium | Eliminates manual type assertions in consumer code |
| Use `query.RegisterTyped`/`DispatchTyped`                    | P2       | High   | Medium | Type-safe query dispatch, eliminates `any` casting |
| Use `query.PaginatedResult[T]`                               | P2       | Medium | Low    | Built-in pagination support for query handlers     |
| Expose `event.EventBus` reactive streams                     | P3       | High   | High   | Real-time projections, notifications via SSE       |
| Adopt generic middleware (`NewRecovery`, `NewLogging`, etc.) | P3       | Medium | High   | Already have our own; only adopt if better         |
| Expose `Lifecycle` close support                             | P3       | Low    | Low    | Already handled via dispatcher.Close()             |
| `CommandStore` integration                                   | P3       | Medium | High   | Command journaling for audit trail                 |

---

## d) TOTALLY FUCKED UP

Nothing is broken. All tests pass, lint is clean, nix checks pass.

### Near misses caught:

1. **Replace directive paths**: Initially used `../../go-cqrs-lite/` (wrong depth). Fixed to `../go-cqrs-lite/`
2. **Transitive dependencies**: First `go mod tidy` failed because `memory/v2` and `schema/v2` needed replace directives too (they're test dependencies of `event/v2`)
3. **LSP cache**: LSP shows ~27 stale errors about missing imports. CLI `go build`/`go test` work fine. This is a known pre-existing LSP issue (AGENTS.md gotcha #5)

---

## e) WHAT WE SHOULD IMPROVE

### Code quality

1. **Remove unused replace directives** — `snapshot/v2`, `memory/v2`, `schema/v2` are in root `go.mod` replace block but not in require. Gopls warns about these.
2. **`go.work.sum` has stale entries** — `go work sync` should clean this but some may be from old v1 modules
3. **FEATURES.md out of sync** — CatalogEntries feature still listed as FULLY_FUNCTIONAL but was removed
4. **Historical docs stale** — 10+ status/planning docs reference old versions. These are historical records and don't need updating, but `ROADMAP.md` and `TODO_LIST.md` should be current.

### Architecture improvements

5. **Leverage v2 generics**: `RegisterTyped`/`DispatchTyped` eliminate boilerplate. Our test helpers could use these.
6. **Leverage `PaginatedResult[T]`**: Our query handlers return `any` — could return typed paginated results.
7. **Reactive event streams**: `EventBus` + `FilterEventType` + `ScanState` would enable real-time SSE/HTMX out-of-band updates without WebSocket complexity.
8. **Generic middleware from go-cqrs-lite**: Already have `NewRecovery`/`NewLogging` upstream. Our `RecoveryMiddleware` duplicates this — could delegate.
9. **Error taxonomy alignment**: v2 has a refined 5-family taxonomy. Our `MapError` already maps these correctly but could be more explicit.

### Type model improvements

10. **`UserIDExtractor` returns `UserID` (alias) not `string`**: This is already good — type-safe extraction.
11. **Could expose `RegisterTyped` wrapper**: A cqrs-htmx wrapper around `command.RegisterTyped` that auto-injects context (user ID, correlation ID, request ID) from HTTP request.
12. **Query response typing**: `HandlerOption` render functions get `any` result — could offer typed variant.

---

## f) TOP 25 THINGS TO DO NEXT (sorted by impact/effort)

### P1 — Quick wins (< 15 min each)

| #   | Item                                              | Impact   | Effort |
| --- | ------------------------------------------------- | -------- | ------ |
| 1   | Commit the v2 migration                           | Critical | 2 min  |
| 2   | Update FEATURES.md #34 → REMOVED                  | Low      | 2 min  |
| 3   | Add root `go.sum` to git                          | Low      | 1 min  |
| 4   | Remove unused replace directives from root go.mod | Low      | 2 min  |
| 5   | Update TODO_LIST.md version refs → v2.0.0         | Low      | 5 min  |
| 6   | Update ROADMAP.md version refs → v2.0.0           | Low      | 5 min  |
| 7   | Clean up go.work.sum stale entries                | Low      | 2 min  |

### P2 — Medium effort (30 min - 2 hours)

| #   | Item                                                             | Impact | Effort |
| --- | ---------------------------------------------------------------- | ------ | ------ |
| 8   | Add `CommandTyped`/`QueryTyped` HandlerOption for typed dispatch | High   | 1 hr   |
| 9   | Expose `PaginatedResult[T]` support in query handlers            | Medium | 30 min |
| 10  | Update all docs/modularization/\*.md version refs                | Low    | 15 min |
| 11  | Run `nix run .#coverage` to verify coverage maintained           | Medium | 5 min  |
| 12  | Add integration test for v2 dispatcher.Close() behavior          | Medium | 30 min |
| 13  | Verify datastar-demo still works end-to-end                      | Medium | 15 min |
| 14  | Clean LSP cache and verify zero LSP errors                       | Low    | 10 min |

### P3 — Larger efforts (2+ hours)

| #   | Item                                                   | Impact    | Effort |
| --- | ------------------------------------------------------ | --------- | ------ |
| 15  | Adopt reactive EventBus for real-time notifications    | Very High | 4 hr   |
| 16  | SSE out-of-band updates via EventBus + HTMX            | Very High | 4 hr   |
| 17  | OpenTelemetry integration using upstream middleware    | High      | 3 hr   |
| 18  | Evaluate generic middleware adoption vs our own        | High      | 2 hr   |
| 19  | SQL store backend for usermgmt                         | High      | 8 hr   |
| 20  | Command audit trail via CommandStore                   | Medium    | 3 hr   |
| 21  | Circuit breaker middleware for external deps           | Medium    | 2 hr   |
| 22  | Usermgmt domain events → EventBus reactive projections | High      | 4 hr   |
| 23  | JWT/OIDC auth integration                              | High      | 6 hr   |
| 24  | Redis session store for usermgmt                       | Medium    | 4 hr   |
| 25  | BadgerDB event store adapter                           | Medium    | 6 hr   |

---

## g) TOP #1 QUESTION I CANNOT FIGURE OUT MYSELF

**Should the `replace` directives be kept or removed before go-cqrs-lite v2.0.0 tags are published?**

- Keeping them: Enables local development and testing against the latest go-cqrs-lite source. But makes the module not portable — other consumers can't `go get` this module without having go-cqrs-lite locally.
- Removing them: Requires go-cqrs-lite v2.0.0 to be tagged and pushed. Since v2.0.0 tags don't exist yet, `go mod tidy` would fail without them.

**Recommendation:** Keep replace directives for now. Create a follow-up task to remove them once v2.0.0 is tagged upstream. Document this in AGENTS.md gotchas.

---

## Migration Breaking Changes Summary

| Upstream Change                        | cqrs-htmx Impact                                   | Resolution                 |
| -------------------------------------- | -------------------------------------------------- | -------------------------- |
| Module paths: `{pkg}` → `{pkg}/v2`     | All imports                                        | ✅ Updated 28 files        |
| `dispatcher.HandlerMeta` removed       | `CommandCatalogEntries()`, `QueryCatalogEntries()` | ✅ Removed methods + tests |
| `dispatcher.CatalogDispatcher` removed | Same                                               | ✅ Removed                 |
| `event.Context()` removed              | Not used                                           | N/A                        |
| `event.Snapshot*` moved to `snapshot/` | Not used                                           | N/A                        |
| `go-error-family` v0.3.0 required      | All modules                                        | ✅ Updated                 |

### API surface fully compatible (no behavioral changes)

All types, functions, and interfaces used by cqrs-htmx exist in v2 with identical signatures:

- `command.Type`, `command.Command`, `command.BasicCommand`, `command.Dispatcher`
- `query.Type`, `query.Query`, `query.BasicQuery`, `query.Dispatcher`
- `event.Option`, `event.WithUserID`, `event.WithCorrelationID`, `event.WithRequestID`
- `event.NewRejection`, `event.NewInfrastructure`, `event.Classify`, `event.Family`
- `id.UserID`, `id.CorrelationID`, `id.RequestID`, `id.New*`, `id.Parse*`, `id.MustParse*`
- `dispatcher.Lifecycle` (used transitively)

### New v2 features available for future adoption

- `command.RegisterTyped[T]` — type-safe command handlers
- `query.RegisterTyped[T]` / `query.DispatchTyped[T]` — type-safe query dispatch
- `query.Pagination` / `query.PaginatedResult[T]` — built-in pagination
- `event.EventBus` / reactive streams — real-time event processing
- Generic middleware suite (recovery, logging, retry, validation, metrics, circuit breaker, OTel)
- `decider.Decider[State]` — pure-function aggregate pattern
