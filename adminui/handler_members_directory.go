package adminui

import (
	"context"
	"net/http"
	"net/url"
	"slices"
	"strings"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	"github.com/larsartmann/cqrs-htmx/v4/htmx"
)

// membersDirectoryData drives the global membership directory page
// (super-admin GET /members): every membership across every tenant, resolved
// to tenant and person, with search, chips, and the shared pager state.
type membersDirectoryData struct {
	Rows  []membersDirectoryRow
	Total int // rows matching search + filters (may exceed len(Rows) when paginated)
	// Search is the active search term; TenantFilter and RoleFilter are the
	// active chip values ("" = unfiltered; an unknown value degrades to "").
	Search       string
	TenantFilter string
	RoleFilter   string
	Sort         string // "tenant" (default) or "joined"
	// TenantChips and RoleChips carry the filter vocabulary with live counts
	// of the search-scoped mix (the tenants-page chip pattern).
	TenantChips []membersDirectoryChip
	RoleChips   []membersDirectoryChip
	BasePath    string
	listPage
}

// membersDirectoryRow is one membership resolved into its tenant and person.
// The embedded memberRow carries the person (email + display name, resolved
// through the directory; bot actors keep their prefixed id) and the grant
// facts (roles, joined-at).
type membersDirectoryRow struct {
	Tenant *usermgmt.Tenant
	memberRow
}

// membersDirectoryChip is one filter chip: its wire value, display label, and
// the count of rows the OTHER active filters leave visible.
type membersDirectoryChip struct {
	Value string
	Label string
	Count int
}

// directory sort vocabulary.
const (
	directorySortTenant = "tenant"
	directorySortJoined = "joined"
)

// parseDirectorySort normalizes the directory ordering; "tenant" is the
// default and any unknown value degrades to it (the tenants-sort pattern).
func parseDirectorySort(raw string) string {
	if raw == directorySortJoined {
		return directorySortJoined
	}
	return directorySortTenant
}

// matchesDirectorySearch reports whether q occurs in the member's email or
// display name (case-insensitive). An empty q matches every row.
func matchesDirectorySearch(row membersDirectoryRow, q string) bool {
	if q == "" {
		return true
	}
	needle := strings.ToLower(q)

	return strings.Contains(strings.ToLower(row.Email), needle) ||
		strings.Contains(strings.ToLower(row.Name), needle)
}

// parseDirectoryRoleFilter normalizes the role chip: only an assignable role
// filters; anything else (including "") degrades to unfiltered so a
// hand-edited URL cannot read as "no memberships exist".
func parseDirectoryRoleFilter(raw string) string {
	if slices.Contains(identitymodel.AssignableRoles(), identitymodel.Role(raw)) {
		return raw
	}
	return ""
}

// compareDirectoryRows orders two rows by the chosen sort key, breaking every
// tie by tenant then actor so a page break never reshuffles rows.
func compareDirectoryRows(a, b membersDirectoryRow, sort string) int {
	if sort == directorySortJoined {
		if c := b.AddedAt.Compare(a.AddedAt); c != 0 { // newest first
			return c
		}
	}
	if a.Tenant != nil && b.Tenant != nil {
		if c := strings.Compare(
			strings.ToLower(a.Tenant.DisplayName), strings.ToLower(b.Tenant.DisplayName),
		); c != 0 {
			return c
		}
	}

	return strings.Compare(a.Actor.PrefixedString(), b.Actor.PrefixedString())
}

// membersDirectoryURL builds a directory URL carrying the current search,
// chips, and ordering (defaults omitted, so the plain /members URL stays
// canonical). Shared by chips, sort toggles, and the pager.
func membersDirectoryURL(basePath string, d membersDirectoryData) string {
	params := url.Values{}
	if d.Search != "" {
		params.Set("q", d.Search)
	}
	if d.TenantFilter != "" {
		params.Set("tenant", d.TenantFilter)
	}
	if d.RoleFilter != "" {
		params.Set("role", d.RoleFilter)
	}
	if d.Sort != "" && d.Sort != directorySortTenant {
		params.Set("sort", d.Sort)
	}
	if len(params) == 0 {
		return basePath + "/members"
	}

	return basePath + "/members?" + params.Encode()
}

// directoryRows resolves every tenant's memberships into display rows. The
// read model has no all-memberships listing method, so rows are resolved
// per-tenant over one AllTenants() pass — one TenantMembers query per tenant,
// the same bound tenantMemberCounts already accepts (upstream ask noted).
func (h *Handler) directoryRows(ctx context.Context) []membersDirectoryRow {
	rows := make([]membersDirectoryRow, 0)
	for _, t := range h.config.Service.AllTenants() {
		for _, m := range h.config.Service.TenantMembers(ctx, t.ID) {
			row := membersDirectoryRow{Tenant: t,
				Actor: m.ActorID, Roles: m.Roles, AddedAt: m.AddedAt}
			if userID, ok := identitymodel.ActorIDAsUserID(m.ActorID); ok {
				if user, found := h.lookupUser(userID); found {
					row.Email = user.Email
					row.Name = user.DisplayName
					row.IsUser = true
				}
			}
			rows = append(rows, row)
		}
	}

	return rows
}

// membersDirectory is the global membership directory (super-admin only).
func (h *Handler) membersDirectory(w http.ResponseWriter, r *http.Request, user *identitymodel.User) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	tenantFilter := r.URL.Query().Get("tenant")
	roleFilter := parseDirectoryRoleFilter(r.URL.Query().Get("role"))
	sort := parseDirectorySort(r.URL.Query().Get("sort"))

	rows := h.directoryRows(r.Context())
	// Chip counts are the search-scoped mix: each chip family counts rows
	// matching the search and the OTHER family's active filter, so a chip
	// never promises rows it would hide.
	tenantChips := h.directoryTenantChips(rows, q, roleFilter)
	roleChips := directoryRoleChips(rows, q, tenantFilter)

	matched := make([]membersDirectoryRow, 0, len(rows))
	for _, row := range rows {
		if tenantFilter != "" && (row.Tenant == nil || row.Tenant.ID.Get() != tenantFilter) {
			continue
		}
		if roleFilter != "" && !hasRole(row.Roles, identitymodel.Role(roleFilter)) {
			continue
		}
		if !matchesDirectorySearch(row, q) {
			continue
		}
		matched = append(matched, row)
	}
	slices.SortFunc(matched, func(a, b membersDirectoryRow) int {
		return compareDirectoryRows(a, b, sort)
	})

	offset, limit, totalPages, page := pageBounds(parsePageQuery(r), len(matched), h.config.PageSize)
	d := membersDirectoryData{
		Rows: matched[offset : offset+limit], Total: len(matched),
		Search: q, TenantFilter: tenantFilter, RoleFilter: roleFilter, Sort: sort,
		TenantChips: tenantChips, RoleChips: roleChips,
		BasePath: h.config.BasePath,
		Page:     page, TotalPages: totalPages,
	}
	p := h.page("Membership directory", patternMembers, user, r)
	if htmx.RenderPartial(r) {
		renderPartial(w, r, membersDirectoryTableContent(p, d))
		return
	}
	renderPage(w, r, membersDirectoryPage(p, d))
}

// directoryTenantChips builds the tenant filter chips: All + one per tenant,
// counts scoped to the current search and role filter.
func (h *Handler) directoryTenantChips(rows []membersDirectoryRow, q, roleFilter string) []membersDirectoryChip {
	counts := make(map[string]int)
	for _, row := range rows {
		if row.Tenant == nil || !matchesDirectorySearch(row, q) {
			continue
		}
		if roleFilter != "" && !hasRole(row.Roles, identitymodel.Role(roleFilter)) {
			continue
		}
		counts[row.Tenant.ID.Get()]++
	}
	chips := make([]membersDirectoryChip, 0, len(counts)+1)
	for _, row := range rows { // AllTenants order is already stable
		if row.Tenant == nil {
			continue
		}
		id := row.Tenant.ID.Get()
		if slices.ContainsFunc(chips, func(c membersDirectoryChip) bool { return c.Value == id }) {
			continue
		}
		label := row.Tenant.DisplayName
		if label == "" {
			label = row.Tenant.Name
		}
		chips = append(chips, membersDirectoryChip{Value: id, Label: label, Count: counts[id]})
	}

	return chips
}

// directoryRoleChips builds the role filter chips: All + one per assignable
// role, counts scoped to the current search and tenant filter.
func directoryRoleChips(rows []membersDirectoryRow, q, tenantFilter string) []membersDirectoryChip {
	counts := make(map[identitymodel.Role]int)
	for _, row := range rows {
		if !matchesDirectorySearch(row, q) {
			continue
		}
		if tenantFilter != "" && (row.Tenant == nil || row.Tenant.ID.Get() != tenantFilter) {
			continue
		}
		for _, role := range row.Roles {
			counts[role]++
		}
	}
	chips := make([]membersDirectoryChip, 0, len(identitymodel.AssignableRoles()))
	for _, role := range identitymodel.AssignableRoles() {
		chips = append(chips, membersDirectoryChip{
			Value: string(role), Label: string(role), Count: counts[role],
		})
	}

	return chips
}
