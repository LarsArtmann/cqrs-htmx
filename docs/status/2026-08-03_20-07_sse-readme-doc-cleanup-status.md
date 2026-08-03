# SSE Re-export Deprecation: README & Doc Comment Cleanup

**Date:** 2026-08-03 20:07
**Session scope:** Post-mortem cleanup of README.md + doc comments + test Describe blocks after the SSE re-export deprecation.

---

## Executive Summary

The biggest miss from the prior session (README.md) is **fixed** — 15+ deprecated symbol references in README.md are now migrated to `sse.*`. Build, tests, and lint pass (0 issues, 0 SA1019). However, this session's sweep revealed **additional gaps in production code doc comments, test Describe block names, and recipe docs** that were NOT caught.

---

## a) FULLY DONE

| What                                                 | Status   | Verification                                                             |
| ---------------------------------------------------- | -------- | ------------------------------------------------------------------------ |
| README.md feature-list bullet (line 37)              | Done     | `sse.Stream`/`sse.Event`                                                 |
| README.md SSE code example (lines 420-429)           | Done     | `sse.NewStream`, `sse.Event`, `sse.Replay`, `sse.LastEventIDFromRequest` |
| README.md SSE API table (lines 460-485)              | Done     | Split into go-sse / cqrs-htmx sections                                   |
| README.md htmx extension table (line 576)            | Done     | `sse.Stream`                                                             |
| README.md file-structure comments (lines 1170-1172)  | Done     | Removed nonexistent `sse_stream.go`, added `event_store_sse.go`          |
| `examples/basic/README.md` line 55                   | Done     | `sse.Event`                                                              |
| `doc.go:135`                                         | Done     | `SSEEventStore` → `sse.EventStore`                                       |
| `sse_event_test.go` Describe blocks (3 inner blocks) | Done     | `sse.WriteEvent`, `sse.Stream`, `sse.EventID`                            |
| Root module build                                    | Passes   | `go build ./...`                                                         |
| Root module tests                                    | Passes   | `go test ./... -count=1 -race`                                           |
| Root module lint                                     | 0 issues | `golangci-lint run ./...`                                                |
| Deprecated reference sweep on README/examples/doc.go | Clean    | `rg` returns 0 hits                                                      |

---

## b) PARTIALLY DONE

### `sse_event_test.go` — Inner Describe blocks renamed, BUT two DescribeTable entry names missed

- **Line 304**: `"ParseSSEEventID rejects IDs with newlines"` — still references deprecated `ParseSSEEventID` (now calls `sse.ParseEventID`)
- **Line 316**: `"NewSSEEventID constructs without validation"` — still references deprecated `NewSSEEventID` (now calls `sse.NewEventID`)

### `feedback_features_test.go` — SSE constants block tests go-sse, not cqrs-htmx

- Line 34-37: `"SSE event name constants"` Describe block now asserts `sse.EventConnected` / `sse.EventHeartbeat` — these are go-sse constants, not cqrs-htmx re-exports.
- **Philosophical question:** Should this block exist at all? It's testing that go-sse's constants equal specific strings. That's testing a dependency's behavior, not cqrs-htmx's.

---

## c) NOT STARTED

### Production code doc comments (5 files, 9 references)

These are `[godoc links]` and plain-text references to deprecated symbols in production code comments. They resolve to the deprecated aliases and will produce stale documentation:

| File                  | Line | Current                   | Should Be                  |
| --------------------- | ---- | ------------------------- | -------------------------- |
| `event_store_sse.go`  | 22   | `[SSEEventStore]`         | `[sse.EventStore]`         |
| `event_store_sse.go`  | 24   | `SSEEventStore interface` | `sse.EventStore interface` |
| `event_store_sse.go`  | 56   | `[SSEEventStore]`         | `[sse.EventStore]`         |
| `event_store_sse.go`  | 101  | `[SSEEventStore]`         | `[sse.EventStore]`         |
| `htmx_extensions.go`  | 29   | `SSEStream`               | `sse.Stream`               |
| `ack.go`              | 65   | `SSEEvent.Data`           | `sse.Event.Data`           |
| `structured_error.go` | 141  | `SSEEvent.Data`           | `sse.Event.Data`           |
| `ws_broadcaster.go`   | 13   | `[SSEStream]`             | `[sse.Stream]`             |

### Test comment/Describe references (5 files)

| File                                | Line | Current                       | Should Be                      |
| ----------------------------------- | ---- | ----------------------------- | ------------------------------ |
| `sse_event_test.go`                 | 304  | `ParseSSEEventID rejects...`  | `sse.ParseEventID rejects...`  |
| `sse_event_test.go`                 | 316  | `NewSSEEventID constructs...` | `sse.NewEventID constructs...` |
| `sse_reconnect_test.go`             | 25   | `LastEventID on SSEStream`    | `LastEventID on sse.Stream`    |
| `sse_broadcaster_test.go`           | 260  | `Broadcaster + SSEStream`     | `Broadcaster + sse.Stream`     |
| `bdd_realtime_test.go`              | 21   | `NewSSEStream's`              | `sse.NewStream's`              |
| `sse_reconnect_integration_test.go` | 17   | `SSEEventStore`               | `sse.EventStore`               |

### Recipe docs (living documentation, not historical)

- **`docs/recipes/offline-command-sync.md`** line 10, 78 — references `SSEStream`, `cqrshtmx.NewSSEStream`. This is a how-to guide consumers follow; it must use the non-deprecated API.

### Verification gaps

- [ ] Full workspace lint sweep not run on: `examples/basic`, `examples/admin-demo`, `e2e/server`
- [ ] `integration_test` module tests not run
- [ ] Coverage gate (`nix run .#coverage-gate`) not run
- [ ] `dashboardui` module lint not verified after migration

---

## d) TOTALLY FUCKED UP

Nothing in this session was broken — all changes compile, pass tests, and produce 0 lint issues.

**However**, looking back at the prior session's execution, the biggest miss was architectural:

- **The prior session migrated the "last consumer" code and declared victory, but never swept doc comments or test Describe block strings.** A `rg '\bSSEEvent\b|\bSSEStream\b'` would have caught all 15 remaining references in seconds. The sed-based bulk migration only caught code symbols, not text inside comments and Describe strings. This is a systemic blind spot of sed-based refactoring: it migrates code but not prose.

---

## e) WHAT WE SHOULD IMPROVE

1. **Always run a prose sweep after a code migration.** After sed-replacing symbols in code, immediately `rg '\bOldSymbolName\b'` across the entire repo — comments, doc strings, and test Describe blocks are always missed by code-only migrations.

2. **Doc links are first-class API surface.** `[SSEEventStore]` in a Go doc comment renders as a clickable link in pkg.go.dev. When the symbol is deprecated, the link points to a deprecated alias. These MUST be updated as part of the deprecation, not as a follow-up.

3. **Recipe docs are living documentation.** `docs/recipes/` is actively linked from README and guides. Code examples in recipes must track current API. A deprecation that leaves recipe examples stale is incomplete.

4. **Test Describe block names are part of the test contract.** When a test for `WriteSSEEvent` is renamed to test `sse.WriteEvent`, the Describe string must match — otherwise test output is misleading.

5. **The sweep should have been done BEFORE declaring "all 8 todos complete" in the prior session.** The post-mortem was written as an afterthought instead of as a pre-completion checklist.

---

## f) Up to 50 Things to Get Done Next

### High priority — doc comment correctness (5 min)

1. [ ] Fix `event_store_sse.go:22` — `[SSEEventStore]` → `[sse.EventStore]`
2. [ ] Fix `event_store_sse.go:24` — `SSEEventStore` → `sse.EventStore`
3. [ ] Fix `event_store_sse.go:56` — `[SSEEventStore]` → `[sse.EventStore]`
4. [ ] Fix `event_store_sse.go:101` — `[SSEEventStore]` → `[sse.EventStore]`
5. [ ] Fix `htmx_extensions.go:29` — `SSEStream` → `sse.Stream`
6. [ ] Fix `ack.go:65` — `SSEEvent.Data` → `sse.Event.Data`
7. [ ] Fix `structured_error.go:141` — `SSEEvent.Data` → `sse.Event.Data`
8. [ ] Fix `ws_broadcaster.go:13` — `[SSEStream]` → `[sse.Stream]`

### High priority — test Describe/comment correctness (5 min)

9. [ ] Fix `sse_event_test.go:304` — `ParseSSEEventID` → `sse.ParseEventID`
10. [ ] Fix `sse_event_test.go:316` — `NewSSEEventID` → `sse.NewEventID`
11. [ ] Fix `sse_reconnect_test.go:25` — `SSEStream` → `sse.Stream`
12. [ ] Fix `sse_broadcaster_test.go:260` — `SSEStream` → `sse.Stream`
13. [ ] Fix `bdd_realtime_test.go:21` — `NewSSEStream's` → `sse.NewStream's`
14. [ ] Fix `sse_reconnect_integration_test.go:17` — `SSEEventStore` → `sse.EventStore`

### Medium priority — recipe docs (10 min)

15. [ ] Fix `docs/recipes/offline-command-sync.md` — `SSEStream` → `sse.Stream`, `cqrshtmx.NewSSEStream` → `sse.NewStream`

### Medium priority — verification sweeps (15 min)

16. [ ] Run lint sweep: `examples/basic`, `examples/admin-demo`, `e2e/server`
17. [ ] Run test sweep: `integration_test` module
18. [ ] Run coverage gate: `nix run .#coverage-gate`
19. [ ] Run `dashboardui` lint after migration

### Medium priority — feedback_features_test.go decision

20. [ ] Decide: remove the `feedback_features_test.go` SSE constants block (tests go-sse, not cqrs-htmx), convert to testing that the deprecated aliases still equal the go-sse constants (backward-compat test), or leave as-is.

### Low priority — historical docs (annotate, don't rewrite)

21. [ ] `docs/feedback/2026-07-05_overview-consumer-feedback.md` — historical feedback referencing deprecated API. Annotate with deprecation note or leave as historical record.
22. [ ] `docs/status/archive/2026-06-08_02-12_sse-ws-polish-v2.2-adoption.md` — archived status, leave as historical.
23. [ ] `docs/status/2026-07-26_21-50_session-self-critique-sse-reconnect-replay.md` — historical, leave.
24. [ ] `dashboardui/CHANGELOG.md` — historical changelog, leave.

### Low priority — long-term decisions

25. [ ] Decide v5.0 deletion timeline for deprecated SSE re-export aliases
26. [ ] Consider adding a backward-compat test that exercises every deprecated alias (ensures aliases don't rot)
27. [ ] Consider adding a `go vet` / staticcheck CI check that NEW code doesn't introduce SA1019 warnings on deprecated aliases
28. [ ] Update `docs/guides/` — check if any guides reference deprecated SSE symbols
29. [ ] Check `adminui/README.md`, `loginpage/README.md`, `dashboardui/README.md` for deprecated SSE references (sweep showed clean but worth double-checking)
30. [ ] Add a "Migration: SSE re-export deprecation" section to README.md or a guide doc

### Stretch — broader quality

31. [ ] Run full `nix run .#lint` across all 18 workspace modules
32. [ ] Run full `nix run .#test` across all 18 workspace modules
33. [ ] Verify `feedback_features_test.go` — the entire file may have other blocks testing deprecated cqrs-htmx re-exports now pointing at go-sse
34. [ ] Check `benchmark_server_test.go` — appeared in the unqualified SSE sweep; verify it's using `sse.*` correctly
35. [ ] Audit whether the `// Deprecated:` messages are consistent in format across all 14 symbols in `sse_event.go` and `sse_store.go`

---

## g) Questions (3 max — things I cannot figure out myself)

### Q1: v5.0 deletion timeline?

Should the deprecated SSE re-export aliases (`SSEEvent`, `NewSSEStream`, `WriteSSEEvent`, etc.) be **deleted in v5.0**, or **kept indefinitely** as zero-cost backward-compat shims?

- **Context:** Type aliases have zero runtime cost. But keeping them means the deprecated surface area grows forever. The httputil re-exports (`CSRFMiddleware`, `ServerTimingMiddleware`) set a precedent — they were kept. But the Aggregate→Stream migration in go-cqrs-lite was eventually deleted.
- **Why I can't decide:** This is a product/versioning philosophy decision, not a technical one.

### Q2: Should `feedback_features_test.go` SSE constants block stay, go, or convert?

The block now asserts `sse.EventConnected == "connected"` and `sse.EventHeartbeat == "heartbeat"` — testing go-sse's constants, not cqrs-htmx's deprecated re-exports.

Options:

- **Remove** — cqrs-htmx tests shouldn't verify a dependency's constant values
- **Convert** — assert that the deprecated aliases (`cqrshtmx.SSEEventConnected`) still equal `sse.EventConnected` (backward-compat test)
- **Keep** — the test documents the expected wire-format values, which is useful regardless of where the constant lives

### Q3: Should I fix the historical docs (feedback, archived status reports)?

`docs/feedback/2026-07-05_overview-consumer-feedback.md` and several archived status reports reference deprecated SSE symbols in code examples.

- **Fix in place** — misleading for anyone who copy-pastes from them
- **Annotate** — add a deprecation note at the top without rewriting
- **Leave alone** — historical documents are snapshots in time; changing them rewrites history

---

## Metrics

| Metric                                                | Value         |
| ----------------------------------------------------- | ------------- |
| Deprecated `cqrshtmx.SSE*` refs in README.md          | 0 (was 15+)   |
| Deprecated refs in `examples/basic/README.md`         | 0 (was 1)     |
| Deprecated `[godoc]` links in production .go comments | 9 (NOT FIXED) |
| Deprecated refs in test Describe/comment strings      | 6 (NOT FIXED) |
| Deprecated refs in recipe docs                        | 2 (NOT FIXED) |
| Build status                                          | Passing       |
| Test status                                           | Passing       |
| Lint status                                           | 0 issues      |
| Coverage gate                                         | Not verified  |
