package main

// A REAL usermgmt + loginpage stack for Playwright's WebAuthn
// virtual-authenticator specs (plan W10 / D11 / S57): the only browser-level
// proof that the rendered login page drives the full register → begin →
// finish and login → begin → finish ceremonies end to end.
//
// Everything is in-memory and disposable: the process exists per test run,
// so durable stores would only slow the suite down. The WebAuthn provider
// is the REAL go-webauthn implementation — the virtual authenticator on the
// Playwright side produces genuine attestation/assertion bytes, so origin
// and RPID must match the browser's (localhost:18923).

import (
	"log"
	"net/http"

	loginpage "github.com/larsartmann/cqrs-htmx/loginpage/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	usermgmtwebauthn "github.com/larsartmann/cqrs-htmx/usermgmt/webauthn/v4"
	"github.com/larsartmann/httputil"
)

// e2eWebAuthnOrigin matches the Playwright webServer base URL. The port is
// pinned by playwright.config.ts (baseURL) — keep the two in sync.
const e2eWebAuthnOrigin = "http://localhost:18923"

// mountLoginPage wires usermgmt (auth routes, sessions, real WebAuthn), the
// login page, and a session-gated post-login landing page onto mux. The
// whole subtree runs behind httputil's CSRF middleware — the documented
// consumer posture (loginpage README "Quick start") and the reason the page
// ships a csrf-token meta tag for assets/login.js to round-trip.
func mountLoginPage(mux *http.ServeMux) error {
	passkeys, err := usermgmtwebauthn.New(usermgmtwebauthn.Config{
		RPID:          "localhost",
		RPDisplayName: "cqrs-htmx e2e",
		RPOrigins:     []string{e2eWebAuthnOrigin},
	})
	if err != nil {
		return err
	}

	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{WebAuthn: passkeys})
	if err != nil {
		return err
	}

	login, err := loginpage.New(loginpage.Config{
		Service: svc,
		Title:   "E2E Sign in",
		Brand:   "cqrs-htmx e2e",
		// Deterministic nonce so the render golden proves nonce plumbing
		// without depending on a per-request value.
		NonceFromRequest: func(*http.Request) string { return "e2e-fixed-nonce" },
		Redirect:         "/post-login",
	})
	if err != nil {
		return err
	}

	sub := http.NewServeMux()
	usermgmt.NewAuthHandler(svc, usermgmt.HandlerConfig{Secure: new(bool)}).RegisterRoutes(sub)
	sub.Handle("GET /login", login)
	sub.HandleFunc("GET /post-login", postLoginHandler())

	// Session enrichment (so /post-login can read UserFromContext) + CSRF,
	// the documented consumer posture: httputil issues the token the login
	// page's meta tag feeds back through assets/login.js's X-CSRF-Token.
	session := usermgmt.NewSessionMiddleware(svc, "session_token")
	csrf := httputil.CSRFMiddleware(httputil.CSRFConfig{})
	wrapped := session(csrf(sub))

	mux.Handle("/login", wrapped)
	mux.Handle("/post-login", wrapped)
	mux.Handle("/auth/", wrapped)

	return nil
}

// postLoginHandler is the session-gated landing page the login page
// redirects to: it renders the signed-in user's email from the session
// context, so the specs assert a REAL session (not just a URL change).
func postLoginHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := usermgmt.UserFromContext(r.Context())
		if !ok || user == nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		log.Printf("post-login: signed in as %s", user.Email)
		_, _ = w.Write([]byte(`<!DOCTYPE html>
<html lang="en"><head><meta charset="utf-8"><title>post-login</title></head>
<body data-signed-in="` + user.Email + `"><main id="post-login-main">Signed in as <span id="post-login-email">` + user.Email + `</span></main></body></html>`))
	}
}
