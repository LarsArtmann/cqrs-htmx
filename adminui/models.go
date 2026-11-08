package adminui

import (
	"net/url"
	"slices"
	"strings"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	"github.com/larsartmann/templ-components/display"
)

// navItem is a single sidebar entry. Icon is a key resolved by the svgIcon
// templ component (e.g. "dashboard", "users", "tenants", "members", "audit").
// Verbatim marks consumer ExtraNav entries: their Href is rendered as-is,
// without the BasePath prefix built-in entries get.
type navItem struct {
	Href     string
	Label    string
	Icon     string
	Active   bool
	Verbatim bool
}

// pageData carries everything the Layout needs to render a full page. The page
// body itself is supplied as templ children (see the *Page components).
type pageData struct {
	Title     string
	BasePath  string
	Accent    string
	Brand     string
	User      *identitymodel.User
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

// statCard is one metric tile on the dashboard. Tone and Href are optional
// enrichment: the tone colors the icon tile (semantic — blue for people,
// purple for tenants, green for audit volume), and Href turns the card into
// a drill-down link. Change/Trend stay unset on purpose (honest UI): the
// handler has no period-over-period data, and decorative numbers are worse
// than none.
type statCard struct {
	Label string
	Value string
	Icon  string
	Tone  display.StatTone
	Href  string
}

// classNoUnderline is the anchor-decoration suppressor shared by linked
// stat cards and templ-components buttons rendered as links (BaseProps.Class).
const classNoUnderline = "no-underline"

// statCardClass returns the extra classes for a stat card: linked cards drop
// the anchor underline so the drill-down reads as a card, not a text link.
func statCardClass(href string) string {
	if href == "" {
		return ""
	}

	return classNoUnderline
}

// tenantSubtitle renders the tenant identity line under the Members page
// header ("Display Name · machine-name"); empty when the tenant is absent.
func tenantSubtitle(t *usermgmt.Tenant) string {
	if t == nil {
		return ""
	}
	return t.DisplayName + " · " + t.Name
}

// dashboardData drives the overview page.
type dashboardData struct {
	Stats    []statCard
	StatsURL string
	BasePath string
	Recent   []usermgmt.AuditEntry
	// NoAdminGrant is true when nobody holds an admin-tier grant in the
	// wildcard domain — the panel is one custom authorizer away from being
	// unreachable, and the dashboard says so instead of staying quiet.
	NoAdminGrant bool
}

// listPage carries the pagination footer state shared by all list pages.
// TotalPages is always >= 1; the footer renders only when > 1.
type listPage struct {
	Page       int // current page, 1-based, clamped by pageBounds
	TotalPages int
}

// usersListData drives the users index.
type usersListData struct {
	Users    []*identitymodel.User
	Total    int // total matching the search + filter (may exceed len(Users) when paginated)
	Search   string
	Filter   userFilter // active status chip ("" = none)
	BasePath string
	listPage
}

// userDetailData drives a single user's page.
type userDetailData struct {
	User                *identitymodel.User
	BasePath            string
	TenantRoles         map[string][]identitymodel.Role // domain -> roles
	PlatformRoles       []identitymodel.Role            // roles in the wildcard domain ("*")
	GrantableAdminRoles []identitymodel.Role            // the roles the platform-access card offers
	IsSelf              bool                            // the shown user is the session user (self-lockout warnings)
	Timeline            userTimeline                    // recent events about this user's stream
	ConfiguredProviders []string                        // OAuth2 providers configured on the Service (for the link/unlink card)
	UnlinkExternalBase  string                          // URL prefix for unlink POSTs: append "/{provider}/unlink"
}

// tenantsListData drives the tenants index.
type tenantsListData struct {
	Tenants  []*usermgmt.Tenant
	Total    int
	BasePath string
	listPage
}

// tenantDetailData drives a single tenant's page with its members. It is also
// reused for the tenant-admin members page.
type tenantDetailData struct {
	Tenant           *usermgmt.Tenant
	Members          []memberRow
	AssignableRoles  []identitymodel.Role
	BasePath         string
	AddMemberURL     string
	RemoveMemberBase string
	UpdateRoleBase   string // append "/{actor}" for role-change POSTs
}

// memberRow is a flattened membership for display.
type memberRow struct {
	Actor identitymodel.ActorID
	Roles []identitymodel.Role
}

// toMemberRows converts a membership slice to display rows.
func toMemberRows(memberships []*identitymodel.Membership) []memberRow {
	members := make([]memberRow, 0, len(memberships))
	for _, m := range memberships {
		members = append(members, memberRow{Actor: m.ActorID, Roles: m.Roles})
	}
	return members
}

// hasRole reports whether roles contains role.
func hasRole(roles []identitymodel.Role, role identitymodel.Role) bool {
	return slices.Contains(roles, role)
}

// removeCredentialConfirm builds the passkey-removal confirm text: removing
// the user's last passkey leaves them dependent on linked OAuth2 providers
// (or locked out entirely), and the admin deserves to know which world they
// are choosing.
func removeCredentialConfirm(d userDetailData, cred identitymodel.WebAuthnCredential) string {
	name := cred.Name
	if name == "" {
		name = "this passkey"
	}
	if len(d.User.Credentials) == 1 && len(d.User.ExternalAccounts) == 0 {
		return "Remove " + name + "? It is the user's ONLY way to sign in — they will be " +
			"locked out until a passkey is re-enrolled or a provider is linked. This cannot be undone lightly."
	}
	if len(d.User.Credentials) == 1 {
		return "Remove " + name + "? It is the user's last passkey — afterwards they can only " +
			"sign in via a linked OAuth2 provider."
	}
	return "Remove " + name + "? The user can no longer sign in with it."
}

// revokeRolesConfirm builds the platform-role revocation confirm text. When
// the admin is revoking their own access, the warning escalates: the panel
// itself becomes unreachable for them.
func revokeRolesConfirm(d userDetailData) string {
	if d.IsSelf {
		return "Revoke YOUR OWN platform roles? You will lock yourself out of this panel " +
			"until another admin re-grants them."
	}
	return "Revoke all platform roles for " + d.User.Email +
		"? They lose admin-panel access until re-granted."
}

// userFilterChip is one rendered status chip on the users index.
type userFilterChip struct {
	Label  string
	Href   string
	Active bool
}

// labelVerified is the one display label for the verified-email concept:
// the dashboard adoption card and the users filter chip share it.
const labelVerified = "Verified"

// userFilterChipsList builds the chip row, preserving an active search: the
// active chip toggles the filter OFF (keeps q), the others toggle their own
// filter ON (also keeping q).
func userFilterChipsList(d usersListData) []userFilterChip {
	base := d.BasePath + "/users"
	type spec struct {
		filter userFilter
		label  string
	}
	specs := []spec{
		{userFilterVerified, labelVerified},
		{userFilterUnverified, "Unverified"},
		{userFilterMFA, "MFA on"},
		{userFilterExternal, "Has provider"},
	}
	chips := make([]userFilterChip, 0, len(specs))
	for _, s := range specs {
		href := base
		params := make([]string, 0, 2)
		if d.Search != "" {
			params = append(params, "q="+url.QueryEscape(d.Search))
		}
		if s.filter != d.Filter {
			params = append(params, "filter="+string(s.filter))
		}
		if len(params) > 0 {
			href += "?" + strings.Join(params, "&")
		}
		chips = append(chips, userFilterChip{Label: s.label, Href: href, Active: s.filter == d.Filter})
	}
	return chips
}

// auditData drives the audit log page.
type auditData struct {
	Entries  []usermgmt.AuditEntry
	Total    int // total matching the search (may exceed len(Entries) when paginated)
	Search   string
	BasePath string
	listPage
}

// userTimeline is the per-user audit view on the user detail page: the most
// recent events about this user's aggregate stream, latest-first, already
// email-resolved by the handler.
type userTimeline struct {
	Entries []usermgmt.AuditEntry
	Total   int
}
