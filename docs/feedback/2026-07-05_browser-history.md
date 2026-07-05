# Consumer Feedback — cqrs-htmx

**From:** browser-history project (github.com/larsartmann/browser-history)
**Date:** 2026-07-05
**Version used:** v3.3.0 (cqrs-htmx/v3, usermgmt/v3)
**Consumer:** Crush (AI assistant) + Lars

---

## What Works Great

### Middleware Chain helper (`cqrshtmx.Chain`)

```go
chain := cqrshtmx.Chain(
    cqrshtmx.SecurityHeadersMiddleware,
    cqrshtmx.RecoveryMiddleware,
    usermgmt.NewSessionMiddleware(...),
    cqrshtmx.CSRFMiddleware(...),
    cqrshtmx.ContextEnrichmentMiddleware(...),
    cqrshtmx.RateLimiterMiddleware(...),
    cqrshtmx.RequestLoggingSlog(logger),
)
```

Clean, composable, type-safe. The variadic `func(http.Handler) http.Handler` pattern is idiomatic Go middleware. We restructured to a slice-based approach for conditional CSRF inclusion and it worked seamlessly with `Chain(middlewares...)`.

### `HTMXScriptHandler()` — embed and serve htmx.js

One line: `mux.Handle("GET /htmx.js", cqrshtmx.HTMXScriptHandler())`. Serves the embedded htmx.min.js with correct Content-Type and caching headers. Eliminated our fragile `fetch-tailwind.sh` + static file approach.

### SecurityHeadersMiddleware

X-Content-Type-Options, X-Frame-Options, Referrer-Policy — correct defaults out of the box. We test for these headers in `TestSecurityHeaders`.

### usermgmt passwordless WebAuthn

The `usermgmt.Service` + `SQLiteSessionStore` integration is clean. `NewSessionMiddleware("session_token")` → `UserIDFromRequest(r)` → `cqrshtmx.ParseUserID(uid)` bridges auth into the cqrs-htmx context. Works on first try.

### CSRFConfig is thorough

TrustedOrigins, TrustedProxies, AllowPlaintextBypass, SameSite, Secure, Domain, Path, MaxAge, ErrorHandler — every option I'd want. The `ForbiddenErrorHandler` default is sensible. Validation (`SameSite=None requires Secure=true`) catches misconfiguration.

---

## Pain Points

### 1. **SKILL.md documents v4 but we use v3 — major confusion**

**Problem:** The SKILL.md shows `import "github.com/larsartmann/cqrs-htmx/v4"` and documents `App.Command`, `App.Query`, `CSRFMiddleware`, `HTMXScriptHandler` as v4 features. But the project uses v3.3.0.

**Impact:** I spent 30+ minutes examining git tags to verify whether each feature exists in v3.3.0 or only v4. This was the **#1 blocker** in the planning phase — it determined whether CSRF, HTMXScriptHandler, and App.Command/Query were "small migration" or "major version upgrade first."

**Result:** All features exist in v3.3.0. But I had to `git show v3.3.0:csrf_middleware.go` and `git show v3.3.0:app.go` to verify each one.

**Suggestion:**

- (a) Add a "v3 vs v4" section to the SKILL.md showing which features are available in which version
- (b) Or add version annotations to each feature in the skill: "Available since: v3.3.0"
- (c) Or just update all import paths to show the latest stable version

### 2. CSRF testing is painful — nosurf token masking is undocumented

**Problem:** `CSRFMiddleware` uses justinas/nosurf, which uses **token masking**. The cookie value is NOT the same as the valid header token. A masked token is derived from the cookie value per-request.

**Impact:** My first attempt to fix CSRF-broken tests was:

1. GET /health → extract csrf_token cookie
2. Set X-CSRF-Token header to the cookie value
3. POST with that header

This **failed** because nosurf's masking means the header must contain a freshly-masked token derived from the cookie — you can't just echo the cookie. I wasted ~15 minutes before giving up and using a config flag to disable CSRF in tests.

**Suggestion:**

- (a) Provide a test helper: `cqrshtmx.CSRFTestToken(ts.URL) string` that handles the nosurf dance
- (b) Document the masking behavior in the CSRFConfig docs
- (c) Add a `CSRFConfig.Testing bool` flag that bypasses validation (cleaner than a separate config field)

### 3. `App.Command/Query` returns `http.HandlerFunc` — incompatible with Huma

**Problem:** The SKILL.md presents `App.Command("CreateVisit", DecodeJSON(...), ValidateCommand(...))` as THE way to build handlers. But `App.Command()` returns `http.HandlerFunc`, which is fundamentally incompatible with Huma's `huma.Register(api, operation, handler)` model.

**Impact:** We evaluated migrating 575 lines of Huma handlers to App.Command/Query and correctly identified it as VERSCHLIMMBESSER — it would lose OpenAPI docs, type-safe I/O structs, and automatic validation.

**Suggestion:**

- (a) Document in the SKILL.md that App.Command/Query is for greenfield or stdlib-mux projects
- (b) Add a "Using cqrs-htmx with Huma" recipe showing middleware-only integration (what we do)
- (c) Consider a Huma adapter: `cqrshtmx.HumaCommand(api, method, path, cmdType, opts...)` that bridges both

### 4. CSRFResponseHeaderMiddleware exists but is undocumented in the skill

**Problem:** `CSRFResponseHeaderMiddleware` sets the CSRF token in a response header (so HTMX can pick it up via `hx-headers`). I found it by reading the source. The SKILL.md only mentions `CSRFMiddleware`.

**Suggestion:** Document the middleware trio: `CSRFMiddleware` (validates) + `CSRFResponseHeaderMiddleware` (exposes token for HTMX) + `ContextEnrichmentMiddleware` (sets request context).

### 5. RateLimiterConfig.KeyExtractor error handling

**Problem:** The `KeyExtractor` function returns a string but has no error path. If IP parsing fails (`net.SplitHostPort` error), we return `r.RemoteAddr` as fallback, but the rate limiter has no way to log or alert on this.

**Minor:** Not a real problem — just a note that the KeyExtractor contract assumes "always succeeds."

---

## Minor Notes

- **`cqrshtmx.ParseUserID` / `cqrshtmx.UserIDFromContext`** — clean bridge between string-based usermgmt IDs and typed cqrs-htmx UserID. Works great.
- **`RecoveryMiddleware`** — panic recovery with stack trace logging. Saved us during development. Good defaults.
- **The `go.mod` has many indirect deps** (justinas/nosurf, watermill, etc.) — not a problem, just noting the dependency surface is large.

---

## Summary

cqrs-htmx v3.3.0 is a **solid middleware + auth library** that works well with both stdlib mux and Huma. The main pain is **documentation**: the SKILL.md describes v4 without clarifying v3 availability, and CSRF testing patterns are undocumented. The App.Command/Query builder is a different paradigm that doesn't compose with existing router frameworks — documenting this explicitly would prevent wasted evaluation effort.

---

## Resolution Status (2026-07-05)

### Pain Points — Resolutions

| #   | Suggestion                                                               | Status       | Notes                                                                                                                                                                                                        |
| --- | ------------------------------------------------------------------------ | ------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| 1   | SKILL.md documents v4 but we use v3 — add "v3 vs v4" section             | **DONE**     | "v3 vs v4" section added to SKILL.md. All features documented in the skill are confirmed available since v3.3.0+. Root module API is unchanged between v3 and v4                                             |
| 2a  | Provide test helper `CSRFTestToken(ts.URL) string`                       | **DONE**     | `cqrshtmx.CSRFTestToken(mw)` implemented in `csrf_testing.go`. Handles nosurf token masking automatically                                                                                                    |
| 2b  | Document the nosurf masking behavior in CSRFConfig docs                  | **DONE**     | nosurf masking documented in gotchas.md with full explanation + CSRFTestToken usage pattern                                                                                                                  |
| 2c  | Add `CSRFConfig.Testing bool` flag                                       | **NOT DONE** | Chose the test helper approach instead — cleaner than a config flag that bypasses security                                                                                                                   |
| 3a  | Document that App.Command/Query is for greenfield or stdlib-mux projects | **DONE**     | SKILL.md now has "SSR / HTMX apps" section documenting when to use the pipeline vs manual handlers                                                                                                           |
| 3b  | Add "Using cqrs-htmx with Huma" recipe                                   | **NOT DONE** | Deferred — cross-framework recipe belongs in examples/. Noted in status report                                                                                                                               |
| 3c  | Consider Huma adapter `cqrshtmx.HumaCommand(...)`                        | **NOT DONE** | Deferred — design decision needs consumer validation                                                                                                                                                         |
| 4   | Document `CSRFResponseHeaderMiddleware`                                  | **DONE**     | Added to SKILL.md discoverability section: "The middleware trio is: CSRFMiddleware (validates) + CSRFResponseHeaderMiddleware (exposes token) + ContextEnrichmentMiddleware/app.Middleware() (sets context)" |
| 5   | RateLimiterConfig.KeyExtractor error handling                            | **NOT DONE** | Minor — noted as "not a real problem" in the original feedback                                                                                                                                               |
