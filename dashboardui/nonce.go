package dashboardui

import (
	"net/http"

	"github.com/larsartmann/httputil"
)

// nonce resolves the per-request CSP nonce: Config.NonceFunc when the consumer
// supplied one, and httputil.NonceFromRequest otherwise (the value
// httputil.Nonce puts into the context). A nil request has no nonce — the
// error renderer supports being called without one.
func (d *Dashboard) nonce(r *http.Request) string {
	if r == nil {
		return ""
	}

	if d.config.NonceFunc != nil {
		return d.config.NonceFunc(r)
	}

	return httputil.NonceFromRequest(r)
}
