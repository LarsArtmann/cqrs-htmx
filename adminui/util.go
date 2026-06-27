package adminui

import (
	"strconv"
	"strings"
	"time"

	"github.com/larsartmann/cqrs-htmx/usermgmt/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3"
)

// errConfig returns a Rejection-family error for invalid configuration.
// (The repo bans stdlib error constructors; go-error-family is the standard.)
func errConfig(msg string) error {
	return event.NewRejection("adminui.config", msg)
}

// errForbidden is the Rejection returned when a user lacks panel access.
var errForbidden = event.NewRejection("adminui.forbidden", "access denied")

// relTime renders a coarse relative timestamp like "just now", "3m ago",
// "2h ago", or a date for older entries.
func relTime(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return strconv.Itoa(int(d.Minutes())) + "m ago"
	case d < 24*time.Hour:
		return strconv.Itoa(int(d.Hours())) + "h ago"
	case d < 30*24*time.Hour:
		return strconv.Itoa(int(d.Hours()/24)) + "d ago"
	default:
		return t.UTC().Format("Jan 2, 2006")
	}
}

// roleBadgeKind maps a role string to a badge color class.
func roleBadgeKind(role string) string {
	switch role {
	case "super_admin":
		return "accent"
	case "admin":
		return "blue"
	case "owner":
		return "green"
	case "viewer":
		return "amber"
	default:
		return ""
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

// selectedAttr returns "selected" when role is in current, empty otherwise.
// Used by templ to render the selected attribute on <option> elements.
func selectedAttr(current []usermgmt.Role, role usermgmt.Role) string {
	for _, r := range current {
		if r == role {
			return "selected"
		}
	}
	return ""
}

// badgeColor maps a badge kind string to a CSS color value.
func badgeColor(kind string) string {
	switch kind {
	case "accent":
		return "var(--accent)"
	case "blue":
		return "#2563eb"
	case "green":
		return "#16a34a"
	case "red":
		return "#dc2626"
	case "amber":
		return "#d97706"
	default:
		return "#6b7280"
	}
}
