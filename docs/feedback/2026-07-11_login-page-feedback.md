# Feedback: cqrs-htmx needs a built-in login page

**From:** SwettySwipperWeb integration (second usermgmt consumer)
**Date:** 2026-07-11
**Perspective:** App developer who just wrote 251 lines of hand-rolled login HTML/JS and wants to delete it
**Tone:** Direct, constructive, with concrete API proposals

---

## The problem

cqrs-htmx's `usermgmt` package provides a complete passwordless auth backend: WebAuthn, OAuth2/OIDC, TOTP, session management, CSRF — all wired through clean JSON API endpoints (`POST /auth/webauthn/login/begin`, etc.). The `adminui` package provides a polished templ-based admin dashboard with users, tenants, audit logs.

**But there is no login page.** The library sends unauthenticated users to `/login` (via `defaultLoginRedirect`) and leaves them staring at a 404. Every consumer has to build their own.

SwettySwipperWeb's `login_page.go` is 251 lines of inline HTML + JavaScript that:

1. Collects an email address
2. Calls `POST /auth/webauthn/login/begin` → gets WebAuthn options
3. Invokes `navigator.credentials.get()` (the browser passkey prompt)
4. Serializes the assertion (ArrayBuffer → Base64)
5. Calls `POST /auth/webauthn/login/finish` with the serialized credential
6. Handles success (redirect to `/`), cancellation (`NotAllowedError`), and errors

Plus a separate registration flow that chains 3 API calls (`/auth/register` → `/auth/webauthn/register/begin` → `/auth/webauthn/register/finish`).

This is the **third consumer** (SwettySwipperWeb, after the admin-demo's dev-login shortcut and the basic example's absence of login) that has had to solve this. The admin-demo sidestepped it entirely with a `/dev-login` shortcut that sets a cookie directly — not a real login page.

---

## What a built-in login page should deliver

### 1. A templ component, not a static HTML string

SwettySwipperWeb's `login_page.go` builds the page as a raw Go string with `+ csrfToken +` concatenation. This is fragile, untestable, and can't be themed.

A templ component (or a configurable handler that renders one) would be type-safe, composable with `templ-components` layouts, and consistent with the `adminui` package's approach (which already uses templ for its dashboard).

### 2. WebAuthn ceremony handled client-side

The bulk of the hand-rolled JavaScript is WebAuthn glue:

```javascript
function arrayBufferToBase64(buf) { ... }
function base64ToArrayBuffer(b64) { ... }
function prepareOptions(options) { ... }  // challenge/credentialId Base64 → ArrayBuffer
function serializeCredential(cred) { ... }  // assertion → JSON with Base64 fields
function serializeRegistration(cred) { ... }  // attestation → JSON with Base64 fields
```

These functions are **identical in every consumer**. They are pure mechanical transformations between the WebAuthn API's ArrayBuffer-based types and JSON's Base64-string representation. The library should own this code.

### 3. Adaptive auth-method rendering

The library supports three auth strategies (WebAuthn, OAuth2, TOTP) injected via `ServiceConfig`. The login page should adapt to which strategies are configured:

- **WebAuthn only** → email field + "Sign in with passkey" button
- **OAuth2 configured** → "Sign in with Google/GitHub" buttons alongside passkey
- **TOTP enabled** → after passkey/OAuth success, show TOTP code input (second factor)
- **No strategies configured** → this is a misconfiguration; show a helpful error

Currently each consumer has to figure out which buttons to show by reading `ServiceConfig` and hand-coding the conditional UI.

### 4. Registration flow

WebAuthn registration is a **three-step chain** (`/auth/register` → `/auth/webauthn/register/begin` → `/auth/webauthn/register/finish`). The ordering, the `user_id` threading between steps, the `credential_name` query parameter — these are all library implementation details that the consumer shouldn't need to know about.

The login page component should optionally include a registration section (toggleable via config) that handles the full enrollment ceremony.

### 5. CSRF token integration

The page needs the CSRF token in two places: a hidden form field (for progressive enhancement) and a JS-accessible hidden input (for fetch headers). The current hand-rolled version duplicates the token:

```html
<input type="hidden" name="_csrf" value="{token}" />
<input type="hidden" id="csrf-token" value="{token}" />
```

The library already has `cqrshtmx.CSRFTokenFormField(r)` — the login page should use it automatically.

### 6. Error handling and UX states

WebAuthn ceremonies have specific failure modes that need user-friendly messages:

- `NotAllowedError` → "Passkey prompt was cancelled or timed out"
- `SecurityError` → "This domain is not authorized for passkeys"
- Network errors → "Could not reach the server"
- 401 from `/finish` → "Authentication failed — invalid credential"

Every consumer writes these same error handlers. They should be built in.

---

## Proposed API

### Option A: Login page handler (recommended)

A new sub-module `usermgmt/loginpage` (or directly in `usermgmt`) that provides an HTTP handler:

```go
import loginpage "github.com/larsartmann/cqrs-htmx/usermgmt/loginpage/v4"

loginHandler := loginpage.New(loginpage.Config{
    Service:    svc,            // *usermgmt.Service — reads configured auth strategies
    Title:      "My App",       // page title
    Redirect:   "/",            // post-login redirect
    AccentColor: "#0ea5e9",     // theme
    CSSPath:    "/css/app.css", // optional: consumer's stylesheet
    // Show registration section (default: true)
    AllowRegistration: true,
})

mux.Handle("GET /login", loginHandler)
```

The handler:

- Renders a templ page with email + passkey button (+ OAuth2 buttons if configured)
- Embeds the WebAuthn JavaScript (Base64/ArrayBuffer helpers, ceremony flow)
- Includes CSRF token automatically
- Handles all error states with user-friendly messages
- Redirects to `Config.Redirect` on success

### Option B: Templ component only

For consumers who want full layout control:

```go
import loginpage "github.com/larsartmann/cqrs-htmx/usermgmt/loginpage/v4"

// In your own templ template:
@loginpage.Form(loginpage.Props{
    CSRFToken: cqrshtmx.CSRFTokenFormField(r),
    AuthMethods: svc.ConfiguredAuthMethods(), // []AuthMethod{WebAuthn, OAuth2, ...}
    Redirect: "/",
})
```

The component renders the form + button + embedded JS, the consumer wraps it in their own layout.

### Option C: Both (ideal)

Provide the raw templ component (Option B) AND a convenience handler that wraps it in a minimal standalone page (Option A). This mirrors how `adminui` works: `panel.Handler()` gives you the full page, but the templ components are also available for embedding.

---

## Why this matters

### Every consumer pays the same tax

SwettySwipperWeb's `login_page.go` is 251 lines. The admin-demo avoids the problem with a dev shortcut. The basic example has no login at all. This means:

1. **New consumers hit a wall immediately** — they wire up `usermgmt`, point `LoginRedirect` at `/login`, and then... nothing. They have to build the page from scratch.
2. **The WebAuthn JavaScript is copy-paste boilerplate** — the ArrayBuffer/Base64 serialization is deterministic and identical everywhere. It's not consumer-specific logic.
3. **Security risk** — hand-rolled auth pages are where mistakes happen. A consumer who forgets to include the CSRF token in fetch headers, or mishandles the `user_id` threading between registration steps, creates a security hole.

### The library already does this for admin

`adminui` provides a complete templ-based admin dashboard out of the box. The login page is arguably **more important** than the admin panel — without it, users can't reach the admin panel. The asymmetry is surprising.

### Alignment with the "batteries included" philosophy

cqrs-htmx already provides: embedded htmx.js, embedded SSE extension, CSRF middleware, rate limiting, security headers, error mapping, session middleware, admin panel. A login page is the missing piece that would make the library a complete drop-in auth solution.

---

## What the page should NOT do

To stay focused and avoid over-scoping:

- **No password fields** — auth is passwordless; don't add a password fallback
- **No custom styling framework** — use Tailwind classes (matching `adminui` and `templ-components`) but let consumers override via `Config.AccentColor` and `Config.CSSPath`
- **No account management UI** — profile/credential management belongs in the consumer app or adminui, not the login page
- **No email sending** — email verification flows stay as JSON API endpoints; the login page just shows "check your email" on success

---

## Scope for a first version (MVP)

The smallest useful version:

1. `GET /login` handler rendering a templ page
2. Email input + "Sign in with passkey" button
3. Embedded WebAuthn JS (begin → browser prompt → finish → redirect)
4. CSRF token auto-included
5. Error states with friendly messages
6. `Config{Service, Title, Redirect, AccentColor}` — that's it

OAuth2 buttons, registration flow, and TOTP second-factor can come in v2. The MVP alone would let SwettySwipperWeb delete all 251 lines of `login_page.go`.

---

## Summary

| Aspect                   | Current state                                                   | With built-in login page                     |
| ------------------------ | --------------------------------------------------------------- | -------------------------------------------- |
| Time to first login      | 2-4 hours (build page, write JS, debug WebAuthn, handle errors) | 3 lines (`loginpage.New(cfg)`, `mux.Handle`) |
| WebAuthn JS correctness  | Each consumer writes & debugs independently                     | Library-owned, tested once                   |
| CSRF integration         | Manual token threading                                          | Automatic                                    |
| Auth-method adaptivity   | Hardcoded per consumer                                          | Reads `ServiceConfig`                        |
| Consistency with adminui | Login page looks nothing like admin panel                       | Shared templ/Tailwind design language        |

The `usermgmt` backend is excellent — clean API, well-documented, flexible injection of auth strategies. A built-in login page would complete the "passwordless auth in a box" story that the library is 90% of the way to delivering.

---

_This feedback is offered with appreciation for a genuinely well-designed library. The login page gap is the single biggest friction point in adopting `usermgmt` — closing it would make the package a turnkey auth solution._
