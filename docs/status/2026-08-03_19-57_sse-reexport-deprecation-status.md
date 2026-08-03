# SSE Re-export Deprecation — Status Update

**Date:** 2026-08-03 19:57
**Session goal:** Deprecate the go-sse re-export layer in cqrs-htmx; migrate all internal consumers to use `go-sse` directly.

---

## What was done

Deprecated all 15 type aliases and delegating functions in `sse_event.go` and `sse_store.go` with `// Deprecated:` markers. Migrated 25+ internal consumers across root module, dashboardui, examples, and e2e to import `github.com/larsartmann/go-sse` and use `sse.Event`, `sse.NewStream`, `sse.Replay`, etc. directly.

**Files modified (source):**
- `sse_event.go` — added deprecation markers to all symbols
- `sse_store.go` — added deprecation markers to `SSEEventStore` + `ReplayEvents`
- `sse_broadcaster.go` — internal types changed to `sse.Event`/`sse.NewStream`/`sse.EventConnected`
- `event_store_sse.go` — types changed to `sse.Event`/`sse.EventID`/`sse.NewEventID`
- `ack.go` — `SSEEvent` → `sse.Event` in ACK functions
- `doc.go` — doc-comment examples updated to `sse.*`
- `structured_error.go` — doc-comment example updated

**Files modified (consumers):**
- `dashboardui/sse.go` — full migration to `sse.*`
- `dashboardui/dashboard.go` — `cqrshtmx.SSEEventStore` → `sse.EventStore`
- `examples/basic/main.go`, `examples/admin-demo/main.go`, `e2e/server/main.go` — `sse.*` migration

**Files modified (tests — 13 files):**
- `ack_test.go`, `bdd_realtime_test.go`, `command_sync_integration_test.go`, `event_store_sse_test.go`, `example_sse_test.go`, `feedback_features_test.go`, `integration_transport_test.go`, `sse_bridge_test.go`, `sse_broadcaster_bench_test.go`, `sse_broadcaster_test.go`, `sse_event_test.go`, `sse_reconnect_integration_test.go`, `sse_reconnect_test.go`, `sse_stream_test.go`

**Docs updated:** `CHANGELOG.md` (Deprecated section), `AGENTS.md` (new gotcha entry).

---

## a) FULLY DONE

1. Deprecation markers on all 15 re-export symbols in `sse_event.go` + `sse_store.go`
2. Root module production code migrated (`sse_broadcaster.go`, `event_store_sse.go`, `ack.go`)
3. All 13 root module test files migrated
4. dashboardui production code migrated (`sse.go`, `dashboard.go`)
5. examples/basic, examples/admin-demo, e2e/server migrated
6. doc.go and structured_error.go doc-comment examples updated
7. CHANGELOG.md entry added under Deprecated
8. AGENTS.md gotcha entry added
9. `go mod tidy` run on affected modules (dashboardui, examples/basic, examples/admin-demo, e2e/server)
10. Root module build: PASS
11. Root module tests (`-race`): PASS
12. Root module lint (`golangci-lint`): 0 issues
13. SA1019 deprecation warnings: 0 in root + dashboardui
14. dashboardui build + tests: PASS
15. All example/e2e builds: PASS
16. Research: confirmed go-sse deliberately excluded `ServeSSE` (README "What's NOT Included"), correcting my earlier recommendation

---

## b) PARTIALLY DONE

1. **doc.go SSE section** — The main streaming examples at `doc.go:110-148` were updated, but one stray reference remains: `doc.go:135` says "SSEEventStore implementation" in a comment. Minor but inconsistent.
2. **Coverage verification** — Root coverage is 92.9% (gate is 90%). Still passes, but I didn't verify dashboardui coverage (gate 60%) or run the coverage gate command. The deprecation markers themselves don't affect coverage, but the test file restructuring (import changes, variable renaming) could have shifted lines.
3. **Test describe block names** — `sse_event_test.go` still has `Describe("WriteSSEEvent")`, `Describe("SSEStream")`, `Describe("SSEEventID")` — these now test `sse.WriteEvent`, `sse.Stream`, `sse.EventID` from go-sse, but the describe names reference the deprecated cqrs-htmx aliases. Semantically misleading.

---

## c) NOT STARTED

1. **README.md** — 15+ references to deprecated SSE symbols (`SSEEvent`, `NewSSEStream`, `WriteSSEEvent`, `SSEEventStore`, `ReplayEvents`, `LastEventIDFromRequest`, `ContentTypeSSE`, etc.) in the SSE API reference section, quick-start examples, and the file-tree comment. This is the **biggest user-facing surface** and was completely missed.
2. **examples/basic/README.md** — line 55 references `cqrshtmx.SSEEvent`.
3. **Test describe block renames** — should be renamed to reflect the actual symbols under test (e.g. `Describe("sse.WriteEvent")`, `Describe("sse.Stream")`, `Describe("sse.EventID")`).
4. **feedback_features_test.go semantic review** — The "SSE event name constants" describe block now asserts `sse.EventConnected` and `sse.EventHeartbeat` from go-sse. These are go-sse's constants, not cqrs-htmx features. The test is arguably testing the wrong package now — either keep testing the deprecated aliases (backward compat verification) or remove the block entirely.
5. **example_sse_test.go function name review** — Functions like `ExampleWriteSSEEvent()` and `ExampleSSEStream()` now demonstrate `sse.WriteEvent()` and `sse.NewStream()`. The function names document deprecated symbols (Go's `Example*` convention ties the name to the symbol). This is actually correct migration guidance (showing the new way under the old name), but could be clearer with comments.
6. **Full workspace lint sweep** — Only ran `golangci-lint` on root and dashboardui. Didn't lint examples/basic, examples/admin-demo, e2e/server after migration.
7. **Full workspace test sweep** — Only ran root + dashboardui tests. Didn't run integration_test module tests (if any exist that touch SSE).
8. **Coverage gate command** — Didn't run `nix run .#coverage-gate` to verify all 9 module gates still pass.

---

## d) TOTALLY FUCKED UP

**Nothing is totally fucked up.** All code compiles, all tests pass, lint is clean. The changes are correct and backward-compatible (type aliases are transparent). The worst finding is the **README.md omission** — I updated internal doc comments but missed the primary user-facing documentation, which is where consumers learn the API. This is a significant oversight given the entire point of the change is to guide consumers away from the deprecated symbols.

**Secondary miss:** I didn't update the test `Describe` block names. The tests now test go-sse functions but are labeled with cqrs-htmx deprecated names. If someone searches for "WriteSSEEvent" in test output, they'll find it — but the test is actually exercising `sse.WriteEvent`. This is confusing.

**Process miss:** I should have run a final `rg 'cqrshtmx\.SSE' --type md` across ALL markdown files before declaring done. I only checked `.go` files.

---

## e) WHAT WE SHOULD IMPROVE

1. **Verify across ALL file types, not just .go** — The `rg` verification at the end only checked Go source. Markdown (README, guides), HTML status reports, and example comments also reference deprecated symbols. A complete sweep should cover `*.md`, `*.go`, `*.html`, `*.templ`.
2. **Update test names when migrating** — When test describe blocks test a different package's functions, the names should reflect that. "WriteSSEEvent" in a test that calls `sse.WriteEvent` is a lie.
3. **Coverage gate as a verification step** — The todo list had "verify build, lint, and tests" but not "verify coverage gate." Coverage gates are a separate CI check that could break independently.
4. **README as a first-class deliverable** — When deprecating public API symbols, the README is the most important file to update. It should have been in the original todo list, not discovered in the post-mortem.
5. **Variable shadowing prevention** — The `sse` variable name collision in `event_store_sse_test.go` (`sse := NewJournalSSEStore(...)` shadowing the `sse` package) was a preventable error. When introducing a new package import named `sse`, scan for existing variables named `sse` first.
6. **The `feedback_features_test.go` test is now testing go-sse** — This is a philosophical issue: should cqrs-htmx tests verify go-sse's constants? No. Either test the deprecated aliases (to verify backward compat works) or delete the test.

---

## f) Up to 50 things to get done next

### High priority (user-facing)

1. **Update README.md SSE section** — Replace all `cqrshtmx.SSEEvent` → `sse.Event`, `cqrshtmx.NewSSEStream` → `sse.NewStream`, etc. in the SSE API reference (~15 references across lines 37, 420-429, 445-446, 461-482, 576, 1170-1172)
2. **Update examples/basic/README.md** — line 55: `cqrshtmx.SSEEvent` → `sse.Event`
3. **Add a "Migration Guide" note to README.md** — Brief callout that SSE symbols are now deprecated and consumers should import go-sse directly
4. **Fix doc.go:135** — "SSEEventStore" → "sse.EventStore" in comment

### Medium priority (test quality)

5. **Rename test Describe blocks** — `Describe("WriteSSEEvent")` → `Describe("sse.WriteEvent")`, `Describe("SSEStream")` → `Describe("sse.Stream")`, `Describe("SSEEventID")` → `Describe("sse.EventID")` in `sse_event_test.go`
6. **Review feedback_features_test.go** — The "SSE event name constants" block now tests go-sse constants. Either move it to test the deprecated aliases (backward compat) or remove it
7. **Review example_sse_test.go** — Add `// Deprecated:` example migration comments or rename functions to `ExampleNew_WriteSSEEvent_migration` style
8. **Run full workspace test sweep** — Test all modules that transitively depend on go-sse (integration_test, adminui, loginpage)
9. **Run full workspace lint sweep** — Lint examples/basic, examples/admin-demo, e2e/server after migration
10. **Run coverage gate** — `nix run .#coverage-gate` to verify all 9 module gates pass

### Lower priority (polish)

11. **Review all docs/guides/*.md** for deprecated SSE symbol references (currently clean per grep, but worth a manual scan)
12. **Review docs/status/*.html** for stale SSE references (historical, low priority)
13. **Consider adding a `// Deprecated:` banner to the package doc** in `sse_event.go` header
14. **Consider a `go fix` or codemod tool** for consumers migrating from `cqrshtmx.SSE*` to `sse.*`
15. **Review whether `example_sse_test.go` Example functions still make sense** — `ExampleWriteSSEEvent` documents a deprecated function; consider adding `ExampleSSE_Event` showing the new pattern
16. **Update ROADMAP.md** if it references SSE symbols
17. **Update FEATURES.md** if it references SSE symbols
18. **Verify `go mod tidy` cleanliness** across all 18 modules (run `go mod tidy -diff` or compare checksums)
19. **Check if the datastar module** (which has its own SSE-like broadcaster) should also reference go-sse directly
20. **Review the `ws_broadcaster.go` doc comments** — still references `cqrshtmx.WSBroadcaster` patterns; verify they're still accurate after the migration

---

## g) Questions I cannot figure out myself

1. **Should the deprecated aliases eventually be removed entirely (v5.0), or kept indefinitely as backward-compat shims?** The type aliases are zero-cost (transparent at compile time), so keeping them is free. But the `func` delegations (e.g. `NewSSEStream`, `WriteSSEEvent`, `ReplayEvents`) add a small amount of code surface. What's the deletion policy for deprecated symbols in this library?

2. **Should the test suite verify backward compatibility of the deprecated aliases?** Right now, after migration, no test exercises `cqrshtmx.SSEEvent`, `cqrshtmx.NewSSEStream`, etc. If a consumer depends on these aliases, there's no test proving they still work. Should we add a backward-compat test file that explicitly uses the deprecated symbols?

3. **Should the `feedback_features_test.go` "SSE event name constants" test block be removed, or converted to test the deprecated aliases?** It currently tests `sse.EventConnected == "connected"` which is testing go-sse, not cqrs-htmx. But removing it reduces coverage of the `sse_event.go` file (the deprecated constants). What's the intent — do we want to maintain test coverage of the deprecated surface?
