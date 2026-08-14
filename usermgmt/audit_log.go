package usermgmt

import (
	"context"
	"sync"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/projection/v4"
)

// AuditEntry represents a single auditable action derived from events.
//
// AggregateID and ActorID use branded/kind-discriminated types to prevent
// accidental cross-assignment. ActorID serializes as "kind:raw" (e.g.
// "user:01JX...", "system:scheduler"). UserID is populated only when the
// actor is a human user (Kind == ActorUser) — it is a convenience field
// for filtering by user identity.
type AuditEntry struct {
	EventType   event.Type  `json:"event_type"`
	AggregateID id.StreamID `json:"aggregate_id"`
	OccurredAt  time.Time   `json:"occurred_at"`
	ActorID     id.ActorID  `json:"actor_id"`
	UserID      UserID      `json:"user_id,omitzero"`
	Email       string      `json:"email,omitempty"`
	Action      string      `json:"action"`
}

// Audit action constants — the stable vocabulary recorded in AuditEntry.Action.
const (
	AuditActionRegister           = "register"
	AuditActionRolesUpdated       = "roles_updated"
	AuditActionEmailChanged       = "email_changed"
	AuditActionDisplayNameChanged = "display_name_changed"
	AuditActionUserDeleted        = "user_deleted"
	AuditActionCredentialAdded    = "credential_added"   //nolint:gosec // G101: audit action label, not a credential
	AuditActionCredentialRemoved  = "credential_removed" //nolint:gosec // G101: audit action label, not a credential
	AuditActionEmailVerified      = "email_verified"
	AuditActionTOTPEnabled        = "totp_enabled"
	AuditActionTOTPDisabled       = "totp_disabled"
)

// AuditLog is a projection that records all user-related events as audit entries.
// It provides a queryable log of who did what, when — useful for compliance
// and security monitoring.
type AuditLog struct {
	mu      sync.RWMutex
	entries []AuditEntry
}

//nolint:exhaustruct // intentional: zero-value mutex and nil slice are correct initial state
func NewAuditLog() *AuditLog {
	return &AuditLog{}
}

func (*AuditLog) Name() string { return "audit-log" }

func (*AuditLog) EventTypes() []event.Type { return allUserEventTypes }

func (a *AuditLog) Handle(_ context.Context, evt event.Event) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	//nolint:exhaustruct // Email not available at projection level
	entry := AuditEntry{
		EventType:   evt.Type(),
		AggregateID: evt.StreamID(),
		OccurredAt:  evt.OccurredAt(),
		ActorID:     evt.Metadata().ActorID,
		Action:      auditActionFor(evt.Type()),
	}

	// Only populate UserID when the actor is a human user. Bot, system,
	// and service actors have non-UserID identifiers — storing them as
	// UserID would create invalid references in the audit trail.
	if uid, ok := ActorIDAsUserID(evt.Metadata().ActorID); ok {
		entry.UserID = uid
	}

	a.entries = append(a.entries, entry)

	return nil
}

func auditActionFor(t event.Type) string {
	switch t {
	case eventUserRegistered:
		return AuditActionRegister
	case eventRolesUpdated:
		return AuditActionRolesUpdated
	case eventEmailChanged:
		return AuditActionEmailChanged
	case eventDisplayNameChanged:
		return AuditActionDisplayNameChanged
	case eventUserDeleted:
		return AuditActionUserDeleted
	case eventCredentialAdded:
		return AuditActionCredentialAdded
	case eventCredentialRemoved:
		return AuditActionCredentialRemoved
	case eventEmailVerified:
		return AuditActionEmailVerified
	case eventTOTPEnabled:
		return AuditActionTOTPEnabled
	case eventTOTPDisabled:
		return AuditActionTOTPDisabled
	default:
		return string(t)
	}
}

// Entries returns a copy of all audit entries, ordered chronologically.
func (a *AuditLog) Entries() []AuditEntry {
	a.mu.RLock()
	defer a.mu.RUnlock()
	result := make([]AuditEntry, len(a.entries))
	copy(result, a.entries)
	return result
}

// EntriesFor returns audit entries for a specific user (by aggregate ID).
// Use id.ParseStreamID(user.ID.Get()) to convert a usermgmt.UserID.
func (a *AuditLog) EntriesFor(aggregateID id.StreamID) []AuditEntry {
	a.mu.RLock()
	defer a.mu.RUnlock()
	var result []AuditEntry
	for _, e := range a.entries {
		if e.AggregateID == aggregateID {
			result = append(result, e)
		}
	}
	return result
}

// Count returns the total number of audit entries.
func (a *AuditLog) Count() int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return len(a.entries)
}

// Recent returns the last n audit entries (or all if n > len).
func (a *AuditLog) Recent(n int) []AuditEntry {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if n >= len(a.entries) {
		result := make([]AuditEntry, len(a.entries))
		copy(result, a.entries)
		return result
	}
	start := len(a.entries) - n
	result := make([]AuditEntry, n)
	copy(result, a.entries[start:])
	return result
}

var _ projection.Projection = (*AuditLog)(nil)
