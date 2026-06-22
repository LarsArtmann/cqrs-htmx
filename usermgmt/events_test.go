package usermgmt

import (
	"context"
	"testing"
)

func TestService_EventHandler_Register(t *testing.T) {
	var captured any
	svc, err := NewService(ServiceConfig{
		EventHandler: func(_ UserID, evt any) {
			captured = evt
		},
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	_, err = svc.Register(context.Background(), RegisterRequest{
		ID: NewUserID("u1"), Email: "evt@test.com",
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
}

func TestService_EventHandler_PanicRecovered(t *testing.T) {
	svc, err := NewService(ServiceConfig{
		EventHandler: func(_ UserID, _ any) {
			panic("boom")
		},
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	_, err = svc.Register(context.Background(), RegisterRequest{
		ID: NewUserID("u1"), Email: "panic@test.com",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
}

func TestService_EventHandler_Nil(t *testing.T) {
	svc, err := NewService(ServiceConfig{})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	_, err = svc.Register(context.Background(), RegisterRequest{
		ID: NewUserID("u1"), Email: "nil@test.com",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
}
