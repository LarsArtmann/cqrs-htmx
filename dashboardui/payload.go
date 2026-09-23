package dashboardui

import (
	"net/http"
)

// csrfToken propagates the consumer's CSRF token from the incoming request
// into pageData so rendered forms can embed it (htmx.CSRFToken posts it as
// "csrf_token", httputil.CSRFMiddleware's default field). The "_csrf" form
// fallback keeps consumers working who configured FieldName accordingly.
func csrfToken(r *http.Request) string {
	if token := r.Header.Get("X-Csrf-Token"); token != "" {
		return token
	}

	if token := r.FormValue("csrf_token"); token != "" {
		return token
	}

	return r.FormValue("_csrf")
}
