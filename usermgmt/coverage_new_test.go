package usermgmt

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// expireSessionEntry manually sets a session entry's expiry to the past.
// Used by eviction tests to simulate expired sessions without waiting.
func expireSessionEntry(store *webauthnSessionStore, key string) {
	store.mu.Lock()
	defer store.mu.Unlock()
	if e, ok := store.sessions[key]; ok {
		e.expiresAt = time.Now().Add(-time.Hour)
		store.sessions[key] = e
	}
}

// --- Credential HTTP endpoint tests ---

func TestHandler_ListCredentials_Unauthenticated(t *testing.T) {
	svc := newTestServiceWithAuthz(t)
	h := NewAuthHandler(svc, HandlerConfig{Secure: new(bool)})
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/auth/credentials", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	assertStatusCode(t, w, http.StatusUnauthorized)
}

func TestHandler_ListCredentials_WithCredentials(t *testing.T) {
	svc := newTestServiceWithAuthz(t)
	registerTestUser(t, svc, "u1", "credlist@test.com")

	cred := WebAuthnCredential{
		CredentialCore: CredentialCore{ID: []byte{0xAB, 0xCD}, Name: "my-key", PublicKey: []byte{1, 2, 3}},
	}
	addTestCredential(t, svc, NewUserID("u1"), cred)

	user, err := svc.GetUser(context.Background(), NewUserID("u1"))
	if err != nil {
		t.Fatalf("GetUser: %v", err)
	}

	h := NewAuthHandler(svc, HandlerConfig{Secure: new(bool)})
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/auth/credentials", nil)
	req = req.WithContext(WithUser(context.Background(), user))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	assertStatusCode(t, w, http.StatusOK)

	result := decodeJSON[credentialListResponse](t, w)
	if len(result.Credentials) != 1 {
		t.Fatalf("expected 1 credential, got %d", len(result.Credentials))
	}
	if result.Credentials[0].Name != "my-key" {
		t.Errorf("expected name 'my-key', got %q", result.Credentials[0].Name)
	}
	expectedID := base64.RawURLEncoding.EncodeToString([]byte{0xAB, 0xCD})
	if result.Credentials[0].ID != expectedID {
		t.Errorf("expected id %q, got %q", expectedID, result.Credentials[0].ID)
	}
}

func TestHandler_DeleteCredential_Success(t *testing.T) {
	svc := newTestServiceWithAuthz(t)
	registerTestUser(t, svc, "u1", "delcred@test.com")

	credID := []byte{0x01, 0x02, 0x03}
	cred := WebAuthnCredential{CredentialCore: CredentialCore{ID: credID, Name: "to-delete", PublicKey: []byte{9}}}
	addTestCredential(t, svc, NewUserID("u1"), cred)

	user, err := svc.GetUser(context.Background(), NewUserID("u1"))
	if err != nil {
		t.Fatalf("GetUser: %v", err)
	}

	h := NewAuthHandler(svc, HandlerConfig{Secure: new(bool)})
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	encodedID := base64.RawURLEncoding.EncodeToString(credID)
	req := httptest.NewRequest(http.MethodDelete, "/auth/credentials/"+encodedID, nil)
	req = req.WithContext(WithUser(context.Background(), user))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	assertStatusCode(t, w, http.StatusOK)

	user2, _ := svc.GetUser(context.Background(), NewUserID("u1"))
	if len(user2.Credentials) != 0 {
		t.Errorf("expected 0 credentials after delete, got %d", len(user2.Credentials))
	}
}

func TestHandler_DeleteCredential_Unauthenticated(t *testing.T) {
	svc := newTestServiceWithAuthz(t)
	h := NewAuthHandler(svc, HandlerConfig{Secure: new(bool)})
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodDelete, "/auth/credentials/AAAA", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	assertStatusCode(t, w, http.StatusUnauthorized)
}

func TestHandler_DeleteCredential_InvalidEncoding(t *testing.T) {
	svc := newTestServiceWithAuthz(t)
	registerTestUser(t, svc, "u1", "badenc@test.com")

	user, err := svc.GetUser(context.Background(), NewUserID("u1"))
	if err != nil {
		t.Fatalf("GetUser: %v", err)
	}

	h := NewAuthHandler(svc, HandlerConfig{Secure: new(bool)})
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodDelete, "/auth/credentials/!!!invalid-base64!!!", nil)
	req = req.WithContext(WithUser(context.Background(), user))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	assertStatusCode(t, w, http.StatusBadRequest)
}

// --- WebAuthn session eviction tests ---

func TestWebAuthnSessionStore_EvictExpired(t *testing.T) {
	store := newWebAuthnSessionStore(0)
	store.Save("future", []byte("future-data"))
	store.Save("expired", []byte("expired-data"))
	expireSessionEntry(store, "expired")

	evicted := store.EvictExpired()
	if evicted != 1 {
		t.Errorf("expected 1 evicted, got %d", evicted)
	}

	if _, err := store.Get("expired"); err == nil {
		t.Error("expected error for expired session")
	}
}

func TestWebAuthnSessionStore_EvictExpired_KeepsUnexpired(t *testing.T) {
	store := newWebAuthnSessionStore(0)
	store.Save("future", []byte("future-data"))

	evicted := store.EvictExpired()
	if evicted != 0 {
		t.Errorf("expected 0 evicted, got %d", evicted)
	}

	if _, err := store.Get("future"); err != nil {
		t.Errorf("expected future session to still exist: %v", err)
	}
}

func TestService_Stop_StopsWebAuthnEviction(t *testing.T) {
	svc := newWebAuthnTestServiceWithConfig(t)
	svc.Stop()
	svc.Stop() // double-stop should be safe
}

func TestService_Stop_NoWebAuthn(t *testing.T) {
	svc := newTestService(t)
	svc.Stop()
}

// --- CasbinProjection coverage tests ---

func TestNewCasbinProjection_NilAuthz(t *testing.T) {
	_, err := NewCasbinProjection(nil)
	if err == nil {
		t.Error("expected error for nil authz")
	}
}

func TestCasbinProjection_EventTypes_IncludesCredentials(t *testing.T) {
	a := newTestAuthz(t)
	p, err := NewCasbinProjection(a)
	if err != nil {
		t.Fatalf("NewCasbinProjection: %v", err)
	}
	types := p.EventTypes()

	foundCredAdded := false
	foundCredRemoved := false
	for _, et := range types {
		if et == eventCredentialAdded {
			foundCredAdded = true
		}
		if et == eventCredentialRemoved {
			foundCredRemoved = true
		}
	}
	if !foundCredAdded {
		t.Error("EventTypes() missing CredentialAdded")
	}
	if !foundCredRemoved {
		t.Error("EventTypes() missing CredentialRemoved")
	}
}

// --- AccountLockout integration with BeginLogin ---

func newLockoutTestService(t *testing.T, config LockoutConfig) *Service {
	t.Helper()
	svc, err := NewService(ServiceConfig{
		Lockout:  NewAccountLockout(config),
		WebAuthn: testWebAuthnProvider{},
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	t.Cleanup(svc.Stop)
	return svc
}

func TestBeginLogin_AccountLocked(t *testing.T) {
	svc := newLockoutTestService(t, LockoutConfig{
		MaxAttempts: 2,
		Duration:    time.Minute,
	})

	registerTestUser(t, svc, "u1", "locked@test.com")

	// Record enough failures to trigger lockout
	svc.lockout.RecordFailure("locked@test.com")
	svc.lockout.RecordFailure("locked@test.com")

	_, err := svc.BeginLogin(context.Background(), "locked@test.com")
	if err == nil {
		t.Fatal("expected ErrAccountLocked")
	}
	if !errors.Is(err, ErrAccountLocked) {
		t.Errorf("expected ErrAccountLocked, got: %v", err)
	}
}

func TestBeginLogin_LockoutNotTriggered(t *testing.T) {
	svc := newLockoutTestService(t, LockoutConfig{
		MaxAttempts: 5,
		Duration:    time.Minute,
	})

	registerTestUser(t, svc, "u1", "unlocked@test.com")
	// Add a credential so BeginLogin doesn't fail with ErrNoCredentials
	cred := WebAuthnCredential{CredentialCore: CredentialCore{ID: []byte{1}, PublicKey: []byte{2}}}
	addTestCredential(t, svc, NewUserID("u1"), cred)

	// Single failure should not lock
	svc.lockout.RecordFailure("unlocked@test.com")

	_, err := svc.BeginLogin(context.Background(), "unlocked@test.com")
	if err != nil {
		t.Errorf("expected success, got: %v", err)
	}
}
