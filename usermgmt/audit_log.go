package usermgmt

import (
	"context"
	"sync"
	"time"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
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
	// Detail carries the failure reason class for login_failed entries
	// (bounded identitymodel.AuthFailureReason); empty otherwise.
	Detail string `json:"detail,omitzero"`
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
	AuditActionCredentialUsed     = "credential_used"    //nolint:gosec // G101: audit action label, not a credential
	AuditActionEmailVerified      = "email_verified"
	AuditActionTOTPEnabled        = "totp_enabled"
	AuditActionTOTPDisabled       = "totp_disabled"
	AuditActionLoginSucceeded     = "login_succeeded"
	AuditActionLoginFailed        = "login_failed"
	AuditActionRegistrationClosed = "registration_closed"
)

// DefaultAuditLogLimit is the number of most recent entries the in-memory audit
// projection retains when built with NewAuditLog: roughly the last hours of a
// normally busy deployment, sized so a compliance glance at recent activity
// survives a bounded window.
//
// The window is a convenience view, not the durable record. The journal holds
// every event (and the Auth-typed auth-audit streams hold login attempts), so a
// bounded window loses no history that matters; it only bounds memory.
const DefaultAuditLogLimit = 10_000

// AuditLog is a projection that records all user-related events as audit entries.
// It provides a queryable log of who did what, when — useful for compliance
// and security monitoring.
//
// The entry list is a bounded recent-window (DefaultAuditLogLimit by default,
// see NewAuditLogWithLimit): once the window is full the oldest entry is
// evicted as each new one arrives, so a long-running embedder cannot grow the
// slice for the lifetime of the process. Dropped() reports how many entries
// have rolled out of the window.
type AuditLog struct {
	mu      sync.RWMutex
	entries []AuditEntry // ring storage: grows to limit, then overwrites oldest
	// head is the index of the OLDEST entry once the ring is full; it stays 0
	// until then, which is what makes the "not yet full" case a plain order.
	head    int
	limit   int
	dropped int
}

// NewAuditLog returns an audit-log projection bounded to DefaultAuditLogLimit
// retained entries. Use NewAuditLogWithLimit to size the window differently.
func NewAuditLog() *AuditLog {
	return NewAuditLogWithLimit(DefaultAuditLogLimit)
}

// NewAuditLogWithLimit returns an audit-log projection that retains at most
// limit entries, discarding the oldest as new ones arrive.
//
// A non-positive limit falls back to DefaultAuditLogLimit instead of meaning
// "unbounded": an unbounded in-memory audit log is the process-lifetime leak
// this projection is bounded to avoid, so there is deliberately no way to
// re-enable it by accident. The journal remains the durable record.
//
//nolint:exhaustruct // intentional: zero-value mutex and empty ring are the correct initial state
func NewAuditLogWithLimit(limit int) *AuditLog {
	if limit <= 0 {
		limit = DefaultAuditLogLimit
	}

	return &AuditLog{limit: limit}
}

func (*AuditLog) Name() string { return "audit-log" }

func (*AuditLog) EventTypes() []event.Type { return allAuditedEventTypes }

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

	// login_failed entries carry the bounded reason class as Detail —
	// decoded from the payload, never from raw error text.
	if evt.Type() == eventLoginFailed {
		if p, err := event.DecodePayloadAuto[identitymodel.LoginFailedPayload](evt); err == nil {
			entry.Detail = string(p.Reason)
		}
	}

	a.appendEntry(entry)

	return nil
}

// appendEntry adds entry to the ring, evicting the oldest retained entry once
// the window is full. Callers must hold the write lock.
func (a *AuditLog) appendEntry(entry AuditEntry) {
	if len(a.entries) < a.limit {
		a.entries = append(a.entries, entry)

		return
	}

	a.entries[a.head] = entry
	a.head = (a.head + 1) % a.limit
	a.dropped++
}

// entryAt returns the i-th retained entry in chronological order (0 = oldest).
// Callers must hold at least the read lock, and i must be < len(a.entries).
func (a *AuditLog) entryAt(i int) AuditEntry {
	return a.entries[(a.head+i)%a.limit]
}

// auditActions maps each audited event type to its stable action vocabulary.
// Table-driven: the vocabulary grows with the event catalog without growing
// branching complexity.
var auditActions = map[event.Type]string{ //nolint:gochecknoglobals // table-driven vocabulary, single package-private map
	eventUserRegistered:     AuditActionRegister,
	eventRolesUpdated:       AuditActionRolesUpdated,
	eventEmailChanged:       AuditActionEmailChanged,
	eventDisplayNameChanged: AuditActionDisplayNameChanged,
	eventUserDeleted:        AuditActionUserDeleted,
	eventCredentialAdded:    AuditActionCredentialAdded,
	eventCredentialRemoved:  AuditActionCredentialRemoved,
	eventCredentialUsed:     AuditActionCredentialUsed,
	eventEmailVerified:      AuditActionEmailVerified,
	eventTOTPEnabled:        AuditActionTOTPEnabled,
	eventTOTPDisabled:       AuditActionTOTPDisabled,
	eventLoginSucceeded:     AuditActionLoginSucceeded,
	eventLoginFailed:        AuditActionLoginFailed,
	eventRegistrationClosed: AuditActionRegistrationClosed,
}

func auditActionFor(t event.Type) string {
	if action, ok := auditActions[t]; ok {
		return action
	}

	return string(t)
}

// Entries returns a copy of the retained audit entries, ordered
// chronologically (oldest first).
func (a *AuditLog) Entries() []AuditEntry {
	a.mu.RLock()
	defer a.mu.RUnlock()
	result := make([]AuditEntry, len(a.entries))
	for i := range result {
		result[i] = a.entryAt(i)
	}

	return result
}

// EntriesFor returns audit entries for a specific user (by aggregate ID).
// Use id.ParseStreamID(user.ID.Get()) to convert a usermgmt.UserID.
func (a *AuditLog) EntriesFor(aggregateID id.StreamID) []AuditEntry {
	a.mu.RLock()
	defer a.mu.RUnlock()
	var result []AuditEntry
	for i := range a.entries {
		if e := a.entryAt(i); e.AggregateID == aggregateID {
			result = append(result, e)
		}
	}

	return result
}

// Count returns the number of entries currently retained. It never exceeds
// Limit(); evicted entries are counted by Dropped().
func (a *AuditLog) Count() int {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return len(a.entries)
}

// Limit returns the maximum number of entries the window retains.
func (a *AuditLog) Limit() int {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return a.limit
}

// Dropped returns how many entries have been evicted from the window since
// construction. Count()+Dropped() is the number of events the projection has
// handled.
func (a *AuditLog) Dropped() int {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return a.dropped
}

// Recent returns the last n retained audit entries in chronological order
// (or all of them if n > Count()).
// A non-positive n returns an empty slice.
func (a *AuditLog) Recent(n int) []AuditEntry {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if n <= 0 {
		return []AuditEntry{}
	}
	if n > len(a.entries) {
		n = len(a.entries)
	}
	start := len(a.entries) - n
	result := make([]AuditEntry, n)
	for i := range result {
		result[i] = a.entryAt(start + i)
	}

	return result
}

var _ projection.Projection = (*AuditLog)(nil)
