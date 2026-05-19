package cqrshtmx

import (
	"html"
	"net/http"
)

// CSRFTokenHTMLMeta returns an HTML meta tag containing the CSRF token.
// Use this in your HTML templates to make the token available to JavaScript
// or HTMX without manual attribute construction.
//
//	<head>
//	    {{ cqrshtmx.CSRFTokenHTMLMeta .Request | safeHTML }}
//	</head>
//
// The token is HTML-escaped to prevent XSS. If no token is in the
// request context (CSRFMiddleware not applied), returns an empty string.
func CSRFTokenHTMLMeta(r *http.Request) string {
	token := csrfTokenFromRequest(r)
	if token == "" {
		return ""
	}
	return `<meta name="csrf-token" content="` + html.EscapeString(token) + `">`
}

// CSRFTokenHXHeaders returns an HTMX hx-headers attribute with the CSRF token.
// Apply this to your <body> tag or any element that makes HTMX requests.
//
//	<body {{ cqrshtmx.CSRFTokenHXHeaders .Request | safeAttr }} >
//
// Or in Go handler code before template rendering:
//
//	data["HXHeaders"] = cqrshtmx.CSRFTokenHXHeaders(r)
//
// The token is HTML-escaped to prevent XSS. Returns an empty string if no
// token is present in context.
func CSRFTokenHXHeaders(r *http.Request) string {
	token := csrfTokenFromRequest(r)
	if token == "" {
		return ""
	}
	return `hx-headers='{"X-CSRF-Token":"` + html.EscapeString(token) + `"}'`
}

// CSRFTokenFormField returns a hidden input HTML element containing the CSRF token.
// Use this for standard HTML forms (non-HTMX) that submit via POST.
//
//	<form method="POST" action="/game/create">
//	    {{ cqrshtmx.CSRFTokenFormField .Request | safeHTML }}
//	    <!-- other form fields -->
//	</form>
//
// The token is HTML-escaped. Returns an empty string if no token is present.
func CSRFTokenFormField(r *http.Request) string {
	token := csrfTokenFromRequest(r)
	if token == "" {
		return ""
	}
	return `<input type="hidden" name="` + html.EscapeString(
		defaultCSRFFieldName,
	) + `" value="` + html.EscapeString(
		token,
	) + `">`
}
