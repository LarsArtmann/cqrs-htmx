package cqrshtmx

import (
	"encoding/json"
	"html"
	"net/http"
)

// csrfTokenFormatted returns a formatted string using the CSRF token from the
// request, or an empty string if no token is present. The format function
// receives the HTML-escaped token.
func csrfTokenFormatted(r *http.Request, format func(escaped string) string) string {
	token := csrfTokenFromRequest(r)
	if token == "" {
		return ""
	}
	return format(html.EscapeString(token))
}

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
	return csrfTokenFormatted(r, func(tok string) string {
		return `<meta name="csrf-token" content="` + tok + `">`
	})
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
	jsonVal, err := json.Marshal(map[string]string{"X-CSRF-Token": token})
	if err != nil {
		return ""
	}
	return `hx-headers='` + html.EscapeString(string(jsonVal)) + `'`
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
	return csrfTokenFormatted(r, func(tok string) string {
		return `<input type="hidden" name="` + html.EscapeString(
			defaultCSRFFieldName,
		) + `" value="` + tok + `">`
	})
}
