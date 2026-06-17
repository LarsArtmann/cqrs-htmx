package usermgmt

import (
	"context"
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

// TestAuditLog_AllEventActions feeds every event type through the audit log and
// confirms auditActionFor maps each to a human-readable action.
func TestAuditLog_AllEventActions(t *testing.T) {
	log := NewAuditLog()
	ctx := context.Background()
	aggID := id.NewAggregateID()

	cases := []struct {
		eventType  event.Type
		wantAction string
	}{
		{eventUserRegistered, AuditActionRegister},
		{eventRolesUpdated, AuditActionRolesUpdated},
		{eventEmailChanged, AuditActionEmailChanged},
		{eventDisplayNameChanged, AuditActionDisplayNameChanged},
		{eventUserDeleted, AuditActionUserDeleted},
		{eventCredentialAdded, AuditActionCredentialAdded},
		{eventCredentialRemoved, AuditActionCredentialRemoved},
		{eventEmailVerified, AuditActionEmailVerified},
		{eventTOTPEnabled, AuditActionTOTPEnabled},
		{eventTOTPDisabled, AuditActionTOTPDisabled},
	}

	for i, tc := range cases {
		payload, _ := marshalPayload(UserDeletedPayload{}) // payload shape is irrelevant to the action mapping
		evt, err := event.NewEvent(tc.eventType, aggID, aggregateTypeUser, event.Version(i+1), payload)
		if err != nil {
			t.Fatalf("create %s event: %v", tc.eventType, err)
		}
		if err := log.Handle(ctx, evt); err != nil {
			t.Fatalf("handle %s: %v", tc.eventType, err)
		}
	}

	entries := log.Entries()
	if len(entries) != len(cases) {
		t.Fatalf("expected %d entries, got %d", len(cases), len(entries))
	}
	for i, tc := range cases {
		if entries[i].Action != tc.wantAction {
			t.Errorf("event %s: action = %q, want %q", tc.eventType, entries[i].Action, tc.wantAction)
		}
	}
}

func TestAuditLog_UnknownEventActionDefault(t *testing.T) {
	log := NewAuditLog()
	aggID := id.NewAggregateID()
	payload, _ := marshalPayload(UserDeletedPayload{})
	evt, _ := event.NewEvent(event.Type("user.unknown_thing"), aggID, aggregateTypeUser, 1, payload)

	if err := log.Handle(context.Background(), evt); err != nil {
		t.Fatalf("handle unknown event: %v", err)
	}
	if log.Recent(1)[0].Action != "user.unknown_thing" {
		t.Errorf("expected default action to be the raw type, got %q", log.Recent(1)[0].Action)
	}
}

func TestAuditLog_Recent_ReturnsAllWhenNGreaterThanLen(t *testing.T) {
	log := NewAuditLog()
	ctx := context.Background()
	aggID := id.NewAggregateID()
	for i := range 3 {
		payload, _ := marshalPayload(UserDeletedPayload{})
		evt, _ := event.NewEvent(eventUserRegistered, aggID, aggregateTypeUser, event.Version(i+1), payload)
		_ = log.Handle(ctx, evt)
	}

	all := log.Recent(100)
	if len(all) != 3 {
		t.Fatalf("expected all 3 entries, got %d", len(all))
	}
}

func TestEmailVerification_Send_UserNotFound(t *testing.T) {
	svc := newTestServiceWithEmailVerification(t)
	if _, err := svc.SendVerificationEmail(context.Background(), NewUserID("ghost")); err == nil {
		t.Fatal("expected error for non-existent user")
	}
}

func TestEmailVerification_SendEmailCallbackError(t *testing.T) {
	svc := newTestServiceWithEmailVerificationAndCallback(t, func(_ context.Context, _, _ string) error {
		return errors.New("smtp down")
	})
	reg := registerTestUser(t, svc, "ev-cb-err", "evcberr@test.com")

	if _, err := svc.SendVerificationEmail(context.Background(), reg.User.ID); err == nil {
		t.Fatal("expected error when send callback fails")
	}
}
