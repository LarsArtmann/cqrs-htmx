package usermgmt

import (
	"context"
	"errors"
	"testing"
)

func TestService_Close_DefaultInMemory(t *testing.T) {
	t.Parallel()
	svc, err := NewService(ServiceConfig{})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	if err := svc.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	if err := svc.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}

func TestService_GracefulClose_ActiveContext(t *testing.T) {
	t.Parallel()
	svc, err := NewService(ServiceConfig{})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := svc.GracefulClose(ctx); err != nil {
		t.Fatalf("GracefulClose: %v", err)
	}
}

func TestService_GracefulClose_CancelledContext(t *testing.T) {
	t.Parallel()
	svc, err := NewService(ServiceConfig{})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := svc.GracefulClose(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestService_Stop_Idempotent(t *testing.T) {
	t.Parallel()
	svc, err := NewService(ServiceConfig{})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	svc.Stop()
	svc.Stop()
}

func TestEventSourcedSetup_Close_Idempotent(t *testing.T) {
	t.Parallel()
	setup, err := DefaultEventSourcedSetup()
	if err != nil {
		t.Fatalf("DefaultEventSourcedSetup: %v", err)
	}
	if err := setup.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	if err := setup.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}
