package usermgmt

import (
	"context"
	"errors"
	"testing"
)

func newTestServiceConfig() ServiceConfig {
	return ServiceConfig{
		BcryptCost: minBcryptCost,
	}
}

func newTestServiceWithConfig(t *testing.T, cfg ServiceConfig) *Service {
	t.Helper()
	svc, err := NewService(cfg)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	return svc
}

func newTestServiceWithAuthz(t *testing.T) *Service {
	t.Helper()
	return newTestServiceWithConfig(t, ServiceConfig{
		Authz:      newTestAuthz(t),
		BcryptCost: minBcryptCost,
	})
}

func assertErrorIs(t *testing.T, err, target error, msg string) {
	t.Helper()
	if !errors.Is(err, target) {
		t.Errorf("expected %v, got %v (%s)", target, err, msg)
	}
}

func registerTestUser(t *testing.T, svc *Service, id, email, password string) *RegisterResponse {
	t.Helper()
	resp, err := svc.Register(context.Background(), RegisterRequest{
		ID: NewUserID(id), Email: email, Password: password,
	})
	if err != nil {
		t.Fatalf("registerTestUser %s: %v", id, err)
	}
	return resp
}
