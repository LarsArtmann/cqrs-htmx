package cqrshtmx

import "net/http"

// HTMXRedirect issues an HTMX-aware redirect: HX-Redirect for HTMX requests
// (so the browser replaces the page without a full navigation round-trip), or
// a standard 303 See Other for normal navigation. The path should already be
// sanitized via [SafeRedirectPath] when it originates from user-controlled
// input.
func HTMXRedirect(w http.ResponseWriter, r *http.Request, path string) {
	if IsHTMXRequest(r) {
		w.Header().Set("Hx-Redirect", path)
		w.WriteHeader(http.StatusOK)

		return
	}

	http.Redirect(
		w,
		r,
		path,
		http.StatusSeeOther,
	)
}

// SafeRedirectPath returns path if it is a safe same-origin redirect target
// (starts with "/" but not "//" or "/\"), otherwise "/". Protocol-relative
// URLs ("//"), scheme-bearing URLs ("https://..."), and backslash variants
// ("/\\evil.com" — WHATWG URL parsing treats "\" as "/" in special schemes,
// so browsers would follow it off-site) are rejected because browsers would
// follow them off-site, enabling open-redirect attacks.
func SafeRedirectPath(path string) string {
	if path == "" || path[0] != '/' || len(path) > 1 && (path[1] == '/' || path[1] == '\\') {
		return "/"
	}

	return path
}
