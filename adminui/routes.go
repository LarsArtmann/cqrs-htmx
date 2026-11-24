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
	patternUsers   = "/users"
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
		{
			method: http.MethodGet, pattern: "/-/ext/cqrs-query.js", panel: panelAssets, asset: cqrsQueryExtHandler(),
			mode: routeModeBoth, why: "vendored HTMX QUERY extension, no data",
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
		{
			method: http.MethodGet, pattern: "/audit/export.csv", panel: panelAudit, guarded: h.auditExportCSV,
			mode: routeModeBoth,
			why:  "the audit export carries the retained window's events with their actors",
		},
		{
			method: http.MethodGet, pattern: "/audit/export.json", panel: panelAudit, guarded: h.auditExportJSON,
			mode: routeModeBoth,
			why:  "the audit export carries the retained window's events with their actors",
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
		h.superAdminUserExportRoutes(),
		h.superAdminUserAccountRoutes(),
		h.superAdminUserCredentialRoutes(),
		h.superAdminUserTenantAccessRoutes(),
		h.superAdminTenantRoutes(),
		h.superAdminTenantMemberRoutes(),
		h.superAdminMembersDirectoryRoutes(),
		h.superAdminMergeRoutes(),
	)
}

// superAdminUserRoutes administer the user DIRECTORY across every tenant:
// listing, creation, detail, and deletion. Writes that change how an
// existing user AUTHENTICATES live in [Handler.superAdminUserAccountRoutes].
//
//nolint:exhaustruct // rows intentionally omit inapplicable fields (asset vs guarded, write)
func (h *Handler) superAdminUserRoutes() []routeSpec {
	return []routeSpec{
		{
			method: http.MethodGet, pattern: patternUsers, panel: panelUsers, guarded: h.usersIndex,
			mode: routeModeSuperAdmin, why: "the user list spans every tenant",
		},
		{
			method: cqrshtmx.MethodQuery, pattern: "/users/search", panel: panelUsers, guarded: h.userSearch,
			mode: routeModeSuperAdmin,
			why:  "body-carrying user search (RFC 10008): a safe read whose criteria travel as a request body",
		},
		{
			method: http.MethodPost, pattern: "/users/search", panel: panelUsers, guarded: h.userSearch,
			mode: routeModeSuperAdmin,
			why:  "the search fallback twin: byte-identical responses for clients that cannot send QUERY",
		},
		{
			method: http.MethodGet, pattern: "/users/new", panel: panelUsers, guarded: h.userNew,
			mode: routeModeSuperAdmin, why: "the creation form itself reveals the directory shape",
		},
		{
			method: http.MethodPost, pattern: patternUsers, panel: panelUsers, guarded: h.userCreate,
			mode: routeModeSuperAdmin, write: true, why: "creating accounts is platform-admin territory",
		},
		{
			method: http.MethodGet, pattern: "/users/import", panel: panelUsers, guarded: h.userImportPage,
			mode: routeModeSuperAdmin, why: "the import form reveals nothing beyond the page it renders",
		},
		{
			method: http.MethodPost, pattern: "/users/import", panel: panelUsers, guarded: h.userImport,
			mode: routeModeSuperAdmin, write: true, why: "bulk import creates accounts from the uploaded file",
		},
		{
			method: http.MethodGet, pattern: "/users/{id}", panel: panelUsers, guarded: h.userDetail,
			mode: routeModeSuperAdmin, why: "user detail exposes credentials metadata",
		},
		{
			method: http.MethodGet, pattern: "/users/{id}/activity", panel: panelUsers, guarded: h.userActivity,
			mode: routeModeSuperAdmin,
			why:  "the per-user activity tail pages the audit window for one user's stream",
		},
		{
			method: http.MethodPost, pattern: "/users/{id}/delete", panel: panelUsers, guarded: h.userDelete,
			mode: routeModeSuperAdmin, write: true, why: "deleting users is platform-admin territory",
		},
	}
}

// superAdminUserExportRoutes read the directory in bulk: the full export and
// the filtered-view export, in CSV and JSON. Reads only, but the payload is
// every user's public profile, so they stay super-admin.
//
//nolint:exhaustruct // rows intentionally omit inapplicable fields (asset vs guarded, write)
func (h *Handler) superAdminUserExportRoutes() []routeSpec {
	return []routeSpec{
		{
			method: http.MethodGet, pattern: "/users/export.csv", panel: panelUsers, guarded: h.userExportCSV,
			mode: routeModeSuperAdmin, why: "the export carries every user's public profile",
		},
		{
			method: http.MethodGet, pattern: "/users/export.json", panel: panelUsers, guarded: h.userExportJSON,
			mode: routeModeSuperAdmin, why: "the export carries every user's public profile",
		},
		{
			method:  http.MethodGet,
			pattern: "/users/export.filtered.csv",
			panel:   panelUsers,
			guarded: h.userExportFilteredCSV,
			mode:    routeModeSuperAdmin,
			why:     "the filtered export carries the matched users' public profiles",
		},
		{
			method:  http.MethodGet,
			pattern: "/users/export.filtered.json",
			panel:   panelUsers,
			guarded: h.userExportFilteredJSON,
			mode:    routeModeSuperAdmin,
			why:     "the filtered export carries the matched users' public profiles",
		},
	}
}

// superAdminUserAccountRoutes write the AUTHENTICATION surface of one user:
// profile fields, credentials, external accounts, and platform roles. Each
// row changes how the target user signs in or who they can be — the highest-
// stakes writes in the panel, so each carries its own why.
//
//nolint:exhaustruct // rows intentionally omit inapplicable fields (asset vs guarded, write)
func (h *Handler) superAdminUserAccountRoutes() []routeSpec {
	return []routeSpec{
		{
			method:  http.MethodPost,
			pattern: "/users/{id}/display-name",
			panel:   panelUsers,
			guarded: h.userUpdateDisplayName,
			mode:    routeModeSuperAdmin,
			write:   true,
			why:     "editing a profile field is platform-admin territory",
		},
		{
			method: http.MethodPost, pattern: "/users/{id}/email", panel: panelUsers, guarded: h.userUpdateEmail,
			mode: routeModeSuperAdmin, write: true, why: "changing a sign-in email changes how the user authenticates",
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
		{
			method:  http.MethodPost,
			pattern: "/users/{id}/roles",
			panel:   panelUsers,
			guarded: h.userGrantPlatformRoles,
			mode:    routeModeSuperAdmin,
			write:   true,
			why:     "platform roles decide who can reach this panel",
		},
		{
			method:  http.MethodPost,
			pattern: "/users/{id}/roles/revoke",
			panel:   panelUsers,
			guarded: h.userRevokePlatformRoles,
			mode:    routeModeSuperAdmin,
			write:   true,
			why:     "revoking platform roles can lock a user (or the admin) out",
		},
		{
			method:  http.MethodPost,
			pattern: "/users/{id}/resend-verification",
			panel:   panelUsers,
			guarded: h.userResendVerification,
			mode:    routeModeSuperAdmin,
			write:   true,
			why:     "sending mail as a user-visible action must be admin-gated",
		},
	}
}

// superAdminUserCredentialRoutes manage a user's authentication FACTORS from
// the user side: passkey removal and backup-JSON restore.
//
//nolint:exhaustruct // rows intentionally omit inapplicable fields (asset vs guarded, write)
func (h *Handler) superAdminUserCredentialRoutes() []routeSpec {
	return []routeSpec{
		{
			method:  http.MethodPost,
			pattern: "/users/{id}/credentials/{credential}/delete",
			panel:   panelUsers,
			guarded: h.userRemoveCredential,
			mode:    routeModeSuperAdmin,
			write:   true,
			why:     "removing a passkey changes how the user authenticates",
		},
		{
			method:  http.MethodPost,
			pattern: "/users/{id}/credentials/restore",
			panel:   panelUsers,
			guarded: h.userRestoreCredential,
			mode:    routeModeSuperAdmin,
			write:   true,
			why:     "restoring a passkey from backup JSON re-grants an authentication factor",
		},
	}
}

// superAdminUserTenantAccessRoutes manage a user's TENANT memberships from the
// user side — the mirror of the tenant roster's membership writes. Both paths
// dispatch the same durable commands; the admin starts wherever they are.
//
//nolint:exhaustruct // rows intentionally omit inapplicable fields (asset vs guarded, write)
func (h *Handler) superAdminUserTenantAccessRoutes() []routeSpec {
	return []routeSpec{
		{
			method:  http.MethodPost,
			pattern: "/users/{id}/tenants",
			panel:   panelUsers,
			guarded: h.userGrantTenantAccess,
			mode:    routeModeSuperAdmin,
			write:   true,
			why:     "tenant membership grants a user authority inside that tenant",
		},
		{
			method:  http.MethodPost,
			pattern: "/users/{id}/tenants/{tenant}",
			panel:   panelUsers,
			guarded: h.userUpdateTenantRole,
			mode:    routeModeSuperAdmin,
			write:   true,
			why:     "changing a member's role re-shapes authority inside that tenant",
		},
		{
			method:  http.MethodPost,
			pattern: "/users/{id}/tenants/{tenant}/delete",
			panel:   panelUsers,
			guarded: h.userRevokeTenantAccess,
			mode:    routeModeSuperAdmin,
			write:   true,
			why:     "tenant membership removal revokes a user's access inside that tenant",
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
			method:  http.MethodGet,
			pattern: "/tenants/{id}/roster.csv",
			panel:   panelTenants,
			guarded: h.tenantRosterCSV,
			mode:    routeModeSuperAdmin,
			why:     "the roster export carries the tenant's full membership list (read access mirrors GET /tenants/{id})",
		},
		{
			method:  http.MethodPost,
			pattern: "/tenants/{id}/transfer",
			panel:   panelTenants,
			guarded: h.tenantTransferOwnership,
			mode:    routeModeSuperAdmin,
			write:   true,
			why:     "ownership transfer grants owner to one member and demotes the prior owner — tenant-shaping authority",
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

// superAdminMembersDirectoryRoutes read memberships across every tenant in
// one place: the global directory. Reads only, but the payload spans every
// tenant's roster, so it stays super-admin. The tenant-admin GET /members row
// (the roster) shares the pattern but never the mode, so the two never mount
// in the same panel instance.
//
//nolint:exhaustruct // rows intentionally omit inapplicable fields (asset vs guarded, write)
func (h *Handler) superAdminMembersDirectoryRoutes() []routeSpec {
	return []routeSpec{
		{
			method: http.MethodGet, pattern: patternMembers, panel: panelMembers, guarded: h.membersDirectory,
			mode: routeModeSuperAdmin, why: "the membership directory spans every tenant's rosters",
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
			method:  http.MethodGet,
			pattern: "/members/roster.csv",
			panel:   panelMembers,
			guarded: h.membersRosterCSV,
			mode:    routeModeTenantAdmin,
			why:     "the roster export carries the scoped tenant's full membership list (read access mirrors GET /members)",
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
// route outside the table cannot exist. In ReadOnly mode the write:true rows
// are skipped HERE, at registration: the write handlers never mount, so a
// mutating request falls through to the method catch-all and answers 405 —
// the refusal is structural (no route exists to reach), not a runtime check.
func (h *Handler) registerRoutes(mux *http.ServeMux) {
	for _, spec := range h.table {
		if !spec.mode.servesIn(h.config.Mode) {
			continue
		}
		if h.config.ReadOnly && spec.write {
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
//
// ReadOnly reflects too: the write:true rows are absent, so an allowlist or
// matrix built from this manifest never names a route this instance refuses.
func (h *Handler) Routes() []Route {
	rows := make([]Route, 0, len(h.table))

	for _, spec := range h.table {
		if !spec.mode.servesIn(h.config.Mode) {
			continue
		}
		if h.config.ReadOnly && spec.write {
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

// superAdminMergeRoutes are the duplicate-reconciliation surfaces: the two
// duplicates reports and the merge/rename writes they post to. Merging
// identities consolidates authority and history, so every write stays
// super-admin.
//
//nolint:exhaustruct // rows intentionally omit inapplicable fields (asset vs guarded, write)
func (h *Handler) superAdminMergeRoutes() []routeSpec {
	return []routeSpec{
		{
			method:  http.MethodGet,
			pattern: "/users/duplicates",
			panel:   panelUsers,
			guarded: h.userDuplicates,
			mode:    routeModeSuperAdmin,
			why:     "the duplicates report spans every tenant's users",
		},
		{
			method:  http.MethodPost,
			pattern: "/users/{id}/merge",
			panel:   panelUsers,
			guarded: h.userMerge,
			mode:    routeModeSuperAdmin,
			write:   true,
			why:     "consolidating duplicate identities is terminal and platform-wide",
		},
		{
			method:  http.MethodGet,
			pattern: "/tenants/duplicates",
			panel:   panelTenants,
			guarded: h.tenantDuplicates,
			mode:    routeModeSuperAdmin,
			why:     "the duplicates report is the platform roster's mirror",
		},
		{
			method:  http.MethodPost,
			pattern: "/tenants/{id}/rename",
			panel:   panelTenants,
			guarded: h.tenantRename,
			mode:    routeModeSuperAdmin,
			write:   true,
			why:     "renaming a tenant relabels every tenant-scoped surface",
		},
		{
			method:  http.MethodPost,
			pattern: "/tenants/{id}/merge",
			panel:   panelTenants,
			guarded: h.tenantMerge,
			mode:    routeModeSuperAdmin,
			write:   true,
			why:     "consolidating duplicate tenants moves every member's authority",
		},
	}
}
