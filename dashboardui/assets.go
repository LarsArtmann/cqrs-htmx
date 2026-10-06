package dashboardui

import (
	"bytes"
	"embed"
	"fmt"
	"hash/fnv"
	"io/fs"
	"net/http"
	"time"
)

//go:embed assets/dashboard-tw.css
var assetsFS embed.FS

// newAssetHandler reads an embedded asset once and returns a handler serving
// it with immutable long-lived caching (the root module's HTMXScriptHandler
// posture) and a content-security-policy-friendly content type. The read
// happens at construction time so a missing asset surfaces as an error from
// [New] instead of a panic at first request.
func newAssetHandler(fsys fs.FS, name, contentType string) (http.Handler, error) {
	sub, err := fs.Sub(fsys, "assets")
	if err != nil {
		return nil, errConfig(fmt.Sprintf("asset subtree %q: %v", "assets", err))
	}

	data, err := fs.ReadFile(sub, name)
	if err != nil {
		return nil, errConfig(fmt.Sprintf("missing embedded asset %q: %v", name, err))
	}

	tag := contentETag(name, data)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("ETag", tag)
		// Zero modtime disables Last-Modified; the ETag drives 304 responses.
		http.ServeContent(w, r, name, time.Time{}, bytes.NewReader(data))
	}), nil
}

// contentETag derives a strong ETag from the asset's bytes (FNV-1a 64), so
// the tag changes with the content instead of relying on a hand-bumped
// version constant (the stale dashboardui-v4.9.0 class: the compiled bundle
// changed, the const did not, and every consumer's cached copy silently
// diverged). Quoted per RFC 7232 — net/http's ETag matching requires a
// leading quote. The dashboard.js handler in layout.go shares this
// derivation so both served assets follow one rule.
func contentETag(name string, data []byte) string {
	h := fnv.New64a()
	_, _ = h.Write(data)

	return fmt.Sprintf(`"dashboardui-%s-%016x"`, name, h.Sum64())
}
