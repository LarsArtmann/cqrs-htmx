package usermgmt

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

func TestAuditLog_RecordsRegistration(t *testing.T) {
	auditLog := NewAuditLog()
	svc := newTestServiceWithConfig(t, ServiceConfig{AuditLog: auditLog})

	registerTestUser(t, svc, "al1", "al1@test.com")

	entries := auditLog.Entries()
	if len(entries) == 0 {
		t.Fatal("expected at least one audit entry after registration")
	}

	last := entries[len(entries)-1]
	if last.Action != AuditActionRegister {
		t.Errorf("action = %q, want %s", last.Action, AuditActionRegister)
	}
	if last.EventType != "UserRegistered" {
		t.Errorf("eventType = %q, want UserRegistered", last.EventType)
	}
}

func TestAuditLog_MultipleEvents(t *testing.T) {
	auditLog := NewAuditLog()
	svc := newTestServiceWithConfig(t, ServiceConfig{AuditLog: auditLog})

	reg := registerTestUser(t, svc, "al2", "al2@test.com")

	if err := svc.ChangeEmail(context.Background(), reg.User.ID, "new@test.com"); err != nil {
		t.Fatalf("ChangeEmail: %v", err)
	}
	if err := svc.ChangeDisplayName(context.Background(), reg.User.ID, "New Name"); err != nil {
		t.Fatalf("ChangeDisplayName: %v", err)
	}

	entries := auditLog.Entries()
	if len(entries) < 3 {
		t.Fatalf("expected at least 3 audit entries, got %d", len(entries))
	}

	actions := make(map[string]bool)
	for _, e := range entries {
		actions[e.Action] = true
	}
	if !actions[AuditActionRegister] {
		t.Error("missing 'register' action")
	}
	if !actions[AuditActionEmailChanged] {
		t.Error("missing 'email_changed' action")
	}
	if !actions[AuditActionDisplayNameChanged] {
		t.Error("missing 'display_name_changed' action")
	}
}

func TestAuditLog_EntriesFor(t *testing.T) {
	auditLog := NewAuditLog()
	svc := newTestServiceWithConfig(t, ServiceConfig{AuditLog: auditLog})

	reg1 := registerTestUser(t, svc, "al3", "al3@test.com")
	reg2 := registerTestUser(t, svc, "al4", "al4@test.com")

	if err := svc.ChangeEmail(context.Background(), reg1.User.ID, "al3-new@test.com"); err != nil {
		t.Fatalf("ChangeEmail: %v", err)
	}

	entries1 := auditLog.EntriesFor(mustParseAggID(t, reg1.User.ID.Get()))
	if len(entries1) < 2 {
		t.Errorf("expected at least 2 entries for user1, got %d", len(entries1))
	}

	entries2 := auditLog.EntriesFor(mustParseAggID(t, reg2.User.ID.Get()))
	if len(entries2) < 1 {
		t.Errorf("expected at least 1 entry for user2, got %d", len(entries2))
	}

	// Verify all entries for user1 have the right aggregate ID
	expectedAggID := mustParseAggID(t, reg1.User.ID.Get())
	for _, e := range entries1 {
		if e.AggregateID != expectedAggID {
			t.Errorf("entry aggregate ID mismatch: %q vs %q", e.AggregateID, expectedAggID)
		}
	}
}

func TestAuditLog_Recent(t *testing.T) {
	auditLog := NewAuditLog()
	svc := newTestServiceWithConfig(t, ServiceConfig{AuditLog: auditLog})

	for i := range 5 {
		registerTestUser(t, svc, "ar"+string(rune('1'+i)), "ar"+string(rune('1'+i))+"@test.com")
	}

	recent := auditLog.Recent(2)
	if len(recent) != 2 {
		t.Errorf("expected 2 recent entries, got %d", len(recent))
	}

	all := auditLog.Entries()
	if len(all) < 5 {
		t.Fatalf("expected at least 5 total entries, got %d", len(all))
	}

	// Recent should return the LAST entries
	if recent[1].AggregateID != all[len(all)-1].AggregateID {
		t.Error("recent should return the last entries chronologically")
	}
}

func mustParseAggID(t *testing.T, s string) id.AggregateID {
	t.Helper()
	a, err := id.ParseAggregateID(s)
	if err != nil {
		t.Fatalf("ParseAggregateID(%q): %v", s, err)
	}
	return a
}

func TestAuditLog_Count(t *testing.T) {
	auditLog := NewAuditLog()
	svc := newTestServiceWithConfig(t, ServiceConfig{AuditLog: auditLog})

	if auditLog.Count() != 0 {
		t.Errorf("expected 0 entries initially, got %d", auditLog.Count())
	}

	registerTestUser(t, svc, "ac1", "ac1@test.com")

	if auditLog.Count() == 0 {
		t.Error("expected entries after registration")
	}
}

func TestService_AuditLog_Accessor(t *testing.T) {
	auditLog := NewAuditLog()
	svc := newTestServiceWithConfig(t, ServiceConfig{AuditLog: auditLog})

	if svc.AuditLog() != auditLog {
		t.Error("AuditLog() should return the configured audit log")
	}
}

func TestService_AuditLog_NilWhenNotConfigured(t *testing.T) {
	svc := newTestService(t)
	if svc.AuditLog() != nil {
		t.Error("AuditLog() should return nil when not configured")
	}
}
