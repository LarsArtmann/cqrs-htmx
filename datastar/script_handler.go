package datastar

import (
	"fmt"
	"net/http"
)

// ScriptHandler returns an http.Handler that serves the embedded Datastar
// JavaScript (v1.0.2) with appropriate Content-Type and long-lived cache
// headers. Mount it at any path, e.g.:
//
//	mux.Handle("GET /datastar.js", ds.ScriptHandler())
func ScriptHandler() http.Handler {
	return ScriptHandlerWith(datastarJS, datastarVersion)
}

// ScriptHandlerWith returns an http.Handler that serves the given Datastar
// JavaScript with appropriate Content-Type and long-lived cache headers.
// Use this when you want to serve a different Datastar version than the
// embedded default — e.g., a custom build or a newer release.
//
// The version string is used for the ETag header so conditional requests work
// correctly.
//
//	mux.Handle("/static/datastar.js",
//	    ds.ScriptHandlerWith(customJS, "1.1.0"))
func ScriptHandlerWith(js []byte, version string) http.Handler {
	return serveJS(js, fmt.Sprintf(`"datastar-%s"`, version))
}

// serveJS is the shared handler for serving JavaScript with long-lived caching.
// Mirrors the unexported serveJS in the root cqrs-htmx module — the function is
// duplicated intentionally to avoid importing the root module just for this
// 20-line helper (which would pull the entire HTMX dependency tree).
func serveJS(js []byte, etag string) http.Handler {
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

// ScriptTag returns an HTML <script type="module"> tag pointing at the given
// path, suitable for embedding in a template or templ component. Datastar
// requires type="module" unlike HTMX which uses a regular script tag.
//
//	ds.ScriptTag("/datastar.js")
//	// => <script type="module" src="/datastar.js"></script>
func ScriptTag(path string) string {
	return `<script type="module" src="` + path + `"></script>`
}

// Version returns the version string of the embedded Datastar JavaScript.
func Version() string { return datastarVersion }
