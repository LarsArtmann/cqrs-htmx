package cqrshtmx

import (
	"net/http"
)

// HTMXScriptHandler returns an http.Handler that serves the embedded HTMX
// JavaScript (v2.0.9, minified) with appropriate Content-Type and long-lived
// cache headers. Mount it at any path, e.g.:
//
//	mux.Handle("/static/htmx.js", cqrshtmx.HTMXScriptHandler())
func HTMXScriptHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		w.Header().Set("ETag", `"htmx-2.0.9"`)
		if r.Header.Get("If-None-Match") == `"htmx-2.0.9"` {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		_, _ = w.Write(htmxJS)
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
