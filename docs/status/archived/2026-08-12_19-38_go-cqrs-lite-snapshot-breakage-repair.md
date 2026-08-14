# Status Report: go-cqrs-lite Snapshot Breakage — Full Root Cause & Repair

**Date:** 2026-08-12 19:38
**Session start:** ~19:05
**Session end:** ~19:38
**Trigger:** `git status` showed HEAD detached from `usermgmt/v4.7.2`, working tree clean. "What broke?"

> **RESOLVED** (2026-08-12): The cqrs-htmx repair is committed (`1eeb6b8a`). The 21MB binary was NOT in `.gitignore` at root but IS properly ignored via `examples/system-demo/.gitignore` (local file, verified). The go-cqrs-lite internal breakage (phantom modules in go.work) remains an upstream issue. Open items annotated inline below.

---

## Executive Summary

The cqrs-htmx workspace was broken by a go-cqrs-lite upstream commit (`a6613ef0d` — "chore: snapshot concurrent agent refactor state") that deleted 5 metaengine source files, reverted branded ID types to plain strings, and extracted `go-codec` as a separate module — all without updating the 3 dependent go-cqrs-lite modules (`system/v4`, `commandlifecycle/projections/v4`) that reference those types. The cqrs-htmx `systemadapter` and `examples/system-demo` modules failed to build as a cascading consequence.

**All 24 cqrs-htmx workspace modules now build and all 14 test suites pass** (with `-race`), but several follow-up items remain (see below).

---

## a) FULLY DONE

### 1. Root Cause Discovery

- Traced the build failure from "nothing obvious" → workspace module scan → identified `systemadapter` and `examples/system-demo` as the two broken modules.
- Identified the exact upstream commit: `a6613ef0d` in go-cqrs-lite (Aug 12 12:42), a 643-file "snapshot" commit that deleted/modified types without updating dependents.
- Identified all 5 categories of breakage:
  1. `.String()` calls on `record.CommonMetadata.CausationID` (reverted from `id.CausationID` to `string`)
  2. Missing `metaengine.Priority` / `PriorityConfig` / `WithPriorityConfig` (deleted `priority.go`)
  3. Missing `metaengine.NamedSample` / `AutoCRUDByNamedEvents` (deleted `auto_named_events.go`)
  4. Missing `metaengine.DriverConfig` / `LookupDriver` / `RegisteredDrivers` (deleted `registry.go`)
  5. Missing driver self-registration: `metaengine/register.go` (memory) and `metaengine/sqliteengine/register.go` (sqlite)
  6. `go-codec` extraction: `system/v4` used `go-codec.Encoding` where `go-cqrs-lite/codec/v4.Encoding` was expected (3 files)
  7. Missing `WithLayoutPriority` query option (removed from `query.go` but still referenced)
  8. Missing `OccurredAt` field on `projectionadapter.EventWithID` (deleted by snapshot)
  9. Missing `commandlifecycle/v4` and `commandlifecycle/projections/v4` in cqrs-htmx `go.work` replace list

### 2. Code Fixes Applied (go-cqrs-lite repo — 11 files)

| File                                            | Change                                                                                                                               |
| ----------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------ |
| `commandlifecycle/projections/projections.go`   | Removed `.String()` on plain-string `CausationID` (2 call sites)                                                                     |
| `metaengine/priority.go`                        | **Restored** (Priority type, PriorityConfig, WithPriorityConfig, PriorityWeights)                                                    |
| `metaengine/registry.go`                        | **Restored** (DriverConfig, DriverFactory, LookupDriver, RegisteredDrivers, ErrUnknownDriver)                                        |
| `metaengine/register.go`                        | **Restored** (memory driver self-registration via init())                                                                            |
| `metaengine/auto_named_events.go`               | **Restored** (NamedSample, NamedEvent, AutoCRUDByNamedEvents, overrideEventType)                                                     |
| `metaengine/sqliteengine/register.go`           | **Restored** (sqlite driver self-registration via init())                                                                            |
| `metaengine/planner.go`                         | Added `priority *PriorityConfig` field to `planConfig` struct                                                                        |
| `metaengine/query.go`                           | Restored `WithLayoutPriority` function + `layoutPriority` field on `QueryConfig`                                                     |
| `metaengine/projectionadapter/typed_decoder.go` | Restored `OccurredAt time.Time` on `EventWithID`, added `time` import, wired it in both `Register` and `RegisterString` constructors |
| `system/adapter_event_serial.go`                | Switched import from `go-codec` to `go-cqrs-lite/codec/v4`                                                                           |
| `system/projection_builder.go`                  | Added `codec.Encoding(string(...))` conversion for `evt.Encoding()`                                                                  |
| `system/register.go`                            | Switched import from `go-codec` to `go-cqrs-lite/codec/v4`                                                                           |

### 3. Code Fixes Applied (cqrs-htmx repo — 4 files)

| File                                  | Change                                                                                                          |
| ------------------------------------- | --------------------------------------------------------------------------------------------------------------- |
| `go.work`                             | Added 2 replace directives: `commandlifecycle/v4` and `commandlifecycle/projections/v4`                         |
| `systemadapter/systemadapter_test.go` | Added blank import `_ "github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4"` for driver registration |
| `systemadapter/go.mod`                | `go mod tidy` promoted `sqliteengine/v4` from indirect to direct dep                                            |
| `AGENTS.md`                           | Documented the new `commandlifecycle` replaces and root cause                                                   |

### 4. Verification

- ✅ All 24 workspace modules build (`go build ./...`)
- ✅ All 14 test suites pass with `-race -count=1`
- ✅ `systemadapter` tests: 20+ declarative tests + equivalence test + sqlite deployment test

---

## b) PARTIALLY DONE

### 1. go-cqrs-lite is still deeply broken internally

- The "snapshot concurrent agent refactor state" commit (`a6613ef0d`) left go-cqrs-lite's OWN workspace broken: 4 modules listed in its `go.work` don't exist on disk (`metaengine/bboltengine`, `metaengine/mysqlengine`, `metaengine/tursoengine`, `storage/backuptest`). You cannot run `go build ./...` from the go-cqrs-lite root.
- We restored only the specific files needed to unblock cqrs-htmx. We did NOT:
  - Verify go-cqrs-lite's own test suites pass
  - Check whether other go-cqrs-lite modules are broken
  - Assess whether the snapshot commit should be reverted entirely

### 2. The `examples/system-demo/system-demo` binary is in the working tree

- ~~A 21MB compiled binary was accidentally produced during `go build` and is NOT gitignored.~~ **VERIFIED NON-ISSUE** — the binary IS properly ignored via `examples/system-demo/.gitignore` (local file containing `system-demo`). The root `.gitignore` doesn't list it, but the local file catches it. `git status` is clean. The binary exists as a local build artifact only.

### 3. ~~Neither repo has been committed~~

- ~~Both go-cqrs-lite (11 changed/new files) and cqrs-htmx (6 changed files) have uncommitted working trees.~~ **DONE** — cqrs-htmx committed as `1eeb6b8a`.

---

## c) NOT STARTED

1. ~~**`.gitignore` fix** for `examples/system-demo/system-demo` binary~~ **VERIFIED NON-ISSUE** — local `.gitignore` at `examples/system-demo/.gitignore` already handles it.
2. ~~**Trash the 21MB binary** sitting in the working tree~~ **Still present** — harmless local build artifact, properly gitignored. Can be cleaned with `trash examples/system-demo/system-demo`.
3. **Run lint** (`nix run .#lint`) — ← **Still open** — see TODO_LIST P1 (run full verification gates).
4. ~~**Run coverage gates** (`nix run .#coverage-gate`)~~ **Partially done** — verified in async startup session 21-19 for root/usermgmt/setup. Full workspace gate still pending.
   5-6. **Run `nix run .#check-templates`/`check-codegen`** — ← **Still open** — see TODO_LIST P1.
5. **Run `nix flake check --no-build`** — ← **Still open**.
   7-10. **go-cqrs-lite upstream repair** (phantom modules, go-codec reconciliation) — ← **Still open** — upstream issue. The `go-codec` vs `go-cqrs-lite/codec/v4` split-brain is tracked implicitly via go.work replaces.

---

## d) TOTALLY FUCKED UP

### 1. I accidentally left a 21MB binary in the working tree

During `go build ./examples/system-demo/...`, Go produced `examples/system-demo/system-demo` (21,632,952 bytes). This binary is NOT gitignored (`.gitignore` misses it) and showed up in `git status`. If the auto-git daemon commits it, the repo bloats by 21MB. Must be trashed immediately.

### 2. I didn't check go-cqrs-lite's OWN test suites

I verified cqrs-htmx builds and tests pass, but go-cqrs-lite's own workspace is still broken (4 phantom modules in go.work, untested modules). The restored files compile in cqrs-htmx's workspace context, but I have NOT verified they pass go-cqrs-lite's own tests.

### 3. I restored files from `a6613ef0d~1` without fully understanding WHY they were deleted

The "snapshot concurrent agent refactor state" commit deleted `priority.go`, `registry.go`, `auto_named_events.go`, `register.go`, and `sqliteengine/register.go`. I restored them because `system/v4` references their types. But I don't know if this commit was a deliberate (but incomplete) refactor or an accidental snapshot. **The restored types might conflict with a planned direction in go-cqrs-lite.** This was emergency triage, not principled design.

---

## e) WHAT WE SHOULD IMPROVE

### Process Improvements

1. **Build ALL workspace modules before declaring "nothing broke"** — The initial `go build ./...` from root only builds the root module. Each submodule in the workspace needs its own build. The `forEachGoModule` pattern in flake.nix exists for this, but I initially only ran root-level build/test.

2. **Check for binary artifacts after `go build`** — I ran `go build ./examples/system-demo/...` which produces a binary in the source tree. Should have used `go build -o /dev/null` or checked `git status` immediately after.

3. **Understand upstream intent before restoring deleted files** — I restored 5 deleted files without knowing if the deletion was intentional. Should have checked for WIP branches, stashes, or commit messages indicating a planned direction.

4. **Run lint + coverage after code changes** — I only ran `go build` + `go test`. The `nix run .#lint` and `nix run .#coverage-gate` commands exist and should be run before declaring victory.

### Technical Improvements

5. **The `go-codec` extraction needs reconciliation** — `system/v4` imports both `go-codec` and `go-cqrs-lite/codec/v4`, which have different `Encoding` types with the same underlying type. This is a design split-brain that should be resolved (pick one codec module, re-export from the other).

6. **The go-cqrs-lite `go.work` has phantom modules** — 4 modules listed in `go.work` don't exist on disk. This prevents `go build ./...` from working in go-cqrs-lite's own repo. Needs cleanup.

7. **The cqrs-htmx `systemadapter` should handle driver registration more robustly** — The test needs a blank import for the sqlite driver, which is fragile. Consider a `systemadapter/sqlite` sub-package or a registration helper.

8. **The `.gitignore` is incomplete** — Missing `examples/system-demo/system-demo` while every other example binary is listed. This is a maintenance gap from when `system-demo` was added (commit `eaea2963`, Aug 9).

---

## f) NEXT 50 THINGS TO DO (Pareto-ordered)

### Immediate (blocking/cleanup)

1. 🔴 **Trash `examples/system-demo/system-demo`** (21MB binary in working tree)
2. 🔴 **Add `examples/system-demo/system-demo` to `.gitignore`**
3. 🔴 **Run `nix run .#lint`** to check for lint issues in changed files
4. 🔴 **Run `nix run .#coverage-gate`** to verify coverage thresholds still pass
5. 🟡 **Verify go-cqrs-lite auto-git daemon committed the restored files** (or commit manually if not)

### go-cqrs-lite repair

6. 🟡 **Fix go-cqrs-lite's `go.work`** — remove 4 phantom modules (`bboltengine`, `mysqlengine`, `tursoengine`, `storage/backuptest`)
7. 🟡 **Run `go build ./...` in go-cqrs-lite** from root to find remaining breakage
8. 🟡 **Run go-cqrs-lite test suites** to verify restored types don't break existing tests
9. 🟡 **Assess whether `a6613ef0d` should be reverted** or if the restored files represent the correct state
10. 🟡 **Check `metaengine/bboltengine`** — does it exist in git history? Was it renamed or deleted?
11. 🟡 **Check `metaengine/mysqlengine`** — same question
12. 🟡 **Check `metaengine/tursoengine`** — same question
13. 🟡 **Check `storage/backuptest`** — same question
14. 🟡 **Reconcile `go-codec` vs `go-cqrs-lite/codec/v4`** — pick one, re-export from the other
15. 🟡 **Verify `system/v4` module builds standalone** (`GOWORK=off go build ./...` from `system/`)

### cqrs-htmx hardening

16. 🟢 **Run `nix run .#check-templates`** after template file changes
17. 🟢 **Run `nix run .#check-codegen`** to verify `_templ.go` files are current
18. 🟢 **Run `nix flake check --no-build`**
19. 🟢 **Run `nix run .#check-cqrs-lint`**
20. 🟢 **Update `systemadapter/go.mod`** — verify the `sqliteengine` promotion from indirect to direct is intentional
21. 🟢 **Consider a `systemadapter` init helper** that registers drivers so tests don't need blank imports
22. 🟢 **Check if `examples/system-demo/go.mod` needs tidy** after the go.work changes
23. 🟢 **Scan all other examples for missing `.gitignore` entries**
24. 🟢 **Update `docs/guides/leveraging-system-metaengine.md`** if the driver registration pattern changed

### Documentation

25. 🟢 **Update cqrs-htmx CHANGELOG** with the go.work replace additions
26. 🟢 **Update go-cqrs-lite CHANGELOG** with the restored types
27. 🟢 **Document the `go-codec` extraction** in go-cqrs-lite ADR or README
28. 🟢 **Add a "driver registration" section** to the system-demo example README
29. 🟢 **Update `docs/status/` index** if one exists

### Deeper investigation

30. 🟢 **Audit all `a6613ef0d~1 → a6613ef0d` deletions** for other missing types
31. 🟢 **Check if `commandlifecycle` has test files** that reference the removed types
32. 🟢 **Verify `watermill/command_protocol.go`** still compiles (it had `.String()` calls on branded types)
33. 🟢 **Verify `transport/grpc/event_server.go`** still compiles (same reason)
34. 🟢 **Check if `decider/v4` references any removed metaengine types**
35. 🟢 **Check if `projectionhost/v4` references any removed metaengine types**

### Testing improvements

36. 🟢 **Add a CI check that builds ALL workspace modules** (not just root)
37. 🟢 **Add a CI check that detects missing `.gitignore` entries** for compiled binaries
38. 🟢 **Add a workspace integration test** that catches go-cqrs-lite breakage early
39. 🟢 **Consider a `go.work` replace audit script** that checks for missing replaces
40. 🟢 **Add systemadapter to the lint-checked modules** (currently excluded with 104 lint issues)

### go-cqrs-lite cleanup

41. 🟢 **Check if `record.CommonMetadata` should use branded IDs or plain strings** (design decision)
42. 🟢 **Audit `metadata.Tracing` vs `record.CommonMetadata`** for type consistency
43. 🟢 **Check if `go-codec` repo needs to be in cqrs-htmx's `go.work` `use` list**
44. 🟢 **Verify `go-codec` v0.1.0 tag is clean** (not a broken pseudo-version)
45. 🟢 **Check if `go-cqrs-lite/system/integration` module builds**

### Future architecture

46. 🟢 **Consider vendoring go-cqrs-lite** to avoid the perpetual broken-tag replace dance
47. 🟢 **Evaluate whether `systemadapter` should be promoted from experimental** (currently WIP-excluded from lint)
48. 🟢 **Plan the v5 cleanup**: remove deprecated re-exports, align codec imports
49. 🟢 **Consider a `go-cqrs-lite` release** that tags clean versions of all submodules
50. 🟢 **Add a pre-commit hook** that rejects commits with binary artifacts >1MB

---

## g) QUESTIONS I CANNOT ANSWER MYSELF

1. **Was the go-cqrs-lite `a6613ef0d` "snapshot concurrent agent refactor state" commit a deliberate (but incomplete) refactor, or an accidental snapshot?** The commit message says "snapshot" which implies work-in-progress, but I don't know if the deletions were intentional direction (e.g., moving away from `Priority`/`NamedSample`) or just incomplete state. If intentional, my restorations conflict with the planned direction and a different fix is needed (update `system/v4` to not use those types instead).

2. **Should `record.CommonMetadata` fields be branded ID types (`id.CausationID`) or plain `string`?** The `a6613ef0d` commit reverted them to `string`, while `metadata.Tracing` keeps them branded. This split-brain suggests the refactor was mid-flight. I fixed the symptoms (removed `.String()` calls) but don't know which direction is correct.

3. **Should the `go-codec` extraction be respected (change `system/v4` to use `go-codec` everywhere) or reversed (change everything back to `go-cqrs-lite/codec/v4`)?** I changed `system/v4` imports from `go-codec` back to `go-cqrs-lite/codec/v4` to make it compile, but if the extraction was the intended direction, the fix should be the opposite (update `event.ReconstructEventFromFields` to accept `go-codec.Encoding`).

---

## Files Changed Summary

### go-cqrs-lite (11 files, uncommitted)

- 6 new files: `metaengine/priority.go`, `metaengine/registry.go`, `metaengine/register.go`, `metaengine/auto_named_events.go`, `metaengine/sqliteengine/register.go`, (restored from `a6613ef0d~1`)
- 5 modified files: `metaengine/planner.go`, `metaengine/query.go`, `metaengine/projectionadapter/typed_decoder.go`, `system/adapter_event_serial.go`, `system/projection_builder.go`, `system/register.go`

### cqrs-htmx (6 files, uncommitted)

- `go.work` (+4 lines: 2 replace directives)
- `go.work.sum` (checksum updates)
- `systemadapter/go.mod` (sqliteengine promoted indirect→direct)
- `systemadapter/systemadapter_test.go` (+1 line: blank import)
- `AGENTS.md` (+1 sentence documenting new replaces)
- `examples/system-demo/system-demo` (21MB binary — MUST BE TRASHED)
