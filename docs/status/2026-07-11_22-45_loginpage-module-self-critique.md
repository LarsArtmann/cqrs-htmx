# Status: loginpage Module — Self-Critique & Remaining Work

**Date:** 2026-07-11 22:45\
**Session:** Built `loginpage` module from scratch (2 commits), then did brutal self-review\
**Branch:** master (pushed)\
**Commits:** `00bef1d` (initial), `0591c9b` (OAuth2 + type model + fixes)

---

## What We Built

A new Go module (`github.com/larsartmann/cqrs-htmx/loginpage/v4`) — the 9th module in the cqrs-htmx workspace. A self-contained, passwordless login page that eliminates 200+ lines of hand-rolled HTML/JS every consumer currently writes.

### Module stats

| Metric                                  | Value                                           |
| --------------------------------------- | ----------------------------------------------- |
| Production Go (non-test, non-generated) | 352 lines (5 files)                             |
| Generated templ output                  | 253 lines (1 file)                              |
| Tests                                   | 410 lines, 21 test functions                    |
| CSS                                     | 238 lines                                       |
| JavaScript                              | 287 lines                                       |
| Coverage                                | 78.4%                                           |
| Lint                                    | 0 issues                                        |
| BuildFlow                               | 36/36 passed                                    |
| Direct deps                             | 4 (cqrs-htmx, usermgmt, templ, go-error-family) |

---

## A) FULLY DONE

| Item                     | Details                                                                                                                                                              |
| ------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Module structure         | go.mod, go.sum, go.work entry, .golangci.yml, flake.nix apps, CI scripts                                                                                             |
| Config type model        | `Config` struct with Service/Title/Brand/Redirect/AccentColor/CSSPath/NoRegistration/AuthPrefix/OAuth2Buttons/CredentialName                                         |
| Handler                  | `New(Config) → *Handler`, `ServeHTTP`, `Mount(mux, pattern)`                                                                                                         |
| Option B support         | Exported `Page(PageData)` templ component + `NewPageData(Config, *Request)` helper                                                                                   |
| WebAuthn login JS        | begin → navigator.credentials.get → serialize assertion → finish → redirect                                                                                          |
| WebAuthn registration JS | register → begin → navigator.credentials.create → serialize attestation → finish → redirect                                                                          |
| CSRF auto-included       | Meta tag (for JS fetch headers) + hidden form field (progressive enhancement)                                                                                        |
| Error handling JS        | NotAllowedError → "cancelled", SecurityError → "domain not authorized", network → "could not reach server"                                                           |
| Adaptive rendering       | WebAuthn form when HasWebAuthn, OAuth2 buttons from Config.OAuth2Buttons, no-auth error state                                                                        |
| OAuth2 button rendering  | Full-page redirect links to `/auth/oauth/{provider}/begin`, divider when both WebAuthn + OAuth2                                                                      |
| Theming                  | AccentColor CSS var, dark mode via prefers-color-scheme, branded favicon (inline SVG data-URI)                                                                       |
| Self-contained           | CSS + JS inlined via go:embed + templ.Raw — zero external asset requests                                                                                             |
| usermgmt additions       | `Service.HasWebAuthn()`, `Service.HasOAuth2()`, `Service.HasTOTP()` methods                                                                                          |
| README                   | 119 lines — quick start, config table, adaptive rendering table, what it doesn't do                                                                                  |
| AGENTS.md                | Architecture tree, module table, Key Decisions section                                                                                                               |
| Tests                    | 21 tests: config validation, page rendering, CSRF, endpoints, OAuth2 buttons, divider, favicon, no-auth state, registration toggle, safe redirects, helper functions |

---

## B) PARTIALLY DONE

| Item                   | What's done                                                                                              | What's missing                                                                                                                          |
| ---------------------- | -------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------- |
| OAuth2 integration     | `HasOAuth2()` on Service, `OAuth2Button` type, `Config.OAuth2Buttons` config, templ rendering with links | No auto-detection (consumer must manually list providers); no display-name mapping                                                      |
| Registration flow      | Full 3-step ceremony in JS, server-side ID validation exists                                             | Client generates UUID (`crypto.randomUUID()`) — security concern; should be server-side                                                 |
| Error messages         | WebAuthn ceremony errors (NotAllowed, Security, Abort)                                                   | API error extraction only check `error`/`detail`/`message` — missing RFC 7807 `title`/`status` fields from StructuredError              |
| Templ component export | `Page(PageData)` exported, `NewPageData` helper exists                                                   | `PageData` has unexported fields (`inlineCSS`, `inlineJS`, `configJSON`) that consumers can't set via `NewPageData` without the handler |

---

## C) NOT STARTED

| Item                                        | Impact              | Notes                                                                                 |
| ------------------------------------------- | ------------------- | ------------------------------------------------------------------------------------- |
| WebAuthn browser feature detection          | HIGH                | No `window.PublicKeyCredential` check — old browsers get raw TypeError                |
| OAuth2 provider auto-detection              | HIGH                | `oauth2.Provider` has no `Names()` method; Service can't enumerate providers          |
| Server-side user ID generation              | CRITICAL (security) | Client mints identity via `crypto.randomUUID()` — server accepts any non-empty string |
| Login demo example (`examples/login-demo/`) | MEDIUM              | No runnable showcase; admin-demo uses dev shortcut                                    |
| .gitignore exception for page_templ.go      | LOW                 | File committed via `git add -f`; needs `!loginpage/page_templ.go` rule                |
| Coverage gap to 80%                         | LOW                 | Currently 78.4%, CI gate threshold is 80% — will fail `nix run .#coverage-gate`       |

---

## D) TOTALLY FUCKED UP

| Item                                                    | What went wrong                                                                                                                                                                                      | Severity                                                        |
| ------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------- |
| **First commit included unrelated adminui changes**     | Commit `00bef1d` bundled adminui templ-components changes (from a prior session) into the loginpage commit — those files were already modified at conversation start                                 | Low (cosmetic — commit message is accurate, just messy history) |
| **`page_templ.go` not in first commit**                 | The `.gitignore` rule `*_templ.go` excluded it. Had to `git add -f` in the second commit. Should have added a `.gitignore` exception immediately                                                     | Low (fixed in second commit)                                    |
| **Coverage below CI gate**                              | The flake.nix `coverage-gate` app checks loginpage at 80% threshold but actual coverage is 78.4%. This will FAIL CI                                                                                  | **Medium** — needs 1.6% more coverage or threshold adjustment   |
| **`firstRune` duplicated in handler.go AND page.templ** | The templ calls `firstRune(p.Brand)` and handler.go defines it. But the templ also needs the function accessible. Currently works because they're in the same package, but it's an implicit coupling | Low                                                             |

---

## E) WHAT WE SHOULD IMPROVE

### Architecture

1. **Server-side ID generation is a security must** — The client sending `crypto.randomUUID()` means a malicious client can send predictable/crafted IDs to target specific aggregate IDs. The fix: `RegisterRequest.ID` should auto-generate when empty (backward compatible), and login.js should read the server-returned ID from the response.
2. **OAuth2 auto-detection eliminates friction** — Today consumers must manually list `OAuth2Button{Provider: "google", Label: "Google"}`. With `oauth2.Provider.Names()` + `Service.ConfiguredOAuth2Providers()`, the login page could auto-discover and render buttons with zero config.
3. **Browser detection prevents broken UX** — Calling `navigator.credentials.get()` on a browser without WebAuthn support throws a confusing TypeError. A 3-line check (`if (!window.PublicKeyCredential)`) shows a friendly fallback.

### Type Model

4. **`PageData` unexported fields create a leaky abstraction** — `inlineCSS`, `inlineJS`, `configJSON`, `authPrefix` are unexported but essential for rendering. Consumers using `NewPageData` get them populated, but anyone constructing `PageData` by hand (e.g. for testing) must know to set them. This is acceptable (the struct doc says "use NewPageData") but could be cleaner with a builder pattern or factory function.
5. **`OAuth2Button` is too simple** — No icon, no brand color, no provider-specific styling. A future improvement could add an optional `IconSVG` field or a provider-to-brand map.

### Testing

6. **78.4% coverage is below the 80% gate** — The gap is in `renderPage`'s error path (hard to trigger) and some branches in `buildPageData`. Needs targeted test additions.
7. **No JavaScript tests** — The WebAuthn ceremony JS is 287 lines of untested code. While JS testing isn't the Go module's responsibility, a manual test checklist or integration test would increase confidence.

### Developer Experience

8. **No runnable demo** — `admin-demo` has `/dev-login` shortcut. A `login-demo` with real WebAuthn (using `go-webauthn/webauthn` test vectors or a virtual authenticator) would let consumers try before they integrate.
9. **OAuth2 buttons need manual labels** — Provider name "google" should auto-map to "Google" (capitalized) or "Sign in with Google".

---

## F) NEXT 50 THINGS TO DO

### Critical (Security & Bugs)

| # | Task                                                                                                                      | Effort | Impact      |
| - | ------------------------------------------------------------------------------------------------------------------------- | ------ | ----------- |
| 1 | **Server-side user ID generation**: Make `RegisterRequest.ID` auto-generated when empty; remove client UUID from login.js | Medium | CRITICAL    |
| 2 | **Fix coverage gate**: Add tests to reach 80% OR lower threshold to 78% in flake.nix                                      | Low    | CI-breaking |
| 3 | **WebAuthn browser detection**: Add `window.PublicKeyCredential` check + fallback UI                                      | Low    | High UX     |

### High Value (Developer Experience)

| # | Task                                                                                     | Effort  | Impact   |
| - | ---------------------------------------------------------------------------------------- | ------- | -------- |
| 4 | **OAuth2 provider auto-detection**: `Names()` on oauth2.Provider + auto-populate buttons | Medium  | High DX  |
| 5 | **OAuth2 display names**: Map "google" → "Google", "github" → "GitHub"                   | Low     | Polish   |
| 6 | **Login demo example**: `examples/login-demo/` — go run, open :8098                      | Medium  | Adoption |
| 7 | **AGENTS.md update**: Document OAuth2Buttons, CredentialName, NewPageData                | Low     | Docs     |
| 8 | **.gitignore exception**: Add `!loginpage/page_templ.go`                                 | Trivial | DX       |

### Medium Value (Polish & Correctness)

| #  | Task                                                                         | Effort  | Impact      |
| -- | ---------------------------------------------------------------------------- | ------- | ----------- |
| 9  | JS error extraction: handle RFC 7807 StructuredError `title`/`detail` fields | Low     | Correctness |
| 10 | OAuth2 button brand icons (inline SVG for Google/GitHub/Microsoft)           | Low     | Polish      |
| 11 | Add `autofocus` to email field only when WebAuthn is the sole method         | Trivial | UX          |
| 12 | Keyboard navigation: ensure tab order is correct in OAuth2-only mode         | Trivial | A11y        |
| 13 | `aria-live="polite"` on error div for screen readers                         | Trivial | A11y        |
| 14 | Loading state for OAuth2 buttons (CSS active state on click)                 | Trivial | UX          |
| 15 | Empty-state message when OAuth2 provider returns error on callback           | Low     | UX          |

### Testing & CI

| #  | Task                                                        | Effort  | Impact     |
| -- | ----------------------------------------------------------- | ------- | ---------- |
| 16 | Test `buildPageData` with nil request (doesn't panic)       | Trivial | Coverage   |
| 17 | Test `renderPage` error path (mock templ.Component failure) | Low     | Coverage   |
| 18 | Test OAuth2 button URL construction with AuthPrefix         | Trivial | Coverage   |
| 19 | Test `faviconURI` returns valid SafeURL                     | Trivial | Coverage   |
| 20 | Test `oauthBeginURL` method                                 | Trivial | Coverage   |
| 21 | Integration test: full handler with WebAuthn mock provider  | Medium  | Confidence |
| 22 | Integration test: OAuth2 button click → redirect URL        | Low     | Confidence |
| 23 | Fuzz test: `safeRedirectPath` with random inputs            | Low     | Robustness |

### Architecture & Type Model

| #  | Task                                                                                                     | Effort  | Impact        |
| -- | -------------------------------------------------------------------------------------------------------- | ------- | ------------- |
| 24 | Consider `PageData` builder pattern instead of struct                                                    | Low     | Clean code    |
| 25 | Move `firstRune` to a shared util (adminui has `initials()` that's similar)                              | Low     | DRY           |
| 26 | Consider `OAuth2Button.IconSVG` field for brand icons                                                    | Low     | Extensibility |
| 27 | Add `Config.RedirectOnExists` for "already registered" flow                                              | Low     | Feature       |
| 28 | Consider `Config.Labels` map for i18n (button text, error messages)                                      | Medium  | i18n          |
| 29 | Expose `LoginPageCSS` and `LoginPageJS` as exported vars for consumers who want to serve them separately | Trivial | Flexibility   |

### Security Hardening

| #  | Task                                                                       | Effort  | Impact   |
| -- | -------------------------------------------------------------------------- | ------- | -------- |
| 30 | CSP nonce support: allow consumers to pass a nonce for inline script/style | Medium  | Security |
| 31 | Rate-limit registration form submissions client-side                       | Low     | Security |
| 32 | Honeypot field for bot detection                                           | Low     | Security |
| 33 | Add `Cross-Origin-Opener-Policy` header                                    | Trivial | Security |
| 34 | Consider `X-Frame-Options: DENY` to prevent clickjacking                   | Trivial | Security |

### Ecosystem Integration

| #  | Task                                                                             | Effort  | Impact        |
| -- | -------------------------------------------------------------------------------- | ------- | ------------- |
| 35 | Wire loginpage into `examples/admin-demo/` replacing `/dev-login`                | Medium  | Showcase      |
| 36 | Add loginpage to the `nix run .#test` and `nix run .#build` apps (DONE — verify) | —       | Done          |
| 37 | Add loginpage to `check-version-drift.sh` (verify it's covered)                  | Trivial | CI            |
| 38 | Create ADR for loginpage design decisions                                        | Low     | Documentation |
| 39 | Add to `docs/migrations/` if upgrading from hand-rolled login                    | Low     | Adoption      |

### Documentation

| #  | Task                                                         | Effort | Impact    |
| -- | ------------------------------------------------------------ | ------ | --------- |
| 40 | Package-level example in `doc.go` (Runnable godoc example)   | Low    | DX        |
| 41 | Screenshots in README for each auth-method combination       | Low    | Adoption  |
| 42 | Migration guide: "Delete your hand-rolled login_page.go"     | Low    | Adoption  |
| 43 | Blog post draft: "Passwordless auth in a box with cqrs-htmx" | Medium | Marketing |

### Polish & Nice-to-Haves

| #  | Task                                                            | Effort  | Impact  |
| -- | --------------------------------------------------------------- | ------- | ------- |
| 44 | Animated transitions between login/register sections            | Low     | Polish  |
| 45 | "Remember this device" checkbox (conditional mediation)         | Medium  | Feature |
| 46 | Passwordless biometric prompt icon (fingerprint SVG)            | Trivial | Polish  |
| 47 | Configurable card width (`Config.CardWidth`)                    | Trivial | Polish  |
| 48 | Footer text config (`Config.FooterHTML`)                        | Trivial | Polish  |
| 49 | Multiple card layouts (centered card, split-screen, full-width) | Medium  | Feature |
| 50 | Dark/light mode toggle button (not just prefers-color-scheme)   | Low     | Feature |

---

## G) TOP 2 QUESTIONS

### 1. Should `RegisterRequest.ID` be removed from the client-facing API entirely, or kept as optional?

**Context:** The current API requires `id` in the JSON body (`service_register.go:Validate` rejects empty). Making it auto-generated when empty is backward compatible — existing callers that provide an ID keep working, but login.js can stop sending it and use the server response instead.

**Why I can't figure this out alone:** Removing the field is a **breaking API change** for any consumer that currently calls `POST /auth/register` with their own ID. Keeping it as optional is safer but leaves a confusing "when do I set this?" question. The right answer depends on how many consumers have hardcoded IDs in their registration flow.

### 2. Should the loginpage module depend on `usermgmt/oauth2` for provider auto-detection, or should the provider names be passed through `ServiceConfig`?

**Context:** Adding `Names()` to `oauth2.Provider` is easy, but `loginpage` currently does NOT import `usermgmt/oauth2` (it only imports `usermgmt` core). If we add `Service.ConfiguredOAuth2Providers()` that returns `[]string`, loginpage stays decoupled. But if we want typed provider configs (display names, icons), we'd need either a new interface method or a direct dependency on the oauth2 module.

**Why I can't figure this out alone:** The clean architecture says "no direct dep on oauth2" — but the pragmatic answer might be "just add a `ProviderInfo` struct to `usermgmt` core that carries name + label + icon, and have the oauth2 module populate it." This is a module boundary decision that affects the v4 API surface.
