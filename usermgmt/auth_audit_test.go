package usermgmt

import (
	"context"
	"encoding/json/v2"
	"errors"
	"strings"
	"testing"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/go-codec"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

// newAuditTestService builds a Service on the default in-memory store for
// auth-audit journal tests.
func newAuditTestService(t *testing.T) *Service {
	t.Helper()

	svc, err := NewService(ServiceConfig{})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	t.Cleanup(func() { _ = svc.Close() })

	return svc
}

func journalEventsByType(t *testing.T, svc *Service, evtType event.Type) []event.Event {
	t.Helper()

	journal, ok := svc.Journal().(event.Journal)
	if !ok {
		t.Fatal("service journal does not support ReadAll")
	}

	all, err := journal.ReadAll(context.Background())
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}

	var result []event.Event
	for _, evt := range all {
		if evt.Type() == evtType {
			result = append(result, evt)
		}
	}

	return result
}

func TestAuthAudit_LoginSuccessJournaled(t *testing.T) {
	t.Parallel()

	svc := newAuditTestService(t)
	userID := id.NewUserID()

	svc.authAudit.loginSuccess(context.Background(), userID, identitymodel.AuthMethodWebAuthN)

	events := journalEventsByType(t, svc, eventLoginSucceeded)
	if len(events) != 1 {
		t.Fatalf("expected exactly 1 LoginSucceeded event, got %d", len(events))
	}

	evt := events[0]

	if evt.StreamID().Get() == "" {
		t.Error("auth event must live on a non-empty Auth stream")
	}

	var payload identitymodel.LoginSucceededPayload
	if err := json.Unmarshal(evt.Payload(), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}

	if payload.Method != identitymodel.AuthMethodWebAuthN || payload.Provider != "" {
		t.Errorf("payload = %+v, want method=webauthn provider=\"\"", payload)
	}

	if evt.Metadata().ActorID != id.NewUserActor(userID) {
		t.Errorf("actor = %v, want the login user's actor", evt.Metadata().ActorID)
	}
}

func TestAuthAudit_LoginFailureCarriesReasonClassOnly(t *testing.T) {
	t.Parallel()

	svc := newAuditTestService(t)
	userID := id.NewUserID()

	svc.authAudit.loginFailure(
		context.Background(),
		userID,
		identitymodel.AuthMethodTOTP,
		identitymodel.AuthFailureCodeInvalid,
	)

	events := journalEventsByType(t, svc, eventLoginFailed)
	if len(events) != 1 {
		t.Fatalf("expected exactly 1 LoginFailed event, got %d", len(events))
	}

	payload := string(events[0].Payload())

	if !strings.Contains(payload, `"reason":"code_invalid"`) {
		t.Errorf("payload must carry the bounded reason class, got %s", payload)
	}

	for _, forbidden := range []string{"secret", "totp_code", "password"} {
		if strings.Contains(payload, forbidden) {
			t.Errorf("payload must never carry %q material, got %s", forbidden, payload)
		}
	}
}

func TestAuthAudit_RegistrationClosedZeroActor(t *testing.T) {
	t.Parallel()

	svc := newAuditTestService(t)

	svc.authAudit.registrationClosed(context.Background())

	events := journalEventsByType(t, svc, eventRegistrationClosed)
	if len(events) != 1 {
		t.Fatalf("expected exactly 1 RegistrationClosed event, got %d", len(events))
	}

	if !events[0].Metadata().ActorID.IsZero() {
		t.Errorf("registration probe carries no user identity, got actor %v", events[0].Metadata().ActorID)
	}
}

// failingSaveStore fails Save (the audit append path) while passing reads
// through to the in-memory store — the best-effort contract's adversary.
type failingSaveStore struct {
	*memory.MemoryStore
}

var errAuditSaveFailed = errors.New("audit save offline")

func (f *failingSaveStore) Save(
	ctx context.Context,
	ref id.StreamRef,
	events []event.Event,
	expectedVersion event.Version,
) error {
	return errAuditSaveFailed
}

func TestAuthAudit_BestEffortOnStoreFailure(t *testing.T) {
	t.Parallel()

	store := &failingSaveStore{MemoryStore: memory.NewMemoryStore()}
	svc, err := NewService(ServiceConfig{EventStore: store})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	t.Cleanup(func() { _ = svc.Close() })

	// MUST NOT panic or propagate: audit complements auth, never gates it.
	svc.authAudit.loginSuccess(context.Background(), id.NewUserID(), identitymodel.AuthMethodWebAuthN)
	svc.authAudit.loginFailure(
		context.Background(),
		id.NewUserID(),
		identitymodel.AuthMethodTOTP,
		identitymodel.AuthFailureCodeInvalid,
	)
	svc.authAudit.registrationClosed(context.Background())

	journal, ok := svc.Journal().(event.Journal)
	if !ok {
		t.Fatal("service journal does not support ReadAll")
	}

	all, readErr := journal.ReadAll(context.Background())
	if readErr != nil {
		t.Fatalf("ReadAll: %v", readErr)
	}

	if len(all) != 0 {
		t.Errorf("failed saves must not leave partial events, got %d", len(all))
	}
}

func TestAuthAudit_ReadModelExcludesAuthEventTypes(t *testing.T) {
	t.Parallel()

	authTypes := map[event.Type]bool{
		eventLoginSucceeded:     true,
		eventLoginFailed:        true,
		eventRegistrationClosed: true,
	}

	for _, evtType := range allUserEventTypes {
		if authTypes[evtType] {
			t.Errorf("user event type list must not contain auth event %s", evtType)
		}
	}

	readModel := &UserReadModel{}
	for _, evtType := range readModel.EventTypes() {
		if authTypes[evtType] {
			t.Errorf("user read model must not subscribe to auth event %s", evtType)
		}
	}

	audit := NewAuditLog()
	audited := map[event.Type]bool{}
	for _, evtType := range audit.EventTypes() {
		audited[evtType] = true
	}

	for evtType := range authTypes {
		if !audited[evtType] {
			t.Errorf("AuditLog must subscribe to auth event %s", evtType)
		}
	}
}

func TestAuditLog_LoginFailedEntryCarriesDetail(t *testing.T) {
	t.Parallel()

	audit := NewAuditLog()

	userID := id.NewUserID()
	evt, err := event.New(
		eventLoginFailed,
		id.NewStreamID(),
		aggregateTypeAuth,
		event.Version(0).Increment(),
		identitymodel.LoginFailedPayload{
			SchemaVersion: currentSchemaVersion,
			Method:        identitymodel.AuthMethodTOTP,
			Reason:        identitymodel.AuthFailureCodeInvalid,
		},
		event.WithCodec(codec.JSONCodec{}),
		event.WithActor(id.NewUserActor(userID)),
	)
	if err != nil {
		t.Fatalf("event.New: %v", err)
	}

	if err := audit.Handle(context.Background(), evt); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	entries := audit.Entries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 audit entry, got %d", len(entries))
	}

	entry := entries[0]

	if entry.Action != AuditActionLoginFailed {
		t.Errorf("action = %q, want %q", entry.Action, AuditActionLoginFailed)
	}

	if entry.Detail != string(identitymodel.AuthFailureCodeInvalid) {
		t.Errorf("detail = %q, want the bounded reason class", entry.Detail)
	}

	if entry.UserID != userID {
		t.Errorf("user = %v, want the journalled actor's user", entry.UserID)
	}
}
