package adminui

import (
	"cmp"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/larsartmann/templ-components/display"
	"github.com/larsartmann/templ-components/forms"
)

// errConfig returns a Rejection-family error for invalid configuration.
// (The repo bans stdlib error constructors; go-error-family is the standard.)
func errConfig(msg string) error {
	return errorfamily.NewRejection("adminui.config", msg)
}

// errForbidden is the Rejection returned when a user lacks panel access.
//
// It aliases identity-model's sentinel instead of defining an adminui-specific
// code, so a caller can recognise a denial with cqrshtmx.IsForbidden
// (code "forbidden") no matter which module produced it. The former
// "adminui.forbidden" code made the panel's 403 the one denial a shared
// classifier could not see.
var errForbidden error = identitymodel.ErrForbidden

// roleBadgeType maps a role string to a templ-components BadgeType.
func roleBadgeType(role string) display.BadgeType {
	switch role {
	case "super_admin":
		return display.BadgePrimary
	case "admin":
		return display.BadgeInfo
	case "owner":
		return display.BadgeSuccess
	case "viewer":
		return display.BadgeWarning
	default:
		return display.BadgeNeutral
	}
}

// initials returns up to two uppercase initials for an email or name.
func initials(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "?"
	}
	if at := strings.IndexByte(s, '@'); at > 0 {
		s = s[:at]
	}
	parts := strings.FieldsFunc(s, func(r rune) bool { return r == ' ' || r == '.' || r == '_' || r == '-' })
	if len(parts) == 0 {
		return strings.ToUpper(s[:1])
	}
	if len(parts) == 1 {
		return strings.ToUpper(parts[0][:min(len(parts[0]), 2)])
	}
	return strings.ToUpper(parts[0][:1] + parts[1][:1])
}

// trimTrailingSlash removes a single trailing "/" so BasePath never ends with one.
// "/" becomes "" (root mount), which is handled by the router.
func trimTrailingSlash(s string) string {
	return strings.TrimRight(s, "/")
}

// capList returns at most [MaxListRows] entries from in along with the original
// total length, so views can show a "showing N of M" note when truncated.
func capList[T any](in []T) ([]T, int) {
	total := len(in)
	if total <= MaxListRows {
		return in, total
	}
	return in[:MaxListRows], total
}

// listPageSize is the number of rows per page on paginated lists (users,
// tenants, audit). Every row stays reachable — the pagination footer advances
// the window — while a single page render stays bounded on large datasets.
const listPageSize = 50

// pageBounds clamps a 1-based page request against a total row count and
// returns the slice window plus footer metadata: offset/limit select the
// page's rows, totalPages is always >= 1 (an empty list is one empty page),
// and current is the clamped page number. pageSize <= 0 falls back to
// listPageSize.
func pageBounds(page, total, pageSize int) (offset, limit, totalPages, current int) {
	if pageSize <= 0 {
		pageSize = listPageSize
	}
	totalPages = max((total+pageSize-1)/pageSize, 1)
	current = min(max(page, 1), totalPages)
	offset = (current - 1) * pageSize
	limit = max(min(pageSize, total-offset), 0)
	return offset, limit, totalPages, current
}

// parsePageQuery reads the "page" query parameter, treating missing,
// malformed, or non-positive values as page 1.
func parsePageQuery(r *http.Request) int {
	n, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || n < 1 {
		return 1
	}
	return n
}

// listPageURL builds the pagination BaseURL for a list page, preserving an
// active search query so page links keep the filter applied. The pagination
// component appends or replaces the page parameter on this URL.
func listPageURL(base, search string) string {
	if search == "" {
		return base
	}
	return base + "?" + url.Values{"q": []string{search}}.Encode()
}

// parseRolesForm reads the multi-value `roles` checkbox field, trims each
// entry, deduplicates, and validates every value against AssignableRoles().
// A membership may hold several roles; none checked or an unknown value
// refuses the whole request (ok=false) — the caller shows the toast, nothing
// is dispatched, and no role is silently dropped.
func parseRolesForm(r *http.Request) ([]identitymodel.Role, bool) {
	raw := r.Form["roles"]
	if len(raw) == 0 {
		return nil, false
	}
	assignable := identitymodel.AssignableRoles()
	roles := make([]identitymodel.Role, 0, len(raw))
	seen := make(map[identitymodel.Role]bool, len(raw))
	for _, entry := range raw {
		role := identitymodel.Role(strings.TrimSpace(entry))
		if role == "" || !slices.Contains(assignable, role) {
			return nil, false
		}
		if seen[role] {
			continue
		}
		seen[role] = true
		roles = append(roles, role)
	}
	if len(roles) == 0 {
		return nil, false
	}
	return roles, true
}

// tenantSelectOptions builds the "grant tenant access" select: a disabled
// prompt, then every live tenant ordered by display name (label carries the
// slug too, because two tenants can share a display name). Deleted tenants are
// skipped — granting access inside a tombstoned tenant is a dead end.
func tenantSelectOptions(tenants []*usermgmt.Tenant) []forms.SelectOption {
	sorted := slices.Clone(tenants)
	slices.SortFunc(sorted, func(a, b *usermgmt.Tenant) int {
		return strings.Compare(strings.ToLower(a.DisplayName), strings.ToLower(b.DisplayName))
	})
	opts := make([]forms.SelectOption, 0, len(sorted)+1)
	opts = append(opts, forms.SelectOption{
		Value: "", Label: "Choose tenant…", Selected: true, Disabled: true,
	})
	for _, t := range sorted {
		if t.Deleted {
			continue
		}
		label := t.DisplayName
		if label == "" {
			label = t.Name
		}
		opts = append(opts, forms.SelectOption{
			Value:    t.ID.Get(),
			Label:    label + " · " + t.Name,
			Selected: false,
			Disabled: false,
		})
	}
	return opts
}

// withoutTenantOptions removes tenants the user already belongs to from the
// grant select. A membership is added, not replaced (the decider refuses a
// duplicate), so offering an existing tenant would only produce an error; the
// row-level role control is the correct place to change it.
func withoutTenantOptions(options []forms.SelectOption, access []tenantAccessRow) []forms.SelectOption {
	member := make(map[string]bool, len(access))
	for _, row := range access {
		member[row.TenantID] = true
	}
	out := make([]forms.SelectOption, 0, len(options))
	for _, opt := range options {
		if member[opt.Value] {
			continue
		}
		out = append(out, opt)
	}
	// Only the disabled prompt is left: every live tenant is joined, and a
	// select with nothing selectable is a dead end — return nil so the card
	// renders the member-of-every-tenant state instead of the form.
	if len(out) == 1 && out[0].Value == "" {
		return nil
	}
	return out
}

// parseTenantStatusFilter normalizes the tenants list' lifecycle filter.
// Unknown values degrade to the unfiltered list rather than an empty one — a
// hand-edited URL must not read as "no tenants exist".
func parseTenantStatusFilter(raw string) string {
	switch raw {
	case "active", "suspended", "deleted":
		return raw
	}
	return ""
}

// matchesTenantStatus applies the lifecycle filter. The precedence mirrors
// the status badge exactly (suspended wins over deleted), so a chip never
// promises a row its badge would render differently.
func matchesTenantStatus(t *usermgmt.Tenant, status string) bool {
	switch status {
	case "active":
		return !t.Suspended && !t.Deleted
	case "suspended":
		return t.Suspended
	case "deleted":
		return t.Deleted && !t.Suspended
	}
	return true
}

// tenantStatusCounts counts the matched tenants per lifecycle state (badge
// precedence), so the filter chips report the search-scoped mix.
func tenantStatusCounts(tenants []*usermgmt.Tenant) map[string]int {
	counts := map[string]int{"active": 0, "suspended": 0, "deleted": 0}
	for _, t := range tenants {
		switch {
		case t.Suspended:
			counts["suspended"]++
		case t.Deleted:
			counts["deleted"]++
		default:
			counts["active"]++
		}
	}
	return counts
}

// parseTenantSort normalizes the tenants list ordering; "name" is the
// default and any unknown value degrades to it.
func parseTenantSort(raw string) string {
	if raw == "members" {
		return raw
	}
	return "name"
}

// sortTenants orders the matched tenants in place: by display name
// (case-insensitive), or by member count descending with display name as the
// tiebreak so the order is stable.
func sortTenants(tenants []*usermgmt.Tenant, sort string, counts map[identitymodel.TenantID]int) {
	slices.SortFunc(tenants, func(a, b *usermgmt.Tenant) int {
		if sort == "members" {
			if ca, cb := counts[a.ID], counts[b.ID]; ca != cb {
				return cmp.Compare(cb, ca)
			}
		}
		return strings.Compare(strings.ToLower(a.DisplayName), strings.ToLower(b.DisplayName))
	})
}

// tenantListURL builds a tenants-list URL carrying the current search plus
// the requested lifecycle filter and ordering (defaults omitted, so the
// plain /tenants URL stays canonical).
func tenantListURL(d tenantsListData, status, sort string) string {
	params := url.Values{}
	if d.Search != "" {
		params.Set("q", d.Search)
	}
	if status != "" {
		params.Set("status", status)
	}
	if sort != "" && sort != "name" {
		params.Set("sort", sort)
	}
	if len(params) == 0 {
		return d.BasePath + "/tenants"
	}
	return d.BasePath + "/tenants?" + params.Encode()
}
