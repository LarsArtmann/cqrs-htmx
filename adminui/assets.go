package adminui

import (
	"bytes"
	"embed"
	"io/fs"
	"net/http"
	"time"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v3"
)

//go:embed assets/admin-tw.css assets/admin.js
var assetsFS embed.FS

// assetHandler serves a single embedded file with long-lived caching and a
// content-security-policy-friendly content type.
func assetHandler(name, contentType string) http.Handler {
	sub, _ := fs.Sub(assetsFS, "assets")
	data, err := fs.ReadFile(sub, name)
	if err != nil {
		// Guarded by go:embed at compile time; unreachable.
		panic("adminui: missing embedded asset " + name)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Cache-Control", "public, max-age=86400")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("ETag", assetETag)
		// Zero modtime disables Last-Modified; ETag still drives 304 responses.
		http.ServeContent(w, r, name, time.Time{}, bytes.NewReader(data))
	})
}

// assetETag lets ServeContent answer If-None-Match with a 304.
const assetETag = `adminui-v2`

// htmxScriptHandler serves the embedded HTMX script (v2.0.9) from the root
// cqrs-htmx module, so the panel is fully self-contained.
func htmxScriptHandler() http.Handler { return cqrshtmx.HTMXScriptHandler() }
