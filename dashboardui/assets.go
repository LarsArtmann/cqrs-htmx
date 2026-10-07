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
// instead of a panic at first request. The read ([cqrshtmx.AssetFromFS]) and
// serving logic live in the root module so adminui and dashboardui cannot
// drift apart (the stale-version-ETag incident class is guarded in exactly
// one place); only the embed declaration and error presentation stay local.
func newAssetHandler(fsys fs.FS, name, contentType string) (http.Handler, error) {
	data, err := cqrshtmx.AssetFromFS(fsys, name)
	if err != nil {
		return nil, errConfig(fmt.Sprintf("embedded asset %q: %v", name, err))
	}

	return cqrshtmx.ServeAsset(name, contentType, data), nil
}
