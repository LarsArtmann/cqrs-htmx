package usermgmt

import (
	"context"
	"sync"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
)

// AuditEntry represents a single auditable user action derived from events.
type AuditEntry struct {
	EventType   string    `json:"event_type"`
	AggregateID string    `json:"aggregate_id"`
	OccurredAt  time.Time `json:"occurred_at"`
	UserID      string    `json:"user_id"`
	Email       string    `json:"email,omitempty"`
	Action      string    `json:"action"`
}

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
		EventType:   string(evt.Type()),
		AggregateID: evt.AggregateID().String(),
		OccurredAt:  evt.OccurredAt(),
		UserID:      evt.Metadata().UserID.String(),
		Action:      auditActionFor(evt.Type()),
	}

	a.entries = append(a.entries, entry)

	return nil
}

func auditActionFor(t event.Type) string {
	switch t {
	case eventUserRegistered:
		return "register"
	case eventRolesUpdated:
		return "roles_updated"
	case eventEmailChanged:
		return "email_changed"
	case eventDisplayNameChanged:
		return "display_name_changed"
	case eventUserDeleted:
		return "user_deleted"
	case eventCredentialAdded:
		return "credential_added"
	case eventCredentialRemoved:
		return "credential_removed"
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
func (a *AuditLog) EntriesFor(aggregateID string) []AuditEntry {
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

var _ event.Projection = (*AuditLog)(nil)
