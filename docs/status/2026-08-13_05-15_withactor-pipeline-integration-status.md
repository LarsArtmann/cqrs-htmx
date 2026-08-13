# Status Report: WithActor Full Pipeline Integration

**Date:** 2026-08-13 05:15
**Session Goal:** Close the metadata gap — make commands and queries carry actor metadata automatically through the dispatch pipeline (not just events)
**Verdict:** MOSTLY DONE — pipeline wired and tested, but verification gaps remain

---

## What Was Done

### Fully Done

1. **go-cqrs-lite: `ApplyOptions` on `*BasicCommand`** (`command/command.go`) — new method enables post-construction metadata enrichment. Options that set already-populated fields overwrite them (documented). Backward-compatible additive API.

2. **go-cqrs-lite: `ApplyOptions` on `*BasicQuery`** (`query/query.go`) — mirror of the command version.

3. **go-cqrs-lite builds and tests pass** — `go build ./command/... ./query/...` and `go test ./command/... ./query/...` both clean.

4. **cqrs-htmx: Pipeline enrichment wired into ALL 4 dispatch paths** (`handler.go`):
   - `handleCommandDispatch` (untyped command) → calls `enrichCommandFromContext(ctx, cmd)` before `Dispatch`
   - `handleQueryDispatch` (untyped query) → calls `enrichQueryFromContext(ctx, qry)` before `Dispatch`
   - `handleCommandTypedDispatch` (typed command) → calls `enrichCommandFromContext(ctx, q)` before `Dispatch`
   - `handleQueryTypedDispatch` (typed query) → calls `enrichQueryFromContext(ctx, q)` before `DispatchTyped`

5. **cqrs-htmx: `enrichCommandFromContext` and `enrichQueryFromContext` helpers** (`handler.go`) — type-assert to `*BasicCommand`/`*BasicQuery`, call `ApplyOptions` with `CommandOptionsFromContext(ctx)`/`QueryOptionsFromContext(ctx)`. Silently skips custom Command/Query implementations (not `*BasicCommand`).

6. **cqrs-htmx: 4 new pipeline integration tests** (`context_actor_test.go`):
   - `TestDispatchPipeline_CommandCarriesActorMetadata` — full HTTP → decode → enrich → dispatch, verifies command carries ActorID (kind=ActorUser) + UserID
   - `TestDispatchPipeline_QueryCarriesActorMetadata` — same for query path
   - `TestDispatchPipeline_CommandNoActorForAnonymousRequest` — anonymous request → zero ActorID (no enrichment)
   - `TestDispatchPipeline_CommandPreservesDecoderSetMetadata` — decoder pre-sets bot actor, pipeline overwrites with user actor (transport identity takes precedence)

7. **Previous session's work still intact** — `ActorIDFromUser`, auto-derivation in `ContextEnrichmentMiddleware`/`enrichUserID`, `CommandOptionsFromContext`/`QueryOptionsFromContext`, 12 unit tests, AGENTS.md docs.

8. **Root module builds clean, `go vet` clean, all root tests pass with `-race`** (4.04s).

### Partially Done

1. **Verification scope limited to root module.** Full workspace tests (`GOEXPERIMENT=jsonv2 go test ./... -count=1 -race` across all 24 modules) NOT run. The pipeline enrichment changes affect every module that dispatches commands/queries through `cqrshtmx.App` — usermgmt, dashboardui, setup, adminui, integration_test, and examples all use the root module's `App.Command`/`App.Query` handlers. Any of them could have tests that assert on metadata absence or rely on the old behavior (commands carrying zero metadata).

2. **`CommandOptionsFromContext`/`QueryOptionsFromContext` are now called by the pipeline** (no longer dead code), but they were previously tested only in isolation. The pipeline integration tests prove they work end-to-end, but there's no test verifying the helpers produce the exact same options as `EventOptionsFromContext` for the same context (consistency guarantee).

3. **AGENTS.md gotcha entry from previous session** describes `CommandOptionsFromContext`/`QueryOptionsFromContext` as consumer-facing helpers. Now they're pipeline internals. The documentation should be updated to reflect the automatic wiring.

### Not Started

1. **No lint run** — `nix run .#lint` or `GOEXPERIMENT=jsonv2 golangci-lint run` not executed on any module. New code in `handler.go` (enrichment helpers), `context.go` (command/query options), and `context_actor_test.go` (pipeline tests) may trigger lint findings.

2. **No coverage check** — `nix run .#coverage` / `nix run .#coverage-gate` not run.

3. **No cqrs-lint check** — `nix run .#check-cqrs-lint` not run.

4. **No CHANGELOG entry** — AGENTS.md convention says completed work goes to CHANGELOG.md.

5. **go-cqrs-lite tests not fully verified** — only `command/` and `query/` packages tested. Other packages (event, id, metadata, dispatcher, etc.) not re-verified after the `ApplyOptions` addition.

6. **No test for custom Command implementations** — the enrichment silently skips commands that are not `*BasicCommand`. No test verifies this path (e.g., a custom struct implementing `command.Command` but not `*BasicCommand`).

7. **No typed command/query test** — pipeline integration tests cover untyped dispatch only. `handleCommandTypedDispatch` and `handleQueryTypedDispatch` enrichment is wired but not specifically tested.

8. **No doc comment update on `ContextEnrichmentMiddleware`** — still says "extracts the user ID" but now also derives ActorID.

9. **No doc comment update on `enrichUserID`** — still says "extracts the user ID" but now also derives ActorID.

10. **No example/demo update** — examples not updated to show the actor metadata flow.

11. **go-cqrs-lite CHANGELOG not updated** for `ApplyOptions` addition.

### Totally Fucked Up

**Nothing catastrophic.** But there's a subtle design decision in the pipeline enrichment that could surprise consumers:

- **Pipeline enrichment OVERWRITES decoder-set metadata.** If a decoder sets `command.WithActor(botActor)` and the request is authenticated, the pipeline replaces the bot actor with the user actor. This is intentional (transport-scoped identity takes precedence), and there's a test for it, but it's a behavior change from the previous state where commands carried EXACTLY what the decoder set. A consumer who intentionally sets metadata in their decoder and expects it to survive dispatch will have it silently overwritten.

  **This is the correct design** — the pipeline should own request-scoped metadata, not the decoder — but it's a breaking behavior change that's not documented.

---

## What We Should Improve

### Critical (blocks confidence)

1. **Run the full workspace test suite.** The pipeline change touches the hottest path in the library. Every downstream module needs verification.

2. **Run lint + cqrs-lint.** New exports (`ApplyOptions`, `enrichCommandFromContext`, `enrichQueryFromContext`) and new test code may have findings.

3. **Document the overwrite behavior.** `enrichCommandFromContext`/`enrichQueryFromContext` silently overwrites decoder-set metadata. Add a doc comment explaining this is intentional and why.

### Important

4. **Test custom Command/Query enrichment skip.** Verify that a non-`*BasicCommand` implementation dispatches correctly (enrichment silently skipped, no panic).

5. **Test typed dispatch paths.** `handleCommandTypedDispatch` and `handleQueryTypedDispatch` have enrichment wired but no specific test.

6. **Update AGENTS.md** — the WithActor gotcha should describe the pipeline as automatic, not just "consumer-facing helpers."

7. **Add CHANGELOG entry** for both go-cqrs-lite (ApplyOptions) and cqrs-htmx (pipeline enrichment).

8. **Consider whether enrichment should MERGE instead of OVERWRITE.** Currently `ApplyOptions` overwrites. But if a decoder sets a correlation ID and the context also has one, the context wins. This is probably correct (context is the source of truth for request-scoped data), but should be documented.

### Nice to Have

9. **Add a test verifying EventOptionsFromContext, CommandOptionsFromContext, and QueryOptionsFromContext produce consistent metadata** for the same context.

10. **Update ContextEnrichmentMiddleware and enrichUserID doc comments** to mention ActorID derivation.

11. **Consider a guide** (`docs/guides/actor-and-audit-trail.md`) explaining the full chain.

---

## Up to 50 Things to Get Done Next

1. Run full workspace test suite (`GOEXPERIMENT=jsonv2 go test ./... -count=1 -race` across all 24 modules)
2. Run `nix run .#lint` on all 12 lint-checked modules
3. Run `nix run .#check-cqrs-lint`
4. Run `nix run .#coverage` / `nix run .#coverage-gate`
5. Add CHANGELOG entry for cqrs-htmx pipeline enrichment
6. Add CHANGELOG entry for go-cqrs-lite ApplyOptions
7. Update AGENTS.md WithActor gotcha to reflect automatic pipeline wiring (not consumer helpers)
8. Test that custom Command implementations (not *BasicCommand) dispatch without enrichment (no panic)
9. Test typed command dispatch path carries actor metadata
10. Test typed query dispatch path carries actor metadata
11. Update ContextEnrichmentMiddleware doc comment to mention ActorID derivation
12. Update enrichUserID doc comment to mention ActorID derivation
13. Document the overwrite-vs-merge behavior of pipeline enrichment
14. Add a test for EventOptions/CommandOptions/QueryOptions consistency on same context
15. Verify go-cqrs-lite full test suite passes (`go test ./... -count=1`)
16. Update go.work replace comment if needed (ApplyOptions is new unreleased API)
17. Check if usermgmt decide/dispatch functions are affected by pipeline enrichment
18. Check if setup module's dispatch paths are affected
19. Check if integration_test module tests pass with pipeline enrichment
20. Add a test: impersonation + pipeline enrichment (impersonation middleware sets ActorID before enrichment)
21. Consider whether enrichUserID should be renamed to enrichIdentity
22. Run `nix run .#test-fuzz` to verify no fuzz regressions
23. Run `nix run .#test-flake` to verify no flake regressions
24. Run `nix flake check --no-build`
25. Update examples/basic to show actor metadata
26. Add docs/guides/actor-and-audit-trail.md
27. Consider adding ApplyOptions tests in go-cqrs-lite (verify it works after New())
28. Verify go-cqrs-lite CHANGELOG format and add entry
29. Check if the overwrite behavior needs a config flag (probably not — "just works")
30. Verify the pipeline enrichment doesn't double-apply options (middleware sets context, pipeline reads context — single application)
31. Add integration test: full request → command → event → audit log ActorID chain
32. Consider whether query metadata enrichment matters for audit (queries are read-side, may not need actor)
33. Check if the enrichment helpers should be exported (currently unexported — consumers can't opt out)
34. Consider whether enrichment should be configurable per-handler (HandlerOption to disable)
35. Benchmark the enrichment overhead (negligible — type assert + nil check for zero context)
36. Check if datastar module is affected (it has its own dispatch paths via SSE)
37. Check if systemadapter dispatch is affected
38. Verify dashboardui correctly displays actor metadata from commands (not just events)
39. Verify adminui correctly displays actor metadata
40. Consider whether the enrichment should also set CausationID (currently only actor/userID/correlationID/requestID)
41. Add a test for enrichment with all context fields set (actor + userID + correlationID + requestID)
42. Consider whether WithCustomMetadata should be propagated (currently only standard fields)
43. Check if the pipeline enrichment interacts correctly with BeforeDispatchHook
44. Verify that middleware-based enrichment (ContextEnrichmentMiddleware) and pipeline enrichment are complementary (middleware sets context, pipeline reads context)
45. Document that pipeline enrichment is the SECOND layer (middleware sets context → pipeline injects into command/query)
46. Consider whether the enrichment should run before or after ValidateCommand (currently after decode, before dispatch — so validation sees un-enriched command)
47. Check if ValidateCommand needs to see enriched metadata (probably not — validation is domain logic)
48. Consider adding a HandlerOption to control enrichment behavior (opt-out for specific handlers)
49. Review the test names for clarity (TestDispatchPipeline_* is good, but could be more descriptive)
50. Consider a migration note for consumers who manually set metadata in decoders (will be overwritten by pipeline)

---

## Questions I Cannot Answer Myself

1. **Should the pipeline enrichment overwrite decoder-set metadata, or should it merge (only fill in missing fields)?** Currently it overwrites — a decoder that sets `command.WithActor(botActor)` will have it replaced by the authenticated user actor. This is probably correct (transport identity is authoritative), but a consumer who intentionally sets bot/system actors in their decoder for internal commands would be surprised. Should I add a "don't overwrite if already set" flag, or is overwrite-always the right default?

2. **Should the enrichment be configurable per-handler (e.g., `cqrshtmx.SkipEnrichment` HandlerOption)?** Some handlers might dispatch commands on behalf of the system (not the authenticated user). A per-handler opt-out would let those handlers bypass enrichment. But this adds complexity for an edge case — the consumer could also just not use `App.Command` for those cases.

3. **Does go-cqrs-lite need to publish new tags for `ApplyOptions`?** The cqrs-htmx go.work has local replaces, so it works in development. But hermetic GOWORK=off builds will fail until go-cqrs-lite publishes command/v4.6.0+ and query/v4.5.0+ (or whatever versions include `ApplyOptions`). Should I add go.work replace notes for this?
