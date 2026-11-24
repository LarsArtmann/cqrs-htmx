package adminui

import (
	"strings"
	"time"

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

// The table is immutable after init; auditLabel falls back to the raw action
// for anything unmapped, so unknown actions degrade honestly.
//
//nolint:gochecknoglobals // immutable display lookup table, read-only after init
var auditLabels = map[string]string{
	"register":             "User registered",
	"roles_updated":        "Roles updated",
	"email_changed":        "Email changed",
	"display_name_changed": "Display name changed",
	"user_deleted":         "User deleted",
	"credential_added":     "Passkey added",
	"credential_removed":   "Passkey removed",
	"credential_used":      "Passkey used",
	"email_verified":       "Email verified",
	"totp_enabled":         "MFA enabled",
	"totp_disabled":        "MFA disabled",
	"login_succeeded":      "Login succeeded",
	"login_failed":         "Login failed",
	"registration_closed":  "Registration closed",
	"session_started":      "Session started",
	"session_ended":        "Session ended",
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
// (Tenant-name matching was tried and reverted: the audit log records NO
// tenant lifecycle entries — commandAuditMiddleware covers user/session
// commands only — so a name query could never land on anything. Recorded as
// the usermgmt upstream ask in TODO_LIST item (h).)
func filterAuditEntries(entries []usermgmt.AuditEntry, needle string) []usermgmt.AuditEntry {
	if needle == "" {
		return entries
	}
	want := strings.ToLower(needle)
	out := make([]usermgmt.AuditEntry, 0, len(entries))
	for _, e := range entries {
		if strings.Contains(strings.ToLower(e.Action), want) ||
			strings.Contains(strings.ToLower(e.Email), want) ||
			strings.Contains(strings.ToLower(e.AggregateID.Get()), want) ||
			strings.Contains(strings.ToLower(auditLabel(e.Action)), want) {
			out = append(out, e)
		}
	}
	return out
}

// filterAuditRange keeps entries that occurred inside the inclusive
// date-only range. Both bounds are server-local midnight dates: the from day
// and the to day each count whole. A nil bound is open-ended. Callers pass
// nil for an unparseable bound (the handler surfaces the parse issue
// instead of filtering on it).
func filterAuditRange(entries []usermgmt.AuditEntry, from, to *time.Time) []usermgmt.AuditEntry {
	if from == nil && to == nil {
		return entries
	}
	out := make([]usermgmt.AuditEntry, 0, len(entries))
	for _, e := range entries {
		if from != nil && e.OccurredAt.Before(*from) {
			continue
		}
		if to != nil && !e.OccurredAt.Before(to.AddDate(0, 0, 1)) {
			continue // after the whole to-day: outside the inclusive range
		}
		out = append(out, e)
	}
	return out
}

// filterAuditActor keeps entries whose actor kind matches the chip. The chip
// value is compared against the audit entries' own actor-kind spelling
// (id.ActorKind.String()); an empty chip keeps every kind.
func filterAuditActor(entries []usermgmt.AuditEntry, actor string) []usermgmt.AuditEntry {
	if actor == "" {
		return entries
	}
	out := make([]usermgmt.AuditEntry, 0, len(entries))
	for _, e := range entries {
		if e.ActorID.Kind().String() == actor {
			out = append(out, e)
		}
	}
	return out
}

// auditChange is one before/after value pair for a profile-change entry —
// what the diff renderer shows as the old value struck through and the new
// value bold.
type auditChange struct {
	Field string // "Display name" | "Email"
	Old   string
	New   string
}

// auditDiffOldClass / auditDiffNewClass are the diff renderer's two emphasis
// tokens: the old value struck through, the new value bold. They live beside
// the extraction so every surface that shows a change diff renders the pair
// identically (audit_labels.go is the display source).
const (
	auditDiffOldClass = "line-through"
	auditDiffNewClass = "font-semibold"
)

// auditChangeApplies reports whether an action is a profile change whose row
// carries the diff slot: a real diff when the entry carries values, the
// honest "values not recorded" note when it does not.
func auditChangeApplies(action string) bool {
	switch action {
	case usermgmt.AuditActionDisplayNameChanged, usermgmt.AuditActionEmailChanged:
		return true
	}
	return false
}

// auditChangeFor extracts the before/after values for a profile-change entry.
//
// DATA-BLOCKED at this seam (verified 2026-11-24 against the projection and
// the event payloads, gotcha-60 discipline: the consumer checked the actual
// data, not the doc comments): usermgmt's AuditEntry carries NO change
// payload — its Detail field holds only the login-failure reason class and
// the session-end cause — and the DisplayNameChanged/EmailChanged event
// payloads hold only the NEW value, never the old one. The panel's seam
// (AuditLog + ReadModel) has no journal access, so no old value exists
// anywhere adminui can reach. Until the upstream ask lands (audit entries
// carrying change payloads), this returns ok=false for every entry and the
// table renders the honest note instead of a fabricated diff.
func auditChangeFor(_ usermgmt.AuditEntry) (auditChange, bool) {
	return auditChange{}, false
}
