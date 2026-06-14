package cqrshtmx

import (
	"fmt"
	"net/http"
)

// HTMXScriptHandler returns an http.Handler that serves the embedded HTMX
// JavaScript (v2.0.9, minified) with appropriate Content-Type and long-lived
// cache headers. Mount it at any path, e.g.:
//
//	mux.Handle("/static/htmx.js", cqrshtmx.HTMXScriptHandler())
func HTMXScriptHandler() http.Handler {
	return HTMXScriptHandlerWith(htmxJS, htmxVersion)
}

// HTMXScriptHandlerWith returns an http.Handler that serves the given HTMX
// JavaScript with appropriate Content-Type and long-lived cache headers.
// Use this when you want to serve a different HTMX version than the embedded
// default — e.g., a custom build, a beta, or htmx 4.0.
//
// The version string is used for the ETag header so conditional requests work
// correctly.
//
//	mux.Handle("/static/htmx.js",
//	    cqrshtmx.HTMXScriptHandlerWith(customJS, "4.0.0-beta4"))
func HTMXScriptHandlerWith(js []byte, version string) http.Handler {
	etag := fmt.Sprintf(`"htmx-%s"`, version)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		w.Header().Set("ETag", etag)
		if r.Header.Get("If-None-Match") == etag {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		_, _ = w.Write(js)
	})
}

// HTMXScriptTag returns an HTML <script> tag pointing at the given path,
// suitable for embedding in a template or templ component.
//
//	cqrshtmx.HTMXScriptTag("/static/htmx.js")
//	// => <script src="/static/htmx.js"></script>
func HTMXScriptTag(path string) string {
	return `<script src="` + path + `"></script>`
}

// HTMXCDNBaseURL is the default CDN base URL for loading HTMX.
const HTMXCDNBaseURL = "https://unpkg.com/htmx.org"

// HTMXCDNScriptTag returns an HTML <script> tag that loads HTMX from the unpkg
// CDN. Pass an empty version string to use the embedded library version.
//
//	cqrshtmx.HTMXCDNScriptTag("")       // uses embedded version (2.0.9)
//	cqrshtmx.HTMXCDNScriptTag("4.0.0")  // loads htmx 4.0.0
func HTMXCDNScriptTag(version string) string {
	if version == "" {
		version = htmxVersion
	}
	return fmt.Sprintf(`<script src="%s@%s"></script>`, HTMXCDNBaseURL, version)
}
