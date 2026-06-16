package usermgmt

import (
	"testing"
	"time"
)

func TestVerificationTokenStore_EvictExpired(t *testing.T) {
	s := newVerificationTokenStore()
	entry := verificationEntry{
		userID:    NewUserID("u1"),
		email:     "u1@test.com",
		expiresAt: time.Now().Add(-1 * time.Second),
	}
	s.mu.Lock()
	s.tokens["expired"] = entry
	s.tokens["valid"] = verificationEntry{
		userID:    NewUserID("u2"),
		email:     "u2@test.com",
		expiresAt: time.Now().Add(1 * time.Hour),
	}
	s.mu.Unlock()

	if got := s.EvictExpired(); got != 1 {
		t.Errorf("EvictExpired() = %d, want 1", got)
	}

	s.mu.Lock()
	if _, ok := s.tokens["expired"]; ok {
		t.Error("expired token was not evicted")
	}
	if _, ok := s.tokens["valid"]; !ok {
		t.Error("valid token was evicted")
	}
	s.mu.Unlock()
}

func TestPendingTOTPStore_EvictExpired(t *testing.T) {
	s := newPendingTOTPStore()
	s.mu.Lock()
	s.secrets["expired"] = pendingTOTPSecret{
		secret:    []byte{1, 2, 3},
		expiresAt: time.Now().Add(-1 * time.Second),
	}
	s.secrets["valid"] = pendingTOTPSecret{
		secret:    []byte{4, 5, 6},
		expiresAt: time.Now().Add(1 * time.Hour),
	}
	s.mu.Unlock()

	if got := s.EvictExpired(); got != 1 {
		t.Errorf("EvictExpired() = %d, want 1", got)
	}

	s.mu.Lock()
	if _, ok := s.secrets["expired"]; ok {
		t.Error("expired pending secret was not evicted")
	}
	if _, ok := s.secrets["valid"]; !ok {
		t.Error("valid pending secret was evicted")
	}
	s.mu.Unlock()
}

func TestService_StopTerminatesEvictionGoroutines(t *testing.T) {
	svc, err := NewService(ServiceConfig{
		EmailVerification: &EmailVerificationConfig{},
		TOTPConfig:        &TOTPConfig{},
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	// Stop should complete without blocking and be idempotent.
	svc.Stop()
	svc.Stop()
}
