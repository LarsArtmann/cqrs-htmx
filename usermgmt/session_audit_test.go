package usermgmt

import (
	"context"
	"encoding/json/v2"
	"errors"
	"strings"
	"testing"
	"time"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
)

// rawInMemorySessions unwraps the auditing decorator to the underlying
// in-memory store — tests manipulate session rows directly to stage states
// (expiry) the service APIs cannot produce on demand.
func rawInMemorySessions(t *testing.T, svc *Service) *InMemorySessionStore {
	t.Helper()

	auditing, ok := svc.sessions.(*auditingSessionStore)
	if !ok {
		t.Fatalf("svc.sessions = %T, want *auditingSessionStore", svc.sessions)
	}

	inner, ok := auditing.inner.(*InMemorySessionStore)
	if !ok {
		t.Fatalf("decorated store = %T, want *InMemorySessionStore", auditing.inner)
	}

	return inner
}

func sessionStartsInJournal(t *testing.T, svc *Service) []identitymodel.SessionStartedPayload {
	t.Helper()

	events := journalEventsByType(t, svc, eventSessionStarted)

	payloads := make([]identitymodel.SessionStartedPayload, 0, len(events))
	for _, evt := range events {
		var payload identitymodel.SessionStartedPayload
		if err := json.Unmarshal(evt.Payload(), &payload); err != nil {
			t.Fatalf("unmarshal SessionStarted payload: %v", err)
		}
		payloads = append(payloads, payload)
	}

	return payloads
}

func TestSessionAudit_RegisterEmitsSessionStarted(t *testing.T) {
	t.Parallel()

	svc := newAuditTestService(t)
	reg := registerTestUser(t, svc, "sa1", "sa1@test.com")

	payloads := sessionStartsInJournal(t, svc)
	if len(payloads) != 1 {
		t.Fatalf("expected exactly 1 SessionStarted event, got %d", len(payloads))
	}

	payload := payloads[0]

	if payload.ActorID != ActorIDFromUser(reg.User.ID).PrefixedString() {
		t.Errorf("payload.ActorID = %q, want the registrant's actor", payload.ActorID)
	}

	if payload.Impersonator != "" || payload.Reason != "" {
		t.Errorf("direct-login session must not carry impersonation fields, got %+v", payload)
	}

	if payload.TokenHash != identitymodel.SessionTokenHash(reg.Session.Token) {
		t.Error("payload.TokenHash does not match the issued token's hash")
	}

	if payload.StartedAt.IsZero() || payload.ExpiresAt.Before(payload.StartedAt) {
		t.Errorf("payload window malformed: started=%v expires=%v", payload.StartedAt, payload.ExpiresAt)
	}
}

func TestSessionAudit_TokenNeverJournaled(t *testing.T) {
	t.Parallel()

	svc := newAuditTestService(t)
	reg := registerTestUser(t, svc, "sa2", "sa2@test.com")

	events := journalEventsByType(t, svc, eventSessionStarted)
	if len(events) != 1 {
		t.Fatalf("expected 1 SessionStarted event, got %d", len(events))
	}

	if body := string(events[0].Payload()); strings.Contains(body, reg.Session.Token) {
		t.Errorf("session token leaked into the journal payload: %s", body)
	}
}

func TestSessionAudit_ImpersonationForensicsSurviveSessionDeletion(t *testing.T) {
	t.Parallel()

	svc := setupTestService(t)
	ctx := context.Background()

	adminID := registerTestUser(t, svc, "imp-admin", "admin@impersonate.test").User.ID
	grantSuperAdmin(t, svc, adminID)
	targetID := registerTestUser(t, svc, "imp-target", "target@impersonate.test").User.ID

	session, err := svc.BeginImpersonation(ctx, adminID, targetID, "quarterly-access-review")
	if err != nil {
		t.Fatalf("BeginImpersonation: %v", err)
	}

	if err := svc.EndImpersonation(ctx, session.Token); err != nil {
		t.Fatalf("EndImpersonation: %v", err)
	}

	// The forensic question from cqrs-htmx#25: after the session rows are
	// gone (they are — EndImpersonation deleted the only copy), the journal
	// alone must answer "who impersonated user X, when, and why".
	starts := sessionStartsInJournal(t, svc)
	var impersonation *identitymodel.SessionStartedPayload

	for i := range starts {
		if starts[i].Impersonator != "" {
			impersonation = &starts[i]
			break
		}
	}

	if impersonation == nil {
		t.Fatal("no impersonation SessionStarted event in the journal")
	}

	if impersonation.Impersonator != ActorIDFromUser(adminID).PrefixedString() {
		t.Errorf("impersonator = %q, want %q", impersonation.Impersonator, ActorIDFromUser(adminID).PrefixedString())
	}

	if impersonation.ActorID != ActorIDFromUser(targetID).PrefixedString() {
		t.Errorf(
			"impersonated actor = %q, want the target %q",
			impersonation.ActorID,
			ActorIDFromUser(targetID).PrefixedString(),
		)
	}

	if impersonation.Reason != "quarterly-access-review" {
		t.Errorf("reason = %q, want the recorded justification", impersonation.Reason)
	}

	if impersonation.StartedAt.Before(time.Now().Add(-time.Hour)) {
		t.Errorf("started at = %v, want a current timestamp", impersonation.StartedAt)
	}

	// The end of the impersonation window is journaled too, correlated by
	// the token hash.
	ends := journalEventsByType(t, svc, eventSessionEnded)
	foundEnd := false

	for _, evt := range ends {
		var payload identitymodel.SessionEndedPayload
		if err := json.Unmarshal(evt.Payload(), &payload); err != nil {
			t.Fatalf("unmarshal SessionEnded payload: %v", err)
		}

		if payload.TokenHash == impersonation.TokenHash {
			foundEnd = true

			if payload.Cause != identitymodel.SessionEndImpersonationEnded {
				t.Errorf("end cause = %q, want %q", payload.Cause, identitymodel.SessionEndImpersonationEnded)
			}

			if payload.Impersonator != impersonation.Impersonator {
				t.Errorf("end impersonator = %q, want %q", payload.Impersonator, impersonation.Impersonator)
			}
		}
	}

	if !foundEnd {
		t.Error("no SessionEnded event correlates with the impersonation token hash")
	}

	if _, err := svc.sessions.Find(ctx, session.Token); err == nil {
		t.Error("session row still present — the durability premise is broken")
	}
}

func TestSessionAudit_LogoutEmitsSessionEndedCauseLogout(t *testing.T) {
	t.Parallel()

	svc := setupTestService(t)
	ctx := context.Background()
	reg := registerTestUser(t, svc, "imp-logout", "logout@test.com")

	if err := svc.Logout(ctx, reg.Session.Token); err != nil {
		t.Fatalf("Logout: %v", err)
	}

	ends := journalEventsByType(t, svc, eventSessionEnded)
	if len(ends) != 1 {
		t.Fatalf("expected 1 SessionEnded event, got %d", len(ends))
	}

	var payload identitymodel.SessionEndedPayload
	if err := json.Unmarshal(ends[0].Payload(), &payload); err != nil {
		t.Fatalf("unmarshal SessionEnded payload: %v", err)
	}

	if payload.Cause != identitymodel.SessionEndLogout {
		t.Errorf("cause = %q, want %q", payload.Cause, identitymodel.SessionEndLogout)
	}

	if payload.ActorID != ActorIDFromUser(reg.User.ID).PrefixedString() {
		t.Errorf("actor = %q, want the session holder", payload.ActorID)
	}

	if payload.TokenHash != identitymodel.SessionTokenHash(reg.Session.Token) {
		t.Error("token hash does not correlate with the started session")
	}
}

func TestSessionAudit_LazyExpiryEmitsExpiredCause(t *testing.T) {
	t.Parallel()

	svc := setupTestService(t)
	ctx := context.Background()
	reg := registerTestUser(t, svc, "imp-expired", "expired@test.com")

	store := rawInMemorySessions(t, svc)
	store.mu.Lock()
	for _, s := range store.sessions {
		s.ExpiresAt = time.Now().Add(-time.Hour)
	}
	store.mu.Unlock()

	if _, err := svc.Authenticate(ctx, reg.Session.Token); !errors.Is(err, ErrSessionExpired) {
		t.Fatalf("Authenticate err = %v, want ErrSessionExpired", err)
	}

	ends := journalEventsByType(t, svc, eventSessionEnded)
	if len(ends) != 1 {
		t.Fatalf("expected 1 SessionEnded event, got %d", len(ends))
	}

	var payload identitymodel.SessionEndedPayload
	if err := json.Unmarshal(ends[0].Payload(), &payload); err != nil {
		t.Fatalf("unmarshal SessionEnded payload: %v", err)
	}

	if payload.Cause != identitymodel.SessionEndExpired {
		t.Errorf("cause = %q, want %q", payload.Cause, identitymodel.SessionEndExpired)
	}
}

func TestSessionAudit_DeleteUserEmitsRevokedCause(t *testing.T) {
	t.Parallel()

	svc := setupTestService(t)
	ctx := context.Background()
	reg := registerTestUser(t, svc, "imp-revoked", "revoked@test.com")

	if err := svc.DeleteUser(ctx, reg.User.ID, "audit test"); err != nil {
		t.Fatalf("DeleteUser: %v", err)
	}

	ends := journalEventsByType(t, svc, eventSessionEnded)
	found := false

	for _, evt := range ends {
		var payload identitymodel.SessionEndedPayload
		if err := json.Unmarshal(evt.Payload(), &payload); err != nil {
			t.Fatalf("unmarshal SessionEnded payload: %v", err)
		}

		if payload.Cause == identitymodel.SessionEndRevoked {
			found = true

			if payload.ActorID != ActorIDFromUser(reg.User.ID).PrefixedString() {
				t.Errorf("revoked actor = %q, want the deleted user", payload.ActorID)
			}

			if payload.TokenHash != "" {
				t.Errorf("user-scoped revocation must not carry a token hash, got %q", payload.TokenHash)
			}
		}
	}

	if !found {
		t.Error("no revoked-cause SessionEnded event after user deletion")
	}
}

func TestSessionAudit_UnknownTokenLogoutJournalsNothing(t *testing.T) {
	t.Parallel()

	svc := setupTestService(t)
	ctx := context.Background()

	if err := svc.Logout(ctx, "never-issued-token"); err != nil {
		t.Fatalf("Logout with unknown token: %v", err)
	}

	if ends := journalEventsByType(t, svc, eventSessionEnded); len(ends) != 0 {
		t.Errorf("unknown-token logout journaled %d end events, want 0", len(ends))
	}
}

func TestSessionAudit_AuditLogRecordsSessionLifecycle(t *testing.T) {
	t.Parallel()

	auditLog := NewAuditLog()
	svc := newTestServiceWithConfig(t, ServiceConfig{AuditLog: auditLog})
	ctx := context.Background()

	reg := registerTestUser(t, svc, "imp-auditlog", "auditlog@test.com")
	if err := svc.Logout(ctx, reg.Session.Token); err != nil {
		t.Fatalf("Logout: %v", err)
	}

	var started, ended bool

	for _, entry := range auditLog.Entries() {
		switch entry.Action {
		case AuditActionSessionStarted:
			started = true
		case AuditActionSessionEnded:
			ended = true

			if entry.Detail != string(identitymodel.SessionEndLogout) {
				t.Errorf("session_ended detail = %q, want the logout cause", entry.Detail)
			}
		}
	}

	if !started || !ended {
		t.Errorf("audit log missing session lifecycle entries: started=%v ended=%v", started, ended)
	}
}

func TestSessionAudit_DecoratorPreservesStoreContract(t *testing.T) {
	t.Parallel()

	svc := newAuditTestService(t)
	ctx := context.Background()

	// Find stays a pure passthrough: it must not journal anything.
	if _, err := svc.sessions.Find(ctx, "unknown"); err == nil {
		t.Fatal("Find with unknown token should fail")
	}

	if ends := journalEventsByType(t, svc, eventSessionEnded); len(ends) != 0 {
		t.Errorf("Find journaled %d end events, want 0", len(ends))
	}
}
