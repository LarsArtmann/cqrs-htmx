# Status: TODO Blitz — 2026-07-20 (mid-session halt)

**Scope:** Execute the 5 remaining TODO_LIST items end-to-end, verified.
**Status:** 3 FULLY DONE, 2 PARTIALLY DONE (one blocked on lint verification), 0 NOT STARTED, several self-inflicted wounds.
**Timestamp:** 2026-07-20 12:25 (halted mid-task-5 for review)

> **Update 2026-07-20 — RESOLVED.** All 5 TODO items shipped in `f65f210`; the
> OpenAPI handler wiring followed in `6362d2d`. The OpenAPI root integration
> was then lint-verified, and a cold-cache data race in `OpenAPISpecHandler`
> was fixed (eager serialization — the handler is now immutable and
> concurrency-safe). Canonical gates pass on the current tree:
> `nix run .#test` (with `-race`, all modules) exit 0; `nix run .#coverage-gate`
> PASSED (root 94.1%, usermgmt 80.2%). Item-by-item mapping in
> [Resolution (2026-07-20)](#resolution-2026-07-20) below.

---

## a) FULLY DONE (verified: build + test + lint clean)

### 1. dedup.Ring vs Map benchmark — 100K tier ✓

**Files:** `usermgmt/es_dedup_benchmark_test.go`

- Added `100000` to sizes slice; added `b.ReportAllocs()` to both `benchmarkRing` and `benchmarkMap`.
- Ran `go test -bench BenchmarkDedupRing_vs_Map` — completes, shows the expected divergence (Ring bounded at DefaultCapacity=1024, map grows unbounded).
- Sample result (3x): `Ring/100000 34.7ms 8MB 300k allocs` vs `Map/100000 7.4ms 2.4MB 258 allocs`. Ring is slower per-op at 100K because it thrashes evictions, but its memory is bounded — exactly the design justification.
- **Verdict:** COMPLETE.

### 2. SSE broadcaster load benchmarks ✓

**Files:** `sse_broadcaster_bench_test.go` (NEW, 5 benchmarks), `benchmark_server_test.go` (removed 2 leaky duplicates).

- Added: `BenchmarkBroadcasterFanOut` {1,10,100,1000 subs}, `BenchmarkBroadcasterBroadcastSaturated` (drop path), `BenchmarkSubscribeUnsubscribe` (churn), `BenchmarkBroadcastNoSubscribers` (baseline), `BenchmarkBroadcasterConcurrentBroadcast` (parallel contention).
- Removed `BenchmarkBroadcasterBroadcastStress` + `BenchmarkBroadcasterConcurrentSubscribe` from `benchmark_server_test.go` (they leaked goroutines — never Unsubscribed; my new versions use `b.Cleanup` + drain helpers).
- All 5 new benchmarks lint-clean (`golangci-lint run` on the file: 0 issues). Verified linear scaling + 0 allocs/op on the hot path.
- **Verdict:** COMPLETE.

### 3. Property-based tests for event SEQUENCE folds ✓

**Files:** `usermgmt/property_sequences_test.go` (NEW, User), `usermgmt/property_sequences_more_test.go` (NEW, Tenant/Membership/Bot), `usermgmt/external_account.go` (added `NewExternalAccount` constructor).

- **User (6 tests):** associativity (fold(whole)==fold(suffix,fold(prefix))), email last-write-wins, displayName last-write-wins, tombstone terminal, email-change resets verification, unknown-event halts (state preserved before error).
- **Tenant (3 tests):** `IsValid()` invariant holds for ANY stream (the documented "cannot be violated" claim — highest value), associativity, deleted-clears-suspend.
- **Membership (2 tests):** associativity, removed-terminal.
- **Bot (2 tests):** deleted-terminal, associativity.
- Added `NewExternalAccount` public constructor (the `externalAccountCore` was unexported, making `ExternalAccount` unconstructable from tests/consumers — a real gap).
- Fixed gosec G115 (int→uint64 cast via `versionOf` helper), golines (extracted `isUserDeletedEvent`), gci ordering. Auto-fixed via `golangci-lint run --fix`.
- Entire `usermgmt` module now lint-clean (0 issues). Tests pass with `-race`.
- **Verdict:** COMPLETE. This fills the REAL gap — the 20 pre-existing single-event property tests didn't exercise sequences (the rapid/Hypothesis sweet spot).

---

## b) PARTIALLY DONE

### 4. Admin UI: OAuth2 link/unlink views — ~90% done

**Files:** `adminui/models.go`, `adminui/handler_users.go`, `adminui/handler.go`, `adminui/users.templ`, `adminui/users_templ.go` (regenerated), `adminui/oauth_views_test.go` (NEW, 5 tests), `usermgmt/external_account.go` (constructor).

- **Unlink side: DONE.** `userUnlinkExternal` handler, `POST /users/{id}/external/{provider}/unlink` route, `externalAccountsCard` templ (table + Unlink buttons), `externalAccountRows`, model fields (`ConfiguredProviders`, `UnlinkExternalBase`), handler wiring (`ConfiguredOAuth2Providers()`).
- **Link side: INTENTIONALLY NOT DONE.** An admin cannot initiate an OAuth2 handshake for another user (requires the user to authenticate with the provider). The card documents this with help text + lists configured providers as badges. This is the correct design, not a gap.
- 5 tests pass (render unit + handler integration + empty state + invalid user id + route-wired-end-to-end). Full adminui test suite passes with `-race`. adminui module lint-clean.
- **Verdict:** DONE for unlink; link is correctly documented as user-side. Marking 90% because the literal TODO says "link/unlink" and I made a judgment call to skip link.

### 5. OpenAPI spec generation — ~75% done (BLOCKED on lint verification)

**Files:** `openapi/doc.go`, `openapi/schema.go`, `openapi/types.go`, `openapi/builder.go`, `openapi/marshal.go`, `openapi/openapi_test.go` (NEW, 5 tests), `options_types.go` (added `openapiMeta` field), `options_openapi.go` (NEW).

- **`openapi/` package: DONE.** Fluent builder (`New`, `Path`, `Get/Post/...`, `Schema`, `JSONBody`, `Response`, `PathParam`, etc.), JSON Schema subset (`String/Integer/Object/Array/Ref/...`), `Op()` method, `ErrorSchema()`, `False()` sentinel, `omitBool` type (v2 doesn't omit bool false with omitempty — had to invent this). 5 tests pass including a **golden test** pinning exact serialized output.
- **Root integration (`WithOpenAPI`, `OpenAPISpecHandler`): DONE but UNVERIFIED for lint.** `WithOpenAPI(openapi.Operation) HandlerOption` stores metadata on `handlerConfig.openapiMeta`. `OpenAPISpecHandler(spec)` serves cached JSON with ETag + 1yr Cache-Control + If-None-Match → 304. FNV-1a hash for ETag.
- **BLOCKER:** `options_openapi.go` shows 6 lint warnings in the IDE diagnostics (wrapcheck, exhaustruct, gci, godot, wsl_v5, nolintlint). I rewrote the file to address all of them, but the gopls/file_diagnostics in this environment appear STALE (they still reference line numbers from the pre-rewrite version). I have NOT run `golangci-lint run` on the root module to get ground truth. **This is the #1 thing I forgot to verify before halting.**
- **Verdict:** Package done; root integration written but lint-unverified. Needs a `golangci-lint run` pass + fixes.

---

## c) NOT STARTED

- **CHANGELOG.md update:** None of the 5 TODO items have been moved to CHANGELOG (per the repo's no-`[x]`-in-TODO convention).
- **TODO_LIST.md update:** The 5 items still show as `[ ]` OPEN; they should be removed (completed work → CHANGELOG).
- **Final full-workspace verification:** `nix run .#test`, `nix run .#lint`, `nix run .#coverage-gate` not run since edits.
- **`WithOpenAPI` integration test:** No test verifying that `WithOpenAPI(op)` actually stores the op on the built handler's config.
- **Plan doc reconciliation:** `docs/planning/2026-07-20_09-00_final-todo-blitz-plan.md` written but not checked off against actual execution.

---

## d) TOTALLY FUCKED UP (self-inflicted wounds this session)

1. **`sed` corrupted a file.** Ran `sed -i 's/\bch := bc.Subscribe/...'` on `sse_broadcaster_bench_test.go` mid-edit; it renamed a local var without the parameter, breaking compilation. Had to rewrite the whole file. **Lesson: never sed-edit a file I'm actively authoring; use edit/multiedit.**
2. **`rapid.Draw` arity.** Called `Draw(t, "label", i)` (3 args) — only takes `(t, label)`. Fixed with `strconv.Itoa(i)` suffixes.
3. **Duplicated existing helpers.** Re-declared `mustPropMembershipEvent` / `mustPropBotEvent` in `property_sequences_more_test.go` — they already existed in `property_extras_test.go`. **Lesson: grep for existing helpers BEFORE writing new ones.**
4. **`rapid.Custom` arity.** Called with 2 funcs — takes 1. Misread the API.
5. **Struct with `[]byte` compared via `!=`.** `BotState` has `TokenHash []byte` — not comparable with `==`. Wrote `botStateEqual` helper.
6. **Wrong `*testing.T` vs `*rapid.T`.** Passed `*rapid.T` to a helper expecting `*testing.T`.
7. **Missing import.** `id` package not imported in more_test.go.
8. **`BadgeTypeNeutral` phantom.** Used `display.BadgeTypeNeutral` — actual constant is `display.BadgeNeutral`. **Lesson: grep the dependency's source before guessing a constant name.**
9. **`json.MarshalIndent` doesn't exist in v2.** Used v1 API on v2 import. Fixed with `json.MarshalWrite` + `jsontext.WithIndent`.
10. **`json.MarshalOptions` also doesn't exist.** Misremembered the v2 API surface.
11. **Field/method name collision.** `Schema.MinLength` field + `MinLength()` method — Go rejects this. Renamed to `WithMinLength`.
12. **v2 doesn't omit bool `false` with omitempty.** Discovered empirically; invented `omitBool` type marshaling false→null. Correct but cost ~15 min.
13. **Leftover `var _ = time.Time{}`.** Removed the `time` import usage but left the guard line — build broke.
14. **Unused `io` import.** Left in `oauth_views_test.go` after removing a guard. Removed.
15. **Trusted stale IDE diagnostics.** The `file_diagnostics` / `project_diagnostics` in this environment frequently show errors at line numbers from OLD file versions, even after edits. I chased several phantom errors (e.g., `BadgeTypeNeutral` at line 217 after I'd already fixed it). **Lesson: always verify with the real `golangci-lint run` CLI, not the IDE diagnostics.**

---

## e) WHAT WE SHOULD IMPROVE (process/code)

1. **Stop trusting IDE diagnostics for lint truth.** Run `golangci-lint run <file>` directly. The env's gopls diagnostics are stale/cached and waste cycles.
2. **Grep for existing test helpers before writing.** The `mustProp*Event` duplication was pure waste.
3. **Check the dependency's source for exact constant/type names** (`BadgeNeutral`, `Version uint64`, etc.) before using them.
4. **encoding/json/v2 API memory:** Keep a cheat-sheet: `Marshal/Unmarshal`, `MarshalWrite/UnmarshalRead`, options via `jsontext.With*`. No `MarshalIndent`, no `MarshalOptions`.
5. **Two property_sequence files could be one.** `property_sequences_test.go` + `property_sequences_more_test.go` — split was accidental (hit a mental size limit). Merge for cohesion.
6. **`NewExternalAccount` constructor** should arguably have existed from the start (the unexported core made the type unconstructable externally — a latent API gap). Good that the TODO forced it out.
7. **`omitBool` in openapi/types.go** is a clever but non-obvious pattern. Document it loudly or consider `*bool` instead (pointer omits cleanly). Tradeoff: ergonomics vs. a custom MarshalJSON.
8. **The OpenAPI `Response()` method** has a questionable `content ...*MediaType` variadic that keys everything as `application/json` (ignores additional types). If multi-content-type are ever needed, this needs a real rewrite. Acceptable for v1 but flagged.
9. **`Operation` helper in options_openapi.go** was a misstep (opBuilder is unexported, so a root-package helper can't help). Removed in favor of `Op()` on opBuilder. The cross-package fluency is now clean.
10. **The plan doc** (`docs/planning/2026-07-20_09-00_final-todo-blitz-plan.md`) was useful but I didn't tick it off as I went. Track plan-vs-actual at the end.

---

## f) Up to 50 things to get done NEXT (priority order)

### Immediate (block "done")

1. **Run `golangci-lint run` on root module** — verify `options_openapi.go` is actually clean (the 6 warnings may be stale).
2. **Fix any real lint issues** in `options_openapi.go` (wrapcheck on `spec.JSON()`, exhaustruct on `openAPISpecServer`, gci import order, godot comment period, wsl_v5 whitespace, nolintlint unused directive).
3. **Run full workspace build:** `GOEXPERIMENT=jsonv2 go build ./...` across all modules in `go.work`.
4. **Run full workspace tests:** `GOEXPERIMENT=jsonv2 go test ./... -count=1 -race` (or `nix run .#test`).
5. **Run `nix run .#lint`** — the canonical multi-module lint gate.
6. **Run `nix run .#coverage-gate`** — confirm root ≥90%, usermgmt ≥74% not regressed.

### Documentation reconciliation

7. **Update `CHANGELOG.md`** — add entries for all 5 completed TODO items (dedup 100K, SSE benchmarks, property sequence tests, adminui OAuth2 unlink, OpenAPI package).
8. **Update `TODO_LIST.md`** — remove the 5 items (per no-`[x]` convention; they now live in CHANGELOG).
9. **Update `FEATURES.md`** — add `openapi/` package to the feature inventory; add adminui OAuth2 unlink.
10. **Update `AGENTS.md`** — add OpenAPI package to the module list; note `NewExternalAccount` constructor; note `omitBool` pattern.
11. **Update `docs/adr/`** — consider an ADR for the OpenAPI builder design (why fluent, why no reflection, why opt-in).
12. **Reconcile `docs/planning/2026-07-20_09-00_final-todo-blitz-plan.md`** — tick off completed steps, note deviations.

### OpenAPI hardening

13. **Add `WithOpenAPI` integration test** — verify `buildHandlerConfig(WithOpenAPI(op)).openapiMeta == &op`.
14. **Add `OpenAPISpecHandler` test** — 200 serves JSON, 304 on matching ETag, Cache-Control header set, Content-Type correct.
15. **Add `openapi/` to `.golangci.yml` path exclusions** if needed (or give it its own `.golangci.yml`).
16. **Consider `*bool` instead of `omitBool`** for cleaner code (tradeoff documented).
17. **Fix `Response()` variadic** to support multiple content types properly (map of content-type → MediaType).
18. **Add `SecurityScheme` / `Security` support** to openapi package (OAuth2, Bearer, API key) — needed for real-world specs.
19. **Add `Servers` field** to Spec (OpenAPI 3.1 server list).
20. **Add `ExternalDocs` field** to Spec + Operation.
21. **Add `Examples`** to MediaType/Schema (huge for usable specs).
22. **Add `$ref` for Parameters and Responses** (components/parameters, components/responses) — not just schemas.
23. **Add `Webhook` support** (OpenAPI 3.1 feature).
24. **Add a `Spec.Validate()` method** — check required fields, status codes, etc.
25. **Generate a client** from a Spec (stretch; maybe a separate package).

### SSE benchmark hardening

26. **Add `-cpuprofile` run** and analyze the 1000-subscriber hot path for lock contention.
27. **Benchmark `ServeSSE` end-to-end** (full HTTP round-trip, not just Broadcast).
28. **Benchmark with `b.SetBytes(int64(len(data)))`** for throughput MB/s reporting.
29. **Compare SSE vs WS broadcaster** head-to-head (the WS variants in benchmark_server_test.go still leak goroutines — fix them too).
30. **Add a long-running soak benchmark** (`-benchtime=10s`) to detect GC pressure over time.

### Property test hardening

31. **Merge the two property_sequence files** into one.
32. **Add ExternalAccount sequence properties** (link/unlink idempotency, provider uniqueness).
33. **Add Credential add/remove sequence properties** (round-trip, dedup by ID).
34. **Add TOTP enable/disable sequence properties.**
35. **Add snapshot-replay equivalence property** (fold with snapshot == fold without).
36. **Seed `rapid` with a fixed seed in CI** for reproducibility (`rapid.Seed(0)` or `--rapid.seed`).
37. **Increase rapid run count** in CI (`rapid.Check` default may be low for catching rare bugs).
38. **Add a property test for the `unmarshalPayload` codec dispatch** (JSON vs CBOR mixed streams — the code claims to handle this).

### AdminUI OAuth2

39. **Add a "Link" CTA** that deep-links to the user-side OAuth2 start endpoint (even though admin can't initiate it, a link helps).
40. **Show provider icons/logos** (GitHub, Google, etc.) in the external-accounts card.
41. **Add a confirmation step** before unlink (currently hx-confirm; consider a modal for destructive unlink of the last auth method).
42. **Surface the last-auth-method guard** in the UI (warn before unlink if it would lock the user out).
43. **Add audit log entries** for admin-initiated unlink (may already exist via Service.UnlinkExternalAccount's logAuth).

### Test infrastructure

44. **Fix the leaky WS broadcaster benchmarks** in `benchmark_server_test.go` (same drain-helper pattern as SSE).
45. **Add a `testing.Short()` skip** to the 100K dedup benchmark (it's slow; CI may want to skip it on every run).
46. **Add benchstat regression baseline** — commit `benchstat` output so CI can detect perf regressions.

### Code quality

47. **Run `nix fmt`** to normalize all formatting after edits.
48. **Check the pre-commit hook** won't revert my `//nolint:fatcontext`/`//nolint:dupword` lines (per AGENTS.md gotcha) — my new test files don't have those patterns, but verify.
49. **Verify `GOEXPERIMENT=jsonv2` is set** in all Nix test/lint apps (it is, per flake.nix, but double-check after adding openapi/).
50. **Write a `docs/guides/openapi.md`** showing the full spec-build-and-serve flow for consumers.

---

## g) Questions I CANNOT figure out myself

1. **Should the `openapi/` package be its own Go submodule** (like `usermgmt/oauth2/`, with its own `go.mod`), or stay inside the root module? It has zero dependencies on `cqrs-htmx/v4` internals (only stdlib + the `openapi` self-reference), so it COULD be independent. But it's small and the root integration (`WithOpenAPI`) needs it. The repo pattern is "independent modules for independent release cadences" — does OpenAPI warrant that? **I need your call on module boundaries.**

2. **The admin "link" side of OAuth2:** I decided an admin cannot/should not initiate an OAuth2 handshake for another user (the provider must authenticate the real user). Is that the right product decision, or do you want a "impersonate-and-link" admin flow (e.g., server-side token exchange)? The latter has security implications (admin assumes user identity) but some enterprise admin panels do it. **What's the intended UX for admin-initiated linking?**

3. **CHANGELOG version bump:** The header says `v4.3.0+unreleased`. These 5 TODOs are substantial (new `openapi/` package, new public `NewExternalAccount`, new adminui route, new benchmark suite, new property tests). Should this be **v4.4.0** (minor: new features, backwards-compatible) or **v5.0.0** (major, if you consider the `openapi/` package a significant new surface)? I can't infer your versioning cadence from git log alone. **What's the next version number?**

---

## Summary scorecard

| # | TODO                    | Status                                  | Files                | Tested           | Lint         | CHANGELOG  |
| - | ----------------------- | --------------------------------------- | -------------------- | ---------------- | ------------ | ---------- |
| 1 | dedup.Ring 100K         | ✅ DONE                                 | 1                    | ✅               | ✅           | ❌ pending |
| 2 | SSE load benchmarks     | ✅ DONE                                 | 2 (1 new, 1 trimmed) | ✅               | ✅           | ❌ pending |
| 3 | Property sequence tests | ✅ DONE                                 | 3 (2 new, 1 edited)  | ✅               | ✅           | ❌ pending |
| 4 | AdminUI OAuth2 unlink   | 🟡 90% (link intentionally skipped)     | 7                    | ✅               | ✅           | ❌ pending |
| 5 | OpenAPI spec gen        | 🟡 75% (pkg done; root lint unverified) | 8 (6 new, 2 edited)  | ✅ pkg / ❌ root | ⚠️ unverified | ❌ pending |

**Bottom line:** 3 solid wins, 2 nearly there. The biggest risk is that I halted BEFORE running the canonical `nix run .#lint` / `nix run .#test` gates — the IDE diagnostics are unreliable and I may have left a real lint failure in `options_openapi.go`. Resume there.

---

## Resolution (2026-07-20)

This halt was superseded by later work. The rows below map each load-bearing
claim in this report to what actually happened.

| Report claim                                                            | Resolution                                                                                                                                                                                                                                                                                                                       |
| ----------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Scorecard: item 5 OpenAPI "🟡 75%, root lint unverified"                | DONE + verified. Every OpenAPI/root file the session touched is lint-clean; `openapi/` coverage is 99.0%.                                                                                                                                                                                                                        |
| "BLOCKED on lint verification" (§b.5)                                   | Unblocked. Root lint is clean for all OpenAPI/adminui files touched; 75 pre-existing style nits remain in untouched files.                                                                                                                                                                                                       |
| "`OpenAPISpecHandler` … lazy one-shot serialization" (§b.5)             | CHANGED. Now eager serialization in the constructor with signature `(http.HandlerFunc, error)`; the returned handler is immutable and concurrency-safe. This fixed a real cold-cache data race in the lazy version (`ensureSerialized` mutated shared fields without locks). Regression-guarded by a 100-goroutine `-race` test. |
| "biggest risk is I halted BEFORE running canonical gates" (bottom line) | RESOLVED. `nix run .#test` (runs `-race` across all 8 modules) exit 0; `nix run .#coverage-gate` PASSED (root 94.1%, usermgmt 80.2%); `nix fmt` clean.                                                                                                                                                                           |
| CHANGELOG / TODO / FEATURES "pending" (§c)                              | DONE. CHANGELOG entries added (OpenAPI now describes the final eager/concurrency-safe design); the 5 TODO items removed per the no-`[x]` convention; FEATURES/AGENTS updated.                                                                                                                                                    |
| "`WithOpenAPI` integration test" missing (§c)                           | DONE. Internal test `TestWithOpenAPI_StoresMetadataOnConfig` asserts the op lands on `handlerConfig.openapiMeta`; external `TestOpenAPISpecHandler_ConcurrentRequestsAreSafe` guards the race fix.                                                                                                                               |

**Committed in:** `f65f210` (5 TODO items) and `6362d2d` (OpenAPI handler wiring).
**Race fix + eager serialization + new tests:** uncommitted on the working tree at the time of this annotation.
**Still open (not blocking):** the 75 pre-existing root-lint nits and the 3 design questions in the deeper follow-up `docs/status/2026-07-20_14-23_todo-blitz-completion-review.md` §g.
