package adminui

import (
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	"github.com/larsartmann/templ-components/display"
	"github.com/larsartmann/templ-components/forms"
	"github.com/larsartmann/templ-components/icons"
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
	// TenantSwitcher lists live tenants for the header jump control. Populated
	// only in super-admin mode (ModeTenantAdmin is scoped to one tenant by
	// construction, so a switcher there would be a lie); empty hides the
	// control entirely.
	TenantSwitcher []tenantSwitchOption
	// CurrentTenantID marks the switcher entry matching the tenant being
	// viewed. Empty (every page except tenant detail) marks nothing.
	CurrentTenantID string
	// ReadOnly renders the sidebar footer's "Read-only" notice chip. Set from
	// [Config.ReadOnly] at page build; false (the default) renders the footer
	// unchanged.
	ReadOnly bool
	// Labels is the page's merged display-label table (defaults +
	// [Config.Labels] overrides, merged once at [New]). Templates read the
	// covered string set ONLY through p.label(key) / label(p.Labels, key) —
	// never hardcoded — so an override re-words every covered surface at
	// once. An unmapped key renders the key itself, never blank.
	Labels map[string]string
	// Lang is the BCP-47 tag rendered as the <html lang> attribute (default
	// "en") and the key that picks the date shapes for p.date/p.dateTime.
	Lang string
	// Dir is the writing direction rendered as the <html dir> attribute
	// ("ltr" default, "rtl" supported).
	Dir string
	// Loc is the render's time location backing p.date/p.dateTime. Set from
	// the render clock's location — never time.Local spelled literally.
	Loc *time.Location
}

// tenantSwitchOption is one entry of the header tenant switcher: a LIVE
// tenant (suspended/deleted ones are not navigable targets) with its detail
// path and the current marker.
type tenantSwitchOption struct {
	ID          string
	DisplayName string
	Current     bool
}

// tenantSwitchCurrentName is the trigger label for a tenant-scoped view: the
// display name of the tenant being viewed. Falls back to the neutral label
// when no current tenant is set (defensive — the templ branch never calls it
// with an empty CurrentTenantID).
func tenantSwitchCurrentName(p pageData) string {
	for _, opt := range p.TenantSwitcher {
		if opt.Current {
			return opt.DisplayName
		}
	}
	return label(p.Labels, labelNavTenants)
}

// tenantSwitchItems renders the switcher's dropdown items: one link per live
// tenant (display-name order), the tenant being viewed carrying the check
// icon as its current marker. Each item carries data-tenant-search (display
// name + slug, lowercased) for the switcher's in-dropdown filter (F158): the
// filter is a client-side narrow of the server-rendered list, never a
// second source of tenant truth.
func tenantSwitchItems(p pageData) []display.DropdownItem {
	items := make([]display.DropdownItem, 0, len(p.TenantSwitcher))
	for _, opt := range p.TenantSwitcher {
		item := display.DropdownItem{
			Text:     opt.DisplayName,
			Href:     p.BasePath + "/tenants/" + opt.ID,
			Icon:     "",
			Kind:     display.DropdownItemLink,
			External: false,
			Disabled: false,
			Attrs: templ.Attributes{
				"data-tenant-search": strings.ToLower(opt.DisplayName + " " + opt.ID),
			},
		}
		if opt.Current {
			item.Icon = icons.Check
		}
		items = append(items, item)
	}
	return items
}

// roleDescriptions is the help-text vocabulary for the role checkbox group
// (F153): one line per assignable role, saying what the role CONFERES so an
// admin granting it knows the blast radius before the checkbox, not after.
// A function, not a package var: package-level state stays reserved for the
// display tables.
func roleDescriptions(role identitymodel.Role) string {
	switch role {
	case identitymodel.RoleOwner:
		return "Full control of this tenant, including its lifecycle. There should be exactly one."
	case identitymodel.RoleAdmin:
		return "Manage members and roles inside this tenant."
	case identitymodel.RoleUser:
		return "Standard member access to this tenant."
	case identitymodel.RoleViewer:
		return "Read-only access to this tenant."
	}
	return ""
}

// tenantTransferConfirm composes the transfer confirm for ONE pairing
// (F163): the current owner and the receiving member are both NAMED — ids
// alone would make an irreversible authority move a lookup exercise. The
// text also states the two-sided effect. (T16 batch-3: interpolated confirm
// bodies composed in Go helpers stay hardcoded English.)
func tenantTransferConfirm(priorName, targetName string) string {
	return "Transfer ownership from " + priorName + " to " + targetName + "? " +
			targetName + " gains the owner role and keeps their other roles; " +
			priorName + " loses owner but keeps any other roles they hold."
}

// statCard is one metric tile on the dashboard. Tone and Href are optional
// enrichment: the tone colors the icon tile (semantic — blue for people,
// purple for tenants, green for audit volume), and Href turns the card into
// a drill-down link. Change/Trend carry the audit-window delta: they are set
// ONLY from real window math — a window that cannot support a comparison
// (zero events) leaves them empty so the card renders no change at all
// instead of a fabricated "0% → 0%" stability claim.
type statCard struct {
	Label string
	Value string
	Icon  string
	Tone  display.StatTone
	Href  string
	// Change is the signed delta between the window's later half and its
	// earlier half (e.g. "+2", "−1", "±0"). Empty = no honest comparison.
	Change string
	// Trend colors the Change: TrendUp/TrendDown for real deltas, TrendNone
	// when the window is empty or the counts are equal.
	Trend display.TrendDirection
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

// dayCount is one day's event count for the registrations sparkline. Day is
// the local date's midnight; Count includes that day.
type dayCount struct {
	Day   time.Time
	Count int
}

// auditWindowSpan is the oldest→newest reach of the audit window's retained
// entries. HasSpan is false when the window holds no entries — an empty
// window gets no caption dates because there are none to state.
type auditWindowSpan struct {
	From    time.Time
	To      time.Time
	HasSpan bool
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
	// BusFactor is true when exactly ONE user holds every admin-tier grant
	// in the wildcard domain: one departure locks the platform out. The
	// banner names the risk instead of letting a single point of failure
	// stay invisible.
	BusFactor bool
	// Window is the audit projection's retention picture backing every
	// derived card. Zero facts render no caption.
	Window auditWindowFacts
	// WindowSpan is the oldest→newest reach of the retained entries, so the
	// honesty caption can state what the window covers ("from X to Y").
	WindowSpan auditWindowSpan
	// Registrations is the per-day registration count for the sparkline
	// (oldest → newest across a trailing fixed range). Nil when no audit
	// log exists.
	Registrations []dayCount
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
	LastSeen      time.Time // zero when the audit window records no activity
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

// authMethods is the deployment's sign-in capability picture, probed from
// the usermgmt Service, as it bears on the user detail page. A card for a
// channel the deployment cannot offer is dead weight — it advertises actions
// the user can never take — so the template gates the Passkeys, TOTP, and
// External-accounts surfaces on these facts instead of always rendering an
// honest-looking empty state for something that is structurally impossible.
type authMethods struct {
	HasWebAuthn bool     // a WebAuthn provider is configured (passkeys possible)
	HasTOTP     bool     // a TOTP provider is configured (MFA possible)
	HasOAuth2   bool     // an OAuth2 provider is configured (external links possible)
	Providers   []string // configured OAuth2 provider names; nil when no OAuth2 or the provider cannot enumerate
}

// authMethodsFromService reads the Service's published capability probes.
// It is the ONLY source of these facts: user data must not be inferred
// backwards into capability (a user with zero passkeys says nothing about
// whether passkeys are possible here).
func authMethodsFromService(svc *usermgmt.Service) authMethods {
	return authMethods{
		HasWebAuthn: svc.HasWebAuthn(),
		HasTOTP:     svc.HasTOTP(),
		HasOAuth2:   svc.HasOAuth2(),
		Providers:   svc.ConfiguredOAuth2Providers(),
	}
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
	// AuthMethods is the deployment capability picture (WebAuthn/TOTP/OAuth2
	// configured) that gates the sign-in cards: a card for a channel the
	// Service cannot serve never renders, rather than rendering an empty
	// state that implies the user could fill it.
	AuthMethods authMethods

	// TenantAccess is the user's tenant memberships, resolved to tenant
	// identity and ordered by tenant name. The user page is where an admin
	// usually starts ("who is this person?"), so the multi-tenant picture
	// belongs here, not only on each tenant's roster.
	TenantAccess []tenantAccessRow
	// TenantOptions feeds the "grant access" select: every live tenant, so
	// access can be added from the user side without a trip to /tenants.
	// Empty means the user already belongs to every live tenant — the grant
	// form would be a dead end and is replaced by an honest state.
	TenantOptions []forms.SelectOption
	// TenantAdminCounts maps tenant id -> how many of its members hold an
	// admin-tier role (admin or owner). The revoke confirm reads it to
	// escalate when the shown user holds the tenant's only admin-tier role.
	TenantAdminCounts map[string]int
	// AssignableRoles is the role vocabulary the grant form offers.
	AssignableRoles []identitymodel.Role
	// AddTenantAccessURL is the POST target for grants. UpdateTenantBase and
	// RemoveTenantBase are prefixes for per-tenant writes: append "/{tenant}"
	// to change roles, "/{tenant}/delete" to revoke.
	AddTenantAccessURL string
	UpdateTenantBase   string
	RemoveTenantBase   string
	// SignOutURL is the sign-out destination the embedding application mounts
	// (Config.LogoutURL). The panel owns no session endpoint of its own, so
	// when the consumer has not mounted one the self-service card says so
	// honestly instead of rendering a sign-out link that 404s.
	SignOutURL string
}

// cannotSignIn is the lockout truth table: every sign-in path in usermgmt
// starts from a WebAuthn credential, a linked external account, or TOTP — a
// user holding none of the three is locked out of their own account and can
// only be rescued by an admin-side action (supervised re-enrollment). This
// is the first-class rescue signal the panel previously could not express:
// it had to be re-derived per confirm dialog, and never appeared as a
// standing state on the page.
func cannotSignIn(u *identitymodel.User) bool {
	return u != nil &&
		len(u.Credentials) == 0 &&
		len(u.ExternalAccounts) == 0 &&
		!u.TOTPEnabled
}

// cannotSignInCopy is the lockout badge's escalation copy: the same
// blast-radius language the last-passkey removal confirm uses ("locked out
// until a passkey is re-enrolled or a provider is linked"), because the
// badge is that confirm's warning made a standing fact about the account.
func cannotSignInCopy() string {
	return "No passkey, no linked provider, and no TOTP — this user has no way to " +
		"sign in and is locked out of their account until a passkey is re-enrolled, " +
		"a provider is linked, or TOTP is enabled."
}

// userDetailDefinitionItems builds the Details card rows. The TOTP row is
// included only when the Service can do TOTP at all: an always-"Off" row for
// a capability the deployment lacks reads as a setting someone could flip,
// not as an absence.
// labelEmail is the Details-card row label for the user's email address —
// the same column word the forms and tables spell elsewhere in the panel.
const labelEmail = "Email"

func userDetailDefinitionItems(d userDetailData) []display.DefinitionItem {
	items := []display.DefinitionItem{
		{Term: "User ID", Detail: "", DetailComponent: copyableID(d.User.ID.Get().String())},
		{Term: labelEmail, Detail: d.User.Email, DetailComponent: nil},
		{Term: "Verified", Detail: "", DetailComponent: verifiedDetail(d)},
		// The token lifetime is consumer-configured and unreadable from the
		// published Service surface — the row names that honestly (F168)
		// instead of displaying a TTL the panel would have to invent.
		{Term: "Verification token", Detail: verificationTTLCopy(), DetailComponent: nil},
	}
	if d.AuthMethods.HasTOTP {
		items = append(items, display.DefinitionItem{
			Term:            "TOTP MFA",
			Detail:          "",
			DetailComponent: totpBadge(d.User.TOTPEnabled),
		})
	}
	return items
}

// oauthSubjectCell is the OAuth identity card's subject pair: the visible
// text is the subject id truncated to a glanceable width; the full id lives
// in the title attribute (and stays copyable there). Provider subjects are
// opaque issuer-assigned strings, sometimes long — truncation is display
// only, never a rewrite of the value.
func oauthSubjectShort(ea identitymodel.ExternalAccount) string {
	const subjectShowLen = 14
	if len(ea.Subject) <= subjectShowLen {
		return ea.Subject
	}
	return ea.Subject[:subjectShowLen] + "…"
}

func oauthSubjectTitle(ea identitymodel.ExternalAccount) string {
	return "Subject: " + ea.Subject
}

// verificationTTLCopy is the verification-token lifetime row's honest answer.
// The TTL lives in usermgmt's consumer-side EmailVerificationConfig and the
// published Service surface exposes NO accessor for it — the panel cannot
// read the configured value, so it says who sets it and names the library
// default instead of inventing a number.
func verificationTTLCopy() string {
	return "Consumer-configured — set by the embedding application " +
		"(usermgmt default: 24h); the panel cannot read the configured value."
}

// signOutAbsentCopy is the self-service card's honest fallback when the
// consumer has not mounted a sign-out endpoint: the panel owns no session
// routes, so the card names the missing configuration instead of rendering
// a link that would 404.
func signOutAbsentCopy() string {
	return "Sign-out is mounted by the embedding application via Config.LogoutURL; " +
		"no sign-out URL is configured, so this panel has no sign-out link."
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
	Tenants []tenantRow
	Total   int
	Search  string
	// Status is the active lifecycle filter ("", "active", "suspended",
	// "deleted"); empty renders the All chip as current.
	Status string
	// Sort is the active ordering ("name" default, "members").
	Sort string
	// StatusCounts reports the matched tenants per lifecycle state so the
	// filter chips carry live counts of the current search.
	StatusCounts map[string]int
	BasePath     string
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

	// RoleDistribution answers "who holds which roles here" at a glance:
	// one count per assignable role that at least one member holds, in
	// AssignableRoles order. Empty when the roster is empty — the identity
	// card renders a dash rather than a fabricated zero row.
	RoleDistribution []roleCount

	// TotalMembers is the FULL roster size (F157): Members is the visible
	// page once the roster outgrows Config.PageSize, so confirms and the
	// identity card must quote the whole tenant, not the page.
	TotalMembers int

	// AdminTierCount is the FULL-roster count of admin/owner holders. The
	// last-admin confirm escalations use it instead of scanning the visible
	// page: a paginated roster must not hide a second admin from the guard.
	AdminTierCount int

	// Page/TotalPages drive the roster pager (F157). TotalPages <= 1
	// renders no pager; RosterURL is the pager's base (page param appended).
	Page       int
	TotalPages int
	RosterURL  string

	// RosterExportURL is the CSV export of the FULL roster (F156).
	RosterExportURL string

	// TransferURL is the ownership-transfer POST target (T22). Set only on
	// the super-admin tenant detail; empty hides the transfer form entirely
	// (tenant-admin mode has no tenant lifecycle surface).
	TransferURL string

	// TransferCandidates are the members eligible to RECEIVE ownership (the
	// target select). Each entry names the person AND carries the composed
	// two-sided confirm body for that pairing (F163).
	TransferCandidates []transferCandidate

	// Tombstone is set when the tenant is DELETED (F154): the detail page
	// renders the read-only tombstone and never the live view.
	Tombstone *tenantTombstone
}

// transferCandidate is one ownership-transfer target option: the person named
// both in the select and in the per-pairing confirm body.
type transferCandidate struct {
	Email       string
	ConfirmBody string
}

// tenantTombstone is a deleted tenant's read-only identity (F154). Every
// field is honest: Recorded=false means the audit window no longer holds the
// TenantDeleted entry, and the view says "not recorded" instead of guessing.
type tenantTombstone struct {
	DeletedAt time.Time
	DeletedBy string
	// Recorded is true only when the audit window yielded the deletion entry.
	Recorded bool
}

// roleCount is one row of the tenant identity card's role distribution: how
// many members hold a given role in this tenant.
type roleCount struct {
	Role  identitymodel.Role
	Count int
}

// roleDistribution counts, per assignable role, how many of members hold it.
// Roles nobody holds are omitted: a list of zeros is not an answer.
func roleDistribution(members []memberRow) []roleCount {
	counts := make(map[identitymodel.Role]int, len(identitymodel.AssignableRoles()))
	for _, m := range members {
		for _, r := range m.Roles {
			counts[r]++
		}
	}
	dist := make([]roleCount, 0, len(counts))
	for _, r := range identitymodel.AssignableRoles() {
		if counts[r] > 0 {
			dist = append(dist, roleCount{Role: r, Count: counts[r]})
		}
	}
	return dist
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
	AddedBy string    // the directory-resolved initiator of the grant; empty when unattributed
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

// holdsAdminTier reports whether the role set contains admin or owner — the
// roles a tenant needs at least one holder of to stay administrable.
func holdsAdminTier(roles []identitymodel.Role) bool {
	for _, r := range roles {
		if r == identitymodel.RoleAdmin || r == identitymodel.RoleOwner {
			return true
		}
	}
	return false
}


// memberRemoveConfirm builds the roster remove confirm text, escalating like
// the passkey case when the member holds the tenant's ONLY admin-tier role:
// the admin is about to leave a tenant nobody can administer.
func memberRemoveConfirm(d tenantDetailData, m memberRow) string {
	msg := "Remove " + m.displayName() + " from this tenant?"
	if holdsAdminTier(m.Roles) && d.AdminTierCount <= 1 {
		msg += " They hold the only admin or owner role here — the tenant would be left without an admin."
	}
	return msg
}

// tenantRevokeConfirm builds the user-page revoke confirm text with the same
// last-admin escalation, keyed on the per-tenant admin-tier counts the
// handler computed for the visible rows.
func tenantRevokeConfirm(d userDetailData, access tenantAccessRow) string {
	msg := "Remove " + d.User.Email + " from " + access.Name + "? They lose the roles they hold there."
	if holdsAdminTier(access.Roles) && d.TenantAdminCounts[access.TenantID] <= 1 {
		msg += " They hold the only admin or owner role there — the tenant would be left without an admin."
	}
	return msg
}

// roleDistributionSummary renders the distribution as the identity card's
// one-line answer: "owner ×1 · admin ×2 · viewer ×3".
func roleDistributionSummary(dist []roleCount) string {
	parts := make([]string, 0, len(dist))
	for _, rc := range dist {
		parts = append(parts, fmt.Sprintf("%s ×%d", rc.Role, rc.Count))
	}
	return strings.Join(parts, " · ")
}

// lastAdminDowngradeCopy is the shared escalation copy for the DOWNGRADE
// half of the last-admin guard (T12): removal of the sole admin-tier member
// already escalates its confirm; changing that member's ROLES can leave the
// tenant just as unadministrable, and the count of admin-tier holders is
// named so the admin knows exactly how thin the margin is.
func lastAdminDowngradeCopy(adminTierCount int) string {
	return fmt.Sprintf(
		"Only %d member(s) hold admin or owner in this tenant, and this change would leave it with zero — "+
			"the last admin cannot be downgraded. Grant admin or owner to another member first.",
		adminTierCount,
	)
}

// lastAdminDowngradeRefused is the server-side refusal toast for the same
// guard: the decider (es_membership_decide.go decideUpdateMemberRoles) has no
// domain-level last-admin rule, so the panel is the only guard — and it must
// refuse, not just warn.
func lastAdminDowngradeRefusalMessage() string {
	return "Role update refused: the last admin cannot be downgraded in this tenant. " +
		"Grant admin or owner to another member first."
}

// memberRoleChangeConfirm builds the roster role-change confirm text,
// escalating when the member holds the tenant's ONLY admin-tier role: a
// checkbox change that drops admin/owner strands the tenant exactly like a
// removal would.
func memberRoleChangeConfirm(d tenantDetailData, m memberRow) string {
	msg := "Change the roles of " + m.displayName() + " in this tenant?"
	if holdsAdminTier(m.Roles) && d.AdminTierCount <= 1 {
		msg += " " + lastAdminDowngradeCopy(1)
	}
	return msg
}

// tenantRoleChangeConfirm builds the user-page role-change confirm text with
// the same downgrade escalation, keyed on the per-tenant admin-tier counts
// the handler computed for the visible rows.
func tenantRoleChangeConfirm(d userDetailData, access tenantAccessRow) string {
	msg := "Change " + d.User.Email + "'s roles in " + access.Name + "?"
	if holdsAdminTier(access.Roles) && d.TenantAdminCounts[access.TenantID] <= 1 {
		msg += " " + lastAdminDowngradeCopy(d.TenantAdminCounts[access.TenantID])
	}
	return msg
}

// tenantSuspendConfirm builds the suspend confirm text with the impact
// spelled out: the admin is about to cut access for a known number of
// people, and "some members" would hide the blast radius.
func tenantSuspendConfirm(d tenantDetailData) string {
	return fmt.Sprintf("Suspend this tenant? Its %d member(s) lose access until it is reactivated.", d.TotalMembers)
}

// tenantDeleteConfirm builds the delete confirm text with the same impact
// truth: deletion revokes every membership, and the admin should see how
// many before confirming something irreversible.
func tenantDeleteConfirm(d tenantDetailData) string {
	return fmt.Sprintf("Delete this tenant and revoke its %d membership(s)? This cannot be undone.", d.TotalMembers)
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
