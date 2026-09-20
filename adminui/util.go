package adminui

import (
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
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
var errForbidden = errorfamily.NewRejection("adminui.forbidden", "access denied")

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

// badgeKindToType maps adminui's internal badge kind strings to
// templ-components BadgeType values.
func badgeKindToType(kind string) display.BadgeType {
	switch kind {
	case "green":
		return display.BadgeSuccess
	case "blue":
		return display.BadgeInfo
	case "amber":
		return display.BadgeWarning
	case "red":
		return display.BadgeError
	case "accent":
		return display.BadgePrimary
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
	totalPages = (total + pageSize - 1) / pageSize
	if totalPages < 1 {
		totalPages = 1
	}
	current = page
	if current < 1 {
		current = 1
	}
	if current > totalPages {
		current = totalPages
	}
	offset = (current - 1) * pageSize
	limit = min(pageSize, total-offset)
	if limit < 0 {
		limit = 0
	}
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

// roleSelectOptions builds a []forms.SelectOption from assignable roles,
// marking the member's current roles as selected. Used by the members table
// role dropdowns.
func roleSelectOptions(assignable []identitymodel.Role, current []identitymodel.Role) []forms.SelectOption {
	opts := make([]forms.SelectOption, len(assignable))
	for i, r := range assignable {
		opts[i] = forms.SelectOption{
			Value:    string(r),
			Label:    string(r),
			Selected: slices.Contains(current, r),
			Disabled: false,
		}
	}
	return opts
}
