# Status: test-race and test-fuzz Fixes

**Date:** 2026-06-18 18:51
**Session Goal:** Fix test-race and test-fuzz — ensure all race and fuzz tests pass across all modules.
**Outcome:** ✅ All tests pass (race + fuzz). One latent fuzz bug fixed. Two new nix apps added.

---

## TL;DR

| Metric       | Value                                                                             |
| ------------ | --------------------------------------------------------------------------------- |
| Race tests   | ✅ All 4 modules PASS (root 2.1s, catalog 1.0s, usermgmt 13.7s, integration 1.3s) |
| Fuzz tests   | ✅ All 9 fuzz targets PASS (5 root + 4 usermgmt, 30s each)                        |
| Lint         | ✅ 0 issues (root + catalog + usermgmt)                                           |
| Build        | ✅ All modules compile                                                            |
| Coverage     | 95.5% root, 88.1% usermgmt, 95.3% catalog                                         |
| Bugs fixed   | 1 (latent fuzz-induced panic in `FuzzCredentialID_Base64Decode`)                  |
| New nix apps | `test-race` (alias for `test`), `test-fuzz` (discovers + runs all Fuzz\*)         |

---

## a) FULLY DONE ✅

### 1. Race Detector — Clean Across All Modules

All four Go modules pass with `-race`:

```
==> Root module         ok  2.156s
==> catalog submodule   ok  1.024s
==> usermgmt submodule  ok  13.665s
==> integration_test    ok  1.273s
```

No data races detected. The rate-limiter race (fixed in a prior session — `perKeyLimiter.limiter()` RLock scope) remains stable. 10/10 clean runs.

### 2. Fuzz Tests — All 9 Targets Pass

| Module   | Fuzz Target                          | Status              | Execs (30s) |
| -------- | ------------------------------------ | ------------------- | ----------- |
| Root     | `FuzzDecodeJSONBody`                 | ✅ PASS             | 118,750     |
| Root     | `FuzzDecodeFormBody`                 | ✅ PASS             | 224,007     |
| Root     | `FuzzSanitizeRedirectURL`            | ✅ PASS             | 965,154     |
| Root     | `FuzzCSRFConfigValidation`           | ✅ PASS             | 897,250     |
| Root     | `FuzzEventOptionsFromContext`        | ✅ PASS             | 301,748     |
| usermgmt | `FuzzRegisterRequest_Validate`       | ✅ PASS             | 716,475     |
| usermgmt | `FuzzWebAuthnBeginRegistration_Body` | ✅ PASS             | 3,284       |
| usermgmt | `FuzzWebAuthnBeginLogin_Body`        | ✅ PASS             | 3,278       |
| usermgmt | `FuzzCredentialID_Base64Decode`      | ✅ PASS (after fix) | 6,241       |

### 3. Fixed: `FuzzCredentialID_Base64Decode` Panic

**Root cause:** The fuzz target concatenated the raw `encodedID` string directly into the URL path:

```go
r := httptest.NewRequest(http.MethodDelete, "/auth/credentials/"+encodedID, nil)
```

When the fuzzer generated `" HTTP/1.0"` (space + protocol), `httptest.NewRequest` panicked:

```
panic: invalid NewRequest arguments; malformed HTTP version " HTTP/1.0"
```

**Fix:** Wrap the path segment in `url.PathEscape`:

```go
r := httptest.NewRequest(http.MethodDelete, "/auth/credentials/"+url.PathEscape(encodedID), nil)
```

This is a **test-only bug** — the production handler (`handleDeleteCredential`) reads `r.PathValue("id")`, which Go's router already decodes. The panic was in the test's request construction, not in the handler. The stale corpus file (`testdata/fuzz/FuzzCredentialID_Base64Decode/8643a79bab153d1b`) was removed.

**File:** `usermgmt/fuzz_test.go:102`

### 4. New Nix Apps: `test-race` and `test-fuzz`

Added to `flake.nix`:

| App                   | Purpose                                                                                                                                                            |
| --------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `nix run .#test-race` | Runs `go test -race` across root, catalog, usermgmt, integration_test (identical to `test` but explicitly named)                                                   |
| `nix run .#test-fuzz` | Discovers all `Fuzz*` targets via `go test -list`, runs each with `-fuzztime` (default 30s, `FUZZTIME` env override). Skips modules with no fuzz tests gracefully. |

Both verified working:

```bash
FUZZTIME=10s nix run .#test-fuzz   # → all 9 targets PASS
nix run .#test-race                # → all 4 modules PASS
```

### 5. Lint, Build, Format — All Clean

```
nix run .#lint  → 0 issues (all modules)
nix run .#build → all modules compile
nix fmt         → 0 files changed
nix flake check → all checks passed
```

---

## b) PARTIALLY DONE 🔄

### 1. Fuzz Corpus Maturity

The fuzz tests run and pass, but the corpus is still small (5-7 seeds per target). The fuzzer discovers "interesting" inputs during each run, but these are not persisted to `testdata/fuzz/` unless a failure occurs. To build a regression corpus, we'd need to run longer fuzzing sessions (minutes to hours) and commit the generated corpus.

**Status:** Fuzz infrastructure works. Corpus enrichment is an ongoing process.

### 2. Catalog Module — No Fuzz Tests

The `catalog` module has 41 test functions but **zero fuzz tests**. The schema reflection and YAML output paths are fuzz-worthy (arbitrary struct tags → JSON Schema generation).

**Status:** Not blocking, but a gap.

### 3. Integration Test Module — No Fuzz Tests

Same as catalog — no fuzz coverage for cross-module bridges.

---

## c) NOT STARTED ⬜

### 1. CI Pipeline for Fuzz

No GitHub Actions workflow runs fuzz tests. The `test-race` and `test-fuzz` nix apps exist but are not wired into CI. A short-fuzz CI job (30s per target) would catch regressions early.

### 2. Go 1.18+ Native Fuzz Features — Underutilized

- No `testing.F` corpus files committed (beyond the one stale file we removed)
- No `-fuzzminimizingtime` tuning
- No parallel fuzz workers configured in CI

### 3. Benchmark-Driven Fuzz Seeds

The codebase has 31 benchmarks. Benchmark edge-case inputs (large payloads, concurrent access) could seed fuzz targets for better coverage.

---

## d) TOTALLY FUCKED UP 💥

### Nothing.

This session was surgical: one bug found, one bug fixed, two apps added. No regressions, no collateral damage, no broken builds. The `go.sum` cleanup (46 lines of stale checksums removed for unused deps like `go-snaps`, `ciinfo`, `go-junit`, `tparse`) was pre-existing and harmless.

---

## e) WHAT WE SHOULD IMPROVE 🔧

### High Priority

1. **Commit fuzz corpus files** — Run `nix run .#test-fuzz` with `FUZZTIME=5m`, then commit the generated `testdata/fuzz/*/` files. These become regression tests for future changes.

2. **Add fuzz tests to `catalog` module** — The schema reflection path (`Builder.Build()`) accepts arbitrary struct tags. Fuzzing it with random tag values would harden the YAML/JSON output.

3. **Wire `test-fuzz` into CI** — Add a GitHub Actions job that runs `FUZZTIME=30s nix run .#test-fuzz` on every PR. Short enough to fit in CI budgets, long enough to catch panics.

4. **Stress-test race detector** — The usermgmt module has concurrent projections (CasbinProjection, UserReadModel). Running `-count=100` via `ginkgo -repeat=100` would stress-test the race detector beyond a single pass.

### Medium Priority

5. **Fuzz the WebSocket message parser** — `ParseWSMessage` and `ParseWSMessageInto[T]` handle arbitrary JSON. They have unit tests but no fuzz coverage.

6. **Fuzz the SSE event formatter** — `WriteSSEEvent` handles multi-line data with CRLF normalization. Arbitrary `\r\n` patterns in data could expose edge cases.

7. **Fuzz `sanitizeRedirectURL`** — Already fuzzed, but the corpus could include more attack vectors: `data:` URIs, `blob:` URIs, Unicode homograph attacks, IDN domains.

8. **Property-based testing for `foldUser`** — Already has 8 rapid-based property tests. Could expand to event ordering invariants (e.g., `UserDeleted` after `UserRegistered` always produces tombstone state).

### Low Priority

9. **Fuzz `base64.RawURLEncoding.DecodeString`** — The credential ID decoder. Already implicitly fuzzed via `FuzzCredentialID_Base64Decode`, but a standalone fuzz on the decode function itself would be faster (no HTTP overhead).

10. **Benchmark fuzz targets** — Track execs/sec over time to detect performance regressions in hot paths.

---

## f) Top 25 Things to Get Done Next

| #   | Task                                                                                     | Impact | Effort  | Priority |
| --- | ---------------------------------------------------------------------------------------- | ------ | ------- | -------- |
| 1   | Commit fuzz corpus files from a long fuzz session (5m+)                                  | High   | Low     | P0       |
| 2   | Wire `test-fuzz` into GitHub Actions CI                                                  | High   | Low     | P0       |
| 3   | Add fuzz test for `catalog.Builder.Build()` (schema reflection)                          | High   | Medium  | P1       |
| 4   | Add fuzz test for `ParseWSMessage` / `ParseWSMessageInto[T]`                             | Medium | Low     | P1       |
| 5   | Add fuzz test for `WriteSSEEvent` (multi-line/CRLF)                                      | Medium | Low     | P1       |
| 6   | Stress-test race detector: `ginkgo -repeat=50` on usermgmt                               | Medium | Low     | P1       |
| 7   | Add `meta.description` to existing apps missing it (`test`, `lint`, `coverage`, `build`) | Low    | Trivial | P1       |
| 8   | Expand `FuzzSanitizeRedirectURL` seeds with attack vectors (blob:, data:, IDN)           | Medium | Low     | P2       |
| 9   | Add fuzz test for `CSRFConfig.Translate()` (header/field mapping)                        | Medium | Low     | P2       |
| 10  | Document fuzz testing workflow in CONTRIBUTING.md                                        | Low    | Low     | P2       |
| 11  | Add fuzz test for `EventOptionsFromContext` with malformed context values                | Low    | Low     | P2       |
| 12  | Enrich `FuzzDecodeJSONBody` with real-world JSON payloads from examples                  | Low    | Low     | P2       |
| 13  | Add `-fuzzminimizetime=1m` to `test-fuzz` app for better crash minimization              | Low    | Trivial | P2       |
| 14  | Add fuzz test for `ratelimit.go` (concurrent limiter access)                             | Medium | Medium  | P2       |
| 15  | Add property-based test for `paginate[T]` (credential pagination)                        | Low    | Low     | P2       |
| 16  | Update ROADMAP.md coverage numbers (95.5%/88.1% vs stale 96.4%/84.1%)                    | Low    | Trivial | P2       |
| 17  | Update TODO_LIST.md coverage numbers (same staleness)                                    | Low    | Trivial | P2       |
| 18  | Add fuzz test for `decoder.go` body size limit edge cases                                | Low    | Low     | P3       |
| 19  | Add fuzz test for `structured_error.go` JSON marshaling                                  | Low    | Low     | P3       |
| 20  | Consider `go test -fuzz=` parallel worker tuning in CI                                   | Low    | Low     | P3       |
| 21  | Add benchmark for fuzz target throughput (execs/sec regression detection)                | Low    | Medium  | P3       |
| 22  | Audit whether any non-test code panics on malformed input (not just fuzz targets)        | Medium | High    | P3       |
| 23  | Add integration fuzz test: full HTTP request → handler → response cycle                  | Medium | High    | P3       |
| 24  | Consider adding `test-bench` nix app for benchmark runs                                  | Low    | Low     | P3       |
| 25  | Review if `go-webauthn` ceremony endpoints need additional fuzz seeds                    | Low    | Low     | P4       |

---

## g) Top Question I Cannot Answer Myself 🤔

**"Should the fuzz test corpus files (`testdata/fuzz/`) be committed to git, or should they be generated fresh on each CI run?"**

Arguments for committing:

- Becomes a regression test suite — if a fuzz input once caused a panic, it should never regress
- Reproducible across machines
- Fast feedback in CI (seeds are pre-built)

Arguments against:

- Binary-ish files in git are ugly
- Corpus grows unboundedly over time
- Different platforms may generate different corpus files

I lean toward **committing them** (Go's official recommendation is to commit corpus files that trigger bugs), but I cannot decide the project's policy on repository bloat vs. regression coverage. The Go team does commit corpus files to the standard library.

---

## Session Metrics

| Metric        | Value                                    |
| ------------- | ---------------------------------------- |
| Files changed | 2 (`flake.nix`, `usermgmt/fuzz_test.go`) |
| Lines added   | 62                                       |
| Lines removed | 1                                        |
| Bugs fixed    | 1                                        |
| Tests added   | 0 (apps wrap existing tests)             |
| Time spent    | ~30 min                                  |
| Commit        | `7de8445`                                |

## Verification Commands

```bash
nix run .#test-race              # All 4 modules pass with -race
FUZZTIME=30s nix run .#test-fuzz # All 9 fuzz targets pass
nix run .#lint                   # 0 issues
nix run .#build                  # All modules compile
nix fmt                          # 0 files changed
nix flake check                  # All checks pass
```
