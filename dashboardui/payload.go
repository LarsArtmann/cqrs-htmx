package dashboardui

import (
	"net/http"
)

func csrfToken(r *http.Request) string {
	if token := r.Header.Get("X-Csrf-Token"); token != "" {
		return token
	}

	return r.FormValue("_csrf")
}
