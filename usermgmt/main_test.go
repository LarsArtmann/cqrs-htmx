package usermgmt

import "testing"

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
