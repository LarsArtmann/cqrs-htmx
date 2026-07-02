package usermgmt

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
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

var benchAggID = id.NewAggregateID() //nolint:gochecknoglobals // benchmark fixture

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
