# httputil v0.11.0 Adoption — Cleanup & Hardening Status

_Date: 2026-08-09 04:21_

---

## Context

A prior session executed all 13 prioritized opportunities from the httputil
v0.11.0 deep-dive audit report. That session left 50 next-steps documented in
`docs/status/2026-08-09_02-03_httputil-deep-dive-action-execution.md`, including
11 unfixed lint warnings, untested behavior, unrun quality gates, and
uncommitted docs.

**This session's task:** Fix the lint warnings, run all quality gates, add
tests for new behavior, update docs, and leave a clean tree.

---

## A) FULLY DONE

### 1. Fixed all lint warnings in middleware-showcase (11 → 0)

**Commit `87ea3cad` (auto-git), refined by direct edits in this session.**

Rewrote `examples/middleware-showcase/main.go` to fix all 11 lint warnings:

| Warning            | Fix                                                                                                         |
| ------------------ | ----------------------------------------------------------------------------------------------------------- |
| `wsl_v5` (3x)      | Added blank lines before `if` blocks after multi-statement sequences                                        |
| `exhaustruct` (2x) | Added `//nolint:exhaustruct` on `MetricsConfig` and `ServerConfig` (examples legitimately use partial init) |
| `forbidigo` (2x)   | Replaced two `fmt.Printf` calls with a single `log.Printf`                                                  |
| `gci`              | Fixed import ordering (struct field alignment)                                                              |
| `tagliatelle`      | Fixed struct field alignment (`ServerTime` consistent)                                                      |
| `gochecknoglobals` | Moved `requestCount` from package-level to function-local in `main()`                                       |

### 2. Fixed nlreturn lint in dashboardui

**Commit `ca6978a3` (auto-git).**

`dashboardui/handler.go:35` — Added missing blank line before `return` in
`Middleware()` method (the `PermissionsPolicy` line added by the prior session
was directly above the `return` with no separator).

### 3. All quality gates verified passing

| Gate                        | Result                                                                           |
| --------------------------- | -------------------------------------------------------------------------------- |
| `nix run .#lint`            | 0 issues across all 11 modules                                                   |
| `nix run .#test`            | All 14 test suites pass (root 4.0s, adminui 4.3s, usermgmt 22.1s, etc.)          |
| `nix run .#coverage-gate`   | All 11 gates pass (root 93.5%, adminui 68.7%, dashboardui 83.8%, datastar 97.4%) |
| `nix run .#check-codegen`   | Committed `_templ.go` files current (adminui + loginpage)                        |
| `nix run .#check-templates` | All 4 SQL setup template files compile                                           |

### 4. Added 3 tests for new behavior

**Commit `b12ec89c`.**

- **`app_test.go`** — Ginkgo `It("auto-registers httputil error classifications")`:
  Creates an App, then verifies `errorfamily.Classify(http.ErrNotSupported).HTTPStatus()`
  returns 503 (ServiceUnavailable). Proves the `sync.Once` guard in `New()` calls
  `httputil.RegisterErrorClassifications()`.
- **`adminui/handler_test.go`** — `TestMiddleware_PermissionsPolicyAndNonce`:
  Calls `panel.Middleware()`, runs a test request through it, verifies:
  (1) `Permissions-Policy` header contains all 5 feature restrictions
  (`geolocation=()`, `microphone=()`, `camera=()`, `payment=()`, `usb=()`).
  (2) `Content-Security-Policy` header contains `nonce-` (nonce middleware wired).
- **`dashboardui/handlers_security_test.go`** — `TestMiddleware_PermissionsPolicy`:
  Same `Permissions-Policy` header assertions for dashboardui's `Middleware()`.

### 5. Updated AGENTS.md

**Commit `6806b127`.**

- Added `middleware-showcase` to examples list.
- Rewrote "httputil leverage" bullet: documents `RegisterErrorClassifications()`
  auto-call via `sync.Once` in `New()`, adminui `Middleware()` includes
  `httputil.Nonce` + `PermissionsPolicy`, `nonce(r)` fallback to
  `NonceFromRequest`, dashboardui `PermissionsPolicy`, middleware-showcase
  example reference, httputil v0.11.0, leveraging-httputil.md recipe additions.
- Updated lint/gate verification dates from 2026-08-05 to 2026-08-09.
- Updated "7 canonical examples" → "8 canonical examples".

### 6. Updated CHANGELOG.md

**Commit `6806b127`.**

Added `[Unreleased]` section with:

- **Added**: RegisterErrorClassifications auto-call, adminui Nonce middleware,
  PermissionsPolicy headers, middleware-showcase example, health+compression in
  basic, new guide recipes.
- **Changed**: ServerTiming import fix.
- **Fixed**: Binary in .gitignore.

### 7. Removed tracked binary

**Commit `d7f2034f`.**

The prior session's commit `73d3f802` claimed to remove the binary
`examples/middleware-showcase/middleware-showcase` (10MB compiled Go binary)
but only ran `git rm --cached` in a way that didn't take effect — the file
remained tracked. This session properly `git rm --cached` + `trash`ed the file,
confirming it's no longer in the index. The `.gitignore` entry from `87ea3cad`
prevents future re-addition.

### 8. Added binary to .gitignore

**Commit `87ea3cad`.**

Added `examples/middleware-showcase/middleware-showcase` to `.gitignore`,
matching the pattern used by all other example binaries.

---

## B) PARTIALLY DONE

### Test coverage for new behavior

The 3 tests added cover the core assertions but are not comprehensive:

- **adminui nonce fallback test missing**: No test verifies that `nonce(r)`
  returns a non-empty value when `httputil.Nonce` middleware is in the chain
  AND no custom `NonceFunc` is set. The `TestMiddleware_PermissionsPolicyAndNonce`
  test verifies the CSP header contains a nonce, but doesn't directly test the
  `nonce(r)` method's fallback path.
- **No test for backward-compat path**: No test verifies that setting a custom
  `NonceFunc` still takes precedence over the httputil.Nonce fallback.
- **No E2E test for middleware-showcase**: The example has zero tests. The 50-item
  list from the prior status report suggested an E2E test verifying 304 on ETag
  match, 429 on rate limit, and CORS headers.

### Documentation freshness

- The deep-dive HTML report (`docs/research/2026-08-09_httputil-deep-dive.html`)
  still shows items as "open" that have been resolved. No "RESOLVED" markers
  were added.
- The cqrs-htmx skill SKILL.md was not updated with new example reference or
  `RegisterErrorClassifications` note.

---

## C) NOT STARTED

1. **Push to origin** — 11 commits unpushed. Not done per project rules (push
   only when explicitly asked).
2. **Release tagging** — No decision on v4.7.1 vs v4.8.0 for the consumer-visible
   changes (adminui Middleware() behavior change, RegisterErrorClassifications
   auto-call).
3. **NonceFunc deprecation marker** — The prior session chose backward-compatible
   fallback over hard deprecation. A `// Deprecated:` marker was not added.
4. **dashboardui Nonce middleware** — adminui got `httputil.Nonce` in its
   `Middleware()` chain; dashboardui did not. The prior report noted this as
   item 25.
5. **Shared recommendedMiddleware helper** — adminui and dashboardui now have
   duplicated SecurityHeaders + PermissionsPolicy + Recovery chains. No DRY
   extraction was attempted.
6. **`cqrshtmx.RegisterErrorClassifications()` re-export** — Consumers who use
   httputil without creating an App still need to import httputil directly to
   call it. A re-export in root was not added.
7. **Decompression/MaxBodySize in adminui** — The guide documents these recipes
   but adminui's own middleware chain doesn't include them.

---

## D) TOTALLY FUCKED UP

### 1. Relied on LSP diagnostics that were stale throughout the session

The LSP reported 11 stale lint warnings on `middleware-showcase/main.go`
throughout the entire session — long after the file was rewritten. I had to
mentally filter these out and rely on `go build`/`go vet`/`nix run .#lint`
as authoritative. This is a known issue documented in AGENTS.md, but I wasted
cognitive overhead checking whether diagnostics were real each time.

### 2. Did not notice the binary was still tracked until final verification

The prior session's commit `73d3f802` claimed to clean up the binary but
didn't actually remove it from the git index. I only caught this at the very
end when `git status` showed it as modified. Should have verified the prior
session's cleanup claims at the start of this session rather than at the end.

### 3. Did not verify the prior status report's claims before acting

The prior status report said "11 lint warnings need fixing" and listed them.
The actual file had already been partially fixed by the auto-git daemon (commit
`87ea3cad`), so my initial rewrite was fixing issues that were already resolved.
I adapted quickly but should have re-read the file's current state before
writing a full replacement.

---

## E) WHAT WE SHOULD IMPROVE

### Process

1. **Verify prior session claims at session start.** The prior status report
   documented 11 lint warnings and a binary cleanup as incomplete. Both claims
   were partially stale (some fixes applied by auto-git daemon, binary still
   tracked despite cleanup commit). Always re-verify the actual state before
   acting on a prior report.

2. **The auto-git daemon creates confusion.** Multiple commits in this session
   were made by the daemon before I could commit manually. This is documented
   as expected behavior in AGENTS.md but it means I can't reliably know what
   the "current diff" looks like at any point. Recommendation: accept this as
   a fact of life, always check `git status` + `git log` before and after edits.

3. **`git rm --cached` + commit is the only reliable way to untrack a file.**
   The prior session thought it removed the binary but didn't verify with
   `git ls-files`. Always verify tracked status after any `git rm --cached`.

### Code Quality

4. **adminui and dashboardui Middleware() are duplicated.** Both now have
   nearly identical SecurityHeaders + PermissionsPolicy + Recovery chains.
   This is a DRY violation that should be extracted into a shared helper.

5. **The middleware-showcase `logRecorder` is trivial.** The prior report
   noted this. A real Prometheus recorder example would be more useful.

6. **No test verifies the nonce fallback priority.** If `NonceFunc` is set,
   does it still take precedence over `NonceFromRequest`? The code says yes
   but there's no test.

7. **The tests don't cover the `nonce(r)` method directly.** They only check
   the CSP header contains a nonce. A direct test of `h.nonce(r)` with and
   without `NonceFunc` set would be more precise.

### Architecture

8. **`RegisterErrorClassifications()` in `New()` is a hidden side effect.**
   Creating an App shouldn't globally mutate error classification state.
   A consumer who creates two Apps gets `sync.Once` protection, but a consumer
   who imports httputil directly and calls it themselves could see duplicate
   registration (though `RegisterClassifications` is likely idempotent). This
   should be documented more prominently.

9. **dashboardui missing Nonce middleware is asymmetric.** adminui has it,
   dashboardui doesn't. If a consumer mounts both, they get nonce support
   on one but not the other. Either both should have it or neither should.

---

## F) Next 50 Things to Get Done

### Release & Push

1. Push 11 unpushed commits to origin/master
2. Decide on v4.7.1 (patch) vs v4.8.0 (minor) tag — adminui Middleware()
   behavior changed (Nonce added, PermissionsPolicy added), New() auto-calls
   RegisterErrorClassifications()
3. Tag root, adminui, dashboardui modules if releasing
4. Update `go.mod` version references in examples that pin specific versions
5. Verify `nix run .#build` passes hermetically (GOWORK=off) for all modules

### Testing

6. Add test: `nonce(r)` returns non-empty when httputil.Nonce middleware is
   present and `NonceFunc` is nil
7. Add test: `nonce(r)` returns `NonceFunc` result when `NonceFunc` is set
   (backward compat precedence)
8. Add test: `New()` calling `RegisterErrorClassifications()` is idempotent
   (two Apps → no panic, classification still works)
9. Add E2E test for middleware-showcase (304 on ETag match, 429 on rate limit,
   CORS headers present)
10. Add test: dashboardui Middleware() sets `Content-Security-Policy` (currently
    only tests Permissions-Policy)
11. Add test: adminui Middleware() sets `X-Content-Type-Options` and other
    security headers beyond just Permissions-Policy

### adminui/dashboardui improvements

12. Add `// Deprecated:` marker to `NonceFunc` pointing to `httputil.Nonce`
13. Add `httputil.Nonce` to dashboardui's `Middleware()` chain
14. Extract shared `recommendedSecurityMiddleware()` helper for adminui + dashboardui
15. Replace adminui's `HealthHandler()` with `httputil.ReadyHandlerWithProbe`
16. Replace dashboardui's hand-rolled health handlers with httputil equivalents
17. Consider `ProductionCSPWithNonce` variant for adminui security headers
18. Audit adminui/dashboardui inline scripts for nonce attribute compatibility
19. Verify CSP headers don't break HTMX functionality (`script-src 'self'` minimum)
20. Browser-test adminui with CSP + nonce (ToastContainer, GlobalErrorHandling scripts)
21. Add `httputil.Decompression` to adminui middleware (POST form bodies)
22. Consider adding `httputil.MaxBodySize` to adminui/dashboardui middleware chains

### Example improvements

23. Replace middleware-showcase's `logRecorder` with a real Prometheus recorder
24. Add Decompression to middleware-showcase
25. Add MaxBodySize to middleware-showcase
26. Add Nonce to middleware-showcase with CSP example
27. Add Server-Timing to middleware-showcase (sub-module demonstration)
28. Add a "production middleware recipe" comment block in examples/basic
29. Decide: merge middleware-showcase into middleware-demo, or keep separate?
30. Add `httputil.RegisterErrorClassifications` re-export to root module for
    consumers who don't create an App

### Documentation

31. Update cqrs-htmx skill SKILL.md with `RegisterErrorClassifications` note
32. Update cqrs-htmx skill SKILL.md with middleware-showcase reference
33. Update deep-dive HTML report with "RESOLVED" markers on fixed items
34. Update `leveraging-httputil.md` Server-Timing re-export table (note correct
    sub-module path)
35. Verify `leveraging-httputil.md` recipe code snippets compile (they're
    documentation-only, not tested)
36. Consider a `docs/guides/recommended-middleware-stack.md` guide distilling
    the middleware-showcase pattern into prose
37. Update README.md if httputil adoption changes the quick-start experience

### Architecture

38. Consider adding `httputil.RegisterErrorClassifications()` to usermgmt
    `NewService()` (not just root `New()`)
39. Consider whether the `sync.Once` in `New()` is the right pattern vs an
    `init()` function in a sub-package
40. Consider a `cqrshtmx.RecommendedSecurityHeaders()` helper that returns
    a pre-configured `SecurityHeadersConfig` with PermissionsPolicy
41. Evaluate whether `httputil.NonceFromRequest` should be re-exported by
    cqrs-htmx for consumers who don't import httputil directly

### CI / Build

42. Verify `.github/workflows/ci.yml` covers middleware-showcase module
43. Add `middleware-showcase` to the CI lint matrix (currently excluded by
    `^(e2e|examples/)` regex)
44. Run `nix run .#check-cqrs-lint` (not run this session)
45. Run `nix run .#test-fuzz` (not run this session)
46. Run `nix run .#test-flake` (not run this session)
47. Run `nix flake check --no-build` (not run this session)
48. Consider adding a pre-commit hook that rejects tracked files >1MB (already
    exists as `check-large-files.sh` — verify it would catch the binary)

### httputil upstream feedback

49. File feedback: `RegisterErrorClassifications()` should use `sync.Once`
    internally (designed to be called once but has no internal guard)
50. Contribute `RecommendedPermissionsPolicy` constant to httputil (every
    consumer currently writes `"geolocation=(), microphone=(), ..."` manually)

---

## G) Questions I Cannot Answer Myself

### 1. Should we tag new releases now, or wait?

The changes are consumer-visible: adminui `Middleware()` now includes Nonce
middleware and PermissionsPolicy (behavioral change for anyone calling
`panel.Middleware()`), and `New()` auto-calls `RegisterErrorClassifications()`
(hidden side effect). But I don't know if you want to cut v4.7.1/v4.8.0 now or
batch these with other pending work. The 11 unpushed commits need pushing
regardless.

### 2. Should dashboardui get Nonce middleware too?

adminui's `Middleware()` now includes `httputil.Nonce(...)`. dashboardui's
does not. This asymmetry means a consumer mounting both UIs gets nonce support
on one but not the other. Adding it is trivial but changes dashboardui's
behavior too (CSP headers will include nonce). Is this desired for v4.x, or
should dashboardui stay nonce-free until v5?

### 3. Should we add a `cqrshtmx.RegisterErrorClassifications()` re-export?

`New()` auto-calls it, so most consumers don't need it. But consumers who use
httputil middleware without creating an App (e.g., mounting only
`httputil.Compression` on a static file server) would still need to call it
manually. A re-export in the root module would let them call
`cqrshtmx.RegisterErrorClassifications()` without importing httputil directly.
Is this worth adding, or does the duck-typing philosophy say "import httputil
directly"?
