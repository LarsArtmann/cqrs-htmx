package adminui

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/a-h/templ"
)

// The i18n seam (plan T16, F114–F120): a MINIMAL override mechanism, not a
// translation engine. One display-label table (the single source for the
// covered string set — nav, chrome, confirms, empty states, page headers,
// table headers, primary actions), a consumer override map merged at [New],
// honest fallback (an unknown key renders the KEY ITSELF, never blank),
// lang/dir attributes, and locale-aware date shapes. No locale files, no
// pluralization system, no message-format machinery: if a deployment needs
// those, it needs a real i18n framework, and this seam is deliberately not
// one.

// Label keys for the covered set. Keys are namespaced by surface so an
// override config reads like an inventory; the DEFAULT VALUES are exactly
// the strings the templates hardcoded before the seam, so a default-config
// render is byte-identical to the pre-seam output.
const (
	labelNavDashboard            = "nav.dashboard"
	labelNavUsers                = "nav.users"
	labelNavTenants              = "nav.tenants"
	labelNavMembers              = "nav.members"
	labelNavAudit                = "nav.audit"
	labelNavEventStore           = "nav.event_store"
	labelChromeSections          = "chrome.admin_sections"
	labelChromeMenu              = "chrome.menu"
	labelChromeSynced            = "chrome.synced"
	labelChromeSignOut           = "chrome.sign_out"
	labelChromeTheme             = "chrome.theme_toggle"
	labelChromeReadOnly          = "chrome.read_only"
	labelChromeFooter            = "chrome.footer"
	labelChromeSkip              = "chrome.skip_to_content"
	labelChromePalette           = "chrome.palette"
	labelChromePalettePH         = "chrome.palette_placeholder"
	labelChromePaletteEmpty      = "chrome.palette_empty"
	labelChromePaletteHelpTitle  = "chrome.palette_help_title"
	labelChromePaletteHelpOpen   = "chrome.palette_help_open"
	labelChromePaletteHelpHelp   = "chrome.palette_help_help"
	labelChromePaletteHelpNav    = "chrome.palette_help_nav"
	labelChromePaletteHelpJump   = "chrome.palette_help_jump"
	labelChromePaletteHelpClose  = "chrome.palette_help_close"
	labelPrefsTitle              = "prefs.title"
	labelPrefsSave               = "prefs.save"
	labelPrefsPresetName         = "prefs.preset_name"
	labelPrefsApply              = "prefs.apply"
	labelPrefsDelete             = "prefs.delete"
	labelPrefsColumns            = "prefs.columns"
	labelPrefsColAccess          = "prefs.col_access"
	labelPrefsColLastSeen        = "prefs.col_last_seen"
	labelPrefsDensity            = "prefs.density"
	labelPrefsDensityCompact     = "prefs.density_compact"
	labelPrefsDensityComfortable = "prefs.density_comfortable"
	labelPrefsNoPresets          = "prefs.no_presets"

	labelConfirmTitle  = "confirm.title"
	labelConfirmBody   = "confirm.body"
	labelConfirmCancel = "confirm.cancel"
	labelConfirmOK     = "confirm.ok"

	labelEmptyUsersSearchTitle  = "empty.users.search.title"
	labelEmptyUsersSearchHint   = "empty.users.search.hint"
	labelEmptyUsersSearchAction = "empty.users.search.action"
	labelEmptyUsersNoneTitle    = "empty.users.none.title"
	labelEmptyUsersNoneHint     = "empty.users.none.hint"
	labelEmptyUsersNoneAction   = "empty.users.none.action"

	labelEmptyTenantsSearchTitle  = "empty.tenants.search.title"
	labelEmptyTenantsSearchHint   = "empty.tenants.search.hint"
	labelEmptyTenantsSearchAction = "empty.tenants.search.action"
	labelEmptyTenantsNoneTitle    = "empty.tenants.none.title"
	labelEmptyTenantsNoneHint     = "empty.tenants.none.hint"
	labelEmptyTenantsNoneAction   = "empty.tenants.none.action"

	labelEmptyMembersTitle = "empty.members.title"
	labelEmptyMembersHint  = "empty.members.hint"

	labelEmptyAuditSearchTitle  = "empty.audit.search.title"
	labelEmptyAuditSearchHint   = "empty.audit.search.hint"
	labelEmptyAuditSearchAction = "empty.audit.search.action"
	labelEmptyAuditNoneTitle    = "empty.audit.none.title"
	labelEmptyAuditNoneHint     = "empty.audit.none.hint"

	labelEmptyActivityTitle = "empty.activity.title"
	labelEmptyActivityHint  = "empty.activity.hint"

	labelEmptyDirectoryNoneTitle    = "empty.directory.none.title"
	labelEmptyDirectoryNoneHint     = "empty.directory.none.hint"
	labelEmptyDirectoryNoneAction   = "empty.directory.none.action"
	labelEmptyDirectorySearchTitle  = "empty.directory.search.title"
	labelEmptyDirectorySearchHint   = "empty.directory.search.hint"
	labelEmptyDirectorySearchAction = "empty.directory.search.action"

	labelPageUsers           = "page.users"
	labelPageTenants         = "page.tenants"
	labelPageMembers         = "page.members"
	labelPageAudit           = "page.audit"
	labelPageImportUsers     = "page.import_users"
	labelPageDirectory       = "page.membership_directory"
	labelPageDirectorySub    = "page.membership_directory.subtitle"
	labelCardRecentActivity  = "card.recent_activity"
	labelBannerNoAdminGrant  = "banner.no_admin_grant.lead"
	labelBannerNoAdminBody   = "banner.no_admin_grant.body"
	labelSparkNoRegistrars   = "sparkline.no_registrations"
	labelDetailNoActivity    = "detail.no_activity"
	labelDetailNoExternals   = "detail.no_external_accounts"
	labelDetailNoPasskeysPre = "detail.no_passkeys.prefix"

	labelHeaderUser      = "header.users.user"
	labelHeaderAccess    = "header.users.access"
	labelHeaderStatus    = "header.status"
	labelHeaderLastSeen  = "header.users.last_seen"
	labelHeaderCreated   = "header.users.created"
	labelHeaderTenant    = "header.tenant"
	labelHeaderMembers   = "header.members_count"
	labelHeaderMember    = "header.member"
	labelHeaderRoles     = "header.roles"
	labelHeaderAdded     = "header.added"
	labelHeaderJoined    = "header.joined"
	labelHeaderAction    = "header.action"
	labelHeaderWho       = "header.who"
	labelHeaderWhen      = "header.when"
	labelHeaderAggregate = "header.aggregate"

	labelActionNewUser      = "action.new_user"
	labelActionImportUsers  = "action.import_users"
	labelActionExportCSV    = "action.export_csv"
	labelActionExportFCsv   = "action.export_filtered_csv"
	labelActionExportFJson  = "action.export_filtered_json"
	labelActionNew          = "action.new"
	labelActionImport       = "action.import"
	labelActionReactivate   = "action.reactivate"
	labelActionSuspend      = "action.suspend"
	labelActionDelete       = "action.delete"
	labelActionNewTenant    = "action.new_tenant"
	labelActionCreateUser   = "action.create_user"
	labelActionCreateTenant = "action.create_tenant"
	labelActionBackUsers    = "action.back_to_users"
	labelActionBrowse       = "action.browse_tenants"

	labelChipAll       = "chip.all"
	labelChipActive    = "chip.active"
	labelChipSuspended = "chip.suspended"
	labelChipDeleted   = "chip.deleted"
	labelSortWord      = "label.sort"
	labelSortName      = "label.name"

	labelSearchTenantsPH   = "search.tenants.placeholder"
	labelSearchTenantsAria = "search.tenants.aria"
	labelSearchUsersPH     = "search.users.placeholder"
	labelSearchUsersAria   = "search.users.aria"
	labelSearchAuditPH     = "search.audit.placeholder"
	labelSearchAuditAria   = "search.audit.aria"
	labelSearchDirPH       = "search.directory.placeholder"
	labelSearchDirAria     = "search.directory.aria"

	labelConfirmUsersDelete      = "confirm.users.delete"
	labelConfirmUsersEmail       = "confirm.users.change_email"
	labelConfirmUsersPasskey     = "confirm.users.remove_passkey" //nolint:gosec // label key, not a credential
	labelConfirmUsersRoles       = "confirm.users.revoke_platform_roles"
	labelConfirmTenantRoles      = "confirm.tenant_access.change_roles"
	labelConfirmTenantRevoke     = "confirm.tenant_access.revoke"
	labelConfirmUsersUnlink      = "confirm.users.unlink_account"
	labelConfirmTenReactivate    = "confirm.tenants.reactivate"
	labelConfirmTenReactivateBod = "confirm.tenants.reactivate.body"
	labelConfirmTenSuspend       = "confirm.tenants.suspend"
	labelConfirmTenDelete        = "confirm.tenants.delete"
	labelConfirmMemberRoles      = "confirm.members.change_roles"
	labelConfirmMemberRemove     = "confirm.members.remove"

	// T21/T22 tenant surface pack.
	labelActionTransferOwner  = "action.transfer_ownership"
	labelActionExportRoster   = "action.export_roster"
	labelConfirmTenTransfer   = "confirm.tenants.transfer"
	labelTombstoneTitle       = "tombstone.title"
	labelTombstoneNotRecorded = "tombstone.not_recorded"
	labelSearchSwitcherPH     = "search.switcher.placeholder"
	labelSearchSwitcherAria   = "search.switcher.aria"

	// Batch-3 form pack (f#20): user-facing form field labels, help texts,
	// and the form submit buttons that belong to those forms. Values are
	// exactly the strings the forms hardcoded before this batch, so a
	// default-config render stays byte-identical.
	labelFormImportFile            = "form.import.file"
	labelFormImportFileHelp        = "form.import.file.help"
	labelFormNewEmail              = "form.new.email"
	labelFormNewEmailHelp          = "form.new.email.help"
	labelFormNewDisplayName        = "form.new.display_name"
	labelFormNewDisplayNameHelp    = "form.new.display_name.help"
	labelFormProfileDisplayName    = "form.profile.display_name"
	labelFormProfileEmail          = "form.profile.email"
	labelFormProfileEmailHelp      = "form.profile.email.help"
	labelActionSaveDisplayName     = "action.save_display_name"
	labelActionSaveEmail           = "action.save_email"
	labelFormSelfDisplayName       = "form.self.display_name"
	labelFormGrantTenant           = "form.grant.tenant"
	labelActionGrantAccess         = "action.grant_access"
	labelActionSavePlatformRoles   = "action.save_platform_roles"
	labelActionRevokeAll           = "action.revoke_all"
	labelFormRestoreBackupLabel    = "form.credential_restore.backup"
	labelActionRestoreCredential   = "action.restore_credential"
	labelFormBulkRole              = "form.bulk.role"
	labelAriaBulkSelectAll         = "aria.users.select_all"
	labelActionResendVerification  = "action.resend_verification"
	labelActionGrantRole           = "action.grant_role"
	labelActionExportSelection     = "action.export_selection_csv"
	labelFormTenantSlug            = "form.tenant.slug"
	labelFormTenantSlugHelp        = "form.tenant.slug.help"
	labelFormTenantDisplayName     = "form.tenant.display_name"
	labelFormTenantDisplayNameHelp = "form.tenant.display_name.help"
	labelFormMemberEmail           = "form.member.email"
	labelFormMemberRole            = "form.member.role"
	labelFormOwnership             = "form.member.ownership"

	// Batch-3 interpolation pack (f#19): confirm bodies, escalation
	// suffixes, and aria labels that EMBED VALUES (user email, tenant name,
	// passkey name, member counts). The defaults carry the %s/%d verb form;
	// labelf interpolates through the merged table exactly like label
	// resolves static keys. Values are byte-identical to the strings the
	// handlers/templates composed by concatenation before this pack.
	labelConfirmUsersDeleteBody      = "confirm.users.delete.body"
	labelConfirmUsersEmailBody       = "confirm.users.change_email.body"
	labelConfirmUsersRestoreBody     = "confirm.users.restore_passkey.body"
	labelConfirmUsersPasskeyBody     = "confirm.users.remove_passkey.body"      //nolint:gosec // label key, not a credential
	labelConfirmUsersPasskeyLastBody = "confirm.users.remove_passkey.last.body" //nolint:gosec // label key, not a credential
	labelConfirmUsersPasskeyOnlyBody = "confirm.users.remove_passkey.only.body" //nolint:gosec // label key, not a credential
	labelConfirmUsersRevokeSelfBody  = "confirm.users.revoke_roles.self.body"
	labelConfirmUsersRevokeBody      = "confirm.users.revoke_roles.body"
	labelConfirmUsersUnlinkBody      = "confirm.users.unlink.body"
	labelConfirmTenSuspendBody       = "confirm.tenants.suspend.body"
	labelConfirmTenDeleteBody        = "confirm.tenants.delete.body"
	labelConfirmTenTransferBody      = "confirm.tenants.transfer.body"
	labelConfirmTenTransferGrantBody = "confirm.tenants.transfer.grant_only.body"
	labelConfirmMemberRemoveBody     = "confirm.members.remove.body"
	labelConfirmMemberRolesBody      = "confirm.members.change_roles.body"
	labelConfirmTenantRevokeBody     = "confirm.tenant_access.revoke.body"
	labelConfirmTenantRolesBody      = "confirm.tenant_access.change_roles.body"
	labelConfirmLastAdminHere        = "confirm.last_admin.only_holder_here"
	labelConfirmLastAdminThere       = "confirm.last_admin.only_holder_there"
	labelConfirmLastAdminDowngrade   = "confirm.last_admin.downgrade"
	labelAriaSortBy                  = "aria.users.sort_by"
	labelAriaSelectRow               = "aria.users.select_row"
	labelAriaRevokeAccess            = "aria.users.revoke_access"
	labelAriaSetRolesIn              = "aria.users.set_roles_in"
	labelAriaSetRolesFor             = "aria.members.set_roles_for"
	labelAriaMemberRemove            = "aria.members.remove"
	labelAriaSparkline               = "aria.dashboard.registrations"
)

// defaultLabels is the display table: the single source for every covered
// string. Immutable after init; label() falls back to the raw key for
// anything unmapped, so a typo renders the key itself — visible, honest,
// never blank.
//
// The VALUES repeat across keys by design ("Change roles" guards two
// surfaces; "Members" is both a nav item and a column header) — a display
// table is not a place to dedupe English strings into constants.
//
//nolint:gochecknoglobals // immutable display lookup table, read-only after init (same posture as auditLabels)
var defaultLabels = map[string]string{ //nolint:gosec // a display-value map, not credentials
	labelNavDashboard:            "Dashboard",
	labelNavUsers:                "Users",   //nolint:goconst // display value shared by design
	labelNavTenants:              "Tenants", //nolint:goconst // display value shared by design
	labelNavMembers:              "Members", //nolint:goconst // display value shared by design
	labelNavAudit:                "Audit log",
	labelNavEventStore:           "Event store",
	labelChromeSections:          "Admin sections",
	labelChromeMenu:              "Menu",
	labelChromeSynced:            "Synced",
	labelChromeSignOut:           "Sign out",
	labelChromeTheme:             "Toggle theme",
	labelChromeReadOnly:          "Read-only",
	labelChromeFooter:            "cqrs-htmx admin",
	labelChromeSkip:              "Skip to content",
	labelChromePalette:           "Command palette",
	labelChromePalettePH:         "Jump to a page, user, tenant…",
	labelChromePaletteEmpty:      "No results found.",
	labelChromePaletteHelpTitle:  "Keyboard shortcuts",
	labelChromePaletteHelpOpen:   "open the palette",
	labelChromePaletteHelpHelp:   "toggle this help",
	labelChromePaletteHelpNav:    "move through results",
	labelChromePaletteHelpJump:   "jump to the selected item",
	labelChromePaletteHelpClose:  "close the palette",
	labelPrefsTitle:              "Views & display",
	labelPrefsSave:               "Save view",
	labelPrefsPresetName:         "View name",
	labelPrefsApply:              "Apply view",
	labelPrefsDelete:             "Delete view",
	labelPrefsColumns:            "Columns:",
	labelPrefsColAccess:          "Access / tenants",
	labelPrefsColLastSeen:        "Last seen",
	labelPrefsDensity:            "Density:",
	labelPrefsDensityCompact:     "Compact",
	labelPrefsDensityComfortable: "Comfortable",
	labelPrefsNoPresets:          "No saved views yet.",

	labelConfirmTitle:  "Are you sure?",
	labelConfirmBody:   "This action cannot be undone.",
	labelConfirmCancel: "Cancel",
	labelConfirmOK:     "Confirm",

	labelEmptyUsersSearchTitle:  "No users match your search",
	labelEmptyUsersSearchHint:   "No registered user matches the search term.",
	labelEmptyUsersSearchAction: "Clear search",
	labelEmptyUsersNoneTitle:    "No users yet",
	labelEmptyUsersNoneHint:     "Registered users will appear here.",
	labelEmptyUsersNoneAction:   "Create the first user",

	labelEmptyTenantsSearchTitle:  "No tenants match your search",
	labelEmptyTenantsSearchHint:   "No tenant name or slug matches the search term.",
	labelEmptyTenantsSearchAction: "Clear search",
	labelEmptyTenantsNoneTitle:    "No tenants yet",
	labelEmptyTenantsNoneHint:     "Create a tenant to scope roles and memberships.",
	labelEmptyTenantsNoneAction:   "New tenant",

	labelEmptyMembersTitle: "No members",
	labelEmptyMembersHint:  "Add a person below to grant tenant access.",

	labelEmptyAuditSearchTitle:  "No events match your filters",
	labelEmptyAuditSearchHint:   "No recorded action matches the active filters.",
	labelEmptyAuditSearchAction: "Clear filters",
	labelEmptyAuditNoneTitle:    "No audit events",
	labelEmptyAuditNoneHint:     "User and tenant changes will be recorded here.",

	labelEmptyActivityTitle: "No activity yet",
	labelEmptyActivityHint:  "User and tenant events will appear here.",

	labelEmptyDirectoryNoneTitle:    "No memberships yet",
	labelEmptyDirectoryNoneHint:     "No one has been added to any tenant. Add a member from a tenant's page to see them here.",
	labelEmptyDirectoryNoneAction:   "Browse tenants",
	labelEmptyDirectorySearchTitle:  "No memberships match your filters",
	labelEmptyDirectorySearchHint:   "No membership matches the current search and filter chips.",
	labelEmptyDirectorySearchAction: "Clear filters",

	labelPageUsers:          "Users",
	labelPageTenants:        "Tenants",
	labelPageMembers:        "Members",
	labelPageAudit:          "Audit log",
	labelPageImportUsers:    "Import users",
	labelPageDirectory:      "Membership directory",
	labelPageDirectorySub:   "Every membership across every tenant",
	labelCardRecentActivity: "Recent activity",
	labelBannerNoAdminGrant: "No admin grant exists.",
	labelBannerNoAdminBody: "Nobody holds super_admin or admin in the global domain — you are seeing this " +
		"via a custom authorizer. Grant platform roles from a user's detail page before the last door closes.",
	labelSparkNoRegistrars:   "No registrations recorded in the audit window yet.",
	labelDetailNoActivity:    "No recorded activity yet.",
	labelDetailNoExternals:   "No external accounts linked.",
	labelDetailNoPasskeysPre: "No passkeys enrolled. The user signs in via a linked provider or enrolls",

	labelHeaderUser:      "User",
	labelHeaderAccess:    "Access",
	labelHeaderStatus:    "Status",
	labelHeaderLastSeen:  "Last seen",
	labelHeaderCreated:   "Created",
	labelHeaderTenant:    "Tenant", //nolint:goconst // display value shared by design
	labelHeaderMembers:   "Members",
	labelHeaderMember:    "Member",
	labelHeaderRoles:     "Roles", //nolint:goconst // display value shared by design
	labelHeaderAdded:     "Added",
	labelHeaderJoined:    "Joined",
	labelHeaderAction:    "Action",
	labelHeaderWho:       "Who",
	labelHeaderWhen:      "When",
	labelHeaderAggregate: "Aggregate",

	labelActionNewUser:      "New user",
	labelActionImportUsers:  "Import users",
	labelActionExportCSV:    "Export CSV",
	labelActionExportFCsv:   "Export filtered CSV",
	labelActionExportFJson:  "Export filtered JSON",
	labelActionNew:          "New",
	labelActionImport:       "Import",
	labelActionReactivate:   "Reactivate",
	labelActionSuspend:      "Suspend",
	labelActionDelete:       "Delete",
	labelActionNewTenant:    "New tenant",
	labelActionCreateUser:   "Create user",
	labelActionCreateTenant: "Create tenant",
	labelActionBackUsers:    "Back to users",
	labelActionBrowse:       "Browse tenants",

	labelChipAll:       "All",
	labelChipActive:    "Active",
	labelChipSuspended: "Suspended",
	labelChipDeleted:   "Deleted",
	labelSortWord:      "Sort",
	labelSortName:      "Name",

	labelSearchTenantsPH:   "Search tenants…",
	labelSearchTenantsAria: "Search tenants",
	labelSearchUsersPH:     "Search users…",
	labelSearchUsersAria:   "Search users",
	labelSearchAuditPH:     "Search actions, people, aggregates…",
	labelSearchAuditAria:   "Search audit log",
	labelSearchDirPH:       "Search email or name…",
	labelSearchDirAria:     "Search the directory",

	labelConfirmUsersDelete:      "Delete user",
	labelConfirmUsersEmail:       "Change email",
	labelConfirmUsersPasskey:     "Remove passkey",
	labelConfirmUsersRoles:       "Revoke platform roles",
	labelConfirmTenantRoles:      "Change roles",
	labelConfirmTenantRevoke:     "Revoke tenant access",
	labelConfirmUsersUnlink:      "Unlink account",
	labelConfirmTenReactivate:    "Reactivate tenant",
	labelConfirmTenReactivateBod: "Reactivate this tenant? Members regain access immediately.",
	labelConfirmTenSuspend:       "Suspend tenant",
	labelConfirmTenDelete:        "Delete tenant",
	labelConfirmMemberRoles:      "Change roles",
	labelConfirmMemberRemove:     "Remove member",

	labelActionTransferOwner:  "Transfer ownership",
	labelActionExportRoster:   "Export roster CSV",
	labelConfirmTenTransfer:   "Transfer ownership",
	labelTombstoneTitle:       "Deleted tenant",
	labelTombstoneNotRecorded: "not recorded",
	labelSearchSwitcherPH:     "Filter tenants…",
	labelSearchSwitcherAria:   "Filter tenants",

	labelFormImportFile:            "File",
	labelFormImportFileHelp:        "A .csv or .json file of users (email + optional display_name).",
	labelFormNewEmail:              "Email", //nolint:goconst // display value shared by design
	labelFormNewEmailHelp:          "The address the user will sign in and be found by.",
	labelFormNewDisplayName:        "Display name", //nolint:goconst // display value shared by design
	labelFormNewDisplayNameHelp:    "Optional. Shown instead of the email where space allows.",
	labelFormProfileDisplayName:    "Display name",
	labelFormProfileEmail:          "Email",
	labelFormProfileEmailHelp:      "Changing the email changes how the user signs in and is found.",
	labelActionSaveDisplayName:     "Save display name",
	labelActionSaveEmail:           "Save email",
	labelFormSelfDisplayName:       "Your display name",
	labelFormGrantTenant:           "Tenant",
	labelActionGrantAccess:         "Grant access",
	labelActionSavePlatformRoles:   "Save platform roles",
	labelActionRevokeAll:           "Revoke all",
	labelFormRestoreBackupLabel:    "Paste the credential backup JSON (exported from the user's own account page)",
	labelActionRestoreCredential:   "Restore credential",
	labelFormBulkRole:              "Role",
	labelAriaBulkSelectAll:         "Select all users on this page",
	labelActionResendVerification:  "Resend verification",
	labelActionGrantRole:           "Grant role",
	labelActionExportSelection:     "Export selection (CSV)",
	labelFormTenantSlug:            "Tenant slug",
	labelFormTenantSlugHelp:        "A unique, URL-safe identifier. It cannot be changed later.",
	labelFormTenantDisplayName:     "Display name",
	labelFormTenantDisplayNameHelp: "Optional. Shown in the panel instead of the slug.",
	labelFormMemberEmail:           "Add member email",
	labelFormMemberRole:            "Role",
	labelFormOwnership:             "Ownership:",

	labelConfirmUsersDeleteBody:      "Delete user %s? This cannot be undone.",
	labelConfirmUsersEmailBody:       "Change this user's email? OAuth providers match by email — the user may lose linked access until re-linked.",
	labelConfirmUsersRestoreBody:     "Restore a passkey from backup JSON? The uploaded credential becomes a trusted sign-in factor immediately — restore it only from a backup you control.",
	labelConfirmUsersPasskeyBody:     "Remove %s? The user can no longer sign in with it.",
	labelConfirmUsersPasskeyLastBody: "Remove %s? It is the user's last passkey — afterwards they can only sign in via a linked OAuth2 provider.",
	labelConfirmUsersPasskeyOnlyBody: "Remove %s? It is the user's ONLY way to sign in — they will be " +
		"locked out until a passkey is re-enrolled or a provider is linked. This cannot be undone lightly.",
	labelConfirmUsersRevokeSelfBody: "Revoke YOUR OWN platform roles? You will lock yourself out of this panel " +
		"until another admin re-grants them.",
	labelConfirmUsersRevokeBody:      "Revoke all platform roles for %s? They lose admin-panel access until re-granted.",
	labelConfirmUsersUnlinkBody:      "Unlink %s from %s? The account can be linked again later.",
	labelConfirmTenSuspendBody:       "Suspend this tenant? Its %d member(s) lose access until it is reactivated.",
	labelConfirmTenDeleteBody:        "Delete this tenant and revoke its %d membership(s)? This cannot be undone.",
	labelConfirmTenTransferBody:      "Transfer ownership from %s to %s? %s gains the owner role and keeps their other roles; %s loses owner but keeps any other roles they hold.",
	labelConfirmTenTransferGrantBody: "Grant ownership to %s? This tenant currently has no owner, so nobody will be demoted.",
	labelConfirmMemberRemoveBody:     "Remove %s from this tenant?",
	labelConfirmMemberRolesBody:      "Change the roles of %s in this tenant?",
	labelConfirmTenantRevokeBody:     "Remove %s from %s? They lose the roles they hold there.",
	labelConfirmTenantRolesBody:      "Change %s's roles in %s?",
	labelConfirmLastAdminHere:        "They hold the only admin or owner role here — the tenant would be left without an admin.",
	labelConfirmLastAdminThere:       "They hold the only admin or owner role there — the tenant would be left without an admin.",
	labelConfirmLastAdminDowngrade: "Only %d member(s) hold admin or owner in this tenant, and this change would leave it with zero — " +
		"the last admin cannot be downgraded. Grant admin or owner to another member first.",
	labelAriaSortBy:       "Sort by %s",
	labelAriaSelectRow:    "Select %s",
	labelAriaRevokeAccess: "Revoke access to %s",
	labelAriaSetRolesIn:   "Set roles in %s",
	labelAriaSetRolesFor:  "Set roles for %s",
	labelAriaMemberRemove: "Remove %s from this tenant",
	labelAriaSparkline:    "%d registrations over %d days",
}

// label resolves one display string from a merged label table. The fallback
// is the KEY ITSELF: an unmapped key must stay visible (a blank button or
// header is a silent lie about the surface), so missing overrides and
// untranslated keys degrade to the table key text, never to "".
func label(labels map[string]string, key string) string {
	if s, ok := labels[key]; ok {
		return s
	}
	return key
}

// labelf resolves one display string and interpolates the args into it —
// label() when no args are given, fmt.Sprintf through the resolved string
// otherwise. The f#19 interpolation seam for confirm bodies and aria labels
// that embed values (user email, tenant name, member counts): the TABLE
// carries the verb form ("Delete user %s? This cannot be undone.") so a
// consumer override controls the whole sentence, values included.
//
// Honest failure modes, both visible rather than silent: an unknown key
// renders the key itself (same contract as label), and an override that
// drops a verb renders fmt's %!(EXTRA…) marker — ugly, but it names the
// arity mistake at the exact surface that made it.
func labelf(labels map[string]string, key string, args ...any) string {
	s := label(labels, key)
	if len(args) == 0 {
		return s
	}
	return fmt.Sprintf(s, args...)
}

// mergeLabels copies the default table and applies the consumer's overrides
// on top. Unknown keys were already rejected at [New] (validateLabels), so
// every entry here replaces a real table value — an override can never be
// silently inert.
func mergeLabels(overrides map[string]string) map[string]string {
	merged := make(map[string]string, len(defaultLabels)+len(overrides))
	maps.Copy(merged, defaultLabels)
	maps.Copy(merged, overrides)
	return merged
}

// validateLabels rejects override keys that are not in the table. Honest by
// rejection: an accepted-but-inert override would hide a typo from the
// consumer forever (the page renders the default string and nothing tells
// them). Rejection at [New] names every unknown key.
func validateLabels(overrides map[string]string) error {
	if len(overrides) == 0 {
		return nil
	}
	unknown := make([]string, 0, len(overrides))
	for k := range overrides {
		if _, ok := defaultLabels[k]; !ok {
			unknown = append(unknown, k)
		}
	}
	if len(unknown) == 0 {
		return nil
	}
	slices.Sort(unknown)
	return errConfig(
		"Config.Labels contains keys that are not in the label table (overrides must name existing keys): " +
			strings.Join(unknown, ", "),
	)
}

// localeDateLayouts returns the (date, dateTime) layouts for a locale.
// HONEST SCOPE: this is a tiny shape table, not a locale engine. The default
// locale ("en", also the empty-locale default) keeps the exact shapes the
// templates hardcoded before the seam, so goldens do not churn unless a
// Locale is configured; "de" ships its true day-first dotted shape; every
// other locale falls back to the unambiguous ISO shape rather than a wrong
// guess. New locales with genuinely different shapes are added as table rows.
func localeDateLayouts(locale string) (date, dateTime string) {
	lang := localeLanguage(locale)
	switch lang {
	case "en", "":
		return "Jan 2, 2006", "Jan 2, 2006 15:04 MST"
	case "de":
		return "02.01.2006", "02.01.2006 15:04"
	default:
		return "2006-01-02", "2006-01-02 15:04"
	}
}

// localeLanguage extracts the primary language subtag ("de-AT" → "de").
func localeLanguage(locale string) string {
	if i := strings.IndexByte(locale, '-'); i > 0 {
		return strings.ToLower(locale[:i])
	}
	return strings.ToLower(locale)
}

// localeRegex validates Config.Locale as a BCP-47-shaped tag (primary
// subtag plus optional script/region subtags). Templ escapes attribute
// values anyway; the validation exists so a nonsense Locale fails at [New]
// instead of shipping a lying lang attribute to browsers and screen readers.
var localeRegex = regexp.MustCompile(`^[A-Za-z]{2,8}(-[A-Za-z0-9]{2,8})*$`)

// validateLocale admits the empty default (→ "en"), any well-formed language
// tag, and nothing else.
func validateLocale(locale string) error {
	if locale == "" || localeRegex.MatchString(locale) {
		return nil
	}
	return errConfig(
		"Config.Locale must be a BCP-47 language tag (e.g. \"en\", \"de\", \"pt-BR\"); got " + locale,
	)
}

// dirLTR / dirRTL are the only writing directions the panel ships; both the
// attribute value and the CSS logical-property layout assume one of them.
const (
	dirLTR = "ltr"
	dirRTL = "rtl"
)

// validateDir admits the empty default (→ ltr) and the two honest values.
func validateDir(dir string) error {
	switch dir {
	case "", dirLTR, dirRTL:
		return nil
	default:
		return errConfig(
			`Config.Dir must be "ltr" or "rtl"; got ` + dir,
		)
	}
}

// htmlLangAttrs builds the <html>-level lang/dir attributes from the page
// state. Always present: a lang attribute is an accessibility baseline (WCAG
// 3.1.1), and dir defaults to ltr so the layout never depends on the
// browser's guess.
func htmlLangAttrs(p pageData) templ.Attributes {
	return templ.Attributes{"lang": p.Lang, "dir": p.Dir}
}

// label resolves one display string through the page's merged label table.
// The page components read labels ONLY via this method (or the free
// label(p.Labels, key) inside sub-components that take the map) — never as
// hardcoded strings, for the covered set.
func (p pageData) label(key string) string {
	return label(p.Labels, key)
}

// labelf is the page-method form of [labelf]: it resolves and interpolates
// through the page's merged table. Templ components call p.labelf(key,
// args…) exactly where they called string concatenation before f#19.
func (p pageData) labelf(key string, args ...any) string {
	return labelf(p.Labels, key, args...)
}

// date renders a date in the page locale's shape and location. Per render:
// the location comes from the render's clock (never time.Local spelled
// literally — gosmopolitan bans it, and the render seam keeps tests
// deterministic).
func (p pageData) date(t time.Time) string {
	dateLayout, _ := localeDateLayouts(p.Lang)
	return t.In(p.loc()).Format(dateLayout)
}

// dateTime renders a date+time like [pageData.date], keeping the timezone
// suffix in the default locale's shape.
func (p pageData) dateTime(t time.Time) string {
	_, dateTimeLayout := localeDateLayouts(p.Lang)
	return t.In(p.loc()).Format(dateTimeLayout)
}

// loc returns the render's location, defaulting to the process clock's
// location when unset (defensive — h.page always sets it).
func (p pageData) loc() *time.Location {
	if p.Loc != nil {
		return p.Loc
	}
	return time.Now().Location()
}
