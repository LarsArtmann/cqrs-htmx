package cqrshtmx

import (
	"bytes"
	"io/fs"
	"net/http"
	"time"
)

// AssetETag derives the ETag ServeAsset sets for the given asset name and
// bytes: a quoted, content-derived FNV-1a hash (RFC 7232 strong validator —
// net/http's If-None-Match matching requires the leading quote). The tag
// changes whenever the name or content changes, so a rebuilt asset is always
// refetched (the stale hand-bumped version constant class: the compiled
// bundle changed, the const did not, and every consumer's cached copy
// silently diverged). Exposed for handlers and tests that need to reference
// the served tag out-of-band.
func AssetETag(name string, data []byte) string {
	return `"` + name + "-" + hashTag(data) + `"`
}

// ServeAsset returns an http.Handler serving pre-loaded asset bytes with the
// library's immutable-asset posture: a content-derived ETag ([AssetETag]),
// 1-year immutable Cache-Control, and X-Content-Type-Options: nosniff — the
// same rule [HTMXScriptHandler] applies to htmx.js, generalized to any
// embedded asset (CSS, JS, fonts, ...). The zero modtime disables
// Last-Modified so the ETag alone drives 304 responses; http.ServeContent
// answers conditional requests and byte ranges.
//
// Read the bytes eagerly (e.g. via fs.ReadFile over an embed.FS subtree) so
// a missing asset surfaces as an error at construction time instead of a
// panic at first request:
//
//	data, err := fs.ReadFile(assetsFS, "assets/app.css")
//	// ... handle err ...
//	mux.Handle("GET /-/app.css", cqrshtmx.ServeAsset("app.css", "text/css; charset=utf-8", data))
func ServeAsset(name, contentType string, data []byte) http.Handler {
	tag := AssetETag(name, data)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("ETag", tag)
		http.ServeContent(w, r, name, time.Time{}, bytes.NewReader(data))
	})
}

// AssetFromFS reads name from the "assets" subtree of fsys — the read half
// of the embed-asset recipe shared by every UI module (each module must
// declare its own embed.FS, but the subtree + ReadFile plumbing lives here
// so the modules cannot drift). Failures surface as Infrastructure-family
// errors — a missing embed is a packaging defect, not user error — so
// callers can pass them through verbatim. Read eagerly so a missing asset
// surfaces at construction time instead of a panic at first request.
func AssetFromFS(fsys fs.FS, name string) ([]byte, error) {
	sub, err := fs.Sub(fsys, "assets")
	if err != nil {
		return nil, errorfamily.NewInfrastructure("asset_fs", fmt.Sprintf("asset subtree %q: %v", "assets", err))
	}

	data, err := fs.ReadFile(sub, name)
	if err != nil {
		return nil, errorfamily.NewInfrastructure("asset_fs", fmt.Sprintf("embedded asset %q: %v", name, err))
	}

	return data, nil
}
