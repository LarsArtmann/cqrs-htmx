package usermgmt

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newWebAuthnTestService(t *testing.T) *Service {
	t.Helper()
	return newWebAuthnTestServiceWithConfig(t)
}

func TestWebAuthn_BeginRegistration_NotConfigured(t *testing.T) {
	svc := newTestService(t)
	_, err := svc.BeginRegistration(context.Background(), NewUserID("u1"))
	if err == nil {
		t.Fatal("expected error when WebAuthn not configured")
	}
}

func TestWebAuthn_BeginRegistration_UserNotFound(t *testing.T) {
	svc := newWebAuthnTestService(t)
	_, err := svc.BeginRegistration(context.Background(), NewUserID("nonexistent"))
	if err == nil {
		t.Fatal("expected error for nonexistent user")
	}
}

func TestWebAuthn_BeginRegistration_Success(t *testing.T) {
	svc := newWebAuthnTestService(t)
	registerTestUser(t, svc, "u1", "webauthn@test.com")

	resp, err := svc.BeginRegistration(context.Background(), NewUserID("u1"))
	if err != nil {
		t.Fatalf("BeginRegistration: %v", err)
	}
	if len(resp.Options) == 0 {
		t.Fatal("expected non-empty options")
	}
}

func TestWebAuthn_BeginRegistration_WithSessionStored(t *testing.T) {
	svc := newWebAuthnTestService(t)
	registerTestUser(t, svc, "u1", "session@test.com")

	_, err := svc.BeginRegistration(context.Background(), NewUserID("u1"))
	if err != nil {
		t.Fatalf("BeginRegistration: %v", err)
	}
	_, err = svc.webauthnSessions.Get(NewUserID("u1").Get().String())
	if err != nil {
		t.Errorf("expected session data stored, got: %v", err)
	}
}

func TestWebAuthn_FinishRegistration_NoSession(t *testing.T) {
	svc := newWebAuthnTestService(t)
	registerTestUser(t, svc, "u1", "finreg@test.com")

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	err := svc.FinishRegistration(context.Background(), NewUserID("u1"), req, "my-passkey")
	if err == nil {
		t.Fatal("expected error when no session exists")
	}
}

func TestWebAuthn_BeginLogin_NotConfigured(t *testing.T) {
	svc := newTestService(t)
	_, err := svc.BeginLogin(context.Background(), "any@test.com")
	if err == nil {
		t.Fatal("expected error when WebAuthn not configured")
	}
}

func TestWebAuthn_BeginLogin_UserNotFound(t *testing.T) {
	svc := newWebAuthnTestService(t)
	_, err := svc.BeginLogin(context.Background(), "nobody@test.com")
	if err == nil {
		t.Fatal("expected error for nonexistent user")
	}
}

func TestWebAuthn_BeginLogin_NoCredentials(t *testing.T) {
	svc := newWebAuthnTestService(t)
	registerTestUser(t, svc, "u1", "nocred@test.com")

	_, err := svc.BeginLogin(context.Background(), "nocred@test.com")
	if err == nil {
		t.Fatal("expected error for user with no credentials")
	}
}

func TestWebAuthn_BeginLogin_Success(t *testing.T) {
	svc := newWebAuthnTestService(t)
	registerTestUser(t, svc, "u1", "login@test.com")

	fakeCred := WebAuthnCredential{
		credentialCore: credentialCore{
			ID:              []byte{0x01, 0x02, 0x03},
			PublicKey:       []byte{0x04, 0x05, 0x06},
			AttestationType: "none",
			Transports:      []string{"internal"},
			BackupEligible:  true,
			BackupState:     true,
		},
	}
	if err := svc.AddCredential(context.Background(), NewUserID("u1"), fakeCred); err != nil {
		t.Fatalf("AddCredential: %v", err)
	}

	resp, err := svc.BeginLogin(context.Background(), "login@test.com")
	if err != nil {
		t.Fatalf("BeginLogin: %v", err)
	}
	if len(resp.Options) == 0 {
		t.Fatal("expected non-empty options")
	}
}

func TestWebAuthnSessionStore_CRUD(t *testing.T) {
	store := newWebAuthnSessionStore()

	_, err := store.Get("nonexistent")
	if err == nil {
		t.Fatal("expected error for missing session")
	}

	store.Save("key1", []byte("test-session-data"))
	got, err := store.Get("key1")
	if err != nil {
		t.Fatalf("Get after Save: %v", err)
	}
	if string(got) != "test-session-data" {
		t.Errorf("data = %q", string(got))
	}

	store.Delete("key1")
	_, err = store.Get("key1")
	if err == nil {
		t.Fatal("expected error after Delete")
	}
}
