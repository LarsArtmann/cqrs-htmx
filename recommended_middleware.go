package cqrshtmx

import (
	"net/http"

	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/larsartmann/httputil"
)

// RecommendedPermissionsPolicy is a Permissions-Policy header value that
// disables browser features commonly unnecessary in server-rendered web apps
// (geolocation, microphone, camera, payment, USB). Use it as a starting point
// and tighten further for APIs or single-page apps that need even fewer
// capabilities.
const RecommendedPermissionsPolicy = "geolocation=(), microphone=(), camera=(), payment=(), usb=()"

// RecommendedSecurityMiddleware returns the baseline middleware chain
// recommended for production HTTP services built with cqrs-htmx:
//
//   - RecoveryMiddleware — catches panics and returns 500 instead of crashing.
//     Outermost, so panics from every later layer (including the security
//     layers below) are converted to responses — the ordering
//     httputil.MiddlewareStack.Validate enforces.
//   - SecurityHeaders — X-Content-Type-Options, X-Frame-Options,
//     Referrer-Policy, and RecommendedPermissionsPolicy.
//   - Nonce — generates a per-request CSP nonce and sets a
//     Content-Security-Policy header that allows 'self' + nonce for scripts
//     and styles, plus img-src 'self' data: for the login page's inline
//     SVG favicon.
//
// Both adminui.Handler.Middleware and dashboardui.Dashboard.Middleware delegate
// to this function so that mounting either UI gives the same security posture.
// Consumers building their own UI can call it directly:
//
//	http.ListenAndServe(":8080", cqrshtmx.Chain(
//	    cqrshtmx.RecommendedSecurityMiddleware(),
//	    sessionMW,
//	    httputil.CSRFMiddleware(httputil.CSRFConfig{}),
//	    app.Middleware(),
//	)(mux))
//
// To extend the chain with the stack's guardrails (duplicate-name detection,
// recovery-first validation), start from [RecommendedSecurityStack] instead.
func RecommendedSecurityMiddleware() func(http.Handler) http.Handler {
	stack, err := RecommendedSecurityStack()
	if err != nil {
		// Unreachable by construction: the stack below adds three distinct
		// well-known names with recovery first, so Add cannot collide and
		// Validate cannot fail. A panic here means this file drifted from
		// its own entries — fix the entries, not the panic (MustNew
		// precedent for impossible-by-construction failures).
		panic(err)
	}

	return stack.Middleware()
}

// RecommendedSecurityStack returns the same baseline production chain as
// [RecommendedSecurityMiddleware] as a named, validated
// httputil.MiddlewareStack, so consumers extending it with Add inherit the
// stack's guardrails instead of rebuilding them by hand:
//
//   - duplicate-name detection — adding "csrf" twice (the copy-paste
//     duplication class) fails at construction, not in production;
//   - recovery-first ordering validation — a stack that pushes recovery below
//     another layer stops catching its panics; Validate rejects it.
//
// Entries, outermost first: recovery, security headers, per-request CSP nonce.
// Add consumer middleware AFTER the baseline (Add appends inward), then call
// Validate before Build:
//
//	stack, err := cqrshtmx.RecommendedSecurityStack()
//	if err != nil {
//		return err
//	}
//	if err := stack.Add("session", sessionMW); err != nil {
//		return err
//	}
//	if err := stack.Validate(); err != nil {
//		return err
//	}
//	http.ListenAndServe(":8080", stack.Build(mux))
//
// The stack is safe to extend while an earlier Build is serving; see
// httputil.MiddlewareStack.
func RecommendedSecurityStack() (*httputil.MiddlewareStack, error) {
	securityCfg := httputil.DefaultSecurityHeadersConfig()
	securityCfg.PermissionsPolicy = RecommendedPermissionsPolicy

	entries := []struct {
		name       string
		middleware func(http.Handler) http.Handler
	}{
		{httputil.MiddlewareRecovery, RecoveryMiddleware},
		{httputil.MiddlewareSecurityHeaders, httputil.SecurityHeaders(securityCfg)},
		{httputil.MiddlewareNonce, httputil.Nonce(httputil.NonceConfig{Size: 0, CSPBuilder: recommendedCSP})},
	}

	stack := httputil.NewMiddlewareStack()
	for _, entry := range entries {
		if err := stack.Add(entry.name, entry.middleware); err != nil {
			return nil, errorfamily.Wrapf(err, errorfamily.Classify(err),
				"cqrshtmx.recommended_stack.add_failed", "add %s to the recommended security stack", entry.name)
		}
	}

	if err := stack.Validate(); err != nil {
		return nil, errorfamily.Wrapf(err, errorfamily.Classify(err),
			"cqrshtmx.recommended_stack.validate_failed", "validate the recommended security stack")
	}

	return stack, nil
}

// recommendedCSP extends httputil.RecommendedCSPWithNonce with
// img-src 'self' data:. The login page renders its brand favicon as an
// inline SVG data URI (no extra route to 404), which the default-src
// 'self' fallback would block; images cannot execute script, so data:
// in img-src carries no script risk.
func recommendedCSP(nonce string) string {
	return httputil.RecommendedCSPWithNonce(nonce) + "; img-src 'self' data:"
}
