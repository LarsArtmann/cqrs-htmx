# Status — issue #41 closed in code: the `huma/v4` adapter, its gates, and what I got wrong

**Date:** 2026-11-06, 09:27–10:32 CET
**Repo:** `/home/lars/projects/cqrs-htmx`, branch `master`. The auto-commit daemon committed this session's work incrementally (heuristic commits), so the narrative below cites files and measurements rather than one commit hash.
**Scope of this session:** read and execute [cqrs-htmx#41](https://github.com/LarsArtmann/cqrs-htmx/issues/41) ("Huma-typed consumers cannot reach the App dispatch pipeline"): design and ship the `huma/v4` adapter module, wire it into every gate, document it, verify it, and report honestly on the residue.
**One-sentence verdict:** the issue's first checkbox is delivered as a hermetic, 95.6 %-covered 30th module that changes zero root code and green-lights on every gate that is not already red for someone else's unpublished API; the second deliverable (a consumer who can `go get` it) is blocked on an owner-gated tag, and three design compromises I chose deliberately are listed below as debt rather than hidden.

---

## a) FULLY DONE

### a1. The adapter module exists, and it is the shape the issue asked for

The issue offered two mutually exclusive shapes. I took the first — "optional adapter module (datastar-shaped: own go.mod, depends on huma + root, **no root changes**)" — and explicitly did **not** take the second ("export the pipeline stages from root"), for a shipping reason the issue did not name: a root export would need a root tag before the new module could build hermetically, i.e. the gotcha-30 root-tag-gap window would open on day one and every `GOWORK=off` leg would be red until an owner-gated push. Composing **already-published** root API closed the issue without that window.

Files (14, all new, under `huma/`):

| File                    | Contents                                                                                                                                                                                          |
| ----------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `adapter.go`            | `Config` (dispatchers + `cqrshtmx.PermissionChecker` + `Timeout`), `Adapter`, `New` (refuses a no-dispatcher adapter), `DispatchCommand`/`DispatchQuery` (enrichment + timeout + dispatch), `Authorize` (one decision), `authorizeRequest`, `denialError`, `requireAuthenticated` |
| `register.go`           | `RegisterCommand[I]` (no body, `DefaultStatus` 204/201/202), `RegisterQuery[I,O]` (`respond` maps the dispatch result into the typed output), the stage sequencer, the classified `fail`, and the registration guards |
| `options.go`            | `Option` + `Authorize`/`AuthorizeIn`/`AuthorizeOn[I]`/`RequireAuth`/`ValidateCommand`/`ValidateQuery`/`Guard`/`WithTimeout`, `wrapValidationRejection` (mirrors the root validation stage, context-preserving) |
| `problem.go`            | `ProblemError` (embeds `httperr.StructuredError`, implements `huma.StatusError` + `huma.ContentTypeFilter`), `MapError`, `isStatusError`                                                            |
| `enrich.go`             | the shape-locked `commandOptionApplier`/`queryOptionApplier` twins + `enrichCommand`/`enrichQuery`                                                                                                |
| `doc.go`                | package doc: quick start, identity wiring, and the three deliberate omissions                                                                                                                    |
| `README.md`             | consumer-facing: why, install, quick start, the full option table, the stages-as-callables table, the error document, what it does not do, setup/v4 positioning                                  |
| `LICENSE`               | MIT, matching the other 14 module LICENSE copies                                                                                                                                                  |
| `.golangci.yml`         | the datastar-era adapter config + `exhaustruct_v5`, with wrapcheck's `ignore-package-globs` widened for the "never re-wrap a pipeline error" rule                                                |
| `.cqrs-lint.json`       | `preset: library` + F032 (`no-catalog-import`) disabled with the same written reason as datastar/health                                                                                            |
| `go.mod` / `go.sum`     | module `github.com/larsartmann/cqrs-htmx/huma/v4`, pinning **published** root `v4.16.0`, huma `v2.40.0`, cqrs-lite `command/v4.14.1` + `id/v4.7.4` + `query/v4.10.1`, `go-error-family v0.12.0`     |
| `adapter_test.go` / `stages_test.go` / `enrich_internal_test.go` | 46 test cases (incl. subtests) across the three files                                                                                                       |

### a2. The pipeline is the route builder's pipeline, stage for stage

Authorize → build → **validate** → **guard** → dispatch, each stage's failure mapped by the same `httperr` machinery the HTTP endpoints use:

| Situation                                    | Route pipeline         | Huma adapter           | Proof                                        |
| -------------------------------------------- | ---------------------- | ---------------------- | -------------------------------------------- |
| no authenticated subject, `Authorize`         | 401 `unauthorized`     | 401 `unauthorized`     | `TestAuthorization_Matrix` (8 rows)           |
| policy refuses                               | 403 `forbidden`        | 403 `forbidden`        | same                                          |
| policy engine errors / nil checker           | 503, never a denial    | 503 `enforce_failed` / `enforcer_nil` | same                             |
| `RequireAuth` anonymous                      | 401                    | 401                    | same                                          |
| validator returns a plain error              | 400 `cqrshtmx.validate.failed` | 400, same code, **inner machine context preserved** | `TestValidateCommand_IsARejectionWithTheSharedCode` |
| guard returns `httperr.WithHTTPStatus(err, 418)` | 418                | 418 (an explicit carrier wins) | `TestGuard_RunsAfterBuildAndCanDenyWithAnyStatus` |
| command handler returns rejection/conflict/transient/corruption | 400/409/503/500 | identical | `TestDispatchErrors_MapToTheirFamily` (4 rows) |
| consumer returns `huma.Error422…`             | n/a                    | 422 passed through untouched | `TestBuildErrors_PassHumaStatusErrorsThrough` |
| build returns `(nil, nil)`                   | 500 `ErrDecoderReturnedNil` | 500, same code    | `TestBuildNilCommand_IsAServerFault`          |

Two behaviours that only exist because the tests demanded them:

- **A zero `UserID` is not a subject.** `UserID.String()` on a zero value is the 26-character zero ULID, not `""`; passing it to `cqrshtmx.Decide` turns "nobody is logged in" into a policy denial (403). The adapter checks `IsZero()` first, so anonymous callers get 401 and **never reach the engine** (the matrix asserts `checker.asked == false`). This is the one place the adapter is stricter than root's `authorizeWithRequest`, which assigns `.String()` unconditionally.
- **Validation must not shadow error context.** `httperr.StructuredError` reads `ErrorContext()` from the *outermost* error only, so a bare `errorfamily.WrapRejection` silently drops `form_key`/field names. `wrapValidationRejection` mirrors root's `wrapErrorPreservingContext`, and the test asserts `form_key=url` survives the wrap.

### a3. Errors are one document on both transports

`ProblemError` embeds `httperr.StructuredError`, so a Huma client receives the same RFC 9457 body a JSON endpoint writes: `type` (family), `title`, `status`, `detail` (public-redacted for 5xx), `code` (deepest domain code), `instance` (request ID from the context), plus the `message`/`why`/`fix` extensions. Pinned by `TestMapError_DocumentsMapToProblemDetails` (409 code, 5xx redaction — asserts the stream ID never appears, why/fix present, Huma-status passthrough), `TestProblem_ContentTypes` (`application/problem+json`, `+cbor`, echo for unknown), and `TestRequestID_ReachesTheProblemInstance`. 5xx detail/context redaction is `httperr.NewStructuredErrorWithContext`'s public audience, i.e. the same rule as ADR-0061, not a new policy.

### a4. OpenAPI still belongs to Huma (the issue's third verify item)

`TestOpenAPI_IsStillGeneratedByHuma` registers an operation through the adapter, marshals `api.OpenAPI()` and asserts (a) the path exists, (b) the `ErrorModel` component still exists, (c) the spec still references `components/schemas/ErrorModel` for error responses. The adapter adds no schema and no response entry: it never touches `op.Responses`, so `defineErrors` keeps owning that decision. The one honest caveat is recorded in the README: the body carries `code`/`message`/`why`/`fix` extensions that Huma's `ErrorModel` schema does not declare, which is additive JSON, not a false claim.

### a5. Zero dependencies reached the root module

`grep huma go.mod` → nothing. The root's 19-dependency budget is untouched; `github.com/danielgtaylor/huma/v2` exists only in `huma/go.mod`. `check-dep-budgets.sh` reports `huma: 6 deps (budget: 7)` — the 6 being root, huma, cqrs-lite command/id/query, error-family.

### a6. Every gate that could see the module now sees it

| Surface                          | Change                                                                            | Verified |
| -------------------------------- | --------------------------------------------------------------------------------- | -------- |
| `go.work`                        | `./huma` member (alphabetical, after `./health`)                                   | workspace build of the module green |
| coverage gate                    | `scripts/checks/coverage-thresholds.tsv` row `huma 90` (single source for flake app + CI) | 95.6 % measured, threshold met |
| dep budgets                      | `["huma"]=7` with the 6-current ledger sentence                                    | `✓ All modules within dependency budget` |
| lint (CI matrix)                 | `huma/.golangci.yml` + matrix entry                                                | `0 issues` |
| cqrs-lint                        | module list in `check-cqrs-lint.sh` + `huma/.cqrs-lint.json`                       | 14 candidates ran: huma green (only INFO A009/A025) |
| errorfamily                      | flake `.#errorfamily` `check_module "huma"` + CI scanner args                      | green |
| CI build / test / coverage / tidy | four new steps + `huma` in the `go mod tidy -diff` list                            | file parses as YAML; `goroutine`-free |
| CI tidy-list gap (pre-existing)  | **added the missing `datastar` entry** while editing that line                     | datastar's `go mod tidy -diff` is clean, so this is a safe gap fix |
| dependabot                       | `directory: /huma` + posture comment explaining why huma is the one exception       | YAML parses |
| module isolation                 | auto-discovered from `go.work`                                                     | `github.com/larsartmann/cqrs-htmx/huma/v4 ... OK` |
| version drift / require tags     | auto-discovered                                                                   | `✓ No version drift`, `✓ All 918 larsartmann requires resolve to published tags` |
| skill-claims                     | taught the gate about a new package alias (`cqrsadapter` → `huma`)                 | `220 documented symbol(s) all resolve` |
| prettier                         | `huma/README.md` formatted with the repo's `.prettierrc`                           | `--check` clean |

### a7. Documentation carries the path, not just the API

- `docs/guides/fullstack-wiring.md`: new `### OpenAPI-first APIs: Huma v2 on the same pipeline (the huma/v4 module)` section placed **immediately after** the DataStar route-split section (the issue's third checkbox), with the wiring snippet, the stage order, the identity requirement, the deliberate gaps, and a link from "See Also".
- `README.md`: module-table row, install line, a feature bullet next to the DataStar one, TOC and docs-list entries.
- `AGENTS.md`: a full `huma` bullet in the Architecture module list (options, stages, the two documented divergences, the zero-root-change rationale, the guide pointer), module count 29 → 30, and the dependency-direction sentence (huma → root + Huma v2 + cqrs-lite; nothing depends on it).
- `ROADMAP.md`: open question 13 (the 2026-07-05 "parked until a consumer asks" item) resolved with the demand-signal provenance.
- `CHANGELOG.md` [Unreleased]/Added: the full entry.
- `FEATURES.md`: a `## huma Module` inventory section (7 rows) + the coverage-table row.
- `.agents/skills/cqrs-htmx/SKILL.md`: module-table row, a decision-tree branch in Step 1, a new `## OpenAPI-first JSON APIs (Huma v2)` section (fences parse-checked), and a "Where to look" bullet. `references/gotchas.md`: 29 → 30 modules.
- `docs/PERMISSIONS.md`: the `cqrs-htmx` inventory row now names the adapter as a third enforcement surface ("every Huma operation behind an `Authorize*` option").
- `VERSIONING.md`: the published-module list.
- `TODO_LIST.md`: the owner-gated tag row + the header trued to 30 modules.

### a8. Verification receipts (this session's measurement, not memory)

```
cd huma && GOWORK=off GOEXPERIMENT=jsonv2 go build ./...      → rc 0
cd huma && GOWORK=off GOEXPERIMENT=jsonv2 go vet ./...        → rc 0   (compiles _test.go too — gotcha 2)
cd huma && GOWORK=off GOEXPERIMENT=jsonv2 go test ./... -race → ok, 1.02s
cd huma && golangci-lint run --timeout=5m                     → 0 issues
cd huma && go test -coverprofile                             → 95.6 % of statements (gate 90)
scripts/checks/check-coverage-threshold.sh huma               → huma coverage: 95.6% (threshold: 90%)
scripts/checks/check-dep-budgets.sh                           → huma 6 deps (budget 7) OK
scripts/checks/check-module-isolation.sh                      → huma OK
scripts/checks/check-version-drift.sh --strict                → no drift, 918 tags resolve
scripts/checks/check-require-tags.sh                          → no phantom versions
nix run .#check-cqrs-lint                                     → 14 candidates, huma green
scripts/checks/check-permission-surface.sh                    → no anti-patterns
scripts/checks/check-permission-maturity.sh --ratchet         → 13 services at/above baseline
scripts/checks/check-skill-claims.sh                          → 220 symbols resolve
scripts/checks/check-skill-snippets.sh                        → 44 fences parse (30 signature fences skipped)
scripts/checks/check-readme-splitbrain.sh                     → 104 symbols resolve
scripts/checks/check-readme-quickstart.sh                     → README quick start compiles
scripts/checks/check-docs-freshness.sh / -links.sh / -tail-budget.sh / -corpus-budget.sh → green
scripts/checks/check-todo-hygiene.sh                          → clean (76 rows, 0 dead, preamble 495)
scripts/checks/check-module-layers.sh                         → 4 packages, one-way
scripts/checks/check-reexport-policy.sh --strict              → 47 alias files, 413 decls, unchanged
scripts/checks/check-shellcheck.sh                            → 133 scripts clean
scripts/checks/check-root-files.sh                            → 33 root files, baseline match
scripts/checks/check-conflict-markers.sh                      → 1708 files, no markers
scripts/selftests/test-check-dep-budgets.sh                   → 7 passed
scripts/selftests/test-check-skill-claims.sh                  → 5 passed
```

Composite: `nix run .#check-modules -- --report` → **87 PASS / 7 FAIL of 94 stages**. All seven reds are pre-existing and attributed in §d.

---

## b) PARTIALLY DONE

- **b1. The deliverable a consumer can actually use.** `huma/v4` has **no git tag**, so `go get github.com/larsartmann/cqrs-htmx/huma/v4` fails today. This is a one-command owner action (`git tag huma/v4.0.0 && git push origin huma/v4.0.0`), deliberately not taken because this session does not push. Tracked as a P2 row in `TODO_LIST.md`.
- **b2. Full coverage-gate run.** I measured and gated the huma row. I did **not** run `nix run .#coverage-gate` end to end, because the other sessions' in-flight root/identity-model API means several modules cannot compile hermetically right now — the run would be red for reasons unrelated to this work and would have produced a misleading receipt.
- **b3. `erraudit` / `branching-flow` for the new module.** `nix run .#check-erraudit-strict` was not run for `huma`. `check-branching-flow` ran as part of `check-modules` and is red, but the baseline refresh is documented as another session's settle pass (their TODO note already lists my live files by line), so I did not re-pin it.
- **b4. Prettier across the shared docs I edited.** `huma/README.md` is formatted. `README.md`, `FEATURES.md`, `TODO_LIST.md`, `docs/PERMISSIONS.md`, and the skill file are `prettier --check`-unclean — **including their committed state before this session**, i.e. this tree's markdown is not currently prettier-canonical and no CI step checks it. I deliberately did not run a formatter across files a sibling session is editing (gotcha 53); this is recorded rather than silently "fixed".
- **b5. The `check-modules` tidy-list repair is half a fix.** I added `datastar` (missing since the module was created) because its `go mod tidy -diff` is clean. I did not audit whether any other module is missing from that list; a `go.work`-derived list would be the real fix.
- **b6. Skill/reference depth.** The project skill gained the section and the tables. `references/core-api.md` was left alone (Huma is not core API), but there is no `references/huma.md` — the module README carries that weight for now.

---

## c) NOT STARTED

- **c1.** `examples/huma-demo` (a runnable peer for the 13 existing demos; the module tests are end-to-end via `humago` + `httptest`, which is stronger evidence but less discoverable than `go run .`).
- **c2.** Output-bearing commands (201 + `Location` header, or a response body built from the command). Today that is an escape hatch: use `pipe.DispatchCommand` inside a hand-written Huma handler.
- **c3.** A typed query variant. `RegisterQuery`'s `respond` receives `result any`, so the consumer asserts the type; `Adapter.DispatchQueryTyped[R]` + `QueryTyped[I,Q,R]` would remove that.
- **c4.** An audience/operator knob for `MapError` (ADR-0061's `AudienceOperator`, so an internal Huma API can answer 5xx with raw detail instead of redacted detail).
- **c5.** A Huma `Middleware` helper that installs the adapter's three standard behaviours (authorize-from-context, classified logging, problem mapping) as `huma.UseMiddleware`, for consumers who register operations by hand.
- **c6.** `authztest.Suite` coverage for Huma operations — the proof kit proves route postures and dispatch postures; an adapter-shaped suite (`authztest.HumaSuite`?) does not exist.
- **c7.** A route-posture-style manifest for Huma operations (the L4 gate derives manifests from route tables; Huma ops declare their posture only as an option, so nothing pins them).
- **c8.** `flake.nix` per-module convenience apps (`.#test-huma`, `.#build-huma`). The existing set is partial (no datastar/setup/health/auditlog/systemadapter/identity-model entries), so I did not extend it rather than make the inconsistency mine.

---

## d) TOTALLY FUCKED UP (and: the reds that are not mine)

**My own defects, in order of how much they would have cost a consumer:**

1. **The zero-ULID subject bug — caught only by the test matrix.** My first `authorizeRequest` assigned `UserIDFromContext(ctx).String()` unconditionally, so an anonymous caller was asked about subject `00000000000000000000000000`, got `no_matching_policy`, and received **403 instead of 401** — the classic fail-wrong-shape authz bug, shipped for ~3 minutes. Fixed with an `IsZero()` guard; the matrix now asserts both the status and that the engine was never consulted. Lesson recorded in the module's `denialError` comment.
2. **The context-shadowing validation wrap — caught only by an assertion I nearly dropped.** I wrote `errorfamily.WrapRejection(err, code, msg)` by analogy with root's code before noticing root has a *specific helper* (`wrapErrorPreservingContext`) for exactly this. Without the fix, every Huma validator failure would have lost `form_key`/field context, i.e. the client could show prose but not highlight the bad input — a silent UX regression versus the route path.
3. **A false claim in a doc comment.** `enrich.go` cited `TestCommandOptionApplier_BasicCommandStillSatisfies` as the drift guard — a test that did not exist. I wrote the test (`enrich_internal_test.go`) rather than delete the claim, because the claim was right about what the code needs.
4. **Dead code shipped, then removed.** `authorize`'s `switch` carried a `default: return nil` for an enum with three exhaustive cases (measured 81.8 % branch coverage). Removed; the `switch` now falls through to a single `return nil`.
5. **`requireBuild[T any]` was a broken guard.** `any(build) == nil` is false for a nil func value in an interface (it holds a type), so the panic never fired — the self-test caught it. Replaced with direct typed nil checks per call site.
6. **Dep-budget arithmetic was wrong twice.** I wrote the comment "5 current" for 6 deps and budget 6; then tidy promoted `id/v4` from indirect to direct. The gate only passes because I re-read the number.
7. **`FEATURES.md` test count was wrong within the hour.** I wrote "18 tests across 2 files"; by the time I was done there were 46 cases across 3 files (the drift guard added a file). Fixed, but it is exactly the class of claim that rots — the file's own preamble says counts are gate-derived for this reason.

**Reds in the composite that are NOT mine (each verified by name):**

| Failing stage                | Root cause (verified, not assumed)                                                                                                                |
| ---------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------- |
| `module-isolation`           | `dashboardui` needs `cqrshtmx.DropStats`/`NewDropStats`/`NewBroadcaster(opts)`, `setup` needs `adminui.NavEntry`/`ExtraNav`/`DashboardURL`, `usermgmt` needs `identitymodel.EventSessionStarted`/`SessionEnded`/`SessionEndCause` + `cqrshtmx.DispatchPosture`, `systemadapter` + `integration_test` inherit them — all **unpublished** API from sibling sessions (gotcha-30 tag gap). `huma` itself: **OK**. |
| `module-isolation --examples` | same class plus the published `catalog/v4@v4.12.0` docserver templ-generator poison.                                                                |
| `workspace-build`            | aborts on `examples/catalog-demo` → `catalog/v4@v4.12.0/docserver`: `undefined: templ.JoinURLErrs`/`ResolveAttributeValue` (upstream release needed). |
| `streamid-identity`          | pre-existing (`dashboardui/core/command_fates.go`, already a plan item in the sibling TODO).                                                        |
| `branching-flow`             | baseline churn from concurrent sessions (+7/−3/~169); the sibling TODO row already names my live files by line and defers the re-pin to their settle pass. |
| `route-postures` (+ self-test) | `setup`/`adminui`/`dashboardui` hermetic compile failures = the same tag gap.                                                                     |
| cqrs-lint `.`, `usermgmt`, `dashboardui`, `systemadapter` | strict-load errors on the same unpublished symbols. `huma` green.                                                                |

I did **not** touch any of those files, per the "never revert changes you did not author" rule; the correct fix for six of the seven is a release train (a sibling lane), and for the seventh an upstream release.

**One process hazard observed, not caused:** the auto-commit daemon committed this session's files mid-flight (heuristic commits), so `git status` was regularly empty and a stash (`stash@{0}: Git Town WIP`) exists that is not mine. I never stashed/restored/checked out anything.

---

## e) WHAT WE SHOULD IMPROVE

1. **The issue's option 2 was the wrong option, and the issue did not say why.** Any "export the pipeline stages from root" answer in this repo is tag-blocked, because a new module can only pin *published* root API. Future issues that propose a root export should carry the tag cost explicitly. Worth a line in `docs/gotchas.md` (gotcha 30 extension) — the adapter shape is the pattern to copy.
2. **Two small pieces of pipeline knowledge are now triplicated.** The denial shape (`errorfamily.NewRejection(code(sentinel)).WithCause(sentinel)`) and the validation wrap (`WrapRejection` + context mirroring) exist in root, in `usermgmt` (`audit_context` era shapes) and now in `huma`. Each copy has a comment pointing at its twin, and the enrichment twin has a test — but the other two do not. Either root exports `Decision.DenialError`/`ValidateCommandValue` (breaking the "no root changes" premise and needing a tag) or each twin gains a shape guard. I chose duplication + one guard; that is a compromise, not a solution.
3. **`validationCode` is a copied string constant.** If the root ever renames `cqrshtmx.validate.failed`, the Huma transport silently diverges and the only signal is a test in the wrong module. A cross-module drift test (root package asserting the adapter's constant) is impossible without a dependency in the wrong direction — so the honest fix is for the code to be exported from a place both can import, or for a `check-*` gate to grep both values.
4. **`RegisterQuery`'s `result any` is the weakest signature in the module.** It mirrors `query.Dispatcher.Dispatch`, but the module already imports the query package and could offer `query.DispatchTyped` inside a typed variant. Right now a consumer writes a type assertion that can fail at runtime in a code path the compiler could have checked.
5. **The adapter has no hooks seam.** `BeforeDispatch`/`AfterDispatch` are `*http.Request`-shaped, so Huma consumers lose per-dispatch tracing/metrics wrapping unless they re-derive it. A `WithBeforeDispatch(func(ctx) (ctx, error))` option would be a small, honest addition — deliberately not invented without a consumer.
6. **No `Location`/response-header story for creates.** 201 with a bare body is the common OpenAPI shape; today the module cannot express the `Location` header.
7. **The module's docs live only in its README.** The repo's pattern for deep modules is a `docs/guides/*.md` of their own (datastar has two). One guide, `docs/guides/huma-adapter.md`, would keep `fullstack-wiring.md` (already 420+ lines) from becoming the only home.
8. **`FEATURES.md` and `CHANGELOG.md` are shared hot files in a multi-session tree.** Both were `M` while I edited them. Row-level edits survived, but the pattern (gotcha 47/53) will eventually bite; a "sibling session holds this file" probe before editing large living docs would help.
9. **The `go mod tidy` CI list is hand-maintained and already drifted once** (datastar missing). Derive it from `go.work`.
10. **Nothing pins a Huma operation's posture.** The L4 route manifests are generated from tables and golden-pinned; the adapter's `Authorize*` options are invisible to `check-route-postures`. A consumer could ship 40 ungated operations and every gate would stay green.

---

## f) UP TO 50 THINGS TO GET DONE NEXT

**Delivery of this work**

1. Tag `huma/v4.0.0` and push (owner-gated); then `nix run .#verify-remote-tags` and a real `go get` from a throwaway module.
2. Add the `huma/v4` row to the release-train playbook so the next train re-tags it when its root pin moves.
3. Post the issue #41 closure comment linking `huma/README.md` + the guide section (owner-gated).
4. Re-pin the `huma` root require to the next root tag when the sibling train lands (v4.17.0), and re-run the hermetic legs.

**Adapter depth (in the order I would do them)**

5. `RegisterQuery` typed variant using `query.DispatchTyped[R]` (kills the `result any` assertion).
6. `Adapter.DispatchQueryTyped[R]` exported for hand-written handlers.
7. `RegisterCommand` output support (`respond func(ctx, *I) (*O, error)` variant) so 201 + body + `Location` are expressible.
8. `WithHeaders(func(ctx, *I) http.Header)` or the Huma-native `huma.HeadersError` passthrough documented + tested.
9. `WithBeforeDispatch(func(context.Context) (context.Context, error))` + `WithAfterDispatch(func(context.Context, error))` options.
10. `MapErrorWithAudience(ctx, err, httperr.Audience)` so internal APIs can answer raw 5xx detail (ADR-0061 parity).
11. A `Logger(*slog.Logger)` option instead of `slog.Default()` inside `fail`.
12. Consider `RequireAuthIn(domain)` / a `Subject` override option for bot/service callers.
13. A `huma.UseMiddleware`-shaped helper (`Adapter.Middleware()`) for hand-registered operations.
14. Split `MapError`'s status derivation from the body building, so a consumer can reuse the status alone.
15. Decide whether the module should expose `ProblemError.Retryable` (go-error-family carries it; the route JSON handler does not emit it today).

**Proof and coverage**

16. `authztest` suite for Huma operations (401/403/200 matrix by posture, fail-closed `Verify`).
17. A route-posture-style manifest for Huma ops + a gate that flags an operation registered with no `Authorize*`/`RequireAuth` (the `ungated` posture must be explicit).
18. A cross-module drift test for `validationCode` (or a `check-*` grep gate).
19. Benchmark the adapter's per-request overhead vs `App.Command` (`b.Loop()` + `ReportAllocs`, per the Benchmarks section) — claims about "same pipeline" should have a number.
20. An `examples/huma-demo` (runnable, ~40 lines, one command + one query + one gated op) to peer the other 13 demos.
21. A fuzz test for `AuthorizeOn`'s input-type assertion path and for nil-input Huma handlers.
22. Coverage polish on `authorize` (81.8 %) — the `resolve` error path has no test today.

**Gates and tooling**

23. Derive the CI `go mod tidy` module list from `go.work` (remove the hand list).
24. Extend `check-module-isolation` to print a per-module summary line (the current tail hides which modules ran).
25. Wire a markdown prettier check into CI (or exclude the files it cannot currently agree on) — the tree is not prettier-canonical today and nothing says so.
26. Consider a `check-new-module.sh` scaffold gate: a new `go.work` member must appear in coverage-thresholds, dep-budgets, cqrs-lint list, CI build/test/lint/coverage, errorfamily scan. This session touched nine files for that reason; the gate would have listed them.
27. Add the created module to `docs/README`-style indexes automatically (or state there is none).
28. Re-run `nix run .#check-modules` after the sibling train and note which of the 7 reds closed.

**Docs truth**

29. `docs/guides/huma-adapter.md` (or fold the README's § API into the guide and shrink the README).
30. A `references/huma.md` for the skill, so `SKILL.md` can stay a cheat sheet.
31. Truth-pass `FEATURES.md`'s metrics table (its footnote still says "All 15 gates PASSED on 2026-09-20" and lint "0 issues across all 15 modules (golangci 2.13.2)" — now 16 gates and 2.15.0).
32. Add the `huma` row to `docs/runbooks/release-*` playbooks (see #2).
33. AGENTS gotcha candidate: "a new module can only pin published API" — the adapter-vs-root-export decision that shaped this issue.
34. Record the zero-ULID-subject trap in `docs/gotchas.md` (it is a library-wide authz trap, not a Huma one).
35. Record the `WrapRejection`-shadows-context trap next to the existing gotcha 24 family (root's helper exists; adapters must mirror it).

**Repo hygiene this session surfaced**

36. Archive pass down to a tail of 3 (13 live reports, budget 3, `TAIL_BASELINE` 13) — every session is now bumping the baseline instead of archiving.
37. `docs/CORPUS_BASELINE` is being bumped by whoever edits docs; consider making the ratchet report the per-directory delta so growth stays visible.
38. The composite's 7 reds should be listed in one place with owners (this report's §d table is the draft).
39. The `stash@{0} Git Town WIP` and `stash@{1} root-split WIP` entries are ancient; someone should decide to drop or apply them.
40. `check-modules` prints only the tail 15 lines per failing stage; a `--verbose` flag that dumps full stage output would have saved three separate re-runs this session.

**Consumer-facing follow-through**

41. Offer `huma/v4` to the browser-history consumer that filed the issue, with the exact snippet from the guide, and collect what still feels hand-rolled.
42. Ask that consumer whether they want `RegisterCommand` output support (#7) — it is the most likely second ask.
43. Evaluate whether `setup` should expose its dispatchers behind an explicit opt-in (the real reason a `setup` consumer cannot use this adapter).
44. Consider an `adminui`-style "adapter posture manifest" so Huma ops appear in `Bundle.RoutePostures()`.
45. Decide whether Huma should be mentioned in `VERSIONING.md`'s release-cadence section (new module = new tag train entry).
46. Watch huma v2's release cadence before pinning it loosely; the adapter compiles against `huma.API`, so minor bumps should be safe — a `check-release-train`-style watch for third-party frameworks does not exist.
47. Add `huma` to the fleet-libs utilization audit's inventory (it is a module with a third-party framework dep, a new category).
48. Consider documenting `ProblemError`'s schema-extension caveat in the OpenAPI guide consumers read.
49. Re-check `check-permission-surface` after any adapter change (it scans for raw engine calls; today zero).
50. Put this report's §f into `TODO_LIST.md` incrementally (rows 1, 5, 6, 17, 20, 26 first — they are the ones a consumer will feel).

---

## g) QUESTIONS I CANNOT ANSWER MYSELF

1. **Do you want `huma/v4` tagged now?** Everything is green and it pins published root `v4.16.0`, so `git tag huma/v4.0.0 && git push origin huma/v4.0.0` is the only step between this work and a consumer who can `go get` it — but pushing is your call, and the alternative is to fold it into the next family train.
2. **Should the next adapter slice be output-bearing commands (`201` + `Location` + body) or the posture manifest/gate?** The first is consumer-visible completeness; the second is the L4 story that currently has a hole (nothing pins a Huma operation's gate). I would pick the manifest if the goal is "no ungated surface", and the output support if the browser-history consumer asks for it first.
3. **How much of the composite's red state belongs to this session's lane?** I kept the tree untouched outside my paths, and six of the seven reds are tag-gap/upstream. If you want me to drive the release train (identity-model → root → usermgmt → adminui → dashboardui → setup, plus sweeps), say so — it is a push-heavy lane that I did not enter on my own.

---

## Verification index (what to re-run, in order, ~4 minutes)

```bash
cd huma && GOWORK=off GOEXPERIMENT=jsonv2 go build ./... && GOWORK=off go vet ./... && go test ./... -race
nix run .#check-module-isolation 2>&1 | grep huma     # huma OK; the rest are the documented tag gap
bash scripts/checks/check-dep-budgets.sh
bash scripts/checks/check-version-drift.sh --strict
nix run .#check-cqrs-lint                             # 14 candidates; huma green
bash scripts/checks/check-skill-claims.sh && bash scripts/checks/check-skill-snippets.sh
bash scripts/checks/check-docs-links.sh && bash scripts/checks/check-docs-corpus-budget.sh --ratchet
bash scripts/checks/check-docs-tail-budget.sh --ratchet
```
