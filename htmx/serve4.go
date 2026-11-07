package htmx

import "net/http"

// ScriptHandler4 returns an http.Handler that serves the embedded HTMX 4
// JavaScript (v4.0.1, minified) with the same Content-Type, ETag, and
// long-lived cache headers as [ScriptHandler]. Mount it wherever your htmx-4
// pages load their script — the two versions can coexist in one binary:
//
//	mux.Handle("/static/htmx.js", htmx.ScriptHandler())   // 2.0.11 default
//	mux.Handle("/static/htmx4.js", htmx.ScriptHandler4()) // hx-query native
//
// htmx 4 issues native QUERY requests via `hx-query`; htmx 2 consumers keep
// the POST + X-HTTP-Method-Override fallback (see the root module's
// MethodOverrideMiddleware and docs/guides/http-query-method.md).
func ScriptHandler4() http.Handler {
	return ScriptHandlerWith(htmx4JS, HTMX4Version)
}
