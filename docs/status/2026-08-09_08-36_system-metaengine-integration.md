# Status: go-cqrs-lite system/ and metaengine/ Integration

**Date:** 2026-08-09 08:36
**Session scope:** Integrate cqrs-htmx with go-cqrs-lite's `system/` composition root and `metaengine/` storage planner
**Status:** MOSTLY DONE — working module, tests, example, and guide shipped. Significant gaps remain.

---

## a) FULLY DONE (shipped, committed, tested)

### 1. go.work replaces for system/ and metaengine/
- Added 9 new replace directives: `system/v4`, `metaengine/v4`, `metaengine/sqliteengine/v4`, `metaengine/projectionadapter/v4`, `metaengine/pebbleengine/v4`, `metaengine/pgengine/v4`, `metaengine/duckdbengine/v4`, `record/v4`, `testutil/pgtestcontainer/v4`.
- All required because these tags have broken zero pseudo-versions (same root cause as existing go-cqrs-lite replaces).
- **Build passes:** `go build ./...` — 23 modules, zero errors.

### 2. usermgmt/system_exports.go — 20 exported decide functions
- All 20 `decide*` functions exported as `Decide*` (e.g., `decideRegisterUser` → `DecideRegisterUser`).
- Needed because `system.RegisterCommand` handlers must return `system.Execute[State](ctx, streamID, streamType, decideFn)` — the decide function must be callable from outside the package.
- These are thin wrappers: `return decideRegisterUser(aggID, ...)`.
- **usermgmt tests pass:** `go test ./usermgmt/ -count=1 -race` — 19s, zero failures.

### 3. systemadapter/ module — core API
Three exported functions:

| Export | Purpose |
|--------|---------|
| `DomainConfig()` | Returns `system.DomainConfig` with all 4 deciders (`RegisterDecider`), all 20 commands (`RegisterCommand` + `Execute[State]`), and `ProjectionTypeDecoder` set to `EventTypeDecoder()` |
| `EventTypeDecoder()` | Returns `*projectionadapter.TypeDecoder` mapping all 21 event types to payload structs via `projectionadapter.Register[P](eventType, sample)` |
| `NewProjectionLayer(sys)` | Creates `projectionhost.Host` from system's `SeekableJournal` + `event.Bus`, registers all 6 usermgmt projections (4 read models + Casbin + AuditLog) |

- **3 tests passing (race-clean):**
  - `TestDomainConfig_RegisterUserEndToEnd` — command → system → projection → read model query
  - `TestEventTypeDecoder_All21EventTypesRegistered` — all 21 types decoded
  - `TestDomainConfig_TenantAndAuditLog` — tenant creation + audit log

### 4. examples/system-demo/
- Runnable demo: `system.New(ctx, systemadapter.DomainConfig(), deployment)` → `NewProjectionLayer` → `Start` → dispatch commands → query read models → print topology/health.
- Builds clean.

### 5. docs/guides/leveraging-system-metaengine.md
- Full integration guide: quick start, deployment configs (memory/SQLite/separate projection engine), introspection, safety checks, lifecycle, metaengine projections.

### 6. AGENTS.md updated
- systemadapter/ added to module list with description.
- system-demo added to examples list.
- New bullet point documenting the system/metaengine integration.

---

## b) PARTIALLY DONE

### 1. flake.nix integration
- `systemadapter/` is in `go.work` but **NOT** added to `flake.nix`.
- The `forEachGoModule` auto-discovery SHOULD pick it up for `build`, `test`, `lint`, `coverage` — but this has NOT been verified.
- `coverage-gate` uses hardcoded thresholds — no gate added for systemadapter.
- `.github/workflows/ci.yml` may need manual update for the new module.

### 2. Lint verification
- Tests pass, build passes, but **lint has NOT been run** (`nix run .#lint`).
- Likely issues: exhaustruct on `DeploymentConfig` literals, testpackage violations, potential wrapcheck on systemadapter re-exports.

### 3. Coverage gate
- No coverage threshold set for systemadapter. Given 3 tests over ~360 LOC of domain_config.go + ~120 LOC projections.go + ~90 LOC type_decoder.go, coverage is probably 70-80%.

---

## c) NOT STARTED

### 1. cqrs-lint integration
- `.cqrs-lint.json` not updated to include systemadapter.
- The module IS a library consumer of go-cqrs-lite, so it should be linted.

### 2. Metaengine fold declarations for usermgmt read models
- The current approach uses the existing `projection.Projection` implementations (readModelCore-based).
- The metaengine path (declaring queries with fold functions → `metaengine.Plan` → auto-planned indexes/ADTs) is documented in the guide but **not implemented**.
- This is the "bigger vision" integration where read models become planned, cost-optimized projections.

### 3. System-based setup.New alternative
- `setup.New()` still wires infrastructure manually. No `setup.NewFromSystem()` exists.
- The systemadapter provides the building blocks, but there's no one-liner that says "give me a setup.Bundle from system.New()".

### 4. Casbin projection integration with system
- The CasbinProjection runs on the ProjectionLayer's host, not the system's internal projection host.
- If a consumer also declares metaengine projections, they'd have TWO projection hosts running.
- No unified host management yet.

### 5. SQLite-backed persistence test
- Tests only use `Driver: "memory"`. No test verifies the SQLite deployment path works end-to-end.

### 6. CI workflow update
- `.github/workflows/ci.yml` likely doesn't know about systemadapter or examples/system-demo.

---

## d) TOTALLY FUCKED UP

### 1. DomainConfig has dead code
- `DomainConfigOption`, `domainConfigBuilder`, `WithProjectionHostOptions`, `WithDomainMiddleware` exist but are **empty stubs** that do nothing. `DomainConfig()` accepts opts but ignores them.
- I started building an options pattern, realized it added no value, and left the dead code instead of cleaning it up.

### 2. No go mod tidy on systemadapter
- `go.mod` was written by hand. `go mod tidy` was never run on it. The indirect deps are likely wrong or missing.

### 3. Example go.mod has manual replaces
- `examples/system-demo/go.mod` has `replace` directives that mirror the workspace, which could conflict with published versions once tags are cut.

---

## e) WHAT WE SHOULD IMPROVE

### Architecture

1. **Remove dead options code** from `DomainConfig()` — either implement the options pattern properly or drop it entirely.
2. **Run `go mod tidy`** on systemadapter/go.mod and examples/system-demo/go.mod.
3. **Unify projection hosts** — if both system internal projections and ProjectionLayer exist, they should share one host or have clearly documented separation.
4. **Add `setup.NewFromSystem()`** — a bridge that creates a `setup.Bundle` from a `*system.System`, so consumers can use system.New() and still get adminui/dashboardui/loginpage.
5. **Export identity-model fold functions for metaengine** — the `FoldUser`, `FoldMembership` etc. in identity-model are already exported, but metaengine fold declarations need event-type-to-insert/update/delete semantics that map to identity-model's event types. A `metaenginequeries` helper package would close this gap.
6. **SQLite integration test** — verify the `Driver: "sqlite"` deployment path works end-to-end.
7. **Register systemadapter in flake.nix** explicitly (or verify forEachGoModule picks it up).
8. **Add systemadapter to CI workflow**.

### Quality

9. **Run lint** (`nix run .#lint`) and fix issues.
10. **Set coverage gate** for systemadapter in flake.nix.
11. **Run `nix run .#check-cqrs-lint`** on systemadapter.
12. **Run `nix run .#check-codegen`** — verify nothing broke.
13. **Run `nix run .#check-templates`** — verify nothing broke.
14. **Run `nix flake check --no-build`**.
15. **Add more tests** — membership commands, bot commands, change email flow, delete user cascade, Casbin policy verification after command dispatch.
16. **Add `WithProjectionHostOptions` properly** — let consumers configure DLQ threshold, max restarts, backoff.
17. **Add `WithCommandMiddleware` properly** — let consumers add logging/retry/circuit-breaker middleware.
18. **Document the `nil` context** in `system.Execute` calls — the system ignores the context passed to Execute (uses its own internal repo); this is correct but non-obvious.

### Documentation

19. **Update `docs/guides/leveraging-go-cqrs-lite.md`** — cross-reference systemadapter.
20. **Update `docs/guides/fullstack-wiring.md`** — show system.New() as an alternative setup path.
21. **Add architecture diagram** showing how systemadapter sits between identity-model and system/.
22. **Update FEATURES.md** — systemadapter is a new module feature.

---

## f) Up to 50 things to do next

1. Run `go mod tidy` on `systemadapter/go.mod`
2. Run `go mod tidy` on `examples/system-demo/go.mod`
3. Run `nix run .#lint` and fix all issues in systemadapter
4. Run `nix run .#test` and verify ALL workspace tests pass (not just key modules)
5. Remove dead `DomainConfigOption` / `domainConfigBuilder` code from domain_config.go
6. Add systemadapter to `.github/workflows/ci.yml`
7. Verify `forEachGoModule` in flake.nix picks up systemadapter automatically
8. Add coverage gate for systemadapter in flake.nix
9. Add systemadapter to `.cqrs-lint.json` config
10. Run `nix run .#check-cqrs-lint` on systemadapter
11. Run `nix run .#check-codegen`
12. Run `nix run .#check-templates`
13. Run `nix flake check --no-build`
14. Add SQLite deployment test in systemadapter_test.go
15. Add membership command test (AddMember → verify read model)
16. Add bot command test (RegisterBot → verify read model)
17. Add Casbin authz test (RegisterUser + AddMember → verify policy enforcement)
18. Add change email flow test
19. Add delete user cascade test
20. Add `WaitForDrain` timeout test (verify it actually waits)
21. Implement `WithProjectionHostOptions` properly (accept `[]projectionhost.HostOption`)
22. Implement `WithCommandMiddleware` properly (accept `[]command.Middleware`)
23. Add `ProjectionLayer.Health()` method proxying `Host.Status()`
24. Add `ProjectionLayer.Lag()` method for lag monitoring
25. Write `setup.NewFromSystem()` bridge in setup module
26. Add `cqrshtmx.NewAppFromSystem()` — create App from system's dispatchers
27. Explore metaengine fold declarations for UserReadModel
28. Explore metaengine fold declarations for TenantReadModel
29. Explore metaengine fold declarations for MembershipReadModel
30. Create `systemadapter/metaenginequeries/` sub-package with pre-built QueryDecls
31. Add `systemadapter.RegisterMetaengineProjections(sys)` that auto-declares + plans
32. Add integration test for SQLite-backed system.New() + ProjectionLayer
33. Add integration test for SQLite-backed + checkpoint persistence
34. Add benchmark test for systemadapter vs manual setup.New() path
35. Update `docs/guides/leveraging-go-cqrs-lite.md` with systemadapter cross-reference
36. Update `docs/guides/fullstack-wiring.md` with system.New() alternative path
37. Update FEATURES.md with systemadapter module
38. Update CHANGELOG.md with systemadapter module entry
39. Update ROADMAP.md — mark system/metaengine integration as done, plan metaengine projections
40. Add systemadapter to cqrs-htmx SKILL.md
41. Verify systemadapter builds with `GOWORK=off` (hermetic nix build)
42. Check if systemadapter needs the `go-cqrs-lite/system/v4` tag published clean (currently broken — using local replace)
43. Check if `record/v4` tag is published clean
44. Add godoc comments to all exported functions in systemadapter (verify completeness)
45. Add `// Deprecated` markers on decide* functions if we want to push consumers to Decide* exports
46. Add `example_test.go` showing the full systemadapter usage as runnable documentation
47. Verify `nix run .#test-fuzz` doesn't break on new module
48. Verify `nix run .#test-flake` doesn't break on new module
49. Consider extracting `EventTypeDecoder()` into identity-model (it's pure domain knowledge, not adapter-specific)
50. Consider exporting `decide*` functions from identity-model directly (instead of usermgmt re-exports) to avoid the deprecation alias chain

---

## g) Questions I CANNOT answer myself

### 1. Should systemadapter depend on usermgmt or identity-model directly?

Currently it imports BOTH: usermgmt for decider/decide/readmodel factories, identity-model for command types and event constants. But identity-model re-exports through usermgmt via deprecated aliases. If this module is meant to be the "clean path forward" (v5 direction per ADR-0047), should it import identity-model directly and avoid usermgmt entirely? That would require moving decider factories and read models into identity-model or a separate infrastructure package. The current approach works but carries the SA1019 deprecation chain.

### 2. Is the two-host architecture (ProjectionLayer + system's internal host) acceptable long-term?

The system creates its own internal `projectionhost.Host` when metaengine projections are declared. The `ProjectionLayer` creates a SECOND host for usermgmt projections. Two hosts means two sets of checkpoint stores, two drain cycles, and potential ordering issues if a consumer declares both metaengine and usermgmt projections. Should I instead register usermgmt projections on the system's internal host (requires accessing it before Start), or is the separation intentional?

### 3. Should we publish the go-cqrs-lite system/metaengine/record tags before or after cutting a systemadapter tag?

The systemadapter go.mod references `system/v4 v4.1.0`, `metaengine/v4 v4.6.0`, `projectionadapter/v4 v4.3.0`, `record/v4 v4.0.0`. These tags exist but their go.mod files have broken zero pseudo-versions for sibling requires. The local replaces mask this. We can't publish systemadapter without either (a) clean go-cqrs-lite tags, or (b) systemadapter carrying its own replace directives (ugly). Which should we prioritize?
