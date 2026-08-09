# Session Status: Dependency Upgrade + go-cqrs-lite v3.7.0 Feature Adoption

**Date:** 2026-07-07 02:57\
**Session scope:** Upgrade ALL dependencies to latest, then audit go-cqrs-lite v3.7.0 feature adoption\
**Branch:** master

---

## a) FULLY DONE ✅

### 1. Comprehensive dependency upgrade (all 11 Go modules)

| Dependency                   | Was                   | Now                  | Scope                                                     |
| ---------------------------- | --------------------- | -------------------- | --------------------------------------------------------- |
| go-cqrs-lite (28 submodules) | v3.5.0                | **v3.7.0**           | All modules (otel/v3 stays v3.5.0 — not published higher) |
| go-error-family              | v0.6.0/v0.6.1 (mixed) | **v0.6.1** (unified) | All 11 modules                                            |
| httputil                     | v0.4.0                | **v0.5.0**           | Root, usermgmt, adminui, integration_test, admin-demo     |
| templ-components             | v0.6.1                | **v0.9.0**           | adminui, admin-demo                                       |
| ginkgo/v2                    | v2.31.0               | **v2.32.0**          | Root                                                      |
| gomega                       | v1.42.0               | **v1.42.1**          | Root                                                      |

- **go-cqrs-lite publishing bug solved**: v3.7.0 go.mod files reference internal siblings with zero pseudo-versions (`v3.0.0-00010101000000-000000000000`). Fixed by explicitly pinning ALL 28 go-cqrs-lite modules via `go get` in every consumer go.mod.
- **`.vendor-local/eventtest/go.mod` updated**: Was causing version drift (still had v3.5.0 + v0.6.0). Now in sync with v3.7.0 + v0.6.1.
- **eventtest replace directive**: Added to ALL module go.mod files (not just go.work) — `go mod tidy` under GOWORK=off doesn't apply go.work replaces.

### 2. Feature adoption: `dedup.Ring` (memory improvement)

- **File**: `usermgmt/es_projection_setup.go`
- **What**: Replaced unbounded `map[id.EventID]struct{}` with `dedup.NewRing(dedup.DefaultCapacity)` (1024 entries, ~90KB fixed)
- **Why**: The old map grew to O(journal size) — a 1M-event journal loaded 1M IDs permanently into memory. The Ring is O(1) memory regardless of journal size.
- **Tests**: Updated `es_projection_setup_test.go` — 3 test functions updated to use `*dedup.Ring` instead of `map[id.EventID]struct{}`

### 3. Feature adoption: `codec.ForEncoding` (CBOR compatibility)

- **File**: `usermgmt/es_state.go`
- **What**: `unmarshalPayload` now resolves codec per-event via `codec.ForEncoding(evt.Encoding())` instead of hardcoded `json.Unmarshal`
- **Why**: Consumers who set `event.DefaultCodec = codec.CBORCodec{}` get transparent support. Mixed JSON+CBOR event streams decode correctly per-event.
- **Removed**: `encoding/json` import (no longer needed in this file)

### 4. Documentation updates

- `AGENTS.md`: Updated all version references (go-cqrs-lite v3.7.0, go-error-family v0.6.1, httputil v0.5.0)
- `AGENTS.md`: Updated eventtest gotcha note (now in ALL go.mod files, not just go.work)
- `AGENTS.md`: Updated go-cqrs-lite version note with v3.7.0 publishing bug warning
- `AGENTS.md`: Updated lint discrepancy note (exhaustive + maintidx, not wrapcheck)
- `AGENTS.md`: Added comprehensive go-cqrs-lite v3.7.0 adoption audit (what was adopted, what was intentionally not)

### 5. Verification (all green)

- `nix run .#test`: All 7 test modules pass with `-race`
- `nix run .#check-modules`: Isolation, budgets, zero drift, replace directives — all pass
- `branching-flow errorfamily`: 0 violations (root + usermgmt)
- `nix flake check`: Passed
- `golangci-lint run`: Root 0 issues; usermgmt 2 pre-existing (exhaustive + maintidx — not from this session)
- `nix fmt`: 0 files changed

---

## b) PARTIALLY DONE ⚠️

### 1. CBOR test coverage — MISSING

- Made `unmarshalPayload` encoding-aware, but **did NOT write a test that actually exercises the CBOR path**. The existing tests all create events with the default JSON codec. There's no test verifying that a CBOR-encoded event round-trips through `unmarshalPayload` correctly.
- **Risk**: The `codec.ForEncoding` call could fail on edge cases (empty encoding field, unknown encoding) and we wouldn't know.

### 2. dedup.Ring capacity justification — NOT BENCHMARKED

- Chose `dedup.DefaultCapacity` (1024) based on go-cqrs-lite's documentation claim that it covers the live channel buffer with 4-10x margin. **Did NOT verify** that our `watermill.EventBus` GoChannel backend has a buffer size that fits within this margin.

### 3. go-cqrs-lite v3.6.0/v3.7.0 CHANGELOG — NOT RESEARCHED

- Could not access the GitHub repo (404). Did not find release notes. We know about `dedup`, `scheduling`, `projectionhost`, `CatchUpSubscriber`, `testutil`, `codec.ForEncoding` from reading source — but **there may be other v3.6.0/v3.7.0 features we missed entirely**.

---

## c) NOT STARTED ⬜

1. **CBOR round-trip test** for `unmarshalPayload` (see Partially Done)
2. **Benchmark** comparing dedup.Ring vs old map for typical journal sizes
3. **Research go-cqrs-lite CHANGELOG** for v3.6.0 and v3.7.0 (repo returned 404)
4. **Check httputil v0.5.0 changelog** — what changed from v0.4.0? We only use `httputil.ClientIP`.
5. **Check templ-components v0.7.0/v0.8.0/v0.9.0 changelogs** — what new features? We only use `icons`.
6. **Fix pre-existing lint issues** (exhaustive switch in service_register.go:109, maintidx in es_readmodel.go:44)

---

## d) TOTALLY FUCKED UP 💥

**Nothing catastrophic.** But here are the honest mistakes:

1. **First `go get` attempt failed badly**: Tried upgrading with workspace mode active — cascading errors about zero pseudo-versions. Should have started with `GOWORK=off` from the beginning since the AGENTS.md literally says submodules need it.

2. **Tried clearing `GOPRIVATE`** to use the proxy instead of direct VCS — didn't help because the proxy serves the same broken go.mod files. Wasted a round-trip.

3. **Forgot `.vendor-local/eventtest/go.mod`** entirely on the first pass. The version drift check caught it, but I should have updated it proactively alongside all the other modules. It's a gotcha documented in AGENTS.md that I didn't internalize until the check failed.

4. **Test file not updated alongside source**: Changed `buildLiveHandler` signature in `es_projection_setup.go` but forgot the test file `es_projection_setup_test.go` existed. Build failed on test compilation. Should have grepped for callers before making the change.

---

## e) WHAT WE SHOULD IMPROVE 🔧

1. **Always grep for callers before changing function signatures** — the test file miss was avoidable
2. **Update `.vendor-local/` as part of the upgrade workflow** — not as an afterthought caught by CI
3. **Write a test for every new code path** — the CBOR-aware `unmarshalPayload` has zero test coverage for the CBOR path
4. **Research changelogs BEFORE upgrading** — we upgraded blind and discovered features by reading source after the fact
5. **The exhaustive lint issue in `service_register.go:109`** has been there for a while and is trivial to fix — should fix on sight
6. **The maintidx issue in `es_readmodel.go:Handle`** (complexity 38) is a real code smell — the 12-case switch should be refactored into a dispatch table

---

## f) NEXT 25 THINGS TO DO 📋

| #  | Priority | Task                                                                                                                                                            |
| -- | -------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1  | **P0**   | Write CBOR round-trip test for `unmarshalPayload` — verify a CBOR-encoded event decodes correctly                                                               |
| 2  | **P0**   | Fix exhaustive lint in `service_register.go:109` — add explicit cases for Transient/Corruption/Infrastructure (or add `//nolint:exhaustive` with justification) |
| 3  | **P1**   | Research go-cqrs-lite v3.6.0 + v3.7.0 release notes/changelog (check git log in the repo, not GitHub web)                                                       |
| 4  | **P1**   | Research httputil v0.5.0 changes — is there anything new beyond ClientIP we should adopt?                                                                       |
| 5  | **P1**   | Research templ-components v0.7.0-v0.9.0 changes — new components/icons available?                                                                               |
| 6  | **P1**   | Refactor `es_readmodel.go:Handle` — extract the 12-case switch into a dispatch table or per-event-type handler map (fixes maintidx)                             |
| 7  | **P2**   | Benchmark dedup.Ring vs old map for typical journal sizes (100, 1K, 10K, 100K events)                                                                           |
| 8  | **P2**   | Verify GoChannel bus buffer size fits within dedup.Ring's 1024 capacity (4-10x margin claim)                                                                    |
| 9  | **P2**   | Consider adopting `testutil.NewCmd` in test files to reduce boilerplate command construction                                                                    |
| 10 | **P2**   | Consider adopting `testutil.CapturingSlogHandler` in projection tests to assert lifecycle logging                                                               |
| 11 | **P2**   | Evaluate `projectionhost` for admin-demo's SQLite setup — could replace manual `StartProjections` with managed lifecycle                                        |
| 12 | **P2**   | Fix gopls hints: `errors.As` → `AsType[*event.Error]` in `service_register.go:120`                                                                              |
| 13 | **P2**   | Fix gopls hints: unnecessary type arguments in scenario test files (6+ files)                                                                                   |
| 14 | **P3**   | Document CBOR adoption path in a new ADR — how consumers opt in, migration steps                                                                                |
| 15 | **P3**   | Consider adding `event.WithCodec(codec.CBORCodec{})` as a config option in `EventSourcedConfig`                                                                 |
| 16 | **P3**   | Evaluate `scheduling.TimerStore` for session expiry — could replace `eviction.go` TTL sweep goroutine                                                           |
| 17 | **P3**   | Consider `stack.TypedRepository` migration — bind command types at compile time across all deciders                                                             |
| 18 | **P3**   | Update `docs/migrations/v3-to-v4.md` with the v3.7.0 eventtest replace directive requirement                                                                    |
| 19 | **P3**   | Add a CI job that checks `.vendor-local/eventtest/go.mod` versions match the root module's go-cqrs-lite versions                                                |
| 20 | **P4**   | Explore `prometheus` package in go-cqrs-lite — metrics integration?                                                                                             |
| 21 | **P4**   | Explore `middleware` package in go-cqrs-lite — command/query middleware?                                                                                        |
| 22 | **P4**   | Consider CBOR as default encoding for new events (performance: ~30-50% smaller than JSON for typical payloads)                                                  |
| 23 | **P4**   | Add integration test exercising CBOR end-to-end: create event with CBOR → store → replay → decode in projection                                                 |
| 24 | **P4**   | Evaluate `stack.QueryAuditMiddleware` for query audit logging in admin-demo                                                                                     |
| 25 | **P4**   | Run coverage gate to verify the new dedup.Ring and codec.ForEncoding paths are covered                                                                          |

---

## g) TOP #1 QUESTION ❓

**How do I access the go-cqrs-lite repository to read the v3.6.0/v3.7.0 CHANGELOG?**

The GitHub URL `https://github.com/LarsArtmann/go-cqrs-lite` returned 404 when I tried to fetch it. I researched features by reading the downloaded module source in `$(go env GOMODCACHE)`, but I may have missed undocumented features, behavioral changes, or deprecations that only appear in release notes or git commit history. Is the repo private? Is there a different URL? Should I clone it locally and read `git log v3.5.0..v3.7.0`?

---

## Session metrics

- **Files modified**: ~25 (11 go.mod + 1 go.work + .vendor-local/eventtest/go.mod + 3 source files + 1 test file + AGENTS.md)
- **Tests**: 886+ tests pass across 7 modules with `-race`
- **New dependencies promoted to direct**: `dedup/v3` and `codec/v3` in usermgmt (was indirect via watermill/event)
- **Lines of code changed**: ~150 (source) + ~200 (go.mod files)
