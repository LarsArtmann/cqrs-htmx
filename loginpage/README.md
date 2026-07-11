# loginpage — Passwordless Login Page for cqrs-htmx

A ready-made, self-contained login page that eliminates the 200+ lines of
hand-rolled HTML/JS every cqrs-htmx consumer currently writes.

## What it does

- Renders a polished login page with WebAuthn (passkey) sign-in
- Handles the full WebAuthn ceremony client-side (Base64URL helpers,
  `navigator.credentials` prompts, assertion/attestation serialization)
- Shows OAuth2 sign-in buttons (Google, GitHub, etc.) when configured
- Includes optional registration flow (3-step WebAuthn enrollment)
- Auto-includes CSRF token (meta tag for JS + hidden form field)
- Handles all error states with user-friendly messages
- Zero external asset requests — CSS and JS are inlined via `go:embed`

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
mux.Use(cqrshtmx.CSRFMiddleware(cqrshtmx.CSRFConfig{}))
```

## Configuration

| Field            | Type                | Default      | Description                                       |
| ---------------- | ------------------- | ------------ | ------------------------------------------------- |
| `Service`        | `*usermgmt.Service` | **required** | Provides auth-method detection                    |
| `Title`          | `string`            | `"Sign in"`  | Page `<title>` and heading                        |
| `Brand`          | `string`            | = Title      | App name shown above the form                     |
| `Redirect`       | `string`            | `"/"`        | Post-login redirect (root-relative)               |
| `AccentColor`    | `string`            | `"#4f46e5"`  | Button/highlight color (any CSS color)            |
| `CSSPath`        | `string`            | `""`         | Optional consumer stylesheet URL                  |
| `NoRegistration` | `bool`              | `false`      | Hide the registration section                     |
| `AuthPrefix`     | `string`            | `""`         | URL prefix for auth API (`/api` → `/api/auth/..`) |
| `OAuth2Buttons`  | `[]OAuth2Button`    | `nil`        | OAuth2 provider buttons to render                 |
| `CredentialName` | `string`            | `"Passkey"`  | Label for newly registered credentials            |

## OAuth2 buttons

OAuth2 sign-in buttons are full-page redirects (no JavaScript needed). Add them
via config:

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

## Self-contained

The page inlines all CSS and JavaScript — no external requests, no CDN
dependencies, no build step. One HTML response contains everything. Consumers
can optionally link an additional stylesheet via `Config.CSSPath` for branding
overrides.

## Theming

- `Config.AccentColor` controls buttons, links, and the favicon
- Full dark mode via `prefers-color-scheme: dark`
- No Tailwind dependency — pure CSS variables
- The favicon is an inline SVG data-URI with the brand initial

## What it does NOT do

- **No password fields** — auth is passwordless
- **No account management** — profile/credential management belongs in the
  consumer app or adminui
- **No email sending** — email verification stays as JSON API endpoints
- **No TOTP second-factor UI** — planned for a future version
