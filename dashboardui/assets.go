package dashboardui

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
)

//go:embed assets/dashboard-tw.css
var assetsFS embed.FS

// newAssetHandler reads an embedded asset once and returns a handler serving
// it via the root module's immutable-asset posture ([cqrshtmx.ServeAsset]:
// content-derived ETag, 1-year immutable caching, nosniff). The read happens
// at construction time so a missing asset surfaces as an error from [New]
// instead of a panic at first request. The asset-serving logic itself lives
// in the root module so adminui and dashboardui cannot drift apart (the
// stale-version-ETag incident class is now guarded in exactly one place).
func newAssetHandler(fsys fs.FS, name, contentType string) (http.Handler, error) {
	sub, err := fs.Sub(fsys, "assets")
	if err != nil {
		return nil, errConfig(fmt.Sprintf("asset subtree %q: %v", "assets", err))
	}

	data, err := fs.ReadFile(sub, name)
	if err != nil {
		return nil, errConfig(fmt.Sprintf("missing embedded asset %q: %v", name, err))
	}

	return cqrshtmx.ServeAsset(name, contentType, data), nil
}
