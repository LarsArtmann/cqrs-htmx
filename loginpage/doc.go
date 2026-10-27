// Package loginpage provides a ready-made, passwordless login page for
// applications using the [cqrs-htmx/usermgmt/v4] authentication system.
//
// The login page handles WebAuthn (passkey) login and registration ceremonies
// entirely client-side: the ArrayBuffer/Base64URL serialization, the
// navigator.credentials prompts, CSRF token threading, and user-friendly error
// messages for each WebAuthn failure mode.
//
// # Features
//
//   - Server-side user ID generation (ULID auto-generated, no client-side ID)
//   - OAuth2 auto-detection (buttons populated from Service.ConfiguredOAuth2Providers)
//   - Browser WebAuthn detection (graceful fallback for unsupported browsers)
//   - RFC 7807 error parsing (extracts title from Problem Details JSON)
//   - Accessibility (aria-live on error regions for screen readers)
//   - JS inlined via go:embed; CSS is consumer-compiled Tailwind via Config.CSSPath
//
// # Quick start
//
//	loginHandler, err := loginpage.New(loginpage.Config{
//	    Service:     svc,
//	    Title:       "My App",
//	    Redirect:    "/dashboard",
//	    AccentColor: "#0ea5e9",
//	})
//	if err != nil { log.Fatal(err) }
//	loginHandler.Mount(mux, "/login")
//
// The consumer must also register the usermgmt auth API endpoints
// (svc.AuthHandler().RegisterRoutes) and CSRF middleware on the same mux.
//
// # OAuth2 auto-detection
//
// When [Config.OAuth2Buttons] is empty, the page auto-detects configured OAuth2
// providers via Service.ConfiguredOAuth2Providers() and generates buttons with
// display names (e.g., "google" -> "Google"). Set OAuth2Buttons explicitly to
// override labels or limit which providers are shown.
//
// With [Config.AutoRedirect], a page whose ONLY sign-in method is a single
// OAuth2 provider skips the button: GET answers with a redirect straight to
// the provider's begin endpoint. Multiple providers or a configured WebAuthn
// provider render the page normally.
//
// # Self-contained
//
// The page inlines its CSS and JavaScript — no external asset requests,
// no build step, no CDN dependencies. The consumer can optionally link
// an additional stylesheet via [Config.CSSPath] for branding overrides.
package loginpage
