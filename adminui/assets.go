package adminui

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"time"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
)

//go:embed assets/admin-tw.css assets/admin.js
var assetsFS embed.FS

// newAssetHandler reads an embedded asset once and returns a handler serving
// it with long-lived caching and a content-security-policy-friendly content
// type. The read happens at construction time so a missing asset surfaces as
// an error from [New] instead of a panic at first request.
func newAssetHandler(fsys fs.FS, name, contentType string) (http.Handler, error) {
	sub, err := fs.Sub(fsys, "assets")
	if err != nil {
		return nil, errConfig(fmt.Sprintf("asset subtree %q: %v", "assets", err))
	}

	data, err := fs.ReadFile(sub, name)
	if err != nil {
		return nil, errConfig(fmt.Sprintf("missing embedded asset %q: %v", name, err))
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Cache-Control", "public, max-age=86400")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("ETag", assetETag)
		// Zero modtime disables Last-Modified; ETag still drives 304 responses.
		http.ServeContent(w, r, name, time.Time{}, bytes.NewReader(data))
	}), nil
}

// assetETag lets ServeContent answer If-None-Match with a 304. Quoted per
// RFC 7232 — net/http's etagStrongMatch requires a leading quote.
const assetETag = `"adminui-v3.4.0"`

// htmxScriptHandler serves the embedded HTMX script (v2.0.10) from the root
// cqrs-htmx module, so the panel is fully self-contained.
func htmxScriptHandler() http.Handler { return cqrshtmx.HTMXScriptHandler() }

// syncWorkerHandler serves the offline sync SharedWorker from the root module.
func syncWorkerHandler() http.Handler { return cqrshtmx.SyncWorkerHandler() }

// syncClientHandler serves the offline sync tab-side client from the root module.
func syncClientHandler() http.Handler { return cqrshtmx.SyncClientHandler() }
