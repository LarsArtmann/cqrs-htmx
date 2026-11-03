package adminui

import (
	"net/http"
	"slices"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
)

// Panel names used across the route table and the [Handler.Routes] manifest.
const (
	panelAssets    = "Assets"
	panelDashboard = "Dashboard"
	panelUsers     = "Users"
	panelTenants   = "Tenants"
	panelMembers   = "Members"
	panelAudit     = "Audit log"
	panelFallback  = "Fallback"
)

// Shared route patterns (referenced by the table and the sidebar nav).
const (
	patternTenants = "/tenants"
	patternMembers = "/members"
	patternAudit   = "/audit"
)

// Route is one row of the admin panel's route manifest — behind
// [Handler.Routes]. The manifest honors this instance's Mode (mode-scoped
// rows are absent in the other mode) and this instance's authorizer posture.
type Route struct {
	// Method is the HTTP method the route answers, or "*" for any.
	Method string

	// Pattern is the panel-relative path; wildcards in braces.
	Pattern string

	// Panel names the section the route belongs to.
	Panel string

	// Write marks a mutating route.
	Write bool

	// Posture is what the route requires of its caller, per this instance's
	// configuration (see [Config.AuthorizerPosture] for the custom-authorizer
	// case).
	Posture cqrshtmx.Posture

	// Why is the reason the posture is correct.
	Why string
}

// String renders the route as "METHOD /pattern".
func (r Route) String() string { return r.Method + " " + r.Pattern }

// routeMode scopes a table row to the panel modes that serve it. The zero
// value is invalid on purpose: a row that forgets its mode is served by NO
// mode (and drops out of the manifest) instead of silently appearing in both
// — absence fails loudly at runtime, over-exposure fails only in an audit.
type routeMode uint8

const (
	routeModeInvalid routeMode = iota
	routeModeBoth
	routeModeSuperAdmin
	routeModeTenantAdmin
)

// servesIn reports whether the row belongs to mode.
func (m routeMode) servesIn(mode Mode) bool {
	switch m {
	case routeModeBoth:
		return true
	case routeModeSuperAdmin:
		return mode == ModeSuperAdmin
	case routeModeTenantAdmin:
		return mode == ModeTenantAdmin
	case routeModeInvalid:
		return false // a forgotten mode is served by NO mode, not both
	}

	return false
}

// routeSpec is one row of the admin panel's route table — the single source
// of truth the mux registration ([Handler.routes]) and the manifest
// ([Handler.Routes]) are both generated from. Exactly one of guarded / asset
// is set, validated at [New].
type routeSpec struct {
	method  string // "*" registers without a method prefix (any method)
	pattern string
	panel   string

	// guarded is the session+authorizer-gated handler shape.
	guarded func(http.ResponseWriter, *http.Request, *identitymodel.User)

	// asset is served unguarded (static files, method-refusals).
	asset http.Handler

	mode  routeMode
	write bool
	why   string
}

// buildRouteTable returns the complete route surface in registration order.
// It is built once at [New] and validated there.
func (h *Handler) buildRouteTable() []routeSpec {
	routes := append(h.sharedRoutes(), h.superAdminRoutes()...)
	routes = append(routes, h.tenantAdminRoutes()...)

	return append(routes, h.fallbackRoutes()...)
}

// sharedRoutes are served in both modes: static assets, the dashboard, the
// audit trail, and the fallbacks.
//
//nolint:exhaustruct // rows intentionally omit inapplicable fields (asset vs guarded, write)
func (h *Handler) sharedRoutes() []routeSpec {
	return []routeSpec{
		// --- Static assets (unguarded: no data) ---
		{
			method: http.MethodGet, pattern: "/-/admin-tw.css", panel: panelAssets, asset: h.adminCSS,
			mode: routeModeBoth, why: "static stylesheet, no data",
		},
		{
			method: http.MethodGet, pattern: "/-/admin.js", panel: panelAssets, asset: h.adminJS,
			mode: routeModeBoth, why: "static script, no data",
		},
		{
			method: http.MethodGet, pattern: "/-/htmx.js", panel: panelAssets, asset: htmxScriptHandler(),
			mode: routeModeBoth, why: "vendored HTMX, no data",
		},
		{
			method: http.MethodGet, pattern: "/-/sync-worker.js", panel: panelAssets, asset: syncWorkerHandler(),
			mode: routeModeBoth, why: "static sync worker, no data",
		},
		{
			method: http.MethodGet, pattern: "/-/sync-client.js", panel: panelAssets, asset: syncClientHandler(),
			mode: routeModeBoth, why: "static sync client, no data",
		},

		// --- Dashboard (both modes, guarded) ---
		{
			method: http.MethodGet, pattern: "/", panel: panelDashboard, guarded: h.dashboard,
			mode: routeModeBoth, why: "the dashboard summarizes every identity in the system",
		},
		{
			method: http.MethodGet, pattern: "/partials/stats", panel: panelDashboard, guarded: h.statsPartial,
			mode: routeModeBoth, why: "the stats partial summarizes every identity in the system",
		},

		// --- Audit (both modes, guarded) ---
		{
			method: http.MethodGet, pattern: patternAudit, panel: panelAudit, guarded: h.auditIndex,
			mode: routeModeBoth, why: "the audit trail names every actor and command",
		},
	}
}

// fallbackRoutes close the surface: unknown pages stay behind the gate, and
// unmatched methods answer 405.
//
//nolint:exhaustruct // rows intentionally omit inapplicable fields (asset vs guarded, write)
func (h *Handler) fallbackRoutes() []routeSpec {
	return []routeSpec{
		{
			method:  http.MethodGet,
			pattern: "/*",
			panel:   panelFallback,
			guarded: h.notFoundGuarded(),
			mode:    routeModeBoth,
			why:     "unknown pages stay behind the gate: the 404 page itself reveals the panel exists",
		},
		{
			method: "*", pattern: "/*", panel: panelFallback, asset: h.methodNotAllowed(),
			mode: routeModeBoth, why: "unmatched methods answer 405 regardless of caller — nothing to protect",
		},
	}
}

// superAdminRoutes are the platform-scope surfaces, served only in
// [ModeSuperAdmin]: user administration, the tenant lifecycle, and tenant
// membership writes.
func (h *Handler) superAdminRoutes() []routeSpec {
	return slices.Concat(
		h.superAdminUserRoutes(),
		h.superAdminTenantRoutes(),
		h.superAdminTenantMemberRoutes(),
	)
}

// superAdminUserRoutes administer users across every tenant.
//
//nolint:exhaustruct // rows intentionally omit inapplicable fields (asset vs guarded, write)
func (h *Handler) superAdminUserRoutes() []routeSpec {
	return []routeSpec{
		{
			method: http.MethodGet, pattern: "/users", panel: panelUsers, guarded: h.usersIndex,
			mode: routeModeSuperAdmin, why: "the user list spans every tenant",
		},
		{
			method: http.MethodGet, pattern: "/users/{id}", panel: panelUsers, guarded: h.userDetail,
			mode: routeModeSuperAdmin, why: "user detail exposes credentials metadata",
		},
		{
			method: http.MethodPost, pattern: "/users/{id}/delete", panel: panelUsers, guarded: h.userDelete,
			mode: routeModeSuperAdmin, write: true, why: "deleting users is platform-admin territory",
		},
		{
			method:  http.MethodPost,
			pattern: "/users/{id}/external/{provider}/unlink",
			panel:   panelUsers,
			guarded: h.userUnlinkExternal,
			mode:    routeModeSuperAdmin,
			write:   true,
			why:     "unlinking a user's external account changes how they authenticate",
		},
	}
}

// superAdminTenantRoutes own the tenant lifecycle.
//
//nolint:exhaustruct // rows intentionally omit inapplicable fields (asset vs guarded, write)
func (h *Handler) superAdminTenantRoutes() []routeSpec {
	return []routeSpec{
		{
			method: http.MethodGet, pattern: patternTenants, panel: panelTenants, guarded: h.tenantsIndex,
			mode: routeModeSuperAdmin, why: "the tenant list is the platform roster",
		},
		{
			method: http.MethodGet, pattern: "/tenants/new", panel: panelTenants, guarded: h.tenantNew,
			mode: routeModeSuperAdmin, why: "tenant creation form",
		},
		{
			method: http.MethodPost, pattern: patternTenants, panel: panelTenants, guarded: h.tenantCreate,
			mode: routeModeSuperAdmin, write: true, why: "creating tenants is platform-admin territory",
		},
		{
			method: http.MethodGet, pattern: "/tenants/{id}", panel: panelTenants, guarded: h.tenantDetail,
			mode: routeModeSuperAdmin, why: "tenant detail exposes membership rosters",
		},
		{
			method: http.MethodPost, pattern: "/tenants/{id}/suspend", panel: panelTenants, guarded: h.tenantSuspend,
			mode: routeModeSuperAdmin, write: true, why: "suspending a tenant disables every member of it",
		},
		{
			method:  http.MethodPost,
			pattern: "/tenants/{id}/reactivate",
			panel:   panelTenants,
			guarded: h.tenantReactivate,
			mode:    routeModeSuperAdmin,
			write:   true,
			why:     "reactivation reverses a platform suspension",
		},
		{
			method: http.MethodPost, pattern: "/tenants/{id}/delete", panel: panelTenants, guarded: h.tenantDelete,
			mode: routeModeSuperAdmin, write: true, why: "deleting tenants destroys their history",
		},
	}
}

// superAdminTenantMemberRoutes mutate membership from the tenant detail view.
//
//nolint:exhaustruct // rows intentionally omit inapplicable fields (asset vs guarded, write)
func (h *Handler) superAdminTenantMemberRoutes() []routeSpec {
	return []routeSpec{
		{
			method: http.MethodPost, pattern: "/tenants/{id}/members", panel: panelTenants, guarded: h.tenantAddMember,
			mode: routeModeSuperAdmin, write: true, why: "membership grants tenant access",
		},
		{
			method:  http.MethodPost,
			pattern: "/tenants/{id}/members/{actor}/delete",
			panel:   panelTenants,
			guarded: h.tenantRemoveMember,
			mode:    routeModeSuperAdmin,
			write:   true,
			why:     "membership removal revokes tenant access",
		},
		{
			method:  http.MethodPost,
			pattern: "/tenants/{id}/members/{actor}",
			panel:   panelTenants,
			guarded: h.tenantUpdateMemberRole,
			mode:    routeModeSuperAdmin,
			write:   true,
			why:     "role changes re-shape authority inside the tenant",
		},
	}
}

// tenantAdminRoutes are the tenant-scope surfaces, served only in
// [ModeTenantAdmin].
//
//nolint:exhaustruct // rows intentionally omit inapplicable fields (asset vs guarded, write)
func (h *Handler) tenantAdminRoutes() []routeSpec {
	return []routeSpec{
		// --- Members (tenant admin only) ---
		{
			method: http.MethodGet, pattern: patternMembers, panel: panelMembers, guarded: h.membersIndex,
			mode: routeModeTenantAdmin, why: "the member roster is tenant-internal",
		},
		{
			method: http.MethodPost, pattern: patternMembers, panel: panelMembers, guarded: h.membersAdd,
			mode: routeModeTenantAdmin, write: true, why: "membership grants tenant access",
		},
		{
			method: http.MethodPost, pattern: "/members/{actor}/delete", panel: panelMembers, guarded: h.membersRemove,
			mode: routeModeTenantAdmin, write: true, why: "membership removal revokes tenant access",
		},
		{
			method: http.MethodPost, pattern: "/members/{actor}", panel: panelMembers, guarded: h.membersUpdateRole,
			mode: routeModeTenantAdmin, write: true, why: "role changes re-shape authority inside the tenant",
		},
	}
}

// registration returns the ServeMux expression for the row: the dashboard
// root registers as "/{$}" (a bare "GET /" would take precedence over every
// route), the wildcard fallbacks register as subtree roots, and a "*"
// method registers without a method prefix.
func (s routeSpec) registration() string {
	switch {
	case s.method == "*" && s.pattern == "/*":
		return "/"
	case s.method == "*":
		return s.pattern
	case s.pattern == "/":
		return s.method + " /{$}"
	case s.pattern == "/*":
		return s.method + " /"
	}

	return s.method + " " + s.pattern
}

// notFoundGuarded adapts notFoundHandler to the guarded handler shape.
func (h *Handler) notFoundGuarded() func(http.ResponseWriter, *http.Request, *identitymodel.User) {
	return func(w http.ResponseWriter, r *http.Request, _ *identitymodel.User) {
		h.notFoundHandler(w, r)
	}
}

// methodNotAllowed answers 405 for any request no other row matched.
func (h *Handler) methodNotAllowed() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.writeErrorPage(
			w,
			r,
			http.StatusMethodNotAllowed,
			"Method not allowed",
			"This endpoint does not support that HTTP method.",
		)
	}
}

// validateRouteTable fails fast on a malformed row at construction time.
func validateRouteTable(table []routeSpec) error {
	for _, spec := range table {
		if spec.method == "" || spec.pattern == "" {
			return errConfig("route table row needs a method and a pattern: " + spec.pattern)
		}
		if (spec.guarded == nil) == (spec.asset == nil) {
			return errConfig(
				"route table row " + spec.method + " " + spec.pattern + " needs exactly one of guarded/asset",
			)
		}
		if spec.mode == routeModeInvalid {
			return errConfig(
				"route table row " + spec.method + " " + spec.pattern + " has no mode (a forgotten mode is served by NO mode, not both)",
			)
		}
		if spec.why == "" {
			return errConfig("route table row " + spec.method + " " + spec.pattern + " needs a Why")
		}
	}

	return nil
}

// registerRoutes serves the table on mux — the ONLY registration path, so a
// route outside the table cannot exist.
func (h *Handler) registerRoutes(mux *http.ServeMux) {
	for _, spec := range h.table {
		if !spec.mode.servesIn(h.config.Mode) {
			continue
		}

		if spec.guarded != nil {
			mux.Handle(spec.registration(), h.guard(spec.guarded))
			continue
		}

		mux.Handle(spec.registration(), spec.asset)
	}
}

// Routes returns the manifest of HTTP routes this instance serves: the exact
// set [Handler.Handler] answers, honoring the configured Mode (mode-scoped
// rows are absent in the other mode) and this instance's authorizer posture.
// The mux and the manifest are generated from the same table, so a route
// cannot exist outside the manifest.
//
// Use it for consumer-side routing decisions: reverse-proxy allowlists,
// policy engines, upgrade diffing, or feeding [authztest.Undeclared].
func (h *Handler) Routes() []Route {
	rows := make([]Route, 0, len(h.table))

	for _, spec := range h.table {
		if !spec.mode.servesIn(h.config.Mode) {
			continue
		}

		posture := h.authorizerPosture()
		if spec.asset != nil {
			posture = cqrshtmx.PosturePublic
		}

		rows = append(rows, Route{
			Method:  spec.method,
			Pattern: spec.pattern,
			Panel:   spec.panel,
			Write:   spec.write,
			Posture: posture,
			Why:     spec.why,
		})
	}

	return rows
}

// authorizerPosture resolves the posture guarded routes report: the declared
// [Config.AuthorizerPosture], or role-gated — the truth for both the default
// authorizer ([RequireAnyRole]-built) and role-based custom authorizers.
func (h *Handler) authorizerPosture() cqrshtmx.Posture {
	if h.config.AuthorizerPosture != "" {
		return h.config.AuthorizerPosture
	}

	return cqrshtmx.PostureRoleGated
}
