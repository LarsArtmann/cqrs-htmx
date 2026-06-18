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

// csrfTokenCaptureHandler wraps a middleware with a handler that captures the
// CSRF token into *token. When captureOnce is true, the token is captured only
// the first time (subsequent captures keep the original value).
func csrfTokenCaptureHandler(
	middleware func(http.Handler) http.Handler,
	token *string,
	captureOnce bool,
) http.Handler {
	return middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if *token == "" || !captureOnce {
			*token = cqrshtmx.CSRFTokenFromContext(r.Context())
		}
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
	handler := csrfTokenCaptureHandler(middleware, &token, true)

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
