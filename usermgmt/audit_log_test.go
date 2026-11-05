package usermgmt

import (
	"context"
	"sync"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

func TestAuditLog_RecordsRegistration(t *testing.T) {
	auditLog := NewAuditLog()
	svc := newTestServiceWithConfig(t, ServiceConfig{AuditLog: auditLog})

	registerTestUser(t, svc, "al1", "al1@test.com")

	entries := auditLog.Entries()
	if len(entries) == 0 {
		t.Fatal("expected at least one audit entry after registration")
	}

	// Registration also opens the first session, journaled as a
	// SessionStarted audit fact after the UserRegistered entry — find the
	// register entry instead of assuming it is the last one.
	var register *AuditEntry

	for i := range entries {
		if entries[i].Action == AuditActionRegister {
			register = &entries[i]
			break
		}
	}

	if register == nil {
		t.Fatalf("no register entry among %d audit entries", len(entries))
	}

	if register.EventType != "UserRegistered" {
		t.Errorf("eventType = %q, want UserRegistered", register.EventType)
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

	entries1 := auditLog.EntriesFor(mustParseAggID(t, reg1.User.ID.Get().String()))
	if len(entries1) < 2 {
		t.Errorf("expected at least 2 entries for user1, got %d", len(entries1))
	}

	entries2 := auditLog.EntriesFor(mustParseAggID(t, reg2.User.ID.Get().String()))
	if len(entries2) < 1 {
		t.Errorf("expected at least 1 entry for user2, got %d", len(entries2))
	}

	// Verify all entries for user1 have the right aggregate ID
	expectedAggID := mustParseAggID(t, reg1.User.ID.Get().String())
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

func mustParseAggID(t *testing.T, s string) id.StreamID {
	t.Helper()
	a, err := id.ParseStreamID(s)
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

func TestAuditLog_BotActorDoesNotPopulateUserID(t *testing.T) {
	auditLog := NewAuditLog()
	botActor := id.NewBotActor("bot-001")
	aggID := id.NewStreamID()
	evt, err := event.New(
		eventUserRegistered, aggID, aggregateTypeUser, 1,
		map[string]string{"email": "bot@test.com"},
		event.WithActor(botActor),
	)
	if err != nil {
		t.Fatalf("event.New: %v", err)
	}
	if err := auditLog.Handle(context.Background(), evt); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	entries := auditLog.Entries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	entry := entries[0]

	var zero UserID
	if entry.UserID != zero {
		t.Errorf("UserID should be zero for bot actor, got %s", entry.UserID.Get())
	}
	if entry.ActorID.Kind() != id.ActorBot {
		t.Errorf("ActorID kind = %v, want ActorBot", entry.ActorID.Kind())
	}
	if entry.ActorID.String() != "bot-001" {
		t.Errorf("ActorID raw = %q, want %q", entry.ActorID.String(), "bot-001")
	}
}

func TestAuditLog_UserActorPopulatesUserID(t *testing.T) {
	auditLog := NewAuditLog()
	uid := GenerateUserID()
	aggID := id.NewStreamID()
	evt, err := event.New(
		eventUserRegistered, aggID, aggregateTypeUser, 1,
		map[string]string{"email": "human@test.com"},
		event.WithActor(id.NewUserActor(uid)),
	)
	if err != nil {
		t.Fatalf("event.New: %v", err)
	}
	if err := auditLog.Handle(context.Background(), evt); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	entries := auditLog.Entries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	entry := entries[0]

	if entry.UserID != uid {
		t.Errorf("UserID = %s, want %s", entry.UserID.Get(), uid.Get())
	}
	if entry.ActorID.Kind() != id.ActorUser {
		t.Errorf("ActorID kind = %v, want ActorUser", entry.ActorID.Kind())
	}
}

// handleUserRegistered records one UserRegistered event for a fresh aggregate
// and returns the aggregate ID, so ring-order assertions can name their streams.
func handleUserRegistered(t *testing.T, log *AuditLog) id.StreamID {
	t.Helper()
	aggID := id.NewStreamID()
	evt, err := event.New(
		eventUserRegistered, aggID, aggregateTypeUser, 1,
		map[string]string{"email": "ring@test.com"},
	)
	if err != nil {
		t.Fatalf("event.New: %v", err)
	}
	if err := log.Handle(context.Background(), evt); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	return aggID
}

func TestAuditLog_CapEvictsOldestAndRetainsNewest(t *testing.T) {
	const limit = 3
	auditLog := NewAuditLogWithLimit(limit)

	ids := make([]id.StreamID, 0, 5)
	for range 5 {
		ids = append(ids, handleUserRegistered(t, auditLog))
	}

	if got := auditLog.Limit(); got != limit {
		t.Errorf("Limit() = %d, want %d", got, limit)
	}
	if got := auditLog.Count(); got != limit {
		t.Errorf("Count() = %d, want retained %d", got, limit)
	}
	if got := auditLog.Dropped(); got != 2 {
		t.Errorf("Dropped() = %d, want 2", got)
	}
	if got := auditLog.Count() + auditLog.Dropped(); got != 5 {
		t.Errorf("Count()+Dropped() = %d, want the 5 handled events", got)
	}

	entries := auditLog.Entries()
	if len(entries) != limit {
		t.Fatalf("entries = %d, want %d", len(entries), limit)
	}
	for i, want := range ids[2:] {
		if entries[i].AggregateID != want {
			t.Errorf("entries[%d] = %s, want %s (oldest-first, newest retained)",
				i, entries[i].AggregateID, want)
		}
	}

	recent := auditLog.Recent(2)
	if len(recent) != 2 || recent[0].AggregateID != ids[3] || recent[1].AggregateID != ids[4] {
		t.Errorf("Recent(2) = %v, want the two newest in chronological order", recent)
	}
}

func TestAuditLog_EntriesForAfterEviction(t *testing.T) {
	auditLog := NewAuditLogWithLimit(2)

	evicted := handleUserRegistered(t, auditLog)
	retained := handleUserRegistered(t, auditLog)
	newest := handleUserRegistered(t, auditLog)

	if got := auditLog.EntriesFor(evicted); len(got) != 0 {
		t.Errorf("EntriesFor(evicted) = %d entries, want 0", len(got))
	}
	for _, want := range []id.StreamID{retained, newest} {
		if got := auditLog.EntriesFor(want); len(got) != 1 {
			t.Errorf("EntriesFor(%s) = %d entries, want 1", want, len(got))
		}
	}
}

func TestAuditLog_RecentIsOrderedAfterWrap(t *testing.T) {
	auditLog := NewAuditLogWithLimit(3)

	ids := make([]id.StreamID, 0, 5)
	for range 5 {
		ids = append(ids, handleUserRegistered(t, auditLog))
	}

	all := auditLog.Recent(0)
	if len(all) != 0 {
		t.Errorf("Recent(0) = %d entries, want an empty slice", len(all))
	}
	if got := auditLog.Recent(99); len(got) != 3 {
		t.Fatalf("Recent(99) = %d entries, want the 3 retained", len(got))
	}
	for i, want := range ids[2:] {
		if got := auditLog.Recent(3)[i].AggregateID; got != want {
			t.Errorf("Recent(3)[%d] = %s, want %s", i, got, want)
		}
	}
}

func TestAuditLog_NonPositiveLimitFallsBackToDefault(t *testing.T) {
	for _, limit := range []int{0, -5} {
		if got := NewAuditLogWithLimit(limit).Limit(); got != DefaultAuditLogLimit {
			t.Errorf("NewAuditLogWithLimit(%d).Limit() = %d, want %d", limit, got, DefaultAuditLogLimit)
		}
	}

	if got := NewAuditLog().Limit(); got != DefaultAuditLogLimit {
		t.Errorf("NewAuditLog().Limit() = %d, want %d", got, DefaultAuditLogLimit)
	}
}

func TestAuditLog_ConcurrentAppendsUnderRace(t *testing.T) {
	// B55 (C021 follow-up): concurrent Handle appends against concurrent
	// readers must be race-clean and count-exact. The window is smaller than
	// the write volume so the ring WRAPS mid-test — appendEntry's head
	// arithmetic and the dropped counter mutate under contention, and each
	// event gets a fresh aggregate ID so a lost or double-written slot is
	// detectable as a duplicate AggregateID among the retained entries.
	const (
		writers     = 8
		perWriter   = 64
		readers     = 4
		windowLimit = 64
	)

	log := NewAuditLogWithLimit(windowLimit)
	writersFinished := make(chan struct{})

	var writersDone sync.WaitGroup
	writersDone.Add(writers)
	for range writers {
		go func() {
			defer writersDone.Done()

			for range perWriter {
				aggID := id.NewStreamID()
				evt, err := event.New(
					eventUserRegistered, aggID, aggregateTypeUser, 1,
					map[string]string{"email": "race@test.com"},
					event.WithActor(id.NewUserActor(GenerateUserID())),
				)
				if err != nil {
					t.Errorf("event.New: %v", err)
					return
				}
				if err := log.Handle(context.Background(), evt); err != nil {
					t.Errorf("Handle: %v", err)
					return
				}
			}
		}()
	}

	var readersDone sync.WaitGroup
	readersDone.Add(readers)
	for range readers {
		go func() {
			defer readersDone.Done()

			for {
				select {
				case <-writersFinished:
					return
				default:
				}

				entries := log.Entries()
				if len(entries) > windowLimit {
					t.Errorf("Entries() len = %d, want <= %d", len(entries), windowLimit)
					return
				}
				log.Recent(16)
				log.Count()
				log.EntriesFor(id.NewStreamID())
			}
		}()
	}

	writersDone.Wait()
	close(writersFinished)
	readersDone.Wait()

	total := writers * perWriter
	if got := log.Count(); got != windowLimit {
		t.Fatalf("Count() = %d, want %d (window full after %d appends)", got, windowLimit, total)
	}
	if got := log.Dropped(); got != total-windowLimit {
		t.Fatalf("Dropped() = %d, want %d", got, total-windowLimit)
	}

	retained := log.Entries()
	seen := make(map[id.StreamID]struct{}, len(retained))
	for _, entry := range retained {
		if _, dup := seen[entry.AggregateID]; dup {
			t.Fatalf("duplicate AggregateID %s among retained entries — a ring slot was lost or double-written", entry.AggregateID)
		}
		seen[entry.AggregateID] = struct{}{}
	}
	if len(seen) != windowLimit {
		t.Fatalf("distinct retained aggregates = %d, want %d", len(seen), windowLimit)
	}
}

func TestAuditLog_CapAppliesToTheServiceProjectionPath(t *testing.T) {
	auditLog := NewAuditLogWithLimit(2)
	svc := newTestServiceWithConfig(t, ServiceConfig{AuditLog: auditLog})

	registerTestUser(t, svc, "cap1", "cap1@test.com")
	registerTestUser(t, svc, "cap2", "cap2@test.com")
	registerTestUser(t, svc, "cap3", "cap3@test.com")

	if got := auditLog.Count(); got != 2 {
		t.Errorf("Count() = %d, want the 2-entry window", got)
	}
	if got := auditLog.Dropped(); got != 1 {
		t.Errorf("Dropped() = %d, want 1", got)
	}
}
