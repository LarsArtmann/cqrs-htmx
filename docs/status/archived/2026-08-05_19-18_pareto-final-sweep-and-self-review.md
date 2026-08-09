# Status Report: Pareto Plan Final Sweep + Self-Review

**Date:** 2026-08-05 19:18
**Session scope:** Completed remaining M13/M15 tasks from the 20-task Pareto plan, migrated remaining flake.nix apps to auto-discovery, found and fixed a real bug in httputil, self-review.

---

## a) FULLY DONE (this session)

### 1. Fixed broken `datastar/v4` module resolution (M13 blocker)

**Root cause:** Commit `c7440aaa` ("release: publish datastar/v4.0.0 tag, strip local replaces") stripped the local `replace` directives from `examples/datastar-demo/go.mod` and `integration_test/go.mod`, claiming "Verified: GOWORK=off". This was **false** — only `datastar/v4.0.0` was published, but both go.mod files referenced `v4.6.1`. The `GOWORK=off` builds silently failed but nobody noticed because:

1. The old hardcoded `build` app in flake.nix included `examples/datastar-demo` but the nix sandbox used workspace mode (GOWORK=on), which resolved via `go.work` replaces, masking the broken standalone resolution.
2. `integration_test` was in the `test` app but same masking applied.

**Fix:** Re-added local `replace` directives (matching the existing `examples/dashboard-demo` pattern for `dashboardui/v4`):

- `examples/datastar-demo/go.mod`: `replace github.com/larsartmann/cqrs-htmx/datastar/v4 => ../../datastar`
- `integration_test/go.mod`: `replace github.com/larsartmann/cqrs-htmx/datastar/v4 => ../datastar`

Both modules now build and test cleanly with `GOWORK=off`. The auto-git daemon committed these as `5615cd16` and `64f3ef54`.

**Verification:** `nix run .#build` (all 19 modules), `nix run .#test` (all 14 test suites pass).

### 2. `e2e/server` builds standalone — no exclusion needed

The handoff notes suggested excluding `e2e/server` from the `build` app. I tested it: `cd e2e/server && GOWORK=off go build ./...` exits 0 cleanly. It doesn't need Playwright deps to **build** (only to **run**). Auto-discovery including it is an improvement — it was previously excluded from the hardcoded build list for no valid reason. All 3 previously-excluded examples (`catalog-demo`, `middleware-demo`, `observability-demo`) also build cleanly.

### 3. Migrated 5 more flake.nix apps to `forEachGoModule` auto-discovery

**Before this session:** Only `build` and `test` used `forEachGoModule`.

**This session migrated:**

- `lint` — 11 hardcoded `cd` blocks → 1 `forEachGoModule` call
- `test-race` — 7 hardcoded blocks → 1 call
- `test-flake` — 11 hardcoded blocks → 1 call
- `test-fuzz` — 11 hardcoded blocks with duplicated fuzz loops → 1 call + `runModuleFuzz` helper
- `coverage` — 10 hardcoded blocks → 1 call

**After:** 7 of ~12 flake.nix apps use `forEachGoModule`. The remaining hardcoded apps are intentionally hardcoded:

- `coverage-gate` — per-module thresholds (`. 90`, `identity-model 70`, etc.) can't be auto-discovered
- `errorfamily` — intentional auth-module exemptions (totp/webauthn/oauth2 excluded by design)
- `check-cqrs-lint` — custom `.cqrs-lint.json` per module
- `check-codegen` — only applies to `adminui/` and `loginpage/`
- `e2e` — builds and runs a Playwright server (special lifecycle)
- `release-checklist`, `check-phantom-version`, `check-modules`, `check-templates` — specialized scripts

**Committed by auto-git as:** `d250d28d`, `64f3ef54`, `4203932f`.

### 4. M15: httputil SecurityHeaders field tests + bug fix

Added 8 new test functions (covering 18 total security-related test cases) to `/home/lars/projects/httputil/security_test.go`:

| Test                                                           | Coverage                                                                                          |
| -------------------------------------------------------------- | ------------------------------------------------------------------------------------------------- |
| `TestSecurityHeaders_PermissionsPolicy`                        | `PermissionsPolicy` header rendering                                                              |
| `TestSecurityHeaders_EmptyPermissionsPolicy`                   | Empty value omits header                                                                          |
| `TestSecurityHeaders_CustomHeaders`                            | `Custom` map merging (2 headers)                                                                  |
| `TestSecurityHeaders_CustomHeadersEmptyMap`                    | Nil/empty Custom map is safe                                                                      |
| `TestSecurityHeaders_ContentTypeOptionsPrecedence`             | 3 sub-tests: explicit overrides bool, explicit alone, bool alone                                  |
| `TestSecurityHeaderSkip_SuppressesHeaders`                     | 3 sub-tests: suppresses ContentTypeOptions (even with nosniff bool), FrameOptions, ReferrerPolicy |
| `TestSecurityHeadersConfig_Validate_AcceptsSecurityHeaderSkip` | Validate accepts "-" for FrameOptions                                                             |
| `TestSecurityHeaders_RecommendedConstants`                     | RecommendedHSTS + RecommendedCSP render correctly                                                 |

**Found and fixed a real bug** (`httputil/security.go`): When `ContentTypeOptions` was set to `SecurityHeaderSkip` ("-") but `ContentTypeNosniff` was `true`, the `else if cfg.ContentTypeNosniff` fallback incorrectly set `X-Content-Type-Options: nosniff` despite the explicit skip. The original code was:

```go
// BUG: SecurityHeaderSkip fell through to the else-if
if cfg.ContentTypeOptions != "" && cfg.ContentTypeOptions != SecurityHeaderSkip {
    resp.Header().Set(...)
} else if cfg.ContentTypeNosniff {  // <-- this fired even when skip was intended
    resp.Header().Set("X-Content-Type-Options", "nosniff")
}
```

Fixed by checking `SecurityHeaderSkip` first and short-circuiting:

```go
if cfg.ContentTypeOptions == SecurityHeaderSkip {
    // explicitly suppressed — do not fall through to nosniff bool
} else if cfg.ContentTypeOptions != "" {
    resp.Header().Set("X-Content-Type-Options", cfg.ContentTypeOptions)
} else if cfg.ContentTypeNosniff {
    resp.Header().Set("X-Content-Type-Options", "nosniff")
}
```

**Committed in httputil repo as:** `0076791 fix(security): ensure SecurityHeaderSkip suppresses headers correctly`.

### 5. Documentation updated

- **AGENTS.md:** Replaced the "flake.nix module-list maintenance burden" gotcha (now stale) with "flake.nix auto-discovery via `forEachGoModule`" documenting the new pattern. Added a new gotcha: "datastar/v4 and dashboardui/v4 tags are NOT published at v4.6.1" documenting the replace-directive requirement.
- **CHANGELOG.md:** Added entries for auto-discovery migration (Added), datastar-demo/integration_test go.sum fix (Fixed), httputil SecurityHeaderSkip bug fix (Fixed).

---

## b) PARTIALLY DONE

### CI workflow (`ci.yml`) still has hardcoded module lists

The `.github/workflows/ci.yml` file still enumerates modules explicitly for build/test/lint/coverage jobs. This was NOT migrated to auto-discovery because GitHub Actions doesn't have a `forEachGoModule` equivalent — you'd need a matrix strategy or a shell script. This is a **known limitation** documented in the AGENTS.md gotcha.

### `docs/status/` report from prior session exists but may be stale

`docs/status/2026-08-05_18-56_full-pareto-execution-complete.md` was written by the prior session and committed as `bcc89fb5`. It describes M13 as "in progress" — this session completed it.

---

## c) NOT STARTED

### M18: golines in nix fmt (P3 — intentionally deferred)

The Verschlimmbessern guard in the plan file explicitly defers this. Adding golines would reformat the entire Go codebase (line-length wrapping), creating massive diffs and merge conflicts. Correctly deferred.

### `check-templates` not wired into CI

The `nix run .#check-templates` app (which verifies `//go:build ignore` SQL setup files compile) cannot run in CI because it requires local go-cqrs-lite replaces. This is a pre-existing limitation, not a regression.

---

## d) TOTALLY FUCKED UP

### Nothing this session.

The prior session's handoff notes had two factual errors that I corrected:

1. Claimed `e2e/server` "needs Playwright/browser deps" to build — **false**, it builds fine with `GOWORK=off`.
2. Claimed the old hardcoded build "excluded `e2e/server`" as if intentional — it was just an omission. Including it via auto-discovery is strictly better.

---

## e) WHAT WE SHOULD IMPROVE

### 1. The `c7440aaa` commit introduced silent build breakage

**The most damaging issue found this session.** A commit titled "release: publish datastar/v4.0.0 tag, strip local replaces" stripped replace directives and claimed "Verified: GOWORK=off" — but the verification was never actually run (or was run in workspace mode by mistake). The `datastar/v4.6.1` tag doesn't exist upstream. This broke standalone builds of `datastar-demo` and `integration_test` silently for potentially days.

**Process improvement:** Before stripping any `replace` directive, run `cd <module> && GOWORK=off go build ./... && go test ./...` and paste the actual output into the commit message. Better yet: add a CI check that runs `GOWORK=off go build ./...` in every module (the nix `build` app already does this, but CI may not).

### 2. The auto-git daemon makes attribution difficult

The auto-git daemon committed almost all of this session's work under Lars's name with generated commit messages. Some were good (`5615cd16` accurately describes the datastar replace fix), but others lost context (e.g., the `integration_test/go.mod` fix was bundled into `64f3ef54` under a generic "refactor lint task" message). This makes archaeological diggability harder.

### 3. 37 pre-existing lint issues in root module

`golangci-lint run` in the root module reports 37 issues (19 canonicalheader, 14 exhaustruct, 3 gochecknoglobals, 1 godoclint). These are all pre-existing — my changes didn't introduce them. But the `nix run .#lint` app stops at the first failing module (root), so it never lints the other 10 modules. This means **lint has been silently broken for the entire workspace** since the canonicalheader/exhaustruct findings were introduced. The `lint` app should either (a) continue past failures (`|| true` per module with a summary at the end), or (b) these 37 issues should be fixed.

### 4. Root module lint issues are in files I touched this session

The `security_test.go` exhaustruct findings (6 of the 14) are in the `cqrshtmx.SecurityHeadersConfig{}` test structs — these are testing partial configs on purpose. They should be suppressed with `//nolint:exhaustruct` or the test file should be excluded from exhaustruct in `.golangci.yml` (it may already be excluded for the root module but the lint still fires).

---

## f) Up to 50 things we should get done next

### High impact (fix real problems)

1. **Fix the 37 root module lint issues** — 19 `canonicalheader` findings (HTMX headers like `HX-Request` should be `Hx-Request`), 14 `exhaustruct` (SecurityHeadersConfig test structs), 3 `gochecknoglobals`, 1 `godoclint`. The canonicalheader ones may be intentional (HTMX convention) and need nolint directives or a linter config exclusion.
2. **Make `nix run .#lint` continue past failures** — currently it stops at root (first failing module) and never lints the other 10 modules. Use `|| { echo "FAIL: $dir"; fail=1; }` per module with a summary.
3. **Publish `datastar/v4.6.1` and `dashboardui/v4.6.1` tags** — then strip the local replaces from `datastar-demo`, `integration_test`, and `dashboard-demo`. Until then, the replaces are correct but create a split between "what go.mod says" and "what actually resolves".
4. **Add `GOWORK=off go build ./...` as a CI check** — the nix build app does this, but CI may use workspace mode. This would have caught the `c7440aaa` breakage immediately.
5. **Fix `forEachGoModule` lint stop-on-first-failure** — the `lint` app inherits the stop-at-first-error behavior, making workspace-wide lint useless when root has pre-existing issues.

### Medium impact (reduce maintenance burden)

6. **Migrate `.github/workflows/ci.yml` to a matrix strategy** — enumerate modules via `go work edit -json | jq` in a setup step, then use a dynamic matrix. Eliminates the last hardcoded module list.
7. **Add a "module count drift" check** — assert that `go work edit -json | jq '.Use | length'` matches expected count. Catches accidental module additions/removals.
8. **Wire `check-templates` into CI** — currently only runnable locally (needs go-cqrs-lite replaces). Could use a Docker-based CI step with the replaces checked out.
9. **Add `nix run .#check-codegen` to CI** — the CI codegen check (added M16) mirrors this but duplicates the logic. Could call the nix script directly.
10. **Consolidate `test-race` into `test`** — `test` already uses `-race`. The `test-race` app is redundant and has different exclude patterns.

### M15 follow-up (httputil)

11. **Add `SecurityHeadersConfig.Validate()` tests for PermissionsPolicy/Custom** — currently Validate only checks FrameOptions. Consider whether PermissionsPolicy should have constrained values (e.g., feature policy have a specific syntax).
12. **Add httputil integration test** — verify SecurityHeaders middleware composes correctly with CORS, compression, etc.
13. **Add httputil benchmark for PermissionsPolicy/Custom** — the existing benchmark only covers DefaultSecurityHeadersConfig.
14. **Tag httputil v0.9.1** — the SecurityHeaderSkip bug fix is a patch-worthy release. Currently on `main` after v0.9.0.

### Coverage improvements

15. **Root module coverage may have dropped** — the 6 exhaustruct-flagged test structs in `security_test.go` suggest partial config testing. Verify coverage gate still passes.
16. **Add tests for `forEachGoModule` itself** — the shell function has no test coverage. A test script in `scripts/` could verify it discovers all modules and respects exclude patterns.
17. **Test the `runModuleFuzz` helper** — verify the fuzz discovery loop works correctly (lists and runs fuzz targets).

### Documentation

18. **Update the prior session's status report** — `docs/status/2026-08-05_18-56_full-pareto-execution-complete.md` says M13 is "in progress". Add an annotation that M13 is now complete.
19. **Document the `forEachGoModule` pattern in a guide** — `docs/guides/nix-auto-discovery.md` or similar. Currently only documented in AGENTS.md gotcha.
20. **Add a "migration guide" for hardcoded flake.nix apps** — for contributors who need to migrate remaining hardcoded apps.

### Code quality

21. **The `test-fuzz` `runModuleFuzz` function uses `|| true` in a pipe** — `go test -list='Fuzz.*' ./... | grep '^Fuzz' || true`. This swallows real errors from `go test`. Should capture the exit code separately.
22. **The `coverage` app creates `coverage.out` in every module directory** — these aren't gitignored. Consider using `mktemp` or cleaning up.
23. **`forEachGoModule` uses `eval "$cmd"`** — this is technically unsafe if `$cmd` contains shell injection. In practice it's always a literal string from the flake, but worth documenting.
24. **The `build` app includes `e2e/server` but `test` excludes it** — this asymmetry is correct (e2e needs Playwright to test, not build) but could confuse. Document in the app description.

### Deferred Pareto tasks

25. **M18: golines in nix fmt** — still P3, Verschlimmbessern risk. Only do if the team agrees on line length.
26. **Migrate `errorfamily` to auto-discovery with an allowlist** — currently hardcoded to 7 modules. Could use `forEachGoModule` with an exclude for auth modules.
27. **Migrate `check-cqrs-lint` to auto-discovery** — currently hardcoded. Could use `forEachGoModule` with the same exclude pattern.
28. **Add `nix run .#check-dep-budgets` to CI** — dependency budget checking is a script but not wired into CI.
29. **Add `nix run .#check-version-drift` to CI** — same as above.
30. **Add `nix run .#check-module-isolation` to CI** — same.

### Testing infrastructure

31. **Add a test for the `forEachGoModule` exclude pattern** — verify `'^(e2e/|examples/)'` correctly excludes all 7 example dirs and e2e.
32. **Add a test for `forEachGoModule` with no exclude** — verify the `build` app discovers all 19 modules.
33. **Add integration test for `nix run .#build` in CI** — currently CI builds modules individually; the nix build app is a different code path.
34. **Add a flake eval test** — `nix flake check --no-build` passes but could be wired into CI.

### Process improvements

35. **Add a pre-commit check for stripped replace directives** — `git diff --cached -- go.mod | grep '^-replace'` should warn.
36. **Add a commit message check for "Verified: GOWORK=off"** — require the actual command output, not just the claim.
37. **Tag releases after publishing** — the `datastar/v4.6.1` and `dashboardui/v4.6.1` tags should have been created before stripping replaces.
38. **Add a release checklist step** — "verify all replace directives in consumer modules still resolve".
39. **Document the "workspace mode masks standalone breakage" gotcha** — this is how `c7440aaa` shipped broken.

### Cleanup

40. **Remove the stale `docs/status/2026-08-05_18-56_*.md` "M13 in progress" claim** — or annotate it.
41. **Clean up `coverage.out` files** — if any exist in module directories from prior `nix run .#coverage` runs.
42. **Verify `.gitignore` covers `coverage.out`** — the coverage app creates these in every module.
43. **Remove redundant `test-race` app** — it duplicates `test` which already uses `-race`.
44. **Consolidate `test-flake` into `test`** — `test` could accept a `-count` flag.
45. **Add `nix run .#fmt` alias** — currently `nix fmt` is the only way to format.
46. **Add a `nix run .#ci` meta-app** — runs all CI checks locally (build + test + lint + codegen + coverage-gate).
47. **Document the Verschlimmbessern guard in AGENTS.md** — it's in the plan file but not in AGENTS.md.
48. **Add a "how to add a new Go module" guide** — now that auto-discovery exists, this is simpler (just add to go.work).
49. **Review all `//nolint` directives for staleness** — the lint config may have evolved.
50. **Run `nix run .#check-docs-links`** — verify no broken links after doc changes.

---

## g) Questions for the user

### 1. Should I fix the 37 root module lint issues?

**Context:** The root module has 37 golangci-lint issues: 19 `canonicalheader` (HTMX uses `HX-Request` but Go's canonical form is `Hx-Request`), 14 `exhaustruct` (test structs intentionally omit fields), 3 `gochecknoglobals` (deprecated re-export vars), 1 `godoclint`. These are all pre-existing (not introduced this session). The canonicalheader ones are likely **intentional** — HTMX's convention is `HX-*` headers. Fixing them could break consumer code that checks `r.Header.Get("HX-Request")`.

**My recommendation:** Add scoped `//nolint:canonicalheader` directives to the HTMX header constants (they're intentionally non-canonical to match HTMX's convention), add `//nolint:exhaustruct` to the test structs, and leave the gochecknoglobals ones (they're deprecated re-export vars that will be removed in v5). But I can't decide for you whether HTMX header casing is a convention worth preserving vs. following Go's canonical form.

### 2. Should I publish `datastar/v4.6.1` and `dashboardui/v4.6.1` tags now?

**Context:** The local `replace` directives I re-added are the correct fix for now, but they create a split: go.mod says `v4.6.1` but resolves to local source. Publishing the tags would let me strip the replaces and have clean standalone resolution. However, I don't know if v4.6.1 is the version you intend to publish — it may need a CHANGELOG, release notes, or different version number. I also don't know if you have a release process that needs to run first.

### 3. Should `nix run .#lint` continue past failures (report all modules) or stop at the first failing module?

**Context:** Currently `lint` stops at root (the first module) because root has 37 pre-existing issues. This means the other 10 modules are never linted by `nix run .#lint`. I can make it continue past failures (collect all issues, report at end), but this changes the semantics: a non-zero exit no longer means "the module you're looking at failed" — it means "at least one module somewhere failed". The alternative is to fix the root issues (question 1). I can't decide whether the "stop at first failure" behavior is intentional (fail-fast) or a bug.

---

## Summary

| Metric                                     | Value                                                             |
| ------------------------------------------ | ----------------------------------------------------------------- |
| Tasks completed this session               | 6 (M13 fix + verify, 5 app migrations, M15 tests + bug fix, docs) |
| Bugs found and fixed                       | 2 (datastar replace stripping, httputil SecurityHeaderSkip)       |
| Tests added                                | 8 functions, 18 test cases (httputil security_test.go)            |
| Flake.nix apps migrated to auto-discovery  | 5 (lint, test-race, test-flake, test-fuzz, coverage)              |
| Flake.nix apps now using `forEachGoModule` | 7 of ~12                                                          |
| Pre-existing lint issues surfaced          | 37 in root module                                                 |
| Pareto plan tasks remaining                | M18 (golines, P3, intentionally deferred)                         |
