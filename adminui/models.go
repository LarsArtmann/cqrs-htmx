package adminui

import (
	"github.com/larsartmann/cqrs-htmx/usermgmt/v3"
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
	Search   string
	BasePath string
}

// userDetailData drives a single user's page.
type userDetailData struct {
	User        *usermgmt.User
	BasePath    string
	TenantRoles map[string][]usermgmt.Role // domain -> roles
}

// tenantsListData drives the tenants index.
type tenantsListData struct {
	Tenants  []*usermgmt.Tenant
	BasePath string
}

// tenantDetailData drives a single tenant's page with its members. It is also
// reused for the tenant-admin members page.
type tenantDetailData struct {
	Tenant           *usermgmt.Tenant
	Members          []memberRow
	BasePath         string
	AddMemberURL     string // POST target to add a member
	RemoveMemberBase string // append "/{actor}/delete" per row
}

// memberRow is a flattened membership for display.
type memberRow struct {
	Actor string
	Roles []usermgmt.Role
}

// auditData drives the audit log page.
type auditData struct {
	Entries  []usermgmt.AuditEntry
	BasePath string
}
