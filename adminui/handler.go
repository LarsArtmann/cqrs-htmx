package adminui

import (
	"net/http"

	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/httputil"
)

// Handler is a mounted admin panel. Build it with [New] and register it on a
// router with [Handler.Mount] or [Handler.Handler].
//
// The panel expects the consumer's session middleware to have placed the
// authenticated [*usermgmt.User] in the request context (see
// [usermgmt.NewSessionMiddleware]). Requests without an authenticated user, or
// users that fail [Config.Authorizer], receive 401/403.
type Handler struct {
	config Config
	nav    []navItem
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
	return &Handler{config: config, nav: buildNav(config.Mode)}, nil
}

// buildNav returns the sidebar entries for the given mode.
func buildNav(mode Mode) []navItem {
	if mode == ModeTenantAdmin {
		return []navItem{
			{Href: "/", Label: "Dashboard", Icon: iconDashboard},
			{Href: "/members", Label: "Members", Icon: iconMembers},
			{Href: "/audit", Label: "Audit log", Icon: iconAudit},
		}
	}
	return []navItem{
		{Href: "/", Label: "Dashboard", Icon: iconDashboard},
		{Href: "/users", Label: "Users", Icon: iconUsers},
		{Href: "/tenants", Label: "Tenants", Icon: iconTenants},
		{Href: "/audit", Label: "Audit log", Icon: iconAudit},
	}
}

// page builds the pageData shell for a page, marking the nav item whose Href
// matches active. Pass active="/" for the dashboard. r is used to extract the
// CSRF token (if a CSRF middleware is active); the token is empty otherwise.
func (h *Handler) page(title, active string, user *usermgmt.User, r *http.Request) pageData {
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

// nonce returns the per-request CSP nonce, or "" when no NonceFunc is set.
func (h *Handler) nonce(r *http.Request) string {
	if h.config.NonceFunc == nil {
		return ""
	}
	return h.config.NonceFunc(r)
}

// guard wraps a handler with authentication + authorization. The wrapped
// handler receives the authenticated user.
func (h *Handler) guard(fn func(http.ResponseWriter, *http.Request, *usermgmt.User)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, _ := usermgmt.UserFromContext(r.Context())
		if user == nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if err := h.config.Authorizer(user); err != nil {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		// Inject the now-validated user into the page data via context-free
		// closure: handlers pass it to page renderers directly.
		fn(w, r, user)
	}
}

// routes builds the internal route table (paths relative to BasePath) and
// returns a handler serving them.
func (h *Handler) routes() http.Handler {
	mux := http.NewServeMux()

	// --- Static assets ---
	mux.Handle("GET /-/admin-tw.css", assetHandler("admin-tw.css", "text/css; charset=utf-8"))
	mux.Handle("GET /-/admin.js", assetHandler("admin.js", "text/javascript; charset=utf-8"))
	mux.Handle("GET /-/htmx.js", htmxScriptHandler())
	mux.Handle("GET /-/sync-worker.js", syncWorkerHandler())
	mux.Handle("GET /-/sync-client.js", syncClientHandler())

	// --- Dashboard ---
	mux.HandleFunc("GET /{$}", h.guard(h.dashboard))

	// --- Users (super admin only) ---
	if h.config.Mode == ModeSuperAdmin {
		mux.HandleFunc("GET /users", h.guard(h.usersIndex))
		mux.HandleFunc("GET /users/{id}", h.guard(h.userDetail))
		mux.HandleFunc("POST /users/{id}/delete", h.guard(h.userDelete))
		mux.HandleFunc("POST /users/{id}/external/{provider}/unlink", h.guard(h.userUnlinkExternal))
	}

	// --- Tenants (super admin) ---
	if h.config.Mode == ModeSuperAdmin {
		mux.HandleFunc("GET /tenants", h.guard(h.tenantsIndex))
		mux.HandleFunc("GET /tenants/new", h.guard(h.tenantNew))
		mux.HandleFunc("POST /tenants", h.guard(h.tenantCreate))
		mux.HandleFunc("GET /tenants/{id}", h.guard(h.tenantDetail))
		mux.HandleFunc("POST /tenants/{id}/suspend", h.guard(h.tenantSuspend))
		mux.HandleFunc("POST /tenants/{id}/reactivate", h.guard(h.tenantReactivate))
		mux.HandleFunc("POST /tenants/{id}/delete", h.guard(h.tenantDelete))
		mux.HandleFunc("POST /tenants/{id}/members", h.guard(h.tenantAddMember))
		mux.HandleFunc("POST /tenants/{id}/members/{actor}/delete", h.guard(h.tenantRemoveMember))
		mux.HandleFunc("POST /tenants/{id}/members/{actor}", h.guard(h.tenantUpdateMemberRole))
	}

	// --- Members (tenant admin) ---
	if h.config.Mode == ModeTenantAdmin {
		mux.HandleFunc("GET /members", h.guard(h.membersIndex))
		mux.HandleFunc("POST /members", h.guard(h.membersAdd))
		mux.HandleFunc("POST /members/{actor}/delete", h.guard(h.membersRemove))
		mux.HandleFunc("POST /members/{actor}", h.guard(h.membersUpdateRole))
	}

	// --- Audit ---
	mux.HandleFunc("GET /audit", h.guard(h.auditIndex))

	return mux
}

// Handler returns an http.Handler serving the whole panel at root-relative
// paths. Mount it under a prefix with http.StripPrefix, or use [Handler.Mount].
func (h *Handler) Handler() http.Handler { return h.routes() }

// Mount registers the panel on mux at pattern (e.g. "/admin/"). A trailing
// slash is required by the standard mux for prefix matching. Use "/" to host
// the panel at the site root.
//
// The pattern is registered without a method, so it conflicts with a
// method-specific "GET /" catch-all on the same mux. Register any site-root
// index as "GET /{$}" or "/" (no method) to avoid a ServeMux panic.
func (h *Handler) Mount(mux *http.ServeMux, pattern string) {
	mux.Handle(pattern, http.StripPrefix(pattern, h.routes()))
}

// Middleware returns the standard middleware chain the panel recommends:
// panic recovery and baseline security headers (X-Content-Type-Options,
// X-Frame-Options, Referrer-Policy). It reuses [cqrshtmx.RecoveryMiddleware]
// and [cqrshtmx.SecurityHeadersMiddleware] from the root module.
//
// Wrap it around the panel — and compose your session and CSRF middleware:
//
//	panel.Mount(mux, "/admin/")
//	mux.Use(sessionMW, csrfMW, panel.Middleware()) // pseudo: chain as you prefer
//
// This is optional: the panel works without it, but recovery + security
// headers are recommended for any production deployment.
func (h *Handler) Middleware() func(http.Handler) http.Handler {
	return cqrshtmx.Chain(cqrshtmx.SecurityHeadersMiddleware, cqrshtmx.RecoveryMiddleware)
}
