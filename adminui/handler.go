package adminui

import (
	"net/http"
	"strings"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/httputil"
	"github.com/larsartmann/templ-components/icons"
)

// Handler is a mounted admin panel. Build it with [New] and register it on a
// router with [Handler.Mount] or [Handler.Handler].
//
// The panel expects the consumer's session middleware to have placed the
// authenticated [*identitymodel.User] in the request context (see
// [usermgmt.NewSessionMiddleware]). Requests without an authenticated user, or
// users that fail [Config.Authorizer], receive 401/403.
type Handler struct {
	config   Config
	nav      []navItem
	adminCSS http.Handler
	adminJS  http.Handler
	table    []routeSpec
}

// New builds an admin panel from config, applying defaults to empty fields and
// validating the result. It returns an error only for invalid configuration
// (e.g. a nil Service).
func New(config Config) (*Handler, error) {
	config, err := config.withDefaults()
	if err != nil {
		return nil, err
	}
	if config.Authorizer == nil {
		config.Authorizer = defaultAuthorizer(config)
	}

	adminCSS, err := newAssetHandler(assetsFS, "admin-tw.css", "text/css; charset=utf-8")
	if err != nil {
		return nil, err
	}

	adminJS, err := newAssetHandler(assetsFS, "admin.js", "text/javascript; charset=utf-8")
	if err != nil {
		return nil, err
	}

	h := &Handler{ //nolint:exhaustruct // table is built and validated immediately below
		config:   config,
		nav:      buildNav(config),
		adminCSS: adminCSS,
		adminJS:  adminJS,
	}
	h.table = h.buildRouteTable()
	if err := validateRouteTable(h.table); err != nil {
		return nil, err
	}

	return h, nil
}

// buildNav returns the sidebar entries: the mode's built-in sections, then
// the consumer's ExtraNav entries (verbatim hrefs), then the optional
// dashboardui cross-link. ExtraNav precedes the cross-link so consumer
// ordering is stable regardless of DashboardURL.
func buildNav(config Config) []navItem {
	var nav []navItem
	if config.Mode == ModeTenantAdmin {
		nav = []navItem{
			{Href: "/", Label: "Dashboard", Icon: iconDashboard},
			{Href: "/members", Label: "Members", Icon: iconMembers},
			{Href: "/audit", Label: "Audit log", Icon: iconAudit},
		}
	} else {
		nav = []navItem{
			{Href: "/", Label: "Dashboard", Icon: iconDashboard},
			{Href: "/users", Label: "Users", Icon: iconUsers},
			{Href: "/tenants", Label: "Tenants", Icon: iconTenants},
			{Href: "/audit", Label: "Audit log", Icon: iconAudit},
		}
	}
	for _, entry := range config.ExtraNav {
		iconKey := string(entry.Icon)
		if iconKey == "" {
			iconKey = string(icons.Link)
		}
		nav = append(nav, navItem{Href: entry.Href, Label: entry.Label, Icon: iconKey, Verbatim: true})
	}
	if config.DashboardURL != "" {
		nav = append(nav, navItem{
			Href: config.DashboardURL, Label: "Event store", Icon: string(icons.Chart), Verbatim: true,
		})
	}
	return nav
}

// page builds the pageData shell for a page, marking the nav item whose Href
// matches active. Pass active="/" for the dashboard. r is used to extract the
// CSRF token (if a CSRF middleware is active); the token is empty otherwise.
func (h *Handler) page(title, active string, user *identitymodel.User, r *http.Request) pageData {
	nav := make([]navItem, len(h.nav))
	for i, n := range h.nav {
		n.Active = n.Href == active
		nav[i] = n
	}
	return pageData{
		Title:     title,
		BasePath:  h.config.BasePath,
		Accent:    h.config.AccentColor,
		Brand:     h.config.Title,
		Nav:       nav,
		User:      user,
		LogoutURL: h.config.LogoutURL,
		SSEURL:    h.config.SSEURL,
		CSRFToken: httputil.CSRFTokenFormField(r),
		CSRFMeta:  httputil.CSRFTokenHTMLMeta(r),
		Nonce:     h.nonce(r),
	}
}

// nonce returns the per-request CSP nonce. When Config.NonceFunc is set
// (backward compatibility), it takes precedence. Otherwise the nonce is read
// from the request context via httputil.NonceFromRequest, which works
// automatically when the consumer adds httputil.Nonce middleware (included in
// the recommended Middleware() chain). Returns "" when neither is available.
func (h *Handler) nonce(r *http.Request) string {
	if h.config.NonceFunc != nil {
		return h.config.NonceFunc(r)
	}
	return httputil.NonceFromRequest(r)
}

// guard wraps a handler with authentication + authorization. The wrapped
// handler receives the authenticated user.
func (h *Handler) guard(fn func(http.ResponseWriter, *http.Request, *identitymodel.User)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, _ := usermgmt.UserFromContext(r.Context())
		if user == nil {
			h.writeErrorPage(
				w,
				r,
				http.StatusUnauthorized,
				"Sign in required",
				"Your session is missing or expired. Sign in again to continue.",
			)
			return
		}
		if err := h.config.Authorizer(user); err != nil {
			h.writeErrorPage(w, r, http.StatusForbidden, "Access denied", "Your role does not allow this action.")
			return
		}
		// Inject the now-validated user into the page data via context-free
		// closure: handlers pass it to page renderers directly.
		fn(w, r, user)
	}
}

// routes builds the panel's route table (paths relative to BasePath) and
// returns a handler serving it. The table — not this function — is the
// single source of truth; [Handler.Routes] manifests the same rows, so a
// route outside the table cannot exist.
func (h *Handler) routes() http.Handler {
	mux := http.NewServeMux()
	h.registerRoutes(mux)
	return mux
}

// Handler returns an http.Handler serving the whole panel at root-relative
// paths. Mount it under a prefix with http.StripPrefix, or use [Handler.Mount].
func (h *Handler) Handler() http.Handler { return h.routes() }

// Config returns the resolved configuration (defaults applied) behind the
// panel. Read-only snapshot for inspection and tests.
func (h *Handler) Config() Config { return h.config }

// Mount registers the panel on mux at pattern (e.g. "/admin/"). A trailing
// slash is required by the standard mux for prefix matching. Use "/" to host
// the panel at the site root.
//
// The pattern is registered without a method, so it conflicts with a
// method-specific "GET /" catch-all on the same mux. Register any site-root
// index as "GET /{$}" or "/" (no method) to avoid a ServeMux panic.
func (h *Handler) Mount(mux *http.ServeMux, pattern string) {
	// Strip the prefix WITHOUT its trailing slash: StripPrefix("/admin/")
	// would turn "/admin/users" into the relative path "users", which the
	// inner ServeMux's path cleaning then answers with a redirect to
	// "/users" (301 on Go <= 1.21, 307 since) — every panel route would
	// bounce. Stripping "/admin" keeps the leading slash: "/admin/users"
	// becomes "/users" and matches the inner patterns directly. The root
	// mount ("/" pattern) trims to "" and StripPrefix returns the panel
	// unchanged, which is correct.
	mux.Handle(pattern, http.StripPrefix(strings.TrimSuffix(pattern, "/"), h.routes()))
}

// Middleware returns the standard middleware chain the panel recommends:
// security headers (X-Content-Type-Options, X-Frame-Options, Referrer-Policy,
// Permissions-Policy), per-request CSP nonce, and panic recovery.
//
// It delegates to [cqrshtmx.RecommendedSecurityMiddleware] so that the panel
// has the same security posture as dashboardui.
//
// Wrap it around the panel — and compose your session and CSRF middleware:
//
//	panel.Mount(mux, "/admin/")
//	mux.Use(sessionMW, csrfMW, panel.Middleware()) // pseudo: chain as you prefer
//
// This is optional: the panel works without it, but recovery + security
// headers are recommended for any production deployment.
func (h *Handler) Middleware() func(http.Handler) http.Handler {
	return cqrshtmx.RecommendedSecurityMiddleware()
}
