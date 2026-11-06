package adminui

import (
	"strings"

	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	"github.com/larsartmann/templ-components/display"
)

// auditLabel maps an AuditEntry.Action to a human-readable label. The raw
// action stays the wire vocabulary; this is the display layer's one source of
// truth — the audit table, the dashboard's recent activity, and the per-user
// timeline all render through it, so a new action upgrades everywhere at once.
func auditLabel(action string) string {
	if label, ok := auditLabels[action]; ok {
		return label
	}
	return action
}

var auditLabels = map[string]string{
	"register":            "User registered",
	"roles_updated":       "Roles updated",
	"email_changed":       "Email changed",
	"display_name_changed": "Display name changed",
	"user_deleted":        "User deleted",
	"credential_added":    "Passkey added",
	"credential_removed":  "Passkey removed",
	"credential_used":     "Passkey used",
	"email_verified":      "Email verified",
	"totp_enabled":        "MFA enabled",
	"totp_disabled":       "MFA disabled",
	"login_succeeded":     "Login succeeded",
	"login_failed":        "Login failed",
	"registration_closed": "Registration closed",
	"session_started":     "Session started",
	"session_ended":       "Session ended",
}

// auditActionBadge chooses the badge tone for an action: destructive changes
// read as errors, refused logins as warnings, successful logins as success,
// and routine sessions stay quiet. Mutations keep the primary accent the
// audit table always used.
func auditActionBadge(action string) display.BadgeType {
	switch action {
	case "user_deleted":
		return display.BadgeError
	case "login_failed", "registration_closed":
		return display.BadgeWarning
	case "login_succeeded", "email_verified":
		return display.BadgeSuccess
	case "session_started", "session_ended":
		return display.BadgeNeutral
	default:
		return display.BadgePrimary
	}
}

// filterAuditEntries narrows entries (latest-first) to those matching needle
// across the action, the resolved actor email, and the aggregate id — one
// omnibox instead of three inputs. Case-insensitive substring on all three.
func filterAuditEntries(entries []usermgmt.AuditEntry, needle string) []usermgmt.AuditEntry {
	if needle == "" {
		return entries
	}
	want := strings.ToLower(needle)
	out := make([]usermgmt.AuditEntry, 0, len(entries))
	for _, e := range entries {
		if strings.Contains(strings.ToLower(e.Action), want) ||
			strings.Contains(strings.ToLower(e.Email), want) ||
			strings.Contains(strings.ToLower(e.AggregateID.String()), want) ||
			strings.Contains(strings.ToLower(auditLabel(e.Action)), want) {
			out = append(out, e)
		}
	}
	return out
}
