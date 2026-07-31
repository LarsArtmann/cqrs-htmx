package usermgmt

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

func BenchmarkService_Register(b *testing.B) {
	b.ResetTimer()
	for range b.N {
		svc, _ := NewService(ServiceConfig{})
		_, _ = svc.Register(context.Background(), RegisterRequest{
			ID:    NewUserID(strings.Repeat("a", 26)),
			Email: "bench@test.com",
		})
	}
}

func BenchmarkSession_TokenMatches(b *testing.B) {
	s, _ := NewSession(NewUserID("bench"), time.Hour)
	b.ResetTimer()
	for range b.N {
		_ = s.TokenMatches(s.Token)
	}
}

var benchAggID = id.NewStreamID() //nolint:gochecknoglobals // benchmark fixture

func benchEvent(b *testing.B, eventType event.Type, payload any) event.Event {
	b.Helper()
	data, err := marshalPayload(payload)
	if err != nil {
		b.Fatalf("marshal: %v", err)
	}
	evt, err := event.NewEvent(eventType, benchAggID, aggregateTypeUser, 1, data)
	if err != nil {
		b.Fatalf("create event: %v", err)
	}
	return evt
}

func BenchmarkFoldUser_Registration(b *testing.B) {
	evt := benchEvent(b, eventUserRegistered, UserRegisteredPayload{
		Email:       "bench@test.com",
		DisplayName: "Bench",
		Roles:       []Role{RoleUser, RoleViewer},
	})
	b.ResetTimer()
	for range b.N {
		_, _ = foldUser(UserState{}, evt)
	}
}

func BenchmarkFoldUser_FourEventSequence(b *testing.B) {
	evtReg := benchEvent(b, eventUserRegistered, UserRegisteredPayload{
		Email: "fold@test.com",
		Roles: []Role{RoleUser},
	})
	evtEmail := benchEvent(b, eventEmailChanged, EmailChangedPayload{
		Email: "new@test.com",
	})
	evtDisplay := benchEvent(b, eventDisplayNameChanged, DisplayNameChangedPayload{
		DisplayName: "Folder",
	})
	evtRoles := benchEvent(b, eventRolesUpdated, RolesUpdatedPayload{
		Roles:  []Role{RoleAdmin},
		Domain: "test",
	})
	b.ResetTimer()
	for range b.N {
		state := UserState{}
		state, _ = foldUser(state, evtReg)
		state, _ = foldUser(state, evtEmail)
		state, _ = foldUser(state, evtDisplay)
		_, _ = foldUser(state, evtRoles)
	}
}

func BenchmarkReadModel_Handle(b *testing.B) {
	m := NewUserReadModel()
	evt := benchEvent(b, eventUserRegistered, UserRegisteredPayload{
		Email: "rm@test.com",
		Roles: []Role{RoleUser},
	})
	b.ResetTimer()
	for range b.N {
		_ = m.Handle(context.Background(), evt)
	}
}

func newBenchWebAuthnService(tb testing.TB) *Service {
	tb.Helper()
	svc, err := NewService(ServiceConfig{
		WebAuthn: testWebAuthnProvider{},
	})
	if err != nil {
		tb.Fatalf("NewService: %v", err)
	}
	tb.Cleanup(svc.Stop)
	return svc
}

func BenchmarkBeginRegistration(b *testing.B) {
	svc := newBenchWebAuthnService(b)
	reg, _ := svc.Register(context.Background(), RegisterRequest{
		ID: NewUserID(strings.Repeat("a", 26)), Email: "benchreg@test.com",
	})
	b.ResetTimer()
	for range b.N {
		_, _ = svc.BeginRegistration(context.Background(), reg.User.ID)
	}
}

func BenchmarkBeginLogin(b *testing.B) {
	svc := newBenchWebAuthnService(b)
	reg, _ := svc.Register(context.Background(), RegisterRequest{
		ID: NewUserID(strings.Repeat("b", 26)), Email: "benchlogin@test.com",
	})
	b.ResetTimer()
	for range b.N {
		_, _ = svc.BeginLogin(context.Background(), reg.User.Email)
	}
}

// BenchmarkStateCache_ColdVsWarm quantifies the performance improvement from
// the decider.WithStateCache that usermgmt wires by default.
//
// "cold" creates a fresh service (no cached state) — the repository replays
// all events from version 0 on each invocation.
// "warm" reuses the service (cache populated by prior writes) — the repository
// loads only events since the cached version.
//
// Run: GOEXPERIMENT=jsonv2 go test -bench=BenchmarkStateCache -benchmem -count=3
func BenchmarkStateCache_ColdVsWarm(b *testing.B) {
	ctx := context.Background()
	const eventCount = 50

	b.Run("cold", func(b *testing.B) {
		for range b.N {
			b.StopTimer()

			store := memory.NewMemoryStore()
			seedSvc, _ := NewService(ServiceConfig{EventStore: store})
			userID := seedBenchUser(b, seedSvc, eventCount)
			seedSvc.Stop()

			coldSvc, _ := NewService(ServiceConfig{EventStore: store})
			b.StartTimer()

			_ = coldSvc.ChangeEmail(ctx, userID, "cold@test.com")

			b.StopTimer()
			coldSvc.Stop()
		}
	})

	b.Run("warm", func(b *testing.B) {
		svc, _ := NewService(ServiceConfig{})
		defer svc.Close() //nolint:errcheck // benchmark cleanup

		userID := seedBenchUser(b, svc, eventCount)

		b.ResetTimer()
		for range b.N {
			_ = svc.ChangeEmail(ctx, userID, "warm@test.com")
		}
	})
}

func seedBenchUser(b *testing.B, svc *Service, eventCount int) UserID {
	b.Helper()
	ctx := context.Background()

	resp, err := svc.Register(ctx, RegisterRequest{
		ID:    GenerateUserID(),
		Email: "bench@example.com",
	})
	if err != nil {
		b.Fatalf("Register: %v", err)
	}

	for i := range eventCount {
		if err := svc.ChangeEmail(ctx, resp.User.ID, "bench-"+strconv.Itoa(i)+"@test.com"); err != nil {
			b.Fatalf("ChangeEmail %d: %v", i, err)
		}
	}

	return resp.User.ID
}
