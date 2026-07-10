package cqrshtmx

import (
	"context"
	"net/http"
	"net/http/httptest"
)

// CSRFTestToken extracts a valid CSRF token AND cookie by making a GET request
// through the given middleware chain. The middleware must include CSRFMiddleware.
// CSRFResponseHeaderMiddleware is optional — without it, the token is
// extracted from the request context instead of the response header.
//
// nosurf uses token masking: the cookie value is NOT the same as the valid
// header token. A masked token is derived from the cookie per-request.
// This helper handles that dance automatically, returning both the masked
// token and the cookie that nosurf set. In a real cross-request scenario
// (GET → extract → POST), the POST needs BOTH the header token and the cookie.
//
// Usage:
//
//	mw := cqrshtmx.Chain(
//	    cqrshtmx.CSRFMiddleware(cqrshtmx.CSRFConfig{}),
//	    cqrshtmx.CSRFResponseHeaderMiddleware,
//	)
//	token, cookie := cqrshtmx.CSRFTestToken(mw)
//	// Use both in your POST request:
//	req := httptest.NewRequest("POST", "/", body)
//	req.Header.Set("X-CSRF-Token", token)
//	req.AddCookie(cookie)
func CSRFTestToken(middleware func(http.Handler) http.Handler) (string, *http.Cookie) {
	var ctxToken string

	handler := middleware(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		ctxToken = CSRFTokenFromContext(r.Context())
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	handler.ServeHTTP(w, r)

	var cookie *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == defaultCSRFCookieName {
			cookie = c
			break
		}
	}

	if hdr := w.Header().Get(defaultCSRFHeaderName); hdr != "" {
		return hdr, cookie
	}
	return ctxToken, cookie
}
