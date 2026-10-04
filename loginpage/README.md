# loginpage — Passwordless Login Page for cqrs-htmx

A ready-made, self-contained login page that eliminates the 200+ lines of
hand-rolled HTML/JS every cqrs-htmx consumer currently writes.

> Sibling panels: [adminui](../adminui/README.md) manages users and tenants
> (identity operations); [dashboardui](../dashboardui/README.md) introspects
> the event store (CQRS/ES observability). [setup/v4](../setup/README.md)
> mounts all three in one call.

## What it does

- Renders a polished login page with WebAuthn (passkey) sign-in
- Handles the full WebAuthn ceremony client-side (Base64URL helpers,
  `navigator.credentials` prompts, assertion/attestation serialization)
- **Auto-detects OAuth2 providers** — buttons populated from `Service.ConfiguredOAuth2Providers()`
- **Server-side user ID generation** — registration auto-generates ULID, no client-side ID
- **Browser WebAuthn detection** — graceful fallback for unsupported browsers
- **RFC 7807 error parsing** — extracts `title` from Problem Details JSON responses
- **Accessibility** — `aria-live="polite"` on error regions for screen readers
- Includes optional registration flow (3-step WebAuthn enrollment)
- Auto-includes CSRF token (meta tag for JS + hidden form field)
- Handles all error states with user-friendly messages
- **templ-components design system** — built on `layout.Base`, `forms.Input`,
  `display.Button`, and `feedback.Alert`; every class is a Tailwind utility
  compiled by the consumer (see [Styling](#styling))
- **CSP-nonce support** — set `NonceFromRequest` to have the inline scripts
  carry the same nonce your middleware wrote into the CSP header
- **Server-driven theming** — `Theme`/`ThemeFromRequest` render the resolved
  color scheme SSR-first (body class, no theme script), so a cookie stays the
  single source of truth
- WebAuthn ceremony JS embedded via `go:embed` — zero runtime JS dependencies

## Quick start

```go
import loginpage "github.com/larsartmann/cqrs-htmx/loginpage/v4"

loginHandler, err := loginpage.New(loginpage.Config{
    Service:     svc,
    Title:       "My App",
    Redirect:    "/dashboard",
    AccentColor: "#0ea5e9",
})
if err != nil {
    log.Fatal(err)
}

mux.Handle("GET /login", loginHandler)
```

The consumer must also register the usermgmt auth endpoints and CSRF middleware:

```go
svc.AuthHandler().RegisterRoutes(mux)
mux.Use(httputil.CSRFMiddleware(httputil.CSRFConfig{}))
```

## Configuration

| Field            | Type                | Default      | Description                                       |
| ---------------- | ------------------- | ------------ | ------------------------------------------------- |
| `Service`        | `*usermgmt.Service` | **required** | Provides auth-method detection                    |
| `Title`          | `string`            | `"Sign in"`  | Page `<title>` and heading                        |
| `Brand`          | `string`            | = Title      | App name shown above the form                     |
| `Redirect`       | `string`            | `"/"`        | Post-login redirect (root-relative)               |
| `AccentColor`    | `string`            | `"#4f46e5"`  | Accent used only for the SVG favicon (any CSS color) |
| `CSSPath`        | `string`            | `"/app.css"` | URL of the consumer's compiled Tailwind stylesheet |
| `Theme`          | `string`            | `""`         | Force `"light"`/`"dark"` (empty = prefers-color-scheme) |
| `ThemeFromRequest` | `func(*http.Request) string` | `nil` | Per-request theme hook (e.g. cookie); wins over `Theme` |
| `NonceFromRequest` | `func(*http.Request) string` | `nil` | CSP nonce for the inline scripts (base64url, sanitized) |
| `NoRegistration` | `bool`              | `false`      | Hide the registration section                     |
| `RegisterFirst`  | `bool`              | `false`      | Render the registration section on load (e.g. a `/register` route) |
| `AuthPrefix`     | `string`            | `""`         | URL prefix for auth API (`/api` → `/api/auth/..`) |
| `OAuth2Buttons`  | `[]OAuth2Button`    | **auto**     | OAuth2 provider buttons (auto-detected if empty)  |
| `CredentialName` | `string`            | `"Passkey"`  | Label for newly registered credentials            |

## OAuth2 buttons

OAuth2 sign-in buttons are full-page redirects (no JavaScript needed). When
`OAuth2Buttons` is nil or empty, the page auto-detects configured providers
via `Service.ConfiguredOAuth2Providers()` and generates buttons with display
names (e.g., "google" -> "Google", "azure-ad" -> "Azure Ad").

You can also set them explicitly:

```go
loginpage.Config{
    Service: svc,
    OAuth2Buttons: []loginpage.OAuth2Button{
        {Provider: "google", Label: "Sign in with Google"},
        {Provider: "github", Label: "Sign in with GitHub"},
    },
}
```

The `Provider` string must match a key in your `oauth2.Provider` configuration.
Buttons link to `{AuthPrefix}/auth/oauth/{provider}/begin`.

## Option B: Embed in your own layout

For consumers who want full layout control, use the exported templ component:

```go
data, err := loginpage.NewPageData(cfg, r)
// Then in your templ template:
// @loginpage.Page(data)
```

## Adaptive rendering

The page adapts to the configured auth strategies:

| Configuration     | Rendered UI                              |
| ----------------- | ---------------------------------------- |
| WebAuthn only     | Email field + "Sign in with passkey"     |
| OAuth2 only       | OAuth2 buttons                           |
| WebAuthn + OAuth2 | Passkey form + divider + OAuth2 buttons  |
| No strategies     | "No authentication method is configured" |

Registration section appears only when WebAuthn is configured (since
registration requires a WebAuthn enrollment ceremony).

## Styling

The page is built entirely on [templ-components](https://github.com/larsartmann/templ-components):
`layout.Base` (page shell), `forms.Input`, `display.Button`, `feedback.Alert`.
All classes are Tailwind v4 utilities, so the consumer must compile a
stylesheet that scans this package. With Tailwind v4's CSS-first config:

```css
@import "tailwindcss" source(none);
@source "../go.sum"; /* or the module cache path */
@source "TEMPL_COMPONENTS_DIR/{layout,forms,display,feedback,utils}/**/*";
@source "LOGINPAGE_DIR/**/*";  /* page.templ + page_templ.go + assets/login.js */
@custom-variant dark (&:where(.dark, .dark *));
```

Serve the compiled file and point `Config.CSSPath` at it (default `/app.css`).
The WebAuthn ceremony JS is still inlined via `go:embed` — the only classes it
adds at runtime (`lp-spinner`, `animate-spin`, `border-*`) live in this
package's `assets/login.js`, which the `@source` scan picks up.

## Theming

- Empty `Theme`/`ThemeFromRequest` — follows `prefers-color-scheme` via the
  library's ThemeScript
- Resolved theme (`"light"`/`"dark"`) — applied SSR-first: the body carries
  the class on first paint and no theme script is emitted, so cookie-driven
  consumers keep a single source of truth
- `Config.AccentColor` colors the generated SVG favicon (brand initial on a
  rounded square); markup colors come from the Tailwind classes
- `NoIndex` is always emitted — login pages should stay out of search results

## Browser support

The page detects WebAuthn support at load time. If `navigator.credentials` or
`PublicKeyCredential` is unavailable (older browsers, insecure contexts), the
WebAuthn login/registration sections are hidden and a fallback message is shown
with a link to OAuth2 buttons if configured.

## What it does NOT do

- **No password fields** — auth is passwordless
- **No account management** — profile/credential management belongs in the
  consumer app or adminui
- **No email sending** — email verification stays as JSON API endpoints
- **No TOTP second-factor UI** — planned for a future version
