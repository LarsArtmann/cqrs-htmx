package adminui

import (
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
)

// navItem is a single sidebar entry. Icon is a key resolved by the svgIcon
// templ component (e.g. "dashboard", "users", "tenants", "members", "audit").
type navItem struct {
	Href   string
	Label  string
	Icon   string
	Active bool
}

// pageData carries everything the Layout needs to render a full page. The page
// body itself is supplied as templ children (see the *Page components).
type pageData struct {
	Title     string
	BasePath  string
	Accent    string
	Brand     string
	User      *usermgmt.User
	Nav       []navItem
	LogoutURL string
	// CSRFToken is the rendered hidden input for CSRF protection (empty when no
	// CSRF middleware is active). Render it inside every POST form via
	// @templ.Raw(p.CSRFToken).
	CSRFToken string
	// CSRFMeta is a <meta name="csrf-token"> tag for the layout head. HTMX
	// reads it via admin.js to send the token on button POSTs. Empty when no
	// CSRF middleware is active.
	CSRFMeta string
	// SSEURL is the Server-Sent Events endpoint URL. When set, the layout body
	// gets a data-sse-url attribute that admin.js uses to connect an EventSource.
	// Also enables the global sync indicator (.sync-bar) in the header. Empty
	// disables honest UI sync tracking (no SSE connection, no sync bar).
	SSEURL string
	// Nonce is the CSP nonce for inline scripts (ToastContainer,
	// GlobalErrorHandling). Empty when no CSP middleware is active.
	Nonce string
}

// statCard is one metric tile on the dashboard.
type statCard struct {
	Label string
	Value string
	Icon  string
}

// dashboardData drives the overview page.
type dashboardData struct {
	Stats  []statCard
	Recent []usermgmt.AuditEntry
}

// usersListData drives the users index.
type usersListData struct {
	Users    []*usermgmt.User
	Total    int // total matching the search (may exceed len(Users) when capped)
	Search   string
	BasePath string
}

// userDetailData drives a single user's page.
type userDetailData struct {
	User                *usermgmt.User
	BasePath            string
	TenantRoles         map[string][]usermgmt.Role // domain -> roles
	ConfiguredProviders []string                   // OAuth2 providers configured on the Service (for the link/unlink card)
	UnlinkExternalBase  string                     // URL prefix for unlink POSTs: append "/{provider}/unlink"
}

// tenantsListData drives the tenants index.
type tenantsListData struct {
	Tenants  []*usermgmt.Tenant
	Total    int
	BasePath string
}

// tenantDetailData drives a single tenant's page with its members. It is also
// reused for the tenant-admin members page.
type tenantDetailData struct {
	Tenant           *usermgmt.Tenant
	Members          []memberRow
	AssignableRoles  []usermgmt.Role
	BasePath         string
	AddMemberURL     string
	RemoveMemberBase string
	UpdateRoleBase   string // append "/{actor}" for role-change POSTs
}

// memberRow is a flattened membership for display.
type memberRow struct {
	Actor usermgmt.ActorID
	Roles []usermgmt.Role
}

// toMemberRows converts a membership slice to display rows.
func toMemberRows(memberships []*usermgmt.Membership) []memberRow {
	members := make([]memberRow, 0, len(memberships))
	for _, m := range memberships {
		members = append(members, memberRow{Actor: m.ActorID, Roles: m.Roles})
	}
	return members
}

// auditData drives the audit log page.
type auditData struct {
	Entries  []usermgmt.AuditEntry
	BasePath string
}
