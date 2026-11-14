package adminui

import (
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	"github.com/larsartmann/templ-components/display"
	"github.com/larsartmann/templ-components/forms"
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
	Users    []userRow
	Total    int // total matching the search + filter (may exceed len(Users) when paginated)
	Search   string
	Filter   userFilter // active chip ("" = none)
	BasePath string
	listPage
}

// userRow is one user list entry enriched with the access facts an admin
// decides on: the platform roles they hold (wildcard domain) and how many
// tenants they belong to. A directory that reports people but hides who has
// power forces the admin to open every profile to find out.
type userRow struct {
	User          *identitymodel.User
	PlatformRoles []identitymodel.Role
	TenantCount   int
}

// isPlatformAdmin reports whether the row grants any platform (wildcard) role.
func (r userRow) isPlatformAdmin() bool { return len(r.PlatformRoles) > 0 }

// tenantCountLabel is the memberships summary ("1 tenant", "3 tenants"). It is
// built here, not in the template, so the number and the noun render as one
// text node (a screen reader announces "one tenant", not "one" then "tenant").
func (r userRow) tenantCountLabel() string {
	if r.TenantCount == 1 {
		return "1 tenant"
	}
	return strconv.Itoa(r.TenantCount) + " tenants"
}

// userDetailData drives a single user's page.
type userDetailData struct {
	User                *identitymodel.User
	BasePath            string
	PlatformRoles       []identitymodel.Role // roles in the wildcard domain ("*")
	GrantableAdminRoles []identitymodel.Role // the roles the platform-access card offers
	IsSelf              bool                 // the shown user is the session user (self-lockout warnings)
	Timeline            userTimeline         // recent events about this user's stream
	ConfiguredProviders []string             // OAuth2 providers configured on the Service (for the link/unlink card)
	UnlinkExternalBase  string               // URL prefix for unlink POSTs: append "/{provider}/unlink"

	// TenantAccess is the user's tenant memberships, resolved to tenant
	// identity and ordered by tenant name. The user page is where an admin
	// usually starts ("who is this person?"), so the multi-tenant picture
	// belongs here, not only on each tenant's roster.
	TenantAccess []tenantAccessRow
	// TenantOptions feeds the "grant access" select: every live tenant, so
	// access can be added from the user side without a trip to /tenants.
	TenantOptions []forms.SelectOption
	// AssignableRoles is the role vocabulary the grant form offers.
	AssignableRoles []identitymodel.Role
	// AddTenantAccessURL is the POST target for grants; RemoveTenantBase is a
	// URL prefix for removals (append "/{tenant}/delete").
	AddTenantAccessURL string
	RemoveTenantBase   string
}

// tenantAccessRow is one resolved tenant membership on the user page. The
// tenant id is the Casbin domain; Name/Slug come from the tenant read model so
// the row reads as a tenant, not as a domain string.
type tenantAccessRow struct {
	TenantID string
	Name     string // display name
	Slug     string
	Roles    []identitymodel.Role
}

// tenantsListData drives the tenants index.
type tenantsListData struct {
	Tenants  []tenantRow
	Total    int
	Search   string
	BasePath string
	listPage
}

// tenantRow is one tenant list entry plus the facts an admin needs BEFORE
// opening it: how many actors it scopes and its lifecycle state. The count is
// derived per visible row (the read model has no batched count query yet) and
// the list is page-bounded, so this stays O(page × memberships).
type tenantRow struct {
	Tenant      *usermgmt.Tenant
	MemberCount int
}

// matchesTenantSearch reports whether q occurs in the tenant's slug or display
// name (case-insensitive). An empty q matches every tenant.
func matchesTenantSearch(t *usermgmt.Tenant, q string) bool {
	if q == "" {
		return true
	}
	needle := strings.ToLower(q)

	return strings.Contains(strings.ToLower(t.Name), needle) ||
		strings.Contains(strings.ToLower(t.DisplayName), needle)
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

	// UserDetailBase is the URL prefix for member detail links (the panel's
	// ".../users/"). Empty hides the links: tenant-admin mode serves no user
	// directory, and a link to a route that does not exist is a lie.
	UserDetailBase string

	// UserEmailOptions feeds the add-member autocomplete: the directory's
	// email addresses, so an admin picks a person instead of typing one.
	UserEmailOptions []string

	// AddMemberError is the inline guidance shown when the last add attempt
	// named no known account. AddMemberEmail preserves what was typed so the
	// admin can correct it instead of retyping from scratch.
	AddMemberError string
	AddMemberEmail string

	// AddMemberCreateHref is the "create this user first" escape hatch,
	// rendered only when the panel serves user creation (super-admin mode).
	AddMemberCreateHref string
}

// memberRow is a flattened membership for display. User actors are resolved
// to their directory identity: an admin reads a person (email + name), never
// an opaque "user:01HZ…" token. Bot/system/service actors have no directory
// entry and keep their prefixed id.
type memberRow struct {
	Actor   identitymodel.ActorID
	Email   string
	Name    string
	IsUser  bool // the actor is a human user that resolved in the directory
	Roles   []identitymodel.Role
	AddedAt time.Time // zero when the store did not record it
}

// displayName is what the members table leads with: the email for a resolved
// user, the self-describing actor id otherwise.
func (m memberRow) displayName() string {
	if m.Email != "" {
		return m.Email
	}

	return m.Actor.PrefixedString()
}

// toMemberRows converts a membership slice to display rows, resolving user
// actors through lookup (the user read model). A user that no longer resolves
// falls back to its actor id instead of vanishing: a stale membership is a
// fact an admin needs to see, not a row to hide.
func toMemberRows(
	memberships []*identitymodel.Membership,
	lookup func(identitymodel.UserID) (*identitymodel.User, bool),
) []memberRow {
	members := make([]memberRow, 0, len(memberships))
	for _, m := range memberships {
		row := memberRow{Actor: m.ActorID, Roles: m.Roles, AddedAt: m.AddedAt}
		if userID, ok := identitymodel.ActorIDAsUserID(m.ActorID); ok {
			if user, found := lookup(userID); found {
				row.Email = user.Email
				row.Name = user.DisplayName
				row.IsUser = true
			}
		}
		members = append(members, row)
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
		{userFilterAdmin, "Administrators"},
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
