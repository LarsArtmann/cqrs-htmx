package htmx

import (
	"fmt"
	"net/http"

	"github.com/larsartmann/cqrs-htmx/v4/internal/hx"
)

// ScriptHandler returns an http.Handler that serves the embedded HTMX
// JavaScript (v2.0.11, minified) with appropriate Content-Type and long-lived
// cache headers. Mount it at any path, e.g.:
//
//	mux.Handle("/static/htmx.js", cqrshtmx.ScriptHandler())
func ScriptHandler() http.Handler {
	return ScriptHandlerWith(htmxJS, htmxVersion)
}

// ScriptHandlerWith returns an http.Handler that serves the given HTMX
// JavaScript with appropriate Content-Type and long-lived cache headers.
// Use this when you want to serve a different HTMX version than the embedded
// default — e.g., a custom build, a beta, or htmx 4.0.
//
// The version string is used for the ETag header so conditional requests work
// correctly.
//
//	mux.Handle("/static/htmx.js",
//	    cqrshtmx.ScriptHandlerWith(customJS, "4.0.0-beta4"))
func ScriptHandlerWith(js []byte, version string) http.Handler {
	return hx.ServeJS(js, fmt.Sprintf(`"htmx-%s"`, version))
}

// ScriptTag returns an HTML <script> tag pointing at the given path,
// suitable for embedding in a template or templ component.
//
//	cqrshtmx.ScriptTag("/static/htmx.js")
//	// => <script src="/static/htmx.js"></script>
func ScriptTag(path string) string {
	return `<script src="` + path + `"></script>`
}

// CDNBaseURL is the default CDN base URL for loading HTMX.
const CDNBaseURL = "https://unpkg.com/htmx.org"

// CDNScriptTag returns an HTML <script> tag that loads HTMX from the unpkg
// CDN. Pass an empty version string to use the embedded library version.
//
//	cqrshtmx.CDNScriptTag("")       // uses embedded version (2.0.11)
//	cqrshtmx.CDNScriptTag("4.0.0")  // loads htmx 4.0.0
func CDNScriptTag(version string) string {
	if version == "" {
		version = htmxVersion
	}

	return fmt.Sprintf(`<script src="%s@%s"></script>`, CDNBaseURL, version)
}
