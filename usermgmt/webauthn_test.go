package usermgmt

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-webauthn/webauthn/webauthn"
)

func newWebAuthnTestService(t *testing.T) *Service {
	t.Helper()
	cfg := localTestWebAuthnConfig()
	cfg.RPDisplayName = "Test App"
	return newWebAuthnTestServiceWithConfig(t, cfg)
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
	if resp.Options == nil {
		t.Fatal("expected non-nil options")
	}
	if resp.Options.Response.RelyingParty.Name != "Test App" {
		t.Errorf("RP name = %q, want Test App", resp.Options.Response.RelyingParty.Name)
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
	if resp.Options == nil {
		t.Fatal("expected non-nil options")
	}
}

func TestWebAuthnServiceAdapter(t *testing.T) {
	user := &User{
		ID:          NewUserID("adapter-test"),
		Email:       "adapter@test.com",
		DisplayName: "Adapter",
		Credentials: []WebAuthnCredential{
			{credentialCore: credentialCore{
				ID:              []byte{0x01},
				PublicKey:       []byte{0x02},
				AttestationType: "none",
				Transports:      []string{"internal"},
			}},
		},
	}

	waUser := &webauthnUser{user: user}

	expectedID := NewUserID("adapter-test").Get().String()
	if string(waUser.WebAuthnID()) != expectedID {
		t.Errorf("WebAuthnID = %q, want %q", waUser.WebAuthnID(), expectedID)
	}
	if waUser.WebAuthnName() != "adapter@test.com" {
		t.Errorf("WebAuthnName = %q", waUser.WebAuthnName())
	}
	if waUser.WebAuthnDisplayName() != "Adapter" {
		t.Errorf("WebAuthnDisplayName = %q", waUser.WebAuthnDisplayName())
	}
	creds := waUser.WebAuthnCredentials()
	if len(creds) != 1 {
		t.Fatalf("expected 1 credential, got %d", len(creds))
	}
}

func TestWebAuthn_CredentialRoundTrip(t *testing.T) {
	original := WebAuthnCredential{
		credentialCore: credentialCore{
			ID:              []byte{0xAA, 0xBB, 0xCC},
			PublicKey:       []byte{0x01, 0x02},
			AttestationType: "packed",
			Transports:      []string{"usb", "nfc"},
			AAGUID:          []byte{0x00, 0x01},
			SignCount:       42,
			BackupEligible:  true,
			BackupState:     false,
			Name:            "YubiKey 5C",
		},
	}

	waCred := toWebAuthnCredential(original)
	if string(waCred.ID) != string(original.ID) {
		t.Errorf("ID mismatch")
	}
	if waCred.Authenticator.SignCount != 42 {
		t.Errorf("SignCount = %d", waCred.Authenticator.SignCount)
	}

	converted := fromWebAuthnCredential(&waCred, original.Name)
	if string(converted.ID) != string(original.ID) {
		t.Errorf("round-trip ID mismatch")
	}
	if converted.SignCount != original.SignCount {
		t.Errorf("round-trip SignCount mismatch: %d vs %d", converted.SignCount, original.SignCount)
	}
	if len(converted.Transports) != 2 {
		t.Errorf("round-trip Transports len = %d", len(converted.Transports))
	}
}

func TestWebAuthnSessionStore_CRUD(t *testing.T) {
	store := newWebAuthnSessionStore()

	_, err := store.Get("nonexistent")
	if err == nil {
		t.Fatal("expected error for missing session")
	}

	data := &webauthn.SessionData{
		Challenge: "test-challenge",
	}
	store.Save("key1", data)
	got, err := store.Get("key1")
	if err != nil {
		t.Fatalf("Get after Save: %v", err)
	}
	if got.Challenge != "test-challenge" {
		t.Errorf("Challenge = %q", got.Challenge)
	}

	store.Delete("key1")
	_, err = store.Get("key1")
	if err == nil {
		t.Fatal("expected error after Delete")
	}
}
