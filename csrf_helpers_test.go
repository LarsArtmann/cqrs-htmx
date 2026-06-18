package cqrshtmx_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"time"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v2"
)

func defaultCSRFConfig() cqrshtmx.CSRFConfig {
	return cqrshtmx.CSRFConfig{
		CookieName:     "",
		HeaderName:     "",
		FieldName:      "",
		MaxAge:         24 * time.Hour,
		Secure:         false,
		SameSite:       http.SameSiteLaxMode,
		Domain:         "",
		Path:           "/",
		TrustedOrigins: nil,
		ErrorHandler:   cqrshtmx.ForbiddenErrorHandler,
	}
}

// csrfConfigWith returns a copy of defaultCSRFConfig with the given overrides applied.
func csrfConfigWith(overrides func(*cqrshtmx.CSRFConfig)) cqrshtmx.CSRFConfig {
	cfg := defaultCSRFConfig()
	overrides(&cfg)
	return cfg
}

// csrfTokenOnceHandler wraps a middleware with a handler that captures the CSRF token once.
func csrfTokenOnceHandler(middleware func(http.Handler) http.Handler, token *string) http.Handler {
	return middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if *token == "" {
			*token = cqrshtmx.CSRFTokenFromContext(r.Context())
		}
		w.WriteHeader(http.StatusOK)
	}))
}

// csrfTokenCaptureHandler wraps a middleware with a handler that always captures the CSRF token.
func csrfTokenCaptureHandler(
	middleware func(http.Handler) http.Handler,
	token *string,
) http.Handler {
	return middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*token = cqrshtmx.CSRFTokenFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))
}

// csrfGETThenPOST performs the common CSRF test pattern:
// GET to obtain a masked token, then POST with the token in the specified header/field.
func csrfGETThenPOST(
	middleware func(http.Handler) http.Handler,
	headerName, fieldName string,
) int {
	var token string
	handler := csrfTokenOnceHandler(middleware, &token)

	w1 := httptest.NewRecorder()
	r1 := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	handler.ServeHTTP(w1, r1)

	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"/",
		strings.NewReader("{}"),
	)
	r2.Header.Set("Sec-Fetch-Site", "same-origin")
	if headerName != "" {
		r2.Header.Set(headerName, token)
	}
	if fieldName != "" {
		body := fieldName + "=" + url.QueryEscape(token)
		r2 = httptest.NewRequestWithContext(
			context.Background(),
			http.MethodPost,
			"/",
			strings.NewReader(body),
		)
		r2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		r2.Header.Set("Sec-Fetch-Site", "same-origin")
	}
	for _, c := range w1.Result().Cookies() {
		r2.AddCookie(c)
	}
	handler.ServeHTTP(w2, r2)

	return w2.Code
}
