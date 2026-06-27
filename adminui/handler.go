package adminui

import (
	"net/http"

	"github.com/larsartmann/cqrs-htmx/usermgmt/v3"
)

// Handler is a mounted admin panel. Build it with [New] and register it on a
// router with [Handler.Mount] or [Handler.Handler].
//
// The panel expects the consumer's session middleware to have placed the
// authenticated [*usermgmt.User] in the request context (see
// [usermgmt.NewSessionMiddleware]). Requests without an authenticated user, or
// users that fail [Config.Authorizer], receive 401/403.
type Handler struct {
	cfg Config
	nav []navItem
}

// New builds an admin panel from cfg, applying defaults to empty fields and
// validating the result. It returns an error only for invalid configuration
// (e.g. a nil Service).
func New(cfg Config) (*Handler, error) {
	cfg, err := cfg.withDefaults()
	if err != nil {
		return nil, err
	}
	if cfg.Authorizer == nil {
		cfg.Authorizer = defaultAuthorizer(cfg)
	}
	return &Handler{cfg: cfg, nav: buildNav(cfg.Mode)}, nil
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
// matches active. Pass active="/" for the dashboard.
func (h *Handler) page(title, active string, user *usermgmt.User) pageData {
	nav := make([]navItem, len(h.nav))
	for i, n := range h.nav {
		n.Active = n.Href == active
		nav[i] = n
	}
	return pageData{
		Title:     title,
		BasePath:  h.cfg.BasePath,
		Accent:    h.cfg.AccentColor,
		Brand:     h.cfg.Title,
		Nav:       nav,
		User:      user,
		LogoutURL: h.cfg.LogoutURL,
	}
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
		if err := h.cfg.Authorizer(user); err != nil {
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
	mux.Handle("GET /-/admin.css", assetHandler("admin.css", "text/css; charset=utf-8"))
	mux.Handle("GET /-/admin.js", assetHandler("admin.js", "text/javascript; charset=utf-8"))
	mux.Handle("GET /-/htmx.js", htmxScriptHandler())

	// --- Dashboard ---
	mux.HandleFunc("GET /{$}", h.guard(h.dashboard))

	// --- Users (super admin only) ---
	if h.cfg.Mode == ModeSuperAdmin {
		mux.HandleFunc("GET /users", h.guard(h.usersIndex))
		mux.HandleFunc("GET /users/{id}", h.guard(h.userDetail))
		mux.HandleFunc("POST /users/{id}/delete", h.guard(h.userDelete))
	}

	// --- Tenants (super admin) ---
	if h.cfg.Mode == ModeSuperAdmin {
		mux.HandleFunc("GET /tenants", h.guard(h.tenantsIndex))
		mux.HandleFunc("GET /tenants/new", h.guard(h.tenantNew))
		mux.HandleFunc("POST /tenants", h.guard(h.tenantCreate))
		mux.HandleFunc("GET /tenants/{id}", h.guard(h.tenantDetail))
		mux.HandleFunc("POST /tenants/{id}/suspend", h.guard(h.tenantSuspend))
		mux.HandleFunc("POST /tenants/{id}/reactivate", h.guard(h.tenantReactivate))
		mux.HandleFunc("POST /tenants/{id}/delete", h.guard(h.tenantDelete))
	}

	// --- Members (tenant admin) ---
	if h.cfg.Mode == ModeTenantAdmin {
		mux.HandleFunc("GET /members", h.guard(h.membersIndex))
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
func (h *Handler) Mount(mux *http.ServeMux, pattern string) {
	mux.Handle(pattern, http.StripPrefix(pattern, h.routes()))
}
