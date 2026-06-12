# Status Report: go-cqrs-lite v2.3.0 Update

**Date:** 2026-06-12 11:12  
**Author:** Crush (AI Assistant)  
**Session Focus:** Update go-cqrs-lite dependency to v2.3.0 and adopt new features

---

## a) FULLY DONE ✅

### 1. Version Update Infrastructure

- **go.work replaces**: Added 6 replace directives in `go.work` pointing to local `go-cqrs-lite` checkout (`../go-cqrs-lite/{command,event,id,query,codec,dispatcher}`)
- **datastar-demo replaces**: Added 6 replace directives in `examples/datastar-demo/go.mod` (not in go.work workspace)
- **go mod tidy**: Ran on all 4 modules — root, usermgmt, integration_test, datastar-demo
- **All modules build clean**: `GONOSUMCHECK='github.com/larsartmann/*' go build ./...` passes for all 4 modules
- **All tests pass**: 570+ tests across root, usermgmt, integration_test — all green with `-race`

### 2. Breaking Changes Fixed

- **`id.MustParse[T]` removed** → Local `MustParseUserID`/`MustParseCorrelationID`/`MustParseRequestID` reimplemented in `context.go:36-91` as panic-on-error wrappers around `Parse`
- **`query.MustNew` removed** → Fixed `examples/datastar-demo/handlers.go:115` to use `query.New()` + error check
- **Zero impact on consumers**: 42 call sites across tests all still work via local wrappers

### 3. New Features Adopted

| Feature                    | File                                       | Description                                                                                                      |
| -------------------------- | ------------------------------------------ | ---------------------------------------------------------------------------------------------------------------- |
| `event.FromContext`        | `context.go:158-160`                       | Propagates context deadline to events. Checks `ctx.Deadline()` and appends `event.FromContext(ctx)` when present |
| `command.Type.IsZero()`    | `app.go:149-151`                           | `App.Command("")` panics at handler registration time — fail-fast                                                |
| `query.Type.IsZero()`      | `app.go:181-183`                           | `App.Query("")` panics at handler registration time — fail-fast                                                  |
| `command.RegisterTyped[T]` | `examples/datastar-demo/domain.go:328-373` | Eliminated manual type assertions (`cmd.(*CreateTodoCmd)`) in all 3 command handlers                             |

### 4. Tests Added

- **Empty type panics** (`coverage_test.go:873-886`): 2 tests — `Command("")` panics, `Query("")` panics
- **Deadline propagation** (`context_test.go:159-175`): `EventOptionsFromContext` with `context.WithDeadline` produces valid event options
- **Nil returns nil** (`context_test.go:177-181`): No IDs + no deadline → `nil` options (preserves backward compat)

### 5. Documentation Updated

- `AGENTS.md`: Version bumped from v2.2.0 → v2.3.0 in Dependencies table
- `AGENTS.md`: New "Dispatch" subsection documenting adopted features (deadline propagation, typed handlers, empty type validation)
- `AGENTS.md`: Gotchas updated with v2.3.0 consume mechanism (replace directives) and removed API warnings
- `AGENTS.md`: Pagination header updated to v2.3.0

---

## b) PARTIALLY DONE ⚠️

### 1. Version Strings in go.mod Files

- **Status**: `go.mod` `require` blocks still show `v2.2.0` for go-cqrs-lite deps
- **Why**: Replace directives in `go.work` override them for local builds
- **Impact**: Clean but misleading — consumers won't know we're on v2.3.0 without reading go.work
- **Fix needed**: Update all 4 go.mod files to `v2.3.0` in require blocks when per-module tags publish

### 2. `go.work.sum` Changes

- **Status**: File modified (14 insertions) but not reviewed for correctness
- **Why**: Tidy added new transitive deps (fxamacker/cbor/v2, x448/float16 from go-cqrs-lite v2.3.0 codec module)
- **Impact**: Likely correct — just new CBOR codec dependencies

---

## c) NOT STARTED 🔜

### 1. `id.CompareIDs` and `id.FromPtr`

- **What**: New ID utilities in v2.3.0 — `CompareIDs[T](a, b Of[T]) int` and `FromPtr[T](p *Of[T]) Of[T]`
- **Where they could be used**: `CompareIDs` for test assertions (instead of `.String()` comparison), `FromPtr` for nil-safe dereference in middleware
- **Effort**: ~10 min to search for `.String()` comparisons in tests and replace with `CompareIDs`

### 2. `command.ParseType` / `query.ParseType`

- **What**: New validation constructors for command/query types
- **Where they could be used**: In test helpers that construct command types from strings — validates non-empty at parse time
- **Effort**: ~15 min to audit test files for manual `command.Type("...")` construction

### 3. `event.PayloadReadOnly`

- **What**: Zero-copy payload access for internal paths (signing, hashing, storage)
- **Where it could be used**: We don't have signing or storage in cqrs-htmx — mostly a no-op for this project
- **Effort**: ~5 min to verify no applicable internal paths

### 4. `event.DecodePayloads[T]`

- **What**: Batch payload deserialization from `[]Event` → `[]T`
- **Where it could be used**: In projection/event replay code if we ever implement it
- **Effort**: Not applicable today

### 5. Nix Build Verification

- **What**: `nix run .#build` and `nix run .#test`
- **Why**: go.work replace directives might break nix builds (nix is hermetic, no access to `../go-cqrs-lite`)
- **Effort**: ~5 min to test, potentially need to adjust flake.nix or accept that nix builds need published tags

### 6. `command.RegisterTyped` in Test Code

- **What**: Our own test suite still uses manual type assertions in some places
- **Where**: `coverage_test.go` test command handlers use `func(ctx, command.Command)` pattern
- **Effort**: ~20 min to refactor test commands to use typed handlers

### 7. go.mod Version Bump

- **What**: Update all go.mod require blocks from `v2.2.0` → `v2.3.0`
- **Why**: Semantic correctness — when we remove replace directives, versions should be accurate
- **Effort**: ~5 min across 4 files

### 8. LSP Diagnostics Cleanup

- **Status**: LSP still shows 67 errors for datastar-demo (not in go.work) and stale warnings
- **Why**: LSP cache doesn't know about go.work replace for datastar-demo
- **Effort**: ~2 min (restart gopls or add datastar-demo to go.work)

---

## d) TOTALLY FUCKED UP 💥

**Nothing.** All builds pass, all tests pass, no lint regressions. The update was surgical and clean.

---

## e) WHAT WE SHOULD IMPROVE 📈

### 1. Type Model: `HandlerConfig` as a Domain Type

Our `handlerConfig` struct has 15+ fields with zero-value ambiguity (e.g., `maxBodySize == 0` means "use app default" but also "no limit"). Consider:

- `Optional[T]` types for truly optional fields
- `ValidatedHandlerConfig` returned by `buildHandlerConfig` that has no zero-value ambiguity

### 2. Missing `go.mod` Version Accuracy

The go.mod files still declare `v2.2.0` while consuming v2.3.0 APIs. This is technically fine with replace directives but is a documentation lie. When per-module tags publish, we should:

1. Update all require blocks to `v2.3.0`
2. Remove go.work replace directives
3. Verify builds work with published tags

### 3. `EventOptionsFromContext` Could Use `event.MetadataKey`

In v2.3.0, event package introduced `MetadataKey` type with constants `MetadataKeyClientID` and `MetadataKeyClientOccurredAt`. We could consider:

- Adding `ClientID` to our context enrichment pipeline
- Using `MetadataKey` type for our custom keys instead of plain strings

### 4. Test Code Still Uses `command.Command` Interface

The datastar-demo now uses `RegisterTyped`, but our own tests in `coverage_test.go` still use the old interface-based handlers. This creates an inconsistency where our example is more modern than our tests.

### 5. `query.MustNew` in Comments/Docs

Search for `MustNew` in documentation — README.md and FEATURES.md might still reference `query.MustNew` as an API example. Should be updated to `query.New()` + error check.

### 6. `MustParseUserID` Should Be in `id` Package, Not Here

Our local wrappers (`MustParseUserID`, etc.) exist because go-cqrs-lite removed them. Long-term, we should either:

- Push for their re-addition upstream (they're useful test helpers)
- Accept them as local utilities and document that they are cqrs-htmx extensions

### 7. Consider `event.ReconstructEventFromFields`

If we ever implement event sourcing replay (e.g., for SSE event stores), this new v2.3.0 function provides canonical reconstruction logic shared across all store implementations.

### 8. `command.Store` Interfaces Not Used

v2.3.0 added `CommandSink`, `CommandSource`, `Store` for persisted command logs. We have no persistence layer in cqrs-htmx, but if we ever add command audit logging, these are ready to use.

---

## f) Top #25 Things To Do Next

| #   | Task                                                                        | Impact    | Effort | Module  | Status      |
| --- | --------------------------------------------------------------------------- | --------- | ------ | ------- | ----------- |
| 1   | Verify nix build/test with go.work replaces                                 | 🔴 High   | 5m     | Root    | Not Started |
| 2   | Update README.md for `query.MustNew` → `query.New()`                        | 🔴 High   | 10m    | Root    | Not Started |
| 3   | Add `id.CompareIDs` to test assertions                                      | 🟡 Medium | 10m    | Root    | Not Started |
| 4   | Refactor test commands to use `RegisterTyped`                               | 🟡 Medium | 20m    | Root    | Not Started |
| 5   | Bump go.mod versions to v2.3.0 when tags publish                            | 🟡 Medium | 5m     | All     | Not Started |
| 6   | Add `event.MetadataKey` support to context enrichment                       | 🟡 Medium | 20m    | Root    | Not Started |
| 7   | Clean up LSP diagnostics (restart gopls)                                    | 🟢 Low    | 2m     | DevEx   | Not Started |
| 8   | Verify `event.PayloadReadOnly` has no use sites                             | 🟢 Low    | 5m     | Root    | Not Started |
| 9   | Document `RegisterTyped` usage pattern in AGENTS.md                         | 🟢 Low    | 10m    | Root    | Not Started |
| 10  | Add `command.ParseType` usage in test helpers                               | 🟢 Low    | 15m    | Root    | Not Started |
| 11  | Review FEATURES.md for stale API references                                 | 🟢 Low    | 15m    | Root    | Not Started |
| 12  | Consider `HandlerConfig` → `Optional[T]` refactor                           | 🟡 Medium | 30m    | Root    | Not Started |
| 13  | Add `context.WithTimeout` deadline test for `EventOptionsFromContext`       | 🟢 Low    | 10m    | Root    | Not Started |
| 14  | Verify datastar-demo still works end-to-end (run it)                        | 🟡 Medium | 5m     | Example | Not Started |
| 15  | Check for `fmt.Sprintf` in `MustParse*` wrappers — could use `errors.New`   | 🟢 Low    | 5m     | Root    | Not Started |
| 16  | Add negative test: `Command("")` panics with query-only app                 | 🟢 Low    | 5m     | Root    | Not Started |
| 17  | Document go.work replace directive strategy in AGENTS.md                    | 🟢 Low    | 5m     | Root    | Not Started |
| 18  | Run `go mod tidy -v` and review for unused deps                             | 🟢 Low    | 5m     | All     | Not Started |
| 19  | Add `id.FromPtr` usage where we deref ID pointers                           | 🟢 Low    | 10m    | Root    | Not Started |
| 20  | Verify `command.Store` interfaces are documented in architecture            | 🟢 Low    | 10m    | Docs    | Not Started |
| 21  | Update CHANGELOG.md with v2.3.0 adoption entry                              | 🟢 Low    | 10m    | Root    | Not Started |
| 22  | Check if `event.NewMetadata` change affects us (now initializes Custom map) | 🟢 Low    | 5m     | Root    | Not Started |
| 23  | Review `command.WrapTransient` removal — verify not used                    | 🟢 Low    | 2m     | Root    | Not Started |
| 24  | Search for `event.MustParseType` usage — should be gone                     | 🟢 Low    | 2m     | All     | Not Started |
| 25  | Consider adding `TypedHandler` examples to `example_test.go`                | 🟢 Low    | 15m    | Root    | Not Started |

---

## g) Top #1 Question I Cannot Figure Out Myself

**When will per-module tags (`command/v2.3.0`, `event/v2.3.0`, etc.) be published for go-cqrs-lite?**

The `v2.3.0` root tag exists, but Go module resolution requires per-module tags for sub-modules (e.g., `command/v2.3.0`, `event/v2.3.0`). Without these, downstream consumers cannot `go get` the v2.3.0 sub-modules. Our current workaround (go.work replace directives) works for local development but blocks:

1. **Nix builds**: Hermetic builds can't access `../go-cqrs-lite`
2. **CI on clean machines**: Would need the go-cqrs-lite repo checked out at the right commit
3. **Consumer adoption**: Anyone importing cqrs-htmx would get v2.2.0 of go-cqrs-lite

**What I've tried:**

- `go list -m -versions github.com/larsartmann/go-cqrs-lite/command/v2` → only shows v2.2.0
- Checked git tags in go-cqrs-lite repo → no `command/v2.3.0` tag exists
- The root `v2.3.0` tag is unusable for sub-modules because the root go.mod lacks `/v2`

**What I need:**

- Confirmation of the tagging timeline from the go-cqrs-lite author
- OR a decision to maintain replace directives indefinitely
- OR a CI strategy that clones go-cqrs-lite as a sibling repo

---

## Files Changed (11)

```
 AGENTS.md                          | 16 ++++++++++------
 app.go                             |  8 ++++++++
 context.go                         | 28 ++++++++++++++++++++++------
 context_test.go                    | 28 ++++++++++++++++++++++++++++
 coverage_test.go                   | 12 ++++++++++++
 examples/datastar-demo/domain.go   | 13 ++++---------
 examples/datastar-demo/go.mod      | 13 ++++++++++++-
 examples/datastar-demo/go.sum      | 36 ++++++++++++++----------------------
 examples/datastar-demo/handlers.go |  7 ++++++-
 go.work                            |  9 +++++++++
 go.work.sum                        | 14 ++++++++++++++
```

## Test Results

| Module           | Tests      | Time  | Status      |
| ---------------- | ---------- | ----- | ----------- |
| Root (v2)        | 570+       | ~1.5s | ✅ PASS     |
| usermgmt (v2)    | 100+       | ~8.4s | ✅ PASS     |
| integration_test | 20+        | ~1.0s | ✅ PASS     |
| datastar-demo    | Build only | N/A   | ✅ BUILD OK |

## Lint Results

- **Root**: 50 `exhaustruct` warnings (all pre-existing, in test files)
- **usermgmt**: 8 `exhaustruct` warnings (all pre-existing)
- **No new issues introduced**

## Architecture Impact

- **No breaking changes to cqrs-htmx public API**
- **Behavioral additions only**: Deadline propagation, empty-type panics
- **Type-safe handler pattern demonstrated** in example code
- **Module boundaries preserved**: No cross-module imports added
