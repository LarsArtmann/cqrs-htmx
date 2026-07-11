// Package loginpage provides a ready-made, passwordless login page for
// applications using the [cqrs-htmx/usermgmt/v4] authentication system.
//
// The login page handles WebAuthn (passkey) login and registration ceremonies
// entirely client-side: the ArrayBuffer/Base64URL serialization, the
// navigator.credentials prompts, CSRF token threading, and user-friendly error
// messages for each WebAuthn failure mode.
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
// # Self-contained
//
// The page inlines its CSS and JavaScript — no external asset requests,
// no build step, no CDN dependencies. The consumer can optionally link
// an additional stylesheet via [Config.CSSPath] for branding overrides.
package loginpage
