package dashboardui

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/a-h/templ"
	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/httputil"
	"github.com/larsartmann/templ-components/errorpage"
	"github.com/larsartmann/templ-components/icons"
)

// Handler returns the root HTTP handler for the dashboard. All internal
// routes are relative (no prefix). Use [Dashboard.Mount] if you want
// prefix stripping.
func (d *Dashboard) Handler() http.Handler { return d.routes() }

// Mount registers the dashboard under the given pattern on a ServeMux.
// The pattern is stripped from internal route paths.
//
//	mux := http.NewServeMux()
//	d.Mount(mux, "/dashboard/")
//
// The pattern is registered without a method (it matches all methods), so it
// conflicts with a method-specific "GET /" catch-all on the same mux. Register
// any site-root index as "GET /{$}" or "/" (no method) to avoid a ServeMux
// panic at registration.
func (d *Dashboard) Mount(mux *http.ServeMux, pattern string) {
	prefix := trimTrailingSlash(pattern)
	mux.Handle(pattern, http.StripPrefix(prefix, d.routes()))
}

// Middleware returns the recommended middleware chain for the dashboard.
// It delegates to [cqrshtmx.RecommendedSecurityMiddleware] so that the
// dashboard has the same security posture as adminui: security headers,
// per-request CSP nonce, and panic recovery.
func (d *Dashboard) Middleware() func(http.Handler) http.Handler {
	return cqrshtmx.RecommendedSecurityMiddleware()
}

func (d *Dashboard) routes() http.Handler {
	mux := http.NewServeMux()

	for _, spec := range d.routeTable() {
		if !spec.enabled(d.caps, d.config.ReadOnly) {
			continue
		}

		mux.Handle(spec.method+" "+registrationPattern(spec.pattern), d.wrapped(spec))
	}

	// Catch-all: styled 404 for any unmatched GET route under the dashboard.
	mux.HandleFunc("GET /", d.guard(d.notFoundHandler))

	return mux
}

// versionzRoute applies the optional versionz auth guard: with
// VersionzRequireAuth set, the endpoint goes through the configured
// Authorizer (403 on denial). Without the flag — or without an Authorizer —
// it stays public like the other observability probes.
func (d *Dashboard) versionzRoute() http.HandlerFunc {
	if !d.config.VersionzRequireAuth {
		return d.versionzHandler
	}

	return d.guard(d.versionzHandler)
}

// notFoundHandler renders a styled 404 page (templ-components errorpage):
// the bare component for HTMX swaps, the shell document with dashboard
// stylesheets otherwise.
func (d *Dashboard) notFoundHandler(w http.ResponseWriter, r *http.Request) {
	props := errorpage.DefaultNotFound404Props()
	props.GoHomeHref = d.config.BasePath + "/"
	props.Links = []errorpage.NotFoundLink{
		{Text: "Overview", Href: d.config.BasePath + "/", Icon: icons.Home},
	}

	var b strings.Builder
	if err := errorpage.NotFound404(props).Render(r.Context(), &b); err != nil {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("404 page not found\n"))

		return
	}

	w.Header().Set("Content-Type", contentTypeHTML)
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusNotFound)

	if cqrshtmx.IsHTMXRequest(r) {
		_, _ = w.Write([]byte(b.String()))

		return
	}

	if err := errorShell(
		props.Title,
		d.config.BasePath,
		httputil.NonceFromRequest(r),
		templ.Raw(b.String()),
	).Render(r.Context(), w); err != nil {
		slog.ErrorContext(r.Context(), "dashboardui: render 404 shell", "error", err)
	}
}
