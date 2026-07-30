# httputil Consolidation — Brutal Self-Review & Status

> **Date:** 2026-07-30 16:38
> **Session goal:** Execute the httputil consolidation plan — move Server-Timing, CSRF, and rate limiting from cqrs-htmx root into httputil via re-export aliases.

---

## a) FULLY DONE

### Phase 1: Server-Timing → httputil
- **httputil/server_timing.go** (436 LOC): Full W3C Server Timing implementation ported. Added `WrapServerTiming()` helper, `HeaderServerTiming` constant, `NewServerTiming()` constructor (was unexported `newServerTiming`). Includes its own `delegatingWriter` type (Flush/Hijack/Push/Unwrap delegation).
- **httputil test suite:** `server_timing_test.go` (641 LOC), `server_timing_bench_test.go` (98 LOC), `server_timing_fuzz_test.go` (103 LOC) — all ported, go-cqrs-lite imports removed, tests adapted to httputil's `Chain` signature.
- **cqrs-htmx/server_timing_reexport.go:** Type alias (`ServerTiming`), 6 var aliases, `headerServerTiming` const alias.
- **cqrs-htmx/app.go:** `applyServerTiming` now delegates to `httputil.WrapServerTiming` (3 lines, was 15).
- **cqrs-htmx/server_timing_integration_test.go:** 6 integration tests retained (App.ServerTiming end-to-end, Chain composition).
- **Verification:** Build + test + race pass on all 15 modules.

### Phase 2: CSRF → httputil
- **httputil/csrf.go** (595 LOC): Full CSRF implementation consolidated from 5 cqrs-htmx files into one file. Exported previously-internal helpers: `ConfigureNosurfHandler`, `SetPlaintextHTTPOrigin`, `TranslateCSRFHeaders`, `CSRFTokenFromRequest`. Added `ValidateCSRF()` function (extracted from cqrs-htmx's `executeCSRFValidation`).
- **httputil/csrf_test.go** (212 LOC): 14 tests covering GET token context, POST rejection/acceptance, response header middleware, config defaults, validation, cookie invalidation, HTML helpers.
- **cqrs-htmx/csrf_reexport.go:** Type aliases (`CSRFConfig`, `ErrorHandler = CSRFErrorHandler`), 12 var aliases, 3 const aliases.
- **cqrs-htmx/csrf_handler.go:** Rewritten to use `httputil.ValidateCSRF` — nosurf import eliminated.
- **cqrs-htmx/errors.go:** Removed `ErrCSRFConfig` definition (now aliased from httputil). Removed `ErrorHandler` type definition (now aliased from httputil). Comments updated.
- **cqrs-htmx go.mod:** `justinas/nosurf` completely removed (not even indirect).
- **Verification:** Build + test + race pass on all 15 modules including usermgmt (which uses `*cqrshtmx.RateLimiter`).

### Phase 3: Rate Limiting → httputil
- **httputil/ratelimit_keyed.go** (368 LOC): Full keyed rate limiter with min-heap eviction, MaxKeys cap, monitoring API. Ported `perKeyLimiter`, `evictionHeap`, `evictionEntry`, all eviction logic.
- **httputil/ratelimit_keyed_test.go** (161 LOC): 8 tests covering allow/reject, empty key exempt, ActiveKeys, MaxKeys cap, heap push non-ptr, defaults.
- **cqrs-htmx/ratelimit_reexport.go:** Type aliases (`RateLimiterConfig = KeyedRateLimiterConfig`, `RateLimiter = KeyedRateLimiter`, `KeyExtractor`), 5 var aliases, 3 const aliases.
- **cqrs-htmx go.mod:** `golang.org/x/time` demoted from direct to indirect.
- **Verification:** Build + test + race pass on all 15 modules.

### Phase 4: Documentation
- **cqrs-htmx/AGENTS.md:** Root module description updated, key dependencies updated (nosurf + x/time noted as now transitive via httputil).
- **cqrs-htmx/CHANGELOG.md:** `[Unreleased] → Changed` entry added.

---

## b) PARTIALLY DONE

### httputil documentation — NOT updated
- **httputil/doc.go:** Still lists the old feature set (CORS, client IP, compression, ETag, etc.) — does NOT mention Server-Timing, CSRF, or keyed rate limiting.
- **httputil/README.md:** Not updated.
- **httputil/AGENTS.md:** Not updated.
- **httputil/FEATURES.md:** Not updated.
- **httputil/CHANGELOG.md:** Not updated.

### cqrs-htmx doc.go — NOT updated
- Still references patterns as if CSRF/rate-limiting are implemented locally.

### cqrs-htmx SKILL.md — NOT updated
- The skill's cheat sheet and module table still describe CSRF/rate-limiting as root features without noting they're re-exports.

---

## c) NOT STARTED

1. **httputil needs a new git tag** — cqrs-htmx's `go.work` has `replace github.com/larsartmann/httputil => /home/lars/projects/httputil`. Until httputil publishes v0.8.0 (or similar), cqrs-htmx cannot remove the replace directive from go.work.
2. **Neither repo was pushed** — both have local commits only.
3. **Coverage check** — did not run `nix run .#coverage-gate` or `nix run .#coverage` on either repo.
4. **Nix build verification** — did not run `nix run .#build` or `nix run .#test` (used raw `go build`/`go test` instead).

---

## d) TOTALLY FUCKED UP / PROBLEMS FOUND

### 1. httputil depguard BLOCKS nosurf import (CRITICAL — lint fails)
**`httputil/.golangci.yml` depguard rules only allow:** `$gostd`, `$module`, `go-error-family`, `golang.org/x/time`. The new `csrf.go` imports `github.com/justinas/nosurf`, which depguard rejects. **httputil lint is currently broken.** Must add `github.com/justinas/nosurf` to the depguard allow list.

### 2. httputil has TWO competing rate limiting APIs (CONFUSING)
- **Old:** `ratelimit.go` — `RateLimiter` (interface), `TokenBucketLimiter` (struct), `RateLimitConfig`, `RateLimit()` middleware. Uses O(n) sweep eviction.
- **New:** `ratelimit_keyed.go` — `KeyedRateLimiter` (struct), `KeyedRateLimiterConfig`, `KeyedRateLimiterMiddleware()`. Uses min-heap eviction, MaxKeys cap, monitoring.
- **Problem:** Consumers now face a choice between two overlapping APIs with no guidance. The old `TokenBucketLimiter` is strictly inferior (O(n) eviction, no MaxKeys, no monitoring). It should either be deprecated or the two should be consolidated.

### 3. httputil has TWO ResponseWriter wrapper types (DUPLICATION)
- `wrapper.go` defines `responseWrapper` (with status tracking, header buffering).
- `server_timing.go` defines `delegatingWriter` (Flush/Hijack/Push/Unwrap delegation).
- These serve different purposes but overlap significantly. `responseWrapper` has its own `Hijack()` and `Flush()` methods. A unified hierarchy would be cleaner.

### 4. `ErrorHandler` naming is now misleading
- cqrs-htmx's `ErrorHandler` type was a GENERAL error handler type (used by Config.ErrorHandler, recovery, etc.).
- It is now aliased to `httputil.CSRFErrorHandler`, which sounds CSRF-specific.
- The type signature is identical (`func(w, r, err)`), so it works, but the name `CSRFErrorHandler` implies a narrower scope than the alias is used for.
- **Better:** httputil should rename `CSRFErrorHandler` to just `ErrorHandler` (or `HTTPErrorHandler`) since it's a general-purpose type that happens to be used by CSRF.

### 5. `ForbiddenErrorHandler` vs `ForbiddenCSRFHandler` name mismatch
- cqrs-htmx consumers know it as `ForbiddenErrorHandler`.
- httputil calls it `ForbiddenCSRFHandler`.
- The alias `var ForbiddenErrorHandler = httputil.ForbiddenCSRFHandler` preserves backward compat but creates naming inconsistency.

### 6. `canonicalheader` lint warnings carried over
- The CSRF code has 7 `canonicalheader` warnings (`X-CSRF-Token` is not canonical, should be `X-Csrf-Token`). These are pre-existing (nosurf uses `X-CSRF-Token`), but they now also appear in httputil's lint output.
- Not a runtime bug — `http.Header.Set` canonicalizes automatically — but lint is noisy.

### 7. Old test files in cqrs-htmx are now redundant (1288 LOC of dead weight)
- `csrf_middleware_test.go` (396 LOC), `csrf_advanced_test.go` (134 LOC), `csrf_defaults_test.go` (44 LOC), `csrf_handlers_test.go` (180 LOC), `csrf_helpers_test.go` (100 LOC), `ratelimit_test.go` (278 LOC), `ratelimit_integration_test.go` (156 LOC) — total 1288 LOC.
- These test implementation details that now live in httputil. They pass because re-exports make symbols available, but they're testing httputil behavior through cqrs-htmx aliases — pure redundancy with httputil's own tests.
- Should be deleted or dramatically slimmed down to smoke tests only.

### 8. `errors.go` has a confusing comment block where `type ErrorHandler` used to be
- The replacement left a comment saying "ErrorHandler is now an alias..." followed by the old doc comment for `DefaultErrorHandler`. Reads awkwardly — looks like the type definition is missing.

### 9. `contentTypePlain` constant duplication
- httputil/csrf.go defines `const contentTypePlain = "text/plain; charset=utf-8"`.
- cqrs-htmx/constants.go defines `const ContentTypePlain = "text/plain; charset=utf-8"`.
- Same value, different names, different packages. Minor but sloppy.

---

## e) WHAT WE SHOULD IMPROVE

1. **Fix the depguard issue IMMEDIATELY** — httputil lint is broken. Add nosurf to the allow list.
2. **Consolidate httputil's two rate limiting APIs** — either deprecate `TokenBucketLimiter` or fold its interface into the new keyed limiter.
3. **Rename `CSRFErrorHandler` to `ErrorHandler` in httputil** — it's a general-purpose type, not CSRF-specific. The CSRFConfig.ErrorHandler field just happens to use it.
4. **Delete or slim the 1288 LOC of redundant test files** in cqrs-htmx root.
5. **Unify the ResponseWriter wrapper types** in httputil — one delegating base, not two.
6. **Update ALL httputil documentation** (doc.go, README, AGENTS.md, FEATURES.md, CHANGELOG).
7. **Publish httputil v0.8.0** so cqrs-htmx can drop the go.work replace.
8. **Run `nix run .#test` and `nix run .#coverage-gate`** for canonical verification.

---

## f) Up to 50 Things to Get Done Next

| #  | Task                                                                                  | Priority   | Effort    |
| -- | ------------------------------------------------------------------------------------- | ---------- | --------- |
| 1  | Fix httputil depguard: add `justinas/nosurf` to allow list                           | CRITICAL   | 2 min     |
| 2  | Run httputil lint clean (verify 0 issues after depguard fix)                          | CRITICAL   | 5 min     |
| 3  | Run cqrs-htmx lint on root module (verify no new issues from re-exports)              | HIGH       | 5 min     |
| 4  | Delete redundant CSRF test files from cqrs-htmx (1288 LOC → ~200 LOC smoke tests)     | HIGH       | 20 min    |
| 5  | Rename `CSRFErrorHandler` → `ErrorHandler` in httputil (it's general-purpose)         | HIGH       | 10 min    |
| 6  | Rename `ForbiddenCSRFHandler` → `ForbiddenHandler` in httputil (consistency)          | HIGH       | 5 min     |
| 7  | Deprecate httputil's old `TokenBucketLimiter` (add `// Deprecated:` comment)          | HIGH       | 5 min     |
| 8  | Or: consolidate `TokenBucketLimiter` into `KeyedRateLimiter` (one API to rule them all) | MEDIUM     | 30 min    |
| 9  | Fix `errors.go` comment block (clean up the awkward `ErrorHandler` alias comment)     | MEDIUM     | 5 min     |
| 10 | Update httputil/doc.go (add Server-Timing, CSRF, keyed rate limiting)                 | HIGH       | 5 min     |
| 11 | Update httputil/README.md (new feature list)                                          | HIGH       | 10 min    |
| 12 | Update httputil/CHANGELOG.md                                                          | HIGH       | 5 min     |
| 13 | Update httputil/AGENTS.md (new deps, new modules, architectural note)                 | HIGH       | 10 min    |
| 14 | Update httputil/FEATURES.md                                                           | MEDIUM     | 10 min    |
| 15 | Update cqrs-htmx/doc.go (note re-export pattern for middleware)                       | MEDIUM     | 5 min     |
| 16 | Update cqrs-htmx SKILL.md (note re-export pattern)                                    | MEDIUM     | 10 min    |
| 17 | Add `nosurf` to httputil go.mod require block (verify it's not just go.sum)           | HIGH       | 2 min     |
| 18 | Unify `delegatingWriter` and `responseWrapper` in httputil                            | LOW        | 30 min    |
| 19 | Deduplicate `contentTypePlain` constant (export from httputil or share)               | LOW        | 5 min     |
| 20 | Run `nix run .#test` on cqrs-htmx (canonical Nix verification)                        | HIGH       | 10 min    |
| 21 | Run `nix run .#coverage-gate` on cqrs-htmx (verify thresholds still pass)             | HIGH       | 10 min    |
| 22 | Run `nix run .#test` on httputil (canonical Nix verification)                         | HIGH       | 5 min     |
| 23 | Publish httputil v0.8.0 tag                                                           | HIGH       | 5 min     |
| 24 | Remove go.work replace for httputil after tag published                               | HIGH       | 2 min     |
| 25 | Update cqrs-htmx go.mod to require httputil v0.8.0                                    | HIGH       | 2 min     |
| 26 | Push cqrs-htmx to origin                                                              | HIGH       | 2 min     |
| 27 | Push httputil to origin                                                               | HIGH       | 2 min     |
| 28 | Add `// Deprecated:` on httputil's `RateLimiter` interface pointing to `KeyedRateLimiter` | MEDIUM  | 5 min     |
| 29 | Consider: should httputil's `RateLimit()` middleware use `KeyedRateLimiter` internally? | MEDIUM  | 20 min    |
| 30 | Add integration test: cqrs-htmx re-export types ARE httputil types (type identity)     | MEDIUM     | 10 min    |
| 31 | Fix `canonicalheader` warnings (use `X-Csrf-Token` or add nolint)                      | LOW        | 10 min    |
| 32 | Verify examples still compile after nosurf removal from root go.mod                   | HIGH       | 5 min     |
| 33 | Check if `golang.org/x/time` can be fully removed (not even indirect)                 | LOW        | 5 min     |
| 34 | Add httputil coverage gate for new files (server_timing, csrf, ratelimit_keyed)        | MEDIUM     | 10 min    |
| 35 | Consider: export `delegatingWriter` from httputil (consumers may want it)              | LOW        | 5 min     |
| 36 | Update cqrs-htmx/.golangci.yml if any new lint exceptions needed                       | MEDIUM     | 5 min     |
| 37 | Update cqrs-htmx/TODO_LIST.md (mark consolidation as done, add follow-ups)             | MEDIUM     | 5 min     |
| 38 | Review: does `usermgmt/http.go` still compile cleanly with `*cqrshtmx.RateLimiter`?   | VERIFIED   | 0 min     |
| 39 | Add ADR for the httputil consolidation decision                                       | MEDIUM     | 15 min    |
| 40 | Verify: `go mod tidy` doesn't pull nosurf back into cqrs-htmx root                    | HIGH       | 2 min     |
| 41 | Consider: should httputil's `Chain` be used instead of cqrs-htmx's in tests?           | LOW        | 5 min     |
| 42 | Check `httputil/httpspec/` tests still pass with new CSRF/Server-Timing additions     | HIGH       | 5 min     |
| 43 | Add CSRF `TrustedProxies` integration test in httputil                                 | MEDIUM     | 15 min    |
| 44 | Add Server-Timing middleware benchmark through Chain in httputil                       | LOW        | 10 min    |
| 45 | Consider: move `ContentTypePlain` to httputil and alias from cqrs-htmx                 | LOW        | 10 min    |
| 46 | Review all `//nolint:` directives in ported code — may need httputil-specific ones     | MEDIUM     | 10 min    |
| 47 | Update `docs/guides/csrf-trusted-proxies.md` if it references internal helpers         | LOW        | 5 min     |
| 48 | Verify `flake.nix` coverage gate thresholds still make sense after code moves          | MEDIUM     | 5 min     |
| 49 | Consider: extract `WrapServerTiming` as a more general `Wrap` pattern                  | LOW        | 10 min    |
| 50 | Celebrate — 2 deps removed, 3 features consolidated, zero consumer API breakage       | DONE       | 0 min     |

---

## g) Questions

### Q1: Should we delete httputil's old `TokenBucketLimiter` + `RateLimiter` interface + `RateLimit()` middleware entirely, or keep them with a `// Deprecated:` marker?
The old API (O(n) sweep, no MaxKeys, no monitoring) is strictly inferior to the new `KeyedRateLimiter`. But httputil may have external consumers who depend on the `RateLimiter` interface. I cannot check this without knowing httputil's consumer landscape. Deleting would be cleaner; deprecating is safer.

### Q2: Should httputil cut a v0.8.0 tag now (with the depguard fix), or wait until the old rate limiter deprecation/consolidation is also done?
Cutting now unblocks cqrs-htmx from the go.work replace. Waiting means one release instead of two. The consolidation work is ~30 min but the decision affects release management.

### Q3: The `ErrorHandler` type alias situation — should I rename httputil's `CSRFErrorHandler` to match cqrs-htmx's `ErrorHandler`, or rename cqrs-htmx's to `CSRFErrorHandler`?
cqrs-htmx's `ErrorHandler` is used in `Config.ErrorHandler`, `recovery.go`, and many tests — it's a general-purpose type. httputil's `CSRFErrorHandler` is scoped by name. Renaming httputil's to `ErrorHandler` (or `HTTPErrorHandler`) seems right, but you may have opinions on httputil's naming conventions.
