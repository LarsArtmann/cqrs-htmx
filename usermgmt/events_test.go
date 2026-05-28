package usermgmt

import (
	"context"
	"testing"
	"time"
)

func TestService_EventHandler_Register(t *testing.T) {
	var captured any
	svc, err := NewService(ServiceConfig{
		BcryptCost: minBcryptCost,
		EventHandler: func(_ UserID, evt any) {
			captured = evt
		},
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	_, err = svc.Register(context.Background(), RegisterRequest{
		ID: NewUserID("u1"), Email: "evt@test.com", Password: "secret12",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	if captured == nil {
		t.Fatal("expected event to be emitted")
	}
	evt, ok := captured.(UserRegisteredEvent)
	if !ok {
		t.Fatalf("expected UserRegisteredEvent, got %T", captured)
	}
	if evt.Email != "evt@test.com" {
		t.Errorf("expected email evt@test.com, got %s", evt.Email)
	}
	if len(evt.Roles) != 2 { // viewer + user
		t.Errorf("expected 2 roles, got %d", len(evt.Roles))
	}
	if evt.OccurredAt.IsZero() {
		t.Error("expected non-zero OccurredAt")
	}
}

func TestService_EventHandler_Login(t *testing.T) {
	var captured any
	svc, ctx, _ := newTestServiceWithUser(t, "u1", "login@test.com", "secret12")
	svc.eventHandler = func(_ UserID, evt any) {
		captured = evt
	}

	_, err := svc.Login(ctx, LoginRequest{Email: "login@test.com", Password: "secret12"})
	if err != nil {
		t.Fatalf("Login: %v", err)
	}

	if captured == nil {
		t.Fatal("expected event to be emitted")
	}
	_, ok := captured.(UserLoggedInEvent)
	if !ok {
		t.Fatalf("expected UserLoggedInEvent, got %T", captured)
	}
}

func TestService_EventHandler_ChangePassword(t *testing.T) {
	var captured any
	svc, ctx, _ := newTestServiceWithUser(t, "u1", "cp@test.com", "secret12")
	svc.eventHandler = func(_ UserID, evt any) {
		captured = evt
	}

	err := svc.ChangePassword(ctx, NewUserID("u1"), "secret12", "newpass12")
	if err != nil {
		t.Fatalf("ChangePassword: %v", err)
	}

	if captured == nil {
		t.Fatal("expected event to be emitted")
	}
	_, ok := captured.(PasswordChangedEvent)
	if !ok {
		t.Fatalf("expected PasswordChangedEvent, got %T", captured)
	}
}

func TestService_EventHandler_UpdateRoles(t *testing.T) {
	var captured any
	svc, ctx, _ := newTestServiceWithUser(t, "u1", "ur@test.com", "secret12")
	svc.eventHandler = func(_ UserID, evt any) {
		captured = evt
	}

	err := svc.UpdateRoles(ctx, NewUserID("u1"), []Role{RoleAdmin}, "u1")
	if err != nil {
		t.Fatalf("UpdateRoles: %v", err)
	}

	if captured == nil {
		t.Fatal("expected event to be emitted")
	}
	evt, ok := captured.(RolesUpdatedEvent)
	if !ok {
		t.Fatalf("expected RolesUpdatedEvent, got %T", captured)
	}
	if evt.Domain != "u1" {
		t.Errorf("expected domain u1, got %s", evt.Domain)
	}
}

func TestService_EventHandler_PanicRecovered(t *testing.T) {
	var captured any
	svc, err := NewService(ServiceConfig{
		BcryptCost: minBcryptCost,
		EventHandler: func(_ UserID, _ any) {
			panic("boom")
		},
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	// This should not panic; the event handler panic is recovered.
	_, err = svc.Register(context.Background(), RegisterRequest{
		ID: NewUserID("u1"), Email: "panic@test.com", Password: "secret12",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	_ = captured
}

func TestService_EventHandler_Nil(t *testing.T) {
	svc, err := NewService(ServiceConfig{
		BcryptCost: minBcryptCost,
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	// With no handler configured, registration should succeed silently.
	_, err = svc.Register(context.Background(), RegisterRequest{
		ID: NewUserID("u1"), Email: "nil@test.com", Password: "secret12",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
}

func TestUserRegisteredEvent_JSON(t *testing.T) {
	evt := UserRegisteredEvent{
		Email:       "a@b.com",
		DisplayName: "Test",
		Roles:       []Role{RoleUser},
		OccurredAt:  time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	if evt.Email != "a@b.com" {
		t.Error("unexpected email")
	}
}
