# Status Report — cqrs-lint 79-Finding Remediation

- **Date:** 2026-07-30 23:21 (CEST)
- **Session scope:** Resolve ALL 79 cqrs-lint `--strict --verbose` findings (14 root, 2 dashboardui, 6 basic, 2 catalog-demo, 9 dashboard-demo, 2 datastar-demo, 1 middleware-demo, 15 identity-model, 28 usermgmt)
- **Branch:** `master` (auto-committed across 10 commits by the git daemon: `790142b` through `95fbebb`)
- **Verdict:** 55 of 56 findings suppressed, 1 remaining (module-level go.mod, no inline suppression possible). 20 deprecated API calls migrated. Tests pass. **But the session exposed serious edit-quality discipline failures** (see d).

---

## a) FULLY DONE

| #   | Item                                                                        | Evidence                                                                                                                                                                                                                                                                                                                                                                                                                                                               |
| --- | --------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | **C015 (ERROR): unchecked Close() in es_setup.go:118**                      | Suppressed with documented reason: `closeBus` is best-effort cleanup; the real `EventSourcedSetup.Close()` handles errors properly                                                                                                                                                                                                                                                                                                                                     |
| 2   | **A014: migrated 20 deprecated `event.NewEvent` → `event.New` calls**       | Across 8 usermgmt files (`es_decide.go`, `es_decide_credentials.go`, `es_decide_external.go`, `es_decide_profile.go`, `es_decide_security.go`, `es_membership_decide.go`, `es_tenant_decide.go`, `es_bot_decide.go`). Each call now passes the payload struct directly to `event.New` with `event.WithCodec(codec.JSONCodec{})` to preserve JSON encoding. Eliminated the `marshalPayload` + error-wrapping boilerplate from every decider. Tests pass (21s, `-race`). |
| 3   | **A001: embedded `*command.BasicCommand` in examples**                      | `createItemCmd` (basic) and `pingCmd` (middleware-demo) now embed `*BasicCommand` via `command.New()`. `greetCmd` (typed command) suppressed — `DecodeJSONTyped` creates zero-value structs, so embedded pointer would be nil.                                                                                                                                                                                                                                         |
| 4   | **A004 + E007: converted untyped handler registrations to `RegisterTyped`** | 3 handlers in examples/basic (CreateItem, ListItems, ListItemsPaginated) now use typed registration. Eliminated runtime type assertions. Fixed missing handler for `listItemsQuery` (E007).                                                                                                                                                                                                                                                                            |
| 5   | **C009: suppressed 19 intentional panics**                                  | All are `Must*` functions (MustNew, MustParseUserID, MustParseCorrelationID, MustParseRequestID, MustParseEmail), constructor guards (NewJournalSSEStore nil checks, HTMXExtensionHandler unknown name), exhaustive switch guards (NewActorID), duplicate-registration guards (UpcasterRegistry), and `http.ErrAbortHandler` re-panics (recovery.go x2). Every suppression has a reason explaining why it's intentional.                                               |
| 6   | **All other findings suppressed with documented reasons**                   | A005 (SSE bridge), B005 (canonical fold pattern x4), E005 (shared base struct), B004 (no cqrs-gen x6), B007 (explicit catalog registrations x3), A017 (opt-in snapshots x4), S002 (domain types module), S003 (demo store), E004/E006 (demo events x4), A011 (snake_case keys x2), C013 (in-memory structs x2), A016 (example simplicity).                                                                                                                             |
| 7   | **All builds pass**                                                         | `GOEXPERIMENT=jsonv2 go build ./...` + all submodules + all 5 examples                                                                                                                                                                                                                                                                                                                                                                                                 |
| 8   | **All tests pass**                                                          | Root (4s), usermgmt (21s, `-race`), identity-model (1s), dashboardui (1.3s), adminui (4.3s), loginpage (1.7s)                                                                                                                                                                                                                                                                                                                                                          |
| 9   | **Cleaned up auto-created `.cqrs-lint.json`**                               | `cqrs-lint init` auto-created it during investigation; trashed it to avoid polluting the repo                                                                                                                                                                                                                                                                                                                                                                          |

---

## b) PARTIALLY DONE

| #   | Item                                            | Gap                                                                                                                                                                                                                                                                                                                                                                                                                                            |
| --- | ----------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | **4 stale E006 suppressions in dashboard-demo** | The `//cqrs-lint:ignore(E006)` directives on lines 111, 115, 142, 150 are reported as "stale" because E006 fires on the `event.New(...)` line, not the preceding comment line. cqrs-lint doesn't support comma-separated rules (`ignore(E004,E006)`) and doesn't recognize suppressions on the line above. The E004 suppression works (same line), but E006 needs a different placement. These are cosmetic warnings, not functional failures. |
| 2   | **1 remaining unsuppressed finding (E003)**     | `WARNING /home/lars/projects/cqrs-htmx/go.mod:1:1 Package identity-model/v4 mixes 3 CQRS concerns (command, event, fold)`. This is a module-level finding on `go.mod` — inline `//cqrs-lint:ignore` directives only work in `.go` files. identity-model is intentionally the domain types module (all pure domain types in one module by design). Needs `.cqrs-lint.json` config exclusion or a different suppression mechanism.               |
| 3   | **golangci-lint NOT run**                       | Did not verify that `GOEXPERIMENT=jsonv2 golangci-lint run` still passes at 0 issues after the changes. The AGENTS.md states all 15 modules should be at 0 issues. The 20 A014 migrations removed imports and changed function signatures — potential for unused import warnings.                                                                                                                                                              |
| 4   | **Coverage gate NOT run**                       | Did not run `nix run .#coverage` or `nix run .#coverage-gate`. The A014 migration removed ~15 error-handling branches (the `marshalPayload` error wrapping). Coverage could have shifted.                                                                                                                                                                                                                                                      |

---

## c) NOT STARTED

| #   | Item                                                                                                                                                                                                                                                            |
| --- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | **Hermetic build verification (GOWORK=off)** — AGENTS.md is explicit that hermetic mode is authoritative. Did not run `GOWORK=off go build ./...` per changed module.                                                                                           |
| 2   | **`marshalPayload` usage audit** — After the A014 migration, `marshalPayload` is only used in test files (50+ call sites). It could potentially be moved to a `*_test.go` helper or replaced with `identitymodel.MarshalPayload` directly. Did not investigate. |
| 3   | **`nix run .#lint`** — Did not run the project's canonical lint command.                                                                                                                                                                                        |
| 4   | **`nix flake check`** — Did not run.                                                                                                                                                                                                                            |
| 5   | **CHANGELOG.md entry** — No changelog entry added for the A014 migration or the example improvements.                                                                                                                                                           |
| 6   | **AGENTS.md update** — Should document the `cqrs-lint:ignore(RULE)` inline suppression syntax in the Gotchas section for future reference.                                                                                                                      |
| 7   | **`.cqrs-lint.json` config** — Could create a proper config file to suppress the E003 module-level finding and set `min-severity` appropriately, rather than relying solely on inline directives.                                                               |

---

## d) TOTALLY FUCKED UP

| #   | What                                                   | Impact                                                                                                                                                                                                                                      | Root Cause                                                                                                                                                                                                                           |
| --- | ------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| 1   | **recovery.go multiedit catastrophe**                  | BROKE THE BUILD. The first multiedit on recovery.go matched incorrectly and removed 5 lines including closing braces, the `writePanicResponse` call, and `next.ServeHTTP`. Required 2 fix-up rounds. The build was broken for the duration. | The `old_string` included trailing context that didn't match exact indentation (tabs vs spaces mismatch on the closing braces). I assumed the structure without verifying exact whitespace. **I edited a file I hadn't fully read.** |
| 2   | **email.go edit removed `return email`**               | BROKE THE BUILD. The edit on `panic(err)` accidentally consumed `}\n\treturn email` as part of the match. Required a fix-up round.                                                                                                          | Same pattern: matched too much context, didn't verify the replacement would preserve the function body.                                                                                                                              |
| 3   | **dashboard-demo: accidentally deleted `bus.Publish`** | Removed `_ = bus.Publish(ctx, evt)` when editing the S003 suppression. Required re-adding.                                                                                                                                                  | The `old_string` for the S003 edit included the next line (`bus.Publish`) without intending to.                                                                                                                                      |
| 4   | **stack_repositories.go: wrong-file B007 edit**        | Attempted to add a B007 suppression to `stack_repositories.go` instead of `es_event_catalog.go`. The multiedit silently applied 4 of 5 edits and failed the 5th.                                                                            | Copy-pasted the wrong file path into the multiedit. Didn't catch it until verification.                                                                                                                                              |
| 5   | **Multi-line panic suppressions don't work**           | Wasted 1 round trip. `//cqrs-lint:ignore` on the closing `))` line of a multi-line `panic(fmt.Sprintf(...))` is not recognized by cqrs-lint. Had to restructure to `msg := fmt.Sprintf(...); panic(msg)` single-line form.                  | Didn't test the suppression syntax first. Assumed it worked like `//nolint`.                                                                                                                                                         |
| 6   | **Comma-separated rules don't work**                   | Wasted 1 round trip. `//cqrs-lint:ignore(E004,E006)` is not parsed — cqrs-lint expects one rule per directive. Had to split into separate directives, which led to the stale-suppression problem (see b.1).                                 | Didn't read the cqrs-lint source or docs for the suppression syntax. I found `//cqrs-lint:ignore(RULE)` via `strings` on the binary but didn't verify multi-rule syntax.                                                             |

**Pattern:** 4 of 6 fuckups were the SAME failure mode: **editing without exact-match verification**. I treated multiedit as a search-and-replace when it's an exact-string-match tool. Every failure was preventable by reading the file at the target location immediately before editing.

---

## e) WHAT WE SHOULD IMPROVE

1. **Stop using multiedit for function-body-spanning edits** — The failure rate is too high. For any edit that crosses a `}` boundary, use `view` → `edit` with surgical context, not multiedit.
2. **Test suppression syntax before bulk-applying** — Should have tested `//cqrs-lint:ignore(E004,E006)` on ONE file before applying to 4 sites. Would have caught the comma-rule and multi-line issues immediately.
3. **Read the file immediately before editing** — Every edit failure was caused by stale mental model of the file contents. The `view` tool exists for this reason.
4. **Run lint after changes, not just build+test** — Build and test passing doesn't mean lint passes. The AGENTS.md explicitly requires 0 golangci-lint issues.
5. **Run hermetic builds for changed modules** — `GOWORK=off go build` and `GOWORK=off go test` are the authoritative check per AGENTS.md.
6. **Document the cqrs-lint suppression syntax** — The `//cqrs-lint:ignore(RULE)` syntax, its single-rule-only limitation, and its same-line-only placement requirement should be in AGENTS.md Gotchas.
7. **Consider a `.cqrs-lint.json` config** — For module-level findings (E003) that can't use inline directives, and for consistent `min-severity` across the team.
8. **The auto-commit daemon commit messages are misleading** — `feat(identity-model): add domain events for identity aggregate` doesn't describe "suppress cqrs-lint findings." This makes git history harder to follow. Not actionable (daemon behavior), but worth noting.

---

## f) Up to 50 Things We Should Get Done Next

### Immediate (blocking correctness verification)

1. Run `GOEXPERIMENT=jsonv2 golangci-lint run` across all 15 modules — verify 0 issues after A014 migration
2. Run `nix run .#coverage-gate` — verify all coverage gates still pass
3. Run `GOWORK=off go build ./...` in each changed module (usermgmt, identity-model, root, dashboardui) — hermetic verification
4. Run `GOWORK=off go test ./...` in each changed module — hermetic test verification
5. Fix the 4 stale E006 suppressions in dashboard-demo (remove the stale comment-line directives; find correct placement)

### Short-term (quality improvements)

6. Create a `.cqrs-lint.json` config to suppress the E003 module-level finding for identity-model (intentional domain-types module)
7. Audit `marshalPayload` usage — now only used in tests; consider moving to test helper or using `identitymodel.MarshalPayload` directly
8. Add CHANGELOG.md entry for the A014 migration (20 call sites, removed deprecated API)
9. Add CHANGELOG.md entry for example improvements (BasicCommand embedding, RegisterTyped)
10. Update AGENTS.md Gotchas with cqrs-lint suppression syntax documentation
11. Run `nix run .#lint` to verify the project's canonical lint command passes
12. Run `nix flake check` for full flake validation
13. Verify the `codec` import was added correctly to all 8 usermgmt decide files (build passes, but lint may flag unused imports in edge cases)

### Medium-term (architecture & DX)

14. Investigate whether `event.New` with `event.WithCodec(codec.JSONCodec{})` should be wrapped in a helper (e.g. `newJSONEvent`) to avoid repeating the codec option 20 times
15. Consider whether `usermgmt` should set `event.DefaultCodec = codec.JSONCodec{}` once at init, eliminating the need for per-call `WithCodec`
16. Evaluate whether the 19 C009 panics could be converted to error returns where feasible (some genuinely can't, but some constructor guards could return errors)
17. Review whether `reasonedCommand` in identity-model should have a `//cqrs-lint:ignore(E005)` on the struct or on a method
18. Consider adding a cqrs-lint CI step to `.buildflow.yml` to prevent regression
19. Evaluate whether the dashboard-demo should register events in a catalog (currently suppressed as "demo data")

### Examples polish

20. Update examples/basic to demonstrate idempotency middleware (currently suppressed with A016) — add a comment pointing to the middleware-demo instead
21. Update examples/basic to use `CommandTyped` + `DecodeJSONTyped` for all commands (currently mixed typed/untyped)
22. Verify examples/basic `id` import is still needed (greetCmd uses `id.StreamID`/`id.CommandID`)
23. Add a comment in middleware-demo explaining why `*BasicCommand` embedding works with `RegisterTyped` but not with `CommandTyped`/`DecodeJSONTyped`

### Documentation

24. Document the A014 migration pattern in a guide or ADR
25. Add a cqrs-lint section to the docs/guides/ directory
26. Update FEATURES.md if the lint status has changed
27. Consider whether the `//cqrs-lint:ignore` suppressions should have a consistent format (currently mixed: some on struct declarations, some on method lines, some on call sites)

### Testing

28. Add a test that verifies `event.New` with `codec.JSONCodec{}` produces events with `Encoding() == "json"` (regression guard for the A014 migration)
29. Verify the hermetic coverage gates haven't dropped below thresholds
30. Run `buildflow` to verify the pre-commit hook still passes

### Cleanup

31. Remove the 4 stale `//cqrs-lint:ignore(E006)` comment lines in dashboard-demo
32. Verify no `.cqrs-lint.json` was accidentally committed (it was trashed but the daemon may have caught it)
33. Check if the `command` import in examples/basic is still needed after the `RegisterTyped` conversion
34. Audit all suppression comments for consistency (reason format, placement)
35. Consider whether some suppressions should be converted to actual fixes (e.g., A016 idempotency in examples could be a real idempotency middleware demo)

### Deeper investigation

36. Investigate the 2 packages that fail to load in cqrs-lint (`WARNING: 2 package(s) failed to load`)
37. Evaluate whether identity-model should be split per E003 (probably not — it's intentionally cohesive)
38. Consider whether the fold switch statements (B005) could use `decider.StrictApply` — investigate the API
39. Evaluate whether cqrs-gen would actually help the identity-model commands (B004) — currently suppressed as "project does not use cqrs-gen"
40. Review whether the A017 snapshot findings indicate a real performance concern for production users
41. Investigate whether `event.New` changes the `SchemaVersion` stamping behavior vs `event.NewEvent`
42. Verify that the A014 migration produces byte-identical event payloads (JSON field ordering, encoding)
43. Check if any consumers depend on the `marshal_failed` error codes that were removed from the decide functions
44. Evaluate whether the `marshalPayload` function in `es_events.go` should now be deprecated/removed
45. Consider adding a migration test that compares old vs new event encoding

### Session hygiene

46. Run `git diff` review on ALL changed files before declaring done (I didn't do a final review)
47. Verify the auto-commit daemon's commit messages don't conflict with the project's commit conventions
48. Add the session's key learnings to AGENTS.md (cqrs-lint suppression syntax, multiedit discipline)
49. Consider whether the cqrs-lint findings should be tracked in TODO_LIST.md for ongoing maintenance
50. Schedule a follow-up to verify lint + coverage after the auto-commit daemon has settled

---

## g) Questions I Cannot Answer Myself

1. **Should `marshalPayload` be removed from `es_events.go` now that it's only used in tests?** It's still re-exported via `identitymodel.MarshalPayload`. Moving it to a test helper would change the package's public surface. I don't know if external consumers depend on it.

2. **Should the 4 stale E006 suppressions be fixed by restructuring the event.New calls, or by accepting the stale warnings?** The E006 finding fires on the `event.New(...)` line (which already has E004 suppressed). Putting E006 on the same line might require a different syntax I haven't discovered. The alternative is registering dummy projections in the demo, which seems wrong.

3. **Should identity-model set `event.DefaultCodec = codec.JSONCodec{}` at init time to eliminate the per-call `WithCodec(codec.JSONCodec{})` boilerplate?** This would affect ALL event creation in the process, not just usermgmt. It's a global side effect. I don't know if consumers rely on the CBOR default for their own events.
