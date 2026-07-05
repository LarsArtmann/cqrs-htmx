package cqrshtmx

import (
	"context"
	"net/http"
	"net/http/httptest"
)

// CSRFTestToken extracts a valid CSRF token by making a GET request through
// the given middleware chain. The middleware must include CSRFMiddleware.
// CSRFResponseHeaderMiddleware is optional — without it, the token is
// extracted from the request context instead of the response header.
//
// nosurf uses token masking: the cookie value is NOT the same as the valid
// header token. A masked token is derived from the cookie per-request.
// This helper handles that dance automatically, so test code can just do:
//
//	mw := cqrshtmx.Chain(
//	    cqrshtmx.CSRFMiddleware(cqrshtmx.CSRFConfig{}),
//	    cqrshtmx.CSRFResponseHeaderMiddleware,
//	)
//	token := cqrshtmx.CSRFTestToken(mw)
//	// Now use token in your POST request's X-CSRF-Token header:
//	req := httptest.NewRequest("POST", "/", body)
//	req.Header.Set("X-CSRF-Token", token)
func CSRFTestToken(middleware func(http.Handler) http.Handler) string {
	var ctxToken string

	handler := middleware(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		ctxToken = CSRFTokenFromContext(r.Context())
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	handler.ServeHTTP(w, r)

	if hdr := w.Header().Get(defaultCSRFHeaderName); hdr != "" {
		return hdr
	}
	return ctxToken
}
