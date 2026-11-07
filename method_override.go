package cqrshtmx

import "net/http"

// MethodOverrideHeader is the canonical header carrying the intended HTTP
// method when the wire method is a fallback (RFC 10008 QUERY deployments
// behind proxies or WAFs that reject unknown methods).
const MethodOverrideHeader = "X-HTTP-Method-Override"

// legacyMethodOverrideHeader is the older spelling still sent by some stacks;
// honored when the canonical header is absent.
const legacyMethodOverrideHeader = "X-HTTP-Method"

// MethodQuery is the HTTP QUERY method (RFC 10008): safe, idempotent, and
// request-body-carrying — the read-side twin of POST. net/http has no
// MethodQuery constant, so the library spells the token once.
const MethodQuery = "QUERY"

// MethodOverrideMiddleware rewrites a POST that carries
// "X-HTTP-Method-Override: QUERY" (or the legacy "X-HTTP-Method" spelling)
// into a QUERY request BEFORE routing, so a single "QUERY /path" mux pattern
// serves both the native method and the fallback transport. Every other
// request passes through untouched.
//
// Only QUERY is honored as an override target, by design: this middleware
// exists for safe read fallbacks, never to smuggle unsafe verbs (an
// "X-HTTP-Method-Override: DELETE" is ignored and the request stays a POST).
// The rewrite clones the request, so the caller's *http.Request is never
// mutated.
//
// Mount it INSIDE the security middleware (it wraps the mux), so the CSRF
// layer validates the WIRE method: a fallback POST is token-checked as a
// POST today and forever, regardless of how the upstream safe-method list
// treats QUERY later.
//
//	mux.Handle("QUERY /contacts", app.Query("SearchContacts", ...))
//
//	srv := cqrshtmx.Chain(
//	    cqrshtmx.RecommendedSecurityMiddleware(),
//	    sessionMW,
//	    httputil.CSRFMiddleware(httputil.CSRFConfig{}),
//	    cqrshtmx.HTMXMiddleware,
//	    app.Middleware(),
//	)(cqrshtmx.MethodOverrideMiddleware(mux))
func MethodOverrideMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && overrideTargetsQuery(r) {
			rewritten := r.Clone(r.Context())
			rewritten.Method = MethodQuery
			next.ServeHTTP(w, rewritten)

			return
		}

		next.ServeHTTP(w, r)
	})
}

// overrideTargetsQuery reports whether the request's override header asks
// for exactly the QUERY method.
func overrideTargetsQuery(r *http.Request) bool {
	target := r.Header.Get(MethodOverrideHeader)
	if target == "" {
		target = r.Header.Get(legacyMethodOverrideHeader)
	}

	return target == MethodQuery
}
